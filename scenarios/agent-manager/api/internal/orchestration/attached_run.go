// Package orchestration owns the identity and liveness seam for externally
// started harness sessions without spawning, streaming, or controlling them.
package orchestration

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration/obs"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	coreidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
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
	releaseAdmission, err := o.admitMaintenance(ctx)
	if err != nil {
		return nil, err
	}
	defer releaseAdmission()

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

	// Attribution to the operator who spawned this session from the Web Console
	// UI. The attach request itself is a zero-scope loopback call (LoopbackOnly,
	// so authn.RequireHuman cannot succeed on it), so the human crosses the
	// boundary as a VERIFIED, session-scoped delegation minted by
	// scenario-authenticator and forwarded in the attach. We verify it against the
	// same scenario-authenticator root the owner path uses and bind it to THIS
	// session; only then does the run receive the delegation's exact owner scope
	// ceiling. The attached identity expires no later than that proof. A missing
	// or unverifiable delegation leaves the run unattributed (zero-scope), never
	// refused — a local/agent/loopback attach can therefore never self-assert a
	// human or supply its own authority.
	run.WebConsoleSessionID = req.WebConsoleSessionID
	delegationVerified := false
	delegationExpiresAt := time.Time{}
	if subject, scopes, expiresAt := o.verifiedDelegationOwner(ctx, req.HumanOwnerDelegation, req.WebConsoleSessionID); subject != "" {
		delegationVerified = true
		delegationExpiresAt = expiresAt
		run.OwnerSubject = subject
		run.OwnerScopes = scopes
		if !expiresAt.IsZero() {
			run.OwnerExpiresAt = &expiresAt
		}
	} else if owner, err := authn.RequireHuman(ctx); err == nil && (owner.ExpiresAt.IsZero() || owner.ExpiresAt.After(now)) {
		// Retained for a genuine in-context verified human (non-loopback callers).
		run.OwnerSubject = owner.Subject
		if ownerScopeExpiryUsable(owner.ExpiresAt, now) {
			run.OwnerScopes = append([]string(nil), owner.Scopes...)
			expiry := owner.ExpiresAt
			run.OwnerExpiresAt = &expiry
		}
	}

	expiresAt := now.Add(identity.DefaultTTL)
	if run.OwnerExpiresAt != nil && expiresAt.After(*run.OwnerExpiresAt) {
		expiresAt = *run.OwnerExpiresAt
	}
	identityScopes := append([]string{}, run.OwnerScopes...)
	if run.OwnerSubject == "" {
		identityScopes = []string{}
	}
	token, err := identity.GenerateToken(&identity.Claims{
		RunID:      run.ID,
		Subject:    run.OwnerSubject,
		TaskID:     run.TaskID,
		ProfileKey: kind,
		IssuedAt:   now.Unix(),
		ExpiresAt:  expiresAt.Unix(),
		Scopes:     identityScopes,
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
	issuedAt := time.Unix(now.Unix(), 0).UTC()
	run.IdentityTokenIssuedAt = &issuedAt
	if err := o.runs.Create(ctx, run); err != nil {
		return nil, err
	}
	o.appendAttachedLifecycleEvent(ctx, run.ID, "attached", "harness="+kind+" session="+session)
	// Phase-5 parity with an agent-manager-initiated interactive spawn: a verified
	// human delegation mints a short-lived, session-scoped standing grant so work
	// done inside this operator session inherits the operator's authority, bounded
	// and attributable. Best-effort — a mint failure must not fail the attach.
	if delegationVerified && run.OwnerSubject != "" && run.WebConsoleSessionID != "" && o.standingGrants != nil {
		// Match the attached identity's second-resolution exp claim exactly. The
		// grant owner computes its expiry from one clock read, so elapsed time
		// between attach and persistence cannot extend it past this proof.
		proofExpiry := time.Unix(delegationExpiresAt.Unix(), 0)
		if _, err := o.createSessionGrant(ctx, run.OwnerSubject, run.WebConsoleSessionID, MaxSessionGrantLifetime, proofExpiry); err != nil {
			obs.Component("attached").Warn("attached run: session grant mint failed",
				obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
		}
	}
	return &AttachRunResult{Run: o.attachRunActions(ctx, run), Token: token, ExpiresAt: expiresAt}, nil
}

// verifiedDelegationOwner verifies a forwarded Web Console human-owner delegation
// and returns the delegated operator subject, scopes, and expiry — but ONLY when
// the token cryptographically verifies against the scenario-authenticator root,
// names a human, and its wc_session_id claim equals the attaching session. Any
// failure returns an empty subject, so the caller leaves the run unattributed.
// This is the laundering guard: a bare string, a wrong-key or wrong-audience
// token, a wrong-session delegation, or an expired one all verify to nothing.
func (o *Orchestrator) verifiedDelegationOwner(ctx context.Context, delegation, session string) (string, []string, time.Time) {
	delegation = strings.TrimSpace(delegation)
	if delegation == "" || o.ownerIdentity == nil {
		return "", nil, time.Time{}
	}
	principal, err := o.ownerIdentity.Verify(ctx, delegation)
	if err != nil || !principal.IsHuman() || !ownerScopeExpiryUsable(principal.ExpiresAt, o.now()) {
		return "", nil, time.Time{}
	}
	if boundSession := delegationWCSessionID(delegation); boundSession == "" || boundSession != session {
		// The delegation must be bound to THIS web-console session; a session's
		// delegation cannot be replayed onto another session's attach.
		return "", nil, time.Time{}
	}
	return principal.Subject, append([]string(nil), principal.Scopes...), principal.ExpiresAt
}

// ownerScopeExpiryUsable ensures an owner scope ceiling can produce a valid
// whole-second identity-token expiry that remains within the source proof.
func ownerScopeExpiryUsable(expiresAt, now time.Time) bool {
	return !expiresAt.IsZero() && expiresAt.After(now) && expiresAt.Unix() > now.Unix()
}

// delegationWCSessionID extracts the wc_session_id claim from an already-verified
// delegation token. The shared verifier confirms signature/issuer/audience/expiry
// and does not surface this custom claim, so we read it from the verified
// payload. Authenticity is already established by verifiedDelegationOwner before
// this is trusted; a parse failure yields "" (treated as no binding).
func delegationWCSessionID(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		WCSessionID string `json:"wc_session_id"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	return strings.TrimSpace(claims.WCSessionID)
}

// DetachRun closes an attached run without sending a signal to the harness
// process. The token is revoked before the terminal state is returned.
func (o *Orchestrator) DetachRun(ctx context.Context, id uuid.UUID, reason string) (*domain.Run, error) {
	if caller, ok := coreidentity.PrincipalFromContext(ctx); ok && caller.Kind == coreidentity.ActorAgent {
		verified, err := o.VerifyIdentityToken(ctx, provenance.ForwardedIdentityToken(ctx))
		if !caller.IsVerified() || caller.Subject != id.String() || (!caller.ExpiresAt.IsZero() && !caller.ExpiresAt.After(o.now())) || err != nil || verified == nil || !verified.Valid || verified.Claims == nil || verified.Claims.RunID != id || !time.Unix(verified.Claims.ExpiresAt, 0).After(o.now()) {
			return nil, domain.NewValidationErrorWithCode("authorization", "current attached self identity required", domain.ErrCodePolicyScope)
		}
	}

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
		if caller, ok := coreidentity.PrincipalFromContext(ctx); ok && caller.Kind == coreidentity.ActorAgent {
			return nil, domain.NewValidationErrorWithCode("authorization", "self cleanup requires an attached run", domain.ErrCodePolicyScope)
		}
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
