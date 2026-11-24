package intelligence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/packages/monetization-go"
	"landing-page-business-suite-api/internal/commerce"
)

// Legacy app-scoped spending remains closed until its authenticated feature contract is migrated to the one door.
var ErrBillingAccountRequired = errors.New("Select a business account with authorized membership to use Vrooli credits")
var ErrBillingScopeUnavailable = errors.New("Verified app billing scope is unavailable; contact the app operator to complete LPBS billing setup")

const reservationCleanupTimeout = 5 * time.Second

func requireBillingScope(accountID string, ownerVerified bool) error {
	if strings.TrimSpace(accountID) == "" {
		return ErrBillingAccountRequired
	}
	if !ownerVerified {
		// Caller metadata cannot establish app-scoped authority. Standard
		// inference receives ownerVerified only after a verified caller service
		// principal matches the generated app/feature registry.
		return ErrBillingScopeUnavailable
	}
	return nil
}

// MeteredInferenceService handles AI requests through the LPBS gateway.
// It provides centralized AI access with credit management for all Vrooli applications.
//
// # Architecture
//
// The service coordinates between three concerns:
//   - OpenRouter communication (via OpenRouterClient interface)
//   - Credit management (via UsageService)
//   - User tier lookup (via AccountService)
//
// # Credit Units
//
// Credits are tracked in internal units where 1 internal unit = 1/1,000,000 of a cent.
// For example, a $1.00 value = 100,000,000 internal units.
// Model pricing is expressed as cost per 1K tokens in these internal units.
//
// # Charging Flow
//
// For non-streaming requests:
//  1. Estimate cost based on input length and default completion tokens (with safety margin)
//  2. Atomically reserve credits (check limit + charge) before making the AI call
//  3. After completion, refund difference if actual cost is less than estimate
//  4. If actual cost exceeds estimate, charge the additional amount
//
// For streaming requests:
//  1. Estimate cost and create a credit reservation atomically
//  2. Stream response to client
//  3. Finalize reservation with actual cost after stream completes
//
// # Streaming Credit Reservation System
//
// Streaming requests use a reservation-based system to prevent TOCTOU race conditions:
//
//	┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
//	│ 1. Reserve      │────►│ 2. Stream       │────►│ 3. Finalize     │
//	│    Credits      │     │    Response     │     │    Reservation  │
//	└─────────────────┘     └─────────────────┘     └─────────────────┘
//	      ▼                       ▼                       ▼
//	 Check limit +          Forward chunks         Mark reservation
//	 create pending         to client via          as finalized +
//	 reservation            SSE events             record actual usage
//
// Key behaviors:
//   - Reservations are created atomically with limit checking (SELECT FOR UPDATE)
//   - Pending reservations count toward effective usage for subsequent requests
//   - Reservations auto-expire after 10 minutes (background cleanup every 2 minutes)
//   - If a failed stream reports token usage, settle those measured tokens; if
//     the provider reports no measurable usage, release the reservation
//   - If finalization fails, falls back to direct usage recording
//
// # Why Small Overspend Window is Acceptable
//
// Between reservation and finalization, actual usage may exceed the reservation.
// This is acceptable because:
//  1. The difference is typically small (estimation includes safety margin)
//  2. Users pre-authorized the estimated amount, showing intent
//  3. Streaming requires knowing final token count, which is impossible upfront
//  4. Alternative (pre-charging max_tokens) would significantly overcharge users
//
// The reservation system prevents the more serious issue: concurrent requests
// from multiple sessions exceeding the credit limit.
type MeteredInferenceService struct {
	apiKeyService  APIKeyServicer  // Interface for testing
	usageService   UsageServicer   // Interface for testing
	accountService AccountServicer // Interface for testing
	log            func(event string, fields map[string]interface{})

	// OpenRouter client - injectable for testing
	openRouterClient OpenRouterClient
	clientFactory    func(string, func(string, map[string]interface{})) OpenRouterClient

	// Model pricing in internal units per 1K tokens.
	// Internal unit = 1/1,000,000 of a cent.
	modelPricing            map[string]ModelPricing
	rolePolicies            map[string]RolePolicy
	browserEvidenceAttester BrowserEvidenceAttester
	creditFeatureOwners     monetization.CreditFeatureRegistry
}

// MeteredInferenceServiceOptions configures the metered inference provider service.
type MeteredInferenceServiceOptions struct {
	// These contracts deliberately use the narrow interfaces above rather than
	// root concrete services. The gateway orchestrates provider calls and credit
	// policy; it does not own their persistence implementations.
	APIKeyService  APIKeyServicer
	UsageService   UsageServicer
	AccountService AccountServicer
	Logger         func(event string, fields map[string]interface{})
	// ClientFactory is composition-owned because provider endpoint and metadata
	// are deployment configuration, not intelligence policy.
	ClientFactory func(apiKey string, logger func(string, map[string]interface{})) OpenRouterClient

	// OpenRouterClient allows injecting a custom client for testing.
	// If nil, a real HTTP client is created when the API key is first needed.
	OpenRouterClient OpenRouterClient
	// BrowserEvidenceAttester must be backed by BAS's authenticated verifier.
	// It remains optional for requests without browser evidence; supplying a
	// proof while it is unavailable fails before reservation or provider use.
	BrowserEvidenceAttester BrowserEvidenceAttester
	CreditFeatureOwners     monetization.CreditFeatureRegistry
}

// allowedModels is the whitelist of models users can request.
var allowedModels = AllowedModels()

// NewMeteredInferenceService creates a new metered inference provider service.
func NewMeteredInferenceService(opts MeteredInferenceServiceOptions) *MeteredInferenceService {
	logger := opts.Logger
	if logger == nil {
		logger = func(string, map[string]interface{}) {}
	}

	return &MeteredInferenceService{
		apiKeyService:           opts.APIKeyService,
		usageService:            opts.UsageService,
		accountService:          opts.AccountService,
		log:                     logger,
		clientFactory:           opts.ClientFactory,
		openRouterClient:        opts.OpenRouterClient,
		modelPricing:            defaultModelPricing(),
		rolePolicies:            DefaultRolePolicies(),
		browserEvidenceAttester: opts.BrowserEvidenceAttester,
		creditFeatureOwners:     opts.CreditFeatureOwners,
	}
}

// ExecuteInference applies the stable role contract and delegates to the
// atomic reservation/provider pipeline. Concrete model selection is internal.
func (s *MeteredInferenceService) ExecuteInference(ctx context.Context, userIdentity string, req InferenceRequest) (*AIResponse, error) {
	return s.executeInference(ctx, userIdentity, req)
}

// ExecuteInferenceForCaller admits only a verified scenario principal whose
// source owns the requested role in the generated credit-feature registry.
// The HTTP layer passes the source only after cryptographic service-token
// verification; public JSON cannot populate this value.
func (s *MeteredInferenceService) ExecuteInferenceForCaller(ctx context.Context, userIdentity, callerScenario string, req InferenceRequest) (*AIResponse, error) {
	if err := validateGatewayRootOperationID(req.Metadata.GatewayRootOperationID); err != nil || strings.TrimSpace(req.Metadata.GatewayRootOperationID) == "" {
		return nil, errors.New("owner-scoped paid inference requires a caller-generated root operation ID")
	}
	owner, ok := s.creditFeatureOwners.OwnerForRole(strings.TrimSpace(req.Role))
	if !ok || owner.Scenario != strings.TrimSpace(callerScenario) || owner.AppKey == "" || owner.Feature.ID == "" {
		return nil, ErrBillingScopeUnavailable
	}
	req.creditOwner = &owner
	req.Metadata.AppBundleKey = owner.AppKey
	req.Metadata.Operation = owner.Feature.ID
	return s.executeInference(ctx, userIdentity, req)
}

func (s *MeteredInferenceService) executeInference(ctx context.Context, userIdentity string, req InferenceRequest) (*AIResponse, error) {
	if req.BrowserSession != nil {
		req.Metadata.BrowserSession = req.BrowserSession
	}
	if err := validateInferenceRootOperationID(req.Role, req.Metadata.GatewayRootOperationID); err != nil {
		return nil, err
	}
	if err := validateInferenceBrowserEvidence(req.Role, req.Metadata); err != nil {
		return nil, err
	}
	if err := requireBillingScope(req.BusinessAccountID, req.creditOwner != nil); err != nil {
		return nil, err
	}
	role := strings.TrimSpace(req.Role)
	policy, ok := s.rolePolicies[role]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrRoleNotAllowed, role)
	}
	messages := append([]AIMessage(nil), req.Messages...)
	if strings.TrimSpace(req.ConstraintsJSON) != "" {
		messages = append([]AIMessage{{Role: "system", Content: "Apply these caller constraints exactly:\n" + strings.TrimSpace(req.ConstraintsJSON)}}, messages...)
	}
	return s.ExecuteChat(ctx, userIdentity, AIRequest{
		Model: policy.Model, Messages: messages, Tools: req.Tools, ToolChoice: req.ToolChoice, BusinessAccountID: req.BusinessAccountID, Images: req.Images, MaxTokens: req.MaxTokens, Metadata: req.Metadata, SpendLimit: req.SpendLimit, spendLimitTrusted: true, creditOwnerVerified: req.creditOwner != nil,
	})
}

// ExecuteInferenceStream applies server-owned role policy before entering the
// same reservation and provider pipeline used by the legacy chat stream.
func (s *MeteredInferenceService) ExecuteInferenceStream(ctx context.Context, userIdentity string, req InferenceRequest, w http.ResponseWriter) error {
	return s.executeInferenceStream(ctx, userIdentity, req, w)
}

// ExecuteInferenceStreamForCaller is the streaming equivalent of
// ExecuteInferenceForCaller and carries the same immutable owner binding.
func (s *MeteredInferenceService) ExecuteInferenceStreamForCaller(ctx context.Context, userIdentity, callerScenario string, req InferenceRequest, w http.ResponseWriter) error {
	if err := validateGatewayRootOperationID(req.Metadata.GatewayRootOperationID); err != nil || strings.TrimSpace(req.Metadata.GatewayRootOperationID) == "" {
		return errors.New("owner-scoped paid inference requires a caller-generated root operation ID")
	}
	owner, ok := s.creditFeatureOwners.OwnerForRole(strings.TrimSpace(req.Role))
	if !ok || owner.Scenario != strings.TrimSpace(callerScenario) || owner.AppKey == "" || owner.Feature.ID == "" {
		return ErrBillingScopeUnavailable
	}
	req.creditOwner = &owner
	req.Metadata.AppBundleKey = owner.AppKey
	req.Metadata.Operation = owner.Feature.ID
	return s.executeInferenceStream(ctx, userIdentity, req, w)
}

func (s *MeteredInferenceService) executeInferenceStream(ctx context.Context, userIdentity string, req InferenceRequest, w http.ResponseWriter) error {
	if req.BrowserSession != nil {
		req.Metadata.BrowserSession = req.BrowserSession
	}
	if err := validateInferenceRootOperationID(req.Role, req.Metadata.GatewayRootOperationID); err != nil {
		return err
	}
	if err := validateInferenceBrowserEvidence(req.Role, req.Metadata); err != nil {
		return err
	}
	if err := requireBillingScope(req.BusinessAccountID, req.creditOwner != nil); err != nil {
		return err
	}
	if strings.TrimSpace(req.BusinessAccountID) == "" {
		return errors.New("business_account_id is required for account-scoped streaming")
	}
	if req.SpendLimit != nil {
		return errors.New("spend-limited inference does not support business-account streaming")
	}
	role := strings.TrimSpace(req.Role)
	policy, ok := s.rolePolicies[role]
	if !ok {
		return fmt.Errorf("%w: %s", ErrRoleNotAllowed, role)
	}
	messages := append([]AIMessage(nil), req.Messages...)
	if strings.TrimSpace(req.ConstraintsJSON) != "" {
		messages = append([]AIMessage{{Role: "system", Content: "Apply these caller constraints exactly:\n" + strings.TrimSpace(req.ConstraintsJSON)}}, messages...)
	}
	return s.ExecuteChatStream(ctx, userIdentity, AIRequest{
		Model: policy.Model, Messages: messages, BusinessAccountID: strings.TrimSpace(req.BusinessAccountID),
		Tools: req.Tools, ToolChoice: req.ToolChoice, Images: req.Images, MaxTokens: req.MaxTokens, Metadata: req.Metadata, Stream: true, creditOwnerVerified: req.creditOwner != nil,
	}, w)
}

// defaultModelPricing returns default pricing for common models.
// Prices are in internal units (1/1,000,000 of a cent) per 1K tokens.
// Based on OpenRouter prices as of early 2025.
func defaultModelPricing() map[string]ModelPricing {
	return DefaultModelPricing()
}

// ExecuteChat executes a non-streaming chat completion.
// Credits are checked and charged atomically to prevent TOCTOU race conditions.
func (s *MeteredInferenceService) ExecuteChat(ctx context.Context, userIdentity string, req AIRequest) (*AIResponse, error) {
	if err := validateGatewayRootOperationID(req.Metadata.GatewayRootOperationID); err != nil {
		return nil, err
	}
	if err := validateBrowserEvidenceInput(req.Metadata); err != nil {
		return nil, err
	}
	if req.Metadata.BrowserEvidence != nil && s.browserEvidenceAttester == nil {
		return nil, errors.New("BAS browser evidence verifier is unavailable")
	}
	if err := ensureBrowserEvidenceAttesterReady(ctx, s.browserEvidenceAttester, req.Metadata.BrowserEvidence); err != nil {
		return nil, err
	}
	if err := requireBillingScope(req.BusinessAccountID, req.creditOwnerVerified); err != nil {
		return nil, err
	}
	if req.SpendLimit != nil {
		if strings.TrimSpace(req.BusinessAccountID) != "" {
			return nil, errors.New("spend-limited inference does not support business-account billing")
		}
		if !req.spendLimitTrusted {
			return nil, errors.New("spend-limited inference is available only through an approved capability role")
		}
		return s.executeSpendCapped(ctx, userIdentity, req)
	}
	// Validate model
	if !allowedModels[req.Model] {
		return nil, fmt.Errorf("%w: %s", ErrModelNotAllowed, req.Model)
	}

	// Resolve the selected account's tier for policy. The account and actor remain
	// distinct in the commerce-owned usage ledger.
	billingIdentity, tier, err := s.billingIdentityAndTier(ctx, userIdentity, req.BusinessAccountID)
	if err != nil {
		s.log("metered_inference_get_tier_failed", map[string]interface{}{
			"level":         "error",
			"user_identity": userIdentity,
			"error":         err.Error(),
		})
		return nil, fmt.Errorf("failed to get user tier: %w", err)
	}

	// Estimate cost for pre-check
	estimate := s.estimateInferenceTokens(req)
	estimatedCost := s.calculateCost(req.Model, estimate.prompt, estimate.completion)
	if req.Metadata.BrowserSession != nil {
		return s.executeBrowserSessionChat(ctx, userIdentity, req, tier, estimatedCost)
	}

	// Reserve credits before the provider call; the reservation has an explicit
	// release path so provider failure cannot commit the estimate.
	reservationID, err := s.reserveCredits(ctx, userIdentity, billingIdentity, req.BusinessAccountID, tier, req.Metadata.AppBundleKey, estimatedCost, req.Metadata.BrowserEvidence, req.Metadata.GatewayRootOperationID)
	if err != nil {
		if errors.Is(err, ErrInsufficientCredits) || strings.Contains(err.Error(), "insufficient") {
			return nil, ErrInsufficientCredits
		}
		return nil, fmt.Errorf("credit check failed: %w", err)
	}
	if err := s.bindBrowserEvidence(ctx, userIdentity, req.BusinessAccountID, reservationID, req.Metadata.BrowserEvidence); err != nil {
		if releaseErr := s.releaseReservation(ctx, userIdentity, req.BusinessAccountID, reservationID); releaseErr != nil {
			return nil, errors.Join(fmt.Errorf("browser evidence could not be bound to LPBS reservation: %w", err), ErrInferenceReservationCleanupFailed, releaseErr)
		}
		return nil, fmt.Errorf("browser evidence could not be bound to LPBS reservation: %w", err)
	}

	// Get OpenRouter client
	client, err := s.getOpenRouterClient(ctx)
	if err != nil {
		if releaseErr := s.releaseReservation(ctx, userIdentity, req.BusinessAccountID, reservationID); releaseErr != nil {
			return nil, errors.Join(err, ErrInferenceReservationCleanupFailed, releaseErr)
		}
		return nil, err
	}

	// Convert messages to OpenRouter format
	orMessages := make([]OpenRouterMessage, len(req.Messages))
	for i, msg := range req.Messages {
		orMessages[i] = OpenRouterMessage{Role: msg.Role, Content: msg.Content, Name: msg.Name, ToolCallID: msg.ToolCallID, ToolCalls: msg.ToolCalls}
	}

	// Execute the chat request
	orResp, err := client.Chat(ctx, OpenRouterChatRequest{
		Model:      req.Model,
		Messages:   orMessages,
		Tools:      req.Tools,
		ToolChoice: req.ToolChoice,
	})
	if err != nil {
		if errors.Is(err, ErrProviderRequestRejected) {
			if releaseErr := s.releaseReservation(ctx, userIdentity, req.BusinessAccountID, reservationID); releaseErr != nil {
				return nil, errors.Join(fmt.Errorf("provider request rejected: %w", err), ErrInferenceReservationCleanupFailed, releaseErr)
			}
			return nil, fmt.Errorf("openrouter request was rejected: %w", err)
		}
		uncertainErr := s.markInferenceOutcomeUncertain(userIdentity, req.BusinessAccountID, reservationID, req.Metadata.GatewayRootOperationID)
		return nil, errors.Join(fmt.Errorf("openrouter request may have completed: %w", err), uncertainErr)
	}

	// Calculate actual cost based on real token usage
	actualCost := s.calculateCost(req.Model, orResp.Usage.PromptTokens, orResp.Usage.CompletionTokens)
	browserUsage, err := s.resolveBrowserUsage(ctx, req.Metadata.BrowserEvidence, reservationID)
	if err != nil {
		uncertainErr := s.markInferenceOutcomeUncertain(userIdentity, req.BusinessAccountID, reservationID, req.Metadata.GatewayRootOperationID)
		return nil, errors.Join(fmt.Errorf("browser evidence unavailable; reservation retained: %w", err), uncertainErr)
	}

	if err := s.finalizeReservationWithCost(ctx, userIdentity, req.BusinessAccountID, reservationID, actualCost, req.Metadata.AppBundleKey, req.Model, orResp.Usage.CostMicros, orResp.Usage.PromptTokens, orResp.Usage.CompletionTokens, "openrouter", browserUsage); err != nil {
		uncertainErr := s.markInferenceOutcomeUncertain(userIdentity, req.BusinessAccountID, reservationID, req.Metadata.GatewayRootOperationID)
		return nil, errors.Join(fmt.Errorf("finalize credit reservation: %w", err), uncertainErr)
	}

	s.log("metered_inference_request_completed", map[string]interface{}{
		"level":             "info",
		"user_identity":     userIdentity,
		"model":             req.Model,
		"prompt_tokens":     orResp.Usage.PromptTokens,
		"completion_tokens": orResp.Usage.CompletionTokens,
		"credits_charged":   actualCost,
	})

	return &AIResponse{
		ID:                orResp.ID,
		RootOperationID:   req.Metadata.GatewayRootOperationID,
		Model:             orResp.Model,
		BusinessAccountID: strings.TrimSpace(req.BusinessAccountID),
		Content:           orResp.Content,
		ToolCalls:         orResp.ToolCalls,
		PromptTokens:      orResp.Usage.PromptTokens,
		CompletionTokens:  orResp.Usage.CompletionTokens,
		TotalTokens:       orResp.Usage.TotalTokens,
		CreditsCharged:    actualCost,
		FinishReason:      orResp.FinishReason,
		BrowserUsage:      browserUsage,
	}, nil
}

func (s *MeteredInferenceService) executeBrowserSessionChat(ctx context.Context, actor string, req AIRequest, tier string, estimatedCost int64) (*AIResponse, error) {
	binding := req.Metadata.BrowserSession
	if binding == nil {
		return nil, errors.New("browser session binding is required")
	}
	if err := binding.validate(); err != nil {
		return nil, err
	}
	accountID := strings.TrimSpace(req.BusinessAccountID)
	if accountID == "" || strings.TrimSpace(actor) == "" {
		return nil, ErrBillingAccountRequired
	}
	_, ok := s.usageService.(BusinessAccountUsageServicer)
	if !ok {
		return nil, errors.New("business-account browser session reservation owner is unavailable")
	}
	sessionUsage, ok := s.usageService.(BrowserSessionMeteringServicer)
	if !ok {
		return nil, errors.New("LPBS browser session receipt owner is unavailable")
	}
	if req.Metadata.GatewayRootOperationID == "" {
		return nil, errors.New("Gateway root operation is required for browser session dispatch")
	}
	grantDigest := sha256.Sum256([]byte(binding.GrantReceipt))
	grantFingerprint := hex.EncodeToString(grantDigest[:])
	reservationID := strings.TrimSpace(binding.ReservationID)
	appBundleKey := strings.TrimSpace(req.Metadata.AppBundleKey)
	if appBundleKey == "" {
		appBundleKey = "unattributed"
	}
	if reservationID == "" {
		reservationID = browserEvidenceOperationID(req.Metadata.GatewayRootOperationID)
		reservedID, created, err := sessionUsage.ReserveBusinessAccountBrowserSession(ctx, accountID, actor, tier, appBundleKey, estimatedCost, reservationID)
		if err != nil {
			if errors.Is(err, ErrInsufficientCredits) {
				return nil, ErrInsufficientCredits
			}
			return nil, fmt.Errorf("reserve browser session credits: %w", err)
		}
		if !created {
			return nil, ErrInferenceOperationReplay
		}
		if reservedID != reservationID {
			mismatchErr := errors.New("LPBS returned a different browser session reservation identity")
			if releaseErr := s.releaseReservation(ctx, actor, accountID, reservedID); releaseErr != nil {
				return nil, errors.Join(mismatchErr, ErrInferenceReservationCleanupFailed, releaseErr)
			}
			return nil, mismatchErr
		}
		if err := sessionUsage.BindBusinessAccountBrowserSession(ctx, accountID, actor, reservationID, tier, binding.RunID, binding.SessionID, grantFingerprint); err != nil {
			if releaseErr := s.releaseReservation(ctx, actor, accountID, reservationID); releaseErr != nil {
				return nil, errors.Join(fmt.Errorf("bind browser session to LPBS reservation: %w", err), ErrInferenceReservationCleanupFailed, releaseErr)
			}
			return nil, fmt.Errorf("bind browser session to LPBS reservation: %w", err)
		}
	}
	requestBytes, err := json.Marshal(struct {
		Account                                           string
		Actor                                             string
		Model                                             string
		Messages                                          []AIMessage
		Tools                                             []json.RawMessage
		ToolChoice                                        json.RawMessage
		MaxTokens                                         int
		Root, Reservation, Run, Session, GrantFingerprint string
	}{
		Account: accountID, Actor: actor, Model: req.Model, Messages: req.Messages, Tools: req.Tools, ToolChoice: req.ToolChoice, MaxTokens: req.MaxTokens, Root: req.Metadata.GatewayRootOperationID, Reservation: reservationID, Run: binding.RunID, Session: binding.SessionID, GrantFingerprint: grantFingerprint,
	})
	if err != nil {
		return nil, errors.New("browser inference request fingerprint could not be created")
	}
	requestDigest := sha256.Sum256(requestBytes)
	requestFingerprint := hex.EncodeToString(requestDigest[:])
	claimed, err := sessionUsage.BeginBusinessAccountBrowserGatewayReceipt(ctx, accountID, actor, tier, appBundleKey, reservationID, binding.RunID, binding.SessionID, grantFingerprint, req.Metadata.GatewayRootOperationID, requestFingerprint, estimatedCost)
	if err != nil {
		return nil, err
	}
	if !claimed {
		return nil, commerce.ErrBrowserGatewayOperationReplay
	}
	markUncertain := func(cause error) (*AIResponse, error) {
		_ = sessionUsage.MarkBusinessAccountBrowserGatewayUncertain(ctx, accountID, actor, reservationID, req.Metadata.GatewayRootOperationID)
		return nil, fmt.Errorf("browser inference outcome is not settled; keep the LPBS session hold and reconcile root operation %s: %w", req.Metadata.GatewayRootOperationID, cause)
	}
	client, err := s.getOpenRouterClient(ctx)
	if err != nil {
		return markUncertain(err)
	}
	orMessages := make([]OpenRouterMessage, len(req.Messages))
	for i, msg := range req.Messages {
		orMessages[i] = OpenRouterMessage{Role: msg.Role, Content: msg.Content, Name: msg.Name, ToolCallID: msg.ToolCallID, ToolCalls: msg.ToolCalls}
	}
	orResp, err := client.Chat(ctx, OpenRouterChatRequest{Model: req.Model, Messages: orMessages, Tools: req.Tools, ToolChoice: req.ToolChoice})
	if err != nil {
		return markUncertain(err)
	}
	if orResp == nil || orResp.Usage.PromptTokens < 0 || orResp.Usage.CompletionTokens < 0 || orResp.Usage.CostMicros < 0 {
		return markUncertain(errors.New("provider returned an incomplete accounting receipt"))
	}
	actualCredits := s.calculateCost(req.Model, orResp.Usage.PromptTokens, orResp.Usage.CompletionTokens)
	receiptShape := struct {
		Root, Provider, Model             string
		Credits, Cost, Prompt, Completion int64
	}{req.Metadata.GatewayRootOperationID, "openrouter", orResp.Model, actualCredits, orResp.Usage.CostMicros, int64(orResp.Usage.PromptTokens), int64(orResp.Usage.CompletionTokens)}
	receiptBytes, err := json.Marshal(receiptShape)
	if err != nil {
		return markUncertain(err)
	}
	receiptDigest := sha256.Sum256(receiptBytes)
	receipt := commerce.BrowserGatewayReceipt{RootOperationID: req.Metadata.GatewayRootOperationID, RequestFingerprint: requestFingerprint, State: "settled", Provider: "openrouter", Model: orResp.Model, Credits: actualCredits, CostMicros: orResp.Usage.CostMicros, PromptTokens: int64(orResp.Usage.PromptTokens), CompletionTokens: int64(orResp.Usage.CompletionTokens), ReceiptFingerprint: hex.EncodeToString(receiptDigest[:])}
	if err := sessionUsage.RecordBusinessAccountBrowserGatewayReceipt(ctx, accountID, actor, reservationID, receipt); err != nil {
		return markUncertain(err)
	}
	return &AIResponse{ID: orResp.ID, RootOperationID: req.Metadata.GatewayRootOperationID, BillingOperationID: reservationID, Model: orResp.Model, BusinessAccountID: accountID, Content: orResp.Content, ToolCalls: orResp.ToolCalls, PromptTokens: orResp.Usage.PromptTokens, CompletionTokens: orResp.Usage.CompletionTokens, TotalTokens: orResp.Usage.TotalTokens, CreditsCharged: 0, FinishReason: orResp.FinishReason, AccountingStatus: "pending_browser_session_close"}, nil
}

// ReconcileClosedBrowserSession settles a session-level LPBS reservation only
// after the BAS/AM owner verifier has consumed a closed-session proof. Every
// model receipt remains a child operation; browser time is attached once to
// the session aggregate and never inferred from Gateway fields.
func (s *MeteredInferenceService) ReconcileClosedBrowserSession(ctx context.Context, actor, accountID string, binding BrowserSessionBinding, proof BrowserEvidenceProof) error {
	if err := requireBillingScope(accountID, false); err != nil {
		return err
	}
	return s.reconcileClosedBrowserSessionAuthorized(ctx, actor, accountID, binding, proof)
}

// ReconcileClosedBrowserSessionFromAgentManager is called only by the HTTP
// adapter behind lpbsidentity.AgentManagerBrowserSessionSettlement. The
// caller supplies LPBS-derived account and actor ownership plus AM's exact
// run/session/grant binding; proof verification remains delegated to AM/BAS.
func (s *MeteredInferenceService) ReconcileClosedBrowserSessionFromAgentManager(ctx context.Context, actor, accountID string, binding BrowserSessionBinding, proof BrowserEvidenceProof) error {
	return s.reconcileClosedBrowserSessionAuthorized(ctx, actor, accountID, binding, proof)
}

func (s *MeteredInferenceService) reconcileClosedBrowserSessionAuthorized(ctx context.Context, actor, accountID string, binding BrowserSessionBinding, proof BrowserEvidenceProof) error {
	if err := binding.validate(); err != nil {
		return err
	}
	if binding.ReservationID == "" || proof.ReservationID != "" || proof.RunID != binding.RunID || proof.GrantReceipt != binding.GrantReceipt {
		return errors.New("closed BAS proof must match the bound run, grant and session reservation")
	}
	if err := proof.validate(); err != nil {
		return err
	}
	if s.browserEvidenceAttester == nil {
		return errors.New("BAS browser evidence verifier is unavailable")
	}
	if err := ensureBrowserEvidenceAttesterReady(ctx, s.browserEvidenceAttester, &proof); err != nil {
		return err
	}
	accountID, actor = strings.TrimSpace(accountID), strings.TrimSpace(actor)
	usageService, ok := s.usageService.(BrowserSessionMeteringServicer)
	if !ok {
		return errors.New("LPBS browser session settlement owner is unavailable")
	}
	aggregate, err := usageService.BusinessAccountBrowserGatewayAggregate(ctx, accountID, actor, binding.ReservationID)
	if err != nil {
		return err
	}
	if aggregate.Operations == 0 {
		return errors.New("LPBS session has no completed Gateway operation receipt")
	}
	grantDigest := sha256.Sum256([]byte(binding.GrantReceipt))
	grantFingerprint := hex.EncodeToString(grantDigest[:])
	proof.ReservationID = binding.ReservationID
	attested, err := s.browserEvidenceAttester.VerifyAndConsumeBrowserEvidence(ctx, proof)
	if err != nil {
		return errors.New("BAS closed-session proof verification or consumption failed")
	}
	if err := attested.validateFor(proof); err != nil {
		return err
	}
	if attested.SessionID != binding.SessionID {
		return errors.New("BAS closed-session proof belongs to a different session")
	}
	proofFingerprint := browserEvidenceFingerprint(proof.Reference)
	return usageService.FinalizeBusinessAccountBrowserSession(ctx, accountID, actor, binding.ReservationID, binding.RunID, binding.SessionID, grantFingerprint, proofFingerprint, "", aggregate, attested.SessionMinutes, attested.Actions, attested.Bytes)
}

func validateBrowserEvidenceInput(metadata AIMetadata) error {
	if err := metadata.BrowserUsage.validate(); err != nil {
		return err
	}
	if metadata.BrowserUsage != (BrowserUsage{}) {
		return errors.New("caller-reported browser usage is not accepted; provide a BAS broker evidence proof")
	}
	if metadata.BrowserEvidence != nil {
		return metadata.BrowserEvidence.validate()
	}
	return nil
}

func ensureBrowserEvidenceAttesterReady(ctx context.Context, attester BrowserEvidenceAttester, proof *BrowserEvidenceProof) error {
	if proof == nil || attester == nil {
		return nil
	}
	if readiness, ok := attester.(BrowserEvidenceAttesterReadiness); ok {
		if err := readiness.Ready(ctx); err != nil {
			return errors.New("BAS browser evidence verifier identity is unavailable")
		}
	}
	return nil
}

func (s *MeteredInferenceService) resolveBrowserUsage(ctx context.Context, proof *BrowserEvidenceProof, reservationID string) (BrowserUsage, error) {
	if proof == nil {
		return BrowserUsage{}, nil
	}
	if s == nil || s.browserEvidenceAttester == nil {
		return BrowserUsage{}, errors.New("BAS browser evidence verifier is unavailable")
	}
	boundProof := *proof
	boundProof.ReservationID = strings.TrimSpace(reservationID)
	if boundProof.ReservationID == "" {
		return BrowserUsage{}, errors.New("browser evidence requires the active LPBS reservation")
	}
	attested, err := s.browserEvidenceAttester.VerifyAndConsumeBrowserEvidence(ctx, boundProof)
	if err != nil {
		return BrowserUsage{}, errors.New("BAS browser evidence verification or consumption failed")
	}
	if err := attested.validateFor(boundProof); err != nil {
		return BrowserUsage{}, err
	}
	return BrowserUsage{SessionMinutes: attested.SessionMinutes, Actions: attested.Actions, Bytes: attested.Bytes, RunID: attested.RunID, SessionID: attested.SessionID, EvidenceFingerprint: browserEvidenceFingerprint(proof.Reference)}, nil
}

func (s *MeteredInferenceService) bindBrowserEvidence(ctx context.Context, actorIdentity, businessAccountID, reservationID string, proof *BrowserEvidenceProof) error {
	if proof == nil {
		return nil
	}
	if strings.TrimSpace(businessAccountID) == "" {
		return errors.New("BAS browser evidence requires an LPBS business-account reservation")
	}
	accountUsage, ok := s.usageService.(BusinessAccountUsageServicer)
	if !ok {
		return errors.New("business-account browser evidence authority is unavailable")
	}
	return accountUsage.BindBusinessAccountBrowserEvidence(ctx, businessAccountID, actorIdentity, reservationID, proof.RunID, browserEvidenceFingerprint(proof.Reference))
}

func browserEvidenceFingerprint(reference string) string {
	digest := sha256.Sum256([]byte(reference))
	return hex.EncodeToString(digest[:])
}

func browserEvidenceOperationID(reference string) string {
	digest := sha256.Sum256([]byte("lpbs-browser-evidence-reservation-v1:" + reference))
	value := digest[:16]
	value[6] = (value[6] & 0x0f) | 0x50
	value[8] = (value[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", value[0:4], value[4:6], value[6:8], value[8:10], value[10:16])
}

func (s *MeteredInferenceService) billingIdentityAndTier(ctx context.Context, userIdentity, businessAccountID string) (string, string, error) {
	businessAccountID = strings.TrimSpace(businessAccountID)
	if businessAccountID == "" {
		return "", "", ErrBillingAccountRequired
	}
	if s.accountService == nil {
		return "", "", errors.New("business-account subscription authority is unavailable")
	}
	subscription, err := s.accountService.GetSubscriptionForBusinessAccountContext(ctx, userIdentity, businessAccountID)
	if err != nil {
		return "", "", fmt.Errorf("resolve business-account subscription: %w", err)
	}
	tier := "free"
	if subscription != nil && subscription.PlanTier != nil && strings.TrimSpace(*subscription.PlanTier) != "" {
		tier = strings.TrimSpace(*subscription.PlanTier)
	}
	return userIdentity, tier, nil
}

func (s *MeteredInferenceService) reserveCredits(ctx context.Context, actorIdentity, billingIdentity, businessAccountID, tier, appBundleKey string, amount int64, proof *BrowserEvidenceProof, rootOperationID string) (string, error) {
	if strings.TrimSpace(businessAccountID) == "" {
		return s.usageService.ReserveCredits(ctx, billingIdentity, tier, "ai_credits", amount)
	}
	accountUsage, ok := s.usageService.(BusinessAccountUsageServicer)
	if !ok {
		return "", errors.New("business-account usage authority is unavailable")
	}
	if strings.TrimSpace(appBundleKey) == "" {
		return "", errors.New("business-account inference requires a canonical app owner")
	}
	if proof != nil || rootOperationID != "" {
		operationSource := rootOperationID
		if operationSource == "" && proof != nil {
			operationSource = proof.Reference
		}
		operationID := browserEvidenceOperationID(operationSource)
		reservationID, created, err := accountUsage.ReserveBusinessAccountInferenceCreditsWithID(ctx, businessAccountID, actorIdentity, tier, appBundleKey, amount, operationID)
		if err != nil {
			return "", err
		}
		if !created {
			if proof != nil {
				return "", ErrBrowserEvidenceReplay
			}
			return "", ErrInferenceOperationReplay
		}
		return reservationID, nil
	}
	return accountUsage.ReserveBusinessAccountCredits(ctx, businessAccountID, actorIdentity, tier, "ai_credits", amount)
}

func (s *MeteredInferenceService) releaseReservation(ctx context.Context, actorIdentity, businessAccountID, reservationID string) error {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), reservationCleanupTimeout)
	defer cancel()
	var err error
	if strings.TrimSpace(businessAccountID) == "" {
		err = s.usageService.ReleaseReservation(cleanupCtx, reservationID)
	} else if accountUsage, ok := s.usageService.(BusinessAccountUsageServicer); !ok {
		err = errors.New("business-account usage authority is unavailable")
	} else {
		err = accountUsage.ReleaseBusinessAccountReservation(cleanupCtx, businessAccountID, actorIdentity, reservationID)
	}
	if err != nil {
		s.log("metered_inference_reservation_release_failed", map[string]interface{}{
			"level": "error", "reservation_id": reservationID, "error_type": fmt.Sprintf("%T", err), "security": true,
		})
	}
	return err
}

func (s *MeteredInferenceService) markInferenceOutcomeUncertain(actorIdentity, businessAccountID, reservationID, operationID string) error {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), reservationCleanupTimeout)
	defer cancel()
	var retainErr error
	if strings.TrimSpace(businessAccountID) != "" {
		accountUsage, ok := s.usageService.(BusinessAccountUsageServicer)
		if !ok {
			retainErr = errors.New("business-account reservation retainer is unavailable")
		} else {
			retainErr = accountUsage.RetainUncertainBusinessAccountReservation(cleanupCtx, businessAccountID, actorIdentity, reservationID)
		}
	} else if retainer, ok := s.usageService.(UncertainReservationRetainer); ok {
		retainErr = retainer.RetainUncertainReservation(cleanupCtx, reservationID)
	} else {
		retainErr = errors.New("reservation retainer is unavailable")
	}
	if retainErr != nil {
		s.log("metered_inference_uncertain_reservation_retain_failed", map[string]interface{}{
			"level": "error", "reservation_id": reservationID, "operation_id": operationID,
			"error_type": fmt.Sprintf("%T", retainErr), "security": true,
		})
		return errors.Join(ErrInferenceOutcomeUncertain, ErrInferenceReservationRetentionFailed)
	}
	s.log("metered_inference_uncertain_reservation_retained", map[string]interface{}{
		"level": "warn", "reservation_id": reservationID, "operation_id": operationID,
	})
	return ErrInferenceOutcomeUncertain
}

func (s *MeteredInferenceService) finalizeReservationWithCost(ctx context.Context, actorIdentity, businessAccountID, reservationID string, actualAmount int64, appBundleKey, model string, costMicros int64, promptTokens, completionTokens int, provider string, browser BrowserUsage) error {
	if strings.TrimSpace(businessAccountID) == "" {
		return finalizeUsageWithBrowserUsage(s.usageService, ctx, reservationID, actualAmount, appBundleKey, model, costMicros, promptTokens, completionTokens, provider, browser)
	}
	accountUsage, ok := s.usageService.(BusinessAccountUsageServicer)
	if !ok {
		return errors.New("business-account usage authority is unavailable")
	}
	return accountUsage.FinalizeBusinessAccountReservationWithBrowserUsage(ctx, businessAccountID, actorIdentity, reservationID, actualAmount, appBundleKey, model, costMicros, promptTokens, completionTokens, provider, browser.SessionMinutes, browser.Actions, browser.Bytes, browser.RunID, browser.SessionID, browser.EvidenceFingerprint)
}

func (s *MeteredInferenceService) recordUsage(ctx context.Context, actorIdentity, businessAccountID, tier string, report UsageReport) error {
	if strings.TrimSpace(businessAccountID) == "" {
		return s.usageService.RecordUsage(ctx, report)
	}
	accountUsage, ok := s.usageService.(BusinessAccountUsageServicer)
	if !ok {
		return errors.New("business-account usage authority is unavailable")
	}
	operationID := ""
	if report.OperationID != nil {
		operationID = *report.OperationID
	}
	return accountUsage.RecordBusinessAccountUsage(ctx, businessAccountID, actorIdentity, tier, report.LimitKey, report.Amount, operationID, report.AppBundleKey, report.Metadata["model"], parseNonnegativeInt64(report.Metadata["cost_micros"]), parseNonnegativeInt(report.Metadata["prompt_tokens"]), parseNonnegativeInt(report.Metadata["completion_tokens"]), report.Metadata["provider"])
}

func (s *MeteredInferenceService) adjustUsage(ctx context.Context, actorIdentity, businessAccountID, limitKey string, amount int64, reason string) error {
	if strings.TrimSpace(businessAccountID) == "" {
		return s.usageService.AdjustUsage(ctx, actorIdentity, limitKey, amount, reason)
	}
	accountUsage, ok := s.usageService.(BusinessAccountUsageServicer)
	if !ok {
		return errors.New("business-account usage authority is unavailable")
	}
	return accountUsage.AdjustBusinessAccountUsage(ctx, businessAccountID, limitKey, amount)
}

func parseNonnegativeInt64(value string) int64 {
	parsed, _ := strconv.ParseInt(value, 10, 64)
	if parsed < 0 {
		return 0
	}
	return parsed
}

func parseNonnegativeInt(value string) int {
	parsed, _ := strconv.Atoi(value)
	if parsed < 0 {
		return 0
	}
	return parsed
}

// getUserTier retrieves the user's subscription tier.
// Returns "free" if the tier cannot be determined.
// Logs security events when defaulting to free tier unexpectedly.
func (s *MeteredInferenceService) getUserTier(ctx context.Context, userIdentity string) (string, error) {
	if s.accountService == nil {
		// Log this as it might indicate misconfiguration
		s.log("tier_lookup_no_account_service", map[string]interface{}{
			"level":         "warn",
			"user_identity": userIdentity,
			"default_tier":  "free",
			"security":      true,
		})
		return "free", nil
	}

	sub, err := s.accountService.GetSubscriptionContext(ctx, userIdentity)
	if err != nil {
		// Log with security tag - this could indicate tier bypass attempt or service issue
		s.log("tier_lookup_failed_defaulting_to_free", map[string]interface{}{
			"level":         "warn",
			"user_identity": userIdentity,
			"error":         err.Error(),
			"security":      true,
		})
		return "free", nil
	}

	if sub == nil || sub.PlanTier == nil || *sub.PlanTier == "" {
		return "free", nil
	}

	return *sub.PlanTier, nil
}

// tokenEstimate holds prompt and completion token estimates.
type tokenEstimate struct {
	prompt     int
	completion int
}

// estimateTokens estimates token counts for a request.
// This is a rough approximation used for pre-authorization.
// Includes a safety margin to reduce the frequency of underestimation,
// which leads to smoother UX with fewer post-request adjustments.
func (s *MeteredInferenceService) estimateTokens(messages []AIMessage, maxTokens int) tokenEstimate {
	estimate := EstimateTokens(messages, maxTokens)
	return tokenEstimate{prompt: estimate.Prompt, completion: estimate.Completion}
}

func (s *MeteredInferenceService) estimateInferenceTokens(req AIRequest) tokenEstimate {
	estimate := EstimateInferenceTokens(req.Messages, req.Tools, req.ToolChoice, req.MaxTokens)
	return tokenEstimate{prompt: estimate.Prompt, completion: estimate.Completion}
}

// calculateCost calculates the cost in internal units for a request.
// Internal unit = 1/1,000,000 of a cent.
func (s *MeteredInferenceService) calculateCost(model string, promptTokens, completionTokens int) int64 {
	return CalculateCost(s.modelPricing, model, promptTokens, completionTokens)
}

// GetAvailableModels returns the list of models available through the gateway.
func (s *MeteredInferenceService) GetAvailableModels() []string {
	models := make([]string, 0, len(allowedModels))
	for model := range allowedModels {
		models = append(models, model)
	}
	return models
}

// Compile-time interface check
var _ MeteredInferenceProvider = (*MeteredInferenceService)(nil)
