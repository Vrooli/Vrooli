package library

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"

	internalbindings "program-runtime/internal/bindings"
	"program-runtime/internal/contracts"
	"program-runtime/internal/library"
	"program-runtime/internal/programs"
)

func TestListLibraryWithoutQueryIncludesDeclaredContractRows(t *testing.T) {
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db,
		apidb.SchemaProviderFunc(programs.Schema),
		apidb.SchemaProviderFunc(internalbindings.Schema),
		apidb.SchemaProviderFunc(library.Schema)))

	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../.."))
	index := contracts.NewIndex()
	require.NoError(t, index.Load(repoRoot))

	h := &handler{repo: library.NewRepository(db), contracts: index}
	response, err := h.ListLibrary(context.Background(), connect.NewRequest(&programsv1.ListLibraryRequest{Limit: 100}))
	require.NoError(t, err)

	var found bool
	for _, program := range response.Msg.GetPrograms() {
		if program.GetName() == "program-runtime.setpoint-read" {
			found = true
			require.Equal(t, "contract", program.GetKind())
			require.Equal(t, "program-runtime", program.GetScenario())
			break
		}
	}
	require.True(t, found, "bare library list must include declared contract rows")
	got, err := h.GetLibrary(context.Background(), connect.NewRequest(&programsv1.GetLibraryRequest{Name: "browser-automation-studio.smoke-flow"}))
	require.NoError(t, err)
	require.Contains(t, got.Msg.GetProgram().GetSource(), "smoke-flow")
	require.Equal(t, "contract", got.Msg.GetProgram().GetKind())
}
