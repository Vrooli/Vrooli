package process

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func processTestClient(baseURL string) *Client {
	apiClient := cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}),
		func() cliutil.APIBaseOptions {
			return cliutil.APIBaseOptions{Override: baseURL}
		},
		nil,
	)
	return NewClient(apiClient)
}

func TestRunKillRejectsInvalidPID(t *testing.T) {
	err := Run(nil, []string{"kill", "dep-123", "not-a-number"})
	if err == nil {
		t.Fatal("expected invalid PID error")
	}
	if !strings.Contains(err.Error(), "invalid PID") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunRestartRejectsInvalidType(t *testing.T) {
	err := Run(nil, []string{"restart", "dep-123", "worker", "name"})
	if err == nil {
		t.Fatal("expected invalid type error")
	}
	if !strings.Contains(err.Error(), "type must be 'scenario' or 'resource'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunControlDefaultsTypeToAll(t *testing.T) {
	var gotReq ControlRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/deployments/dep-123/actions/process" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"action":"restart","results":[]}`))
	}))
	defer server.Close()

	client := processTestClient(server.URL)
	if err := Run(client, []string{"control", "dep-123", "restart"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if gotReq.Type != "all" {
		t.Fatalf("expected default type 'all', got %q", gotReq.Type)
	}
	if gotReq.Action != "restart" {
		t.Fatalf("expected action restart, got %q", gotReq.Action)
	}
}
