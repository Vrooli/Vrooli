package adoptions_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/adoptions"
)

func TestFSScenarioFileReaderReadExistsAndRestoreRaw(t *testing.T) {
	root := t.TempDir()
	reader := adoptions.NewFSScenarioFileReader(root)
	ctx := context.Background()
	path := "ui/src/components/Card.tsx"

	missing, err := reader.Exists(ctx, "consumer", path)
	require.NoError(t, err)
	require.False(t, missing)
	_, err = reader.Read(ctx, "consumer", path)
	require.Error(t, err)

	require.NoError(t, reader.WriteRaw(ctx, "consumer", path, []byte("original bytes\n")))
	present, err := reader.Exists(ctx, "consumer", path)
	require.NoError(t, err)
	require.True(t, present)
	body, err := reader.Read(ctx, "consumer", path)
	require.NoError(t, err)
	require.Equal(t, []byte("original bytes\n"), body)

	require.NoError(t, reader.WriteRaw(ctx, "consumer", path, []byte("restored exactly\n")))
	body, err = os.ReadFile(filepath.Join(root, "consumer", filepath.FromSlash(path)))
	require.NoError(t, err)
	require.Equal(t, []byte("restored exactly\n"), body)

	// Adopted source files are created beneath the target scenario, and the
	// writer tolerates consumers that do not have a local Prettier install.
	createdPath, err := reader.Write(ctx, "consumer", "ui/src/Generated.tsx", []byte("export const Generated = true;\n"))
	require.NoError(t, err)
	created, err := os.ReadFile(createdPath)
	require.NoError(t, err)
	require.Equal(t, []byte("export const Generated = true;\n"), created)
}

func TestFSScenarioFileReaderRejectsScenarioPathTraversal(t *testing.T) {
	reader := adoptions.NewFSScenarioFileReader(t.TempDir())
	_, err := reader.Read(context.Background(), "consumer", "../../outside.tsx")
	require.ErrorContains(t, err, "escapes scenario root")
	_, err = reader.Exists(context.Background(), "consumer", "../../outside.tsx")
	require.ErrorContains(t, err, "escapes scenario root")
	require.ErrorContains(t, reader.WriteRaw(context.Background(), "consumer", "../../outside.tsx", []byte("x")), "escapes scenario root")
}

func TestFSScenarioFileReaderWritesStayInsideScenarioRoot(t *testing.T) {
	root := t.TempDir()
	reader := adoptions.NewFSScenarioFileReader(root)
	outside := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "consumer"), 0o755))
	require.NoError(t, os.Symlink(outside, filepath.Join(root, "consumer", "linked")))

	err := reader.WriteRaw(context.Background(), "consumer", "linked/escape.tsx", []byte("outside"))
	require.Error(t, err)
	_, err = os.Stat(filepath.Join(outside, "escape.tsx"))
	require.ErrorIs(t, err, os.ErrNotExist)
}
