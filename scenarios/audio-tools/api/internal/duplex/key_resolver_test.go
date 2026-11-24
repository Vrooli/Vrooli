package duplex

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"

	"github.com/stretchr/testify/require"
)

type resolverAuthority struct {
	value    string
	err      error
	identity credentialauthority.Identity
	field    string
}

func TestAuthorityProviderErrorIsRedacted(t *testing.T) {
	const secret = "sk-never-include-this"
	provider := &OpenAILiveProvider{Keys: errorKeyResolver{err: errors.New("store failure: " + secret)}}
	_, openErr := provider.Open(context.Background(), Request{Model: "gpt-live-1.1"}, &eventSink{})
	require.Error(t, openErr)
	require.False(t, strings.Contains(openErr.Error(), secret))
	require.EqualError(t, openErr, "duplex: OpenAI credential resolution failed")
}

func (a *resolverAuthority) Require(identity credentialauthority.Identity, field string) (string, error) {
	a.identity, a.field = identity, field
	return a.value, a.err
}

func TestAuthorityKeyResolverUsesCanonicalAddressAndResolvesPerCall(t *testing.T) {
	authority := &resolverAuthority{value: "sk-authority-key"}
	resolver, err := NewAuthorityKeyResolver(authority)
	require.NoError(t, err)

	key, err := resolver.ResolveOpenAIKey(context.Background())
	require.NoError(t, err)
	require.Equal(t, "sk-authority-key", key)
	require.Equal(t, "api-key", authority.field)
	wantIdentity, err := credentialauthority.ParseIdentity("vrooli/openai-gpt-live")
	require.NoError(t, err)
	require.Equal(t, wantIdentity, authority.identity)

	authority.value = "sk-rotated-key"
	key, err = resolver.ResolveOpenAIKey(context.Background())
	require.NoError(t, err)
	require.Equal(t, "sk-rotated-key", key, "the key is read on every session open")
}

func TestLazyAuthorityKeyResolverDefersAuthorityAndSecretReadsUntilCalled(t *testing.T) {
	authority := &resolverAuthority{value: "sk-lazy-authority-key"}
	created := 0
	resolver := &LazyAuthorityKeyResolver{newAuthority: func() (authorityReader, error) {
		created++
		return authority, nil
	}}
	require.Zero(t, created, "constructing the production resolver must not access credentials")

	key, err := resolver.ResolveOpenAIKey(context.Background())
	require.NoError(t, err)
	require.Equal(t, "sk-lazy-authority-key", key)
	require.Equal(t, 1, created)
	wantIdentity, err := credentialauthority.ParseIdentity("vrooli/openai-gpt-live")
	require.NoError(t, err)
	require.Equal(t, wantIdentity, authority.identity)
	require.Equal(t, "api-key", authority.field)

	authority.value = "sk-rotated-lazy-key"
	key, err = resolver.ResolveOpenAIKey(context.Background())
	require.NoError(t, err)
	require.Equal(t, "sk-rotated-lazy-key", key)
	require.Equal(t, 2, created, "each provider open re-resolves through the authority")
}

func TestAuthorityKeyResolverReadsCredentialAuthorityEncryptedStore(t *testing.T) {
	authority, err := credentialauthority.OpenEncryptedFile(filepath.Join(t.TempDir(), "credentials.enc"), "test-only-passphrase")
	require.NoError(t, err)
	identity, err := credentialauthority.ParseIdentity("vrooli/openai-gpt-live")
	require.NoError(t, err)
	require.NoError(t, authority.Put(identity, "api-key", "sk-test-authority-value"))

	resolver, err := NewAuthorityKeyResolver(authority)
	require.NoError(t, err)
	key, err := resolver.ResolveOpenAIKey(context.Background())
	require.NoError(t, err)
	require.Equal(t, "sk-test-authority-value", key)
	// Having a securely provisioned credential must not promote an unqualified
	// provider/model pair. Billing evidence and the product authorization path
	// remain independent gates.
	require.False(t, CatalogEligibility("openai-gpt-live", "gpt-live-1.1"))
}

func TestAuthorityKeyResolverFailsClosedAndRedactsAuthorityErrors(t *testing.T) {
	const secret = "sk-never-include-this"
	for _, authority := range []*resolverAuthority{{}, {err: errors.New("provider failed for " + secret)}} {
		resolver, err := NewAuthorityKeyResolver(authority)
		require.NoError(t, err)
		key, resolveErr := resolver.ResolveOpenAIKey(context.Background())
		require.Empty(t, key)
		require.Error(t, resolveErr)
		require.NotContains(t, resolveErr.Error(), secret)
		require.EqualError(t, resolveErr, "duplex: OpenAI credential is unavailable")
	}
}

type errorKeyResolver struct{ err error }

func (r errorKeyResolver) ResolveOpenAIKey(context.Context) (string, error) { return "", r.err }
