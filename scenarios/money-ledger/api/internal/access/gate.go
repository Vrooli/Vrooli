package access

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/api-core/scopecatalog"
)

// Refusal names why a caller is not admitted as a writer.
type Refusal string

const (
	RefusedNone              Refusal = ""
	RefusedSignedOut         Refusal = "signed_out"
	RefusedUnverified        Refusal = "unverified"
	RefusedNotOwner          Refusal = "not_owner"
	RefusedAgent             Refusal = "agent"
	RefusedUndeclaredService Refusal = "undeclared_service"
	RefusedConflict          Refusal = "conflict"
)

// Caller is the gate's verdict on who a request is: the actor it writes as,
// or why it is not a writer.
type Caller struct {
	Actor   Actor
	Refusal Refusal
	Reason  string
	// SignedInAs is the verified person's email, or subject when the provider
	// gives none, whether or not that person may write. Empty for services,
	// agents and unverified callers.
	SignedInAs string
}

// Writer reports whether the caller resolved to an actor.
func (c Caller) Writer() bool { return c.Actor != "" }

// Gate authenticates every request with the shared provider chain and then
// applies Policy. It is the only place a Money Ledger route is authorized.
type Gate struct {
	authentication authn.Config
}

// NewGate builds the gate over the provider chain and owner-binding policy in
// cfg. An empty cfg verifies nobody: every open read still works and every
// write is refused.
func NewGate(cfg authn.Config) *Gate { return &Gate{authentication: cfg} }

// Middleware refuses a cross-site browser request, authenticates the request,
// refuses a route Policy does not declare, and admits a restricted route only
// for an actor it names. An admitted writer's actor is in the request context
// for the handler.
func (g *Gate) Middleware(next http.Handler) http.Handler {
	authorize := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if crossSite(r) {
			refuse(w, r, connect.NewError(connect.CodePermissionDenied, errors.New("money-ledger answers only its own pages and local programs, not other websites")))
			return
		}
		rule, declared := Policy[r.URL.Path]
		if !declared {
			refuse(w, r, connect.NewError(connect.CodePermissionDenied, errors.New("this route is not declared in Money Ledger's access policy")))
			return
		}
		if rule.Open() {
			next.ServeHTTP(w, r)
			return
		}
		caller := g.Resolve(r.Context())
		if !caller.Writer() {
			refuse(w, r, refusalError(caller))
			return
		}
		if !rule.Admits(caller.Actor) {
			refuse(w, r, connect.NewError(connect.CodePermissionDenied, errors.New(string(caller.Actor)+" may not use this operation")))
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), actorKey{}, caller.Actor)))
	})
	return authn.Middleware(g.authentication)(authorize)
}

// Resolve decides who the authenticated request is. It never trusts a claim
// the request makes about itself: only the verified principal the shared
// middleware placed in the context, checked against the owner-binding policy
// and the declared service writers.
func (g *Gate) Resolve(ctx context.Context) Caller {
	if provenance.FromContext(ctx).IsVerifiedAgent() {
		return Caller{Refusal: RefusedAgent, Reason: "an agent run cannot write to Money Ledger; the owner or a declared service writes it"}
	}
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || !principal.IsVerified() {
		status, _ := identity.StatusFromContext(ctx)
		switch {
		case status.State == identity.StateConflict || principal.Kind == identity.ActorConflict:
			return Caller{Refusal: RefusedConflict, Reason: "this request presents two different identities; sign out of the Money Ledger session or the other sign-in, then try again"}
		case status.State == identity.StateSignedOut || status.State == "":
			return Caller{Refusal: RefusedSignedOut, Reason: "no identity was presented"}
		}
		return Caller{Refusal: RefusedUnverified, Reason: "the presented identity could not be verified; it may have expired"}
	}
	switch principal.Kind {
	case identity.ActorAgent:
		return Caller{Refusal: RefusedAgent, Reason: "an agent run cannot write to Money Ledger; the owner or a declared service writes it"}
	case identity.ActorHuman:
		person := principal.Email
		if strings.TrimSpace(person) == "" {
			person = principal.Subject
		}
		if !g.ownerBound(principal) {
			return Caller{Refusal: RefusedNotOwner, Reason: "this person is not named as the owner of these records by the host's owner-binding policy", SignedInAs: person}
		}
		if !scopecatalog.Resolve(principal.Scopes, WriteCapability) {
			return Caller{Refusal: RefusedNotOwner, Reason: "the owner binding does not grant " + WriteCapability, SignedInAs: person}
		}
		return Caller{Actor: Operator, SignedInAs: person}
	case identity.ActorService:
		writer, declared := serviceWriters[principal.Subject]
		if !declared || principal.Source != identity.SourceScenarioAuthenticator {
			return Caller{Refusal: RefusedUndeclaredService, Reason: "this service is not a declared Money Ledger writer"}
		}
		if !scopecatalog.Resolve(principal.Scopes, writer.scope) {
			return Caller{Refusal: RefusedUndeclaredService, Reason: "this service principal does not hold " + writer.scope}
		}
		return Caller{Actor: writer.actor}
	}
	return Caller{Refusal: RefusedUnverified, Reason: "the presented identity could not be verified"}
}

// ResolveSession decides who a browser session that was just established is,
// as Resolve will on the next request that presents it. Sign-in and renewal
// use it to report the new session's access in the same response.
func (g *Gate) ResolveSession(ctx context.Context, accessToken string) Caller {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "/", http.NoBody)
	if err != nil {
		return Caller{Refusal: RefusedUnverified, Reason: "the new session could not be verified"}
	}
	// The token is presented as a bearer, the form every provider reads first,
	// so the verdict does not depend on which cookie name a provider watches.
	request.Header.Set("Authorization", "Bearer "+accessToken)
	principal, err := g.authentication.Authenticate(ctx, request)
	if err != nil {
		return Caller{Refusal: RefusedUnverified, Reason: "the new session could not be verified"}
	}
	return g.Resolve(identity.WithPrincipal(ctx, principal))
}

// crossSite reports a request a browser made from another website. Browsers
// mark every request with Sec-Fetch-Site and page script cannot change it;
// local programs send none. Refusing these keeps other websites from reading
// the ledger or riding the owner's session, whatever the UI server's CORS
// policy allows.
func crossSite(r *http.Request) bool {
	switch strings.TrimSpace(r.Header.Get("Sec-Fetch-Site")) {
	case "", "same-origin", "none":
		return false
	}
	return true
}

// ownerBound reports whether the host's owner-binding policy names this
// verified person. Being able to sign in is not ownership: the identity
// provider holds many accounts, and its scopes are self-service, so only the
// operator-owned binding file distinguishes the owner. The shared
// authenticator has already replaced a bound subject with its canonical
// subject, so a match is a binding from one of the principal's verified
// providers to exactly this subject.
func (g *Gate) ownerBound(principal identity.Principal) bool {
	sources := append([]identity.AuthSource{principal.Source}, principal.Sources...)
	for _, binding := range g.authentication.OwnerBindings {
		if strings.TrimSpace(binding.CanonicalSubject) == "" || binding.CanonicalSubject != principal.Subject {
			continue
		}
		for _, source := range sources {
			if source == binding.Source {
				return true
			}
		}
	}
	return false
}

// AuthorizeAdapter refuses a write under an adapter reserved for another
// actor. Handlers call it for every write that names an adapter.
func AuthorizeAdapter(ctx context.Context, adapterID string) error {
	owner, reserved := reservedAdapters[strings.TrimSpace(adapterID)]
	if !reserved {
		return nil
	}
	if actor := ActorFrom(ctx); actor != owner {
		return connect.NewError(connect.CodePermissionDenied, errors.New("adapter "+strings.TrimSpace(adapterID)+" is written only by "+string(owner)))
	}
	return nil
}

// AuthorizeAdapterRegistration keeps the compute invoice adapter owner-managed
// while reserving every use of it to the narrow invoice writer.
func AuthorizeAdapterRegistration(ctx context.Context, adapterID string) error {
	if strings.TrimSpace(adapterID) == ComputeManagerInvoiceAdapter {
		if ActorFrom(ctx) == Operator {
			return nil
		}
		return connect.NewError(connect.CodePermissionDenied, errors.New("the Compute Manager invoice adapter is registered only by the owner"))
	}
	return AuthorizeAdapter(ctx, adapterID)
}

type actorKey struct{}

// ActorFrom returns the actor the gate admitted for this request. It is empty
// on an open route; a handler for a write never sees it empty because the gate
// refuses the request first.
func ActorFrom(ctx context.Context) Actor {
	actor, _ := ctx.Value(actorKey{}).(Actor)
	return actor
}

// WithActor marks ctx as admitted for actor. It exists for store and handler
// tests that run below the gate; production contexts get the actor only from
// Middleware.
func WithActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, actorKey{}, actor)
}

func refusalError(caller Caller) *connect.Error {
	code := connect.CodePermissionDenied
	if caller.Refusal == RefusedSignedOut || caller.Refusal == RefusedUnverified {
		code = connect.CodeUnauthenticated
	}
	return connect.NewError(code, errors.New("money-ledger refused this request: "+caller.Reason))
}

var errorWriter = connect.NewErrorWriter()

// refuse answers in the caller's protocol: a Connect, gRPC or gRPC-Web error
// for an RPC, and a JSON error for a plain HTTP route.
func refuse(w http.ResponseWriter, r *http.Request, err *connect.Error) {
	if errorWriter.IsSupported(r) {
		_ = errorWriter.Write(w, r, err)
		return
	}
	status := http.StatusForbidden
	if err.Code() == connect.CodeUnauthenticated {
		status = http.StatusUnauthorized
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": err.Code().String(), "message": err.Message()})
}
