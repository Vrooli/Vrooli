package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/match"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/subscription"
)

func (s *Server) enqueueSubscriptions(ctx context.Context, event store.Event) error {
	queue, ok := s.subStore.(subscription.QueueStore)
	if !ok {
		return nil
	}
	subs, err := s.subStore.List(ctx, subscription.ListFilters{Enabled: boolPtr(true)})
	if err != nil {
		return err
	}
	var payloadValue any
	if json.Valid(event.Payload) {
		payloadValue = json.RawMessage(event.Payload)
	} else if len(event.Payload) > 0 {
		// Event storage is intentionally bytes-oriented because protobuf event
		// payloads are valid inputs too. Preserve opaque payloads losslessly
		// rather than making an invalid JSON webhook body.
		payloadValue = map[string]string{"encoding": "base64", "data": base64.StdEncoding.EncodeToString(event.Payload)}
	}
	payload := subscription.WebhookPayload{
		EventID: event.EventID, EventType: event.EventType, SourceScenario: event.SourceScenario,
		Payload: payloadValue, DeliveredAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	for _, sub := range subs {
		if sub.DeliveryType != subscription.DeliveryWebhook || !match.Glob(sub.EventPattern, event.EventType) {
			continue
		}
		if sub.SourceFilter != "" && !match.Glob(sub.SourceFilter, event.SourceScenario) {
			continue
		}
		if err := queue.Enqueue(ctx, sub.ID, event.EventID, string(encoded)); err != nil {
			return err
		}
	}
	return nil
}

func boolPtr(value bool) *bool { return &value }

func (s *Server) runSubscriptionDispatcher(ctx context.Context) {
	queue, ok := s.subStore.(subscription.QueueStore)
	if !ok {
		return
	}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.drainSubscriptionQueue(ctx, queue)
		}
	}
}

func (s *Server) drainSubscriptionQueue(ctx context.Context, queue subscription.QueueStore) {
	deliveries, err := queue.Due(ctx, 25)
	if err != nil {
		log.Printf("subscription queue read: %v", err)
		return
	}
	for _, delivery := range deliveries {
		sub, err := s.subStore.Get(ctx, delivery.SubscriptionID)
		if err != nil {
			_ = queue.MarkFailed(ctx, delivery.ID, delivery.SubscriptionID, "subscription no longer exists", time.Now().UTC(), true)
			continue
		}
		var payload subscription.WebhookPayload
		if err := json.Unmarshal([]byte(delivery.PayloadJSON), &payload); err != nil {
			_ = queue.MarkFailed(ctx, delivery.ID, delivery.SubscriptionID, "invalid queued payload", time.Now().UTC(), true)
			continue
		}
		if err := s.webhookDeliverer.Deliver(ctx, sub.DeliveryTarget, payload); err != nil {
			attempt := delivery.Attempts + 1
			deadLetter := attempt >= 5
			delay := time.Second << min(attempt-1, 4)
			_ = queue.MarkFailed(ctx, delivery.ID, delivery.SubscriptionID, safeDeliveryError(err), time.Now().UTC().Add(delay), deadLetter)
			continue
		}
		_ = queue.MarkDelivered(ctx, delivery.ID, delivery.SubscriptionID, time.Now().UTC())
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func safeDeliveryError(err error) string {
	value := strings.TrimSpace(err.Error())
	if len(value) > 500 {
		return value[:500]
	}
	return value
}
