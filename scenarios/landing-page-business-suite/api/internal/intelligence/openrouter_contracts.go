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
	Model     string                    `json:"model"`
	Messages  []OpenRouterMessage       `json:"messages"`
	MaxTokens int                       `json:"max_tokens,omitempty"`
	Stream    bool                      `json:"stream,omitempty"`
	Usage     *OpenRouterUsageRequest   `json:"usage,omitempty"`
	Provider  *OpenRouterProviderPolicy `json:"provider,omitempty"`
}

type OpenRouterUsageRequest struct {
	Include bool `json:"include"`
}

type OpenRouterMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type OpenRouterProviderPolicy struct {
	Only              []string           `json:"only"`
	AllowFallbacks    bool               `json:"allow_fallbacks"`
	RequireParameters bool               `json:"require_parameters"`
	MaxPrice          OpenRouterMaxPrice `json:"max_price"`
}

type OpenRouterMaxPrice struct {
	Prompt     float64 `json:"prompt"`
	Completion float64 `json:"completion"`
	Request    float64 `json:"request"`
	Image      float64 `json:"image"`
}

type OpenRouterEndpointMetadata struct {
	Provider          string
	ContextTokens     int
	PromptPrice       float64
	CompletionPrice   float64
	RequestPrice      float64
	ImagePrice        float64
	RequestPriceKnown bool
	ImagePriceKnown   bool
}

// OpenRouterEndpointMetadataClient is optional so existing test and legacy
// clients remain compatible; hard-cap requests require the live metadata seam.
type OpenRouterEndpointMetadataClient interface {
	CurrentEndpoints(context.Context, string) ([]OpenRouterEndpointMetadata, error)
}

type OpenRouterChatResponse struct {
	ID           string          `json:"id"`
	Model        string          `json:"model"`
	Content      string          `json:"content"`
	FinishReason string          `json:"finish_reason,omitempty"`
	Usage        OpenRouterUsage `json:"usage"`
}

type OpenRouterUsage struct {
	PromptTokens     int   `json:"prompt_tokens"`
	CompletionTokens int   `json:"completion_tokens"`
	TotalTokens      int   `json:"total_tokens"`
	CostMicros       int64 `json:"cost_micros"`
	CostReported     bool  `json:"-"`
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
