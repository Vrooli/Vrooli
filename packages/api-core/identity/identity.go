// Package identity defines the provider-neutral request identity contract.
// Authentication proves a principal; domain packages still own authorization
// and exact approval for state-changing operations.
package identity

import (
	"context"
	"errors"
	"strings"
	"time"
)

type ActorKind string

const (
	ActorUnknown  ActorKind = "unknown"
	ActorHuman    ActorKind = "human"
	ActorAgent    ActorKind = "agent"
	ActorService  ActorKind = "service"
	ActorConflict ActorKind = "conflict"
)

func (k ActorKind) IsHuman() bool { return k == ActorHuman }

type AuthSource string

const (
	SourceUnknown               AuthSource = "unknown"
	SourceCloudflareAccess      AuthSource = "cloudflare_access"
	SourceScenarioAuthenticator AuthSource = "scenario_authenticator"
	SourcePersonalLocal         AuthSource = "personal_local"
	SourceAgentProvenance       AuthSource = "agent_provenance"
)

type FailureClass string

const (
	FailureMissing     FailureClass = "missing"
	FailureInvalid     FailureClass = "invalid"
	FailureExpired     FailureClass = "expired"
	FailureUnavailable FailureClass = "unavailable"
	FailureService     FailureClass = "service"
	FailureConflict    FailureClass = "conflict"
)

// Failure is safe to return to callers and logs. It deliberately contains
// only bounded metadata and never the credential that caused the failure.
type Failure struct {
	Class           FailureClass
	Source          AuthSource
	KeyID           string
	Issuer          string
	AudienceMatched bool
}

func (f Failure) Error() string {
	class := strings.TrimSpace(string(f.Class))
	if class == "" {
		class = string(FailureInvalid)
	}
	source := strings.TrimSpace(string(f.Source))
	if source == "" {
		source = string(SourceUnknown)
	}
	return "authentication " + class + " for " + source
}

func NewFailure(class FailureClass, source AuthSource) *Failure {
	return &Failure{Class: class, Source: source}
}

func FailureFromError(err error) (Failure, bool) {
	if err == nil {
		return Failure{}, false
	}
	var pointer *Failure
	if errors.As(err, &pointer) && pointer != nil {
		return *pointer, true
	}
	var value Failure
	if errors.As(err, &value) {
		return value, true
	}
	if failure, ok := err.(*Failure); ok && failure != nil {
		return *failure, true
	}
	if failure, ok := err.(Failure); ok {
		return failure, true
	}
	return Failure{}, false
}

// Principal is the only identity shape that domain authorization should
// consume. Subject and claims are accepted only after provider verification.
type Principal struct {
	Kind    ActorKind
	Subject string
	// Owner is the verified account on whose behalf an agent operates. It is
	// never substituted for the actor's Subject in attribution or approvals.
	Owner     string
	Email     string
	Realm     string
	Roles     []string
	Scopes    []string
	Verified  bool
	Source    AuthSource
	Sources   []AuthSource
	Issuer    string
	Audience  string
	ExpiresAt time.Time
	// Grant is present only on a live, Agent Manager-verified interactive
	// delegation. The approver remains distinct from this agent's Subject.
	Grant *OperationGrant
}

type OperationGrant struct {
	ID         string    `json:"id"`
	Policy     string    `json:"policy"`
	Approver   string    `json:"approver"`
	Operations []string  `json:"operations"`
	Targets    []string  `json:"targets"`
	ExpiresAt  time.Time `json:"expires_at"`
	// Rules pairs each reviewed policy's operations with its own targets when
	// one consent covers several policies. When present, Allows matches only
	// within a rule, so an operation of one policy never reaches a target of
	// another; Operations and Targets are then the display union.
	Rules []OperationRule `json:"rules,omitempty"`
}

// OperationRule is one reviewed policy inside a multi-policy grant.
type OperationRule struct {
	Policy     string   `json:"policy"`
	Operations []string `json:"operations"`
	Targets    []string `json:"targets"`
}

// Allows uses exact operations and explicit domain-wide or exact object
// targets. It never accepts arbitrary glob or prefix expressions.
func (g *OperationGrant) Allows(operation, target string, now time.Time) bool {
	if g == nil || g.ID == "" || g.Approver == "" || !now.Before(g.ExpiresAt) {
		return false
	}
	if len(g.Rules) > 0 {
		for _, rule := range g.Rules {
			if operationTargetAllowed(rule.Operations, rule.Targets, operation, target) {
				return true
			}
		}
		return false
	}
	return operationTargetAllowed(g.Operations, g.Targets, operation, target)
}

func operationTargetAllowed(operations, targets []string, operation, target string) bool {
	op := false
	for _, allowed := range operations {
		if allowed == operation {
			op = true
		}
	}
	if !op || target == "" {
		return false
	}
	domain, _, ok := strings.Cut(target, ":")
	if !ok || domain == "" {
		return false
	}
	for _, allowed := range targets {
		if allowed == target || allowed == domain+":*" {
			return true
		}
		// Scoped prefix wildcard: `domain:prefix/*` covers any target under
		// that prefix (e.g. `agent-manager:effort/*` covers
		// `agent-manager:effort/notification-hub-redesign`). The trailing
		// slash in the retained prefix anchors the match at a path boundary,
		// so `effort/*` never matches a sibling like `effortish/x`, and the
		// retained `domain:` segment keeps the match inside one domain.
		if prefix, ok := strings.CutSuffix(allowed, "/*"); ok && strings.HasPrefix(target, prefix+"/") {
			return true
		}
	}
	return false
}

func (p Principal) IsVerified() bool {
	return p.Verified && strings.TrimSpace(p.Subject) != "" && p.Kind != ActorUnknown && p.Kind != ActorConflict
}

func (p Principal) IsHuman() bool { return p.IsVerified() && p.Kind == ActorHuman }

func (p Principal) OwnerSubject() string {
	if !p.IsVerified() {
		return ""
	}
	if p.Kind == ActorAgent {
		return p.Owner
	}
	return p.Subject
}

type AuthState string

const (
	StateSignedOut AuthState = "signed_out"
	StateVerified  AuthState = "verified"
	StateAgent     AuthState = "agent"
	StateService   AuthState = "service"
	StateConflict  AuthState = "conflict"
	StateExpired   AuthState = "expired"
	StateError     AuthState = "error"
)

// Status is a browser-safe authentication read model. RecoveryURL is a
// configured navigation target, never a URL containing a credential.
type Status struct {
	State           AuthState
	Authenticated   bool
	Source          AuthSource
	Principal       Principal
	FailureClass    FailureClass
	RecoveryURL     string
	Reason          string
	ProviderSources []AuthSource
}

type (
	contextKey       struct{}
	statusContextKey struct{}
)

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, contextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(contextKey{}).(Principal)
	return principal, ok && principal.IsVerified()
}

func WithStatus(ctx context.Context, status Status) context.Context {
	return context.WithValue(ctx, statusContextKey{}, status)
}

func StatusFromContext(ctx context.Context) (Status, bool) {
	status, ok := ctx.Value(statusContextKey{}).(Status)
	return status, ok
}
