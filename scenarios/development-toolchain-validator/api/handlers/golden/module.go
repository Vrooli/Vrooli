package golden

import (
	"log"

	"github.com/vrooli/api-core/database"

	"development-toolchain-validator/internal/module"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	goldenconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/golden/golden_v1connect"

	internalgolden "development-toolchain-validator/internal/golden"
)

// Module returns the golden domain's contribution to the API. Production
// callers pass a real SubprocessRunner via ModuleWithRunner so the
// regenerate RPC can shell out to vrooli scenario generate; tests use
// ModuleWithRunner with an in-memory fake runner.
func Module(db *database.RoutedDB, clk schedule.Clock, logger *log.Logger) module.Module {
	return ModuleWithRunner(db, clk, internalgolden.NewSubprocessRunner("", ""), logger)
}

// ModuleWithRunner is the explicit-injection variant. Used by tests to
// supply a deterministic RegeneratorRunner.
func ModuleWithRunner(db *database.RoutedDB, clk schedule.Clock, runner internalgolden.RegeneratorRunner, logger *log.Logger) module.Module {
	repo := internalgolden.NewSQLiteRepository(db, clk)
	svc := internalgolden.NewService(repo, clk, runner)
	connectPath, connectHandler := goldenconnect.NewGoldenServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "golden",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the internal package schema so the modules registry
// can collect endpoint descriptors and schema from one symbol per
// handler package.
func Schema() string { return internalgolden.Schema() }

// Endpoints is the machine-readable description of the golden module's
// public surface. Connect-RPC method paths reference the generated
// *Procedure constants so adding or renaming an RPC in golden.proto
// breaks this file at compile time. The complementary global parity
// test asserts every rpc has exactly one entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "golden_list",
		Path:        goldenconnect.GoldenServiceListGoldensProcedure,
		Method:      "POST",
		Summary:     "List goldens",
		Description: "Returns every registered template-pristine golden ordered by slug.",
		Category:    "golden",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"goldens": "array<Golden>"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "List goldens", Curl: "curl http://localhost:${API_PORT}/vrooli.development_toolchain_validator.v1.golden.GoldenService/ListGoldens -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "golden_get",
		Path:        goldenconnect.GoldenServiceGetGoldenProcedure,
		Method:      "POST",
		Summary:     "Get a golden by slug",
		Description: "Returns the golden whose slug matches the request.",
		Category:    "golden",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"slug": "string"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"golden": "Golden"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing slug"},
			{Status: 404, Code: "not_found", Description: "No golden with that slug exists"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Get golden", Curl: "curl http://localhost:${API_PORT}/vrooli.development_toolchain_validator.v1.golden.GoldenService/GetGolden -H 'Content-Type: application/json' -d '{\"slug\":\"reference-react-vite\"}'"},
		},
	},
	{
		ID:          "golden_register",
		Path:        goldenconnect.GoldenServiceRegisterGoldenProcedure,
		Method:      "POST",
		Summary:     "Register a golden",
		Description: "Persists a generated-golden record. Slug must be unique. Validation runs materialize the substrate from template metadata into a managed generated path.",
		Category:    "golden",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"slug":             "string (required, kebab-case)",
				"template_id":      "string (required)",
				"template_version": "string (required)",
				"path":             "string (optional logical root, repo-relative)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"golden": "Golden"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Invalid slug or missing required field"},
			{Status: 409, Code: "already_exists", Description: "A golden with that slug is already registered"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{
			{Name: "Register generated golden", Curl: "curl http://localhost:${API_PORT}/vrooli.development_toolchain_validator.v1.golden.GoldenService/RegisterGolden -H 'Content-Type: application/json' -d '{\"slug\":\"reference-react-vite\",\"template_id\":\"react-vite\",\"template_version\":\"1.0.1\"}'"},
		},
	},
	{
		ID:          "golden_update",
		Path:        goldenconnect.GoldenServiceUpdateGoldenProcedure,
		Method:      "POST",
		Summary:     "Update a golden",
		Description: "Patches the logical root and/or template_version of an existing generated golden. Empty fields are left unchanged.",
		Category:    "golden",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"slug":             "string (required)",
				"path":             "string (optional)",
				"template_version": "string (optional)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"golden": "Golden"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Invalid input"},
			{Status: 404, Code: "not_found", Description: "No golden with that slug exists"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{
			{Name: "Update golden", Curl: "curl http://localhost:${API_PORT}/vrooli.development_toolchain_validator.v1.golden.GoldenService/UpdateGolden -H 'Content-Type: application/json' -d '{\"slug\":\"reference-react-vite\",\"template_version\":\"1.0.2\"}'"},
		},
	},
	{
		ID:          "golden_delete",
		Path:        goldenconnect.GoldenServiceDeleteGoldenProcedure,
		Method:      "POST",
		Summary:     "Delete a golden",
		Description: "Removes the golden record. Does not touch files on disk.",
		Category:    "golden",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"slug": "string (required)"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing slug"},
			{Status: 404, Code: "not_found", Description: "No golden with that slug exists"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{
			{Name: "Delete golden", Curl: "curl http://localhost:${API_PORT}/vrooli.development_toolchain_validator.v1.golden.GoldenService/DeleteGolden -H 'Content-Type: application/json' -d '{\"slug\":\"reference-react-vite\"}'"},
		},
	},
	{
		ID:          "golden_regenerate",
		Path:        goldenconnect.GoldenServiceRegenerateGoldenProcedure,
		Method:      "POST",
		Summary:     "Regenerate a golden",
		Description: "Invokes the configured regenerator runner (production: vrooli scenario generate --force) to re-materialize the golden on disk and refresh its template version pin.",
		Category:    "golden",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"slug": "string (required)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"golden": "Golden"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing slug"},
			{Status: 404, Code: "not_found", Description: "No golden with that slug exists"},
			{Status: 500, Code: "internal", Description: "Regenerator runner failure"},
		},
		Examples: []module.Example{
			{Name: "Regenerate golden", Curl: "curl http://localhost:${API_PORT}/vrooli.development_toolchain_validator.v1.golden.GoldenService/RegenerateGolden -H 'Content-Type: application/json' -d '{\"slug\":\"reference-react-vite\"}'"},
		},
	},
}
