package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	clitest "swarm-manager/cli/internal/testutil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
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
	stub := &stubOperatingModeHandler{
		catalog: func(*apipb.OperatingModeCatalogRequest) (*apipb.OperatingModeCatalogResponse, error) {
			return &apipb.OperatingModeCatalogResponse{
				Modes: []*apipb.OperatingModeCatalogEntry{{
					Mode: "holistic-loop", Label: "Holistic", Description: "desc", UsageCount: 3,
					ScopeKind: "initiative", RunStrategy: "sequential_handoff", Default: true,
				}},
			}, nil
		},
	}
	app := newOperatingModeTestApp(t, stub)
	out := clitest.CaptureStdout(t, func() error { return app.cmdOperatingModeList([]string{}) })
	if !strings.Contains(out, "holistic-loop [default] — Holistic") {
		t.Errorf("default mark missing: %q", out)
	}
	if !strings.Contains(out, "usage=3 initiative(s)") {
		t.Errorf("usage line missing: %q", out)
	}
}

func TestCmdOperatingModeList_Empty(t *testing.T) {
	stub := &stubOperatingModeHandler{
		catalog: func(*apipb.OperatingModeCatalogRequest) (*apipb.OperatingModeCatalogResponse, error) {
			return &apipb.OperatingModeCatalogResponse{}, nil
		},
	}
	app := newOperatingModeTestApp(t, stub)
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
	var gotMode string
	stub := &stubOperatingModeHandler{
		getMode: func(req *apipb.OperatingModeGetRequest) (*apipb.OperatingModeDetailResponse, error) {
			gotMode = req.GetMode()
			return &apipb.OperatingModeDetailResponse{
				Entry: &apipb.OperatingModeCatalogEntry{
					Mode: "holistic-loop", Label: "Holistic", ScopeKind: "initiative",
					RunStrategy: "sequential_handoff", UsageCount: 2,
				},
			}, nil
		},
	}
	app := newOperatingModeTestApp(t, stub)
	out := clitest.CaptureStdout(t, func() error { return app.cmdOperatingModeGet([]string{"--mode", " holistic-loop "}) })
	if gotMode != "holistic-loop" {
		t.Errorf("mode selector not trimmed: %q", gotMode)
	}
	if !strings.Contains(out, "holistic-loop") || !strings.Contains(out, "Sequential handoff") {
		t.Errorf("detail missing humanized strategy: %q", out)
	}
}

func TestCmdOperatingModeGet_ShowPromptRendersSelectedPhase(t *testing.T) {
	var gotRender *apipb.OperatingModeRenderSimulationRequest
	stub := &stubOperatingModeHandler{
		simulateMode: func(req *apipb.OperatingModeSimulateRequest) (*apipb.OperatingModeSimulationResponse, error) {
			if req.GetMode() != "holistic-loop" || req.GetPreset() != "branchy" {
				t.Fatalf("simulate request = mode %q preset %q", req.GetMode(), req.GetPreset())
			}
			return &apipb.OperatingModeSimulationResponse{
				Mode:         "holistic-loop",
				ActivePreset: "branchy",
				Trace: []*apipb.OperatingModeSimulationStep{
					{Index: 0, Phase: "investigate"},
					{Index: 1, Phase: "execute"},
				},
			}, nil
		},
		renderPrompt: func(req *apipb.OperatingModeRenderSimulationRequest) (*apipb.OperatingModeRenderPromptResponse, error) {
			gotRender = req
			return &apipb.OperatingModeRenderPromptResponse{
				Mode:       "holistic-loop",
				Preset:     "branchy",
				StepIndex:  1,
				Phase:      "execute",
				SkillId:    "swarm-manager-holistic-loop-execute",
				ProfileKey: "swarm-manager/deep-work",
				Prompt:     "Execute the next slice.",
			}, nil
		},
	}
	app := newOperatingModeTestApp(t, stub)
	out := clitest.CaptureStdout(t, func() error {
		return app.cmdOperatingModeGet([]string{"holistic-loop", "--phase", "execute", "--preset", "branchy", "--show-prompt"})
	})
	if gotRender.GetMode() != "holistic-loop" || gotRender.GetPreset() != "branchy" || gotRender.GetStepIndex() != 1 {
		t.Fatalf("render request = %+v", gotRender)
	}
	if !strings.Contains(out, "SkillID: swarm-manager-holistic-loop-execute") || !strings.Contains(out, "Execute the next slice.") {
		t.Fatalf("prompt output = %q", out)
	}
}

func TestCmdOperatingModeGet_ShowPromptDegradedPrintsVariables(t *testing.T) {
	stub := &stubOperatingModeHandler{
		simulateMode: func(*apipb.OperatingModeSimulateRequest) (*apipb.OperatingModeSimulationResponse, error) {
			return &apipb.OperatingModeSimulationResponse{
				Mode:         "holistic-loop",
				ActivePreset: "happy-path",
				Trace:        []*apipb.OperatingModeSimulationStep{{Index: 0, Phase: "investigate"}},
			}, nil
		},
		renderPrompt: func(*apipb.OperatingModeRenderSimulationRequest) (*apipb.OperatingModeRenderPromptResponse, error) {
			return &apipb.OperatingModeRenderPromptResponse{
				Mode:           "holistic-loop",
				Preset:         "happy-path",
				Phase:          "investigate",
				SkillId:        "swarm-manager-holistic-loop-investigate",
				Variables:      map[string]string{"INITIATIVE_TITLE": "Fixture initiative"},
				Degraded:       true,
				DegradedReason: "prompt client not wired",
			}, nil
		},
	}
	app := newOperatingModeTestApp(t, stub)
	out := clitest.CaptureStdout(t, func() error {
		return app.cmdOperatingModeGet([]string{"--mode", "holistic-loop", "--phase", "investigate", "--show-prompt"})
	})
	if !strings.Contains(out, "degraded") || !strings.Contains(out, "INITIATIVE_TITLE: Fixture initiative") {
		t.Fatalf("degraded prompt output = %q", out)
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
