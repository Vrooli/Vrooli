package settlement_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"

	"treasury/internal/instrument"
	"treasury/internal/mandate"
	"treasury/internal/rail"
	x402rail "treasury/internal/rail/x402"
	"treasury/internal/settlement"
)

// This joins the production SQLite repository, the instrument Credential
// Authority resolver, and the x402 HTTP adapter around the exact crash cut
// after a possible paid dispatch but before terminal completion.
func TestX402CrashRecoveryUsesProtectedInputAcrossSQLiteAndAdapter(t *testing.T) {
	var endpoint string
	var paidRequests atomic.Int64
	var firstPayload, firstIdempotencyKey string
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Payment-Signature") == "" {
			required := map[string]any{
				"x402Version": 2,
				"resource":    map[string]string{"url": endpoint},
				"accepts": []any{map[string]any{
					"scheme": "exact", "network": "eip155:8453", "amount": "2500000",
					"payTo": "0x3333333333333333333333333333333333333333", "maxTimeoutSeconds": 300,
					"asset": "0x2222222222222222222222222222222222222222",
					"extra": map[string]string{"assetTransferMethod": "eip3009", "name": "USD Coin", "version": "2"},
				}},
			}
			encoded, err := json.Marshal(required)
			if err != nil {
				t.Errorf("marshal synthetic offer: %v", err)
				writer.WriteHeader(http.StatusInternalServerError)
				return
			}
			writer.Header().Set("Payment-Required", base64.StdEncoding.EncodeToString(encoded))
			writer.WriteHeader(http.StatusPaymentRequired)
			return
		}
		payload := request.Header.Get("Payment-Signature")
		if request.Header.Get("X-Treasury-Idempotency-Key") != "settle-key-1" {
			t.Errorf("unexpected idempotency key %q", request.Header.Get("X-Treasury-Idempotency-Key"))
		}
		switch paidRequests.Add(1) {
		case 1:
			firstPayload = payload
			firstIdempotencyKey = request.Header.Get("X-Treasury-Idempotency-Key")
			writer.WriteHeader(http.StatusInternalServerError) // possible accepted payment, lost conclusive receipt
			return
		case 2:
			writer.WriteHeader(http.StatusInternalServerError) // keep reconciliation unknown
			return
		}
		if request.Header.Get("X-Treasury-Idempotency-Key") != firstIdempotencyKey {
			t.Errorf("replay changed idempotency key")
		}
		if payload != firstPayload {
			t.Errorf("replay changed signed payload")
		}
		receipt, err := json.Marshal(map[string]any{"success": true, "payer": "0x1111111111111111111111111111111111111111", "transaction": "0xconfirmed", "network": "eip155:8453"})
		if err != nil {
			t.Errorf("marshal synthetic receipt: %v", err)
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		writer.Header().Set("Payment-Response", base64.StdEncoding.EncodeToString(receipt))
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	endpoint = "https://api.example/priced"
	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	clientHTTP := &http.Client{Transport: rewriteTransport{base: server.Client().Transport, target: serverURL}}
	signer := &syntheticTypedDataSigner{}
	x402Adapter, err := x402rail.New(clientHTTP, signer)
	require.NoError(t, err)
	registry, err := rail.NewRegistry(x402Adapter)
	require.NoError(t, err)
	fixture := newFixture(t, &adapter{name: "x402"})
	fixture.registry = registry

	authority, err := credentialauthority.OpenEncryptedFile(filepath.Join(t.TempDir(), "credentials.enc"), "test-only-passphrase")
	require.NoError(t, err)
	credentialClient, err := credentialclient.NewInProcess(credentialclient.InProcessOptions{Authority: authority})
	require.NoError(t, err)
	identity, err := credentialauthority.ParseIdentity("treasury/test-credential")
	require.NoError(t, err)
	credentialJSON, err := json.Marshal(map[string]any{
		"endpoint_url": endpoint, "signer_url": "http://127.0.0.1:8545",
		"account": "0x1111111111111111111111111111111111111111", "networks": []string{"eip155:8453"},
		"assets": map[string]any{"0x2222222222222222222222222222222222222222": map[string]any{"decimals": 6, "currency": "USD"}},
	})
	require.NoError(t, err)
	_, err = credentialClient.Provision(context.Background(), credentialclient.ProvisionRequest{Identity: string(identity), Field: "value", Value: string(credentialJSON)})
	require.NoError(t, err)
	credentialResolver, err := instrument.NewCredentialClientResolver(credentialClient)
	require.NoError(t, err)
	mandateSigner, err := mandate.NewHMACSigner([]byte("settlement-test-signing-key"))
	require.NoError(t, err)
	mandates := mandate.NewService(mandate.NewSQLiteRepository(fixture.db), fixture.clock, mandateSigner)
	instruments := instrument.NewService(instrument.NewSQLiteRepository(fixture.db), mandates, registry, credentialResolver, fixture.clock)
	repository := &failOnceCompleteRepository{Repository: fixture.settlements}
	fixture.service = settlement.NewService(repository, fixture.authorizations, instruments, registry, identityVerifier{}, fixture.clock)

	_, err = fixture.service.Settle(context.Background(), fixture.input)
	require.ErrorContains(t, err, "simulated completion interruption")
	orphan, err := fixture.settlements.Get(context.Background(), fixture.input.ID)
	require.NoError(t, err)
	require.Equal(t, settlement.OutcomeCalling, orphan.Outcome)
	require.NotEmpty(t, orphan.ExternalID)
	require.NotEmpty(t, orphan.ReceiptReference)
	require.NotContains(t, orphan.ReceiptReference, firstPayload)
	require.EqualValues(t, 1, paidRequests.Load())

	_, err = fixture.service.Settle(context.Background(), fixture.input)
	require.ErrorIs(t, err, settlement.ErrInProgress)

	stillUnknown, err := fixture.service.ResolveUnknown(context.Background(), orphan.ID)
	require.NoError(t, err)
	require.Equal(t, settlement.OutcomeUnknown, stillUnknown.Outcome)
	require.Equal(t, orphan.ReceiptReference, stillUnknown.ReceiptReference, "inconclusive query must preserve only the Credential Authority reference")
	require.EqualValues(t, 2, paidRequests.Load())
	var storedPayload bool
	err = fixture.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM settlements WHERE id=? AND (external_id=? OR receipt_reference=?))`, fixture.input.ID, firstPayload, firstPayload).Scan(&storedPayload)
	require.NoError(t, err)
	require.False(t, storedPayload, "an inconclusive replay must not write the signed authorization to SQLite")

	resolved, err := fixture.service.ResolveUnknown(context.Background(), orphan.ID)
	require.NoError(t, err)
	require.Equal(t, settlement.OutcomeSettled, resolved.Outcome)
	require.Equal(t, "0xconfirmed", resolved.ExternalID)
	require.EqualValues(t, 3, paidRequests.Load())
	require.EqualValues(t, 1, signer.calls.Load(), "reconciliation replays the persisted signature without signing again")
	require.Equal(t, firstIdempotencyKey, fixture.input.IdempotencyKey)
	decoded, err := base64.StdEncoding.DecodeString(firstPayload)
	require.NoError(t, err)
	payloadDigest := sha256.Sum256([]byte(firstPayload))
	require.Equal(t, "sha256:"+hex.EncodeToString(payloadDigest[:]), orphan.ExternalID)
	require.Contains(t, string(decoded), endpoint)

	err = fixture.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM settlements WHERE id=? AND (external_id=? OR receipt_reference=?))`, fixture.input.ID, firstPayload, firstPayload).Scan(&storedPayload)
	require.NoError(t, err)
	require.False(t, storedPayload, "SQLite must never retain the signed authorization")
	_, err = credentialResolver.ResolveSettlementRecoveryInput(context.Background(), orphan.ReceiptReference)
	require.Error(t, err, "terminal reconciliation deletes the protected replay credential")
}

type failOnceCompleteRepository struct {
	settlement.Repository
	failed atomic.Bool
}

func (r *failOnceCompleteRepository) Complete(ctx context.Context, id string, outcome settlement.Outcome, result settlement.RailResult, updatedAt, retainUntil string, artifacts ...settlement.CompletionArtifacts) (settlement.Record, error) {
	if r.failed.CompareAndSwap(false, true) {
		return settlement.Record{}, errors.New("simulated completion interruption")
	}
	return r.Repository.Complete(ctx, id, outcome, result, updatedAt, retainUntil, artifacts...)
}

type syntheticTypedDataSigner struct{ calls atomic.Int64 }

func (s *syntheticTypedDataSigner) SignTypedData(context.Context, x402rail.Credential, map[string]any) (string, error) {
	s.calls.Add(1)
	return "0x" + strings.Repeat("11", 65), nil
}

type rewriteTransport struct {
	base   http.RoundTripper
	target *url.URL
}

func (r rewriteTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	copy := request.Clone(request.Context())
	urlCopy := *request.URL
	urlCopy.Scheme = r.target.Scheme
	urlCopy.Host = r.target.Host
	copy.URL = &urlCopy
	copy.Host = r.target.Host
	return r.base.RoundTrip(copy)
}
