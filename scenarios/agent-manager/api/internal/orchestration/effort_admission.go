// effort_admission.go enforces immutable finite approval at native admission and lifecycle boundaries.
package orchestration

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/effortauthority"
)

type effortAdmission struct {
	binding  effortauthority.Binding
	policy   effortauthority.Policy
	intent   effortauthority.Intent
	proof    *effortauthority.Proof
	recovery bool
	reserved bool
}

func bindEffortCaller(req *CreateRunRequest) {
	if req.effort == nil || req.caller == nil {
		return
	}
	b := req.effort.binding
	req.caller.EffortID = b.PolicyID
	req.caller.EffortEpoch = b.Epoch
	req.caller.EffortDigest = b.PolicyDigest
}
func (o *Orchestrator) authenticateEffortCaller(ctx context.Context, req *CreateRunRequest) error {
	// A third offered channel is never an anonymous/human fallback. Mixing
	// effort authority and another credential is intentionally unsupported.
	if req.OwnerToken != "" || req.RunIdentityToken != "" || len(req.EffortProof) > 24000 || o.effortAuthority == nil {
		return effortauthority.ErrRefused
	}
	if req.effort == nil && (effortHasOverrides(req) || req.RequestedScopes != nil || req.ExpectedOwnerSubject != "") {
		return effortauthority.ErrRefused
	}
	raw, e := base64.RawURLEncoding.DecodeString(req.EffortProof)
	if e != nil {
		return effortauthority.ErrRefused
	}
	var proof effortauthority.Proof
	if e = json.Unmarshal(raw, &proof); e != nil {
		return effortauthority.ErrRefused
	}
	if proof.Intent.Effect != "run.create" {
		return effortauthority.ErrRefused
	}
	b, e := o.effortAuthority.CheckProof(ctx, proof)
	if e != nil {
		return e
	}
	p, e := o.effortAuthority.CheckBinding(ctx, b)
	if e != nil {
		return e
	}
	req.effort = &effortAdmission{binding: b, policy: p, intent: proof.Intent, proof: &proof}
	req.OwnerSubject = b.Owner
	req.OwnerScopes = append([]string{}, p.Scopes...)
	deadline := b.Deadline
	req.OwnerExpiresAt = &deadline
	req.caller = &domain.CreateRunCaller{Kind: "effort", Subject: b.Owner}
	bindEffortCaller(req)
	return narrowCreateRunScopes(req)
}
func (o *Orchestrator) inheritEffortCaller(ctx context.Context, req *CreateRunRequest, parent *domain.Run) error {
	if req.effort == nil && effortHasOverrides(req) {
		return effortauthority.ErrRefused
	}
	b := *parent.ResolvedConfig.Admission.Effort
	p, e := o.effortAuthority.CheckBinding(ctx, b)
	if e != nil || !p.AllowChildren {
		return effortauthority.ErrRefused
	}
	old := parent.ResolvedConfig.Admission.EffortIntent
	if old == nil {
		return effortauthority.ErrRefused
	}
	i := *old
	i.Effect = "run.child"
	i.ParentRunID = parent.ID.String()
	i.SourceRunID = ""
	i.IdempotencyKey = req.IdempotencyKey
	req.effort = &effortAdmission{binding: b, policy: p, intent: i}
	return nil
}

// EffortProfileDigest pins the existing native owner's complete profile. It
// includes permissions, native budget and timestamps; no field is widened.
func EffortProfileDigest(p *domain.AgentProfile) string { return effortauthority.Digest(p) }
func effortTaskDigest(t *domain.Task) string {
	// Lifecycle timestamps/status change after admission. Pin executable inputs.
	attachments := t.ContextAttachments
	if len(attachments) == 0 {
		attachments = nil
	}
	return effortauthority.Digest(struct {
		Title, Description, ScopePath, ProjectRoot string
		Attachments                                []domain.ContextAttachment
	}{t.Title, t.Description, t.ScopePath, t.ProjectRoot, attachments})
}
func effortNativeInput(req CreateRunRequest, t *domain.Task, profile string) effortauthority.RunInput {
	parent := ""
	if req.ParentRunID != nil {
		parent = req.ParentRunID.String()
	}
	return effortauthority.RunInput{TaskID: req.TaskID.String(), TaskDigest: effortTaskDigest(t), Profile: profile, ParentRunID: parent, IdempotencyKey: req.IdempotencyKey, Tag: req.Tag, Environment: req.Environment}
}
func (o *Orchestrator) reserveEffortAdmission(ctx context.Context, req *CreateRunRequest, t *domain.Task, cfg *domain.RunConfig) error {
	if req.effort == nil {
		return nil
	}
	a := req.effort
	if req.ExecutionMode.Normalized() != domain.ExecutionModeCodecPipe {
		return fmt.Errorf("%w: finite interactive/imported launch is unqualified", effortauthority.ErrRefused)
	}
	for _, ref := range req.WorkReferences {
		if ref != nil && ref.Kind == "effort" && (ref.Id != a.policy.Effort || ref.Revision != a.policy.Revision) {
			return effortauthority.ErrRefused
		}
	}
	if req.IdempotencyKey == "" || req.Force || req.ForceInPlace || t.ProjectRoot != a.policy.Repository || t.ScopePath != a.policy.Repository || !filepath.IsAbs(t.ProjectRoot) || filepath.Clean(t.ProjectRoot) != t.ProjectRoot {
		return effortauthority.ErrRefused
	}
	// Resolve symlinks on both sides, never treat a sibling/symlink as scope.
	real, e := filepath.EvalSymlinks(t.ProjectRoot)
	if e != nil || real != t.ProjectRoot {
		return effortauthority.ErrRefused
	}
	var profile *domain.AgentProfile
	if req.profileAdmission != nil {
		profile = req.profileAdmission.profile
	} else if req.AgentProfileID != nil && o.profiles != nil {
		profile, e = o.profiles.Get(ctx, *req.AgentProfileID)
		if e != nil {
			return effortauthority.ErrRefused
		}
	}
	if profile == nil || a.policy.Profiles[profile.ProfileKey] != EffortProfileDigest(profile) {
		return fmt.Errorf("%w: profile differs from approved snapshot", effortauthority.ErrRefused)
	}
	current, e := o.profiles.GetByKey(ctx, profile.ProfileKey)
	if e != nil || current == nil || EffortProfileDigest(current) != EffortProfileDigest(profile) {
		return fmt.Errorf("%w: current profile changed during admission", effortauthority.ErrRefused)
	}
	// Ordinary effort starts/children use an existing unmodified profile. A
	// private recovery is already bound to the source's retained config.

	// Finite approval may narrow native ceilings, never increase them. Full
	// approved ceilings are conservatively reserved across children/recovery.
	seconds := a.policy.MaxRunSeconds
	i := a.intent
	i.Profile = profile.ProfileKey
	i.ProfileDigest = EffortProfileDigest(profile)
	i.Repository = t.ProjectRoot
	i.IdempotencyKey = req.IdempotencyKey
	if !a.recovery {
		i.InputDigest = effortauthority.Digest(effortNativeInput(*req, t, profile.ProfileKey))
	}
	i.Turns = a.policy.MaxTurns
	i.ToolCalls = a.policy.MaxToolCalls
	i.RunSeconds = seconds
	if a.proof != nil && effortauthority.Digest(i) != effortauthority.Digest(a.proof.Intent) {
		return fmt.Errorf("%w: native input differs from signed intent", effortauthority.ErrRefused)
	}
	if a.reserved {
		if effortauthority.Digest(i) != effortauthority.Digest(a.intent) {
			return effortauthority.ErrRefused
		}
		return nil
	}
	reservation, replay, e := o.effortAuthority.Reserve(ctx, a.binding, i, a.proof)
	if e != nil {
		return e
	}
	if replay {
		// The durable native replay branch has already run. Even a failed pending
		// reservation is uncertainty, not permission to send another dispatch.
		return domain.NewStateError("Run", "effort_admission_unresolved", "create", "finite intent already reserved; reconcile original owner receipt, no redispatch")
	}
	a.intent = reservation.Intent
	a.reserved = true
	return nil
}
func (o *Orchestrator) acceptEffortRun(ctx context.Context, r *domain.Run) error {
	if r.ResolvedConfig == nil || r.ResolvedConfig.Admission == nil || r.ResolvedConfig.Admission.Effort == nil {
		return nil
	}
	return o.effortAuthority.Accept(ctx, *r.ResolvedConfig.Admission.Effort, r.IdempotencyKey, r.ID.String())
}

// SyncEffortTerminal is an exact native-owner reconciliation, not a caller's
// budget release. Unknown/deleted runs keep their conservative reservations.
func (o *Orchestrator) SyncEffortTerminal(ctx context.Context, b effortauthority.Binding, key, runID string) error {
	id, e := uuid.Parse(runID)
	if e != nil {
		return effortauthority.ErrRefused
	}
	r, e := o.runs.Get(ctx, id)
	if e != nil || r == nil || r.ResolvedConfig == nil || r.ResolvedConfig.Admission == nil || r.ResolvedConfig.Admission.Effort == nil || *r.ResolvedConfig.Admission.Effort != b || r.IdempotencyKey != key || !r.Status.IsTerminal() {
		return effortauthority.ErrRefused
	}
	if o.finiteNativeTerminal == nil {
		return effortauthority.ErrRefused
	}
	if e := o.finiteNativeTerminal.Terminal(ctx, runID); e != nil {
		return e
	}
	if e := o.effortAuthority.Accept(ctx, b, key, runID); e != nil {
		return e
	}
	return o.effortAuthority.Settle(ctx, b, key, runID)
}

func (o *Orchestrator) prepareEffortRecovery(ctx context.Context, request ResumeFromFailedRunRequest, source *domain.Run, req *CreateRunRequest, task *domain.Task, cfg *domain.RunConfig) error {
	if o.finiteNativeFactory.Enabled() {
		return effortauthority.ErrRefused
	}
	b := *source.ResolvedConfig.Admission.Effort
	policy, e := o.effortAuthority.CheckBinding(ctx, b)
	if e != nil || !policy.AllowRecovery || request.EffortProof == "" {
		return effortauthority.ErrRefused
	}
	raw, e := base64.RawURLEncoding.DecodeString(request.EffortProof)
	if e != nil || len(raw) > 18000 {
		return effortauthority.ErrRefused
	}
	var proof effortauthority.Proof
	if json.Unmarshal(raw, &proof) != nil {
		return effortauthority.ErrRefused
	}
	verified, e := o.effortAuthority.CheckProof(ctx, proof)
	if e != nil || verified != b || proof.Intent.Effect != "run.recover" || proof.Intent.SourceRunID != source.ID.String() || proof.Intent.IdempotencyKey != req.IdempotencyKey || proof.Intent.InputDigest != effortauthority.Digest(request) {
		return effortauthority.ErrRefused
	}
	if source.OwnerSubject != b.Owner || source.OwnerExpiresAt == nil || source.OwnerExpiresAt.After(b.Deadline) {
		return effortauthority.ErrRefused
	}
	req.effort = &effortAdmission{binding: b, policy: policy, intent: proof.Intent, proof: &proof, recovery: true}
	req.caller = &domain.CreateRunCaller{Kind: "effort-recovery", Subject: b.Owner, RunID: source.ID}
	bindEffortCaller(req)
	// Recheck and reserve before the durable replacement claim, never after it.
	return o.reserveEffortAdmission(ctx, req, task, cfg)
}

func narrowEffortConfig(req CreateRunRequest, cfg *domain.RunConfig, now time.Time) {
	if req.effort == nil {
		return
	}
	p := req.effort.policy
	if cfg.MaxTurns <= 0 || cfg.MaxTurns > p.MaxTurns {
		cfg.MaxTurns = p.MaxTurns
	}
	if cfg.MaxToolCalls <= 0 || cfg.MaxToolCalls > p.MaxToolCalls {
		cfg.MaxToolCalls = p.MaxToolCalls
	}
	ceiling := time.Duration(p.MaxRunSeconds) * time.Second
	if remaining := p.Deadline.Sub(now); remaining < ceiling {
		ceiling = remaining
	}
	if cfg.Timeout <= 0 || cfg.Timeout > ceiling {
		cfg.Timeout = ceiling
	}
}

func effortHasOverrides(req *CreateRunRequest) bool {
	return (req.ProfileRef != nil && (req.ProfileRef.Defaults != nil || req.ProfileRef.UpdateExisting)) || req.RoleRef != nil || req.MaxTurns != nil || req.MaxToolCalls != nil || req.Timeout != nil || req.Model != nil || req.Effort != nil || req.PreferredRunner != "" || req.AllowedTools != nil || req.DeniedTools != nil || req.SkipPermissionPrompt != nil || req.EnableBrowser != nil || req.ExtraFlags != nil || req.NetworkAccess != nil || req.AllowedPaths != nil || req.DeniedPaths != nil || req.AllowedEffects != nil || req.RequireEffectContainment || req.ResultSpec != nil || req.Until != "" || req.SandboxConfig != nil || req.ExistingSandboxID != nil || req.ExecutionMode != "" || req.Prompt != "" || len(req.SourceRunIDs) > 0 || req.SourceInvestigationRunID != nil || req.RunMode != nil || req.ConversationID != ""
}

// Continuation of a B-bound run keeps its original capacity and deadline. It
// cannot use an expired/revoked grant to reach wake/resume bookkeeping/dispatch.
// Ordinary AUTH-01 runs retain their owning lifecycle contract.
func (o *Orchestrator) checkEffortContinuation(ctx context.Context, r *domain.Run) error {
	if o.finiteNativeFactory.Enabled() {
		return effortauthority.ErrRefused
	}
	if r == nil || r.ResolvedConfig == nil || r.ResolvedConfig.Admission == nil || r.ResolvedConfig.Admission.Effort == nil {
		return nil
	}
	b := *r.ResolvedConfig.Admission.Effort
	p, e := o.effortAuthority.CheckBinding(ctx, b)
	if e != nil || !p.AllowRecovery || r.OwnerSubject != b.Owner || r.OwnerExpiresAt == nil || r.OwnerExpiresAt.After(b.Deadline) || !r.OwnerExpiresAt.After(o.now()) {
		return effortauthority.ErrRefused
	}
	if r.AgentProfileID == nil {
		return effortauthority.ErrRefused
	}
	profile, e := o.profiles.Get(ctx, *r.AgentProfileID)
	if e != nil || profile == nil || p.Profiles[profile.ProfileKey] != EffortProfileDigest(profile) {
		return effortauthority.ErrRefused
	}
	slot, e := o.effortAuthority.ReadReservation(ctx, b, r.IdempotencyKey)
	if e != nil || !slot.NativeBound || slot.RunID != r.ID.String() || slot.Terminal {
		return effortauthority.ErrRefused
	}
	remaining := b.Deadline.Sub(o.now())
	if remaining <= 0 {
		return effortauthority.ErrRefused
	}
	if r.ResolvedConfig.Timeout <= 0 || r.ResolvedConfig.Timeout > remaining {
		r.ResolvedConfig.Timeout = remaining
	}
	return nil
}
