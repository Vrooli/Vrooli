package intelligence

import (
	"context"
	"net/http"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

// UsageServicer is the narrow credit-policy contract the metered inference provider needs.
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

type UsageMetadataFinalizer interface {
	FinalizeReservationWithMetadata(context.Context, string, int64, string, string) error
}

type UsageCostFinalizer interface {
	FinalizeReservationWithCost(context.Context, string, int64, string, string, int64, int, int, string) error
}

type CostCapUsageServicer interface {
	ReserveUSDSpendCap(context.Context, string, string, int64, string) (string, int64, int64, error)
	RetainUncertainReservation(context.Context, string) error
}

func finalizeUsage(s UsageServicer, ctx context.Context, reservationID string, amount int64, appBundleKey, model string) error {
	if detailed, ok := s.(UsageMetadataFinalizer); ok {
		return detailed.FinalizeReservationWithMetadata(ctx, reservationID, amount, appBundleKey, model)
	}
	return s.FinalizeReservation(ctx, reservationID, amount)
}

func finalizeUsageWithCost(s UsageServicer, ctx context.Context, reservationID string, amount int64, appBundleKey, model string, costMicros int64, promptTokens, completionTokens int, provider string) error {
	if detailed, ok := s.(UsageCostFinalizer); ok {
		return detailed.FinalizeReservationWithCost(ctx, reservationID, amount, appBundleKey, model, costMicros, promptTokens, completionTokens, provider)
	}
	return finalizeUsage(s, ctx, reservationID, amount, appBundleKey, model)
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
	Model             string      `json:"model"`
	Messages          []AIMessage `json:"messages"`
	Images            []AIImage   `json:"images,omitempty"`
	Stream            bool        `json:"stream"`
	MaxTokens         int         `json:"max_tokens,omitempty"`
	Metadata          AIMetadata  `json:"metadata,omitempty"`
	SpendLimit        *SpendLimit `json:"spend_limit,omitempty"`
	spendLimitTrusted bool
}

// InferenceRequest is the public, provider-neutral metered contract. Callers
// select a capability role and constraints; concrete provider model slugs are
// resolved only by the metered service policy.
type InferenceRequest struct {
	Role            string      `json:"role"`
	Messages        []AIMessage `json:"messages"`
	ConstraintsJSON string      `json:"constraints_json,omitempty"`
	MaxTokens       int         `json:"max_tokens,omitempty"`
	Metadata        AIMetadata  `json:"metadata,omitempty"`
	Images          []AIImage   `json:"images,omitempty"`
	SpendLimit      *SpendLimit `json:"spend_limit,omitempty"`
}

// SpendLimit carries a caller-authorized maximum into the canonical metered
// owner. It is a hard cap policy request, not a price quote.
type SpendLimit struct {
	MaxCostMicros    int64    `json:"max_cost_micros"`
	OperationID      string   `json:"operation_id"`
	InputFingerprint string   `json:"input_fingerprint"`
	ReferenceSHA256  []string `json:"reference_sha256"`
	ImageCount       int32    `json:"image_count"`
	FundingRoute     string   `json:"funding_route"`
}

type AIImage struct {
	Data      string `json:"data"`
	MediaType string `json:"media_type"`
	SHA256    string `json:"sha256"`
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
	ID                 string                          `json:"id"`
	Model              string                          `json:"model"`
	Content            string                          `json:"content"`
	PromptTokens       int                             `json:"prompt_tokens"`
	CompletionTokens   int                             `json:"completion_tokens"`
	TotalTokens        int                             `json:"total_tokens"`
	CreditsCharged     int64                           `json:"credits_charged"`
	FinishReason       string                          `json:"finish_reason,omitempty"`
	SpendAuthorization *inferencev1.SpendAuthorization `json:"spend_authorization,omitempty"`
	AccountingStatus   string                          `json:"accounting_status,omitempty"`
	ActualCostMicros   *int64                          `json:"actual_cost_micros,omitempty"`
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

// MeteredInferenceProvider is the transport-facing AI capability contract.
//
// seam: MeteredInferenceProvider lets HTTP tests verify request handling without a provider,
// credit store, or account database.
type MeteredInferenceProvider interface {
	ExecuteChat(context.Context, string, AIRequest) (*AIResponse, error)
	ExecuteChatStream(context.Context, string, AIRequest, http.ResponseWriter) error
	GetAvailableModels() []string
	HealthCheck(context.Context) error
}
