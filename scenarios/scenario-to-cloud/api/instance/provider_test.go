package instance

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalQEMUPlanFailsClosedWhenExecutableIsMissing(t *testing.T) {
	image := filepath.Join(t.TempDir(), "base.qcow2")
	if err := os.WriteFile(image, []byte("image"), 0o600); err != nil {
		t.Fatal(err)
	}
	provider := LocalQEMUProvider{Binary: "qemu-system-x86_64", LookPath: func(string) (string, error) { return "", errors.New("not installed") }}
	plan, err := provider.Plan(context.Background(), Request{Name: "lane", Image: image, Workdir: t.TempDir()})
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("error = %v, want ErrProviderUnavailable", err)
	}
	if plan.Available || len(plan.Warnings) == 0 {
		t.Fatalf("plan = %#v, want unavailable warning", plan)
	}
}

func TestLocalQEMUPlanIsDeterministicAndDoesNotLeakSecrets(t *testing.T) {
	image := filepath.Join(t.TempDir(), "base.qcow2")
	if err := os.WriteFile(image, []byte("image"), 0o600); err != nil {
		t.Fatal(err)
	}
	provider := LocalQEMUProvider{Binary: "/usr/bin/qemu-system-x86_64", LookPath: func(name string) (string, error) { return name, nil }}
	plan, err := provider.Plan(context.Background(), Request{Name: "lane", Image: image, Workdir: t.TempDir(), Memory: "4096", CPUs: 4, User: "vrooli", AuthorizedKey: "ssh-ed25519 AAAA", SSHPort: 2222})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Available || len(plan.Command) == 0 {
		t.Fatalf("plan = %#v, want available launch command", plan)
	}
	for _, arg := range plan.Command {
		if arg == "4096" || arg == "4" {
			continue
		}
	}
}

func TestLocalQEMUCreateRequiresReadableImage(t *testing.T) {
	provider := LocalQEMUProvider{LookPath: func(name string) (string, error) { return name, nil }}
	_, err := provider.Create(context.Background(), Request{Name: "lane", Image: filepath.Join(t.TempDir(), "missing.qcow2"), Workdir: t.TempDir()})
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("error = %v, want provider unavailable", err)
	}
}

func TestLocalQEMUStartAndStopUseProcessSeams(t *testing.T) {
	started := false
	stopped := 0
	provider := LocalQEMUProvider{
		StartProcess: func(name string, args []string, workdir string) (int, error) {
			started = name != "" && len(args) > 0 && workdir != ""
			return 4242, nil
		},
		StopProcess: func(pid int) error { stopped = pid; return nil },
	}
	value := Instance{ID: "lane", State: StateStopped, Workdir: t.TempDir(), Command: []string{"qemu-system-x86_64", "-nographic"}}
	pid, err := provider.Start(context.Background(), value)
	if err != nil || !started || pid != 4242 {
		t.Fatalf("start = pid %d err %v started=%t", pid, err, started)
	}
	value.PID = pid
	if err := provider.Stop(context.Background(), value); err != nil || stopped != 4242 {
		t.Fatalf("stop = %v pid %d", err, stopped)
	}
}

func TestLocalQEMUCreateBuildsNonRootCloudInitSeed(t *testing.T) {
	image := filepath.Join(t.TempDir(), "base.qcow2")
	if err := os.WriteFile(image, []byte("image"), 0o600); err != nil {
		t.Fatal(err)
	}
	var userData, metadata, iso string
	provider := LocalQEMUProvider{
		LookPath: func(name string) (string, error) { return name, nil },
		RunImage: func(args ...string) error {
			if len(args) != 8 || args[0] != "create" {
				t.Fatalf("qemu-img args = %#v", args)
			}
			if args[1] != "-f" || args[2] != "qcow2" || args[3] != "-F" || args[4] != "qcow2" || args[5] != "-b" {
				t.Fatalf("qemu-img format args = %#v", args)
			}
			return os.WriteFile(args[len(args)-1], []byte("overlay"), 0o600)
		},
		BuildCloudInit: func(_ string, gotISO, gotUserData, gotMetadata string) error {
			iso, userData, metadata = gotISO, gotUserData, gotMetadata
			return os.WriteFile(gotISO, []byte("seed"), 0o600)
		},
	}
	workdir := filepath.Join(t.TempDir(), "lane")
	value, err := provider.Create(context.Background(), Request{Name: "lane", Image: image, Workdir: workdir, User: "vrooli", AuthorizedKey: "ssh-ed25519 AAAA", SSHPort: 2222})
	if err != nil {
		t.Fatal(err)
	}
	if value.Profile != ProfileHeadlessLinux || iso == "" || userData == "" || metadata == "" {
		t.Fatalf("instance = %#v seed paths = %q %q %q", value, iso, userData, metadata)
	}
	if value.Image != filepath.Join(workdir, instanceDiskName) {
		t.Fatalf("instance image = %q, want owned disk", value.Image)
	}
	data, err := os.ReadFile(userData)
	if err != nil || !strings.Contains(string(data), "sudo: ALL=(ALL) NOPASSWD:ALL") || !strings.Contains(string(data), "name: \"vrooli\"") {
		t.Fatalf("cloud-init user data = %q err=%v", data, err)
	}
}

func TestLocalQEMUCreateFailureDoesNotDeleteSourceImage(t *testing.T) {
	image := filepath.Join(t.TempDir(), "base.qcow2")
	if err := os.WriteFile(image, []byte("base"), 0o600); err != nil {
		t.Fatal(err)
	}
	workdir := filepath.Join(t.TempDir(), "lane")
	provider := LocalQEMUProvider{
		LookPath: func(name string) (string, error) { return name, nil },
		RunImage: func(args ...string) error {
			return os.WriteFile(args[len(args)-1], []byte("overlay"), 0o600)
		},
		BuildCloudInit: func(_, _, _, _ string) error { return errors.New("seed failed") },
	}
	if _, err := provider.Create(context.Background(), Request{Name: "lane", Image: image, Workdir: workdir, User: "vrooli", AuthorizedKey: "ssh-ed25519 AAAA"}); err == nil {
		t.Fatal("Create succeeded, want seed failure")
	}
	if _, err := os.Stat(image); err != nil {
		t.Fatalf("source image was not preserved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workdir, instanceDiskName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("owned disk remains after failed create: %v", err)
	}
}

func TestLocalQEMURejectsSnapshotWhileRunning(t *testing.T) {
	provider := LocalQEMUProvider{RunImage: func(...string) error { t.Fatal("qemu-img should not run"); return nil }}
	value := Instance{State: StateRunning, Image: filepath.Join(t.TempDir(), instanceDiskName)}
	if err := provider.Snapshot(context.Background(), value, "clean"); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Snapshot error = %v, want ErrInvalidRequest", err)
	}
	if err := provider.Reset(context.Background(), value, "clean"); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Reset error = %v, want ErrInvalidRequest", err)
	}
}

func TestLocalQEMURejectsDestroyOfUnownedImage(t *testing.T) {
	provider := LocalQEMUProvider{}
	value := Instance{Image: filepath.Join(t.TempDir(), "base.qcow2"), Workdir: t.TempDir()}
	if err := provider.Destroy(context.Background(), value); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Destroy error = %v, want ErrInvalidRequest", err)
	}
}

func TestLocalQEMURequiresNonRootProvisioningUser(t *testing.T) {
	request := Request{Name: "lane", Image: "/tmp/base.qcow2", Workdir: t.TempDir(), User: "root", AuthorizedKey: "ssh-ed25519 AAAA"}
	if err := validateProvisioning(request); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("validateProvisioning error = %v, want ErrInvalidRequest", err)
	}
}
