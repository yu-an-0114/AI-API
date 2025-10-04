package image

import (
	"errors"
)

// Processor 圖片處理器
type Processor struct {
	maxSize int
}

// NewProcessor 創建圖片處理器
func NewProcessor(maxSize int) *Processor {
	return &Processor{
		maxSize: maxSize,
	}
}

// Compress 壓縮圖片
func (p *Processor) Compress(imageData string) (string, error) {
	if imageData == "" {
		return "", errors.New("image data is empty")
	}
	return imageData, nil
}

// FormatImageData 格式化圖片數據
func (p *Processor) FormatImageData(imageData string) (string, error) {
	if imageData == "" {
		return "", errors.New("image data is empty")
	}
	return imageData, nil
}
