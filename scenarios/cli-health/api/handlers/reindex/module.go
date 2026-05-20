package reindex

import (
	"log"

	"cli-health/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	reindexv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/reindex"
	reindexconnect "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/reindex/reindex_v1connect"
)

// ProtoFile exposes the reindex domain's proto FileDescriptor so the global
// parity test can walk it without importing the gen/go package directly.
var ProtoFile = reindexv1.File_cli_health_v1_reindex_reindex_proto

// Module returns the reindex domain's contribution to the API: the Connect
// ReindexService handler mounted at the generated procedure paths.
func Module(logger *log.Logger) module.Module {
	connectPath, connectHandler := reindexconnect.NewReindexServiceHandler(NewConnectHandler(Deps{
		Logger: logger,
	}))
	return module.Module{
		Name: "reindex",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — Phase 1 stub keeps reindex stateless. Phase 3 adds
// a jobs table for status polling and crash recovery.
func Schema() string { return "" }

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "reindex_run",
		Path:        reindexconnect.ReindexServiceReindexProcedure,
		Method:      "POST",
		Summary:     "Start a reindex job",
		Description: "Queues a reconcile job that upserts/deletes points in Qdrant. Phase 1 stub returns Unimplemented.",
		Category:    "reindex",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario": "string (optional; empty means all)",
				"dry_run":  "boolean",
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
			{Status: 501, Code: "unimplemented", Description: "Phase 1 stub; Phase 3 wires the orchestrator"},
		},
		Examples: []module.Example{
			{Name: "Reindex (dry-run)", Curl: "curl http://localhost:${API_PORT}/vrooli.cli_health.v1.reindex.ReindexService/Reindex -H 'Content-Type: application/json' -d '{\"dry_run\":true}'"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "cli-health reindex run",
			Args:    []string{"[--scenario", "<name>]", "[--dry-run]"},
		},
	},
	{
		ID:          "reindex_status",
		Path:        reindexconnect.ReindexServiceReindexStatusProcedure,
		Method:      "POST",
		Summary:     "Poll reindex job status",
		Description: "Returns the progress, state, and (on failure) error for a previously-started reindex job.",
		Category:    "reindex",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"job_id": "string (required)"},
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
			{Status: 501, Code: "unimplemented", Description: "Phase 1 stub; Phase 3 wires the job store"},
		},
		Examples: []module.Example{
			{Name: "Job status", Curl: "curl http://localhost:${API_PORT}/vrooli.cli_health.v1.reindex.ReindexService/ReindexStatus -H 'Content-Type: application/json' -d '{\"job_id\":\"job-abc\"}'"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "cli-health reindex status",
			Args:    []string{"<job_id>"},
		},
	},
	{
		ID:          "reindex_cancel",
		Path:        reindexconnect.ReindexServiceReindexCancelProcedure,
		Method:      "POST",
		Summary:     "Cancel a reindex job",
		Description: "Requests cooperative cancellation of an in-flight job. Idempotent: cancelling a terminal job returns cancelled=false.",
		Category:    "reindex",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"job_id": "string (required)"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"job_id":    "string",
				"cancelled": "boolean",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 501, Code: "unimplemented", Description: "Phase 1 stub; Phase 3 wires the orchestrator"},
		},
		Examples: []module.Example{
			{Name: "Cancel job", Curl: "curl http://localhost:${API_PORT}/vrooli.cli_health.v1.reindex.ReindexService/ReindexCancel -H 'Content-Type: application/json' -d '{\"job_id\":\"job-abc\"}'"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "cli-health reindex cancel",
			Args:    []string{"<job_id>"},
		},
	},
}
