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
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	adoptionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptions/adoptions_v1connect"

	"react-component-library/internal/adoptions"
	"react-component-library/internal/clock"
	"react-component-library/internal/components"
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

func (l *componentsLibrary) GetContent(ctx context.Context, id string) (components.Content, error) {
	return l.svc.GetContent(ctx, id)
}

func (l *componentsLibrary) GetVersion(ctx context.Context, componentID, version string) (components.ComponentVersion, error) {
	return l.svc.GetVersion(ctx, componentID, version)
}

// defaultScenariosRoot resolves the on-disk root the adopted-file
// reader walks. Override via ADOPTIONS_SCENARIOS_ROOT env. Default is
// the repo's top-level `scenarios/` so apply/refresh can write and read
// peer-scenario trees without extra wiring.
func defaultScenariosRoot() (string, error) {
	if path := strings.TrimSpace(os.Getenv("ADOPTIONS_SCENARIOS_ROOT")); path != "" {
		return path, nil
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve adoptions scenarios root: runtime caller unavailable")
	}
	path := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
	path = filepath.Join(path, "scenarios")
	return path, nil
}

// Endpoints is the machine-readable description of the adoptions
// module's public surface.
var Endpoints = []module.EndpointDescriptor{
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
		CLIMapping: &module.CLIMapping{Command: "react-component-library adoptions list"},
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
				"component_id":      "string",
				"scenario":          "string",
				"adopted_path":      "string",
				"version":           "string (optional)",
				"confirm_overwrite": "bool",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"adoption": "Adoption", "written_path": "string"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing required field or unknown component_id"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		CLIMapping: &module.CLIMapping{Command: "react-component-library adoptions apply", Args: []string{"<component_id>", "<scenario>", "<adopted_path>"}},
	},
	{
		ID:          "adoptions_reapply",
		Path:        adoptionsconnect.AdoptionsServiceReapplyAdoptionProcedure,
		Method:      "POST",
		Summary:     "Reapply an adoption",
		Description: "Overwrites an adopted file from a selected library version. Local modifications require explicit confirmation.",
		Category:    "adoptions",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string", "version": "string (optional)", "confirm_local_overwrite": "bool"},
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
		CLIMapping: &module.CLIMapping{Command: "react-component-library adoptions reapply", Args: []string{"<id>"}},
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
		CLIMapping: &module.CLIMapping{Command: "react-component-library adoptions delete", Args: []string{"<id>"}},
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
		CLIMapping: &module.CLIMapping{Command: "react-component-library adoptions refresh"},
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
		CLIMapping: &module.CLIMapping{Command: "react-component-library adoptions resolve-path", Args: []string{"<component_id>", "<scenario>"}},
	},
}
