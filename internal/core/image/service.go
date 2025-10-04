package image

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"strings"
	"time"

	_ "image/gif" // 支援 GIF

	_ "golang.org/x/image/webp" // 支援 WebP
)

// Service 圖片處理服務
type Service struct {
	maxSizeBytes int64
	httpClient   *http.Client
}

// NewService 創建新的圖片處理服務
func NewService(maxSizeBytes int64) *Service {
	return &Service{
		maxSizeBytes: maxSizeBytes,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ProcessImage 處理圖片
func (s *Service) ProcessImage(imageData string) (string, error) {
	// 檢查是否為 URL
	if strings.HasPrefix(imageData, "http://") || strings.HasPrefix(imageData, "https://") {
		// 下載圖片
		resp, err := s.httpClient.Get(imageData)
		if err != nil {
			return "", fmt.Errorf("failed to download image: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("failed to download image: status code %d", resp.StatusCode)
		}

		// 讀取圖片數據
		imageBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read image data: %w", err)
		}

		// 檢查文件大小
		if int64(len(imageBytes)) > s.maxSizeBytes {
			return "", fmt.Errorf("image size exceeds maximum limit of %d bytes", s.maxSizeBytes)
		}

		// 解碼圖片
		img, format, err := image.Decode(bytes.NewReader(imageBytes))
		if err != nil {
			return "", fmt.Errorf("failed to decode image: %w", err)
		}

		// 檢查圖片格式
		if !isSupportedFormat(format) {
			return "", fmt.Errorf("unsupported image format: %s", format)
		}

		// 將圖片轉換為 JPEG 格式
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
			return "", fmt.Errorf("failed to encode image as JPEG: %w", err)
		}

		// 重新編碼為 base64
		encodedData := base64.StdEncoding.EncodeToString(buf.Bytes())
		return fmt.Sprintf("data:image/jpeg;base64,%s", encodedData), nil
	}

	// 處理 base64 格式
	if !strings.HasPrefix(imageData, "data:image/") {
		return "", fmt.Errorf("invalid image data format")
	}

	// 解析 base64 數據
	parts := strings.Split(imageData, ",")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid base64 data format")
	}

	// 解碼 base64 數據
	decodedData, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 data: %w", err)
	}

	// 檢查文件大小
	if int64(len(decodedData)) > s.maxSizeBytes {
		return "", fmt.Errorf("image size exceeds maximum limit of %d bytes", s.maxSizeBytes)
	}

	// 解碼圖片
	img, format, err := image.Decode(bytes.NewReader(decodedData))
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	// 檢查圖片格式
	if !isSupportedFormat(format) {
		return "", fmt.Errorf("unsupported image format: %s", format)
	}

	// 將圖片轉換為 JPEG 格式
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return "", fmt.Errorf("failed to encode image as JPEG: %w", err)
	}

	// 重新編碼為 base64
	encodedData := base64.StdEncoding.EncodeToString(buf.Bytes())
	return fmt.Sprintf("data:image/jpeg;base64,%s", encodedData), nil
}

// isSupportedFormat 檢查圖片格式是否支援
func isSupportedFormat(format string) bool {
	supportedFormats := map[string]bool{
		"jpeg": true,
		"jpg":  true,
		"png":  true,
		"gif":  true,
		"webp": true,
	}
	return supportedFormats[format]
}

// ValidateImage 驗證圖片
func (s *Service) ValidateImage(imageData string) error {
	// 檢查是否為 URL
	if strings.HasPrefix(imageData, "http://") || strings.HasPrefix(imageData, "https://") {
		// 下載圖片
		resp, err := s.httpClient.Get(imageData)
		if err != nil {
			return fmt.Errorf("failed to download image: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to download image: status code %d", resp.StatusCode)
		}

		// 讀取圖片數據
		imageBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read image data: %w", err)
		}

		// 檢查文件大小
		if int64(len(imageBytes)) > s.maxSizeBytes {
			return fmt.Errorf("image size exceeds maximum limit of %d bytes", s.maxSizeBytes)
		}

		// 解碼圖片
		_, format, err := image.Decode(bytes.NewReader(imageBytes))
		if err != nil {
			return fmt.Errorf("failed to decode image: %w", err)
		}

		// 檢查圖片格式
		if !isSupportedFormat(format) {
			return fmt.Errorf("unsupported image format: %s", format)
		}

		return nil
	}

	// 處理 base64 格式
	if !strings.HasPrefix(imageData, "data:image/") {
		return fmt.Errorf("invalid image data format")
	}

	// 解析 base64 數據
	parts := strings.Split(imageData, ",")
	if len(parts) != 2 {
		return fmt.Errorf("invalid base64 data format")
	}

	// 解碼 base64 數據
	decodedData, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("failed to decode base64 data: %w", err)
	}

	// 檢查文件大小
	if int64(len(decodedData)) > s.maxSizeBytes {
		return fmt.Errorf("image size exceeds maximum limit of %d bytes", s.maxSizeBytes)
	}

	// 解碼圖片
	_, format, err := image.Decode(bytes.NewReader(decodedData))
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// 檢查圖片格式
	if !isSupportedFormat(format) {
		return fmt.Errorf("unsupported image format: %s", format)
	}

	return nil
}
