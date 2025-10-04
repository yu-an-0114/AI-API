package recipe

import (
	"context"
	"fmt"
	"strings"

	"recipe-generator/internal/core/ai/cache"
	"recipe-generator/internal/core/ai/openrouter"
	"recipe-generator/internal/core/ai/service"
)

// Service 食譜服務基礎結構
type Service struct {
	aiService    *service.Service
	cacheManager *cache.CacheManager
}

// NewService 創建新的食譜服務
func NewService(aiService *service.Service, cacheManager *cache.CacheManager) *Service {
	return &Service{
		aiService:    aiService,
		cacheManager: cacheManager,
	}
}

// handleAIResponse 處理 AI 回應
func (s *Service) handleAIResponse(resp *openrouter.Response, err error) (string, error) {
	if err != nil {
		return "", fmt.Errorf("AI service error: %w", err)
	}

	if resp == nil || len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("empty AI response")
	}

	contentText := resp.Choices[0].Message.Content
	return strings.TrimSpace(contentText), nil
}

// getCacheKey 生成緩存鍵
func (s *Service) getCacheKey(prefix string, data string) string {
	return fmt.Sprintf("%s:%s", prefix, data)
}

// getFromCache 從緩存獲取數據
func (s *Service) getFromCache(ctx context.Context, key string) (string, error) {
	if s.cacheManager == nil {
		return "", nil
	}
	return s.cacheManager.Get(ctx, "recipe", key)
}

// setToCache 將數據存入緩存
func (s *Service) setToCache(ctx context.Context, key string, value string) error {
	if s.cacheManager == nil {
		return nil
	}
	return s.cacheManager.Set(ctx, "recipe", key, value)
}
