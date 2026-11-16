package identity

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/consumeridentity"
	apiidentity "github.com/vrooli/api-core/identity"
)

func TestLPBSConsumerProviderVerifiesAccountSubjectAndFailsClosed(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	signer, err := consumeridentity.NewSigner("fixture-key", key, "lpbs.fixture", time.Hour)
	require.NoError(t, err)
	token, _, err := signer.Sign(consumeridentity.Claims{UserID: "account-42", Subject: "account-42", Email: "user@example.test"})
	require.NoError(t, err)
	keys, err := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: "fixture-key", Key: &key.PublicKey}).JWKS()
	require.NoError(t, err)
	provider := NewLPBSConsumerProvider("lpbs.fixture")
	provider.cache = consumeridentity.NewCache("lpbs.fixture", 0, time.Hour, func(*http.Request) ([]byte, error) { return keys, nil })

	request := httptest.NewRequest(http.MethodPost, "/consent", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	principal, err := provider.VerifyRequest(context.Background(), request)
	require.NoError(t, err)
	require.True(t, principal.IsHuman())
	require.Equal(t, "account-42", principal.Subject)
	require.Equal(t, apiidentity.SourceConsumerAccessToken, principal.Source)

	for _, raw := range []string{"", "Bearer forged", "Bearer " + token[:len(token)-4] + "xxxx"} {
		bad := httptest.NewRequest(http.MethodPost, "/consent", nil)
		if raw != "" {
			bad.Header.Set("Authorization", raw)
		}
		_, err := provider.VerifyRequest(context.Background(), bad)
		require.Error(t, err)
	}
}
