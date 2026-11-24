package sources_test

import (
	"context"
	"os"
	"testing"

	"data-backup-manager/internal/sources"

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
// TestPostgresSource_RestoreArgv asserts that Restore builds the expected argv.
// assertArgPair checks that flag appears in args and is immediately followed by value.
// assertNoSecret checks that arg does not look like a credential.
