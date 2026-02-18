package controllers

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	response "project/services/responses"
	"project/services/s3"
)

var (
	s3Controller   *S3Controller
	s3ControllerOnce sync.Once
)

func initS3Controller() {
	uploader, err := s3.NewUploaderFromEnv()
	if err != nil {
		return
	}
	s3Controller = NewS3Controller(uploader)
}

// S3Controller 處理 S3 相關 API
type S3Controller struct {
	uploader *s3.Uploader
}

// NewS3Controller 建立 S3 控制器
func NewS3Controller(uploader *s3.Uploader) *S3Controller {
	return &S3Controller{uploader: uploader}
}

// S3GetImageHandler 供 route 註冊用：觸發時才從環境變數建立 S3，未設定則回 503
func S3GetImageHandler(c *gin.Context) {
	s3ControllerOnce.Do(initS3Controller)
	if s3Controller == nil {
		response.New(c).Fail(http.StatusServiceUnavailable, "S3 未設定").Send()
		return
	}
	s3Controller.GetImage(c)
}

// GetImageReq 取得圖片 Presigned URL 的請求
// 建議 DB 至少存 s3_key，查詢時帶入即可
type GetImageReq struct {
	// S3 物件完整 key，格式：food-images/{userID}/{timestamp}.jpg
	// 上傳成功時 Upload() 回傳的值，存進 DB 後查詢用
	S3Key string `json:"s3_key" binding:"required"`
}

// GetImage 回傳 S3 圖片的 Presigned URL
// POST /s3/getImage
// Body: {"s3_key": "food-images/U80b35e04529b5a8be1fc2b4545240e7d/20260218_111336.jpg"}
func (sc *S3Controller) GetImage(c *gin.Context) {
	var req GetImageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.New(c).Fail(http.StatusBadRequest, "s3_key 必填").Send()
		return
	}

	url, err := sc.uploader.PresignGetURL(c.Request.Context(), req.S3Key, 1*time.Hour)
	if err != nil {
		response.New(c).Fail(http.StatusInternalServerError, "產生圖片連結失敗").Send()
		return
	}

	response.New(c).Success("OK").SetData(gin.H{"url": url}).Send()
}
