package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	"web-console/internal/backend"
	"web-console/internal/policy"
	"web-console/internal/ptyfake"
	"web-console/internal/sessionstore"
)

func TestOpenCodeCLIUsesAccountBoundProxyToolRoundTripWithLPBSFixture(t *testing.T) {
	openCodePath, err := exec.LookPath("opencode")
	if err != nil {
		t.Skip("OpenCode CLI is not installed")
	}
	activePaidCodingSessions = newPaidCodingSessionRegistry()
	t.Cleanup(func() { activePaidCodingSessions.Clear() })

	workDir := t.TempDir()
	fixturePath := filepath.Join(workDir, "fixture.txt")
	if err := os.WriteFile(fixturePath, []byte("account-bound fixture content\n"), 0o600); err != nil {
		t.Fatalf("write tool fixture: %v", err)
	}

	var mu sync.Mutex
	var requestCount int
	var toolCallSent bool
	accountIDs := []string{}
	roles := []string{}
	toolCounts := []int{}
	lpbs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/refresh" {
			_, _ = w.Write([]byte(`{"access_token":"fixture-lpbs-access","refresh_token":"fixture-lpbs-refresh","expires_at":"2030-01-01T00:00:00Z"}`))
			return
		}
		if r.URL.Path != "/api/v1/ai/inference-stream" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer fixture-lpbs-access" {
			t.Errorf("LPBS authorization header is missing the resolved fixture token")
		}
		var request struct {
			Role              string            `json:"role"`
			BusinessAccountID string            `json:"business_account_id"`
			Tools             []json.RawMessage `json:"tools"`
			Metadata          map[string]string `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode account-scoped LPBS request: %v", err)
			http.Error(w, "invalid fixture request", http.StatusBadRequest)
			return
		}
		mu.Lock()
		requestCount++
		toolCounts = append(toolCounts, len(request.Tools))
		accountIDs = append(accountIDs, request.BusinessAccountID)
		roles = append(roles, request.Role)
		sendToolCall := len(request.Tools) > 0 && !toolCallSent
		if sendToolCall {
			toolCallSent = true
		}
		hasSentToolCall := toolCallSent
		mu.Unlock()
		if request.Metadata["operation"] != "interactive_coding" {
			t.Errorf("LPBS coding request omitted operation attribution: %#v", request)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		writeEvent := func(event map[string]any) {
			encoded, _ := json.Marshal(event)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", encoded)
		}
		if sendToolCall {
			writeEvent(map[string]any{"type": "chunk", "tool_calls": []any{map[string]any{
				"index": 0, "id": "call_fixture_read", "type": "function",
				"function": map[string]string{"name": "read", "arguments": fmt.Sprintf(`{"filePath":%q}`, fixturePath)},
			}}})
		} else if hasSentToolCall {
			writeEvent(map[string]any{"type": "chunk", "content": "fixture-ok"})
		} else {
			writeEvent(map[string]any{"type": "chunk", "content": "fixture-ready"})
		}
		writeEvent(map[string]any{"type": "done", "usage": map[string]int{
			"prompt_tokens": 10, "completion_tokens": 2, "total_tokens": 12, "credits_charged": 50,
		}})
	}))
	defer lpbs.Close()

	resolver := &credentialclient.ConsumerSessionResolver{
		Credentials: &businessAccountCredentialFixture{}, LPBSBaseURL: lpbs.URL, HTTPClient: lpbs.Client(),
		Now: func() time.Time { return time.Date(2029, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
	manager := newSessionManagerWithFactory(ptyfake.NewFactory())
	created, err := manager.CreateWithOptions(context.Background(), "/bin/sh", 80, 24, backend.ID("standard"), (*policy.Policy)(nil), workDir, false)
	if err != nil {
		t.Fatalf("create live fixture session: %v", err)
	}
	defer func() { _ = manager.Delete(context.Background(), created.ID) }()
	store := sessionstore.NewInMemory()
	if err := store.Save(context.Background(), sessionstore.Metadata{
		ID: created.ID, AgentType: sessionstore.AgentOpenCode, BusinessAccountID: "acct-selected", Status: sessionstore.StatusLive,
	}); err != nil {
		t.Fatalf("save live account-bound session: %v", err)
	}
	token, err := activePaidCodingSessions.Bind(created.ID, "acct-selected")
	if err != nil {
		t.Fatalf("bind fixture session credential: %v", err)
	}
	proxy := &Server{
		subscriptionResolver: resolver, sessionStore: store, sessions: manager,
		paidCodingProtocolQualified: true,
	}
	webConsole := httptest.NewServer(http.HandlerFunc(proxy.paidCodingCompletionsHandler))
	defer webConsole.Close()
	inlineConfig, err := openCodeConfig(webConsole.URL+"/api/v1/paid-coding/v1", token)
	if err != nil {
		t.Fatalf("build runtime OpenCode config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, openCodePath, "run", "--pure", "--model", "vrooli/coding", "--auto", "--dir", workDir, "--format", "json", "Read fixture.txt and repeat its contents.")
	command.Dir = workDir
	command.Env = envWithOverrides(os.Environ(), map[string]string{
		"OPENCODE_CONFIG_CONTENT": inlineConfig,
		"XDG_CONFIG_HOME":         filepath.Join(workDir, "config"),
		"XDG_DATA_HOME":           filepath.Join(workDir, "data"),
		"XDG_STATE_HOME":          filepath.Join(workDir, "state"),
	})
	output, err := command.CombinedOutput()
	if err != nil {
		safeOutput := strings.ReplaceAll(string(output), token, "<session-credential>")
		t.Fatalf("OpenCode fixture run failed: %v; output: %s", err, safeOutput)
	}
	if !strings.Contains(string(output), "fixture-ok") {
		t.Fatalf("OpenCode did not return the fixture completion: %s", output)
	}
	mu.Lock()
	defer mu.Unlock()
	if requestCount < 2 || !toolCallSent {
		t.Fatalf("LPBS fixture requests=%d tools=%v, want a declared tool call and follow-up completion", requestCount, toolCounts)
	}
	for i := range accountIDs {
		if accountIDs[i] != "acct-selected" || roles[i] != paidCodingRole {
			t.Fatalf("LPBS request %d authority = account %q role %q", i+1, accountIDs[i], roles[i])
		}
	}
}

// TestOpenCodeCLIUsesAccountBoundProxyAgainstExternalLPBSFixture is invoked by
// LPBS's owner integration test while its real metered inference handler and
// SQLite ledger are running on an isolated HTTP fixture. The ordinary
// Web-Console-only OpenCode test above keeps its local fixture; this case proves
// the same runtime protocol crosses both owners' actual HTTP handlers.
func TestOpenCodeCLIUsesAccountBoundProxyAgainstExternalLPBSFixture(t *testing.T) {
	lpbsURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WEB_CONSOLE_EXTERNAL_LPBS_URL")), "/")
	if lpbsURL == "" {
		t.Skip("set WEB_CONSOLE_EXTERNAL_LPBS_URL from the LPBS cross-owner fixture")
	}
	openCodePath, err := exec.LookPath("opencode")
	if err != nil {
		t.Skip("OpenCode CLI is not installed")
	}
	activePaidCodingSessions = newPaidCodingSessionRegistry()
	t.Cleanup(func() { activePaidCodingSessions.Clear() })

	workDir := t.TempDir()
	manager := newSessionManagerWithFactory(ptyfake.NewFactory())
	created, err := manager.CreateWithOptions(context.Background(), "/bin/sh", 80, 24, backend.ID("standard"), (*policy.Policy)(nil), workDir, false)
	if err != nil {
		t.Fatalf("create live fixture session: %v", err)
	}
	defer func() { _ = manager.Delete(context.Background(), created.ID) }()
	store := sessionstore.NewInMemory()
	if err := store.Save(context.Background(), sessionstore.Metadata{
		ID: created.ID, AgentType: sessionstore.AgentOpenCode, BusinessAccountID: "acct-selected", Status: sessionstore.StatusLive,
	}); err != nil {
		t.Fatalf("save live account-bound session: %v", err)
	}
	token, err := activePaidCodingSessions.Bind(created.ID, "acct-selected")
	if err != nil {
		t.Fatalf("bind fixture session credential: %v", err)
	}
	resolver := &credentialclient.ConsumerSessionResolver{
		Credentials: &businessAccountCredentialFixture{}, LPBSBaseURL: lpbsURL,
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
	}
	proxy := &Server{subscriptionResolver: resolver, sessionStore: store, sessions: manager, paidCodingProtocolQualified: true}
	webConsole := httptest.NewServer(http.HandlerFunc(proxy.paidCodingCompletionsHandler))
	defer webConsole.Close()
	inlineConfig, err := openCodeConfig(webConsole.URL+"/api/v1/paid-coding/v1", token)
	if err != nil {
		t.Fatalf("build runtime OpenCode config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, openCodePath, "run", "--pure", "--model", "vrooli/coding", "--auto", "--dir", workDir, "--format", "json", "Reply with the exact phrase external-lpbs-fixture-ok and no other text.")
	command.Dir = workDir
	command.Env = envWithOverrides(os.Environ(), map[string]string{
		"OPENCODE_CONFIG_CONTENT": inlineConfig,
		"XDG_CONFIG_HOME":         filepath.Join(workDir, "config"),
		"XDG_DATA_HOME":           filepath.Join(workDir, "data"),
		"XDG_STATE_HOME":          filepath.Join(workDir, "state"),
	})
	output, err := command.CombinedOutput()
	if err != nil {
		safeOutput := strings.ReplaceAll(string(output), token, "<session-credential>")
		t.Fatalf("OpenCode cross-owner fixture run failed: %v; output: %s", err, safeOutput)
	}
	if !strings.Contains(string(output), "external-lpbs-fixture-ok") {
		t.Fatalf("OpenCode did not return the external LPBS fixture completion: %s", output)
	}
}

func envWithOverrides(environment []string, overrides map[string]string) []string {
	result := make([]string, 0, len(environment)+len(overrides))
	for _, entry := range environment {
		name, _, _ := strings.Cut(entry, "=")
		if _, replaced := overrides[name]; !replaced {
			result = append(result, entry)
		}
	}
	for name, value := range overrides {
		result = append(result, name+"="+value)
	}
	return result
}
