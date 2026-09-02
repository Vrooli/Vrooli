package main

import (
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

func TestCmdSessionsCreateAndAttachUseScriptableContracts(t *testing.T) {
	var requests []struct {
		method string
		path   string
		body   map[string]any
	}
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entry := struct {
			method string
			path   string
			body   map[string]any
		}{method: r.Method, path: r.URL.Path, body: map[string]any{}}
		_ = json.NewDecoder(r.Body).Decode(&entry.body)
		requests = append(requests, entry)
		if strings.HasSuffix(r.URL.Path, "/context") {
			_, _ = w.Write([]byte(`{"session":{"id":"sess_1","title":"swarm operations session","kind":"swarm_operations","status":"draft","created_at":"2026-08-29T00:00:00Z","updated_at":"2026-08-29T00:00:00Z","staged_context_refs":[{"type":"backlog_item","ref":"chore/one"},{"type":"goal","ref":"delivery"}]}}`))
			return
		}
		_, _ = w.Write([]byte(`{"session":{"id":"sess_1","title":"swarm operations session","kind":"swarm_operations","status":"draft","created_at":"2026-08-29T00:00:00Z","updated_at":"2026-08-29T00:00:00Z"}}`))
	}))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	if err := app.cmdSessionsCreate([]string{"--kind", "swarm_operations", "--starter-job", "operations-triage-staleness"}); err != nil {
		t.Fatal(err)
	}
	if err := app.cmdSessionsAttach([]string{"--id", "sess_1", "--entity", "backlog_item/chore/one", "--entity", "goal/delivery"}); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 2 {
		t.Fatalf("requests = %+v", requests)
	}
	if requests[0].method != http.MethodPost || requests[0].path != "/api/v1/agent-sessions" || requests[0].body["starter_job_id"] != "operations-triage-staleness" {
		t.Fatalf("create request = %+v", requests[0])
	}
	refs, ok := requests[1].body["context_refs"].([]any)
	if requests[1].path != "/api/v1/agent-sessions/sess_1/context" || !ok || len(refs) != 2 {
		t.Fatalf("attach request = %+v", requests[1])
	}
}

func TestCmdSessionsCreateCanDeclareProposalTarget(t *testing.T) {
	var requestBody map[string]any
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/proposal-sessions" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"id":"sess_target","kind":"swarm_operations","status":"draft","proposal_target":{"type":"backlog_item","ref":"research/item","name":"Research item"}}`))
	}))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	if err := app.cmdSessionsCreate([]string{"--kind", "swarm_operations", "--starter-job", "proposal-reconcile-item", "--target", "backlog_item/research/item", "--target-name", "Research item"}); err != nil {
		t.Fatal(err)
	}
	target, ok := requestBody["target"].(map[string]any)
	if !ok || target["type"] != "backlog_item" || target["ref"] != "research/item" || target["name"] != "Research item" {
		t.Fatalf("target payload = %+v", requestBody["target"])
	}
}

func TestCmdSessionsLifecycleAndProposalCommandsUseOwnedEndpoints(t *testing.T) {
	var paths []string
	var applyBody map[string]any
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.RequestURI())
		if strings.HasSuffix(r.URL.Path, "/apply") {
			_ = json.NewDecoder(r.Body).Decode(&applyBody)
		}
		if strings.HasSuffix(r.URL.Path, "/events") {
			_, _ = w.Write([]byte(`{"events":[],"has_more":false,"next_after_sequence":0}`))
			return
		}
		_, _ = w.Write([]byte(`{"session":{"id":"sess_1","title":"Session","kind":"swarm_operations","status":"running","created_at":"2026-08-29T00:00:00Z","updated_at":"2026-08-29T00:00:00Z"}}`))
	}))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	commands := []func() error{
		func() error { return app.cmdSessionsStart([]string{"--id", "sess_1", "--message", "Begin"}) },
		func() error { return app.cmdSessionsContinue([]string{"--id", "sess_1", "--message", "Continue"}) },
		func() error {
			return app.cmdSessionsEvents([]string{"--id", "sess_1", "--after-sequence", "4", "--limit", "20"})
		},
		func() error {
			return app.cmdSessionsProposalApply([]string{"--id", "sess_1", "--proposal", "prop_1", "--accept", "mutation_1"})
		},
		func() error {
			return app.cmdSessionsProposalRevise([]string{"--id", "sess_1", "--proposal", "prop_1", "--note", "More evidence"})
		},
		func() error {
			return app.cmdSessionsProposalAcceptKeep([]string{"--id", "sess_1", "--proposal", "prop_1"})
		},
	}
	for _, command := range commands {
		if err := command(); err != nil {
			t.Fatal(err)
		}
	}
	wantPaths := []string{
		"POST /api/v1/agent-sessions/sess_1/start",
		"POST /api/v1/agent-sessions/sess_1/continue",
		"GET /api/v1/agent-sessions/sess_1/events?after_sequence=4&limit=20",
		"POST /api/v1/agent-sessions/sess_1/proposals/prop_1/apply",
		"POST /api/v1/agent-sessions/sess_1/proposals/prop_1/revise",
		"POST /api/v1/agent-sessions/sess_1/proposals/prop_1/accept-keep",
	}
	if strings.Join(paths, "\n") != strings.Join(wantPaths, "\n") {
		t.Fatalf("paths = %v, want %v", paths, wantPaths)
	}
	accepted, ok := applyBody["accepted_mutation_ids"].([]any)
	if !ok || len(accepted) != 1 || accepted[0] != "mutation_1" {
		t.Fatalf("apply body = %+v", applyBody)
	}
}

func TestEveryAgentSessionEndpointHasCLICommandOrRecordedReason(t *testing.T) {
	capability := map[string]string{
		"GET /api/v1/agent-sessions":                                                   "list",
		"POST /api/v1/agent-sessions":                                                  "create",
		"POST /api/v1/agent-sessions/batch":                                            "create-batch",
		"GET /api/v1/agent-sessions/{session_id}":                                      "get",
		"PATCH /api/v1/agent-sessions/{session_id}/kind":                               "reason:UI kind changes carry local composer state; the web client owns that interaction",
		"POST /api/v1/agent-sessions/{session_id}/context":                             "attach",
		"DELETE /api/v1/agent-sessions/{session_id}":                                   "delete",
		"GET /api/v1/agent-sessions/{session_id}/startup-brief":                        "startup-brief",
		"POST /api/v1/agent-sessions/{session_id}/startup-brief":                       "startup-brief",
		"POST /api/v1/agent-sessions/{session_id}/attachments":                         "reason:binary multipart upload remains web-owned until the CLI has a governed file-stream primitive",
		"GET /api/v1/agent-sessions/{session_id}/attachments/{attachment_id}":          "reason:binary attachment retrieval is exposed through the web transcript, not structured session automation",
		"POST /api/v1/agent-sessions/{session_id}/start":                               "start",
		"POST /api/v1/agent-sessions/{session_id}/continue":                            "continue",
		"POST /api/v1/agent-sessions/{session_id}/complete":                            "complete",
		"POST /api/v1/agent-sessions/reap":                                            "reap",
		"POST /api/v1/agent-sessions/{session_id}/disposition":                         "disposition",
		"POST /api/v1/agent-sessions/{session_id}/prompt-preview":                      "prompt-preview",
		"GET /api/v1/agent-sessions/{session_id}/events":                               "events",
		"POST /api/v1/agent-sessions/{session_id}/refresh":                             "get",
		"POST /api/v1/agent-sessions/{session_id}/cancel":                              "reason:cancel remains an emergency lifecycle action available through Agent Manager run-stop",
		"POST /api/v1/agent-sessions/{session_id}/proposals/{proposal_id}/apply":       "proposal-apply",
		"POST /api/v1/agent-sessions/{session_id}/proposals/{proposal_id}/accept-keep": "proposal-accept-keep",
		"POST /api/v1/agent-sessions/{session_id}/proposals/{proposal_id}/revise":      "proposal-revise",
		"POST /api/v1/agent-sessions/{session_id}/proposals/{proposal_id}/wait":         "proposal-wait",
		"POST /api/v1/proposal-sessions":                                               "create",
		"GET /api/v1/proposal-sessions":                                                "reason:the existing proposals commands own proposal-session listing",
		"GET /api/v1/agent-sessions/{session_id}/artifacts":                            "reason:proposal detail commands render session artifacts with their decisions",
		"GET /api/v1/artifacts/by-entity":                                              "reason:entity-oriented proposal retrieval owns this cross-session index",
	}
	handlerSource, err := os.ReadFile("../api/internal/agentsessions/handler.go")
	if err != nil {
		t.Fatal(err)
	}
	registerSource, err := os.ReadFile("domains/sessions/register.go")
	if err != nil {
		t.Fatal(err)
	}
	routePattern := regexp.MustCompile(`r\.HandleFunc\("([^"]+)"[^\n]+Methods\(([^\n]+)\)`)
	methodPattern := regexp.MustCompile(`http\.Method([A-Za-z]+)`)
	seen := map[string]bool{}
	for _, match := range routePattern.FindAllStringSubmatch(string(handlerSource), -1) {
		for _, method := range methodPattern.FindAllStringSubmatch(match[2], -1) {
			key := strings.ToUpper(method[1]) + " " + match[1]
			owner, ok := capability[key]
			if !ok {
				t.Errorf("endpoint %s has no CLI counterpart or recorded reason", key)
				continue
			}
			seen[key] = true
			if strings.HasPrefix(owner, "reason:") {
				if len(strings.TrimPrefix(owner, "reason:")) < 20 {
					t.Errorf("endpoint %s reason is not durable enough: %q", key, owner)
				}
				continue
			}
			if !strings.Contains(string(registerSource), `APICommand("`+owner+`"`) {
				t.Errorf("endpoint %s claims missing sessions command %q", key, owner)
			}
		}
	}
	for endpoint := range capability {
		if !seen[endpoint] {
			t.Errorf("stale endpoint capability entry %s", endpoint)
		}
	}
}

func TestCmdSessionsDeleteRequiresConfirmation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	if err := app.cmdSessionsDelete([]string{"--id", "sess_1"}); err == nil {
		t.Fatal("cmdSessionsDelete without --yes error = nil, want confirmation error")
	}
}

func TestCmdSessionsDeleteCallsDeleteEndpoint(t *testing.T) {
	var method, path string
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"session_id":"sess_1"}`))
	}))
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	out := clitest.CaptureStdout(t, func() error {
		return app.cmdSessionsDelete([]string{"--id", "sess_1", "--yes"})
	})

	if method != http.MethodDelete || path != "/api/v1/agent-sessions/sess_1" {
		t.Fatalf("request = %s %s, want DELETE /api/v1/agent-sessions/sess_1", method, path)
	}
	if !containsAll(out, "Session sess_1 deleted.", "were preserved") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCmdSessionsGetRefreshCallsRefreshEndpoint(t *testing.T) {
	var method, path string
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"session":{"id":"sess_1","title":"Plan","kind":"meta_orchestration","status":"failed","run_id":"run-1","created_at":"2026-05-06T00:00:00Z","updated_at":"2026-05-06T00:01:00Z"}}`))
	}))
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	out := clitest.CaptureStdout(t, func() error {
		return app.cmdSessionsGet([]string{"--id", "sess_1", "--refresh"})
	})

	if method != http.MethodPost || path != "/api/v1/agent-sessions/sess_1/refresh" {
		t.Fatalf("request = %s %s, want POST /api/v1/agent-sessions/sess_1/refresh", method, path)
	}
	if !containsAll(out, "Plan (failed)", "Run ID: run-1") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestSessionsCommandsRegistered(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	for _, args := range [][]string{
		{"sessions", "list"},
		{"sessions", "get"},
		{"sessions", "delete"},
	} {
		_ = app.Run(args)
	}
}

func containsAll(value string, needles ...string) bool {
	for _, needle := range needles {
		if !strings.Contains(value, needle) {
			return false
		}
	}
	return true
}
