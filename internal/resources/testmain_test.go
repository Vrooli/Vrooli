package resources

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/testenv"
)

// Resource lifecycle storage and artifact acquisition have independent roots
// from contract-routed operator state. All three must be isolated: setting
// VROOLI_STORAGE_ROOT alone leaves native update ledgers in the operator's XDG
// state directory. Global native roots take precedence over inherited class
// overrides, including when an elevated test resolves the invoking user's home.
func TestMain(m *testing.M) {
	os.Exit(runResourceTests(m))
}

type resourceTestEnvValue struct {
	value string
	set   bool
}

var inheritedResourceTestEnvironment map[string]resourceTestEnvValue
var inheritedResourceTestHome string

func runResourceTests(m *testing.M) int {
	root, err := os.MkdirTemp("", "vrooli-resource-tests-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create isolated resource storage: %v\n", err)
		return 1
	}
	defer os.RemoveAll(root)
	// HOME/XDG also select Go's caches. Retain those toolchain locations
	// before moving product state so subprocess compilation stays warm.
	// Fixed native fixture payloads may intentionally start without HOME.
	home, err := os.UserHomeDir()
	if err != nil {
		home = filepath.Join(root, "home")
	}
	inheritedResourceTestHome = home
	cache, cacheErr := os.UserCacheDir()
	config, configErr := os.UserConfigDir()
	if cacheErr != nil {
		cache = filepath.Join(root, "cache")
	}
	if configErr != nil {
		config = filepath.Join(root, "config")
	}
	for key, value := range map[string]string{
		"GOPATH": filepath.Join(home, "go"), "GOCACHE": filepath.Join(cache, "go-build"), "GOENV": filepath.Join(config, "go", "env"),
	} {
		if _, set := os.LookupEnv(key); !set {
			if err := os.Setenv(key, value); err != nil {
				fmt.Fprintf(os.Stderr, "pin %s: %v\n", key, err)
				return 1
			}
		}
	}
	inheritedResourceTestEnvironment = make(map[string]resourceTestEnvValue)
	for key, value := range map[string]string{
		"VROOLI_RESOURCE_STORAGE_ROOT": filepath.Join(root, "storage"),
		"VROOLI_RESOURCE_ARTIFACT_DIR": filepath.Join(root, "artifacts"),
		"HOME":                         filepath.Join(root, "home"),
		"USERPROFILE":                  filepath.Join(root, "home"),
		"XDG_CONFIG_HOME":              filepath.Join(root, "config"),
		"XDG_DATA_HOME":                filepath.Join(root, "data"),
		"XDG_STATE_HOME":               filepath.Join(root, "state"),
		"XDG_CACHE_HOME":               filepath.Join(root, "cache"),
		"APPDATA":                      filepath.Join(root, "config"),
		"LOCALAPPDATA":                 filepath.Join(root, "data"),
		"SUDO_USER":                    "",
		"SUDO_UID":                     "",
		"SUDO_GID":                     "",
	} {
		previous, set := os.LookupEnv(key)
		inheritedResourceTestEnvironment[key] = resourceTestEnvValue{previous, set}
		if err := os.Setenv(key, value); err != nil {
			fmt.Fprintf(os.Stderr, "isolate %s: %v\n", key, err)
			return 1
		}
	}
	if err := os.MkdirAll(os.Getenv("HOME"), 0o700); err != nil {
		fmt.Fprintf(os.Stderr, "create private resource test home: %v\n", err)
		return 1
	}
	return testenv.RunWithIsolatedStorage(m)
}

// Only the deliberately selected live fault drill may recover its inherited
// native paths. An inherited live flag must never disable package isolation.
// Its caller validates the exact operation/PID/digest before calling here.
func restoreExplicitLiveResourceStorage(t *testing.T) {
	t.Helper()
	const name = "TestManagedUpdateCurrentHostLivePrimaryFailureDrill"
	selected := flag.Lookup("test.run").Value.String()
	if t.Name() != name || (selected != name && selected != "^"+name+"$") {
		t.Fatal("live resource fault drill requires its exact single-test -run selector")
	}
	for key, original := range inheritedResourceTestEnvironment {
		if original.set {
			t.Setenv(key, original.value)
		} else {
			// Setenv installs testing's restoration before removing this key.
			t.Setenv(key, "")
			if err := os.Unsetenv(key); err != nil {
				t.Fatal(err)
			}
		}
	}
}
