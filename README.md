# LINE Bot（Go）

以 Go 實作的 LINE Bot Webhook 服務，行為與原 Firebase Cloud Functions 版本一致，可透過 Docker 部署。

## 功能

- **POST /line/webhook**：LINE Platform 的 Webhook 端點，接收訊息並回覆「你說: {使用者訊息}」
- **GET /**：健康檢查，回傳 `{"status":"ok","message":"LINE Bot Webhook API is running"}`

## 環境變數

| 變數 | 必填 | 說明 |
|------|------|------|
| `LINE_CHANNEL_SECRET` | 是 | LINE 頻道 Secret |
| `LINE_CHANNEL_ACCESS_TOKEN` | 是 | LINE 頻道 Access Token |
| `PORT` | 否 | 監聽 port，預設 `8080` |

複製 `env.example` 為 `.env` 並填入實際值：

```bash
cp env.example .env
```

## 本地執行

需安裝 Go 1.21+：

```bash
go mod tidy
go run main.go
```

或使用環境變數：

```bash
export LINE_CHANNEL_SECRET=xxx
export LINE_CHANNEL_ACCESS_TOKEN=xxx
go run main.go
```

## Docker 部署

建置映像：

```bash
docker build -t linebot .
```

> 注意：Dockerfile 會自動處理依賴，無需手動執行 `go mod tidy`

執行容器：

```bash
docker run --rm -p 8080:8080 \
  -e LINE_CHANNEL_SECRET=your_secret \
  -e LINE_CHANNEL_ACCESS_TOKEN=your_token \
  linebot
```

或使用 `.env`：

```bash
docker run --rm -p 8080:8080 --env-file .env linebot
```

### 使用 Docker Compose 運行（推薦）

**首次運行（建置並啟動）：**

1. 複製環境變數檔案並填入你的 LINE 憑證：
```bash
cp env.example .env
# 編輯 .env 檔案，填入你的 LINE_CHANNEL_SECRET 和 LINE_CHANNEL_ACCESS_TOKEN
```

2. 建置並啟動服務：
```bash
docker compose up --build -d
```

**日常運行（已建置過）：**

```bash
# 啟動服務
docker compose up -d

# 查看日誌
docker compose logs -f linebot

# 停止服務
docker compose down

# 重新建置並啟動（更新程式碼後）
docker compose up --build -d
```

服務會 listen 在 `http://localhost:8080`。

**驗證服務是否運行：**

```bash
# 健康檢查
curl http://localhost:8080/

# 應該會回傳：{"status":"ok","message":"LINE Bot Webhook API is running"}
```

### 開發模式（熱更新）

如果你在開發時需要熱更新功能（修改代碼後自動重新編譯並重啟），可以使用開發模式：

#### 使用 Docker Compose（推薦）

**首次建置並啟動：**
```bash
# 確保已設定 .env 檔案
cp env.example .env
# 編輯 .env 填入你的 LINE 憑證

# 建置並啟動（前台運行，可看到即時日誌）
docker compose -f docker-compose.dev.yml up --build

# 或背景執行
docker compose -f docker-compose.dev.yml up --build -d
```

**日常使用：**
```bash
# 啟動服務
docker compose -f docker-compose.dev.yml up -d

# 查看日誌（即時監看）
docker compose -f docker-compose.dev.yml logs -f linebot

# 停止服務
docker compose -f docker-compose.dev.yml down

# 重新建置（更新 Dockerfile 或依賴時）
docker compose -f docker-compose.dev.yml up --build -d
```

#### 使用 Docker CLI（純 Docker 命令）

**建置映像：**
```bash
docker build -f Dockerfile.dev -t linebot:dev .
```

**運行容器（熱更新模式）：**
```bash
# 前台運行
docker run --rm -p 8080:8080 \
  --env-file .env \
  -v ${PWD}:/app \
  -v /app/tmp \
  -v /app/vendor \
  linebot:dev

# 或背景執行
docker run -d --name linebot-dev \
  -p 8080:8080 \
  --env-file .env \
  -v ${PWD}:/app \
  -v /app/tmp \
  -v /app/vendor \
  linebot:dev
```

**查看日誌：**
```bash
docker logs -f linebot-dev
```

**停止容器：**
```bash
docker stop linebot-dev
docker rm linebot-dev
```

**開發模式特點：**
- ✅ 自動監聽代碼變化
- ✅ 自動重新編譯並重啟服務
- ✅ 無需手動重建容器
- ✅ 使用 volume 掛載，本地修改立即生效

**注意：**
- 開發模式使用 `air` 工具實現熱重載
- 首次啟動會自動安裝依賴
- 修改 `.go` 文件後約 1 秒內會自動重載
- 確保 `.env` 檔案已正確設定

## Webhook URL

部署後將 LINE Developers Console 的 Webhook URL 設為：

- 本機／Docker：`https://你的網域或 IP:port/line/webhook`

需為 HTTPS（LINE 僅支援 HTTPS），本機測試可用 ngrok 等工具。
