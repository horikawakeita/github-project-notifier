.PHONY: build run clean install help

# デフォルトターゲット
.DEFAULT_GOAL := help

# ビルド設定
BINARY_NAME=github-project-notifier
DIST_DIR=dist

# ヘルプ
help:
	@echo "GitHub Project Notifier - 利用可能なコマンド:"
	@echo ""
	@echo "  make build     - 全プラットフォーム用にビルド"
	@echo "  make run       - 開発モードで実行"
	@echo "  make install   - 依存関係をインストール"
	@echo "  make clean     - ビルド成果物を削除"
	@echo "  make test      - テストを実行"
	@echo ""

# 依存関係のインストール
install:
	@echo "📦 依存関係をインストール中..."
	go mod tidy
	go mod download

# 開発モードで実行
run: install
	@echo "🚀 開発モードで実行中..."
	go run main.go

# ビルド（全プラットフォーム）
build: install
	@echo "🔧 全プラットフォーム用にビルド中..."
	@mkdir -p $(DIST_DIR)
	
	@echo "🖥️  現在のOS用..."
	go build -ldflags="-s -w" -o $(DIST_DIR)/$(BINARY_NAME) main.go
	
	@echo "🐧 Linux用..."
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(DIST_DIR)/$(BINARY_NAME)-linux main.go
	
	@echo "🪟 Windows用..."
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(DIST_DIR)/$(BINARY_NAME).exe main.go
	
	@echo "🍎 macOS (Intel)用..."
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(DIST_DIR)/$(BINARY_NAME)-macos-intel main.go
	
	@echo "🍎 macOS (Apple Silicon)用..."
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(DIST_DIR)/$(BINARY_NAME)-macos-arm64 main.go
	
	@echo ""
	@echo "✅ ビルド完了！"
	@ls -lh $(DIST_DIR)/

# 現在のOS用のみビルド
build-local: install
	@echo "🔧 現在のOS用にビルド中..."
	@mkdir -p $(DIST_DIR)
	go build -ldflags="-s -w" -o $(DIST_DIR)/$(BINARY_NAME) main.go
	@echo "✅ ビルド完了: $(DIST_DIR)/$(BINARY_NAME)"

# テスト実行
test:
	@echo "🧪 テストを実行中..."
	go test -v ./...

# クリーンアップ
clean:
	@echo "🧹 ビルド成果物を削除中..."
	rm -rf $(DIST_DIR)
	@echo "✅ クリーンアップ完了"

# 設定ファイルの準備
setup:
	@if [ ! -f .env ]; then \
		echo "⚙️  設定ファイルを準備中..."; \
		cp config.env.example .env; \
		echo "✅ .env ファイルを作成しました。"; \
		echo ""; \
		echo "📝 設定してください:"; \
		echo "  1. PROJECT_OWNER（組織名またはユーザー名）"; \
		echo "  2. PROJECT_NUMBER（プロジェクト番号）"; \
		echo "  3. MATTERMOST_WEBHOOK_URL"; \
		echo ""; \
		echo "💡 PROJECT_NUMBER はプロジェクトのURLから確認できます:"; \
		echo "  https://github.com/users/USERNAME/projects/1 → PROJECT_NUMBER=1"; \
	else \
		echo "ℹ️  .env ファイルは既に存在します"; \
	fi