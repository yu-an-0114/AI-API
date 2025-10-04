package common

import (
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Logger 全局日誌實例
	Logger  *zap.Logger
	LogMode string // 只宣告，不初始化

	// 定義日誌級別的顏色
	levelColors = map[zapcore.Level]string{
		zapcore.DebugLevel: "\033[36m", // 青色
		zapcore.InfoLevel:  "\033[32m", // 綠色
		zapcore.WarnLevel:  "\033[33m", // 黃色
		zapcore.ErrorLevel: "\033[31m", // 紅色
		zapcore.FatalLevel: "\033[35m", // 紫色
	}
	resetColor = "\033[0m"
)

// 自定義編碼器配置
func getEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "", // 移除 logger 名稱
		CallerKey:      "", // 移除調用者信息
		MessageKey:     "msg",
		StacktraceKey:  "", // 移除堆棧跟踪
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    customLevelEncoder,
		EncodeTime:     customTimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   nil, // 移除調用者編碼器
	}
}

// 自定義時間格式
func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("15:04:05.000")) // 添加毫秒級別的時間戳
}

// 自定義級別編碼器（添加顏色）
func customLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	color := levelColors[l]
	level := l.String()
	// 統一級別顯示長度
	switch l {
	case zapcore.DebugLevel:
		level = "DBG"
	case zapcore.InfoLevel:
		level = "INF"
	case zapcore.WarnLevel:
		level = "WRN"
	case zapcore.ErrorLevel:
		level = "ERR"
	case zapcore.FatalLevel:
		level = "FAT"
	}
	enc.AppendString(color + level + resetColor)
}

// InitLogger 初始化日誌系統
func InitLogger(logLevel string) error {
	// 設置日誌級別
	var level zapcore.Level
	switch strings.ToLower(logLevel) {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	case "fatal":
		level = zapcore.FatalLevel
	default:
		level = zapcore.InfoLevel
	}

	// 讀取 LOG_MODE（必須在 .env 載入後）
	LogMode = os.Getenv("LOG_MODE")

	// 創建日誌目錄
	if err := os.MkdirAll("logs", 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// 創建日誌文件
	logFile, err := os.OpenFile("logs/app.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// 創建多個輸出目標
	fileWriter := zapcore.AddSync(logFile)
	consoleWriter := zapcore.AddSync(os.Stdout)

	// 創建多個核心
	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(getEncoderConfig()),
		fileWriter,
		level,
	)
	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(getEncoderConfig()),
		consoleWriter,
		level,
	)

	// 合併多個核心
	core := zapcore.NewTee(fileCore, consoleCore)

	// 創建 logger，移除一些默認字段
	Logger = zap.New(core,
		zap.AddCallerSkip(1),
		zap.Fields(
			zap.String("service", "recipe-generator"),
		),
	)

	// 替換全局 logger
	zap.ReplaceGlobals(Logger)

	return nil
}

// LogInfo 記錄信息日誌
func LogInfo(msg string, fields ...zap.Field) {
	if LogMode == "concise" {
		// 只允許 API middleware logger.go 的 "請求完成" log 以及伺服器啟動/關閉訊息輸出
		if msg != "請求完成" && msg != "啟動應用" && msg != "Server exited" && msg != "Shutting down server..." {
			return
		}
	}
	// 過濾掉包含圖片數據的字段
	filteredFields := make([]zap.Field, 0, len(fields))
	for _, field := range fields {
		if field.Key == "image" || strings.Contains(field.Key, "image_data") || strings.Contains(field.Key, "base64") {
			continue
		}
		filteredFields = append(filteredFields, field)
	}
	Logger.Info(msg, filteredFields...)
}

// LogError 記錄錯誤日誌
func LogError(msg string, fields ...zap.Field) {
	// 過濾掉包含圖片數據的字段
	filteredFields := make([]zap.Field, 0, len(fields))
	for _, field := range fields {
		if field.Key == "image" || strings.Contains(field.Key, "image_data") || strings.Contains(field.Key, "base64") {
			continue
		}
		filteredFields = append(filteredFields, field)
	}
	Logger.Error(msg, filteredFields...)
}

// LogWarn 記錄警告日誌
func LogWarn(msg string, fields ...zap.Field) {
	// 過濾掉包含圖片數據的字段
	filteredFields := make([]zap.Field, 0, len(fields))
	for _, field := range fields {
		if field.Key == "image" || strings.Contains(field.Key, "image_data") || strings.Contains(field.Key, "base64") {
			continue
		}
		filteredFields = append(filteredFields, field)
	}
	Logger.Warn(msg, filteredFields...)
}

// LogDebug 記錄調試日誌
func LogDebug(msg string, fields ...zap.Field) {
	// 過濾掉包含圖片數據的字段
	filteredFields := make([]zap.Field, 0, len(fields))
	for _, field := range fields {
		if field.Key == "image" || strings.Contains(field.Key, "image_data") || strings.Contains(field.Key, "base64") {
			continue
		}
		filteredFields = append(filteredFields, field)
	}
	Logger.Debug(msg, filteredFields...)
}

// LogFatal 記錄致命錯誤日誌
func LogFatal(msg string, fields ...zap.Field) {
	Logger.Fatal(msg, fields...)
}

// Sync 同步日誌緩衝
func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}

// LogCacheHit 記錄快取命中
func LogCacheHit(cacheType, key string) {
	LogInfo("快取命中", zap.String("類型", cacheType))
}

// LogCacheMiss 記錄快取未命中
func LogCacheMiss(cacheType, key string) {
	LogInfo("快取未命中", zap.String("類型", cacheType))
}

// LogAICall 記錄 AI 調用
func LogAICall(prompt string, duration time.Duration, err error, requestID string) {
	if err != nil {
		LogError("AI 請求失敗",
			zap.Error(err),
			zap.Duration("耗時", duration),
		)
		return
	}
	LogInfo("AI 請求成功",
		zap.Duration("耗時", duration),
	)
}

// LogImageProcessing 記錄圖片處理相關的日誌
func LogImageProcessing(level string, msg string, fields ...zap.Field) {
	// 過濾掉包含圖片數據的字段
	filteredFields := make([]zap.Field, 0, len(fields))
	for _, field := range fields {
		if field.Key == "image" ||
			strings.Contains(field.Key, "image_data") ||
			strings.Contains(field.Key, "base64") ||
			field.Key == "has_image" {
			continue
		}
		filteredFields = append(filteredFields, field)
	}

	// 根據日誌級別記錄
	switch level {
	case "info":
		LogInfo("圖片處理資訊", filteredFields...)
	case "error":
		LogError("圖片處理失敗", filteredFields...)
	case "warn":
		LogWarn("圖片處理警告", filteredFields...)
	default:
		LogInfo("圖片處理資訊", filteredFields...)
	}
}
