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

// MaxSessionGrantLifetime hard-caps a human-session delegation grant (Phase 5).
// A tunnel-UI session grant is bounded to the session, but a forgotten-open
// session must never confer indefinite authority, so its lifetime is the lesser
// of the requested session lifetime and this cap. It is deliberately short: a
// session grant lets work spawned inside one operator session inherit that
// operator's authority, which should expire soon after the operator steps away.
const MaxSessionGrantLifetime = 4 * time.Hour

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

// CreateSessionGrant mints a Phase-5 human-session delegation grant: a
// short-lived run-creation grant keyed to a verified human subject AND a
// web-console session id, so work spawned inside that operator session inherits
// the operator's authority, bounded and attributable. The caller MUST have
// already verified the human (a verified-independent-human spawn) — this records
// the authority that gate authorized; it never decides who may call it. The
// lifetime is min(requested, MaxSessionGrantLifetime): a forgotten-open session
// cannot confer indefinite authority. It is revoked when the session closes.
func (o *Orchestrator) CreateSessionGrant(ctx context.Context, humanSubject, sessionID string, requestedTTL time.Duration) (*domain.StandingGrant, error) {
	return o.createSessionGrant(ctx, humanSubject, sessionID, requestedTTL, time.Time{})
}

// createSessionGrant can additionally cap a session grant at an absolute
// proof expiry. Keep that ceiling private so callers cannot turn it into a
// general-purpose grant policy surface.
func (o *Orchestrator) createSessionGrant(ctx context.Context, humanSubject, sessionID string, requestedTTL time.Duration, proofExpiresAt time.Time) (*domain.StandingGrant, error) {
	if o.standingGrants == nil {
		return nil, errors.New("standing grant store is not configured")
	}
	humanSubject = strings.TrimSpace(humanSubject)
	sessionID = strings.TrimSpace(sessionID)
	if humanSubject == "" || sessionID == "" {
		return nil, domain.NewValidationError("session_grant", "session grant requires a human subject and a session id")
	}
	ttl := requestedTTL
	if ttl <= 0 || ttl > MaxSessionGrantLifetime {
		ttl = MaxSessionGrantLifetime
	}
	now := o.now()
	expires := now.Add(ttl)
	if !proofExpiresAt.IsZero() && expires.After(proofExpiresAt) {
		expires = proofExpiresAt
	}
	if !expires.After(now) {
		return nil, domain.NewValidationError("session_grant", "session grant proof expiry must be in the future")
	}
	id := uuid.NewString()
	g := &domain.StandingGrant{
		ID:        id,
		Subject:   humanSubject,
		SessionID: sessionID,
		Grant: &requestidentity.OperationGrant{
			ID:         id,
			Policy:     "run-creation",
			Approver:   humanSubject,
			Operations: []string{"run.create"},
			Targets:    []string{standingGrantRunCreateTarget},
			ExpiresAt:  expires,
		},
		State:      domain.StandingGrantApproved,
		ApprovedBy: humanSubject,
		ApprovedAt: now,
		ExpiresAt:  expires,
	}
	if err := o.standingGrants.Put(ctx, g); err != nil {
		return nil, err
	}
	return g, nil
}

// RevokeSessionGrants revokes every active session-scoped grant for a
// (human subject, session id) pair so a closed session denies on the next
// lookup. It is idempotent; grants for other sessions or subjects are untouched.
func (o *Orchestrator) RevokeSessionGrants(ctx context.Context, humanSubject, sessionID string) error {
	if o.standingGrants == nil {
		return nil
	}
	humanSubject = strings.TrimSpace(humanSubject)
	sessionID = strings.TrimSpace(sessionID)
	if humanSubject == "" || sessionID == "" {
		return nil
	}
	grants, err := o.standingGrants.ActiveForSubject(ctx, humanSubject, o.now())
	if err != nil {
		return err
	}
	for _, g := range grants {
		if g.SessionID == sessionID {
			if _, err := o.standingGrants.Revoke(ctx, g.ID, o.now()); err != nil {
				return err
			}
		}
	}
	return nil
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

// ListStandingGrantsForApprover returns the bounded history attributed to one
// verified owner. The repository applies the owner filter before its limit.
func (o *Orchestrator) ListStandingGrantsForApprover(ctx context.Context, approver string, limit int) ([]*domain.StandingGrant, error) {
	if o.standingGrants == nil {
		return nil, errors.New("standing grant store is not configured")
	}
	if strings.TrimSpace(approver) == "" {
		return nil, errors.New("standing grant approver is required")
	}
	return o.standingGrants.ListForApprover(ctx, approver, limit)
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
			if grant.NotificationPublishedAt == nil {
				o.NotifyAuthorizationRequested(ctx, grant)
			}
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
	// Push the operator to the review UI for this pending interactive request
	// (gap G3). Best-effort: the durable request already stands, so a publish
	// failure never fails the request. It issues no authority — approval still
	// requires the operator's live verified login via ApproveAuthorization.
	o.NotifyAuthorizationRequested(ctx, grant)
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
		runs, err := o.runs.List(ctx, repository.RunListFilter{ListFilter: repository.ListFilter{Limit: 100, Offset: offset}, NonTerminalOnly: true})
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
		if run.ParentRunID != nil {
			return nil, errors.New("delegated runs retain their original authority; interactive grants cannot replace it")
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
	run, err := o.GetStoredRun(ctx, g.RunID)
	if err != nil {
		return nil, err
	}
	if run == nil || run.Status.IsTerminal() || run.IdentityTokenRevokedAt != nil {
		return &IdentityVerifyResult{Error: "run_inactive"}, nil
	}
	return &IdentityVerifyResult{Valid: true, Claims: claims, RunStatus: run.Status}, nil
}

// VerifyIdentityPair verifies two independently issued credentials without
// combining their scopes. The baseline proves the ordinary run identity; the
// operation token proves only its own currently approved operation grant.
func (o *Orchestrator) VerifyIdentityPair(ctx context.Context, baselineToken, operationToken string) (*IdentityPairVerifyResult, error) {
	baselineToken = strings.TrimSpace(baselineToken)
	operationToken = strings.TrimSpace(operationToken)
	if baselineToken == "" || operationToken == "" || baselineToken == operationToken {
		return &IdentityPairVerifyResult{Error: "invalid_pair"}, nil
	}
	baseline, err := o.VerifyIdentityToken(ctx, baselineToken)
	if err != nil {
		return nil, err
	}
	if baseline == nil || !baseline.Valid || baseline.Claims == nil || baseline.Claims.Purpose != "" || baseline.Claims.Meta["authorization_id"] != "" {
		return &IdentityPairVerifyResult{Error: "invalid_pair"}, nil
	}
	operation, err := o.VerifyIdentityToken(ctx, operationToken)
	if err != nil {
		return nil, err
	}
	if operation == nil || !operation.Valid || operation.Claims == nil || operation.Claims.Meta["authorization_id"] == "" {
		return &IdentityPairVerifyResult{Error: "invalid_pair"}, nil
	}

	baselineRun, err := o.runs.GetByTokenHash(ctx, identity.HashToken(baselineToken))
	if err != nil {
		return nil, err
	}
	operationRun, err := o.GetStoredRun(ctx, operation.Claims.RunID)
	if err != nil {
		return nil, err
	}
	if baselineRun == nil || operationRun == nil || baselineRun.ID != baseline.Claims.RunID || operationRun.ID != operation.Claims.RunID || baselineRun.Status.IsTerminal() || operationRun.Status.IsTerminal() || baselineRun.IdentityTokenRevokedAt != nil || operationRun.IdentityTokenRevokedAt != nil {
		return &IdentityPairVerifyResult{Error: "invalid_pair"}, nil
	}
	owner := strings.TrimSpace(baselineRun.OwnerSubject)
	if owner == "" || owner != strings.TrimSpace(operationRun.OwnerSubject) || baselineRun.RunnerPID <= 0 {
		return &IdentityPairVerifyResult{Error: "invalid_pair"}, nil
	}

	grantID, err := uuid.Parse(operation.Claims.Meta["authorization_id"])
	if err != nil {
		return &IdentityPairVerifyResult{Error: "invalid_pair"}, nil
	}
	grant, err := o.GetAuthorization(ctx, grantID)
	if err != nil {
		return nil, err
	}
	// VerifyIdentityToken already checked the current full process binding and
	// grant state. Repeat the owner and PID relation here against repository
	// records so a valid but unrelated grant cannot be paired with this run.
	if grant == nil || grant.Grant == nil || grant.RunID != operationRun.ID || grant.Binding.PID != baselineRun.RunnerPID || strings.TrimSpace(grant.Grant.Approver) != owner {
		return &IdentityPairVerifyResult{Error: "invalid_pair"}, nil
	}
	// A descendant may inherit the broker's reachability but cannot borrow a
	// parent grant. A child is pairable only with its own run's authorization.
	if baselineRun.ParentRunID != nil && grant.RunID != baselineRun.ID {
		return &IdentityPairVerifyResult{Error: "invalid_pair"}, nil
	}
	return &IdentityPairVerifyResult{Valid: true, Operation: operation}, nil
}

// ResolveAuthorizationCredentials returns a legacy grant-only token, an
// ownerless attached run's grant-only compatibility token, or an owner-attributed
// baseline plus a separately verified operation token. An owner-attributed
// baseline is never replaced by a grant, and grant failure does not disable its
// ordinary identity use.
type authorizationCredentialPurpose uint8

const (
	authorizationCredentialPurposeDefault authorizationCredentialPurpose = iota
	authorizationCredentialPurposeOrdinaryIdentity
)

func (o *Orchestrator) ResolveAuthorizationCredentials(ctx context.Context, pid int, baselineToken string) (AuthorizationCredentials, error) {
	return o.resolveAuthorizationCredentials(ctx, pid, baselineToken, authorizationCredentialPurposeDefault)
}

// resolveOrdinaryRunIdentityCredential is used only by the local identity
// reporting command. It shares the exact peer, managed-run, expiry and renewal
// checks with normal credential resolution but never returns an operation grant.
func (o *Orchestrator) resolveOrdinaryRunIdentityCredential(ctx context.Context, pid int, baselineToken string) (AuthorizationCredentials, error) {
	credentials, err := o.resolveAuthorizationCredentials(ctx, pid, baselineToken, authorizationCredentialPurposeOrdinaryIdentity)
	if err != nil {
		return credentials, err
	}
	if credentials.BaselineToken == "" || credentials.LegacyToken != "" || credentials.OperationToken != "" {
		return ordinaryIdentityCredentialFailure(nil)
	}
	return credentials, nil
}

func ordinaryIdentityCredentialFailure(cause error) (AuthorizationCredentials, error) {
	return AuthorizationCredentials{OperationRefusal: "identity_refresh_unavailable"}, identityRefreshUnavailable(cause)
}

func (o *Orchestrator) resolveAuthorizationCredentials(ctx context.Context, pid int, baselineToken string, purpose authorizationCredentialPurpose) (AuthorizationCredentials, error) {
	baselineToken = strings.TrimSpace(baselineToken)
	if purpose == authorizationCredentialPurposeOrdinaryIdentity && baselineToken == "" {
		return ordinaryIdentityCredentialFailure(nil)
	}
	// Resolve managed-run ownership from the verified broker peer even when no
	// baseline header was supplied. A managed child must not be able to omit
	// that header (or let it expire) and fall back to an ancestor's grant-only
	// credential.
	var managedRun *domain.Run
	if pid > 1 {
		binding, err := cliutil.ObserveProcessBinding(pid)
		if err != nil {
			if purpose == authorizationCredentialPurposeOrdinaryIdentity {
				return ordinaryIdentityCredentialFailure(err)
			}
			if baselineToken != "" {
				return AuthorizationCredentials{BaselineToken: baselineToken, OperationRefusal: "process_binding_unavailable"}, nil
			}
			return AuthorizationCredentials{}, err
		}
		managedID, err := o.managedAuthorizationRun(ctx, binding)
		if err != nil {
			if purpose == authorizationCredentialPurposeOrdinaryIdentity {
				return ordinaryIdentityCredentialFailure(err)
			}
			if baselineToken != "" {
				return AuthorizationCredentials{BaselineToken: baselineToken, OperationRefusal: "managed_run_unavailable"}, nil
			}
			return AuthorizationCredentials{}, err
		}
		if managedID != uuid.Nil {
			managedRun, err = o.GetStoredRun(ctx, managedID)
			if err != nil {
				if purpose == authorizationCredentialPurposeOrdinaryIdentity {
					return ordinaryIdentityCredentialFailure(err)
				}
				if baselineToken != "" {
					return AuthorizationCredentials{BaselineToken: baselineToken, OperationRefusal: "managed_run_unavailable"}, nil
				}
				return AuthorizationCredentials{}, err
			}
		}
	}
	var requiredRunID uuid.UUID
	if managedRun != nil && managedRun.ParentRunID != nil {
		requiredRunID = managedRun.ID
	}
	if purpose == authorizationCredentialPurposeOrdinaryIdentity {
		claims, err := identity.VerifyTokenSignature(baselineToken, o.identitySecret)
		if err != nil || claims == nil || claims.RunID == uuid.Nil || claims.Purpose != "" || claims.Meta["authorization_id"] != "" {
			return ordinaryIdentityCredentialFailure(err)
		}
		if managedRun != nil && managedRun.ID != claims.RunID {
			return AuthorizationCredentials{OperationRefusal: "baseline_run_mismatch"}, &domain.AuthorizationError{Code: "baseline_run_mismatch"}
		}
	}
	if baselineToken != "" {
		if inheritedClaims, verifyErr := identity.VerifyToken(baselineToken, o.identitySecret); verifyErr == nil && inheritedClaims.Meta["authorization_id"] != "" {
			// Existing grant-only identities remain on the legacy bounded path;
			// they are never reinterpreted as a baseline for a second grant.
			legacy, found, resolveErr := o.resolveAuthorizationCredentialForRun(ctx, pid, requiredRunID)
			if resolveErr != nil {
				return AuthorizationCredentials{OperationRefusal: authorizationRefusalCode(resolveErr)}, resolveErr
			}
			if found && legacy != "" {
				return AuthorizationCredentials{LegacyToken: legacy}, nil
			}
			if requiredRunID != uuid.Nil {
				return AuthorizationCredentials{OperationRefusal: "managed_child_requires_own_grant"}, nil
			}
			return AuthorizationCredentials{LegacyToken: baselineToken}, nil
		}
	}
	if baselineToken == "" {
		legacy, found, resolveErr := o.resolveAuthorizationCredentialForRun(ctx, pid, requiredRunID)
		if resolveErr != nil {
			return AuthorizationCredentials{OperationRefusal: authorizationRefusalCode(resolveErr)}, resolveErr
		}
		if !found {
			return AuthorizationCredentials{}, nil
		}
		return AuthorizationCredentials{LegacyToken: legacy}, nil
	}

	verifiedBaseline, verifyErr := o.VerifyIdentityToken(ctx, baselineToken)
	renewedBaseline := false
	if verifyErr != nil || verifiedBaseline == nil || !verifiedBaseline.Valid || verifiedBaseline.Claims == nil || verifiedBaseline.Claims.Purpose != "" || verifiedBaseline.Claims.Meta["authorization_id"] != "" {
		// Ordinary API verification remains expiry-strict. Only the local
		// credential broker can use a signed expired attached-root token as a
		// renewal anchor after rechecking the exact launch owner/session proof.
		if claims, signatureErr := identity.VerifyTokenSignature(baselineToken, o.identitySecret); signatureErr == nil && requiredRunID == uuid.Nil && claims.ExpiresAt > 0 && claims.ExpiresAt <= o.now().Unix() && claims.Purpose == "" && claims.Meta["authorization_id"] == "" {
			if renewed, ok, renewalErr := o.resolveAttachedBaselineRenewal(ctx, pid, baselineToken, claims); renewalErr != nil {
				if purpose == authorizationCredentialPurposeOrdinaryIdentity {
					return ordinaryIdentityCredentialFailure(renewalErr)
				}
				// Identity refresh is not a prerequisite for an already independent
				// grant. Preserve that established grant-only path if it remains
				// current, but never return the expired baseline as an identity.
				grantCredentials, handled, grantErr := o.resolveExpiredAttachedBaselineGrant(ctx, pid, baselineToken, managedRun)
				if grantErr != nil {
					return grantCredentials, grantErr
				}
				if handled && grantCredentials.LegacyToken != "" {
					return grantCredentials, nil
				}
				return AuthorizationCredentials{OperationRefusal: "identity_refresh_unavailable"}, renewalErr
			} else if ok {
				// Continue through the normal ownerless-root resolution below. It
				// may return this ordinary baseline or the existing independently
				// bound grant-only credential; renewal must not bypass grant lookup.
				baselineToken = renewed
				verifiedBaseline, verifyErr = o.VerifyIdentityToken(ctx, baselineToken)
				if verifyErr != nil || verifiedBaseline == nil || !verifiedBaseline.Valid || verifiedBaseline.Claims == nil || verifiedBaseline.Claims.Purpose != "" || verifiedBaseline.Claims.Meta["authorization_id"] != "" {
					return AuthorizationCredentials{OperationRefusal: "identity_refresh_unavailable"}, identityRefreshUnavailable(verifyErr)
				}
				renewedBaseline = true
			}
		}
		if !renewedBaseline {
			if purpose == authorizationCredentialPurposeOrdinaryIdentity {
				return ordinaryIdentityCredentialFailure(verifyErr)
			}
			// If renewal is unavailable, preserve only the existing independent
			// grant path. Grant-only authorization does not make the expired anchor
			// an API identity.
			credentials, handled, resolveErr := o.resolveExpiredAttachedBaselineGrant(ctx, pid, baselineToken, managedRun)
			if resolveErr != nil {
				return credentials, resolveErr
			}
			if handled {
				return credentials, nil
			}
			return AuthorizationCredentials{BaselineToken: baselineToken, OperationRefusal: "baseline_invalid"}, nil
		}
	}
	if purpose == authorizationCredentialPurposeOrdinaryIdentity {
		if verifiedBaseline == nil || !verifiedBaseline.Valid || verifiedBaseline.Claims == nil || verifiedBaseline.Claims.RunID == uuid.Nil {
			return ordinaryIdentityCredentialFailure(nil)
		}
		if managedRun != nil && managedRun.ID != verifiedBaseline.Claims.RunID {
			return AuthorizationCredentials{OperationRefusal: "baseline_run_mismatch"}, &domain.AuthorizationError{Code: "baseline_run_mismatch"}
		}
		return AuthorizationCredentials{BaselineToken: baselineToken}, nil
	}
	// An ownerless attached baseline carries no delegated scopes and cannot be
	// paired to a human-approved operation grant. Preserve the pre-pair legacy
	// behavior for a valid ownerless root: resolve only the grant independently
	// bound to this peer, and return it grant-only. The resolver's exact managed
	// child restriction still prevents an ownerless child from borrowing its
	// parent's grant.
	if strings.TrimSpace(verifiedBaseline.Claims.Subject) == "" {
		baselineRun, lookupErr := o.runs.GetByTokenHash(ctx, identity.HashToken(baselineToken))
		if lookupErr != nil {
			return AuthorizationCredentials{BaselineToken: baselineToken, OperationRefusal: "baseline_lookup_unavailable"}, nil
		}
		if baselineRun != nil && baselineRun.ID == verifiedBaseline.Claims.RunID && strings.TrimSpace(baselineRun.OwnerSubject) == "" && baselineRun.ExecutionMode.Normalized() == domain.ExecutionModeAttached {
			credentials, handled, resolveErr := o.resolveAttachedBaselineGrant(ctx, pid, baselineRun, managedRun)
			if resolveErr != nil {
				return AuthorizationCredentials{BaselineToken: baselineToken, OperationRefusal: authorizationRefusalCode(resolveErr)}, nil
			}
			if handled && credentials.LegacyToken != "" {
				return credentials, nil
			}
			return AuthorizationCredentials{BaselineToken: baselineToken, OperationRefusal: credentials.OperationRefusal}, nil
		}
	}
	grantToken, found, resolveErr := o.resolveAuthorizationCredentialForRun(ctx, pid, requiredRunID)
	if resolveErr != nil {
		return AuthorizationCredentials{BaselineToken: baselineToken, OperationRefusal: authorizationRefusalCode(resolveErr)}, nil
	}
	if !found || grantToken == "" {
		return AuthorizationCredentials{BaselineToken: baselineToken}, nil
	}
	pair, pairErr := o.VerifyIdentityPair(ctx, baselineToken, grantToken)
	if pairErr != nil || pair == nil || !pair.Valid {
		return AuthorizationCredentials{BaselineToken: baselineToken, OperationRefusal: "pair_unavailable"}, nil
	}
	return AuthorizationCredentials{BaselineToken: baselineToken, OperationToken: grantToken}, nil
}

func (o *Orchestrator) resolveExpiredAttachedBaselineGrant(ctx context.Context, pid int, baselineToken string, peerRun *domain.Run) (AuthorizationCredentials, bool, error) {
	anchorHash := identity.HashToken(baselineToken)
	baseline, err := o.runs.GetByTokenHash(ctx, anchorHash)
	if err != nil {
		return AuthorizationCredentials{OperationRefusal: "baseline_lookup_unavailable"}, true, err
	}
	if baseline == nil {
		if store, ok := o.runs.(repository.AttachedIdentityRenewalRepository); ok {
			baseline, err = store.GetByIdentityRenewalAnchorHash(ctx, anchorHash)
			if err != nil {
				return AuthorizationCredentials{OperationRefusal: "baseline_lookup_unavailable"}, true, err
			}
		}
	}
	if baseline == nil {
		return AuthorizationCredentials{}, false, nil
	}
	return o.resolveAttachedBaselineGrant(ctx, pid, baseline, peerRun)
}

func (o *Orchestrator) resolveAttachedBaselineGrant(ctx context.Context, pid int, baseline, peerRun *domain.Run) (AuthorizationCredentials, bool, error) {
	if baseline == nil || baseline.Status.IsTerminal() || baseline.IdentityTokenRevokedAt != nil || baseline.RunnerPID <= 1 {
		return AuthorizationCredentials{}, false, nil
	}
	if peerRun != nil && peerRun.ID != baseline.ID {
		// A managed process may not present an ancestor/top-level token to
		// recover that ancestor's independent grant.
		return AuthorizationCredentials{OperationRefusal: "baseline_run_mismatch"}, true, nil
	}
	requiredRunID := uuid.Nil
	if baseline.ParentRunID != nil || (peerRun != nil && peerRun.ParentRunID != nil) {
		requiredRunID = baseline.ID
	} else if baseline.ExecutionMode.Normalized() != domain.ExecutionModeAttached {
		return AuthorizationCredentials{}, false, nil
	}
	grantToken, found, err := o.resolveAuthorizationCredentialForRun(ctx, pid, requiredRunID)
	if err != nil {
		return AuthorizationCredentials{OperationRefusal: authorizationRefusalCode(err)}, true, err
	}
	if !found || grantToken == "" {
		return AuthorizationCredentials{OperationRefusal: "grant_unavailable"}, true, nil
	}
	verified, err := o.VerifyIdentityToken(ctx, grantToken)
	if err != nil || verified == nil || !verified.Valid || verified.Claims == nil || verified.Claims.Meta["authorization_id"] == "" {
		return AuthorizationCredentials{OperationRefusal: "grant_unavailable"}, true, nil
	}
	grantID, err := uuid.Parse(verified.Claims.Meta["authorization_id"])
	if err != nil {
		return AuthorizationCredentials{OperationRefusal: "grant_unavailable"}, true, nil
	}
	grant, err := o.GetAuthorization(ctx, grantID)
	if err != nil {
		return AuthorizationCredentials{OperationRefusal: "grant_unavailable"}, true, err
	}
	if grant == nil || grant.Grant == nil || grant.Binding.PID != baseline.RunnerPID || (strings.TrimSpace(baseline.OwnerSubject) != "" && strings.TrimSpace(grant.Grant.Approver) != strings.TrimSpace(baseline.OwnerSubject)) {
		return AuthorizationCredentials{OperationRefusal: "grant_binding_mismatch"}, true, nil
	}
	return AuthorizationCredentials{LegacyToken: grantToken}, true, nil
}

func authorizationRefusalCode(err error) string {
	var refused *domain.AuthorizationError
	if errors.As(err, &refused) && strings.TrimSpace(refused.Code) != "" {
		return refused.Code
	}
	return "unavailable"
}

// ResolveAuthorizationCredential is called only after Unix peer verification.
// Newest non-pending consent wins; revocation never resurrects an older grant.
func (o *Orchestrator) ResolveAuthorizationCredential(ctx context.Context, pid int) (string, bool, error) {
	return o.resolveAuthorizationCredentialForRun(ctx, pid, uuid.Nil)
}

// resolveAuthorizationCredentialForRun can restrict an inherited process
// grant to the exact managed run represented by the peer. An empty run ID
// preserves legacy resolution for ownerless attached roots and grant-only
// identities.
func (o *Orchestrator) resolveAuthorizationCredentialForRun(ctx context.Context, pid int, requiredRunID uuid.UUID) (string, bool, error) {
	for root, hops := pid, 0; root > 1 && hops < 256; hops++ {
		grants, err := o.ListAuthorizations(ctx, root)
		if err != nil {
			return "", true, err
		}
		for _, g := range grants {
			if requiredRunID != uuid.Nil && (g == nil || g.RunID != requiredRunID) {
				continue
			}
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
