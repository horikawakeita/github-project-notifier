package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// GitHub GraphQL APIのレスポンス構造体
type GitHubResponse struct {
	Data struct {
		Node struct {
			Items struct {
				Nodes []ProjectItem `json:"nodes"`
			} `json:"items"`
		} `json:"node"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// Projectアイテムの構造体
type ProjectItem struct {
	ID      string      `json:"id"`
	Content interface{} `json:"content"`
	FieldValues struct {
		Nodes []FieldValue `json:"nodes"`
	} `json:"fieldValues"`
}

// フィールド値の構造体
type FieldValue struct {
	Field struct {
		Name string `json:"name"`
	} `json:"field"`
	Name string `json:"name"`
	Text string `json:"text"`
}

// Mattermost通知用の構造体
type MattermostMessage struct {
	Text        string `json:"text"`
	Username    string `json:"username"`
	IconEmoji   string `json:"icon_emoji"`
	Channel     string `json:"channel"`
	Attachments []struct {
		Color string `json:"color"`
		Title string `json:"title"`
		Text  string `json:"text"`
	} `json:"attachments"`
}

// 設定構造体
type Config struct {
	GitHubToken        string
	ProjectID          string
	ProjectOwner       string
	ProjectNumber      int
	ProjectViewNumber  int
	MattermostURL      string
	TargetStatus       string
	StatusFieldName    string
	InsecureSkipVerify bool
}

// Project検索用のレスポンス構造体
type ProjectSearchResponse struct {
	Data struct {
		User         *ProjectOwner `json:"user"`
		Organization *ProjectOwner `json:"organization"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type ProjectOwner struct {
	ProjectsV2 struct {
		Nodes []Project `json:"nodes"`
	} `json:"projectsV2"`
}

type Project struct {
	ID     string `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
}

func main() {
	// .envファイルを読み込み
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// GitHub Tokenを取得（環境変数またはgh auth token）
	githubToken, err := getGitHubToken()
	if err != nil {
		log.Fatal("GitHub Token取得エラー:", err)
	}

	// 設定を読み込み
	projectNumber, _ := strconv.Atoi(os.Getenv("PROJECT_NUMBER"))
	projectViewNumber, _ := strconv.Atoi(getEnvWithDefault("PROJECT_VIEW_NUMBER", "1"))
	insecureSkipVerify := strings.ToLower(os.Getenv("INSECURE_SKIP_VERIFY")) == "true"
	config := &Config{
		GitHubToken:        githubToken,
		ProjectID:          os.Getenv("PROJECT_ID"),
		ProjectOwner:       os.Getenv("PROJECT_OWNER"),
		ProjectNumber:      projectNumber,
		ProjectViewNumber:  projectViewNumber,
		MattermostURL:      os.Getenv("MATTERMOST_WEBHOOK_URL"),
		TargetStatus:       getEnvWithDefault("TARGET_STATUS", "In review"),
		StatusFieldName:    getEnvWithDefault("STATUS_FIELD_NAME", "Status"),
		InsecureSkipVerify: insecureSkipVerify,
	}

	// Project IDが設定されていない場合は自動取得を試行
	if config.ProjectID == "" {
		if config.ProjectOwner != "" && config.ProjectNumber > 0 {
			log.Printf("Project IDが設定されていません。%s/#%d のProject IDを取得中...", config.ProjectOwner, config.ProjectNumber)
			projectID, err := getProjectID(config)
			if err != nil {
				log.Fatal("Project ID取得エラー:", err)
			}
			config.ProjectID = projectID
			log.Printf("Project ID を取得しました: %s", projectID)
		}
	}

	if err := validateConfig(config); err != nil {
		log.Fatal("設定エラー:", err)
	}

	// GitHub APIからProjectアイテムを取得
	items, err := getProjectItems(config)
	if err != nil {
		log.Fatal("GitHub APIエラー:", err)
	}

	// In reviewステータスのアイテムをフィルタ
	inReviewItems := filterItemsByStatus(items, config.TargetStatus, config.StatusFieldName)

	if len(inReviewItems) == 0 {
		log.Println("In reviewステータスのアイテムはありません")
		return
	}

	// Mattermostに通知
	if err := sendMattermostNotification(config, inReviewItems); err != nil {
		log.Fatal("Mattermost通知エラー:", err)
	}

	log.Printf("成功: %d個のIn reviewアイテムをMattermostに通知しました", len(inReviewItems))
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getGitHubToken() (string, error) {
	// 1. 環境変数をチェック
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		log.Println("GitHub Token: 環境変数 GITHUB_TOKEN を使用")
		return token, nil
	}

	// 2. gh auth token コマンドを実行
	log.Println("GitHub Token: gh auth token コマンドを実行中...")
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gh auth token コマンドの実行に失敗しました。ghコマンドがインストールされ、認証済みか確認してください: %w", err)
	}

	token := strings.TrimSpace(string(output))
	if token == "" {
		return "", fmt.Errorf("gh auth token からトークンを取得できませんでした。'gh auth login' で認証してください")
	}

	log.Println("GitHub Token: gh auth token コマンドから取得")
	return token, nil
}

func getProjectID(config *Config) (string, error) {
	// GitHub GraphQL APIクエリでProject IDを取得
	query := `
	query($owner: String!) {
		user(login: $owner) {
			projectsV2(first: 100) {
				nodes {
					id
					number
					title
				}
			}
		}
		organization(login: $owner) {
			projectsV2(first: 100) {
				nodes {
					id
					number
					title
				}
			}
		}
	}`

	requestBody := map[string]interface{}{
		"query": query,
		"variables": map[string]interface{}{
			"owner": config.ProjectOwner,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("JSONマーシャリングエラー: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.github.com/graphql", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("HTTPリクエスト作成エラー: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+config.GitHubToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTPリクエストエラー: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("レスポンス読み取りエラー: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API エラー (ステータス: %d): %s", resp.StatusCode, string(body))
	}

	var searchResp ProjectSearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return "", fmt.Errorf("JSONアンマーシャリングエラー: %w", err)
	}

	// エラーがあってもorganizationまたはuserのデータが取得できていれば続行
	if len(searchResp.Errors) > 0 {
		log.Printf("GraphQL で部分的にエラーが発生しましたが続行します: %s", searchResp.Errors[0].Message)
	}

	// デバッグ用コードは削除（本番環境では不要）
	// log.Printf("GraphQL レスポンス (成功): %s", string(body))

	// ユーザーまたは組織のプロジェクトから対象を検索
	var projects []Project
	if searchResp.Data.User != nil {
		log.Printf("ユーザープロジェクトが見つかりました: %d個", len(searchResp.Data.User.ProjectsV2.Nodes))
		projects = append(projects, searchResp.Data.User.ProjectsV2.Nodes...)
	}
	if searchResp.Data.Organization != nil {
		log.Printf("組織プロジェクトが見つかりました: %d個", len(searchResp.Data.Organization.ProjectsV2.Nodes))
		projects = append(projects, searchResp.Data.Organization.ProjectsV2.Nodes...)
	}

	// プロジェクト番号で検索
	for _, project := range projects {
		if project.Number == config.ProjectNumber {
			log.Printf("見つかったプロジェクト: #%d - %s", project.Number, project.Title)
			return project.ID, nil
		}
	}

	// プロジェクトが見つからない場合のエラーハンドリング
	if len(projects) == 0 {
		// データが全く取得できない場合のみエラー
		if len(searchResp.Errors) > 0 {
			return "", fmt.Errorf("GitHub GraphQL エラー: %s", searchResp.Errors[0].Message)
		}
		return "", fmt.Errorf("プロジェクトデータが取得できませんでした")
	}

	// プロジェクトが見つからない場合は利用可能なプロジェクト一覧を表示
	log.Printf("プロジェクト #%d が見つかりませんでした。利用可能なプロジェクト:", config.ProjectNumber)
	for _, project := range projects {
		log.Printf("  #%d - %s (ID: %s)", project.Number, project.Title, project.ID)
	}

	return "", fmt.Errorf("プロジェクト #%d が %s に見つかりませんでした", config.ProjectNumber, config.ProjectOwner)
}

func validateConfig(config *Config) error {
	if config.GitHubToken == "" {
		return fmt.Errorf("GitHub Token が取得できませんでした")
	}
	if config.ProjectID == "" {
		if config.ProjectOwner == "" || config.ProjectNumber == 0 {
			return fmt.Errorf("PROJECT_ID または (PROJECT_OWNER と PROJECT_NUMBER) の組み合わせが設定されていません")
		}
		return fmt.Errorf("Project ID の取得に失敗しました")
	}
	if config.MattermostURL == "" {
		return fmt.Errorf("MATTERMOST_WEBHOOK_URL環境変数が設定されていません")
	}
	return nil
}

func getProjectItems(config *Config) ([]ProjectItem, error) {
	// GitHub GraphQL APIクエリ
	query := `
	query($projectId: ID!) {
		node(id: $projectId) {
			... on ProjectV2 {
				items(first: 100) {
					nodes {
						id
						content {
							... on Issue {
								title
								url
								assignees(first: 10) {
									nodes {
										login
										name
									}
								}
							}
							... on PullRequest {
								title
								url
								assignees(first: 10) {
									nodes {
										login
										name
									}
								}
							}
							... on DraftIssue {
								title
								assignees(first: 10) {
									nodes {
										login
										name
									}
								}
							}
						}
						fieldValues(first: 10) {
							nodes {
								... on ProjectV2ItemFieldSingleSelectValue {
									field {
										... on ProjectV2SingleSelectField {
											name
										}
									}
									name
								}
								... on ProjectV2ItemFieldTextValue {
									field {
										... on ProjectV2Field {
											name
										}
									}
									text
								}
							}
						}
					}
				}
			}
		}
	}`

	requestBody := map[string]interface{}{
		"query": query,
		"variables": map[string]interface{}{
			"projectId": config.ProjectID,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("JSONマーシャリングエラー: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.github.com/graphql", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("HTTPリクエスト作成エラー: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+config.GitHubToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTPリクエストエラー: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("レスポンス読み取りエラー: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API エラー (ステータス: %d): %s", resp.StatusCode, string(body))
	}

	var githubResp GitHubResponse
	if err := json.Unmarshal(body, &githubResp); err != nil {
		return nil, fmt.Errorf("JSONアンマーシャリングエラー: %w", err)
	}

	if len(githubResp.Errors) > 0 {
		return nil, fmt.Errorf("GitHub GraphQL エラー: %s", githubResp.Errors[0].Message)
	}

	return githubResp.Data.Node.Items.Nodes, nil
}

func filterItemsByStatus(items []ProjectItem, targetStatus, statusFieldName string) []ProjectItem {
	var filtered []ProjectItem

	for _, item := range items {
		for _, fieldValue := range item.FieldValues.Nodes {
			if fieldValue.Field.Name == statusFieldName && (fieldValue.Name == targetStatus || fieldValue.Text == targetStatus) {
				filtered = append(filtered, item)
				break
			}
		}
	}

	return filtered
}

// アイテムの表示名を取得（Content.TitleまたはTitleフィールドから）
func getItemDisplayName(item ProjectItem) string {
	// 1. Content内のTitleをチェック
	if contentMap, ok := item.Content.(map[string]interface{}); ok {
		if title, exists := contentMap["title"]; exists {
			if titleStr, ok := title.(string); ok && titleStr != "" {
				return titleStr
			}
		}
	}

	// 2. フィールドからTitleを探す
	for _, fieldValue := range item.FieldValues.Nodes {
		if fieldValue.Field.Name == "Title" {
			if fieldValue.Text != "" {
				return fieldValue.Text
			}
			if fieldValue.Name != "" {
				return fieldValue.Name
			}
		}
	}

	// 3. どちらもない場合はデフォルト名
	return "無題のアイテム"
}

// アイテムのAssigneesを取得
func getItemAssignees(item ProjectItem) []string {
	var assignees []string
	
	// Content内のAssigneesをチェック
	if contentMap, ok := item.Content.(map[string]interface{}); ok {
		if assigneesData, exists := contentMap["assignees"]; exists {
			if assigneesMap, ok := assigneesData.(map[string]interface{}); ok {
				if nodes, exists := assigneesMap["nodes"]; exists {
					if nodesList, ok := nodes.([]interface{}); ok {
						for _, node := range nodesList {
							if nodeMap, ok := node.(map[string]interface{}); ok {
								login, _ := nodeMap["login"].(string)
								name, _ := nodeMap["name"].(string)
								
								if name != "" {
									assignees = append(assignees, name)
								} else {
									assignees = append(assignees, login)
								}
							}
						}
					}
				}
			}
		}
	}
	
	return assignees
}

func sendMattermostNotification(config *Config, items []ProjectItem) error {
	message := MattermostMessage{}

	// プロジェクトURLを生成（特定のビューを指定）
	var projectURL string
	if config.ProjectViewNumber > 1 {
		projectURL = fmt.Sprintf("https://github.com/orgs/%s/projects/%d/views/%d", config.ProjectOwner, config.ProjectNumber, config.ProjectViewNumber)
	} else {
		projectURL = fmt.Sprintf("https://github.com/orgs/%s/projects/%d", config.ProjectOwner, config.ProjectNumber)
	}
	
	// メインメッセージにアイテム名も含める
	if len(items) == 1 {
		// アイテム名を取得（タイトルまたはTitleフィールドから）
		itemName := getItemDisplayName(items[0])
		message.Text = fmt.Sprintf("📋 **%s** がレビュー待ちです\n[プロジェクトを確認する](%s)", itemName, projectURL)
	} else {
		message.Text = fmt.Sprintf("📋 レビュー待ちのアイテムが%d件あります\n[プロジェクトを確認する](%s)", len(items), projectURL)
	}

	// アイテムごとにattachmentを作成
	for _, item := range items {
		itemName := getItemDisplayName(item)
		assignees := getItemAssignees(item)
		
		// Assignees表示用のテキストを作成
		var assigneeText string
		if len(assignees) > 0 {
			assigneeText = "👤 " + strings.Join(assignees, ", ")
		} else {
			assigneeText = "👤 未割り当て"
		}

		attachment := struct {
			Color string `json:"color"`
			Title string `json:"title"`
			Text  string `json:"text"`
		}{
			Color: "warning",
			Title: itemName,
			Text:  assigneeText,
		}
		message.Attachments = append(message.Attachments, attachment)
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("JSONマーシャリングエラー: %w", err)
	}

	req, err := http.NewRequest("POST", config.MattermostURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("HTTPリクエスト作成エラー: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// SSL証明書の検証設定
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.InsecureSkipVerify,
		},
	}
	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
	
	if config.InsecureSkipVerify {
		log.Println("警告: SSL証明書の検証をスキップします")
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTPリクエストエラー: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Mattermost通知エラー (ステータス: %d): %s", resp.StatusCode, string(body))
	}

	return nil
}