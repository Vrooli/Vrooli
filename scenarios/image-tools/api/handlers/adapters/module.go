package adapters

import (
	"log"

	internaladapters "image-tools/internal/adapters"
	internalmodels "image-tools/internal/models"
	"image-tools/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	adaptersconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/adapters/adapters_v1connect"
)

// Module returns the adapters domain's contribution: the generated Connect-RPC
// AdaptersServiceHandler over the declarative conditioning-adapter catalog. The
// registry (validated seed catalog) is loaded once in main.go and shared; the
// enabled-state overlay is persisted in SQLite via the adapter store.
func Module(db *database.RoutedDB, reg *internaladapters.Registry, models *internalmodels.Registry, installer *internaladapters.Installer, jobs JobSubmitter, estimateInstallSeconds EstimateInstallSecondsFunc, logger *log.Logger) module.Module {
	store := internaladapters.NewStore(db)
	connectPath, connectHandler := adaptersconnect.NewAdaptersServiceHandler(NewConnectHandler(Deps{
		Registry:               reg,
		Store:                  store,
		Installer:              installer,
		Models:                 models,
		Jobs:                   jobs,
		EstimateInstallSeconds: estimateInstallSeconds,
		Logger:                 logger,
	}))
	return module.Module{
		Name: "adapters",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internaladapters.Schema so the modules registry can collect
// endpoint descriptors and schema from one symbol per handler package.
func Schema() string { return internaladapters.Schema() }

// Endpoints describes the adapters module's public surface. Connect-RPC method
// paths reference the generated *Procedure constants, so adding or renaming an
// RPC in adapters.proto breaks this file at compile time. The global parity test
// (TestProtoConnectParity) asserts every rpc has exactly one entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "adapters_list",
		Path:        adaptersconnect.AdaptersServiceListAdaptersProcedure,
		Method:      "POST",
		Summary:     "List adapters",
		Description: "Returns conditioning-adapter catalog entries, optionally filtered by kind and/or compatible architecture, with effective (overlay-aware) enabled + install state.",
		Category:    "adapters",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"kind":         "string (optional; lora|controlnet|ip-adapter)",
				"architecture": "string (optional; e.g. sd15, sdxl)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"adapters": "array<Adapter>"},
		},
		Examples: []module.Example{
			{Name: "List all adapters", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.adapters.AdaptersService/ListAdapters -H 'Content-Type: application/json' -d '{}'"},
			{Name: "List sdxl loras", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.adapters.AdaptersService/ListAdapters -H 'Content-Type: application/json' -d '{\"kind\":\"lora\",\"architecture\":\"sdxl\"}'"},
		},
	},
	{
		ID:          "adapters_get",
		Path:        adaptersconnect.AdaptersServiceGetAdapterProcedure,
		Method:      "POST",
		Summary:     "Get an adapter by id",
		Description: "Returns one adapter catalog entry by id with effective enabled + install state.",
		Category:    "adapters",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"adapter": "Adapter"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No adapter with that id exists"},
			{Status: 500, Code: "internal", Description: "Adapter-state load failure"},
		},
		Examples: []module.Example{
			{Name: "Get an adapter", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.adapters.AdaptersService/GetAdapter -H 'Content-Type: application/json' -d '{\"id\":\"controlnet-canny-sdxl\"}'"},
		},
	},
	{
		ID:          "adapters_set_enabled",
		Path:        adaptersconnect.AdaptersServiceSetAdapterEnabledProcedure,
		Method:      "POST",
		Summary:     "Enable or disable an adapter",
		Description: "Toggles an adapter's runtime-enabled state, persisted in the SQLite overlay over the read-only seed catalog.",
		Category:    "adapters",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"id":      "string",
				"enabled": "bool",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"adapter": "Adapter"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No adapter with that id exists"},
			{Status: 500, Code: "internal", Description: "Adapter-state persistence failure"},
		},
		Examples: []module.Example{
			{Name: "Enable an adapter", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.adapters.AdaptersService/SetAdapterEnabled -H 'Content-Type: application/json' -d '{\"id\":\"controlnet-canny-sdxl\",\"enabled\":true}'"},
		},
	},
	{
		ID:          "adapters_install",
		Path:        adaptersconnect.AdaptersServiceInstallAdapterProcedure,
		Method:      "POST",
		Summary:     "Install (download) an adapter",
		Description: "Downloads an adapter's weights as a durable job (checksum pinned on first download, free disk space checked first). Returns a job id + ETA; block once on jobs wait. An already-installed adapter returns already_installed=true with no job.",
		Category:    "adapters",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"job_id":            "string (empty when already installed)",
				"eta_seconds":       "int",
				"size_mb_approx":    "int",
				"already_installed": "bool",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No adapter with that id exists"},
			{Status: 500, Code: "internal", Description: "Install job submission failure"},
			{Status: 501, Code: "unimplemented", Description: "Installation unavailable (read-only wiring)"},
		},
		Examples: []module.Example{
			{Name: "Install an adapter", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.adapters.AdaptersService/InstallAdapter -H 'Content-Type: application/json' -d '{\"id\":\"controlnet-canny-sdxl\"}'"},
		},
	},
	{
		ID:          "adapters_remove",
		Path:        adaptersconnect.AdaptersServiceRemoveAdapterProcedure,
		Method:      "POST",
		Summary:     "Remove an adapter's weights",
		Description: "Deletes an adapter's downloaded weights and clears its install record. A local adapter is unlinked; its referenced path is never deleted.",
		Category:    "adapters",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"removed": "bool"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No adapter with that id exists"},
			{Status: 500, Code: "internal", Description: "Weight removal failure"},
		},
		Examples: []module.Example{
			{Name: "Remove an adapter", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.adapters.AdaptersService/RemoveAdapter -H 'Content-Type: application/json' -d '{\"id\":\"controlnet-canny-sdxl\"}'"},
		},
	},
	{
		ID:          "adapters_inspect_source",
		Path:        adaptersconnect.AdaptersServiceInspectAdapterSourceProcedure,
		Method:      "POST",
		Summary:     "Inspect an adapter import source (dry run)",
		Description: "Dry-runs a guided import: inspects a HuggingFace repo id, direct weight URL, or local path WITHOUT installing anything, returning the inferred kind (+evidence), the inferred compatible architecture (with confidence + evidence the user confirms), license/NSFW, approximate size, and the proposed catalog entry.",
		Category:    "adapters",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"source": "string (HF repo id, URL, or local path)"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"source":       "string",
				"repo_id":      "string",
				"revision":     "string",
				"kind":         "KindInference (kind, evidence)",
				"architecture": "ArchitectureInference (architecture, confidence, evidence)",
				"license":      "string",
				"nsfw":         "bool",
				"size_bytes":   "uint64",
				"proposed":     "Adapter",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing/unresolvable source"},
			{Status: 501, Code: "unimplemented", Description: "Import unavailable (read-only wiring)"},
		},
		Examples: []module.Example{
			{Name: "Inspect a ControlNet", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.adapters.AdaptersService/InspectAdapterSource -H 'Content-Type: application/json' -d '{\"source\":\"diffusers/controlnet-canny-sdxl-1.0\"}'"},
		},
	},
	{
		ID:          "adapters_import",
		Path:        adaptersconnect.AdaptersServiceImportAdapterProcedure,
		Method:      "POST",
		Summary:     "Import an adapter by source (confirm + install)",
		Description: "Composes inspect → operator-confirmed fields (kind/architecture/preprocessor/attestation) → an add-only custom entry → a durable install job (mirrors InstallAdapter: returns job id + ETA; block once on jobs wait). Imports default to local tier with a user-imported provenance label; public/BYOK serving requires attest_commercial_rights.",
		Category:    "adapters",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"source":                   "string (HF repo id, URL, or local path)",
				"id":                       "string (new entry id; must not collide with a seed adapter)",
				"name":                     "string (optional display name)",
				"kind":                     "string (required when inference is ambiguous; else overrides)",
				"architecture":             "string (required when inference is none; else overrides)",
				"preprocessor":             "string (ControlNet preprocessor; ignored for other kinds)",
				"attest_commercial_rights": "bool (allow public/BYOK serving — decision D4)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"adapter":           "Adapter",
				"job_id":            "string (empty when already installed or no job runner)",
				"eta_seconds":       "int",
				"already_installed": "bool",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing source/id, unresolved kind/architecture, or seed-id collision"},
			{Status: 500, Code: "internal", Description: "Persistence or job submission failure"},
			{Status: 501, Code: "unimplemented", Description: "Custom adapters / import unavailable (read-only wiring)"},
		},
		Examples: []module.Example{
			{Name: "Import a ControlNet", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.adapters.AdaptersService/ImportAdapter -H 'Content-Type: application/json' -d '{\"source\":\"diffusers/controlnet-canny-sdxl-1.0\",\"id\":\"imported-canny\",\"kind\":\"controlnet\",\"architecture\":\"sdxl\"}'"},
		},
	},
	{
		ID:          "adapters_compatible",
		Path:        adaptersconnect.AdaptersServiceListCompatibleAdaptersProcedure,
		Method:      "POST",
		Summary:     "List adapters compatible with a base model",
		Description: "Returns every catalog adapter compatible with a base model (by architecture; resolve via model_id or pass architecture directly), annotated with enabled/installed/ready state so the picker can show why an adapter is or is not yet offerable.",
		Category:    "adapters",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"model_id":     "string (resolves the model's architecture)",
				"architecture": "string (alternative to model_id; one is required)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"architecture": "string (the architecture compatibility was computed against)",
				"adapters":     "array<Adapter>",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Neither model_id nor architecture supplied"},
			{Status: 404, Code: "not_found", Description: "No model with that id exists"},
			{Status: 500, Code: "internal", Description: "Adapter-state load failure"},
		},
		Examples: []module.Example{
			{Name: "Compatible with sdxl", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.adapters.AdaptersService/ListCompatibleAdapters -H 'Content-Type: application/json' -d '{\"architecture\":\"sdxl\"}'"},
		},
	},
	{
		ID:          "adapters_doctor",
		Path:        adaptersconnect.AdaptersServiceDoctorCatalogProcedure,
		Method:      "POST",
		Summary:     "Diagnose adapter catalog integrity",
		Description: "Checks adapter-catalog installability (enabled adapters need a concrete fetch strategy; repo sources pinned), commercial-use policy (conditional adapters not enabled), and checksum metadata coherence.",
		Category:    "adapters",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"ok":       "bool",
				"findings": "array<CatalogFinding>",
			},
		},
		Examples: []module.Example{
			{Name: "Doctor catalog", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.adapters.AdaptersService/DoctorCatalog -H 'Content-Type: application/json' -d '{}'"},
		},
	},
}
