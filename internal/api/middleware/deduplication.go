package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"recipe-generator/internal/infrastructure/config"
	"recipe-generator/internal/pkg/common"
)

var (
	// 請求緩存，用於去重
	requestCache = struct {
		sync.RWMutex
		requests map[string]time.Time
	}{
		requests: make(map[string]time.Time),
	}

	// 啟動自動清理 goroutine（只啟動一次）
	cleanupOnce sync.Once
)

// 啟動自動清理 goroutine
func startDeduplicationCleanup(cfg *config.Config) {
	cleanupOnce.Do(func() {
		interval := 10 * time.Minute
		window := 1 * time.Second
		if cfg != nil && cfg.DedupWindow > 0 {
			window = cfg.DedupWindow
		}
		go func() {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for range ticker.C {
				now := time.Now()
				requestCache.Lock()
				for k, t := range requestCache.requests {
					if now.Sub(t) > 10*window {
						delete(requestCache.requests, k)
					}
				}
				requestCache.Unlock()
			}
		}()
	})
}

// Deduplication 請求去重中間件，支援從 config 取得 dedupWindow
func Deduplication(cfg *config.Config) gin.HandlerFunc {
	startDeduplicationCleanup(cfg)
	return func(c *gin.Context) {
		dedupWindow := 1 * time.Second
		if cfg != nil && cfg.DedupWindow > 0 {
			dedupWindow = cfg.DedupWindow
		}

		// 只處理 POST 請求
		if c.Request.Method != "POST" {
			c.Next()
			return
		}

		// 計算請求體哈希
		bodyHash := ""
		if c.Request.Body != nil {
			// 讀取請求體
			body, err := io.ReadAll(c.Request.Body)
			if err != nil {
				common.LogError("Failed to read request body", zap.Error(err))
				c.Next()
				return
			}

			// 計算哈希
			hash := sha256.Sum256(body)
			bodyHash = hex.EncodeToString(hash[:])

			// 恢復請求體
			c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		}

		// 生成請求指紋
		fingerprint := c.Request.Method + ":" + c.Request.URL.Path
		if bodyHash != "" {
			fingerprint += ":" + bodyHash
		}

		// 檢查是否是重複請求
		now := time.Now()
		requestCache.RLock()
		if lastTime, exists := requestCache.requests[fingerprint]; exists {
			if now.Sub(lastTime) <= dedupWindow {
				requestCache.RUnlock()
				c.JSON(429, gin.H{
					"error": "Request too frequent",
					"code":  "TOO_MANY_REQUESTS",
				})
				c.Abort()
				return
			}
		}
		requestCache.RUnlock()

		// 記錄請求
		requestCache.Lock()
		requestCache.requests[fingerprint] = now
		requestCache.Unlock()

		c.Next()
	}
}
