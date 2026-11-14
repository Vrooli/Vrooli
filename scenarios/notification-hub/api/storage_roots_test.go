package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/storage"
)

// storageEnv resolves storage from a fixed environment and home, never the
// process's own.
func storageRoots(t *testing.T, env map[string]string) (storage.Paths, string) {
	t.Helper()
	home := t.TempDir()
	cfg := storage.ResolverConfig{
		AppID: "vrooli", Profile: storage.ProfileAuto,
		EnvGet:      func(key string) string { return env[key] },
		UserHomeDir: func() (string, error) { return home, nil },
	}
	paths, err := resolveScenarioStorageRoots(cfg)
	require.NoError(t, err)
	return paths, home
}

func databaseDir(t *testing.T, env map[string]string, home string) string {
	t.Helper()
	path, err := storage.SQLitePath(storage.SQLiteConfig{
		Scenario: "notification-hub", EnvGet: func(key string) string { return env[key] },
		UserHomeDir: func() (string, error) { return home, nil },
	})
	require.NoError(t, err)
	return filepath.Dir(path)
}

func lifecycleEnv(dataDir string) map[string]string {
	return map[string]string{"SCENARIO_NAME": "notification-hub", "VROOLI_SCENARIO": "notification-hub", "SCENARIO_DATA_DIR": dataDir}
}

// A packet's rows are in the database and its files are in the data root; the
// two must name one directory however the instance was launched.
func TestScenarioStorageRoots_DataRootIsWhereTheDatabaseLives(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "scenarios", "notification-hub", "data")
	tests := map[string]struct {
		env  map[string]string
		want func(home string) string
	}{
		"a lifecycle-launched live instance keeps its database, and so its files, in the assigned data directory": {
			env:  lifecycleEnv(dataDir),
			want: func(string) string { return dataDir },
		},
		"an instance with no lifecycle data directory uses the class root": {
			env:  map[string]string{},
			want: func(home string) string { return filepath.Join(home, ".vrooli", "data", "vrooli", "notification-hub") },
		},
		"a shadow variant keeps to its own namespace, not the live data directory": {
			env: func() map[string]string {
				env := lifecycleEnv(dataDir)
				env["VROOLI_STORAGE_NAMESPACE"], env["VROOLI_VARIANT"] = "notification-hub_shadow", "shadow"
				return env
			}(),
			want: func(home string) string {
				return filepath.Join(home, ".vrooli", "data", "vrooli", "notification-hub_shadow")
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			paths, home := storageRoots(t, tc.env)
			require.Equal(t, tc.want(home), paths.DataDir)
			require.Equal(t, databaseDir(t, tc.env, home), paths.DataDir, "the files' root is the database's directory")
		})
	}
}

func TestScenarioStorageRoots_OtherClassesAreUnchanged(t *testing.T) {
	paths, home := storageRoots(t, lifecycleEnv(filepath.Join(t.TempDir(), "data")))
	require.Equal(t, filepath.Join(home, ".vrooli", "state", "vrooli", "notification-hub"), paths.StateDir,
		"only the data class follows the database; the VAPID key stays in state")
}
