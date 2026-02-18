package routes

import (
	"project/controllers"
	"project/middlewares"

	"github.com/gin-gonic/gin"
)

// Setup 註冊所有路由（/s3/getImage 觸發時才從環境變數判斷是否可用）
func Setup(r *gin.Engine) {
	r.GET("/", controllers.Health)
	r.POST("/s3/getImage", controllers.S3GetImageHandler)

	// LINE Webhook（LineController 由 middleware.LineControllerMiddleware 注入）
	// dev: https://f16e-118-232-75-172.ngrok-free.app/line/webhook
	// prod: https://my-go-line-bot.zeabur.app/line/webhook
	r.POST("/line/webhook", middlewares.WebhookFromContext)
}
