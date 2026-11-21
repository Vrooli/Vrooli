package receiptfunding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestLPBSCreditLotClientSendsAcceptedReceiptAndReturnsActualOwnerID(t *testing.T) {
	var received ingestpb.AcceptedCryptoReceipt
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/v1/internal/crypto-funding/accepted-receipts", r.URL.Path)
		require.Equal(t, "Bearer lpbs-test-service-credential", r.Header.Get("Authorization"))
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
	client, err := NewLPBSCreditLotClient(server.URL, LPBSCredentialFunc(func(_ context.Context, identity, field string) (string, error) {
		credentialReads++
		require.Equal(t, LPBSServiceCredentialIdentity, identity)
		require.Equal(t, "service-secret", field)
		return "lpbs-test-service-credential", nil
	}), server.Client())
	require.NoError(t, err)
	link, err := client.GrantAcceptedReceipt(database.WithTestMode(context.Background()), &ingestpb.AcceptedCryptoReceipt{Id: "ledger-accepted-42"})
	require.NoError(t, err)
	require.Equal(t, 1, credentialReads)
	require.Equal(t, "ledger-accepted-42", received.Id)
	require.Equal(t, CreditLotLink{ID: "actual-lpbs-lot", Owner: "landing-page-business-suite", ReceiptID: "ledger-accepted-42"}, link)
}
