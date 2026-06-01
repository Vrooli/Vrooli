package sources_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"data-backup-manager/internal/sources"
	"data-backup-manager/internal/sources/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPostgresSource_DumpRestore is a gated integration test. It skips unless
// DBM_SOURCE_INTEGRATION=1, which requires resource-postgres to be available.
func TestPostgresSource_DumpRestore(t *testing.T) {
	if value, ok := os.LookupEnv("DBM_SOURCE_INTEGRATION"); !ok || value != "1" {
		t.Skip("set DBM_SOURCE_INTEGRATION=1 to run postgres integration tests")
	}

	ctx := context.Background()
	reg := sources.NewProductionRegistry(sources.ExecRunner{})
	cap, err := reg.Capturer(sources.KindPostgres)
	require.NoError(t, err)

	stageDir := t.TempDir()
	art, err := cap.Capture(ctx, sources.CaptureSpec{
		Locator:  "dbm_test",
		StageDir: stageDir,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, art.Path)

	err = cap.Restore(ctx, sources.RestoreSpec{
		Locator:      "dbm_test",
		ArtifactPath: art.Path,
		Target:       "dbm_test_restored",
	})
	require.NoError(t, err)
}

// TestPostgresSource_CaptureArgv asserts that Capture builds the expected
// resource-postgres argv and that no secret appears in any argument.
func TestPostgresSource_CaptureArgv(t *testing.T) {
	t.Parallel()

	fake := &mocks.FakeCommandRunner{}
	reg := sources.NewProductionRegistry(fake)
	cap, err := reg.Capturer(sources.KindPostgres)
	require.NoError(t, err)

	stageDir := t.TempDir()
	_, _ = cap.Capture(context.Background(), sources.CaptureSpec{
		Locator:  "mydb",
		StageDir: stageDir,
	})

	require.Len(t, fake.Calls, 1, "Capture must make exactly one CLI call")
	call := fake.Calls[0]
	assert.Equal(t, "resource-postgres", call.Name)

	args := call.Args
	assert.Equal(t, "dump", args[0], "first arg must be subcommand 'dump'")
	assertArgPair(t, args, "--database", "mydb")
	assertArgPair(t, args, "--output", filepath.Join(stageDir, "dump.pgdump"))

	// No secret must appear anywhere in argv.
	for _, a := range args {
		assertNoSecret(t, a)
	}
}

// TestPostgresSource_RestoreArgv asserts that Restore builds the expected argv.
func TestPostgresSource_RestoreArgv(t *testing.T) {
	t.Parallel()

	fake := &mocks.FakeCommandRunner{}
	reg := sources.NewProductionRegistry(fake)
	cap, err := reg.Capturer(sources.KindPostgres)
	require.NoError(t, err)

	artifactPath := "/tmp/dump.pgdump"
	_ = cap.Restore(context.Background(), sources.RestoreSpec{
		Locator:      "mydb",
		ArtifactPath: artifactPath,
		Target:       "mydb_restored",
	})

	require.Len(t, fake.Calls, 1)
	call := fake.Calls[0]
	assert.Equal(t, "resource-postgres", call.Name)

	args := call.Args
	assert.Equal(t, "restore", args[0])
	assertArgPair(t, args, "--database", "mydb_restored")
	assertArgPair(t, args, "--input", artifactPath)

	for _, a := range args {
		assertNoSecret(t, a)
	}
}

// assertArgPair checks that flag appears in args and is immediately followed by value.
func assertArgPair(t *testing.T, args []string, flag, value string) {
	t.Helper()
	for i, a := range args {
		if a == flag {
			if i+1 < len(args) {
				assert.Equal(t, value, args[i+1], "value after %s", flag)
				return
			}
			t.Errorf("flag %q has no value", flag)
			return
		}
	}
	t.Errorf("flag %q not found in args %v", flag, args)
}

// assertNoSecret checks that arg does not look like a credential.
func assertNoSecret(t *testing.T, arg string) {
	t.Helper()
	lower := strings.ToLower(arg)
	suspects := []string{"password", "passwd", "secret", "token", "key=", "credential"}
	for _, s := range suspects {
		if strings.Contains(lower, s) {
			t.Errorf("argv contains possible secret: %q", arg)
		}
	}
}
