package imageai

import (
	"sync"
	"time"
)

const contextTTL = 10 * time.Minute

// UserImageContext 紀錄使用者上一則成功辨識的圖片，供儲存觸發使用。
type UserImageContext struct {
	UserID     string
	ContentID  string
	ReplyToken string
	ExpiresAt  int64
}

var (
	mu      sync.RWMutex
	contextMap = make(map[string]*UserImageContext)
)

// Set 儲存使用者成功辨識圖片的 context，效期 10 分鐘。
func Set(userID, contentID, replyToken string) {
	if userID == "" || contentID == "" {
		return
	}
	expiresAt := time.Now().Add(contextTTL).Unix()
	mu.Lock()
	defer mu.Unlock()
	contextMap[userID] = &UserImageContext{
		UserID:     userID,
		ContentID:  contentID,
		ReplyToken: replyToken,
		ExpiresAt:  expiresAt,
	}
}

// Get 取得使用者上一則成功辨識的 context，若已過期則清除並回傳 nil。
func Get(userID string) *UserImageContext {
	if userID == "" {
		return nil
	}
	now := time.Now().Unix()
	mu.Lock()
	defer mu.Unlock()
	ctx, ok := contextMap[userID]
	if !ok || ctx == nil {
		return nil
	}
	if ctx.ExpiresAt < now {
		delete(contextMap, userID)
		return nil
	}
	return ctx
}
