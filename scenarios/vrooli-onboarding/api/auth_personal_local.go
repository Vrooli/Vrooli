package main

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

// onboardingPersonalLocalProvider matches Git Control Tower's bundled
// personal_local contract. The local runtime boundary is the proof in this
// mode: requests must arrive from loopback and the principal is bound to the
// current OS user. Remote and shared deployments never use this provider.
type onboardingPersonalLocalProvider struct {
	currentUser func() (*user.User, error)
}

func newOnboardingPersonalLocalProvider() authn.Provider {
	return onboardingPersonalLocalProvider{currentUser: user.Current}
}

func (onboardingPersonalLocalProvider) Source() identity.AuthSource {
	return identity.SourcePersonalLocal
}

func (p onboardingPersonalLocalProvider) VerifyRequest(ctx context.Context, req *http.Request) (identity.Principal, error) {
	if req == nil || !isOnboardingLoopbackRequest(req) {
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
			"vrooli-onboarding:read", "vrooli-onboarding:write", "vrooli-onboarding:destructive",
		}, Verified: true, Source: p.Source(), Sources: []identity.AuthSource{p.Source()},
	}, nil
}

func isOnboardingLoopbackRequest(req *http.Request) bool {
	if req == nil {
		return false
	}
	host := strings.TrimSpace(req.RemoteAddr)
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

var _ authn.Provider = onboardingPersonalLocalProvider{}
