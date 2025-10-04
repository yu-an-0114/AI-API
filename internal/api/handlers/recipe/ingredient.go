package recipe

import (
	"encoding/json"
	"net/http"
	"strings"

	"recipe-generator/internal/core/ai/image"
	"recipe-generator/internal/core/recipe"
	"recipe-generator/internal/pkg/common"

	"go.uber.org/zap"
)

// IngredientRecognitionRequest 食材識別請求
type IngredientRecognitionRequest struct {
	Image           string `json:"image" binding:"required"`
	DescriptionHint string `json:"description_hint,omitempty"`
}

// IngredientRecognitionResponse 食材識別響應
type IngredientRecognitionResponse struct {
	Ingredients []Ingredient `json:"ingredients"`
	Equipment   []Equipment  `json:"equipment"`
	Summary     string       `json:"summary"`
}

// HandleIngredientRecognition 處理食材識別請求
func HandleIngredientRecognition(ingredientService *recipe.IngredientService, imageService *image.Processor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 生成請求 ID
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = common.GenerateUUID()
			w.Header().Set("X-Request-ID", requestID)
		}

		// 解析請求
		var req IngredientRecognitionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			common.LogError("Invalid request format",
				zap.Error(err),
				zap.String("request_id", requestID))
			common.WriteErrorResponse(w, http.StatusBadRequest, "Invalid request format")
			return
		}

		// 驗證圖片格式（加強）
		if req.Image == "" || !strings.HasPrefix(req.Image, "data:image/") {
			common.LogError("Invalid image format (handler)",
				zap.String("request_id", requestID),
				zap.String("image_type", getImageType(req.Image)),
				zap.Int("image_length", len(req.Image)),
			)
			common.WriteErrorResponse(w, http.StatusBadRequest, "Invalid image format")
			return
		}

		// 處理圖片
		processedImage, err := imageService.FormatImageData(req.Image)
		if err != nil {
			common.LogError("Image processing failed",
				zap.Error(err),
				zap.String("request_id", requestID))
			common.WriteErrorResponse(w, http.StatusInternalServerError, "Image processing failed")
			return
		}

		// 識別食材
		result, err := ingredientService.IdentifyIngredient(r.Context(), processedImage)
		if err != nil {
			// 根據錯誤訊息內容判斷是否屬於用戶端錯誤
			if strings.Contains(err.Error(), "image format") || strings.Contains(err.Error(), "base64") {
				common.LogError("Invalid image format (service)",
					zap.Error(err),
					zap.String("request_id", requestID))
				common.WriteErrorResponse(w, http.StatusBadRequest, "Invalid image format")
				return
			}
			// 其他錯誤回傳 500
			common.LogError("Failed to identify ingredients",
				zap.Error(err),
				zap.String("request_id", requestID))
			common.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to identify ingredients")
			return
		}

		// 構建響應
		response := IngredientRecognitionResponse{
			Ingredients: make([]Ingredient, len(result.Ingredients)),
			Equipment:   make([]Equipment, len(result.Equipment)),
			Summary:     result.Summary,
		}

		// 轉換食材信息
		for i, ing := range result.Ingredients {
			response.Ingredients[i] = Ingredient{
				Name:        ing.Name,
				Type:        ing.Type,
				Amount:      ing.Amount,
				Unit:        ing.Unit,
				Preparation: ing.Preparation,
			}
		}

		// 轉換設備信息
		for i, equip := range result.Equipment {
			response.Equipment[i] = Equipment{
				Name:        equip.Name,
				Type:        equip.Type,
				Size:        equip.Size,
				Material:    equip.Material,
				PowerSource: equip.PowerSource,
			}
		}

		// 返回響應
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			common.LogError("Failed to encode response",
				zap.Error(err),
				zap.String("request_id", requestID))
			common.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to encode response")
			return
		}

		// 記錄成功
		common.LogInfo("Successfully identified ingredients",
			zap.String("request_id", requestID),
			zap.Int("ingredients_count", len(result.Ingredients)),
			zap.Int("equipment_count", len(result.Equipment)))
	}
}
