package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/line/line-bot-sdk-go/v7/linebot"
)

func main() {
	channelSecret := os.Getenv("LINE_CHANNEL_SECRET")
	channelToken := os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
	if channelSecret == "" || channelToken == "" {
		log.Fatal("LINE_CHANNEL_SECRET 與 LINE_CHANNEL_ACCESS_TOKEN 必須設定")
	}

	bot, err := linebot.New(channelSecret, channelToken)
	if err != nil {
		log.Fatalf("建立 LINE Bot 客戶端失敗: %v", err)
	}

	// 根據環境變數設定 Gin 模式
	env := os.Getenv("GIN_MODE")
	if env == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// 建立 Gin router
	r := gin.Default()

	// 健康檢查端點
	r.GET("/", healthHandler)

	// LINE Webhook 端點
	r.POST("/line/webhook", lineWebhookHandler(bot))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("LINE Bot 服務啟動於 port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("HTTP 服務錯誤: %v", err)
	}
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "LINE Bot Webhook API is running",
	})
}

func lineWebhookHandler(bot *linebot.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		events, err := bot.ParseRequest(c.Request)
		if err != nil {
			if err == linebot.ErrInvalidSignature {
				log.Printf("Webhook 簽章驗證失敗")
				c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
				return
			}
			log.Printf("解析 Webhook 失敗: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}

		for _, event := range events {
			go handleEvent(bot, event)
		}

		c.JSON(http.StatusOK, gin.H{"message": "OK"})
	}
}

func handleEvent(bot *linebot.Client, event *linebot.Event) {
	switch event.Type {
	case linebot.EventTypeMessage:
		handleMessage(bot, event)
	// 可在此擴充其他事件類型
	// case linebot.EventTypeFollow:
	// 	handleFollow(bot, event)
	// case linebot.EventTypeUnfollow:
	// 	handleUnfollow(bot, event)
	default:
		log.Printf("未處理的事件類型: %s", event.Type)
	}
}

func handleMessage(bot *linebot.Client, event *linebot.Event) {
	switch message := event.Message.(type) {
	case *linebot.TextMessage:
		userID := event.Source.UserID
		if userID == "" {
			userID = "unknown"
		}
		log.Printf("收到訊息: %s (來自: %s)", message.Text, userID)

		if _, err := bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("你說: "+message.Text)).Do(); err != nil {
			log.Printf("回覆訊息失敗: %v", err)
		}
	// 可在此擴充其他訊息類型
	// case *linebot.ImageMessage:
	// 	// 處理圖片訊息
	// case *linebot.StickerMessage:
	// 	// 處理貼圖訊息
	default:
		log.Printf("未處理的訊息類型: %T", event.Message)
	}
}
