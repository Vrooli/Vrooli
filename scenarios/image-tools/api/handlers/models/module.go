package models

import (
	"log"

	internalcaps "image-tools/internal/capabilities"
	internalmodels "image-tools/internal/models"
	"image-tools/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	modelsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models/models_v1connect"
)

// Module returns the models domain's contribution: the generated Connect-RPC
// ModelsService handler over the declarative registry. The registry (validated
// seed catalog) is loaded once in main.go and shared; the enabled-state overlay
// is persisted in SQLite via the models store.
func Module(db *database.RoutedDB, reg *internalmodels.Registry, probe internalcaps.Probe, logger *log.Logger) module.Module {
	store := internalmodels.NewStore(db)
	connectPath, connectHandler := modelsconnect.NewModelsServiceHandler(NewConnectHandler(Deps{
		Registry: reg,
		Store:    store,
		Probe:    probe,
		Logger:   logger,
	}))
	return module.Module{
		Name: "models",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internalmodels.Schema so the modules registry can collect
// endpoint descriptors and schema from one symbol per handler package.
func Schema() string { return internalmodels.Schema() }

// Endpoints describes the models module's public surface. Connect-RPC method
// paths reference the generated *Procedure constants, so adding or renaming an
// RPC in models.proto breaks this file at compile time. The global parity test
// (TestProtoConnectParity) asserts every rpc has exactly one entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "models_list",
		Path:        modelsconnect.ModelsServiceListModelsProcedure,
		Method:      "POST",
		Summary:     "List models",
		Description: "Returns model registry entries, optionally filtered to one operation, with effective (overlay-aware) enabled state.",
		Category:    "models",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"operation": "string (optional; filter to one op)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"models": "array<Model>"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Unknown operation filter"},
			{Status: 500, Code: "internal", Description: "Model-state load failure"},
		},
		Examples: []module.Example{
			{Name: "List all models", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/ListModels -H 'Content-Type: application/json' -d '{}'"},
			{Name: "List upscale models", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/ListModels -H 'Content-Type: application/json' -d '{\"operation\":\"upscale\"}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "image-tools models list", Args: []string{"--operation", "<op>"}},
	},
	{
		ID:          "models_get",
		Path:        modelsconnect.ModelsServiceGetModelProcedure,
		Method:      "POST",
		Summary:     "Get a model by id",
		Description: "Returns one model registry entry by id with effective enabled state.",
		Category:    "models",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"model": "Model"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No model with that id exists"},
			{Status: 500, Code: "internal", Description: "Model-state load failure"},
		},
		Examples: []module.Example{
			{Name: "Get a model", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/GetModel -H 'Content-Type: application/json' -d '{\"id\":\"sd-1.5\"}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "image-tools models get", Args: []string{"<id>"}},
	},
	{
		ID:          "models_operations",
		Path:        modelsconnect.ModelsServiceListOperationsProcedure,
		Method:      "POST",
		Summary:     "List operations",
		Description: "Returns the registry operation vocabulary in declaration order.",
		Category:    "models",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"operations": "array<string>"},
		},
		Examples: []module.Example{
			{Name: "List operations", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/ListOperations -H 'Content-Type: application/json' -d '{}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "image-tools models operations"},
	},
	{
		ID:          "models_select",
		Path:        modelsconnect.ModelsServiceSelectModelProcedure,
		Method:      "POST",
		Summary:     "Preview model selection for an operation",
		Description: "Previews which enabled model would run for an operation on the probed host (honoring the per-op default and any override) without executing. Surfaces the hardware-fit reason and warnings.",
		Category:    "models",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"operation":   "string (required; registry op)",
				"override_id": "string (optional; force a model id)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"model":      "Model",
				"gpu_viable": "bool",
				"reason":     "string",
				"warnings":   "array<string>",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Unknown operation or invalid override"},
			{Status: 412, Code: "failed_precondition", Description: "No enabled model can run for the operation on this host"},
			{Status: 500, Code: "internal", Description: "Host probe or model-state load failure"},
		},
		Examples: []module.Example{
			{Name: "Select for upscale", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/SelectModel -H 'Content-Type: application/json' -d '{\"operation\":\"upscale\"}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "image-tools models select", Args: []string{"<operation>", "--override", "<id>"}},
	},
	{
		ID:          "models_set_enabled",
		Path:        modelsconnect.ModelsServiceSetModelEnabledProcedure,
		Method:      "POST",
		Summary:     "Enable or disable a model",
		Description: "Toggles a model's runtime-enabled state, persisted in the SQLite overlay over the read-only seed catalog.",
		Category:    "models",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"id":      "string",
				"enabled": "bool",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"model": "Model"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No model with that id exists"},
			{Status: 500, Code: "internal", Description: "Model-state persistence failure"},
		},
		Examples: []module.Example{
			{Name: "Enable a quality tier", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/SetModelEnabled -H 'Content-Type: application/json' -d '{\"id\":\"real-esrgan\",\"enabled\":true}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "image-tools models enable", Args: []string{"<id>"}},
	},
	{
		ID:          "models_blocklist",
		Path:        modelsconnect.ModelsServiceListBlocklistProcedure,
		Method:      "POST",
		Summary:     "List blocklisted models",
		Description: "Returns the license-encumbered models that must never be seeded or adopted, with the reason each is excluded.",
		Category:    "models",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"entries": "array<BlocklistEntry>"},
		},
		Examples: []module.Example{
			{Name: "List blocklist", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/ListBlocklist -H 'Content-Type: application/json' -d '{}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "image-tools models blocklist"},
	},
}
