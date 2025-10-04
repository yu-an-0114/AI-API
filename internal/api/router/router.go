package router

import (
	"net/http"

	"recipe-generator/internal/pkg/common"

	"go.uber.org/zap"
)

// responseWriter 響應記錄器
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader 實現 http.ResponseWriter 介面
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// errorHandler 錯誤處理中間件
func errorHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				common.LogError("Server panic recovered",
					zap.Any("error", err),
					zap.String("path", r.URL.Path),
					zap.String("method", r.Method),
				)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		// 創建響應記錄器
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// 處理請求
		next.ServeHTTP(rw, r)
	})
}
