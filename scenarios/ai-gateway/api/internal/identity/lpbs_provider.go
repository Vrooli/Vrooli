package identity

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/consumeridentity"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/identity"
)

// LPBSConsumerProvider verifies the signed LPBS browser access token used by
// consumer-funded scenarios. It trusts only LPBS's published JWKS and issuer.
type LPBSConsumerProvider struct {
	issuer string
	cache  *consumeridentity.Cache
}

func NewLPBSConsumerProvider(issuer string) *LPBSConsumerProvider {
	issuer = strings.TrimSpace(issuer)
	if issuer == "" {
		issuer = "https://vrooli.com"
	}
	provider := &LPBSConsumerProvider{issuer: issuer}
	provider.cache = consumeridentity.NewCache(issuer, consumeridentity.DefaultLeeway, consumeridentity.DefaultCacheTTL, func(req *http.Request) ([]byte, error) {
		base, err := discovery.ResolveScenarioURLDefault(req.Context(), "landing-page-business-suite")
		if err != nil {
			return nil, fmt.Errorf("resolve LPBS signing keys: %w", err)
		}
		keyRequest, err := http.NewRequestWithContext(req.Context(), http.MethodGet, strings.TrimRight(base, "/")+"/.well-known/jwks.json", nil)
		if err != nil {
			return nil, err
		}
		response, err := (&http.Client{Timeout: 5 * time.Second}).Do(keyRequest)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("LPBS signing-key endpoint returned %s", response.Status)
		}
		return io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	})
	return provider
}

func (p *LPBSConsumerProvider) Source() identity.AuthSource {
	return identity.SourceConsumerAccessToken
}

func (p *LPBSConsumerProvider) VerifyRequest(ctx context.Context, req *http.Request) (identity.Principal, error) {
	if p == nil || p.cache == nil || req == nil {
		return identity.Principal{}, identity.NewFailure(identity.FailureUnavailable, identity.SourceConsumerAccessToken)
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
	if raw == "" {
		return identity.Principal{}, identity.NewFailure(identity.FailureMissing, identity.SourceConsumerAccessToken)
	}
	claims, err := p.cache.Verify(req, raw)
	if err != nil {
		class := identity.FailureInvalid
		if errors.Is(err, consumeridentity.ErrKeySetUnavailable) {
			class = identity.FailureUnavailable
		} else if errors.Is(err, consumeridentity.ErrExpired) {
			class = identity.FailureExpired
		}
		return identity.Principal{}, identity.NewFailure(class, identity.SourceConsumerAccessToken)
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
