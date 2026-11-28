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
	if err := validateGatewayRootOperationID(req.Metadata.GatewayRootOperationID); err != nil {
		return err
	}
	if err := validateBrowserEvidenceInput(req.Metadata); err != nil {
		return err
	}
	if req.Metadata.BrowserEvidence != nil && s.browserEvidenceAttester == nil {
		return errors.New("BAS browser evidence verifier is unavailable")
	}
	if err := ensureBrowserEvidenceAttesterReady(ctx, s.browserEvidenceAttester, req.Metadata.BrowserEvidence); err != nil {
		return err
	}
	if err := requireBillingScope(req.BusinessAccountID, req.creditOwnerVerified); err != nil {
		return err
	}
	if !allowedModels[req.Model] {
		return fmt.Errorf("%w: %s", ErrModelNotAllowed, req.Model)
	}

	billingIdentity, tier, err := s.billingIdentityAndTier(ctx, userIdentity, req.BusinessAccountID)
	if err != nil {
		return fmt.Errorf("failed to get user tier: %w", err)
	}

	estimate := s.estimateInferenceTokens(req)
	estimatedCost := s.calculateCost(req.Model, estimate.prompt, estimate.completion)
	reservationID, err := s.reserveCredits(ctx, userIdentity, req.BusinessAccountID, tier, req.Metadata.AppBundleKey, estimatedCost, req.Metadata.BrowserEvidence, req.Metadata.GatewayRootOperationID)
	if err != nil {
		if errors.Is(err, ErrInsufficientCredits) || strings.Contains(err.Error(), "insufficient") {
			return ErrInsufficientCredits
		}
		return fmt.Errorf("reserve credits failed: %w", err)
	}
	if err := s.bindBrowserEvidence(ctx, userIdentity, req.BusinessAccountID, reservationID, req.Metadata.BrowserEvidence); err != nil {
		if releaseErr := s.releaseReservation(ctx, userIdentity, req.BusinessAccountID, reservationID); releaseErr != nil {
			return errors.Join(fmt.Errorf("browser evidence could not be bound to LPBS reservation: %w", err), ErrInferenceReservationCleanupFailed, releaseErr)
		}
		return fmt.Errorf("browser evidence could not be bound to LPBS reservation: %w", err)
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		if releaseErr := s.releaseStreamingReservation(ctx, userIdentity, req.BusinessAccountID, reservationID); releaseErr != nil {
			return errors.Join(ErrStreamingNotSupported, ErrInferenceReservationCleanupFailed, releaseErr)
		}
		return ErrStreamingNotSupported
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	client, err := s.getOpenRouterClient(ctx)
	if err != nil {
		if releaseErr := s.releaseStreamingReservation(ctx, userIdentity, req.BusinessAccountID, reservationID); releaseErr != nil {
			return errors.Join(err, ErrInferenceReservationCleanupFailed, releaseErr)
		}
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
		unsupportedErr := errors.New("tool-call streaming is unsupported by the configured provider client")
		if releaseErr := s.releaseStreamingReservation(ctx, userIdentity, req.BusinessAccountID, reservationID); releaseErr != nil {
			return errors.Join(unsupportedErr, ErrInferenceReservationCleanupFailed, releaseErr)
		}
		return unsupportedErr
	} else {
		usage, err = client.ChatStream(ctx, providerRequest, func(content string) { emit(OpenRouterStreamEvent{Content: content}) })
	}
	if err != nil {
		if usage != nil && usage.UsageReported && usage.PromptTokens >= 0 && usage.CompletionTokens >= 0 && usage.TotalTokens >= 0 {
			promptTokens := usage.PromptTokens
			actualCost := s.calculateCost(req.Model, promptTokens, usage.CompletionTokens)
			browserUsage, evidenceErr := s.resolveBrowserUsage(ctx, req.Metadata.BrowserEvidence, reservationID)
			settleErr := evidenceErr
			if settleErr == nil {
				settleErr = s.finalizeReservationWithCost(ctx, userIdentity, req.BusinessAccountID, reservationID, actualCost, req.Metadata.AppBundleKey, req.Model, usage.CostMicros, promptTokens, usage.CompletionTokens, "openrouter", browserUsage)
			}
			if settleErr != nil {
				s.log("partial_stream_finalize_failed", map[string]interface{}{
					"level": "error", "user_identity": userIdentity, "reservation_id": reservationID, "actual_cost": actualCost, "error": settleErr.Error(),
				})
				uncertainErr := s.markInferenceOutcomeUncertain(userIdentity, req.BusinessAccountID, reservationID, req.Metadata.GatewayRootOperationID)
				return errors.Join(fmt.Errorf("partial stream settlement failed: %w", settleErr), uncertainErr, fmt.Errorf("streaming failed: %w", err))
			}
		} else {
			if errors.Is(err, ErrProviderRequestRejected) {
				if releaseErr := s.releaseReservation(ctx, userIdentity, req.BusinessAccountID, reservationID); releaseErr != nil {
					return errors.Join(fmt.Errorf("provider request rejected: %w", err), ErrInferenceReservationCleanupFailed, releaseErr)
				}
			} else {
				uncertainErr := s.markInferenceOutcomeUncertain(userIdentity, req.BusinessAccountID, reservationID, req.Metadata.GatewayRootOperationID)
				return errors.Join(fmt.Errorf("streaming failed: %w", err), uncertainErr)
			}
		}
		return fmt.Errorf("streaming failed: %w", err)
	}
	if usage == nil {
		uncertainErr := s.markInferenceOutcomeUncertain(userIdentity, req.BusinessAccountID, reservationID, req.Metadata.GatewayRootOperationID)
		return errors.Join(errors.New("streaming failed: provider returned no authoritative usage"), uncertainErr)
	}

	promptTokens := usage.PromptTokens
	if promptTokens == 0 {
		promptTokens = estimate.prompt
	}
	actualCost := s.calculateCost(req.Model, promptTokens, usage.CompletionTokens)
	browserUsage, evidenceErr := s.resolveBrowserUsage(ctx, req.Metadata.BrowserEvidence, reservationID)
	if evidenceErr != nil {
		uncertainErr := s.markInferenceOutcomeUncertain(userIdentity, req.BusinessAccountID, reservationID, req.Metadata.GatewayRootOperationID)
		return errors.Join(fmt.Errorf("browser evidence unavailable; reservation retained: %w", evidenceErr), uncertainErr)
	}

	if err := s.finalizeReservationWithCost(ctx, userIdentity, req.BusinessAccountID, reservationID, actualCost, req.Metadata.AppBundleKey, req.Model, usage.CostMicros, promptTokens, usage.CompletionTokens, "openrouter", browserUsage); err != nil {
		s.log("finalize_reservation_failed", map[string]interface{}{
			"level": "error", "user_identity": userIdentity, "reservation_id": reservationID, "actual_cost": actualCost, "error": err.Error(),
		})
		if strings.TrimSpace(req.BusinessAccountID) != "" {
			uncertainErr := s.markInferenceOutcomeUncertain(userIdentity, req.BusinessAccountID, reservationID, req.Metadata.GatewayRootOperationID)
			return errors.Join(fmt.Errorf("finalize stream reservation: %w", err), uncertainErr)
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
		PromptTokens     int          `json:"prompt_tokens"`
		CompletionTokens int          `json:"completion_tokens"`
		TotalTokens      int          `json:"total_tokens"`
		CreditsCharged   int64        `json:"credits_charged"`
		BrowserUsage     BrowserUsage `json:"browser_usage,omitempty"`
	}{
		PromptTokens: promptTokens, CompletionTokens: usage.CompletionTokens,
		TotalTokens: promptTokens + usage.CompletionTokens, CreditsCharged: actualCost, BrowserUsage: browserUsage,
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

func (s *MeteredInferenceService) releaseStreamingReservation(ctx context.Context, userIdentity, businessAccountID, reservationID string) error {
	if err := s.releaseReservation(ctx, userIdentity, businessAccountID, reservationID); err != nil {
		s.log("reservation_release_failed", map[string]interface{}{
			"level": "warn", "user_identity": userIdentity, "reservation_id": reservationID, "error": err.Error(),
		})
		return err
	}
	return nil
}
