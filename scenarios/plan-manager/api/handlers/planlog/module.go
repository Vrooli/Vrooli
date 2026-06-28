// Package planlog is the API handler for the LogService — the execution-log
// ledger domain. It is the proto translation edge over internal/planlog; all
// business logic lives there behind seams.
package planlog

import (
	"context"
	"log"

	"plan-manager/internal/clock"
	internalexecution "plan-manager/internal/execution"
	"plan-manager/internal/module"
	internalplanlog "plan-manager/internal/planlog"
	internalplans "plan-manager/internal/plans"

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
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger) module.Module {
	plansSvc := internalplans.NewService(internalplans.Deps{
		Repo:  internalplans.NewSQLiteRepository(db, clk),
		Clock: clk,
	})
	execRepo := internalexecution.NewSQLiteRepository(db, clk)

	svc := internalplanlog.NewService(internalplanlog.Deps{
		Repo:     internalplanlog.NewSQLiteRepository(db, clk),
		Resolver: resolverAdapter{plans: plansSvc, executions: execRepo},
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

// resolverAdapter maps a `plan_or_execution` handle to the canonical plan id and
// (when the handle was an execution) the execution id. It checks the execution
// store first (an execution id binds the entry to a run + its plan), then falls
// back to resolving the handle as a plan id/slug through the plans SSOT.
type resolverAdapter struct {
	plans      internalplans.Service
	executions internalexecution.Repository
}

func (a resolverAdapter) Resolve(ctx context.Context, handle string) (planID, executionID string, ok bool, err error) {
	if e, found, gerr := a.executions.GetExecution(ctx, handle); gerr != nil {
		return "", "", false, gerr
	} else if found {
		return e.PlanID, e.ID, true, nil
	}
	plan, gerr := a.plans.Get(ctx, handle)
	if gerr != nil {
		// A non-resolving handle is not a hard error here — the service decides
		// whether an unresolved handle is acceptable (it is for filters).
		return "", "", false, nil
	}
	return plan.ID, "", true, nil
}

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
