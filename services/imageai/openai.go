package imageai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	defaultModel   = "gpt-4o-mini"
	promptTemplate = `請辨識這張圖片中的食物，回覆格式規定如下：
- 若有食物：只回傳食物名稱，以頓號或逗號分隔，例如「白飯、炒蛋、青菜」。
- 若無食物：只回傳「無食物」。
不要加任何說明、標點以外的多餘文字。`
)

// RecognizeFood 使用 OpenAI Responses API 辨識圖片中的食物。
// base64Image 為 JPEG base64 編碼，回傳精簡食物文字與是否成功。
func RecognizeFood(ctx context.Context, base64Image string) (foods string, success bool, err error) {
	token := os.Getenv("OPEN_AI_TOKEN")
	if token == "" {
		return "", false, fmt.Errorf("OPEN_AI_TOKEN 未設定")
	}
	model := os.Getenv("OPENAI_IMAGE_MODEL")
	if model == "" {
		model = defaultModel
	}

	// Responses API 格式：input 為 user message，content 含 input_text 與 input_image
	body := map[string]any{
		"model": model,
		"input": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "input_text", "text": promptTemplate},
					{
						"type":      "input_image",
						"image_url": fmt.Sprintf("data:image/jpeg;base64,%s", base64Image),
					},
				},
			},
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		return "", false, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/responses", bytes.NewReader(data))
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("OpenAI API 回傳 %d", resp.StatusCode)
	}

	var result struct {
		OutputText string            `json:"output_text"`
		Output     []json.RawMessage `json:"output"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", false, err
	}
	content := result.OutputText
	if content == "" && len(result.Output) > 0 {
		content = extractTextFromOutput(result.Output)
	}
	return content, content != "", nil
}

// extractTextFromOutput 從 Responses API 的 output 陣列取出文字（content 可能為 string 或 array）。
func extractTextFromOutput(output []json.RawMessage) string {
	for _, raw := range output {
		var item struct {
			Type    string          `json:"type"`
			Text    string          `json:"text,omitempty"`
			Content json.RawMessage `json:"content,omitempty"`
		}
		if json.Unmarshal(raw, &item) != nil {
			continue
		}
		if item.Text != "" {
			return item.Text
		}
		if item.Type != "message" || len(item.Content) == 0 {
			continue
		}
		// content 可能是 string 或 array of {type, text}
		var s string
		if json.Unmarshal(item.Content, &s) == nil {
			return s
		}
		var parts []struct {
			Type string `json:"type"`
			Text string `json:"text,omitempty"`
		}
		if json.Unmarshal(item.Content, &parts) == nil {
			for _, p := range parts {
				if (p.Type == "text" || p.Type == "output_text") && p.Text != "" {
					return p.Text
				}
			}
		}
	}
	return ""
}

// RecognizeFoodFromBytes 從 JPEG 位元組辨識食物，內部轉 base64 後呼叫 RecognizeFood。
func RecognizeFoodFromBytes(ctx context.Context, imgBytes []byte) (foods string, success bool, err error) {
	b64 := base64.StdEncoding.EncodeToString(imgBytes)
	return RecognizeFood(ctx, b64)
}
