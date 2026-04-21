package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCmdAISearchSearch_RejectsEmptyArgs(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	run := app.cmdAISearchSearch("")
	if err := run([]string{}); err == nil {
		t.Fatal("expected usage error for empty args")
	}
}

func TestCmdAISearchSearch_SendsExpectedPayload(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/search/ai" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		_, _ = w.Write([]byte(`{"results":[{"entity":"backlog","id":"alpha","score":0.9,"scorePercent":90,"payload":{"title":"Alpha","status":"ready"}}],"total":1,"query":"retry","entity":"backlog","fallback":"none","latencyMs":12}`))
	}))
	defer server.Close()

	t.Setenv("SWARM_MANAGER_API_BASE", server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	run := app.cmdAISearchSearch("backlog")
	if err := run([]string{"retry", "semantics", "--limit", "10", "--status", "ready,queued", "--kind", "idea,execute", "--initiative", "obs-core", "--include-archived"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if got["query"] != "retry semantics" {
		t.Errorf("query: %v", got["query"])
	}
	if got["entity"] != "backlog" {
		t.Errorf("expected entity override backlog, got %v", got["entity"])
	}
	if got["limit"].(float64) != 10 {
		t.Errorf("limit: %v", got["limit"])
	}
	filters, ok := got["filters"].(map[string]any)
	if !ok {
		t.Fatalf("expected filters map, got %T", got["filters"])
	}
	if filters["include_archived"] != true {
		t.Errorf("include_archived: %v", filters["include_archived"])
	}
	if filters["initiative"] != "obs-core" {
		t.Errorf("initiative: %v", filters["initiative"])
	}
	status := filters["status"].([]any)
	if len(status) != 2 || status[0] != "ready" {
		t.Errorf("status csv: %v", status)
	}
	kind := filters["kind"].([]any)
	if len(kind) != 2 {
		t.Errorf("kind csv: %v", kind)
	}
}

func TestCmdAISearchStatus_RendersOperational(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"available":true,"ollama":true,"qdrant":true,"indexedBacklog":5,"indexedInitiatives":2,"onDiskBacklog":5,"onDiskInitiatives":2}`))
	}))
	defer server.Close()
	t.Setenv("SWARM_MANAGER_API_BASE", server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	if err := app.cmdAISearchStatus([]string{}); err != nil {
		t.Fatalf("status: %v", err)
	}
}

func TestCmdAISearchReindex_StartsJob(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/search/ai/reindex" && r.Method == http.MethodPost {
			called = true
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"running":true,"message":"Reindex started","indexed":0,"total":0}`))
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()
	t.Setenv("SWARM_MANAGER_API_BASE", server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	if err := app.cmdAISearchReindex([]string{}); err != nil {
		t.Fatalf("reindex: %v", err)
	}
	if !called {
		t.Error("expected reindex endpoint called")
	}
}

func TestCmdAISearchReindexStatus_RendersProgress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/search/ai/reindex/status" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"running":false,"indexed":5,"total":5,"finishedAt":"2026-04-20T00:00:00Z"}`))
	}))
	defer server.Close()
	t.Setenv("SWARM_MANAGER_API_BASE", server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	if err := app.cmdAISearchReindexStatus([]string{}); err != nil {
		t.Fatalf("status: %v", err)
	}
}

func TestCmdAISearchReindex_WaitPollsUntilComplete(t *testing.T) {
	var statusCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/search/ai/reindex":
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"running":true,"total":3}`))
		case "/api/v1/search/ai/reindex/status":
			statusCalls++
			if statusCalls < 2 {
				_, _ = w.Write([]byte(`{"running":true,"indexed":1,"total":3}`))
				return
			}
			_, _ = w.Write([]byte(`{"running":false,"indexed":3,"total":3}`))
		default:
			t.Fatalf("unexpected: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("SWARM_MANAGER_API_BASE", server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	if err := app.cmdAISearchReindex([]string{"--wait"}); err != nil {
		t.Fatalf("reindex --wait: %v", err)
	}
	if statusCalls < 2 {
		t.Errorf("expected at least 2 poll calls, got %d", statusCalls)
	}
}

func TestCmdAISearchSearch_FallbackRendersHint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[],"total":0,"query":"x","entity":"both","fallback":"unavailable","latencyMs":1}`))
	}))
	defer server.Close()
	t.Setenv("SWARM_MANAGER_API_BASE", server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	run := app.cmdAISearchSearch("")
	if err := run([]string{"x"}); err != nil {
		t.Fatalf("search: %v", err)
	}
}

func TestAISearchCommandsRegistered(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	// Every run attempt without API config fails; we just assert the commands
	// are routed without a "command not found" error.
	cases := [][]string{
		{"ai-search", "status"},
		{"ai-search", "reindex-status"},
		{"ai-search", "query", "hello"},
		{"backlog", "search-ai", "retry"},
		{"initiatives", "search-ai", "obs"},
	}
	for _, args := range cases {
		err := app.Run(args)
		// Expected error mentions API/HTTP — never "unknown command".
		if err != nil && strings.Contains(strings.ToLower(err.Error()), "unknown command") {
			t.Errorf("%v → command not registered: %v", args, err)
		}
	}
}
