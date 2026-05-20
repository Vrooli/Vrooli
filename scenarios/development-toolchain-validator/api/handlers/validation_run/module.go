// Package validation_run is the HTTP/Connect handler edge for the
// validation_run domain. The lifecycle worker lives in
// internal/validation_run/worker.go and is started by main.go.
package validation_run

import (
	"log"

	"github.com/vrooli/api-core/database"

	"development-toolchain-validator/internal/clock"
	"development-toolchain-validator/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	vrunconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_run/validation_run_v1connect"

	vrun "development-toolchain-validator/internal/validation_run"
)

// ModuleDeps groups the seams Module needs from main.go. Constructor
// injection rather than build-it-here so the same Repository instance
// the worker uses is the one the handler reads from.
type ModuleDeps struct {
	DB     *database.RoutedDB
	Clock  clock.Clock
	Logger *log.Logger
	Notify func()
}

// Module returns the validation_run domain's contribution to the API.
func Module(deps ModuleDeps) module.Module {
	repo := vrun.NewSQLiteRepository(deps.DB)
	svc := vrun.NewService(repo, deps.Clock, deps.Notify)
	connectPath, connectHandler := vrunconnect.NewValidationRunServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  deps.Logger,
	}))
	return module.Module{
		Name: "validation_run",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the internal package schema.
func Schema() string { return vrun.Schema() }

// Endpoints is the machine-readable description of the validation_run
// module's public surface.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "validation_run_start",
		Path:        vrunconnect.ValidationRunServiceStartProcedure,
		Method:      "POST",
		Summary:     "Start a validation run",
		Description: "Queues a new (skill|tool, golden) validation run. Returns immediately; an in-process worker advances the lifecycle to terminal and writes a ValidationRecord.",
		Category:    "validation_run",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"tuple_kind":  "TupleKind (required)",
				"subject_id":  "string (required) — skill id or tool name",
				"golden_slug": "string (required)",
				"force":       "bool",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"run": "ValidationRun"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing tuple_kind/subject_id/golden_slug"},
			{Status: 503, Code: "unavailable", Description: "A required outbound dependency is not running"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{
			{Name: "Start skill run", Curl: "curl http://localhost:${API_PORT}/vrooli.development_toolchain_validator.v1.validation_run.ValidationRunService/Start -H 'Content-Type: application/json' -d '{\"tuple_kind\":1,\"subject_id\":\"plan-skill-discovery\",\"golden_slug\":\"reference-react-vite\"}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "development-toolchain-validator validation start", Args: []string{"--skill <id> | --tool <name>", "--golden <slug>", "[--force]", "[--wait]"}},
	},
	{
		ID:          "validation_run_get",
		Path:        vrunconnect.ValidationRunServiceGetProcedure,
		Method:      "POST",
		Summary:     "Get a validation run by id",
		Description: "Returns the validation run's current operational state. Poll this for terminal status.",
		Category:    "validation_run",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string (required)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"run": "ValidationRun"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing id"},
			{Status: 404, Code: "not_found", Description: "No run with that id"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Get run", Curl: "curl http://localhost:${API_PORT}/vrooli.development_toolchain_validator.v1.validation_run.ValidationRunService/Get -H 'Content-Type: application/json' -d '{\"id\":\"<uuid>\"}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "development-toolchain-validator validation get", Args: []string{"<id>"}},
	},
	{
		ID:          "validation_run_list_active",
		Path:        vrunconnect.ValidationRunServiceListActiveProcedure,
		Method:      "POST",
		Summary:     "List active validation runs",
		Description: "Returns all runs whose status is not TERMINAL.",
		Category:    "validation_run",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"runs": "array<ValidationRun>"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "List active", Curl: "curl http://localhost:${API_PORT}/vrooli.development_toolchain_validator.v1.validation_run.ValidationRunService/ListActive -H 'Content-Type: application/json' -d '{}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "development-toolchain-validator validation list-active"},
	},
}
