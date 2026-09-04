package intelligence

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

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
//   - If stream fails, reservation is released (no usage recorded)
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
	modelPricing map[string]ModelPricing
	rolePolicies map[string]RolePolicy
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
		apiKeyService:    opts.APIKeyService,
		usageService:     opts.UsageService,
		accountService:   opts.AccountService,
		log:              logger,
		clientFactory:    opts.ClientFactory,
		openRouterClient: opts.OpenRouterClient,
		modelPricing:     defaultModelPricing(),
		rolePolicies:     DefaultRolePolicies(),
	}
}

// ExecuteInference applies the stable role contract and delegates to the
// atomic reservation/provider pipeline. Concrete model selection is internal.
func (s *MeteredInferenceService) ExecuteInference(ctx context.Context, userIdentity string, req InferenceRequest) (*AIResponse, error) {
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
		Model: policy.Model, Messages: messages, MaxTokens: req.MaxTokens, Metadata: req.Metadata,
	})
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
	// Validate model
	if !allowedModels[req.Model] {
		return nil, fmt.Errorf("%w: %s", ErrModelNotAllowed, req.Model)
	}

	// Get user's subscription tier
	tier, err := s.getUserTier(ctx, userIdentity)
	if err != nil {
		s.log("metered_inference_get_tier_failed", map[string]interface{}{
			"level":         "error",
			"user_identity": userIdentity,
			"error":         err.Error(),
		})
		return nil, fmt.Errorf("failed to get user tier: %w", err)
	}

	// Estimate cost for pre-check
	estimate := s.estimateTokens(req.Messages, req.MaxTokens)
	estimatedCost := s.calculateCost(req.Model, estimate.prompt, estimate.completion)

	// Reserve credits before the provider call; the reservation has an explicit
	// release path so provider failure cannot commit the estimate.
	reservationID, err := s.usageService.ReserveCredits(ctx, userIdentity, tier, "ai_credits", estimatedCost)
	if err != nil {
		if errors.Is(err, ErrInsufficientCredits) || strings.Contains(err.Error(), "insufficient") {
			return nil, ErrInsufficientCredits
		}
		return nil, fmt.Errorf("credit check failed: %w", err)
	}

	// Get OpenRouter client
	client, err := s.getOpenRouterClient(ctx)
	if err != nil {
		_ = s.usageService.ReleaseReservation(ctx, reservationID)
		return nil, err
	}

	// Convert messages to OpenRouter format
	orMessages := make([]OpenRouterMessage, len(req.Messages))
	for i, msg := range req.Messages {
		orMessages[i] = OpenRouterMessage(msg)
	}

	// Execute the chat request
	orResp, err := client.Chat(ctx, OpenRouterChatRequest{
		Model:    req.Model,
		Messages: orMessages,
	})
	if err != nil {
		_ = s.usageService.ReleaseReservation(ctx, reservationID)
		return nil, fmt.Errorf("openrouter request failed: %w", err)
	}

	// Calculate actual cost based on real token usage
	actualCost := s.calculateCost(req.Model, orResp.Usage.PromptTokens, orResp.Usage.CompletionTokens)

	/*
			Legacy estimate adjustment is intentionally disabled: the reservation
			finalizer below owns the single authoritative charge/refund operation.
			Keeping a second adjustment here would double-charge successful requests.
		if false {
			if actualCost > estimatedCost {
				// Charge additional credits if actual cost exceeds estimate
				extraCost := actualCost - estimatedCost
				if err := s.usageService.RecordUsage(ctx, UsageReport{
					UserIdentity: userIdentity,
					LimitKey:     "ai_credits",
					Amount:       extraCost,
					AppBundleKey: req.Metadata.AppBundleKey,
					Operation:    req.Metadata.Operation,
					Metadata:     map[string]string{"type": "cost_adjustment"},
				}); err != nil {
					s.log("cost_adjustment_charge_failed", map[string]interface{}{
						"level":         "warn",
						"user_identity": userIdentity,
						"extra_cost":    extraCost,
						"error":         err.Error(),
					})
				}
			} else if actualCost < estimatedCost {
				// Refund overage when actual cost is less than estimated
				refundAmount := estimatedCost - actualCost
				refundPercent := float64(refundAmount) / float64(estimatedCost) * 100

				// Log refund with security flag if unusually large (>50% indicates estimation drift)
				logLevel := "debug"
				if refundPercent > 50 {
					logLevel = "warn"
				}
				s.log("credit_refund_issued", map[string]interface{}{
					"level":          logLevel,
					"user_identity":  userIdentity,
					"refund_amount":  refundAmount,
					"refund_percent": refundPercent,
					"estimated_cost": estimatedCost,
					"actual_cost":    actualCost,
					"security":       refundPercent > 50,
				})

				if err := s.usageService.AdjustUsage(ctx, userIdentity, "ai_credits", -refundAmount, "estimate_refund"); err != nil {
					s.log("refund_adjustment_failed", map[string]interface{}{
						"level":          "warn",
						"user_identity":  userIdentity,
						"refund_amount":  refundAmount,
						"estimated_cost": estimatedCost,
						"actual_cost":    actualCost,
						"error":          err.Error(),
					})
				}
			}
		}
	*/
	if err := s.usageService.FinalizeReservation(ctx, reservationID, actualCost); err != nil {
		return nil, fmt.Errorf("finalize credit reservation: %w", err)
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
		ID:               orResp.ID,
		Model:            orResp.Model,
		Content:          orResp.Content,
		PromptTokens:     orResp.Usage.PromptTokens,
		CompletionTokens: orResp.Usage.CompletionTokens,
		TotalTokens:      orResp.Usage.TotalTokens,
		CreditsCharged:   actualCost,
		FinishReason:     orResp.FinishReason,
	}, nil
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
