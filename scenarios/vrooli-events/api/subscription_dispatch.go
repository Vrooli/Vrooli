package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/bridge"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/convert"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/match"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/subscription"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
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
	payload, err := webhookPayload(event)
	if err != nil {
		return err
	}
	for _, sub := range subs {
		// Bridge admission requires the production shared transaction, never
		// this lightweight mocked/non-atomic fallback.
		if bridge.Reserved(sub) {
			continue
		}
		if !matchesWebhook(sub, event) {
			continue
		}
		if err := queue.Enqueue(ctx, sub.ID, event.EventID, payload); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) enqueueSubscriptionsTx(ctx context.Context, tx *sql.Tx, queue subscription.AtomicQueueStore, event store.Event) error {
	subs, err := queue.ListTx(ctx, tx, subscription.ListFilters{Enabled: boolPtr(true)})
	if err != nil {
		return err
	}
	payload, err := webhookPayload(event)
	if err != nil {
		return err
	}
	for _, sub := range subs {
		if bridge.Reserved(sub) {
			if s.flywheel == nil || !bridge.ValidSubscription(sub) || sub.EventPattern != event.EventType || sub.SourceFilter != event.SourceScenario {
				continue
			}
			projected, selected, err := s.flywheel.Project(ctx, event)
			if err != nil {
				return err
			}
			if selected {
				if err := queue.EnqueueTx(ctx, tx, sub.ID, event.EventID, projected); err != nil {
					return err
				}
			}
			continue
		}
		if !matchesWebhook(sub, event) {
			continue
		}
		if err := queue.EnqueueTx(ctx, tx, sub.ID, event.EventID, payload); err != nil {
			return err
		}
	}
	return nil
}

func matchesWebhook(sub subscription.Subscription, event store.Event) bool {
	return sub.DeliveryType == subscription.DeliveryWebhook && match.Glob(sub.EventPattern, event.EventType) &&
		(sub.SourceFilter == "" || match.Glob(sub.SourceFilter, event.SourceScenario))
}

func webhookPayload(event store.Event) (string, error) {
	var payloadValue any
	if env, err := convert.EventToEnvelope(event); err == nil && env.Data != nil {
		if message, unmarshalErr := anypb.UnmarshalNew(env.Data, proto.UnmarshalOptions{}); unmarshalErr == nil {
			if structured, ok := message.(*structpb.Struct); ok {
				payloadValue = structured.AsMap()
			} else if typed, ok := message.(proto.Message); ok {
				// Typed protobuf subscribers receive the inner message, not the
				// canonical event envelope. This keeps webhook payloads stable
				// across the generic event store and typed receivers such as
				// agent-manager's ProgramEvent webhook.
				if encoded, marshalErr := proto.Marshal(typed); marshalErr == nil {
					payloadValue = map[string]string{"encoding": "base64", "data": base64.StdEncoding.EncodeToString(encoded)}
				}
			}
		}
	}
	if payloadValue == nil && json.Valid(event.Payload) {
		payloadValue = json.RawMessage(event.Payload)
	} else if payloadValue == nil && len(event.Payload) > 0 {
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
		return "", err
	}
	return string(encoded), nil
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
	var deliveries []subscription.QueuedDelivery
	var err error
	if s.flywheel == nil {
		if selective, ok := queue.(interface {
			DueExcludingNames(context.Context, int, string, []string, string) ([]subscription.QueuedDelivery, error)
		}); ok {
			spec := bridge.SubscriptionSpec()
			deliveries, err = selective.DueExcludingNames(ctx, 25, spec.DeliveryTarget, []string{spec.Name, bridge.TerminalSubscriptionSpec().Name}, bridge.WakeSchema)
		} else {
			deliveries, err = queue.Due(ctx, 25)
		}
	} else {
		deliveries, err = queue.Due(ctx, 25)
	}
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
		if bridge.Reserved(sub) || bridge.IsWakeBody(delivery.PayloadJSON) {
			// Fail closed across configuration changes. No raw/generic fallback,
			// and disabled installations leave rows untouched for owner review.
			if s.flywheel == nil {
				continue
			}
			if err := s.flywheel.Deliver(ctx, sub, delivery); err != nil {
				attempt := delivery.Attempts%subscription.DeliveryAttemptsPerCycle + 1
				_ = queue.MarkFailed(ctx, delivery.ID, delivery.SubscriptionID, "local-work delivery rejected", time.Now().UTC().Add(time.Second<<(attempt-1)), attempt >= subscription.DeliveryAttemptsPerCycle)
				continue
			}
			_ = queue.MarkDelivered(ctx, delivery.ID, delivery.SubscriptionID, time.Now().UTC())
			continue
		}
		var payload subscription.WebhookPayload
		if err := json.Unmarshal([]byte(delivery.PayloadJSON), &payload); err != nil {
			_ = queue.MarkFailed(ctx, delivery.ID, delivery.SubscriptionID, "invalid queued payload", time.Now().UTC(), true)
			continue
		}
		if err := s.webhookDeliverer.Deliver(ctx, sub.DeliveryTarget, payload); err != nil {
			attemptInCycle := delivery.Attempts%subscription.DeliveryAttemptsPerCycle + 1
			deadLetter := attemptInCycle >= subscription.DeliveryAttemptsPerCycle
			delay := time.Second << min(attemptInCycle-1, subscription.DeliveryAttemptsPerCycle-1)
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
