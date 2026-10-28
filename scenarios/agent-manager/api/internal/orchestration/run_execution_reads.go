// Responsibility: serve execution progress, policy and health queries without launching work.
package orchestration

import (
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/health"
	"agent-manager/internal/identity"
	"agent-manager/internal/repository"
	"context"
	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
	"time"
)

// GetRunProgress returns the current progress of a run for display.
// This provides visibility into what phase a run is in and estimated completion.
func (o *Orchestrator) GetRunProgress(ctx context.Context, id uuid.UUID) (*domain.RunProgress, error) {
	run, err := o.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}

	progress := &domain.RunProgress{
		Phase:            run.Phase,
		PhaseDescription: run.Phase.Description(),
		PercentComplete:  run.ProgressPercent,
		LastUpdate:       run.UpdatedAt,
	}

	// Calculate elapsed time
	if run.StartedAt != nil {
		progress.ElapsedTime = time.Since(*run.StartedAt)
	}

	// Add current action description based on phase
	switch run.Phase {
	case domain.RunPhaseExecuting:
		progress.CurrentAction = "Agent is working on the task"
	case domain.RunPhaseAwaitingReview:
		progress.CurrentAction = "Changes ready for review"
	case domain.RunPhaseApplying:
		progress.CurrentAction = "Applying approved changes"
	}

	return progress, nil
}

// ListStaleRuns returns runs that appear to have stalled based on their last heartbeat.
// This enables monitoring and automatic recovery of stuck runs.
func (o *Orchestrator) ListStaleRuns(ctx context.Context, staleDuration time.Duration) ([]*domain.Run, error) {
	// Get all running runs
	runningStatus := domain.RunStatusRunning
	runs, err := o.runs.List(ctx, repository.RunListFilter{
		Status: &runningStatus,
	})
	if err != nil {
		return nil, err
	}

	// Filter to stale runs
	var staleRuns []*domain.Run
	for _, run := range runs {
		if run.IsStale(staleDuration) {
			staleRuns = append(staleRuns, run)
		}
	}

	return staleRuns, nil
}

// NOTE: Approval operations (ApproveRun, RejectRun, PartialApprove)
// are implemented in approval.go for cognitive load reduction.

// -----------------------------------------------------------------------------
// Event Operations
// -----------------------------------------------------------------------------

func (o *Orchestrator) GetRunEvents(ctx context.Context, runID uuid.UUID, opts event.GetOptions) ([]*domain.RunEvent, error) {
	if o.events == nil {
		return nil, domain.NewConfigMissingError("eventStore", "not configured", nil)
	}
	return o.events.Get(ctx, runID, opts)
}

func (o *Orchestrator) GetRunEventSummary(ctx context.Context, runID uuid.UUID, opts event.GetOptions) (*event.Summary, error) {
	if o.events == nil {
		return nil, domain.NewConfigMissingError("eventStore", "not configured", nil)
	}
	return o.events.Summarize(ctx, runID, opts)
}

func (o *Orchestrator) StreamRunEvents(ctx context.Context, runID uuid.UUID, opts event.StreamOptions) (<-chan *domain.RunEvent, error) {
	if o.events == nil {
		return nil, domain.NewConfigMissingError("eventStore", "not configured", nil)
	}
	return o.events.Stream(ctx, runID, opts)
}

// -----------------------------------------------------------------------------
// Diff Operations
// -----------------------------------------------------------------------------

func (o *Orchestrator) GetRunDiff(ctx context.Context, runID uuid.UUID) (*sandbox.DiffResult, error) {
	run, err := o.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}

	if run.SandboxID == nil {
		if run.ExecutionMode.Normalized() == domain.ExecutionModeImported {
			return nil, &domain.ValidationError{Field: "sandboxId", Message: "imported run has no sandbox"}
		}
		return nil, &domain.ValidationError{Field: "sandboxId", Message: "run has no sandbox"}
	}

	if o.sandbox == nil {
		return nil, domain.NewConfigMissingError("sandbox", "provider not configured", nil)
	}

	diff, err := o.sandbox.GetDiff(ctx, *run.SandboxID)
	if err != nil {
		return nil, err
	}
	if diff != nil {
		diff.FileCountVerified = sandbox.IsVerifiedFileCount(diff, *run.SandboxID)
	}
	return diff, nil
}

// -----------------------------------------------------------------------------
// Model Policy Operations
// -----------------------------------------------------------------------------

func (o *Orchestrator) GetModelHealthSnapshot(ctx context.Context) (health.Snapshot, error) {
	if o.healthStore == nil {
		return health.Snapshot{
			Models:  map[string]map[string]health.ModelEntry{},
			Runners: map[string]health.RunnerEntry{},
		}, nil
	}
	return o.healthStore.Snapshot(ctx)
}

// ExplainProfilePolicy resolves the profile against the active catalog without
// creating a run. The returned snapshot includes the same availability
// preflight and precedence explanation run creation would persist.
func (o *Orchestrator) ExplainProfilePolicy(ctx context.Context, profileID uuid.UUID) (*domain.ExecutionPolicySnapshot, error) {
	cfg, _, err := o.resolveRunConfig(ctx, CreateRunRequest{AgentProfileID: &profileID})
	if err != nil {
		return nil, err
	}
	if cfg == nil || cfg.PolicySnapshot == nil {
		return nil, domain.NewValidationError("rolePolicyCatalog", "profile resolution produced no policy snapshot")
	}
	return cfg.PolicySnapshot, nil
}

// ExplainRunPolicy returns only the immutable snapshot stored with the run. It
// never reconstructs historical provenance from the current catalog.
func (o *Orchestrator) ExplainRunPolicy(ctx context.Context, runID uuid.UUID) (*domain.ExecutionPolicySnapshot, error) {
	run, err := o.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run.ResolvedConfig == nil || run.ResolvedConfig.PolicySnapshot == nil {
		return nil, nil
	}
	return run.ResolvedConfig.PolicySnapshot, nil
}

// -----------------------------------------------------------------------------
// Config Accessors
// -----------------------------------------------------------------------------

func (o *Orchestrator) GetDefaultProjectRoot() string {
	return o.config.DefaultProjectRoot
}

// ValidatePath delegates path validation to the sandbox provider.
func (o *Orchestrator) ValidatePath(ctx context.Context, path string, projectRoot string) (*sandbox.PathValidationResult, error) {
	if o.sandbox == nil {
		return nil, domain.NewConfigMissingError("sandbox", "provider not configured", nil)
	}
	return o.sandbox.ValidatePath(ctx, path, projectRoot)
}

type sessionBindingVerificationKey struct{}

// WithSessionBindingVerification requests the optional owner proof for a narrow read.
func WithSessionBindingVerification(ctx context.Context) context.Context {
	return context.WithValue(ctx, sessionBindingVerificationKey{}, true)
}

// VerifyIdentityToken validates a signed agent identity token and returns the
// embedded claims along with the current run status.
func (o *Orchestrator) VerifyIdentityToken(ctx context.Context, token string) (*IdentityVerifyResult, error) {
	if len(o.identitySecret) == 0 {
		return &IdentityVerifyResult{Valid: false, Error: "identity system not configured"}, nil
	}

	claims, err := identity.VerifyToken(token, o.identitySecret)
	if err != nil {
		return &IdentityVerifyResult{Valid: false, Error: err.Error()}, nil
	}
	if claims.Purpose != "" {
		return &IdentityVerifyResult{Valid: false, Error: "credential is not an active run identity"}, nil
	}

	// Only this verifier may project the session-binding claim.
	delete(claims.Meta, "verified_web_console_session_id")
	if claims.Meta["authorization_id"] != "" {
		return o.verifyAuthorization(ctx, token, claims)
	}

	// A valid signature alone is insufficient. The token must still be the
	// active token recorded for the same run and must not have been revoked at
	// run completion. This turns a signed, time-limited bearer token into a
	// live run credential rather than a 24-hour post-completion capability.
	tokenHash := identity.HashToken(token)
	run, err := o.runs.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return &IdentityVerifyResult{Valid: false, Error: "identity token is not active"}, nil
	}
	if run.ID != claims.RunID {
		return &IdentityVerifyResult{Valid: false, Error: "identity token does not match its active run"}, nil
	}
	if run.ExecutionMode.Normalized() == domain.ExecutionModeInteractive && (run.WebConsoleSessionID == "" || claims.Meta["web_console_session_id"] != run.WebConsoleSessionID) {
		return &IdentityVerifyResult{Valid: false, Error: "interactive identity token does not match its persisted session binding"}, nil
	}
	if run.IdentityTokenRevokedAt != nil {
		return &IdentityVerifyResult{Valid: false, Error: "identity token has been revoked"}, nil
	}

	// Project the persisted session only after checking the active run and signed
	// attachment coordinates. A caller-supplied observation never grants it.
	if requested, _ := ctx.Value(sessionBindingVerificationKey{}).(bool); requested && run.Status == domain.RunStatusRunning && run.WebConsoleSessionID != "" {
		mode := run.ExecutionMode.Normalized()
		attached := mode == domain.ExecutionModeAttached && claims.Meta["execution_mode"] == string(mode) && run.HarnessKind != "" && run.HarnessSessionID != "" && claims.Meta["harness_kind"] == run.HarnessKind && claims.Meta["harness_session"] == run.HarnessSessionID && o.verifiedAttachedSessionBinding(ctx, run)
		if mode == domain.ExecutionModeInteractive || attached {
			if claims.Meta == nil {
				claims.Meta = make(map[string]string)
			}
			claims.Meta["verified_web_console_session_id"] = run.WebConsoleSessionID
		}
	}

	if err := o.projectVerifiedWorkflowIdentity(ctx, run, claims); err != nil {
		return nil, err
	}
	return &IdentityVerifyResult{
		Valid:     true,
		Claims:    claims,
		RunStatus: run.Status,
	}, nil
}

// A persisted attach header is only a locator. Recheck it against the existing
// session-owner lease proof and the current kernel process identity.
func (o *Orchestrator) verifiedAttachedSessionBinding(ctx context.Context, run *domain.Run) bool {
	if o.sessionBindingProofReader == nil || run.RunnerPID <= 1 {
		return false
	}
	proof, err := o.sessionBindingProofReader.ReadSessionBinding(ctx, SessionBindingRequest{RunID: run.ID, HarnessSessionID: run.HarnessSessionID, WebConsoleSessionID: run.WebConsoleSessionID, HarnessKind: run.HarnessKind, ProcessID: run.RunnerPID})
	if err != nil {
		return false
	}
	binding, err := cliutil.ObserveProcessBinding(run.RunnerPID)
	if err != nil {
		return false
	}
	if proof.OwnerKind != "web-console-session" || !validSessionBindingTuple(proof, run.ID, run.HarnessSessionID, run.WebConsoleSessionID, run.HarnessKind, binding) {
		return false
	}
	latest, err := o.runs.Get(ctx, run.ID)
	return err == nil && sameRenewalBinding(run, latest) && latest.IdentityTokenHash == run.IdentityTokenHash
}

// Status Operations
// -----------------------------------------------------------------------------

func (o *Orchestrator) GetHealth(ctx context.Context) (*HealthStatus, error) {
	status := &HealthStatus{
		Status:    "healthy",
		Service:   "agent-manager",
		Timestamp: o.now().UTC().Format(time.RFC3339),
		Readiness: true,
		Dependencies: &HealthDependencies{
			Runners: make(map[string]*DependencyStatus),
		},
	}

	// Check database (repositories configured)
	if o.profiles != nil && o.workflows != nil && o.tasks != nil && o.runs != nil {
		status.Dependencies.Database = &DependencyStatus{Connected: true, Storage: o.storageLabel}
	} else {
		msg := "core or workflow repository not configured"
		status.Dependencies.Database = &DependencyStatus{
			Connected: false,
			Error:     &msg,
			Storage:   o.storageLabel,
		}
	}
	if o.workflows != nil && o.workflowExecutions != nil && o.workflowEngine != nil {
		status.Dependencies.WorkflowRuntime = &DependencyStatus{Connected: true, Storage: o.storageLabel}
	} else {
		msg := "workflow catalog, execution repository, or interpreter is not configured"
		status.Dependencies.WorkflowRuntime = &DependencyStatus{Connected: false, Error: &msg, Storage: o.storageLabel}
		status.Readiness = false
		status.Status = "degraded"
	}

	// Check sandbox
	if o.sandbox != nil {
		available, msg := o.sandbox.IsAvailable(ctx)
		status.Dependencies.Sandbox = &DependencyStatus{
			Connected: available,
		}
		if !available && msg != "" {
			status.Dependencies.Sandbox.Error = &msg
		}
	} else {
		msg := "not configured"
		status.Dependencies.Sandbox = &DependencyStatus{
			Connected: false,
			Error:     &msg,
		}
	}

	// Check runners
	if o.runners != nil {
		for _, r := range o.runners.List() {
			available, msg := r.IsAvailable(ctx)
			depStatus := &DependencyStatus{
				Connected: available,
			}
			if !available && msg != "" {
				depStatus.Error = &msg
			}
			status.Dependencies.Runners[string(r.Type())] = depStatus
		}
	}

	// Count active runs
	if o.runs != nil {
		runningStatus := domain.RunStatusRunning
		runs, _ := o.runs.List(ctx, repository.RunListFilter{Status: &runningStatus})
		status.ActiveRuns = len(runs)
	}

	// Count queued tasks
	if o.tasks != nil {
		tasks, _ := o.tasks.List(ctx, repository.ListFilter{})
		var queued int
		for _, t := range tasks {
			if t.Status == domain.TaskStatusQueued {
				queued++
			}
		}
		status.QueuedTasks = queued
	}

	return status, nil
}
