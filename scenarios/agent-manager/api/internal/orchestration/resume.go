// Package orchestration: ResumeFromFailedRun creates a brand-new run that
// inherits the original task + profile of a failed/cancelled run and is
// seeded with that run's transcript and diff. This is the missing middle
// ground between Retry (fresh start, no context) and Continue (Codex session
// resume, fragile and only when SessionID is present): it lets the agent pick
// up where the prior attempt left off without redoing completed work.
package orchestration

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/maintenance"
	"agent-manager/internal/repository"
	"agent-manager/internal/runreport"

	"github.com/google/uuid"
)

// ResumeFromFailedRun creates a new run that resumes the work of a failed
// or cancelled run.
func (o *Orchestrator) ResumeFromFailedRun(
	ctx context.Context,
	req ResumeFromFailedRunRequest,
) (*domain.Run, error) {
	o.wakeMu.Lock()
	defer o.wakeMu.Unlock()
	return o.resumeFromFailedRun(ctx, req, false)
}

func (o *Orchestrator) resumeFromFailedRun(ctx context.Context, req ResumeFromFailedRunRequest, automatic bool) (_ *domain.Run, returnErr error) {
	// Until the source and its durable claim can be read, this may be a replay
	// of already accepted work. Read failures must not become REFUSED.
	effectsPossible := true
	defer func() {
		if !effectsPossible {
			returnErr = domain.RefuseBeforeEffects(returnErr)
		}
	}()
	failedRun, err := o.GetRun(ctx, req.RunID)
	if err != nil {
		return nil, err
	}
	effectsPossible = false
	if failedRun.ExecutionMode.Normalized() == domain.ExecutionModeImported {
		return nil, importedRunLifecycleError("resume-from-failed")
	}
	if allowed, reason := domain.CanResumeFromFailureRun(failedRun); !allowed {
		return nil, domain.NewValidationError("runId", reason)
	}
	if automatic && (failedRun.Status != domain.RunStatusFailed || failedRun.CancelRequestedAt != nil || failedRun.ExecutionMode.Normalized() == domain.ExecutionModeInteractive || strings.TrimSpace(failedRun.SessionID) == "" || failedRun.ResolvedConfig == nil || failedRun.ResolvedConfig.RunnerType != domain.RunnerTypeOpenCode) {
		return nil, domain.NewValidationError("runId", "automatic fresh recovery requires a failed managed OpenCode run with a retained session and no cancellation")
	}
	// Look up acceptance before preflight, policy changes, maintenance or any
	// effects. A source claim survives cache expiry and replacement deletion.
	effectsPossible = true
	replayed, claimed, err := o.freshRecoveryAccepted(ctx, req)
	if err != nil {
		return nil, err
	}
	if replayed != nil {
		return replayed, nil
	}
	if claimed {
		return nil, domain.NewStateError("Run", "recovery_unresolved", "fresh recovery", "source was already claimed; replacement acceptance is missing or was deleted; owner reconciliation required, no redispatch")
	}
	effectsPossible = false
	// Fresh identity cannot shed the source's revocable dispatcher grant.
	// Until this owner route can enroll the replacement under that grant,
	// refuse new work rather than turning retained scopes into authority.
	if failedRun.DispatchBinding != nil {
		return nil, domain.NewValidationErrorWithCode("authorization", "fresh recovery of a dispatcher-bound source is unsupported; qualified dispatch enrollment is required", domain.ErrCodePolicyScope)
	}
	if failedRun.OwnerSubject != "" && (failedRun.OwnerExpiresAt == nil || !failedRun.OwnerExpiresAt.After(o.now())) {
		return nil, domain.NewValidationErrorWithCode("authorization", "fresh recovery requires current owner authority; retained authority is missing or expired", domain.ErrCodePolicyScope)
	}
	if automatic {
		if err := excludeLiveExecutor(ctx, failedRun); err != nil {
			return nil, err
		}
		err = o.validateContinuationSession(ctx, failedRun)
		var runnerErr *domain.RunnerError
		if !errors.As(err, &runnerErr) || runnerErr.Code() != domain.ErrCodeRunnerSessionExpired {
			if err != nil {
				return nil, err
			}
			return nil, domain.NewValidationError("sessionId", "native session has not been proven missing")
		}
	}

	originalTask, err := o.GetTask(ctx, failedRun.TaskID)
	if err != nil {
		return nil, err
	}

	attachments, err := o.buildResumeFromFailureAttachments(ctx, failedRun, originalTask, req.CustomContext)
	if err != nil {
		return nil, err
	}

	for _, id := range req.AttachmentIDs {
		if strings.TrimSpace(id) == "" {
			continue
		}
		attachments = append(attachments, domain.ContextAttachment{
			Type:         "image",
			AttachmentID: id,
			Label:        "Uploaded image",
			Key:          "resume-image-" + id,
			Tags:         []string{"image", "resume"},
		})
	}

	prompt := buildResumePrompt(originalTask.Description, failedRun.ID, req.CustomContext)

	createReq := CreateRunRequest{
		TaskID:            originalTask.ID,
		AgentProfileID:    failedRun.AgentProfileID,
		Tag:               resumeTag(failedRun.Tag),
		SourceRunIDs:      []uuid.UUID{failedRun.ID},
		ConversationID:    failedRun.ConversationID,
		Prompt:            prompt,
		RunMode:           &failedRun.RunMode,
		ExecutionMode:     failedRun.ExecutionMode,
		SandboxConfig:     failedRun.SandboxConfig,
		ExistingSandboxID: failedRun.SandboxID,
		Environment:       failedRun.CustomEnv,
		WorkReferences:    failedRun.WorkReferences,
		WorkloadKind:      failedRun.Workload.Kind,
		WorkloadKey:       failedRun.Workload.Key,
		WorkloadInstance:  failedRun.Workload.Instance,
		OwnerSubject:      failedRun.OwnerSubject,
		OwnerScopes:       failedRun.OwnerScopes,
		OwnerExpiresAt:    failedRun.OwnerExpiresAt,
		RequestedScopes:   failedRun.RequestedScopes,
		IdempotencyKey:    freshRecoveryKey(failedRun.ID),
	}
	if cfg := failedRun.ResolvedConfig; cfg != nil {
		createReq.PreferredRunner = string(cfg.RunnerType)
		if cfg.RoleRef != "" {
			createReq.RoleRef = &cfg.RoleRef
		}
		if cfg.Model != "" {
			createReq.Model = &cfg.Model
		}
		createReq.MaxTurns, createReq.Timeout, createReq.Effort = &cfg.MaxTurns, &cfg.Timeout, &cfg.Effort
		createReq.AllowedTools, createReq.DeniedTools = append([]string{}, cfg.AllowedTools...), append([]string{}, cfg.DeniedTools...)
		createReq.AllowedPaths, createReq.DeniedPaths = append([]string{}, cfg.AllowedPaths...), append([]string{}, cfg.DeniedPaths...)
		createReq.AllowedEffects, createReq.RequireEffectContainment = cfg.AllowedEffects, cfg.RequireEffectContainment
		createReq.SkipPermissionPrompt, createReq.EnableBrowser = &cfg.SkipPermissionPrompt, &cfg.Features.EnableBrowser
		createReq.NetworkAccess, createReq.ExtraFlags = &cfg.NetworkAccess, cfg.ExtraFlags
		createReq.ResultSpec, createReq.Until = cfg.ResultSpec, cfg.Until
		if cfg.SandboxConfig != nil {
			createReq.SandboxConfig = cfg.SandboxConfig
		}
	}

	ctx, releaseAdmission, err := o.admitMaintenanceContext(ctx)
	if err != nil {
		return nil, err
	}
	defer releaseAdmission()
	claims, ok := o.runs.(repository.RunFreshRecoveryClaimer)
	if !ok {
		return nil, domain.NewStateError("Run", "unknown", "claim fresh recovery", "durable source claim owner is unavailable")
	}
	// A failed claim can have lost to another owner. Preserve uncertainty until
	// its exact durable replacement receipt is readable; never fall through.
	effectsPossible = true
	won, err := claims.ClaimFreshRecovery(ctx, failedRun.ID, failedRun.LifecycleVersion, freshRecoveryRequestHash(req), !automatic)
	if err != nil {
		return nil, err
	}
	if !won {
		if replayed, _, err := o.freshRecoveryAccepted(ctx, req); err != nil || replayed != nil {
			return replayed, err
		}
		return nil, domain.NewStateError("Run", "recovery_unresolved", "claim fresh recovery", "source lifecycle changed or another owner claimed recovery; no redispatch")
	}
	// Missing acceptance is never release authority. Only the owner's explicit
	// pre-effect error class permits a hash/version-fenced release below.
	newRun, err := o.createRun(ctx, createReq, &runRecoveryContext{attachments: attachments, config: failedRun.ResolvedConfig})
	if err != nil {
		if domain.IsPreEffectRefusal(err) {
			released, releaseErr := claims.ReleaseFreshRecoveryClaim(context.WithoutCancel(ctx), failedRun.ID, failedRun.LifecycleVersion+1, freshRecoveryRequestHash(req))
			if releaseErr != nil || !released {
				// Do not wrap the refusal marker: an unresolved release may now
				// belong to accepted work and must remain UNKNOWN to callers.
				return nil, domain.NewStateError("Run", "recovery_unresolved", "release fresh recovery claim", fmt.Sprintf("creation refused before effects but claim release is unresolved: refusal=%v release=%v", err, releaseErr))
			}
		}
		return nil, err
	}
	accepted, _, err := o.freshRecoveryAccepted(ctx, req)
	if err != nil {
		return nil, err
	}
	if accepted == nil || accepted.ID != newRun.ID {
		return nil, domain.NewStateError("Run", "recovery_unresolved", "fresh recovery", "creation response has no matching durable source/replacement receipt")
	}
	return o.attachRunActions(ctx, accepted), nil
}

// RecoverMissingSessionRun is the conservative owner-local automatic route.
// Cancellation remains available only through the explicit recovery API.
func (o *Orchestrator) RecoverMissingSessionRun(ctx context.Context, req ResumeFromFailedRunRequest) (_ *domain.Run, returnErr error) {
	o.wakeMu.Lock()
	defer o.wakeMu.Unlock()
	return o.resumeFromFailedRun(ctx, req, true)
}

// resumeTag derives a tag for the resumed run that preserves traceability
// to the original attempt. When the original tag is unset it falls back to
// the empty string and CreateRun assigns a default.
func resumeTag(prevTag string) string {
	prevTag = strings.TrimSpace(prevTag)
	if prevTag == "" {
		return ""
	}
	if strings.HasSuffix(prevTag, "-resume") {
		return prevTag
	}
	return prevTag + "-resume"
}

// excludeLiveExecutor delegates physical scope to the canonical control-plane
// owner. Only complete, exact-run absence permits fresh recovery; unavailable
// or ambiguous evidence never becomes permission to replace an executor.
func excludeLiveExecutor(ctx context.Context, run *domain.Run) error {
	if err := maintenance.ExcludeExecutor(ctx, run); err != nil {
		return domain.NewStateError("Run", "executor_not_excluded", "fresh recovery", err.Error())
	}
	return nil
}

// buildResumeFromFailureAttachments composes the context the resumed agent
// needs: the original task's attachments, plus a human-readable view of the
// prior attempt (overview, timeline, diff, history, user guidance).
func (o *Orchestrator) buildResumeFromFailureAttachments(
	ctx context.Context,
	failedRun *domain.Run,
	originalTask *domain.Task,
	customContext string,
) ([]domain.ContextAttachment, error) {
	attachments := make([]domain.ContextAttachment, 0, len(originalTask.ContextAttachments)+5)
	attachments = append(attachments, originalTask.ContextAttachments...)

	short := shortID(failedRun.ID)
	prevKey := "prev-" + short

	report, err := o.BuildRunReport(ctx, failedRun.ID)
	if err != nil {
		return nil, err
	}
	attachments = append(attachments, domain.ContextAttachment{
		Type: "note", Key: "previous-run-report-" + prevKey, Label: "Previous Attempt " + prevKey,
		Content: runreport.Text(report), Format: "markdown", Priority: "high",
		Summary: "Bounded diagnostics for the failed attempt", Tags: []string{"run", "resume", "report"},
	})

	diff, err := o.GetRunDiff(ctx, failedRun.ID)
	if err == nil && diff != nil {
		diffJSON, err := marshalJSON(diff)
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, domain.ContextAttachment{
			Type:     "note",
			Key:      fmt.Sprintf("prev-run-diff-%s", short),
			Label:    fmt.Sprintf("Previous Attempt Diff %s", short),
			Content:  diffJSON,
			Format:   "json",
			Priority: "high",
			Summary:  fmt.Sprintf("Code changes already present on disk from failed run %s (%d files, %d bytes)", short, failedRun.ChangedFiles, failedRun.TotalSizeBytes),
			Tags:     []string{"run", "diff", "resume"},
		})
	}

	if failedRun.AgentProfileID != nil {
		if att, ok := o.buildHistoricalContext(ctx, failedRun, prevKey); ok {
			attachments = append(attachments, att)
		}
	}

	if strings.TrimSpace(customContext) != "" {
		attachments = append(attachments, domain.ContextAttachment{
			Type:     "note",
			Key:      "user-resume-context",
			Label:    "Additional Context for Resume",
			Content:  customContext,
			Format:   "markdown",
			Priority: "high",
			Summary:  "User-provided guidance for completing the remaining work",
			Tags:     []string{"user", "context", "resume"},
		})
	}

	return attachments, nil
}

// buildResumePrompt frames the resumed run for the agent: the original task
// goal, then explicit instructions to reuse work that's already on disk and
// only complete what's left.
func buildResumePrompt(originalPrompt string, failedRunID uuid.UUID, customContext string) string {
	var sb strings.Builder

	sb.WriteString(strings.TrimSpace(originalPrompt))
	sb.WriteString("\n\n---\n\n")

	sb.WriteString("## Prior Attempt\n\n")
	sb.WriteString(fmt.Sprintf("This task was previously attempted in run `%s`, which did not complete successfully. ", failedRunID))
	sb.WriteString("The attached context contains:\n\n")
	sb.WriteString("- **Previous Attempt Run Overview**: status, error, and execution metadata\n")
	sb.WriteString("- **Run Timeline**: the full event log of what the prior agent did\n")
	sb.WriteString("- **Previous Attempt Diff**: code changes that were made and may already be on disk\n\n")
	sb.WriteString("**Your job**:\n\n")
	sb.WriteString("1. Review the prior timeline and diff to understand what was already attempted.\n")
	sb.WriteString("2. Detect work that is already present on disk — do NOT redo it. Treat completed changes as done.\n")
	sb.WriteString("3. Identify the remaining work needed to satisfy the original task and complete it.\n")
	sb.WriteString("4. If the prior attempt left the workspace in a broken or inconsistent state, fix that first, then continue.\n")
	sb.WriteString("5. If the original task is no longer feasible (e.g. requirements have shifted or the prior attempt revealed a blocker), explain clearly and stop rather than guessing.\n\n")

	sb.WriteString("## Diagnostics\n\n")
	sb.WriteString("Use these CLI commands to fetch additional detail beyond the attached context:\n\n")
	sb.WriteString(fmt.Sprintf("```bash\nagent-manager run get %s\nagent-manager run events %s\nagent-manager run diff %s\n```\n", failedRunID, failedRunID, failedRunID))

	if strings.TrimSpace(customContext) != "" {
		sb.WriteString("\n## User-Provided Guidance\n\n")
		sb.WriteString(strings.TrimSpace(customContext))
		sb.WriteString("\n")
	}

	return sb.String()
}
