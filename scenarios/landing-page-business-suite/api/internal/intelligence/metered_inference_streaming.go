package intelligence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// ExecuteChatStream executes a streaming chat completion via Server-Sent Events.
// Uses credit reservations to prevent TOCTOU race conditions where concurrent
// streaming requests could exceed the credit limit.
func (s *MeteredInferenceService) ExecuteChatStream(ctx context.Context, userIdentity string, req AIRequest, w http.ResponseWriter) error {
	if !allowedModels[req.Model] {
		return fmt.Errorf("%w: %s", ErrModelNotAllowed, req.Model)
	}

	billingIdentity, tier, err := s.billingIdentityAndTier(ctx, userIdentity, req.BusinessAccountID)
	if err != nil {
		return fmt.Errorf("failed to get user tier: %w", err)
	}

	estimate := s.estimateInferenceTokens(req)
	estimatedCost := s.calculateCost(req.Model, estimate.prompt, estimate.completion)
	reservationID, err := s.reserveCredits(ctx, userIdentity, billingIdentity, req.BusinessAccountID, tier, estimatedCost)
	if err != nil {
		if errors.Is(err, ErrInsufficientCredits) || strings.Contains(err.Error(), "insufficient") {
			return ErrInsufficientCredits
		}
		return fmt.Errorf("reserve credits failed: %w", err)
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.releaseStreamingReservation(ctx, userIdentity, req.BusinessAccountID, reservationID)
		return ErrStreamingNotSupported
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	client, err := s.getOpenRouterClient(ctx)
	if err != nil {
		s.releaseStreamingReservation(ctx, userIdentity, req.BusinessAccountID, reservationID)
		return err
	}

	orMessages := make([]OpenRouterMessage, len(req.Messages))
	for i, msg := range req.Messages {
		orMessages[i] = OpenRouterMessage{Role: msg.Role, Content: msg.Content, Name: msg.Name, ToolCallID: msg.ToolCallID, ToolCalls: msg.ToolCalls}
	}

	providerRequest := OpenRouterChatRequest{Model: req.Model, Messages: orMessages, Tools: req.Tools, ToolChoice: req.ToolChoice}
	emit := func(event OpenRouterStreamEvent) {
		eventData, _ := json.Marshal(AIStreamEvent{Type: "chunk", Content: event.Content, ToolCalls: event.ToolCalls})
		fmt.Fprintf(w, "data: %s\n\n", eventData)
		flusher.Flush()
	}
	var usage *OpenRouterUsage
	if eventClient, ok := client.(OpenRouterStreamEventClient); ok {
		usage, err = eventClient.ChatStreamEvents(ctx, providerRequest, emit)
	} else if len(req.Tools) > 0 {
		s.releaseStreamingReservation(ctx, userIdentity, req.BusinessAccountID, reservationID)
		return errors.New("tool-call streaming is unsupported by the configured provider client")
	} else {
		usage, err = client.ChatStream(ctx, providerRequest, func(content string) { emit(OpenRouterStreamEvent{Content: content}) })
	}
	if err != nil {
		if usage != nil && usage.UsageReported && usage.PromptTokens >= 0 && usage.CompletionTokens >= 0 && usage.TotalTokens >= 0 && (usage.PromptTokens > 0 || usage.CompletionTokens > 0) {
			promptTokens := usage.PromptTokens
			actualCost := s.calculateCost(req.Model, promptTokens, usage.CompletionTokens)
			if settleErr := s.finalizeReservationWithCost(ctx, userIdentity, req.BusinessAccountID, reservationID, actualCost, req.Metadata.AppBundleKey, req.Model, usage.CostMicros, promptTokens, usage.CompletionTokens, "openrouter"); settleErr != nil {
				s.log("partial_stream_finalize_failed", map[string]interface{}{
					"level": "error", "user_identity": userIdentity, "reservation_id": reservationID, "actual_cost": actualCost, "error": settleErr.Error(),
				})
				if req.BusinessAccountID != "" {
					if accountUsage, ok := s.usageService.(BusinessAccountUsageServicer); ok {
						if retainErr := accountUsage.RetainUncertainBusinessAccountReservation(ctx, reservationID); retainErr != nil {
							s.log("partial_stream_reservation_retain_failed", map[string]interface{}{"level": "error", "user_identity": userIdentity, "reservation_id": reservationID, "error": retainErr.Error(), "security": true})
						}
					} else {
						s.log("partial_stream_reservation_retain_unavailable", map[string]interface{}{"level": "error", "user_identity": userIdentity, "reservation_id": reservationID, "security": true})
					}
				} else if retainer, ok := s.usageService.(UncertainReservationRetainer); ok {
					if retainErr := retainer.RetainUncertainReservation(ctx, reservationID); retainErr != nil {
						s.log("partial_stream_reservation_retain_failed", map[string]interface{}{
							"level": "error", "user_identity": userIdentity, "reservation_id": reservationID, "error": retainErr.Error(), "security": true,
						})
					}
				} else {
					s.log("partial_stream_reservation_retain_unavailable", map[string]interface{}{
						"level": "error", "user_identity": userIdentity, "reservation_id": reservationID, "security": true,
					})
				}
			}
		} else {
			s.releaseStreamingReservation(ctx, userIdentity, req.BusinessAccountID, reservationID)
		}
		return fmt.Errorf("streaming failed: %w", err)
	}
	if usage == nil {
		s.releaseStreamingReservation(ctx, userIdentity, req.BusinessAccountID, reservationID)
		return errors.New("streaming failed: provider returned no usage")
	}

	promptTokens := usage.PromptTokens
	if promptTokens == 0 {
		promptTokens = estimate.prompt
	}
	actualCost := s.calculateCost(req.Model, promptTokens, usage.CompletionTokens)

	if err := s.finalizeReservationWithCost(ctx, userIdentity, req.BusinessAccountID, reservationID, actualCost, req.Metadata.AppBundleKey, req.Model, usage.CostMicros, promptTokens, usage.CompletionTokens, "openrouter"); err != nil {
		s.log("finalize_reservation_failed", map[string]interface{}{
			"level": "error", "user_identity": userIdentity, "reservation_id": reservationID, "actual_cost": actualCost, "error": err.Error(),
		})
		if strings.TrimSpace(req.BusinessAccountID) != "" {
			if accountUsage, ok := s.usageService.(BusinessAccountUsageServicer); ok {
				if retainErr := accountUsage.RetainUncertainBusinessAccountReservation(ctx, reservationID); retainErr != nil {
					s.log("finalize_reservation_retain_failed", map[string]interface{}{"level": "error", "user_identity": userIdentity, "reservation_id": reservationID, "error": retainErr.Error(), "security": true})
				}
			} else {
				s.log("finalize_reservation_retain_unavailable", map[string]interface{}{"level": "error", "user_identity": userIdentity, "reservation_id": reservationID, "security": true})
			}
		} else if fallbackErr := s.recordUsage(ctx, userIdentity, req.BusinessAccountID, tier, UsageReport{
			UserIdentity: billingIdentity, LimitKey: "ai_credits", Amount: actualCost,
			AppBundleKey: req.Metadata.AppBundleKey, Operation: req.Metadata.Operation,
			Metadata: map[string]string{
				"model": req.Model, "provider": "openrouter",
				"cost_micros":       fmt.Sprintf("%d", usage.CostMicros),
				"prompt_tokens":     fmt.Sprintf("%d", promptTokens),
				"completion_tokens": fmt.Sprintf("%d", usage.CompletionTokens),
			},
		}); fallbackErr != nil {
			s.log("finalize_fallback_record_failed", map[string]interface{}{
				"level": "error", "user_identity": userIdentity, "reservation_id": reservationID, "actual_cost": actualCost, "error": fallbackErr.Error(), "security": true,
			})
		}
	}

	doneEvent := AIStreamEvent{Type: "done", Usage: &struct {
		PromptTokens     int   `json:"prompt_tokens"`
		CompletionTokens int   `json:"completion_tokens"`
		TotalTokens      int   `json:"total_tokens"`
		CreditsCharged   int64 `json:"credits_charged"`
	}{
		PromptTokens: promptTokens, CompletionTokens: usage.CompletionTokens,
		TotalTokens: promptTokens + usage.CompletionTokens, CreditsCharged: actualCost,
	}}
	eventData, _ := json.Marshal(doneEvent)
	fmt.Fprintf(w, "data: %s\n\n", eventData)
	flusher.Flush()

	s.log("metered_inference_stream_completed", map[string]interface{}{
		"level": "info", "user_identity": userIdentity, "model": req.Model,
		"prompt_tokens": promptTokens, "completion_tokens": usage.CompletionTokens,
		"credits_charged": actualCost, "reservation_id": reservationID,
	})
	return nil
}

func (s *MeteredInferenceService) releaseStreamingReservation(ctx context.Context, userIdentity, businessAccountID, reservationID string) {
	if err := s.releaseReservation(ctx, userIdentity, businessAccountID, reservationID); err != nil {
		s.log("reservation_release_failed", map[string]interface{}{
			"level": "warn", "user_identity": userIdentity, "reservation_id": reservationID, "error": err.Error(),
		})
	}
}
