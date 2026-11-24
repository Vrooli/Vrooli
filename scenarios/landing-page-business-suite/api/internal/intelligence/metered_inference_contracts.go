package intelligence

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"net/http"
	"strings"

	"github.com/vrooli/vrooli/packages/monetization-go"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"landing-page-business-suite-api/internal/commerce"
)

var ErrBrowserEvidenceReplay = errors.New("browser evidence is already bound to an LPBS inference reservation")
var ErrInferenceOperationReplay = errors.New("Gateway inference operation already has an LPBS reservation; duplicate dispatch refused")
var ErrInferenceOutcomeUncertain = errors.New("inference provider outcome is uncertain")
var ErrInferenceReservationRetentionFailed = errors.New("uncertain inference reservation could not be retained")
var ErrInferenceReservationCleanupFailed = errors.New("inference reservation cleanup failed")
var ErrProviderRequestRejected = errors.New("provider definitively rejected the request")

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

// BrowserUsageFinalizer adds measured browser workload to the same immutable
// LPBS settlement as model usage. It does not create a separate wallet entry.
type BrowserUsageFinalizer interface {
	FinalizeReservationWithBrowserUsage(context.Context, string, int64, string, string, int64, int, int, string, BrowserUsage) error
}

// BrowserEvidenceAttester resolves and consumes an opaque BAS broker proof
// through Agent Manager's dedicated LPBS attestation operation. Implementations
// return only a closed, broker-measured session bound to the active LPBS
// reservation. Caller-provided counters are never trusted.
type BrowserEvidenceAttester interface {
	VerifyAndConsumeBrowserEvidence(context.Context, BrowserEvidenceProof) (AttestedBrowserUsage, error)
}

// BrowserEvidenceAttesterReadiness lets a production attester verify its
// operator-issued service identity before LPBS reserves credits or dispatches
// inference. Test/local adapters may omit it; the concrete AM client implements
// it and fails closed when the credential or target is unavailable.
type BrowserEvidenceAttesterReadiness interface {
	Ready(context.Context) error
}

type BrowserEvidenceProof struct {
	RunID         string `json:"run_id"`
	GrantReceipt  string `json:"grant_receipt"`
	Reference     string `json:"proof_reference"`
	ReservationID string `json:"-"` // assigned by LPBS after reserve; never caller supplied
}

type AttestedBrowserUsage struct {
	RunID          string
	ReservationID  string
	SessionID      string
	State          string
	SessionMinutes int64
	Actions        int64
	Bytes          int64
}

func (p BrowserEvidenceProof) validate() error {
	if strings.TrimSpace(p.RunID) == "" || strings.TrimSpace(p.GrantReceipt) == "" || len(p.GrantReceipt) > 512 || strings.ContainsAny(p.GrantReceipt, "\r\n") || strings.TrimSpace(p.Reference) == "" || len(p.Reference) > 256 || strings.ContainsAny(p.Reference, "\r\n") {
		return errors.New("browser evidence requires a run, exact grant receipt, and opaque proof reference")
	}
	return nil
}

func validateGatewayRootOperationID(value string) error {
	if value == "" {
		return nil
	}
	if len(value) < 32 || len(value) > 128 || strings.TrimSpace(value) != value || strings.ContainsAny(value, " \t\r\n") {
		return errors.New("gateway_root_operation_id must be a bounded caller-generated recovery key")
	}
	return nil
}

func validateInferenceRootOperationID(role, value string) error {
	if err := validateGatewayRootOperationID(value); err != nil {
		return err
	}
	if strings.HasPrefix(strings.TrimSpace(role), "browser.") && value == "" {
		return errors.New("browser inference requires gateway_root_operation_id for reservation replay protection")
	}
	return nil
}

func validateInferenceBrowserEvidence(role string, metadata AIMetadata) error {
	if err := validateBrowserEvidenceInput(metadata); err != nil {
		return err
	}
	browserRole := strings.HasPrefix(strings.TrimSpace(role), "browser.")
	if metadata.BrowserEvidence != nil {
		return errors.New("closed BAS proof is accepted only after provider operations finish and the session closes")
	}
	if metadata.BrowserSession != nil {
		if !browserRole || metadata.BrowserUsage != (BrowserUsage{}) {
			return errors.New("browser session inference requires a browser role and rejects pre-close proofs or caller counters")
		}
		if err := metadata.BrowserSession.validate(); err != nil {
			return err
		}
		return nil
	}
	if browserRole {
		return errors.New("browser inference requires BAS run, grant and session binding; reconcile its closed-session proof after close")
	}
	return nil
}

func (b BrowserSessionBinding) validate() error {
	runID, err := uuid.Parse(strings.TrimSpace(b.RunID))
	if err != nil || runID == uuid.Nil || runID.String() != b.RunID {
		return errors.New("browser session requires the canonical Agent Manager run id")
	}
	if strings.TrimSpace(b.SessionID) == "" || len(b.SessionID) > 128 || strings.ContainsAny(b.SessionID, "\r\n") || strings.TrimSpace(b.SessionID) != b.SessionID {
		return errors.New("browser session requires a bounded BAS session id")
	}
	if strings.TrimSpace(b.GrantReceipt) == "" || len(b.GrantReceipt) > 512 || strings.ContainsAny(b.GrantReceipt, "\r\n") || strings.TrimSpace(b.GrantReceipt) != b.GrantReceipt {
		return errors.New("browser session requires the exact Agent Manager grant receipt")
	}
	if b.ReservationID != "" {
		id, err := uuid.Parse(b.ReservationID)
		if err != nil || id == uuid.Nil || id.String() != b.ReservationID {
			return errors.New("browser session reservation id must be a canonical LPBS UUID")
		}
	}
	return nil
}

func (u AttestedBrowserUsage) validateFor(proof BrowserEvidenceProof) error {
	if strings.TrimSpace(u.RunID) != strings.TrimSpace(proof.RunID) || strings.TrimSpace(u.ReservationID) != strings.TrimSpace(proof.ReservationID) || strings.TrimSpace(u.SessionID) == "" || u.State != "closed" {
		return errors.New("BAS evidence does not match the requested closed browser run and LPBS reservation")
	}
	return (BrowserUsage{SessionMinutes: u.SessionMinutes, Actions: u.Actions, Bytes: u.Bytes}).validate()
}

type UncertainReservationRetainer interface {
	RetainUncertainReservation(context.Context, string) error
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
	return finalizeUsageWithBrowserUsage(s, ctx, reservationID, amount, appBundleKey, model, costMicros, promptTokens, completionTokens, provider, BrowserUsage{})
}

func finalizeUsageWithBrowserUsage(s UsageServicer, ctx context.Context, reservationID string, amount int64, appBundleKey, model string, costMicros int64, promptTokens, completionTokens int, provider string, browser BrowserUsage) error {
	if detailed, ok := s.(BrowserUsageFinalizer); ok {
		return detailed.FinalizeReservationWithBrowserUsage(ctx, reservationID, amount, appBundleKey, model, costMicros, promptTokens, completionTokens, provider, browser)
	}
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
	OperationID  *string
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
	GetSubscriptionForBusinessAccountContext(context.Context, string, string) (*shared.SubscriptionStatus, error)
}

// BusinessAccountUsageServicer persists the selected billing audience and the
// authenticated actor as separate identities. Account-scoped inference fails
// closed when this owner interface is unavailable.
type BusinessAccountUsageServicer interface {
	ReserveBusinessAccountCredits(context.Context, string, string, string, string, int64) (string, error)
	ReserveBusinessAccountCreditsWithID(context.Context, string, string, string, string, int64, string) (string, bool, error)
	ReserveBusinessAccountInferenceCreditsWithID(context.Context, string, string, string, string, int64, string) (string, bool, error)
	BindBusinessAccountBrowserEvidence(context.Context, string, string, string, string, string) error
	FinalizeBusinessAccountReservationWithCost(context.Context, string, string, string, int64, string, string, int64, int, int, string) error
	FinalizeBusinessAccountReservationWithBrowserUsage(context.Context, string, string, string, int64, string, string, int64, int, int, string, int64, int64, int64, string, string, string) error
	ReleaseBusinessAccountReservation(context.Context, string, string, string) error
	RetainUncertainBusinessAccountReservation(context.Context, string, string, string) error
	RecordBusinessAccountUsage(context.Context, string, string, string, string, int64, string, string, string, int64, int, int, string) error
	AdjustBusinessAccountUsage(context.Context, string, string, int64) error
}

// BrowserSessionMeteringServicer is the LPBS owner contract for one session
// hold with multiple immutable Gateway operation receipts and post-close
// reconciliation. It extends no other wallet and accepts no caller counters.
type BrowserSessionMeteringServicer interface {
	ReserveBusinessAccountBrowserSession(context.Context, string, string, string, string, int64, string) (string, bool, error)
	BindBusinessAccountBrowserSession(context.Context, string, string, string, string, string, string, string) error
	BeginBusinessAccountBrowserGatewayReceipt(context.Context, string, string, string, string, string, string, string, string, string, string, int64) (bool, error)
	RecordBusinessAccountBrowserGatewayReceipt(context.Context, string, string, string, commerce.BrowserGatewayReceipt) error
	MarkBusinessAccountBrowserGatewayUncertain(context.Context, string, string, string, string) error
	BusinessAccountBrowserGatewayAggregate(context.Context, string, string, string) (commerce.BrowserGatewayAggregate, error)
	FinalizeBusinessAccountBrowserSession(context.Context, string, string, string, string, string, string, string, string, commerce.BrowserGatewayAggregate, int64, int64, int64) error
}

// APIKeyServicer retrieves the provider credential selected by the operator.
//
// seam: APIKeyServicer keeps encrypted key storage outside the AI domain.
type APIKeyServicer interface {
	Get(context.Context, string) (string, error)
}

// AIRequest is one chat-completion request accepted by the gateway.
type AIRequest struct {
	Model               string            `json:"model"`
	Messages            []AIMessage       `json:"messages"`
	Tools               []json.RawMessage `json:"tools,omitempty"`
	ToolChoice          json.RawMessage   `json:"tool_choice,omitempty"`
	BusinessAccountID   string            `json:"business_account_id,omitempty"`
	Images              []AIImage         `json:"images,omitempty"`
	Stream              bool              `json:"stream"`
	MaxTokens           int               `json:"max_tokens,omitempty"`
	Metadata            AIMetadata        `json:"metadata,omitempty"`
	SpendLimit          *SpendLimit       `json:"spend_limit,omitempty"`
	spendLimitTrusted   bool
	creditOwnerVerified bool
}

// InferenceRequest is the public, provider-neutral metered contract. Callers
// select a capability role and constraints; concrete provider model slugs are
// resolved only by the metered service policy.
type InferenceRequest struct {
	Role              string                 `json:"role"`
	Messages          []AIMessage            `json:"messages"`
	Tools             []json.RawMessage      `json:"tools,omitempty"`
	ToolChoice        json.RawMessage        `json:"tool_choice,omitempty"`
	BusinessAccountID string                 `json:"business_account_id,omitempty"`
	ConstraintsJSON   string                 `json:"constraints_json,omitempty"`
	MaxTokens         int                    `json:"max_tokens,omitempty"`
	Metadata          AIMetadata             `json:"metadata,omitempty"`
	Images            []AIImage              `json:"images,omitempty"`
	SpendLimit        *SpendLimit            `json:"spend_limit,omitempty"`
	BrowserSession    *BrowserSessionBinding `json:"browser_session,omitempty"`
	creditOwner       *monetization.CreditFeatureOwner
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
	Role       string               `json:"role"`
	Content    string               `json:"content"`
	Name       string               `json:"name,omitempty"`
	ToolCallID string               `json:"tool_call_id,omitempty"`
	ToolCalls  []OpenRouterToolCall `json:"tool_calls,omitempty"`
}

type AIMetadata struct {
	AppBundleKey string `json:"app_bundle_key,omitempty"`
	Operation    string `json:"operation,omitempty"`
	// GatewayRootOperationID is an idempotency correlation key copied from the
	// AI Gateway request. It authorizes no access and does not supply usage.
	GatewayRootOperationID string                 `json:"gateway_root_operation_id,omitempty"`
	BrowserEvidence        *BrowserEvidenceProof  `json:"browser_evidence,omitempty"`
	BrowserSession         *BrowserSessionBinding `json:"browser_session,omitempty"`
	// BrowserUsage is retained only for wire compatibility. The metered service
	// rejects non-zero caller values; billing counters must come from BAS evidence.
	BrowserUsage BrowserUsage `json:"browser_usage,omitempty"`
}

// BrowserSessionBinding carries the existing AM run/grant and BAS session
// scope to LPBS. GrantReceipt is hashed by LPBS before persistence and is
// verified with the opaque closed proof only after BAS closes the session.
type BrowserSessionBinding struct {
	ReservationID string `json:"reservation_id,omitempty"`
	RunID         string `json:"run_id"`
	SessionID     string `json:"session_id"`
	GrantReceipt  string `json:"grant_receipt"`
}

// BrowserUsage is a measured, non-price usage attribution attached to an
// existing inference settlement. LPBS's existing credit authority remains the
// sole wallet; pricing this meter is a separate entitlement policy decision.
type BrowserUsage struct {
	SessionMinutes      int64  `json:"session_minutes,omitempty"`
	Actions             int64  `json:"actions,omitempty"`
	Bytes               int64  `json:"bytes,omitempty"`
	RunID               string `json:"run_id,omitempty"`
	SessionID           string `json:"session_id,omitempty"`
	EvidenceFingerprint string `json:"evidence_fingerprint,omitempty"`
}

func (u BrowserUsage) validate() error {
	if u.SessionMinutes < 0 || u.Actions < 0 || u.Bytes < 0 {
		return errors.New("browser usage counters must be non-negative")
	}
	return nil
}

type AIResponse struct {
	ID                 string                          `json:"id"`
	RootOperationID    string                          `json:"root_operation_id,omitempty"`
	BillingOperationID string                          `json:"billing_operation_id,omitempty"`
	Model              string                          `json:"model"`
	BusinessAccountID  string                          `json:"business_account_id,omitempty"`
	Content            string                          `json:"content"`
	ToolCalls          []OpenRouterToolCall            `json:"tool_calls,omitempty"`
	PromptTokens       int                             `json:"prompt_tokens"`
	CompletionTokens   int                             `json:"completion_tokens"`
	TotalTokens        int                             `json:"total_tokens"`
	CreditsCharged     int64                           `json:"credits_charged"`
	FinishReason       string                          `json:"finish_reason,omitempty"`
	SpendAuthorization *inferencev1.SpendAuthorization `json:"spend_authorization,omitempty"`
	AccountingStatus   string                          `json:"accounting_status,omitempty"`
	ActualCostMicros   *int64                          `json:"actual_cost_micros,omitempty"`
	BrowserUsage       BrowserUsage                    `json:"browser_usage,omitempty"`
}

type AIStreamEvent struct {
	Type      string                    `json:"type"`
	Content   string                    `json:"content,omitempty"`
	ToolCalls []OpenRouterToolCallDelta `json:"tool_calls,omitempty"`
	Error     string                    `json:"error,omitempty"`
	Usage     *struct {
		PromptTokens     int          `json:"prompt_tokens"`
		CompletionTokens int          `json:"completion_tokens"`
		TotalTokens      int          `json:"total_tokens"`
		CreditsCharged   int64        `json:"credits_charged"`
		BrowserUsage     BrowserUsage `json:"browser_usage,omitempty"`
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
