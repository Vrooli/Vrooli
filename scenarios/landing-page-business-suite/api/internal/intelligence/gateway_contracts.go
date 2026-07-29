package intelligence

import (
	"context"
	"net/http"

	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

// UsageServicer is the narrow credit-policy contract the AI gateway needs.
//
// seam: UsageServicer keeps AI-provider orchestration independent of commerce
// persistence. API composition supplies commerce.UsageService; gateway tests
// use a domain-local fake.
type UsageServicer interface {
	ReserveAndCharge(context.Context, string, string, string, int64, UsageReport) error
	ReserveCredits(context.Context, string, string, string, int64) (string, error)
	FinalizeReservation(context.Context, string, int64) error
	ReleaseReservation(context.Context, string) error
	AdjustUsage(context.Context, string, string, int64, string) error
	RecordUsage(context.Context, UsageReport) error
}

// UsageReport is the minimum credit-accounting data emitted by the gateway.
// It is intentionally domain-owned: the API composition adapter converts it
// to commerce persistence types, avoiding a sibling-domain import.
type UsageReport struct {
	UserIdentity string
	LimitKey     string
	Amount       int64
	AppBundleKey string
	Operation    string
	Metadata     map[string]string
}

// AccountServicer supplies the subscription context used to apply AI policy.
//
// seam: AccountServicer prevents provider orchestration from owning account
// lookup or subscription persistence.
type AccountServicer interface {
	GetSubscriptionContext(context.Context, string) (*shared.SubscriptionStatus, error)
}

// APIKeyServicer retrieves the provider credential selected by the operator.
//
// seam: APIKeyServicer keeps encrypted key storage outside the AI domain.
type APIKeyServicer interface {
	Get(context.Context, string) (string, error)
}

// AIRequest is one chat-completion request accepted by the gateway.
type AIRequest struct {
	Model     string      `json:"model"`
	Messages  []AIMessage `json:"messages"`
	Stream    bool        `json:"stream"`
	MaxTokens int         `json:"max_tokens,omitempty"`
	Metadata  AIMetadata  `json:"metadata,omitempty"`
}

type AIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AIMetadata struct {
	AppBundleKey string `json:"app_bundle_key,omitempty"`
	Operation    string `json:"operation,omitempty"`
}

type AIResponse struct {
	ID               string `json:"id"`
	Model            string `json:"model"`
	Content          string `json:"content"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	CreditsCharged   int64  `json:"credits_charged"`
	FinishReason     string `json:"finish_reason,omitempty"`
}

type AIStreamEvent struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
	Usage   *struct {
		PromptTokens     int   `json:"prompt_tokens"`
		CompletionTokens int   `json:"completion_tokens"`
		TotalTokens      int   `json:"total_tokens"`
		CreditsCharged   int64 `json:"credits_charged"`
	} `json:"usage,omitempty"`
}

// ModelPricing is the credit cost for 1K provider tokens.
type ModelPricing struct {
	PromptCostPer1K     int64 `json:"prompt_cost_per_1k"`
	CompletionCostPer1K int64 `json:"completion_cost_per_1k"`
}

// Gateway is the transport-facing AI capability contract.
//
// seam: Gateway lets HTTP tests verify request handling without a provider,
// credit store, or account database.
type Gateway interface {
	ExecuteChat(context.Context, string, AIRequest) (*AIResponse, error)
	ExecuteChatStream(context.Context, string, AIRequest, http.ResponseWriter) error
	GetAvailableModels() []string
	HealthCheck(context.Context) error
}
