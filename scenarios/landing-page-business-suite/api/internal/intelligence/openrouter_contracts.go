package intelligence

import (
	"context"
	"net/http"
	"time"
)

// OpenRouterClient is the provider seam used by the metered inference provider.
//
// seam: OpenRouterClient makes provider transport substitutable in metered inference
// tests and keeps HTTP implementation details out of credit orchestration.
type OpenRouterClient interface {
	Chat(context.Context, OpenRouterChatRequest) (*OpenRouterChatResponse, error)
	ChatStream(context.Context, OpenRouterChatRequest, func(string)) (*OpenRouterUsage, error)
	VerifyAPIKey(context.Context) error
}

type OpenRouterChatRequest struct {
	Model    string              `json:"model"`
	Messages []OpenRouterMessage `json:"messages"`
	Stream   bool                `json:"stream,omitempty"`
}

type OpenRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenRouterChatResponse struct {
	ID           string          `json:"id"`
	Model        string          `json:"model"`
	Content      string          `json:"content"`
	FinishReason string          `json:"finish_reason,omitempty"`
	Usage        OpenRouterUsage `json:"usage"`
}

type OpenRouterUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenRouterClientOptions configures the provider HTTP implementation.
type OpenRouterClientOptions struct {
	APIKey     string
	BaseURL    string
	Referer    string
	Title      string
	Timeout    time.Duration
	HTTPClient *http.Client
	Logger     func(string, map[string]interface{})
}
