package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// aiGatewayRateLimiter enforces 60 requests per minute per user for AI endpoints.
// Package-level for singleton behavior; use UseTimeProvider() in tests to control time.
var aiGatewayRateLimiter = NewRateLimiter(60, time.Minute)

// handleAIChat handles non-streaming AI chat completion requests.
// POST /api/v1/ai/chat
//
// Responsibilities:
//   - Authentication (via middleware that sets user in context)
//   - Rate limiting
//   - Request validation (format, bounds)
//   - Delegating to the AIGateway service
//   - Mapping errors to HTTP responses
//
// The handler does NOT handle credit checking - that's the service's responsibility.
func handleAIChat(svc AIGateway) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Get user from context (JWT auth handled by middleware)
		userIdentity := getUserEmail(r.Context())
		if userIdentity == "" {
			writeJSONError(w, http.StatusUnauthorized, "Authentication required", ApiErrorTypeUnauthorized)
			return
		}

		// 2. Rate limiting
		if !aiGatewayRateLimiter.Allow(userIdentity) {
			writeJSONError(w, http.StatusTooManyRequests, "Rate limit exceeded. Please try again later.", ApiErrorTypeRateLimited)
			return
		}

		// 3. Decode request
		var req AIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
			return
		}

		// 4. Validate request (handler owns input validation)
		if err := validateAIRequest(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error(), ApiErrorTypeValidation)
			return
		}

		// 5. Execute chat completion (service handles business logic + credits)
		resp, err := svc.ExecuteChat(r.Context(), userIdentity, req)
		if err != nil {
			handleAIError(w, err)
			return
		}

		// 6. Return response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// handleAIStream handles streaming AI chat completion requests via Server-Sent Events.
// POST /api/v1/ai/stream
//
// SSE Response Format:
//   - Content chunks:  data: {"type":"chunk","content":"partial text"}\n\n
//   - Completion:      data: {"type":"done","usage":{...}}\n\n
//   - Errors:          data: {"type":"error","error":"message"}\n\n
func handleAIStream(svc AIGateway) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Get user from context (JWT auth)
		userIdentity := getUserEmail(r.Context())
		if userIdentity == "" {
			writeJSONError(w, http.StatusUnauthorized, "Authentication required", ApiErrorTypeUnauthorized)
			return
		}

		// 2. Rate limiting
		if !aiGatewayRateLimiter.Allow(userIdentity) {
			writeJSONError(w, http.StatusTooManyRequests, "Rate limit exceeded. Please try again later.", ApiErrorTypeRateLimited)
			return
		}

		// 3. Decode request
		var req AIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
			return
		}

		// 4. Validate request
		if err := validateAIRequest(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error(), ApiErrorTypeValidation)
			return
		}

		// Force streaming mode
		req.Stream = true

		// 5. Execute streaming chat completion
		if err := svc.ExecuteChatStream(r.Context(), userIdentity, req, w); err != nil {
			// If headers haven't been written yet, send error as JSON
			// Otherwise, send as SSE error event
			if w.Header().Get("Content-Type") != "text/event-stream" {
				handleAIError(w, err)
			} else {
				// Send error as SSE event
				errorEvent := AIStreamEvent{
					Type:  "error",
					Error: err.Error(),
				}
				eventData, _ := json.Marshal(errorEvent)
				w.Write([]byte("data: "))
				w.Write(eventData)
				w.Write([]byte("\n\n"))
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
			}
			return
		}
	}
}

// handleAIModels returns the list of available AI models.
// GET /api/v1/ai/models
// Public endpoint - no authentication required.
func handleAIModels(svc AIGateway) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		models := svc.GetAvailableModels()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"models": models,
		})
	}
}

// handleAIUsage returns AI usage statistics for the current user.
// GET /api/v1/ai/usage
// Requires user authentication.
func handleAIUsage(svc AIGateway, usageSvc *UsageService, accountSvc *AccountService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIdentity := getUserEmail(r.Context())
		if userIdentity == "" {
			writeJSONError(w, http.StatusUnauthorized, "Authentication required", ApiErrorTypeUnauthorized)
			return
		}

		// Get the user's tier
		tier := "free"
		if accountSvc != nil {
			sub, err := accountSvc.GetSubscription(userIdentity)
			if err == nil && sub != nil && sub.PlanTier != nil {
				tier = *sub.PlanTier
			}
		}

		// Get usage summary
		summary, err := usageSvc.GetUsageSummary(r.Context(), userIdentity, tier)
		if err != nil {
			logStructuredError("ai_usage_fetch_failed", map[string]interface{}{
				"error":         err.Error(),
				"user_identity": userIdentity,
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to retrieve usage data", ApiErrorTypeServerError)
			return
		}

		// Extract AI credits specifically
		aiCreditsUsed := summary.Usage["ai_credits"]
		aiCreditsLimit := summary.Limits["ai_credits"]
		aiCreditsRemaining := summary.Remaining["ai_credits"]

		// Convert to display format.
		// Internal unit = 1/1,000,000 of a cent, so 100,000 internal units = $0.001
		// Dividing by 100,000 gives a user-friendly "credit" unit.
		displayUsed := float64(aiCreditsUsed) / 100000.0
		displayLimit := float64(aiCreditsLimit) / 100000.0
		displayRemaining := float64(aiCreditsRemaining) / 100000.0

		// Handle unlimited
		if aiCreditsLimit < 0 {
			displayLimit = -1
			displayRemaining = -1
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user_identity":        userIdentity,
			"tier":                 tier,
			"billing_period":       summary.BillingPeriod,
			"reset_date":           summary.ResetDate.Format(time.RFC3339),
			"ai_credits_used":      aiCreditsUsed,
			"ai_credits_limit":     aiCreditsLimit,
			"ai_credits_remaining": aiCreditsRemaining,
			"display": map[string]interface{}{
				"used":      displayUsed,
				"limit":     displayLimit,
				"remaining": displayRemaining,
				"unit":      "credits",
			},
		})
	}
}

// handleAIHealth checks if the AI gateway is healthy.
// GET /api/v1/ai/health
// Public endpoint - no authentication required.
func handleAIHealth(svc AIGateway) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := svc.HealthCheck(r.Context())

		w.Header().Set("Content-Type", "application/json")

		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"healthy": false,
				"error":   err.Error(),
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"healthy": true,
		})
	}
}

// AI Request Validation Constants
const (
	maxMessageLength = 100 * 1024 // 100KB per message
	maxMessages      = 100        // Maximum messages per request
	maxMaxTokens     = 16384      // Maximum value for max_tokens parameter
)

// validateAIRequest validates an AI request.
// This is presentation-layer validation (format, bounds) not business logic.
func validateAIRequest(req *AIRequest) error {
	if req.Model == "" {
		return errors.New("model is required")
	}

	if len(req.Messages) == 0 {
		return errors.New("at least one message is required")
	}

	// Validate message count
	if len(req.Messages) > maxMessages {
		return errors.New("too many messages (maximum 100)")
	}

	// Validate each message
	for i, msg := range req.Messages {
		if msg.Role == "" {
			return errors.New("message role is required")
		}
		if msg.Role != "user" && msg.Role != "assistant" && msg.Role != "system" {
			return errors.New("message role must be 'user', 'assistant', or 'system'")
		}
		if len(msg.Content) > maxMessageLength {
			return errors.New("message content exceeds maximum length (100KB)")
		}
		if i > 0 && msg.Content == "" {
			return errors.New("message content cannot be empty")
		}
	}

	// Validate max_tokens if provided
	if req.MaxTokens < 0 {
		return errors.New("max_tokens must be non-negative")
	}
	if req.MaxTokens > maxMaxTokens {
		return errors.New("max_tokens exceeds maximum (16384)")
	}

	return nil
}

// handleAIError maps AI service errors to HTTP responses.
// Centralizes error mapping to keep handlers thin.
func handleAIError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInsufficientCredits):
		writeJSONError(w, http.StatusPaymentRequired, "Insufficient credits for this operation", "insufficient_credits")
	case errors.Is(err, ErrNoAPIKeyConfigured):
		writeJSONError(w, http.StatusServiceUnavailable, "AI service not configured", ApiErrorTypeServerError)
	case errors.Is(err, ErrModelNotAllowed):
		writeJSONError(w, http.StatusBadRequest, err.Error(), ApiErrorTypeValidation)
	case errors.Is(err, ErrOpenRouterError):
		writeJSONError(w, http.StatusBadGateway, "AI provider error", ApiErrorTypeServerError)
	case errors.Is(err, ErrStreamingNotSupported):
		writeJSONError(w, http.StatusNotImplemented, "Streaming not supported", ApiErrorTypeServerError)
	default:
		logStructuredError("ai_gateway_error", map[string]interface{}{
			"error": err.Error(),
		})
		writeJSONError(w, http.StatusInternalServerError, "AI request failed", ApiErrorTypeServerError)
	}
}
