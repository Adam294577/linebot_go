package service

import (
	"log"
	"net/http"

	"github.com/line/line-bot-sdk-go/v7/linebot"
)

// LineBotService 封裝 LINE Bot 客戶端與事件處理邏輯
type LineBotService struct {
	bot *linebot.Client
}

// NewLineBotService 建立 LINE Bot 服務
func NewLineBotService(channelSecret, channelToken string) (*LineBotService, error) {
	bot, err := linebot.New(channelSecret, channelToken)
	if err != nil {
		return nil, err
	}
	return &LineBotService{bot: bot}, nil
}

// ParseRequest 解析 Webhook 請求並驗證簽章，回傳事件列表
func (s *LineBotService) ParseRequest(req *http.Request) ([]*linebot.Event, error) {
	return s.bot.ParseRequest(req)
}

// HandleEvents 處理一組 Webhook 事件（可擴充不同事件類型）
func (s *LineBotService) HandleEvents(events []*linebot.Event) {
	for _, event := range events {
		go s.handleEvent(event)
	}
}

func (s *LineBotService) handleEvent(event *linebot.Event) {
	switch event.Type {
	case linebot.EventTypeMessage:
		s.handleMessage(event)
	// 可在此擴充其他事件類型
	// case linebot.EventTypeFollow:
	// 	s.handleFollow(event)
	// case linebot.EventTypeUnfollow:
	// 	s.handleUnfollow(event)
	default:
		log.Printf("未處理的事件類型: %s", event.Type)
	}
}

func (s *LineBotService) handleMessage(event *linebot.Event) {
	switch message := event.Message.(type) {
	case *linebot.TextMessage:
		userID := event.Source.UserID
		if userID == "" {
			userID = "unknown"
		}
		log.Printf("收到訊息: %s (來自: %s)", message.Text, userID)

		if _, err := s.bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("你說: "+message.Text)).Do(); err != nil {
			log.Printf("回覆訊息失敗: %v", err)
		}
	// 可在此擴充其他訊息類型
	// case *linebot.ImageMessage:
	// 	s.handleImageMessage(event, message)
	// case *linebot.StickerMessage:
	// 	s.handleStickerMessage(event, message)
	default:
		log.Printf("未處理的訊息類型: %T", event.Message)
	}
}
