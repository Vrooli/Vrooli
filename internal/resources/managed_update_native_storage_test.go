package resources

import (
	"path/filepath"
	"testing"

	runtimestorage "github.com/vrooli/vrooli/internal/resources/runtime/storage"
)

// Production wrappers carry HOME and XDG_STATE_HOME rather than inheriting
// arbitrary resource-root overrides. Match the parent's private global state
// class to that bounded child environment without exposing operator storage.
func isolateNativeUpdateFixtureStorage(t *testing.T, root string) string {
	t.Helper()
	storageRoot := filepath.Join(root, "storage")
	stateHome := filepath.Join(storageRoot, "state")
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", storageRoot)
	t.Setenv("XDG_STATE_HOME", stateHome)
	return stateHome
}

func TestNativeUpdateFixtureStorageMatchesBoundedWrapperEnvironment(t *testing.T) {
	root := t.TempDir()
	stateHome := isolateNativeUpdateFixtureStorage(t, root)
	parent, err := resourceStoragePaths("native-fixture")
	if err != nil {
		t.Fatal(err)
	}
	runtime := ManagedUpdateRuntime{Home: root, StateHome: stateHome}
	spec := managedUpdateWrapperSpec(ManagedServiceUpdate{}, ManagedUpdateLaunchIntent{Runtime: runtime})
	values := managedServiceEnvValues(spec.Env)
	child, err := runtimestorage.NewResolver(runtimestorage.ResolverConfig{
		RuntimeOS: "linux", EnvGet: func(key string) string { return values[key] },
		UserHomeDir:   func() (string, error) { return root, nil },
		UserConfigDir: func() (string, error) { return filepath.Join(root, "config"), nil },
		UserCacheDir:  func() (string, error) { return filepath.Join(root, "cache"), nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	paths, err := child.Resolve(runtimestorage.Options{ResourceID: "native-fixture"})
	if err != nil || paths.StateDir != parent.StateDir {
		t.Fatalf("bounded wrapper cannot read the parent's private admission: parent=%s child=%s err=%v", parent.StateDir, paths.StateDir, err)
	}
}
