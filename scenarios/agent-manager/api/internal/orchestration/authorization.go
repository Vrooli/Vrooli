// Responsibility: bind independently approved grants to live sessions and enforce their durable authority ceilings.
package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	requestidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/scopecatalog"
	"github.com/vrooli/cli-core/cliutil"
)

type AuthorizationRequest struct {
	RunID          uuid.UUID `json:"run_id"`
	PID            int       `json:"pid"`
	HarnessKind    string    `json:"harness_kind"`
	HarnessSession string    `json:"harness_session"`
	Policy         string    `json:"policy"`
	Targets        []string  `json:"targets"`
	// Rules asks for several reviewed policies in one consent, each with its
	// own targets. It replaces Policy/Targets; the two forms are exclusive.
	Rules           []domain.AuthorizationRule `json:"rules,omitempty"`
	DurationSeconds int64                      `json:"duration_seconds"`
}

// maxAuthorizationRules bounds one consent to what an owner can review.
const maxAuthorizationRules = 8

// normalizedRules validates the requested rules against reviewed policies.
func (req AuthorizationRequest) normalizedRules() ([]domain.AuthorizationRule, error) {
	rules := req.Rules
	if len(rules) == 0 {
		rules = []domain.AuthorizationRule{{Policy: req.Policy, Targets: req.Targets}}
	} else if req.Policy != "" || len(req.Targets) != 0 {
		return nil, domain.NewValidationError("rules", "use either rules or policy and targets, not both")
	}
	if len(rules) > maxAuthorizationRules {
		return nil, domain.NewValidationError("rules", "at most 8 policies per consent")
	}
	seen := map[string]bool{}
	out := make([]domain.AuthorizationRule, 0, len(rules))
	for _, rule := range rules {
		policy, ok := authn.InteractivePolicy(rule.Policy)
		if !ok {
			return nil, domain.NewValidationError("policy", "unknown reviewed policy")
		}
		if seen[rule.Policy] {
			return nil, domain.NewValidationError("rules", "each policy may appear once; list all of its targets together")
		}
		seen[rule.Policy] = true
		if len(rule.Targets) == 0 || len(rule.Targets) > 64 {
			return nil, domain.NewValidationError("targets", "one to 64 explicit targets required")
		}
		for _, target := range rule.Targets {
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
		out = append(out, domain.AuthorizationRule{Policy: rule.Policy, Targets: append([]string(nil), rule.Targets...)})
	}
	return out, nil
}

// ruleSummary is the display policy name and target union of a rule set.
func ruleSummary(rules []domain.AuthorizationRule) (string, []string) {
	names := make([]string, 0, len(rules))
	var targets []string
	for _, rule := range rules {
		names = append(names, rule.Policy)
		for _, target := range rule.Targets {
			if !slices.Contains(targets, target) {
				targets = append(targets, target)
			}
		}
	}
	return strings.Join(names, "+"), targets
}

// AuthorizationReviewWindow gives an asynchronous operator time to review.
// It grants no authority and is independent of the permission lifetime,
// which begins at approval with fresh human proof and a live session binding.
const AuthorizationReviewWindow = 7 * 24 * time.Hour

// MaxAuthorizationLifetime bounds an approved interactive grant. Grants stay
// narrow by target and end early when the session changes or the run ends.
const MaxAuthorizationLifetime = 7 * 24 * time.Hour

// MaxStandingGrantLifetime bounds an operator-approved standing grant. It is
// deliberately longer than the interactive cap: a standing grant exists for
// recurring unattended work (heartbeats), so too short a TTL silently
// reintroduces the heartbeat breakage this effort fixes. 30 days bounded —
// revocable at any time and concurrency-capped — is the reliability/exposure
// balance. It governs ONLY standing grants; interactive grants keep the 7-day cap.
const MaxStandingGrantLifetime = 30 * 24 * time.Hour

func WithAuthorizations(repo domain.AuthorizationRepository) Option {
	return func(o *Orchestrator) { o.authorizations = repo }
}

// WithStandingGrants supplies the durable, subject-keyed standing-grant store so
// run admission can attribute a service-standing run to the operator who
// approved the grant. The middleware already authorized the operation through
// the same store; this resolves the accountable owner for the created run.
func WithStandingGrants(repo domain.StandingGrantRepository) Option {
	return func(o *Orchestrator) { o.standingGrants = repo }
}

// standingGrantRunCreateTarget mirrors handlers.RunCreationTarget. Create-run
// names no specific run id, so a run-creation grant covers the domain-wide
// agent-manager object. It is duplicated (not imported) to avoid an
// orchestration→handlers import cycle; both must stay in step.
const standingGrantRunCreateTarget = "agent-manager:*"

// standingGrantOwner returns the accountable owner (the grant's approver) of the
// first active standing grant for subject that authorizes run.create, or empty
// when none applies. It never authorizes — the middleware already did — it only
// attributes; limits are enforced at the gate, not re-checked here.
func (o *Orchestrator) standingGrantOwner(ctx context.Context, subject string) (string, error) {
	if o.standingGrants == nil || subject == "" {
		return "", nil
	}
	grants, err := o.standingGrants.ActiveForSubject(ctx, subject, o.now())
	if err != nil {
		return "", err
	}
	for _, g := range grants {
		if g.Grant != nil && g.Grant.Allows("run.create", standingGrantRunCreateTarget, o.now()) {
			return g.Grant.Approver, nil
		}
	}
	return "", nil
}

// StandingGrantRequest is one operator decision to grant a subject bounded,
// durable run-creation authority. Approver is set by the handler from the
// verified human — never from the request body — so a caller cannot name
// someone else as the accountable owner.
type StandingGrantRequest struct {
	Subject   string
	Approver  string
	Limits    authn.StandingGrantLimits
	SessionID string
	TTL       time.Duration
}

// CreateStandingGrant writes an operator-approved run-creation standing grant.
// It never decides who may call it — the handler enforces the verified
// independent-human gate; this records the authority that gate authorized. The
// operation and target are fixed to run.create on agent-manager so this endpoint
// cannot mint authority for any other operation.
func (o *Orchestrator) CreateStandingGrant(ctx context.Context, req StandingGrantRequest) (*domain.StandingGrant, error) {
	if o.standingGrants == nil {
		return nil, errors.New("standing grant store is not configured")
	}
	subject := strings.TrimSpace(req.Subject)
	approver := strings.TrimSpace(req.Approver)
	if subject == "" || approver == "" {
		return nil, domain.NewValidationError("subject", "standing grant requires a subject and an approver")
	}
	if req.TTL <= 0 || req.TTL > MaxStandingGrantLifetime {
		return nil, domain.NewValidationError("ttl", "standing grant lifetime must be positive and within the maximum standing-grant lifetime")
	}
	if req.Limits.MaxConcurrentRuns < 0 || req.Limits.MaxRunBudgetUSD < 0 {
		return nil, domain.NewValidationError("limits", "standing grant limits cannot be negative")
	}
	now := o.now()
	expires := now.Add(req.TTL)
	id := uuid.NewString()
	g := &domain.StandingGrant{
		ID:        id,
		Subject:   subject,
		SessionID: strings.TrimSpace(req.SessionID),
		Grant: &requestidentity.OperationGrant{
			ID:         id,
			Policy:     "run-creation",
			Approver:   approver,
			Operations: []string{"run.create"},
			Targets:    []string{standingGrantRunCreateTarget},
			ExpiresAt:  expires,
		},
		Limits:     req.Limits,
		State:      domain.StandingGrantApproved,
		ApprovedBy: approver,
		ApprovedAt: now,
		ExpiresAt:  expires,
	}
	if err := o.standingGrants.Put(ctx, g); err != nil {
		return nil, err
	}
	return g, nil
}

// RevokeStandingGrant revokes a standing grant by id so the next lookup denies
// it. Like create, the caller authorization (verified independent human) is the
// handler's responsibility.
func (o *Orchestrator) RevokeStandingGrant(ctx context.Context, id string) (bool, error) {
	if o.standingGrants == nil {
		return false, errors.New("standing grant store is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return false, domain.NewValidationError("id", "standing grant id is required")
	}
	return o.standingGrants.Revoke(ctx, id, o.now())
}

// ListStandingGrants returns recorded standing grants (newest first) for the
// operator admin surface. Like create/revoke, the caller gate is the handler's
// job; this is a read of the durable record.
func (o *Orchestrator) ListStandingGrants(ctx context.Context, limit int) ([]*domain.StandingGrant, error) {
	if o.standingGrants == nil {
		return nil, errors.New("standing grant store is not configured")
	}
	return o.standingGrants.List(ctx, limit)
}

func (o *Orchestrator) RequestAuthorization(ctx context.Context, req AuthorizationRequest) (*domain.AuthorizationGrant, error) {
	if o.authorizations == nil || len(o.identitySecret) == 0 {
		return nil, errors.New("authorization authority is not configured")
	}
	rules, err := req.normalizedRules()
	if err != nil {
		return nil, err
	}
	if req.DurationSeconds <= 0 || req.DurationSeconds > int64(MaxAuthorizationLifetime/time.Second) {
		return nil, domain.NewValidationError("duration_seconds", "must be between 1 second and 7 days")
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
	// Omitted run IDs cannot disguise retained managed authority as a fresh session.
	if req.RunID == uuid.Nil {
		req.RunID, err = o.managedAuthorizationRun(ctx, binding)
		if err != nil {
			return nil, err
		}
	}
	previous, err := o.authorizations.List(ctx, req.PID)
	if err != nil {
		return nil, err
	}
	// An identical pending request is returned unchanged. A different one is
	// superseded by the new request: a session asks again for everything it
	// needs instead of waiting for an outdated request to be decided.
	var outdated []*domain.AuthorizationGrant
	for _, grant := range previous {
		if grant.Binding != binding || grant.EffectiveState(o.now()) != domain.AuthorizationPending {
			continue
		}
		if grant.RunID == req.RunID && grant.DurationSeconds == req.DurationSeconds && grant.HarnessKind == req.HarnessKind && grant.HarnessSession == req.HarnessSession && slices.EqualFunc(grant.PolicyRules(), rules, func(a, b domain.AuthorizationRule) bool {
			return a.Policy == b.Policy && slices.Equal(a.Targets, b.Targets)
		}) {
			return grant, nil
		}
		outdated = append(outdated, grant)
	}
	now := o.now().UTC()
	policyName, targets := ruleSummary(rules)
	grant := &domain.AuthorizationGrant{ID: uuid.New(), RunID: req.RunID, State: domain.AuthorizationPending, Policy: policyName, Targets: targets, DurationSeconds: req.DurationSeconds, Binding: binding, HarnessKind: req.HarnessKind, HarnessSession: req.HarnessSession, CreatedAt: now, RequestExpiresAt: now.Add(AuthorizationReviewWindow)}
	if len(rules) > 1 {
		grant.Rules = rules
	}
	if err := o.authorizations.Create(ctx, grant); err != nil {
		return nil, err
	}
	for _, old := range outdated {
		old.State = domain.AuthorizationSuperseded
		old.SupersededBy = &grant.ID
		// Losing the race to an approval is fine: the approved grant stands.
		if _, err := o.authorizations.Replace(ctx, old, domain.AuthorizationPending); err != nil {
			return nil, err
		}
	}
	return grant, nil
}

// managedAuthorizationRun detects a current execution without trusting a caller-supplied run ID.
// approvalPolicy is the union of the reviewed policies a request names: the
// scopes the approver and run ceilings must cover, and the display operations.
// Allows still pairs each operation with its own targets through the rules.
func approvalPolicy(g *domain.AuthorizationGrant) (authn.Policy, error) {
	var union authn.Policy
	for _, rule := range g.PolicyRules() {
		policy, ok := authn.InteractivePolicy(rule.Policy)
		if !ok {
			return authn.Policy{}, errors.New("reviewed policy no longer available")
		}
		for _, field := range []struct{ src, dst *[]string }{{&policy.Scopes, &union.Scopes}, {&policy.Operations, &union.Operations}, {&policy.Domains, &union.Domains}} {
			for _, value := range *field.src {
				if !slices.Contains(*field.dst, value) {
					*field.dst = append(*field.dst, value)
				}
			}
		}
	}
	return union, nil
}

// MissingAuthorizationCapabilities reports the current human capability gap
// without approving, attaching, or minting authority. Browser preflight and the
// final owner check use the same reviewed policy; possession of these scopes
// does not replace independent-human, lifetime, process or lineage checks.
func MissingAuthorizationCapabilities(ctx context.Context, g *domain.AuthorizationGrant) ([]string, error) {
	policy, err := approvalPolicy(g)
	if err != nil {
		return nil, err
	}
	return missingAuthorizationCapabilities(ctx, policy), nil
}

func missingAuthorizationCapabilities(ctx context.Context, policy authn.Policy) []string {
	var missing []string
	for _, scope := range append([]string{"agent-manager:write"}, policy.Scopes...) {
		if _, err := authn.RequireCapability(ctx, scope); err != nil && !slices.Contains(missing, scope) {
			missing = append(missing, scope)
		}
	}
	return missing
}

func (o *Orchestrator) managedAuthorizationRun(ctx context.Context, binding cliutil.ProcessBinding) (uuid.UUID, error) {
	managedID := uuid.Nil
	for offset := 0; ; offset += 100 {
		runs, err := o.runs.List(ctx, repository.RunListFilter{ListFilter: repository.ListFilter{Limit: 100, Offset: offset}})
		if err != nil {
			return uuid.Nil, err
		}
		for _, run := range runs {
			if run == nil || run.Status.IsTerminal() || run.RunnerPID <= 1 || run.ExecutionMode.Normalized() == domain.ExecutionModeAttached {
				continue
			}
			root, err := cliutil.ObserveProcessBinding(run.RunnerPID)
			if err != nil {
				continue
			}
			if cliutil.ProcessDescendsFrom(binding.PID, root) || cliutil.ProcessDescendsFrom(run.RunnerPID, binding) {
				if managedID != uuid.Nil && managedID != run.ID {
					return uuid.Nil, errors.New("process binding contains multiple managed runs; select an exact run")
				}
				managedID = run.ID
			}
		}
		if len(runs) < 100 {
			break
		}
	}
	return managedID, nil
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
	p, err := authn.RequireIndependentHuman(ctx)
	if err != nil {
		return nil, err
	}
	// A shared OS account or app-local token cannot attest independent consent.
	if p.ExpiresAt.IsZero() || !p.ExpiresAt.After(o.now()) {
		return nil, errors.New("unexpired independent human proof required")
	}
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
	switch g.EffectiveState(o.now()) {
	case domain.AuthorizationPending:
	case domain.AuthorizationExpired:
		return nil, domain.NewValidationError("authorization", "request expired at "+g.RequestExpiresAt.UTC().Format(time.RFC3339)+"; ask the agent to prepare a new request")
	case domain.AuthorizationSuperseded:
		return nil, domain.NewValidationError("authorization", "request was replaced by a newer request ("+g.SupersededBy.String()+"); approve that one instead")
	default:
		return nil, domain.NewValidationError("authorization", "request is already "+g.State)
	}
	policy, err := approvalPolicy(g)
	if err != nil {
		return nil, err
	}
	if missing := missingAuthorizationCapabilities(ctx, policy); len(missing) > 0 {
		return nil, fmt.Errorf("%w: approving this request requires %s; update the verified owner's capability policy, then review this request again", authn.ErrCapability, strings.Join(missing, ", "))
	}
	live, err := cliutil.ObserveProcessBinding(g.Binding.PID)
	if err != nil || live != g.Binding {
		return nil, errors.New("authorization session has ended or changed")
	}
	managedID, err := o.managedAuthorizationRun(ctx, live)
	if err != nil {
		return nil, err
	}
	if managedID != uuid.Nil && managedID != g.RunID {
		return nil, errors.New("managed run appeared or changed after consent preparation; prepare a new request")
	}
	// The approver's login proves who consented; the consent then lasts the
	// lifetime they approved, not the remainder of their browser session.
	now, expires := o.now().UTC(), o.now().UTC().Add(time.Duration(g.DurationSeconds)*time.Second)
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
		if run.ExecutionMode.Normalized() != domain.ExecutionModeAttached {
			if run.ResolvedConfig == nil || run.ResolvedConfig.Admission == nil || run.ResolvedConfig.Admission.FrozenIdentityScopes == nil {
				return nil, errors.New("historical run has no frozen identity ceiling; it cannot adopt a broader interactive grant")
			}
			for _, scope := range policy.Scopes {
				if !scopecatalog.Resolve(*run.ResolvedConfig.Admission.FrozenIdentityScopes, scope) {
					return nil, errors.New("grant exceeds frozen identity ceiling")
				}
			}
		}
		for _, scope := range policy.Scopes {
			if !scopecatalog.Resolve(run.OwnerScopes, scope) {
				return nil, errors.New("grant exceeds retained run admission ceiling")
			}
		}
		if run.RequestedScopes != nil {
			for _, scope := range policy.Scopes {
				if !scopecatalog.Resolve(run.RequestedScopes, scope) {
					return nil, errors.New("grant exceeds retained requested scope ceiling")
				}
			}
		}
		if run.AgentProfileID != nil {
			if o.profiles == nil {
				return nil, errors.New("profile scope ceiling unavailable")
			}
			profile, err := o.profiles.Get(ctx, *run.AgentProfileID)
			if err != nil {
				return nil, err
			}
			if profile == nil {
				return nil, errors.New("profile scope ceiling unavailable")
			}
			for _, scope := range policy.Scopes {
				if !scopecatalog.Resolve(profile.IdentityScopeCeiling(), scope) {
					return nil, errors.New("grant exceeds declared profile scope ceiling")
				}
			}
		}
		if run.OwnerExpiresAt != nil && expires.After(*run.OwnerExpiresAt) {
			expires = *run.OwnerExpiresAt
		}
	}
	if !expires.After(now) {
		return nil, errors.New("retained run owner proof expired")
	}
	// Claim the decision durably before attachment. Replays cannot create runs.
	g.State = "approving"
	g.ApprovedAt = &now
	g.ApproverSource = p.Source
	g.ApproverIssuer = p.Issuer
	g.ApproverAudience = p.Audience
	g.ApproverProofExpiresAt = p.ExpiresAt
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
	if len(g.Rules) > 0 {
		for _, rule := range g.Rules {
			reviewed, _ := authn.InteractivePolicy(rule.Policy)
			g.Grant.Rules = append(g.Grant.Rules, requestidentity.OperationRule{Policy: rule.Policy, Operations: append([]string(nil), reviewed.Operations...), Targets: append([]string(nil), rule.Targets...)})
		}
	}
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
	p, err := authn.RequireIndependentHuman(ctx)
	if err != nil {
		return nil, err
	}
	if p.ExpiresAt.IsZero() || !p.ExpiresAt.After(o.now()) {
		return nil, errors.New("unexpired independent human proof required")
	}
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
	if g.State == "revoked" {
		return &IdentityVerifyResult{Error: "revoked"}, nil
	}
	if g.Grant != nil && !o.now().Before(g.Grant.ExpiresAt) {
		return &IdentityVerifyResult{Error: "expired"}, nil
	}
	if g.State != "approved" || g.Grant == nil || g.RunID != claims.RunID || g.TokenHash != identity.HashToken(token) {
		return &IdentityVerifyResult{Error: "invalid"}, nil
	}

	live, err := cliutil.ObserveProcessBinding(g.Binding.PID)
	if err != nil || live != g.Binding {
		return &IdentityVerifyResult{Error: "session_changed"}, nil
	}
	run, err := o.GetRun(ctx, g.RunID)
	if err != nil {
		return nil, err
	}
	if run == nil || run.Status.IsTerminal() || run.IdentityTokenRevokedAt != nil {
		return &IdentityVerifyResult{Error: "run_inactive"}, nil
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
			state := g.EffectiveState(o.now())
			if g.State == domain.AuthorizationPending || state == domain.AuthorizationApproving || state == domain.AuthorizationSuperseded {
				// Requests that never became consent carry no authority.
				continue
			}
			if !cliutil.ProcessDescendsFrom(pid, g.Binding) {
				continue
			}
			if state == domain.AuthorizationExpired {
				// Lapsed consent returns the session to the identity it held
				// before the grant. It never resurrects an older grant.
				return "", false, nil
			}
			if g.State != "approved" {
				return "", true, &domain.AuthorizationError{Code: "revoked"}
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
				return "", true, &domain.AuthorizationError{Code: verified.Error}
			}
			return token, true, nil
		}
		root = cliutil.ProcessParent(root)
	}
	return "", false, nil
}
