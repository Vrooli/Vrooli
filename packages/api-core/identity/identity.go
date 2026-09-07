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
	Kind      ActorKind
	Subject   string
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
}

func (p Principal) IsVerified() bool {
	return p.Verified && strings.TrimSpace(p.Subject) != "" && p.Kind != ActorUnknown && p.Kind != ActorConflict
}

func (p Principal) IsHuman() bool { return p.IsVerified() && p.Kind == ActorHuman }

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
