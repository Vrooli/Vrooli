// This file creates runs and their initial persisted execution state.
package orchestration

import (
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/metrics"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/orchestration/phases"
	"agent-manager/internal/orchestration/spawn"
	"agent-manager/internal/policy"
	"agent-manager/internal/tokenaccounting"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/effortauthority"
	isolation "github.com/vrooli/vrooli/packages/nativeisolation"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/proto"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

func (o *Orchestrator) CreateRun(ctx context.Context, req CreateRunRequest) (*domain.Run, error) {
	// Public caller projections are not authority, including in-process callers.
	req.caller, req.profileAdmission, req.effort = nil, nil, nil
	req.OwnerSubject, req.OwnerScopes, req.OwnerExpiresAt = "", nil, nil
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

func (o *Orchestrator) createRun(ctx context.Context, req CreateRunRequest, recovery *runRecoveryContext) (*domain.Run, error) {
	if len(req.WorkReferences) > 100 {
		return nil, domain.NewValidationErrorWithHint("work_references", "at most 100 declarations are allowed", "supply a bounded selected workload")
	}
	req.WorkReferences = append([]*eventpb.WorkReference(nil), req.WorkReferences...)
	for i, ref := range req.WorkReferences {
		if ref == nil || ref.GetKind() == "" || ref.GetId() == "" || proto.Size(ref) > 4096 {
			return nil, domain.NewValidationErrorWithHint("work_references", "references need bounded kind and identity", "supply canonical workload references")
		}
		req.WorkReferences[i] = proto.Clone(ref).(*eventpb.WorkReference)
	}
	if req.caller == nil {
		// Private lifecycle creation retains its separately admitted contract.
		if err := o.resolveCreateRunIdentity(ctx, &req); err != nil {
			return nil, err
		}
	}
	if req.effort != nil && o.finiteNativeFactory.CheckBinding(req.effort.binding.PolicyID, isolation.Binding{PolicyDigest: req.effort.binding.PolicyDigest, ProfileDigest: req.effort.intent.ProfileDigest, Repository: req.effort.intent.Repository, Deadline: req.effort.binding.Deadline}) != nil {
		return nil, effortauthority.ErrRefused
	}
	if o.finiteNativeFactory.Enabled() && (req.effort == nil || recovery != nil || req.effort.intent.Effect != "run.create") {
		return nil, effortauthority.ErrRefused
	}
	reserved := false
	failCreation := func() {
		if reserved {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
		}
	}
	// Accepted identity outlives the one-hour cache and the deletable run row.
	// Only a proven absence of durable acceptance may enter new admission.
	if req.IdempotencyKey != "" && o.runs != nil {
		original, err := o.runs.GetByIdempotencyKey(ctx, req.IdempotencyKey)
		if err != nil {
			return nil, err
		}
		if original != nil {
			if err := validateRunIdentityReplay(original, req); err != nil {
				return nil, err
			}
			if err := o.acceptEffortRun(ctx, original); err != nil {
				return nil, err
			}
			o.markIdempotencyComplete(ctx, req.IdempotencyKey, original.ID, "Run")
			return o.GetRun(ctx, original.ID)
		}
	}

	// IDEMPOTENCY: Check if this request has already been processed
	if req.IdempotencyKey != "" && o.idempotency != nil {
		existing, err := o.idempotency.Check(ctx, req.IdempotencyKey)
		if err != nil {
			return nil, err
		}
		if existing != nil {
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
						if err := o.acceptEffortRun(ctx, reconciled); err != nil {
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
		}

	}
	releaseAdmission, err := o.admitMaintenance(ctx)
	if err != nil {
		return nil, err
	}
	defer releaseAdmission()

	// SLOT ENFORCEMENT: Check capacity unless Force is set
	if !req.Force && o.config.MaxConcurrentRuns > 0 && o.runs != nil {
		// Count active runs (both Running and Starting count against the limit)
		runningCount, err := o.runs.CountByStatus(ctx, domain.RunStatusRunning)
		if err != nil {
			failCreation()
			return nil, err
		}
		startingCount, err := o.runs.CountByStatus(ctx, domain.RunStatusStarting)
		if err != nil {
			failCreation()
			return nil, err
		}

		activeCount := runningCount + startingCount
		if activeCount >= o.config.MaxConcurrentRuns {
			failCreation()
			return nil, &domain.CapacityExceededError{
				Resource: "concurrent_runs",
				Current:  activeCount,
				Maximum:  o.config.MaxConcurrentRuns,
			}
		}
	}

	// Get task
	task, err := o.GetTask(ctx, req.TaskID)
	if err != nil {
		failCreation()
		return nil, err
	}

	// Normalize a copy; rejected admission must not update task storage.
	taskCopy := *task
	task = &taskCopy
	taskRootChanged := false
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
			taskRootChanged = true
		}
	}

	if req.AgentProfileID != nil && req.ProfileRef != nil {
		failCreation()
		return nil, domain.NewValidationErrorWithHint("agentProfileId/profileRef", "only one profile reference is allowed",
			"provide either agentProfileId or profileRef")
	}

	// Resolve configuration: profile (if provided) + inline overrides
	if recovery != nil {
		copy := *task
		copy.ContextAttachments = recovery.attachments
		task = &copy
	}
	if req.caller != nil && req.ProfileRef != nil {
		var err error
		req.profileAdmission, err = o.planProfileAdmission(ctx, req.ProfileRef)
		if err != nil {
			return nil, err
		}
	}
	runID := uuid.New()
	resolvedConfig, profile, err := o.resolveRunConfig(ctx, req)
	if err != nil {
		failCreation()
		return nil, err
	}
	if resolvedConfig == nil {
		failCreation()
		return nil, domain.NewInternalError("run configuration resolver returned nil configuration", nil)
	}
	if recovery != nil && recovery.config != nil {
		// Recovery is not a new profile selection. Copy the complete immutable
		// execution contract, including empty deny/allow lists, features, skill
		// experiment, extra flags, sandbox policy and candidate order. Current
		// admission/policy gates below may refuse it, never silently broaden it.
		data, err := json.Marshal(recovery.config)
		if err != nil {
			return nil, err
		}
		pinned := new(domain.RunConfig)
		if err := json.Unmarshal(data, pinned); err != nil {
			return nil, err
		}
		if pinned.PolicySnapshot == nil {
			pinned.PolicySnapshot = resolvedConfig.PolicySnapshot
		}
		if err := validateExecutionModel(pinned); err != nil {
			return nil, domain.RefuseBeforeEffects(err)
		}
		pinned.Admission = nil // The replacement earns its own admission receipt.
		resolvedConfig = pinned
		if err := o.validateToolRestriction(resolvedConfig); err != nil {
			return nil, err
		}
	}
	// `until` acceptance is unconditional: native delivery is gated by the
	// resolved spawn capability (NativeObjective) at execution time, not by a
	// coarse runner capability flag. A runner without native support still
	// receives the completion contract as a prompt suffix.
	if recovery == nil {
		applyCanary(resolvedConfig.PolicySnapshot, runID.String(), resolvedConfig.Model)
	}

	sandboxConfig, err := o.resolveSandboxConfig(req, profile)
	if err != nil {
		failCreation()
		return nil, err
	}
	if recovery != nil && resolvedConfig.SandboxConfig != nil {
		sandboxConfig = resolvedConfig.SandboxConfig
	}

	// Evaluate policies
	var policyDecision *policy.Decision
	if o.policy != nil {
		policyDecision, err = o.policy.EvaluateRunRequest(ctx, policy.EvaluateRequest{
			Task:          task,
			Profile:       profile,
			RequestedMode: valueOrDefault(req.RunMode, domain.RunModeSandboxed),
			ForceInPlace:  req.ForceInPlace,
		})
		if err != nil {
			failCreation()
			return nil, domain.NewInternalError("policy evaluation failed", err)
		}
		if !policyDecision.Allowed {
			failCreation()
			return nil, &domain.PolicyViolationError{
				PolicyID:   policyDecision.DenialPolicy.ID,
				PolicyName: policyDecision.DenialPolicy.Name,
				Rule:       "run_request",
				Message:    policyDecision.DenialReason,
			}
		}
	}

	// Determine run mode.
	//
	// SandboxConfig.Mode is the single source of truth; DeriveRunMode
	// translates the resolved Mode to a RunMode without consulting any
	// other input. See docs/internal/SEAMS.md (RunMode decision boundary)
	// and docs/internal/INVARIANTS.md.
	//
	// Decision priority (highest first):
	//   1. Explicit caller override via req.RunMode
	//   2. ForceInPlace (policy must permit; orchestrator validates that
	//      the resolved sandbox mode is at or above policy's required
	//      minimum below)
	//   3. Derived from sandboxConfig.Mode
	runMode := domain.DeriveRunMode(sandboxConfig)
	if req.RunMode != nil {
		runMode = *req.RunMode
	} else if req.ForceInPlace {
		runMode = domain.RunModeInPlace
	}

	// Resolve declaration-only spawn preferences against the selected runner's
	// published capabilities before the run becomes immutable.
	var spawnSkips []SpawnPreferenceSkip
	if recovery == nil && profile != nil && profile.SpawnPolicy != nil {
		selected, err := o.runners.Get(resolvedConfig.RunnerType)
		if err != nil {
			failCreation()
			return nil, domain.NewValidationError("spawnPolicy", "selected runner is unavailable: "+err.Error())
		}
		resolution, err := ResolveSpawnPolicy(profile.SpawnPolicy, selected.Capabilities())
		if err != nil {
			failCreation()
			return nil, domain.NewValidationError("spawnPolicy", err.Error())
		}
		req.ExecutionMode = domain.ExecutionMode(resolution.ExecutionMode)
		sandboxConfig.Mode = domain.SandboxMode(resolution.SandboxMode)
		runMode = domain.DeriveRunMode(sandboxConfig)
		spawnSkips = resolution.Skipped
		if resolution.Fallback != "" {
			// A declared capability was used because no preferred combination was
			// feasible. Surface it on the run's policy snapshot so run get and the
			// execution record show why a non-preferred substrate was used.
			if resolvedConfig.PolicySnapshot != nil {
				reason := "spawn_fallback:" + resolution.Fallback
				if resolvedConfig.PolicySnapshot.SelectionReason == "" {
					resolvedConfig.PolicySnapshot.SelectionReason = reason
				} else {
					resolvedConfig.PolicySnapshot.SelectionReason += ";" + reason
				}
			}
		}
	}

	// Revalidate after spawn preferences: selecting another mode must not
	// discard a runtime write policy inherited from the owner profile.
	if err := validateSandboxConfig(sandboxConfig); err != nil {
		failCreation()
		return nil, err
	}
	if err := validateExecutionContainment(req.ExecutionMode, sandboxConfig.Mode, resolvedConfig); err != nil {
		failCreation()
		return nil, err
	}

	// Enforce the policy-declared minimum sandbox mode. The policy layer
	// expresses sandbox requirements as a minimum SandboxMode rather
	// than a bool so a higher-strictness policy can require Protected
	// while still allowing Tracking-mode runs through other paths.
	if policyDecision != nil && policyDecision.RequiredSandboxMode != domain.SandboxModeUnspecified {
		resolvedMode := domain.SandboxModeOff
		if sandboxConfig != nil {
			resolvedMode = sandboxConfig.Mode.Effective()
		}
		if !resolvedMode.AtLeast(policyDecision.RequiredSandboxMode) {
			failCreation()
			return nil, domain.NewValidationErrorWithHint(
				"sandboxConfig.mode",
				"resolved sandbox mode is below the policy-required minimum",
				fmt.Sprintf("policy requires Mode >= %q; resolved Mode is %q",
					policyDecision.RequiredSandboxMode, resolvedMode),
			)
		}
	}

	if err := o.validateScopePath(task, runMode, req.ExistingSandboxID, false); err != nil {
		failCreation()
		return nil, err
	}

	existingSandboxWorkDir := ""
	var sandboxToStart *uuid.UUID
	if req.ExistingSandboxID != nil {
		if runMode != domain.RunModeSandboxed {
			failCreation()
			return nil, domain.NewValidationErrorWithHint("existingSandboxId", "existing sandbox requires sandboxed run mode",
				"set runMode to sandboxed or set sandboxConfig.mode to a sandbox-enabled value (tracking/protected)")
		}
		if o.sandbox == nil {
			failCreation()
			return nil, domain.NewConfigMissingError("sandbox", "provider not configured", nil)
		}

		sbx, err := o.sandbox.Get(ctx, *req.ExistingSandboxID)
		if err != nil {
			failCreation()
			return nil, err
		}
		switch sbx.Status {
		case sandbox.SandboxStatusDeleted, sandbox.SandboxStatusRejected, sandbox.SandboxStatusApproved, sandbox.SandboxStatusError:
			failCreation()
			return nil, domain.NewValidationErrorWithHint("existingSandboxId", "sandbox is not reusable",
				fmt.Sprintf("sandbox status is %s", sbx.Status))
		case sandbox.SandboxStatusStopped:
			sandboxToStart = &sbx.ID
		}

		if trimmed := strings.TrimSpace(task.ProjectRoot); trimmed != "" && strings.TrimSpace(sbx.ProjectRoot) != "" && trimmed != sbx.ProjectRoot {
			failCreation()
			return nil, domain.NewValidationErrorWithHint("existingSandboxId", "sandbox project root does not match task",
				fmt.Sprintf("task projectRoot=%q, sandbox projectRoot=%q", trimmed, sbx.ProjectRoot))
		}
		if trimmed := strings.TrimSpace(task.ScopePath); trimmed != "" && strings.TrimSpace(sbx.ScopePath) != "" && trimmed != sbx.ScopePath {
			failCreation()
			return nil, domain.NewValidationErrorWithHint("existingSandboxId", "sandbox scope path does not match task",
				fmt.Sprintf("task scopePath=%q, sandbox scopePath=%q", trimmed, sbx.ScopePath))
		}

		if sbx.WorkDir != "" {
			existingSandboxWorkDir = sbx.WorkDir
		} else {
			workDir, err := o.sandbox.GetWorkspacePath(ctx, sbx.ID)
			if err != nil {
				failCreation()
				return nil, err
			}
			existingSandboxWorkDir = workDir
		}
	}

	// Create the run with progress tracking initialized
	profileID := req.AgentProfileID
	if profile != nil {
		profileID = &profile.ID
	}
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
		return nil, fmt.Errorf("invalid workload kind %q", workload.Kind)
	}
	tag := strings.TrimSpace(req.Tag)
	if tag == "" && workload.Key != "" {
		tag = workload.Key
		if workload.Instance != "" {
			tag += "#" + workload.Instance
		}
	}
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
	resolvedConfig.Billing = billing
	// Split the prompt before persisting the immutable resolved config so the
	// known injected instruction estimate survives event pruning and replay.
	systemPrompt, userMessage := domain.BuildSplitPrompt(task.Description, task.ContextAttachments, req.Prompt)
	if strings.TrimSpace(systemPrompt) != "" {
		estimate := tokenaccounting.EstimateText(systemPrompt)
		resolvedConfig.PreambleInjectedTokens = estimate.Tokens
		resolvedConfig.PreambleTokenBasis = estimate.Basis
	}
	// Bind the caller's requested settings to the owner-resolved effective
	// settings before persistence so a fresh reader can tell requested from
	// effective without consulting mutable policy or a transcript.
	// Native controls depend on the final sandbox policy, including retained
	// recovery grants. Attach it before recording admission, not afterward.
	resolvedConfig.SandboxConfig = sandboxConfig
	narrowEffortConfig(req, resolvedConfig, o.now())
	resolvedConfig.Admission = buildRunAdmission(req, resolvedConfig)
	// Record the runner-native control arguments the selected codec emits for
	// the resolved configuration, so the "passed" layer is durable alongside
	// the requested and effective layers.
	o.recordPassedInvocation(ctx, resolvedConfig.Admission, resolvedConfig)
	// Dependent-delegation prerequisite gate: a run created as a child of an
	// admitted parent is delegated work. It may proceed only when the parent
	// carries a live qualification receipt whose effective identity matches this
	// run's resolved identity exactly, and retains its execution restrictions.
	// The qualification probe has no parent and is not subject to this gate.
	if err := o.admitDependentDelegation(ctx, req, resolvedConfig); err != nil {
		failCreation()
		return nil, err
	}
	if err := o.revalidateCreateRunCaller(ctx, req); err != nil {
		return nil, err
	}
	if err := o.reserveEffortAdmission(ctx, &req, task, resolvedConfig); err != nil {
		return nil, err
	}
	// Caller, owner narrowing, profile policy and dependent qualification have
	// all passed. Only now may admission reserve or mutate durable state.
	if req.IdempotencyKey != "" && o.idempotency != nil {
		// Reserve only after owner admission; accepted replays above remain reads.
		if _, err := o.idempotency.Reserve(ctx, req.IdempotencyKey, 1*time.Hour); err != nil {
			// If reservation fails, another request beat us to it
			return nil, domain.NewStateError("Run", "creating", "create",
				"a run creation with this idempotency key is already in progress")
		}
	}
	reserved = true
	if req.effort != nil {
		b := req.effort.binding
		i := req.effort.intent
		resolvedConfig.Admission.Effort = &b
		resolvedConfig.Admission.EffortIntent = &i
	}
	if err := o.preflightScopePath(task, runMode, req.ExistingSandboxID); err != nil {
		failCreation()
		return nil, err
	}
	if sandboxToStart != nil {
		if err := o.sandbox.Start(ctx, *sandboxToStart); err != nil {
			failCreation()
			return nil, err
		}
	}
	if taskRootChanged && o.tasks != nil {
		task.UpdatedAt = o.now()
		if err := o.tasks.Update(ctx, task); err != nil {
			failCreation()
			return nil, err
		}
	}
	run := &domain.Run{
		ID:                       runID,
		TaskID:                   task.ID,
		AgentProfileID:           profileID, // May be nil if inline config used
		Tag:                      tag,       // Custom tag for identification
		Label:                    strings.TrimSpace(task.Title),
		LabelSource:              domain.RunLabelSourceDerived,
		OwnerSubject:             req.OwnerSubject,
		OwnerScopes:              slices.Clone(req.OwnerScopes),
		RequestedScopes:          slices.Clone(req.RequestedScopes),
		OwnerExpiresAt:           req.OwnerExpiresAt,
		Workload:                 workload,
		Billing:                  billing,
		SourceRunIDs:             req.SourceRunIDs,
		WorkReferences:           req.WorkReferences,
		SourceInvestigationRunID: req.SourceInvestigationRunID,
		RunMode:                  runMode,
		ExecutionMode:            req.ExecutionMode,
		Status:                   domain.RunStatusPending,
		Phase:                    domain.RunPhaseQueued,
		ProgressPercent:          0,
		IdempotencyKey:           req.IdempotencyKey,
		ApprovalState:            domain.ApprovalStateNone,
		ResolvedConfig:           resolvedConfig,
		SandboxConfig:            sandboxConfig,
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

	if err := o.runs.Create(ctx, run); err != nil {
		failCreation()
		return nil, err
	}
	if err := o.acceptEffortRun(ctx, run); err != nil {
		return nil, domain.NewStateError("Run", "effort_acceptance_unresolved", "create", "run persisted but finite receipt unresolved; reconcile original run, no redispatch")
	}
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
		for _, skipped := range spawnSkips {
			message := fmt.Sprintf("spawn preference skipped: executionMode=%s sandboxMode=%s reason=%s", skipped.ExecutionMode, skipped.SandboxMode, skipped.Reason)
			if err := o.events.Append(ctx, run.ID, domain.NewLogEvent(run.ID, "info", message)); err != nil {
				obs.Component("orchestrator").Warn("failed to append spawn-preference event", obs.KeyRunID, run.ID.String(), "eventType", "log", obs.KeyError, err.Error())
			}
		}
	}

	// Mark idempotency as complete
	o.markIdempotencyComplete(ctx, req.IdempotencyKey, run.ID, "Run")

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

	// Emit the initial user prompt as the first message event.
	// We emit the user message (context + task), not the system prompt,
	// since the system prompt is runner-internal instructions.
	if o.events != nil && strings.TrimSpace(userMessage) != "" {
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
			userEvent = domain.NewMessageEventWithAttachments(run.ID, "user", userMessage, attInfo)
		} else {
			userEvent = domain.NewMessageEvent(run.ID, "user", userMessage)
		}
		if err := o.appendAndBroadcastEvents(ctx, run.ID, userEvent); err != nil {
			obs.Component("orchestrator").Warn("failed to append initial user message", obs.KeyRunID, run.ID.String(), "eventType", "message", obs.KeyError, err.Error())
		}
	}

	// Hand the executor body to the spawn dispatcher. Enqueue is the
	// only path through which a run begins — direct goroutine spawning
	// would skip startup serialization (codex SQLite WAL contention)
	// and queue-depth surfacing.
	// Snapshot before enqueue: attachRunActions projects the caller's record
	// immediately after enqueue, while the dispatcher may start execution at
	// once. The asynchronous executor must not copy that record concurrently.
	executionRun := *run
	if err := o.dispatcher.Enqueue(&spawn.Job{
		RunID:      run.ID,
		RunMode:    run.RunMode,
		RunnerType: runnerTypeOrEmpty(run),
		Sink:       o.dispatcherSink(run.ID),
		Fn: func(started spawn.StartedFn) {
			defer obs.RecoverToFailure("run execution dispatch", func(failure obs.PanicFailure) {
				o.recoverPanickedRun(run, failure)
			})
			executionRunCopy := executionRun
			o.executeRun(context.WithoutCancel(ctx), &executionRunCopy, task, profile, userMessage, systemPrompt, existingSandboxWorkDir, imageAttachments, req.Environment, started)
		},
		OnPanic: func(failure obs.PanicFailure) {
			o.recoverPanickedRun(run, failure)
		},
	}); err != nil {
		failCreation()
		return nil, err
	}

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
