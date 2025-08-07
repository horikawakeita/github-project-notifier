#!/bin/bash

# GitHub Project Notifier ビルドスクリプト

set -e

echo "🔧 GitHub Project Notifier をビルドしています..."

# 依存関係のインストール
echo "📦 依存関係を解決中..."
go mod tidy

# ビルドディレクトリの作成
mkdir -p dist

# 現在のOS用ビルド
echo "🖥️  現在のOS用をビルド中..."
go build -ldflags="-s -w" -o dist/github-project-notifier main.go

# Linux用ビルド
echo "🐧 Linux用をビルド中..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/github-project-notifier-linux main.go

# Windows用ビルド
echo "🪟 Windows用をビルド中..."
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/github-project-notifier.exe main.go

# macOS用ビルド（Intel）
echo "🍎 macOS (Intel)用をビルド中..."
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/github-project-notifier-macos-intel main.go

# macOS用ビルド（Apple Silicon）
echo "🍎 macOS (Apple Silicon)用をビルド中..."
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/github-project-notifier-macos-arm64 main.go

# ファイルサイズの表示
echo ""
echo "✅ ビルド完了！生成されたファイル:"
ls -lh dist/

echo ""
echo "🚀 使用方法:"
echo "1. config.env.example を .env にコピーして設定を記入"
echo "2. ./dist/github-project-notifier を実行"
echo ""
echo "📝 詳細は README.md をご確認ください"