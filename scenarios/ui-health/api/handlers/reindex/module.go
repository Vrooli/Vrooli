package reindex

import (
	"log"

	"ui-health/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	reindexv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/reindex"
	reindexconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/reindex/reindex_v1connect"
)

var ProtoFile = reindexv1.File_ui_health_v1_reindex_reindex_proto

func Module(logger *log.Logger, reindexer Reindexer) module.Module {
	connectPath, connectHandler := reindexconnect.NewReindexServiceHandler(NewConnectHandler(Deps{
		Logger:    logger,
		Reindexer: reindexer,
	}))
	return module.Module{
		Name: "reindex",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "reindex_run",
		Path:        reindexconnect.ReindexServiceReindexProcedure,
		Method:      "POST",
		Summary:     "Start a reindex job",
		Description: "Queues a reconcile job that upserts/deletes points in the ui-health-surface Qdrant collection.",
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
		Examples: []module.Example{
			{Name: "Reindex (dry-run)", Curl: "curl http://localhost:${API_PORT}/vrooli.ui_health.v1.reindex.ReindexService/Reindex -H 'Content-Type: application/json' -d '{\"dry_run\":true}'"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "ui-health reindex run",
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
		Examples: []module.Example{
			{Name: "Job status", Curl: "curl http://localhost:${API_PORT}/vrooli.ui_health.v1.reindex.ReindexService/ReindexStatus -H 'Content-Type: application/json' -d '{\"job_id\":\"job-abc\"}'"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "ui-health reindex status",
			Args:    []string{"<job_id>"},
		},
	},
	{
		ID:          "reindex_cancel",
		Path:        reindexconnect.ReindexServiceReindexCancelProcedure,
		Method:      "POST",
		Summary:     "Cancel a reindex job",
		Description: "Requests cooperative cancellation of an in-flight job.",
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
		Examples: []module.Example{
			{Name: "Cancel job", Curl: "curl http://localhost:${API_PORT}/vrooli.ui_health.v1.reindex.ReindexService/ReindexCancel -H 'Content-Type: application/json' -d '{\"job_id\":\"job-abc\"}'"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "ui-health reindex cancel",
			Args:    []string{"<job_id>"},
		},
	},
}
