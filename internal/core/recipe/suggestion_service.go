package recipe

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"recipe-generator/internal/core/ai/cache"
	"recipe-generator/internal/core/ai/service"
	"recipe-generator/internal/pkg/common"

	"go.uber.org/zap"
)

// SuggestionService 食譜推薦服務
type SuggestionService struct {
	aiService    *service.Service
	cacheManager *cache.CacheManager
	lastRecipes  sync.Map
}

// NewSuggestionService 創建新的食譜推薦服務
func NewSuggestionService(aiService *service.Service, cacheManager *cache.CacheManager) *SuggestionService {
	return &SuggestionService{
		aiService:    aiService,
		cacheManager: cacheManager,
	}
}

// ---------------- 寬鬆版中繼結構：忽略 ar_parameters 型別 ----------------

type looseRecipe struct {
	DishName        string              `json:"dish_name"`
	DishDescription string              `json:"dish_description"`
	Ingredients     []common.Ingredient `json:"ingredients"`
	Equipment       []common.Equipment  `json:"equipment"`
	Recipe          []looseStep         `json:"recipe"`
}

type looseAction struct {
	Action            string   `json:"action"`
	ToolRequired      string   `json:"tool_required"`
	MaterialRequired  []string `json:"material_required"`
	TimeMinutes       int      `json:"time_minutes"`
	InstructionDetail string   `json:"instruction_detail"`
}

type looseStep struct {
	StepNumber         int                     `json:"step_number"`
	ARtype             common.ARtype           `json:"ARtype,omitempty"`
	ARParameters       *common.ARActionParams  `json:"ar_parameters,omitempty"`
	Title              string                  `json:"title"`
	Description        string                  `json:"description"`
	Actions            []looseAction           `json:"actions"`
	EstimatedTotalTime string                  `json:"estimated_total_time"`
	Temperature        string                  `json:"temperature"`
	Warnings           string                  `json:"warnings"`
	Notes              string                  `json:"notes"`
}

// ---------------------------------------------------------------

// SuggestRecipes 根據可用食材和設備推薦食譜
func (s *SuggestionService) SuggestRecipes(ctx context.Context, req *common.RecipeByIngredientsRequest) (*common.Recipe, error) {
	// 驗證必要欄位
	cm := strings.TrimSpace(req.Preference.CookingMethod)
	if cm == "" {
		cm = "未指定"
	}
	ss := strings.TrimSpace(req.Preference.ServingSize)
	if ss == "" {
		ss = "未指定"
	}

	key := buildSuggestionKey(req)
	var previousRecipe string
	if key != "" {
		if val, ok := s.lastRecipes.Load(key); ok {
			if str, okCast := val.(string); okCast {
				previousRecipe = str
			}
		}
	}

	prompt := fmt.Sprintf(`請根據以下可用食材和設備，推薦適合的食譜(並且用繁體中文回答）。

可用食材：
%s

可用設備：
%s

烹飪偏好：
- 烹飪方式：%s
- 飲食限制：%s
- 份量：%s

要求：
1. 只根據提供的食材和設備推薦內容，不要添加未出現的食材或設備
2. 不要使用預設值或猜測值，若無法確定請填寫 "未知"
3. 每個步驟都要非常詳細，適合新手操作
4. 動作描述要具體明確，包含具體的時間和溫度
5. 注意事項要特別提醒新手容易忽略的細節
6. 所有字段都必須使用雙引號
7. 不需要考慮可讀性，請省略所有空格和換行，返回最緊湊的 JSON 格式
8. 推薦的食譜要優先使用已有的食材和設備
9. 如果某些食材或設備不足，可以建議替代方案
10. 每個食譜都要考慮到烹飪難度和時間
11. time_minutes 欄位必須是整數，不能有小數點（以秒為單位）
12. warnings 欄位必須是字串類型，如果沒有警告事項請填寫 null
13. 每個步驟都必須包含 warnings 欄位，不能省略此欄位
14. 不要使用\n，不需要換行
15. 所有欄位都必須要有不能漏掉，如果不知道填什麼請留空 "" or null
16. 只回傳一個獨立的json，不要回傳多個json
17. "ingredient":"ingredient" 不要直接寫 ingredient，如果是調味料或液體要使用具體英文小寫名稱，不得出現空白或非 ASCII 字元
18. 除了 ar_parameters 內部欄位維持英文，其餘所有欄位內容一律使用繁體中文描述
19. 每個步驟只能描述一個主要的烹飪動作，對應單一的 ARtype
20. 每個步驟只允許一個 action 物件，內容需與該 ARtype 完整對應
21. 每個步驟必須提供 ARtype 與 ar_parameters，且 ar_parameters.type 必須等於 ARtype
22. ar_parameters 欄位若無資料請填 null，ingredient 必須使用具體英文小寫名稱，不得使用 "ingredient"、"food" 等泛用詞
23. 所有設備名稱與 ar_parameters.container 只能使用提供的設備清單中可對應的英文容器名稱，不得新增其他設備或容器
24. 嚴格輸出單一 JSON 物件，不要額外輸出自然語言或程式碼區塊
請以以下 JSON 格式返回（僅作為範例，請勿直接複製內容）：
{
    "dish_name": "菜名",
    "dish_description": "描述",
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
    "recipe": [
        {
            "step_number": 1,
            "ARtype": "stir",
            "ar_parameters": {
                "type": "stir",
                "container": "pan",
                "ingredient": "egg",
                "color": null,
                "time": null,
                "temperature": null,
                "flameLevel": null
            },
            "title": "步驟標題",
            "description": "步驟描述",
            "actions": [{
                "action": "動作",
                "tool_required": "工具",
                "material_required": ["材料"],
                "time_minutes": 1,
                "instruction_detail": "細節"
            }],
            "estimated_total_time": "時間",
            "temperature": "火侯",
            "warnings": "警告事項",
            "notes": "備註"
        }
    ]
}`,
	common.FormatIngredients(req.AvailableIngredients),
	common.FormatEquipment(req.AvailableEquipment),
	cm,
	strings.Join(req.Preference.DietaryRestrictions, "、"),
	ss)

	if previousRecipe != "" {
		prompt += fmt.Sprintf("\n\n上一次生成的食譜 JSON：%s\n請務必提供全新的食譜，確保菜名、步驟描述或食材搭配與上述內容明顯不同，避免輸出與前一次相同或僅做微幅調整的內容。\n", previousRecipe)
	}
	uniqueToken := fmt.Sprintf("SessionToken:%d", time.Now().UnixNano())
	prompt += fmt.Sprintf("\n請忽略識別碼 %s，該識別碼僅用於避免快取，請勿在輸出中提到它。\n", uniqueToken)

	common.LogDebug("SuggestRecipes 組裝的 prompt", zap.String("prompt", prompt))

	resp, err := s.aiService.ProcessRequest(ctx, prompt, "")
	if err != nil {
		return nil, fmt.Errorf("AI service error: %w", err)
	}
	if resp == nil || resp.Content == "" {
		return nil, fmt.Errorf("empty AI response")
	}

	content := strings.TrimSpace(resp.Content)
	// 去除 markdown/fence：取第一個 { 到最後一個 }
	if start, end := strings.Index(content, "{"), strings.LastIndex(content, "}"); start != -1 && end != -1 && end > start {
		content = content[start : end+1]
	}

	// 先用「寬鬆版」解析，忽略 ar_parameters 內的型別雜訊
	var lr looseRecipe
	if err := common.ParseJSON(content, &lr); err != nil {
		fixed := common.QuoteJSONKeys(content)
		if fixed != content {
			if ferr := common.ParseJSON(fixed, &lr); ferr == nil {
				common.LogWarn("AI 回傳 JSON 含未加引號鍵，已於 suggestion 流程自動修正",
					zap.Int("ai_response_length", len(content)),
				)
				content = fixed
			} else {
				common.LogError("AI 回應解析失敗(loose)", zap.Error(err), zap.Int("ai_response_length", len(content)))
				return nil, fmt.Errorf("failed to parse AI response (loose): %w", err)
			}
		} else {
			common.LogError("AI 回應解析失敗(loose)", zap.Error(err), zap.Int("ai_response_length", len(content)))
			return nil, fmt.Errorf("failed to parse AI response (loose): %w", err)
		}
	}

	// 將寬鬆版轉成正式的 common.Recipe（包含 ARtype / ar_parameters，若缺漏會後續補齊）
	var result common.Recipe
	result.DishName = lr.DishName
	result.DishDescription = lr.DishDescription
	result.Ingredients = lr.Ingredients
	result.Equipment = lr.Equipment
	result.Recipe = make([]common.RecipeStep, len(lr.Recipe))

	for i, st := range lr.Recipe {
		// 先建立 step（不含 actions），避免引用未知型別
		result.Recipe[i] = common.RecipeStep{
			StepNumber:         st.StepNumber,
			Title:              st.Title,
			Description:        st.Description,
			EstimatedTotalTime: st.EstimatedTotalTime,
			Temperature:        st.Temperature,
			Warnings:           st.Warnings,
			Notes:              st.Notes,
			// Actions 下面以 JSON round-trip 指定到正確目標型別
		}

		// 用 JSON round-trip 把 looseAction 轉成「目標欄位的實際型別」
		if len(st.Actions) > 0 {
			if b, err := json.Marshal(st.Actions); err == nil {
				_ = common.ParseJSONBytes(b, &result.Recipe[i].Actions)
			}
		}

		result.Recipe[i].ARtype = st.ARtype
		if st.ARParameters != nil {
			cp := *st.ARParameters
			result.Recipe[i].ARParameters = &cp
		}
	}

	// 檢查並補充食譜整體
	if result.DishName == "" {
		result.DishName = "未知菜名"
	}
	if result.DishDescription == "" {
		result.DishDescription = "無描述"
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

	// 逐步處理：請 AI 產生 AR 參數（ar_parameters），後端僅驗證與回填
	for i := range result.Recipe {
		// 確保 step_number 正確
		result.Recipe[i].StepNumber = i + 1

		// 補齊與 AR 無關的基礎欄位
		if result.Recipe[i].Title == "" {
			result.Recipe[i].Title = fmt.Sprintf("步驟 %d", i+1)
		}
		if result.Recipe[i].Description == "" {
			result.Recipe[i].Description = "無描述"
		}
		if result.Recipe[i].EstimatedTotalTime == "" {
			result.Recipe[i].EstimatedTotalTime = "未知"
		}
		if result.Recipe[i].Temperature == "" || result.Recipe[i].Temperature == "null" {
			result.Recipe[i].Temperature = "中火"
		}
		if result.Recipe[i].Warnings == "" || result.Recipe[i].Warnings == "null" {
			result.Recipe[i].Warnings = "無"
		}
		if result.Recipe[i].Notes == "" || result.Recipe[i].Notes == "null" {
			result.Recipe[i].Notes = "無備註"
		}

		// 動作資訊補齊的邏輯會在批次產生 AR 參數後再執行
	}

	containerChoices := inferContainerChoices(result.Equipment)

	for i := range result.Recipe {
		params := result.Recipe[i].ARParameters
		if params != nil {
			if err := validateARParams(*params); err == nil {
				if result.Recipe[i].ARtype != "" && result.Recipe[i].ARtype != params.Type {
					common.LogWarn("ARtype 與 ar_parameters.type 不一致，已覆寫為 ar_parameters.type",
						zap.Int("step", i+1),
						zap.String("title", result.Recipe[i].Title),
						zap.String("ARtype", string(result.Recipe[i].ARtype)),
						zap.String("parameter_type", string(params.Type)),
					)
				}
				result.Recipe[i].ARtype = params.Type
				continue
			} else {
				common.LogWarn("AI 回傳的 AR 參數驗證失敗，使用回退邏輯",
					zap.Int("step", i+1),
					zap.String("title", result.Recipe[i].Title),
					zap.Error(err),
				)
			}
		} else {
			common.LogWarn("AI 未提供 AR 參數，使用回退邏輯",
				zap.Int("step", i+1),
				zap.String("title", result.Recipe[i].Title),
			)
		}

		fallback, ferr := fallbackARParams(result.Recipe[i], containerChoices, result.Ingredients)
		if ferr != nil {
			common.LogWarn("AR 參數回退失敗，採用預設值",
				zap.Int("step", result.Recipe[i].StepNumber),
				zap.String("title", result.Recipe[i].Title),
				zap.Error(ferr),
			)
			fallback = defaultARParams(containerChoices)
		}
		if fallback == nil {
			return nil, fmt.Errorf("ar_parameters missing for step %d (%s): model failed to produce valid AR JSON and default fallback unavailable", result.Recipe[i].StepNumber, result.Recipe[i].Title)
		}
		common.LogWarn("AR 參數使用回退結果",
			zap.Int("step", result.Recipe[i].StepNumber),
			zap.String("title", result.Recipe[i].Title),
			zap.String("fallback_type", string(fallback.Type)),
		)
		result.Recipe[i].ARtype = fallback.Type
		result.Recipe[i].ARParameters = fallback
	}

	for i := range result.Recipe {
		if len(result.Recipe[i].Actions) > 1 {
			common.LogWarn("偵測到多個 actions，僅保留第一個以符合單一步驟限制",
				zap.Int("step", result.Recipe[i].StepNumber),
				zap.Int("action_count", len(result.Recipe[i].Actions)),
			)
			result.Recipe[i].Actions = append([]common.RecipeAction(nil), result.Recipe[i].Actions[0])
		}

		for j := range result.Recipe[i].Actions {
			if result.Recipe[i].Actions[j].Action == "" {
				result.Recipe[i].Actions[j].Action = "無動作"
			}
			if result.Recipe[i].Actions[j].ToolRequired == "" || result.Recipe[i].Actions[j].ToolRequired == "null" {
				result.Recipe[i].Actions[j].ToolRequired = "無"
			}
			if result.Recipe[i].Actions[j].InstructionDetail == "" {
				result.Recipe[i].Actions[j].InstructionDetail = "無細節說明"
			}
			if result.Recipe[i].Actions[j].TimeMinutes <= 0 {
				result.Recipe[i].Actions[j].TimeMinutes = 1
			}
			if result.Recipe[i].Actions[j].MaterialRequired == nil {
				result.Recipe[i].Actions[j].MaterialRequired = []string{}
			}
		}
	}

	// 驗證必要欄位
	if len(result.Recipe) == 0 {
		return nil, fmt.Errorf("recipe steps cannot be empty")
	}

	if key != "" {
		if b, err := json.Marshal(result); err == nil {
			s.lastRecipes.Store(key, string(b))
		} else {
			common.LogWarn("無法緩存前次食譜以避免重複",
				zap.Error(err),
			)
		}
	}

	return &result, nil
}

// ===================== Helpers =====================

// 只提供容器候選給 AI 參考；不在後端代填
func inferContainerChoices(eqs []common.Equipment) []string {
	set := map[string]struct{}{}
	for _, eq := range eqs {
		name := eq.Name + eq.Type
		switch {
		case strings.Contains(name, "平底鍋"):
			set["pan"] = struct{}{}
		case strings.Contains(name, "炒鍋"), strings.Contains(name, "鑊"):
			set["wok"] = struct{}{}
		case strings.Contains(name, "鍋"):
			set["pot"] = struct{}{}
		case strings.Contains(name, "碗"):
			set["bowl"] = struct{}{}
		case strings.Contains(name, "盤"):
			set["plate"] = struct{}{}
		case strings.Contains(name, "杯"):
			set["cup"] = struct{}{}
		}
	}
	if len(set) == 0 {
		// 若設備看不出來，就給一組通用候選
		return []string{"pan", "pot", "bowl", "plate", "wok", "cup"}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// 嚴格驗證（加入 ARtype 白名單）
func validateARParams(p common.ARActionParams) error {
	if p.Type == "" {
		return fmt.Errorf("missing type")
	}

	// --- ARtype 白名單（與 iOS/前端一致的 13 種） ---
	validTypes := map[common.ARtype]struct{}{
		common.ARPutIntoContainer: {},
		common.ARStir:             {},
		common.ARPourLiquid:       {},
		common.ARFlipPan:          {},
		common.ARCountdown:        {},
		common.ARTemperature:      {},
		common.ARFlame:            {},
		common.ARSprinkle:         {},
		common.ARTorch:            {},
		common.ARCut:              {},
		common.ARPeel:             {},
		common.ARFlip:             {},
		common.ARBeatEgg:          {},
	}
	if _, ok := validTypes[p.Type]; !ok {
		return fmt.Errorf("invalid type: %s", p.Type)
	}

	// --- 依類型檢查必要欄位 ---
	switch p.Type {
	case common.ARPutIntoContainer:
		if p.Container == "" || p.Ingredient == nil || *p.Ingredient == "" {
			return fmt.Errorf("putIntoContainer requires ingredient & container")
		}
	case common.ARStir, common.ARSprinkle, common.ARFlip:
		if p.Container == "" || p.Ingredient == nil || *p.Ingredient == "" {
			return fmt.Errorf("%s requires ingredient & container", p.Type)
		}
	case common.ARFlipPan, common.ARBeatEgg:
		if p.Container == "" {
			return fmt.Errorf("%s requires container", p.Type)
		}
	case common.ARPourLiquid:
		if p.Container == "" || p.Color == nil || *p.Color == "" || p.Ingredient == nil || *p.Ingredient == "" {
			return fmt.Errorf("pourLiquid requires container, color & ingredient")
		}
	case common.ARCountdown:
		if p.Container == "" || p.Time.IsNil() {
			return fmt.Errorf("countdown requires time & container")
		}
	case common.ARTemperature:
		if p.Container == "" || p.Temperature.IsNil() {
			return fmt.Errorf("temperature requires temperature & container")
		}
	case common.ARFlame:
		if p.Container == "" || p.FlameLevel == nil {
			return fmt.Errorf("flame requires flameLevel & container")
		}
	case common.ARTorch, common.ARCut, common.ARPeel:
		if p.Ingredient == nil || *p.Ingredient == "" {
			return fmt.Errorf("%s requires ingredient", p.Type)
		}
	}

	// 若未啟用座標欄位，這段可留註解
	// if p.Coordinate != nil && len(p.Coordinate) != 3 {
	// 	return fmt.Errorf("coordinate must be [x,y,z] or null")
	// }

	return nil
}

func fallbackARParams(step common.RecipeStep, containerChoices []string, recipeIngredients []common.Ingredient) (*common.ARActionParams, error) {
	typeGuess := inferARTypeFromStep(step)
	if typeGuess == "" {
		typeGuess = common.ARPutIntoContainer
	}

	params := &common.ARActionParams{
		Type: typeGuess,
	}

	if requiresContainer(typeGuess) {
		params.Container = chooseFallbackContainer(containerChoices)
		if params.Container == "" {
			params.Container = "bowl"
		}
	}

	if requiresIngredient(typeGuess) {
		ing := inferIngredientIdentifier(step, recipeIngredients)
		if ing == "" {
			ing = "ingredient"
		}
		params.Ingredient = strPtr(ing)
	} else {
		params.Ingredient = nil
	}

	if typeGuess == common.ARPourLiquid {
		color := inferColorIdentifier(step)
		if color == "" {
			color = "clear"
		}
		params.Color = strPtr(color)
	} else {
		params.Color = nil
	}

	if err := validateARParams(*params); err != nil {
		return nil, err
	}

	return params, nil
}

func inferARTypeFromStep(step common.RecipeStep) common.ARtype {
	text := strings.ToLower(step.Title + " " + step.Description)
	for _, act := range step.Actions {
		text += " " + strings.ToLower(act.Action)
		text += " " + strings.ToLower(act.InstructionDetail)
	}

	switch {
	case strings.Contains(text, "torch"), strings.Contains(text, "炙"), strings.Contains(text, "火烤"):
		return common.ARTorch
	case strings.Contains(text, "sprinkle"), strings.Contains(text, "撒"), strings.Contains(text, "灑"):
		return common.ARSprinkle
	case strings.Contains(text, "pour"), strings.Contains(text, "倒入"), strings.Contains(text, "淋"), strings.Contains(text, "倒上"):
		return common.ARPourLiquid
	case strings.Contains(text, "peel"), strings.Contains(text, "去皮"):
		return common.ARPeel
	case strings.Contains(text, "flip"), strings.Contains(text, "翻面"):
		return common.ARFlip
	case strings.Contains(text, "cut"), strings.Contains(text, "切"), strings.Contains(text, "slice"), strings.Contains(text, "chop"):
		return common.ARCut
	case strings.Contains(text, "stir"), strings.Contains(text, "mix"), strings.Contains(text, "攪拌"), strings.Contains(text, "拌"):
		return common.ARStir
	case strings.Contains(text, "put"), strings.Contains(text, "放入"), strings.Contains(text, "加入"):
		return common.ARPutIntoContainer
	case strings.Contains(text, "beat"), strings.Contains(text, "whisk"), strings.Contains(text, "打蛋"):
		return common.ARBeatEgg
	default:
		return common.ARBeatEgg
	}
}

func requiresContainer(t common.ARtype) bool {
	switch t {
	case common.ARPutIntoContainer, common.ARStir, common.ARPourLiquid, common.ARFlipPan,
		common.ARCountdown, common.ARTemperature, common.ARFlame, common.ARSprinkle,
		common.ARFlip, common.ARBeatEgg:
		return true
	default:
		return false
	}
}

func requiresIngredient(t common.ARtype) bool {
	switch t {
	case common.ARPutIntoContainer, common.ARStir, common.ARPourLiquid, common.ARSprinkle,
		common.ARTorch, common.ARCut, common.ARPeel, common.ARFlip:
		return true
	default:
		return false
	}
}

func chooseFallbackContainer(candidates []string) string {
	if len(candidates) > 0 {
		return candidates[0]
	}
	return "bowl"
}

func defaultARParams(containerChoices []string) *common.ARActionParams {
	container := chooseFallbackContainer(containerChoices)
	if container == "" {
		container = "bowl"
	}
	params := &common.ARActionParams{
		Type:      common.ARBeatEgg,
		Container: container,
	}
	return params
}

func buildSuggestionKey(req *common.RecipeByIngredientsRequest) string {
	if req == nil {
		return ""
	}
	ingParts := make([]string, 0, len(req.AvailableIngredients))
	for _, ing := range req.AvailableIngredients {
		part := fmt.Sprintf("%s|%s|%s|%s|%s",
			strings.ToLower(strings.TrimSpace(ing.Name)),
			strings.ToLower(strings.TrimSpace(ing.Type)),
			strings.ToLower(strings.TrimSpace(ing.Amount)),
			strings.ToLower(strings.TrimSpace(ing.Unit)),
			strings.ToLower(strings.TrimSpace(ing.Preparation)),
		)
		ingParts = append(ingParts, part)
	}
	sort.Strings(ingParts)

	eqParts := make([]string, 0, len(req.AvailableEquipment))
	for _, eq := range req.AvailableEquipment {
		part := fmt.Sprintf("%s|%s|%s|%s|%s",
			strings.ToLower(strings.TrimSpace(eq.Name)),
			strings.ToLower(strings.TrimSpace(eq.Type)),
			strings.ToLower(strings.TrimSpace(eq.Size)),
			strings.ToLower(strings.TrimSpace(eq.Material)),
			strings.ToLower(strings.TrimSpace(eq.PowerSource)),
		)
		eqParts = append(eqParts, part)
	}
	sort.Strings(eqParts)

	restrictions := append([]string(nil), req.Preference.DietaryRestrictions...)
	for i := range restrictions {
		restrictions[i] = strings.ToLower(strings.TrimSpace(restrictions[i]))
	}
	sort.Strings(restrictions)

	keyParts := []string{
		strings.Join(ingParts, ";"),
		strings.Join(eqParts, ";"),
		strings.Join(restrictions, ";"),
		strings.ToLower(strings.TrimSpace(req.Preference.CookingMethod)),
		strings.ToLower(strings.TrimSpace(req.Preference.ServingSize)),
	}

	return strings.Join(keyParts, "||")
}

var canonicalIngredientMap = map[string]string{
	"bacon":         "bacon",
	"brazil":        "brazil",
	"brocoli":       "brocoli",
	"butter":        "butter",
	"carrot":        "carrot",
	"cheese":        "cheese",
	"chickenthigh":  "chickenThigh",
	"chicken_thigh": "chickenThigh",
	"chili":         "chili",
	"corn":          "corn",
	"egg":           "egg",
	"garlic":        "garlic",
	"greenpepper":   "green_pepper",
	"green_pepper":  "green_pepper",
	"meat":          "meat",
	"mushroom":      "mushroom",
	"noodle":        "noodle",
	"onion":         "onion",
	"potato":        "potato",
	"salmon":        "salmon",
	"shrimp":        "shrimp",
	"squid":         "squid",
	"tofu":          "tofu",
	"tomato":        "tomato",
	"toast":         "toast",
}

func canonicalizeIngredient(norm string) (string, bool) {
	if norm == "" {
		return "", false
	}
	if val, ok := canonicalIngredientMap[norm]; ok {
		return val, true
	}
	noUnderscore := strings.ReplaceAll(norm, "_", "")
	if val, ok := canonicalIngredientMap[noUnderscore]; ok {
		return val, true
	}
	return "", false
}

func inferIngredientIdentifier(step common.RecipeStep, recipeIngredients []common.Ingredient) string {
	candidates := make([]string, 0)
	for _, act := range step.Actions {
		candidates = append(candidates, act.MaterialRequired...)
		candidates = append(candidates, act.Action)
		candidates = append(candidates, act.InstructionDetail)
	}
	candidates = append(candidates, step.Title, step.Description)
	for _, ing := range recipeIngredients {
		candidates = append(candidates, ing.Name, ing.Type, ing.Preparation)
	}
	var firstCandidate string
	for _, cand := range candidates {
		norm := normalizeIdentifierCandidate(cand)
		if norm == "" {
			continue
		}
		if canonical, ok := canonicalizeIngredient(norm); ok {
			return formatIngredientIdentifier(canonical)
		}
		if firstCandidate == "" {
			firstCandidate = norm
		}
	}
	for _, ing := range recipeIngredients {
		nameNorm := normalizeIdentifierCandidate(ing.Name)
		if canonical, ok := canonicalizeIngredient(nameNorm); ok {
			return formatIngredientIdentifier(canonical)
		}
		if firstCandidate == "" && nameNorm != "" {
			firstCandidate = nameNorm
		}
		prepNorm := normalizeIdentifierCandidate(ing.Preparation)
		if canonical, ok := canonicalizeIngredient(prepNorm); ok {
			return formatIngredientIdentifier(canonical)
		}
		if firstCandidate == "" && prepNorm != "" {
			firstCandidate = prepNorm
		}
	}
	if firstCandidate != "" {
		return formatIngredientIdentifier(firstCandidate)
	}
	for _, ing := range recipeIngredients {
		raw := strings.TrimSpace(ing.Name)
		if raw != "" {
			return formatIngredientIdentifier(raw)
		}
	}
	return "ingredient"
}

func formatIngredientIdentifier(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return id
	}
	if strings.ContainsAny(id, " _-/;,\t") {
		parts := strings.FieldsFunc(id, func(r rune) bool {
			switch r {
			case ' ', '\t', '_', '-', '/', ';':
				return true
			case ',':
				return true
			default:
				return false
			}
		})
		if len(parts) == 0 {
			return strings.ToLower(id)
		}
		for i := range parts {
			parts[i] = strings.ToLower(strings.TrimSpace(parts[i]))
		}
		if len(parts) == 1 {
			return parts[0]
		}
		return strings.Join(parts, ",")
	}
	if len(id) == 1 {
		return strings.ToLower(id)
	}
	first := strings.ToLower(id[:1])
	return first + id[1:]
}

func normalizeIdentifierCandidate(input string) string {
	if input == "" {
		return ""
	}
	input = strings.ToLower(strings.TrimSpace(input))
	var outRunes []rune
	var lastUnderscore bool
	for _, r := range input {
		switch {
		case r >= 'a' && r <= 'z':
			outRunes = append(outRunes, r)
			lastUnderscore = false
		case r >= '0' && r <= '9':
			if len(outRunes) > 0 {
				outRunes = append(outRunes, r)
			}
		case unicode.IsSpace(r) || r == '-' || r == '_' || r == '/':
			if len(outRunes) > 0 && !lastUnderscore {
				outRunes = append(outRunes, '_')
				lastUnderscore = true
			}
		default:
			// ignore other characters
		}
	}
	result := strings.Trim(string(outRunes), "_")
	if result == "" {
		return ""
	}
	return result
}

func inferColorIdentifier(step common.RecipeStep) string {
	for _, act := range step.Actions {
		if norm := normalizeColorCandidate(act.InstructionDetail); norm != "" {
			return norm
		}
	}
	return "clear"
}

func normalizeColorCandidate(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return ""
	}
	words := strings.FieldsFunc(input, func(r rune) bool {
		return !(r >= 'a' && r <= 'z')
	})
	for _, w := range words {
		if len(w) > 0 && w[0] >= 'a' && w[0] <= 'z' {
			return w
		}
	}
	return ""
}


func strPtr(s string) *string {
	return &s
}
