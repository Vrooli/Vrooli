package adoptions

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	intakeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptionintake/adoptionintake_v1connect"
	adoptionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptions"
	adoptionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptions/adoptions_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

func TestLinkedImportPresentRecognizesPackageAdoption(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "PasswordManagerApp.tsx")
	require.NoError(t, os.WriteFile(source, []byte(`import { Input } from "@vrooli/react-component-library/Input/1.3.6";`), 0o644))

	require.True(t, linkedImportPresent(root, "./Input/1.3.6"))
	require.False(t, linkedImportPresent(root, "./Dialog/1.3.7"))
}

func TestResolveDurableFilesUsesTemplateAndScenarioOverrides(t *testing.T) {
	root := t.TempDir()
	scenario := "sample"
	write := func(path, body string) {
		t.Helper()
		full := filepath.Join(root, filepath.FromSlash(path))
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(body), 0o600))
	}
	write("scenarios/sample/.vrooli/service.json", `{"generation":{"template":{"id":"react-vite"}}}`)
	write("templates/scenarios/react-vite/ui/manifest.json", `{"files":{"designTokens":{"path":"src/theme/{locale}/tokens.css"},"appEntry":{"path":"src/main.tsx"},"unknown":{"path":"ignored"}}}`)
	write("scenarios/sample/.vrooli/ui-manifest.json", `{"files":{"designTokens":{"path":"src/theme/fr/tokens.css"},"selectorRegistry":{"path":"src/selectors.ts"}}}`)

	got := resolveDurableFiles(root, scenario)
	require.Equal(t, "src/theme/fr/tokens.css", got["designTokens"])
	require.Equal(t, "src/main.tsx", got["appEntry"])
	require.Equal(t, "ui/src/consts/selectors.library.ts", got["librarySelectors"])
	require.Equal(t, "src/selectors.ts", got["selectorRegistry"])
	require.NotContains(t, got, "unknown")
}

func TestResolveDurableFilesFallsBackOnMissingOrInvalidMetadata(t *testing.T) {
	want := map[string]string{"designTokens": "ui/src/design-tokens.css", "librarySelectors": "ui/src/consts/selectors.library.ts", "selectorRegistry": "ui/src/consts/selectors.ts", "appEntry": "ui/src/main.tsx"}
	for _, setup := range []func(*testing.T, string){
		func(*testing.T, string) {},
		func(t *testing.T, root string) {
			path := filepath.Join(root, "scenarios/sample/.vrooli/service.json")
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
			require.NoError(t, os.WriteFile(path, []byte("{"), 0o600))
		},
	} {
		root := t.TempDir()
		setup(t, root)
		require.Equal(t, want, resolveDurableFiles(root, "sample"))
	}
}

func TestStringsProviderWiredRequiresProviderMarker(t *testing.T) {
	root := t.TempDir()
	entry := filepath.Join(root, "main.tsx")
	require.False(t, stringsProviderWired(entry))
	require.NoError(t, os.WriteFile(entry, []byte("export function App() {}"), 0o600))
	require.False(t, stringsProviderWired(entry))
	require.NoError(t, os.WriteFile(entry, []byte("// vrooli:library-strings-provider start\n<App />"), 0o600))
	require.True(t, stringsProviderWired(entry))
}

func TestFindRepoRootFromScenarioCLI(t *testing.T) {
	root, err := findRepoRoot()
	require.NoError(t, err)
	require.DirExists(t, filepath.Join(root, "scenarios", "react-component-library", "api"))
}

type adoptionsService struct {
	mu                sync.Mutex
	listResp          *adoptionsv1.ListAdoptionsResponse
	listScenariosResp *adoptionsv1.ListScenariosResponse
	effectiveResp     *adoptionsv1.ListEffectiveAdoptionsResponse
	effectiveReq      *adoptionsv1.ListEffectiveAdoptionsRequest
	preflightReq      *adoptionsv1.PreflightAdoptionRequest
	preflightResp     *adoptionsv1.PreflightAdoptionResponse
	syncResp          *adoptionsv1.SyncScenarioTokensResponse
	pruneReq          *adoptionsv1.PruneScenarioTokensRequest
	pruneResp         *adoptionsv1.PruneScenarioTokensResponse
	linkReq           *adoptionsv1.LinkAdoptionRequest
	linkResp          *adoptionsv1.LinkAdoptionResponse
	batchReq          *adoptionsv1.BatchApplyAdoptionsRequest
	batchResp         *adoptionsv1.BatchApplyAdoptionsResponse
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
	tokenSyncReqs     []*adoptionsv1.SyncScenarioTokensRequest
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

func (s *adoptionsService) ListEffectiveAdoptions(_ context.Context, req *connect.Request[adoptionsv1.ListEffectiveAdoptionsRequest]) (*connect.Response[adoptionsv1.ListEffectiveAdoptionsResponse], error) {
	s.effectiveReq = req.Msg
	if s.effectiveResp == nil {
		s.effectiveResp = &adoptionsv1.ListEffectiveAdoptionsResponse{}
	}
	return connect.NewResponse(s.effectiveResp), nil
}

func (s *adoptionsService) PreflightAdoption(_ context.Context, req *connect.Request[adoptionsv1.PreflightAdoptionRequest]) (*connect.Response[adoptionsv1.PreflightAdoptionResponse], error) {
	s.preflightReq = req.Msg
	if s.preflightResp == nil {
		s.preflightResp = &adoptionsv1.PreflightAdoptionResponse{ComponentId: "cmp-btn", Scenario: "swarm-manager", Version: "1.0.0"}
	}
	return connect.NewResponse(s.preflightResp), nil
}

func (s *adoptionsService) SyncScenarioTokens(_ context.Context, req *connect.Request[adoptionsv1.SyncScenarioTokensRequest]) (*connect.Response[adoptionsv1.SyncScenarioTokensResponse], error) {
	s.mu.Lock()
	s.tokenSyncReqs = append(s.tokenSyncReqs, req.Msg)
	s.mu.Unlock()
	if s.syncResp == nil {
		s.syncResp = &adoptionsv1.SyncScenarioTokensResponse{}
	}
	return connect.NewResponse(s.syncResp), nil
}

func (s *adoptionsService) PruneScenarioTokens(_ context.Context, req *connect.Request[adoptionsv1.PruneScenarioTokensRequest]) (*connect.Response[adoptionsv1.PruneScenarioTokensResponse], error) {
	s.pruneReq = req.Msg
	if s.pruneResp == nil {
		s.pruneResp = &adoptionsv1.PruneScenarioTokensResponse{}
	}
	return connect.NewResponse(s.pruneResp), nil
}

func (s *adoptionsService) LinkAdoption(_ context.Context, req *connect.Request[adoptionsv1.LinkAdoptionRequest]) (*connect.Response[adoptionsv1.LinkAdoptionResponse], error) {
	s.linkReq = req.Msg
	if s.linkResp == nil {
		s.linkResp = &adoptionsv1.LinkAdoptionResponse{Adoption: sampleAdoption(), PackagePath: "file:../../../packages/react-component-library"}
	}
	return connect.NewResponse(s.linkResp), nil
}

func (s *adoptionsService) EjectAdoption(_ context.Context, _ *connect.Request[adoptionsv1.EjectAdoptionRequest]) (*connect.Response[adoptionsv1.EjectAdoptionResponse], error) {
	return connect.NewResponse(&adoptionsv1.EjectAdoptionResponse{Adoption: sampleAdoption(), WrittenPath: "/tmp/Button.tsx"}), nil
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

func (s *adoptionsService) BatchApplyAdoptions(_ context.Context, req *connect.Request[adoptionsv1.BatchApplyAdoptionsRequest]) (*connect.Response[adoptionsv1.BatchApplyAdoptionsResponse], error) {
	s.batchReq = req.Msg
	if s.batchResp == nil {
		s.batchResp = &adoptionsv1.BatchApplyAdoptionsResponse{}
	}
	return connect.NewResponse(s.batchResp), nil
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

func (s *adoptionsService) ForkReportAdoptions(_ context.Context, req *connect.Request[adoptionsv1.RefreshAdoptionsRequest]) (*connect.Response[adoptionsv1.RefreshAdoptionsResponse], error) {
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
	intakePath, intakeHandler := intakeconnect.NewAdoptionIntakeServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	mux.Handle(intakePath, intakeHandler)
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
	msg, err := h.listCall(ctx)
	require.NoError(t, err)
	require.NoError(t, ctx.RenderList(h.listReport(ctx, msg)))
	require.Len(t, svc.listReqs, 1)
	require.Equal(t, "swarm-manager", svc.listReqs[0].Scenario)
	require.Equal(t, int32(50), svc.listReqs[0].Limit)
	require.Contains(t, out.String(), "Found 1 adoption(s).")
	require.Contains(t, out.String(), "behind")
}

func TestTokensSync_ForwardsCandidateAndVersion(t *testing.T) {
	svc := &adoptionsService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "scenario", Required: true}},
		Flags:       []cliapp.Flag{{Name: "dry-run"}, {Name: "candidate"}, {Name: "version"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"scenario": "system-monitor"},
		Flags:       map[string]string{"dry-run": "true", "candidate": "react-component-library:Chart", "version": "1.0.12"},
	})
	_, err := h.syncTokensCall(ctx)
	require.NoError(t, err)
	require.Len(t, svc.tokenSyncReqs, 1)
	require.Equal(t, "system-monitor", svc.tokenSyncReqs[0].Scenario)
	require.True(t, svc.tokenSyncReqs[0].DryRun)
	require.Equal(t, "react-component-library:Chart", svc.tokenSyncReqs[0].CandidateComponentId)
	require.Equal(t, "1.0.12", svc.tokenSyncReqs[0].CandidateVersion)
}

func TestListEffectiveForwardsFiltersAndOmitsRowsWithoutParent(t *testing.T) {
	svc := &adoptionsService{effectiveResp: &adoptionsv1.ListEffectiveAdoptionsResponse{Adoptions: []*adoptionsv1.EffectiveAdoption{
		{SourceLibraryId: "rcl:Button", SourceVersion: "1.2.0", ParentAdoption: sampleAdoption()},
		{SourceLibraryId: "rcl:Input", SourceVersion: "2.0.0", Mediated: true, ParentAdoption: &adoptionsv1.Adoption{Id: "parent-2", Scenario: "web-console"}},
		{SourceLibraryId: "orphan", SourceVersion: "0.1.0"},
	}}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "component-id"}}, Flags: []cliapp.Flag{{Name: "limit"}}}, cliapptest.TestRunContextOptions{Positionals: map[string]string{"component-id": "catalog.asset"}, Flags: map[string]string{"limit": "17"}})
	msg, err := newHandlers(core).listEffectiveCall(ctx)
	require.NoError(t, err)
	require.Equal(t, "catalog.asset", svc.effectiveReq.ComponentId)
	require.Equal(t, int32(17), svc.effectiveReq.Limit)
	require.NoError(t, ctx.RenderList(newHandlers(core).listEffectiveReport(ctx, msg)))
	require.Contains(t, out.String(), "Found 2 effective adoption(s).")
	require.Contains(t, out.String(), "direct rcl:Button@1.2.0 via swarm-manager (ad-1)")
	require.Contains(t, out.String(), "indirect rcl:Input@2.0.0 via web-console (parent-2)")
	require.NotContains(t, out.String(), "orphan")
}

func TestPreflightCallForwardsInputsAndReportRendersWarnings(t *testing.T) {
	svc := &adoptionsService{preflightResp: &adoptionsv1.PreflightAdoptionResponse{
		ComponentId: "rcl:Button", Scenario: "web-console", Version: "1.4.0", Blocking: true,
		RequiredTokens: []string{"color.primary"}, RequiredTokenPatterns: []string{"space.*"}, UnsatisfiedTokens: []string{"color.secondary"},
		VersionStatus: "behind", MaturityRung: "R2", MaturityFloor: "R1", I18NVerdict: "pass", SelectorVerdict: "pass", Warnings: []string{"manual review required"},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "component-id"}, {Name: "scenario"}}, Flags: []cliapp.Flag{{Name: "version"}}}, cliapptest.TestRunContextOptions{Positionals: map[string]string{"component-id": "rcl:Button", "scenario": "web-console"}, Flags: map[string]string{"version": "1.4.0"}})
	msg, err := h.preflightCall(ctx)
	require.NoError(t, err)
	require.Equal(t, "rcl:Button", svc.preflightReq.ComponentId)
	require.Equal(t, "web-console", svc.preflightReq.Scenario)
	require.Equal(t, "1.4.0", svc.preflightReq.Version)
	require.NoError(t, ctx.RenderList(h.preflightReport(ctx, msg)))
	require.Contains(t, out.String(), "UNSATISFIED color.secondary")
	require.Contains(t, out.String(), "manual review required")
}

func TestSyncAndPruneTokenCommandsRenderChangedSummaries(t *testing.T) {
	svc := &adoptionsService{
		syncResp:  &adoptionsv1.SyncScenarioTokensResponse{Scenario: "web-console", Added: []string{"color.primary"}, Changed: true},
		pruneResp: &adoptionsv1.PruneScenarioTokensResponse{Scenario: "web-console", Removed: []string{"color.legacy"}, Retained: []string{"color.primary"}, Changed: true},
	}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario"}}, Flags: []cliapp.Flag{{Name: "dry-run"}, {Name: "apply"}, {Name: "candidate"}, {Name: "version"}}}
	ctx, out := cliapptest.NewCapturedRunContext(core, schema, cliapptest.TestRunContextOptions{Positionals: map[string]string{"scenario": "web-console"}, Flags: map[string]string{"dry-run": "false", "apply": "true", "candidate": "rcl:Button", "version": "1.2.0"}})
	syncMsg, err := h.syncTokensCall(ctx)
	require.NoError(t, err)
	require.NoError(t, h.syncTokensHuman(ctx, syncMsg))
	pruneMsg, err := h.pruneTokensCall(ctx)
	require.NoError(t, err)
	require.NoError(t, h.pruneTokensHuman(ctx, pruneMsg))
	require.Equal(t, "web-console", svc.pruneReq.Scenario)
	require.True(t, svc.pruneReq.Apply)
	require.Contains(t, out.String(), "Token sync for web-console: added=1 changed=true")
	require.Contains(t, out.String(), "color.primary")
	require.Contains(t, out.String(), "Token prune for web-console: removed=1 changed=true")
	require.Contains(t, out.String(), "color.legacy")
}

func TestLinkCallForwardsOptionsAndBuildsMutationReport(t *testing.T) {
	svc := &adoptionsService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "component-id"}, {Name: "scenario"}}, Flags: []cliapp.Flag{{Name: "version"}, {Name: "import-subpath"}, {Name: "confirm-existing"}}}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"component-id": "rcl:Button", "scenario": "web-console"},
		Flags:       map[string]string{"version": "1.2.0", "import-subpath": "./Button/1.2.0", "confirm-existing": "true"},
	})
	msg, err := h.linkCall(ctx)
	require.NoError(t, err)
	require.Equal(t, "rcl:Button", svc.linkReq.ComponentId)
	require.Equal(t, "web-console", svc.linkReq.Scenario)
	require.Equal(t, "1.2.0", svc.linkReq.Version)
	require.Equal(t, "./Button/1.2.0", svc.linkReq.ImportSubpath)
	require.True(t, svc.linkReq.ConfirmExisting)
	report := h.linkReport(ctx, msg)
	require.Contains(t, report.Result[0], "Linked ad-1 through file:")
	require.Contains(t, report.Changes[0], "rcl:Button")
}

func TestBatchApplyForwardsItemsAndReportsOnlySuccessfulRows(t *testing.T) {
	svc := &adoptionsService{batchResp: &adoptionsv1.BatchApplyAdoptionsResponse{
		Results:            []*adoptionsv1.BatchApplyItemResult{{Adoption: sampleAdoption()}, {Adoption: nil}},
		SharedDependencies: []string{"ui/src/shared.ts"},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	items := []*adoptionsv1.BatchApplyItem{{ComponentId: "rcl:Button", Scenario: "web-console", AdoptedPath: "ui/Button.tsx", Version: "1.2.0"}}
	require.NoError(t, h.batchApply(ctx, items))
	require.Len(t, svc.batchReq.Items, 1)
	require.Equal(t, "ui/Button.tsx", svc.batchReq.Items[0].AdoptedPath)
	require.Contains(t, out.String(), "Batch-applied 1 adoption(s); shared dependencies=1.")
	require.Contains(t, out.String(), "rcl:Button")
}

func TestPreflightReportGuidesMissingCatalogEvidence(t *testing.T) {
	h := &handlers{}
	report := h.preflightReport(nil, &adoptionsv1.PreflightAdoptionResponse{
		ComponentId: "react-component-library:Chart", Scenario: "system-monitor", Version: "1.0.12",
		CatalogAssetId: "visualization.chart", CatalogLatestVersion: "1.0.12", I18NVerdict: "not-measured", SelectorVerdict: "not-measured",
	})
	joined := strings.Join(report.Results, "\n")
	require.Contains(t, joined, "i18n coverage=not-measured; selector coverage=not-measured")
	require.Contains(t, joined, "catalog record-gate i18n --asset-id visualization.chart")
	require.Contains(t, joined, "catalog record-gate selector-coverage --asset-id visualization.chart")
	require.Contains(t, joined, "adoptions preflight react-component-library:Chart system-monitor --version 1.0.12")
	require.Contains(t, joined, "linking stays blocked until both measures pass")
}

func TestPreflightReportDoesNotSuggestLatestGateForOlderPin(t *testing.T) {
	h := &handlers{}
	report := h.preflightReport(nil, &adoptionsv1.PreflightAdoptionResponse{
		ComponentId: "react-component-library:Chart", Scenario: "system-monitor", Version: "1.0.11",
		CatalogAssetId: "visualization.chart", CatalogLatestVersion: "1.0.12", I18NVerdict: "not-measured", SelectorVerdict: "not-measured",
	})
	joined := strings.Join(report.Results, "\n")
	require.NotContains(t, joined, "catalog record-gate i18n --asset-id visualization.chart")
	require.Contains(t, joined, "Catalog gates measure current visualization.chart@1.0.12")
	require.Contains(t, joined, "selected react-component-library:Chart@1.0.11 needs exact-version gate evidence")
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
	_, err := h.listCall(ctx)
	require.Error(t, err)
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
