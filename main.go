package main

import (
	"log"
	"os"

	"linebot/controller"
	"linebot/middleware"
	"linebot/route"
	linebotsvc "linebot/service/linebot"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 從 .env 載入環境變數（若檔案不存在則略過）
	_ = godotenv.Load()

	channelSecret := os.Getenv("LINE_CHANNEL_SECRET")
	channelToken := os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
	if channelSecret == "" || channelToken == "" {
		log.Fatal("LINE_CHANNEL_SECRET 與 LINE_CHANNEL_ACCESS_TOKEN 必須在 .env 或環境變數中設定")
	}

	lineService, err := linebotsvc.NewLineBotService(channelSecret, channelToken)
	if err != nil {
		log.Fatal(err)
	}
	lineCtrl := controller.NewLineController(lineService)

	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.Default()
	r.SetTrustedProxies([]string{"127.0.0.1", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "::1"})

	r.Use(middleware.LineControllerMiddleware(lineCtrl))
	route.Setup(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}
	log.Printf("LINE Bot 服務啟動於 port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("HTTP 服務錯誤: %v", err)
	}
}
