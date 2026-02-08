package controller

import (
	"github.com/gin-gonic/gin"
	"linebot/service/responses"
)

// Health 健康檢查
func Health(c *gin.Context) {
	response.New(c).Success("LINE Bot Webhook API is running").Send()
}
