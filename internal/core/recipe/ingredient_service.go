package recipe

import (
	"context"
	"fmt"
	"strings"

	"recipe-generator/internal/core/ai/cache"
	"recipe-generator/internal/core/ai/image"
	"recipe-generator/internal/core/ai/service"
	"recipe-generator/internal/pkg/common"

	"go.uber.org/zap"
)

// IngredientService 食材識別服務
type IngredientService struct {
	aiService    *service.Service
	cacheManager *cache.CacheManager
	imageService *image.Processor
}

// NewIngredientService 創建新的食材識別服務
func NewIngredientService(aiService *service.Service, cacheManager *cache.CacheManager, imageService *image.Processor) *IngredientService {
	return &IngredientService{
		aiService:    aiService,
		cacheManager: cacheManager,
		imageService: imageService,
	}
}

// IdentifyIngredient 識別圖片中的食材和設備
func (s *IngredientService) IdentifyIngredient(ctx context.Context, imageData string) (*common.IngredientRecognitionResult, error) {
	// 驗證圖片
	if imageData == "" {
		return nil, fmt.Errorf("invalid image: image data is empty")
	}

	// 處理圖片
	processedImage, err := s.imageService.FormatImageData(imageData)
	if err != nil {
		return nil, fmt.Errorf("failed to process image: %w", err)
	}

	// 構建提示
	prompt := `請仔細分析圖片中的食材和設備，並提供詳細的識別結果(並且用繁體中文回答）(不需要考慮可讀性，請省略所有空格和換行，返回最緊湊的 JSON 格式)。
		要求：
		1. 只識別圖片中實際可見的食材和設備
		2. 不要添加圖片中未出現的物品
		3. 根據圖片內容判斷數量、單位和處理方式
		4. 如果無法確定某個屬性，請使用 "未知" 而不是猜測
		5. 所有欄位必須使用雙引號
		6. 不要使用預設值或猜測值
		7. 不要使用\n，不需要換行
		8. 不需要考慮可讀性，請省略所有空格和換行，返回最緊湊的 JSON 格式
		請以以下 JSON 格式返回：
		{
			"ingredients": [
				{
					"name": "食材名稱",
					"type": "食材類型",
					"amount": "數量",
					"unit": "單位",
					"preparation": "處理方式"
				}
			],
			"equipment": [
				{
					"name": "設備名稱",
					"type": "設備類型",
					"size": "尺寸",
					"material": "材質",
					"power_source": "能源類型"
				}
			],
			"summary": "辨識內容摘要，方便使用者核對確認"
		}`

	// 發送請求到 AI 服務
	response, err := s.aiService.ProcessRequest(ctx, prompt, processedImage)
	if err != nil {
		return nil, fmt.Errorf("failed to process request: %w", err)
	}

	// 解析響應
	content := response.Content
	content = strings.TrimSpace(content)
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start != -1 && end != -1 && end > start {
		content = content[start : end+1]
	}
	var result common.IngredientRecognitionResult
	if err := common.ParseJSON(content, &result); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	// 檢查並補充空值
	if result.Summary == "" {
		result.Summary = "無摘要"
	}

	// 檢查並補充食材資訊
	for i := range result.Ingredients {
		if result.Ingredients[i].Name == "" {
			result.Ingredients[i].Name = "未知食材"
		}
		if result.Ingredients[i].Type == "" {
			result.Ingredients[i].Type = "未知類型"
		}
		if result.Ingredients[i].Amount == "" {
			result.Ingredients[i].Amount = "適量"
		}
		if result.Ingredients[i].Unit == "" {
			result.Ingredients[i].Unit = "份"
		}
		if result.Ingredients[i].Preparation == "" {
			result.Ingredients[i].Preparation = "無特殊處理"
		}
	}

	// 檢查並補充設備資訊
	for i := range result.Equipment {
		if result.Equipment[i].Name == "" {
			result.Equipment[i].Name = "未知設備"
		}
		if result.Equipment[i].Type == "" {
			result.Equipment[i].Type = "未知類型"
		}
		if result.Equipment[i].Size == "" {
			result.Equipment[i].Size = "標準"
		}
		if result.Equipment[i].Material == "" {
			result.Equipment[i].Material = "未知"
		}
		if result.Equipment[i].PowerSource == "" {
			result.Equipment[i].PowerSource = "未知"
		}
	}

	// 記錄成功信息，但不包含詳細內容
	common.LogInfo("Successfully identified ingredients",
		zap.Int("ingredients_count", len(result.Ingredients)),
		zap.Int("equipment_count", len(result.Equipment)))

	return &result, nil
}

func (s *IngredientService) IdentifyIngredients(ctx context.Context, imageData string, descriptionHint string) (*common.IngredientRecognitionResult, error) {
	// 構建提示詞
	prompt := fmt.Sprintf(`請分析圖片中的食材和設備，並以 JSON 格式返回結果。格式如下：
{
    "ingredients": [
        {
            "name": "食材名稱",
            "type": "食材類型",
            "amount": "數量",
            "unit": "單位",
            "preparation": "處理方式"
        }
    ],
    "equipment": [
        {
            "name": "設備名稱",
            "type": "設備類型",
            "size": "尺寸",
            "material": "材質",
            "power_source": "電源類型"
        }
    ],
    "summary": "識別內容摘要"
}

%s`, descriptionHint)

	// 調用 AI 服務
	response, err := s.aiService.ProcessRequest(ctx, prompt, imageData)
	if err != nil {
		return nil, fmt.Errorf("failed to process request: %w", err)
	}

	// 解析響應
	var result common.IngredientRecognitionResult
	if err := common.ParseJSON(response.Content, &result); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	// 記錄成功信息，但不包含詳細內容
	common.LogInfo("Successfully identified ingredients",
		zap.Int("ingredients_count", len(result.Ingredients)),
		zap.Int("equipment_count", len(result.Equipment)))

	return &result, nil
}
