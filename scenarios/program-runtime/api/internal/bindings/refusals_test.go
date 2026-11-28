package bindings

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
)

func TestRefusalRepositoryPersistsGovernanceFailure(t *testing.T) { // [REQ:PRT-P1-007]
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)))
	repo := NewRefusalRepository(d)
	now := time.Date(2026, 8, 11, 14, 0, 0, 0, time.UTC)
	require.NoError(t, repo.RecordRefusal(context.Background(), "sess_1", "PROVENANCE_AGENT", "ops/delete", "missing explicit grant", now))
	var sessionID, provenance, bindingID, reason, occurred string
	require.NoError(t, d.QueryRowContext(context.Background(), `SELECT session_id, provenance, binding_id, reason, occurred_at FROM refusals`).Scan(&sessionID, &provenance, &bindingID, &reason, &occurred))
	require.Equal(t, "sess_1", sessionID)
	require.Equal(t, "PROVENANCE_AGENT", provenance)
	require.Equal(t, "ops/delete", bindingID)
	require.Equal(t, "missing explicit grant", reason)
	require.Equal(t, now.Format(time.RFC3339Nano), occurred)
}
