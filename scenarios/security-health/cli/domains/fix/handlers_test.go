package fix

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	testutil "github.com/vrooli/cli-core/cliapptest"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

type fakeValidationService struct {
	scenariovalidationconnect.UnimplementedScenarioValidationServiceHandler
	gotPreview *scenariovalidationv1.FixRequest
	gotApply   *scenariovalidationv1.FixRequest
}

func (f *fakeValidationService) PreviewFix(_ context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	f.gotPreview = req.Msg
	return connect.NewResponse(&scenariovalidationv1.FixResponse{
		Scenario: req.Msg.GetScenario(),
		Candidates: []*scenariovalidationv1.FixCandidate{{
			RuleId:      "security-health.security-headers-missing",
			FilePath:    "api/internal/server/server.go",
			Description: "Register security headers middleware.",
		}},
	}), nil
}

func (f *fakeValidationService) ApplyFix(_ context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	f.gotApply = req.Msg
	return connect.NewResponse(&scenariovalidationv1.FixResponse{
		Scenario: req.Msg.GetScenario(),
		Applied:  true,
	}), nil
}

func mountValidation(t *testing.T, svc scenariovalidationconnect.ScenarioValidationServiceHandler) *cliapp.ScenarioApp {
	t.Helper()
	path, handler := scenariovalidationconnect.NewScenarioValidationServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return testutil.NewTestApp(t, mux)
}

// [REQ:REQ-P0-021]
func TestPreviewFixSendsScenarioAndDedupedRuleIDs(t *testing.T) {
	svc := &fakeValidationService{}
	app := mountValidation(t, svc)
	h := newHandlers(app)
	ctx, _ := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "scenario"}},
		Flags:       []cliapp.Flag{{Name: "rule"}},
	}, cliapptest.TestRunContextOptions{
		JSON:        true,
		Positionals: map[string]string{"scenario": "demo"},
		Flags:       map[string]string{"rule": "security-health.security-headers-missing,security-health.security-headers-missing"},
	})

	require.NoError(t, h.preview(ctx))
	require.NotNil(t, svc.gotPreview)
	require.Equal(t, "demo", svc.gotPreview.GetScenario())
	require.Equal(t, []string{"security-health.security-headers-missing"}, svc.gotPreview.GetRuleIds())
}

// [REQ:REQ-P0-021]
func TestApplyFixCallsApplyEndpoint(t *testing.T) {
	svc := &fakeValidationService{}
	app := mountValidation(t, svc)
	h := newHandlers(app)
	ctx, _ := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "scenario"}},
		Flags:       []cliapp.Flag{{Name: "rule"}},
	}, cliapptest.TestRunContextOptions{
		JSON:        true,
		Positionals: map[string]string{"scenario": "demo"},
	})

	require.NoError(t, h.apply(ctx))
	require.NotNil(t, svc.gotApply)
	require.Equal(t, "demo", svc.gotApply.GetScenario())
}
