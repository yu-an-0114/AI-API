package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// ParseJSON 解析 JSON 字符串到結構體
func ParseJSON(data string, v interface{}) error {
	return decodeJSON(strings.NewReader(data), v, false)
}

// ParseJSONStrict 解析 JSON 字符串到結構體（禁止未知欄位）
func ParseJSONStrict(data string, v interface{}) error {
	return decodeJSON(strings.NewReader(data), v, true)
}

// ParseJSONBytes 解析 JSON 位元組切片到結構體
func ParseJSONBytes(data []byte, v interface{}) error {
	return decodeJSON(bytes.NewReader(data), v, false)
}

// ParseJSONBytesStrict 解析 JSON 位元組切片到結構體（禁止未知欄位）
func ParseJSONBytesStrict(data []byte, v interface{}) error {
	return decodeJSON(bytes.NewReader(data), v, true)
}

// DecodeJSON 使用統一設定解析 JSON
func DecodeJSON(r io.Reader, v interface{}) error {
	return decodeJSON(r, v, false)
}

// DecodeJSONStrict 使用統一設定解析 JSON，禁止未知欄位
func DecodeJSONStrict(r io.Reader, v interface{}) error {
	return decodeJSON(r, v, true)
}

func decodeJSON(r io.Reader, v interface{}, disallowUnknown bool) error {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	if disallowUnknown {
		dec.DisallowUnknownFields()
	}

	if err := dec.Decode(v); err != nil {
		return err
	}

	// 確保沒有多餘資料
	for {
		t, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		// 若讀到額外 token，視為錯誤
		if t != nil {
			return fmt.Errorf("unexpected extra JSON data")
		}
	}
}

var unquotedKeyPattern = regexp.MustCompile(`([{\[,]\s*)([A-Za-z_][A-Za-z0-9_]*)\s*:`)

// QuoteJSONKeys 將未加雙引號的鍵補上雙引號
func QuoteJSONKeys(raw string) string {
	return unquotedKeyPattern.ReplaceAllString(raw, `$1"$2":`)
}

// ToJSON 將結構體轉換為 JSON 字符串
func ToJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// StringSliceToString 將字符串切片轉換為逗號分隔的字符串
func StringSliceToString(slice []string) string {
	if len(slice) == 0 {
		return ""
	}
	return strings.Join(slice, "、")
}

// IngredientSliceToString 將食材切片轉換為格式化的字符串
func IngredientSliceToString(ingredients []Ingredient) string {
	if len(ingredients) == 0 {
		return ""
	}

	var parts []string
	for _, ing := range ingredients {
		part := fmt.Sprintf("%s(%s)", ing.Name, ing.Type)
		if ing.Amount != "" {
			part += fmt.Sprintf(" %s%s", ing.Amount, ing.Unit)
		}
		if ing.Preparation != "" {
			part += fmt.Sprintf("，%s", ing.Preparation)
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "、")
}

// EquipmentSliceToString 將設備切片轉換為格式化的字符串
func EquipmentSliceToString(equipment []Equipment) string {
	if len(equipment) == 0 {
		return ""
	}

	var parts []string
	for _, eq := range equipment {
		part := fmt.Sprintf("%s(%s)", eq.Name, eq.Type)
		if eq.Size != "" {
			part += fmt.Sprintf("，%s", eq.Size)
		}
		if eq.Material != "" {
			part += fmt.Sprintf("，%s", eq.Material)
		}
		if eq.PowerSource != "" {
			part += fmt.Sprintf("，%s", eq.PowerSource)
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "、")
}

// AIRequest AI 請求結構
type AIRequest struct {
	Prompt    string `json:"prompt"`
	ImageData string `json:"image_data,omitempty"`
	Model     string `json:"model"`
}

// AIResponse AI 響應結構
type AIResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Message 消息結構
type Message struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

// Content 內容結構
type Content struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL 圖片 URL 結構
type ImageURL struct {
	URL string `json:"url"`
}
