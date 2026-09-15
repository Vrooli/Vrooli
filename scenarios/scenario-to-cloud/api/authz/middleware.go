package authz

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"scenario-to-cloud/apierrors"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/scopecatalog"
	sessioncore "github.com/vrooli/vrooli/packages/session-core"
)

// TargetResolver resolves a deployment id to its binding identity without
// exposing the locator. found=false means no such deployment.
type TargetResolver interface {
	ResolveTarget(ctx context.Context, deploymentID string) (target Target, found bool, err error)
}

// TargetResolverFunc adapts a function to TargetResolver.
type TargetResolverFunc func(ctx context.Context, deploymentID string) (Target, bool, error)

// ResolveTarget implements TargetResolver.
func (f TargetResolverFunc) ResolveTarget(ctx context.Context, id string) (Target, bool, error) {
	return f(ctx, id)
}

// Decision is the admitted request context handed to handlers. Handlers call
// Enforcer.RequireEffect before any side effect so revocation between
// admission and effect is caught.
type Decision struct {
	Principal identity.Principal
	Route     Route
	// Target is set for target-bound routes.
	Target *Target
	// ExpiresAt is the principal's credential expiry (zero when unbounded).
	ExpiresAt time.Time
	request   *http.Request
}

type decisionKey struct{}

// DecisionFromContext returns the admission decision for the request.
func DecisionFromContext(ctx context.Context) (Decision, bool) {
	decision, ok := ctx.Value(decisionKey{}).(Decision)
	return decision, ok
}

// Enforcer applies the route table to every request.
type Enforcer struct {
	cfg      Config
	resolver TargetResolver

	mu       sync.Mutex
	inflight map[string]int
}

// New validates the table and configuration and returns the enforcer.
func New(cfg Config, resolver TargetResolver) (*Enforcer, error) {
	if findings := Validate(); len(findings) > 0 {
		return nil, fmt.Errorf("authz route table invalid: %s", strings.Join(findings, "; "))
	}
	if !cfg.Authn.Enabled() {
		return nil, errors.New("authz requires at least one authentication provider; anonymous access is never configured")
	}
	if !cfg.BindLoopback && len(cfg.AllowedHosts) == 0 {
		return nil, errors.New("authz requires AllowedHosts on a non-loopback bind")
	}
	if cfg.Policy == nil {
		cfg.Policy = NewPolicy(DefaultPolicyPath())
	}
	if resolver == nil {
		return nil, errors.New("authz requires a target resolver")
	}
	return &Enforcer{cfg: cfg, resolver: resolver, inflight: map[string]int{}}, nil
}

// Config returns the resolved configuration (read-only use).
func (e *Enforcer) Config() Config { return e.cfg }

// Middleware is the default-deny boundary. It must be installed on the router
// so mux.CurrentRoute resolves the matched template.
func (e *Enforcer) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route, ok := e.routeFor(r)
		if !ok {
			e.deny(w, r, Route{Method: r.Method, Path: r.URL.Path}, identity.Principal{}, apierrors.New(apierrors.CodeForbiddenScope, "Route is not classified in the authorization matrix"))
			return
		}
		if !e.hostAllowed(r.Host) {
			e.deny(w, r, route, identity.Principal{}, apierrors.New(apierrors.CodeForbiddenHost, "Host header is not one this API answers for"))
			return
		}
		if route.Effect == EffectPublic {
			next.ServeHTTP(w, r)
			return
		}
		principal, err := e.principal(r)
		if err != nil {
			e.deny(w, r, route, identity.Principal{}, err)
			return
		}
		if denied := e.admit(r, route, principal); denied != nil {
			e.deny(w, r, route, principal, denied)
			return
		}
		decision := Decision{Principal: principal, Route: route, ExpiresAt: principal.ExpiresAt, request: r}
		if route.TargetBound {
			target, found, denied := e.bindTarget(r, principal)
			if denied != nil {
				e.deny(w, r, route, principal, denied)
				return
			}
			if found {
				decision.Target = &target
			}
		}
		if route.BodyTarget {
			unrestricted, err := e.cfg.Policy.Unrestricted(principal)
			if err != nil {
				e.deny(w, r, route, principal, apierrors.Internal("Authorization policy unavailable", err))
				return
			}
			if !unrestricted {
				e.deny(w, r, route, principal, forbiddenTarget())
				return
			}
		}
		if route.NoStore {
			w.Header().Set("Cache-Control", "no-store")
		}
		if r.Body != nil && r.Body != http.NoBody {
			limit := e.cfg.maxBodyBytes()
			if r.ContentLength > limit {
				e.deny(w, r, route, principal, apierrors.New(apierrors.CodeRequestTooLarge, "Request body exceeds the management limit").WithDetail("limit_bytes", limit))
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
		}
		if route.Effectful() {
			release, ok := e.acquire(principal)
			if !ok {
				e.deny(w, r, route, principal, apierrors.New(apierrors.CodeTooManyRequests, "Too many in-flight effectful requests for this principal").WithRetryable(true))
				return
			}
			defer release()
		}
		e.audit("authz.admit", route, principal, decision.Target, "admitted", "")
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), decisionKey{}, decision)))
	})
}

func (e *Enforcer) routeFor(r *http.Request) (Route, bool) {
	current := mux.CurrentRoute(r)
	if current == nil {
		return Route{}, false
	}
	template, err := current.GetPathTemplate()
	if err != nil {
		return Route{}, false
	}
	return Lookup(r.Method, template)
}

// principal resolves the verified principal. The shared api-core middleware
// normally runs ahead of this boundary (server.Config.Authentication); when it
// did not, the same provider chain is evaluated here so a handler can never be
// reached without one authentication pass.
func (e *Enforcer) principal(r *http.Request) (identity.Principal, *apierrors.Error) {
	ctx := r.Context()
	if principal, ok := identity.PrincipalFromContext(ctx); ok {
		return principal, nil
	}
	var failure error
	if status, ok := identity.StatusFromContext(ctx); ok {
		failure = &identity.Failure{Class: status.FailureClass, Source: status.Source}
		if status.FailureClass == "" {
			failure = identity.NewFailure(identity.FailureMissing, status.Source)
		}
	} else {
		principal, err := e.cfg.Authn.Authenticate(ctx, r)
		if err == nil {
			return principal, nil
		}
		failure = err
	}
	return identity.Principal{}, e.unauthenticated(failure)
}

func (e *Enforcer) unauthenticated(cause error) *apierrors.Error {
	reason := string(identity.FailureMissing)
	if failure, ok := identity.FailureFromError(cause); ok && failure.Class != "" {
		reason = string(failure.Class)
	}
	err := apierrors.New(apierrors.CodeUnauthenticated, "A verified operator principal is required").WithDetail("reason", reason)
	action := apierrors.NextAction{Owner: "operator", Kind: "sign_in", Label: "Sign in and retry"}
	switch {
	case strings.TrimSpace(e.cfg.Authn.RecoveryURL) != "":
		action.Reference = e.cfg.Authn.RecoveryURL
	case e.cfg.Mode == ModePersonalLocal:
		action.Reference = "docs/reference/configuration.md#authentication"
		action.Label = "Use the runtime-owned local session token, or configure a shared provider and present a bearer token"
	default:
		action.Reference = "docs/reference/configuration.md#authentication"
	}
	return err.WithNextAction(action)
}

func (e *Enforcer) admit(r *http.Request, route Route, principal identity.Principal) *apierrors.Error {
	if principal.Kind != identity.ActorHuman && !route.ServiceAllowed {
		return apierrors.New(apierrors.CodeForbiddenScope, "Only a human operator session may use this route").WithDetail("actor_kind", string(principal.Kind))
	}
	if !scopecatalog.Resolve(principal.Scopes, route.Scope) {
		return apierrors.New(apierrors.CodeForbiddenScope, "Principal lacks the required scope").WithDetail("required_scope", route.Scope)
	}
	revoked, err := e.cfg.Policy.Revoked(principal)
	if err != nil {
		return apierrors.Internal("Authorization policy unavailable", err)
	}
	if revoked {
		return apierrors.New(apierrors.CodeForbiddenRevoked, "Principal grant has been revoked")
	}
	if e.expired(principal) {
		return e.unauthenticated(identity.NewFailure(identity.FailureExpired, principal.Source))
	}
	if route.Effect == EffectInteractive || stateChanging(r.Method) {
		if denied := e.originCheck(r, route); denied != nil {
			return denied
		}
	}
	return nil
}

// bindTarget resolves the deployment named by {id} and checks the
// principal's grant. An unrestricted principal with an unknown id passes
// through so the handler writes its own typed deployment_not_found; a
// restricted principal never learns whether the id exists.
func (e *Enforcer) bindTarget(r *http.Request, principal identity.Principal) (Target, bool, *apierrors.Error) {
	id := strings.TrimSpace(mux.Vars(r)["id"])
	if id == "" {
		return Target{}, false, apierrors.New(apierrors.CodeInvalidRequest, "Deployment id is required")
	}
	unrestricted, err := e.cfg.Policy.Unrestricted(principal)
	if err != nil {
		return Target{}, false, apierrors.Internal("Authorization policy unavailable", err)
	}
	target, found, err := e.resolver.ResolveTarget(r.Context(), id)
	if err != nil {
		return Target{}, false, apierrors.Internal("Failed to resolve deployment target", err)
	}
	if !found {
		if !unrestricted {
			return Target{}, false, forbiddenTarget()
		}
		return Target{}, false, nil
	}
	allowed, err := e.cfg.Policy.Allows(principal, target)
	if err != nil {
		return Target{}, false, apierrors.Internal("Authorization policy unavailable", err)
	}
	if !allowed {
		return Target{}, false, forbiddenTarget()
	}
	return target, true, nil
}

// forbiddenTarget carries no environment, key, or host so a refused caller
// learns nothing about the deployment.
func forbiddenTarget() *apierrors.Error {
	return apierrors.New(apierrors.CodeForbiddenTarget, "Principal is not granted this deployment target")
}

func (e *Enforcer) expired(principal identity.Principal) bool {
	return !principal.ExpiresAt.IsZero() && !e.cfg.now().Before(principal.ExpiresAt)
}

func stateChanging(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

func (e *Enforcer) originCheck(r *http.Request, route Route) *apierrors.Error {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		if route.Effect == EffectInteractive {
			return apierrors.New(apierrors.CodeForbiddenOrigin, "Interactive sessions require a browser Origin")
		}
		if strings.EqualFold(strings.TrimSpace(r.Header.Get("Sec-Fetch-Site")), "cross-site") {
			return apierrors.New(apierrors.CodeForbiddenOrigin, "Cross-site request refused")
		}
		return nil
	}
	if !e.OriginAllowed(origin, r.Host) {
		return apierrors.New(apierrors.CodeForbiddenOrigin, "Cross-origin state change refused")
	}
	return nil
}

// OriginAllowed reports whether a browser origin may change state: same
// origin as Host, an explicitly allowed origin, or (on a loopback bind) any
// loopback origin, which is the same trust boundary personal_local already
// grants to every loopback process.
func (e *Enforcer) OriginAllowed(origin, host string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return false
	}
	if sessioncore.SameOrigin(origin, host) {
		return true
	}
	for _, allowed := range e.cfg.AllowedOrigins {
		if strings.EqualFold(strings.TrimRight(allowed, "/"), strings.TrimRight(origin, "/")) {
			return true
		}
	}
	if e.cfg.BindLoopback {
		if parsed, err := url.Parse(origin); err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && isLoopbackHost(parsed.Host) {
			return true
		}
	}
	return false
}

func (e *Enforcer) hostAllowed(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" {
		return false
	}
	if len(e.cfg.AllowedHosts) == 0 {
		return isLoopbackHost(host)
	}
	bare := host
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		bare = parsed
	}
	for _, allowed := range e.cfg.AllowedHosts {
		if strings.EqualFold(allowed, host) || strings.EqualFold(allowed, bare) {
			return true
		}
	}
	return false
}

func (e *Enforcer) acquire(principal identity.Principal) (func(), bool) {
	key := string(principal.Source) + "|" + principal.Subject
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.inflight[key] >= e.cfg.effectConcurrency() {
		return nil, false
	}
	e.inflight[key]++
	return func() {
		e.mu.Lock()
		defer e.mu.Unlock()
		if e.inflight[key] <= 1 {
			delete(e.inflight, key)
			return
		}
		e.inflight[key]--
	}, true
}

// Recheck re-validates the principal's credential and grant. It is called at
// the effect boundary so a credential or grant revoked after admission cannot
// carry an effect through.
func (e *Enforcer) Recheck(ctx context.Context, principal identity.Principal) *apierrors.Error {
	decision, ok := DecisionFromContext(ctx)
	if !ok || decision.request == nil {
		return apierrors.New(apierrors.CodeForbiddenRevoked, "No admission decision for this request")
	}
	if decision.Principal.Subject != principal.Subject || decision.Principal.Source != principal.Source {
		return apierrors.New(apierrors.CodeForbiddenRevoked, "Principal does not match the admitted request")
	}
	fresh, err := e.cfg.Authn.Authenticate(ctx, decision.request)
	if err != nil || fresh.Subject != principal.Subject || fresh.Source != principal.Source {
		return apierrors.New(apierrors.CodeForbiddenRevoked, "Principal credential no longer verifies")
	}
	if e.expired(fresh) {
		return apierrors.New(apierrors.CodeForbiddenRevoked, "Principal credential has expired")
	}
	revoked, perr := e.cfg.Policy.Revoked(fresh)
	if perr != nil {
		return apierrors.Internal("Authorization policy unavailable", perr)
	}
	if revoked {
		return apierrors.New(apierrors.CodeForbiddenRevoked, "Principal grant has been revoked")
	}
	if decision.Target != nil {
		allowed, perr := e.cfg.Policy.Allows(fresh, *decision.Target)
		if perr != nil {
			return apierrors.Internal("Authorization policy unavailable", perr)
		}
		if !allowed {
			return forbiddenTarget()
		}
	}
	return nil
}

// RequireEffect is called by effectful handlers immediately before their
// first side effect. It confirms the route was admitted as effectful, that the
// deployment matches the admitted target, re-validates the principal, and
// writes the audit line.
func (e *Enforcer) RequireEffect(ctx context.Context, deploymentID string, effect Effect) *apierrors.Error {
	decision, ok := DecisionFromContext(ctx)
	if !ok {
		return apierrors.New(apierrors.CodeForbiddenScope, "Request was not admitted by the authorization boundary")
	}
	if !decision.Route.Effectful() {
		return apierrors.New(apierrors.CodeForbiddenScope, "Route is not admitted for effects").WithDetail("route", decision.Route.Key())
	}
	if decision.Target != nil && strings.TrimSpace(deploymentID) != "" && decision.Target.DeploymentID != deploymentID {
		e.audit("authz.effect", decision.Route, decision.Principal, decision.Target, "denied", apierrors.CodeForbiddenTarget)
		return forbiddenTarget()
	}
	if denied := e.Recheck(ctx, decision.Principal); denied != nil {
		e.audit("authz.effect", decision.Route, decision.Principal, decision.Target, "denied", denied.Code)
		return denied
	}
	e.audit("authz.effect", decision.Route, decision.Principal, decision.Target, "admitted", string(effect))
	return nil
}

// EffectGate wraps a handler so RequireEffect runs before it. It is used for
// package-owned handlers that cannot call the enforcer themselves.
func (e *Enforcer) EffectGate(effect Effect, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if denied := e.RequireEffect(r.Context(), mux.Vars(r)["id"], effect); denied != nil {
			apierrors.Write(w, denied)
			return
		}
		next(w, r)
	}
}

// SessionContext bounds an interactive session by the principal's credential
// expiry so a socket cannot outlive the grant that opened it.
func (e *Enforcer) SessionContext(ctx context.Context) (context.Context, context.CancelFunc) {
	decision, ok := DecisionFromContext(ctx)
	if !ok || decision.ExpiresAt.IsZero() {
		return context.WithCancel(ctx)
	}
	return context.WithDeadline(ctx, decision.ExpiresAt)
}

func (e *Enforcer) deny(w http.ResponseWriter, r *http.Request, route Route, principal identity.Principal, err *apierrors.Error) {
	e.audit("authz.deny", route, principal, nil, "denied", err.Code)
	if route.NoStore {
		w.Header().Set("Cache-Control", "no-store")
	}
	apierrors.Write(w, err)
}

// audit writes one structured line. It carries actor subject, source, scope,
// route, deployment id, target key and outcome; never headers, bodies, or
// credential values.
func (e *Enforcer) audit(msg string, route Route, principal identity.Principal, target *Target, outcome, code string) {
	if e.cfg.Logger == nil {
		return
	}
	fields := map[string]any{
		"actor":   principal.Subject,
		"source":  string(principal.Source),
		"kind":    string(principal.Kind),
		"scope":   route.Scope,
		"effect":  string(route.Effect),
		"route":   route.Key(),
		"outcome": outcome,
	}
	if code != "" {
		fields["code"] = code
	}
	if target != nil {
		fields["deployment_id"] = target.DeploymentID
		fields["target_key"] = target.Key
		fields["environment"] = target.Environment
	}
	e.cfg.Logger(msg, fields)
}
