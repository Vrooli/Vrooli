package policygate

import (
	"context"
	"net"
	"net/http"
	"os/user"
	"strings"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
)

// PersonalLocalProvider is the desktop-only authority for GCT's bundled
// personal_local mode. It is deliberately limited to loopback requests and
// the current OS user. It never consumes the supervisor bearer token and it
// refuses to promote verified agent provenance into a human principal.
type PersonalLocalProvider struct {
	currentUser func() (*user.User, error)
}

func NewPersonalLocalProvider() authn.Provider {
	return PersonalLocalProvider{currentUser: user.Current}
}

func (PersonalLocalProvider) Source() identity.AuthSource { return identity.SourcePersonalLocal }

func (p PersonalLocalProvider) VerifyRequest(ctx context.Context, req *http.Request) (identity.Principal, error) {
	if req == nil {
		return identity.Principal{}, identity.NewFailure(identity.FailureInvalid, p.Source())
	}
	// A same-origin UI proxy also connects over loopback. Its offered browser
	// proof belongs to the corresponding verifier, never the server's OS user.
	// Invalid or expired offered proof must fail there without local fallback.
	if len(req.Header.Values("Authorization")) > 0 || len(req.Header.Values("Cf-Access-Jwt-Assertion")) > 0 {
		return identity.Principal{}, identity.NewFailure(identity.FailureMissing, p.Source())
	}
	for _, name := range []string{accessTokenCookieName, "access_token"} {
		if _, err := req.Cookie(name); err == nil {
			return identity.Principal{}, identity.NewFailure(identity.FailureMissing, p.Source())
		}
	}
	if provenance.FromContext(ctx).IsVerifiedAgent() {
		// This provider supplies human OS authority only. Let the shared
		// middleware project the verified agent's bounded identity; rejecting
		// it here would veto legitimate read capabilities as invalid proof.
		return identity.Principal{}, identity.NewFailure(identity.FailureMissing, p.Source())
	}
	if !isLoopbackRequest(req) {
		return identity.Principal{}, identity.NewFailure(identity.FailureInvalid, p.Source())
	}
	lookup := p.currentUser
	if lookup == nil {
		lookup = user.Current
	}
	current, err := lookup()
	if err != nil || current == nil || strings.TrimSpace(current.Uid) == "" {
		return identity.Principal{}, identity.NewFailure(identity.FailureUnavailable, p.Source())
	}
	return identity.Principal{
		Kind: identity.ActorHuman, Subject: "osuser:" + strings.TrimSpace(current.Uid),
		Realm: "personal_local", Scopes: []string{
			"git-control-tower:read", "git-control-tower:write", "git-control-tower:destructive",
		}, Verified: true, Source: p.Source(), Sources: []identity.AuthSource{p.Source()},
	}, nil
}

func isLoopbackRequest(req *http.Request) bool {
	host := strings.TrimSpace(req.RemoteAddr)
	if host == "" {
		return false
	}
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

var _ authn.Provider = PersonalLocalProvider{}
