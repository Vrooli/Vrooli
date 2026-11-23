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

func TestLPBSConsumerProviderVerifiesScopedCreditAssertion(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	now := time.Now().UTC()
	signer, err := consumeridentity.NewSigner("fixture-key", key, "lpbs.fixture", time.Minute)
	require.NoError(t, err)
	signer.Now = func() time.Time { return now }
	jwks, err := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: signer.KeyID, Key: &key.PublicKey}).JWKS()
	require.NoError(t, err)
	provider := &LPBSConsumerProvider{cache: consumeridentity.NewCache("lpbs.fixture", 0, time.Hour, func(*http.Request) ([]byte, error) { return jwks, nil })}
	expect := consumeridentity.CreditAssertionExpectation{BusinessAccountID: "acct-1", ActorID: "actor-1", AppKey: "audio-tools", FeatureID: "audio-tools.summarize", MaxCredits: 500, Audience: consumeridentity.CreditGatewayAudience, OperationID: "op-1", Source: consumeridentity.CreditAssertionSource}
	token, _, err := signer.SignCreditAssertion(expect, 20*time.Second)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/private/provider-routes/plan", nil)
	require.NoError(t, provider.VerifyCreditAssertion(req, token, expect))
	wrong := expect
	wrong.AppKey = "web-console"
	require.Error(t, provider.VerifyCreditAssertion(req, token, wrong))
	wrong = expect
	wrong.OperationID = "op-replay"
	require.Error(t, provider.VerifyCreditAssertion(req, token, wrong))
	wrong = expect
	wrong.Audience = "scenario:landing-page-business-suite"
	require.Error(t, provider.VerifyCreditAssertion(req, token, wrong))
}

func TestLPBSRunCallerAssertionBindsVerifiedPrincipalToRoleAndOperation(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	signer, err := consumeridentity.NewSigner("fixture-key", key, "lpbs.fixture", time.Minute)
	require.NoError(t, err)
	expect := consumeridentity.CreditAssertionExpectation{
		BusinessAccountID: "account-a", ActorID: "actor-a", AppKey: "web-console",
		FeatureID: "ai-gateway.run/extract.structured", MaxCredits: 500,
		Audience: consumeridentity.CreditGatewayAudience, OperationID: "operation-a",
		Source: consumeridentity.CreditAssertionSource,
	}
	token, _, err := signer.SignCreditAssertion(expect, 20*time.Second)
	require.NoError(t, err)
	jwks, err := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: signer.KeyID, Key: &key.PublicKey}).JWKS()
	require.NoError(t, err)
	provider := &LPBSConsumerProvider{cache: consumeridentity.NewCache("lpbs.fixture", 0, time.Hour, func(*http.Request) ([]byte, error) { return jwks, nil })}
	req := httptest.NewRequest(http.MethodPost, "/ai-gateway.v1.inference.InferenceService/Run", nil)

	caller, err := provider.VerifyRunCallerAssertion(req, token, "extract.structured", "operation-a")
	require.NoError(t, err)
	require.Equal(t, RunCaller{BusinessAccountID: "account-a", ActorID: "actor-a", AppKey: "web-console", OperationID: "operation-a"}, caller)

	_, err = provider.VerifyRunCallerAssertion(req, token, "classify.fast", "operation-a")
	require.Error(t, err, "role must match the LPBS-signed scope")
	_, err = provider.VerifyRunCallerAssertion(req, token, "extract.structured", "operation-b")
	require.Error(t, err, "root operation id must match the LPBS-signed scope")
	_, err = provider.VerifyRunCallerAssertion(req, "forged", "extract.structured", "operation-a")
	require.Error(t, err)
}

func TestLPBSConsumerProviderAcceptsPublishedRotationOverlapAcrossRestart(t *testing.T) {
	oldSigner, err := consumeridentity.GenerateSigner("old-key", "lpbs.fixture", time.Minute)
	require.NoError(t, err)
	newSigner, err := consumeridentity.GenerateSigner("new-key", "lpbs.fixture", time.Minute)
	require.NoError(t, err)
	oldToken, _, err := oldSigner.SignCreditAssertion(testCreditExpectation("operation-old"), 20*time.Second)
	require.NoError(t, err)
	newToken, _, err := newSigner.SignCreditAssertion(testCreditExpectation("operation-new"), 20*time.Second)
	require.NoError(t, err)

	overlap, err := consumeridentity.NewKeySet(
		consumeridentity.PublicKey{ID: oldSigner.KeyID, Key: &oldSigner.Private.PublicKey},
		consumeridentity.PublicKey{ID: newSigner.KeyID, Key: &newSigner.Private.PublicKey},
	).JWKS()
	require.NoError(t, err)
	fetch := func(*http.Request) ([]byte, error) { return overlap, nil }
	first := NewLPBSConsumerProvider("lpbs.fixture")
	first.cache = consumeridentity.NewCache("lpbs.fixture", 0, time.Hour, fetch)
	req := httptest.NewRequest(http.MethodPost, "/private", nil)
	require.NoError(t, first.VerifyCreditAssertion(req, oldToken, testCreditExpectation("operation-old")))
	require.NoError(t, first.VerifyCreditAssertion(req, newToken, testCreditExpectation("operation-new")))

	// A restarted Gateway has no warm key cache and must recover both sides of
	// the overlap from the issuer's published JWKS before it can admit work.
	restarted := NewLPBSConsumerProvider("lpbs.fixture")
	restarted.cache = consumeridentity.NewCache("lpbs.fixture", 0, time.Hour, fetch)
	require.NoError(t, restarted.VerifyCreditAssertion(req, oldToken, testCreditExpectation("operation-old")))
	require.NoError(t, restarted.VerifyCreditAssertion(req, newToken, testCreditExpectation("operation-new")))
}

func testCreditExpectation(operation string) consumeridentity.CreditAssertionExpectation {
	return consumeridentity.CreditAssertionExpectation{
		BusinessAccountID: "acct-1", ActorID: "actor-1", AppKey: "audio-tools",
		FeatureID: "audio-tools.summarize", MaxCredits: 500,
		Audience: consumeridentity.CreditGatewayAudience, OperationID: operation,
		Source: consumeridentity.CreditAssertionSource,
	}
}
