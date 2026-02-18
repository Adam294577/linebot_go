package linebot

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"project/services/imageai"
	logsvc "project/services/log"
	"project/services/s3"

	"github.com/line/line-bot-sdk-go/v7/linebot"
)

// LineBotService 封裝 LINE Bot 客戶端與事件處理邏輯
type LineBotService struct {
	bot         *linebot.Client
	s3Uploader  *s3.Uploader
}

// NewLineBotService 建立 LINE Bot 服務（直接傳入憑證與 S3 Uploader）
func NewLineBotService(channelSecret, channelToken string, s3Uploader *s3.Uploader) (*LineBotService, error) {
	bot, err := linebot.New(channelSecret, channelToken)
	if err != nil {
		return nil, err
	}
	return &LineBotService{bot: bot, s3Uploader: s3Uploader}, nil
}

// NewLineBotServiceFromEnv 從環境變數建立 LINE Bot 服務（含 S3 Uploader）
func NewLineBotServiceFromEnv() (*LineBotService, error) {
	channelSecret := os.Getenv("LINE_CHANNEL_SECRET")
	channelToken := os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
	if channelSecret == "" || channelToken == "" {
		return nil, errors.New("LINE_CHANNEL_SECRET 與 LINE_CHANNEL_ACCESS_TOKEN 必須設定")
	}
	var s3Uploader *s3.Uploader
	if u, err := s3.NewUploaderFromEnv(); err == nil {
		s3Uploader = u
	}
	return NewLineBotService(channelSecret, channelToken, s3Uploader)
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
		s.handleTextMessage(event, message)
	case *linebot.ImageMessage:
		s.handleImageMessage(event, message)
	// case *linebot.StickerMessage:
	// 	s.handleStickerMessage(event, message)
	default:
		log.Printf("未處理的訊息類型: %T", event.Message)
		replyText(s.bot, event.ReplyToken, "請上傳食物圖片，我會幫你辨識圖片中的食物。")
	}
}

// handleTextMessage 處理文字訊息：儲存關鍵字觸發上傳；否則引導上傳圖片。
func (s *LineBotService) handleTextMessage(event *linebot.Event, message *linebot.TextMessage) {
	userID := event.Source.UserID
	if userID == "" {
		userID = "unknown"
	}
	log.Printf("收到訊息: %s (來自: %s)", message.Text, userID)

	text := message.Text
	if strings.Contains(strings.ToLower(text), "save") || strings.Contains(text, "儲存") {
		s.handleSaveImage(event, userID)
		return
	}

	replyText(s.bot, event.ReplyToken, "請上傳食物圖片，我會幫你辨識圖片中的食物。")
}

// handleSaveImage 處理儲存指令：若 context 有上一則成功辨識的圖片則上傳 S3，否則引導先上傳。
func (s *LineBotService) handleSaveImage(event *linebot.Event, userID string) {
	imgCtx := imageai.Get(userID)
	if imgCtx == nil {
		replyText(s.bot, event.ReplyToken, "請先上傳食物圖片再儲存")
		return
	}
	if s.s3Uploader == nil {
		replyText(s.bot, event.ReplyToken, "上傳失敗（S3 未設定）")
		return
	}

	contentResp, err := s.bot.GetMessageContent(imgCtx.ContentID).Do()
	if err != nil {
		logsvc.Error("上傳失敗 userID=%s 取得圖片失敗 err=%s", userID, err.Error())
		replyText(s.bot, event.ReplyToken, "上傳失敗")
		return
	}
	defer contentResp.Content.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	contentType := contentResp.ContentType
	if contentType == "" {
		contentType = "image/jpeg"
	}

	_, err = s.s3Uploader.Upload(ctx, userID, contentResp.Content, contentType)
	if err != nil {
		logsvc.Error("上傳失敗 userID=%s S3上傳失敗 err=%s", userID, err.Error())
		replyText(s.bot, event.ReplyToken, "上傳失敗")
		return
	}

	logsvc.Info("上傳成功 userID=%s", userID)
	replyText(s.bot, event.ReplyToken, "上傳成功")
}

// handleImageMessage 處理圖片訊息：下載、縮放、辨識食物、回覆，成功時寫入 context。
func (s *LineBotService) handleImageMessage(event *linebot.Event, message *linebot.ImageMessage) {
	userID := event.Source.UserID
	if userID == "" {
		userID = "unknown"
	}

	contentResp, err := s.bot.GetMessageContent(message.ID).Do()
	if err != nil {
		logsvc.Error("辨識失敗 userID=%s 取得圖片失敗 err=%s", userID, err.Error())
		replyText(s.bot, event.ReplyToken, "無法取得圖片，請再試一次")
		return
	}
	defer contentResp.Content.Close()

	imgBytes, err := io.ReadAll(contentResp.Content)
	if err != nil {
		logsvc.Error("辨識失敗 userID=%s 讀取圖片失敗 err=%s", userID, err.Error())
		replyText(s.bot, event.ReplyToken, "無法取得圖片，請再試一次")
		return
	}

	resized, _, err := imageai.Resize(bytes.NewReader(imgBytes))
	if err != nil {
		logsvc.Error("辨識失敗 userID=%s 圖片縮放失敗 err=%s", userID, err.Error())
		replyText(s.bot, event.ReplyToken, "圖片格式有誤，請重傳")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	foods, success, err := imageai.RecognizeFoodFromBytes(ctx, resized)
	if err != nil {
		logsvc.Error("辨識失敗 userID=%s API辨識失敗 err=%s", userID, err.Error())
		replyText(s.bot, event.ReplyToken, "辨識失敗，請稍後再試")
		return
	}
	if !success || foods == "" {
		foods = "無法辨識圖片中的食物"
	}

	if _, err := s.bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(foods)).Do(); err != nil {
		logsvc.Error("辨識失敗 userID=%s 回覆訊息失敗 err=%s", userID, err.Error())
		return
	}

	if success && foods != "無法辨識圖片中的食物" {
		logsvc.Info("辨識成功 userID=%s", userID)
		imageai.Set(userID, message.ID, event.ReplyToken)
	}
}

func replyText(bot *linebot.Client, token, text string) {
	if _, err := bot.ReplyMessage(token, linebot.NewTextMessage(text)).Do(); err != nil {
		log.Printf("回覆訊息失敗: %v", err)
	}
}
