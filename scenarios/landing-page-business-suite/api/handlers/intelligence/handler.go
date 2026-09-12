// Package intelligence owns the HTTP transport edge for the credit-accounted metered inference provider.
package intelligence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"landing-page-business-suite-api/internal/intelligence"
)

// Limiter is the narrow rate-limit policy required at the HTTP boundary.
// seam: Limiter lets handler tests exercise rejection paths without global rate state.
type Limiter interface {
	Allow(string) bool
}

// Dependencies are supplied by API composition. The handler package never
// reads authentication context, creates rate limiters, or reaches root helpers.
type Dependencies struct {
	Service          intelligence.MeteredInferenceProvider
	Usage            func(context.Context, string, string) (UsageSummary, error)
	SubscriptionTier func(context.Context, string) (string, error)
	UserRateLimiter  Limiter
	IPRateLimiter    Limiter
	IPKeyFunc        func(*http.Request) string
	UserIdentity     func(context.Context) string
	WriteJSONError   func(http.ResponseWriter, int, string, string)
	Log              func(string, map[string]interface{})
	LogError         func(string, map[string]interface{})
}

// UsageSummary is the transport-neutral usage view needed by AI endpoints.
// Composition converts the commerce model before it reaches this boundary.
type UsageSummary struct {
	BillingPeriod string
	ResetDate     time.Time
	Usage         map[string]int64
	Limits        map[string]int64
	Remaining     map[string]int64
}

// Handler translates AI HTTP requests to the intelligence domain.
type Handler struct{ deps Dependencies }

func New(deps Dependencies) *Handler { return &Handler{deps: deps} }

// handleAIChat handles non-streaming AI chat completion requests.
// POST /api/v1/ai/chat
//
// Responsibilities:
//   - Authentication (via middleware that sets user in context)
//   - Rate limiting
//   - Request validation (format, bounds)
//   - Delegating to the intelligence metered inference service
//   - Mapping errors to HTTP responses
//
// The handler does NOT handle credit checking - that's the service's responsibility.
func (h *Handler) Chat() http.HandlerFunc {
	deps := h.deps
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Get user from context (JWT auth handled by middleware)
		userIdentity := deps.UserIdentity(r.Context())
		if userIdentity == "" {
			deps.WriteJSONError(w, http.StatusUnauthorized, "Authentication required", "unauthorized")
			return
		}

		// 2. Rate limiting (per-user)
		if !deps.UserRateLimiter.Allow(userIdentity) {
			deps.WriteJSONError(w, http.StatusTooManyRequests, "Rate limit exceeded. Please try again later.", "rate_limited")
			return
		}

		// 3. IP-based rate limiting (defense in depth against multi-account abuse)
		clientIP := deps.IPKeyFunc(r)
		if !deps.IPRateLimiter.Allow(clientIP) {
			deps.Log("ai_rate_limit_ip_exceeded", map[string]interface{}{
				"level":         "warn",
				"client_ip":     clientIP,
				"user_identity": userIdentity,
				"security":      true,
			})
			deps.WriteJSONError(w, http.StatusTooManyRequests, "Rate limit exceeded.", "rate_limited")
			return
		}

		// 4. Decode request
		var req intelligence.AIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			deps.WriteJSONError(w, http.StatusBadRequest, "Invalid request body", "validation")
			return
		}

		// 5. Validate request (handler owns input validation)
		if err := ValidateRequest(&req); err != nil {
			deps.WriteJSONError(w, http.StatusBadRequest, err.Error(), "validation")
			return
		}

		// 6. Execute chat completion (service handles business logic + credits)
		resp, err := deps.Service.ExecuteChat(r.Context(), userIdentity, req)
		if err != nil {
			WriteError(deps, w, err)
			return
		}

		// 7. Return response
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			deps.LogError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// Inference accepts the role-based metered contract. It intentionally has no
// model field: provider selection and pricing are server-owned policy.
func (h *Handler) Inference() http.HandlerFunc {
	deps := h.deps
	return func(w http.ResponseWriter, r *http.Request) {
		identity := deps.UserIdentity(r.Context())
		if identity == "" {
			deps.WriteJSONError(w, http.StatusUnauthorized, "Authentication required", "unauthorized")
			return
		}
		if !deps.UserRateLimiter.Allow(identity) || !deps.IPRateLimiter.Allow(deps.IPKeyFunc(r)) {
			deps.WriteJSONError(w, http.StatusTooManyRequests, "Rate limit exceeded. Please try again later.", "rate_limited")
			return
		}
		var req intelligence.InferenceRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			deps.WriteJSONError(w, http.StatusBadRequest, "Invalid request body", "validation")
			return
		}
		if err := ValidateInferenceRequest(&req); err != nil {
			deps.WriteJSONError(w, http.StatusBadRequest, err.Error(), "validation")
			return
		}
		service, ok := deps.Service.(*intelligence.MeteredInferenceService)
		if !ok {
			deps.WriteJSONError(w, http.StatusServiceUnavailable, "metered inference provider is unavailable", "unavailable")
			return
		}
		response, err := service.ExecuteInference(r.Context(), identity, req)
		if err != nil {
			WriteError(deps, w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		// Do not expose the policy-resolved provider model to consumers.
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": response.ID, "role": req.Role, "content": response.Content,
			"prompt_tokens": response.PromptTokens, "completion_tokens": response.CompletionTokens,
			"total_tokens": response.TotalTokens, "credits_charged": response.CreditsCharged,
			"finish_reason": response.FinishReason,
		})
	}
}

func ValidateInferenceRequest(req *intelligence.InferenceRequest) error {
	if strings.TrimSpace(req.Role) == "" {
		return errors.New("role is required")
	}
	if len(req.Messages) == 0 {
		return errors.New("at least one message is required")
	}
	for _, message := range req.Messages {
		if strings.TrimSpace(message.Role) == "" || strings.TrimSpace(message.Content) == "" {
			return errors.New("message role and content are required")
		}
	}
	if req.MaxTokens < 0 || req.MaxTokens > maxMaxTokens {
		return fmt.Errorf("max_tokens must be between 0 and %d", maxMaxTokens)
	}
	if strings.TrimSpace(req.ConstraintsJSON) != "" && !json.Valid([]byte(req.ConstraintsJSON)) {
		return errors.New("constraints_json must be valid JSON")
	}
	return nil
}

// handleAIStream handles streaming AI chat completion requests via Server-Sent Events.
// POST /api/v1/ai/stream
//
// SSE Response Format:
//   - Content chunks:  data: {"type":"chunk","content":"partial text"}\n\n
//   - Completion:      data: {"type":"done","usage":{...}}\n\n
//   - Errors:          data: {"type":"error","error":"message"}\n\n
func (h *Handler) Stream() http.HandlerFunc {
	deps := h.deps
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Get user from context (JWT auth)
		userIdentity := deps.UserIdentity(r.Context())
		if userIdentity == "" {
			deps.WriteJSONError(w, http.StatusUnauthorized, "Authentication required", "unauthorized")
			return
		}

		// 2. Rate limiting (per-user)
		if !deps.UserRateLimiter.Allow(userIdentity) {
			deps.WriteJSONError(w, http.StatusTooManyRequests, "Rate limit exceeded. Please try again later.", "rate_limited")
			return
		}

		// 3. IP-based rate limiting (defense in depth against multi-account abuse)
		clientIP := deps.IPKeyFunc(r)
		if !deps.IPRateLimiter.Allow(clientIP) {
			deps.Log("ai_rate_limit_ip_exceeded", map[string]interface{}{
				"level":         "warn",
				"client_ip":     clientIP,
				"user_identity": userIdentity,
				"security":      true,
			})
			deps.WriteJSONError(w, http.StatusTooManyRequests, "Rate limit exceeded.", "rate_limited")
			return
		}

		// 4. Decode request
		var req intelligence.AIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			deps.WriteJSONError(w, http.StatusBadRequest, "Invalid request body", "validation")
			return
		}

		// 5. Validate request
		if err := ValidateRequest(&req); err != nil {
			deps.WriteJSONError(w, http.StatusBadRequest, err.Error(), "validation")
			return
		}

		// Force streaming mode
		req.Stream = true

		// 6. Execute streaming chat completion
		if err := deps.Service.ExecuteChatStream(r.Context(), userIdentity, req, w); err != nil {
			// If headers haven't been written yet, send error as JSON
			// Otherwise, send as SSE error event
			if w.Header().Get("Content-Type") != "text/event-stream" {
				WriteError(deps, w, err)
			} else {
				// Send error as SSE event
				errorEvent := intelligence.AIStreamEvent{
					Type:  "error",
					Error: err.Error(),
				}
				eventData, _ := json.Marshal(errorEvent)
				_, _ = w.Write([]byte("data: "))
				_, _ = w.Write(eventData)
				_, _ = w.Write([]byte("\n\n"))
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
func (h *Handler) Models() http.HandlerFunc {
	deps := h.deps
	return func(w http.ResponseWriter, r *http.Request) {
		models := deps.Service.GetAvailableModels()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"models": models,
		}); err != nil {
			deps.LogError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// handleAIUsage returns AI usage statistics for the current user.
// GET /api/v1/ai/usage
// Requires user authentication.
func (h *Handler) Usage() http.HandlerFunc {
	deps := h.deps
	return func(w http.ResponseWriter, r *http.Request) {
		userIdentity := deps.UserIdentity(r.Context())
		if userIdentity == "" {
			deps.WriteJSONError(w, http.StatusUnauthorized, "Authentication required", "unauthorized")
			return
		}

		// Get the user's tier
		tier := resolveTier(r.Context(), userIdentity, deps.SubscriptionTier)

		// Get usage summary
		if deps.Usage == nil {
			deps.WriteJSONError(w, http.StatusInternalServerError, "Usage service is not configured", "server_error")
			return
		}
		summary, err := deps.Usage(r.Context(), userIdentity, tier)
		if err != nil {
			deps.LogError("ai_usage_fetch_failed", map[string]interface{}{
				"error":         err.Error(),
				"user_identity": userIdentity,
			})
			deps.WriteJSONError(w, http.StatusInternalServerError, "Failed to retrieve usage data", "server_error")
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
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
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
		}); err != nil {
			deps.LogError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func resolveTier(ctx context.Context, identity string, lookup func(context.Context, string) (string, error)) string {
	if lookup == nil {
		return "free"
	}
	tier, err := lookup(ctx, identity)
	if err != nil || tier == "" {
		return "free"
	}
	return tier
}

// handleAIHealth checks if the metered inference provider is healthy.
// GET /api/v1/ai/health
// Public endpoint - no authentication required.
func (h *Handler) Health() http.HandlerFunc {
	deps := h.deps
	return func(w http.ResponseWriter, r *http.Request) {
		err := deps.Service.HealthCheck(r.Context())

		w.Header().Set("Content-Type", "application/json")

		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			if encErr := json.NewEncoder(w).Encode(map[string]interface{}{
				"healthy": false,
				"error":   err.Error(),
			}); encErr != nil {
				deps.LogError("encode_response_failed", map[string]interface{}{"error": encErr.Error()})
			}
			return
		}

		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"healthy": true,
		}); err != nil {
			deps.LogError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
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
func ValidateRequest(req *intelligence.AIRequest) error {
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
func WriteError(deps Dependencies, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, intelligence.ErrInsufficientCredits):
		deps.WriteJSONError(w, http.StatusPaymentRequired, "Insufficient credits for this operation", "insufficient_credits")
	case errors.Is(err, intelligence.ErrNoAPIKeyConfigured):
		deps.WriteJSONError(w, http.StatusServiceUnavailable, "AI service not configured", "server_error")
	case errors.Is(err, intelligence.ErrModelNotAllowed):
		deps.WriteJSONError(w, http.StatusBadRequest, err.Error(), "validation")
	case errors.Is(err, intelligence.ErrProvider):
		deps.WriteJSONError(w, http.StatusBadGateway, "AI provider error", "server_error")
	case errors.Is(err, intelligence.ErrStreamingNotSupported):
		deps.WriteJSONError(w, http.StatusNotImplemented, "Streaming not supported", "server_error")
	default:
		deps.LogError("metered_inference_error", map[string]interface{}{
			"error": err.Error(),
		})
		deps.WriteJSONError(w, http.StatusInternalServerError, "AI request failed", "server_error")
	}
}
