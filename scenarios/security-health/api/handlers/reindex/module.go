package reindex

import (
	"fmt"
	"log"

	"security-health/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	reindexv1 "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/reindex"
	reindexconnect "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/reindex/reindex_v1connect"
)

// ProtoFile exposes the reindex domain's proto FileDescriptor for the global
// parity test.
var ProtoFile = reindexv1.File_security_health_v1_reindex_reindex_proto

// errUnknownJob is the shared not-found error for status/cancel.
func errUnknownJob(jobID string) error { return fmt.Errorf("reindex job %q not found", jobID) }

// Module mounts the ReindexService handler, sharing the dependencies.Service
// instance with the dependencies module + reconcile loop.
func Module(logger *log.Logger, svc Reindexer) module.Module {
	connectPath, connectHandler := reindexconnect.NewReindexServiceHandler(NewConnectHandler(Deps{
		Logger:  logger,
		Service: svc,
	}))
	return module.Module{
		Name: "reindex",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — reindex owns no tables (it drives the dependencies corpus).
func Schema() string { return "" }

// Endpoints describes the reindex module's public surface.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "reindex_reindex",
		Path:        reindexconnect.ReindexServiceReindexProcedure,
		Method:      "POST",
		Summary:     "Reconcile the fleet dependency corpus (async)",
		Description: "Re-discovers every scenario's lockfiles, re-annotates vuln status, and reconciles the SBOM corpus. dry_run returns planned upsert/delete counts without writing; a real run starts an async job and returns its job_id.",
		Category:    "reindex",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"scenario": "string (empty = all)", "dry_run": "boolean"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"job_id": "string", "planned_upserts": "int32", "planned_deletes": "int32", "dry_run": "boolean"},
		},
		Examples: []module.Example{
			{Name: "Dry-run reindex", Curl: "curl http://localhost:${API_PORT}/vrooli.security_health.v1.reindex.ReindexService/Reindex -H 'Content-Type: application/json' -d '{\"dry_run\":true}'"},
		},
	},
	{
		ID:          "reindex_status",
		Path:        reindexconnect.ReindexServiceReindexStatusProcedure,
		Method:      "POST",
		Summary:     "Poll a reindex job",
		Description: "Reports progress (processed/total) and terminal state for a reindex job.",
		Category:    "reindex",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"job_id": "string"}},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"job_id": "string", "state": "string", "processed": "int32", "total": "int32", "error": "string"},
		},
		Errors: []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "No job with that id"}},
	},
	{
		ID:          "reindex_cancel",
		Path:        reindexconnect.ReindexServiceReindexCancelProcedure,
		Method:      "POST",
		Summary:     "Cancel a reindex job",
		Description: "Requests cooperative cancellation of a reindex job. Idempotent; cancelling a terminal job reports cancelled=false.",
		Category:    "reindex",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"job_id": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"job_id": "string", "cancelled": "boolean"}},
		Errors:      []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "No job with that id"}},
	},
}
