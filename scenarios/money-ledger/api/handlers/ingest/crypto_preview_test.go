package ingest_test

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	rpc "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest/ingest_v1connect"

	ingestion "money-ledger/handlers/ingest"
	"money-ledger/internal/ledger"
)

func TestCryptoPreviewTransportPreservesUnitsAndNeverAcceptsApprovalOrWrites(t *testing.T) {
	sqlDB := databasetest.NewSQLite(t)
	db := database.NewFromPrimary(sqlDB)
	ctx := context.Background()
	require.NoError(t, database.EnsureSchemas(ctx, sqlDB, database.SchemaProviderFunc(func() string { return (&ledger.Store{}).Schema() })))
	service := ingestion.NewService(db, log.New(io.Discard, "", 0))
	_, handler := rpc.NewIngestServiceHandler(service)
	server := httptest.NewServer(handler)
	defer server.Close()
	client := rpc.NewIngestServiceClient(server.Client(), server.URL)
	in := &pb.PreviewCryptoObservationRequest{Network: "eip155:84532", AssetId: "eip155:84532/slip44:60", Address: "0x" + strings.Repeat("1", 40), AtomicAmount: "5901522149285533025181", Decimals: 18, State: pb.CryptoObservationState_CRYPTO_OBSERVATION_STATE_FINALIZED, LegKind: pb.CryptoLegKind_CRYPTO_LEG_KIND_BALANCE, SourceReference: "fixture:transport-not-owned", BlockHash: "0x" + strings.Repeat("2", 64)}
	out, err := client.PreviewCryptoObservation(ctx, connect.NewRequest(in))
	require.NoError(t, err)
	require.Equal(t, in.AtomicAmount, out.Msg.AtomicAmount)
	require.Equal(t, "withheld", out.Msg.PostingState)
	require.Equal(t, "uncredited", out.Msg.BudgetState)
	// Unknown caller assertions cannot turn this read-only contract into approval.
	body := `{"network":"eip155:84532","assetId":"eip155:84532/slip44:60","address":"0x1111111111111111111111111111111111111111","atomicAmount":"1","decimals":18,"state":3,"legKind":1,"sourceReference":"fixture:untrusted-approval","blockHash":"0x2222222222222222222222222222222222222222222222222222222222222222","ownershipApproved":true,"valuationApproved":true}`
	req, err := http.NewRequest(http.MethodPost, server.URL+rpc.IngestServicePreviewCryptoObservationProcedure, bytes.NewBufferString(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connect-Protocol-Version", "1")
	response, err := server.Client().Do(req)
	require.NoError(t, err)
	defer response.Body.Close()
	payload, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	if response.StatusCode == 200 {
		require.Contains(t, string(payload), `"postingState":"withheld"`)
		require.Contains(t, string(payload), `"budgetState":"uncredited"`)
	} else {
		require.Equal(t, http.StatusBadRequest, response.StatusCode)
	}
	for _, table := range []string{"postings", "ingest_receipts", "adapters", "ledger_audit"} {
		var count int
		require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table).Scan(&count))
		require.Zero(t, count)
	}
}
