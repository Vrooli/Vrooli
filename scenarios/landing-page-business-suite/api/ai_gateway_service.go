package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// AIGatewayService handles AI requests through the LPBS gateway.
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
type AIGatewayService struct {
	db             *sql.DB
	apiKeyService  *APIKeyService
	usageService   *UsageService
	accountService *AccountService
	limitsService  *LimitsService
	log            func(event string, fields map[string]interface{})

	// OpenRouter client - injectable for testing
	openRouterClient OpenRouterClient

	// Model pricing in internal units per 1K tokens.
	// Internal unit = 1/1,000,000 of a cent.
	modelPricing map[string]ModelPricing
}

// ModelPricing defines the cost per 1K tokens for a model.
// Costs are in internal units (1/1,000,000 of a cent).
type ModelPricing struct {
	PromptCostPer1K     int64 `json:"prompt_cost_per_1k"`     // Cost per 1K input tokens
	CompletionCostPer1K int64 `json:"completion_cost_per_1k"` // Cost per 1K output tokens
}

// AIGatewayServiceOptions configures the AI gateway service.
type AIGatewayServiceOptions struct {
	DB             *sql.DB
	APIKeyService  *APIKeyService
	UsageService   *UsageService
	AccountService *AccountService
	LimitsService  *LimitsService
	Logger         func(event string, fields map[string]interface{})

	// OpenRouterClient allows injecting a custom client for testing.
	// If nil, a real HTTP client is created when the API key is first needed.
	OpenRouterClient OpenRouterClient
}

// AIRequest is the request to execute an AI chat completion.
type AIRequest struct {
	Model     string      `json:"model"`
	Messages  []AIMessage `json:"messages"`
	Stream    bool        `json:"stream"`
	MaxTokens int         `json:"max_tokens,omitempty"`
	Metadata  AIMetadata  `json:"metadata,omitempty"`
}

// AIMessage represents a chat message.
type AIMessage struct {
	Role    string `json:"role"` // "user", "assistant", "system"
	Content string `json:"content"`
}

// AIMetadata contains optional metadata about the request for tracking.
type AIMetadata struct {
	AppBundleKey string `json:"app_bundle_key,omitempty"` // e.g., "browser-automation-studio"
	Operation    string `json:"operation,omitempty"`      // e.g., "ai.analysis", "ai.workflow_generate"
}

// AIResponse is the response from an AI chat completion.
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

// AIStreamEvent represents a Server-Sent Event for streaming responses.
type AIStreamEvent struct {
	Type    string `json:"type"`              // "chunk", "done", "error"
	Content string `json:"content,omitempty"` // For chunk events
	Error   string `json:"error,omitempty"`   // For error events
	Usage   *struct {
		PromptTokens     int   `json:"prompt_tokens"`
		CompletionTokens int   `json:"completion_tokens"`
		TotalTokens      int   `json:"total_tokens"`
		CreditsCharged   int64 `json:"credits_charged"`
	} `json:"usage,omitempty"` // For done events
}

// allowedModels is the whitelist of models users can request.
var allowedModels = map[string]bool{
	"openai/gpt-4o":               true,
	"openai/gpt-4o-mini":          true,
	"anthropic/claude-3.5-sonnet": true,
	"anthropic/claude-3-haiku":    true,
	"google/gemini-pro-1.5":       true,
	"google/gemini-flash-1.5":     true,
}

// Token estimation constants with safety margin to reduce underestimation frequency.
const (
	charsPerToken               = 4     // Approximate characters per token
	tokenEstimationSafetyMargin = 1.5   // 50% buffer to reduce underestimation
	defaultCompletionTokens     = 1000  // Default expected completion length
	messageFramingOverhead      = 10    // Overhead per message for role/framing
)

// NewAIGatewayService creates a new AI gateway service.
func NewAIGatewayService(opts AIGatewayServiceOptions) *AIGatewayService {
	logger := opts.Logger
	if logger == nil {
		logger = logStructured
	}

	return &AIGatewayService{
		db:               opts.DB,
		apiKeyService:    opts.APIKeyService,
		usageService:     opts.UsageService,
		accountService:   opts.AccountService,
		limitsService:    opts.LimitsService,
		log:              logger,
		openRouterClient: opts.OpenRouterClient,
		modelPricing:     defaultModelPricing(),
	}
}

// defaultModelPricing returns default pricing for common models.
// Prices are in internal units (1/1,000,000 of a cent) per 1K tokens.
// Based on OpenRouter prices as of early 2025.
func defaultModelPricing() map[string]ModelPricing {
	return map[string]ModelPricing{
		// GPT-4o family
		"openai/gpt-4o":      {PromptCostPer1K: 2500000, CompletionCostPer1K: 10000000}, // $2.50/$10 per 1M tokens
		"openai/gpt-4o-mini": {PromptCostPer1K: 150000, CompletionCostPer1K: 600000},    // $0.15/$0.60 per 1M tokens
		// Claude family
		"anthropic/claude-3.5-sonnet": {PromptCostPer1K: 3000000, CompletionCostPer1K: 15000000}, // $3/$15 per 1M
		"anthropic/claude-3-haiku":    {PromptCostPer1K: 250000, CompletionCostPer1K: 1250000},   // $0.25/$1.25 per 1M
		// Gemini family
		"google/gemini-pro-1.5":   {PromptCostPer1K: 1250000, CompletionCostPer1K: 5000000}, // $1.25/$5 per 1M
		"google/gemini-flash-1.5": {PromptCostPer1K: 75000, CompletionCostPer1K: 300000},    // $0.075/$0.30 per 1M
		// Default fallback for unknown models
		"default": {PromptCostPer1K: 1000000, CompletionCostPer1K: 2000000},
	}
}

// ExecuteChat executes a non-streaming chat completion.
// Credits are checked and charged atomically to prevent TOCTOU race conditions.
func (s *AIGatewayService) ExecuteChat(ctx context.Context, userIdentity string, req AIRequest) (*AIResponse, error) {
	// Validate model
	if !allowedModels[req.Model] {
		return nil, fmt.Errorf("%w: %s", ErrModelNotAllowed, req.Model)
	}

	// Get user's subscription tier
	tier, err := s.getUserTier(ctx, userIdentity)
	if err != nil {
		s.log("ai_gateway_get_tier_failed", map[string]interface{}{
			"level":         "error",
			"user_identity": userIdentity,
			"error":         err.Error(),
		})
		return nil, fmt.Errorf("failed to get user tier: %w", err)
	}

	// Estimate cost for pre-check
	estimate := s.estimateTokens(req.Messages, req.MaxTokens)
	estimatedCost := s.calculateCost(req.Model, estimate.prompt, estimate.completion)

	// Reserve credits atomically (check + tentative charge)
	if err := s.usageService.ReserveAndCharge(ctx, userIdentity, tier, "ai_credits", estimatedCost, UsageReportRequest{
		UserIdentity: userIdentity,
		LimitKey:     "ai_credits",
		Amount:       estimatedCost,
		AppBundleKey: req.Metadata.AppBundleKey,
		Operation:    req.Metadata.Operation,
	}); err != nil {
		if errors.Is(err, ErrInsufficientCredits) || strings.Contains(err.Error(), "insufficient") {
			return nil, ErrInsufficientCredits
		}
		return nil, fmt.Errorf("credit check failed: %w", err)
	}

	// Get OpenRouter client
	client, err := s.getOpenRouterClient(ctx)
	if err != nil {
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
		return nil, fmt.Errorf("openrouter request failed: %w", err)
	}

	// Calculate actual cost based on real token usage
	actualCost := s.calculateCost(req.Model, orResp.Usage.PromptTokens, orResp.Usage.CompletionTokens)

	// Adjust credits if actual cost differs from estimate
	if actualCost > estimatedCost {
		// Charge additional credits if actual cost exceeds estimate
		extraCost := actualCost - estimatedCost
		if err := s.usageService.RecordUsage(ctx, UsageReportRequest{
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

	s.log("ai_gateway_request_completed", map[string]interface{}{
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

// ExecuteChatStream executes a streaming chat completion via Server-Sent Events.
// Uses credit reservations to prevent TOCTOU race conditions where concurrent
// streaming requests could exceed the credit limit.
func (s *AIGatewayService) ExecuteChatStream(ctx context.Context, userIdentity string, req AIRequest, w http.ResponseWriter) error {
	// Validate model
	if !allowedModels[req.Model] {
		return fmt.Errorf("%w: %s", ErrModelNotAllowed, req.Model)
	}

	// Get user's subscription tier
	tier, err := s.getUserTier(ctx, userIdentity)
	if err != nil {
		return fmt.Errorf("failed to get user tier: %w", err)
	}

	// Estimate cost for reservation
	estimate := s.estimateTokens(req.Messages, req.MaxTokens)
	estimatedCost := s.calculateCost(req.Model, estimate.prompt, estimate.completion)

	// Reserve credits atomically (prevents TOCTOU race condition)
	reservationID, err := s.usageService.ReserveCredits(ctx, userIdentity, tier, "ai_credits", estimatedCost)
	if err != nil {
		if errors.Is(err, ErrInsufficientCredits) || strings.Contains(err.Error(), "insufficient") {
			return ErrInsufficientCredits
		}
		return fmt.Errorf("reserve credits failed: %w", err)
	}

	// Set up SSE headers
	flusher, ok := w.(http.Flusher)
	if !ok {
		// Release reservation on failure
		if err := s.usageService.ReleaseReservation(ctx, reservationID); err != nil {
			s.log("reservation_release_failed", map[string]interface{}{
				"level":          "warn",
				"user_identity":  userIdentity,
				"reservation_id": reservationID,
				"error":          err.Error(),
			})
		}
		return ErrStreamingNotSupported
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Get OpenRouter client
	client, err := s.getOpenRouterClient(ctx)
	if err != nil {
		// Release reservation on failure
		if releaseErr := s.usageService.ReleaseReservation(ctx, reservationID); releaseErr != nil {
			s.log("reservation_release_failed", map[string]interface{}{
				"level":          "warn",
				"user_identity":  userIdentity,
				"reservation_id": reservationID,
				"error":          releaseErr.Error(),
			})
		}
		return err
	}

	// Convert messages to OpenRouter format
	orMessages := make([]OpenRouterMessage, len(req.Messages))
	for i, msg := range req.Messages {
		orMessages[i] = OpenRouterMessage(msg)
	}

	// Stream response, forwarding chunks to client as SSE events
	usage, err := client.ChatStream(ctx, OpenRouterChatRequest{
		Model:    req.Model,
		Messages: orMessages,
	}, func(content string) {
		// Forward chunk to client as SSE event
		event := AIStreamEvent{
			Type:    "chunk",
			Content: content,
		}
		eventData, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", eventData)
		flusher.Flush()
	})
	if err != nil {
		// Release reservation on failure
		if releaseErr := s.usageService.ReleaseReservation(ctx, reservationID); releaseErr != nil {
			s.log("reservation_release_failed", map[string]interface{}{
				"level":          "warn",
				"user_identity":  userIdentity,
				"reservation_id": reservationID,
				"error":          releaseErr.Error(),
			})
		}
		return fmt.Errorf("streaming failed: %w", err)
	}

	// Use estimate for prompt tokens if not provided by stream
	promptTokens := usage.PromptTokens
	if promptTokens == 0 {
		promptTokens = estimate.prompt
	}

	// Calculate actual cost
	actualCost := s.calculateCost(req.Model, promptTokens, usage.CompletionTokens)

	// Finalize reservation with actual usage
	if err := s.usageService.FinalizeReservation(ctx, reservationID, actualCost); err != nil {
		s.log("finalize_reservation_failed", map[string]interface{}{
			"level":          "error",
			"user_identity":  userIdentity,
			"reservation_id": reservationID,
			"actual_cost":    actualCost,
			"error":          err.Error(),
		})
		// Fallback: try to record usage directly if finalize fails
		if fallbackErr := s.usageService.RecordUsage(ctx, UsageReportRequest{
			UserIdentity: userIdentity,
			LimitKey:     "ai_credits",
			Amount:       actualCost,
			AppBundleKey: req.Metadata.AppBundleKey,
			Operation:    req.Metadata.Operation,
		}); fallbackErr != nil {
			s.log("finalize_fallback_record_failed", map[string]interface{}{
				"level":          "error",
				"user_identity":  userIdentity,
				"reservation_id": reservationID,
				"actual_cost":    actualCost,
				"error":          fallbackErr.Error(),
				"security":       true,
			})
		}
	}

	// Send done event with usage info
	doneEvent := AIStreamEvent{
		Type: "done",
		Usage: &struct {
			PromptTokens     int   `json:"prompt_tokens"`
			CompletionTokens int   `json:"completion_tokens"`
			TotalTokens      int   `json:"total_tokens"`
			CreditsCharged   int64 `json:"credits_charged"`
		}{
			PromptTokens:     promptTokens,
			CompletionTokens: usage.CompletionTokens,
			TotalTokens:      promptTokens + usage.CompletionTokens,
			CreditsCharged:   actualCost,
		},
	}
	eventData, _ := json.Marshal(doneEvent)
	fmt.Fprintf(w, "data: %s\n\n", eventData)
	flusher.Flush()

	s.log("ai_gateway_stream_completed", map[string]interface{}{
		"level":             "info",
		"user_identity":     userIdentity,
		"model":             req.Model,
		"prompt_tokens":     promptTokens,
		"completion_tokens": usage.CompletionTokens,
		"credits_charged":   actualCost,
		"reservation_id":    reservationID,
	})

	return nil
}

// getOpenRouterClient returns the OpenRouter client, creating one if necessary.
func (s *AIGatewayService) getOpenRouterClient(ctx context.Context) (OpenRouterClient, error) {
	// If a client was injected (e.g., for testing), use it
	if s.openRouterClient != nil {
		return s.openRouterClient, nil
	}

	// Otherwise, create a real client using the stored API key
	apiKey, err := s.apiKeyService.Get(ctx, "openrouter")
	if err != nil {
		return nil, fmt.Errorf("get api key: %w", err)
	}
	if apiKey == "" {
		return nil, ErrNoAPIKeyConfigured
	}

	return NewOpenRouterClient(OpenRouterClientOptions{
		APIKey: apiKey,
		Logger: s.log,
	}), nil
}

// getUserTier retrieves the user's subscription tier.
// Returns "free" if the tier cannot be determined.
// Logs security events when defaulting to free tier unexpectedly.
func (s *AIGatewayService) getUserTier(ctx context.Context, userIdentity string) (string, error) {
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

	sub, err := s.accountService.GetSubscription(userIdentity)
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
func (s *AIGatewayService) estimateTokens(messages []AIMessage, maxTokens int) tokenEstimate {
	// Calculate raw prompt characters
	promptChars := 0
	for _, msg := range messages {
		promptChars += len(msg.Content) + len(msg.Role) + messageFramingOverhead
	}

	// Convert to tokens and apply safety margin
	rawPromptTokens := promptChars / charsPerToken
	promptTokens := int(float64(rawPromptTokens) * tokenEstimationSafetyMargin)

	// Estimate completion tokens
	completionTokens := defaultCompletionTokens
	if maxTokens > 0 {
		// User specified max_tokens - use it directly without multiplying
		// (they've explicitly set their limit)
		completionTokens = maxTokens
	} else {
		// Apply safety margin to default completion estimate
		completionTokens = int(float64(completionTokens) * tokenEstimationSafetyMargin)
	}

	return tokenEstimate{
		prompt:     promptTokens,
		completion: completionTokens,
	}
}

// calculateCost calculates the cost in internal units for a request.
// Internal unit = 1/1,000,000 of a cent.
func (s *AIGatewayService) calculateCost(model string, promptTokens, completionTokens int) int64 {
	pricing, ok := s.modelPricing[model]
	if !ok {
		pricing = s.modelPricing["default"]
	}

	// Cost per 1K tokens, so divide by 1000
	promptCost := int64(promptTokens) * pricing.PromptCostPer1K / 1000
	completionCost := int64(completionTokens) * pricing.CompletionCostPer1K / 1000

	return promptCost + completionCost
}

// GetAvailableModels returns the list of models available through the gateway.
func (s *AIGatewayService) GetAvailableModels() []string {
	models := make([]string, 0, len(allowedModels))
	for model := range allowedModels {
		models = append(models, model)
	}
	return models
}

// HealthCheck verifies the AI gateway can function.
func (s *AIGatewayService) HealthCheck(ctx context.Context) error {
	// Check if OpenRouter API key is configured
	client, err := s.getOpenRouterClient(ctx)
	if err != nil {
		return err
	}

	// Verify the key works
	return client.VerifyAPIKey(ctx)
}

// UseOpenRouterClient allows injecting a custom OpenRouter client.
// This is the primary testing seam for the AI gateway service.
func (s *AIGatewayService) UseOpenRouterClient(client OpenRouterClient) {
	s.openRouterClient = client
}

// Compile-time interface check
var _ AIGateway = (*AIGatewayService)(nil)
