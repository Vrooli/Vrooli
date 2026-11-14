package integrations

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRetrySubscriptionsRecoversAndMaintainsRoutes(t *testing.T) {
	attempts := map[string]int{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	RetrySubscriptions(ctx, time.Millisecond, func(_ context.Context, kind string) error {
		attempts[kind]++
		if kind == "notification" && attempts[kind] == 2 {
			cancel()
		}
		if kind == "decision" && attempts[kind] == 1 {
			return errors.New("Events unavailable")
		}
		return nil
	}, "decision", "notification")
	if attempts["decision"] != 2 || attempts["notification"] != 2 {
		t.Fatalf("unexpected attempts: %v", attempts)
	}
}

func TestDecisionSubscriptionRepairsNamedRowAfterPortChange(t *testing.T) {
	updated := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`[{"id":42,"name":"notification-hub-decisions","event_pattern":"*.decision_requested.v1","delivery_target":"http://old/events","enabled":false}]`))
			return
		}
		if r.Method != http.MethodPut || r.URL.Path != "/api/v1/subscriptions/42" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if payload["delivery_target"] != "http://new/events" || payload["enabled"] != true {
			t.Errorf("wrong replacement: %+v", payload)
		}
		updated = true
	}))
	defer server.Close()
	if err := EnsureDecisionSubscription(context.Background(), server.URL, "http://new/events"); err != nil {
		t.Fatal(err)
	}
	if !updated {
		t.Fatal("named subscription not repaired")
	}
}

func TestSubscriptionListDecodeFailureDoesNotCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("must not write after invalid inventory: %s", r.Method)
		}
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer server.Close()
	if err := EnsureDecisionSubscription(context.Background(), server.URL, "http://new/events"); err == nil {
		t.Fatal("invalid inventory must fail")
	}
}

func TestRetrySubscriptionsHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0
	RetrySubscriptions(ctx, time.Hour, func(context.Context, string) error { attempts++; cancel(); return errors.New("offline") }, "decision")
	if attempts != 1 {
		t.Fatalf("attempts = %d", attempts)
	}
}
