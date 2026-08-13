package adoptions

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	adoptionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptions"
	adoptionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptions/adoptions_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

type adoptionsService struct {
	mu                sync.Mutex
	listResp          *adoptionsv1.ListAdoptionsResponse
	listScenariosResp *adoptionsv1.ListScenariosResponse
	effectiveResp     *adoptionsv1.ListEffectiveAdoptionsResponse
	applyResp         *adoptionsv1.ApplyAdoptionResponse
	reapplyResp       *adoptionsv1.ReapplyAdoptionResponse
	deleteResp        *adoptionsv1.DeleteAdoptionResponse
	refreshResp       *adoptionsv1.RefreshAdoptionsResponse
	resolveResp       *adoptionsv1.ResolveAdoptionPathResponse
	suggestResp       *adoptionsv1.SuggestAdoptionsResponse
	reconcileResp     *adoptionsv1.ReconcileAdoptionsResponse
	reconvergeResp    *adoptionsv1.ReconvergeAdoptionsResponse
	reconvergeReqs    []*adoptionsv1.ReconvergeAdoptionsRequest
	listReqs          []*adoptionsv1.ListAdoptionsRequest
	listScenariosReqs []*adoptionsv1.ListScenariosRequest
	applyReqs         []*adoptionsv1.ApplyAdoptionRequest
	reapplyReqs       []*adoptionsv1.ReapplyAdoptionRequest
	refreshReqs       []*adoptionsv1.RefreshAdoptionsRequest
	resolveReqs       []*adoptionsv1.ResolveAdoptionPathRequest
	suggestReqs       []*adoptionsv1.SuggestAdoptionsRequest
	reconcileReqs     []*adoptionsv1.ReconcileAdoptionsRequest
	discoverResp      *adoptionsv1.DiscoverAdoptionsResponse
	discoverReqs      []*adoptionsv1.DiscoverAdoptionsRequest
	confirmResp       *adoptionsv1.ConfirmDiscoveryResponse
	confirmReqs       []*adoptionsv1.ConfirmDiscoveryRequest
}

func (s *adoptionsService) DiscoverAdoptions(_ context.Context, req *connect.Request[adoptionsv1.DiscoverAdoptionsRequest]) (*connect.Response[adoptionsv1.DiscoverAdoptionsResponse], error) {
	s.mu.Lock()
	s.discoverReqs = append(s.discoverReqs, req.Msg)
	s.mu.Unlock()
	if s.discoverResp == nil {
		s.discoverResp = &adoptionsv1.DiscoverAdoptionsResponse{}
	}
	return connect.NewResponse(s.discoverResp), nil
}

func (s *adoptionsService) ConfirmDiscovery(_ context.Context, req *connect.Request[adoptionsv1.ConfirmDiscoveryRequest]) (*connect.Response[adoptionsv1.ConfirmDiscoveryResponse], error) {
	s.mu.Lock()
	s.confirmReqs = append(s.confirmReqs, req.Msg)
	s.mu.Unlock()
	if s.confirmResp == nil {
		s.confirmResp = &adoptionsv1.ConfirmDiscoveryResponse{Adoption: sampleAdoption(), WrittenPath: "/tmp/Input.tsx", Similarity: 0.97}
	}
	return connect.NewResponse(s.confirmResp), nil
}

func (s *adoptionsService) ListAdoptions(_ context.Context, req *connect.Request[adoptionsv1.ListAdoptionsRequest]) (*connect.Response[adoptionsv1.ListAdoptionsResponse], error) {
	s.mu.Lock()
	s.listReqs = append(s.listReqs, req.Msg)
	s.mu.Unlock()
	if s.listResp == nil {
		s.listResp = &adoptionsv1.ListAdoptionsResponse{}
	}
	return connect.NewResponse(s.listResp), nil
}

func (s *adoptionsService) ListScenarios(_ context.Context, req *connect.Request[adoptionsv1.ListScenariosRequest]) (*connect.Response[adoptionsv1.ListScenariosResponse], error) {
	s.mu.Lock()
	s.listScenariosReqs = append(s.listScenariosReqs, req.Msg)
	s.mu.Unlock()
	if s.listScenariosResp == nil {
		s.listScenariosResp = &adoptionsv1.ListScenariosResponse{}
	}
	return connect.NewResponse(s.listScenariosResp), nil
}

func (s *adoptionsService) ListEffectiveAdoptions(_ context.Context, _ *connect.Request[adoptionsv1.ListEffectiveAdoptionsRequest]) (*connect.Response[adoptionsv1.ListEffectiveAdoptionsResponse], error) {
	if s.effectiveResp == nil {
		s.effectiveResp = &adoptionsv1.ListEffectiveAdoptionsResponse{}
	}
	return connect.NewResponse(s.effectiveResp), nil
}

func (s *adoptionsService) ApplyAdoption(_ context.Context, req *connect.Request[adoptionsv1.ApplyAdoptionRequest]) (*connect.Response[adoptionsv1.ApplyAdoptionResponse], error) {
	s.mu.Lock()
	s.applyReqs = append(s.applyReqs, req.Msg)
	s.mu.Unlock()
	if s.applyResp == nil {
		s.applyResp = &adoptionsv1.ApplyAdoptionResponse{Adoption: sampleAdoption(), WrittenPath: "/tmp/Button.tsx"}
	}
	return connect.NewResponse(s.applyResp), nil
}

func (s *adoptionsService) ReapplyAdoption(_ context.Context, req *connect.Request[adoptionsv1.ReapplyAdoptionRequest]) (*connect.Response[adoptionsv1.ReapplyAdoptionResponse], error) {
	s.mu.Lock()
	s.reapplyReqs = append(s.reapplyReqs, req.Msg)
	s.mu.Unlock()
	if s.reapplyResp == nil {
		s.reapplyResp = &adoptionsv1.ReapplyAdoptionResponse{Adoption: sampleAdoption(), WrittenPath: "/tmp/Button.tsx"}
	}
	return connect.NewResponse(s.reapplyResp), nil
}

func (s *adoptionsService) DeleteAdoption(_ context.Context, _ *connect.Request[adoptionsv1.DeleteAdoptionRequest]) (*connect.Response[adoptionsv1.DeleteAdoptionResponse], error) {
	if s.deleteResp == nil {
		s.deleteResp = &adoptionsv1.DeleteAdoptionResponse{}
	}
	return connect.NewResponse(s.deleteResp), nil
}

func (s *adoptionsService) RefreshAdoptions(_ context.Context, req *connect.Request[adoptionsv1.RefreshAdoptionsRequest]) (*connect.Response[adoptionsv1.RefreshAdoptionsResponse], error) {
	s.mu.Lock()
	s.refreshReqs = append(s.refreshReqs, req.Msg)
	s.mu.Unlock()
	if s.refreshResp == nil {
		s.refreshResp = &adoptionsv1.RefreshAdoptionsResponse{}
	}
	return connect.NewResponse(s.refreshResp), nil
}

func (s *adoptionsService) ReconcileAdoptions(_ context.Context, req *connect.Request[adoptionsv1.ReconcileAdoptionsRequest]) (*connect.Response[adoptionsv1.ReconcileAdoptionsResponse], error) {
	s.mu.Lock()
	s.reconcileReqs = append(s.reconcileReqs, req.Msg)
	s.mu.Unlock()
	if s.reconcileResp == nil {
		s.reconcileResp = &adoptionsv1.ReconcileAdoptionsResponse{}
	}
	return connect.NewResponse(s.reconcileResp), nil
}

func (s *adoptionsService) ReconvergeAdoptions(_ context.Context, req *connect.Request[adoptionsv1.ReconvergeAdoptionsRequest]) (*connect.Response[adoptionsv1.ReconvergeAdoptionsResponse], error) {
	s.mu.Lock()
	s.reconvergeReqs = append(s.reconvergeReqs, req.Msg)
	s.mu.Unlock()
	if s.reconvergeResp == nil {
		s.reconvergeResp = &adoptionsv1.ReconvergeAdoptionsResponse{}
	}
	return connect.NewResponse(s.reconvergeResp), nil
}

func (s *adoptionsService) ResolveAdoptionPath(_ context.Context, req *connect.Request[adoptionsv1.ResolveAdoptionPathRequest]) (*connect.Response[adoptionsv1.ResolveAdoptionPathResponse], error) {
	s.mu.Lock()
	s.resolveReqs = append(s.resolveReqs, req.Msg)
	s.mu.Unlock()
	if s.resolveResp == nil {
		s.resolveResp = &adoptionsv1.ResolveAdoptionPathResponse{
			Path:   "ui/src/components/Button.tsx",
			Source: adoptionsv1.ResolveSource_RESOLVE_SOURCE_TEMPLATE_MANIFEST,
			Slot:   "ui-primitive",
		}
	}
	return connect.NewResponse(s.resolveResp), nil
}

func (s *adoptionsService) SuggestAdoptions(_ context.Context, req *connect.Request[adoptionsv1.SuggestAdoptionsRequest]) (*connect.Response[adoptionsv1.SuggestAdoptionsResponse], error) {
	s.mu.Lock()
	s.suggestReqs = append(s.suggestReqs, req.Msg)
	s.mu.Unlock()
	if s.suggestResp == nil {
		s.suggestResp = &adoptionsv1.SuggestAdoptionsResponse{}
	}
	return connect.NewResponse(s.suggestResp), nil
}

func connectAPI(t *testing.T, svc *adoptionsService) http.Handler {
	t.Helper()
	path, handler := adoptionsconnect.NewAdoptionsServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func sampleAdoption() *adoptionsv1.Adoption {
	ts := timestamppb.New(time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC))
	return &adoptionsv1.Adoption{
		Id:                   "ad-1",
		ComponentId:          "cmp-btn",
		LibraryId:            "rcl:Button",
		Scenario:             "swarm-manager",
		AdoptedPath:          "ui/Button.tsx",
		AdoptedVersion:       "1.0.0",
		LibraryVersionStatus: adoptionsv1.LibraryVersionStatus_LIBRARY_VERSION_STATUS_BEHIND,
		LocalStatus:          adoptionsv1.LocalStatus_LOCAL_STATUS_CLEAN,
		StatusDetail:         "library at 1.1.0",
		CreatedAt:            ts,
		RefreshedAt:          ts,
	}
}

func TestAdoptionsList_ForwardsFiltersAndRenders(t *testing.T) {
	svc := &adoptionsService{listResp: &adoptionsv1.ListAdoptionsResponse{
		Adoptions: []*adoptionsv1.Adoption{sampleAdoption()},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "component-id"}, {Name: "scenario"}, {Name: "limit"}},
	}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"scenario": "swarm-manager", "limit": "50"},
	})
	require.NoError(t, h.list(ctx))
	require.Len(t, svc.listReqs, 1)
	require.Equal(t, "swarm-manager", svc.listReqs[0].Scenario)
	require.Equal(t, int32(50), svc.listReqs[0].Limit)
	require.Contains(t, out.String(), "Found 1 adoption(s).")
	require.Contains(t, out.String(), "behind")
}

func TestAdoptionsListScenarios_RendersTypedDisplayOptions(t *testing.T) {
	svc := &adoptionsService{listScenariosResp: &adoptionsv1.ListScenariosResponse{
		Scenarios: []*adoptionsv1.ScenarioOption{{Name: "swarm-manager", DisplayName: "Swarm Manager"}, {Name: "scenario-qa", DisplayName: "scenario-qa"}},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.listScenarios(ctx))
	require.Len(t, svc.listScenariosReqs, 1)
	require.Contains(t, out.String(), "Found 2 selectable scenario(s).")
	require.Contains(t, out.String(), "Swarm Manager (swarm-manager)")
	require.Contains(t, out.String(), "scenario-qa")
}

func TestAdoptionsDiscover_ForwardsThresholdAndRendersEvidence(t *testing.T) {
	svc := &adoptionsService{discoverResp: &adoptionsv1.DiscoverAdoptionsResponse{
		Scanned:       3,
		MinSimilarity: 0.7,
		Candidates: []*adoptionsv1.DiscoveryCandidate{{
			Scenario: "experience-manager", AdoptedPath: "ui/src/components/ui/input.tsx",
			ComponentId: "cmp-input", LibraryId: "rcl:Input", Version: "1.1.0",
			Similarity: 0.97, SharedLines: 24, Evidence: []string{"Sørensen–Dice line similarity 0.970 against rcl:Input@1.1.0 (Input.tsx)"},
		}},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "scenario"}, {Name: "min-similarity"}, {Name: "limit"}},
	}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"scenario": "experience-manager", "min-similarity": "0.7", "limit": "50"},
	})
	require.NoError(t, h.discover(ctx))
	require.Len(t, svc.discoverReqs, 1)
	require.Equal(t, "experience-manager", svc.discoverReqs[0].Scenario)
	require.InDelta(t, 0.7, svc.discoverReqs[0].MinSimilarity, 0.001)
	require.Equal(t, int32(50), svc.discoverReqs[0].Limit)
	require.Contains(t, out.String(), "surfaced 1 candidate(s)")
	require.Contains(t, out.String(), "rcl:Input")
}

func TestAdoptionsReconverge_ForwardsScopeAndRendersOutcomes(t *testing.T) {
	svc := &adoptionsService{reconvergeResp: &adoptionsv1.ReconvergeAdoptionsResponse{
		Scanned: 9, Behind: 9, Reapplied: 8, Flagged: 1,
		Outcomes: []*adoptionsv1.ReconvergeOutcome{
			{
				Scenario: "template-manager", LibraryId: "rcl:Button", AdoptionId: "row-1",
				AdoptedVersion: "1.1.0", TargetVersion: "1.2.0",
				Action: adoptionsv1.ReconvergeAction_RECONVERGE_ACTION_REAPPLIED,
				Files: []*adoptionsv1.ReconvergeFileOutcome{{
					AdoptedPath: "ui/src/components/ui/button.tsx",
					LocalStatus: adoptionsv1.LocalStatus_LOCAL_STATUS_CLEAN,
				}},
			},
			{
				Scenario: "template-manager", LibraryId: "rcl:StatusBadge", AdoptionId: "row-2",
				AdoptedVersion: "1.0.0", TargetVersion: "1.1.0",
				Action: adoptionsv1.ReconvergeAction_RECONVERGE_ACTION_FLAGGED_MODIFIED,
				Files: []*adoptionsv1.ReconvergeFileOutcome{{
					AdoptedPath: "ui/src/components/ui/status-badge.tsx",
					LocalStatus: adoptionsv1.LocalStatus_LOCAL_STATUS_MODIFIED,
				}},
			},
		},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "scenario"}, {Name: "apply"}},
	}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"scenario": "template-manager", "apply": "true"},
	})
	require.NoError(t, h.reconverge(ctx))
	require.Len(t, svc.reconvergeReqs, 1)
	require.Equal(t, "template-manager", svc.reconvergeReqs[0].Scenario)
	require.True(t, svc.reconvergeReqs[0].Apply)
	require.Contains(t, out.String(), "Applied reconverge")
	require.Contains(t, out.String(), "reapplied 8")
	require.Contains(t, out.String(), "flagged-modified")
	require.Contains(t, out.String(), "rcl:Button")
}

func TestAdoptionsDiscover_RejectsBadThreshold(t *testing.T) {
	core := clitest.NewTestApp(t, connectAPI(t, &adoptionsService{}))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "scenario"}, {Name: "min-similarity"}, {Name: "limit"}},
	}, cliapptest.TestRunContextOptions{Flags: map[string]string{"min-similarity": "high"}})
	require.Error(t, h.discover(ctx))
}

func TestAdoptionsConfirmDiscovery_ForwardsPositionals(t *testing.T) {
	svc := &adoptionsService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "scenario"}, {Name: "adopted-path"}, {Name: "component-id"}, {Name: "version"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{
			"scenario":     "experience-manager",
			"adopted-path": "ui/src/components/ui/input.tsx",
			"component-id": "cmp-input",
			"version":      "1.1.0",
		},
	})
	require.NoError(t, h.confirmDiscovery(ctx))
	require.Len(t, svc.confirmReqs, 1)
	require.Equal(t, "experience-manager", svc.confirmReqs[0].Scenario)
	require.Equal(t, "ui/src/components/ui/input.tsx", svc.confirmReqs[0].AdoptedPath)
	require.Equal(t, "cmp-input", svc.confirmReqs[0].ComponentId)
	require.Equal(t, "1.1.0", svc.confirmReqs[0].Version)
	require.Contains(t, out.String(), "Injected provenance header")
}

func TestAdoptionsList_RejectsBadLimit(t *testing.T) {
	core := clitest.NewTestApp(t, connectAPI(t, &adoptionsService{}))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "component-id"}, {Name: "scenario"}, {Name: "limit"}},
	}, cliapptest.TestRunContextOptions{Flags: map[string]string{"limit": "x"}})
	require.Error(t, h.list(ctx))
}

func TestAdoptionsApply_ForwardsPositionalsAndVersion(t *testing.T) {
	svc := &adoptionsService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "component-id"}, {Name: "scenario"}, {Name: "adopted-path"}},
		Flags:       []cliapp.Flag{{Name: "version"}, {Name: "replace-existing"}, {Name: "confirm-overwrite"}, {Name: "override-validation"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{
			"component-id": "cmp-btn",
			"scenario":     "swarm-manager",
			"adopted-path": "ui/Button.tsx",
		},
		Flags: map[string]string{"version": "1.0.0", "replace-existing": "true", "confirm-overwrite": "true", "override-validation": "true"},
	})
	require.NoError(t, h.apply(ctx))
	require.Len(t, svc.applyReqs, 1)
	require.Equal(t, "cmp-btn", svc.applyReqs[0].ComponentId)
	require.Equal(t, "ui/Button.tsx", svc.applyReqs[0].AdoptedPath)
	require.Equal(t, "1.0.0", svc.applyReqs[0].Version)
	require.True(t, svc.applyReqs[0].ConfirmOverwrite)
	require.True(t, svc.applyReqs[0].ReplaceExisting)
	require.True(t, svc.applyReqs[0].OverrideValidation)
}

func TestAdoptionsReapply_ForwardsValidationOverride(t *testing.T) {
	svc := &adoptionsService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "id"}},
		Flags:       []cliapp.Flag{{Name: "version"}, {Name: "confirm-local-overwrite"}, {Name: "override-validation"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"id": "ad-1"},
		Flags:       map[string]string{"version": "1.0.0", "confirm-local-overwrite": "true", "override-validation": "true"},
	})
	require.NoError(t, h.reapply(ctx))
	require.Len(t, svc.reapplyReqs, 1)
	require.True(t, svc.reapplyReqs[0].ConfirmLocalOverwrite)
	require.True(t, svc.reapplyReqs[0].OverrideValidation)
}

func TestAdoptionsRefresh_SummaryLineIncludesCounts(t *testing.T) {
	svc := &adoptionsService{refreshResp: &adoptionsv1.RefreshAdoptionsResponse{
		Adoptions:      []*adoptionsv1.Adoption{sampleAdoption()},
		LibraryCurrent: 2, LibraryBehind: 1, LocalClean: 2, LocalModified: 0, LocalMissing: 1,
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "component-id"}},
	}, cliapptest.TestRunContextOptions{Flags: map[string]string{"component-id": "cmp-btn"}})
	require.NoError(t, h.refresh(ctx))
	require.Len(t, svc.refreshReqs, 1)
	require.Equal(t, "cmp-btn", svc.refreshReqs[0].ComponentId)
	body := out.String()
	require.Contains(t, body, "current=2")
	require.Contains(t, body, "behind=1")
	require.Contains(t, body, "missing=1")
}
