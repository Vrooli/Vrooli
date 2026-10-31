package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	"web-console/internal/backend"
	"web-console/internal/policy"
	"web-console/internal/ptyfake"
	"web-console/internal/sessionstore"
)

func TestPaidCodingProxyBindsOpenCodeRequestToSessionAccountAndStreamsTools(t *testing.T) {
	activePaidCodingSessions = newPaidCodingSessionRegistry()
	t.Cleanup(func() { activePaidCodingSessions.Clear() })
	providerTool := json.RawMessage(`{"type":"function","function":{"name":"read_file","parameters":{"type":"object"}}}`)
	lpbs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/refresh":
			_, _ = w.Write([]byte(`{"access_token":"lpbs-access","refresh_token":"lpbs-refresh","expires_at":"2030-01-01T00:00:00Z"}`))
		case "/api/v1/ai/inference-stream":
			if r.Header.Get("Authorization") != "Bearer lpbs-access" {
				t.Errorf("LPBS auth = %q", r.Header.Get("Authorization"))
			}
			var request struct {
				Role              string            `json:"role"`
				BusinessAccountID string            `json:"business_account_id"`
				Messages          []json.RawMessage `json:"messages"`
				Tools             []json.RawMessage `json:"tools"`
				Metadata          map[string]string `json:"metadata"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode LPBS request: %v", err)
			}
			if request.Role != paidCodingRole || request.BusinessAccountID != "acct-selected" || len(request.Messages) != 1 || len(request.Tools) != 1 || string(request.Tools[0]) != string(providerTool) {
				t.Errorf("LPBS request = %#v", request)
			}
			if request.Metadata["operation"] != "interactive_coding" {
				t.Errorf("operation attribution = %#v", request.Metadata)
			}
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: {\"type\":\"chunk\",\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"read_file\",\"arguments\":\"{\\\"path\\\":\\\"src/main.go\\\"}\"}}]}\n\n"))
			_, _ = w.Write([]byte("data: {\"type\":\"done\",\"usage\":{\"prompt_tokens\":30,\"completion_tokens\":8,\"total_tokens\":38,\"credits_charged\":10}}\n\n"))
		default:
			t.Errorf("unexpected LPBS path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer lpbs.Close()

	credentials := &businessAccountCredentialFixture{}
	resolver := &credentialclient.ConsumerSessionResolver{
		Credentials: credentials, LPBSBaseURL: lpbs.URL, HTTPClient: lpbs.Client(),
		Now: func() time.Time { return time.Date(2029, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
	manager := newSessionManagerWithFactory(ptyfake.NewFactory())
	created, err := manager.CreateWithOptions(context.Background(), "/bin/sh", 80, 24, backend.ID("standard"), (*policy.Policy)(nil), t.TempDir(), false)
	if err != nil {
		t.Fatalf("create fixture session: %v", err)
	}
	defer func() { _ = manager.Delete(context.Background(), created.ID) }()
	store := sessionstore.NewInMemory()
	if err := store.Save(context.Background(), sessionstore.Metadata{ID: created.ID, AgentType: sessionstore.AgentOpenCode, BusinessAccountID: "acct-selected", Status: sessionstore.StatusLive}); err != nil {
		t.Fatalf("save fixture metadata: %v", err)
	}
	token, err := activePaidCodingSessions.Bind(created.ID, "acct-selected")
	if err != nil {
		t.Fatalf("bind fixture session: %v", err)
	}
	srv := &Server{subscriptionResolver: resolver, sessionStore: store, sessions: manager, paidCodingProtocolQualified: true}
	body := `{"model":"vrooli/coding","stream":true,"messages":[{"role":"user","content":"inspect src/main.go"}],"tools":[{"type":"function","function":{"name":"read_file","parameters":{"type":"object"}}}],"tool_choice":"auto"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/paid-coding/v1/chat/completions", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	srv.paidCodingCompletionsHandler(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("paid coding status=%d body=%q", response.Code, response.Body.String())
	}
	for _, expected := range []string{`"model":"vrooli/coding"`, `"tool_calls":[{"index":0,"id":"call_1"`, `"finish_reason":"tool_calls"`, `"prompt_tokens":30`, "data: [DONE]"} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("paid coding response missing %q: %s", expected, response.Body.String())
		}
	}
}

func TestPaidCodingProxyRejectsWrongSessionAccountBeforeProviderRequest(t *testing.T) {
	activePaidCodingSessions = newPaidCodingSessionRegistry()
	t.Cleanup(func() { activePaidCodingSessions.Clear() })
	manager := newSessionManagerWithFactory(ptyfake.NewFactory())
	created, err := manager.CreateWithOptions(context.Background(), "/bin/sh", 80, 24, backend.ID("standard"), (*policy.Policy)(nil), t.TempDir(), false)
	if err != nil {
		t.Fatalf("create fixture session: %v", err)
	}
	defer func() { _ = manager.Delete(context.Background(), created.ID) }()
	store := sessionstore.NewInMemory()
	if err := store.Save(context.Background(), sessionstore.Metadata{ID: created.ID, AgentType: sessionstore.AgentOpenCode, BusinessAccountID: "acct-other", Status: sessionstore.StatusLive}); err != nil {
		t.Fatalf("save fixture metadata: %v", err)
	}
	token, err := activePaidCodingSessions.Bind(created.ID, "acct-selected")
	if err != nil {
		t.Fatalf("bind fixture session: %v", err)
	}
	srv := &Server{sessionStore: store, sessions: manager, paidCodingProtocolQualified: true}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/paid-coding/v1/chat/completions", strings.NewReader(`{"messages":[{"role":"user","content":"x"}]}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	srv.paidCodingCompletionsHandler(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("mismatched account status = %d, want 401", response.Code)
	}
	if _, ok := activePaidCodingSessions.Lookup(token); ok {
		t.Fatal("stale mismatched account binding was not revoked")
	}
}

func TestPaidCodingProxyRemainsUnavailableUntilProtocolQualification(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/paid-coding/v1/chat/completions", strings.NewReader(`{}`))
	response := httptest.NewRecorder()
	(&Server{}).paidCodingCompletionsHandler(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("unqualified route status = %d, want 503", response.Code)
	}
}

func TestPaidCodingStreamErrorsUseProtocolErrorAndRejectIncompleteStreams(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "upstream error", body: "data: {\"type\":\"error\",\"error\":\"provider failed\"}\n\n"},
		{name: "malformed event", body: "data: {not-json}\n\n"},
		{name: "early eof", body: "data: {\"type\":\"chunk\",\"content\":\"partial\"}\n\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			(&Server{}).forwardPaidCodingStream(context.Background(), response, strings.NewReader(tt.body), "vrooli/coding")
			body := response.Body.String()
			if !strings.Contains(body, `"error":{"message":`) || !strings.Contains(body, `"type":"server_error"`) {
				t.Fatalf("stream response lacks a protocol error object: %s", body)
			}
			if strings.Contains(body, `"finish_reason":"stop"`) || strings.Contains(body, "data: [DONE]") {
				t.Fatalf("failed stream was presented as a completed response: %s", body)
			}
		})
	}
}

func TestPaidCodingProxyCancellationReachesLPBSStream(t *testing.T) {
	activePaidCodingSessions = newPaidCodingSessionRegistry()
	t.Cleanup(func() { activePaidCodingSessions.Clear() })
	upstreamStarted := make(chan struct{})
	upstreamCanceled := make(chan struct{})
	lpbs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/refresh" {
			_, _ = w.Write([]byte(`{"access_token":"lpbs-access","refresh_token":"lpbs-refresh","expires_at":"2030-01-01T00:00:00Z"}`))
			return
		}
		if r.URL.Path != "/api/v1/ai/inference-stream" {
			http.NotFound(w, r)
			return
		}
		close(upstreamStarted)
		w.Header().Set("Content-Type", "text/event-stream")
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		<-r.Context().Done()
		close(upstreamCanceled)
	}))
	defer lpbs.Close()
	resolver := &credentialclient.ConsumerSessionResolver{
		Credentials: &businessAccountCredentialFixture{}, LPBSBaseURL: lpbs.URL, HTTPClient: lpbs.Client(),
		Now: func() time.Time { return time.Date(2029, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
	manager := newSessionManagerWithFactory(ptyfake.NewFactory())
	created, err := manager.CreateWithOptions(context.Background(), "/bin/sh", 80, 24, backend.ID("standard"), (*policy.Policy)(nil), t.TempDir(), false)
	if err != nil {
		t.Fatalf("create fixture session: %v", err)
	}
	defer func() { _ = manager.Delete(context.Background(), created.ID) }()
	store := sessionstore.NewInMemory()
	if err := store.Save(context.Background(), sessionstore.Metadata{ID: created.ID, AgentType: sessionstore.AgentOpenCode, BusinessAccountID: "acct-selected", Status: sessionstore.StatusLive}); err != nil {
		t.Fatalf("save fixture metadata: %v", err)
	}
	token, err := activePaidCodingSessions.Bind(created.ID, "acct-selected")
	if err != nil {
		t.Fatalf("bind fixture session: %v", err)
	}
	srv := &Server{subscriptionResolver: resolver, sessionStore: store, sessions: manager, paidCodingProtocolQualified: true}
	ctx, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequest(http.MethodPost, "/api/v1/paid-coding/v1/chat/completions", strings.NewReader(`{"stream":true,"messages":[{"role":"user","content":"wait"}]}`)).WithContext(ctx)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	completed := make(chan struct{})
	go func() {
		srv.paidCodingCompletionsHandler(response, request)
		close(completed)
	}()
	select {
	case <-upstreamStarted:
	case <-time.After(2 * time.Second):
		cancel()
		t.Fatal("Web Console proxy did not start LPBS stream")
	}
	cancel()
	select {
	case <-upstreamCanceled:
	case <-time.After(2 * time.Second):
		t.Fatal("LPBS stream did not observe cancellation")
	}
	select {
	case <-completed:
	case <-time.After(2 * time.Second):
		t.Fatal("Web Console proxy did not finish after cancellation")
	}
}
