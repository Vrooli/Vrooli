package versions_test

import (
	"context"
	"io"
	"log"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"
	versionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/versions"
	versionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/versions/versions_v1connect"

	"react-component-library/handlers/versions"
	"react-component-library/internal/components"
	localdb "react-component-library/internal/database"
	"react-component-library/internal/versionledger"
)

func TestMountedListVersionLedgerMapsStoredProgressionAndRequiresLedger(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db,
		database.SchemaProviderFunc(localdb.SystemSchema),
		database.SchemaProviderFunc(components.Schema),
		database.SchemaProviderFunc(versionledger.Schema),
	))
	_, err := db.ExecContext(ctx, `INSERT INTO version_ledger
		(library_id, version, created_at, released_at, retired_at, lifecycle_state, gate_pass_count, gate_fail_count, test_runs, test_pass_rate, adoption_current, adoption_peak, file_count, lines_of_code, dependency_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"react-component-library:Button", "1.2.0", "2026-08-01T10:00:00Z", "2026-08-02T11:00:00Z", "", "released", 4, 1, 5, 0.8, 2, 3, 7, 215, 2)
	require.NoError(t, err)

	ledger := versionledger.NewRepository(db, t.TempDir())
	module := versions.ModuleWithLedger(db, schedule.System(), nil, log.New(io.Discard, "", 0), ledger)
	router := mux.NewRouter()
	module.Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := versionsconnect.NewVersionLifecycleServiceClient(server.Client(), server.URL)

	got, err := client.ListVersionLedger(ctx, connect.NewRequest(&versionsv1.ListVersionLedgerRequest{LibraryId: "react-component-library:Button", Window: "all"}))
	require.NoError(t, err)
	require.Len(t, got.Msg.GetRows(), 1)
	row := got.Msg.GetRows()[0]
	require.Equal(t, "react-component-library:Button", row.GetLibraryId())
	require.Equal(t, "1.2.0", row.GetVersion())
	require.Equal(t, "2026-08-01T10:00:00Z", row.GetCreatedAt())
	require.Equal(t, "2026-08-02T11:00:00Z", row.GetReleasedAt())
	require.Equal(t, "released", row.GetLifecycleState())
	require.Equal(t, int32(4), row.GetGatePassCount())
	require.Equal(t, int32(1), row.GetGateFailCount())
	require.Equal(t, int32(5), row.GetTestRuns())
	require.InDelta(t, 0.8, row.GetTestPassRate(), 0.0001)
	require.Equal(t, int32(2), row.GetAdoptionCurrent())
	require.Equal(t, int32(3), row.GetAdoptionPeak())
	require.Equal(t, int32(7), row.GetFileCount())
	require.Equal(t, int32(215), row.GetLinesOfCode())
	require.Equal(t, int32(2), row.GetDependencyCount())

	withoutLedger := versions.Module(db, schedule.System(), nil, log.New(io.Discard, "", 0))
	missingRouter := mux.NewRouter()
	withoutLedger.Mount(missingRouter)
	missingServer := httptest.NewServer(missingRouter)
	defer missingServer.Close()
	missingClient := versionsconnect.NewVersionLifecycleServiceClient(missingServer.Client(), missingServer.URL)
	_, err = missingClient.ListVersionLedger(ctx, connect.NewRequest(&versionsv1.ListVersionLedgerRequest{}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}
