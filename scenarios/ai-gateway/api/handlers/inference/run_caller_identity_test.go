package inference_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/consumeridentity"
	dbtest "github.com/vrooli/api-core/databasetest"
	apiidentity "github.com/vrooli/api-core/identity"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"

	handler "ai-gateway/handlers/inference"
	gatewayidentity "ai-gateway/internal/identity"
	"ai-gateway/internal/inference"
)

type runCallerCaptureRepository struct {
	calls     int
	principal apiidentity.Principal
	caller    gatewayidentity.RunCaller
	digests   []string
}

func (r *runCallerCaptureRepository) Run(ctx context.Context, request inference.ProviderRequest) (inference.ProviderResult, error) {
	r.calls++
	r.principal, _ = apiidentity.PrincipalFromContext(ctx)
	r.caller, _ = gatewayidentity.RunCallerFromContext(ctx)
	r.digests = append(r.digests, request.RequestDigest)
	return inference.ProviderResult{ValueJSON: `{}`}, nil
}

func TestRunCarriesOnlyVerifiedLPBSCallerIntoProviderContext(t *testing.T) {
	provider, token := signedRunCallerProvider(t)
	repository := &runCallerCaptureRepository{}
	run := &inferencev1.RunRequest{
		RootOperationId: "run-operation-00000000000000000000000000000000",
		Source:          "synthetic public prompt", SchemaJson: `{"type":"object"}`, Instruction: "Return an empty object.",
		Role: "extract.structured", PrivacyProvenance: "ac3_fixture.explicit_public_input",
	}
	request := connect.NewRequest(run)
	request.Header().Set(gatewayidentity.RunCallerAssertionHeader, token)
	response, err := handler.NewConnectHandler(inference.NewService(repository), provider).Run(t.Context(), request)
	require.NoError(t, err)
	require.True(t, response.Msg.GetValidated())
	require.Equal(t, 1, repository.calls)
	require.True(t, repository.principal.IsHuman())
	require.True(t, repository.principal.Verified)
	require.Equal(t, "actor-a", repository.principal.Subject)
	require.Equal(t, "account-a", repository.principal.Owner)
	require.Equal(t, "account-a", repository.caller.BusinessAccountID)
	require.Equal(t, "actor-a", repository.caller.ActorID)
	require.Equal(t, run.GetRootOperationId(), repository.caller.OperationID)
}

func TestRunCarriesLPBSAssertionAcrossConnectHTTPBoundary(t *testing.T) {
	provider, token := signedRunCallerProvider(t)
	repository := &runCallerCaptureRepository{}
	path, route := inferenceconnect.NewInferenceServiceHandler(handler.NewConnectHandler(inference.NewService(repository), provider))
	mux := http.NewServeMux()
	mux.Handle(path, route)
	server := httptest.NewServer(authn.Middleware(authn.Config{Providers: []authn.Provider{provider}})(mux))
	t.Cleanup(server.Close)
	client := inferenceconnect.NewInferenceServiceClient(server.Client(), server.URL)
	request := connect.NewRequest(&inferencev1.RunRequest{
		RootOperationId: "run-operation-00000000000000000000000000000000",
		Source:          "synthetic public prompt", SchemaJson: `{"type":"object"}`, Instruction: "Return an empty object.",
		Role: "extract.structured", PrivacyProvenance: "ac3_fixture.explicit_public_input",
	})
	request.Header().Set(gatewayidentity.RunCallerAssertionHeader, token)
	request.Header().Set("Authorization", "Bearer "+token)
	response, err := client.Run(t.Context(), request)
	require.NoError(t, err)
	require.True(t, response.Msg.GetValidated())
	require.Equal(t, 1, repository.calls)
	require.True(t, repository.principal.Verified)
	require.Equal(t, "account-a", repository.principal.Owner)
	require.Equal(t, "actor-a", repository.principal.Subject)
}

func TestRunRejectsForgedLPBSCallerBeforeProviderDispatch(t *testing.T) {
	provider, _ := signedRunCallerProvider(t)
	repository := &runCallerCaptureRepository{}
	request := connect.NewRequest(&inferencev1.RunRequest{
		RootOperationId: "run-operation-00000000000000000000000000000000",
		Source:          "synthetic public prompt", SchemaJson: `{"type":"object"}`, Instruction: "Return an empty object.",
		Role: "extract.structured", PrivacyProvenance: "ac3_fixture.explicit_public_input",
	})
	request.Header().Set(gatewayidentity.RunCallerAssertionHeader, "forged-token")
	_, err := handler.NewConnectHandler(inference.NewService(repository), provider).Run(t.Context(), request)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.Zero(t, repository.calls, "invalid identity must fail before inference provider runs")
}

func signedRunCallerProvider(t *testing.T) (*gatewayidentity.LPBSConsumerProvider, string) {
	return signedRunCallerProviderForAccount(t, "account-a", "actor-a")
}

func signedRunCallerProviderForAccount(t *testing.T, account, actor string) (*gatewayidentity.LPBSConsumerProvider, string) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	signer, err := consumeridentity.NewSigner("fixture-key", privateKey, "lpbs.fixture", time.Minute)
	require.NoError(t, err)
	operation := "run-operation-00000000000000000000000000000000"
	expect := consumeridentity.CreditAssertionExpectation{
		BusinessAccountID: account, ActorID: actor, AppKey: "web-console",
		FeatureID: "ai-gateway.run/extract.structured", MaxCredits: 500,
		Audience: consumeridentity.CreditGatewayAudience, OperationID: operation,
		Source: consumeridentity.CreditAssertionSource,
	}
	token, _, err := signer.SignCreditAssertion(expect, 20*time.Second)
	require.NoError(t, err)
	jwks, err := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: signer.KeyID, Key: &privateKey.PublicKey}).JWKS()
	require.NoError(t, err)
	provider := gatewayidentity.NewLPBSConsumerProviderWithCache("lpbs.fixture", consumeridentity.NewCache("lpbs.fixture", 0, time.Hour, func(*http.Request) ([]byte, error) { return jwks, nil }))
	return provider, token
}

func TestRunWithoutLPBSAssertionDoesNotInventHumanPrincipal(t *testing.T) {
	repository := &runCallerCaptureRepository{}
	request := connect.NewRequest(&inferencev1.RunRequest{
		Source: "synthetic public prompt", SchemaJson: `{"type":"object"}`, Instruction: "Return an empty object.",
		Role: "extract.structured", PrivacyProvenance: "ac3_fixture.explicit_public_input",
	})
	_, err := handler.NewConnectHandler(inference.NewService(repository)).Run(t.Context(), request)
	require.NoError(t, err)
	require.Equal(t, 1, repository.calls)
	require.False(t, repository.principal.Verified)
}

func TestRunRecoveryDigestIsBoundToVerifiedAccount(t *testing.T) {
	provider, accountAToken, accountBToken := signedRunCallersProvider(t)
	root := "run-operation-00000000000000000000000000000000"
	requestFor := func(token string) *connect.Request[inferencev1.RunRequest] {
		request := connect.NewRequest(&inferencev1.RunRequest{
			RootOperationId: root, Source: "synthetic public prompt", SchemaJson: `{"type":"object"}`,
			Instruction: "Return an empty object.", Role: "extract.structured", PrivacyProvenance: "ac3_fixture.explicit_public_input",
		})
		request.Header().Set(gatewayidentity.RunCallerAssertionHeader, token)
		return request
	}
	repository := &runCallerCaptureRepository{}
	service := inference.NewService(repository)
	runHandler := handler.NewConnectHandler(service, provider)
	_, err := runHandler.Run(t.Context(), requestFor(accountAToken))
	require.NoError(t, err)
	_, err = runHandler.Run(t.Context(), requestFor(accountBToken))
	require.NoError(t, err)
	require.Len(t, repository.digests, 2)
	require.NotEqual(t, repository.digests[0], repository.digests[1])
	db := dbtest.NewSQLite(t)
	_, err = db.ExecContext(t.Context(), inference.AttemptSchema())
	require.NoError(t, err)
	ledger := inference.NewSQLAttemptLedger(db)
	_, err = ledger.Begin(t.Context(), root, repository.digests[0], "fixture-device-generation", 0, 0, "metered", "extract.structured")
	require.NoError(t, err)
	_, err = ledger.Begin(t.Context(), root, repository.digests[1], "fixture-device-generation", 0, 0, "metered", "extract.structured")
	require.ErrorIs(t, err, inference.ErrRootConflict, "the shared hosted ledger must reject another account's receipt lookup")
}

func signedRunCallersProvider(t *testing.T) (*gatewayidentity.LPBSConsumerProvider, string, string) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	signer, err := consumeridentity.NewSigner("fixture-key", privateKey, "lpbs.fixture", time.Minute)
	require.NoError(t, err)
	issue := func(account, actor string) string {
		token, _, signErr := signer.SignCreditAssertion(consumeridentity.CreditAssertionExpectation{
			BusinessAccountID: account, ActorID: actor, AppKey: "web-console",
			FeatureID: "ai-gateway.run/extract.structured", MaxCredits: 500,
			Audience: consumeridentity.CreditGatewayAudience, OperationID: "run-operation-00000000000000000000000000000000",
			Source: consumeridentity.CreditAssertionSource,
		}, 20*time.Second)
		require.NoError(t, signErr)
		return token
	}
	jwks, err := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: signer.KeyID, Key: &privateKey.PublicKey}).JWKS()
	require.NoError(t, err)
	provider := gatewayidentity.NewLPBSConsumerProviderWithCache("lpbs.fixture", consumeridentity.NewCache("lpbs.fixture", 0, time.Hour, func(*http.Request) ([]byte, error) { return jwks, nil }))
	return provider, issue("account-a", "actor-a"), issue("account-b", "actor-b")
}
