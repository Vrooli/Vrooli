// Package trials is the API handler for the TrialsService — the empirical
// local-model gate. It is the proto translation edge over internal/trials; all
// business logic lives in internal/trials behind seams.
package trials

import (
	"log"

	"meta-optimization-manager/internal/clock"
	internalcoverage "meta-optimization-manager/internal/coverage"
	"meta-optimization-manager/internal/module"
	internaltrials "meta-optimization-manager/internal/trials"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	trialsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/trials/trials_v1connect"
)

// Module returns the trials domain's contribution to the API: the generated
// TrialsService Connect-RPC handler, backed by the Guide-space task generator
// (sharing the coverage domain's space reader), the committed fixture corpus,
// the agent-manager sandboxed-spawn runner, the MoM-owned Evaluator, and the
// SQLite trials history + gate registry.
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger) module.Module {
	svc := internaltrials.NewService(internaltrials.Deps{
		Tasks:     internaltrials.NewTaskGenerator(internalcoverage.NewSpaceReader()),
		Fixtures:  internaltrials.NewFixtureResolver(),
		Runner:    internaltrials.NewRunner(),
		Evaluator: internaltrials.NewEvaluator(logger),
		Repo:      internaltrials.NewSQLiteRepository(db, clk),
		Clock:     clk,
	})
	connectPath, connectHandler := trialsconnect.NewTrialsServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "trials",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internaltrials.Schema so the modules registry collects
// endpoints and schema from one symbol per handler package.
func Schema() string { return internaltrials.Schema() }

// Endpoints is the machine-readable description of the trials module's public
// surface. The Connect-RPC method paths reference the generated *Procedure
// constants, so renaming an RPC in trials.proto breaks this at compile time; the
// global parity test asserts every RPC has exactly one entry.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "trials_list_tasks",
		Path:        trialsconnect.TrialsServiceListTrialTasksProcedure,
		Method:      "POST",
		Summary:     "List trial tasks",
		Description: "Returns the trial task suite generated from the Guide space (one task per Guide row + negative honesty cases), optionally filtered by suite.",
		Category:    "trials",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"tasks": "array<TrialTask>"},
		},
	},
	{
		ID:          "trials_run",
		Path:        trialsconnect.TrialsServiceRunTrialsProcedure,
		Method:      "POST",
		Summary:     "Run trials (explicit invocation)",
		Description: "Reconciles MoM's declared role-only profile, then dispatches a task or suite through Agent Manager's sandboxed runner, evaluates the produced diff against the fixture oracle, and records the runs (OT-P1-001). EXPLICIT INVOCATION ONLY — never auto-runs; always sandboxed.",
		Category:    "trials",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"runs": "array<TrialRun>"},
		},
	},
	{
		ID:          "trials_history",
		Path:        trialsconnect.TrialsServiceGetTrialHistoryProcedure,
		Method:      "POST",
		Summary:     "Trial history trend",
		Description: "Returns the success-rate / tokens / wall-time trend over time plus the most recent runs, optionally filtered by task or suite.",
		Category:    "trials",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"points": "array<TrialHistoryPoint>", "recent_runs": "array<TrialRun>"},
		},
	},
	{
		ID:          "trials_show",
		Path:        trialsconnect.TrialsServiceGetTrialRunProcedure,
		Method:      "POST",
		Summary:     "Show one trial run",
		Description: "Returns one run by id with its verdict, sandbox diff ref, tokens, and wall-time.",
		Category:    "trials",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"run": "TrialRun"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No run with that id"},
		},
	},
	{
		ID:          "trials_gate_coverage",
		Path:        trialsconnect.TrialsServiceGetGateCoverageProcedure,
		Method:      "POST",
		Summary:     "Guide-gate coverage",
		Description: "Returns the % of Guide-space tasks that have at least one live empirical gate (the recursive Guide-coverage metric).",
		Category:    "trials",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"guide_tasks_total": "int", "guide_tasks_with_gate": "int", "gate_coverage_ratio": "double"},
		},
	},
}
