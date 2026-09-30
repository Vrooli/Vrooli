package deps

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	testutil "github.com/vrooli/cli-core/cliapptest"

	dependenciesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/dependencies"
	dependenciesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/dependencies/dependencies_v1connect"
)

type fakeDependencyService struct {
	dependenciesconnect.UnimplementedDependencyServiceHandler
	gotList    *dependenciesv1.ListVulnerabilitiesRequest
	gotExplain *dependenciesv1.ExplainVulnerabilityRequest
}

func (f *fakeDependencyService) ListVulnerabilities(_ context.Context, req *connect.Request[dependenciesv1.ListVulnerabilitiesRequest]) (*connect.Response[dependenciesv1.ListVulnerabilitiesResponse], error) {
	f.gotList = req.Msg
	return connect.NewResponse(&dependenciesv1.ListVulnerabilitiesResponse{
		Total: 1,
		Vulnerabilities: []*dependenciesv1.VulnerabilityRecord{{
			VulnerabilityId:    "GHSA-1234",
			Ecosystem:          dependenciesv1.Ecosystem_ECOSYSTEM_NPM,
			Name:               "vite",
			Version:            "5.0.0",
			NormalizedSeverity: "high",
			Confidence:         dependenciesv1.EvidenceConfidence_EVIDENCE_CONFIDENCE_DEGRADED,
			Scenarios:          []string{"demo"},
			FixedRanges:        []*dependenciesv1.FixedVersionRange{{Range: ">=5.1.0", Version: "5.1.0"}},
		}},
	}), nil
}

func (f *fakeDependencyService) ExplainVulnerability(_ context.Context, req *connect.Request[dependenciesv1.ExplainVulnerabilityRequest]) (*connect.Response[dependenciesv1.ExplainVulnerabilityResponse], error) {
	f.gotExplain = req.Msg
	return connect.NewResponse(&dependenciesv1.ExplainVulnerabilityResponse{
		Found: true,
		Vulnerability: &dependenciesv1.VulnerabilityRecord{
			VulnerabilityId:    req.Msg.GetVulnerabilityId(),
			Ecosystem:          dependenciesv1.Ecosystem_ECOSYSTEM_NPM,
			Name:               "vite",
			Version:            "5.0.0",
			NormalizedSeverity: "high",
			Confidence:         dependenciesv1.EvidenceConfidence_EVIDENCE_CONFIDENCE_ADVISORY,
			Scenarios:          []string{"demo"},
		},
	}), nil
}

func mountDeps(t *testing.T, svc dependenciesconnect.DependencyServiceHandler) *cliapp.ScenarioApp {
	t.Helper()
	path, handler := dependenciesconnect.NewDependencyServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return testutil.NewTestApp(t, mux)
}

func TestVulnerabilities_JSONUsesStructuredEndpoint(t *testing.T) {
	svc := &fakeDependencyService{}
	h := newHandlers(mountDeps(t, svc))
	ctx, out := cliapptest.NewCapturedRunContext(h.core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "package"}},
		Flags: []cliapp.Flag{
			{Name: "ecosystem"},
			{Name: "scenario"},
			{Name: "vulnerability"},
			{Name: "minimum-confidence"},
			{Name: "limit"},
		},
	}, cliapptest.TestRunContextOptions{
		JSON:        true,
		Positionals: map[string]string{"package": "vite"},
		Flags:       map[string]string{"ecosystem": "npm", "limit": "5"},
	})

	require.NoError(t, h.vulnerabilities(ctx))
	require.Equal(t, dependenciesv1.Ecosystem_ECOSYSTEM_NPM, svc.gotList.GetEcosystem())
	require.Equal(t, "vite", svc.gotList.GetPackageName())
	require.Equal(t, int32(5), svc.gotList.GetLimit())

	var got map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	require.Equal(t, float64(1), got["total"])
	require.NotEmpty(t, got["vulnerabilities"])
}

func TestExplain_JSONUsesStructuredEndpoint(t *testing.T) {
	svc := &fakeDependencyService{}
	h := newHandlers(mountDeps(t, svc))
	ctx, out := cliapptest.NewCapturedRunContext(h.core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "vulnerability"}},
		Flags:       []cliapp.Flag{{Name: "ecosystem"}, {Name: "package"}},
	}, cliapptest.TestRunContextOptions{
		JSON:        true,
		Positionals: map[string]string{"vulnerability": "GHSA-1234"},
		Flags:       map[string]string{"ecosystem": "npm", "package": "vite"},
	})

	require.NoError(t, h.explain(ctx))
	require.Equal(t, "GHSA-1234", svc.gotExplain.GetVulnerabilityId())
	require.Equal(t, dependenciesv1.Ecosystem_ECOSYSTEM_NPM, svc.gotExplain.GetEcosystem())
	require.Equal(t, "vite", svc.gotExplain.GetPackageName())

	var got map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	require.Equal(t, true, got["found"])
	require.NotEmpty(t, got["vulnerability"])
}
