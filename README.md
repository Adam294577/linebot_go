# LINE Bot（Go）

以 Go 實作的 LINE Bot Webhook 服務，本地開發可用 [Air](https://github.com/air-verse/air) 熱更新。

## 功能

- **POST /line/webhook**：LINE Webhook 端點，接收訊息並回覆「你說: {使用者訊息}」
- **GET /**：健康檢查，回傳 `{"status":"ok","message":"LINE Bot Webhook API is running"}`

## 設定方式（config 檔 + 環境變數）

程式依 **`ENV`** 讀取對應的 config 檔，未設時為 `prod`：

| ENV 值 | 讀取檔案 |
|--------|----------|
| `dev`  | `config/config_dev.yaml` |
| `prod`（或未設） | `config/config_prod.yaml` |

**LINE 憑證**可寫在 config 檔的 `Server.LINE_CHANNEL_SECRET`、`Server.LINE_CHANNEL_ACCESS_TOKEN`；若同時設了**環境變數** `LINE_CHANNEL_SECRET`、`LINE_CHANNEL_ACCESS_TOKEN`，會覆寫 config 內的值。

若**沒有對應的 config 檔**（例如部署時只有環境變數），則改為只從環境變數讀取 LINE 憑證。

| 變數 | 說明 |
|------|------|
| `ENV` | 執行環境，`dev` / `prod`，預設 `prod` |
| `LINE_CHANNEL_SECRET` | LINE 頻道 Secret（可放 config 或環境變數，環境變數優先） |
| `LINE_CHANNEL_ACCESS_TOKEN` | LINE 頻道 Access Token（同上） |
| `PORT` | 監聽 port，預設 `8090` |

---

## 初次安裝

1. **設定 LINE 憑證**（擇一即可）：
   - **方式 A**：編輯 `config/config_dev.yaml`，在 `Server` 下填入 `LINE_CHANNEL_SECRET`、`LINE_CHANNEL_ACCESS_TOKEN`。
   - **方式 B**：設定環境變數 `LINE_CHANNEL_SECRET`、`LINE_CHANNEL_ACCESS_TOKEN`（或複製 `env.example` 為 `.env` 後在 shell 載入）。

2. 本機需安裝 [Go](https://go.dev/dl/)（建議與 `go.mod` 同版，如 1.25）。開發時吃 `config_dev.yaml` 請設 `ENV=dev`。

---

## 如何運行（本地）

### 一般執行

```powershell
# 吃 config_dev.yaml
$env:ENV="dev"; go run .

# 或吃 config_prod.yaml（預設）
go run .
```

### 開發熱更新（Air）

安裝 [Air](https://github.com/air-verse/air) 後，在專案根目錄執行：

```powershell
# 開發時吃 config_dev.yaml，改程式會自動重編譯並重啟
$env:ENV="dev"; air -c .air.toml
```

服務位址：`http://localhost:8090`。可用 `curl http://localhost:8090/` 做健康檢查。

---

## 依賴調整後

修改 `go.mod` 後在專案根目錄執行：

```powershell
go mod tidy
```

---

## Webhook URL

部署後在 LINE Developers Console 將 Webhook URL 設為：

`https://你的網域或 IP:port/line/webhook`

LINE 僅支援 HTTPS；本機測試可用 [ngrok](https://ngrok.com/) 等工具。
