package sketch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	captureH "react-component-library/handlers/designcapture"
	"react-component-library/internal/availability"
	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/components"
	"react-component-library/internal/designcapture"
	"react-component-library/internal/designcritique"
	"react-component-library/internal/designinference"
	"react-component-library/internal/module"
	"react-component-library/internal/reconcile"
	internalsearch "react-component-library/internal/search"
	internal "react-component-library/internal/sketch"

	experienceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/experience/experience_v1connect"
	reviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/review"
	workspaceassistconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspaceassist/workspaceassist_v1connect"
	workspacecandidateconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspacecandidate/workspacecandidate_v1connect"
	workspacecaptureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspacecapture/workspacecapture_v1connect"
)

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
	InferenceFor    func(context.Context) (designinference.Repository, error)
	CritiquesFor    func(context.Context) (designcritique.Repository, error)
	RepositoryFor   func(context.Context) (designcapture.Repository, error)
	Dispatcher      designcapture.Dispatcher
	AdoptExperience func(context.Context, string, string, []byte) (*workspacev1.Workspace, string, string, string, string, bool, error)
}

func ModuleWithCapture(repoRoot string, logger *log.Logger, scanner reconcile.Scanner, routes *filerouting.RoutedRoots, factory func(context.Context) (*availability.Snapshot, error), assets components.DependencyReader, renderer CompositionRenderer, captures CaptureConfig) module.Module {
	return buildModule(repoRoot, logger, scanner, []*filerouting.RoutedRoots{routes}, factory, renderer, assets, captures)
}

type AcceptanceEvaluator interface {
	ValidateAcceptanceCandidate(context.Context, *reviewv1.CheckCandidateAcceptanceRequest) error
	EvaluateAcceptance(context.Context, *reviewv1.CheckCandidateAcceptanceRequest) (*reviewv1.CandidateAcceptanceCheck, error)
}

type AcceptanceStore interface {
	RecordAcceptance(context.Context, string, string, internal.AcceptanceIntent, func(context.Context) (internal.AcceptanceFacts, error)) (internal.AcceptanceDecision, error)
	ReadAcceptance(string, string, string) (internal.AcceptanceDecision, error)
}

type AcceptancePorts struct {
	Evaluator    AcceptanceEvaluator
	StoreFor     func(context.Context) (AcceptanceStore, error)
	CritiquesFor func(context.Context) (designcritique.Repository, error)
	CapturesFor  func(context.Context) (designcapture.Repository, error)
	Screenshots  designcapture.ScreenshotResolver
	RecordWrite  func(context.Context)
}

func ModuleWithCaptureAndAcceptance(repoRoot string, logger *log.Logger, scanner reconcile.Scanner, routes *filerouting.RoutedRoots, factory func(context.Context) (*availability.Snapshot, error), assets components.DependencyReader, renderer CompositionRenderer, captures CaptureConfig, catalogSearch internalsearch.CatalogSearch) (module.Module, AcceptancePorts) {
	routed := []*filerouting.RoutedRoots{routes}
	deps := buildDeps(repoRoot, logger, scanner, routed, factory, renderer, assets, captures)
	deps.ImportSearch = catalogSearch
	return moduleFromDeps(repoRoot, logger, routed, deps), acceptancePorts(deps)
}

func buildModule(repoRoot string, logger *log.Logger, scanner reconcile.Scanner, routes []*filerouting.RoutedRoots, factory func(context.Context) (*availability.Snapshot, error), renderer CompositionRenderer, assets components.DependencyReader, captures ...CaptureConfig) module.Module {
	deps := buildDeps(repoRoot, logger, scanner, routes, factory, renderer, assets, captures...)
	return moduleFromDeps(repoRoot, logger, routes, deps)
}

func buildDeps(repoRoot string, logger *log.Logger, scanner reconcile.Scanner, routes []*filerouting.RoutedRoots, factory func(context.Context) (*availability.Snapshot, error), renderer CompositionRenderer, assets components.DependencyReader, captures ...CaptureConfig) Deps {
	resolver := &reconcile.Resolver{ToolingRoot: repoRoot, ScenariosRoot: filepath.Join(repoRoot, "scenarios"), Scanner: scanner}
	deps := Deps{Assets: assets, Renderer: renderer, Store: internal.NewStore(repoRoot), Reconciler: resolver, Logger: logger, Availability: factory}
	if len(captures) > 0 {
		deps.InferenceFor = captures[0].InferenceFor
		deps.CritiquesFor = captures[0].CritiquesFor
		deps.CapturesFor = captures[0].RepositoryFor
		deps.CaptureDispatcher = captures[0].Dispatcher
		deps.AdoptExperience = captures[0].AdoptExperience
	}
	deps.Catalog = func(ctx context.Context) ([]catalogcoverage.Asset, error) {
		return catalogcoverage.LoadCatalogContext(ctx, filepath.Join(repoRoot, "scenarios", "react-component-library", "catalog"))
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
	return deps
}

func acceptancePorts(deps Deps) AcceptancePorts {
	handler := NewConnectHandler(deps)
	ports := AcceptancePorts{Evaluator: handler, CritiquesFor: deps.CritiquesFor, CapturesFor: deps.CapturesFor, RecordWrite: deps.RecordWrite}
	if screenshots, ok := deps.CaptureDispatcher.(designcapture.ScreenshotResolver); ok {
		ports.Screenshots = screenshots
	}
	ports.StoreFor = func(ctx context.Context) (AcceptanceStore, error) {
		request, err := handler.forRequest(ctx)
		if err != nil {
			return nil, err
		}
		store, ok := request.deps.Store.(AcceptanceStore)
		if !ok {
			return nil, connectxUnavailable("acceptance persistence unavailable")
		}
		return store, nil
	}
	return ports
}

func connectxUnavailable(message string) error {
	return connect.NewError(connect.CodeUnavailable, errors.New(message))
}

func moduleFromDeps(repoRoot string, logger *log.Logger, routes []*filerouting.RoutedRoots, deps Deps) module.Module {
	core := NewConnectHandler(deps)
	experiencePath, experienceHandler := experienceconnect.NewExperienceServiceHandler(core)
	capturePath, captureHandler := workspacecaptureconnect.NewWorkspaceCaptureServiceHandler(core)
	assistPath, assistHandler := workspaceassistconnect.NewWorkspaceAssistServiceHandler(workspaceAssistHandler{port: core})
	candidatePath, candidateHandler := workspacecandidateconnect.NewWorkspaceCandidateServiceHandler(workspaceCandidateHandler{port: core})
	return module.Module{
		Name: "sketch",
		Mount: func(router *mux.Router) {
			referenceAssets := &referenceAssetHTTP{repoRoot: repoRoot, routes: routes}
			referenceGeneration := newReferenceGenerationHTTP(repoRoot, logger)
			router.HandleFunc("/reference-assets", referenceAssets.upload).Methods(http.MethodPost)
			router.HandleFunc("/reference-assets/{scenario}/{page}/{name}", referenceAssets.serve).Methods(http.MethodGet)
			router.HandleFunc("/reference-assets/generate", referenceGeneration.generate).Methods(http.MethodPost)
			// REST callers resolve the conventional /api/v1 suffix; retain the
			// short aliases for local tools and existing scenario conventions.
			router.HandleFunc("/api/v1/reference-assets", referenceAssets.upload).Methods(http.MethodPost)
			router.HandleFunc("/api/v1/reference-assets/{scenario}/{page}/{name}", referenceAssets.serve).Methods(http.MethodGet)
			router.HandleFunc("/api/v1/reference-assets/generate", referenceGeneration.generate).Methods(http.MethodPost)
			if deps.CapturesFor != nil {
				captureH.MountTarget(router, deps.CapturesFor)
			}
			connectx.RegisterServices(router,
				connectx.ServiceMount{Path: experiencePath, Handler: experienceHandler},
				connectx.ServiceMount{Path: capturePath, Handler: captureHandler},
				connectx.ServiceMount{Path: assistPath, Handler: assistHandler},
				connectx.ServiceMount{Path: candidatePath, Handler: candidateHandler},
			)
		},
		Endpoints: Endpoints,
	}
}

type referenceAssetHTTP struct {
	repoRoot string
	routes   []*filerouting.RoutedRoots
}

func (h *referenceAssetHTTP) root(ctx context.Context) (string, error) {
	if len(h.routes) == 0 || h.routes[0] == nil {
		return h.repoRoot, nil
	}
	return requestWorkspaceRoot(ctx, h.repoRoot, h.routes[0])
}

func (h *referenceAssetHTTP) upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		http.Error(w, "malformed multipart form", http.StatusBadRequest)
		return
	}
	scenario, page := r.FormValue("scenario"), r.FormValue("page")
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()
	root, err := h.root(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusPreconditionFailed)
		return
	}
	id, name, size, err := internal.SaveReferenceAsset(root, scenario, page, header.Filename, header.Header.Get("Content-Type"), file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"id": id, "name": name, "size": size, "mime": header.Header.Get("Content-Type"), "url": fmt.Sprintf("/api/v1/reference-assets/%s/%s/%s", scenario, page, name)})
}

func (h *referenceAssetHTTP) serve(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	root, err := h.root(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusPreconditionFailed)
		return
	}
	path, err := internal.ReferenceAssetPath(root, vars["scenario"], vars["page"], vars["name"])
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, path)
}

func Schema() string { return "" }

var Endpoints = []module.EndpointDescriptor{
	{ID: "sketch_infer", Path: workspaceassistconnect.WorkspaceAssistServiceInferSketchProcedure, Method: "POST", Summary: "Dispatch governed typed design alternatives with durable idempotency", Category: "sketch"},
	{ID: "sketch_inference_get", Path: workspaceassistconnect.WorkspaceAssistServiceGetSketchInferenceProcedure, Method: "POST", Summary: "Read an existing design inference without redispatch", Category: "sketch"},
	{ID: "candidate_refine", Path: workspaceassistconnect.WorkspaceAssistServiceRefineCandidateProcedure, Method: "POST", Summary: "Refine an exact candidate within a recorded region scope and round budget", Category: "sketch"},
	{ID: "capture_screenshot", Path: workspacecaptureconnect.WorkspaceCaptureServiceGetCaptureScreenshotProcedure, Method: "POST", Summary: "Resolve a verified capture screenshot through its browser owner", Category: "sketch"},
	{ID: "capture_retry", Path: workspacecaptureconnect.WorkspaceCaptureServiceRetryCaptureProcedure, Method: "POST", Summary: "Start a new linked attempt of a terminal capture", Category: "sketch"},
	{ID: "capture_cancel", Path: workspacecaptureconnect.WorkspaceCaptureServiceCancelCaptureProcedure, Method: "POST", Summary: "Request cancellation of the existing capture producer", Category: "sketch"},
	{ID: "capture_attach", Path: workspacecaptureconnect.WorkspaceCaptureServiceAttachCaptureProcedure, Method: "POST", Summary: "Verify and attach producer-owned capture evidence", Category: "sketch"},
	{ID: "candidate_capture", Path: workspacecaptureconnect.WorkspaceCaptureServiceCaptureCandidateProcedure, Method: "POST", Summary: "Capture an exact candidate render through BAS", Category: "sketch"},
	{ID: "capture_get", Path: workspacecaptureconnect.WorkspaceCaptureServiceGetCaptureProcedure, Method: "POST", Summary: "Read durable capture operation identity and state", Category: "sketch"},
	{ID: "candidate_list", Path: workspacecandidateconnect.WorkspaceCandidateServiceListCandidatesProcedure, Method: "POST", Summary: "List verified saved candidate revisions for a page", Category: "sketch"},
	{ID: "candidate_place_asset", Path: workspacecandidateconnect.WorkspaceCandidateServicePlaceCandidateAssetProcedure, Method: "POST", Summary: "Derive a candidate with an exact published region asset and story fixture", Category: "sketch"},
	{ID: "candidate_map_regions", Path: workspacecandidateconnect.WorkspaceCandidateServiceMapCandidateRegionsProcedure, Method: "POST", Summary: "Derive an immutable candidate with explicit semantic port mappings", Category: "sketch"},
	{ID: "candidate_save", Path: workspacecandidateconnect.WorkspaceCandidateServiceSaveCandidateProcedure, Method: "POST", Summary: "Persist an immutable candidate without changing the page", Category: "sketch"},
	{ID: "candidate_get", Path: workspacecandidateconnect.WorkspaceCandidateServiceGetCandidateProcedure, Method: "POST", Summary: "Read an exact immutable candidate", Category: "sketch"},
	{ID: "candidate_render", Path: workspacecandidateconnect.WorkspaceCandidateServiceRenderCandidateProcedure, Method: "POST", Summary: "Render an immutable candidate independently of current page edits", Category: "sketch"},
	{ID: "sketch_propose", Path: workspaceassistconnect.WorkspaceAssistServiceProposeSketchProcedure, Method: "POST", Summary: "Propose bounded catalog-backed page alternatives", Category: "sketch"},
	{ID: "sketch_render", Path: workspacecandidateconnect.WorkspaceCandidateServiceRenderSketchProcedure, Method: "POST", Summary: "Render an exact authored design revision", Category: "sketch"},
	{ID: "design_pages", Path: experienceconnect.ExperienceServiceListDesignPagesProcedure, Method: "POST", Summary: "Discover scenario design pages and contract gaps", Category: "sketch"},
	{ID: "design_adopt", Path: experienceconnect.ExperienceServiceAdoptExperiencePageProcedure, Method: "POST", Summary: "Import a composition-bearing experience page into a workspace with an immutable recovery receipt", Category: "sketch"},
	{ID: "sketch_history", Path: experienceconnect.ExperienceServiceGetHistoryProcedure, Method: "POST", Summary: "Read immutable design revisions", Category: "sketch"},
	{ID: "sketch_recover", Path: experienceconnect.ExperienceServiceRecoverSketchProcedure, Method: "POST", Summary: "Recover an interrupted page apply", Category: "sketch"},
	{ID: "sketch_get", Path: experienceconnect.ExperienceServiceGetSketchProcedure, Method: "POST", Summary: "Read a scenario page sketch", Category: "sketch"},
	{ID: "sketch_put", Path: experienceconnect.ExperienceServicePutSketchProcedure, Method: "POST", Summary: "Write a scenario page sketch", Category: "sketch"},
	{ID: "sketch_verify", Path: experienceconnect.ExperienceServiceVerifySketchProcedure, Method: "POST", Summary: "Verify page regions against observed source", Category: "sketch"},
	{ID: "sketch_import", Path: experienceconnect.ExperienceServiceImportPageProcedure, Method: "POST", Summary: "Decompose a page through the catalog", Category: "sketch"},
	{ID: "sketch_place", Path: experienceconnect.ExperienceServicePlaceProcedure, Method: "POST", Summary: "Place a catalog asset into a region", Category: "sketch"},
	{ID: "sketch_placeholder", Path: experienceconnect.ExperienceServicePlaceholderProcedure, Method: "POST", Summary: "Record a genuinely new region need", Category: "sketch"},
	{ID: "sketch_note", Path: experienceconnect.ExperienceServiceAddNoteProcedure, Method: "POST", Summary: "Add a scoped sketch note", Category: "sketch"},
	{ID: "sketch_unplace", Path: experienceconnect.ExperienceServiceUnplaceProcedure, Method: "POST", Summary: "Move a placement to the unplaced set", Category: "sketch"},
	{ID: "sketch_template", Path: experienceconnect.ExperienceServiceSetTemplateProcedure, Method: "POST", Summary: "Swap a page template with an explicit remap", Category: "sketch"},
	{ID: "sketch_brief", Path: experienceconnect.ExperienceServiceBuildBriefProcedure, Method: "POST", Summary: "Build an ordered implementation brief", Category: "sketch"},
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
