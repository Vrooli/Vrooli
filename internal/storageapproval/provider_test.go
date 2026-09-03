package storageapproval

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
