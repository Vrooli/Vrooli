package main

import (
	"net/http"
	"strings"
	"testing"

	clitest "github.com/vrooli/cli-core/cliapptest"
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

func TestCmdBacklogReconcileCounts_LabelsRecordAndEventSurfaces(t *testing.T) {
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/overview":
			_, _ = w.Write([]byte(`{"summary":{"total_items":4}}`))
		case "/api/v1/backlog/export":
			_, _ = w.Write([]byte("---\npre_filter_total: 4\nexcluded:\n  - filter: \"archived\"\n    rule: \"exclude archived records\"\n    count: 1\n  - filter: \"status\"\n    rule: \"include selected statuses\"\n    count: 1\nitems_count: 2\n---\n"))
		case "/api/v1/stats":
			_, _ = w.Write([]byte(`{"event_count":9,"throughput":{"created_last_7_days":3,"completed_last_7_days":1},"dashboard":{"total_backlog_size":7,"total_completed_all_time":2}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	app := newAppT(t)
	out := clitest.CaptureStdout(t, func() error { return app.cmdBacklogReconcileCounts([]string{"--json"}) })
	for _, want := range []string{`"overview_total": 4`, `"totals_agree": true`, `"arithmetic_closes": true`, `"event_count": 9`, "append-only lifecycle events"} {
		if !strings.Contains(out, want) {
			t.Errorf("reconciliation output missing %q: %s", want, out)
		}
	}
}
