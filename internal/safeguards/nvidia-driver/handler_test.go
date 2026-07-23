package nvidiadriver

import (
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestInspectMissingDriverBuildsRunningKernelRepair(t *testing.T) {
	restore := stub(t)
	defer restore()
	setNvidiaPCI(t)
	DriverReadyFn = func() bool { return false }
	RunningKernelFn = func() (string, error) { return "7.0.0-28-generic", nil }
	InstalledPackagesFn = func() ([]string, error) {
		return []string{"nvidia-driver-580-open", "linux-modules-nvidia-580-open-generic-hwe-24.04"}, nil
	}
	PackageAvailableFn = func(string) bool { return true }
	status := newHandler().Inspect(linuxHost(), requirement())
	if status.ExecutionState != hostreqkit.ExecutionPending || !strings.Contains(status.PackageName, "linux-modules-nvidia-580-open-7.0.0-28-generic") {
		t.Fatalf("unexpected status: %#v", status)
	}
	if !strings.Contains(status.PackageName, "linux-modules-nvidia-580-open-generic-hwe-24.04") {
		t.Fatalf("future-kernel meta package missing: %q", status.PackageName)
	}
}

func TestApplyReturnsTypedRebootRequired(t *testing.T) {
	restore := stub(t)
	defer restore()
	status := hostreqkit.ItemStatus{ExecutionState: hostreqkit.ExecutionPending, PackageName: "nvidia-driver-580-open, linux-modules-nvidia-580-open-7.0.0-28-generic"}
	var gotName string
	var gotArgs []string
	hostreqkit.RunCommandFn = func(name string, args []string, _ hostreqkit.EnsureOptions) error {
		gotName, gotArgs = name, args
		return nil
	}
	hostreqkit.RunningAsRootFn = func() bool { return true }
	RemoteDesktopActiveFn = func() bool { return false }
	out, err := newHandler().Apply(linuxHost(), status, hostreqkit.EnsureOptions{})
	if err != nil || out.ExecutionState != hostreqkit.ExecutionRebootRequired {
		t.Fatalf("Apply() = %#v, %v", out, err)
	}
	if gotName != "apt-get" || !strings.Contains(strings.Join(gotArgs, " "), "--no-install-recommends") {
		t.Fatalf("unexpected transaction: %s %v", gotName, gotArgs)
	}
}

func TestApplyBlocksLiveDriverTransitionDuringRemoteDesktopSession(t *testing.T) {
	restore := stub(t)
	defer restore()
	RemoteDesktopActiveFn = func() bool { return true }
	status := hostreqkit.ItemStatus{ExecutionState: hostreqkit.ExecutionPending, PackageName: "nvidia-driver-580-open"}
	out, err := newHandler().Apply(linuxHost(), status, hostreqkit.EnsureOptions{})
	if err != nil || out.ExecutionState != hostreqkit.ExecutionManualActionRequired || out.BlockingReason != hostreqkit.BlockingNeedsMaintenanceWindow {
		t.Fatalf("Apply() = %#v, %v", out, err)
	}
}

func TestInspectNoNvidiaHardwareIsNotApplicable(t *testing.T) {
	restore := stub(t)
	defer restore()
	status := newHandler().Inspect(linuxHost(), requirement())
	if status.ExecutionState != hostreqkit.ExecutionNotApplicable {
		t.Fatalf("status = %#v", status)
	}
}

func newHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.SafeguardManifest{Name: "nvidia_driver", Handler: "nvidia_driver"})
}
func linuxHost() hostreqkit.Host { return hostreqkit.Host{OS: "linux", PackageManager: "apt-get"} }
func requirement() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{Name: "nvidia_driver", Kind: hostreqspec.KindSafeguard, Required: true}
}

func stub(t *testing.T) func() {
	t.Helper()
	oldReadDir, oldRead, oldRun, oldRoot, oldRemote := readDirFn, hostreqkit.ReadFileFn, hostreqkit.RunCommandFn, hostreqkit.RunningAsRootFn, RemoteDesktopActiveFn
	readDirFn = func(string) ([]os.DirEntry, error) { return nil, fs.ErrNotExist }
	hostreqkit.ReadFileFn = func(string) ([]byte, error) { return nil, fs.ErrNotExist }
	hostreqkit.RunCommandFn = func(string, []string, hostreqkit.EnsureOptions) error { return errors.New("unexpected command") }
	hostreqkit.RunningAsRootFn = func() bool { return false }
	RemoteDesktopActiveFn = func() bool { return false }
	return func() {
		readDirFn, hostreqkit.ReadFileFn, hostreqkit.RunCommandFn, hostreqkit.RunningAsRootFn, RemoteDesktopActiveFn = oldReadDir, oldRead, oldRun, oldRoot, oldRemote
	}
}

func setNvidiaPCI(t *testing.T) {
	t.Helper()
	readDirFn = func(string) ([]os.DirEntry, error) { return []os.DirEntry{fakeEntry("0000:01:00.0")}, nil }
	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if strings.HasSuffix(path, "/vendor") {
			return []byte("0x10de\n"), nil
		}
		if strings.HasSuffix(path, "/class") {
			return []byte("0x030000\n"), nil
		}
		return nil, fs.ErrNotExist
	}
}

type fakeEntry string

func (f fakeEntry) Name() string             { return string(f) }
func (fakeEntry) IsDir() bool                { return true }
func (fakeEntry) Type() fs.FileMode          { return fs.ModeDir }
func (fakeEntry) Info() (fs.FileInfo, error) { return nil, nil }
