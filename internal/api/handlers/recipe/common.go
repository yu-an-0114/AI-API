package recipe

import (
	"encoding/base64"
	"strings"
)

// getImageType 獲取圖片類型（用於日誌記錄）
func getImageType(image string) string {
	if image == "" {
		return "empty"
	}
	if strings.HasPrefix(image, "http://") || strings.HasPrefix(image, "https://") {
		return "url"
	}
	if strings.HasPrefix(image, "data:image/") {
		parts := strings.Split(image, ";base64,")
		if len(parts) == 2 {
			return "base64_data_uri_" + strings.TrimPrefix(parts[0], "data:image/")
		}
		return "invalid_data_uri"
	}
	if _, err := base64.StdEncoding.DecodeString(image); err == nil {
		return "base64"
	}
	return "unknown_format"
}

// getImagePrefix 獲取圖片前綴（用於日誌記錄）
func getImagePrefix(image string) string {
	if strings.HasPrefix(image, "data:image/") {
		return "[IMAGE_DATA]"
	}
	if strings.HasPrefix(image, "http") {
		return "[IMAGE_URL]"
	}
	if strings.HasPrefix(image, "/9j/") || strings.HasPrefix(image, "iVBORw0KGgo") {
		return "[BASE64_DATA]"
	}
	return "[UNKNOWN_FORMAT]"
}
