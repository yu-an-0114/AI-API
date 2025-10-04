package provider

import (
	"context"
	"time"
)

// Message 表示與 AI 模型的對話消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Request 表示發送到 AI 提供者的請求
type Request struct {
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Stop        []string  `json:"stop,omitempty"`
}

// Response 表示從 AI 提供者收到的響應
type Response struct {
	Content string `json:"content"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Provider 定義 AI 提供者介面
type Provider interface {
	// Generate 生成 AI 響應
	Generate(ctx context.Context, req *Request) (*Response, error)

	// GetModel 獲取當前使用的模型名稱
	GetModel() string

	// GetTimeout 獲取請求超時時間
	GetTimeout() time.Duration

	// Close 關閉提供者連接
	Close() error
}

// Config 定義 AI 提供者配置
type Config struct {
	APIKey     string
	Model      string
	Timeout    time.Duration
	MaxRetries int
	BaseURL    string
}
