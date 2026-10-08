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
	if req == nil || !isLoopbackRequest(req) {
		return identity.Principal{}, identity.NewFailure(identity.FailureInvalid, p.Source())
	}
	if provenance.FromContext(ctx).IsVerifiedAgent() {
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
