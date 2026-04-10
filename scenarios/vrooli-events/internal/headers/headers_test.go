package headers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// [REQ:DI-006] X-Source-Scenario header injection tests

func TestInjectSource_SetsHeader(t *testing.T) {
	// [REQ:DI-006] InjectSource sets the header correctly
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	InjectSource(req, "my-scenario")

	got := req.Header.Get(HeaderSourceScenario)
	if got != "my-scenario" {
		t.Fatalf("expected header %q, got %q", "my-scenario", got)
	}
}

func TestExtractSource_ReadsHeader(t *testing.T) {
	// [REQ:DI-006] ExtractSource reads the header correctly
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(HeaderSourceScenario, "other-scenario")

	got := ExtractSource(req)
	if got != "other-scenario" {
		t.Fatalf("expected %q, got %q", "other-scenario", got)
	}
}

func TestSourceTransport_AddsHeader(t *testing.T) {
	// [REQ:DI-006] SourceTransport adds header to requests
	var receivedHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get(HeaderSourceScenario)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := &http.Client{
		Transport: &SourceTransport{Scenario: "transport-test"},
	}

	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if receivedHeader != "transport-test" {
		t.Fatalf("expected header %q, got %q", "transport-test", receivedHeader)
	}
}

func TestNewClient_CreatesConfiguredClient(t *testing.T) {
	// [REQ:DI-006] NewClient creates a properly configured client
	var receivedHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get(HeaderSourceScenario)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewClient("new-client-scenario")
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if receivedHeader != "new-client-scenario" {
		t.Fatalf("expected header %q, got %q", "new-client-scenario", receivedHeader)
	}
}

func TestInjectSource_EmptyScenario(t *testing.T) {
	// [REQ:DI-006] Empty scenario still sets header (no panic)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	InjectSource(req, "")

	got := req.Header.Get(HeaderSourceScenario)
	if got != "" {
		t.Fatalf("expected empty header value, got %q", got)
	}
}
