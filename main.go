package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"project/controllers"
	"project/cron"
	"project/middlewares"
	"project/routes"
	linebotsvc "project/services/linebot"
	"project/services/log"
	response "project/services/responses"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// @title Landtop API
// @version 1.0
// @description Landtop API文檔
// @host localhost:8002
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description 請輸入 Bearer Token，格式為：Bearer <token>
// @schemes http https

// init 在程式啟動時載入 .env 並初始化 viper，僅使用環境變數作為設定來源
func init() {
	// 先嘗試載入 .env（若不存在則忽略錯誤）
	if err := godotenv.Load(); err != nil {
		fmt.Println("godotenv.Load() skipped or .env not found:", err)
	}

	// 初始化配置來源：只使用環境變數（包含 .env 載入後的值）
	initConfig()
}

// initConfig 初始化配置：不再強制依賴 YAML 檔，只透過環境變數（含 .env）取得設定
func initConfig() {
	// 設置環境變數替換規則：將配置路徑中的點號替換為下劃線
	// 例如 Server.Website.Port 可以通過 SERVER_WEBSITE_PORT 環境變數覆蓋
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
}

var HttpServer *gin.Engine

func main() {
	// 捕獲panic不崩潰
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("recover error", err)
		}
	}()
	App(HttpServer)
}

func App(HttpServer *gin.Engine) {
	numCPUs := runtime.NumCPU()
	log.Info("CPU cores: %d", numCPUs)

	// // 初始化並檢查 Redis 連接（在啟動其他服務前）
	// redisClient := redis.NewRedisClient()
	// if redisClient.IsAvailable() {
	// 	fmt.Println("✓ Redis 緩存功能已啟用")
	// } else {
	// 	fmt.Println("⚠ Redis 緩存功能未啟用，將使用優雅降級模式（直接查詢資料庫）")
	// }
	// redisClient.Close() // 關閉測試連接，後續使用時會重新創建

	// 啟動Gin服務
	HttpServer = gin.Default()
	// 設定信任的 Proxy（請修改為你的反向代理 IP）
	if err := HttpServer.SetTrustedProxies(nil); err != nil {
		fmt.Println("設定信任Proxy錯誤")
		return
	}

	// 初始化 LINE Bot Service 與 Controller，並透過 middleware 注入到 context
	// LineBotService 使用 project/services/imageai 進行圖片食物辨識（resize、openai、context）
	lineService, err := linebotsvc.NewLineBotServiceFromEnv()
	if err != nil {
		log.Error("初始化 LINE Bot 服務失敗: %v", err)
	} else {
		lineController := controllers.NewLineController(lineService)
		// 將 LineController 中介層掛上，讓 /line/webhook 能從 context 取得
		HttpServer.Use(middlewares.LineControllerMiddleware(lineController))
	}

	// 啟動伺服器
	// 優先使用環境變數 PORT（雲平台標準做法）
	port := os.Getenv("PORT")
	if port == "" {
		// 如果沒有環境變數，使用配置文件中的端口
		port = viper.GetString("Server.Website.Port")
	}
	if port == "" {
		port = "8002" // 默認端口
	}

	// 使用 middleware（CORS 需要最先執行，排除 Swagger 路径）
	HttpServer.Use(
		func(ctx *gin.Context) {
			// 排除 Swagger 路径
			if strings.HasPrefix(ctx.Request.URL.Path, "/swagger") {
				ctx.Next()
				return
			}
			middlewares.CORS()(ctx)
		},
		func(ctx *gin.Context) {
			// 排除 Swagger 路径
			if strings.HasPrefix(ctx.Request.URL.Path, "/swagger") {
				ctx.Next()
				return
			}
			// 設定變數
			ctx.Set("requestID", ctx.Request.Header.Get("X-Request-ID"))
			ctx.Next()
		},
		func(ctx *gin.Context) {
			// 排除 Swagger 路径
			if strings.HasPrefix(ctx.Request.URL.Path, "/swagger") {
				ctx.Next()
				return
			}
			middlewares.Logger()(ctx)
		},
		gin.Recovery(),
	)

	// 執行排程
	go cron.Run()
	// 注冊路由
	routes.Setup(HttpServer)

	// 當Route不存在時的處理
	HttpServer.NoRoute(func(ctx *gin.Context) {
		resp := response.New(ctx)
		resp.Fail(http.StatusNotFound, "路由不存在").Send()
	})

	startServer(HttpServer, port)
}

func startServer(router *gin.Engine, port string) {
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: router,
	}

	go func() {
		fmt.Printf("伺服器運行於 %s port \n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("listen: %s\n", err.Error())
		}
	}()

	// 优雅关闭逻辑
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM) // Windows支持这两个信号
	<-quit
	fmt.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Server forced to shutdown: %s\n", err.Error())
	}
	fmt.Println("Server exiting")
}
