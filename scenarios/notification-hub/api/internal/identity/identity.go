package identity

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/owneridentity"
	"github.com/vrooli/api-core/provenance"
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
}

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
		return Producer{Recipient: recipient, Key: "run:" + strings.TrimSpace(agent.RunID)}, nil
	}
	subject, err := Subject(ctx, headers, verifier)
	if err != nil {
		return Producer{}, err
	}
	return Producer{Recipient: subject, Key: "owner:" + subject}, nil
}

// ProducerForDirect accepts notification and decision-request creation from
// host processes without identity. Existing verified identities still get
// separate source buckets, while empty-credential callers share one bucket.
// A supplied but invalid credential remains an error.
func ProducerForDirect(ctx context.Context, headers http.Header, verifier Verifier, recipient string) (Producer, error) {
	producer, err := ProducerFor(ctx, headers, verifier, recipient)
	if owner, ok := ownerFromContext(ctx); ok && owner.refused {
		return Producer{}, err
	}
	if err == nil || hasProducerCredential(headers) {
		return producer, err
	}
	recipient = strings.TrimSpace(recipient)
	if recipient == "" {
		return Producer{}, err
	}
	return Producer{Recipient: recipient, Key: "source:anonymous"}, nil
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
	return identity, nil
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
