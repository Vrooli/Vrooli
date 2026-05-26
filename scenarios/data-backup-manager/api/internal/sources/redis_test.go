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

// TestRedisSource_PrefixDumpRestore is a gated integration test. It skips
// unless DBM_SOURCE_INTEGRATION=1, which requires resource-redis to be available.
func TestRedisSource_PrefixDumpRestore(t *testing.T) {
	if os.Getenv("DBM_SOURCE_INTEGRATION") != "1" {
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
func TestRedisSource_CaptureArgv(t *testing.T) {
	t.Parallel()

	fake := &mocks.FakeCommandRunner{}
	reg := sources.NewProductionRegistry(fake)
	cap, err := reg.Capturer(sources.KindRedis)
	require.NoError(t, err)

	stageDir := t.TempDir()
	_, _ = cap.Capture(context.Background(), sources.CaptureSpec{
		Locator:  "myapp:",
		StageDir: stageDir,
	})

	require.Len(t, fake.Calls, 1)
	call := fake.Calls[0]
	assert.Equal(t, "resource-redis", call.Name)

	args := call.Args
	assert.Equal(t, "dump", args[0])
	assertArgPair(t, args, "--prefix", "myapp:")
	assertArgPair(t, args, "--output", filepath.Join(stageDir, "dump.rdb"))

	for _, a := range args {
		assertNoSecret(t, a)
	}
}

// TestRedisSource_RestoreArgv asserts that Restore builds the expected argv.
func TestRedisSource_RestoreArgv(t *testing.T) {
	t.Parallel()

	fake := &mocks.FakeCommandRunner{}
	reg := sources.NewProductionRegistry(fake)
	cap, err := reg.Capturer(sources.KindRedis)
	require.NoError(t, err)

	artifactPath := "/tmp/dump.rdb"
	_ = cap.Restore(context.Background(), sources.RestoreSpec{
		Locator:      "myapp:",
		ArtifactPath: artifactPath,
		Target:       "myapp_restored:",
	})

	require.Len(t, fake.Calls, 1)
	call := fake.Calls[0]
	assert.Equal(t, "resource-redis", call.Name)

	args := call.Args
	assert.Equal(t, "restore", args[0])
	assertArgPair(t, args, "--prefix", "myapp_restored:")
	assertArgPair(t, args, "--input", artifactPath)

	for _, a := range args {
		assertNoSecret(t, a)
	}
}
