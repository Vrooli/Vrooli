package themes_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	db "github.com/vrooli/api-core/databasetest"
	"react-component-library/internal/themes"
)

func TestSQLiteReplaceBuiltinsReconcilesRegistry(t *testing.T) {
	repo := newSQLiteThemeRepo(t)
	ctx := context.Background()
	require.NoError(t, repo.UpsertBuiltin(ctx, themes.Theme{ID: "retired", Name: "Retired"}))
	require.NoError(t, repo.UpsertBuiltin(ctx, themes.Theme{ID: "kept", Name: "Old name", Tokens: map[string]string{"--tone": "old"}}))

	require.NoError(t, repo.ReplaceBuiltins(ctx, []themes.Theme{
		{ID: "kept", Name: "Canonical name", Tokens: map[string]string{"--tone": "new"}},
		{ID: "added", Name: "Added"},
	}))
	rows, err := repo.ListBuiltins(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, "added", rows[0].ID)
	require.Equal(t, "kept", rows[1].ID)
	require.Equal(t, "Canonical name", rows[1].Name)
	require.Equal(t, "new", rows[1].Tokens["--tone"])
}

func TestSQLiteReplaceBuiltinsRollsBackRemovalWhenSeedWriteFails(t *testing.T) {
	sqlDB := db.NewSQLite(t)
	_, err := sqlDB.ExecContext(context.Background(), themes.Schema())
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(context.Background(), `CREATE TRIGGER reject_theme BEFORE INSERT ON builtin_themes WHEN NEW.id = 'fail' BEGIN SELECT RAISE(ABORT, 'simulated failure'); END`)
	require.NoError(t, err)
	repo := themes.NewSQLiteRepository(sqlDB)
	ctx := context.Background()
	require.NoError(t, repo.UpsertBuiltin(ctx, themes.Theme{ID: "preserve", Name: "Existing"}))

	err = repo.ReplaceBuiltins(ctx, []themes.Theme{{ID: "fail", Name: "Rejected"}})
	require.Error(t, err)
	got, err := repo.GetBuiltin(ctx, "preserve")
	require.NoError(t, err, "superseded rows must survive a failed transactional replacement")
	require.Equal(t, "Existing", got.Name)
}

func TestSQLiteReplaceBuiltinsRejectsInvalidIDWithoutDeletingExistingRows(t *testing.T) {
	repo := newSQLiteThemeRepo(t)
	ctx := context.Background()
	require.NoError(t, repo.UpsertBuiltin(ctx, themes.Theme{ID: "kept", Name: "Kept"}))
	require.Error(t, repo.ReplaceBuiltins(ctx, []themes.Theme{{ID: " "}}))
	got, err := repo.GetBuiltin(ctx, "kept")
	require.NoError(t, err)
	require.Equal(t, "Kept", got.Name)
}

func TestFSDesignMDReaderResolvesScenarioAndRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "demo"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "demo", "DESIGN.md"), []byte("design source"), 0o600))
	reader := themes.NewFSDesignMDReader(root)
	raw, err := reader.Read(context.Background(), " demo ")
	require.NoError(t, err)
	require.Equal(t, "design source", string(raw))
	for _, invalid := range []string{"", "../outside", "demo/child", "demo\\child"} {
		_, err := reader.Read(context.Background(), invalid)
		require.Error(t, err, "scenario %q must not escape its configured root", invalid)
	}
	_, err = reader.Read(context.Background(), "missing")
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestToConnectErrorPreservesThemeErrorCategories(t *testing.T) {
	require.NoError(t, themes.ToConnectError(nil))
	tests := []struct {
		name string
		err  error
		code connect.Code
	}{
		{name: "not found", err: themes.ErrThemeNotFound{ID: "missing"}, code: connect.CodeNotFound},
		{name: "invalid design", err: themes.ErrInvalidDesignMD{Scenario: "demo", Reason: "missing tokens"}, code: connect.CodeInvalidArgument},
		{name: "missing design", err: themes.ErrScenarioDesignMDMissing{Scenario: "demo", Cause: os.ErrNotExist}, code: connect.CodeNotFound},
		{name: "unexpected", err: errors.New("database failure"), code: connect.CodeInternal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.code, connect.CodeOf(themes.ToConnectError(tt.err)))
		})
	}
}
