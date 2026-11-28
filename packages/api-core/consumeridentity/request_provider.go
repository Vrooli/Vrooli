package consumeridentity

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/identity"
)

// RequestProvider verifies LPBS browser access tokens and adapts their verified
// human identity to API-core's request authentication contract.
type RequestProvider struct {
	Cache *Cache
}

func NewRequestProvider(cache *Cache) *RequestProvider {
	return &RequestProvider{Cache: cache}
}

func (p *RequestProvider) Source() identity.AuthSource { return identity.SourceConsumerAccessToken }

func (p *RequestProvider) VerifyRequest(_ context.Context, req *http.Request) (identity.Principal, error) {
	if p == nil || p.Cache == nil || req == nil {
		return identity.Principal{}, identity.NewFailure(identity.FailureUnavailable, identity.SourceConsumerAccessToken)
	}
	raw := RequestAccessToken(req)
	if raw == "" {
		return identity.Principal{}, identity.NewFailure(identity.FailureMissing, identity.SourceConsumerAccessToken)
	}
	claims, err := p.Cache.Verify(req, raw)
	if err != nil {
		return identity.Principal{}, identity.NewFailure(ClassifyRequestError(err), identity.SourceConsumerAccessToken)
	}
	subject := strings.TrimSpace(claims.UserID)
	if subject == "" {
		subject = strings.TrimSpace(claims.Subject)
	}
	if subject == "" {
		return identity.Principal{}, identity.NewFailure(identity.FailureInvalid, identity.SourceConsumerAccessToken)
	}
	return identity.Principal{Kind: identity.ActorHuman, Subject: subject, Email: strings.TrimSpace(claims.Email), Verified: true, Source: identity.SourceConsumerAccessToken, Issuer: claims.Issuer}, nil
}

// RequestAccessToken extracts the bearer value accepted by LPBS-backed APIs.
func RequestAccessToken(req *http.Request) string {
	if req == nil {
		return ""
	}
	raw := strings.TrimSpace(req.Header.Get("Authorization"))
	if scheme, token, ok := strings.Cut(raw, " "); ok && strings.EqualFold(scheme, "Bearer") {
		raw = strings.TrimSpace(token)
	}
	if raw == "" {
		for _, cookieName := range []string{"access_token", "__Host-access_token"} {
			if cookie, err := req.Cookie(cookieName); err == nil {
				raw = strings.TrimSpace(cookie.Value)
				if raw != "" {
					break
				}
			}
		}
	}
	return raw
}

func ClassifyRequestError(err error) identity.FailureClass {
	if errors.Is(err, ErrKeySetUnavailable) {
		return identity.FailureUnavailable
	}
	if errors.Is(err, ErrExpired) {
		return identity.FailureExpired
	}
	return identity.FailureInvalid
}
