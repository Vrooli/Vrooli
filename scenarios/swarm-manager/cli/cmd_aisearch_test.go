package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	clitest "github.com/vrooli/cli-core/cliapptest"
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
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	run := app.cmdAISearchSearch("backlog")
	if err := run([]string{"retry", "semantics", "--limit", "10", "--status", "ready,queued", "--kind", "idea,execute", "--goal", "obs-core", "--include-archived"}); err != nil {
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
	if filters["goal"] != "obs-core" {
		t.Errorf("goal: %v", filters["goal"])
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
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"available":true,"ollama":true,"qdrant":true,"indexedBacklog":5,"indexedGoals":2,"onDiskBacklog":5,"onDiskGoals":2}`))
	}))
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	if err := app.cmdAISearchStatus([]string{}); err != nil {
		t.Fatalf("status: %v", err)
	}
}

func TestCmdAISearchReconcile_StartsJob(t *testing.T) {
	called := false
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/search/ai/reconcile" && r.Method == http.MethodPost {
			called = true
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"running":true,"startedAt":"2026-04-20T00:00:00Z"}`))
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	if err := app.cmdAISearchReconcile([]string{}); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if !called {
		t.Error("expected reconcile endpoint called")
	}
}

func TestCmdAISearchReconcileStatus_RendersProgress(t *testing.T) {
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/search/ai/reconcile/status" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"running":false,"finishedAt":"2026-04-20T00:00:00Z","lastResult":{"upsertedBacklog":5,"upsertedGoal":0,"deletedBacklog":2,"deletedGoal":0}}`))
	}))
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	if err := app.cmdAISearchReconcileStatus([]string{}); err != nil {
		t.Fatalf("status: %v", err)
	}
}

func TestCmdAISearchReconcile_WaitPollsUntilComplete(t *testing.T) {
	var statusCalls int
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/search/ai/reconcile":
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"running":true,"startedAt":"2026-04-20T00:00:00Z"}`))
		case "/api/v1/search/ai/reconcile/status":
			statusCalls++
			if statusCalls < 2 {
				_, _ = w.Write([]byte(`{"running":true,"startedAt":"2026-04-20T00:00:00Z"}`))
				return
			}
			_, _ = w.Write([]byte(`{"running":false,"finishedAt":"2026-04-20T00:00:01Z","lastResult":{"upsertedBacklog":3}}`))
		default:
			t.Fatalf("unexpected: %s", r.URL.Path)
		}
	}))
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	if err := app.cmdAISearchReconcile([]string{"--wait"}); err != nil {
		t.Fatalf("reconcile --wait: %v", err)
	}
	if statusCalls < 2 {
		t.Errorf("expected at least 2 poll calls, got %d", statusCalls)
	}
}

func TestCmdAISearchReconcile_DryRun_RendersDriftReport(t *testing.T) {
	// In production, the cli-core preflight calls HTTPClient.SetDryRun(true)
	// when --dry-run is on the command line, so every outgoing request carries
	// X-Dry-Run. Tests bypass preflight and set both signals manually.
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/search/ai/reconcile" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Dry-Run") != "true" {
			t.Fatalf("expected X-Dry-Run=true header, got %q", r.Header.Get("X-Dry-Run"))
		}
		_, _ = w.Write([]byte(`{"dry_run":true,"plan":{"plannedAt":"2026-04-20T00:00:00Z","toDeleteBacklog":["g1","g2"],"unchangedBacklog":50,"legacyBacklog":3}}`))
	}))
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	app.globalDry = true
	app.core.HTTPClient.SetDryRun(true)
	if err := app.cmdAISearchReconcile([]string{}); err != nil {
		t.Fatalf("reconcile dry-run: %v", err)
	}
}

func TestCmdAISearchSearch_FallbackRendersHint(t *testing.T) {
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[],"total":0,"query":"x","entity":"both","fallback":"unavailable","latencyMs":1}`))
	}))
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
		{"search", "status"},
		{"search", "reindex-status"},
		{"search", "query", "hello"},
	}
	for _, args := range cases {
		err := app.Run(args)
		// Expected error mentions API/HTTP — never "unknown command".
		if err != nil && strings.Contains(strings.ToLower(err.Error()), "unknown command") {
			t.Errorf("%v → command not registered: %v", args, err)
		}
	}
}
