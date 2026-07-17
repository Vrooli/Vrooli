package components

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	componentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components"
	componentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components/components_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "react-component-library/cli/internal/testutil"
)

// componentsService is a hand-written ComponentsServiceHandler used as a fake
// API behind the Connect mux. Mirrors the notes-domain test stub shape.
type componentsService struct {
	mu             sync.Mutex
	listResp       *componentsv1.ListComponentsResponse
	getResp        *componentsv1.GetComponentResponse
	byLibIDResp    *componentsv1.GetComponentByLibraryIdResponse
	indexResp      *componentsv1.IndexComponentsResponse
	contentGetResp *componentsv1.GetComponentContentResponse
	contentSetResp *componentsv1.UpdateComponentContentResponse
	initResp       *componentsv1.InitializeComponentResponse
	ingestResp     *componentsv1.IngestComponentResponse
	versionResp    *componentsv1.CreateComponentVersionResponse
	manifestResp   *componentsv1.UpdateComponentManifestResponse
	examplesResp   *componentsv1.ListComponentExamplesResponse
	styleFitResp   *componentsv1.ValidateStyleFitResponse
	listErr        error
	getErr         error
	byLibIDErr     error
	indexErr       error
	contentGetErr  error
	contentSetErr  error
	initErr        error
	ingestErr      error
	versionErr     error
	manifestErr    error
	examplesErr    error
	styleFitErr    error
	listReqs       []*componentsv1.ListComponentsRequest
	getReqs        []string
	byLibIDReqs    []string
	contentSetReqs []*componentsv1.UpdateComponentContentRequest
	initReqs       []*componentsv1.InitializeComponentRequest
	ingestReqs     []*componentsv1.IngestComponentRequest
	versionReqs    []*componentsv1.CreateComponentVersionRequest
	manifestReqs   []*componentsv1.UpdateComponentManifestRequest
	examplesReqs   []*componentsv1.ListComponentExamplesRequest
	styleFitReqs   []*componentsv1.ValidateStyleFitRequest
}

func (s *componentsService) IngestComponent(_ context.Context, req *connect.Request[componentsv1.IngestComponentRequest]) (*connect.Response[componentsv1.IngestComponentResponse], error) {
	s.mu.Lock()
	s.ingestReqs = append(s.ingestReqs, req.Msg)
	s.mu.Unlock()
	if s.ingestErr != nil {
		return nil, s.ingestErr
	}
	if s.ingestResp == nil {
		s.ingestResp = &componentsv1.IngestComponentResponse{Component: sampleComponent()}
	}
	return connect.NewResponse(s.ingestResp), nil
}

func (s *componentsService) ListComponents(_ context.Context, req *connect.Request[componentsv1.ListComponentsRequest]) (*connect.Response[componentsv1.ListComponentsResponse], error) {
	s.mu.Lock()
	s.listReqs = append(s.listReqs, req.Msg)
	s.mu.Unlock()
	if s.listErr != nil {
		return nil, s.listErr
	}
	if s.listResp == nil {
		s.listResp = &componentsv1.ListComponentsResponse{}
	}
	return connect.NewResponse(s.listResp), nil
}

func (s *componentsService) GetComponent(_ context.Context, req *connect.Request[componentsv1.GetComponentRequest]) (*connect.Response[componentsv1.GetComponentResponse], error) {
	s.mu.Lock()
	s.getReqs = append(s.getReqs, req.Msg.Id)
	s.mu.Unlock()
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.getResp == nil {
		s.getResp = &componentsv1.GetComponentResponse{}
	}
	return connect.NewResponse(s.getResp), nil
}

func (s *componentsService) GetComponentByLibraryId(_ context.Context, req *connect.Request[componentsv1.GetComponentByLibraryIdRequest]) (*connect.Response[componentsv1.GetComponentByLibraryIdResponse], error) {
	s.mu.Lock()
	s.byLibIDReqs = append(s.byLibIDReqs, req.Msg.LibraryId)
	s.mu.Unlock()
	if s.byLibIDErr != nil {
		return nil, s.byLibIDErr
	}
	if s.byLibIDResp == nil {
		s.byLibIDResp = &componentsv1.GetComponentByLibraryIdResponse{}
	}
	return connect.NewResponse(s.byLibIDResp), nil
}

func (s *componentsService) IndexComponents(_ context.Context, _ *connect.Request[componentsv1.IndexComponentsRequest]) (*connect.Response[componentsv1.IndexComponentsResponse], error) {
	if s.indexErr != nil {
		return nil, s.indexErr
	}
	if s.indexResp == nil {
		s.indexResp = &componentsv1.IndexComponentsResponse{}
	}
	return connect.NewResponse(s.indexResp), nil
}

func (s *componentsService) GetComponentContent(_ context.Context, _ *connect.Request[componentsv1.GetComponentContentRequest]) (*connect.Response[componentsv1.GetComponentContentResponse], error) {
	if s.contentGetErr != nil {
		return nil, s.contentGetErr
	}
	if s.contentGetResp == nil {
		s.contentGetResp = &componentsv1.GetComponentContentResponse{}
	}
	return connect.NewResponse(s.contentGetResp), nil
}

func (s *componentsService) UpdateComponentContent(_ context.Context, req *connect.Request[componentsv1.UpdateComponentContentRequest]) (*connect.Response[componentsv1.UpdateComponentContentResponse], error) {
	s.mu.Lock()
	s.contentSetReqs = append(s.contentSetReqs, req.Msg)
	s.mu.Unlock()
	if s.contentSetErr != nil {
		return nil, s.contentSetErr
	}
	if s.contentSetResp == nil {
		s.contentSetResp = &componentsv1.UpdateComponentContentResponse{}
	}
	return connect.NewResponse(s.contentSetResp), nil
}

func (s *componentsService) InitializeComponent(_ context.Context, req *connect.Request[componentsv1.InitializeComponentRequest]) (*connect.Response[componentsv1.InitializeComponentResponse], error) {
	s.mu.Lock()
	s.initReqs = append(s.initReqs, req.Msg)
	s.mu.Unlock()
	if s.initErr != nil {
		return nil, s.initErr
	}
	if s.initResp == nil {
		s.initResp = &componentsv1.InitializeComponentResponse{Component: sampleComponent()}
	}
	return connect.NewResponse(s.initResp), nil
}

func (s *componentsService) CreateComponentVersion(_ context.Context, req *connect.Request[componentsv1.CreateComponentVersionRequest]) (*connect.Response[componentsv1.CreateComponentVersionResponse], error) {
	s.mu.Lock()
	s.versionReqs = append(s.versionReqs, req.Msg)
	s.mu.Unlock()
	if s.versionErr != nil {
		return nil, s.versionErr
	}
	if s.versionResp == nil {
		s.versionResp = &componentsv1.CreateComponentVersionResponse{Version: &componentsv1.ComponentVersion{Version: req.Msg.Version}}
	}
	return connect.NewResponse(s.versionResp), nil
}

func (s *componentsService) UpdateComponentManifest(_ context.Context, req *connect.Request[componentsv1.UpdateComponentManifestRequest]) (*connect.Response[componentsv1.UpdateComponentManifestResponse], error) {
	s.mu.Lock()
	s.manifestReqs = append(s.manifestReqs, req.Msg)
	s.mu.Unlock()
	if s.manifestErr != nil {
		return nil, s.manifestErr
	}
	if s.manifestResp == nil {
		s.manifestResp = &componentsv1.UpdateComponentManifestResponse{Component: sampleComponent()}
	}
	return connect.NewResponse(s.manifestResp), nil
}

func (s *componentsService) ListComponentVersions(_ context.Context, _ *connect.Request[componentsv1.ListComponentVersionsRequest]) (*connect.Response[componentsv1.ListComponentVersionsResponse], error) {
	return connect.NewResponse(&componentsv1.ListComponentVersionsResponse{}), nil
}

func (s *componentsService) GetComponentVersionContent(_ context.Context, _ *connect.Request[componentsv1.GetComponentVersionContentRequest]) (*connect.Response[componentsv1.GetComponentVersionContentResponse], error) {
	return connect.NewResponse(&componentsv1.GetComponentVersionContentResponse{}), nil
}

func (s *componentsService) ListComponentExamples(_ context.Context, req *connect.Request[componentsv1.ListComponentExamplesRequest]) (*connect.Response[componentsv1.ListComponentExamplesResponse], error) {
	s.mu.Lock()
	s.examplesReqs = append(s.examplesReqs, req.Msg)
	s.mu.Unlock()
	if s.examplesErr != nil {
		return nil, s.examplesErr
	}
	if s.examplesResp == nil {
		s.examplesResp = &componentsv1.ListComponentExamplesResponse{}
	}
	return connect.NewResponse(s.examplesResp), nil
}

func (s *componentsService) ListDesignStyles(_ context.Context, _ *connect.Request[componentsv1.ListDesignStylesRequest]) (*connect.Response[componentsv1.ListDesignStylesResponse], error) {
	return connect.NewResponse(&componentsv1.ListDesignStylesResponse{
		Styles: []*componentsv1.DesignStyle{
			{Id: "vrooli-default", Name: "Vrooli Operational Console", Supports: []string{"templates/scenarios/react-vite"}},
		},
	}), nil
}

func (s *componentsService) ValidateStyleFit(_ context.Context, req *connect.Request[componentsv1.ValidateStyleFitRequest]) (*connect.Response[componentsv1.ValidateStyleFitResponse], error) {
	s.mu.Lock()
	s.styleFitReqs = append(s.styleFitReqs, req.Msg)
	s.mu.Unlock()
	if s.styleFitErr != nil {
		return nil, s.styleFitErr
	}
	if s.styleFitResp == nil {
		s.styleFitResp = &componentsv1.ValidateStyleFitResponse{
			Kind:          componentsv1.StyleFitVerdictKind_STYLE_FIT_VERDICT_KIND_OK,
			ComponentId:   req.Msg.ComponentId,
			Version:       req.Msg.Version,
			Scenario:      req.Msg.Scenario,
			ScenarioStyle: "vrooli-default",
			Affinity:      componentsv1.DesignAffinity_DESIGN_AFFINITY_NATIVE,
			Detail:        "component is native to the scenario design style",
		}
	}
	return connect.NewResponse(s.styleFitResp), nil
}

func connectAPI(t *testing.T, svc *componentsService) http.Handler {
	t.Helper()
	path, handler := componentsconnect.NewComponentsServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func sampleComponent() *componentsv1.Component {
	ts := timestamppb.New(time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC))
	return &componentsv1.Component{
		Id:          "abc",
		LibraryId:   "lib:Button",
		DisplayName: "Button",
		Description: "CTA",
		Slot:        "ui-primitive",
		SourcePath:  "components/Button.tsx",
		Version:     "1.0.0",
		Tags:        []string{"form"},
		DesignStyles: []*componentsv1.ComponentDesignAffinity{
			{StyleId: "vrooli-default", Affinity: componentsv1.DesignAffinity_DESIGN_AFFINITY_NATIVE},
			{StyleId: "vrooli-conversion-landing", Affinity: componentsv1.DesignAffinity_DESIGN_AFFINITY_DISCOURAGED},
		},
		IndexedAt: ts,
		UpdatedAt: ts,
	}
}

func TestComponentsIndex_HumanReport(t *testing.T) {
	svc := &componentsService{indexResp: &componentsv1.IndexComponentsResponse{
		Scanned: 3, Indexed: 2, Skipped: 1, Deleted: 0,
		LibraryIds: []string{"lib:Button", "lib:Card"},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})

	require.NoError(t, h.index(ctx))
	body := out.String()
	require.Contains(t, body, "Scanned 3 file(s); indexed 2, skipped 1, deleted 0.")
	require.Contains(t, body, "lib:Button")
	require.Contains(t, body, "lib:Card")
}

func TestComponentsIngest_ForwardsOriginAndRendersFindings(t *testing.T) {
	svc := &componentsService{ingestResp: &componentsv1.IngestComponentResponse{
		Component: sampleComponent(), ManifestPath: "components/drawer-shell/component.json",
		SourcePath:   "components/drawer-shell/versions/0.1.0-draft.1/drawer-shell.tsx",
		DraftVersion: "0.1.0-draft.1", ChecklistPath: "docs/guides/de-scenario-ification-checklist.md",
		Findings: []*componentsv1.IngestFinding{{Code: "token-violation", Message: "Use tokens."}},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "scenario"}, {Name: "source-file"}, {Name: "slug"}},
		Flags:       []cliapp.Flag{{Name: "display-name"}, {Name: "description"}, {Name: "tags"}, {Name: "slot"}, {Name: "version"}, {Name: "companion-files"}, {Name: "experience-contract"}, {Name: "accept-behavior-loss"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"scenario": "web-console", "source-file": "ui/src/components/DrawerShell.tsx", "slug": "drawer-shell"},
		Flags:       map[string]string{"display-name": "Drawer Shell", "tags": "overlay,layout", "slot": "ui-pattern", "companion-files": "ui/src/components/useFocusTrap.ts", "experience-contract": "experience/components/drawer-shell.json"},
	})

	require.NoError(t, h.ingest(ctx))
	require.Len(t, svc.ingestReqs, 1)
	require.Equal(t, "web-console", svc.ingestReqs[0].Scenario)
	require.Equal(t, []string{"overlay", "layout"}, svc.ingestReqs[0].Tags)
	require.Equal(t, []string{"ui/src/components/useFocusTrap.ts"}, svc.ingestReqs[0].SourceFiles)
	require.Equal(t, "experience/components/drawer-shell.json", svc.ingestReqs[0].ExperienceContractPath)
	require.Contains(t, out.String(), "draft 0.1.0-draft.1")
	require.Contains(t, out.String(), "token-violation")
	require.False(t, svc.ingestReqs[0].AcceptBehaviorLoss)
}

func TestComponentsIngest_BlockedHarvestNamesLossAndPointsAtOverride(t *testing.T) {
	svc := &componentsService{ingestErr: connect.NewError(connect.CodeFailedPrecondition,
		errors.New("harvest of web-console:ui/src/components/DrawerShell.tsx drops 1 origin behavior signal(s): harvest removed event-listener signal addEventListener"))}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "scenario"}, {Name: "source-file"}, {Name: "slug"}},
		Flags:       []cliapp.Flag{{Name: "display-name"}, {Name: "description"}, {Name: "tags"}, {Name: "slot"}, {Name: "version"}, {Name: "companion-files"}, {Name: "experience-contract"}, {Name: "accept-behavior-loss"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"scenario": "web-console", "source-file": "ui/src/components/DrawerShell.tsx", "slug": "drawer-shell"},
	})

	err := h.ingest(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "addEventListener")
	require.Contains(t, err.Error(), "--accept-behavior-loss")
}

func TestComponentsIngest_AcceptBehaviorLossForwardsOverride(t *testing.T) {
	svc := &componentsService{ingestResp: &componentsv1.IngestComponentResponse{
		Component: sampleComponent(), ManifestPath: "components/drawer-shell/component.json",
		SourcePath:   "components/drawer-shell/versions/0.1.0-draft.1/drawer-shell.tsx",
		DraftVersion: "0.1.0-draft.1", ChecklistPath: "docs/guides/de-scenario-ification-checklist.md",
		ParityReport: &componentsv1.IngestParityReport{Acknowledged: true, Findings: []*componentsv1.IngestFinding{{Code: "behavior-lost", Message: "harvest removed event-listener signal addEventListener"}}},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "scenario"}, {Name: "source-file"}, {Name: "slug"}},
		Flags:       []cliapp.Flag{{Name: "display-name"}, {Name: "description"}, {Name: "tags"}, {Name: "slot"}, {Name: "version"}, {Name: "companion-files"}, {Name: "experience-contract"}, {Name: "accept-behavior-loss"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"scenario": "web-console", "source-file": "ui/src/components/DrawerShell.tsx", "slug": "drawer-shell"},
		Flags:       map[string]string{"accept-behavior-loss": "true"},
	})

	require.NoError(t, h.ingest(ctx))
	require.Len(t, svc.ingestReqs, 1)
	require.True(t, svc.ingestReqs[0].AcceptBehaviorLoss)
	require.Contains(t, out.String(), "Accepted 1 behavior-loss finding(s)")
}

func TestComponentsList_ForwardsFiltersAndRenders(t *testing.T) {
	svc := &componentsService{listResp: &componentsv1.ListComponentsResponse{
		Components: []*componentsv1.Component{sampleComponent()},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "match"}, {Name: "tag"}, {Name: "tags"}, {Name: "category"}, {Name: "style"}, {Name: "affinity"}, {Name: "asset-kind"}, {Name: "limit"}},
	}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"match": "btn", "tag": "form", "style": "vrooli-default", "affinity": "native", "asset-kind": "hook", "limit": "50"},
	})

	require.NoError(t, h.list(ctx))
	require.Len(t, svc.listReqs, 1)
	require.Equal(t, "btn", svc.listReqs[0].Match)
	require.Equal(t, "form", svc.listReqs[0].Tag)
	require.Equal(t, int32(50), svc.listReqs[0].Limit)
	require.Empty(t, svc.listReqs[0].Tags)
	require.Empty(t, svc.listReqs[0].Category)
	require.Equal(t, "vrooli-default", svc.listReqs[0].StyleId)
	require.Equal(t, "native", svc.listReqs[0].Affinity)
	require.Equal(t, componentsv1.AssetKind_ASSET_KIND_HOOK, svc.listReqs[0].AssetKind)
	require.Contains(t, out.String(), "Found 1 component(s).")
	require.Contains(t, out.String(), "lib:Button")
	require.Contains(t, out.String(), "v1.0.0")
	require.Contains(t, out.String(), "vrooli-conversion-landing:discouraged")
}

func TestComponentsStyles_RendersCanonicalStyles(t *testing.T) {
	svc := &componentsService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})

	require.NoError(t, h.styles(ctx))
	body := out.String()
	require.Contains(t, body, "Found 1 design style(s).")
	require.Contains(t, body, "vrooli-default")
	require.Contains(t, body, "templates/scenarios/react-vite")
}

func TestComponentsStyleFit_ForwardsInputsAndRendersVerdict(t *testing.T) {
	svc := &componentsService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "component-id", Required: true}, {Name: "scenario", Required: true}},
		Flags:       []cliapp.Flag{{Name: "version"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"component-id": "cmp-1", "scenario": "target-app"},
		Flags:       map[string]string{"version": "1.0.0"},
	})

	require.NoError(t, h.validateStyleFit(ctx))
	require.Len(t, svc.styleFitReqs, 1)
	require.Equal(t, "cmp-1", svc.styleFitReqs[0].ComponentId)
	require.Equal(t, "target-app", svc.styleFitReqs[0].Scenario)
	require.Equal(t, "1.0.0", svc.styleFitReqs[0].Version)
	body := out.String()
	require.Contains(t, body, "Style fit is ok for scenario target-app.")
	require.Contains(t, body, "style=vrooli-default")
	require.Contains(t, body, "component is native")
}

func TestComponentsExamples_ForwardsInputsAndRenders(t *testing.T) {
	svc := &componentsService{examplesResp: &componentsv1.ListComponentExamplesResponse{
		Examples: []*componentsv1.ComponentExample{
			{
				Name:       "primary",
				LibraryId:  "lib:Button",
				Version:    "1.0.0",
				PropsJson:  `{"children":{"$text":"Save changes"}}`,
				SourcePath: "components/Button/versions/1.0.0/examples.json",
			},
		},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "component-id", Required: true}},
		Flags:       []cliapp.Flag{{Name: "version"}, {Name: "limit"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"component-id": "cmp-1"},
		Flags:       map[string]string{"version": "1.0.0", "limit": "10"},
	})

	require.NoError(t, h.examples(ctx))
	require.Len(t, svc.examplesReqs, 1)
	require.Equal(t, "cmp-1", svc.examplesReqs[0].ComponentId)
	require.Equal(t, "1.0.0", svc.examplesReqs[0].Version)
	require.Equal(t, int32(10), svc.examplesReqs[0].Limit)
	body := out.String()
	require.Contains(t, body, "Found 1 example(s).")
	require.Contains(t, body, "primary")
	require.Contains(t, body, "Save changes")
}

func TestComponentsList_ForwardsMultiTagAndCategory(t *testing.T) {
	svc := &componentsService{listResp: &componentsv1.ListComponentsResponse{
		Components: []*componentsv1.Component{sampleComponent()},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "match"}, {Name: "tag"}, {Name: "tags"}, {Name: "category"}, {Name: "style"}, {Name: "affinity"}, {Name: "asset-kind"}, {Name: "limit"}},
	}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"tags": " form , , layout ", "category": "controls"},
	})

	require.NoError(t, h.list(ctx))
	require.Len(t, svc.listReqs, 1)
	require.Equal(t, []string{"form", "layout"}, svc.listReqs[0].Tags,
		"comma-separated --tags is parsed and trimmed; blanks dropped")
	require.Equal(t, "controls", svc.listReqs[0].Category)
}

func TestComponentsList_RejectsBadLimit(t *testing.T) {
	core := clitest.NewTestApp(t, connectAPI(t, &componentsService{}))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "match"}, {Name: "tag"}, {Name: "tags"}, {Name: "category"}, {Name: "style"}, {Name: "affinity"}, {Name: "asset-kind"}, {Name: "limit"}},
	}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"limit": "abc"},
	})
	err := h.list(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "--limit must be an integer")
}

// TestComponentsList_JSONIsProtoWireShape pins the contract that --json output
// is the proto-typed ListComponentsResponse wire shape.
func TestComponentsList_JSONIsProtoWireShape(t *testing.T) {
	svc := &componentsService{listResp: &componentsv1.ListComponentsResponse{
		Components: []*componentsv1.Component{sampleComponent()},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "match"}, {Name: "tag"}, {Name: "tags"}, {Name: "category"}, {Name: "style"}, {Name: "affinity"}, {Name: "asset-kind"}, {Name: "limit"}},
	}, cliapptest.TestRunContextOptions{JSON: true})

	require.NoError(t, h.list(ctx))

	body := out.String()
	require.NotContains(t, body, "summary",
		"--json output must be proto wire shape, not the human ListReport wrapper")
	require.NotContains(t, body, "retrieval_hints")

	var got componentsv1.ListComponentsResponse
	require.NoError(t, protojson.Unmarshal(out.Bytes(), &got))
	require.Len(t, got.Components, 1)
	require.Equal(t, "lib:Button", got.Components[0].LibraryId)
}

func TestComponentsGetByLibraryID_Fetches(t *testing.T) {
	svc := &componentsService{byLibIDResp: &componentsv1.GetComponentByLibraryIdResponse{
		Component: sampleComponent(),
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "library-id"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"library-id": "lib:Button"},
	})
	require.NoError(t, h.getByLibraryID(ctx))
	require.Equal(t, []string{"lib:Button"}, svc.byLibIDReqs)
	require.Contains(t, out.String(), "Fetched component lib:Button.")
}

func TestComponentsContentGet_PrintsBody(t *testing.T) {
	svc := &componentsService{contentGetResp: &componentsv1.GetComponentContentResponse{
		Content:    "// hello",
		SourcePath: "Button.tsx",
		Sha256:     "abc123",
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "id", Required: true}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"id": "abc"},
	})

	require.NoError(t, h.contentGet(ctx))
	body := out.String()
	require.Contains(t, body, "// hello")
	require.Contains(t, body, "sha256=abc123")
}

func TestComponentsContentSet_FromFile(t *testing.T) {
	svc := &componentsService{contentSetResp: &componentsv1.UpdateComponentContentResponse{
		Sha256:     "deadbeef",
		SourcePath: "Button.tsx",
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)

	tmpFile := t.TempDir() + "/new.tsx"
	require.NoError(t, writeFile(tmpFile, "// rewritten\n"))

	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "id", Required: true}, {Name: "file", Required: true}},
		Flags:       []cliapp.Flag{{Name: "expected-sha256"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"id": "abc", "file": tmpFile},
		Flags:       map[string]string{"expected-sha256": "stale"},
	})
	require.NoError(t, h.contentSet(ctx))
	require.Len(t, svc.contentSetReqs, 1)
	require.Equal(t, "abc", svc.contentSetReqs[0].Id)
	require.Equal(t, "// rewritten\n", svc.contentSetReqs[0].Content)
	require.Equal(t, "stale", svc.contentSetReqs[0].ExpectedSha256)
	require.Contains(t, out.String(), "sha256=deadbeef")
}

func TestComponentsInit_ForwardsAuthoringFields(t *testing.T) {
	svc := &componentsService{initResp: &componentsv1.InitializeComponentResponse{
		Component:    sampleComponent(),
		ManifestPath: "components/Header/component.json",
		SourcePath:   "components/Header/versions/0.1.0/Header.tsx",
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "slug", Required: true}},
		Flags: []cliapp.Flag{
			{Name: "library-id"},
			{Name: "display-name"},
			{Name: "description"},
			{Name: "tags"},
			{Name: "version"},
			{Name: "file-name"},
			{Name: "source-file"},
		},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"slug": "Header"},
		Flags: map[string]string{
			"library-id":   "react-component-library:Header",
			"display-name": "Header",
			"description":  "Scenario header",
			"tags":         "layout, navigation",
			"version":      "0.1.0",
			"file-name":    "Header.tsx",
		},
	})

	require.NoError(t, h.init(ctx))
	require.Len(t, svc.initReqs, 1)
	require.Equal(t, "Header", svc.initReqs[0].Slug)
	require.Equal(t, []string{"layout", "navigation"}, svc.initReqs[0].Tags)
	require.Contains(t, out.String(), "components/Header/component.json")
}

func TestComponentsVersionCreate_ForwardsIntent(t *testing.T) {
	svc := &componentsService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "component-id", Required: true}, {Name: "version", Required: true}},
		Flags: []cliapp.Flag{
			{Name: "from-version"},
			{Name: "draft"},
			{Name: "release"},
			{Name: "file-name"},
			{Name: "source-file"},
			{Name: "acknowledge-parity-waiver"},
			{Name: "changelog"},
		},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"component-id": "cmp-1", "version": "0.2.0-beta.1"},
		Flags:       map[string]string{"draft": "true", "from-version": "0.1.0"},
	})

	require.NoError(t, h.versionCreate(ctx))
	require.Len(t, svc.versionReqs, 1)
	require.Equal(t, "cmp-1", svc.versionReqs[0].ComponentId)
	require.Equal(t, "0.2.0-beta.1", svc.versionReqs[0].Version)
	require.Equal(t, componentsv1.ComponentVersionIntent_COMPONENT_VERSION_INTENT_DRAFT, svc.versionReqs[0].Intent)
	require.Contains(t, out.String(), "Created version 0.2.0-beta.1.")
}

func TestComponentsManifestUpdate_ForwardsMetadata(t *testing.T) {
	svc := &componentsService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "component-id", Required: true}},
		Flags: []cliapp.Flag{
			{Name: "display-name"},
			{Name: "description"},
			{Name: "tags"},
			{Name: "latest-version"},
			{Name: "draft-version"},
			{Name: "deprecated-versions"},
		},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"component-id": "cmp-1"},
		Flags: map[string]string{
			"display-name":        "Header",
			"description":         "Updated",
			"tags":                "layout,nav",
			"latest-version":      "1.0.0",
			"deprecated-versions": "0.1.0",
		},
	})

	require.NoError(t, h.manifestUpdate(ctx))
	require.Len(t, svc.manifestReqs, 1)
	require.Equal(t, []string{"layout", "nav"}, svc.manifestReqs[0].Tags)
	require.Equal(t, []string{"0.1.0"}, svc.manifestReqs[0].DeprecatedVersions)
}

func writeFile(path, body string) error {
	return os.WriteFile(path, []byte(body), 0o600)
}

func TestComponentsGet_ReportsNotFound(t *testing.T) {
	svc := &componentsService{getErr: connect.NewError(connect.CodeNotFound, io.ErrUnexpectedEOF)}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "id", Required: true}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"id": "ghost"},
	})

	err := h.get(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not_found")
}
