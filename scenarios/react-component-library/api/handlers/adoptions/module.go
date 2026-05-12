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
	"strings"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/storage"

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
	repo := adoptions.NewSQLiteRepository(db, clk)
	files := adoptions.NewFSScenarioFileReader(scenariosRoot)
	svc := adoptions.NewService(repo, library, files, clk)
	connectPath, connectHandler := adoptionsconnect.NewAdoptionsServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
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

// defaultScenariosRoot resolves the on-disk root the adopted-file
// reader walks. Override via ADOPTIONS_SCENARIOS_ROOT env. Default is
// the repo's top-level `scenarios/` so adoption refresh can read
// peer-scenario trees without extra wiring.
func defaultScenariosRoot() (string, error) {
	if path := strings.TrimSpace(os.Getenv("ADOPTIONS_SCENARIOS_ROOT")); path != "" {
		return path, nil
	}
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("storage resolver: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: "react-component-library"},
		storage.ClassData,
		"adoptions-scenarios",
	)
	if err != nil {
		return "", fmt.Errorf("resolve adoptions scenarios root: %w", err)
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", fmt.Errorf("create adoptions scenarios root: %w", err)
	}
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
		ID:          "adoptions_create",
		Path:        adoptionsconnect.AdoptionsServiceCreateAdoptionProcedure,
		Method:      "POST",
		Summary:     "Create an adoption record",
		Description: "Soft-links a library component to a copy of its source under a target scenario. Validates the component exists; captures a sha256 of the adopted file at create time for later drift comparison.",
		Category:    "adoptions",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"component_id":    "string",
				"scenario":        "string",
				"adopted_path":    "string",
				"adopted_version": "string",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"adoption": "Adoption"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing required field or unknown component_id"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		CLIMapping: &module.CLIMapping{Command: "react-component-library adoptions create", Args: []string{"<component_id>", "<scenario>", "<adopted_path>"}},
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
		Description: "Recomputes status (current / behind / modified / unknown) for every adoption record, or just those for a single component when component_id is supplied.",
		Category:    "adoptions",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"component_id": "string (optional filter)"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"adoptions": "array<Adoption>",
				"current":   "int32",
				"behind":    "int32",
				"modified":  "int32",
				"unknown":   "int32",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Repository or filesystem failure"},
		},
		CLIMapping: &module.CLIMapping{Command: "react-component-library adoptions refresh"},
	},
}
