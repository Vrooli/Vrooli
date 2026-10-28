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

// ProducerSubject accepts a scoped, server-verified run for notification ingress.
// It never makes that run an owner or authorizes recipient decisions.
func ProducerSubject(ctx context.Context, headers http.Header, verifier Verifier, recipient string) (string, error) {
	if owner, ok := ownerFromContext(ctx); ok && owner.refused {
		return "", owneridentity.ErrUnauthenticated
	}
	agent := provenance.FromContext(ctx)
	if headers.Get(cliutil.HeaderAgentIdentityToken) != "" || agent.IsAgent() {
		if !agent.IsVerifiedAgent() {
			return "", owneridentity.ErrUnauthenticated
		}
		if err := agent.RequireScope("notification-hub:write"); err != nil {
			return "", err
		}
		// requested_by is the hub's recipient partition, not the acting run.
		// Keep verified run attribution in canonical request provenance.
		if strings.TrimSpace(recipient) == "" {
			return "", owneridentity.ErrUnauthenticated
		}
		return recipient, nil
	}
	return Subject(ctx, headers, verifier)
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
