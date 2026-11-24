package catalog

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog"
	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/jobs"
)

func TestRunGateReadsWithoutChangingEvidenceAndRecordGatePersists(t *testing.T) {
	ctx := context.Background()
	repoRoot := t.TempDir()
	scenarioRoot := filepath.Join(repoRoot, "scenarios", "react-component-library")
	libraryRoot := filepath.Join(scenarioRoot, "library")
	configPath := filepath.Join(scenarioRoot, "catalog", "config.json")
	require.NoError(t, os.MkdirAll(filepath.Join(libraryRoot, "components"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(libraryRoot, "release-provenance.json"), []byte(`{"schemaVersion":1,"entries":[]}`), 0o600))
	require.NoError(t, os.WriteFile(configPath, []byte(`{
	  "gateTiers": [
	    {"id":"release-integrity"},
	    {"id":"asset-quality"},
	    {"id":"consumer-use"}
	  ],
	  "gates": [
	    {"id":"release-provenance","tier":"release-integrity","reads":"corpus","blocking":true}
	  ]
	}`), 0o600))

	database := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, database, apidb.SchemaProviderFunc(catalogcoverage.Schema)))
	store := catalogcoverage.NewEvidenceStore(database)
	h := &handler{repoRoot: repoRoot, evidence: store, quarantined: map[string]bool{}, jobRunner: jobs.New()}
	readRequest := connect.NewRequest(&catalogv1.RunGateRequest{Gate: "release-provenance"})
	readCtx := context.WithValue(ctx, skipCalibrationContextKey{}, true)

	before, err := store.List(ctx)
	require.NoError(t, err)
	require.Empty(t, before)
	for attempt := 0; attempt < 3; attempt++ {
		response, runErr := h.RunGate(readCtx, readRequest)
		require.NoError(t, runErr)
		require.NotNil(t, response.Msg)
		require.Zero(t, response.Msg.GetEvidenceRowsWritten())
		after, listErr := store.List(ctx)
		require.NoError(t, listErr)
		require.Equal(t, before, after, "read attempt %d changed persisted evidence", attempt+1)
	}
	allRead, err := h.RunGate(readCtx, connect.NewRequest(&catalogv1.RunGateRequest{All: true}))
	require.NoError(t, err)
	require.Zero(t, allRead.Msg.GetEvidenceRowsWritten())
	allReadRows, err := store.List(ctx)
	require.NoError(t, err)
	require.Equal(t, before, allReadRows, "--all read changed persisted evidence")

	writeResponse, err := h.RecordGate(readCtx, connect.NewRequest(&catalogv1.RunGateRequest{Gate: "release-provenance"}))
	require.NoError(t, err)
	require.EqualValues(t, 1, writeResponse.Msg.GetEvidenceRowsWritten())
	written, err := store.List(ctx)
	require.NoError(t, err)
	require.Len(t, written, 1)
	require.Equal(t, "__corpus__", written[0].AssetID)
	require.Equal(t, "release-provenance", written[0].Gate)

	allWrite, err := h.RecordGate(readCtx, connect.NewRequest(&catalogv1.RunGateRequest{All: true}))
	require.NoError(t, err)
	require.EqualValues(t, 1, allWrite.Msg.GetEvidenceRowsWritten())
	written, err = store.List(ctx)
	require.NoError(t, err)
	require.Len(t, written, 2, "explicit --all write did not persist its gate row")

	for attempt := 0; attempt < 3; attempt++ {
		_, runErr := h.RunGate(readCtx, readRequest)
		require.NoError(t, runErr)
		after, listErr := store.List(ctx)
		require.NoError(t, listErr)
		require.Equal(t, written, after, "read after explicit write changed evidence")
	}
}
