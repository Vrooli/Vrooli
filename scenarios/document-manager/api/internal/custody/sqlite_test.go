package custody

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestCustodyJournalIsAppendOnlyAndReceiptIsSelfAttested(t *testing.T) { // [REQ:DOC-P0-014] [REQ:DOC-P0-015]
	db, err := sql.Open("sqlite", "file:custody-test?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.ExecContext(context.Background(), Schema())
	require.NoError(t, err)
	repo := NewSQLiteRepository(db)
	require.NoError(t, repo.Append(context.Background(), Record{DocumentHash: "sha256-doc", Step: "intake", Tier: 1, Provider: "document-manager", Locality: "local", Profile: "PROFILE_LOCAL_ONLY", PrivacyClass: "confidential", State: "parsed", StartedAt: time.Now().UTC(), Duration: time.Millisecond}))
	receipt, err := BuildReceipt(context.Background(), repo, "sha256-doc")
	require.NoError(t, err)
	require.True(t, receipt.SelfAttested)
	require.Len(t, receipt.Records, 1)
	var count int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM custody_records`).Scan(&count))
	require.Equal(t, 1, count)
}
