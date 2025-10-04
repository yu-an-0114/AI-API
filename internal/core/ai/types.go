package ai

// Response AI 響應
type Response struct {
	Choices  []Choice `json:"choices"`
	Usage    Usage    `json:"usage"`
	CacheHit bool     `json:"cache_hit"`
}

// Choice 選擇
type Choice struct {
	Message Message `json:"message"`
}

// Message 消息
type Message struct {
	Content string `json:"content"`
}

// Usage 使用量
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
