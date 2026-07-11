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

	"plan-manager/internal/clock"
	"plan-manager/internal/module"
	internalplans "plan-manager/internal/plans"
	internalvalidation "plan-manager/internal/validation"

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
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger) module.Module {
	plansSvc := internalplans.NewService(internalplans.Deps{
		Repo:  internalplans.NewSQLiteRepository(db, clk),
		Clock: clk,
	})
	root := repoRoot()
	resolver := newCodeFactsReferenceResolver(root)
	store := internalvalidation.NewSQLiteResultStore(db, clk)
	svc := internalvalidation.NewService(internalvalidation.Deps{
		Plans:      planAdapter{svc: plansSvc},
		Resolver:   resolver,
		Staleness:  internalvalidation.NewExistenceStaleness(internalvalidation.NewFileResolver(root)),
		Runner:     internalvalidation.DefaultRunner(),
		Results:    store,
		Operations: store,
		Clock:      clk,
		Commands:   newCLIHealthCommandValidator(),
	})
	// Recovery is idempotent: persisted queued/running operations resume from
	// their child checkpoints and terminal children are never re-dispatched.
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

// repoRoot resolves the repository root so filesystem reference resolution
// treats `[CODE: scenarios/foo/...]` as repo-relative. Order: VROOLI_REPO_ROOT
// env, then a walk up from the working directory for a `.git` marker, then the
// working directory as a last resort (references then resolve relative to it).
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

// Endpoints is the machine-readable description of the validation module's
// public surface; one entry per RPC (the global parity test enforces this).
var Endpoints = []module.EndpointDescriptor{
	endpoint("validation_resolve_references", validationconnect.ValidationServiceResolveReferencesProcedure, "Resolve code references", "Resolves a plan/phase's [CODE:]/[REQ:] references against code-facts (OT-P0-004). Degrades to unresolved when code-facts is down."),
	endpoint("validation_compute_staleness", validationconnect.ValidationServiceComputeStalenessProcedure, "Compute staleness tiers", "Computes staleness tiers for a plan/phase's references (OT-P0-004)."),
	endpoint("validation_derive_baseline_scope", validationconnect.ValidationServiceDeriveBaselineScopeProcedure, "Derive baseline scope", "Derives the exact baseline/validation command set across all affected locations (OT-P0-005)."),
	endpoint("validation_start", validationconnect.ValidationServiceStartValidationProcedure, "Start durable validation", "Persists a validation operation and child set before dispatch; scoped idempotency retries return the original operation."),
	endpoint("validation_operation", validationconnect.ValidationServiceGetValidationOperationProcedure, "Inspect or wait for validation", "Reads a durable validation operation or performs one server-side blocking wait without polling."),
	endpoint("validation_wait", validationconnect.ValidationServiceWaitValidationOperationProcedure, "Wait for validation", "Performs one server-side blocking wait by durable operation id."),
	endpoint("validation_resume", validationconnect.ValidationServiceResumeValidationOperationProcedure, "Resume validation", "Reconciles queued/running child checkpoints after interruption or restart and waits once."),
	endpoint("validation_run", validationconnect.ValidationServiceRunValidationProcedure, "Run validation", "Compatibility blocking validation over the durable operation substrate."),
	endpoint("validation_verify_dod", validationconnect.ValidationServiceVerifyDefinitionOfDoneProcedure, "Verify Definition of Done", "Verifies a plan's DoD against the regression anchor as an oracle (OT-P0-005)."),
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
