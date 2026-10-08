// Package deps is the HTTP-handler home for the deps domain. Exposes
// DepsService (proto: packages/proto/schemas/react-component-library/v1/deps).
//
// RPCs:
//
//	ListDeclarations    — list parsed @deps for a component
//	ValidateAdoption    — produce a verdict (ok/warn/block) for a
//	                      component + target scenario pair
package deps

import (
	"database/sql"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	depsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/deps/deps_v1connect"

	"react-component-library/internal/deps"
	"react-component-library/internal/module"
)

// Module wires the deps domain. pkgs may be nil for installations
// that disable adoption validation — ValidateAdoption then returns
// FailedPrecondition.
func Module(db *sql.DB, pkgs deps.PackageJSONReader, logger *log.Logger) module.Module {
	svc := BuildService(db, pkgs)
	return ModuleFromService(svc, logger)
}

func ModuleFromService(svc deps.Service, logger *log.Logger) module.Module {
	connectPath, connectHandler := depsconnect.NewDepsServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "deps",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internal/deps.Schema for the modules registry.
func Schema() string { return deps.Schema() }

// BuildService constructs a deps.Service backed by SQLite. pkgs may be
// nil for callers that only use SyncForComponent / ListForComponent.
func BuildService(db *sql.DB, pkgs deps.PackageJSONReader) deps.Service {
	repo := deps.NewSQLiteRepository(db)
	return deps.NewService(repo, pkgs)
}

// Endpoints describes the deps module's public surface.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "deps_list_declarations",
		Path:        depsconnect.DepsServiceListDeclarationsProcedure,
		Method:      "POST",
		Summary:     "List parsed @deps declarations for a component",
		Description: "Returns the declarations the indexer recorded for the component's @deps header field. Empty slice when the header was absent or the component has not been indexed.",
		Category:    "deps",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"component_id": "string"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"declarations": "array<DepDeclaration>"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
	},
	{
		ID:          "deps_validate_adoption",
		Path:        depsconnect.DepsServiceValidateAdoptionProcedure,
		Method:      "POST",
		Summary:     "Validate a component's declared deps against a target scenario",
		Description: "Reads the target scenario's package.json (via the storage resolver) and intersects each declared semver range against the resolved version. Returns ok / warn / block plus a per-dep issues list the UI surfaces in the adoption modal.",
		Category:    "deps",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"component_id": "string",
				"scenario":     "string (target scenario id)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"kind":   "VerdictKind (ok|warn|block)",
				"issues": "array<DepIssue>",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "Target scenario's package.json missing"},
			{Status: 412, Code: "failed_precondition", Description: "Component has no declarations on file (re-run components index)"},
			{Status: 500, Code: "internal", Description: "Repository or filesystem failure"},
		},
	},
}
