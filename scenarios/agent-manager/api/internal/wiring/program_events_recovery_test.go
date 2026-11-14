package wiring

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDecisionSubscriptionReconcilesChangedPortAndDisabledRow(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		t.Run(map[bool]string{true: "old-port", false: "disabled"}[enabled], func(t *testing.T) {
			var updated bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodGet:
					_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 42, "name": "agent-manager-decision-answers", "event_pattern": askResolvedEventPattern, "delivery_target": "http://127.0.0.1:1111/answers", "enabled": enabled}})
				case http.MethodPut:
					if r.URL.Path != "/api/v1/subscriptions/42" {
						t.Errorf("wrong row: %s", r.URL.Path)
					}
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					if body["delivery_target"] != "http://127.0.0.1:2222/answers" || body["enabled"] != true || body["event_pattern"] != askResolvedEventPattern {
						t.Errorf("wrong replacement: %+v", body)
					}
					updated = true
				default:
					t.Errorf("unexpected request: %s", r.Method)
				}
			}))
			defer server.Close()
			if err := reconcileEventSubscriptionAt(context.Background(), server.URL, "agent-manager-decision-answers", askResolvedEventPattern, "http://127.0.0.1:2222/answers"); err != nil {
				t.Fatal(err)
			}
			if !updated {
				t.Fatal("existing named subscription was not repaired")
			}
		})
	}
}

func TestEventSubscriptionRetriesUnavailableOwnerIndependently(t *testing.T) {
	var first, second int
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	retryEventSubscriptions(ctx, time.Millisecond,
		func(context.Context) error {
			first++
			if first == 1 {
				return errors.New("Events unavailable")
			}
			return nil
		},
		func(context.Context) error {
			second++
			if second == 2 {
				cancel()
			}
			return nil
		},
	)
	if first != 2 || second != 2 {
		t.Fatalf("both routes must be maintained after recovery: first=%d second=%d", first, second)
	}
}

func TestEventSubscriptionRetryStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0
	retryEventSubscriptions(ctx, time.Hour, func(context.Context) error { attempts++; cancel(); return errors.New("unavailable") })
	if attempts != 1 {
		t.Fatalf("cancelled retry attempted %d times", attempts)
	}
}

func TestWebhookRecoversAfterCredentialOwnerUnavailableAtStartup(t *testing.T) {
	resolved := false
	lookups := 0
	secret := recoveringEventsSecret("", func() (string, error) {
		lookups++
		if !resolved {
			return "", errors.New("credential owner unavailable")
		}
		return "signing-secret", nil
	})
	handler := recoveringEventsWebhook(secret, func(value string) http.Handler {
		if value != "signing-secret" {
			t.Fatalf("incorrect secret passed to verifier")
		}
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	})
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, request)
	if first.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing verifier must be retryable and fail closed: %d", first.Code)
	}
	resolved = true
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, request)
	third := httptest.NewRecorder()
	handler.ServeHTTP(third, request)
	if second.Code != http.StatusNoContent || third.Code != http.StatusNoContent || lookups != 2 {
		t.Fatalf("recovery/cache: codes=%d,%d lookups=%d", second.Code, third.Code, lookups)
	}
}
