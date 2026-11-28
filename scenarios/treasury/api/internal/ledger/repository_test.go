package ledger

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestListEmissionsFiltersExactIdentityAndStatus(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(Schema())
	require.NoError(t, err)
	for _, row := range []struct{ id, externalID, status string }{
		{"pending", "base:0xabc:4", "queued"},
		{"accepted", "base:0xabc:4", "accepted"},
		{"other", "base:0xdef:2", "accepted"},
	} {
		_, err = db.Exec(`INSERT INTO ledger_emissions(id,settlement_id,external_id,adapter_id,account_id,book_id,amount_minor,currency,basis,occurred_at,fetched_at,description,status,attempts,last_error,created_at,accepted_at) VALUES(?,?,?,'treasury','account','book',-1200,'USD','authoritative','2026-11-01T12:00:00Z','2026-11-01T12:00:01Z','test',?,1,'',?,'')`, row.id, row.id, row.externalID, row.status, "2026-11-01T12:00:02Z")
		require.NoError(t, err)
	}

	values, err := NewSQLiteRepository(db).List(context.Background(), " base:0xabc:4 ", "ACCEPTED", 20)
	require.NoError(t, err)
	require.Len(t, values, 1)
	require.Equal(t, "accepted", values[0].ID)
	require.Equal(t, "base:0xabc:4", values[0].ExternalID)
	require.Equal(t, StatusAccepted, values[0].Status)
	require.ErrorContains(t, func() error {
		_, err := NewSQLiteRepository(db).List(context.Background(), "", "failed", 10)
		return err
	}(), "unsupported ledger emission status")
}

func TestAcceptedReceiptReadbackRequiresOneAcceptedPosting(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(Schema())
	require.NoError(t, err)
	insert := `INSERT INTO ledger_emissions(id,settlement_id,external_id,adapter_id,account_id,book_id,amount_minor,currency,basis,occurred_at,fetched_at,description,status,attempts,last_error,created_at,accepted_at,posting_id,receipt_id) VALUES(?,?,?,'treasury','account','book',100,'USD','authoritative','2026-11-14T12:00:00Z','2026-11-14T12:00:00Z','x402 receipt',?,?, '', '2026-11-14T12:00:00Z','2026-11-14T12:00:00Z',?,?)`
	_, err = db.Exec(insert, "queued", "queued", "treasury-x402-inflow:0xabc", "queued", 0, "", "")
	require.NoError(t, err)
	repo := NewSQLiteRepository(db)
	_, err = repo.GetAcceptedReceiptByExternalID(context.Background(), "treasury-x402-inflow:0xabc")
	require.ErrorIs(t, err, ErrNotFound)
	_, err = db.Exec(insert, "accepted", "accepted", "treasury-x402-inflow:0xabc", "accepted", 1, "posting-1", "receipt-1")
	require.NoError(t, err)
	got, err := repo.GetAcceptedReceiptByExternalID(context.Background(), "treasury-x402-inflow:0xabc")
	require.NoError(t, err)
	require.Equal(t, AcceptedReceipt{ExternalID: "treasury-x402-inflow:0xabc", PostingID: "posting-1", ReceiptID: "receipt-1", AmountMinor: 100, Currency: "USD", Basis: "authoritative", AcceptedAt: mustParseTime(t, "2026-11-14T12:00:00Z")}, got)
	_, err = db.Exec(insert, "accepted-duplicate", "accepted-duplicate", "treasury-x402-inflow:0xabc", "accepted", 1, "posting-2", "receipt-2")
	require.NoError(t, err)
	_, err = repo.GetAcceptedReceiptByExternalID(context.Background(), "treasury-x402-inflow:0xabc")
	require.ErrorIs(t, err, ErrAmbiguousAcceptedReceipt)
}

func mustParseTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339Nano, value)
	require.NoError(t, err)
	return parsed
}
