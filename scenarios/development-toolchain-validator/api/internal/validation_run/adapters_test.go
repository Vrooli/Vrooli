package validation_run_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"development-toolchain-validator/internal/golden"
	vrun "development-toolchain-validator/internal/validation_run"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"

	localdb "development-toolchain-validator/internal/database"

	apidb "github.com/vrooli/api-core/database"
)

type fakeGoldenRunner struct {
	in golden.RegenerateRunnerInput
}

func (f *fakeGoldenRunner) Regenerate(_ context.Context, in golden.RegenerateRunnerInput) (golden.RegenerateRunnerOutput, error) {
	f.in = in
	if err := os.MkdirAll(in.Path, 0o755); err != nil {
		return golden.RegenerateRunnerOutput{}, err
	}
	if err := os.WriteFile(filepath.Join(in.Path, "generated.txt"), []byte("ok\n"), 0o644); err != nil {
		return golden.RegenerateRunnerOutput{}, err
	}
	return golden.RegenerateRunnerOutput{TemplateVersion: "1.0.2"}, nil
}

func TestGoldenMaterializerFromRepo_GeneratesAndCleansEphemeralPath(t *testing.T) {
	d := db.NewSQLite(t)
	ctx := context.Background()
	require.NoError(t, apidb.EnsureSchemas(ctx,
		d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(golden.Schema),
	))
	require.NoError(t, golden.EnsureColumns(ctx, d))

	clk := scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	repo := golden.NewSQLiteRepository(d, clk)
	_, err := repo.Create(ctx, golden.Golden{
		Slug:                  "reference-react-vite",
		TemplateID:            "react-vite",
		TemplateVersionPinned: "1.0.1",
		Path:                  ".vrooli/generated-goldens/reference-react-vite",
		LogicalRoot:           ".vrooli/generated-goldens/reference-react-vite",
	})
	require.NoError(t, err)

	base := t.TempDir()
	runner := &fakeGoldenRunner{}
	mat, err := vrun.GoldenMaterializerFromRepo{
		Repo:    repo,
		Runner:  runner,
		BaseDir: base,
	}.Materialize(ctx, "reference-react-vite")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(mat.PhysicalPath, base))
	require.Equal(t, ".vrooli/generated-goldens/reference-react-vite", mat.LogicalRoot)
	require.Equal(t, "reference-react-vite", runner.in.Slug)
	require.Equal(t, "react-vite", runner.in.TemplateID)
	require.Equal(t, "1.0.1", runner.in.TemplateVersion)
	require.Equal(t, mat.PhysicalPath, runner.in.Path)
	require.FileExists(t, filepath.Join(mat.PhysicalPath, "generated.txt"))

	stored, err := repo.Get(ctx, "reference-react-vite")
	require.NoError(t, err)
	require.Equal(t, golden.MaterializationStatusReady, stored.LastMaterializedStatus)
	require.Equal(t, mat.PhysicalPath, stored.LastMaterializedPath)
	require.Equal(t, "1.0.2", stored.TemplateVersionPinned)

	require.NoError(t, mat.Cleanup())
	require.NoDirExists(t, mat.PhysicalPath)
}
