package localqemu

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	cloudprovider "compute-manager/internal/provider"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/vrooli/packages/capacity"
)

type refusingAdmission struct{ calls int }

func (a *refusingAdmission) Acquire(context.Context, AdmissionRequest) (*QEMULease, error) {
	a.calls++
	return nil, errors.New("disk floor denied")
}
func (a *refusingAdmission) Reattach(context.Context, CapacityReceipt) (*QEMULease, error) {
	return nil, errors.New("not used")
}
func (a *refusingAdmission) FindOrphan(context.Context, string, string) (CapacityReceipt, bool, error) {
	return CapacityReceipt{}, false, nil
}
func (a *refusingAdmission) Reconcile(context.Context, CapacityReceipt) error        { return nil }
func (a *refusingAdmission) ReleaseResources(context.Context, CapacityReceipt) error { return nil }

func TestAdapterRefusesQEMUCreateBeforeAnyQEMUEffectWithoutCapacityAdmission(t *testing.T) {
	primary := storage.Paths{ConfigDir: filepath.Join(t.TempDir(), "config"), DataDir: filepath.Join(t.TempDir(), "data"), CacheDir: filepath.Join(t.TempDir(), "cache"), LogsDir: filepath.Join(t.TempDir(), "logs"), StateDir: filepath.Join(t.TempDir(), "state")}
	for _, path := range []string{primary.ConfigDir, primary.DataDir, primary.CacheDir, primary.LogsDir, primary.StateDir} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	roots := filerouting.New(primary)
	if _, err := roots.InstallLeasedTestRoots("local-qemu-capacity-refusal", time.Minute, false); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = roots.ClearTestRoots("local-qemu-capacity-refusal") })
	image := filepath.Join(t.TempDir(), "fixture.img")
	if err := os.WriteFile(image, []byte("read-only fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	qemuCreateCalls, starts := 0, 0
	qemu := LocalQEMUProvider{
		MeasureImage: func(string) (int64, error) { return 1 << 30, nil },
		LookPath:     func(name string) (string, error) { return "/fixture/" + name, nil },
		RunImage: func(...string) error {
			qemuCreateCalls++
			return nil
		},
		BuildCloudInit: func(string, string, string, string) error { return nil },
		StartProcess: func(string, []string, string) (int, error) {
			starts++
			return 12345, nil
		},
	}
	admission := &refusingAdmission{}
	adapter := &Adapter{Roots: roots, QEMU: qemu, Admission: admission, RequireCapacity: true}
	ctx := database.WithTestMode(context.Background())
	if err := adapter.RecoverCapacity(ctx); err != nil {
		t.Fatal(err)
	}
	_, err := adapter.Create(ctx, cloudprovider.Spec{
		Region: "local", Size: "headless-linux", Image: image, Memory: "2048", CPUs: 2,
		Profile: string(ProfileHeadlessLinux), SSHUser: "vrooli", SSHAuthorizedKey: "ssh-ed25519 AAAA fixture",
		Tags: map[string]string{"vrooli-request": "capacity-refusal"},
	})
	if err == nil || admission.calls != 1 || qemuCreateCalls != 0 || starts != 0 {
		t.Fatalf("Create err=%v admission=%d qemu-img writes=%d process starts=%d", err, admission.calls, qemuCreateCalls, starts)
	}
}

func TestAdapterMissingBrokerIsNotReadyAndRefusesBeforeQEMU(t *testing.T) {
	root := filepath.Join(t.TempDir(), "manager-data")
	qemuEffects := 0
	adapter := &Adapter{
		DataDir: root, RequireCapacity: true,
		QEMU: LocalQEMUProvider{
			MeasureImage: func(string) (int64, error) { return 1 << 30, nil },
			RunImage:     func(...string) error { qemuEffects++; return nil },
			StartProcess: func(string, []string, string) (int, error) { qemuEffects++; return 1234, nil },
		},
	}
	if err := adapter.RecoverCapacity(context.Background()); err == nil {
		t.Fatal("manager became ready without the shared capacity broker")
	}
	_, err := adapter.Create(context.Background(), cloudprovider.Spec{
		Region: "local", Size: "headless-linux", Image: "/fixture/image", Memory: "2048", CPUs: 2,
		Profile: string(ProfileHeadlessLinux), SSHUser: "runner", SSHAuthorizedKey: "ssh-ed25519 AAAA fixture",
		Tags: map[string]string{"vrooli-request": "missing-broker"},
	})
	if err == nil || qemuEffects != 0 {
		t.Fatalf("Create err=%v QEMU effects=%d without broker", err, qemuEffects)
	}
}

func TestAdapterBrokerRAMAndDiskRefusalsReachNoQEMUEffect(t *testing.T) {
	for _, tc := range []struct {
		name   string
		broker *fakeCapacityBroker
	}{
		{name: "insufficient RAM and CPU", broker: &fakeCapacityBroker{resourceErr: errors.New("fixture insufficient RAM/CPU"), verdict: capacity.Verdict{Kind: "deny", Reason: "insufficient RAM/CPU"}}},
		{name: "120 GiB disk floor", broker: &fakeCapacityBroker{resource: &fakeResourceLease{}, diskErr: errors.New("fixture 120 GiB reserve denied")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "manager-data")
			qemuEffects := 0
			adapter := &Adapter{
				DataDir: root, Admission: NewBrokerAdmission(tc.broker), RequireCapacity: true,
				QEMU: LocalQEMUProvider{
					MeasureImage: func(string) (int64, error) { return 1 << 30, nil },
					RunImage:     func(...string) error { qemuEffects++; return nil },
					StartProcess: func(string, []string, string) (int, error) { qemuEffects++; return 1234, nil },
				},
			}
			if err := adapter.RecoverCapacity(context.Background()); err != nil {
				t.Fatal(err)
			}
			defer adapter.CloseCapacity()
			_, err := adapter.Create(context.Background(), cloudprovider.Spec{
				Region: "local", Size: "headless-linux", Image: "/fixture/base.img", Memory: "2048", CPUs: 2,
				Profile: string(ProfileHeadlessLinux), SSHUser: "runner", SSHAuthorizedKey: "ssh-ed25519 AAAA fixture",
				Tags: map[string]string{"vrooli-request": "capacity-refusal-" + tc.name},
			})
			if err == nil || qemuEffects != 0 {
				t.Fatalf("Create err=%v QEMU image/process effects=%d", err, qemuEffects)
			}
			if tc.name == "insufficient RAM and CPU" && tc.broker.request.OwnerID != "" {
				t.Fatalf("RAM/CPU refusal continued to disk reservation: %#v", tc.broker.request)
			}
			if tc.name == "120 GiB disk floor" && (tc.broker.request.FloorBytes != qemuDiskFloor || tc.broker.resource.releases != 1) {
				t.Fatalf("disk denial did not preserve/refund correct reservation: %#v", tc.broker)
			}
		})
	}
}

func TestAdapterCapacityRecoverySerializesOneLocalQEMUOwner(t *testing.T) {
	primary := storage.Paths{ConfigDir: filepath.Join(t.TempDir(), "config"), DataDir: filepath.Join(t.TempDir(), "data"), CacheDir: filepath.Join(t.TempDir(), "cache"), LogsDir: filepath.Join(t.TempDir(), "logs"), StateDir: filepath.Join(t.TempDir(), "state")}
	for _, path := range []string{primary.ConfigDir, primary.DataDir, primary.CacheDir, primary.LogsDir, primary.StateDir} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	roots := filerouting.New(primary)
	if _, err := roots.InstallLeasedTestRoots("local-qemu-owner-lock", time.Minute, false); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = roots.ClearTestRoots("local-qemu-owner-lock") })
	ctx := database.WithTestMode(context.Background())
	first := &Adapter{Roots: roots, Admission: &refusingAdmission{}, RequireCapacity: true}
	if err := first.RecoverCapacity(ctx); err != nil {
		t.Fatal(err)
	}
	second := &Adapter{Roots: roots, Admission: &refusingAdmission{}, RequireCapacity: true}
	if err := second.RecoverCapacity(ctx); err == nil {
		t.Fatal("second Compute Manager process acquired the local-QEMU owner lock")
	}
	if err := first.CloseCapacity(); err != nil {
		t.Fatal(err)
	}
	if err := second.RecoverCapacity(ctx); err != nil {
		t.Fatalf("second owner could not recover after the first lock was released: %v", err)
	}
	if err := second.CloseCapacity(); err != nil {
		t.Fatal(err)
	}
}

func TestStopAndCleanRefusesToSignalUnrelatedReusedPID(t *testing.T) {
	workdir := t.TempDir()
	marker := filepath.Join(workdir, "keep")
	if err := os.WriteFile(marker, []byte("owned fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	adapter := &Adapter{}
	instance := Instance{ID: "qemu-0123456789abcdef01234567", Provider: "local-qemu", State: StateRunning, PID: os.Getpid(), Workdir: workdir}
	if err := adapter.stopAndClean(context.Background(), instance, nil); err == nil {
		t.Fatal("stopAndClean accepted an unrelated process as the local-QEMU guest")
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("unverified guest workdir was removed: %v", err)
	}
}
