package route

import (
	"github.com/gin-gonic/gin"
	"linebot/controller"
	"linebot/middleware"
)

// Setup 註冊所有路由（LINE Webhook 依賴 middleware 注入 LineController）
func Setup(r *gin.Engine) {
	// 健康檢查
	r.GET("/", controller.Health)

	// LINE Webhook（LineController 由 middleware.LineControllerMiddleware 注入）
	r.POST("/line/webhook", middleware.WebhookFromContext)
}
