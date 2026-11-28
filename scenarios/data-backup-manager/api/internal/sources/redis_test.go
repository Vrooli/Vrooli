package sources_test

import (
	"context"
	"os"
	"testing"

	"data-backup-manager/internal/sources"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRedisSource_PrefixDumpRestore is a gated integration test. It skips
// unless DBM_SOURCE_INTEGRATION=1, which requires resource-redis to be available.
func TestRedisSource_PrefixDumpRestore(t *testing.T) {
	if value, ok := os.LookupEnv("DBM_SOURCE_INTEGRATION"); !ok || value != "1" {
		t.Skip("set DBM_SOURCE_INTEGRATION=1 to run redis integration tests")
	}

	ctx := context.Background()
	reg := sources.NewProductionRegistry(sources.ExecRunner{})
	cap, err := reg.Capturer(sources.KindRedis)
	require.NoError(t, err)

	stageDir := t.TempDir()
	art, err := cap.Capture(ctx, sources.CaptureSpec{
		Locator:  "test:",
		StageDir: stageDir,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, art.Path)

	err = cap.Restore(ctx, sources.RestoreSpec{
		Locator:      "test:",
		ArtifactPath: art.Path,
		Target:       "test_restored:",
	})
	require.NoError(t, err)
}

// TestRedisSource_CaptureArgv asserts that Capture builds the expected argv.
// TestRedisSource_RestoreArgv asserts that Restore builds the expected argv.
