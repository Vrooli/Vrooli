package models

import (
	"context"
	"log"

	internaladapters "image-tools/internal/adapters"
	internalbackends "image-tools/internal/backends"
	internalcaps "image-tools/internal/capabilities"
	internalmodels "image-tools/internal/models"
	"image-tools/internal/module"
	"image-tools/internal/verdicts"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	modelsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models/models_v1connect"
)

// Module returns the models domain's contribution: the generated Connect-RPC
// ModelsService handler over the declarative registry. The registry (validated
// seed catalog) is loaded once in main.go and shared; the enabled-state overlay
// is persisted in SQLite via the models store.
// AdapterSeams supplies the conditioning seams ExplainResolution's --explain
// surface uses to validate + report a requested adapter stack (decision C5). All
// fields nil ⇒ ExplainResolution rejects any request that names an adapter.
type AdapterSeams struct {
	ByID      func(id string) (internaladapters.Adapter, bool)
	Enabled   func(ctx context.Context) (func(id string) bool, error)
	Installed func(id string) bool
}

func Module(db *database.RoutedDB, reg *internalmodels.Registry, previewer *internalmodels.Previewer, verdictSvc *verdicts.Service, probe internalcaps.Probe, installer *internalmodels.Installer, backendReg *internalbackends.Registry, jobs JobSubmitter, estimateInstallSeconds EstimateInstallSecondsFunc, ensurer BackendEnsurer, adapterSeams AdapterSeams, rightsEvaluator internalmodels.RightsEvaluator, storage ModelStorage, capacityManagedGPU bool, consent ConsentRule, logger *log.Logger) module.Module {
	store := internalmodels.NewStore(db)
	connectPath, connectHandler := modelsconnect.NewModelsServiceHandler(NewConnectHandler(Deps{
		CapacityManagedGPU:     capacityManagedGPU,
		Registry:               reg,
		Store:                  store,
		Verdicts:               verdictSvc,
		Previewer:              previewer,
		Probe:                  probe,
		Installer:              installer,
		Backends:               backendReg,
		RightsEvaluator:        rightsEvaluator,
		Jobs:                   jobs,
		OpDefaults:             internalmodels.NewOpDefaultStore(db),
		EstimateInstallSeconds: estimateInstallSeconds,
		Ensurer:                ensurer,
		AdapterByID:            adapterSeams.ByID,
		AdapterEnabled:         adapterSeams.Enabled,
		AdapterInstalled:       adapterSeams.Installed,
		Storage:                storage,
		Consent:                consent,
		Logger:                 logger,
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
		ID: "models_provision", Path: modelsconnect.ModelsServiceProvisionModelsProcedure, Method: "POST",
		Summary: "Provision local model capabilities", Category: "models",
		Description: "Prepares a minimal host-fit model set for every effectively enabled capability, including private runtime, backend tools, model artifacts and load-smoke evidence in one durable timed job. Unsupported or unready capability routes remain visible as gaps and keep the overall readiness result incomplete. Explicit model_ids prepares only the requested models.",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"model_ids": "array<string> (optional)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"job_id": "string", "model_ids": "array<string>"}},
		Errors:      []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "Unknown model"}, {Status: 412, Code: "failed_precondition", Description: "Model is not commercial-clean local content"}},
	},
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
	},
	{
		ID:          "models_override_clear",
		Path:        modelsconnect.ModelsServiceClearModelOverrideProcedure,
		Method:      "POST",
		Summary:     "Clear a model's override",
		Description: "Deletes the model's enabled-state override so it follows the seed again.",
		Category:    "models",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"model": "Model"}},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No such model, or the model has no override"},
		},
		Examples: []module.Example{
			{Name: "Follow the seed again", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/ClearModelOverride -H 'Content-Type: application/json' -d '{\"id\":\"restoreformer-plus-plus\"}'"},
		},
	},
	{
		ID:          "models_set_enabled",
		Path:        modelsconnect.ModelsServiceSetModelEnabledProcedure,
		Method:      "POST",
		Summary:     "Enable or disable a model",
		Description: "Records an enabled-state override over the read-only seed catalog, with who set it, when, why and whether it is pinned. An unpinned override stops applying when the seed's enabled flag later changes (it is marked superseded); a pinned one always wins. Models whose rights the evaluator refuses for local use cannot be enabled.",
		Category:    "models",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"id":      "string",
				"enabled": "bool",
				"actor":   "string (optional; who is setting it, default \"api\")",
				"reason":  "string (optional; required when pinned)",
				"pinned":  "bool (optional; keep the override against later seed changes)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"model": "Model"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No model with that id exists"},
			{Status: 412, Code: "failed_precondition", Description: "Model commercial rights are not verified"},
			{Status: 500, Code: "internal", Description: "Model-state persistence failure"},
		},
		Examples: []module.Example{
			{Name: "Enable a quality tier", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/SetModelEnabled -H 'Content-Type: application/json' -d '{\"id\":\"real-esrgan\",\"enabled\":true}'"},
		},
	},
	{
		ID:          "models_blocklist",
		Path:        modelsconnect.ModelsServiceListBlocklistProcedure,
		Method:      "POST",
		Summary:     "List license gates and exclusions",
		Description: "Returns every catalog gate: license gates refuse only the listed monetized, hosted or redistributed uses (normal local use is allowed); exclusions keep a model out of the catalog for a safety, technical or hardware-fit reason.",
		Category:    "models",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"entries": "array<BlocklistEntry>"},
		},
		Examples: []module.Example{
			{Name: "List blocklist", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/ListBlocklist -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "models_doctor",
		Path:        modelsconnect.ModelsServiceDoctorCatalogProcedure,
		Method:      "POST",
		Summary:     "Diagnose model catalog integrity",
		Description: "Checks seed catalog installability, operation coverage, license-gate agreement (informational for normal use), exclusion overlap, and checksum/source metadata coherence.",
		Category:    "models",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"ok":       "bool",
				"findings": "array<CatalogFinding>",
			},
		},
		Examples: []module.Example{
			{Name: "Doctor catalog", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/DoctorCatalog -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "backends_doctor",
		Path:        modelsconnect.ModelsServiceDoctorBackendsProcedure,
		Method:      "POST",
		Summary:     "Diagnose backend software availability",
		Description: "Checks registered inference backends for host software readiness (PATH/module presence), overlays enabled catalog backend families that have no runtime provider yet, and returns provisioning guidance. Hardware fit remains reported by model selection.",
		Category:    "models",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"ok":       "bool",
				"backends": "array<BackendStatus>",
			},
		},
		Examples: []module.Example{
			{Name: "Doctor backends", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/DoctorBackends -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "backends_ensure",
		Path:        modelsconnect.ModelsServiceEnsureBackendProcedure,
		Method:      "POST",
		Summary:     "Install a missing host-tool backend on demand",
		Description: "Ensures a backend's host tool is installed. Already-present, manual, and capability-gated (not-applicable) tools return immediately with guidance and no job. A fetchable tool submits a durable job (shells `vrooli host install <tool> --json`) and returns a job id + ETA; block once on jobs wait. Pass tool or operation.",
		Category:    "models",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"tool": "string", "operation": "string", "dry_run": "bool"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"tool":              "string",
				"job_id":            "string (empty when no job submitted)",
				"eta_seconds":       "int",
				"already_installed": "bool",
				"manual":            "bool",
				"state":             "string",
				"detail":            "string",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing/unknown tool or operation with no host-tool backend"},
			{Status: 500, Code: "internal", Description: "Probe or job submission failure"},
			{Status: 501, Code: "unimplemented", Description: "Backend provisioning unavailable (read-only wiring)"},
		},
		Examples: []module.Example{
			{Name: "Ensure realesrgan", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/EnsureBackend -H 'Content-Type: application/json' -d '{\"tool\":\"realesrgan-ncnn-vulkan\"}'"},
		},
	},
	{
		ID:          "models_host_summary",
		Path:        modelsconnect.ModelsServiceGetHostSummaryProcedure,
		Method:      "POST",
		Summary:     "Get this host's AI-relevant hardware summary",
		Description: "Returns the machine's GPU name + total/free VRAM, CPU cores, RAM, and os/arch — the snapshot the model-catalog UI uses to render hardware-fit affirmatively (\"Runs on your GPU\") instead of a static requirement chip.",
		Category:    "models",
		Request:     &module.Schema{Type: "object"},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"host": "HostSummary (gpu name/vram/cores/ram/os/arch)"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Host probe failure"},
		},
		Examples: []module.Example{
			{Name: "Host summary", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/GetHostSummary -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "models_operation_candidates",
		Path:        modelsconnect.ModelsServiceListOperationModelsProcedure,
		Method:      "POST",
		Summary:     "List host-aware candidate models for an operation",
		Description: "Returns every model serving an operation, each annotated for this host: hardware fit (will it run, on GPU or CPU, or not at all), backend readiness (is the host program/weights provisioned, and is the install one-click or manual), and a single ready_state the model picker styles on. The data source behind the in-product model picker — unlike SelectModel it returns the full menu (including models that cannot run here) so the picker is transparent about why a model was chosen and what each alternative needs.",
		Category:    "models",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"operation": "string"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"operation":       "string",
				"host":            "HostSummary (gpu name/vram/cores/ram)",
				"candidates":      "array<CandidateModel> (model + fit + backend + ready_state)",
				"selected_id":     "string",
				"selected_reason": "string",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Operation missing or not in the vocabulary"},
			{Status: 500, Code: "internal", Description: "Host probe / model-state load failure"},
		},
		Examples: []module.Example{
			{Name: "Candidates for text_to_image", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/ListOperationModels -H 'Content-Type: application/json' -d '{\"operation\":\"text_to_image\"}'"},
		},
	},
	{
		ID:          "models_explain_resolution",
		Path:        modelsconnect.ModelsServiceExplainResolutionProcedure,
		Method:      "POST",
		Summary:     "Explain the operation→model→technique resolution (dry-run)",
		Description: "Returns the explicit Resolution for an operation on this host — which model would run, whether it serves the op natively or via a derived technique (with the quality caveat), the backend tier, GPU viability, and the operation's safety consent weight — without executing anything. The read-only --explain / dry-run surface over the same resolver the AI submit edge pins into a job.",
		Category:    "models",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"operation":  "string (required; registry op)",
				"model_id":   "string (optional; force a model id)",
				"allow_byok": "bool (optional; permit a paid BYOK cloud tier)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"resolution": "Resolution (model/support/technique/pipeline_class/caveat/weight/tier/gpu_viable/warnings)",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Unknown operation or invalid model id"},
			{Status: 412, Code: "failed_precondition", Description: "No enabled model can run for the operation on this host"},
			{Status: 500, Code: "internal", Description: "Host probe or model-state load failure"},
		},
		Examples: []module.Example{
			{Name: "Explain inpaint", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/ExplainResolution -H 'Content-Type: application/json' -d '{\"operation\":\"inpaint\"}'"},
		},
	},
	{
		ID:          "models_install",
		Path:        modelsconnect.ModelsServiceInstallModelProcedure,
		Method:      "POST",
		Summary:     "Install or runtime-validate a model",
		Description: "Downloads missing weights or revalidates the load-smoke for an installed model through the durable install job. Checksums are pinned on first download and free disk space is checked first. Models without verified commercial rights cannot be installed. Returns a job id + ETA; block once on jobs wait. A runtime-ready installed model returns already_installed=true with no job.",
		Category:    "models",
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
			{Status: 404, Code: "not_found", Description: "No model with that id exists"},
			{Status: 412, Code: "failed_precondition", Description: "Model commercial rights are not verified, or installed runtime is not provisioned or cannot be smoke-validated"},
			{Status: 500, Code: "internal", Description: "Install job submission failure"},
			{Status: 501, Code: "unimplemented", Description: "Installation unavailable (read-only wiring)"},
		},
		Examples: []module.Example{
			{Name: "Install sd-1.5", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/InstallModel -H 'Content-Type: application/json' -d '{\"id\":\"sd-1.5\"}'"},
		},
	},
	{
		ID:          "models_remove",
		Path:        modelsconnect.ModelsServiceRemoveModelProcedure,
		Method:      "POST",
		Summary:     "Remove a model's weights",
		Description: "Deletes a model's downloaded weights and clears its install record. A custom local model is unlinked; its referenced path is never deleted.",
		Category:    "models",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"removed": "bool"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No model with that id exists"},
			{Status: 500, Code: "internal", Description: "Weight removal failure"},
		},
		Examples: []module.Example{
			{Name: "Remove sd-1.5", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/RemoveModel -H 'Content-Type: application/json' -d '{\"id\":\"sd-1.5\"}'"},
		},
	},
	{
		ID:          "models_storage",
		Path:        modelsconnect.ModelsServiceGetModelStorageProcedure,
		Method:      "POST",
		Summary:     "Report model-weight storage and the adaptive offload plan",
		Description: "Returns free-disk headroom above the shared disk floor, the adaptive idle threshold it implies, every installed weight-backed model ranked by offload value with its protections and selection, and recent offload receipts. Read-only.",
		Category:    "models",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"receipt_limit": "int32 (optional; default 20)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"report": "ModelStorageReport", "receipts": "array<ModelOffloadReceipt>"}},
		Errors:      []module.ErrorDesc{{Status: 500, Code: "internal", Description: "Storage facts could not be read"}, {Status: 501, Code: "unimplemented", Description: "Model storage management is not wired"}},
		Examples: []module.Example{
			{Name: "Storage status", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/GetModelStorage -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "models_offload",
		Path:        modelsconnect.ModelsServiceOffloadModelsProcedure,
		Method:      "POST",
		Summary:     "Dry-run or apply the adaptive model offload",
		Description: "Evaluates the adaptive offload policy. Without apply it returns the plan only; with apply it removes the selected weights through the installer and writes one durable receipt per attempt. model_ids restricts the pass to named models (bypassing the idle threshold, never a protection). need_bytes previews making room for an acquisition.",
		Category:    "models",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"apply": "bool (default false = dry-run)", "model_ids": "array<string> (optional)", "need_bytes": "uint64 (optional)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"report": "ModelStorageReport", "receipts": "array<ModelOffloadReceipt>", "applied": "bool"}},
		Errors:      []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "A named model does not exist"}, {Status: 500, Code: "internal", Description: "The policy could not be evaluated"}},
		Examples: []module.Example{
			{Name: "Dry-run", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/OffloadModels -H 'Content-Type: application/json' -d '{}'"},
			{Name: "Offload one idle model", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/OffloadModels -H 'Content-Type: application/json' -d '{\"apply\":true,\"model_ids\":[\"dncnn\"]}'"},
		},
	},
	{
		ID:          "models_keep",
		Path:        modelsconnect.ModelsServiceSetModelKeepProcedure,
		Method:      "POST",
		Summary:     "Pin or unpin a model against offload",
		Description: "keep=true records an operator keep pin so neither the adaptive policy nor a targeted offload removes the model's weights; keep=false removes the pin.",
		Category:    "models",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string", "keep": "bool", "reason": "string (optional)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"id": "string", "kept": "bool"}},
		Errors:      []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "No model with that id exists"}, {Status: 500, Code: "internal", Description: "The pin could not be stored"}},
		Examples: []module.Example{
			{Name: "Keep sd-1.5", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/SetModelKeep -H 'Content-Type: application/json' -d '{\"id\":\"sd-1.5\",\"keep\":true}'"},
		},
	},
	{
		ID:          "models_add_custom",
		Path:        modelsconnect.ModelsServiceAddCustomModelProcedure,
		Method:      "POST",
		Summary:     "Register a custom/local model",
		Description: "Registers a custom or fine-tuned local model merged on top of the read-only seed. The id must not collide with a seed model; a declared local path is verified to exist.",
		Category:    "models",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"model":        "Model (id, operations, backend required)",
				"local_path":   "string (optional; local weights — installed by reference)",
				"download_url": "string (optional; remote source when local_path empty)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"model": "Model"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing id, seed-id collision, or missing local path"},
			{Status: 500, Code: "internal", Description: "Custom-entry persistence failure"},
		},
		Examples: []module.Example{
			{Name: "Register a local upscaler", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/AddCustomModel -H 'Content-Type: application/json' -d '{\"model\":{\"id\":\"my-upscaler\",\"operations\":[\"upscale\"],\"backend\":\"onnxruntime\"},\"local_path\":\"/data/models/my-upscaler\"}'"},
		},
	},
	{
		ID:          "models_inspect_source",
		Path:        modelsconnect.ModelsServiceInspectModelSourceProcedure,
		Method:      "POST",
		Summary:     "Inspect an import source (dry run)",
		Description: "Dry-runs a guided import: inspects a HuggingFace repo id, direct weight URL, or local path WITHOUT installing anything, returning the detected layout, inferred architecture (with confidence + evidence the user confirms), license/NSFW, approximate size, the operations the entry would offer, and the proposed registry entry.",
		Category:    "models",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"source": "string (HF repo id, URL, or local path)"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"source":             "string",
				"repo_id":            "string",
				"revision":           "string",
				"layout":             "ModelLayout",
				"architecture":       "ArchitectureInference (architecture, confidence, evidence)",
				"license":            "string",
				"nsfw":               "bool",
				"size_bytes":         "uint64",
				"pipeline_class":     "string",
				"offered_operations": "array<string>",
				"proposed":           "Model",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing/unresolvable source"},
			{Status: 501, Code: "unimplemented", Description: "Import unavailable (read-only wiring)"},
		},
		Examples: []module.Example{
			{Name: "Inspect an SDXL checkpoint", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/InspectModelSource -H 'Content-Type: application/json' -d '{\"source\":\"stabilityai/stable-diffusion-xl-base-1.0\"}'"},
		},
	},
	{
		ID:          "models_import",
		Path:        modelsconnect.ModelsServiceImportModelProcedure,
		Method:      "POST",
		Summary:     "Import a model by source (confirm + install)",
		Description: "Composes inspect → operator-confirmed fields (architecture/operations/attestation) → an add-only custom entry → a durable install job (mirrors InstallModel: returns job id + ETA; block once on jobs wait). attest_commercial_rights is optional: an unattested import runs locally and keeps hosted, monetized and redistributed use refused; attesting records the rights claim and lifts that. A diffusers repo finds its adapter from its pipeline class; one with no adapter is refused with the missing piece named.",
		Category:    "models",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"source":                   "string (HF repo id, URL, or local path)",
				"id":                       "string (new entry id; must not collide with a seed model)",
				"name":                     "string (optional display name)",
				"architecture":             "string (required when inference is none; else overrides)",
				"operations":               "array<string> (optional; defaults to the architecture's base ops)",
				"attest_commercial_rights": "bool (optional; lifts the hosted/monetized-use refusal; decision D4)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"model":             "Model",
				"job_id":            "string (empty when already installed or no job runner)",
				"eta_seconds":       "int",
				"already_installed": "bool",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing source/id, unresolved architecture, or seed-id collision"},
			{Status: 500, Code: "internal", Description: "Persistence or job submission failure"},
			{Status: 501, Code: "unimplemented", Description: "Custom models / import unavailable (read-only wiring)"},
		},
		Examples: []module.Example{
			{Name: "Import an SDXL checkpoint", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/ImportModel -H 'Content-Type: application/json' -d '{\"source\":\"stabilityai/stable-diffusion-xl-base-1.0\",\"id\":\"imported-sdxl\",\"architecture\":\"sdxl\"}'"},
		},
	},
	{
		ID:          "models_set_default",
		Path:        modelsconnect.ModelsServiceSetDefaultModelProcedure,
		Method:      "POST",
		Summary:     "Pin the default model for an operation",
		Description: "Pins (or clears, with empty model_id) the default model for an operation. Selection applies the pin when a request gives no explicit override. Models without verified commercial rights cannot be pinned.",
		Category:    "models",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"operation": "string",
				"model_id":  "string (empty clears the pin)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"operation": "string", "model_id": "string"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Unknown operation or model does not serve it"},
			{Status: 404, Code: "not_found", Description: "No model with that id exists"},
			{Status: 412, Code: "failed_precondition", Description: "Model commercial rights are not verified"},
			{Status: 500, Code: "internal", Description: "Default persistence failure"},
		},
		Examples: []module.Example{
			{Name: "Pin upscale default", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/SetDefaultModel -H 'Content-Type: application/json' -d '{\"operation\":\"upscale\",\"model_id\":\"real-esrgan\"}'"},
		},
	},
	{
		ID:          "models_list_defaults",
		Path:        modelsconnect.ModelsServiceListDefaultsProcedure,
		Method:      "POST",
		Summary:     "List per-operation default models",
		Description: "Returns the effective default model per operation, marking whether it comes from the seed or an operator pin.",
		Category:    "models",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"defaults": "array<OpDefault>"},
		},
		Examples: []module.Example{
			{Name: "List defaults", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.models.ModelsService/ListDefaults -H 'Content-Type: application/json' -d '{}'"},
		},
	},
}
