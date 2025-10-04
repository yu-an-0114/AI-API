package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"recipe-generator/internal/core/ai/image"
	"recipe-generator/internal/infrastructure/config"
	"recipe-generator/internal/pkg/common"

	"go.uber.org/zap"
)

const (
	baseURL = "https://openrouter.ai/api/v1"
)

// Client OpenRouter API 客戶端
type Client struct {
	httpClient     *http.Client
	config         *config.Config
	imageProcessor *image.Processor
}

// Message 消息結構
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// TextContent 文本內容
type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ImageContent 圖片內容
type ImageContent struct {
	Type     string `json:"type"`
	ImageURL string `json:"image_url"`
}

// Request 表示 API 請求
type Request struct {
	Messages         []Message       `json:"messages"`
	Model            string          `json:"model,omitempty"`
	MaxTokens        int             `json:"max_tokens,omitempty"`
	Temperature      float64         `json:"temperature,omitempty"`
	TopP             float64         `json:"top_p,omitempty"`
	TopK             int             `json:"top_k,omitempty"`
	PresencePenalty  float64         `json:"presence_penalty,omitempty"`
	FrequencyPenalty float64         `json:"frequency_penalty,omitempty"`
	Stream           bool            `json:"stream,omitempty"`
	Provider         *ProviderConfig `json:"provider,omitempty"`
}

// ProviderConfig 表示供應商配置
type ProviderConfig struct {
	Only           []string `json:"only,omitempty"`
	Ignore         []string `json:"ignore,omitempty"`
	Order          []string `json:"order,omitempty"`
	DataCollection string   `json:"data_collection,omitempty"`
}

// Response OpenRouter 響應結構
type Response struct {
	ID       string    `json:"id"`
	Choices  []Choice  `json:"choices"`
	Usage    UsageInfo `json:"usage"`
	CacheHit bool      `json:"cache_hit,omitempty"`
}

// Choice 選擇結構
type Choice struct {
	Message Message `json:"message"`
}

// UsageInfo 使用量信息
type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Error 表示 API 錯誤
type Error struct {
	Error struct {
		Message string      `json:"message"`
		Type    string      `json:"type"`
		Code    interface{} `json:"code"`
	} `json:"error"`
}

// NewClient 創建新的 OpenRouter 客戶端
func NewClient(cfg *config.Config) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		config:         cfg,
		imageProcessor: image.NewProcessor(1024), // 最大尺寸 1024 像素
	}
}

// sanitizeResponse 清理響應內容，移除所有圖片數據
func sanitizeResponse(body []byte) string {
	// 如果是圖片數據，直接返回提示信息
	if strings.Contains(string(body), "data:image/") {
		return "[IMAGE_DATA_REMOVED]"
	}

	// 如果是 base64 數據，直接返回提示信息
	if len(body) > 100 && strings.Contains(string(body), "base64") {
		return "[BASE64_DATA_REMOVED]"
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		// 如果不是 JSON，檢查是否包含 base64 數據
		if strings.Contains(string(body), "base64") {
			return "[BASE64_DATA_REMOVED]"
		}
		return string(body)
	}

	// 清理 messages 中的圖片數據
	if messages, ok := raw["messages"].([]interface{}); ok {
		for i, msg := range messages {
			if msgMap, ok := msg.(map[string]interface{}); ok {
				if content, ok := msgMap["content"].([]interface{}); ok {
					for j, c := range content {
						if cMap, ok := c.(map[string]interface{}); ok {
							if cMap["type"] == "image_url" || strings.Contains(fmt.Sprintf("%v", cMap), "image") {
								content[j] = map[string]interface{}{
									"type":      "image_url",
									"image_url": "[IMAGE_DATA_REMOVED]",
								}
							}
						}
					}
					msgMap["content"] = content
				}
			}
			messages[i] = msg
		}
		raw["messages"] = messages
	}

	// 清理其他可能包含圖片的字段
	for k, v := range raw {
		if str, ok := v.(string); ok {
			if strings.Contains(str, "data:image/") || strings.Contains(str, "base64") {
				raw[k] = "[IMAGE_DATA_REMOVED]"
			}
		}
	}

	// 重新編碼為 JSON
	sanitized, err := json.Marshal(raw)
	if err != nil {
		return "[JSON_PARSING_ERROR]"
	}

	// 最後檢查一次，確保沒有遺漏的圖片數據
	result := string(sanitized)
	if strings.Contains(result, "data:image/") || strings.Contains(result, "base64") {
		return "[IMAGE_DATA_REMOVED]"
	}

	return result
}

// Generate 生成回應
func (c *Client) Generate(ctx context.Context, prompt, imageData string) (*Response, error) {
	// 構建請求
	req := &Request{
		Model: c.config.OpenRouter.Model,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// 如果有圖片數據，添加到請求中
	if imageData != "" {
		req.Messages[0].Content = fmt.Sprintf("%s\n\n圖片：%s", prompt, imageData)
	}

	// 設置模型參數
	req.MaxTokens = 2048
	req.Temperature = 0.7
	req.TopP = 0.9
	req.TopK = 40
	req.PresencePenalty = 0.0
	req.FrequencyPenalty = 0.0

	// 檢查是否為視覺模型
	isVisionModel := strings.Contains(strings.ToLower(req.Model), "vision") || strings.Contains(strings.ToLower(req.Model), "vl")
	if isVisionModel {
		req.MaxTokens = 4096
		req.Temperature = 0.7
		req.Stream = false
	}

	// 準備請求體
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 創建 HTTP 請求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 設置請求頭
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.config.OpenRouter.APIKey)
	httpReq.Header.Set("HTTP-Referer", "https://recipe-generator.com")
	httpReq.Header.Set("X-Title", "Recipe Generator")

	// 發送請求
	common.LogInfo("Sending request to OpenRouter",
		zap.String("model", req.Model),
		zap.Int("messages", len(req.Messages)),
		zap.Bool("is_vision_model", isVisionModel),
	)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		common.LogError("Failed to send request to AI service",
			zap.Error(err),
			zap.String("model", req.Model),
		)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 讀取響應體
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		common.LogError("Failed to read response body",
			zap.Error(err),
			zap.String("model", req.Model),
		)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 清理響應內容（移除所有圖片數據）
	sanitizedBody := sanitizeResponse(body)

	// 檢查 HTTP 狀態碼
	if resp.StatusCode != http.StatusOK {
		common.LogError("AI service returned error status",
			zap.Int("status_code", resp.StatusCode),
			zap.String("model", req.Model),
			zap.String("response", sanitizedBody),
		)
		return nil, fmt.Errorf("AI service error (status %d): %s", resp.StatusCode, sanitizedBody)
	}

	// 解析響應
	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		common.LogError("Failed to parse AI service response",
			zap.Error(err),
			zap.String("model", req.Model),
			zap.String("response", sanitizedBody),
		)
		return nil, fmt.Errorf("failed to parse response: %w (response: %s)", err, sanitizedBody)
	}

	// 檢查響應內容
	if len(response.Choices) == 0 {
		common.LogError("Empty choices in AI service response",
			zap.String("model", req.Model),
			zap.String("response", sanitizedBody),
		)
		return nil, fmt.Errorf("empty choices in response (response: %s)", sanitizedBody)
	}

	// 檢查消息內容
	content := response.Choices[0].Message.Content
	if len(content) == 0 {
		common.LogError("Empty content in AI service response",
			zap.String("model", req.Model),
			zap.String("response", sanitizedBody),
		)
		return nil, fmt.Errorf("empty content in response (response: %s)", sanitizedBody)
	}

	// 記錄成功響應
	common.LogInfo("Successfully generated response from AI service",
		zap.String("model", req.Model),
		zap.Int("content_length", len(content)),
	)

	return &response, nil
}

// Close 關閉客戶端
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}
