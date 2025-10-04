package cache

import (
	"context"
	"encoding/json"
	"fmt"

	"recipe-generator/internal/core/ai"
	"recipe-generator/internal/infrastructure/config"

	"github.com/go-redis/redis/v8"
)

// Service 緩存服務
type Service struct {
	client *redis.Client
	config *config.CacheConfig
}

// NewService 創建緩存服務
func NewService(cfg *config.CacheConfig) (*Service, error) {
	if !cfg.Enabled {
		return &Service{config: cfg}, nil
	}

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// 測試連接
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Service{
		client: client,
		config: cfg,
	}, nil
}

// Get 獲取緩存
func (s *Service) Get(ctx context.Context, prompt string, imageData string) (*ai.Response, error) {
	if !s.config.Enabled || s.client == nil {
		return nil, fmt.Errorf("cache is disabled")
	}

	// 生成緩存鍵
	key := s.generateKey(prompt, imageData)

	// 獲取緩存
	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("cache miss")
		}
		return nil, fmt.Errorf("failed to get cache: %w", err)
	}

	// 解析緩存
	var resp ai.Response
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache: %w", err)
	}

	resp.CacheHit = true
	return &resp, nil
}

// Set 設置緩存
func (s *Service) Set(ctx context.Context, prompt string, imageData string, resp *ai.Response) error {
	if !s.config.Enabled || s.client == nil {
		return nil
	}

	// 生成緩存鍵
	key := s.generateKey(prompt, imageData)

	// 序列化響應
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	// 設置緩存
	if err := s.client.Set(ctx, key, data, s.config.TTL).Err(); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

// generateKey 生成緩存鍵
func (s *Service) generateKey(prompt string, imageData string) string {
	return fmt.Sprintf("ai:response:%s:%s", prompt, imageData)
}
