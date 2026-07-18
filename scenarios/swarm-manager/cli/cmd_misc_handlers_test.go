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
	if !strings.Contains(out, "Found 1 capture(s)") || !strings.Contains(out, strings.Repeat("x", 60)+"...") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestCmdCapturesList_Empty(t *testing.T) {
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"captures":[]}`)) }))
	app := newAppT(t)
	out := clitest.CaptureStdout(t, func() error { return app.cmdCapturesList([]string{}) })
	if !strings.Contains(out, "No captures found.") {
		t.Errorf("empty output = %q", out)
	}
}

func TestCmdCapturesGet_RequiresID(t *testing.T) {
	if err := newAppT(t).cmdCapturesGet([]string{}); err == nil || !strings.Contains(err.Error(), "--id is required") {
		t.Fatalf("expected --id required, got %v", err)
	}
}

func TestCmdStatsSummary_QueryAndJSON(t *testing.T) {
	var gotQuery url.Values
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte(`{"event_count":5,"generated_at":"now"}`))
	}))
	out := clitest.CaptureStdout(t, func() error { return newAppT(t).cmdStatsSummary([]string{"--json"}) })
	if gotQuery.Get("category") != "" || !strings.Contains(out, `"event_count": 5`) {
		t.Errorf("query=%v output=%q", gotQuery, out)
	}
}

func TestCmdStatsCategory_SetsCategoryParam(t *testing.T) {
	var gotQuery url.Values
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte(`{"throughput":{}}`))
	}))
	_ = clitest.CaptureStdout(t, func() error { return newAppT(t).cmdStatsThroughput([]string{"--format", "markdown"}) })
	if gotQuery.Get("category") != "throughput" {
		t.Errorf("category param = %q, want throughput", gotQuery.Get("category"))
	}
}
