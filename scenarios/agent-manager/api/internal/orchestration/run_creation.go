// This file creates runs and their initial persisted execution state.
package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/metrics"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/orchestration/phases"
	"agent-manager/internal/orchestration/spawn"
	"agent-manager/internal/policy"
	"agent-manager/internal/tokenaccounting"

	"github.com/google/uuid"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/proto"
)

const maxWorkReferenceDeclarations = 100

func validateEffectiveRunInput(task *domain.Task, prompt string) error {
	if task == nil {
		return domain.NewValidationError("task", "cannot be nil")
	}
	for i := range task.ContextAttachments {
		if task.ContextAttachments[i].Validate() == nil {
			// Image attachments are passed to the runner separately from the text
			// prompt, so a valid image-only task is still actionable.
			return nil
		}
	}
	systemPrompt, userMessage := domain.BuildSplitPrompt(task.Description, nil, prompt)
	if strings.TrimSpace(systemPrompt) != "" || strings.TrimSpace(userMessage) != "" {
		return nil
	}
	return domain.NewValidationErrorWithHint("prompt", "a run needs an explicit prompt, a task description, or context attachments", "provide task instructions before creating a run")
}

func (o *Orchestrator) CreateRun(ctx context.Context, req CreateRunRequest) (*domain.Run, error) {
	// Public caller projections and environment labels are not workflow authority.
	req.workflowIdentity = nil
	for key := range req.Environment {
		if strings.HasPrefix(key, "VROOLI_WORKFLOW_") {
			return nil, domain.NewValidationError("environment", "workflow environment is reserved for the private workflow launcher")
		}
	}
	// Public caller projections are not authority, including in-process callers.
	req.caller, req.profileAdmission = nil, nil
	req.OwnerSubject, req.OwnerScopes, req.OwnerExpiresAt = "", nil, nil
	// Freeze the caller's exact request before authentication and parent
	// inheritance add derived admission data. Credentials and verified identity
	// projections are excluded by the request's JSON tags and remain checked by
	// the separate identity replay rules.
	digest, err := createRunRequestDigest(req)
	if err != nil {
		return nil, err
	}
	req.requestDigest = digest
	if err := o.authenticateCreateRunCaller(ctx, &req); err != nil {
		return nil, err
	}
	return o.createRun(ctx, req, nil)
}

// createRun accepts invocation-local recovery context while retaining the
// original task and the normal admission, identity, policy and dispatch gates.
type runRecoveryContext struct {
	attachments []domain.ContextAttachment
	config      *domain.RunConfig
}

// createRunFlow carries one run-creation attempt through its admission,
// persistence and dispatch phases. Each phase is a named method; createRun
// runs them in order and stops at the first refusal.
type createRunFlow struct {
	o        *Orchestrator
	req      CreateRunRequest
	recovery *runRecoveryContext
	runID    uuid.UUID
	// cleanups run in reverse order when createRun returns, like defers.
	cleanups []func()

	task                 *domain.Task
	expectedTaskSnapshot domain.Task
	profile              *domain.AgentProfile
	resolvedConfig       *domain.RunConfig
	sandboxConfig        *domain.SandboxConfig
	policyDecision       *policy.Decision
	runMode              domain.RunMode
	spawnSkips           []SpawnPreferenceSkip

	existingSandboxWorkDir string
	sandboxToStart         *uuid.UUID
	qualifiedParent        *domain.Run

	workload     domain.WorkloadRef
	tag          string
	billing      domain.BillingSnapshot
	systemPrompt string
	userMessage  string

	run                *domain.Run
	admission          *spawn.Admission
	admissionHandedOff bool
}

func (f *createRunFlow) cleanup(fn func()) { f.cleanups = append(f.cleanups, fn) }

func (f *createRunFlow) release() {
	for i := len(f.cleanups) - 1; i >= 0; i-- {
		f.cleanups[i]()
	}
}

// failReservation frees the idempotency key a failed attempt reserved so the
// caller can retry. Only the phases after the key is reserved call it.
func (f *createRunFlow) failReservation(ctx context.Context) {
	f.o.markIdempotencyFailed(ctx, f.req.IdempotencyKey)
}

func (o *Orchestrator) createRun(ctx context.Context, req CreateRunRequest, recovery *runRecoveryContext) (*domain.Run, error) {
	f := &createRunFlow{o: o, req: req, recovery: recovery, runID: uuid.New()}
	defer f.release()
	if err := f.prepareRequest(ctx); err != nil {
		return nil, err
	}
	if replay, err := f.replayAccepted(ctx); replay != nil || err != nil {
		return replay, err
	}
	if err := f.admitCapacity(ctx); err != nil {
		return nil, err
	}
	if err := f.loadTask(ctx); err != nil {
		return nil, err
	}
	if err := f.resolveConfiguration(ctx); err != nil {
		return nil, err
	}
	if err := f.evaluatePolicy(ctx); err != nil {
		return nil, err
	}
	f.deriveRunMode()
	if err := f.applySpawnPolicy(); err != nil {
		return nil, err
	}
	if err := f.validateAdmission(); err != nil {
		return nil, err
	}
	if err := f.resolveExistingSandbox(ctx); err != nil {
		return nil, err
	}
	if err := f.freezeRunContract(ctx); err != nil {
		return nil, err
	}
	if err := f.authorizeDelegation(ctx); err != nil {
		return nil, err
	}
	if err := f.reserveIdempotency(ctx); err != nil {
		return nil, err
	}
	f.buildRun(ctx)
	if err := f.persistAttempt(ctx); err != nil {
		return nil, err
	}
	imageAttachments, err := f.launchAdmittedRun(ctx)
	if err != nil {
		return nil, err
	}
	return f.dispatch(ctx, imageAttachments)
}

// prepareRequest freezes the caller's request digest, bounds the declared work
// references and resolves the request's identity.
func (f *createRunFlow) prepareRequest(ctx context.Context) error {
	req := &f.req
	if req.requestDigest == "" {
		digest, err := createRunRequestDigest(*req)
		if err != nil {
			return err
		}
		req.requestDigest = digest
	}
	if len(req.WorkReferences) > maxWorkReferenceDeclarations {
		return domain.NewValidationErrorWithHint("work_references", "at most 100 declarations are allowed", "supply a bounded selected workload")
	}
	req.WorkReferences = append([]*eventpb.WorkReference(nil), req.WorkReferences...)
	for i, ref := range req.WorkReferences {
		if ref == nil || ref.GetKind() == "" || ref.GetId() == "" || proto.Size(ref) > 4096 {
			return domain.NewValidationErrorWithHint("work_references", "references need bounded kind and identity", "supply canonical workload references")
		}
		req.WorkReferences[i] = proto.Clone(ref).(*eventpb.WorkReference)
	}
	if req.caller == nil {
		// Private lifecycle creation retains its separately admitted contract.
		return f.o.resolveCreateRunIdentity(ctx, req)
	}
	return nil
}

// replayAccepted returns the already-accepted run for this idempotency key.
// Accepted identity outlives the one-hour cache and the deletable run row, so
// only a proven absence of durable acceptance may enter new admission.
func (f *createRunFlow) replayAccepted(ctx context.Context) (*domain.Run, error) {
	o, req := f.o, f.req
	if req.IdempotencyKey != "" && o.runs != nil {
		original, err := o.runs.GetByIdempotencyKey(ctx, req.IdempotencyKey)
		if err != nil {
			return nil, err
		}
		if original != nil {
			if err := validateRunIdentityReplay(original, req); err != nil {
				return nil, err
			}
			o.markIdempotencyComplete(ctx, req.IdempotencyKey, original.ID, "Run")
			return o.GetRun(ctx, original.ID)
		}
	}

	if req.IdempotencyKey == "" || o.idempotency == nil {
		return nil, nil
	}
	existing, err := o.idempotency.Check(ctx, req.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}
	// Request already processed - return cached result
	if existing.Status == domain.IdempotencyStatusComplete && existing.EntityID != nil {
		return o.getIdentityBoundRunReplay(ctx, *existing.EntityID, req)
	}
	if existing.Status == domain.IdempotencyStatusPending {
		// A prior accepted dispatch may have persisted its run before
		// the caller lost the response. Reconcile from durable owner
		// state by the run's idempotency key before refusing the
		// replay, so a lost response is not mistaken for an in-flight
		// duplicate. Only when no durable run exists yet is another
		// creation genuinely in progress with this key.
		if o.runs != nil {
			reconciled, err := o.runs.GetByIdempotencyKey(ctx, req.IdempotencyKey)
			if err != nil {
				return nil, err
			}
			if reconciled != nil {
				if err := validateRunIdentityReplay(reconciled, req); err != nil {
					return nil, err
				}
				o.markIdempotencyComplete(ctx, req.IdempotencyKey, reconciled.ID, "Run")
				return o.GetRun(ctx, reconciled.ID)
			}
		}
		return nil, domain.NewStateError("Run", "creating", "create",
			"a run creation with this idempotency key is already in progress")
	}
	// Failed status - allow retry by falling through
	return nil, nil
}

// admitCapacity takes the maintenance admission and enforces the concurrent
// run limit unless Force is set.
func (f *createRunFlow) admitCapacity(ctx context.Context) error {
	o := f.o
	releaseAdmission, err := o.admitMaintenance(ctx)
	if err != nil {
		return err
	}
	f.cleanup(releaseAdmission)

	if f.req.Force || o.config.MaxConcurrentRuns <= 0 || o.runs == nil {
		return nil
	}
	// Count active dispatched runs (Running and Starting); attached
	// interactive sessions never hold a dispatch slot.
	runningCount, err := o.runs.CountCapacityByStatus(ctx, domain.RunStatusRunning)
	if err != nil {
		return err
	}
	startingCount, err := o.runs.CountCapacityByStatus(ctx, domain.RunStatusStarting)
	if err != nil {
		return err
	}
	if activeCount := runningCount + startingCount; activeCount >= o.config.MaxConcurrentRuns {
		return &domain.CapacityExceededError{
			Resource: "concurrent_runs",
			Current:  activeCount,
			Maximum:  o.config.MaxConcurrentRuns,
		}
	}
	return nil
}

// loadTask loads the task and normalizes a copy of it; rejected admission must
// not update task storage.
func (f *createRunFlow) loadTask(ctx context.Context) error {
	o := f.o
	task, err := o.GetTask(ctx, f.req.TaskID)
	if err != nil {
		return err
	}
	taskCopy := *task
	f.expectedTaskSnapshot = taskCopy
	task = &taskCopy
	// Resolve relative project root to absolute (workspace-sandbox requires absolute paths).
	// Fall back to DefaultProjectRoot when the task has no project root set.
	if pr := strings.TrimSpace(task.ProjectRoot); pr == "" || !filepath.IsAbs(pr) {
		resolved := pr
		if resolved == "" {
			resolved = strings.TrimSpace(o.config.DefaultProjectRoot)
		}
		if resolved != "" && !filepath.IsAbs(resolved) {
			if abs, err := filepath.Abs(resolved); err == nil {
				resolved = abs
			}
		}
		if resolved != task.ProjectRoot {
			task.ProjectRoot = resolved
		}
	}
	if f.recovery != nil {
		copy := *task
		copy.ContextAttachments = f.recovery.attachments
		task = &copy
	}
	f.task = task
	return nil
}

// resolveConfiguration resolves the run configuration (profile plus inline
// overrides), pins a recovered run's retained contract, and resolves the
// sandbox configuration.
func (f *createRunFlow) resolveConfiguration(ctx context.Context) error {
	o, req, recovery := f.o, &f.req, f.recovery
	if req.AgentProfileID != nil && req.ProfileRef != nil {
		return domain.NewValidationErrorWithHint("agentProfileId/profileRef", "only one profile reference is allowed",
			"provide either agentProfileId or profileRef")
	}
	if req.caller != nil && req.ProfileRef != nil {
		var err error
		req.profileAdmission, err = o.planProfileAdmission(ctx, req.ProfileRef)
		if err != nil {
			return err
		}
	}
	req.usageModeEligible = recovery == nil
	resolvedConfig, profile, err := o.resolveRunConfig(ctx, *req)
	if err != nil {
		return err
	}
	if resolvedConfig == nil {
		return domain.NewInternalError("run configuration resolver returned nil configuration", nil)
	}
	if recovery != nil && recovery.config != nil {
		// Recovery is not a new profile selection. Copy the complete immutable
		// execution contract, including empty deny/allow lists, features, skill
		// experiment, extra flags, sandbox policy and candidate order. Current
		// admission/policy gates below may refuse it, never silently broaden it.
		data, err := json.Marshal(recovery.config)
		if err != nil {
			return err
		}
		pinned := new(domain.RunConfig)
		if err := json.Unmarshal(data, pinned); err != nil {
			return err
		}
		if pinned.PolicySnapshot == nil {
			pinned.PolicySnapshot = resolvedConfig.PolicySnapshot
		}
		if err := validateExecutionModel(pinned); err != nil {
			return domain.RefuseBeforeEffects(err)
		}
		pinned.Admission = nil // The replacement earns its own admission receipt.
		resolvedConfig = pinned
		if err := o.validateToolRestriction(resolvedConfig); err != nil {
			return err
		}
	}
	if err := o.validateRecordedModelFallback(ctx, *req, resolvedConfig); err != nil {
		return err
	}
	// `until` acceptance is unconditional: native delivery is gated by the
	// resolved spawn capability (NativeObjective) at execution time, not by a
	// coarse runner capability flag. A runner without native support still
	// receives the completion contract as a prompt suffix.
	if recovery == nil {
		applyCanary(resolvedConfig.PolicySnapshot, f.runID.String(), resolvedConfig.Model)
	}

	sandboxConfig, err := o.resolveSandboxConfig(*req, profile)
	if err != nil {
		return err
	}
	if recovery != nil && resolvedConfig.SandboxConfig != nil {
		sandboxConfig = resolvedConfig.SandboxConfig
	}
	f.resolvedConfig, f.profile, f.sandboxConfig = resolvedConfig, profile, sandboxConfig
	return nil
}

// evaluatePolicy refuses a request the run-request policies deny.
func (f *createRunFlow) evaluatePolicy(ctx context.Context) error {
	o := f.o
	if o.policy == nil {
		return nil
	}
	decision, err := o.policy.EvaluateRunRequest(ctx, policy.EvaluateRequest{
		Task:          f.task,
		Profile:       f.profile,
		RequestedMode: valueOrDefault(f.req.RunMode, domain.RunModeSandboxed),
		ForceInPlace:  f.req.ForceInPlace,
	})
	if err != nil {
		return domain.NewInternalError("policy evaluation failed", err)
	}
	if !decision.Allowed {
		return &domain.PolicyViolationError{
			PolicyID:   decision.DenialPolicy.ID,
			PolicyName: decision.DenialPolicy.Name,
			Rule:       "run_request",
			Message:    decision.DenialReason,
		}
	}
	f.policyDecision = decision
	return nil
}

// deriveRunMode picks the run mode.
//
// SandboxConfig.Mode is the single source of truth; DeriveRunMode
// translates the resolved Mode to a RunMode without consulting any
// other input. See docs/internal/SEAMS.md (RunMode decision boundary)
// and docs/internal/INVARIANTS.md.
//
// Decision priority (highest first):
//  1. Explicit caller override via req.RunMode
//  2. ForceInPlace (policy must permit; orchestrator validates that
//     the resolved sandbox mode is at or above policy's required
//     minimum in validateAdmission)
//  3. Derived from sandboxConfig.Mode
func (f *createRunFlow) deriveRunMode() {
	f.runMode = domain.DeriveRunMode(f.sandboxConfig)
	if f.req.RunMode != nil {
		f.runMode = *f.req.RunMode
	} else if f.req.ForceInPlace {
		f.runMode = domain.RunModeInPlace
	}
}

// applySpawnPolicy resolves declaration-only spawn preferences against the
// selected runner's published capabilities before the run becomes immutable.
func (f *createRunFlow) applySpawnPolicy() error {
	profile := f.profile
	if f.recovery != nil || profile == nil || profile.SpawnPolicy == nil {
		return nil
	}
	selected, err := f.o.runners.Get(f.resolvedConfig.RunnerType)
	if err != nil {
		return domain.NewValidationError("spawnPolicy", "selected runner is unavailable: "+err.Error())
	}
	constraints := spawnPolicyConstraintsForRequest(f.req)
	resolution, err := ResolveSpawnPolicyWithConstraints(profile.SpawnPolicy, selected.Capabilities(), constraints)
	if err != nil {
		return domain.NewValidationError("spawnPolicy", err.Error())
	}
	f.req.ExecutionMode = domain.ExecutionMode(resolution.ExecutionMode)
	f.sandboxConfig.Mode = domain.SandboxMode(resolution.SandboxMode)
	f.runMode = domain.DeriveRunMode(f.sandboxConfig)
	f.spawnSkips = resolution.Skipped
	if resolution.Fallback != "" && f.resolvedConfig.PolicySnapshot != nil {
		// A declared capability was used because no preferred combination was
		// feasible. Surface it on the run's policy snapshot so run get and the
		// execution record show why a non-preferred substrate was used.
		reason := "spawn_fallback:" + resolution.Fallback
		snapshot := f.resolvedConfig.PolicySnapshot
		if snapshot.SelectionReason == "" {
			snapshot.SelectionReason = reason
		} else {
			snapshot.SelectionReason += ";" + reason
		}
	}
	return nil
}

// validateAdmission checks the resolved sandbox, containment, input and scope
// path against the policy-required minimum.
func (f *createRunFlow) validateAdmission() error {
	// Revalidate after spawn preferences: selecting another mode must not
	// discard a runtime write policy inherited from the owner profile.
	if err := validateSandboxConfig(f.sandboxConfig); err != nil {
		return err
	}
	if err := validateResolvedRunMode(f.runMode, f.sandboxConfig); err != nil {
		return err
	}
	if err := validateExecutionContainment(f.req.ExecutionMode, f.sandboxConfig.Mode, f.resolvedConfig); err != nil {
		return err
	}
	if err := validateEffectiveRunInput(f.task, f.req.Prompt); err != nil {
		return err
	}

	// Enforce the policy-declared minimum sandbox mode. The policy layer
	// expresses sandbox requirements as a minimum SandboxMode rather
	// than a bool so a higher-strictness policy can require Protected
	// while still allowing Tracking-mode runs through other paths.
	if decision := f.policyDecision; decision != nil && decision.RequiredSandboxMode != domain.SandboxModeUnspecified {
		resolvedMode := domain.SandboxModeOff
		if f.sandboxConfig != nil {
			resolvedMode = f.sandboxConfig.Mode.Effective()
		}
		if !resolvedMode.AtLeast(decision.RequiredSandboxMode) {
			return domain.NewValidationErrorWithHint(
				"sandboxConfig.mode",
				"resolved sandbox mode is below the policy-required minimum",
				fmt.Sprintf("policy requires Mode >= %q; resolved Mode is %q",
					decision.RequiredSandboxMode, resolvedMode),
			)
		}
	}

	return f.o.validateScopePath(f.task, f.runMode, f.req.ExistingSandboxID, false)
}

// A protected launcher requires an actual sandbox. Reject a contradictory
// caller override before creating a run instead of failing after dispatch.
func validateResolvedRunMode(mode domain.RunMode, cfg *domain.SandboxConfig) error {
	if mode == domain.RunModeInPlace && cfg != nil && cfg.Mode.Effective() == domain.SandboxModeProtected {
		return domain.NewValidationErrorWithHint("runMode", "protected sandbox mode requires a sandboxed run", "omit the in-place override or select a profile with sandbox mode off")
	}
	return nil
}

// resolveExistingSandbox validates a reused sandbox against the task and finds
// its work directory, noting whether it must be started first.
func (f *createRunFlow) resolveExistingSandbox(ctx context.Context) error {
	o, req, task := f.o, f.req, f.task
	if req.ExistingSandboxID == nil {
		return nil
	}
	if f.runMode != domain.RunModeSandboxed {
		return domain.NewValidationErrorWithHint("existingSandboxId", "existing sandbox requires sandboxed run mode",
			"set runMode to sandboxed or set sandboxConfig.mode to a sandbox-enabled value (tracking/protected)")
	}
	if o.sandbox == nil {
		return domain.NewConfigMissingError("sandbox", "provider not configured", nil)
	}

	sbx, err := o.sandbox.Get(ctx, *req.ExistingSandboxID)
	if err != nil {
		return err
	}
	switch sbx.Status {
	case sandbox.SandboxStatusDeleted, sandbox.SandboxStatusRejected, sandbox.SandboxStatusApproved, sandbox.SandboxStatusError:
		return domain.NewValidationErrorWithHint("existingSandboxId", "sandbox is not reusable",
			fmt.Sprintf("sandbox status is %s", sbx.Status))
	case sandbox.SandboxStatusStopped:
		f.sandboxToStart = &sbx.ID
	}

	if trimmed := strings.TrimSpace(task.ProjectRoot); trimmed != "" && strings.TrimSpace(sbx.ProjectRoot) != "" && trimmed != sbx.ProjectRoot {
		return domain.NewValidationErrorWithHint("existingSandboxId", "sandbox project root does not match task",
			fmt.Sprintf("task projectRoot=%q, sandbox projectRoot=%q", trimmed, sbx.ProjectRoot))
	}
	if trimmed := strings.TrimSpace(task.ScopePath); trimmed != "" && strings.TrimSpace(sbx.ScopePath) != "" && trimmed != sbx.ScopePath {
		return domain.NewValidationErrorWithHint("existingSandboxId", "sandbox scope path does not match task",
			fmt.Sprintf("task scopePath=%q, sandbox scopePath=%q", trimmed, sbx.ScopePath))
	}

	if sbx.WorkDir != "" {
		f.existingSandboxWorkDir = sbx.WorkDir
		return nil
	}
	workDir, err := o.sandbox.GetWorkspacePath(ctx, sbx.ID)
	if err != nil {
		return err
	}
	f.existingSandboxWorkDir = workDir
	return nil
}

// freezeRunContract fixes the workload, tag, billing snapshot, prompt split
// and admission receipt that the run persists immutably.
func (f *createRunFlow) freezeRunContract(ctx context.Context) error {
	o, req, task, resolvedConfig := f.o, f.req, f.task, f.resolvedConfig
	workload := domain.WorkloadRef{Kind: req.WorkloadKind, Key: strings.TrimSpace(req.WorkloadKey), Instance: strings.TrimSpace(req.WorkloadInstance)}
	if workload.Kind == "" {
		workload.Kind = domain.WorkloadKindAdhoc
	}
	if workload.Key != "" {
		if req.WorkloadKind == "" {
			workload.Kind = domain.WorkloadKindInteractive
		}
	}
	if !workload.Kind.IsValid() {
		return fmt.Errorf("invalid workload kind %q", workload.Kind)
	}
	f.workload = workload
	tag := strings.TrimSpace(req.Tag)
	if tag == "" && workload.Key != "" {
		tag = workload.Key
		if workload.Instance != "" {
			tag += "#" + workload.Instance
		}
	}
	f.tag = tag
	billing := domain.BillingSnapshot{Mode: domain.BillingModeUnknown}
	if resolvedConfig.PolicySnapshot != nil {
		billing = resolvedConfig.PolicySnapshot.SelectedCandidate.Billing
		if billing.Mode == "" {
			billing.Mode = domain.BillingModeUnknown
		}
	}
	if billing.Basis == "" {
		billing.Basis = billing.EffectiveBasis()
	}
	if billing.ObservedAt.IsZero() {
		billing.ObservedAt = time.Now().UTC()
	}
	f.billing = billing
	resolvedConfig.Billing = billing
	// Split the prompt before persisting the immutable resolved config so the
	// known injected instruction estimate survives event pruning and replay.
	f.systemPrompt, f.userMessage = domain.BuildSplitPrompt(task.Description, task.ContextAttachments, req.Prompt)
	if strings.TrimSpace(f.systemPrompt) != "" {
		estimate := tokenaccounting.EstimateText(f.systemPrompt)
		resolvedConfig.PreambleInjectedTokens = estimate.Tokens
		resolvedConfig.PreambleTokenBasis = estimate.Basis
	}
	// Bind the caller's requested settings to the owner-resolved effective
	// settings before persistence so a fresh reader can tell requested from
	// effective without consulting mutable policy or a transcript.
	// Native controls depend on the final sandbox policy, including retained
	// recovery grants. Attach it before recording admission, not afterward.
	resolvedConfig.SandboxConfig = f.sandboxConfig
	resolvedConfig.Admission = buildRunAdmission(req, resolvedConfig)
	identityOwnerScopes := req.OwnerScopes
	if identityOwnerScopes == nil {
		identityOwnerScopes = []string{}
	}
	identityScopes := identity.IntersectScopes(identityOwnerScopes, f.profile.IdentityScopeCeiling(), req.RequestedScopes)
	resolvedConfig.Admission.FrozenIdentityScopes = &identityScopes
	// Record the runner-native control arguments the selected codec emits for
	// the resolved configuration, so the "passed" layer is durable alongside
	// the requested and effective layers.
	o.recordPassedInvocation(ctx, resolvedConfig.Admission, resolvedConfig)
	return nil
}

// authorizeDelegation applies the dependent-delegation prerequisite gate: a
// run created as a child of an admitted parent is delegated work. It may
// proceed only when the parent carries a live qualification receipt whose
// effective identity matches this run's resolved identity exactly, and retains
// its execution restrictions. The qualification probe has no parent and is not
// subject to this gate. It then revalidates the caller and acquires delivery
// admission.
func (f *createRunFlow) authorizeDelegation(ctx context.Context) error {
	o, req := f.o, &f.req
	qualifiedParent, err := o.dependentDelegationParent(ctx, *req, f.resolvedConfig)
	if err != nil {
		return err
	}
	f.qualifiedParent = qualifiedParent
	if qualifiedParent != nil {
		req.WorkReferences, err = inheritParentEffortReferences(qualifiedParent, req.WorkReferences)
		if err != nil {
			return err
		}
		if len(req.WorkReferences) > maxWorkReferenceDeclarations {
			return domain.NewValidationErrorWithHint("work_references", "at most 100 declarations are allowed after parent effort attribution", "reduce explicit declarations to leave room for inherited effort references")
		}
	}
	if err := o.revalidateCreateRunCaller(ctx, *req); err != nil {
		return err
	}
	if o.deliveryEfforts != nil {
		bindings, _, release, err := o.deliveryEfforts.AcquireDeliveryAdmission(ctx, req.WorkReferences, qualifiedParent)
		if err != nil {
			return err
		}
		f.cleanup(release)
		f.resolvedConfig.DeliveryCommissions = bindings
	}
	return nil
}

// reserveIdempotency reserves the request's idempotency key. Caller, owner
// narrowing, profile policy and dependent qualification have all passed; only
// now may admission reserve or mutate durable state.
func (f *createRunFlow) reserveIdempotency(ctx context.Context) error {
	o, key := f.o, f.req.IdempotencyKey
	if key == "" || o.idempotency == nil {
		return nil
	}
	// Reserve only after owner admission; accepted replays above remain reads.
	if _, err := o.idempotency.Reserve(ctx, key, 1*time.Hour); err != nil {
		// If reservation fails, another request beat us to it
		return domain.NewStateError("Run", "creating", "create",
			"a run creation with this idempotency key is already in progress")
	}
	return nil
}

// buildRun assembles the pending run record from the admitted contract.
func (f *createRunFlow) buildRun(ctx context.Context) {
	o, req, task, resolvedConfig := f.o, f.req, f.task, f.resolvedConfig
	profileID := req.AgentProfileID
	if f.profile != nil {
		profileID = &f.profile.ID
	}
	run := &domain.Run{
		ID:                       f.runID,
		TaskID:                   task.ID,
		AgentProfileID:           profileID, // May be nil if inline config used
		Tag:                      f.tag,     // Custom tag for identification
		Label:                    strings.TrimSpace(task.Title),
		LabelSource:              domain.RunLabelSourceDerived,
		OwnerSubject:             req.OwnerSubject,
		OwnerScopes:              slices.Clone(req.OwnerScopes),
		RequestedScopes:          slices.Clone(req.RequestedScopes),
		OwnerExpiresAt:           req.OwnerExpiresAt,
		Workload:                 f.workload,
		Billing:                  f.billing,
		SourceRunIDs:             req.SourceRunIDs,
		WorkReferences:           req.WorkReferences,
		SourceInvestigationRunID: req.SourceInvestigationRunID,
		RunMode:                  f.runMode,
		ExecutionMode:            req.ExecutionMode,
		Status:                   domain.RunStatusPending,
		Phase:                    domain.RunPhaseQueued,
		ProgressPercent:          0,
		IdempotencyKey:           req.IdempotencyKey,
		ApprovalState:            domain.ApprovalStateNone,
		ResolvedConfig:           resolvedConfig,
		SandboxConfig:            f.sandboxConfig,
		ConversationID:           req.ConversationID,
		ParentRunID:              req.ParentRunID,
		// Persist caller-supplied custom env so the continue/wake path can
		// re-inject it. Already VROOLI_*-validated at the API boundary.
		CustomEnv: req.Environment,
		// Provenance: requested is the primary model the preset expanded to at creation.
		// Actual is blank until the executor records the model that actually ran.
		RequestedModel: resolvedConfig.Model,
		CanaryArm:      resolvedConfig.PolicySnapshot.CanaryArm,
		CreatedAt:      o.now(),
		UpdatedAt:      o.now(),
	}
	if run.Label == "" {
		run.Label = "Agent run"
	}
	// Apply Decision D7 precedence (spawner > parent inheritance > fresh
	// UUID). When the spawn surface populates ConversationID directly,
	// step (1) wins; otherwise we inherit from ParentRunID's run when set,
	// or mint a fresh UUID.
	run.ConversationID = domain.ResolveConversationID(run, func(parentID uuid.UUID) (string, bool) {
		parent, perr := o.runs.Get(ctx, parentID)
		if perr != nil || parent == nil {
			return "", false
		}
		return parent.ConversationID, true
	})
	// Populate PromptPreview so WebSocket broadcasts include display text.
	// This is normally a computed field from the List query JOIN, but we need it
	// for real-time broadcasts during execution.
	if len(task.Description) > 120 {
		run.PromptPreview = task.Description[:120]
	} else {
		run.PromptPreview = task.Description
	}
	if req.ExistingSandboxID != nil {
		run.SandboxID = req.ExistingSandboxID
	}
	f.run = run
}

// persistAttempt reserves the dispatcher-owned cancellation fence, then
// atomically persists the current attempt and its Run receipt before any
// sandbox start or runner dispatch. The idempotency key is reserved, so a
// failure here frees it for retry.
func (f *createRunFlow) persistAttempt(ctx context.Context) error {
	o, run := f.o, f.run
	// Reserve the dispatcher-owned cancellation fence before the durable Task
	// claim. Cancellation can close this local gate while the atomic owner
	// transaction decides whether this attempt was admitted.
	admission, err := o.dispatcher.ReserveAdmission(run.ID)
	if err != nil {
		f.failReservation(ctx)
		return err
	}
	f.admission = admission
	f.cleanup(func() {
		if !f.admissionHandedOff {
			o.dispatcher.ReleaseAdmission(run.ID, admission)
		}
	})

	// The repository CAS accepts only a queued unowned task or a terminal
	// exact current attempt; ambiguous legacy histories remain refused rather
	// than selecting a latest Run.
	if o.runs == nil {
		f.failReservation(ctx)
		return domain.NewConfigMissingError("runs", "repository not configured", nil)
	}
	if err := o.runs.CreateTaskAttempt(ctx, f.task, &f.expectedTaskSnapshot, run, f.task.CurrentRunID); err != nil {
		f.failReservation(ctx)
		return err
	}
	// This flow owns its Task view; the repository leaves its inputs immutable.
	f.task.Status = domain.TaskStatusRunning
	id := run.ID
	f.task.CurrentRunID = &id
	f.task.UpdatedAt = run.UpdatedAt
	o.markIdempotencyComplete(ctx, f.req.IdempotencyKey, run.ID, "Run")
	return nil
}

// failAdmittedRun persists the admitted run as failed with cause.
func (f *createRunFlow) failAdmittedRun(ctx context.Context, cause error) error {
	run := f.run
	run.Status = domain.RunStatusFailed
	run.ErrorMsg = cause.Error()
	endedAt := f.o.now()
	run.EndedAt = &endedAt
	if persistErr := f.o.runs.Update(context.WithoutCancel(ctx), run); persistErr != nil {
		return fmt.Errorf("%v; persist admitted run failure: %w", cause, persistErr)
	}
	return cause
}

// launchAdmittedRun starts a stopped reused sandbox, records the launch events
// and metrics, resolves image attachments and emits the initial user message.
// It returns the image attachments the runner receives.
func (f *createRunFlow) launchAdmittedRun(ctx context.Context) ([]runner.Attachment, error) {
	o, task := f.o, f.task
	if err := o.preflightScopePath(task, f.runMode, f.req.ExistingSandboxID); err != nil {
		return nil, f.failAdmittedRun(ctx, err)
	}
	if err := f.startReusedSandbox(ctx); err != nil {
		return nil, err
	}
	f.recordLaunchEvents(ctx)

	// Split instructions (system prompt) from context data (user message).
	// Task description contains methodology/instructions → system prompt.
	// Context attachments contain data/evidence → user message.
	// If an override prompt is provided, it replaces the task description as system prompt.
	// Resolve image attachments from storage so runners receive file paths
	var imageAttachments []runner.Attachment
	if o.storage != nil {
		for _, att := range task.ContextAttachments {
			if att.Type == "image" && att.AttachmentID != "" {
				meta, err := o.storage.Get(ctx, att.AttachmentID)
				if err != nil {
					continue // skip unresolvable attachments
				}
				imageAttachments = append(imageAttachments, runner.Attachment{
					ID:          meta.ID,
					FileName:    meta.FileName,
					ContentType: meta.ContentType,
					FilePath:    o.storage.GetFilePath(meta.StoragePath),
				})
			}
		}
	}
	f.emitInitialPrompt(ctx, imageAttachments)
	if f.admission.Cancelled() {
		return nil, domain.NewStateError("Task", string(domain.TaskStatusCancelled), "create run", "task was cancelled before the dispatcher claim; the admitted Run will not start")
	}
	return imageAttachments, nil
}

// startReusedSandbox starts a stopped reused sandbox under the task-attempt
// cancellation fence.
func (f *createRunFlow) startReusedSandbox(ctx context.Context) error {
	if f.sandboxToStart == nil {
		return nil
	}
	o, run, admission := f.o, f.run, f.admission
	if !admission.BeginExternalStart() {
		return f.failAdmittedRun(ctx, domain.NewStateError("Task", string(f.task.Status), "start run", "task attempt was cancelled before sandbox startup"))
	}
	err := o.sandbox.Start(admission.Context(), *f.sandboxToStart)
	admission.EndExternalStart()
	if err != nil {
		if admission.Cancelled() {
			stopErr := o.StopRun(context.WithoutCancel(ctx), run.ID)
			return domain.NewStateError("Run", "start_uncertain", "create", fmt.Sprintf("sandbox start returned after task cancellation; provider effect may have occurred (start=%v stop=%v)", err, stopErr))
		}
		return f.failAdmittedRun(ctx, err)
	}
	if admission.Cancelled() {
		if stopErr := o.StopRun(context.WithoutCancel(ctx), run.ID); stopErr != nil {
			return domain.NewStateError("Run", "start_uncertain", "create", fmt.Sprintf("sandbox started while task cancellation won; exact run stop failed: %v", stopErr))
		}
		return domain.NewStateError("Run", "start_uncertain", "create", "task was cancelled during sandbox startup; the exact Run was stopped but the provider sandbox may remain active")
	}
	return nil
}

// recordLaunchEvents appends the advisory tool-restriction and skipped
// spawn-preference events and records the run-creation metrics.
func (f *createRunFlow) recordLaunchEvents(ctx context.Context) {
	o, run, resolvedConfig := f.o, f.run, f.resolvedConfig
	if o.toolRestrictionIsAdvisory(resolvedConfig) && o.events != nil {
		declared := "allowedTools"
		if len(resolvedConfig.AllowedTools) == 0 {
			declared = "deniedTools"
		}
		if err := o.events.Append(ctx, run.ID, domain.NewLogEvent(run.ID, "warn",
			fmt.Sprintf("runner %q cannot enforce %s; advisory policy accepted the launch", resolvedConfig.RunnerType, declared))); err != nil {
			obs.Component("orchestrator").Warn("failed to append advisory tool-restriction event", obs.KeyRunID, run.ID.String(), "eventType", "log", obs.KeyError, err.Error())
		}
	}
	if o.events != nil {
		for _, skipped := range f.spawnSkips {
			message := fmt.Sprintf("spawn preference skipped: executionMode=%s sandboxMode=%s reason=%s", skipped.ExecutionMode, skipped.SandboxMode, skipped.Reason)
			if err := o.events.Append(ctx, run.ID, domain.NewLogEvent(run.ID, "info", message)); err != nil {
				obs.Component("orchestrator").Warn("failed to append spawn-preference event", obs.KeyRunID, run.ID.String(), "eventType", "log", obs.KeyError, err.Error())
			}
		}
	}

	// Sandbox-default rollout adoption metrics (Phase D of
	// agent-sandbox-audit-foundation). Three labels capture the rollout
	// state per run: run_mode, sandbox_mode, manual_review.
	sandboxModeLabel := "n/a"
	manualReviewLabel := "false"
	if run.SandboxConfig != nil {
		sandboxModeLabel = string(run.SandboxConfig.Mode.Effective())
		if run.SandboxConfig.ManualReview {
			manualReviewLabel = "true"
		}
	}
	metrics.Get().RecordRunCreated(string(resolvedConfig.RunnerType), string(run.RunMode))
	metrics.Get().RecordSandboxAdoption(string(run.RunMode), sandboxModeLabel, manualReviewLabel)
}

// emitInitialPrompt emits the initial user prompt as the first message event.
// It emits the user message (context + task), not the system prompt, since the
// system prompt is runner-internal instructions.
func (f *createRunFlow) emitInitialPrompt(ctx context.Context, imageAttachments []runner.Attachment) {
	o, run := f.o, f.run
	if o.events == nil || strings.TrimSpace(f.userMessage) == "" {
		return
	}
	// Build attachment metadata for the event so the UI can render image thumbnails
	var attInfo []domain.MessageAttachmentInfo
	for _, att := range imageAttachments {
		meta, err := o.storage.Get(ctx, att.ID)
		if err == nil {
			attInfo = append(attInfo, domain.MessageAttachmentInfo{
				ID:          meta.ID,
				FileName:    meta.FileName,
				ContentType: meta.ContentType,
				URL:         o.storage.GetServingURL(meta.StoragePath),
			})
		}
	}
	var userEvent *domain.RunEvent
	if len(attInfo) > 0 {
		userEvent = domain.NewMessageEventWithAttachments(run.ID, "user", f.userMessage, attInfo)
	} else {
		userEvent = domain.NewMessageEvent(run.ID, "user", f.userMessage)
	}
	if err := o.appendAndBroadcastEvents(ctx, run.ID, userEvent); err != nil {
		obs.Component("orchestrator").Warn("failed to append initial user message", obs.KeyRunID, run.ID.String(), "eventType", "message", obs.KeyError, err.Error())
	}
}

// dispatch hands the executor body to the spawn dispatcher. Enqueue is the
// only path through which a run begins — direct goroutine spawning would skip
// startup serialization (codex SQLite WAL contention) and queue-depth
// surfacing.
func (f *createRunFlow) dispatch(ctx context.Context, imageAttachments []runner.Attachment) (*domain.Run, error) {
	o, run, admission := f.o, f.run, f.admission
	// Snapshot before enqueue: attachRunActions projects the caller's record
	// immediately after enqueue, while the dispatcher may start execution at
	// once. The asynchronous executor must not copy that record concurrently.
	executionRun := *run
	task, profile, req := f.task, f.profile, f.req
	if err := o.dispatcher.EnqueueAdmitted(&spawn.Job{
		RunID:      run.ID,
		RunMode:    run.RunMode,
		RunnerType: runnerTypeOrEmpty(run),
		Sink:       o.dispatcherSink(run.ID),
		Fn: func(started spawn.StartedFn) {
			defer o.dispatcher.ReleaseAdmission(run.ID, admission)
			defer obs.RecoverToFailure("run execution dispatch", func(failure obs.PanicFailure) {
				o.recoverPanickedRun(run, failure)
			})
			executionRunCopy := executionRun
			o.executeRun(admission.Context(), &executionRunCopy, task, profile, f.userMessage, f.systemPrompt, f.existingSandboxWorkDir, imageAttachments, req.Environment, started)
		},
		OnPanic: func(failure obs.PanicFailure) {
			o.recoverPanickedRun(run, failure)
		},
	}, admission); err != nil {
		return nil, f.failAdmittedRun(ctx, err)
	}
	f.admissionHandedOff = true

	return o.attachRunActions(ctx, run), nil
}

// recoverPanickedRun contains a panic at an execution-goroutine boundary. The
// failure state uses the normal phase path so the run reaches the same terminal
// status and broadcaster contract as an ordinary executor error; the full
// stack is retained as a protected run event for postmortem triage.
func (o *Orchestrator) recoverPanickedRun(run *domain.Run, failure obs.PanicFailure) {
	if run == nil {
		obs.Component("orchestrator").Error("recovered run panic without run", obs.KeyError, failure.Error())
		return
	}
	ctx := context.Background()
	phases.FailWithError(ctx, phases.FailWithErrorInput{
		Deps: phases.Deps{Runs: o.runs, Events: o.events, Broadcaster: o.broadcaster},
		Run:  run,
		Err:  failure,
	})
	stackEvent := domain.NewLogEvent(run.ID, "error", "panic recovered in "+failure.Operation+"\n"+failure.Stack)
	if err := o.appendAndBroadcastEvents(ctx, run.ID, stackEvent); err != nil {
		obs.Component("orchestrator").Error("failed to append recovered panic stack event", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
	}
}

func (o *Orchestrator) preflightScopePath(task *domain.Task, runMode domain.RunMode, existingSandboxID *uuid.UUID) error {
	return o.validateScopePath(task, runMode, existingSandboxID, true)
}

// Missing directories are inspected before admission and created only after it.
func (o *Orchestrator) validateScopePath(task *domain.Task, runMode domain.RunMode, existingSandboxID *uuid.UUID, createMissing bool) error {
	if runMode != domain.RunModeSandboxed || existingSandboxID != nil {
		return nil
	}

	scopePath := strings.TrimSpace(task.ScopePath)
	if scopePath == "" {
		return domain.NewValidationError("scopePath", "field is required")
	}

	projectRoot := strings.TrimSpace(task.ProjectRoot)
	if projectRoot == "" {
		projectRoot = strings.TrimSpace(o.config.DefaultProjectRoot)
	}
	if projectRoot == "" && !filepath.IsAbs(scopePath) {
		return domain.NewValidationErrorWithHint("projectRoot", "field is required for sandboxed run",
			"set projectRoot on the task or configure defaultProjectRoot")
	}

	absScopePath := scopePath
	if !filepath.IsAbs(absScopePath) && projectRoot != "" {
		absScopePath = filepath.Join(projectRoot, absScopePath)
	}
	absScopePath = filepath.Clean(absScopePath)

	info, err := os.Stat(absScopePath)
	if err != nil {
		if os.IsNotExist(err) {
			if !createMissing {
				return nil
			}
			if mkErr := os.MkdirAll(absScopePath, 0o755); mkErr != nil {
				return domain.NewValidationErrorWithHint("scopePath", "scope path does not exist",
					fmt.Sprintf("create the directory: %s", absScopePath))
			}
			info, err = os.Stat(absScopePath)
			if err != nil {
				return domain.NewValidationErrorWithHint("scopePath", "unable to stat scope path",
					fmt.Sprintf("check permissions for %s", absScopePath))
			}
		}
		if err != nil {
			return domain.NewValidationErrorWithHint("scopePath", "unable to stat scope path",
				fmt.Sprintf("check permissions for %s", absScopePath))
		}
	}
	if !info.IsDir() {
		return domain.NewValidationErrorWithHint("scopePath", "scope path is not a directory",
			fmt.Sprintf("scope path resolves to %s", absScopePath))
	}

	return nil
}

// markIdempotencyFailed marks an idempotency key as failed (allows retry).
func (o *Orchestrator) markIdempotencyFailed(ctx context.Context, key string) {
	if key == "" || o.idempotency == nil {
		return
	}
	if err := o.idempotency.Fail(ctx, key); err != nil {
		obs.Component("orchestrator").Warn("failed to mark idempotency key failed", "idempotencyKey", key, obs.KeyError, err.Error())
	}
}

// markIdempotencyComplete marks an idempotency key as successfully completed.
func (o *Orchestrator) markIdempotencyComplete(ctx context.Context, key string, entityID uuid.UUID, entityType string) {
	if key == "" || o.idempotency == nil {
		return
	}
	if err := o.idempotency.Complete(ctx, key, entityID, entityType, nil); err != nil {
		obs.Component("orchestrator").Warn("failed to mark idempotency key complete", "idempotencyKey", key, "entityId", entityID.String(), "entityType", entityType, obs.KeyError, err.Error())
	}
}

// spawnPolicyConstraintsForRequest captures caller-selected axes before the
// profile preference resolver can choose among runner capabilities.
func spawnPolicyConstraintsForRequest(req CreateRunRequest) SpawnPolicyConstraints {
	constraints := SpawnPolicyConstraints{
		ExecutionMode: string(req.ExecutionMode),
		ForceInPlace:  req.ForceInPlace,
	}
	if req.SandboxConfig != nil && req.SandboxConfig.Mode != domain.SandboxModeUnspecified {
		constraints.SandboxMode = string(req.SandboxConfig.Mode)
	}
	constraints.RunMode = req.RunMode
	return constraints
}
