package orchestration

import (
	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	requestidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/scopecatalog"
	"github.com/vrooli/cli-core/cliutil"
	"os"
	"strings"
	"time"
)

type AuthorizationRequest struct {
	RunID           uuid.UUID `json:"run_id"`
	PID             int       `json:"pid"`
	HarnessKind     string    `json:"harness_kind"`
	HarnessSession  string    `json:"harness_session"`
	Policy          string    `json:"policy"`
	Targets         []string  `json:"targets"`
	DurationSeconds int64     `json:"duration_seconds"`
}

func WithAuthorizations(repo domain.AuthorizationRepository) Option {
	return func(o *Orchestrator) { o.authorizations = repo }
}

func (o *Orchestrator) RequestAuthorization(ctx context.Context, req AuthorizationRequest) (*domain.AuthorizationGrant, error) {
	if o.authorizations == nil || len(o.identitySecret) == 0 {
		return nil, errors.New("authorization authority is not configured")
	}
	policy, ok := authn.InteractivePolicy(req.Policy)
	if !ok {
		return nil, domain.NewValidationError("policy", "unknown reviewed policy")
	}
	if req.DurationSeconds <= 0 || req.DurationSeconds > 8*60*60 {
		return nil, domain.NewValidationError("duration_seconds", "must be between 1 second and 8 hours")
	}
	if len(req.Targets) == 0 || len(req.Targets) > 64 {
		return nil, domain.NewValidationError("targets", "one to 64 explicit targets required")
	}
	for _, target := range req.Targets {
		d, object, ok := strings.Cut(target, ":")
		valid := false
		for _, allowed := range policy.Domains {
			if d == allowed {
				valid = true
			}
		}
		if !ok || !valid || object == "" || len(target) > 1024 || strings.ContainsAny(object, "\r\n") || (strings.Contains(object, "*") && object != "*") {
			return nil, domain.NewValidationError("targets", "target is outside policy domains or contains a wildcard expression")
		}
	}
	binding, err := cliutil.ObserveProcessBinding(req.PID)
	if err != nil {
		return nil, domain.NewValidationError("pid", "live process binding unavailable")
	}
	if binding.UID != os.Getuid() {
		return nil, domain.NewValidationError("pid", "process belongs to another host account")
	}
	if req.RunID == uuid.Nil && (strings.TrimSpace(req.HarnessKind) == "" || strings.TrimSpace(req.HarnessSession) == "") {
		return nil, domain.NewValidationError("harness", "session identity required for identity-only attachment")
	}
	if len(req.HarnessKind) > 100 || len(req.HarnessSession) > 512 {
		return nil, domain.NewValidationError("harness", "session identity too long")
	}
	previous, err := o.authorizations.List(ctx, req.PID)
	if err != nil {
		return nil, err
	}
	for _, grant := range previous {
		if grant.Binding == binding && grant.State == "pending" && o.now().Before(grant.RequestExpiresAt) {
			return nil, domain.NewValidationError("authorization", "a pending request already exists for this process; inspect or approve it first")
		}
	}
	now := o.now().UTC()
	grant := &domain.AuthorizationGrant{ID: uuid.New(), RunID: req.RunID, State: "pending", Policy: req.Policy, Targets: append([]string(nil), req.Targets...), DurationSeconds: req.DurationSeconds, Binding: binding, HarnessKind: req.HarnessKind, HarnessSession: req.HarnessSession, CreatedAt: now, RequestExpiresAt: now.Add(15 * time.Minute)}
	if err := o.authorizations.Create(ctx, grant); err != nil {
		return nil, err
	}
	return grant, nil
}

func (o *Orchestrator) GetAuthorization(ctx context.Context, id uuid.UUID) (*domain.AuthorizationGrant, error) {
	if o.authorizations == nil {
		return nil, errors.New("authorization authority is not configured")
	}
	g, err := o.authorizations.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, domain.NewNotFoundError("Authorization", id)
	}
	return g, nil
}
func (o *Orchestrator) ListAuthorizations(ctx context.Context, pid int) ([]*domain.AuthorizationGrant, error) {
	if o.authorizations == nil {
		return nil, errors.New("authorization authority is not configured")
	}
	return o.authorizations.List(ctx, pid)
}

func (o *Orchestrator) ApproveAuthorization(ctx context.Context, id uuid.UUID) (*domain.AuthorizationGrant, error) {
	p, err := authn.RequireHuman(ctx)
	if err != nil {
		return nil, err
	}
	// A shared OS account or app-local token cannot attest independent consent.
	if p.Source != requestidentity.SourceCloudflareAccess && p.Source != requestidentity.SourceScenarioAuthenticator {
		return nil, errors.New("external verified human consent is required")
	}
	if _, err := authn.RequireCapability(ctx, "agent-manager:write"); err != nil {
		return nil, err
	}
	g, err := o.GetAuthorization(ctx, id)
	if err != nil {
		return nil, err
	}
	if g.State != "pending" || !o.now().Before(g.RequestExpiresAt) {
		return nil, domain.NewValidationError("authorization", "request is expired or already decided")
	}
	policy, ok := authn.InteractivePolicy(g.Policy)
	if !ok {
		return nil, errors.New("reviewed policy no longer available")
	}
	for _, scope := range policy.Scopes {
		if !scopecatalog.Resolve(p.Scopes, scope) {
			return nil, authn.ErrCapability
		}
	}
	live, err := cliutil.ObserveProcessBinding(g.Binding.PID)
	if err != nil || live != g.Binding {
		return nil, errors.New("authorization session has ended or changed")
	}
	now, expires := o.now().UTC(), o.now().UTC().Add(time.Duration(g.DurationSeconds)*time.Second)
	if !p.ExpiresAt.IsZero() && expires.After(p.ExpiresAt) {
		expires = p.ExpiresAt
	}
	if !expires.After(now) {
		return nil, errors.New("approver credential expired")
	}
	var run *domain.Run
	if g.RunID != uuid.Nil {
		run, err = o.GetRun(ctx, g.RunID)
		if err != nil {
			return nil, err
		}
		if run == nil || run.Status.IsTerminal() || !cliutil.ProcessDescendsFrom(run.RunnerPID, g.Binding) || run.OwnerSubject != p.Subject {
			return nil, errors.New("run is not live and bound to this owner and process")
		}
		if run.ParentRunID != nil || (run.ResolvedConfig != nil && run.ResolvedConfig.Admission != nil && run.ResolvedConfig.Admission.Effort != nil) {
			return nil, errors.New("finite or delegated runs retain their original authority; interactive grants cannot replace it")
		}
		for _, scope := range policy.Scopes {
			if !scopecatalog.Resolve(run.OwnerScopes, scope) {
				return nil, errors.New("grant exceeds retained run admission ceiling")
			}
		}
		if run.OwnerExpiresAt != nil && expires.After(*run.OwnerExpiresAt) {
			expires = *run.OwnerExpiresAt
		}
	}
	// Claim the decision durably before attachment. Replays cannot create runs.
	g.State = "approving"
	changed, err := o.authorizations.Replace(ctx, g, "pending")
	if err != nil {
		return nil, err
	}
	if !changed {
		return nil, errors.New("authorization request already decided")
	}
	if run == nil {
		attached, err := o.AttachRun(ctx, AttachRunRequest{HarnessKind: g.HarnessKind, HarnessSession: g.HarnessSession, ProcessID: g.Binding.PID})
		if err != nil {
			return nil, err
		}
		run = attached.Run
		g.RunID = run.ID
	}
	g.Grant = &requestidentity.OperationGrant{ID: g.ID.String(), Policy: g.Policy, Approver: p.Subject, Operations: append([]string(nil), policy.Operations...), Targets: append([]string(nil), g.Targets...), ExpiresAt: expires}
	metadata, err := json.Marshal(g.Grant)
	if err != nil {
		return nil, err
	}
	claims := &identity.Claims{RunID: run.ID, TaskID: run.TaskID, Subject: p.Subject, Scopes: append([]string(nil), policy.Scopes...), ProfileKey: g.HarnessKind, IssuedAt: now.Unix(), ExpiresAt: expires.Unix(), Meta: map[string]string{"operation_grant": string(metadata), "authorization_id": g.ID.String()}}
	payload, err := json.Marshal(claims)
	if err != nil {
		return nil, err
	}
	g.ClaimsJSON = string(payload)
	token, err := identity.GenerateToken(claims, o.identitySecret)
	if err != nil {
		return nil, err
	}
	g.TokenHash = identity.HashToken(token)
	g.State = "approved"
	changed, err = o.authorizations.Replace(ctx, g, "approving")
	if err != nil {
		return nil, err
	}
	if !changed {
		return nil, errors.New("authorization decision could not be committed")
	}
	return g, nil
}

func (o *Orchestrator) RevokeAuthorization(ctx context.Context, id uuid.UUID) (*domain.AuthorizationGrant, error) {
	p, err := authn.RequireHuman(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := authn.RequireCapability(ctx, "agent-manager:write"); err != nil {
		return nil, err
	}
	g, err := o.GetAuthorization(ctx, id)
	if err != nil {
		return nil, err
	}
	if g.Grant == nil || g.Grant.Approver != p.Subject {
		return nil, errors.New("only the grant approver may revoke this authorization")
	}
	if g.State == "revoked" {
		return g, nil
	}
	if g.State != "approved" {
		return nil, errors.New("authorization is not approved")
	}
	now := o.now().UTC()
	g.RevokedAt = &now
	g.RevokedBy = p.Subject
	g.State = "revoked"
	changed, err := o.authorizations.Replace(ctx, g, "approved")
	if err != nil {
		return nil, err
	}
	if !changed {
		return nil, errors.New("authorization changed concurrently")
	}
	return g, nil
}

func (o *Orchestrator) verifyAuthorization(ctx context.Context, token string, claims *identity.Claims) (*IdentityVerifyResult, error) {
	id, err := uuid.Parse(claims.Meta["authorization_id"])
	if err != nil {
		return &IdentityVerifyResult{Error: "invalid authorization identifier"}, nil
	}
	g, err := o.GetAuthorization(ctx, id)
	if err != nil {
		return nil, err
	}
	if g.State != "approved" || g.Grant == nil || !o.now().Before(g.Grant.ExpiresAt) || g.RunID != claims.RunID || g.TokenHash != identity.HashToken(token) {
		return &IdentityVerifyResult{Error: "authorization is expired, revoked or inactive"}, nil
	}
	live, err := cliutil.ObserveProcessBinding(g.Binding.PID)
	if err != nil || live != g.Binding {
		return &IdentityVerifyResult{Error: "authorization session has ended"}, nil
	}
	run, err := o.GetRun(ctx, g.RunID)
	if err != nil {
		return nil, err
	}
	if run == nil || run.Status.IsTerminal() || run.IdentityTokenRevokedAt != nil {
		return &IdentityVerifyResult{Error: "authorization run is no longer active"}, nil
	}
	return &IdentityVerifyResult{Valid: true, Claims: claims, RunStatus: run.Status}, nil
}

// ResolveAuthorizationCredential is called only after Unix peer verification.
// Newest non-pending consent wins; revocation never resurrects an older grant.
func (o *Orchestrator) ResolveAuthorizationCredential(ctx context.Context, pid int) (string, bool, error) {
	for root, hops := pid, 0; root > 1 && hops < 256; hops++ {
		grants, err := o.ListAuthorizations(ctx, root)
		if err != nil {
			return "", true, err
		}
		for _, g := range grants {
			if g.State == "pending" || g.State == "approving" || !cliutil.ProcessDescendsFrom(pid, g.Binding) {
				continue
			}
			if g.State != "approved" {
				return "", true, errors.New("authorization revoked")
			}
			var claims identity.Claims
			if err := json.Unmarshal([]byte(g.ClaimsJSON), &claims); err != nil {
				return "", true, err
			}
			token, err := identity.GenerateToken(&claims, o.identitySecret)
			if err != nil {
				return "", true, err
			}
			verified, err := o.VerifyIdentityToken(ctx, token)
			if err != nil {
				return "", true, err
			}
			if !verified.Valid {
				return "", true, fmt.Errorf("authorization refused: %s", verified.Error)
			}
			return token, true, nil
		}
		root = cliutil.ProcessParent(root)
	}
	return "", false, nil
}
