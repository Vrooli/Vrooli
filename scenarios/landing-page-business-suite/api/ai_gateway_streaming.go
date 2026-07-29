package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"landing-page-business-suite-api/internal/intelligence"
)

// ExecuteChatStream executes a streaming chat completion via Server-Sent Events.
// Uses credit reservations to prevent TOCTOU race conditions where concurrent
// streaming requests could exceed the credit limit.
func (s *AIGatewayService) ExecuteChatStream(ctx context.Context, userIdentity string, req AIRequest, w http.ResponseWriter) error {
	if !allowedModels[req.Model] {
		return fmt.Errorf("%w: %s", ErrModelNotAllowed, req.Model)
	}

	tier, err := s.getUserTier(ctx, userIdentity)
	if err != nil {
		return fmt.Errorf("failed to get user tier: %w", err)
	}

	estimate := s.estimateTokens(req.Messages, req.MaxTokens)
	estimatedCost := s.calculateCost(req.Model, estimate.prompt, estimate.completion)
	reservationID, err := s.usageService.ReserveCredits(ctx, userIdentity, tier, "ai_credits", estimatedCost)
	if err != nil {
		if errors.Is(err, ErrInsufficientCredits) || strings.Contains(err.Error(), "insufficient") {
			return ErrInsufficientCredits
		}
		return fmt.Errorf("reserve credits failed: %w", err)
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.releaseStreamingReservation(ctx, userIdentity, reservationID)
		return ErrStreamingNotSupported
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	client, err := s.getOpenRouterClient(ctx)
	if err != nil {
		s.releaseStreamingReservation(ctx, userIdentity, reservationID)
		return err
	}

	orMessages := make([]OpenRouterMessage, len(req.Messages))
	for i, msg := range req.Messages {
		orMessages[i] = OpenRouterMessage(msg)
	}

	usage, err := client.ChatStream(ctx, OpenRouterChatRequest{Model: req.Model, Messages: orMessages}, func(content string) {
		eventData, _ := json.Marshal(AIStreamEvent{Type: "chunk", Content: content})
		fmt.Fprintf(w, "data: %s\n\n", eventData)
		flusher.Flush()
	})
	if err != nil {
		s.releaseStreamingReservation(ctx, userIdentity, reservationID)
		return fmt.Errorf("streaming failed: %w", err)
	}

	promptTokens := usage.PromptTokens
	if promptTokens == 0 {
		promptTokens = estimate.prompt
	}
	actualCost := s.calculateCost(req.Model, promptTokens, usage.CompletionTokens)

	if err := s.usageService.FinalizeReservation(ctx, reservationID, actualCost); err != nil {
		s.log("finalize_reservation_failed", map[string]interface{}{
			"level": "error", "user_identity": userIdentity, "reservation_id": reservationID, "actual_cost": actualCost, "error": err.Error(),
		})
		if fallbackErr := s.usageService.RecordUsage(ctx, intelligence.UsageReport{
			UserIdentity: userIdentity, LimitKey: "ai_credits", Amount: actualCost,
			AppBundleKey: req.Metadata.AppBundleKey, Operation: req.Metadata.Operation,
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

	s.log("ai_gateway_stream_completed", map[string]interface{}{
		"level": "info", "user_identity": userIdentity, "model": req.Model,
		"prompt_tokens": promptTokens, "completion_tokens": usage.CompletionTokens,
		"credits_charged": actualCost, "reservation_id": reservationID,
	})
	return nil
}

func (s *AIGatewayService) releaseStreamingReservation(ctx context.Context, userIdentity, reservationID string) {
	if err := s.usageService.ReleaseReservation(ctx, reservationID); err != nil {
		s.log("reservation_release_failed", map[string]interface{}{
			"level": "warn", "user_identity": userIdentity, "reservation_id": reservationID, "error": err.Error(),
		})
	}
}
