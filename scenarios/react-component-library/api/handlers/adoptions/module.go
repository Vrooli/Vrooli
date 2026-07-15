// Package adoptions is the HTTP-handler home for the adoptions
// domain. Exposes AdoptionsService (proto:
// packages/proto/schemas/react-component-library/v1/adoptions).
//
// RPCs:
//
//	ListAdoptions     — filter by component_id / scenario / limit
//	CreateAdoption    — soft FK to components.id; validates exists
//	DeleteAdoption    — by id
//	RefreshAdoptions  — recompute drift status for all (or one component's) rows
package adoptions

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	repocontract "github.com/vrooli/repo-contract-go"

	adoptionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptions/adoptions_v1connect"

	"react-component-library/internal/adoptions"
	"react-component-library/internal/clock"
	"react-component-library/internal/components"
	"react-component-library/internal/deps"
	"react-component-library/internal/module"
)

// Module wires the adoptions domain using the production-default
// scenarios root. Tests should use ModuleWithRoot to inject a temp
// dir + a custom LibraryReader.
func Module(db *sql.DB, clk clock.Clock, library adoptions.LibraryReader, logger *log.Logger) module.Module {
	root, err := defaultScenariosRoot()
	if err != nil {
		logger.Fatalf("adoptions scenarios root: %v", err)
	}
	return ModuleWithRoot(db, clk, library, root, logger)
}

// ModuleWithRoot is the explicit-injection variant used by tests and
// callers that want to point the scenario-file reader at a custom path.
func ModuleWithRoot(db *sql.DB, clk clock.Clock, library adoptions.LibraryReader, scenariosRoot string, logger *log.Logger) module.Module {
	svc, _ := BuildService(db, clk, library, scenariosRoot)
	return ModuleFromService(svc, logger)
}

// ModuleOption customises ModuleFromService. New optional dependencies (e.g.
// the path resolver for ResolveAdoptionPath) are added here instead of
// growing the function signature.
type ModuleOption func(*Deps)

// WithResolver wires the adoption-path resolver and the slot reader used by
// ResolveAdoptionPath. Without this option the RPC returns CodeUnimplemented.
func WithResolver(resolver *adoptions.Resolver, slot SlotReader, library adoptions.LibraryReader) ModuleOption {
	return func(d *Deps) {
		d.Resolver = resolver
		d.SlotReader = slot
		d.Library = library
	}
}

// WithSuggestions wires the real inventory, component, and dependency seams
// used by the explainable adoption candidate RPC.
func WithSuggestions(componentService components.Service, dependencyService deps.Service, inventory InventoryScanner, scenariosRoot string) ModuleOption {
	return func(d *Deps) {
		d.Components = componentService
		d.Dependencies = dependencyService
		d.Inventory = inventory
		d.ScenariosRoot = scenariosRoot
	}
}

// ModuleFromService mounts a prebuilt adoptions.Service. main.go uses
// this variant when the service is shared with sibling modules.
func ModuleFromService(svc adoptions.Service, logger *log.Logger, opts ...ModuleOption) module.Module {
	deps := Deps{
		Service: svc,
		Logger:  logger,
	}
	for _, opt := range opts {
		opt(&deps)
	}
	connectPath, connectHandler := adoptionsconnect.NewAdoptionsServiceHandler(NewConnectHandler(deps))
	return module.Module{
		Name: "adoptions",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internal/adoptions.Schema for the modules registry.
func Schema() string { return adoptions.Schema() }

// BuildService is the shared seam main.go calls to construct one
// adoptions.Service that sibling modules can read from (e.g.,
// versions resolves `adoption:<id>` diff sides through this service
// and the returned ScenarioFileReader).
func BuildService(db *sql.DB, clk clock.Clock, library adoptions.LibraryReader, scenariosRoot string) (adoptions.Service, adoptions.ScenarioFileReader) {
	repo := adoptions.NewSQLiteRepository(db, clk)
	files := adoptions.NewFSScenarioFileReader(scenariosRoot)
	svc := adoptions.NewService(repo, library, files, clk)
	return svc, files
}

// DefaultScenariosRoot exposes the production-default adoption scenarios
// root so main.go can share the same value across the adoptions and
// versions modules.
func DefaultScenariosRoot() (string, error) { return defaultScenariosRoot() }

// LibraryFromComponents adapts a components.Service into the minimal
// LibraryReader the adoptions service depends on. Keeps the package
// graph clean: handlers/adoptions imports internal/components, but
// internal/adoptions does not.
func LibraryFromComponents(svc components.Service) adoptions.LibraryReader {
	return &componentsLibrary{svc: svc}
}

type componentsLibrary struct {
	svc components.Service
}

func (l *componentsLibrary) Get(ctx context.Context, id string) (components.Component, error) {
	return l.svc.Get(ctx, id)
}

func (l *componentsLibrary) List(ctx context.Context, q components.SearchQuery) ([]components.Component, error) {
	return l.svc.List(ctx, q)
}

func (l *componentsLibrary) GetContent(ctx context.Context, id string) (components.Content, error) {
	return l.svc.GetContent(ctx, id)
}

func (l *componentsLibrary) GetVersion(ctx context.Context, componentID, version string) (components.ComponentVersion, error) {
	return l.svc.GetVersion(ctx, componentID, version)
}

func (l *componentsLibrary) ListVersions(ctx context.Context, componentID string, limit int) ([]components.ComponentVersion, error) {
	return l.svc.ListVersions(ctx, componentID, limit)
}

// defaultScenariosRoot resolves the on-disk scenarios root via the canonical
// repo-contract discovery (VROOLI_SOURCE_ROOT / VROOLI_ROOT env vars, then
// CWD, then the executable's directory) plus the contract's declared
// top-level scenarios dir. There is no scenario-local env override: the
// repo-contract envs are the single source of truth.
func defaultScenariosRoot() (string, error) {
	contract, repoRoot, err := repocontract.LoadDefaultFromEnvOrCWD()
	if err != nil {
		return "", fmt.Errorf("resolve scenarios root via repo-contract: %w", err)
	}
	return contract.TopLevelDir(repoRoot, "scenarios")
}

// Endpoints is the machine-readable description of the adoptions
// module's public surface.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "adoptions_reconcile",
		Path:        adoptionsconnect.AdoptionsServiceReconcileAdoptionsProcedure,
		Method:      "POST",
		Summary:     "Backfill adoption records from on-disk provenance",
		Description: "Scans scenario UI source read-only. Dry-run is default; apply writes only RCL adoption records without backlog reporting.",
		Category:    "adoptions",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"apply": "boolean"}},
		Response:    &module.Schema{Type: "object"},
	},
	{
		ID:          "adoptions_reconverge",
		Path:        adoptionsconnect.AdoptionsServiceReconvergeAdoptionsProcedure,
		Method:      "POST",
		Summary:     "Reconverge BEHIND adoptions to the current library version",
		Description: "Batch-reconverges BEHIND adoptions: re-applies CLEAN copies through the validated Reapply path and flags MODIFIED copies for human review without overwriting them. Dry-run is default; apply performs the re-applies. Reports per-adoption and per-file outcomes.",
		Category:    "adoptions",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string (filter)", "apply": "boolean"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"outcomes": "array<ReconvergeOutcome>"}},
	},
	{
		ID:          "adoptions_list",
		Path:        adoptionsconnect.AdoptionsServiceListAdoptionsProcedure,
		Method:      "POST",
		Summary:     "List adoption records",
		Description: "Returns adoption records matching optional component_id / scenario / limit filters.",
		Category:    "adoptions",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"component_id": "string (filter)",
				"scenario":     "string (filter)",
				"limit":        "int32 (max rows, 0 = server default)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"adoptions": "array<Adoption>"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
	},
	{
		ID:          "adoptions_list_effective",
		Path:        adoptionsconnect.AdoptionsServiceListEffectiveAdoptionsProcedure,
		Method:      "POST",
		Summary:     "List direct and mediated adoptions for a component",
		Description: "Returns every adoption that makes the selected component present in a scenario, including provenance through a parent component adoption.",
		Category:    "adoptions",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"component_id": "string (required)",
			"limit":        "int32 (max rows, 0 = server default)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"adoptions": "array<EffectiveAdoption>"}},
		Errors:   []module.ErrorDesc{{Status: 500, Code: "internal", Description: "Repository read failure"}},
	},
	{
		ID:          "adoptions_apply",
		Path:        adoptionsconnect.AdoptionsServiceApplyAdoptionProcedure,
		Method:      "POST",
		Summary:     "Apply a component to a scenario",
		Description: "Copies a selected component version into a target scenario, stamps provenance, and records the adoption.",
		Category:    "adoptions",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"component_id":        "string",
				"scenario":            "string",
				"adopted_path":        "string",
				"version":             "string (optional)",
				"confirm_overwrite":   "bool",
				"replace_existing":    "bool (explicitly replace an existing source file)",
				"override_validation": "bool (explicitly allow a blocking dependency verdict)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"adoption": "Adoption", "written_path": "string", "import_sites": "array<string>"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing required field or unknown component_id"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
	},
	{
		ID:          "adoptions_reapply",
		Path:        adoptionsconnect.AdoptionsServiceReapplyAdoptionProcedure,
		Method:      "POST",
		Summary:     "Reapply an adoption",
		Description: "Overwrites an adopted file from a selected library version. Local modifications require explicit confirmation.",
		Category:    "adoptions",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"id":                      "string",
				"version":                 "string (optional)",
				"confirm_local_overwrite": "bool",
				"override_validation":     "bool (explicitly allow a blocking dependency verdict)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"adoption": "Adoption", "written_path": "string"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing id or local edits without confirmation"},
			{Status: 404, Code: "not_found", Description: "No adoption with that id"},
			{Status: 500, Code: "internal", Description: "Repository or filesystem failure"},
		},
	},
	{
		ID:          "adoptions_delete",
		Path:        adoptionsconnect.AdoptionsServiceDeleteAdoptionProcedure,
		Method:      "POST",
		Summary:     "Delete an adoption record",
		Description: "Removes the adoption record matching the id.",
		Category:    "adoptions",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string"},
		},
		Response: &module.Schema{Type: "object"},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No adoption with that id"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
	},
	{
		ID:          "adoptions_refresh",
		Path:        adoptionsconnect.AdoptionsServiceRefreshAdoptionsProcedure,
		Method:      "POST",
		Summary:     "Refresh drift status for adoption records",
		Description: "Recomputes library-version and local-edit drift statuses for every adoption record, or just those for a single component when component_id is supplied.",
		Category:    "adoptions",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"component_id": "string (optional filter)"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"adoptions":       "array<Adoption>",
				"library_current": "int32",
				"library_behind":  "int32",
				"local_clean":     "int32",
				"local_modified":  "int32",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Repository or filesystem failure"},
		},
	},
	{
		ID:          "adoptions_resolve_path",
		Path:        adoptionsconnect.AdoptionsServiceResolveAdoptionPathProcedure,
		Method:      "POST",
		Summary:     "Resolve the canonical adopted path for a component+scenario",
		Description: "Computes the filesystem path an adoption would land at, using the target scenario's UI manifest (template-manifest source), a heuristic, or a fallback. Read-only.",
		Category:    "adoptions",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"component_id":  "string",
				"scenario":      "string",
				"override_path": "string (optional)",
				"feature":       "string (optional, when slot requiresFeature)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"path":     "string (relative to scenario root)",
				"source":   "ResolveSource enum",
				"slot":     "string (slot name used)",
				"warnings": "array<string>",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing component_id, scenario, or unsubstituted path token"},
			{Status: 501, Code: "unimplemented", Description: "Resolver not configured (server lacks repo-root wiring)"},
		},
	},
	{
		ID:          "adoptions_suggest",
		Path:        adoptionsconnect.AdoptionsServiceSuggestAdoptionsProcedure,
		Method:      "POST",
		Summary:     "Suggest catalog components a scenario should adopt",
		Description: "Returns non-adopted catalog components using real UI inventory matches, design-style fit, and dependency compatibility. Every candidate carries human-readable reasons; no opaque score is emitted.",
		Category:    "adoptions",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string (optional)", "limit": "int32 (optional)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"suggestions": "array<AdoptionSuggestion>"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Invalid scenario"},
			{Status: 501, Code: "unimplemented", Description: "Suggestion dependencies not configured"},
		},
	},
	{
		ID:          "adoptions_discover",
		Path:        adoptionsconnect.AdoptionsServiceDiscoverAdoptionsProcedure,
		Method:      "POST",
		Summary:     "Discover header-less vendored copies by content similarity",
		Description: "Walks scenario UI trees for source files with no @vrooliComponentSource header and scores each against every library component version (Sørensen–Dice line similarity). Read-only: candidates carry similarity evidence for operator review; ConfirmDiscovery performs any write.",
		Category:    "adoptions",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string (optional)", "min_similarity": "double (0 = server default 0.6)", "limit": "int32 (optional)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scanned": "int32", "min_similarity": "double", "candidates": "array<DiscoveryCandidate>"}},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Scanner or repository failure"},
		},
	},
	{
		ID:          "adoptions_confirm_discovery",
		Path:        adoptionsconnect.AdoptionsServiceConfirmDiscoveryProcedure,
		Method:      "POST",
		Summary:     "Confirm a discovery candidate and backfill its provenance",
		Description: "Injects an @vrooliComponentSource header into a header-less scenario file and creates its adoption record, attributed to the named component + version. Refuses files that already carry a header or are already tracked.",
		Category:    "adoptions",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "adopted_path": "string", "component_id": "string", "version": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"adoption": "Adoption", "written_path": "string", "similarity": "double"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing field, file already tagged, or already tracked"},
			{Status: 404, Code: "not_found", Description: "Unknown component_id or version"},
			{Status: 500, Code: "internal", Description: "Repository or filesystem failure"},
		},
	},
}
