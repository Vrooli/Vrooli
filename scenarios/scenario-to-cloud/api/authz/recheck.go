package authz

import (
	"context"
	"net/http"
	"strings"
	"time"

	"scenario-to-cloud/apierrors"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/api-core/scopecatalog"
	"github.com/vrooli/cli-core/cliutil"
)

// Recheck re-validates the principal's credential and grant. It is called at
// the effect boundary so a credential or grant revoked after admission cannot
// carry an effect through.
func (e *Enforcer) Recheck(ctx context.Context, principal identity.Principal) *apierrors.Error {
	decision, denied := admittedDecision(ctx, principal)
	if denied != nil {
		return denied
	}
	fresh, denied := e.recheckPrincipal(ctx, decision, principal)
	if denied != nil {
		return denied
	}
	if denied := e.recheckAgentGrant(fresh, decision); denied != nil {
		return denied
	}
	return e.recheckPolicy(fresh, decision)
}

func admittedDecision(ctx context.Context, principal identity.Principal) (Decision, *apierrors.Error) {
	decision, ok := DecisionFromContext(ctx)
	if !ok || decision.request == nil {
		return Decision{}, apierrors.New(apierrors.CodeForbiddenRevoked, "No admission decision for this request")
	}
	if decision.Principal.Subject != principal.Subject || decision.Principal.Source != principal.Source {
		return Decision{}, apierrors.New(apierrors.CodeForbiddenRevoked, "Principal does not match the admitted request")
	}
	return decision, nil
}

func (e *Enforcer) recheckPrincipal(ctx context.Context, decision Decision, principal identity.Principal) (identity.Principal, *apierrors.Error) {
	if principal.Source == identity.SourceAgentProvenance {
		fresh, ok := e.verifyAgentPrincipal(decision)
		if !ok {
			return identity.Principal{}, apierrors.New(apierrors.CodeForbiddenRevoked, "Agent credential or grant no longer verifies")
		}
		return fresh, nil
	}
	fresh, err := e.cfg.Authn.Authenticate(ctx, decision.request)
	if err != nil || fresh.Subject != principal.Subject || fresh.Source != principal.Source {
		return identity.Principal{}, apierrors.New(apierrors.CodeForbiddenRevoked, "Principal credential no longer verifies")
	}
	return fresh, nil
}

func (e *Enforcer) verifyAgentPrincipal(decision Decision) (identity.Principal, bool) {
	verifier := e.cfg.AgentVerifier
	if verifier == nil {
		verifier = provenance.CLIUtilVerifier{}
	}
	result, err := verifier.Verify(decision.request.Header.Get(cliutil.HeaderAgentIdentityToken))
	if err != nil || result == nil || !result.Valid || result.Claims == nil {
		return identity.Principal{}, false
	}
	return identity.Principal{
		Kind: identity.ActorAgent, Subject: result.Claims.RunID, Owner: result.Claims.Subject,
		Source: identity.SourceAgentProvenance, Verified: true,
		Scopes:    append([]string(nil), result.Claims.Scopes...),
		ExpiresAt: time.Unix(result.Claims.ExpiresAt, 0),
		Grant:     identity.OperationGrantFromMeta(result.Claims.Meta, e.cfg.now()),
	}, true
}

func (e *Enforcer) recheckAgentGrant(fresh identity.Principal, decision Decision) *apierrors.Error {
	if fresh.Kind != identity.ActorAgent || !decision.Route.Effectful() {
		return nil
	}
	target := "scenario-to-cloud:*"
	if decision.Target != nil {
		target = "scenario-to-cloud:deployment/" + decision.Target.DeploymentID
	}
	if !fresh.Grant.Allows("deployment.manage", target, e.cfg.now()) || !scopecatalog.Resolve(fresh.Scopes, decision.Route.Scope) {
		return apierrors.New(apierrors.CodeForbiddenRevoked, "Agent operation grant no longer permits this effect")
	}
	return nil
}

func (e *Enforcer) recheckPolicy(fresh identity.Principal, decision Decision) *apierrors.Error {
	if e.expired(fresh) {
		return apierrors.New(apierrors.CodeForbiddenRevoked, "Principal credential has expired")
	}
	revoked, err := e.cfg.Policy.Revoked(fresh)
	if err != nil {
		return apierrors.Internal("Authorization policy unavailable", err)
	}
	if revoked {
		return apierrors.New(apierrors.CodeForbiddenRevoked, "Principal grant has been revoked")
	}
	return e.recheckTargetPolicy(fresh, decision.Target)
}

func (e *Enforcer) recheckTargetPolicy(fresh identity.Principal, target *Target) *apierrors.Error {
	if target == nil {
		return nil
	}
	allowed, err := e.cfg.Policy.AllowsAt(fresh, *target, e.cfg.now())
	if err != nil {
		return apierrors.Internal("Authorization policy unavailable", err)
	}
	if !allowed {
		return forbiddenTarget()
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
