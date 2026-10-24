package searchcontrol

import (
	"log"

	"business-health/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control"
	controlconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control/control_v1connect"
)

// ProtoFile exposes the shared control domain's proto FileDescriptor so the
// global parity test can walk it without importing the gen/go package directly.
var ProtoFile = controlv1.File_search_hub_v1_control_control_proto

// Module returns the search-control domain's contribution to the API: the Connect
// SearchControlService handler (the shared, token-gated reindex + config-write
// control plane) mounted at the generated procedure paths.
func Module(logger *log.Logger, deps Deps) module.Module {
	connectPath, connectHandler := controlconnect.NewSearchControlServiceHandler(NewConnectHandler(deps))
	return module.Module{
		Name: "searchcontrol",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — reindex job state is held in-process; the index itself
// lives in Qdrant and the tuning SSOT in search.json, not the scenario database.
func Schema() string { return "" }

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "searchcontrol_reindex",
		Path:        controlconnect.SearchControlServiceReindexProcedure,
		Method:      "POST",
		Summary:     "Start a reindex job (token-gated)",
		Description: "Queues a reconcile job that upserts/deletes points in Qdrant. Requires the provider's control token and the enabled control plane.",
		Category:    "searchcontrol",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scope":         "string (optional provider sub-scope; empty means the whole corpus)",
				"dry_run":       "boolean",
				"control_token": "string (required)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"job_id":          "string",
				"planned_upserts": "int32",
				"planned_deletes": "int32",
				"dry_run":         "boolean",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 403, Code: "permission_denied", Description: "Control plane disabled or control token mismatch"},
			{Status: 501, Code: "unimplemented", Description: "Returned only when no reindex backend is configured"},
		},
		Examples: []module.Example{
			{Name: "Reindex (dry-run)", Curl: "curl http://localhost:${API_PORT}/vrooli.search_hub.v1.control.SearchControlService/Reindex -H 'Content-Type: application/json' -d '{\"dry_run\":true,\"control_token\":\"<token>\"}'"},
		},
	},
	{
		ID:          "searchcontrol_reindex_status",
		Path:        controlconnect.SearchControlServiceReindexStatusProcedure,
		Method:      "POST",
		Summary:     "Poll reindex job status (token-gated)",
		Description: "Returns the progress, state, and (on failure) error for a previously-started reindex job.",
		Category:    "searchcontrol",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"job_id": "string (required)", "control_token": "string (required)"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"job_id":    "string",
				"state":     "string",
				"processed": "int32",
				"total":     "int32",
				"error":     "string",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 403, Code: "permission_denied", Description: "Control plane disabled or control token mismatch"},
			{Status: 404, Code: "not_found", Description: "No reindex job with that id"},
		},
		Examples: []module.Example{
			{Name: "Job status", Curl: "curl http://localhost:${API_PORT}/vrooli.search_hub.v1.control.SearchControlService/ReindexStatus -H 'Content-Type: application/json' -d '{\"job_id\":\"job-abc\",\"control_token\":\"<token>\"}'"},
		},
	},
	{
		ID:          "searchcontrol_reindex_cancel",
		Path:        controlconnect.SearchControlServiceReindexCancelProcedure,
		Method:      "POST",
		Summary:     "Cancel a reindex job (token-gated)",
		Description: "Requests cooperative cancellation of an in-flight job. Idempotent: cancelling a terminal job returns cancelled=false.",
		Category:    "searchcontrol",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"job_id": "string (required)", "control_token": "string (required)"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"job_id":    "string",
				"cancelled": "boolean",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 403, Code: "permission_denied", Description: "Control plane disabled or control token mismatch"},
		},
		Examples: []module.Example{
			{Name: "Cancel job", Curl: "curl http://localhost:${API_PORT}/vrooli.search_hub.v1.control.SearchControlService/ReindexCancel -H 'Content-Type: application/json' -d '{\"job_id\":\"job-abc\",\"control_token\":\"<token>\"}'"},
		},
	},
	{
		ID:          "searchcontrol_write_config",
		Path:        controlconnect.SearchControlServiceWriteConfigProcedure,
		Method:      "POST",
		Summary:     "Write a tuning block back to search.json (token-gated)",
		Description: "Validates a new tuning block and atomically rewrites the provider's search.json, triggering a reindex when an index-time factor changed. The search-hub sweep's write-back verb; not an interactive operator command.",
		Category:    "searchcontrol",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"provider_id":   "string (required; the leaf whose tuning is rewritten)",
				"tuning":        "object (full replacement tuning block)",
				"dry_run":       "boolean (validate + diff without writing)",
				"control_token": "string (required)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"written":           "boolean",
				"reindex_triggered": "boolean",
				"reindex_job_id":    "string",
				"effective":         "object (tuning now in effect)",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing provider_id or invalid tuning factor"},
			{Status: 403, Code: "permission_denied", Description: "Control plane disabled or control token mismatch"},
			{Status: 404, Code: "not_found", Description: "provider_id not present in search.json"},
		},
		Examples: []module.Example{
			{Name: "Write config (dry-run)", Curl: "curl http://localhost:${API_PORT}/vrooli.search_hub.v1.control.SearchControlService/WriteConfig -H 'Content-Type: application/json' -d '{\"provider_id\":\"business-health.commands\",\"dry_run\":true,\"control_token\":\"<token>\",\"tuning\":{\"engine\":\"dense\"}}'"},
		},
	},
	{
		ID:          "searchcontrol_write_corpus",
		Path:        controlconnect.SearchControlServiceWriteCorpusProcedure,
		Method:      "POST",
		Summary:     "Write an evaluation corpus back to search.json (token-gated)",
		Description: "Validates a new evaluation corpus (the tests block, carried as an eval EvalSuite) and atomically rewrites ONLY that block in the provider's search.json — the corpus twin of WriteConfig. The search-hub optimizer's corpus write-back verb (e.g. `evals generate --apply`); not an interactive operator command. Never triggers a reindex.",
		Category:    "searchcontrol",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"provider_id":   "string (required; the leaf whose corpus is rewritten)",
				"corpus":        "object (full replacement tests block, as an eval EvalSuite)",
				"dry_run":       "boolean (validate + diff without writing)",
				"control_token": "string (required)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"written":   "boolean",
				"effective": "object (corpus now in effect, as an eval EvalSuite)",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing provider_id or a malformed corpus (case missing id/query, duplicate id)"},
			{Status: 403, Code: "permission_denied", Description: "Control plane disabled or control token mismatch"},
			{Status: 404, Code: "not_found", Description: "provider_id not present in search.json"},
		},
		Examples: []module.Example{
			{Name: "Write corpus (dry-run)", Curl: "curl http://localhost:${API_PORT}/vrooli.search_hub.v1.control.SearchControlService/WriteCorpus -H 'Content-Type: application/json' -d '{\"provider_id\":\"business-health.commands\",\"dry_run\":true,\"control_token\":\"<token>\",\"corpus\":{\"suite_id\":\"business-health.commands.primary\",\"cases\":[]}}'"},
		},
	},
}
