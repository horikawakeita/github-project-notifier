# GitHub Project Notifier

GitHub ProjectV2の「In review」ステータスのアイテムを取得し、Mattermostに通知するツールです。

## 機能

- GitHub GraphQL API（v2）を使用してProjectアイテムを取得
- GitHub CLI (`gh`) との連携によるシームレスなトークン取得
- 指定したステータスのアイテムをフィルタリング  
- Mattermostへのリッチな通知送信（アイテム名・担当者情報付き）
- 複数のプロジェクトアイテムタイプに対応（Issue、PullRequest、DraftIssue）
- 環境依存のない実行ファイルの生成

## 最短実行手順

すぐに使い始めたい場合の手順です（詳細な設定は下記の各セクションを参照）：

### 1. 実行ファイルのダウンロード

OS別のビルド済み実行ファイルが`dist/`ディレクトリに用意されています：

- **Linux**: `dist/github-project-notifier-linux`
- **macOS (Intel)**: `dist/github-project-notifier-macos-intel`
- **macOS (Apple Silicon)**: `dist/github-project-notifier-macos-arm64`
- **Windows**: `dist/github-project-notifier.exe`

### 2. GitHub CLI で認証

```bash
# GitHub CLI をインストール（未インストールの場合）
brew install gh  # macOS
# または sudo apt install gh  # Linux
# または winget install --id GitHub.cli  # Windows

# GitHub にログイン
gh auth login
```

### 3. 設定ファイルの作成

```bash
# 設定例をコピー
cp config.env.example .env

# .env を編集（最低限必要な設定）
PROJECT_OWNER=your-organization-or-username
PROJECT_NUMBER=1
MATTERMOST_WEBHOOK_URL=https://your-mattermost.com/hooks/xxxxxxxx
```

### 4. 実行

```bash
# Linux/macOS の場合
chmod +x dist/github-project-notifier-linux  # または適切なファイル名
./dist/github-project-notifier-linux

# Windows の場合
.\dist\github-project-notifier.exe
```

以上で完了です！レビュー待ちのアイテムがある場合、Mattermostに通知が送信されます。

---

## 必要な準備

### 1. GitHub CLI (gh) のインストールと認証

このツールは自動的に `gh auth token` コマンドを使用してGitHubトークンを取得します。

1. GitHub CLI をインストール:
   ```bash
   # macOS
   brew install gh
   
   # Windows
   winget install --id GitHub.cli
   
   # Linux (Ubuntu/Debian)
   sudo apt install gh
   ```

2. GitHub CLI で認証:
   ```bash
   gh auth login
   ```

**手動でトークンを設定したい場合**:
環境変数 `GITHUB_TOKEN` を設定すると、そちらが優先されます。必要な権限:
- `repo` (プライベートリポジトリのプロジェクトの場合)
- `read:project` (プロジェクトの読み取り)

### 2. プロジェクト情報の設定

プロジェクトを指定する方法は2つあります：

#### 方法1: オーナー名とプロジェクト番号（推奨）

```bash
PROJECT_OWNER=your-organization-or-username
PROJECT_NUMBER=1
```

プロジェクト番号は、GitHubでプロジェクトページのURLに表示される番号です：
- `https://github.com/users/USERNAME/projects/1` → `PROJECT_NUMBER=1`
- `https://github.com/orgs/ORG_NAME/projects/5` → `PROJECT_NUMBER=5`

#### プロジェクトビュー番号の指定（オプション）

特定のビューにリンクしたい場合は、`PROJECT_VIEW_NUMBER`を設定します：
- `https://github.com/orgs/ORG_NAME/projects/5/views/2` → `PROJECT_VIEW_NUMBER=2`
- 未設定またはデフォルト（1）の場合は、プロジェクトのトップページにリンクします

#### 方法2: Project IDを直接指定

```bash
PROJECT_ID=PVT_kwDOAxxxxxxxxxxxxx
```

Project IDは以下の方法で取得できます：
- GitHub CLI: `gh project list --owner OWNER_NAME`
- GraphQL Explorer
- ブラウザの開発者ツール

> **注意**: 方法1の方が簡単で推奨されます。Project IDは自動で取得されます。

### 3. Mattermost Incoming Webhook URL の設定

Mattermostの管理画面でIncoming Webhookを作成し、URLを取得してください。

## 設定

`config.env.example`を`.env`にコピーして設定を記入してください：

```bash
cp config.env.example .env
```

### 環境変数

| 変数名 | 必須 | デフォルト値 | 説明 |
|--------|------|-------------|------|
| `GITHUB_TOKEN` | - | - | GitHub Personal Access Token（省略時は `gh auth token` を使用） |
| `PROJECT_ID` | (*1) | - | GitHub Project ID（直接指定する場合） |
| `PROJECT_OWNER` | (*1) | - | プロジェクトのオーナー名（組織名またはユーザー名） |
| `PROJECT_NUMBER` | (*1) | - | プロジェクト番号（GitHubのURLに表示される番号） |
| `PROJECT_VIEW_NUMBER` | - | `1` | プロジェクトビュー番号（通知リンクで使用） |
| `MATTERMOST_WEBHOOK_URL` | ✓ | - | Mattermost Incoming Webhook URL |
| `TARGET_STATUS` | - | `In review` | 通知対象のステータス |
| `STATUS_FIELD_NAME` | - | `Status` | ステータスフィールドの名前 |
| `INSECURE_SKIP_VERIFY` | - | `false` | SSL証明書検証をスキップ（内部証明書の場合） |

> (*1) `PROJECT_ID` または (`PROJECT_OWNER` + `PROJECT_NUMBER`) のいずれかが必須

## ビルドと実行

### 開発モードで実行

```bash
# 依存関係のインストール
go mod tidy

# 実行
go run main.go
```

### 実行ファイルのビルド

```bash
# 現在のOS用
go build -o github-project-notifier main.go

# Linux用
GOOS=linux GOARCH=amd64 go build -o github-project-notifier-linux main.go

# Windows用
GOOS=windows GOARCH=amd64 go build -o github-project-notifier.exe main.go

# macOS用
GOOS=darwin GOARCH=amd64 go build -o github-project-notifier-macos main.go
```

### 実行

```bash
./github-project-notifier
```

## 定期実行の設定

### cron での設定例

```bash
# 毎日午前9時に実行
0 9 * * * /path/to/github-project-notifier
```

### GitHub Actions での設定例

```yaml
name: GitHub Project Notifier

on:
  schedule:
    - cron: '0 9 * * *'  # 毎日午前9時（UTC）
  workflow_dispatch:  # 手動実行も可能

jobs:
  notify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run notifier
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          PROJECT_ID: ${{ secrets.PROJECT_ID }}
          MATTERMOST_WEBHOOK_URL: ${{ secrets.MATTERMOST_WEBHOOK_URL }}
        run: |
          cd github-project-notifier
          go run main.go
```

## トラブルシューティング

### よくあるエラー

1. **GitHub Token取得エラー**
   - `gh` コマンドがインストールされていない → GitHub CLI をインストール
   - `gh auth login` で認証していない → 認証を実行
   - 環境変数 `GITHUB_TOKEN` を設定している場合は、権限不足または有効期限切れ

2. **Project ID取得エラー**
   - `PROJECT_OWNER` が間違っている → 正しい組織名またはユーザー名を設定
   - `PROJECT_NUMBER` が間違っている → GitHubのプロジェクトURLで番号を確認
   - プロジェクトへのアクセス権限がない → プロジェクトの権限を確認

3. **GitHub API エラー (401 Unauthorized)**
   - GitHub Tokenが無効または権限不足
   - Tokenの有効期限切れ

4. **GitHub GraphQL エラー (Project not found)**
   - Project IDが間違っている
   - Projectへのアクセス権限がない

5. **Mattermost通知エラー**
   - Webhook URLが間違っている
   - Mattermostサーバーへの接続問題

### 利用可能なプロジェクト一覧の確認

プロジェクト番号が分からない場合、ツールを一度実行すると利用可能なプロジェクト一覧が表示されます：

```bash
PROJECT_OWNER=your-org PROJECT_NUMBER=999 ./github-project-notifier
```

間違った番号を指定すると、利用可能なプロジェクト一覧が表示されるので参考にしてください。

### デバッグ

詳細なログを確認したい場合は、ソースコードの`log.Println`を`log.Printf`に変更してより詳細な情報を出力できます。

## 通知形式

通知には以下の情報が含まれます：

### 複数アイテムの場合
```
📋 レビュー待ちのアイテムが2件あります
[プロジェクトを確認する](https://github.com/orgs/OWNER/projects/NUMBER)

[Attachment 1]
タイトル: アイテム名1
担当者: 👤 田中太郎

[Attachment 2] 
タイトル: アイテム名2
担当者: 👤 未割り当て
```

### 単一アイテムの場合
```
📋 アイテム名 がレビュー待ちです
[プロジェクトを確認する](https://github.com/orgs/OWNER/projects/NUMBER)

タイトル: アイテム名
担当者: 👤 田中太郎
```

### サポートするプロジェクトアイテムタイプ

- **Issue**: GitHubリポジトリのIssue
- **PullRequest**: GitHubリポジトリのPull Request  
- **DraftIssue**: プロジェクト内で直接作成されたアイテム

すべてのタイプでアイテム名と担当者（Assignees）情報を正しく表示できます。

## カスタマイズ

- `TARGET_STATUS`環境変数を変更することで、異なるステータスのアイテムを監視可能
- `main.go`のMattermostメッセージ構造を変更することで、通知形式をカスタマイズ可能
- GraphQLクエリを変更することで、取得するフィールドを追加可能

## ライセンス

MIT License