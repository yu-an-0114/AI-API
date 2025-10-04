package middleware

import (
	"time"

	"recipe-generator/internal/pkg/common"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger 日誌中間件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 開始時間
		start := time.Now()
		path := c.Request.URL.Path
		requestID := c.GetHeader("X-Request-ID")

		// 處理請求
		c.Next()

		// 結束時間
		end := time.Now()
		latency := end.Sub(start)

		// 獲取狀態碼
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		userAgent := c.Request.UserAgent()

		// 構建基本日誌字段
		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("ip", clientIP),
			zap.String("user-agent", userAgent),
			zap.Duration("latency", latency),
			zap.String("request_id", requestID),
		}

		// 添加錯誤信息（如果有）
		if len(c.Errors) > 0 {
			fields = append(fields, zap.Strings("errors", c.Errors.Errors()))
		}

		// 根據狀態碼記錄不同級別的日誌
		switch {
		case status >= 500:
			common.LogError("伺服器錯誤",
				append(fields, zap.String("error_type", "server_error"))...,
			)
		case status >= 400:
			common.LogWarn("用戶端錯誤",
				append(fields, zap.String("error_type", "client_error"))...,
			)
		case status >= 300:
			common.LogInfo("重新導向",
				append(fields, zap.String("error_type", "redirect"))...,
			)
		default:
			common.LogInfo("請求完成",
				fields...,
			)
		}
	}
}

// Recovery 恢復中間件
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 記錄基本錯誤信息
				common.LogError("Panic recovered",
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				)

				// 返回簡單的 500 錯誤
				c.AbortWithStatusJSON(500, gin.H{
					"error": "Internal server error",
					"code":  "INTERNAL_ERROR",
				})
			}
		}()

		c.Next()
	}
}
