package routes

import (
	"project/controllers"
	"project/middlewares"

	"github.com/gin-gonic/gin"
)

// Setup 註冊所有路由（LINE Webhook 依賴 middleware 注入 LineController）
func Setup(r *gin.Engine) {
	// 健康檢查
	r.GET("/", controllers.Health)

	// DB 健康檢查
	// r.GET("/db", controllers.HealthDB)

	// LINE Webhook（LineController 由 middleware.LineControllerMiddleware 注入）
	// dev: https://9211-118-150-196-246.ngrok-free.app/line/webhook
	// prod: https://my-go-line-bot.zeabur.app/line/webhook
	r.POST("/line/webhook", middlewares.WebhookFromContext)
}
