package resources

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMigrateLegacyDockerStoragePreservesForeignDataAndRemovesVerifiedContainer(t *testing.T) {
	dataRoot := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataRoot)
	paths, err := resourceStoragePaths("redis")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(paths.DataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(paths.DataDir, "dump.rdb")
	if err := os.WriteFile(marker, []byte("legacy"), 0o600); err != nil {
		t.Fatal(err)
	}

	oldExpectedUID := legacyStorageExpectedUID
	oldNow := legacyStorageNow
	oldRun := runCommandResource
	oldDocker := runDockerLifecycleCommand
	t.Cleanup(func() {
		legacyStorageExpectedUID = oldExpectedUID
		legacyStorageNow = oldNow
		runCommandResource = oldRun
		runDockerLifecycleCommand = oldDocker
	})
	legacyStorageExpectedUID = func() uint32 { return uint32(os.Geteuid()) + 1 }
	legacyStorageNow = func() time.Time { return time.Date(2026, 8, 16, 22, 5, 0, 0, time.UTC) }

	manifest := ResourceManifest{
		Name: "redis",
		Storage: &ResourceStorage{Entries: map[string]ResourceStorageEntry{
			"data": {Kind: "dir", Class: "cache", Regenerable: true},
		}},
	}
	inspect := fmt.Sprintf(`[{"State":{"Running":true},"Mounts":[{"Source":%q,"Destination":"/data"}]}]`, paths.DataDir)
	runCommandResource = func(_ context.Context, cmd *exec.Cmd) commandResult {
		if strings.Contains(strings.Join(cmd.Args, " "), "container inspect vrooli-redis-resource") {
			return commandResult{output: []byte(inspect)}
		}
		return commandResult{err: fmt.Errorf("unexpected command: %v", cmd.Args)}
	}
	var lifecycle [][]string
	runDockerLifecycleCommand = func(_ context.Context, _ *Controller, _ io.Writer, _ io.Writer, args ...string) error {
		lifecycle = append(lifecycle, append([]string(nil), args...))
		return nil
	}

	if err := migrateLegacyDockerStorage(context.Background(), NewController(t.TempDir(), t.TempDir()), manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("legacy marker remained at active path, stat error=%v", err)
	}
	backup := paths.DataDir + ".legacy-docker-20260816T220500.000000000Z"
	if got, err := os.ReadFile(filepath.Join(backup, "dump.rdb")); err != nil || string(got) != "legacy" {
		t.Fatalf("backup data = %q, err=%v", got, err)
	}
	if got, err := os.Stat(paths.DataDir); err != nil || !got.IsDir() {
		t.Fatalf("new active data directory missing: info=%v err=%v", got, err)
	}
	if len(lifecycle) != 2 || lifecycle[0][0] != "stop" || lifecycle[1][0] != "rm" {
		t.Fatalf("legacy lifecycle calls = %#v, want stop then rm", lifecycle)
	}
}

func TestMigrateLegacyDockerStorageFailsClosedOnUnexpectedMount(t *testing.T) {
	dataRoot := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataRoot)
	paths, err := resourceStoragePaths("redis")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(paths.DataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldRun := runCommandResource
	oldDocker := runDockerLifecycleCommand
	t.Cleanup(func() {
		runCommandResource = oldRun
		runDockerLifecycleCommand = oldDocker
	})
	inspect := `[{"State":{"Running":true},"Mounts":[{"Source":"/tmp/not-the-resource","Destination":"/data"}]}]`
	runCommandResource = func(_ context.Context, _ *exec.Cmd) commandResult {
		return commandResult{output: []byte(inspect)}
	}
	called := false
	runDockerLifecycleCommand = func(_ context.Context, _ *Controller, _ io.Writer, _ io.Writer, _ ...string) error {
		called = true
		return nil
	}
	if err := migrateLegacyDockerStorage(context.Background(), NewController(t.TempDir(), t.TempDir()), ResourceManifest{Name: "redis"}); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("unexpectedly mutated a container with an unrelated mount")
	}
	if _, err := os.Stat(paths.DataDir); err != nil {
		t.Fatalf("resource data path changed after failed-closed inspection: %v", err)
	}
}

func TestMigrateLegacyDockerStorageRecognizesDeclaredNestedMount(t *testing.T) {
	dataRoot := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataRoot)
	paths, err := resourceStoragePaths("postgres")
	if err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(paths.DataDir, "instances", "main", "data")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal(err)
	}

	oldRun := runCommandResource
	oldDocker := runDockerLifecycleCommand
	t.Cleanup(func() {
		runCommandResource = oldRun
		runDockerLifecycleCommand = oldDocker
	})
	inspect := fmt.Sprintf(`[{
		"State":{"Running":false},
		"Mounts":[{"Source":%q,"Destination":"/var/lib/postgresql/data"}]
	}]`, nested)
	runCommandResource = func(_ context.Context, _ *exec.Cmd) commandResult {
		return commandResult{output: []byte(inspect)}
	}
	var lifecycle [][]string
	runDockerLifecycleCommand = func(_ context.Context, _ *Controller, _ io.Writer, _ io.Writer, args ...string) error {
		lifecycle = append(lifecycle, append([]string(nil), args...))
		return nil
	}
	manifest := ResourceManifest{
		Name: "postgres",
		Storage: &ResourceStorage{Entries: map[string]ResourceStorageEntry{
			"main_data": {
				Kind:       "dir",
				Class:      "data",
				Subpath:    "instances/main/data",
				Relocation: &ResourceStorageRelocation{Key: "RESOURCE_DATA_DIR", Scope: "container"},
			},
		}},
	}
	if err := migrateLegacyDockerStorage(context.Background(), NewController(t.TempDir(), t.TempDir()), manifest); err != nil {
		t.Fatal(err)
	}
	if len(lifecycle) != 1 || lifecycle[0][0] != "rm" {
		t.Fatalf("lifecycle calls = %#v, want rm for stopped nested mount", lifecycle)
	}
}

func TestMigrateLegacyDockerStorageRefusesForeignDurableNestedMount(t *testing.T) {
	dataRoot := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataRoot)
	paths, err := resourceStoragePaths("postgres")
	if err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(paths.DataDir, "instances", "main", "data")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	oldExpectedUID := legacyStorageExpectedUID
	oldRun := runCommandResource
	oldDocker := runDockerLifecycleCommand
	t.Cleanup(func() {
		legacyStorageExpectedUID = oldExpectedUID
		runCommandResource = oldRun
		runDockerLifecycleCommand = oldDocker
	})
	legacyStorageExpectedUID = func() uint32 { return uint32(os.Geteuid()) + 1 }
	inspect := fmt.Sprintf(`[{
		"State":{"Running":true},
		"Mounts":[{"Source":%q,"Destination":"/var/lib/postgresql/data"}]
	}]`, nested)
	runCommandResource = func(_ context.Context, _ *exec.Cmd) commandResult {
		return commandResult{output: []byte(inspect)}
	}
	called := false
	runDockerLifecycleCommand = func(_ context.Context, _ *Controller, _ io.Writer, _ io.Writer, _ ...string) error {
		called = true
		return nil
	}
	manifest := ResourceManifest{
		Name: "postgres",
		Storage: &ResourceStorage{Entries: map[string]ResourceStorageEntry{
			"main_data": {
				Kind:       "dir",
				Class:      "data",
				Subpath:    "instances/main/data",
				Relocation: &ResourceStorageRelocation{Key: "RESOURCE_DATA_DIR", Scope: "container"},
			},
		}},
	}
	if err := migrateLegacyDockerStorage(context.Background(), NewController(t.TempDir(), t.TempDir()), manifest); err == nil {
		t.Fatal("expected foreign-owned durable mount to require controlled maintenance")
	}
	if called {
		t.Fatal("unexpectedly mutated foreign-owned durable storage")
	}
}

func TestLegacyDockerMountMatchesCanonicalDataOnly(t *testing.T) {
	container := legacyDockerContainer{}
	container.Mounts = []struct {
		Source      string `json:"Source"`
		Destination string `json:"Destination"`
	}{{Source: "/tmp/data", Destination: "/data"}}
	if !legacyDockerMountMatches(container, filepath.Clean("/tmp/data")) {
		t.Fatal("expected canonical /data mount to match")
	}
	container.Mounts[0].Destination = "/config"
	if legacyDockerMountMatches(container, "/tmp/data") {
		t.Fatal("unexpectedly matched non-data mount")
	}
}
