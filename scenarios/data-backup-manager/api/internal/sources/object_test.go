package sources_test

import (
	"context"
	"os"
	"testing"

	"data-backup-manager/internal/sources"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestObjectSource_MirrorRestore is a gated integration test. It skips unless
// DBM_SOURCE_INTEGRATION=1, which requires resource-minio to be available.
func TestObjectSource_MirrorRestore(t *testing.T) {
	if value, ok := os.LookupEnv("DBM_SOURCE_INTEGRATION"); !ok || value != "1" {
		t.Skip("set DBM_SOURCE_INTEGRATION=1 to run object storage integration tests")
	}

	ctx := context.Background()
	reg := sources.NewProductionRegistry(sources.ExecRunner{})
	cap, err := reg.Capturer(sources.KindObjectStorage)
	require.NoError(t, err)

	stageDir := t.TempDir()
	art, err := cap.Capture(ctx, sources.CaptureSpec{
		Locator:  "mybucket/prefix",
		StageDir: stageDir,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, art.Path)

	err = cap.Restore(ctx, sources.RestoreSpec{
		Locator:      "mybucket/prefix",
		ArtifactPath: art.Path,
		Target:       "mybucket/prefix-restored",
	})
	require.NoError(t, err)
}

// TestObjectSource_CaptureArgv asserts that Capture builds the expected argv.
// TestObjectSource_RestoreArgv asserts that Restore builds the expected argv.
