// Package planlog is the API handler for the LogService — the execution-log
// ledger domain. It is the proto translation edge over internal/planlog; all
// business logic lives there behind seams.
package planlog

import (
	"log"

	"plan-manager/internal/clock"
	"plan-manager/internal/module"
	internalplanlog "plan-manager/internal/planlog"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	logconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/log/log_v1connect"
)

// Module returns the log domain's contribution to the API: the generated
// LogService Connect-RPC handler, backed by the log home store (the log_entries
// table), a Resolver over the plans SSOT + execution store (so a plan slug or
// execution id binds entries to the right scope), and the downstream bug/record
// sinks (the documented pending stubs in v1 — durable-local + retryable via
// `log sync`; no API-blocking auto-forward to absent downstreams).
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger, resolver internalplanlog.Resolver) module.Module {
	svc := internalplanlog.NewService(internalplanlog.Deps{
		Repo:     internalplanlog.NewSQLiteRepository(db, clk),
		Resolver: resolver,
		Bugs:     internalplanlog.DefaultBugReporter(),  // pending stub; retry via `log sync`
		Records:  internalplanlog.DefaultRecordWriter(), // pending stub; retry via `log sync`
		Clock:    clk,
	})

	connectPath, connectHandler := logconnect.NewLogServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "log",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema returns the log domain's SQL contribution.
func Schema() string { return internalplanlog.Schema() }

// Endpoints is the machine-readable description of the log module's public
// surface; one entry per RPC (the global parity test enforces this).
var Endpoints = []module.EndpointDescriptor{
	endpoint("log_add_decision", logconnect.LogServiceAddDecisionProcedure, "Record a decision", "Records an in-flow design decision in the plan log ledger."),
	endpoint("log_add_finding", logconnect.LogServiceAddFindingProcedure, "Record a candidate finding", "Records a CANDIDATE finding (a possible bug; never auto-promoted)."),
	endpoint("log_add_bug", logconnect.LogServiceAddBugProcedure, "File a bug report", "Files a bug report and forwards it downstream; the local entry stays durable if sync fails."),
	endpoint("log_add_record", logconnect.LogServiceAddRecordProcedure, "Capture a record", "Captures a reusable record and forwards it to Swarm Manager; durable if sync fails."),
	endpoint("log_add_note", logconnect.LogServiceAddNoteProcedure, "Record a note", "Records a lightweight progress/context note (local-only)."),
	endpoint("log_list", logconnect.LogServiceListEntriesProcedure, "List log entries", "Lists ledger entries for a plan/execution/phase with a compact summary."),
	endpoint("log_get", logconnect.LogServiceGetEntryProcedure, "Get a log entry", "Returns one ledger entry by id, including its downstream reference."),
	endpoint("log_update", logconnect.LogServiceUpdateEntryProcedure, "Update a log entry", "Edits an entry's title/detail/severity/triage and appends evidence."),
	endpoint("log_reassign", logconnect.LogServiceReassignEntryProcedure, "Reassign a log entry", "Moves an existing log entry to another phase by phase id or ordinal."),
	endpoint("log_promote", logconnect.LogServicePromoteEntryProcedure, "Promote a finding", "Promotes a finding into a bug report or record, preserving the original finding."),
	endpoint("log_sync", logconnect.LogServiceSyncEntryProcedure, "Retry downstream sync", "Retries downstream forwarding for a pending/failed bug or record entry."),
}

func endpoint(id, path, summary, description string) module.EndpointDescriptor {
	return module.EndpointDescriptor{
		ID:          id,
		Path:        path,
		Method:      "POST",
		Summary:     summary,
		Description: description,
		Category:    "log",
	}
}
