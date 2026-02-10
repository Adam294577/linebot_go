package controllers

import (
	response "project/services/responses"

	"github.com/gin-gonic/gin"
)

// Health 健康檢查
func Health(c *gin.Context) {
	response.New(c).Success("LINE Bot Webhook API is running").Send()
}
