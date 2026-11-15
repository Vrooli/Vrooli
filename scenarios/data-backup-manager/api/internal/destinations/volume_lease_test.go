package destinations

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"data-backup-manager/internal/destinationreadiness"
)

func TestHostVolumeOperationLeaseWaitsForHandshakeAndTerminatesOnRelease(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses a POSIX executable")
	}
	dir := t.TempDir()
	readyPath := filepath.Join(dir, "ready")
	argsPath := filepath.Join(dir, "args")
	binary := filepath.Join(dir, "vrooli")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" > '" + argsPath + "'\nprintf '{\"status\":\"leased\",\"device\":\"/dev/fixture\"}\\n'\nprintf ready > '" + readyPath + "'\nexec sleep 30\n"
	if err := os.WriteFile(binary, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	identity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/fixture", Filesystem: "ntfs", UUID: "fixture-uuid", Serial: "fixture-serial"}
	release, err := NewHostVolumeOperationLease(binary).Acquire(context.Background(), identity)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	args, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read command args: %v", err)
	}
	for _, want := range []string{"host volume lease", "--device /dev/fixture", "--uuid fixture-uuid", "--serial fixture-serial", "--filesystem ntfs", "--json"} {
		if !strings.Contains(string(args), want) {
			t.Errorf("command args %q do not contain %q", args, want)
		}
	}
	if _, err := os.Stat(readyPath); err != nil {
		t.Fatalf("lease command did not remain alive after handshake: %v", err)
	}
	finished := make(chan struct{})
	go func() {
		release()
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(3 * time.Second):
		t.Fatal("release did not terminate and reap the lease process")
	}
}

func TestHostVolumeOperationLeaseRejectsInvalidIdentityOrHandshake(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses a POSIX executable")
	}
	dir := t.TempDir()
	binary := filepath.Join(dir, "vrooli")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nprintf '{\"status\":\"leased\",\"device\":\"/dev/other\"}\\n'\nexec sleep 30\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	lease := NewHostVolumeOperationLease(binary)
	if _, err := lease.Acquire(context.Background(), destinationreadiness.DeviceIdentity{DevicePath: "/dev/fixture"}); err == nil {
		t.Fatal("lease accepted path-only identity")
	}
	if _, err := lease.Acquire(context.Background(), destinationreadiness.DeviceIdentity{DevicePath: "/dev/fixture", UUID: "fixture-uuid"}); err == nil {
		t.Fatal("lease accepted a readiness handshake for a different device")
	}
}
