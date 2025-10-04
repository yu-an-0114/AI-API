package common

import (
	"net/http"
)

// ErrorResponse 定義 API 錯誤響應結構
type ErrorResponse struct {
	Code    string `json:"code"`              // 錯誤代碼
	Message string `json:"message"`           // 錯誤信息
	Details string `json:"details,omitempty"` // 詳細信息（僅在開發模式顯示）
}

// CustomError 定義自定義錯誤類型
type CustomError struct {
	Code    string // 錯誤代碼
	Message string // 錯誤信息
	Err     error  // 原始錯誤
	Status  int    // HTTP 狀態碼
}

func (e *CustomError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

// NewError 創建新的自定義錯誤
func NewError(code string, message string, status int, err error) *CustomError {
	return &CustomError{
		Code:    code,
		Message: message,
		Status:  status,
		Err:     err,
	}
}

// ValidationError 表示驗證錯誤
type ValidationError struct {
	message string
}

// Error 實現 error 介面
func (e *ValidationError) Error() string {
	return e.message
}

// NewValidationError 創建新的驗證錯誤
func NewValidationError(message string) error {
	return &ValidationError{
		message: message,
	}
}

// IsValidationError 檢查是否為驗證錯誤
func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}

// 預定義錯誤代碼
const (
	// 客戶端錯誤 (4xx)
	ErrCodeInvalidRequest   = "INVALID_REQUEST"    // 400
	ErrCodeUnauthorized     = "UNAUTHORIZED"       // 401
	ErrCodeForbidden        = "FORBIDDEN"          // 403
	ErrCodeNotFound         = "NOT_FOUND"          // 404
	ErrCodeMethodNotAllowed = "METHOD_NOT_ALLOWED" // 405
	ErrCodeRequestTimeout   = "REQUEST_TIMEOUT"    // 408
	ErrCodeConflict         = "CONFLICT"           // 409
	ErrCodeTooManyRequests  = "TOO_MANY_REQUESTS"  // 429

	// 服務器錯誤 (5xx)
	ErrCodeInternalError      = "INTERNAL_ERROR"      // 500
	ErrCodeNotImplemented     = "NOT_IMPLEMENTED"     // 501
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE" // 503
	ErrCodeGatewayTimeout     = "GATEWAY_TIMEOUT"     // 504
)

// 預定義錯誤
var (
	// 客戶端錯誤
	ErrInvalidRequest   = NewError(ErrCodeInvalidRequest, "無效的請求", http.StatusBadRequest, nil)
	ErrUnauthorized     = NewError(ErrCodeUnauthorized, "未授權的訪問", http.StatusUnauthorized, nil)
	ErrForbidden        = NewError(ErrCodeForbidden, "禁止訪問", http.StatusForbidden, nil)
	ErrNotFound         = NewError(ErrCodeNotFound, "資源不存在", http.StatusNotFound, nil)
	ErrMethodNotAllowed = NewError(ErrCodeMethodNotAllowed, "不支持的請求方法", http.StatusMethodNotAllowed, nil)
	ErrRequestTimeout   = NewError(ErrCodeRequestTimeout, "請求超時", http.StatusRequestTimeout, nil)
	ErrConflict         = NewError(ErrCodeConflict, "資源衝突", http.StatusConflict, nil)
	ErrTooManyRequests  = NewError(ErrCodeTooManyRequests, "請求過於頻繁", http.StatusTooManyRequests, nil)

	// 服務器錯誤
	ErrInternalError      = NewError(ErrCodeInternalError, "服務器內部錯誤", http.StatusInternalServerError, nil)
	ErrNotImplemented     = NewError(ErrCodeNotImplemented, "功能未實現", http.StatusNotImplemented, nil)
	ErrServiceUnavailable = NewError(ErrCodeServiceUnavailable, "服務暫時不可用", http.StatusServiceUnavailable, nil)
	ErrGatewayTimeout     = NewError(ErrCodeGatewayTimeout, "網關超時", http.StatusGatewayTimeout, nil)

	// 業務錯誤
	ErrInvalidImageFormat = NewError("INVALID_IMAGE_FORMAT", "無效的圖片格式", http.StatusBadRequest, nil)
	ErrInvalidImageSize   = NewError("INVALID_IMAGE_SIZE", "圖片大小超出限制", http.StatusBadRequest, nil)
	ErrInvalidImageType   = NewError("INVALID_IMAGE_TYPE", "不支持的圖片類型", http.StatusBadRequest, nil)
	ErrCacheFull          = NewError("CACHE_FULL", "緩存已滿", http.StatusServiceUnavailable, nil)
	ErrCacheDisabled      = NewError("CACHE_DISABLED", "緩存已禁用", http.StatusServiceUnavailable, nil)
	ErrAIServiceError     = NewError("AI_SERVICE_ERROR", "AI 服務錯誤", http.StatusServiceUnavailable, nil)
)
