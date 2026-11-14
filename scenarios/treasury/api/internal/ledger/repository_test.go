package ledger

import (
	"context"
	"database/sql"
	"testing"

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
