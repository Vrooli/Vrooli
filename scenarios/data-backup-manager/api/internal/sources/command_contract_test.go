package sources_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"data-backup-manager/internal/sources"
	"data-backup-manager/internal/sources/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type resourceCommandCase struct {
	name          string
	kind          sources.SourceKind
	command       string
	locator       string
	restoreTarget string
	artifactPath  string
	env           map[string]string
	captureArgs   func(stageDir string) []string
	restoreArgs   func(artifactPath string) []string
}

func resourceCommandCases() []resourceCommandCase {
	return []resourceCommandCase{
		{
			name: "postgres", kind: sources.KindPostgres, command: "resource-postgres",
			locator: "mydb", restoreTarget: "mydb_restored", artifactPath: "/tmp/dump.pgdump",
			captureArgs: func(stageDir string) []string {
				return []string{"dump", "--database", "mydb", "--output", filepath.Join(stageDir, "dump.pgdump")}
			},
			restoreArgs: func(artifactPath string) []string {
				return []string{"restore", "--database", "mydb_restored", "--input", artifactPath}
			},
		},
		{
			name: "redis", kind: sources.KindRedis, command: "resource-redis",
			locator: "myapp:", restoreTarget: "myapp_restored:", artifactPath: "/tmp/dump.rdb",
			env: map[string]string{"VROOLI_STORAGE_NAMESPACE": "data-backup-manager", "VROOLI_SCENARIO": "data-backup-manager"},
			captureArgs: func(stageDir string) []string {
				return []string{"dump", "--prefix", "data-backup-manager:myapp:", "--output", filepath.Join(stageDir, "dump.rdb")}
			},
			restoreArgs: func(artifactPath string) []string {
				return []string{"restore", "--prefix", "data-backup-manager:myapp_restored:", "--input", artifactPath}
			},
		},
		{
			name: "qdrant", kind: sources.KindQdrant, command: "resource-qdrant",
			locator: "my_collection", restoreTarget: "my_collection_restored", artifactPath: "/tmp/snapshot.qdrant",
			env: map[string]string{"VROOLI_STORAGE_NAMESPACE": "data-backup-manager", "VROOLI_SCENARIO": "data-backup-manager"},
			captureArgs: func(stageDir string) []string {
				return []string{"snapshot", "create", "--collection", "data-backup-manager_my_collection", "--output", filepath.Join(stageDir, "snapshot.qdrant")}
			},
			restoreArgs: func(artifactPath string) []string {
				return []string{"snapshot", "restore", "--collection", "data-backup-manager_my_collection_restored", "--input", artifactPath}
			},
		},
		{
			name: "object-storage", kind: sources.KindObjectStorage, command: "resource-minio",
			locator: "mybucket/data", restoreTarget: "mybucket/data-restored", artifactPath: "/tmp/mirror",
			captureArgs: func(stageDir string) []string {
				return []string{"mirror", "--source", "mybucket/data", "--dest", filepath.Join(stageDir, "mirror")}
			},
			restoreArgs: func(artifactPath string) []string {
				return []string{"mirror", "--source", artifactPath, "--dest", "mybucket/data-restored"}
			},
		},
	}
}

func TestResourceSourceCaptureCommandContract(t *testing.T) {
	for _, test := range resourceCommandCases() {
		t.Run(test.name, func(t *testing.T) {
			applyResourceCommandEnv(t, test.env)
			runner, capturer := newResourceCommandCapturer(t, test.kind)
			stageDir := t.TempDir()
			_, err := capturer.Capture(context.Background(), sources.CaptureSpec{Locator: test.locator, StageDir: stageDir})
			require.NoError(t, err)
			assertCommandCall(t, runner, test.command, test.captureArgs(stageDir))
		})
	}
}

func TestResourceSourceRestoreCommandContract(t *testing.T) {
	for _, test := range resourceCommandCases() {
		t.Run(test.name, func(t *testing.T) {
			applyResourceCommandEnv(t, test.env)
			runner, capturer := newResourceCommandCapturer(t, test.kind)
			err := capturer.Restore(context.Background(), sources.RestoreSpec{
				Locator: test.locator, ArtifactPath: test.artifactPath, Target: test.restoreTarget,
			})
			require.NoError(t, err)
			assertCommandCall(t, runner, test.command, test.restoreArgs(test.artifactPath))
		})
	}
}

func TestResourceSourceCaptureReturnsCommandFailure(t *testing.T) {
	assertResourceSourceCommandFailure(t, func(ctx context.Context, capturer sources.Capturer, test resourceCommandCase) error {
		_, err := capturer.Capture(ctx, sources.CaptureSpec{Locator: test.locator, StageDir: t.TempDir()})
		return err
	})
}

func TestResourceSourceRestoreReturnsCommandFailure(t *testing.T) {
	assertResourceSourceCommandFailure(t, func(ctx context.Context, capturer sources.Capturer, test resourceCommandCase) error {
		return capturer.Restore(ctx, sources.RestoreSpec{
			Locator: test.locator, ArtifactPath: test.artifactPath, Target: test.restoreTarget,
		})
	})
}

func assertResourceSourceCommandFailure(t *testing.T, invoke func(context.Context, sources.Capturer, resourceCommandCase) error) {
	t.Helper()
	runnerErr := errors.New("resource command failed")
	for _, test := range resourceCommandCases() {
		t.Run(test.name, func(t *testing.T) {
			applyResourceCommandEnv(t, test.env)
			runner, capturer := newResourceCommandCapturer(t, test.kind, mocks.CommandResult{Err: runnerErr})
			require.ErrorIs(t, invoke(context.Background(), capturer, test), runnerErr)
			require.Len(t, runner.Calls, 1)
		})
	}
}

func newResourceCommandCapturer(t *testing.T, kind sources.SourceKind, results ...mocks.CommandResult) (*mocks.FakeCommandRunner, sources.Capturer) {
	t.Helper()
	runner := &mocks.FakeCommandRunner{Results: results}
	capturer, err := sources.NewProductionRegistry(runner).Capturer(kind)
	require.NoError(t, err)
	return runner, capturer
}

func applyResourceCommandEnv(t *testing.T, env map[string]string) {
	t.Helper()
	for name, value := range env {
		t.Setenv(name, value)
	}
}

func assertCommandCall(t *testing.T, runner *mocks.FakeCommandRunner, command string, args []string) {
	t.Helper()
	require.Len(t, runner.Calls, 1)
	assert.Equal(t, command, runner.Calls[0].Name)
	assert.Equal(t, args, runner.Calls[0].Args)
	for _, arg := range runner.Calls[0].Args {
		assertNoSecret(t, arg)
	}
}

func assertNoSecret(t *testing.T, arg string) {
	t.Helper()
	lower := strings.ToLower(arg)
	suspects := []string{"password", "passwd", "secret", "token", "key=", "credential"}
	for _, suspect := range suspects {
		if strings.Contains(lower, suspect) {
			t.Errorf("argv contains possible secret: %q", arg)
		}
	}
}
