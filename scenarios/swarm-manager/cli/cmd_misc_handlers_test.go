package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	clitest "swarm-manager/cli/internal/testutil"
)

func TestCmdCapturesList_RendersAndTruncates(t *testing.T) {
	longText := strings.Repeat("x", 80)
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/captures" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"captures":[{"id":"c1","text":"` + longText + `","status":"new","created":"2024-01-01"}]}`))
	}))
	app := newAppT(t)
	out := clitest.CaptureStdout(t, func() error { return app.cmdCapturesList([]string{}) })
	if !strings.Contains(out, "Found 1 capture(s)") {
		t.Errorf("summary missing: %q", out)
	}
	// text truncated to 60 chars + ellipsis.
	if !strings.Contains(out, strings.Repeat("x", 60)+"...") {
		t.Errorf("truncation missing: %q", out)
	}
}

func TestCmdCapturesList_Empty(t *testing.T) {
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"captures":[]}`))
	}))
	app := newAppT(t)
	out := clitest.CaptureStdout(t, func() error { return app.cmdCapturesList([]string{}) })
	if !strings.Contains(out, "No captures found.") {
		t.Errorf("empty output = %q", out)
	}
}

func TestCmdCapturesGet_RequiresID(t *testing.T) {
	app := newAppT(t)
	if err := app.cmdCapturesGet([]string{}); err == nil || !strings.Contains(err.Error(), "--id is required") {
		t.Fatalf("expected --id required, got %v", err)
	}
}

func TestCmdOperatingModeList_RendersDefaultMark(t *testing.T) {
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/operating-modes" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"modes":[{"mode":"holistic-loop","label":"Holistic","description":"desc","usage_count":3,"scope_kind":"initiative","run_strategy":"sequential_handoff","default":true}]}`))
	}))
	app := newAppT(t)
	out := clitest.CaptureStdout(t, func() error { return app.cmdOperatingModeList([]string{}) })
	if !strings.Contains(out, "holistic-loop [default] — Holistic") {
		t.Errorf("default mark missing: %q", out)
	}
	if !strings.Contains(out, "usage=3 initiative(s)") {
		t.Errorf("usage line missing: %q", out)
	}
}

func TestCmdOperatingModeList_Empty(t *testing.T) {
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"modes":[]}`))
	}))
	app := newAppT(t)
	out := clitest.CaptureStdout(t, func() error { return app.cmdOperatingModeList([]string{}) })
	if !strings.Contains(out, "(none)") {
		t.Errorf("empty output = %q", out)
	}
}

func TestCmdOperatingModeGet_RequiresMode(t *testing.T) {
	app := newAppT(t)
	if err := app.cmdOperatingModeGet([]string{}); err == nil || !strings.Contains(err.Error(), "--mode is required") {
		t.Fatalf("expected --mode required, got %v", err)
	}
}

func TestCmdOperatingModeGet_RendersDetail(t *testing.T) {
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/operating-modes/holistic-loop" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"entry":{"mode":"holistic-loop","label":"Holistic","scope_kind":"initiative","run_strategy":"sequential_handoff","usage_count":2}}`))
	}))
	app := newAppT(t)
	out := clitest.CaptureStdout(t, func() error { return app.cmdOperatingModeGet([]string{"--mode", " holistic-loop "}) })
	if !strings.Contains(out, "holistic-loop") || !strings.Contains(out, "Sequential handoff") {
		t.Errorf("detail missing humanized strategy: %q", out)
	}
}

func TestCmdStatsSummary_QueryAndJSON(t *testing.T) {
	var gotQuery url.Values
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/stats" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte(`{"event_count":5,"generated_at":"now"}`))
	}))
	app := newAppT(t)
	out := clitest.CaptureStdout(t, func() error { return app.cmdStatsSummary([]string{"--json"}) })
	// summary fetch has no category param.
	if gotQuery.Get("category") != "" {
		t.Errorf("summary should not set category, got %q", gotQuery.Get("category"))
	}
	if !strings.Contains(out, `"event_count": 5`) {
		t.Errorf("json output = %q", out)
	}
}

func TestCmdStatsCategory_SetsCategoryParam(t *testing.T) {
	var gotQuery url.Values
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte(`{"throughput":{}}`))
	}))
	app := newAppT(t)
	_ = clitest.CaptureStdout(t, func() error { return app.cmdStatsThroughput([]string{"--format", "markdown"}) })
	if gotQuery.Get("category") != "throughput" {
		t.Errorf("category param = %q, want throughput", gotQuery.Get("category"))
	}
}
