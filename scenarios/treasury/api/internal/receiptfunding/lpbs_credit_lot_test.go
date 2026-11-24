package receiptfunding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/servicetoken"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	"google.golang.org/protobuf/encoding/protojson"
)

type receiptServiceToken string

func (t receiptServiceToken) Authorization(context.Context) (string, error) {
	if t == "" {
		return "", servicetoken.ErrNotIssued
	}
	return string(t), nil
}

type failedReceiptServiceToken struct{ err error }

func (t failedReceiptServiceToken) Authorization(context.Context) (string, error) {
	return "", t.err
}

type staticLPBSCredentialReader struct {
	t      *testing.T
	status credentialclient.CredentialStatus
	token  string
	reads  int
}

func (r *staticLPBSCredentialReader) Status(context.Context, string, string) (credentialclient.CredentialStatus, error) {
	return r.status, nil
}

func (r *staticLPBSCredentialReader) Resolve(_ context.Context, identity, field string) (string, error) {
	r.reads++
	require.Equal(r.t, LPBSServiceCredentialIdentity, identity)
	require.Equal(r.t, LPBSLegacyServiceCredentialField, field)
	return r.token, nil
}

func TestLPBSCreditLotClientSendsAcceptedReceiptAndReturnsActualOwnerID(t *testing.T) {
	var received ingestpb.AcceptedCryptoReceipt
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/v1/internal/crypto-funding/accepted-receipts", r.URL.Path)
		require.Equal(t, "Bearer scoped-lpbs-credit-token", r.Header.Get("Authorization"))
		require.Equal(t, "1", r.Header.Get("X-Vrooli-Test-Mode"))
		var body struct {
			Receipt json.RawMessage `json:"receipt"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.NoError(t, protojson.Unmarshal(body.Receipt, &received))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"credit_lot_id": "actual-lpbs-lot", "receipt_id": received.Id})
	}))
	defer server.Close()

	credentialReads := 0
	client, err := NewLPBSCreditLotClient(server.URL, receiptServiceToken("Bearer scoped-lpbs-credit-token"), LPBSCredentialFunc(func(_ context.Context, identity, field string) (string, error) {
		credentialReads++
		require.Equal(t, LPBSServiceCredentialIdentity, identity)
		require.Equal(t, LPBSLegacyServiceCredentialField, field)
		return "lpbs-test-service-credential", nil
	}), server.Client())
	require.NoError(t, err)
	link, err := client.GrantAcceptedReceipt(database.WithTestMode(context.Background()), &ingestpb.AcceptedCryptoReceipt{Id: "ledger-accepted-42"})
	require.NoError(t, err)
	require.Zero(t, credentialReads, "the scoped principal must be preferred over the migration fallback")
	require.Equal(t, "ledger-accepted-42", received.Id)
	require.Equal(t, CreditLotLink{ID: "actual-lpbs-lot", Owner: "landing-page-business-suite", ReceiptID: "ledger-accepted-42"}, link)
}

func TestLPBSCreditLotClientUsesLegacyCredentialOnlyWhenScopedIdentityUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer legacy-overlap-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"credit_lot_id":"lot-1","receipt_id":"receipt-1"}`))
	}))
	defer server.Close()
	legacy := &staticLPBSCredentialReader{t: t, status: credentialclient.CredentialStatus{ProviderState: "available"}, token: "legacy-overlap-token"}
	client, err := NewLPBSCreditLotClient(server.URL, receiptServiceToken(""), legacy, server.Client())
	require.NoError(t, err)
	_, err = client.GrantAcceptedReceipt(context.Background(), &ingestpb.AcceptedCryptoReceipt{Id: "receipt-1"})
	require.NoError(t, err)
	require.Equal(t, 1, legacy.reads)
}

func TestLPBSCreditLotClientDoesNotUseLegacyAfterScopedPrincipalIsConfigured(t *testing.T) {
	legacy := &staticLPBSCredentialReader{t: t, status: credentialclient.CredentialStatus{ProviderState: "available", Configured: true, Version: "v2"}, token: "must-not-be-read"}
	client, err := NewLPBSCreditLotClient("http://127.0.0.1:1", receiptServiceToken(""), legacy, nil)
	require.NoError(t, err)
	_, err = client.GrantAcceptedReceipt(context.Background(), &ingestpb.AcceptedCryptoReceipt{Id: "receipt-2"})
	require.Error(t, err)
	require.Zero(t, legacy.reads, "failed scoped token exchange must not downgrade to the shared bearer")
}

func TestLPBSCreditLotClientDoesNotDowngradeAfterScopedExchangeFailure(t *testing.T) {
	legacy := &staticLPBSCredentialReader{t: t, status: credentialclient.CredentialStatus{ProviderState: "available"}, token: "must-not-be-used"}
	client, err := NewLPBSCreditLotClient("http://127.0.0.1:1", failedReceiptServiceToken{err: servicetoken.ErrExchangeFailed}, legacy, nil)
	require.NoError(t, err)
	_, err = client.GrantAcceptedReceipt(context.Background(), &ingestpb.AcceptedCryptoReceipt{Id: "receipt-exchange-failure"})
	require.Error(t, err)
	require.Zero(t, legacy.reads, "exchange failure must not be treated as confirmed unissued")
}
