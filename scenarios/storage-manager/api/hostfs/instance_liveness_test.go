package hostfs

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func writeRecord(t *testing.T, dir, name string, record map[string]any) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll %s: %v", dir, err)
	}
	payload, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), payload, 0o644); err != nil {
		t.Fatalf("write record: %v", err)
	}
}

// reapedPID returns the pid of a process that has already exited and been
// waited on, so the kernel holds no entry for it.
func reapedPID(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("true")
	if err := cmd.Run(); err != nil {
		t.Fatalf("run throwaway process: %v", err)
	}
	return cmd.Process.Pid
}

// An instance that was stopped leaves its record directory behind with no
// record files in it. That shape is the whole point of this reader: it is the
// on-disk signature of "stopped", and reading it as "possibly running" would
// make a genuine orphan permanently invisible.
func TestInstanceRegistryEmptyRecordDirectoryIsNotRunning(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "web-console@desktop-smoke"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	running, err := NewInstanceRegistryLiveness(root).RunningInstances(context.Background())
	if err != nil {
		t.Fatalf("RunningInstances() error = %v", err)
	}
	if _, alive := running["web-console@desktop-smoke"]; alive {
		t.Fatal("an empty record directory was reported as running")
	}
	if len(running) != 0 {
		t.Fatalf("expected no running instances, got %v", running)
	}
}

func TestInstanceRegistryLiveRecordIsRunning(t *testing.T) {
	root := t.TempDir()
	writeRecord(t, filepath.Join(root, "web-console@presentation"), "start-api.json", map[string]any{
		"pid": os.Getpid(), "status": "running",
	})

	running, err := NewInstanceRegistryLiveness(root).RunningInstances(context.Background())
	if err != nil {
		t.Fatalf("RunningInstances() error = %v", err)
	}
	if _, alive := running["web-console@presentation"]; !alive {
		t.Fatalf("a live recorded process was not reported as running: %v", running)
	}
}

// A record left behind by a process that died is not evidence of life. Treating
// it as such is the other half of the same false-negative bug as the empty
// directory.
func TestInstanceRegistryStaleRecordIsNotRunning(t *testing.T) {
	root := t.TempDir()
	writeRecord(t, filepath.Join(root, "test-genie@shadow"), "start-api.json", map[string]any{
		"pid": reapedPID(t), "status": "running",
	})
	writeRecord(t, filepath.Join(root, "web-search@shadow"), "start-api.json", map[string]any{
		"pid": os.Getpid(), "status": "exited",
	})

	running, err := NewInstanceRegistryLiveness(root).RunningInstances(context.Background())
	if err != nil {
		t.Fatalf("RunningInstances() error = %v", err)
	}
	if len(running) != 0 {
		t.Fatalf("a dead pid or a non-running status was reported as running: %v", running)
	}
}

// A record that cannot be parsed carries no pid to probe. It is also not
// evidence of a stopped instance, so the conservative direction is to keep the
// instance's storage.
func TestInstanceRegistryMalformedRecordCountsAsRunning(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "audio-tools@presentation")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "start-api.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatalf("write record: %v", err)
	}

	running, err := NewInstanceRegistryLiveness(root).RunningInstances(context.Background())
	if err != nil {
		t.Fatalf("RunningInstances() error = %v", err)
	}
	if _, alive := running["audio-tools@presentation"]; !alive {
		t.Fatal("an unparseable record must not be read as a stopped instance")
	}
}

// A registry that has never been written is a real, readable answer: nothing
// was ever started on this host.
func TestInstanceRegistryMissingRootIsEmptyNotAnError(t *testing.T) {
	running, err := NewInstanceRegistryLiveness(filepath.Join(t.TempDir(), "never-created")).RunningInstances(context.Background())
	if err != nil {
		t.Fatalf("RunningInstances() error = %v", err)
	}
	if len(running) != 0 {
		t.Fatalf("expected an empty set, got %v", running)
	}
}

// An unreadable registry must stay an error all the way up, so the provider
// blocks instead of concluding that nothing is running.
func TestInstanceRegistryUnreadableRootIsAnError(t *testing.T) {
	notADirectory := filepath.Join(t.TempDir(), "registry")
	if err := os.WriteFile(notADirectory, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if _, err := NewInstanceRegistryLiveness(notADirectory).RunningInstances(context.Background()); err == nil {
		t.Fatal("an unreadable registry root must not report an empty set")
	}
	if _, err := NewInstanceRegistryLiveness("").RunningInstances(context.Background()); err == nil {
		t.Fatal("an unset registry root must not report an empty set")
	}
}
