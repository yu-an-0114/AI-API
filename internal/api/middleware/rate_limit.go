package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"recipe-generator/internal/pkg/common"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RateLimiter 限流器結構
type RateLimiter struct {
	mu       sync.Mutex
	tokens   int
	capacity int
	rate     float64
	lastTime time.Time
}

// NewRateLimiter 創建新的限流器
func NewRateLimiter(requests int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:   requests,
		capacity: requests,
		rate:     float64(requests) / window.Seconds(),
		lastTime: time.Now(),
	}
}

// Allow 檢查是否允許請求
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastTime).Seconds()
	rl.lastTime = now

	// 添加新令牌
	newTokens := int(elapsed * rl.rate)
	if newTokens > 0 {
		rl.tokens = min(rl.capacity, rl.tokens+newTokens)
	}

	// 檢查是否有可用令牌
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

// RateLimit 限流中間件
func RateLimit(requests int, window time.Duration) gin.HandlerFunc {
	limiter := NewRateLimiter(requests, window)

	return func(c *gin.Context) {
		if !limiter.Allow() {
			common.LogInfo("Rate limit exceeded",
				zap.String("ip", c.ClientIP()),
				zap.String("path", c.Request.URL.Path),
			)

			c.Header("Retry-After", fmt.Sprintf("%d", int(window.Seconds())))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Too many requests",
				"retry_after": window.Seconds(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// min 返回兩個整數中的較小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
