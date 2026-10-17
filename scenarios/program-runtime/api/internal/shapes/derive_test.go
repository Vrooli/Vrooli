package shapes

import (
	"context"
	"testing"

	internalbindings "program-runtime/internal/bindings"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
)

func TestDeriveExcludesRefusedAndFailedInvocations(t *testing.T) {
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(internalbindings.Schema)))
	for _, row := range []struct{ id, binding, outcome string }{
		{"success", "zeta/ops/read", "success"},
		{"refused", "alpha/ops/read", "refused"},
		{"failed", "beta/ops/read", "failed"},
	} {
		_, err := db.Exec(`INSERT INTO binding_invocations (invocation_id,binding_id,target_scenario,session_id,program_id,provenance,outcome,reason,latency_ms,occurred_at) VALUES (?,?,?,?,?,?,?,?,?,?)`, row.id, row.binding, "demo", "session", "program", "agent", row.outcome, "", 1, "2026-09-03T00:00:00Z")
		require.NoError(t, err)
	}
	got, err := Derive(context.Background(), db, "program")
	require.NoError(t, err)
	require.Equal(t, []string{"zeta/ops/read"}, got)
}

func TestDeriveCollapsesDuplicateSuccessfulInvocations(t *testing.T) {
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(internalbindings.Schema)))
	for i := 0; i < 2; i++ {
		_, err := db.Exec(`INSERT INTO binding_invocations (invocation_id,binding_id,target_scenario,session_id,program_id,provenance,outcome,reason,latency_ms,occurred_at) VALUES (?,?,?,?,?,?,?,?,?,?)`, string(rune('a'+i)), "demo/ops/read", "demo", "session", "program", "agent", "success", "", 1, "2026-09-03T00:00:00Z")
		require.NoError(t, err)
	}
	got, err := Derive(context.Background(), db, "program")
	require.NoError(t, err)
	require.Equal(t, []string{"demo/ops/read"}, got)
}

func TestDeriveSortsBindingIDsAndKeyIsStable(t *testing.T) {
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(internalbindings.Schema)))
	for i, binding := range []string{"z/ops/read", "a/ops/read", "m/ops/read"} {
		_, err := db.Exec(`INSERT INTO binding_invocations (invocation_id,binding_id,target_scenario,session_id,program_id,provenance,outcome,reason,latency_ms,occurred_at) VALUES (?,?,?,?,?,?,?,?,?,?)`, string(rune('a'+i)), binding, "demo", "session", "program", "agent", "success", "", 1, "2026-09-03T00:00:00Z")
		require.NoError(t, err)
	}
	got, err := Derive(context.Background(), db, "program")
	require.NoError(t, err)
	require.Equal(t, []string{"a/ops/read", "m/ops/read", "z/ops/read"}, got)
	require.Equal(t, "a/ops/read|m/ops/read|z/ops/read", Key(got))
}
