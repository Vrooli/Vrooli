// This file creates runs and their initial persisted execution state.
package orchestration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/metrics"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/orchestration/phases"
	"agent-manager/internal/orchestration/spawn"
	"agent-manager/internal/policy"
	"agent-manager/internal/repository"
	"agent-manager/internal/structuredresult"

	"github.com/google/uuid"
)

func (o *Orchestrator) CreateRun(ctx context.Context, req CreateRunRequest) (*domain.Run, error) {
	// IDEMPOTENCY: Check if this request has already been processed
	if req.IdempotencyKey != "" && o.idempotency != nil {
		existing, err := o.idempotency.Check(ctx, req.IdempotencyKey)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			// Request already processed - return cached result
			if existing.Status == domain.IdempotencyStatusComplete && existing.EntityID != nil {
				return o.GetRun(ctx, *existing.EntityID)
			}
			if existing.Status == domain.IdempotencyStatusPending {
				// Another request is in progress with this key
				return nil, domain.NewStateError("Run", "creating", "create",
					"a run creation with this idempotency key is already in progress")
			}
			// Failed status - allow retry by falling through
		}

		// Reserve the idempotency key for this operation
		if _, err := o.idempotency.Reserve(ctx, req.IdempotencyKey, 1*time.Hour); err != nil {
			// If reservation fails, another request beat us to it
			return nil, domain.NewStateError("Run", "creating", "create",
				"a run creation with this idempotency key is already in progress")
		}
	}

	// SLOT ENFORCEMENT: Check capacity unless Force is set
	if !req.Force && o.config.MaxConcurrentRuns > 0 && o.runs != nil {
		// Count active runs (both Running and Starting count against the limit)
		runningCount, err := o.runs.CountByStatus(ctx, domain.RunStatusRunning)
		if err != nil {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
			return nil, err
		}
		startingCount, err := o.runs.CountByStatus(ctx, domain.RunStatusStarting)
		if err != nil {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
			return nil, err
		}

		activeCount := runningCount + startingCount
		if activeCount >= o.config.MaxConcurrentRuns {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
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
		o.markIdempotencyFailed(ctx, req.IdempotencyKey)
		return nil, err
	}

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
			if o.tasks != nil {
				task.UpdatedAt = o.now()
				_ = o.tasks.Update(ctx, task)
			}
		}
	}

	if req.AgentProfileID != nil && req.ProfileRef != nil {
		o.markIdempotencyFailed(ctx, req.IdempotencyKey)
		return nil, domain.NewValidationErrorWithHint("agentProfileId/profileRef", "only one profile reference is allowed",
			"provide either agentProfileId or profileRef")
	}

	// Resolve configuration: profile (if provided) + inline overrides
	resolvedConfig, profile, err := o.resolveRunConfig(ctx, req)
	if err != nil {
		o.markIdempotencyFailed(ctx, req.IdempotencyKey)
		return nil, err
	}

	sandboxConfig, err := o.resolveSandboxConfig(req, profile)
	if err != nil {
		o.markIdempotencyFailed(ctx, req.IdempotencyKey)
		return nil, err
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
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
			return nil, domain.NewInternalError("policy evaluation failed", err)
		}
		if !policyDecision.Allowed {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
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

	// Policy gate (locked decision 5): interactive execution mode is only
	// available for non-protected (in-place) runs. Reject at creation time with
	// an actionable error before the run is persisted or dispatched. The
	// executeInteractiveRun path re-checks this as a backstop.
	if err := domain.ValidateInteractiveRunMode(req.ExecutionMode, sandboxConfig.Mode); err != nil {
		o.markIdempotencyFailed(ctx, req.IdempotencyKey)
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
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
			return nil, domain.NewValidationErrorWithHint(
				"sandboxConfig.mode",
				"resolved sandbox mode is below the policy-required minimum",
				fmt.Sprintf("policy requires Mode >= %q; resolved Mode is %q",
					policyDecision.RequiredSandboxMode, resolvedMode),
			)
		}
	}

	if err := o.preflightScopePath(task, runMode, req.ExistingSandboxID); err != nil {
		o.markIdempotencyFailed(ctx, req.IdempotencyKey)
		return nil, err
	}

	existingSandboxWorkDir := ""
	if req.ExistingSandboxID != nil {
		if runMode != domain.RunModeSandboxed {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
			return nil, domain.NewValidationErrorWithHint("existingSandboxId", "existing sandbox requires sandboxed run mode",
				"set runMode to sandboxed or set sandboxConfig.mode to a sandbox-enabled value (tracking/protected)")
		}
		if o.sandbox == nil {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
			return nil, domain.NewConfigMissingError("sandbox", "provider not configured", nil)
		}

		sbx, err := o.sandbox.Get(ctx, *req.ExistingSandboxID)
		if err != nil {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
			return nil, err
		}
		switch sbx.Status {
		case sandbox.SandboxStatusDeleted, sandbox.SandboxStatusRejected, sandbox.SandboxStatusApproved, sandbox.SandboxStatusError:
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
			return nil, domain.NewValidationErrorWithHint("existingSandboxId", "sandbox is not reusable",
				fmt.Sprintf("sandbox status is %s", sbx.Status))
		case sandbox.SandboxStatusStopped:
			if err := o.sandbox.Start(ctx, sbx.ID); err != nil {
				o.markIdempotencyFailed(ctx, req.IdempotencyKey)
				return nil, err
			}
		}

		if trimmed := strings.TrimSpace(task.ProjectRoot); trimmed != "" && strings.TrimSpace(sbx.ProjectRoot) != "" && trimmed != sbx.ProjectRoot {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
			return nil, domain.NewValidationErrorWithHint("existingSandboxId", "sandbox project root does not match task",
				fmt.Sprintf("task projectRoot=%q, sandbox projectRoot=%q", trimmed, sbx.ProjectRoot))
		}
		if trimmed := strings.TrimSpace(task.ScopePath); trimmed != "" && strings.TrimSpace(sbx.ScopePath) != "" && trimmed != sbx.ScopePath {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
			return nil, domain.NewValidationErrorWithHint("existingSandboxId", "sandbox scope path does not match task",
				fmt.Sprintf("task scopePath=%q, sandbox scopePath=%q", trimmed, sbx.ScopePath))
		}

		if sbx.WorkDir != "" {
			existingSandboxWorkDir = sbx.WorkDir
		} else {
			workDir, err := o.sandbox.GetWorkspacePath(ctx, sbx.ID)
			if err != nil {
				o.markIdempotencyFailed(ctx, req.IdempotencyKey)
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
	run := &domain.Run{
		ID:                       uuid.New(),
		TaskID:                   task.ID,
		AgentProfileID:           profileID, // May be nil if inline config used
		Tag:                      req.Tag,   // Custom tag for identification
		SourceRunIDs:             req.SourceRunIDs,
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
		CreatedAt:      o.now(),
		UpdatedAt:      o.now(),
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
	if run.ResolvedConfig != nil {
		run.ResolvedConfig.SandboxConfig = sandboxConfig
	}
	if req.ExistingSandboxID != nil {
		run.SandboxID = req.ExistingSandboxID
	}

	if err := o.runs.Create(ctx, run); err != nil {
		o.markIdempotencyFailed(ctx, req.IdempotencyKey)
		return nil, err
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
	systemPrompt, userMessage := domain.BuildSplitPrompt(task.Description, task.ContextAttachments, req.Prompt)

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
	if err := o.dispatcher.Enqueue(&spawn.Job{
		RunID:      run.ID,
		RunMode:    run.RunMode,
		RunnerType: runnerTypeOrEmpty(run),
		Sink:       o.dispatcherSink(run.ID),
		Fn: func(started spawn.StartedFn) {
			defer obs.RecoverToFailure("run execution dispatch", func(failure obs.PanicFailure) {
				o.recoverPanickedRun(run, failure)
			})
			o.executeRun(context.WithoutCancel(ctx), run, task, profile, userMessage, systemPrompt, existingSandboxWorkDir, imageAttachments, req.Environment, started)
		},
		OnPanic: func(failure obs.PanicFailure) {
			o.recoverPanickedRun(run, failure)
		},
	}); err != nil {
		o.markIdempotencyFailed(ctx, req.IdempotencyKey)
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

// resolveRunConfig resolves the run configuration from profile and/or inline config.
// Returns the resolved config and the profile (if loaded, may be nil for pure inline config).
func (o *Orchestrator) resolveRunConfig(ctx context.Context, req CreateRunRequest) (*domain.RunConfig, *domain.AgentProfile, error) {
	cfg := domain.DefaultRunConfig()
	var profile *domain.AgentProfile

	// Load profile if provided
	if req.AgentProfileID != nil {
		var err error
		profile, err = o.GetProfile(ctx, *req.AgentProfileID)
		if err != nil {
			return nil, nil, err
		}
		cfg.ApplyProfile(profile)
	}

	// Resolve profile by key if provided
	if req.ProfileRef != nil {
		if req.ProfileRef.Defaults == nil && !req.ProfileRef.UpdateExisting {
			key := strings.TrimSpace(req.ProfileRef.ProfileKey)
			if key == "" {
				return nil, nil, domain.NewValidationErrorWithHint("profileRef.profileKey", "field is required",
					"Provide a stable profile key or inline profile defaults")
			}
			var err error
			profile, err = o.profiles.GetByKey(ctx, key)
			if err != nil {
				return nil, nil, err
			}
			if profile == nil {
				return nil, nil, domain.NewValidationErrorWithHint("profileRef.profileKey", "profile not found",
					"Start the owning scenario so agent-manager can reconcile its manifest-declared profiles")
			}
		} else {
			result, err := o.EnsureProfile(ctx, EnsureProfileRequest{
				ProfileKey:     req.ProfileRef.ProfileKey,
				Defaults:       req.ProfileRef.Defaults,
				UpdateExisting: req.ProfileRef.UpdateExisting,
			})
			if err != nil {
				return nil, nil, err
			}
			profile = result.Profile
		}
		if profile != nil {
			cfg.ApplyProfile(profile)
		}
	}

	// Apply inline overrides
	if req.RoleRef != nil {
		cfg.RoleRef = strings.TrimSpace(*req.RoleRef)
	}
	if req.MaxTurns != nil {
		cfg.MaxTurns = *req.MaxTurns
	}
	if req.Timeout != nil {
		cfg.Timeout = *req.Timeout
	}
	if req.Effort != nil {
		cfg.Effort = *req.Effort
	}
	if req.AllowedTools != nil {
		cfg.AllowedTools = req.AllowedTools
	}
	if req.DeniedTools != nil {
		cfg.DeniedTools = req.DeniedTools
	}
	if req.SkipPermissionPrompt != nil {
		cfg.SkipPermissionPrompt = *req.SkipPermissionPrompt
	}
	// Feature flag overrides
	if req.EnableBrowser != nil {
		cfg.Features.EnableBrowser = *req.EnableBrowser
	}
	// Extra flags overrides (replace per runner type)
	if req.ExtraFlags != nil {
		if cfg.ExtraFlags == nil {
			cfg.ExtraFlags = make(domain.RunnerExtraFlags)
		}
		for rt, flags := range req.ExtraFlags {
			cfg.ExtraFlags[rt] = append([]string(nil), flags...)
		}
	}
	if req.NetworkAccess != nil {
		cfg.NetworkAccess = *req.NetworkAccess
	}
	if req.AllowedPaths != nil {
		cfg.AllowedPaths = req.AllowedPaths
	}
	if req.DeniedPaths != nil {
		cfg.DeniedPaths = req.DeniedPaths
	}
	if req.ResultSpec != nil {
		normalized, err := structuredresult.NormalizeSpec(req.ResultSpec)
		if err != nil {
			return nil, nil, domain.NewValidationErrorWithHint("resultSpec", err.Error(),
				"Use result-spec/v1 with the documented bounded JSON Schema subset")
		}
		cfg.ResultSpec = normalized
	}
	// Validate the resolved config
	if strings.TrimSpace(cfg.RoleRef) == "" {
		return nil, nil, domain.NewValidationErrorWithHint("roleRef", "field is required",
			"Select a portable role from the active role-policy catalog")
	}
	if !cfg.Effort.IsValid() {
		return nil, nil, domain.NewValidationErrorWithHint("effort", "invalid effort", "valid values: low, medium, high, xhigh, max")
	}
	if err := domain.ValidateCanonicalToolList("allowedTools", cfg.AllowedTools); err != nil {
		return nil, nil, err
	}
	if err := domain.ValidateCanonicalToolList("deniedTools", cfg.DeniedTools); err != nil {
		return nil, nil, err
	}
	if err := o.resolveExecutionPolicy(ctx, cfg); err != nil {
		return nil, nil, err
	}
	if err := o.applyModelOverride(ctx, cfg, req.Model); err != nil {
		return nil, nil, err
	}
	if err := o.validateToolRestriction(cfg); err != nil {
		return nil, nil, err
	}

	// Validate extra flags against runner allowlists (delegate to seam)
	if o.flagValidator != nil {
		for rt, flags := range cfg.ExtraFlags {
			if err := o.flagValidator.ValidateFlags(rt, flags); err != nil {
				return nil, nil, err
			}
		}
	}

	return cfg, profile, nil
}

// applyModelOverride applies an explicit per-run model only after role policy
// resolution. The immutable snapshot remains the execution authority, so its
// selected candidate must be updated alongside the resolved config; otherwise
// a later execution attempt could silently revert the caller's override.
func (o *Orchestrator) applyModelOverride(ctx context.Context, cfg *domain.RunConfig, requested *string) error {
	if requested == nil {
		return nil
	}
	model := strings.TrimSpace(*requested)
	if model == "" {
		return domain.NewValidationError("model", "must not be empty when supplied")
	}
	if cfg == nil || cfg.PolicySnapshot == nil {
		return domain.NewValidationError("model", "cannot override model without a resolved execution policy")
	}
	if o.runners != nil {
		runner, err := o.runners.Get(cfg.RunnerType)
		if err != nil {
			return err
		}
		if runner == nil {
			return domain.NewValidationError("model", "selected runner is not registered")
		}
		if err := runner.ProbeModel(ctx, model); err != nil {
			return domain.NewValidationErrorWithHint("model", "model is not available for the selected runner", err.Error())
		}
	}
	cfg.Model = model
	snapshot := cfg.PolicySnapshot
	if snapshot.SelectedIndex < 0 || snapshot.SelectedIndex >= len(snapshot.Candidates) {
		return domain.NewValidationError("model", "resolved execution policy has an invalid selected candidate")
	}
	snapshot.Candidates[snapshot.SelectedIndex].SelectionType = domain.ModelSelectionTypeModel
	snapshot.Candidates[snapshot.SelectedIndex].Model = model
	snapshot.SelectedCandidate = snapshot.Candidates[snapshot.SelectedIndex]
	return nil
}

// validateToolRestriction makes an allowlist fail closed once policy routing
// has selected the actual runner. Advisory is explicit and intentionally does
// not pretend that an unsupported runner enforces the declaration.
func (o *Orchestrator) validateToolRestriction(cfg *domain.RunConfig) error {
	if cfg == nil || (len(cfg.AllowedTools) == 0 && len(cfg.DeniedTools) == 0) || o.runners == nil {
		return nil
	}
	selected, err := o.runners.Get(cfg.RunnerType)
	if err != nil {
		return err
	}
	if selected.Capabilities().SupportsToolRestriction || cfg.ToolRestrictionPolicy.Effective() == domain.ToolRestrictionPolicyAdvisory {
		return nil
	}
	return domain.NewValidationErrorWithCode("toolRestrictionPolicy",
		fmt.Sprintf("runner %q cannot enforce allowedTools or deniedTools", cfg.RunnerType), domain.ErrCodePolicyRunner)
}

func (o *Orchestrator) toolRestrictionIsAdvisory(cfg *domain.RunConfig) bool {
	if cfg == nil || (len(cfg.AllowedTools) == 0 && len(cfg.DeniedTools) == 0) || cfg.ToolRestrictionPolicy.Effective() != domain.ToolRestrictionPolicyAdvisory || o.runners == nil {
		return false
	}
	selected, err := o.runners.Get(cfg.RunnerType)
	return err == nil && selected != nil && !selected.Capabilities().SupportsToolRestriction
}

// resolveExecutionPolicy converts the final profile-plus-override selection
// into a run-owned immutable snapshot. A named policy is resolved once; no
// runtime decision reads mutable catalog state after this function returns.
func (o *Orchestrator) resolveExecutionPolicy(ctx context.Context, cfg *domain.RunConfig) error {
	if cfg == nil {
		return domain.NewValidationError("runConfig", "field is required")
	}
	if strings.TrimSpace(cfg.RoleRef) != "" {
		if o.rolePolicy == nil || o.roleResolver == nil {
			return domain.NewValidationError("rolePolicyCatalog", "role policy state or resource resolver is not configured")
		}
		resolution, err := o.rolePolicy.Resolve(ctx, o.roleResolver, cfg.RoleRef)
		if err != nil {
			return err
		}
		snapshot := resolution.Snapshot()
		if snapshot == nil || len(snapshot.Candidates) == 0 {
			return domain.NewValidationError("rolePolicyCatalog", "role resolution produced no candidates")
		}
		selectedIndex, preflight, err := o.selectInitialCandidate(ctx, snapshot.Candidates)
		if err != nil {
			return err
		}
		snapshot.SelectedIndex = selectedIndex
		snapshot.SelectedCandidate = snapshot.Candidates[selectedIndex]
		snapshot.Explanation.Preflight = preflight
		snapshot.Explanation.Summary = fmt.Sprintf(
			"%s; selected candidate %d (%s/%s)",
			snapshot.Explanation.Summary,
			selectedIndex,
			snapshot.SelectedCandidate.RunnerType,
			snapshot.SelectedCandidate.Model,
		)
		cfg.PolicySnapshot = snapshot
		cfg.RunnerType = snapshot.SelectedCandidate.RunnerType
		cfg.Model = snapshot.SelectedCandidate.Model
		return nil
	}
	return domain.NewValidationError("roleRef", "field is required")
}

func (o *Orchestrator) selectInitialCandidate(ctx context.Context, candidates []domain.ExecutionCandidate) (int, []domain.CandidatePreflight, error) {
	if len(candidates) == 0 {
		return -1, nil, domain.NewValidationError("rolePolicyCatalog", "resolution produced no candidates")
	}
	if o.runners == nil {
		// Minimal unit orchestrators omit adapters. Production always injects
		// the registry before accepting traffic.
		return 0, nil, nil
	}

	checks := make([]domain.CandidatePreflight, 0, len(candidates))
	for index, candidate := range candidates {
		check := domain.CandidatePreflight{Index: index, Candidate: candidate}
		// Availability is resource-resolution evidence for portable roles.
		// Legacy snapshots predate that field, so their zero value must not
		// make every historical/direct candidate unavailable.
		if candidate.ResourceRole != "" && !candidate.Available {
			check.Reason = candidate.Failure
			if check.Reason == "" {
				check.Reason = candidate.FailureCode
			}
			if check.Reason == "" {
				check.Reason = "resource role is unavailable"
			}
			checks = append(checks, check)
			continue
		}
		resolvedRunner, err := o.runners.Get(candidate.RunnerType)
		if err != nil || resolvedRunner == nil {
			check.Reason = "runner is not registered"
			checks = append(checks, check)
			continue
		}
		available, message := resolvedRunner.IsAvailable(ctx)
		if !available {
			check.Reason = strings.TrimSpace(message)
			if check.Reason == "" {
				check.Reason = "runner is unavailable"
			}
			checks = append(checks, check)
			continue
		}
		switch candidate.SelectionType {
		case domain.ModelSelectionTypeModel:
			if err := resolvedRunner.ProbeModel(ctx, candidate.Model); err != nil {
				check.Reason = err.Error()
				checks = append(checks, check)
				continue
			}
		case domain.ModelSelectionTypeRunnerDefault:
			// Catalog/codec conformance already proves runner-default support.
		default:
			check.Reason = "candidate selection type is invalid"
			checks = append(checks, check)
			continue
		}
		check.Available = true
		checks = append(checks, check)
		return index, checks, nil
	}

	reasons := make([]string, 0, len(checks))
	for _, check := range checks {
		reasons = append(reasons, fmt.Sprintf("candidate %d %s/%s: %s", check.Index, check.Candidate.RunnerType, check.Candidate.SelectionType, check.Reason))
	}
	return -1, checks, domain.NewValidationErrorWithHint(
		"rolePolicyCatalog",
		"no policy candidate passed runner/model preflight",
		strings.Join(reasons, "; "),
	)
}

// resolveSandboxConfig produces the effective SandboxConfig for a run.
//
// Contract: the returned config is always non-nil. Callers (including
// tryAutoApproval) rely on this invariant; a nil return historically caused
// silent fall-through to NEEDS_REVIEW for empty sandboxes because there was
// no acceptance config to consult.
//
// Precedence (later overrides earlier):
//  1. Zero-valued default
//  2. profile.SandboxConfig (if present)
//  3. req.SandboxConfig (inline override, if present)
//
// Phase G: Profile/req AllowedPaths/DeniedPaths are merged into the resolved
// SandboxConfig.Acceptance so they become *enforced* at apply-at-run-end
// rather than passed as advisory env vars to runners. This is the
// agent-sandbox-audit-foundation policy-to-sandbox handoff.
func (o *Orchestrator) resolveSandboxConfig(req CreateRunRequest, profile *domain.AgentProfile) (*domain.SandboxConfig, error) {
	// Start from the auditability-contract defaults (Mode=Protected,
	// AutoApply=true, ApplyOnFailure=true, NetworkMode=localhost,
	// NoLock=true). Profile and request overrides clone over the top.
	// Without this baseline, a request with no profile and no inline
	// config would zero-value the struct, dropping Mode to unspecified
	// and silently downgrading to host-tracked execution.
	defaults := domain.DefaultSandboxConfig()
	cfg := defaults
	if profile != nil && profile.SandboxConfig != nil {
		cfg = cloneSandboxConfig(profile.SandboxConfig)
	}
	if req.SandboxConfig != nil {
		cfg = mergeSandboxConfig(cfg, req.SandboxConfig)
	}

	// Backfill enum/string fields that the override left at the proto
	// zero-value. Callers (notably swarm-manager) often send a partial
	// SandboxConfig containing only Acceptance overrides; without this
	// backfill the wholesale-replace clone above would silently strip
	// Mode and NetworkMode to "unspecified", silently downgrading
	// protected runs to tracking. Pointer-typed fields (AutoApply,
	// ApplyOnFailure) and structural fields (Lifecycle, Acceptance) are
	// left intentional-explicit; bool fields (ManualReview, NoLock) are
	// left at the override's value because zero is operator-visible
	// "off" rather than "not provided".
	if cfg.Mode == domain.SandboxModeUnspecified {
		cfg.Mode = defaults.Mode
	}
	if cfg.NetworkMode == "" {
		cfg.NetworkMode = defaults.NetworkMode
	}

	// Push path policy from profile/request into the acceptance layer so
	// workspace-sandbox actually enforces it at apply time. The runner-side
	// advisory env vars are kept for the tracking-mode capability matrix
	// but the load-bearing enforcement now lives at the sandbox boundary.
	allowedPaths := profilePaths(profile, func(p *domain.AgentProfile) []string { return p.AllowedPaths })
	if req.AllowedPaths != nil {
		allowedPaths = req.AllowedPaths
	}
	deniedPaths := profilePaths(profile, func(p *domain.AgentProfile) []string { return p.DeniedPaths })
	if req.DeniedPaths != nil {
		deniedPaths = req.DeniedPaths
	}
	cfg.Acceptance.Allow.PathGlobs = mergeUnique(cfg.Acceptance.Allow.PathGlobs, allowedPaths)
	cfg.Acceptance.Deny.PathGlobs = mergeUnique(cfg.Acceptance.Deny.PathGlobs, deniedPaths)

	cfg = normalizeSandboxConfig(cfg)
	if err := validateSandboxConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// mergeSandboxConfig applies an inline config as a sparse override instead of
// replacing the profile config wholesale. Proto scalar zero values cannot
// distinguish absence from an explicit false, so only non-zero scalar values
// and explicitly-present pointer/structural values override here. This keeps a
// ManualReview-only request from accidentally deleting its profile's lifecycle
// and acceptance contract.
func mergeSandboxConfig(base, override *domain.SandboxConfig) *domain.SandboxConfig {
	if base == nil {
		base = domain.DefaultSandboxConfig()
	}
	merged := cloneSandboxConfig(base)
	if override == nil {
		return merged
	}
	if override.Mode != domain.SandboxModeUnspecified {
		merged.Mode = override.Mode
	}
	if override.NetworkMode != "" {
		merged.NetworkMode = override.NetworkMode
	}
	// A populated config carries an explicit ManualReview=false; a sparse
	// ManualReview-only inline message cannot represent false in proto3 and
	// therefore leaves the profile value intact.
	if override.ManualReview || sandboxConfigHasExplicitScalars(override) {
		merged.ManualReview = override.ManualReview
	}
	if override.AutoApply != nil {
		v := *override.AutoApply
		merged.AutoApply = &v
	}
	if override.ApplyOnFailure != nil {
		v := *override.ApplyOnFailure
		merged.ApplyOnFailure = &v
	}
	if override.NoLock {
		merged.NoLock = true
	}
	if !sandboxLifecycleIsZero(override.Lifecycle) {
		merged.Lifecycle = cloneSandboxConfig(override).Lifecycle
	}
	if !sandboxAcceptanceIsZero(override.Acceptance) {
		merged.Acceptance = cloneSandboxConfig(override).Acceptance
	}
	return merged
}

func sandboxConfigHasExplicitScalars(cfg *domain.SandboxConfig) bool {
	return cfg.Mode != domain.SandboxModeUnspecified || cfg.NetworkMode != "" || cfg.AutoApply != nil || cfg.ApplyOnFailure != nil || cfg.NoLock
}

func sandboxLifecycleIsZero(lifecycle domain.SandboxLifecycleConfig) bool {
	return len(lifecycle.CheckpointOn) == 0 && len(lifecycle.StopOn) == 0 && len(lifecycle.DeleteOn) == 0 && lifecycle.TTL == 0 && lifecycle.IdleTimeout == 0
}

func sandboxAcceptanceIsZero(acceptance domain.SandboxAcceptanceConfig) bool {
	return acceptance.Mode == "" && !acceptance.IgnoreBinary && len(acceptance.Allow.PathGlobs) == 0 && len(acceptance.Allow.Extensions) == 0 && len(acceptance.Deny.PathGlobs) == 0 && len(acceptance.Deny.Extensions) == 0
}

func profilePaths(p *domain.AgentProfile, get func(*domain.AgentProfile) []string) []string {
	if p == nil {
		return nil
	}
	return get(p)
}

func mergeUnique(a, b []string) []string {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, v := range a {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	for _, v := range b {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func cloneSandboxConfig(cfg *domain.SandboxConfig) *domain.SandboxConfig {
	if cfg == nil {
		return nil
	}
	clone := *cfg
	clone.Lifecycle.CheckpointOn = append([]domain.SandboxLifecycleEvent(nil), cfg.Lifecycle.CheckpointOn...)
	clone.Lifecycle.StopOn = append([]domain.SandboxLifecycleEvent(nil), cfg.Lifecycle.StopOn...)
	clone.Lifecycle.DeleteOn = append([]domain.SandboxLifecycleEvent(nil), cfg.Lifecycle.DeleteOn...)
	clone.Acceptance.Allow = cloneSandboxCriteria(cfg.Acceptance.Allow)
	clone.Acceptance.Deny = cloneSandboxCriteria(cfg.Acceptance.Deny)
	if cfg.AutoApply != nil {
		v := *cfg.AutoApply
		clone.AutoApply = &v
	}
	if cfg.ApplyOnFailure != nil {
		v := *cfg.ApplyOnFailure
		clone.ApplyOnFailure = &v
	}
	return &clone
}

func cloneSandboxCriteria(criteria domain.SandboxFileCriteria) domain.SandboxFileCriteria {
	return domain.SandboxFileCriteria{
		PathGlobs:  append([]string(nil), criteria.PathGlobs...),
		Extensions: append([]string(nil), criteria.Extensions...),
	}
}

func normalizeSandboxConfig(cfg *domain.SandboxConfig) *domain.SandboxConfig {
	if cfg == nil {
		return nil
	}
	if cfg.Acceptance.Mode == "" {
		cfg.Acceptance.Mode = "allowlist"
	}
	cfg.Acceptance.Allow = normalizeSandboxCriteria(cfg.Acceptance.Allow)
	cfg.Acceptance.Deny = normalizeSandboxCriteria(cfg.Acceptance.Deny)

	// Default lifecycle cleanup for auto-apply sandboxes.
	//
	// Under the auditability contract (Phase 3b), AutoApply=true (the
	// contract default unless ManualReview=true) means the sandbox is
	// applied at run end. Once applied, leaving the sandbox active
	// indefinitely blocks future runs on the same scope path and leaks
	// overlay mounts. We default deleteOn to ["terminal"] so the sandbox
	// is cleaned up after any terminal event when ManualReview is off.
	//
	// ManualReview=true sandboxes intentionally persist past run end so
	// operators can review; their TTL GC is owned by workspace-sandbox
	// LifecycleReconciler (Phase 4).
	if cfg.GetAutoApply() && !cfg.ManualReview &&
		len(cfg.Lifecycle.CheckpointOn) == 0 && len(cfg.Lifecycle.DeleteOn) == 0 && len(cfg.Lifecycle.StopOn) == 0 {
		cfg.Lifecycle.CheckpointOn = []domain.SandboxLifecycleEvent{
			domain.SandboxLifecycleTurnCompleted,
			domain.SandboxLifecycleTurnFailed,
			domain.SandboxLifecycleTurnCancelled,
		}
		// Use the terminal event emitted by finalize so default sandboxes are
		// deleted instead of remaining checkpointed.
		cfg.Lifecycle.DeleteOn = []domain.SandboxLifecycleEvent{domain.SandboxLifecycleTerminal}
	}

	return cfg
}

func normalizeSandboxCriteria(criteria domain.SandboxFileCriteria) domain.SandboxFileCriteria {
	paths := make([]string, 0, len(criteria.PathGlobs))
	seenPaths := make(map[string]bool)
	for _, p := range criteria.PathGlobs {
		p = strings.TrimSpace(p)
		if p == "" || seenPaths[p] {
			continue
		}
		seenPaths[p] = true
		paths = append(paths, p)
	}

	exts := make([]string, 0, len(criteria.Extensions))
	seenExts := make(map[string]bool)
	for _, ext := range criteria.Extensions {
		ext = strings.TrimSpace(ext)
		if ext == "" {
			continue
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		ext = strings.ToLower(ext)
		if seenExts[ext] {
			continue
		}
		seenExts[ext] = true
		exts = append(exts, ext)
	}

	criteria.PathGlobs = paths
	criteria.Extensions = exts
	return criteria
}

func validateSandboxConfig(cfg *domain.SandboxConfig) error {
	if cfg == nil {
		return nil
	}
	if cfg.Acceptance.Mode != "" && cfg.Acceptance.Mode != "allowlist" {
		return domain.NewValidationError("sandboxConfig.acceptance.mode", "unsupported acceptance mode")
	}
	if cfg.Lifecycle.TTL < 0 {
		return domain.NewValidationError("sandboxConfig.lifecycle.ttl", "ttl cannot be negative")
	}
	if cfg.Lifecycle.IdleTimeout < 0 {
		return domain.NewValidationError("sandboxConfig.lifecycle.idleTimeout", "idleTimeout cannot be negative")
	}
	for _, p := range append(cfg.Acceptance.Allow.PathGlobs, cfg.Acceptance.Deny.PathGlobs...) {
		if filepath.IsAbs(p) || strings.HasPrefix(p, "/") {
			return domain.NewValidationErrorWithHint(
				"sandboxConfig.acceptance.pathGlobs",
				"path globs must be project-root relative",
				"Remove the leading '/' and use project-root relative patterns",
			)
		}
	}
	// Warn when AutoApply is on (the contract default) but no allow
	// criteria are configured. This is valid (empty allow = accept all
	// non-denied files), but surprising enough to warrant a log line —
	// especially since an empty deny (from proto serialization)
	// previously caused silent universal denial.
	if cfg.GetAutoApply() && !cfg.ManualReview &&
		len(cfg.Acceptance.Allow.PathGlobs) == 0 &&
		len(cfg.Acceptance.Allow.Extensions) == 0 {
		obs.Component("sandbox-config").Info("autoApply enabled with no allow criteria; all non-denied files will be applied at run end")
	}
	return nil
}

func (o *Orchestrator) GetRun(ctx context.Context, id uuid.UUID) (*domain.Run, error) {
	run, err := o.runs.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, domain.NewNotFoundError("Run", id)
	}
	return o.attachRunActions(ctx, run), nil
}

// GetRunByImportProvenance resolves an imported external session without
// exposing its filesystem path. Corpus import uses this before parsing so a
// repeated command is read-only for already adopted evidence.
func (o *Orchestrator) GetRunByImportProvenance(ctx context.Context, sourceHarness, sourceSessionID string) (*domain.Run, error) {
	return o.runs.GetByImportProvenance(ctx, sourceHarness, sourceSessionID)
}

func (o *Orchestrator) ListRuns(ctx context.Context, opts RunListOptions) ([]*domain.Run, error) {
	runs, err := o.runs.List(ctx, repository.RunListFilter{
		ListFilter: repository.ListFilter{
			Limit:  opts.Limit,
			Offset: opts.Offset,
		},
		TaskID:                    opts.TaskID,
		AgentProfileID:            opts.AgentProfileID,
		Status:                    opts.Status,
		TagPrefix:                 opts.TagPrefix,
		ScopePrefix:               opts.ScopePrefix,
		InvestigatesRunID:         opts.InvestigatesRunID,
		AppliesInvestigationRunID: opts.AppliesInvestigationRunID,
	})
	if err != nil {
		return nil, err
	}
	return o.attachRunActionsList(ctx, runs), nil
}

// ListParkedRuns returns all parked runs with their await-handle populated. The
// pruned list-columns omit the heavy await_handle field, so each parked run is
// reloaded by ID to recover its handle. Used by the await-handle registry's
// restart recovery (re-spawning waiters on boot).
func (o *Orchestrator) ListParkedRuns(ctx context.Context) ([]*domain.Run, error) {
	parked := domain.RunStatusParked
	rows, err := o.runs.List(ctx, repository.RunListFilter{Status: &parked})
	if err != nil {
		return nil, err
	}
	full := make([]*domain.Run, 0, len(rows))
	for _, row := range rows {
		loaded, err := o.runs.Get(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		if loaded == nil {
			continue
		}
		full = append(full, loaded)
	}
	return full, nil
}

func (o *Orchestrator) DeleteRun(ctx context.Context, id uuid.UUID) error {
	run, err := o.GetRun(ctx, id)
	if err != nil {
		return err
	}
	if allowed, reason := domain.CanDeleteRun(run); !allowed {
		return domain.NewStateError("Run", string(run.Status), "delete", reason)
	}
	return o.runs.Delete(ctx, id)
}

// GetRunByTag retrieves a run by its custom tag.
// Returns NotFoundError if no run with that tag exists.
func (o *Orchestrator) GetRunByTag(ctx context.Context, tag string) (*domain.Run, error) {
	// List all runs with matching tag prefix and find exact match
	runs, err := o.runs.List(ctx, repository.RunListFilter{
		TagPrefix: tag,
	})
	if err != nil {
		return nil, err
	}

	// Find exact match
	for _, run := range runs {
		if run.GetTag() == tag {
			return o.attachRunActions(ctx, run), nil
		}
	}

	return nil, domain.NewNotFoundError("Run", uuid.Nil)
}

// StopRunByTag stops a run identified by its custom tag.
func (o *Orchestrator) StopRunByTag(ctx context.Context, tag string) error {
	run, err := o.GetRunByTag(ctx, tag)
	if err != nil {
		return err
	}
	return o.StopRun(ctx, run.ID)
}

// StopAllRuns stops all running runs, optionally filtered by tag prefix.
func (o *Orchestrator) StopAllRuns(ctx context.Context, opts StopAllOptions) (*StopAllResult, error) {
	result := &StopAllResult{
		FailedIDs: []string{},
	}

	// Get all running or starting runs
	runningStatus := domain.RunStatusRunning
	runs, err := o.runs.List(ctx, repository.RunListFilter{
		Status:    &runningStatus,
		TagPrefix: opts.TagPrefix,
	})
	if err != nil {
		return nil, err
	}

	// Also get starting runs
	startingStatus := domain.RunStatusStarting
	startingRuns, err := o.runs.List(ctx, repository.RunListFilter{
		Status:    &startingStatus,
		TagPrefix: opts.TagPrefix,
	})
	if err != nil {
		return nil, err
	}
	runs = append(runs, startingRuns...)

	// Stop each run
	for _, run := range runs {
		// Skip already stopped runs
		if run.Status == domain.RunStatusComplete ||
			run.Status == domain.RunStatusFailed ||
			run.Status == domain.RunStatusCancelled {
			result.Skipped++
			continue
		}

		if err := o.StopRun(ctx, run.ID); err != nil {
			result.Failed++
			result.FailedIDs = append(result.FailedIDs, run.ID.String())
		} else {
			result.Stopped++
		}
	}

	return result, nil
}

func (o *Orchestrator) StopRun(ctx context.Context, id uuid.UUID) error {
	run, err := o.GetRun(ctx, id)
	if err != nil {
		return err
	}
	if run.ExecutionMode.Normalized() == domain.ExecutionModeImported {
		return importedRunLifecycleError("stop")
	}

	if allowed, reason := domain.CanStopRun(run); !allowed {
		return domain.NewStateError("Run", string(run.Status), "stop", reason)
	}

	// Interactive runs have no local process to signal — the CLI lives in a
	// web-console tmux session. Stop them via the interrupt-then-delete
	// escalation ladder and finalize deterministically (Cancelled), instead of
	// the pgid/terminator path below.
	if run.ExecutionMode.Normalized() == domain.ExecutionModeInteractive {
		return o.stopInteractiveRun(ctx, run)
	}

	// A parked run has no live process to terminate — stopping it cancels the
	// await (clears the handle) and moves the run to cancelled. The waiter that
	// owns the handle observes the terminal status and deregisters (Phase 3).
	if run.Status == domain.RunStatusParked {
		// Cancel the background watcher first so it observes the cancellation and
		// exits without waking the now-cancelled run.
		if o.awaitRegistry != nil {
			o.awaitRegistry.Cancel(id)
		}
		run.AwaitHandle = nil
		endedAt := o.now()
		_, err = o.applyRunStatusTransition(ctx, RunStatusTransitionInput{
			Run:       run,
			NewStatus: domain.RunStatusCancelled,
			Phase:     domain.RunPhaseCompleted,
			Reason:    "Parked run stopped by request",
			EndedAt:   &endedAt,
		})
		return err
	}

	if o.terminator != nil {
		result, err := o.terminator.Terminate(ctx, id)
		if err != nil {
			return err
		}
		if !result.Success {
			return result.Error
		}

		endedAt := o.now()
		_, err = o.applyRunStatusTransition(ctx, RunStatusTransitionInput{
			Run:       run,
			NewStatus: domain.RunStatusCancelled,
			Phase:     domain.RunPhaseCompleted,
			Reason:    "Run stopped by request",
			EndedAt:   &endedAt,
		})
		return err
	}

	// The immutable resolved config is the sole execution authority.
	var runnerType domain.RunnerType
	if run.ResolvedConfig != nil {
		runnerType = run.ResolvedConfig.RunnerType
	}

	// Stop execution if we have a runner type
	if o.runners != nil && runnerType != "" {
		if r, err := o.runners.Get(runnerType); err == nil {
			if err := r.Stop(ctx, run.ID); err != nil {
				return err
			}
		}
	}

	endedAt := o.now()
	_, err = o.applyRunStatusTransition(ctx, RunStatusTransitionInput{
		Run:       run,
		NewStatus: domain.RunStatusCancelled,
		Phase:     domain.RunPhaseCompleted,
		Reason:    "Run stopped by request",
		EndedAt:   &endedAt,
	})
	return err
}

func (o *Orchestrator) RecoverRun(ctx context.Context, id uuid.UUID) (*RecoverResult, error) {
	run, err := o.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if run.ExecutionMode.Normalized() == domain.ExecutionModeImported {
		return nil, importedRunLifecycleError("recover")
	}
	if o.reconciler == nil {
		return nil, domain.NewConfigMissingError("reconciler", "reconciler not configured", nil)
	}
	return o.reconciler.RecoverRun(ctx, id)
}

// ContinueRun continues an existing run's conversation with a follow-up message.
// The message is appended to the run's event stream and the response is streamed back.
