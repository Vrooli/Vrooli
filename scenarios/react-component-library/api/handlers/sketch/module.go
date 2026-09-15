package sketch

import (
	"context"
	"fmt"
	"log"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	captureH "react-component-library/handlers/designcapture"
	"react-component-library/internal/availability"
	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/catalogsearch"
	"react-component-library/internal/components"
	"react-component-library/internal/designcapture"
	"react-component-library/internal/designcritique"
	"react-component-library/internal/designinference"
	"react-component-library/internal/module"
	"react-component-library/internal/reconcile"
	internal "react-component-library/internal/sketch"

	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"
	sketchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch/sketch_v1connect"
)

var ProtoFile = sketchv1.File_react_component_library_v1_sketch_sketch_proto

func Module(repoRoot string, logger *log.Logger, scanner reconcile.Scanner, routes ...*filerouting.RoutedRoots) module.Module {
	return buildModule(repoRoot, logger, scanner, routes, nil, nil, nil)
}
func ModuleWithAvailability(repoRoot string, logger *log.Logger, scanner reconcile.Scanner, routes *filerouting.RoutedRoots, factory func(context.Context) (*availability.Snapshot, error), assets components.DependencyReader, renderers ...CompositionRenderer) module.Module {
	var renderer CompositionRenderer
	if len(renderers) > 0 {
		renderer = renderers[0]
	}
	return buildModule(repoRoot, logger, scanner, []*filerouting.RoutedRoots{routes}, factory, renderer, assets)
}

type CaptureConfig struct {
	InferenceFor  func(context.Context) (designinference.Repository, error)
	CritiquesFor  func(context.Context) (designcritique.Repository, error)
	RepositoryFor func(context.Context) (designcapture.Repository, error)
	Dispatcher    designcapture.Dispatcher
}

func ModuleWithCapture(repoRoot string, logger *log.Logger, scanner reconcile.Scanner, routes *filerouting.RoutedRoots, factory func(context.Context) (*availability.Snapshot, error), assets components.DependencyReader, renderer CompositionRenderer, captures CaptureConfig) module.Module {
	return buildModule(repoRoot, logger, scanner, []*filerouting.RoutedRoots{routes}, factory, renderer, assets, captures)
}
func buildModule(repoRoot string, logger *log.Logger, scanner reconcile.Scanner, routes []*filerouting.RoutedRoots, factory func(context.Context) (*availability.Snapshot, error), renderer CompositionRenderer, assets components.DependencyReader, captures ...CaptureConfig) module.Module {
	resolver := &reconcile.Resolver{ToolingRoot: repoRoot, ScenariosRoot: filepath.Join(repoRoot, "scenarios"), Scanner: scanner}
	deps := Deps{Assets: assets, Renderer: renderer, Store: internal.NewStore(repoRoot), Reconciler: resolver, Logger: logger, Availability: factory}
	if len(captures) > 0 {
		deps.InferenceFor = captures[0].InferenceFor
		deps.CritiquesFor = captures[0].CritiquesFor
		deps.CapturesFor = captures[0].RepositoryFor
		deps.CaptureDispatcher = captures[0].Dispatcher
	}
	deps.Catalog = func(ctx context.Context) ([]catalogcoverage.Asset, error) {
		return catalogcoverage.LoadCatalogContext(ctx, filepath.Join(repoRoot, "scenarios", "react-component-library", "catalog"))
	}
	deps.ImportSearch = func(ctx context.Context) (*catalogsearch.Index, error) {
		index := catalogsearch.NewWithAvailability(factory)
		if factory == nil {
			index = catalogsearch.New()
		}
		err := index.ReindexContext(ctx, filepath.Join(repoRoot, "scenarios", "react-component-library"))
		return index, err
	}
	if len(routes) > 0 && routes[0] != nil {
		deps.StoreFor = routedRepository(repoRoot, routes[0])
		deps.RecordWrite = routes[0].RecordWrite
		deps.ReconcilerFor = func(ctx context.Context) (*reconcile.Resolver, error) {
			root, err := requestWorkspaceRoot(ctx, repoRoot, routes[0])
			if err != nil {
				return nil, err
			}
			if !database.IsTestMode(ctx) {
				return resolver, nil
			}
			factory, ok := scanner.(interface {
				ForWorkspace(string) reconcile.Scanner
			})
			if !ok {
				return nil, fmt.Errorf("inventory scanner does not support a routed workspace")
			}
			routed := factory.ForWorkspace(root)
			if routed == nil {
				return nil, fmt.Errorf("inventory scanner has no routed workspace factory")
			}
			return &reconcile.Resolver{ToolingRoot: repoRoot, ScenariosRoot: filepath.Join(root, "scenarios"), Scanner: routed}, nil
		}
	}
	connectPath, handler := sketchconnect.NewSketchServiceHandler(NewConnectHandler(deps))
	return module.Module{
		Name: "sketch",
		Mount: func(router *mux.Router) {
			if deps.CapturesFor != nil {
				captureH.MountTarget(router, deps.CapturesFor)
			}
			connectx.RegisterServices(router, connectx.ServiceMount{Path: connectPath, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }

var Endpoints = []module.EndpointDescriptor{
	{ID: "sketch_infer", Path: sketchconnect.SketchServiceInferSketchProcedure, Method: "POST", Summary: "Dispatch governed typed design alternatives with durable idempotency", Category: "sketch"},
	{ID: "sketch_inference_get", Path: sketchconnect.SketchServiceGetSketchInferenceProcedure, Method: "POST", Summary: "Read an existing design inference without redispatch", Category: "sketch"},
	{ID: "candidate_refine", Path: sketchconnect.SketchServiceRefineCandidateProcedure, Method: "POST", Summary: "Refine an exact candidate within a recorded region scope and round budget", Category: "sketch"},
	{ID: "capture_screenshot", Path: sketchconnect.SketchServiceGetCaptureScreenshotProcedure, Method: "POST", Summary: "Resolve a verified capture screenshot through its browser owner", Category: "sketch"},
	{ID: "capture_retry", Path: sketchconnect.SketchServiceRetryCaptureProcedure, Method: "POST", Summary: "Start a new linked attempt of a terminal capture", Category: "sketch"},
	{ID: "critique_list", Path: sketchconnect.SketchServiceListCritiquesProcedure, Method: "POST", Summary: "List verified critiques of an exact candidate render", Category: "sketch"},
	{ID: "critique_rubric", Path: sketchconnect.SketchServiceGetCritiqueRubricProcedure, Method: "POST", Summary: "Read the versioned visual rubric and calibration status", Category: "sketch"},
	{ID: "critique_record", Path: sketchconnect.SketchServiceRecordCritiqueProcedure, Method: "POST", Summary: "Verify image evidence and record an attributed visual critique", Category: "sketch"},
	{ID: "critique_get", Path: sketchconnect.SketchServiceGetCritiqueProcedure, Method: "POST", Summary: "Read an immutable visual review fact", Category: "sketch"},
	{ID: "capture_cancel", Path: sketchconnect.SketchServiceCancelCaptureProcedure, Method: "POST", Summary: "Request cancellation of the existing capture producer", Category: "sketch"},
	{ID: "capture_attach", Path: sketchconnect.SketchServiceAttachCaptureProcedure, Method: "POST", Summary: "Verify and attach producer-owned capture evidence", Category: "sketch"},
	{ID: "candidate_capture", Path: sketchconnect.SketchServiceCaptureCandidateProcedure, Method: "POST", Summary: "Capture an exact candidate render through BAS", Category: "sketch"},
	{ID: "capture_get", Path: sketchconnect.SketchServiceGetCaptureProcedure, Method: "POST", Summary: "Read durable capture operation identity and state", Category: "sketch"},
	{ID: "candidate_list", Path: sketchconnect.SketchServiceListCandidatesProcedure, Method: "POST", Summary: "List verified saved candidate revisions for a page", Category: "sketch"},
	{ID: "candidate_place_asset", Path: sketchconnect.SketchServicePlaceCandidateAssetProcedure, Method: "POST", Summary: "Derive a candidate with an exact published region asset and story fixture", Category: "sketch"},
	{ID: "candidate_map_regions", Path: sketchconnect.SketchServiceMapCandidateRegionsProcedure, Method: "POST", Summary: "Derive an immutable candidate with explicit semantic port mappings", Category: "sketch"},
	{ID: "candidate_save", Path: sketchconnect.SketchServiceSaveCandidateProcedure, Method: "POST", Summary: "Persist an immutable candidate without changing the page", Category: "sketch"},
	{ID: "candidate_get", Path: sketchconnect.SketchServiceGetCandidateProcedure, Method: "POST", Summary: "Read an exact immutable candidate", Category: "sketch"},
	{ID: "candidate_render", Path: sketchconnect.SketchServiceRenderCandidateProcedure, Method: "POST", Summary: "Render an immutable candidate independently of current page edits", Category: "sketch"},
	{ID: "sketch_propose", Path: sketchconnect.SketchServiceProposeSketchProcedure, Method: "POST", Summary: "Propose bounded catalog-backed page alternatives", Category: "sketch"},
	{ID: "sketch_render", Path: sketchconnect.SketchServiceRenderSketchProcedure, Method: "POST", Summary: "Render an exact authored design revision", Category: "sketch"},
	{ID: "design_pages", Path: sketchconnect.SketchServiceListDesignPagesProcedure, Method: "POST", Summary: "Discover scenario design pages and contract gaps", Category: "sketch"},
	{ID: "sketch_history", Path: sketchconnect.SketchServiceGetHistoryProcedure, Method: "POST", Summary: "Read immutable design revisions", Category: "sketch"},
	{ID: "sketch_recover", Path: sketchconnect.SketchServiceRecoverSketchProcedure, Method: "POST", Summary: "Recover an interrupted page apply", Category: "sketch"},
	{ID: "sketch_get", Path: sketchconnect.SketchServiceGetSketchProcedure, Method: "POST", Summary: "Read a scenario page sketch", Category: "sketch"},
	{ID: "sketch_put", Path: sketchconnect.SketchServicePutSketchProcedure, Method: "POST", Summary: "Write a scenario page sketch", Category: "sketch"},
	{ID: "sketch_verify", Path: sketchconnect.SketchServiceVerifySketchProcedure, Method: "POST", Summary: "Verify page regions against observed source", Category: "sketch"},
	{ID: "sketch_import", Path: sketchconnect.SketchServiceImportPageProcedure, Method: "POST", Summary: "Decompose a page through the catalog", Category: "sketch"},
	{ID: "sketch_place", Path: sketchconnect.SketchServicePlaceProcedure, Method: "POST", Summary: "Place a catalog asset into a region", Category: "sketch"},
	{ID: "sketch_placeholder", Path: sketchconnect.SketchServicePlaceholderProcedure, Method: "POST", Summary: "Record a genuinely new region need", Category: "sketch"},
	{ID: "sketch_note", Path: sketchconnect.SketchServiceAddNoteProcedure, Method: "POST", Summary: "Add a scoped sketch note", Category: "sketch"},
	{ID: "sketch_unplace", Path: sketchconnect.SketchServiceUnplaceProcedure, Method: "POST", Summary: "Move a placement to the unplaced set", Category: "sketch"},
	{ID: "sketch_template", Path: sketchconnect.SketchServiceSetTemplateProcedure, Method: "POST", Summary: "Swap a page template with an explicit remap", Category: "sketch"},
	{ID: "sketch_brief", Path: sketchconnect.SketchServiceBuildBriefProcedure, Method: "POST", Summary: "Build an ordered implementation brief", Category: "sketch"},
}

// Authored live pages stay in the workspace. Test-mode requests must use an
// installed config lease containing workspace/scenarios/... fixtures. Missing
// or expired leases are refused; they never fall back to live authored files.
func requestWorkspaceRoot(ctx context.Context, repoRoot string, roots *filerouting.RoutedRoots) (string, error) {
	if !database.IsTestMode(ctx) {
		return repoRoot, nil
	}
	selected, err := roots.Pick(ctx, storage.ClassConfig)
	if err != nil {
		return "", err
	}
	primary, err := roots.Pick(context.Background(), storage.ClassConfig)
	if err != nil {
		return "", err
	}
	if selected == primary {
		return "", fmt.Errorf("test-mode sketch access requires an active isolated file lease")
	}
	return filepath.Join(selected, "workspace"), nil
}
func routedRepository(repoRoot string, roots *filerouting.RoutedRoots) func(context.Context) (internal.Repository, error) {
	return func(ctx context.Context) (internal.Repository, error) {
		root, err := requestWorkspaceRoot(ctx, repoRoot, roots)
		if err != nil {
			return nil, err
		}
		return internal.NewStore(root), nil
	}
}
