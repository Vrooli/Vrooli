package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/api-core/eventbus"
)

func TestDecisionSubscriptionReconcilesAfterEventsBecomesAvailable(t *testing.T) {
	var gets, posts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			gets.Add(1)
			_, _ = w.Write([]byte(`[]`))
		case http.MethodPost:
			posts.Add(1)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":1}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	var resolutions, reported, waits atomic.Int32
	resolve := func(context.Context) (string, error) {
		if resolutions.Add(1) <= 5 {
			return "", errors.New("scenario_not_running")
		}
		return server.URL, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	wait := func(ctx context.Context, interval time.Duration) bool {
		if interval != decisionSubscriptionCycle {
			return true
		}
		if waits.Add(1) == 1 {
			return true
		}
		cancel()
		return false
	}
	runDecisionEventSubscriptionReconciler(ctx, "http://127.0.0.1:1234/webhook", resolve,
		decisionSubscriptionCycle, decisionSubscriptionWindow, 0, wait, func(error) { reported.Add(1) })
	if resolutions.Load() != 6 || gets.Load() != 1 || posts.Load() != 1 || reported.Load() != 1 || waits.Load() != 2 {
		t.Fatalf("resolutions=%d gets=%d posts=%d reported=%d waits=%d; want 6/1/1/1/2", resolutions.Load(), gets.Load(), posts.Load(), reported.Load(), waits.Load())
	}
}

func TestDecisionSubscriptionRetriesAreBounded(t *testing.T) {
	var attempts atomic.Int32
	resolve := func(context.Context) (string, error) {
		attempts.Add(1)
		return "", errors.New("scenario_not_running")
	}
	err := reconcileDecisionEventSubscriptionWithRetry(context.Background(), "http://127.0.0.1:1234/webhook", resolve, 0)
	if err == nil {
		t.Fatal("unavailable Events unexpectedly reconciled")
	}
	if attempts.Load() != 5 {
		t.Fatalf("attempts=%d, want the five-attempt bound", attempts.Load())
	}
}

func TestDecisionSubscriptionReconcilerStopsRetriesOnShutdown(t *testing.T) {
	var attempts, waits atomic.Int32
	resolve := func(context.Context) (string, error) {
		attempts.Add(1)
		return "", errors.New("scenario_not_running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	waiting := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		runDecisionEventSubscriptionReconciler(ctx, "http://127.0.0.1:1234/webhook", resolve,
			decisionSubscriptionCycle, decisionSubscriptionWindow, 0, func(ctx context.Context, interval time.Duration) bool {
				if interval != decisionSubscriptionCycle {
					return true
				}
				waits.Add(1)
				close(waiting)
				<-ctx.Done()
				return false
			}, nil)
	}()
	<-waiting
	cancel()
	<-done
	if attempts.Load() != 5 || waits.Load() != 1 {
		t.Fatalf("attempts=%d waits=%d after shutdown; want five attempts and no later cycle", attempts.Load(), waits.Load())
	}
}

func TestDecisionSubscriptionReconciliationDisablesOwnedDuplicates(t *testing.T) {
	pattern := "notification_hub.ask.resolved.v1"
	rows := []eventbus.Subscription{
		{ID: 8, Name: decisionSubscriptionName, OwnerScenario: decisionSubscriptionOwner, EventPattern: pattern, DeliveryType: "webhook", DeliveryTarget: "http://old:1234/hook", Enabled: true},
		{ID: 3, Name: decisionSubscriptionName, OwnerScenario: decisionSubscriptionOwner, EventPattern: "old.event", DeliveryType: "webhook", DeliveryTarget: "http://old:1234/hook", Enabled: true},
		{ID: 2, Name: "legacy-answer-route", OwnerScenario: decisionSubscriptionOwner, EventPattern: pattern, DeliveryType: "webhook", DeliveryTarget: "http://127.0.0.1:4321/hook", Enabled: true},
		{ID: 4, Name: "another-legacy-answer-route", OwnerScenario: decisionSubscriptionOwner, EventPattern: pattern, DeliveryType: "webhook", DeliveryTarget: "http://127.0.0.1:4321/hook", Enabled: true},
		{ID: 5, Name: "unrelated-target", OwnerScenario: decisionSubscriptionOwner, EventPattern: pattern, DeliveryType: "webhook", DeliveryTarget: "http://other:4321/hook", Enabled: true},
		{ID: 6, Name: "unrelated-pattern", OwnerScenario: decisionSubscriptionOwner, EventPattern: "other.event", DeliveryType: "webhook", DeliveryTarget: "http://127.0.0.1:4321/hook", Enabled: true},
		{ID: 7, Name: "other-owner", OwnerScenario: "another-scenario", EventPattern: pattern, DeliveryType: "webhook", DeliveryTarget: "http://127.0.0.1:4321/hook", Enabled: true},
	}
	var puts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(rows)
		case http.MethodPut:
			id, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/api/v1/subscriptions/"), 10, 64)
			if err != nil {
				t.Errorf("parse update id: %v", err)
				http.Error(w, "bad id", http.StatusBadRequest)
				return
			}
			var updated eventbus.Subscription
			if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
				t.Errorf("decode update: %v", err)
				http.Error(w, "bad body", http.StatusBadRequest)
				return
			}
			updated.ID = id
			for i := range rows {
				if rows[i].ID == id {
					rows[i] = updated
				}
			}
			puts.Add(1)
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()
	if err := reconcileDecisionEventSubscriptionWithResolver(context.Background(), "http://127.0.0.1:4321/hook", func(context.Context) (string, error) { return server.URL, nil }); err != nil {
		t.Fatalf("reconcile duplicates: %v", err)
	}
	enabled := 0
	for _, row := range rows {
		if row.Enabled {
			enabled++
		}
	}
	if enabled != 4 || puts.Load() != 4 {
		t.Fatalf("enabled rows=%d PUTs=%d, want one owned route plus three unrelated rows and four canonical/disable updates", enabled, puts.Load())
	}
	for _, row := range rows {
		switch row.ID {
		case 2:
			if !row.Enabled || row.Name != decisionSubscriptionName || row.DeliveryTarget != "http://127.0.0.1:4321/hook" {
				t.Errorf("canonical mixed-name row = %+v", row)
			}
		case 3, 4, 8:
			if row.Enabled {
				t.Errorf("owned duplicate remains enabled: %+v", row)
			}
		case 5, 6, 7:
			if !row.Enabled {
				t.Errorf("unrelated subscription was changed: %+v", row)
			}
		}
	}
}
