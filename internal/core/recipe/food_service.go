package recipe

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"recipe-generator/internal/core/ai/cache"
	"recipe-generator/internal/core/ai/service"
	"recipe-generator/internal/pkg/common"

	"go.uber.org/zap"
)

// FoodService 食物識別服務
type FoodService struct {
	aiService    *service.Service
	cacheManager *cache.CacheManager
}

// NewFoodService 創建新的食物識別服務
func NewFoodService(aiService *service.Service, cacheManager *cache.CacheManager) *FoodService {
	return &FoodService{
		aiService:    aiService,
		cacheManager: cacheManager,
	}
}

// saveRequestData 保存請求數據到文件
func saveRequestData(prompt string, imageData string) error {
	// 創建 logs 目錄（如果不存在）
	logsDir := "logs/ai_requests"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// 生成時間戳
	timestamp := time.Now().Format("20060102_150405")

	// 創建請求數據結構
	requestData := struct {
		Timestamp string `json:"timestamp"`
		Prompt    string `json:"prompt"`
		ImageData string `json:"image_data"`
	}{
		Timestamp: timestamp,
		Prompt:    prompt,
		ImageData: imageData,
	}

	// 將數據轉換為 JSON
	jsonData, err := json.MarshalIndent(requestData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal request data: %w", err)
	}

	// 生成文件名
	filename := filepath.Join(logsDir, fmt.Sprintf("food_recognition_%s.json", timestamp))

	// 寫入文件
	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write request data: %w", err)
	}

	common.LogInfo("已保存 AI 請求數據",
		zap.String("filename", filename),
		zap.Int("prompt_length", len(prompt)),
		zap.Int("image_length", len(imageData)))

	return nil
}

// IdentifyFood 識別圖片中的食物
func (s *FoodService) IdentifyFood(ctx context.Context, imageData string, descriptionHint string) (*common.FoodRecognitionResult, error) {
	// 記錄請求信息
	common.LogInfo("開始處理食物識別請求",
		zap.String("image_type", getImageType(imageData)),
		zap.String("description_hint", descriptionHint),
	)

	// 構建提示詞
	prompt := fmt.Sprintf(`請仔細分析圖片中的食物，並以 JSON 格式返回結果(並且用繁體中文回答）。要求：
1. 只識別圖片中實際可見的食物
2. 不要添加圖片中未出現的食物
3. 如果無法確定某個屬性，請使用 "未知" 而不是猜測
4. 所有欄位必須使用雙引號
5. 不要使用預設值或猜測值
6. 請確保識別結果與圖片內容完全相符
7. 如果圖片中沒有食物，請返回空列表
8. 不要使用\n，不需要換行
9. 根據辨識到的食物給出推論後可能需要用到的食材與製作廚具
10. 不需要考慮可讀性，請省略所有空格和換行，返回最緊湊的 JSON 格式
11. 所有欄位都必須要有不能漏掉，如果不知道填什麼請留空 "" or null
請以以下 JSON 格式返回：
{
    "recognized_foods": [
        {
            "name": "食物名稱",
					"description": "此食物的特徵與可能料理方式說明",
            "possible_ingredients": [
                {
                    "name": "食材名稱",
                    "type": "食材類型"
                }
            ],
            "possible_equipment": [
                {
                    "name": "設備名稱",
                    "type": "設備類型"
                }
            ]
        }
    ]
}
%s`, descriptionHint)

	// 保存請求數據
	// if err := saveRequestData(prompt, imageData); err != nil {
	// 	common.LogError("保存請求數據失敗",
	// 		zap.Error(err),
	// 		zap.String("image_type", getImageType(imageData)))
	// 	// 繼續處理，不中斷請求
	// }

	// 調用 AI 服務
	response, err := s.aiService.ProcessRequest(ctx, prompt, imageData)
	if err != nil {
		common.LogError("AI 服務請求失敗",
			zap.Error(err),
		)
		return nil, err
	}

	// 解析響應
	content := response.Content
	content = strings.TrimSpace(content)
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start != -1 && end != -1 && end > start {
		content = content[start : end+1]
	}
	var result common.FoodRecognitionResult
	if err := common.ParseJSON(content, &result); err != nil {
		common.LogError("AI 響應解析失敗",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	// 檢查並補充空值
	for i := range result.RecognizedFoods {
		if result.RecognizedFoods[i].Name == "" {
			result.RecognizedFoods[i].Name = "未知食物"
		}
		if result.RecognizedFoods[i].Description == "" {
			result.RecognizedFoods[i].Description = "無描述"
		}

		// 檢查並補充可能的食材
		for j := range result.RecognizedFoods[i].PossibleIngredients {
			if result.RecognizedFoods[i].PossibleIngredients[j].Name == "" {
				result.RecognizedFoods[i].PossibleIngredients[j].Name = "未知食材"
			}
			if result.RecognizedFoods[i].PossibleIngredients[j].Type == "" {
				result.RecognizedFoods[i].PossibleIngredients[j].Type = "未知類型"
			}
		}

		// 檢查並補充可能的設備
		for j := range result.RecognizedFoods[i].PossibleEquipment {
			if result.RecognizedFoods[i].PossibleEquipment[j].Name == "" {
				result.RecognizedFoods[i].PossibleEquipment[j].Name = "未知設備"
			}
			if result.RecognizedFoods[i].PossibleEquipment[j].Type == "" {
				result.RecognizedFoods[i].PossibleEquipment[j].Type = "未知類型"
			}
		}
	}

	// 記錄成功信息
	common.LogInfo("食物識別成功",
		zap.Int("foods_count", len(result.RecognizedFoods)),
		zap.String("image_type", getImageType(imageData)),
	)

	return &result, nil
}

// getImageType 獲取圖片類型
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
