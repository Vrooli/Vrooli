package identity

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/hubsender"
	apiidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/owneridentity"
	"github.com/vrooli/api-core/principalrole"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/api-core/scopecatalog"
	"github.com/vrooli/cli-core/cliutil"
)

// OwnerCookieName is the same-origin browser session cookie. It is HttpOnly,
// so page scripts never read the owner token; the browser sends it with every
// same-origin request, including the service worker's.
const OwnerCookieName = "notification_hub_owner_token"

type Verifier interface {
	Validate(context.Context, string) (owneridentity.Identity, error)
}

// WithLeasedTestOwner attaches the disposable fixture identity to a test-mode
// request. Its caller must first pin an active routed test lease; production
// identity is never inferred from the browser's test-fixture header.
func WithLeasedTestOwner(ctx context.Context, subject string) context.Context {
	if !database.IsTestMode(ctx) || strings.TrimSpace(subject) == "" {
		return ctx
	}
	return context.WithValue(ctx, ownerContextKey{}, ownerContext{identity: owneridentity.Identity{Subject: strings.TrimSpace(subject)}})
}

// Producer names the source bucket and configured recipient for a direct
// notification or decision ask. It does not confer owner authority.
type Producer struct {
	// Recipient is the hub's recipient partition (requested_by). An agent
	// always reaches the configured recipient; it cannot choose another one.
	Recipient string
	// Key names the source for the shared device-push bucket. Identityless
	// callers share the anonymous source bucket.
	Key string
	// Conversation is the verified conversation of an agent run (shared with the
	// runs recovered from it and spawned by it); empty for any other producer
	// and for a token that carries none.
	Conversation string
	// Subject is the verified owner account behind the producer: the run's owner
	// for an agent run, the owner for the owner. Empty when none is verified.
	Subject string
	// Source is the scenario a verified service sender sends as, read off its
	// principal name (hubsender); empty for every other producer. It attributes
	// what the sender raises and is never taken from a request.
	Source string
}

// ServiceVerifiers holds the verifier of each service-principal role the Hub
// accepts. A nil verifier turns its role off: a token of that role is then
// never admitted.
type ServiceVerifiers struct {
	// AskReaders verifies askreader tokens (read the asks a source raised as events).
	AskReaders authn.TokenVerifier
	// Senders verifies hubsender tokens (send notices and asks as the source).
	Senders authn.TokenVerifier
}

// serviceKeyPrefix starts the Key of a verified service sender, so its asks are
// recorded as created by the service and read back by the same service.
const serviceKeyPrefix = "service:"

// anonymousSourceKey is the shared source bucket of callers with no identity.
const anonymousSourceKey = "source:anonymous"

// AskSource is the source a direct ask of this producer is stored under: the
// scenario a verified sender sends as, and for every other producer the
// recipient, which is the Hub's own default source. The ask's subject key and
// its withdrawal are scoped by it.
func (p Producer) AskSource() string {
	if p.Source != "" {
		return p.Source
	}
	return p.Recipient
}

// Anonymous reports that the producer presented no identity at all. Such a
// producer may create notifications and asks, nothing that stores or reads host
// data on its behalf.
func (p Producer) Anonymous() bool { return p.Key == anonymousSourceKey }

// ProducerFor resolves an authenticated producer for an ask withdrawal.
// It never grants owner authority: answers, dismissals and owner settings keep
// using Subject.
func ProducerFor(ctx context.Context, headers http.Header, verifier Verifier, recipient string) (Producer, error) {
	if owner, ok := ownerFromContext(ctx); ok && owner.refused {
		return Producer{}, owneridentity.ErrUnauthenticated
	}
	agent := provenance.FromContext(ctx)
	if headers.Get(cliutil.HeaderAgentIdentityToken) != "" || agent.IsAgent() {
		recipient = strings.TrimSpace(recipient)
		if !agent.IsVerifiedAgent() || recipient == "" {
			return Producer{}, owneridentity.ErrUnauthenticated
		}
		// requested_by is the hub's recipient partition, not the acting run.
		// Verified run attribution stays in canonical request provenance.
		return Producer{
			Recipient: recipient, Key: "run:" + strings.TrimSpace(agent.RunID),
			Conversation: strings.TrimSpace(agent.ConversationID), Subject: strings.TrimSpace(agent.Subject),
		}, nil
	}
	subject, err := Subject(ctx, headers, verifier)
	if err != nil {
		return Producer{}, err
	}
	return Producer{Recipient: subject, Key: "owner:" + subject, Subject: subject}, nil
}

// ProducerForDirect accepts notification and decision-request creation from
// host processes without identity. Existing verified identities still get
// separate source buckets, while empty-credential callers share one bucket. A
// verified service sender (a hubsender principal) sends as its own scenario.
// A supplied but invalid credential remains an error: it is never downgraded to
// the anonymous bucket, and a verified service token that is not a sender
// is refused with the reason (principalrole.ErrRefused).
func ProducerForDirect(ctx context.Context, headers http.Header, verifier Verifier, services ServiceVerifiers, recipient string) (Producer, error) {
	producer, err := ProducerFor(ctx, headers, verifier, recipient)
	if err == nil {
		return producer, nil
	}
	if owner, ok := ownerFromContext(ctx); ok && owner.refused {
		return Producer{}, err
	}
	if !hasProducerCredential(headers) {
		recipient = strings.TrimSpace(recipient)
		if recipient == "" {
			return Producer{}, err
		}
		return Producer{Recipient: recipient, Key: anonymousSourceKey}, nil
	}
	return serviceProducer(ctx, headers, services, recipient, err)
}

// serviceProducer admits a bearer that is a verified sender service token. It
// is the last resort for a presented credential, so ownerErr, the owner and
// agent resolution's own refusal, stands unless the bearer is a service token.
func serviceProducer(ctx context.Context, headers http.Header, services ServiceVerifiers, recipient string, ownerErr error) (Producer, error) {
	recipient = strings.TrimSpace(recipient)
	if services.Senders == nil || recipient == "" || strings.TrimSpace(headers.Get(cliutil.HeaderAgentIdentityToken)) != "" {
		return Producer{}, ownerErr
	}
	source, err := hubsender.VerifiedSource(ctx, services.Senders, headers)
	switch {
	case err == nil:
		return Producer{Recipient: recipient, Key: serviceKeyPrefix + source, Source: source}, nil
	case errors.Is(err, principalrole.ErrRefused):
		return Producer{}, err
	}
	return Producer{}, ownerErr
}

// DirectProducerError is the Connect refusal for a direct producer that could
// not be resolved. A service token that verified but is not a sender is
// PermissionDenied and names what is missing, so its operator can fix the
// principal; every other failure keeps the generic unauthenticated answer.
func DirectProducerError(err error) error {
	if errors.Is(err, principalrole.ErrRefused) {
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	return connect.NewError(connect.CodeUnauthenticated, errors.New("a verified agent run, verified owner or notification-hub sender identity is required"))
}

// ResolveDirectProducer applies the shared direct-send identity policy and
// maps refusals to the Connect status callers should return.
func ResolveDirectProducer(ctx context.Context, headers http.Header, verifier Verifier, services ServiceVerifiers, recipient string) (Producer, error) {
	producer, err := ProducerForDirect(ctx, headers, verifier, services, recipient)
	if err != nil {
		return Producer{}, DirectProducerError(err)
	}
	return producer, nil
}

func hasProducerCredential(headers http.Header) bool {
	return strings.TrimSpace(headers.Get(cliutil.HeaderAgentIdentityToken)) != "" ||
		strings.TrimSpace(headers.Get("Authorization")) != "" ||
		strings.TrimSpace(headers.Get("Cf-Access-Jwt-Assertion")) != "" ||
		strings.TrimSpace(headers.Get("Cookie")) != ""
}

// Subject verifies the owner: a Cloudflare Access login first, then the
// credential issued by scenario-authenticator: a bearer token (CLI and
// services) or the owner cookie (browser sign-in). The
// legacy subject header is accepted only when no verifier is wired, which
// keeps isolated handler tests useful while production always fails closed.
func Subject(ctx context.Context, headers http.Header, verifier Verifier) (string, error) {
	identity, err := Owner(ctx, headers, verifier)
	if err != nil {
		return "", err
	}
	return identity.Subject, nil
}

// Owner is Subject with the verified identity's other claims. The request's
// Cloudflare Access login (see OwnerAuthenticator) is the primary owner and
// wins over a bearer or cookie; a presented assertion that failed
// verification is refused rather than falling back.
func Owner(ctx context.Context, headers http.Header, verifier Verifier) (owneridentity.Identity, error) {
	if owner, ok := ownerFromContext(ctx); ok {
		if owner.refused {
			return owneridentity.Identity{}, owneridentity.ErrUnauthenticated
		}
		return owner.identity, nil
	}
	if verifier == nil {
		if subject := strings.TrimSpace(headers.Get("X-Vrooli-Identity-Subject")); subject != "" {
			return owneridentity.Identity{Subject: subject}, nil
		}
	}
	token := Token(headers)
	if token == "" || verifier == nil {
		return owneridentity.Identity{}, owneridentity.ErrUnauthenticated
	}
	identity, err := verifier.Validate(ctx, token)
	if err != nil {
		return owneridentity.Identity{}, err
	}
	if strings.TrimSpace(identity.Subject) == "" {
		return owneridentity.Identity{}, errors.New("verified identity has no subject")
	}
	bindings, err := OwnerBindings()
	if err != nil {
		return owneridentity.Identity{}, owneridentity.ErrUnavailable
	}
	for _, binding := range bindings {
		// The legacy owner surface authorizes writes as well as reads. Do not
		// turn a read-only binding into full owner access by mapping its subject.
		if binding.Source == apiidentity.SourceScenarioAuthenticator && binding.Subject == identity.Subject && scopecatalog.MatchCapability(binding.Scopes, "notification-hub:write") {
			identity.Subject = binding.CanonicalSubject
			break
		}
	}
	return identity, nil
}

// OwnerBindings uses the shared policy parser and this relying scenario's
// resource filter. Only exact verified provider subjects can be mapped.
func OwnerBindings() ([]authn.OwnerBinding, error) {
	cfg, err := authn.FromEnvironment(func(key string) string {
		switch key {
		case "VROOLI_AUTH_PROVIDERS":
			return ""
		case "VROOLI_SCENARIO":
			return "notification-hub"
		default:
			return os.Getenv(key)
		}
	})
	return cfg.OwnerBindings, err
}

// AnsweredBy returns the ACTUAL verified human behind the request — the
// Cloudflare Access subject (or email) captured by OwnerAuthenticator — to
// attribute an answer to the person who really gave it. It returns "" when no
// per-human identity was verified on this request (bearer/cookie/legacy owner
// paths), so the caller falls back to the recipient subject. It never affects
// the recipient-only authorization check, which keeps using the owner subject.
func AnsweredBy(ctx context.Context) string {
	if owner, ok := ownerFromContext(ctx); ok && !owner.refused {
		return strings.TrimSpace(owner.actor)
	}
	return ""
}

// AnswerIdentity returns server-derived attribution facts for a durable Hub
// answer. OwnerAuthenticator sets this context only for a verified human
// principal; bearer/cookie and legacy owner fallback return unknown.
func AnswerIdentity(ctx context.Context) (kind, source string) {
	owner, ok := ownerFromContext(ctx)
	if !ok || owner.refused || strings.TrimSpace(owner.actor) == "" || owner.source == "" {
		return "unknown", ""
	}
	return "verified_human", string(owner.source)
}

// Token returns the bearer token, or the owner cookie when no bearer is sent.
func Token(headers http.Header) string {
	if token := strings.TrimSpace(strings.TrimPrefix(headers.Get("Authorization"), "Bearer ")); token != "" {
		return token
	}
	cookie, err := (&http.Request{Header: headers}).Cookie(OwnerCookieName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}
