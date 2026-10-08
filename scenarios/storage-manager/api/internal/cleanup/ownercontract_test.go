package cleanup

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPScenarioProviderClientDistinguishesOwnerStatesAndPreservesProtectedItems(t *testing.T) {
	var applied OwnerApplyRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/cleanup/estimate":
			if r.URL.Query().Get("min_age_seconds") != "604800" {
				t.Errorf("min_age_seconds = %q", r.URL.Query().Get("min_age_seconds"))
			}
			_ = json.NewEncoder(w).Encode(OwnerEstimateResponse{ProviderID: "owner-cleanup", EstimatedBytes: 100, ItemCount: 2, ObservedAt: time.Now().UTC()})
		case "/api/v1/cleanup/preview":
			_ = json.NewEncoder(w).Encode(OwnerPreviewResponse{ProviderID: "owner-cleanup", MinAgeSeconds: 604800, KeepCount: 10, MaxBytes: 1000, Items: []OwnerPreviewItem{{ID: "old", Path: "/old", Bytes: 100}, {ID: "active", Path: "/active", Bytes: 200, Protected: true}}})
		case "/api/v1/cleanup/apply":
			if err := json.NewDecoder(r.Body).Decode(&applied); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(w).Encode(OwnerApplyResponse{ReclaimedBytes: 100, RemovedItemIDs: []string{"old"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := &HTTPScenarioProviderClient{ResolveURL: func(context.Context, string) (string, error) { return server.URL, nil }, HTTPClient: server.Client()}
	policy := ProviderPolicy{Enabled: true, MinAge: 7 * 24 * time.Hour, MaxBytes: 1000}
	estimate, err := client.Estimate(context.Background(), "owner", policy)
	if err != nil || estimate.EstimatedBytes != 100 {
		t.Fatalf("estimate=%#v err=%v", estimate, err)
	}
	preview, err := client.Preview(context.Background(), "owner", estimate)
	if err != nil || len(preview.Items) != 1 || preview.Items[0].ID != "old" {
		t.Fatalf("preview=%#v err=%v", preview, err)
	}
	if preview.MinAge != 7*24*time.Hour || preview.KeepCount != 10 || preview.MaxBytes != 1000 {
		t.Fatalf("preview policy was not preserved: %#v", preview)
	}
	result, err := client.Apply(context.Background(), ScenarioCleanupRequest{ScenarioID: "owner", ProviderID: "owner-cleanup", IdempotencyKey: "once", Preview: preview})
	if err != nil || result.ReclaimedBytes != 100 || applied.IdempotencyKey != "once" {
		t.Fatalf("apply=%#v request=%#v err=%v", result, applied, err)
	}
	if applied.Preview.MinAgeSeconds != 604800 || applied.Preview.KeepCount != 10 || applied.Preview.MaxBytes != 1000 {
		t.Fatalf("apply policy was weakened in transport: %#v", applied.Preview)
	}
}

func TestHTTPScenarioProviderClientReturnsContractBlockedReasons(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	client := &HTTPScenarioProviderClient{ResolveURL: func(context.Context, string) (string, error) { return server.URL, nil }, HTTPClient: server.Client()}
	estimate, err := client.Estimate(context.Background(), "owner", ProviderPolicy{Enabled: true})
	if err != nil || estimate.BlockedReason != "owner scenario does not implement cleanup" {
		t.Fatalf("404 estimate=%#v err=%v", estimate, err)
	}
	client.ResolveURL = func(context.Context, string) (string, error) { return "", context.DeadlineExceeded }
	estimate, err = client.Estimate(context.Background(), "owner", ProviderPolicy{Enabled: true})
	if err != nil || estimate.BlockedReason != "owner scenario unreachable" {
		t.Fatalf("unreachable estimate=%#v err=%v", estimate, err)
	}
	var nilClient *HTTPScenarioProviderClient
	estimate, err = nilClient.Estimate(context.Background(), "owner", ProviderPolicy{Enabled: true})
	if err != nil || estimate.BlockedReason != "owner scenario client unavailable" {
		t.Fatalf("nil estimate=%#v err=%v", estimate, err)
	}
}

func TestHTTPScenarioProviderClientUsesBoundedOperationTimeouts(t *testing.T) {
	client := &HTTPScenarioProviderClient{}
	if got := client.client(30 * time.Second).Timeout; got != 30*time.Second {
		t.Fatalf("read timeout = %s, want 30s", got)
	}
	if got := client.client(10 * time.Minute).Timeout; got != 10*time.Minute {
		t.Fatalf("apply timeout = %s, want 10m", got)
	}
	custom := &HTTPScenarioProviderClient{HTTPClient: &http.Client{Timeout: 2 * time.Second}}
	if got := custom.client(30 * time.Second).Timeout; got != 2*time.Second {
		t.Fatalf("explicit client timeout = %s, want 2s", got)
	}
}
