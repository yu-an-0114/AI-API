package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"recipe-generator/internal/pkg/common"
)

// BodySizeLimit 限制請求體大小的中間件
func BodySizeLimit(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 檢查 Content-Length
		if c.Request.ContentLength > maxSize {
			common.LogError("Request body too large",
				zap.Int64("content_length", c.Request.ContentLength),
				zap.Int64("max_size", maxSize),
				zap.String("client_ip", c.ClientIP()),
				zap.String("path", c.Request.URL.Path),
			)
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":    "Request body too large",
				"max_size": maxSize,
			})
			return
		}

		// 設置請求體大小限制
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)

		c.Next()
	}
}
