package resources

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/config"
)

const resourceIsolationSentinelEnv = "VROOLI_RESOURCE_TEST_SENTINEL"

// Exercise the real package TestMain in a subprocess, including the exact
// regression that previously published an OpenRGB fixture ledger to live state.
// Every hostile inherited path is a disposable sentinel, never the real home.
func TestResourcePackageStorageIsolation(t *testing.T) {
	for _, fixture := range []struct{ global, noHome bool }{{false, false}, {true, false}, {true, true}} {
		t.Run(fmt.Sprintf("inherited-global=%t/no-home=%t", fixture.global, fixture.noHome), func(t *testing.T) {
			sentinel := t.TempDir()
			marker := filepath.Join(sentinel, "keep")
			if err := os.WriteFile(marker, []byte("operator-owned sentinel\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			before := resourceIsolationSnapshot(t, sentinel)
			env := map[string]string{
				"HOME": sentinel, "USERPROFILE": sentinel,
				"XDG_CONFIG_HOME": filepath.Join(sentinel, ".config"),
				"XDG_DATA_HOME":   filepath.Join(sentinel, ".local", "share"),
				"XDG_STATE_HOME":  filepath.Join(sentinel, ".local", "state"),
				"XDG_CACHE_HOME":  filepath.Join(sentinel, ".cache"),
				"APPDATA":         sentinel, "LOCALAPPDATA": sentinel,
				"VROOLI_STORAGE_ROOT":          filepath.Join(sentinel, "contract"),
				"VROOLI_RESOURCE_ARTIFACT_DIR": filepath.Join(sentinel, "artifacts"),
				resourceIsolationSentinelEnv:   sentinel,
				// Inherited live flags cannot weaken ordinary package isolation.
				"VROOLI_MANAGED_UPDATE_LIVE_FAILURE_REQUEST": "sentinel-only-no-live-operation",
				"SUDO_USER": "vrooli-nonexistent-test-operator", "SUDO_UID": "123456", "SUDO_GID": "123456",
			}
			for _, class := range []string{"CONFIG", "DATA", "CACHE", "LOGS", "STATE"} {
				env["VROOLI_RESOURCE_"+class+"_ROOT"] = filepath.Join(sentinel, strings.ToLower(class))
			}
			if fixture.noHome {
				env["HOME"], env["USERPROFILE"] = "", ""
			}
			if fixture.global {
				env["VROOLI_RESOURCE_STORAGE_ROOT"] = filepath.Join(sentinel, "native")
			}
			command := exec.Command(os.Args[0], "-test.run=^(TestResourcePackageStorageIsolationChild|TestOpenRGBRetirementObservationCannotChangeImmutableAdmission)$", "-test.count=1")
			// Remove ambient resource/SUDO routing rather than accidentally
			// inheriting a real operator root into the regression subprocess.
			for _, value := range os.Environ() {
				key, _, _ := strings.Cut(value, "=")
				if _, replace := env[key]; replace || strings.HasPrefix(key, "VROOLI_RESOURCE_") || strings.HasPrefix(key, "SUDO_") {
					continue
				}
				command.Env = append(command.Env, value)
			}
			for key, value := range env {
				command.Env = append(command.Env, key+"="+value)
			}
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("isolated resource subprocess: %v\n%s", err, output)
			}
			if after := resourceIsolationSnapshot(t, sentinel); !reflect.DeepEqual(before, after) {
				t.Fatalf("package tests changed the live-like sentinel tree: before=%v after=%v", before, after)
			}
		})
	}
}

func TestResourcePackageStorageIsolationChild(t *testing.T) {
	sentinel := os.Getenv(resourceIsolationSentinelEnv)
	if sentinel == "" {
		t.Skip("subprocess storage-isolation fixture")
	}
	for _, key := range []string{"SUDO_USER", "SUDO_UID", "SUDO_GID"} {
		if os.Getenv(key) != "" {
			t.Fatalf("inherited elevated-user routing was not isolated: %s", key)
		}
	}
	paths, err := resourceStoragePaths("openrgb")
	if err != nil {
		t.Fatal(err)
	}
	artifactRoot, err := managedServiceArtifactStoreRoot(sentinel)
	if err != nil {
		t.Fatal(err)
	}
	lockRoot, err := config.VrooliPath("state", "locks", "owned-writes")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{paths.ConfigDir, paths.DataDir, paths.CacheDir, paths.LogsDir, paths.StateDir, artifactRoot, lockRoot, os.Getenv("VROOLI_STORAGE_ROOT")} {
		rel, err := filepath.Rel(sentinel, path)
		if err != nil || !filepath.IsAbs(path) || filepath.IsLocal(rel) {
			t.Fatalf("resource storage escaped package isolation into sentinel: %q (%v)", path, err)
		}
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "fixture-write"), []byte("isolated"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := writeResourceOwnedFileDurable(filepath.Join(paths.StateDir, "durable-fixture"), []byte("owned-write lock fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	locks, err := os.ReadDir(lockRoot)
	hasLock := false
	for _, entry := range locks {
		hasLock = hasLock || strings.HasSuffix(entry.Name(), ".lock")
	}
	if err != nil || !hasLock {
		t.Fatalf("durable write did not create its isolated owner lock: %v", err)
	}
}

func resourceIsolationSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		value := info.Mode().String()
		if !entry.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			value += fmt.Sprintf(" %x", sha256.Sum256(data))
		}
		result[rel] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
