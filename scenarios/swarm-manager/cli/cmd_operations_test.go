package main

import (
	"net/http"
	"testing"

	clitest "swarm-manager/cli/internal/testutil"
)

func TestCmdOperationsBriefJSONCallsBriefEndpoint(t *testing.T) {
	var method, path, query string
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		query = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"generated_at":"2026-05-16T00:00:00Z","window_seconds":10800,"summary":{"active_activity_count":1},"active_work":[],"needs_attention":[],"recent_completions":[],"director_handoffs":[],"recommended_next_actions":[],"drill_down_commands":[],"warnings":[]}`))
	}))
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	out := clitest.CaptureStdout(t, func() error {
		return app.cmdOperationsBrief([]string{"--window", "PT1H", "--json"})
	})

	if method != http.MethodGet || path != "/api/v1/operations/brief" || query != "window=PT1H" {
		t.Fatalf("request = %s %s?%s, want GET /api/v1/operations/brief?window=PT1H", method, path, query)
	}
	if !containsAll(out, `"active_activity_count"`, "1") {
		t.Fatalf("json output = %s", out)
	}
}

func TestCmdOperationsListEncodesFiltersAndPrintsHumanSummary(t *testing.T) {
	var query string
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"generated_at":"2026-05-16T00:00:00Z","window_seconds":10800,"lanes":[{"lane":"execute","active":1,"capacity":3,"queue":0}],"queue":{"depth":2,"max_depth":50},"activities":[{"activity_id":"act_1","run_id":"run_1","owner_type":"backlog","owner_kind":"feature","owner_name":"ship","lane":"execute","status":"running","purpose":"process","requested_at":"2026-05-16T00:00:00Z"}],"recently_finished":[]}`))
	}))
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	out := clitest.CaptureStdout(t, func() error {
		return app.cmdOperationsList([]string{"--lane", "execute", "--status", "running", "--owner-type", "backlog", "--q", "ship"})
	})

	if !containsAll(query, "lane=execute", "status=running", "owner_type=backlog", "q=ship") {
		t.Fatalf("query = %s", query)
	}
	if !containsAll(out, "Summary:", "Active Work:", "ship", "run_1") {
		t.Fatalf("human output = %s", out)
	}
}
