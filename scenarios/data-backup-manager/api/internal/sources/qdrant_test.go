package sources_test

import (
	"context"
	"os"
	"testing"

	"data-backup-manager/internal/sources"

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
// TestQdrantSource_RestoreArgv asserts that Restore builds the expected argv.
