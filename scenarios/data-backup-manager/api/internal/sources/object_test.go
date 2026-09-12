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
func TestObjectSource_CaptureArgv(t *testing.T) {
	t.Parallel()

	fake := &mocks.FakeCommandRunner{}
	reg := sources.NewProductionRegistry(fake)
	cap, err := reg.Capturer(sources.KindObjectStorage)
	require.NoError(t, err)

	stageDir := t.TempDir()
	_, _ = cap.Capture(context.Background(), sources.CaptureSpec{
		Locator:  "mybucket/data",
		StageDir: stageDir,
	})

	require.Len(t, fake.Calls, 1)
	call := fake.Calls[0]
	assert.Equal(t, "resource-minio", call.Name)

	args := call.Args
	assert.Equal(t, "mirror", args[0])
	assertArgPair(t, args, "--source", "mybucket/data")
	assertArgPair(t, args, "--dest", filepath.Join(stageDir, "mirror"))

	for _, a := range args {
		assertNoSecret(t, a)
	}
}

// TestObjectSource_RestoreArgv asserts that Restore builds the expected argv.
func TestObjectSource_RestoreArgv(t *testing.T) {
	t.Parallel()

	fake := &mocks.FakeCommandRunner{}
	reg := sources.NewProductionRegistry(fake)
	cap, err := reg.Capturer(sources.KindObjectStorage)
	require.NoError(t, err)

	artifactPath := "/tmp/mirror"
	_ = cap.Restore(context.Background(), sources.RestoreSpec{
		Locator:      "mybucket/data",
		ArtifactPath: artifactPath,
		Target:       "mybucket/data-restored",
	})

	require.Len(t, fake.Calls, 1)
	call := fake.Calls[0]
	assert.Equal(t, "resource-minio", call.Name)

	args := call.Args
	assert.Equal(t, "mirror", args[0])
	assertArgPair(t, args, "--source", artifactPath)
	assertArgPair(t, args, "--dest", "mybucket/data-restored")

	for _, a := range args {
		assertNoSecret(t, a)
	}
}
