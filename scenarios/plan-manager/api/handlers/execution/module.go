// Package execution is the API handler for the ExecutionService — the
// guided-runner domain. It is the proto translation edge over
// internal/execution; all business logic lives there behind seams.
package execution

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	internalexecution "plan-manager/internal/execution"
	"plan-manager/internal/module"
	internalplanlog "plan-manager/internal/planlog"
	planmodel "plan-manager/internal/planmodel"
	internalplans "plan-manager/internal/plans"
	internalvalidation "plan-manager/internal/validation"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	executionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/execution/execution_v1connect"
)

// Module returns the execution domain's contribution to the API: the generated
// ExecutionService Connect-RPC handler, backed by the execution home store (the
// executions/handoffs/velocity tables), the plans SSOT (the
// PlanStore seam — read + delegated phase transitions), the validation domain
// (the Validator seam — last validation + staleness, degrading to UNKNOWN), and
// the stubbed velocity sink (LOCAL ONLY in v1; no wire to meta-optimization).
// All wired here at the production edge; never imported into internal/execution.
func Module(db *database.RoutedDB, clk schedule.Clock, logger *log.Logger) module.Module {
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
		Plans:       planSourceAdapter{svc: plansSvc},
		Resolver:    resolver,
		Staleness:   internalvalidation.NewExistenceStaleness(resolver),
		Runner:      internalvalidation.DefaultRunner(),
		Collections: newGCTCollectionClient(),
		// Same result store the validation module writes to — execution READS the
		// last stored result here (cheap), never triggering a live run on status/next.
		Results: internalvalidation.NewSQLiteResultStore(db, clk),
		Clock:   clk,
	})

	// The log ledger the LogLedger seam reads for just-in-time summaries and the
	// handoff snapshot. Same store the log handler module writes to; execution
	// only READS compact summaries here (decisions/findings/bugs/records are
	// written through `plan-manager log ...`).
	logSvc := internalplanlog.NewService(internalplanlog.Deps{
		Repo:  internalplanlog.NewSQLiteRepository(db, clk),
		Clock: clk,
	})

	svc := internalexecution.NewService(internalexecution.Deps{
		Repo:      internalexecution.NewSQLiteRepository(db, clk),
		Plans:     planStoreAdapter{svc: plansSvc},
		Validator: validatorAdapter{svc: validationSvc},
		Log:       logLedgerAdapter{svc: logSvc},
		Velocity:  internalexecution.DefaultVelocitySink(), // stub; no MoM wire (v1)
		// GCT state is read only through this sync seam. The execution service
		// persists and renders producer tickets, but never starts or waits for GCT.
		Baseline:  baselineSynchronizerAdapter{svc: validationSvc},
		Preflight: newGCTSourcePreflighter(),
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
	return a.svc.Get(ctx, idOrSlug, internalplans.WorkspaceScope{})
}

func (a planStoreAdapter) UpdatePhase(ctx context.Context, planID, workspaceID, workspaceRoot string, phase internalplans.Phase) (internalplans.Plan, error) {
	return a.svc.UpdatePhase(ctx, planID, internalplans.WorkspaceScope{ID: workspaceID, Root: workspaceRoot}, phase)
}

func (a planStoreAdapter) ExtendChangeBoundary(ctx context.Context, planID, workspaceID, workspaceRoot string, globs []string) (internalplans.Plan, []string, error) {
	return a.svc.ExtendChangeBoundary(ctx, planID, internalplans.WorkspaceScope{ID: workspaceID, Root: workspaceRoot}, globs)
}

// logLedgerAdapter adapts the log domain Service to execution's LogLedger seam.
// It surfaces compact log summaries + captured entries (a cheap store read) into
// the just-in-time context and the canonical handoff. Decisions/findings/bugs/
// records are OWNED by the log domain; execution never writes them.
type logLedgerAdapter struct{ svc internalplanlog.Service }

func (a logLedgerAdapter) Summarize(ctx context.Context, executionID string) (planmodel.LogSummary, []planmodel.LogEntry, error) {
	return a.svc.Summarize(ctx, internalplanlog.Filter{ExecutionID: executionID})
}

func (a logLedgerAdapter) SummarizePhase(ctx context.Context, executionID, phaseID string) (planmodel.LogSummary, []planmodel.LogEntry, error) {
	return a.svc.Summarize(ctx, internalplanlog.Filter{ExecutionID: executionID, PhaseID: phaseID})
}

// planSourceAdapter adapts the plans domain Service to the validation domain's
// PlanSource read seam (method-name shim; the types are the shared plans model).
type planSourceAdapter struct{ svc internalplans.Service }

func (a planSourceAdapter) GetPlan(ctx context.Context, idOrSlug string) (internalplans.Plan, error) {
	return a.svc.Get(ctx, idOrSlug, internalplans.WorkspaceScope{})
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
		ID:              res.ID,
		PlanID:          res.PlanID,
		PhaseID:         res.PhaseID,
		Verdict:         string(res.Verdict),
		Staleness:       res.Staleness,
		CommandsRun:     res.CommandsRun,
		Detail:          res.Detail,
		RanAt:           res.RanAt,
		ExecutionID:     res.ExecutionID,
		OperationID:     res.OperationID,
		ScopeGeneration: res.ScopeGeneration,
		FullInventory:   res.FullInventory,
		RequiredMembers: append([]string(nil), res.RequiredMembers...),
		SelectedMembers: append([]string(nil), res.SelectedMembers...),
	}, true, nil
}

// inputFreshenerAdapter adapts the validation domain Service to execution's
// InputFreshener seam. At execution start it captures the regression-anchor's
// baseline snapshot fresh (CaptureBaseline) and recomputes reference staleness
// (ComputeStaleness) — both owned by validation (which owns git-control-tower) —
// so execution never imports git-control-tower directly. Degradation is honest:
// an incomplete anchor or an absent git-control-tower yields BaselineCaptured=false
// with a Detail, never a fabricated capture.
type baselineSynchronizerAdapter struct{ svc internalvalidation.Service }

func (a baselineSynchronizerAdapter) SyncBaseline(ctx context.Context, planID, baselineName string) (internalexecution.FreshenResult, error) {
	capture, err := a.svc.SyncBaseline(ctx, planID, baselineName)
	if err != nil {
		return internalexecution.FreshenResult{}, err
	}
	res := internalexecution.FreshenResult{
		BaselineCaptured: capture.Captured,
		BaselineName:     capture.BaselineName,
		Detail:           capture.Detail,
	}
	if len(capture.ScenarioTargets) > 0 || len(capture.RepoPaths) > 0 {
		status := internalexecution.BaselineSetStatusPartial
		if capture.Captured {
			status = internalexecution.BaselineSetStatusComplete
		} else if capture.Required == 0 {
			status = internalexecution.BaselineSetStatusDegraded
		}
		res.BaselineSet = internalexecution.BaselineSetState{
			Version: internalexecution.BaselineSetStateSchemaVersion, Name: capture.BaselineName,
			CollectionBranch: capture.CollectionBranch,
			ScenarioTargets:  append([]string(nil), capture.ScenarioTargets...),
			RepoPaths:        append([]string(nil), capture.RepoPaths...),
			Status:           status, Required: capture.Required, Ready: capture.Ready, Pending: capture.Pending,
			Failed: capture.Failed, Skipped: capture.Skipped, Stale: capture.Stale,
			Members: baselineSetMembers(capture.Members), PathSnapshots: baselineSetPathSnapshots(capture.PathSnapshots), Detail: capture.Detail,
		}
	}
	// Reference staleness is REPORTED, never written back to the authored plan.
	// A staleness recompute failure is non-fatal — the baseline capture is the
	// primary freshen action; staleness is advisory.
	if report, stErr := a.svc.ComputeStaleness(ctx, planID, ""); stErr == nil {
		res.StalenessSummary = summarizeStaleness(report)
	}
	return res, nil
}

func baselineSetMembers(members []internalvalidation.BaselineCollectionMember) []internalexecution.BaselineSetMember {
	out := make([]internalexecution.BaselineSetMember, 0, len(members))
	for _, member := range members {
		out = append(out, internalexecution.BaselineSetMember{Scenario: member.Scenario, BaselineName: member.BaselineName, Required: member.Required, Status: member.Status, RunID: member.RunID, GitSHA: member.GitSHA, Error: member.Error})
	}
	return out
}

func baselineSetPathSnapshots(snapshots []internalvalidation.BaselinePathSnapshot) []internalexecution.BaselineSetPathSnapshot {
	out := make([]internalexecution.BaselineSetPathSnapshot, 0, len(snapshots))
	for _, snapshot := range snapshots {
		out = append(out, internalexecution.BaselineSetPathSnapshot{Name: snapshot.Name, Branch: snapshot.Branch, CreatedAt: snapshot.CreatedAt})
	}
	return out
}

// summarizeStaleness renders a short human roll-up of the recomputed reference
// staleness (reported only — authored references are never mutated).
func summarizeStaleness(report internalvalidation.ReferenceReport) string {
	if len(report.References) == 0 {
		return ""
	}
	counts := map[planmodel.StalenessTier]int{}
	for _, ref := range report.References {
		counts[ref.Staleness]++
	}
	overall := report.Overall
	if overall == "" {
		overall = planmodel.StalenessUnknown
	}
	return fmt.Sprintf("staleness: %d reference(s), overall=%s (fresh=%d, lightly_stale=%d, definitely_stale=%d)",
		len(report.References), orStalenessUnknown(overall),
		counts[planmodel.StalenessFresh], counts[planmodel.StalenessLightlyStale], counts[planmodel.StalenessDefinitelyStale])
}

func orStalenessUnknown(tier planmodel.StalenessTier) string {
	if strings.TrimSpace(string(tier)) == "" {
		return "unknown"
	}
	return string(tier)
}

// repoRoot resolves the repository root so the validation Service's filesystem
// reference resolution treats `[CODE: scenarios/foo/...]` as repo-relative.
// Mirrors handlers/validation/module.go: VROOLI_REPO_ROOT env, then a walk up for
// a `.git` marker, then the working directory as a last resort.
func repoRoot() string {
	if root, ok := os.LookupEnv("VROOLI_REPO_ROOT"); ok {
		if root = strings.TrimSpace(root); root != "" {
			return root
		}
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
	endpoint("execution_get_context", executionconnect.ExecutionServiceGetContextProcedure, "Get setup context", "Returns setup context for the current or requested phase without advancing the runner."),
	endpoint("execution_resume", executionconnect.ExecutionServiceResumeProcedure, "Resume execution", "Resolves an existing execution or creates one for a plan and returns setup context without advancing."),
	endpoint("execution_continue", executionconnect.ExecutionServiceContinueExecutionProcedure, "Continue execution", "Resumes or starts an execution and returns the single recommended next runner action without advancing."),
	endpoint("execution_abandon", executionconnect.ExecutionServiceAbandonExecutionProcedure, "Abandon execution", "Terminally abandons an accidental execution while preserving audit history."),
	endpoint("execution_sync_baseline", executionconnect.ExecutionServiceSyncBaselineProcedure, "Synchronize baseline evidence", "Reads the producer-owned GCT collection once and persists typed coverage without starting or waiting for capture."),
	endpoint("execution_amend_scope", executionconnect.ExecutionServiceAmendScopeProcedure, "Amend validation scope", "Records an auditable expansion within the captured baseline inventory and invalidates prior phase evidence."),
	endpoint("execution_adopt_baseline", executionconnect.ExecutionServiceAdoptBaselineProcedure, "Adopt legacy baseline", "Creates a producer ticket or an explicit degraded legacy state without starting or waiting for capture."),
	endpoint("execution_repair_source_scope", executionconnect.ExecutionServiceRepairSourceScopeProcedure, "Repair baseline source scope", "Boundary-checks and re-estimates an informational source-evidence replacement before capture can be issued."),
	endpoint("execution_extend_boundary", executionconnect.ExecutionServiceExtendBoundaryProcedure, "Extend the change boundary", "Appends allow globs to the plan's change boundary mid-execution so validation scope follows a sanctioned scope expansion. Append-only; acceptance_deny still refuses."),
	endpoint("execution_get_next", executionconnect.ExecutionServiceGetNextProcedure, "Advance to next phase", "Advances the runner's pointer to the next actionable phase and returns its injected context."),
	endpoint("execution_transition_phase", executionconnect.ExecutionServiceTransitionPhaseProcedure, "Transition phase status", "Performs a typed phase-status transition; plan status is recomputed from the phase-status set."),
	endpoint("execution_complete", executionconnect.ExecutionServiceCompleteProcedure, "Complete the run", "Runs the thin guided completion process, assembles the canonical handoff, and captures a velocity point (OT-P1-001/002)."),
	endpoint("execution_partial_handoff", executionconnect.ExecutionServicePartialHandoffProcedure, "Create partial handoff", "Persists an honest resumable handoff without marking the execution normally complete."),
	endpoint("execution_get_handoff", executionconnect.ExecutionServiceGetHandoffProcedure, "Get the handoff", "Returns the assembled canonical handoff for an execution."),
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
