package handlers

import (
	"net/http"

	"recipe-generator/internal/core/ai/service"
	"recipe-generator/internal/pkg/common"

	"github.com/gin-gonic/gin"
)

// AIHandler AI 處理器
type AIHandler struct {
	aiService *service.Service
}

// NewAIHandler 創建 AI 處理器
func NewAIHandler(aiService *service.Service) *AIHandler {
	return &AIHandler{
		aiService: aiService,
	}
}

// GenerateRecipe 生成食譜
func (h *AIHandler) GenerateRecipe(c *gin.Context) {
	// 獲取請求參數
	var req struct {
		Prompt      string  `json:"prompt" binding:"required"`
		ImageData   string  `json:"image_data"`
		Model       string  `json:"model"`
		MaxTokens   int     `json:"max_tokens"`
		Temperature float64 `json:"temperature"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": common.ErrInvalidRequest.Error(),
		})
		return
	}

	// 生成回應
	response, err := h.aiService.ProcessRequest(c.Request.Context(), req.Prompt, req.ImageData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 根據實際需求解析 response.Content
	// 這裡假設回傳內容為 JSON 字串，直接轉為 map 回傳
	var result map[string]interface{}
	if err := common.ParseJSON(response.Content, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI response parse error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"response": result,
	})
}
