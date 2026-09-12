// Package orchestration owns the identity and liveness seam for externally
// started harness sessions without spawning, streaming, or controlling them.
package orchestration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"

	"github.com/google/uuid"
)

const attachedRunLivenessGracePeriod = time.Minute

// AttachRun creates an identity-bearing run for a harness process that was
// started outside agent-manager. It deliberately does not resolve a profile,
// create a task, read a transcript, or enqueue any runner work.
func (o *Orchestrator) AttachRun(ctx context.Context, req AttachRunRequest) (*AttachRunResult, error) {
	kind := strings.TrimSpace(req.HarnessKind)
	session := strings.TrimSpace(req.HarnessSession)
	if kind == "" {
		return nil, domain.NewValidationError("harness_kind", "field is required")
	}
	if session == "" {
		return nil, domain.NewValidationError("harness_session_id", "field is required")
	}
	if len(kind) > 100 || len(session) > 512 {
		return nil, domain.NewValidationError("harness", "harness identity is too long")
	}
	if req.ProcessID < 0 {
		return nil, domain.NewValidationError("process_id", "must be positive when supplied")
	}
	if len(o.identitySecret) == 0 {
		return nil, domain.NewConfigMissingError("identity", "identity signing secret is not configured", nil)
	}
	if o.runs == nil {
		return nil, domain.NewConfigMissingError("runs", "run repository is not configured", nil)
	}

	taskID := uuid.Nil
	if req.TaskID != nil {
		if *req.TaskID == uuid.Nil {
			return nil, domain.NewValidationError("task_id", "must be a valid UUID")
		}
		if o.tasks == nil {
			return nil, domain.NewConfigMissingError("tasks", "task repository is not configured", nil)
		}
		task, err := o.tasks.Get(ctx, *req.TaskID)
		if err != nil {
			return nil, err
		}
		if task == nil {
			return nil, domain.NewNotFoundError("Task", *req.TaskID)
		}
		taskID = *req.TaskID
	}

	now := o.now()
	runID := uuid.New()
	label, labelSource := attachedRunLabel(kind, session, req.HarnessTitle)
	run := &domain.Run{
		ID:                 runID,
		TaskID:             taskID,
		Tag:                "attached-" + kind + "-" + runID.String()[:8],
		Label:              label,
		LabelSource:        labelSource,
		RunMode:            domain.RunModeInPlace,
		ExecutionMode:      domain.ExecutionModeAttached,
		HarnessKind:        kind,
		HarnessSessionID:   session,
		Status:             domain.RunStatusRunning,
		Phase:              domain.RunPhaseExecuting,
		StartedAt:          &now,
		ApprovalState:      domain.ApprovalStateNone,
		ProgressPercent:    0,
		FinalizationStatus: domain.RunFinalizationStatusNone,
		RunnerPID:          req.ProcessID,
	}

	expiresAt := now.Add(identity.DefaultTTL)
	token, err := identity.GenerateToken(&identity.Claims{
		RunID:      run.ID,
		TaskID:     run.TaskID,
		ProfileKey: kind,
		IssuedAt:   now.Unix(),
		ExpiresAt:  expiresAt.Unix(),
		Scopes:     []string{},
		Meta: map[string]string{
			"execution_mode":  string(domain.ExecutionModeAttached),
			"harness_kind":    kind,
			"harness_session": session,
		},
	}, o.identitySecret)
	if err != nil {
		return nil, fmt.Errorf("mint attached identity token: %w", err)
	}
	run.IdentityTokenHash = identity.HashToken(token)
	if err := o.runs.Create(ctx, run); err != nil {
		return nil, err
	}
	o.appendAttachedLifecycleEvent(ctx, run.ID, "attached", "harness="+kind+" session="+session)
	return &AttachRunResult{Run: o.attachRunActions(ctx, run), Token: token, ExpiresAt: expiresAt}, nil
}

// DetachRun closes an attached run without sending a signal to the harness
// process. The token is revoked before the terminal state is returned.
func (o *Orchestrator) DetachRun(ctx context.Context, id uuid.UUID, reason string) (*domain.Run, error) {
	if o.runs == nil {
		return nil, domain.NewConfigMissingError("runs", "run repository is not configured", nil)
	}
	run, err := o.runs.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, domain.NewNotFoundError("Run", id)
	}
	if run.ExecutionMode.Normalized() != domain.ExecutionModeAttached {
		return nil, domain.NewStateError("Run", string(run.ExecutionMode), "detach", "only attached runs can be detached")
	}
	if run.Status.IsTerminal() {
		return o.attachRunActions(ctx, run), nil
	}
	now := o.now()
	run.Status = domain.RunStatusComplete
	run.Phase = domain.RunPhaseCompleted
	run.EndedAt = &now
	if strings.TrimSpace(reason) == "" {
		reason = "detached by operator"
	}
	run.ErrorMsg = ""
	if run.IdentityTokenHash != "" {
		run.IdentityTokenRevokedAt = &now
	}
	if err := o.runs.Update(ctx, run); err != nil {
		return nil, err
	}
	o.appendAttachedLifecycleEvent(ctx, run.ID, "detached", reason)
	return o.attachRunActions(ctx, run), nil
}

func attachedRunLabel(kind, session, title string) (string, domain.RunLabelSource) {
	if title = strings.TrimSpace(title); title != "" {
		return title, domain.RunLabelSourceHarness
	}
	return fmt.Sprintf("%s session %s", kind, session), domain.RunLabelSourceHarness
}

func (o *Orchestrator) appendAttachedLifecycleEvent(ctx context.Context, id uuid.UUID, state, detail string) {
	if o.events == nil {
		return
	}
	_ = o.events.Append(ctx, id, domain.NewLogEvent(id, "attached_run_"+state, detail))
}
