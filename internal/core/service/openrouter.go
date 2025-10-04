package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"recipe-generator/internal/infrastructure/config"
	"recipe-generator/internal/pkg/common"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

// OpenRouterService OpenRouter 服務
type OpenRouterService struct {
	config *config.Config
	client *resty.Client
}

// NewOpenRouterService 創建 OpenRouter 服務
func NewOpenRouterService(cfg *config.Config) *OpenRouterService {
	client := resty.New().
		SetBaseURL("https://openrouter.ai/api/v1").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", cfg.OpenRouter.APIKey)).
		SetHeader("HTTP-Referer", "https://recipe-generator.com").
		SetHeader("X-Title", "Recipe Generator")

	return &OpenRouterService{
		config: cfg,
		client: client,
	}
}

// GenerateResponse 生成回應
func (s *OpenRouterService) GenerateResponse(ctx context.Context, prompt string, imageData string) (string, error) {
	// 簡化 prompt：去除多餘換行、前後空白、連續空白合併為一格
	simplePrompt := strings.TrimSpace(prompt)
	simplePrompt = strings.ReplaceAll(simplePrompt, "\n", "")
	simplePrompt = strings.Join(strings.Fields(simplePrompt), "")

	msgContent := []map[string]interface{}{
		{
			"type": "text",
			"text": simplePrompt,
		},
	}
	imageUrlDebug := ""
	if imageData != "" {
		url := imageData
		if !strings.HasPrefix(imageData, "data:image/") {
			url = fmt.Sprintf("data:image/jpeg;base64,%s", imageData)
		}
		msgContent = append(msgContent, map[string]interface{}{
			"type": "image_url",
			"image_url": map[string]string{
				"url": url,
			},
		})
		imageUrlDebug = url
	}
	// debug log image_url 前 60 字元與是否有 data:image/ 前綴
	if imageUrlDebug != "" {
		prefix := ""
		if strings.HasPrefix(imageUrlDebug, "data:image/") {
			prefix = "[HAS_PREFIX]"
		} else {
			prefix = "[NO_PREFIX]"
		}
		if len(imageUrlDebug) > 60 {
			common.LogDebug("OpenRouter image_url debug", zap.String("prefix", prefix), zap.String("image_url_start", imageUrlDebug[:60]))
		} else {
			common.LogDebug("OpenRouter image_url debug", zap.String("prefix", prefix), zap.String("image_url_start", imageUrlDebug))
		}
	}
	// 構建請求
	req := map[string]interface{}{
		"model": s.config.OpenRouter.Model,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": msgContent,
			},
		},
		"max_tokens": s.config.OpenRouter.MaxTokens,
	}

	// 發送請求
	resp, err := s.client.R().
		SetContext(ctx).
		SetBody(req).
		Post("/chat/completions")

	if err != nil {
		return "", fmt.Errorf("failed to send request to OpenRouter: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("OpenRouter API returned error: %s", resp.String())
	}

	// 解析回應
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return "", fmt.Errorf("failed to parse OpenRouter response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in OpenRouter response")
	}

	return result.Choices[0].Message.Content, nil
}
