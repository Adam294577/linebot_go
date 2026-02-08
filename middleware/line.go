package middleware

import (
	"github.com/gin-gonic/gin"
	"linebot/controller"
)

type contextKey string

const lineControllerKey contextKey = "line_controller"

// LineControllerMiddleware 將 LineController 注入 context，供後續 handler 使用
func LineControllerMiddleware(lc *controller.LineController) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(string(lineControllerKey), lc)
		c.Next()
	}
}

// GetLineController 從 context 取得 LineController（由 LineControllerMiddleware 注入）
func GetLineController(c *gin.Context) *controller.LineController {
	v, ok := c.Get(string(lineControllerKey))
	if !ok || v == nil {
		return nil
	}
	return v.(*controller.LineController)
}

// WebhookFromContext 從 context 取得 LineController 並處理 Webhook（給 route 無參數註冊用）
func WebhookFromContext(c *gin.Context) {
	lc := GetLineController(c)
	if lc == nil {
		c.JSON(500, gin.H{"error": "LineController not configured"})
		return
	}
	lc.Webhook(c)
}
