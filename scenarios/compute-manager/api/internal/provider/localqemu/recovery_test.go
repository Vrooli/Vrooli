package localqemu

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	cloudprovider "compute-manager/internal/provider"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

func TestAdapterRecoversPersistedGuestAfterReconstructionAndDestroysOnlyOwnedWorkdir(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	primary := storage.Paths{
		ConfigDir: filepath.Join(t.TempDir(), "config"),
		DataDir:   filepath.Join(t.TempDir(), "data"),
		CacheDir:  filepath.Join(t.TempDir(), "cache"),
		LogsDir:   filepath.Join(t.TempDir(), "logs"),
		StateDir:  filepath.Join(t.TempDir(), "state"),
	}
	for _, path := range []string{primary.ConfigDir, primary.DataDir, primary.CacheDir, primary.LogsDir, primary.StateDir} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	roots := filerouting.New(primary)
	testRoots, err := roots.InstallLeasedTestRoots("local-qemu-recovery-test", time.Minute, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = roots.ClearTestRoots("local-qemu-recovery-test") })
	ctx := database.WithTestMode(context.Background())

	baseImage := filepath.Join(t.TempDir(), "base.qcow2")
	if err := os.WriteFile(baseImage, []byte("fixture base image"), 0o600); err != nil {
		t.Fatal(err)
	}
	first := &Adapter{Roots: roots, QEMU: recoveryFixtureQEMU(nil)}
	created, err := first.Create(ctx, cloudprovider.Spec{
		Region: "local", Size: "headless-linux", Image: baseImage, Memory: "2048", CPUs: 2,
		Profile: string(ProfileHeadlessLinux), SSHUser: "vrooli", SSHPort: port,
		SSHAuthorizedKey: "ssh-ed25519 AAAA recovery-fixture", Tags: map[string]string{"vrooli-request": "recovery-fixture"},
	})
	if err != nil {
		t.Fatal(err)
	}
	ownedWorkdir := filepath.Join(testRoots.DataDir, "local-qemu", created.ID)
	if _, err := os.Stat(filepath.Join(ownedWorkdir, metadataName)); err != nil {
		t.Fatalf("first adapter did not persist metadata: %v", err)
	}

	// A valid-looking but absent record must not invoke QEMU effects or remove
	// another workdir's contents.
	foreignID := "qemu-aaaaaaaaaaaaaaaaaaaaaaaa"
	foreignWorkdir := filepath.Join(testRoots.DataDir, "local-qemu", foreignID)
	if err := os.MkdirAll(foreignWorkdir, 0o700); err != nil {
		t.Fatal(err)
	}
	foreignMarker := filepath.Join(foreignWorkdir, "foreign-owner.marker")
	if err := os.WriteFile(foreignMarker, []byte("owned elsewhere"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stopped []int
	restarted := &Adapter{Roots: roots, QEMU: recoveryFixtureQEMU(func(pid int) error {
		stopped = append(stopped, pid)
		return nil
	})}
	loaded, err := restarted.Describe(ctx, created.ID)
	if err != nil || loaded.ID != created.ID || loaded.Address != "127.0.0.1" || loaded.SSHUser != "vrooli" || loaded.SSHPort != port || loaded.Size != "headless-linux" {
		t.Fatalf("reconstructed Describe() = %#v, %v", loaded, err)
	}
	listed, err := restarted.List(ctx)
	if err != nil || len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("reconstructed List() = %#v, %v", listed, err)
	}
	if err := restarted.Destroy(ctx, foreignID); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Destroy(absent foreign record) = %v, want not-exist", err)
	}
	if len(stopped) != 0 {
		t.Fatalf("absent record invoked QEMU stop for PIDs %v", stopped)
	}
	if _, err := os.Stat(foreignMarker); err != nil {
		t.Fatalf("absent foreign record removed foreign workdir marker: %v", err)
	}

	if err := restarted.Destroy(ctx, created.ID); err != nil {
		t.Fatalf("reconstructed Destroy(owned record) = %v", err)
	}
	if len(stopped) != 1 || stopped[0] != 98765 {
		t.Fatalf("destroy stop effects = %v, want only persisted PID 98765", stopped)
	}
	if _, err := os.Stat(ownedWorkdir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("owned workdir still exists after destroy: %v", err)
	}
	if _, err := os.Stat(foreignMarker); err != nil {
		t.Fatalf("owned destroy affected a foreign workdir: %v", err)
	}
	if _, err := restarted.Describe(ctx, created.ID); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Describe after destroy = %v, want not-exist", err)
	}
	listed, err = restarted.List(ctx)
	if err != nil || len(listed) != 0 {
		t.Fatalf("List after owned destroy = %#v, %v", listed, err)
	}
	if _, err := os.Stat(filepath.Join(primary.DataDir, "local-qemu")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("test-mode create escaped into primary data root: %v", err)
	}
	stats := roots.LeaseStats()
	if stats.TestRootWrites == 0 || stats.PrimaryWritesDuringTestMode != 0 {
		t.Fatalf("routed storage writes = %#v, want test-root-only writes", stats)
	}
}

func TestAdapterDestroyRefusesForeignMetadataWithoutTouchingItsWorkdir(t *testing.T) {
	primary := storage.Paths{
		ConfigDir: filepath.Join(t.TempDir(), "config"),
		DataDir:   filepath.Join(t.TempDir(), "data"),
		CacheDir:  filepath.Join(t.TempDir(), "cache"),
		LogsDir:   filepath.Join(t.TempDir(), "logs"),
		StateDir:  filepath.Join(t.TempDir(), "state"),
	}
	for _, path := range []string{primary.ConfigDir, primary.DataDir, primary.CacheDir, primary.LogsDir, primary.StateDir} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	roots := filerouting.New(primary)
	testRoots, err := roots.InstallLeasedTestRoots("local-qemu-foreign-record-test", time.Minute, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = roots.ClearTestRoots("local-qemu-foreign-record-test") })
	ctx := database.WithTestMode(context.Background())

	foreignID := "qemu-bbbbbbbbbbbbbbbbbbbbbbbb"
	foreignWorkdir := filepath.Join(t.TempDir(), "foreign-owner-vm")
	if err := os.MkdirAll(foreignWorkdir, 0o700); err != nil {
		t.Fatal(err)
	}
	foreignDisk := filepath.Join(foreignWorkdir, instanceDiskName)
	if err := os.WriteFile(foreignDisk, []byte("foreign guest disk"), 0o600); err != nil {
		t.Fatal(err)
	}
	foreignMarker := filepath.Join(foreignWorkdir, "foreign-owner.marker")
	if err := os.WriteFile(foreignMarker, []byte("owned elsewhere"), 0o600); err != nil {
		t.Fatal(err)
	}
	foreignRecordPath, err := metadataPath(testRoots.DataDir, foreignID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(foreignRecordPath), 0o700); err != nil {
		t.Fatal(err)
	}
	foreignRecord := Instance{
		ID: foreignID, Name: "foreign", Provider: "hetzner", State: StateRunning,
		Image: foreignDisk, Workdir: foreignWorkdir, PID: 54321,
		Profile: ProfileHeadlessLinux, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := writeLocalInstance(foreignRecordPath, foreignRecord); err != nil {
		t.Fatal(err)
	}

	var stopCalls []int
	restarted := &Adapter{Roots: roots, QEMU: recoveryFixtureQEMU(func(pid int) error {
		stopCalls = append(stopCalls, pid)
		return nil
	})}
	if err := restarted.Destroy(ctx, foreignID); err == nil {
		t.Fatal("Destroy accepted metadata owned by a different provider")
	}
	if len(stopCalls) != 0 {
		t.Fatalf("foreign metadata caused QEMU stop calls for PIDs %v", stopCalls)
	}
	for _, path := range []string{foreignDisk, foreignMarker, foreignRecordPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("foreign record destroy removed %s: %v", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(primary.DataDir, "local-qemu")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("foreign-record test escaped into primary data root: %v", err)
	}
}

func recoveryFixtureQEMU(stop func(int) error) LocalQEMUProvider {
	if stop == nil {
		stop = func(int) error { return nil }
	}
	return LocalQEMUProvider{
		LookPath: func(name string) (string, error) { return "/fixture/" + name, nil },
		RunImage: func(args ...string) error {
			if len(args) == 0 || args[0] != "create" {
				return fmt.Errorf("unexpected fake image operation: %v", args)
			}
			return os.WriteFile(args[len(args)-1], []byte("fixture guest disk"), 0o600)
		},
		BuildCloudInit: func(_ string, iso, _, _ string) error {
			return os.WriteFile(iso, []byte("fixture cloud-init image"), 0o600)
		},
		StartProcess: func(string, []string, string) (int, error) { return 98765, nil },
		StopProcess:  stop,
	}
}
