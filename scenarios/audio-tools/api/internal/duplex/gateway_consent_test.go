package duplex

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/identity"
)

const testConsentSource = "fixture-provider-consent-source"

func TestGatewayConsentVerifierForwardsIdentityProofAndDoesNotExposeReceipt(t *testing.T) {
	var receiptSeen, authSeen, sourceSeen string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var input struct {
			Receipt string `json:"receipt"`
			Source  string `json:"source"`
		}
		_ = json.Unmarshal(body, &input)
		receiptSeen, sourceSeen, authSeen = input.Receipt, input.Source, r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":true}`))
	}))
	defer server.Close()
	verifier, err := NewGatewayConsentVerifier(server.URL, server.Client())
	require.NoError(t, err)
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "human", Source: identity.SourceConsumerAccessToken, Verified: true})
	ctx, err = WithVerifiedHumanBearer(ctx, "Bearer human-token")
	require.NoError(t, err)
	require.NoError(t, verifier.Verify(ctx, "human", "opaque-receipt", testConsentSource))
	require.Equal(t, "opaque-receipt", receiptSeen)
	require.Equal(t, testConsentSource, sourceSeen)
	require.Equal(t, "Bearer human-token", authSeen)
	require.Error(t, verifier.Verify(context.Background(), "human", "opaque-receipt", testConsentSource))
	require.Error(t, verifier.Verify(ctx, "human", "opaque-receipt", ""))
}

func TestGatewayConsentVerifierResolvesGatewayAtCheckTime(t *testing.T) {
	var resolved int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"valid":true}`))
	}))
	defer server.Close()
	verifier, err := NewGatewayConsentVerifierWithResolver(func(context.Context) (string, error) {
		resolved++
		return server.URL, nil
	}, server.Client())
	require.NoError(t, err)
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "human", Source: identity.SourceConsumerAccessToken, Verified: true})
	ctx, err = WithVerifiedHumanBearer(ctx, "Bearer human-token")
	require.NoError(t, err)
	require.NoError(t, verifier.Verify(ctx, "human", "opaque-receipt", testConsentSource))
	require.Equal(t, 1, resolved)
}

func TestGatewayConsentVerifierDoesNotForwardReceiptAcrossRedirect(t *testing.T) {
	var targetCalls int
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		targetCalls++
		_, _ = w.Write([]byte(`{"valid":true}`))
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Redirect(w, &http.Request{}, target.URL, http.StatusTemporaryRedirect)
	}))
	defer source.Close()

	client := source.Client()
	verifier, err := NewGatewayConsentVerifier(source.URL, client)
	require.NoError(t, err)
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "human", Source: identity.SourceConsumerAccessToken, Verified: true})
	ctx, err = WithVerifiedHumanBearer(ctx, "Bearer human-token")
	require.NoError(t, err)
	require.ErrorIs(t, verifier.Verify(ctx, "human", "receipt-that-must-not-leak", testConsentSource), ErrConsentInvalid)
	require.Zero(t, targetCalls, "receipt body must not be forwarded to a redirect target")
	require.Nil(t, client.CheckRedirect, "constructor must not mutate a shared HTTP client")
}

func TestGatewayConsentVerifierWithResolverDoesNotForwardReceiptAcrossRedirect(t *testing.T) {
	var targetCalls int
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		targetCalls++
		_, _ = w.Write([]byte(`{"valid":true}`))
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer source.Close()

	verifier, err := NewGatewayConsentVerifierWithResolver(func(context.Context) (string, error) {
		return source.URL, nil
	}, source.Client())
	require.NoError(t, err)
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "human", Source: identity.SourceConsumerAccessToken, Verified: true})
	ctx, err = WithVerifiedHumanBearer(ctx, "Bearer human-token")
	require.NoError(t, err)
	require.ErrorIs(t, verifier.Verify(ctx, "human", "receipt-that-must-not-leak", testConsentSource), ErrConsentInvalid)
	require.Zero(t, targetCalls, "receipt body must not be forwarded to a redirect target")
}
