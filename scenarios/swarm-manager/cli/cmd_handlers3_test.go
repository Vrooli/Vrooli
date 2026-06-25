package main

import (
	"net/http"
	"strings"
	"testing"

	clitest "swarm-manager/cli/internal/testutil"
)

func TestCmdOverview_JSONShortcut(t *testing.T) {
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/overview" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"summary":{"total_items":3}}`))
	}))
	app := newAppT(t)
	out := clitest.CaptureStdout(t, func() error { return app.cmdOverview([]string{"--format", "json"}) })
	if !strings.Contains(out, `"total_items": 3`) {
		t.Errorf("json output = %q", out)
	}
}

func TestCmdOverview_Markdown(t *testing.T) {
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"summary":{"total_items":2,"items_by_status":{"open":2},"active_initiatives":1},"items":[],"initiatives":[]}`))
	}))
	app := newAppT(t)
	// Markdown rendering should not error and should produce some output.
	out := clitest.CaptureStdout(t, func() error { return app.cmdOverview([]string{}) })
	if strings.TrimSpace(out) == "" {
		t.Error("markdown overview produced no output")
	}
}

func TestCmdInitiativesDelete_RequiresNameAndDeletes(t *testing.T) {
	app := newAppT(t)
	if err := app.cmdInitiativesDelete([]string{}); err == nil {
		t.Fatal("expected --name required")
	}

	var method, path string
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{}`))
	}))
	app2 := newAppT(t)
	out := clitest.CaptureStdout(t, func() error { return app2.cmdInitiativesDelete([]string{"--name", "obs"}) })
	if method != http.MethodDelete || path != "/api/v1/initiatives/obs" {
		t.Errorf("request = %s %s", method, path)
	}
	if !strings.Contains(out, `Initiative "obs" deleted`) {
		t.Errorf("output = %q", out)
	}
}

func TestCmdInitiativesAddItems_PostsItems(t *testing.T) {
	var path string
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"initiative":{"name":"obs","items":["fix/a","fix/b"]}}`))
	}))
	app := newAppT(t)
	out := clitest.CaptureStdout(t, func() error {
		return app.cmdInitiativesAddItems([]string{"--name", "obs", "--items", "fix/a, fix/b"})
	})
	if path != "/api/v1/initiatives/obs/items" {
		t.Errorf("path = %s", path)
	}
	if !strings.Contains(out, "Total items: 2") {
		t.Errorf("output = %q", out)
	}
}

func TestCmdInitiativesAddItems_Validation(t *testing.T) {
	app := newAppT(t)
	if err := app.cmdInitiativesAddItems([]string{"--name", "obs"}); err == nil {
		t.Error("expected error for missing --items")
	}
	// items present but only whitespace/commas -> zero refs.
	if err := app.cmdInitiativesAddItems([]string{"--name", "obs", "--items", " , , "}); err == nil {
		t.Error("expected error for empty item references")
	}
}

func TestCmdInitiativesRemoveItems_DeletesItems(t *testing.T) {
	var method, path string
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{"initiative":{"name":"obs","items":["fix/a"]}}`))
	}))
	app := newAppT(t)
	out := clitest.CaptureStdout(t, func() error {
		return app.cmdInitiativesRemoveItems([]string{"--name", "obs", "--items", "fix/b"})
	})
	if method != http.MethodDelete || path != "/api/v1/initiatives/obs/items" {
		t.Errorf("request = %s %s", method, path)
	}
	if !strings.Contains(out, "Remaining items: 1") {
		t.Errorf("output = %q", out)
	}
}
