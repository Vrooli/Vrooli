package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

func newAppT(t *testing.T) *App {
	t.Helper()
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	return app
}

func TestCmdExecutionList_EncodesFiltersAndRendersHuman(t *testing.T) {
	var gotQuery url.Values
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/execution" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte(`{"items":[{"execution_id":"e1","backlog_kind":"fix","backlog_name":"alpha","status":"running","mode":"manual","run_id":"r1","task_id":"t1","failure_reason":"boom"}]}`))
	}))

	app := newAppT(t)
	out := clitest.CaptureStdout(t, func() error {
		return app.cmdExecutionList([]string{
			"--status", " running ", "--mode", "manual", "--backlog-kind", "fix",
			"--backlog-name", "alpha", "--started-by", "me",
			"--created-from", "2020-01-01T00:00:00Z", "--created-to", "2021-01-01T00:00:00Z",
		})
	})

	// Filters are trimmed and snake_cased.
	if gotQuery.Get("status") != "running" {
		t.Errorf("status query = %q", gotQuery.Get("status"))
	}
	if gotQuery.Get("backlog_kind") != "fix" || gotQuery.Get("backlog_name") != "alpha" {
		t.Errorf("backlog query = %v", gotQuery)
	}
	if gotQuery.Get("started_by") != "me" || gotQuery.Get("created_from") == "" || gotQuery.Get("created_to") == "" {
		t.Errorf("extended query = %v", gotQuery)
	}
	if !strings.Contains(out, "Found 1 execution run(s)") || !strings.Contains(out, "e1 (running)") {
		t.Errorf("human output missing summary: %q", out)
	}
	if !strings.Contains(out, "Failure: boom") {
		t.Errorf("failure reason not rendered: %q", out)
	}
}

func TestCmdExecutionList_EmptyResults(t *testing.T) {
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	app := newAppT(t)
	out := clitest.CaptureStdout(t, func() error {
		return app.cmdExecutionList([]string{})
	})
	if !strings.Contains(out, "No execution runs found.") {
		t.Errorf("empty output = %q", out)
	}
}

func TestCmdExecutionList_JSONPassthrough(t *testing.T) {
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"items":[{"execution_id":"e9","status":"done"}]}`))
	}))
	app := newAppT(t)
	out := clitest.CaptureStdout(t, func() error {
		return app.cmdExecutionList([]string{"--json"})
	})
	if !strings.Contains(out, `"e9"`) {
		t.Errorf("json output = %q", out)
	}
}

func TestCmdExecutionGet_RequiresID(t *testing.T) {
	app := newAppT(t)
	err := app.cmdExecutionGet([]string{})
	if err == nil || !strings.Contains(err.Error(), "--id is required") {
		t.Fatalf("expected --id required error, got %v", err)
	}
}

func TestCmdExecutionGet_RendersDetail(t *testing.T) {
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/execution/e42" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"execution":{"execution_id":"e42","status":"failed","backlog_kind":"fix","backlog_name":"beta","mode":"yolo","run_id":"r2","failure_reason":"timeout"}}`))
	}))
	app := newAppT(t)
	out := clitest.CaptureStdout(t, func() error {
		return app.cmdExecutionGet([]string{"--id", " e42 "})
	})
	if !strings.Contains(out, "Execution e42 (failed)") {
		t.Errorf("detail header missing: %q", out)
	}
	if !strings.Contains(out, "Backlog: fix/beta") || !strings.Contains(out, "Failure: timeout") {
		t.Errorf("detail body missing: %q", out)
	}
}
