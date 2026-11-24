package localqemu

import (
	"context"
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

func TestAdapterOwnsLocalVMThroughSharedProviderAndOptionalTestControls(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	image := filepath.Join(t.TempDir(), "base.qcow2")
	if err := os.WriteFile(image, []byte("fixture image"), 0o600); err != nil {
		t.Fatal(err)
	}
	primary := storage.Paths{ConfigDir: filepath.Join(t.TempDir(), "config"), DataDir: filepath.Join(t.TempDir(), "data"), CacheDir: filepath.Join(t.TempDir(), "cache"), LogsDir: filepath.Join(t.TempDir(), "logs"), StateDir: filepath.Join(t.TempDir(), "state")}
	roots := filerouting.New(primary)
	testRoots, err := roots.InstallLeasedTestRoots("local-qemu-test", time.Minute, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = roots.ClearTestRoots("local-qemu-test") })
	provider := &Adapter{
		Roots: roots,
		Now:   func() time.Time { return time.Date(2026, 11, 23, 23, 0, 0, 0, time.UTC) },
		QEMU: LocalQEMUProvider{
			LookPath:       func(name string) (string, error) { return "/fixture/" + name, nil },
			RunImage:       func(...string) error { return nil },
			BuildCloudInit: func(string, string, string, string) error { return nil },
			StartProcess:   func(string, []string, string) (int, error) { return 98765, nil },
			StopProcess:    func(int) error { return nil },
		},
	}
	spec := cloudprovider.Spec{
		Region: "local", Size: "headless-linux", Image: image, Memory: "2048", CPUs: 2,
		Profile: string(ProfileHeadlessLinux), SSHUser: "vrooli", SSHPort: port,
		SSHAuthorizedKey: "ssh-ed25519 AAAA fixture", Tags: map[string]string{"vrooli-request": "adapter-test-1"},
	}
	ctx := database.WithTestMode(context.Background())
	created, err := provider.Create(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	if created.Address != "127.0.0.1" || created.SSHPort != port || created.SSHUser != spec.SSHUser {
		t.Fatalf("created locator = %#v", created)
	}
	if _, err := provider.Create(ctx, spec); err != nil {
		t.Fatalf("idempotent create did not return the existing local guest: %v", err)
	}
	loaded, err := provider.Describe(ctx, created.ID)
	if err != nil || loaded.SSHPort != port || loaded.SSHUser != spec.SSHUser {
		t.Fatalf("Describe() = %#v, %v", loaded, err)
	}
	listed, err := provider.List(ctx)
	if err != nil || len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("List() = %#v, %v", listed, err)
	}
	controls, ok := any(provider).(cloudprovider.SnapshotResetProvider)
	if !ok {
		t.Fatal("local-qemu adapter is missing optional snapshot/reset controls")
	}
	if err := controls.Snapshot(ctx, created.ID, "clean"); err != nil {
		t.Fatalf("Snapshot() = %v", err)
	}
	if err := controls.Reset(ctx, created.ID, "clean"); err != nil {
		t.Fatalf("Reset() = %v", err)
	}
	if err := provider.Destroy(ctx, created.ID); err != nil {
		t.Fatalf("Destroy() = %v", err)
	}
	listed, err = provider.List(ctx)
	if err != nil || len(listed) != 0 {
		t.Fatalf("List after destroy = %#v, %v", listed, err)
	}
	if _, err := os.Stat(filepath.Join(testRoots.DataDir, "local-qemu", created.ID)); !os.IsNotExist(err) {
		t.Fatalf("owned test-root guest workdir remains after destroy: %v", err)
	}
	if _, err := os.Stat(filepath.Join(primary.DataDir, "local-qemu")); !os.IsNotExist(err) {
		t.Fatalf("test-mode guest escaped into primary data root: %v", err)
	}
	stats := roots.LeaseStats()
	if stats.TestRootWrites == 0 || stats.PrimaryWritesDuringTestMode != 0 {
		t.Fatalf("file-route writes = %#v, want test-root writes only", stats)
	}
}
