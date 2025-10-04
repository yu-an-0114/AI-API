package health

import (
	"net/http"
	"runtime"
	"time"

	"recipe-generator/internal/infrastructure/config"
	"recipe-generator/internal/pkg/common"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HealthResponse 健康檢查響應
type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version"`
	Runtime   map[string]interface{} `json:"runtime"`
	Queue     *QueueStatus           `json:"queue,omitempty"`
}

// QueueStatus 隊列狀態
type QueueStatus struct {
	QueueLength    int `json:"queue_length"`
	ProcessedCount int `json:"processed_count"`
	MaxQueueSize   int `json:"max_queue_size"`
	Workers        int `json:"workers"`
}

// HealthCheck 健康檢查處理器
func HealthCheck(c *gin.Context) {
	// 獲取配置
	cfg, exists := c.Get("config")
	if !exists {
		common.LogError("Configuration not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Configuration not found",
		})
		return
	}
	config, ok := cfg.(*config.Config)
	if !ok {
		common.LogError("Invalid configuration type in context")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid configuration type",
		})
		return
	}

	// 獲取 AI 服務
	aiSvc, exists := c.Get("ai_service")
	if !exists {
		common.LogError("AI service not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "AI service not found",
		})
		return
	}
	_ = aiSvc // 若未使用，直接忽略

	// 獲取運行時信息
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 構建響應
	response := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
		Version:   config.App.Version,
		Runtime: map[string]interface{}{
			"goroutines": runtime.NumGoroutine(),
			"memory": map[string]interface{}{
				"alloc":       m.Alloc,
				"total_alloc": m.TotalAlloc,
				"sys":         m.Sys,
				"num_gc":      m.NumGC,
			},
		},
	}

	// 如果 AI 服務可用，這裡可擴充隊列狀態（暫不實作）

	// 記錄請求
	common.LogInfo("Health check request",
		zap.String("client_ip", c.ClientIP()),
		zap.String("path", c.Request.URL.Path),
	)

	c.JSON(http.StatusOK, response)
}

// ReadinessCheck 就緒檢查處理器
func ReadinessCheck(c *gin.Context) {
	// TODO: 添加更多檢查
	// - 數據庫連接
	// - 外部服務依賴
	// - 隊列狀態
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}

// LivenessCheck 存活檢查處理器
func LivenessCheck(c *gin.Context) {
	// TODO: 添加更多檢查
	// - 內存使用
	// - CPU 使用
	// - 線程數
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
	})
}
