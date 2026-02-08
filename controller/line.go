package controller

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/line/line-bot-sdk-go/v7/linebot"
	linebotsvc "linebot/service/linebot"
	"linebot/service/responses"
)

// LineController 處理 LINE Webhook 的 API 層（只負責 route 請求/回應）
type LineController struct {
	lineService *linebotsvc.LineBotService
}

// NewLineController 建立 LINE Webhook 控制器（注入由 service 層建立的服務）
func NewLineController(lineService *linebotsvc.LineBotService) *LineController {
	return &LineController{lineService: lineService}
}

// Webhook 處理 LINE Platform 送來的 Webhook POST
func (lc *LineController) Webhook(c *gin.Context) {
	events, err := lc.lineService.ParseRequest(c.Request)
	if err != nil {
		if err == linebot.ErrInvalidSignature {
			log.Printf("Webhook 簽章驗證失敗")
			response.New(c).Fail(400, "Bad Request").Send()
			return
		}
		log.Printf("解析 Webhook 失敗: %v", err)
		response.New(c).Fail(500, "Internal Server Error").Send()
		return
	}

	lc.lineService.HandleEvents(events)
	response.New(c).Success("OK").Send()
}
