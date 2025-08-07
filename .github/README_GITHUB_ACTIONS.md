# GitHub Actions セットアップガイド

このリポジトリでは、GitHub Actionsを使用してGitHub Projectの通知を自動化できます。

## 実行スケジュール

ワークフローは以下のスケジュールで自動実行されます：
- **平日（月〜金）の8時、12時、15時（JST）**
- 手動実行も可能

## 必要な環境変数の設定

GitHubリポジトリの Settings → Secrets and variables → Actions で以下のシークレットを設定してください：

### 必須設定

| 名前 | 説明 | 例 |
|------|------|-----|
| `PROJECT_OWNER` | プロジェクトの所有者（組織名またはユーザー名） | `your-organization` |
| `PROJECT_NUMBER` | プロジェクト番号 | `1` |
| `MATTERMOST_WEBHOOK_URL` | MattermostのWebhook URL | `https://your-mattermost.com/hooks/xxx` |

### オプション設定

| 名前 | 説明 | デフォルト値 |
|------|------|-------------|
| `PROJECT_VIEW_NUMBER` | プロジェクトビュー番号 | `1` |
| `TARGET_STATUS` | 通知対象のステータス | `In review` |
| `STATUS_FIELD_NAME` | ステータスフィールド名 | `Status` |
| `INSECURE_SKIP_VERIFY` | SSL証明書検証をスキップ | `false` |

## GitHub Token について

`GITHUB_TOKEN` は GitHub Actions で自動的に提供されるため、通常は手動設定不要です。
ただし、プライベートリポジトリのプロジェクトにアクセスする場合は、適切な権限を持つPersonal Access Tokenを `GITHUB_TOKEN` として設定してください。

## 手動実行

GitHub のリポジトリページで：
1. "Actions" タブを開く
2. "GitHub Project Notifier" ワークフローを選択
3. "Run workflow" ボタンをクリック

## ワークフローの無効化

ワークフローを一時的に停止したい場合：
1. "Actions" タブを開く
2. "GitHub Project Notifier" ワークフローを選択
3. "Disable workflow" をクリック

## トラブルシューティング

### 実行ログの確認

1. "Actions" タブを開く
2. 実行履歴から該当のワークフローをクリック
3. "notify" ジョブをクリックしてログを確認

### よくある問題

- **プロジェクトが見つからない**: `PROJECT_OWNER` と `PROJECT_NUMBER` が正しく設定されているか確認
- **Mattermost通知が届かない**: `MATTERMOST_WEBHOOK_URL` が正しく設定されているか確認
- **権限エラー**: プライベートリポジトリの場合、適切な権限を持つトークンを設定
