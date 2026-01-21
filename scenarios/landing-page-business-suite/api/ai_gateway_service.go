package main

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AI Gateway errors
var (
	ErrInsufficientCredits  = errors.New("insufficient credits for this operation")
	ErrNoAPIKeyConfigured   = errors.New("no OpenRouter API key configured")
	ErrModelNotAllowed      = errors.New("model not in allowed list")
	ErrAIGatewayUnavailable = errors.New("AI gateway service unavailable")
	ErrOpenRouterError      = errors.New("OpenRouter API error")
	ErrStreamingNotSupported = errors.New("streaming not supported by client")
)

// AIGatewayService handles AI requests through the LPBS gateway.
// It provides centralized AI access with credit management.
type AIGatewayService struct {
	db             *sql.DB
	apiKeyService  *APIKeyService
	usageService   *UsageService
	accountService *AccountService
	limitsService  *LimitsService
	httpClient     *http.Client
	log            func(event string, fields map[string]interface{})

	// Model pricing in internal units per 1K tokens
	// Internal unit = 1/1,000,000 of a cent
	// These are approximate costs for OpenRouter models
	modelPricing map[string]ModelPricing
}

// ModelPricing defines the cost per 1K tokens for a model
type ModelPricing struct {
	PromptCostPer1K     int64 `json:"prompt_cost_per_1k"`
	CompletionCostPer1K int64 `json:"completion_cost_per_1k"`
}

// AIGatewayServiceOptions configures the AI gateway service.
type AIGatewayServiceOptions struct {
	DB             *sql.DB
	APIKeyService  *APIKeyService
	UsageService   *UsageService
	AccountService *AccountService
	LimitsService  *LimitsService
	Logger         func(event string, fields map[string]interface{})
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
	Role    string `json:"role"`    // "user", "assistant", "system"
	Content string `json:"content"`
}

// AIMetadata contains optional metadata about the request.
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

// OpenRouter API types
type openRouterRequest struct {
	Model    string      `json:"model"`
	Messages []AIMessage `json:"messages"`
	Stream   bool        `json:"stream,omitempty"`
}

type openRouterResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type openRouterStreamChunk struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

// NewAIGatewayService creates a new AI gateway service.
func NewAIGatewayService(opts AIGatewayServiceOptions) *AIGatewayService {
	logger := opts.Logger
	if logger == nil {
		logger = logStructured
	}

	svc := &AIGatewayService{
		db:             opts.DB,
		apiKeyService:  opts.APIKeyService,
		usageService:   opts.UsageService,
		accountService: opts.AccountService,
		limitsService:  opts.LimitsService,
		httpClient:     &http.Client{Timeout: 120 * time.Second},
		log:            logger,
		modelPricing:   defaultModelPricing(),
	}

	return svc
}

// defaultModelPricing returns default pricing for common models.
// Prices are in internal units (1/1,000,000 of a cent) per 1K tokens.
// These are approximate OpenRouter prices as of 2024.
func defaultModelPricing() map[string]ModelPricing {
	return map[string]ModelPricing{
		// GPT-4o family
		"openai/gpt-4o":      {PromptCostPer1K: 2500000, CompletionCostPer1K: 10000000},  // $2.50/$10 per 1M
		"openai/gpt-4o-mini": {PromptCostPer1K: 150000, CompletionCostPer1K: 600000},     // $0.15/$0.60 per 1M
		// Claude family
		"anthropic/claude-3.5-sonnet": {PromptCostPer1K: 3000000, CompletionCostPer1K: 15000000},
		"anthropic/claude-3-haiku":    {PromptCostPer1K: 250000, CompletionCostPer1K: 1250000},
		// Gemini family
		"google/gemini-pro-1.5": {PromptCostPer1K: 1250000, CompletionCostPer1K: 5000000},
		"google/gemini-flash-1.5": {PromptCostPer1K: 75000, CompletionCostPer1K: 300000},
		// Default fallback for unknown models
		"default": {PromptCostPer1K: 1000000, CompletionCostPer1K: 2000000},
	}
}

// allowedModels returns the list of models users can request.
var allowedModels = map[string]bool{
	"openai/gpt-4o":               true,
	"openai/gpt-4o-mini":          true,
	"anthropic/claude-3.5-sonnet": true,
	"anthropic/claude-3-haiku":    true,
	"google/gemini-pro-1.5":       true,
	"google/gemini-flash-1.5":     true,
}

// ExecuteChat executes a non-streaming chat completion.
func (s *AIGatewayService) ExecuteChat(ctx context.Context, userIdentity string, req AIRequest) (*AIResponse, error) {
	// 1. Validate model
	if !allowedModels[req.Model] {
		return nil, fmt.Errorf("%w: %s", ErrModelNotAllowed, req.Model)
	}

	// 2. Get user's subscription tier
	tier, err := s.getUserTier(ctx, userIdentity)
	if err != nil {
		s.log("ai_gateway_get_tier_failed", map[string]interface{}{
			"level":         "error",
			"user_identity": userIdentity,
			"error":         err.Error(),
		})
		return nil, fmt.Errorf("failed to get user tier: %w", err)
	}

	// 3. Estimate cost for pre-check
	estimatedTokens := s.estimateTokens(req.Messages, req.MaxTokens)
	estimatedCost := s.calculateCost(req.Model, estimatedTokens.prompt, estimatedTokens.completion)

	// 4. Reserve credits atomically (check + tentative charge)
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

	// 5. Get OpenRouter API key
	apiKey, err := s.apiKeyService.Get(ctx, "openrouter")
	if err != nil {
		return nil, fmt.Errorf("get api key: %w", err)
	}
	if apiKey == "" {
		return nil, ErrNoAPIKeyConfigured
	}

	// 6. Call OpenRouter
	orReq := openRouterRequest{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   false,
	}

	body, err := json.Marshal(orReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("HTTP-Referer", "https://vrooli.com")
	httpReq.Header.Set("X-Title", "Vrooli AI Gateway")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openrouter request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		s.log("openrouter_error", map[string]interface{}{
			"level":  "error",
			"status": resp.StatusCode,
			"body":   string(bodyBytes),
		})
		return nil, fmt.Errorf("%w: status %d", ErrOpenRouterError, resp.StatusCode)
	}

	var orResp openRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&orResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// 7. Calculate actual cost based on real token usage
	actualCost := s.calculateCost(req.Model, orResp.Usage.PromptTokens, orResp.Usage.CompletionTokens)

	// 8. Adjust credits if actual cost differs from estimate
	// If actual cost is higher, charge the difference
	// If actual cost is lower, we don't refund (the estimate is the minimum charge)
	if actualCost > estimatedCost {
		extraCost := actualCost - estimatedCost
		_ = s.usageService.RecordUsage(ctx, UsageReportRequest{
			UserIdentity: userIdentity,
			LimitKey:     "ai_credits",
			Amount:       extraCost,
			AppBundleKey: req.Metadata.AppBundleKey,
			Operation:    req.Metadata.Operation,
			Metadata:     map[string]string{"type": "cost_adjustment"},
		})
	}

	// Extract response content
	content := ""
	finishReason := ""
	if len(orResp.Choices) > 0 {
		content = orResp.Choices[0].Message.Content
		finishReason = orResp.Choices[0].FinishReason
	}

	s.log("ai_gateway_request_completed", map[string]interface{}{
		"level":            "info",
		"user_identity":    userIdentity,
		"model":            req.Model,
		"prompt_tokens":    orResp.Usage.PromptTokens,
		"completion_tokens": orResp.Usage.CompletionTokens,
		"credits_charged":  actualCost,
	})

	return &AIResponse{
		ID:               orResp.ID,
		Model:            orResp.Model,
		Content:          content,
		PromptTokens:     orResp.Usage.PromptTokens,
		CompletionTokens: orResp.Usage.CompletionTokens,
		TotalTokens:      orResp.Usage.TotalTokens,
		CreditsCharged:   actualCost,
		FinishReason:     finishReason,
	}, nil
}

// ExecuteChatStream executes a streaming chat completion via Server-Sent Events.
func (s *AIGatewayService) ExecuteChatStream(ctx context.Context, userIdentity string, req AIRequest, w http.ResponseWriter) error {
	// 1. Validate model
	if !allowedModels[req.Model] {
		return fmt.Errorf("%w: %s", ErrModelNotAllowed, req.Model)
	}

	// 2. Get user's subscription tier
	tier, err := s.getUserTier(ctx, userIdentity)
	if err != nil {
		return fmt.Errorf("failed to get user tier: %w", err)
	}

	// 3. Estimate cost and check credits BEFORE starting stream
	estimatedTokens := s.estimateTokens(req.Messages, req.MaxTokens)
	estimatedCost := s.calculateCost(req.Model, estimatedTokens.prompt, estimatedTokens.completion)

	// Check if user can afford the estimated cost (don't charge yet)
	canProceed, _, err := s.usageService.CheckLimit(ctx, userIdentity, tier, "ai_credits", estimatedCost)
	if err != nil {
		return fmt.Errorf("credit check failed: %w", err)
	}
	if !canProceed {
		return ErrInsufficientCredits
	}

	// 4. Set up SSE headers
	flusher, ok := w.(http.Flusher)
	if !ok {
		return ErrStreamingNotSupported
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// 5. Get OpenRouter API key
	apiKey, err := s.apiKeyService.Get(ctx, "openrouter")
	if err != nil {
		return fmt.Errorf("get api key: %w", err)
	}
	if apiKey == "" {
		return ErrNoAPIKeyConfigured
	}

	// 6. Call OpenRouter with streaming
	orReq := openRouterRequest{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   true,
	}

	body, err := json.Marshal(orReq)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("HTTP-Referer", "https://vrooli.com")
	httpReq.Header.Set("X-Title", "Vrooli AI Gateway")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("openrouter request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w: status %d: %s", ErrOpenRouterError, resp.StatusCode, string(bodyBytes))
	}

	// 7. Stream response chunks
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 64*1024) // Increase buffer for long lines

	var totalPromptTokens, totalCompletionTokens int
	var fullContent strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// Parse SSE data line
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Check for stream end
		if data == "[DONE]" {
			break
		}

		// Parse the chunk
		var chunk openRouterStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			s.log("stream_chunk_parse_error", map[string]interface{}{
				"level": "warn",
				"error": err.Error(),
				"data":  data,
			})
			continue
		}

		// Extract content delta
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			content := chunk.Choices[0].Delta.Content
			fullContent.WriteString(content)

			// Send chunk event
			event := AIStreamEvent{
				Type:    "chunk",
				Content: content,
			}
			eventData, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", eventData)
			flusher.Flush()
		}

		// Capture usage if present (usually in the last chunk)
		if chunk.Usage != nil {
			totalPromptTokens = chunk.Usage.PromptTokens
			totalCompletionTokens = chunk.Usage.CompletionTokens
		}
	}

	if err := scanner.Err(); err != nil {
		s.log("stream_scanner_error", map[string]interface{}{
			"level": "error",
			"error": err.Error(),
		})
	}

	// 8. Calculate and charge actual credits
	// If we didn't get usage from the stream, estimate from content
	if totalPromptTokens == 0 {
		totalPromptTokens = estimatedTokens.prompt
	}
	if totalCompletionTokens == 0 {
		totalCompletionTokens = len(fullContent.String()) / 4 // Rough estimate: 4 chars per token
	}

	actualCost := s.calculateCost(req.Model, totalPromptTokens, totalCompletionTokens)

	// Charge the actual cost
	if err := s.usageService.RecordUsage(ctx, UsageReportRequest{
		UserIdentity: userIdentity,
		LimitKey:     "ai_credits",
		Amount:       actualCost,
		AppBundleKey: req.Metadata.AppBundleKey,
		Operation:    req.Metadata.Operation,
	}); err != nil {
		s.log("stream_charge_failed", map[string]interface{}{
			"level":         "error",
			"user_identity": userIdentity,
			"error":         err.Error(),
		})
	}

	// 9. Send done event with usage info
	doneEvent := AIStreamEvent{
		Type: "done",
		Usage: &struct {
			PromptTokens     int   `json:"prompt_tokens"`
			CompletionTokens int   `json:"completion_tokens"`
			TotalTokens      int   `json:"total_tokens"`
			CreditsCharged   int64 `json:"credits_charged"`
		}{
			PromptTokens:     totalPromptTokens,
			CompletionTokens: totalCompletionTokens,
			TotalTokens:      totalPromptTokens + totalCompletionTokens,
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
		"prompt_tokens":     totalPromptTokens,
		"completion_tokens": totalCompletionTokens,
		"credits_charged":   actualCost,
	})

	return nil
}

// getUserTier retrieves the user's subscription tier.
func (s *AIGatewayService) getUserTier(ctx context.Context, userIdentity string) (string, error) {
	if s.accountService == nil {
		return "free", nil
	}

	sub, err := s.accountService.GetSubscription(userIdentity)
	if err != nil {
		// Log but don't fail - default to free tier
		s.log("get_subscription_failed", map[string]interface{}{
			"level":         "warn",
			"user_identity": userIdentity,
			"error":         err.Error(),
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
func (s *AIGatewayService) estimateTokens(messages []AIMessage, maxTokens int) tokenEstimate {
	// Rough estimation: ~4 characters per token
	promptChars := 0
	for _, msg := range messages {
		promptChars += len(msg.Content) + len(msg.Role) + 10 // +10 for message overhead
	}
	promptTokens := promptChars / 4

	// Estimate completion tokens
	completionTokens := 1000 // Default estimate
	if maxTokens > 0 {
		completionTokens = maxTokens
	}

	return tokenEstimate{
		prompt:     promptTokens,
		completion: completionTokens,
	}
}

// calculateCost calculates the cost in internal units for a request.
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
	key, err := s.apiKeyService.Get(ctx, "openrouter")
	if err != nil {
		return fmt.Errorf("api key check failed: %w", err)
	}
	if key == "" {
		return ErrNoAPIKeyConfigured
	}

	// Verify the key works with a simple API call
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://openrouter.ai/api/v1/auth/key", nil)
	req.Header.Set("Authorization", "Bearer "+key)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("openrouter connectivity check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("openrouter key verification failed: status %d", resp.StatusCode)
	}

	return nil
}
