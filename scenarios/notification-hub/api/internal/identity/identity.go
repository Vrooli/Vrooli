package identity

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/owneridentity"
)

// OwnerCookieName is the same-origin browser session cookie. It is HttpOnly,
// so page scripts never read the owner token; the browser sends it with every
// same-origin request, including the service worker's.
const OwnerCookieName = "notification_hub_owner_token"

type Verifier interface {
	Validate(context.Context, string) (owneridentity.Identity, error)
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
