package storageapproval

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/operatorcapability"
)

func TestProviderDiscoversMissingHostLocalApprovals(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/cleanup/approvals" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"journald":{"host_id":"other-host"}}`))
	}))
	defer server.Close()

	provider := NewWithClient(func() string { return server.URL }, server.Client(), "this-host")
	status, err := provider.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.State != operatorcapability.StateNeedsInput || len(status.MissingInputs) != len(providerIDs) {
		t.Fatalf("status = %+v", status)
	}
}

func TestProviderAppliesOnlySelectedApprovals(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		var body struct {
			HostID string `json:"host_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.HostID != "this-host" {
			t.Fatalf("approval body = %+v, err=%v", body, err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := NewWithClient(func() string { return server.URL }, server.Client(), "this-host")
	inputs, err := provider.Descriptor().ValidateInputs(map[string]json.RawMessage{
		"docker-unused-images":    json.RawMessage(`true`),
		"docker-unused-volumes":   json.RawMessage(`false`),
		"journald":                json.RawMessage(`false`),
		"log-volume-force-rotate": json.RawMessage(`false`),
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Apply(context.Background(), inputs)
	if err != nil || result.State != operatorcapability.StateReady {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if len(calls) != 1 || calls[0] != "POST /api/v1/cleanup/approvals/docker-unused-images" {
		t.Fatalf("calls = %v", calls)
	}
}

func TestProviderVerificationReturnsContextBoundApprovalEvidence(t *testing.T) {
	approvedAt := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/cleanup/approvals" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		approvals := make(map[string]map[string]any, len(providerIDs))
		for _, id := range providerIDs {
			approvals[id] = map[string]any{"host_id": "this-host", "approved_at": approvedAt}
		}
		if err := json.NewEncoder(w).Encode(approvals); err != nil {
			t.Fatal(err)
		}
	}))
	defer server.Close()

	provider := NewWithClient(func() string { return server.URL }, server.Client(), "this-host")
	provider.now = func() time.Time { return approvedAt.Add(time.Hour) }
	registry, err := operatorcapability.NewRegistry(provider)
	if err != nil {
		t.Fatal(err)
	}
	receipts, err := registry.Verify(context.Background(), operatorcapability.VerificationRequest{
		CapabilityID: CapabilityID, TargetID: "local", Environment: "development", AccountIdentity: "operator", Operation: "readiness-check",
		Effect: operatorcapability.EffectBudget{Class: operatorcapability.EffectReadOnly},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(receipts) != len(providerIDs) {
		t.Fatalf("receipts = %d, want %d", len(receipts), len(providerIDs))
	}
	for _, receipt := range receipts {
		if receipt.TargetID != "local" || receipt.Environment != "development" || receipt.AccountIdentity != "operator" || receipt.Operation != "readiness-check" || !receipt.Verified || !receipt.FreshAt(approvedAt.Add(time.Hour)) {
			t.Fatalf("receipt was not bound to the request: %+v", receipt)
		}
		if len(receipt.Coverage) != 3 || receipt.ObservedAt != approvedAt {
			t.Fatalf("receipt lost approval context: %+v", receipt)
		}
	}
}

func TestProviderVerificationRejectsMissingApprovalTimestamp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		approvals := make(map[string]map[string]string, len(providerIDs))
		for _, id := range providerIDs {
			approvals[id] = map[string]string{"host_id": "this-host"}
		}
		_ = json.NewEncoder(w).Encode(approvals)
	}))
	defer server.Close()

	provider := NewWithClient(func() string { return server.URL }, server.Client(), "this-host")
	registry, err := operatorcapability.NewRegistry(provider)
	if err != nil {
		t.Fatal(err)
	}
	_, err = registry.Verify(context.Background(), operatorcapability.VerificationRequest{
		CapabilityID: CapabilityID, TargetID: "local", Operation: "readiness-check",
		Effect: operatorcapability.EffectBudget{Class: operatorcapability.EffectReadOnly},
	})
	if err == nil {
		t.Fatal("verification accepted an approval without an observed approval timestamp")
	}
	var verificationErr *operatorcapability.VerificationError
	if !errors.As(err, &verificationErr) || verificationErr.Code != "storage_approval_invalid" || verificationErr.Retryable || verificationErr.NextAction != "repair-storage-manager-approval" {
		t.Fatalf("missing timestamp error = %v, want typed non-retryable approval failure", err)
	}
}

func TestProviderVerificationClassifiesRateLimitForBoundedRetry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "2")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	provider := NewWithClient(func() string { return server.URL }, server.Client(), "this-host")
	_, err := provider.Verify(context.Background(), operatorcapability.VerificationRequest{
		CapabilityID: CapabilityID, TargetID: "local", Operation: "readiness-check",
		Effect: operatorcapability.EffectBudget{Class: operatorcapability.EffectReadOnly},
	})
	var verificationErr *operatorcapability.VerificationError
	if !errors.As(err, &verificationErr) {
		t.Fatalf("rate limit error = %v, want typed verification error", err)
	}
	if verificationErr.Code != "provider_rate_limited" || !verificationErr.Retryable || verificationErr.RetryAfter != 2*time.Second || verificationErr.NextAction != "retry-after-provider-backoff" {
		t.Fatalf("rate limit classification = %+v", verificationErr)
	}
}
