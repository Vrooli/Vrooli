// Package execution is the API handler for the ExecutionService — the
// guided-runner domain. It is the proto translation edge over
// internal/execution; all business logic lives there behind seams.
package execution

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"plan-manager/internal/clock"
	internalexecution "plan-manager/internal/execution"
	"plan-manager/internal/module"
	internalplans "plan-manager/internal/plans"
	internalvalidation "plan-manager/internal/validation"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	executionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/execution/execution_v1connect"
)

// Module returns the execution domain's contribution to the API: the generated
// ExecutionService Connect-RPC handler, backed by the execution home store (the
// executions/decisions/findings/handoffs/velocity tables), the plans SSOT (the
// PlanStore seam — read + delegated phase transitions), the validation domain
// (the Validator seam — last validation + staleness, degrading to UNKNOWN), and
// the stubbed velocity sink (LOCAL ONLY in v1; no wire to meta-optimization).
// All wired here at the production edge; never imported into internal/execution.
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger) module.Module {
	// The plans SSOT — shared by both the PlanStore seam (read + phase mutate) and
	// the validation Service's PlanSource. One Service instance over the same store.
	plansSvc := internalplans.NewService(internalplans.Deps{
		Repo:  internalplans.NewSQLiteRepository(db, clk),
		Clock: clk,
	})

	// The validation Service the Validator seam wraps — same construction the
	// validation handler module uses (filesystem resolver + existence-staleness
	// floor + LookPath-guarded runner), reading plans through a PlanSource adapter.
	resolver := internalvalidation.NewFileResolver(repoRoot())
	validationSvc := internalvalidation.NewService(internalvalidation.Deps{
		Plans:     planSourceAdapter{svc: plansSvc},
		Resolver:  resolver,
		Staleness: internalvalidation.NewExistenceStaleness(resolver),
		Runner:    internalvalidation.DefaultRunner(),
		// Same result store the validation module writes to — execution READS the
		// last stored result here (cheap), never triggering a live run on status/next.
		Results: internalvalidation.NewSQLiteResultStore(db, clk),
		Clock:   clk,
	})

	svc := internalexecution.NewService(internalexecution.Deps{
		Repo:      internalexecution.NewSQLiteRepository(db, clk),
		Plans:     planStoreAdapter{svc: plansSvc},
		Validator: validatorAdapter{svc: validationSvc},
		Velocity:  internalexecution.DefaultVelocitySink(), // stub; no MoM wire (v1)
		Clock:     clk,
	})

	connectPath, connectHandler := executionconnect.NewExecutionServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "execution",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema returns the execution domain's SQL contribution. Re-exports
// internalexecution.Schema() so the modules registry's per-domain shape stays
// uniform.
func Schema() string { return internalexecution.Schema() }

// planStoreAdapter adapts the plans domain Service to execution's PlanStore seam
// (read + delegated phase transition). The plans Service owns the record; the
// runner never persists plan/phase structure itself.
type planStoreAdapter struct{ svc internalplans.Service }

func (a planStoreAdapter) GetPlan(ctx context.Context, idOrSlug string) (internalplans.Plan, error) {
	return a.svc.Get(ctx, idOrSlug)
}

func (a planStoreAdapter) UpdatePhase(ctx context.Context, planID string, phase internalplans.Phase) (internalplans.Plan, error) {
	return a.svc.UpdatePhase(ctx, planID, phase)
}

// planSourceAdapter adapts the plans domain Service to the validation domain's
// PlanSource read seam (method-name shim; the types are the shared plans model).
type planSourceAdapter struct{ svc internalplans.Service }

func (a planSourceAdapter) GetPlan(ctx context.Context, idOrSlug string) (internalplans.Plan, error) {
	return a.svc.Get(ctx, idOrSlug)
}

// validatorAdapter adapts the validation domain Service to execution's Validator
// seam. It surfaces the LAST STORED validation result (a cheap store read) into
// the just-in-time context — never triggering a live baseline run on status/next.
type validatorAdapter struct{ svc internalvalidation.Service }

func (a validatorAdapter) LastValidation(ctx context.Context, planID, phaseID string) (internalexecution.ValidationResult, bool, error) {
	res, ok, err := a.svc.LastValidation(ctx, planID, phaseID)
	if err != nil || !ok {
		return internalexecution.ValidationResult{}, false, err
	}
	return internalexecution.ValidationResult{
		ID:          res.ID,
		PlanID:      res.PlanID,
		PhaseID:     res.PhaseID,
		Verdict:     string(res.Verdict),
		Staleness:   res.Staleness,
		CommandsRun: res.CommandsRun,
		Detail:      res.Detail,
		RanAt:       res.RanAt,
	}, true, nil
}

// repoRoot resolves the repository root so the validation Service's filesystem
// reference resolution treats `[CODE: scenarios/foo/...]` as repo-relative.
// Mirrors handlers/validation/module.go: VROOLI_REPO_ROOT env, then a walk up for
// a `.git` marker, then the working directory as a last resort.
func repoRoot() string {
	if root := os.Getenv("VROOLI_REPO_ROOT"); root != "" {
		return root
	}
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// Endpoints is the machine-readable description of the execution module's public
// surface; one entry per RPC (the global parity test enforces this).
var Endpoints = []module.EndpointDescriptor{
	endpoint("execution_start", executionconnect.ExecutionServiceStartProcedure, "Start a guided run", "Links a run to a plan and returns the execution (OT-P0-003)."),
	endpoint("execution_get_status", executionconnect.ExecutionServiceGetStatusProcedure, "Get current context", "Returns the just-in-time context for the runner's current phase (OT-P0-003)."),
	endpoint("execution_get_next", executionconnect.ExecutionServiceGetNextProcedure, "Advance to next phase", "Advances the runner's pointer to the next actionable phase and returns its injected context."),
	endpoint("execution_transition_phase", executionconnect.ExecutionServiceTransitionPhaseProcedure, "Transition phase status", "Performs a typed phase-status transition; plan status is recomputed from the phase-status set."),
	endpoint("execution_record_decision", executionconnect.ExecutionServiceRecordDecisionProcedure, "Record a decision", "Captures an in-flow design decision (feeds the handoff)."),
	endpoint("execution_record_finding", executionconnect.ExecutionServiceRecordFindingProcedure, "Record a candidate finding", "Captures an in-flow CANDIDATE finding (never auto-promoted; attribution-keyed dedup)."),
	endpoint("execution_complete", executionconnect.ExecutionServiceCompleteProcedure, "Complete the run", "Runs the thin guided completion process, assembles the canonical handoff, and captures a velocity point (OT-P1-001/002)."),
	endpoint("execution_get_handoff", executionconnect.ExecutionServiceGetHandoffProcedure, "Get the handoff", "Returns the assembled canonical handoff for an execution."),
	endpoint("execution_list_candidate_findings", executionconnect.ExecutionServiceListCandidateFindingsProcedure, "List candidate findings", "Returns candidate findings awaiting operator triage."),
	endpoint("execution_triage_finding", executionconnect.ExecutionServiceTriageFindingProcedure, "Triage a finding", "Promotes or dismisses a candidate finding (operator action)."),
	endpoint("execution_get_velocity", executionconnect.ExecutionServiceGetVelocityProcedure, "Get velocity series", "Returns the per-plan velocity series (LOCAL ONLY)."),
}

func endpoint(id, path, summary, description string) module.EndpointDescriptor {
	return module.EndpointDescriptor{
		ID:          id,
		Path:        path,
		Method:      "POST",
		Summary:     summary,
		Description: description,
		Category:    "execution",
	}
}
