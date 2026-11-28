package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/bridge"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/convert"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/subscription"
)

func terminalBridgeSubscription(t *testing.T, s *Server, enabled bool) int64 {
	t.Helper()
	spec := bridge.TerminalSubscriptionSpec()
	spec.Enabled = enabled
	id, err := s.subStore.Create(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestTerminalBridgeAtomicSelectionReplayAndRestart(t *testing.T) {
	s, f, legacyID, notification := nativeBridgeServer(t)
	ctx := context.Background()
	terminalID := terminalBridgeSubscription(t, s, true)
	intent, id := terminalAdmissionIntent(t, func(e *eventbus.DomainEvent) { e.Occurred = time.Now().UTC().Add(-time.Minute) })
	queue := s.subStore.(subscription.QueueStore)
	// The terminal receipt and selected bridge row must share the owner commit.
	trigger := fmt.Sprintf("CREATE TRIGGER reject_terminal_bridge BEFORE INSERT ON subscription_deliveries WHEN NEW.subscription_id=%d BEGIN SELECT RAISE(ABORT,'fixture failure'); END", terminalID)
	if _, err := s.db.ExecContext(ctx, trigger); err != nil {
		t.Fatal(err)
	}
	if _, err := s.acceptTerminalEventIntent(ctx, intent, id); err == nil {
		t.Fatal("terminal committed without bridge row")
	}
	terminalAdmissionCounts(t, s, 0, 0, 0)
	if _, err := s.db.ExecContext(ctx, "DROP TRIGGER reject_terminal_bridge"); err != nil {
		t.Fatal(err)
	}
	ack, err := s.acceptTerminalEventIntent(ctx, intent, id)
	if err != nil {
		t.Fatal(err)
	}
	terminalAdmissionCounts(t, s, 1, 1, 1)
	due, err := queue.Due(ctx, 10)
	if err != nil || len(due) != 1 || due[0].SubscriptionID != terminalID {
		t.Fatalf("wrong terminal selection: %+v %v", due, err)
	}
	var wake bridge.Wake
	if err := json.Unmarshal([]byte(due[0].PayloadJSON), &wake); err != nil {
		t.Fatal(err)
	}
	if len(wake.Events) != 1 || wake.Events[0].EventType != bridge.TerminalEventType || wake.Events[0].Locator.Schema != "agent-manager.run-terminal.v1" || wake.Activation != "not_approved" || wake.EffectAuthority != "none" {
		t.Fatal("incorrect terminal projection")
	}
	original := due[0].PayloadJSON
	replay, err := s.acceptTerminalEventIntent(ctx, intent, id)
	if err != nil || replay != ack {
		t.Fatal("lost acknowledgment replay changed", err)
	}
	terminalAdmissionCounts(t, s, 1, 1, 1)
	if _, err := s.admitEvent(ctx, notification); err != nil {
		t.Fatal(err)
	}
	due, err = queue.Due(ctx, 10)
	if err != nil || len(due) != 2 || due[0].PayloadJSON != original || due[1].SubscriptionID != legacyID {
		t.Fatalf("cross-selector fanout: %+v %v", due, err)
	}
	// Both selectors survive reconstruction over the same owner state. A lost
	// terminal transport response retains its exact bytes and pending identity.
	resumed := &Server{store: s.store, db: s.db, subStore: s.subStore, flywheel: f}
	f.fail = true
	resumed.drainSubscriptionQueue(ctx, queue)
	state, err := queue.GetDelivery(ctx, terminalID, id)
	if err != nil || state.Status != "pending" || state.Attempts != 1 {
		t.Fatalf("lost response state: %+v %v", state, err)
	}
	if _, err := s.db.ExecContext(ctx, "UPDATE subscription_deliveries SET next_attempt_at='2000-01-01'"); err != nil {
		t.Fatal(err)
	}
	f.fail = false
	resumed.drainSubscriptionQueue(ctx, queue)
	state, err = queue.GetDelivery(ctx, terminalID, id)
	if err != nil || state.Status != "delivered" || len(f.deliveries) != 4 || f.deliveries[0].PayloadJSON != f.deliveries[2].PayloadJSON {
		t.Fatal("terminal retry changed durable projection", err)
	}
}

func TestTerminalBridgeDisabledAndForbiddenBypasses(t *testing.T) {
	for _, mode := range []string{"no-forwarder", "disabled-subscription", "direct", "reconcile", "non-atomic"} {
		t.Run(mode, func(t *testing.T) {
			s, f, _, _ := nativeBridgeServer(t)
			id := terminalBridgeSubscription(t, s, mode != "disabled-subscription")
			ctx := context.Background()
			if mode == "direct" || mode == "reconcile" {
				// A terminal reserved name is still quarantined after target mutation.
				sub, err := s.subStore.Get(ctx, id)
				if err != nil {
					t.Fatal(err)
				}
				sub.DeliveryTarget = "https://attacker.example"
				if err := s.subStore.Update(ctx, sub); err != nil {
					t.Fatal(err)
				}
				r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"event_id":"forged","payload":{"title":"PRIVATE"}}`))
				r.SetPathValue("id", fmt.Sprint(id))
				w := httptest.NewRecorder()
				if mode == "direct" {
					s.handleDeliverSubscription(w, r)
				} else {
					s.handleReconcileSubscriptionDelivery(w, r)
				}
				if w.Code != http.StatusBadRequest {
					t.Fatalf("bypass returned %d", w.Code)
				}
			} else {
				if mode == "no-forwarder" {
					s.flywheel = nil
				}
				intent, eventID := terminalAdmissionIntent(t, func(e *eventbus.DomainEvent) { e.Occurred = time.Now().UTC().Add(-time.Minute) })
				if mode == "non-atomic" {
					// Existing non-atomic admission fallback must never synthesize a wake.
					env, _, err := eventbus.RetainedDomainEventEnvelope(intent, eventID)
					if err != nil {
						t.Fatal(err)
					}
					e, err := convert.EnvelopeToEvent(env)
					if err != nil {
						t.Fatal(err)
					}
					if err := s.enqueueSubscriptions(ctx, e); err != nil {
						t.Fatal(err)
					}
				} else if _, err := s.acceptTerminalEventIntent(ctx, intent, eventID); err != nil {
					t.Fatal(err)
				}
			}
			due, err := s.subStore.(subscription.QueueStore).Due(ctx, 10)
			if err != nil || len(due) != 0 || len(f.deliveries) != 0 {
				t.Fatalf("forbidden bridge effects: %+v %v", due, err)
			}
		})
	}
}

func TestTerminalBridgeDisabledBacklogDoesNotStarveOrdinaryDelivery(t *testing.T) {
	s, _, _, _ := nativeBridgeServer(t)
	id := terminalBridgeSubscription(t, s, true)
	ctx := context.Background()
	intent, eventID := terminalAdmissionIntent(t, func(e *eventbus.DomainEvent) { e.Occurred = time.Now().UTC().Add(-time.Minute) })
	if _, err := s.acceptTerminalEventIntent(ctx, intent, eventID); err != nil {
		t.Fatal(err)
	}
	queue := s.subStore.(subscription.QueueStore)
	due, _ := queue.Due(ctx, 10)
	for i := 0; i < 30; i++ {
		if err := queue.Enqueue(ctx, id, fmt.Sprintf("held-terminal-%d", i), due[0].PayloadJSON); err != nil {
			t.Fatal(err)
		}
	}
	sub, _ := s.subStore.Get(ctx, id)
	sub.Name, sub.DeliveryTarget = "edited-terminal", "https://unused.example"
	if err := s.subStore.Update(ctx, sub); err != nil {
		t.Fatal(err)
	}
	ordinaryID, err := s.subStore.Create(ctx, subscription.Subscription{Name: "ordinary", OwnerScenario: "fixture", EventPattern: "ordinary.*", DeliveryType: subscription.DeliveryWebhook, DeliveryTarget: "https://unused.example", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := queue.Enqueue(ctx, ordinaryID, "ordinary-event", "invalid-json"); err != nil {
		t.Fatal(err)
	}
	s.flywheel = nil
	s.drainSubscriptionQueue(ctx, queue)
	ordinary, err := queue.GetDelivery(ctx, ordinaryID, "ordinary-event")
	if err != nil || ordinary.Status != "dead_letter" || ordinary.Attempts != 1 {
		t.Fatal("terminal backlog starved ordinary queue", err)
	}
	held, err := queue.GetDelivery(ctx, id, eventID)
	if err != nil || held.Status != "pending" || held.Attempts != 0 {
		t.Fatal("disabled terminal row consumed", err)
	}
}

func TestTerminalBridgeEditedTargetAndUntypedBacklogDoesNotStarveOrdinary(t *testing.T) {
	s, _, _, _ := nativeBridgeServer(t)
	id := terminalBridgeSubscription(t, s, true)
	ctx := context.Background()
	sub, err := s.subStore.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	sub.DeliveryTarget = "https://unused.example"
	if err := s.subStore.Update(ctx, sub); err != nil {
		t.Fatal(err)
	}
	queue := s.subStore.(subscription.QueueStore)
	for i := 0; i < 30; i++ {
		if err := queue.Enqueue(ctx, id, fmt.Sprintf("held-untyped-%d", i), "invalid-json"); err != nil {
			t.Fatal(err)
		}
	}
	ordinaryID, err := s.subStore.Create(ctx, subscription.Subscription{Name: "ordinary", OwnerScenario: "fixture", EventPattern: "ordinary.*", DeliveryType: subscription.DeliveryWebhook, DeliveryTarget: "https://unused.example", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := queue.Enqueue(ctx, ordinaryID, "ordinary-event", "invalid-json"); err != nil {
		t.Fatal(err)
	}
	s.flywheel = nil
	s.drainSubscriptionQueue(ctx, queue)
	ordinary, err := queue.GetDelivery(ctx, ordinaryID, "ordinary-event")
	if err != nil || ordinary.Status != "dead_letter" || ordinary.Attempts != 1 {
		t.Fatal("reserved terminal name starved ordinary queue", err)
	}
	held, err := queue.GetDelivery(ctx, id, "held-untyped-0")
	if err != nil || held.Status != "pending" || held.Attempts != 0 {
		t.Fatal("disabled terminal row consumed", err)
	}
}
