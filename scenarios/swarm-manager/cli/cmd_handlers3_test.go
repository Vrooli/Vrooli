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
		_, _ = w.Write([]byte(`{"summary":{"total_items":2,"items_by_status":{"open":2},"active_goals":1},"items":[],"goals":[]}`))
	}))
	app := newAppT(t)
	// Markdown rendering should not error and should produce some output.
	out := clitest.CaptureStdout(t, func() error { return app.cmdOverview([]string{}) })
	if strings.TrimSpace(out) == "" {
		t.Error("markdown overview produced no output")
	}
}
