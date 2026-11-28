package monetization

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQuotaReservationClientUsesAuthenticatedAtomicLPBSContract(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("Authorization") != "Bearer fake-session" {
			t.Errorf("authorization header = %q", r.Header.Get("Authorization"))
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/usage/reservations":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body["limit_key"] != "workflow_executions" || body["amount"] != float64(1) || body["window_seconds"] != float64(0) {
				t.Errorf("reserve payload = %#v", body)
			}
			_, _ = w.Write([]byte(`{"reservation_id":"hold-1","status":"pending"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/usage/reservations/hold-1/finalize":
			var body map[string]int64
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["actual_amount"] != 1 {
				t.Errorf("finalize payload = %#v, err=%v", body, err)
			}
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/usage/reservations/hold-1/release":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected quota request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := QuotaReservationClient{BaseURL: server.URL, HTTPClient: server.Client()}
	id, err := client.Reserve(context.Background(), "fake-session", "workflow_executions", 1, 0)
	if err != nil || id != "hold-1" {
		t.Fatalf("Reserve() = %q, %v", id, err)
	}
	if err := client.Finalize(context.Background(), "fake-session", id, 1); err != nil {
		t.Fatalf("Finalize(): %v", err)
	}
	if err := client.Release(context.Background(), "fake-session", id); err != nil {
		t.Fatalf("Release(): %v", err)
	}
	if requests != 3 {
		t.Fatalf("request count = %d, want 3", requests)
	}
}

func TestQuotaReservationClientRejectsUnsafeAuthoritiesAndRedirects(t *testing.T) {
	if _, err := (QuotaReservationClient{BaseURL: "http://lpbs.example"}).Reserve(context.Background(), "token", "workflow_executions", 1, 0); err == nil {
		t.Fatal("remote plain HTTP authority was accepted")
	}
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://lpbs.example/collect", http.StatusFound)
	}))
	defer redirect.Close()
	if _, err := (QuotaReservationClient{BaseURL: redirect.URL, HTTPClient: redirect.Client()}).Reserve(context.Background(), "token", "workflow_executions", 1, 0); err == nil {
		t.Fatal("redirect response was accepted")
	}
}

func TestQuotaHoldCannotFinalizeMoreThanReserved(t *testing.T) {
	var finalized bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/usage/reservations" {
			_, _ = w.Write([]byte(`{"reservation_id":"hold-1"}`))
			return
		}
		if r.URL.Path == "/api/v1/usage/reservations/hold-1/finalize" {
			finalized = true
		}
	}))
	defer server.Close()
	hold, err := (QuotaReservationClient{BaseURL: server.URL, HTTPClient: server.Client()}).Begin(context.Background(), "token", "workflow_executions", 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := hold.Commit(context.Background(), 2); err == nil {
		t.Fatal("oversized finalization was accepted")
	}
	if finalized {
		t.Fatal("oversized finalization reached LPBS")
	}
}
