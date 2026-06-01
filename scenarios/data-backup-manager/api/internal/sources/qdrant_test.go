package sources_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"data-backup-manager/internal/sources"
	"data-backup-manager/internal/sources/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestQdrantSource_SnapshotRestore is a gated integration test. It skips
// unless DBM_SOURCE_INTEGRATION=1, which requires resource-qdrant to be available.
func TestQdrantSource_SnapshotRestore(t *testing.T) {
	if value, ok := os.LookupEnv("DBM_SOURCE_INTEGRATION"); !ok || value != "1" {
		t.Skip("set DBM_SOURCE_INTEGRATION=1 to run qdrant integration tests")
	}

	ctx := context.Background()
	reg := sources.NewProductionRegistry(sources.ExecRunner{})
	cap, err := reg.Capturer(sources.KindQdrant)
	require.NoError(t, err)

	stageDir := t.TempDir()
	art, err := cap.Capture(ctx, sources.CaptureSpec{
		Locator:  "my_collection",
		StageDir: stageDir,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, art.Path)

	err = cap.Restore(ctx, sources.RestoreSpec{
		Locator:      "my_collection",
		ArtifactPath: art.Path,
		Target:       "my_collection_restored",
	})
	require.NoError(t, err)
}

// TestQdrantSource_CaptureArgv asserts that Capture builds the expected argv.
func TestQdrantSource_CaptureArgv(t *testing.T) {
	t.Parallel()

	fake := &mocks.FakeCommandRunner{}
	reg := sources.NewProductionRegistry(fake)
	cap, err := reg.Capturer(sources.KindQdrant)
	require.NoError(t, err)

	stageDir := t.TempDir()
	_, _ = cap.Capture(context.Background(), sources.CaptureSpec{
		Locator:  "my_collection",
		StageDir: stageDir,
	})

	require.Len(t, fake.Calls, 1)
	call := fake.Calls[0]
	assert.Equal(t, "resource-qdrant", call.Name)

	args := call.Args
	assert.Equal(t, "snapshot", args[0])
	assert.Equal(t, "create", args[1])
	assertArgPair(t, args, "--collection", "my_collection")
	assertArgPair(t, args, "--output", filepath.Join(stageDir, "snapshot.qdrant"))

	for _, a := range args {
		assertNoSecret(t, a)
	}
}

// TestQdrantSource_RestoreArgv asserts that Restore builds the expected argv.
func TestQdrantSource_RestoreArgv(t *testing.T) {
	t.Parallel()

	fake := &mocks.FakeCommandRunner{}
	reg := sources.NewProductionRegistry(fake)
	cap, err := reg.Capturer(sources.KindQdrant)
	require.NoError(t, err)

	artifactPath := "/tmp/snapshot.qdrant"
	_ = cap.Restore(context.Background(), sources.RestoreSpec{
		Locator:      "my_collection",
		ArtifactPath: artifactPath,
		Target:       "my_collection_restored",
	})

	require.Len(t, fake.Calls, 1)
	call := fake.Calls[0]
	assert.Equal(t, "resource-qdrant", call.Name)

	args := call.Args
	assert.Equal(t, "snapshot", args[0])
	assert.Equal(t, "restore", args[1])
	assertArgPair(t, args, "--collection", "my_collection_restored")
	assertArgPair(t, args, "--input", artifactPath)

	for _, a := range args {
		assertNoSecret(t, a)
	}
}
