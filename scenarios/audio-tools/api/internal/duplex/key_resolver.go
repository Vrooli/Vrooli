package duplex

import (
	"context"
	"errors"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const openAILiveProviderID = "openai-gpt-live"

const (
	openAILiveCredentialIdentity = "vrooli/openai-gpt-live" // #nosec G101 -- logical credential identity, not key material.
	openAILiveCredentialField    = "api-key"                // #nosec G101 -- field identifier, not a credential value.
)

type authorityReader interface {
	Require(credentialauthority.Identity, string) (string, error)
}

// AuthorityKeyResolver reads the operator-provisioned API key from Vrooli's
// canonical credential authority. It deliberately retains only the authority
// handle and resolves on each session open so rotation applies without restart.
type AuthorityKeyResolver struct {
	authority authorityReader
	identity  credentialauthority.Identity
}

// LazyAuthorityKeyResolver obtains the default credential authority only when
// the provider is opened. Production composition can retain this resolver
// without making an unavailable or locked store a startup dependency.
type LazyAuthorityKeyResolver struct {
	newAuthority func() (authorityReader, error)
}

// NewLazyDefaultAuthorityKeyResolver constructs a call-time resolver for the
// canonical credential authority. The secret itself is resolved on every
// call, so store unlocks and key rotations are observed without a restart.
func NewLazyDefaultAuthorityKeyResolver() *LazyAuthorityKeyResolver {
	return &LazyAuthorityKeyResolver{newAuthority: func() (authorityReader, error) {
		return credentialauthority.Default()
	}}
}

func (r *LazyAuthorityKeyResolver) ResolveOpenAIKey(ctx context.Context) (string, error) {
	if r == nil || r.newAuthority == nil || ctx == nil || ctx.Err() != nil {
		return "", errors.New("duplex: OpenAI credential is unavailable")
	}
	authority, err := r.newAuthority()
	if err != nil || authority == nil {
		return "", errors.New("duplex: OpenAI credential is unavailable")
	}
	resolver, err := NewAuthorityKeyResolver(authority)
	if err != nil {
		return "", errors.New("duplex: OpenAI credential is unavailable")
	}
	return resolver.ResolveOpenAIKey(ctx)
}

func NewAuthorityKeyResolver(authority authorityReader) (*AuthorityKeyResolver, error) {
	if authority == nil {
		return nil, errors.New("duplex: credential authority is unavailable")
	}
	identity, err := credentialauthority.ParseIdentity(openAILiveCredentialIdentity)
	if err != nil {
		return nil, errors.New("duplex: OpenAI credential identity is invalid")
	}
	return &AuthorityKeyResolver{authority: authority, identity: identity}, nil
}

func (r *AuthorityKeyResolver) ResolveOpenAIKey(ctx context.Context) (string, error) {
	if r == nil || r.authority == nil || r.identity == "" || ctx == nil || ctx.Err() != nil {
		return "", errors.New("duplex: OpenAI credential is unavailable")
	}
	key, err := r.authority.Require(r.identity, openAILiveCredentialField)
	if err != nil || strings.TrimSpace(key) == "" {
		// Keep the secure-store failure taxonomy and any provider diagnostics out
		// of transport errors and logs at this provider boundary.
		return "", errors.New("duplex: OpenAI credential is unavailable")
	}
	return key, nil
}
