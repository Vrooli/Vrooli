package adoptions

import (
	"context"
	"testing"

	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/components"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
)

func TestCatalogGateReaderFailsClosedAndMatchesExactObservation(t *testing.T) {
	ctx := context.Background()
	component := components.Component{CatalogID: "controls.button"}
	require.Equal(t, "not-measured", mustGateVerdict(t, NewCatalogGateReader(nil), component, "1.0.0", "react-vite", "visual"))
	require.Equal(t, "not-measured", mustGateVerdict(t, NewCatalogGateReader(catalogcoverage.NewEvidenceStore(nil)), component, "1.0.0", "react-vite", "visual"))

	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, database, apidb.SchemaProviderFunc(catalogcoverage.Schema)))
	store := catalogcoverage.NewEvidenceStore(database)
	reader := NewCatalogGateReader(store)
	for _, tc := range []struct{ result, want string }{
		{"pass", "pass"}, {"fail", "fail"}, {"unmeasured", "unmeasured"}, {"skipped", "skipped"}, {"unknown", "not-measured"},
	} {
		require.NoError(t, store.Save(ctx, []catalogcoverage.GateEvidence{{AssetID: "controls.button", Target: "react-vite", Gate: "visual", Version: tc.result, Result: tc.result, SourceRevision: "test"}}))
		got, err := reader.GateVerdict(ctx, component, tc.result, "react-vite", "visual")
		require.NoError(t, err)
		require.Equal(t, tc.want, got)
	}
	require.Equal(t, "not-measured", mustGateVerdict(t, reader, components.Component{}, "pass", "react-vite", "visual"))
	require.Equal(t, "not-measured", mustGateVerdict(t, reader, component, "missing", "react-vite", "visual"))
}

func mustGateVerdict(t *testing.T, reader *CatalogGateReader, component components.Component, version, scenario, gate string) string {
	t.Helper()
	got, err := reader.GateVerdict(context.Background(), component, version, scenario, gate)
	require.NoError(t, err)
	return got
}
