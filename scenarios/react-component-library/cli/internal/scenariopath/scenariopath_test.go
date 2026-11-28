package scenariopath

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type environment map[string]string

func (e environment) LookupEnv(key string) (string, bool) {
	value, ok := e[key]
	return value, ok
}

func TestFindWithUsesInjectedScenarioRoot(t *testing.T) {
	previous, err := os.Getwd()
	require.NoError(t, err)
	sandbox := t.TempDir()
	require.NoError(t, os.Chdir(sandbox))
	t.Cleanup(func() { _ = os.Chdir(previous) })

	root := filepath.Join(t.TempDir(), ScenarioName)
	require.NoError(t, os.MkdirAll(filepath.Join(root, "api"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "catalog"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "api", "go.mod"), []byte("module test\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "catalog", "config.json"), []byte("{}"), 0o644))

	got, err := FindWith(environment{"VROOLI_SCENARIO_ROOT": root})
	require.NoError(t, err)
	require.Equal(t, root, got)
}
