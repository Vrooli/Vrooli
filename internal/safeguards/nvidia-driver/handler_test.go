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
	RemoteDesktopStateFn = func() (bool, bool) { return RemoteDesktopActiveFn(), true }
	status := hostreqkit.ItemStatus{ExecutionState: hostreqkit.ExecutionPending, PackageName: "nvidia-driver-580-open"}
	out, err := newHandler().Apply(linuxHost(), status, hostreqkit.EnsureOptions{})
	if err != nil || out.ExecutionState != hostreqkit.ExecutionManualActionRequired || out.BlockingReason != hostreqkit.BlockingNeedsMaintenanceWindow {
		t.Fatalf("Apply() = %#v, %v", out, err)
	}
}

func TestApplyRequiresConsentWhenRemoteDesktopStateIsUndetermined(t *testing.T) {
	restore := stub(t)
	defer restore()
	RemoteDesktopStateFn = func() (bool, bool) { return false, false }
	status := hostreqkit.ItemStatus{ExecutionState: hostreqkit.ExecutionPending, PackageName: "nvidia-driver-580-open"}
	out, err := newHandler().Apply(linuxHost(), status, hostreqkit.EnsureOptions{})
	if err != nil || out.ExecutionState != hostreqkit.ExecutionManualActionRequired || out.BlockingReason != hostreqkit.BlockingUndeterminedNeedsConsent {
		t.Fatalf("Apply() = %#v, %v", out, err)
	}
}

func TestDryRunMatchesRemoteDesktopGate(t *testing.T) {
	restore := stub(t)
	defer restore()
	RemoteDesktopActiveFn = func() bool { return true }
	RemoteDesktopStateFn = func() (bool, bool) { return RemoteDesktopActiveFn(), true }
	status := hostreqkit.ItemStatus{ExecutionState: hostreqkit.ExecutionPending, PackageName: "nvidia-driver-580-open"}
	comparison, err := hostreqkit.CompareDryRunAndApply(newHandler(), linuxHost(), status, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if comparison.DryRun.ExecutionState == hostreqkit.ExecutionWouldApply {
		t.Fatalf("blocked dry-run reported would_apply: %#v", comparison.DryRun)
	}
	if comparison.DryRun.BlockingReason != hostreqkit.BlockingNeedsMaintenanceWindow || comparison.Apply.BlockingReason != hostreqkit.BlockingNeedsMaintenanceWindow {
		t.Fatalf("gate outcomes = dry %#v apply %#v", comparison.DryRun, comparison.Apply)
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
	oldReadDir, oldRead, oldRun, oldRoot, oldRemote, oldRemoteState := readDirFn, hostreqkit.ReadFileFn, hostreqkit.RunCommandFn, hostreqkit.RunningAsRootFn, RemoteDesktopActiveFn, RemoteDesktopStateFn
	// DriverReadyFn and the two persistence probes are restored as well, so a
	// test that overrides them cannot leak into the next one. Their defaults
	// shell out to nvidia-smi, which must never happen in a unit test.
	oldDriver, oldPersistMode, oldPersistPresent := DriverReadyFn, PersistenceModeReadyFn, PersistencedPresentFn
	readDirFn = func(string) ([]os.DirEntry, error) { return nil, fs.ErrNotExist }
	hostreqkit.ReadFileFn = func(string) ([]byte, error) { return nil, fs.ErrNotExist }
	hostreqkit.RunCommandFn = func(string, []string, hostreqkit.EnsureOptions) error { return errors.New("unexpected command") }
	hostreqkit.RunningAsRootFn = func() bool { return false }
	RemoteDesktopActiveFn = func() bool { return false }
	RemoteDesktopStateFn = func() (bool, bool) { return false, true }
	DriverReadyFn = func() bool { return false }
	PersistenceModeReadyFn = func() bool { return false }
	PersistencedPresentFn = func() bool { return true }
	return func() {
		readDirFn, hostreqkit.ReadFileFn, hostreqkit.RunCommandFn, hostreqkit.RunningAsRootFn, RemoteDesktopActiveFn, RemoteDesktopStateFn = oldReadDir, oldRead, oldRun, oldRoot, oldRemote, oldRemoteState
		DriverReadyFn, PersistenceModeReadyFn, PersistencedPresentFn = oldDriver, oldPersistMode, oldPersistPresent
	}
}

// systemdHost is the linux host with the systemd support the device-node
// persistence repair requires.
func systemdHost() hostreqkit.Host {
	host := linuxHost()
	host.SupportsSystemd = true
	return host
}

// readyDriverOnNvidiaHost puts Inspect on the branch where the driver answers
// NVML, which is the only branch device-node durability is evaluated on.
func readyDriverOnNvidiaHost(t *testing.T) {
	t.Helper()
	setNvidiaPCI(t)
	DriverReadyFn = func() bool { return true }
}

// recordCommands captures the privileged transaction Apply performs.
func recordCommands(commands *[]string) {
	hostreqkit.RunningAsRootFn = func() bool { return true }
	hostreqkit.RunCommandFn = func(name string, args []string, _ hostreqkit.EnsureOptions) error {
		// The install source is a temp path, so record only the destination.
		if name == "install" && len(args) > 0 {
			args = args[len(args)-1:]
		}
		*commands = append(*commands, strings.TrimSpace(name+" "+strings.Join(args, " ")))
		return nil
	}
}

func setNvidiaPCI(t *testing.T) {
	t.Helper()
	readDirFn = func(string) ([]os.DirEntry, error) { return []os.DirEntry{fakeEntry("0000:01:00.0")}, nil }
	hostreqkit.ReadFileFn = nvidiaPCIFile
}

// nvidiaPCIFile answers the PCI sysfs probe and nothing else, so a test that
// also needs to serve the drop-in can delegate the rest here.
func nvidiaPCIFile(path string) ([]byte, error) {
	if strings.HasSuffix(path, "/vendor") {
		return []byte("0x10de\n"), nil
	}
	if strings.HasSuffix(path, "/class") {
		return []byte("0x030000\n"), nil
	}
	return nil, fs.ErrNotExist
}

type fakeEntry string

func (f fakeEntry) Name() string             { return string(f) }
func (fakeEntry) IsDir() bool                { return true }
func (fakeEntry) Type() fs.FileMode          { return fs.ModeDir }
func (fakeEntry) Info() (fs.FileInfo, error) { return nil, nil }

// --- device-node durability (the second readiness dimension) ---

func TestInspectReadyDriverWithoutPersistenceIsPending(t *testing.T) {
	restore := stub(t)
	defer restore()
	readyDriverOnNvidiaHost(t)
	status := newHandler().Inspect(systemdHost(), requirement())
	if status.ExecutionState != hostreqkit.ExecutionPending {
		t.Fatalf("execution state = %q, want pending: %#v", status.ExecutionState, status)
	}
	if !status.InstallSupported || status.Applied {
		t.Fatalf("install_supported=%v applied=%v, want repairable and not yet applied", status.InstallSupported, status.Applied)
	}
	if status.PackageName != "" {
		t.Fatalf("package_name = %q, want empty so Apply routes to the persistence plan", status.PackageName)
	}
	if !status.Installed {
		t.Fatal("a ready driver must still report installed")
	}
	if !strings.Contains(strings.Join(status.Notes, "\n"), persistenceDropIn) {
		t.Fatalf("notes do not name the file that will be written: %v", status.Notes)
	}
}

func TestInspectReadyDriverWithPersistenceIsAlreadyPresent(t *testing.T) {
	restore := stub(t)
	defer restore()
	readyDriverOnNvidiaHost(t)
	PersistenceModeReadyFn = func() bool { return true }
	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == persistenceDropIn {
			return []byte(persistenceContent), nil
		}
		return nvidiaPCIFile(path)
	}
	status := newHandler().Inspect(systemdHost(), requirement())
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent || !status.Applied {
		t.Fatalf("status = %#v, want already_present and applied", status)
	}
}

// A driver reporting persistence mode without the managed override is a
// transient state someone set by hand; it must still be made durable.
func TestInspectPersistenceWithoutManagedOverrideIsPending(t *testing.T) {
	restore := stub(t)
	defer restore()
	readyDriverOnNvidiaHost(t)
	PersistenceModeReadyFn = func() bool { return true }
	status := newHandler().Inspect(systemdHost(), requirement())
	if status.ExecutionState != hostreqkit.ExecutionPending {
		t.Fatalf("execution state = %q, want pending: %#v", status.ExecutionState, status)
	}
}

func TestInspectWithoutSystemdDoesNotClaimRepairable(t *testing.T) {
	restore := stub(t)
	defer restore()
	readyDriverOnNvidiaHost(t)
	status := newHandler().Inspect(linuxHost(), requirement())
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent || status.InstallSupported {
		t.Fatalf("status = %#v, want already_present with no repair offered", status)
	}
	if !strings.Contains(strings.Join(status.Notes, "\n"), "no systemd") {
		t.Fatalf("notes must explain why persistence is unmanaged: %v", status.Notes)
	}
}

func TestInspectWithoutPersistencedDoesNotClaimRepairable(t *testing.T) {
	restore := stub(t)
	defer restore()
	readyDriverOnNvidiaHost(t)
	PersistencedPresentFn = func() bool { return false }
	status := newHandler().Inspect(systemdHost(), requirement())
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent || status.InstallSupported {
		t.Fatalf("status = %#v, want already_present with no repair offered", status)
	}
}

func TestApplyPersistenceInstallsOverrideAndReloadsDaemon(t *testing.T) {
	restore := stub(t)
	defer restore()
	var commands []string
	recordCommands(&commands)
	PersistenceModeReadyFn = func() bool { return true }
	status := hostreqkit.ItemStatus{ExecutionState: hostreqkit.ExecutionPending}
	out, err := newHandler().Apply(systemdHost(), status, hostreqkit.EnsureOptions{})
	if err != nil || out.ExecutionState != hostreqkit.ExecutionApplied || !out.Applied {
		t.Fatalf("Apply() = %#v, %v; want applied", out, err)
	}
	want := []string{
		"mkdir -p " + persistenceDropInDir,
		"install " + persistenceDropIn,
		"systemctl daemon-reload",
		"systemctl enable " + persistencedUnit,
		"systemctl restart " + persistencedUnit,
	}
	if strings.Join(commands, "\n") != strings.Join(want, "\n") {
		t.Fatalf("commands =\n%s\nwant\n%s", strings.Join(commands, "\n"), strings.Join(want, "\n"))
	}
}

// The override can be installed correctly and still not be live until a
// reboot. That must be reported as a typed state, never as success.
func TestApplyPersistenceReportsRebootWhenDriverStillDisagrees(t *testing.T) {
	restore := stub(t)
	defer restore()
	var commands []string
	recordCommands(&commands)
	PersistenceModeReadyFn = func() bool { return false }
	status := hostreqkit.ItemStatus{ExecutionState: hostreqkit.ExecutionPending}
	out, err := newHandler().Apply(systemdHost(), status, hostreqkit.EnsureOptions{})
	if err != nil || out.ExecutionState != hostreqkit.ExecutionRebootRequired {
		t.Fatalf("Apply() = %#v, %v; want reboot_required", out, err)
	}
}

func TestApplyPersistenceDryRunTouchesNothing(t *testing.T) {
	restore := stub(t)
	defer restore()
	var commands []string
	recordCommands(&commands)
	status := hostreqkit.ItemStatus{ExecutionState: hostreqkit.ExecutionPending}
	out, err := newHandler().Apply(systemdHost(), status, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil || out.ExecutionState != hostreqkit.ExecutionWouldApply {
		t.Fatalf("Apply() = %#v, %v; want would_apply", out, err)
	}
	if len(commands) != 0 {
		t.Fatalf("dry-run ran commands: %v", commands)
	}
}

func TestApplyPersistenceWithoutPersistencedFails(t *testing.T) {
	restore := stub(t)
	defer restore()
	PersistencedPresentFn = func() bool { return false }
	status := hostreqkit.ItemStatus{ExecutionState: hostreqkit.ExecutionPending}
	out, err := newHandler().Apply(systemdHost(), status, hostreqkit.EnsureOptions{})
	if err != nil || out.ExecutionState != hostreqkit.ExecutionFailed {
		t.Fatalf("Apply() = %#v, %v; want failed", out, err)
	}
}

// The driver-repair plan must keep its maintenance-window gate now that Apply
// dispatches between two plans.
func TestApplyStillRoutesNamedPackagesToDriverRepair(t *testing.T) {
	restore := stub(t)
	defer restore()
	var commands []string
	recordCommands(&commands)
	status := hostreqkit.ItemStatus{ExecutionState: hostreqkit.ExecutionPending, PackageName: "nvidia-driver-580-open"}
	out, err := newHandler().Apply(systemdHost(), status, hostreqkit.EnsureOptions{})
	if err != nil || out.ExecutionState != hostreqkit.ExecutionRebootRequired {
		t.Fatalf("Apply() = %#v, %v; want the package plan", out, err)
	}
	if len(commands) != 1 || !strings.HasPrefix(commands[0], "apt-get install") {
		t.Fatalf("commands = %v, want a single apt-get install", commands)
	}
}
