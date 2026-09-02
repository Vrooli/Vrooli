// Package validation is the API handler for the ValidationService — the
// plan-health domain. It is the proto translation edge over internal/validation;
// all business logic lives there behind seams.
package validation

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	internalexecution "plan-manager/internal/execution"
	"plan-manager/internal/module"
	internalplans "plan-manager/internal/plans"
	internalvalidation "plan-manager/internal/validation"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/validation/validation_v1connect"
)

// Module returns the validation domain's contribution to the API: the generated
// ValidationService Connect-RPC handler, backed by the plans SSOT (read seam),
// the code-facts-backed reference resolver with a filesystem floor, the
// filesystem existence staleness floor plus git-sourced per-reference drift
// refinement, and the LookPath-guarded command runner (the git-control-tower
// baseline-diff oracle). All wired here at the production edge; never imported
// into internal/validation.
func Module(db *database.RoutedDB, clk schedule.Clock, logger *log.Logger) module.Module {
	plansSvc := internalplans.NewService(internalplans.Deps{
		Repo:  internalplans.NewSQLiteRepository(db, clk),
		Clock: clk,
	})
	root := repoRoot()
	resolver := newCodeFactsReferenceResolver(root)
	store := internalvalidation.NewSQLiteResultStore(db, clk)
	executionRepo := internalexecution.NewSQLiteRepository(db, clk)
	svc := internalvalidation.NewService(internalvalidation.Deps{
		Plans:       planAdapter{svc: plansSvc},
		Resolver:    resolver,
		Staleness:   internalvalidation.NewExistenceStaleness(internalvalidation.NewFileResolver(root)),
		Runner:      internalvalidation.DefaultRunner(),
		Collections: newGCTCollectionClient(),
		TestRuns:    newTestGenieRunClient(),
		Inventories: executionInventoryAdapter{repo: executionRepo},
		Results:     store,
		Operations:  store,
		Clock:       clk,
		Commands:    newCLIHealthCommandValidator(),
	})
	// Recovery only preserves tickets for producer-side reattachment and sync.
	// It never resumes or dispatches validation work.
	if err := svc.RecoverPending(context.Background()); err != nil && logger != nil {
		logger.Printf("validation operation recovery: %v", err)
	}
	connectPath, connectHandler := validationconnect.NewValidationServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "validation",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema returns the validation domain's SQL contribution (the validation_results
// table — the last-known result per plan/phase that the execution context server
// reads, so status/next never shell a subprocess). Re-exports
// internalvalidation.Schema() so the modules registry's per-domain shape stays
// uniform.
func Schema() string { return internalvalidation.Schema() }

// planAdapter adapts the plans domain Service to validation's PlanSource read
// seam (the method names differ; the types are the shared plans model).
type planAdapter struct{ svc internalplans.Service }

func (a planAdapter) GetPlan(ctx context.Context, idOrSlug string) (internalplans.Plan, error) {
	return a.svc.Get(ctx, idOrSlug, internalplans.WorkspaceScope{})
}

// executionInventoryAdapter crosses only a narrow read seam: execution owns
// the immutable capture checkpoint, while validation owns comparison mechanics.
type executionInventoryAdapter struct{ repo internalexecution.Repository }

func (a executionInventoryAdapter) LatestBaselineInventory(ctx context.Context, planID string) (internalvalidation.BaselineInventory, bool, error) {
	execution, ok, err := a.repo.LatestExecutionForPlan(ctx, planID)
	if err != nil || !ok || execution.BaselineSet.Name == "" {
		return internalvalidation.BaselineInventory{}, false, err
	}
	return internalvalidation.BaselineInventory{
		Name:            execution.BaselineSet.Name,
		Branch:          execution.BaselineSet.CollectionBranch,
		ScenarioTargets: append([]string(nil), execution.BaselineSet.ScenarioTargets...),
		PathSnapshots:   baselinePathSnapshots(execution.BaselineSet.PathSnapshots),
		Complete:        execution.BaselineSet.Complete(),
	}, true, nil
}

func baselinePathSnapshots(snapshots []internalexecution.BaselineSetPathSnapshot) []internalvalidation.BaselinePathSnapshot {
	out := make([]internalvalidation.BaselinePathSnapshot, 0, len(snapshots))
	for _, snapshot := range snapshots {
		out = append(out, internalvalidation.BaselinePathSnapshot{Name: snapshot.Name, Branch: snapshot.Branch, CreatedAt: snapshot.CreatedAt})
	}
	return out
}

// repoRoot resolves the repository root so filesystem reference resolution
// treats `[CODE: scenarios/foo/...]` as repo-relative. Order: VROOLI_ROOT
// env, then a walk up from the working directory for a `.git` marker, then the
// working directory as a last resort (references then resolve relative to it).
func repoRoot() string {
	if root, ok := os.LookupEnv("VROOLI_ROOT"); ok {
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

// Endpoints is the machine-readable description of the validation module's
// public surface; one entry per RPC (the global parity test enforces this).
var Endpoints = []module.EndpointDescriptor{
	endpoint("validation_resolve_references", validationconnect.ValidationServiceResolveReferencesProcedure, "Resolve code references", "Resolves a plan/phase's [CODE:]/[REQ:] references against code-facts (OT-P0-004). Degrades to unresolved when code-facts is down."),
	endpoint("validation_compute_staleness", validationconnect.ValidationServiceComputeStalenessProcedure, "Compute staleness tiers", "Computes staleness tiers for a plan/phase's references (OT-P0-004)."),
	endpoint("validation_derive_baseline_scope", validationconnect.ValidationServiceDeriveBaselineScopeProcedure, "Derive baseline scope", "Derives the exact baseline/validation command set across all affected locations (OT-P0-005)."),
	endpoint("validation_start", validationconnect.ValidationServiceStartValidationProcedure, "Create validation ticket", "Persists producer-owned validation actions; scoped idempotency retries return the original operation."),
	endpoint("validation_operation", validationconnect.ValidationServiceGetValidationOperationProcedure, "Inspect validation ticket", "Reads a durable validation ticket without starting or waiting for producer work."),
	endpoint("validation_wait", validationconnect.ValidationServiceWaitValidationOperationProcedure, "Legacy validation inspection", "Compatibility inspection route; use the producer wait command rendered by the ticket."),
	endpoint("validation_resume", validationconnect.ValidationServiceResumeValidationOperationProcedure, "Legacy validation inspection", "Compatibility inspection route; producer recovery remains owned by Git Control Tower or Test Genie."),
	endpoint("validation_sync", validationconnect.ValidationServiceSyncValidationProcedure, "Synchronize validation evidence", "Reads producer-owned durable evidence once and atomically records terminal validation truth."),
	endpoint("validation_run", validationconnect.ValidationServiceRunValidationProcedure, "Legacy validation guidance", "Returns the producer-ticket migration route; never dispatches or waits for validation work."),
	endpoint("validation_verify_dod", validationconnect.ValidationServiceVerifyDefinitionOfDoneProcedure, "Legacy DoD guidance", "Returns the selector-free producer-ticket migration route; never dispatches or waits for validation work."),
}

func endpoint(id, path, summary, description string) module.EndpointDescriptor {
	return module.EndpointDescriptor{
		ID:          id,
		Path:        path,
		Method:      "POST",
		Summary:     summary,
		Description: description,
		Category:    "validation",
	}
}
