package policygate

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"git-control-tower/internal/config"
	coreidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"
)

// Principal is supplied by the authentication owner. It is intentionally not
// constructible from an HTTP caller header: headers are untrusted transport
// metadata and cannot grant human authority.
type Principal struct {
	Kind         cliutil.CallerKind
	Subject      string
	Email        string
	Realm        string
	Roles        []string
	Scopes       []string
	Verified     bool
	Issuer       string
	AuthSource   string
	AuthSources  []string
	RepositoryID string
	Intent       *HumanIntent
}

// HumanIntent is the authenticated, resource-bound approval issued by the
// human authority owner. It is not a caller header and it expires. The
// operation and expected revision bind the approval to one reviewed action.
type HumanIntent struct {
	ID               string
	PrincipalID      string
	RepositoryID     string
	Operation        string
	ExpectedRevision string
	SubjectDigest    string
	PolicyVersion    string
	ExpiresAt        time.Time
	ConsumedAt       *time.Time
	Consumed         bool
}

var consumedIntents sync.Map // map[intent ID]*atomic.Bool; owner persists the durable receipt

func (i HumanIntent) ValidFor(repositoryID, operation, revision string, now time.Time) bool {
	return strings.TrimSpace(i.ID) != "" && !i.Consumed && i.ConsumedAt == nil && i.ExpiresAt.After(now) &&
		i.RepositoryID == repositoryID && i.Operation == operation && i.ExpectedRevision == revision
}

func consumeIntent(i *HumanIntent, repositoryID, operation, revision string, now time.Time) bool {
	if i == nil || !i.ValidFor(repositoryID, operation, revision, now) {
		return false
	}
	used, _ := consumedIntents.LoadOrStore(i.ID, &atomic.Bool{})
	return used.(*atomic.Bool).CompareAndSwap(false, true)
}

type principalContextKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(Principal)
	if ok && principal.Verified && strings.TrimSpace(principal.Subject) != "" {
		return principal, true
	}
	shared, sharedOK := coreidentity.PrincipalFromContext(ctx)
	if !sharedOK {
		return Principal{}, false
	}
	return Principal{
		Kind: cliKind(shared.Kind), Subject: shared.Subject, Email: shared.Email,
		Realm: shared.Realm, Roles: append([]string(nil), shared.Roles...),
		Scopes: append([]string(nil), shared.Scopes...), Verified: shared.Verified,
		Issuer: shared.Issuer, AuthSource: string(shared.Source),
		AuthSources: authSources(shared.Sources),
	}, true
}

func cliKind(kind coreidentity.ActorKind) cliutil.CallerKind {
	switch kind {
	case coreidentity.ActorHuman:
		return cliutil.CallerKindHuman
	case coreidentity.ActorAgent:
		return cliutil.CallerKindVrooliAgent
	case coreidentity.ActorService:
		return cliutil.CallerKindExternalAgent
	default:
		return cliutil.CallerKindUnknown
	}
}

func authSources(sources []coreidentity.AuthSource) []string {
	result := make([]string, 0, len(sources))
	for _, source := range sources {
		if value := strings.TrimSpace(string(source)); value != "" {
			result = append(result, value)
		}
	}
	return result
}

// WithIntent attaches a server-issued intent to the request context. It is
// deliberately separate from Principal so a request header cannot manufacture
// either half of the authorization contract.
func WithIntent(ctx context.Context, intent HumanIntent) context.Context {
	principal, _ := PrincipalFromContext(ctx)
	principal.Intent = &intent
	return WithPrincipal(ctx, principal)
}

// IntentFromContext returns the pending server-issued intent, if any.
func IntentFromContext(ctx context.Context) (HumanIntent, bool) {
	principal, ok := PrincipalFromContext(ctx)
	if !ok || principal.Intent == nil {
		return HumanIntent{}, false
	}
	return *principal.Intent, true
}

// ConsumedIntentFromContext returns only an intent that has already passed the
// durable single-use store. Writer services use this as their final domain
// guard; a principal alone is never enough to reach a repository mutator.
func ConsumedIntentFromContext(ctx context.Context) (HumanIntent, bool) {
	intent, ok := IntentFromContext(ctx)
	if !ok || strings.TrimSpace(intent.ID) == "" || !intent.Consumed && intent.ConsumedAt == nil {
		return HumanIntent{}, false
	}
	return intent, true
}

func DecideAuthenticated(ctx context.Context, cmd CommandSpec, policy config.PolicyConfig) (Decision, Principal) {
	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		return DecisionDeny, Principal{Kind: cliutil.CallerKindUnknown}
	}
	// Every mutating operation requires a server-issued, exact, single-use
	// intent. Human identity establishes who may receive one; it is not itself
	// approval for an arbitrary writer call.
	if principal.Intent != nil && consumeIntent(principal.Intent, cmd.RepositoryID, cmd.Name, cmd.ExpectedRevision, time.Now().UTC()) {
		return DecisionAllow, principal
	}
	return DecisionDeny, principal
}
