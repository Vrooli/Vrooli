package visual

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	clitest "github.com/vrooli/cli-core/cliapptest"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"
	visualpb "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"
	visualconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth/visualhealth_v1connect"
)

type fakeVisualHealthService struct {
	analyzeReqs []*visualpb.AnalyzeArtifactsRequest
	compareReqs []*visualpb.CompareArtifactsRequest
}

func (f *fakeVisualHealthService) AnalyzeArtifacts(_ context.Context, req *connect.Request[visualpb.AnalyzeArtifactsRequest]) (*connect.Response[visualpb.AnalyzeArtifactsResponse], error) {
	f.analyzeReqs = append(f.analyzeReqs, req.Msg)
	return connect.NewResponse(&visualpb.AnalyzeArtifactsResponse{
		Scenario: req.Msg.GetScenario(),
		RunId:    req.Msg.GetRunId(),
		Verdict:  "failed",
		Findings: []*visualpb.VisualFinding{{
			Code:     "visual_dom_blank",
			Severity: visualpb.VisualSeverity_VISUAL_SEVERITY_ERROR,
			Message:  "DOM snapshot contains no meaningful visible text",
		}},
	}), nil
}

func (f *fakeVisualHealthService) CompareArtifacts(_ context.Context, req *connect.Request[visualpb.CompareArtifactsRequest]) (*connect.Response[visualpb.CompareArtifactsResponse], error) {
	f.compareReqs = append(f.compareReqs, req.Msg)
	return connect.NewResponse(&visualpb.CompareArtifactsResponse{
		Deltas: []*visualpb.VisualDelta{{
			Page:            "/",
			Status:          "changed",
			ChangedFraction: 0.25,
		}},
	}), nil
}

func (f *fakeVisualHealthService) ListRules(context.Context, *connect.Request[visualpb.ListRulesRequest]) (*connect.Response[visualpb.ListRulesResponse], error) {
	return connect.NewResponse(&visualpb.ListRulesResponse{
		Rules: []*visualpb.VisualRule{{
			Id:       "visual_dom_blank",
			Category: visualpb.VisualCategory_VISUAL_CATEGORY_DOM,
			Severity: visualpb.VisualSeverity_VISUAL_SEVERITY_ERROR,
		}},
	}), nil
}

func connectAPI(t *testing.T, svc *fakeVisualHealthService) http.Handler {
	t.Helper()
	path, handler := visualconnect.NewVisualHealthServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func TestAnalyzeArtifactsReadsRequestFileAndRendersFindings(t *testing.T) {
	svc := &fakeVisualHealthService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	reqPath := writeRequestFile(t, `{"scenario":"demo","run_id":"run-1","steps":[{"step_id":"load","dom_html":"<main></main>"}]}`)
	ctx, out := cliapptest.NewCapturedRunContext(core, requestSchema(), cliapptest.TestRunContextOptions{
		Flags: map[string]string{"request": reqPath},
	})

	require.NoError(t, h.analyzeArtifacts(ctx))
	require.Len(t, svc.analyzeReqs, 1)
	require.Equal(t, "demo", svc.analyzeReqs[0].GetScenario())
	require.Contains(t, out.String(), "Visual verdict: failed (1 finding(s)).")
	require.Contains(t, out.String(), "visual_dom_blank")
}

func TestCompareArtifactsReadsRequestFileAndRendersDeltas(t *testing.T) {
	svc := &fakeVisualHealthService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	reqPath := writeRequestFile(t, `{"scenario":"demo","base_run_id":"base","current_run_id":"current"}`)
	ctx, out := cliapptest.NewCapturedRunContext(core, requestSchema(), cliapptest.TestRunContextOptions{
		Flags: map[string]string{"request": reqPath},
	})

	require.NoError(t, h.compareArtifacts(ctx))
	require.Len(t, svc.compareReqs, 1)
	require.Equal(t, "base", svc.compareReqs[0].GetBaseRunId())
	require.Contains(t, out.String(), "Compared 1 visual artifact(s).")
	require.Contains(t, out.String(), "/ changed changed=0.2500")
}

func TestRulesRendersRuleRegistry(t *testing.T) {
	core := clitest.NewTestApp(t, connectAPI(t, &fakeVisualHealthService{}))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})

	require.NoError(t, h.rules(ctx))
	require.Contains(t, out.String(), "Found 1 visual-health rule(s).")
	require.Contains(t, out.String(), "visual_dom_blank")
}

func requestSchema() cliapp.ArgSchema {
	return cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "request"}}}
}

func writeRequestFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "request.json")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	return path
}
