package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	clitest "swarm-manager/cli/internal/testutil"
)

func TestCmdPortfolioBrief_RendersBrief(t *testing.T) {
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/portfolio/brief" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"brief":{"title":"Today","ref":"r1","selected_at":"now","summary":"line one\n\nline two"}}`))
	}))
	app := newAppT(t)
	out := clitest.CaptureStdout(t, func() error { return app.cmdPortfolioBrief([]string{}) })
	if !strings.Contains(out, "Today") || !strings.Contains(out, "Ref: r1") {
		t.Errorf("brief header missing: %q", out)
	}
	if !strings.Contains(out, "line one") || !strings.Contains(out, "line two") {
		t.Errorf("summary lines missing: %q", out)
	}
}

func TestCmdInitiativesCandidates_PassesPurpose(t *testing.T) {
	var gotQuery url.Values
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/initiative-candidates" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte(`{"brief":{"title":"Candidate"}}`))
	}))
	app := newAppT(t)
	out := clitest.CaptureStdout(t, func() error {
		return app.cmdInitiativesCandidates([]string{"--purpose", " readiness "})
	})
	if gotQuery.Get("purpose") != "readiness" {
		t.Errorf("purpose query = %q", gotQuery.Get("purpose"))
	}
	if !strings.Contains(out, "Candidate") {
		t.Errorf("output = %q", out)
	}
}

func TestRunExecutionMutation_StartPostsAndRenders(t *testing.T) {
	var method, path string
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{"execution":{"execution_id":"e1","status":"started","backlog_kind":"fix","backlog_name":"a"}}`))
	}))
	app := newAppT(t)
	out := clitest.CaptureStdout(t, func() error { return app.cmdExecutionStart([]string{"--id", " e1 "}) })
	if method != http.MethodPost || path != "/api/v1/execution/e1/start" {
		t.Errorf("request = %s %s", method, path)
	}
	if !strings.Contains(out, "Execution start: e1") || !strings.Contains(out, "Status: started") {
		t.Errorf("output = %q", out)
	}
}

func TestRunExecutionMutation_RequiresID(t *testing.T) {
	app := newAppT(t)
	if err := app.cmdExecutionCancel([]string{}); err == nil || !strings.Contains(err.Error(), "--id is required") {
		t.Fatalf("expected --id required, got %v", err)
	}
}

func TestCmdExecutionRetry_PostsNoteAndRenders(t *testing.T) {
	var path string
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"execution":{"execution_id":"e2","status":"queued","backlog_kind":"fix","backlog_name":"b"}}`))
	}))
	app := newAppT(t)
	out := clitest.CaptureStdout(t, func() error {
		return app.cmdExecutionRetry([]string{"--id", "e1", "--note", "retrying"})
	})
	if path != "/api/v1/execution/e1/retry" {
		t.Errorf("path = %s", path)
	}
	if !strings.Contains(out, "New attempt: e2 (queued)") {
		t.Errorf("output = %q", out)
	}
}

func TestCmdCircuitBreakerReset_RequiresItem(t *testing.T) {
	app := newAppT(t)
	if err := app.cmdCircuitBreakerReset([]string{}); err == nil || !strings.Contains(err.Error(), "--item is required") {
		t.Fatalf("expected --item required, got %v", err)
	}
}

func TestCmdCircuitBreakerReset_Posts(t *testing.T) {
	var path string
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{}`))
	}))
	app := newAppT(t)
	out := clitest.CaptureStdout(t, func() error {
		return app.cmdCircuitBreakerReset([]string{"--item", "fix/alpha"})
	})
	if path != "/api/v1/execution/circuit-breaker/reset" {
		t.Errorf("path = %s", path)
	}
	if !strings.Contains(out, "Circuit breaker reset for fix/alpha") {
		t.Errorf("output = %q", out)
	}
}

func TestCmdCapturesDelete_RequiresIDAndDeletes(t *testing.T) {
	app := newAppT(t)
	if err := app.cmdCapturesDelete([]string{}); err == nil {
		t.Fatal("expected --id required")
	}

	var method, path string
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{}`))
	}))
	app2 := newAppT(t)
	out := clitest.CaptureStdout(t, func() error { return app2.cmdCapturesDelete([]string{"--id", "c1"}) })
	if method != http.MethodDelete || path != "/api/v1/captures/c1" {
		t.Errorf("request = %s %s", method, path)
	}
	if !strings.Contains(out, "Deleted capture c1") {
		t.Errorf("output = %q", out)
	}
}

func TestCmdCapturesClassify_PostsAndRenders(t *testing.T) {
	var path string
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"task_id":"t1","run_id":"r1"}`))
	}))
	app := newAppT(t)
	out := clitest.CaptureStdout(t, func() error { return app.cmdCapturesClassify([]string{"--id", "c9"}) })
	if path != "/api/v1/captures/c9/classify" {
		t.Errorf("path = %s", path)
	}
	if !strings.Contains(out, "Run ID: r1") || !strings.Contains(out, "Task ID: t1") {
		t.Errorf("output = %q", out)
	}
}
