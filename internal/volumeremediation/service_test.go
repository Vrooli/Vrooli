package volumeremediation

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeObserver struct {
	state State
	err   error
	calls int
	// after, when set, is returned from the second observation onwards so a
	// test can model the host actually changing.
	after *State
}

func (f *fakeObserver) Observe(context.Context, string) (State, error) {
	f.calls++
	if f.err != nil {
		return State{}, f.err
	}
	if f.after != nil && f.calls > 1 {
		return *f.after, nil
	}
	return f.state, nil
}

type fakeRunner struct {
	out  []byte
	err  error
	argv [][]string
}

func (f *fakeRunner) Run(_ context.Context, argv []string) ([]byte, error) {
	f.argv = append(f.argv, append([]string(nil), argv...))
	return f.out, f.err
}

func elementsDevice() Device {
	return Device{
		Path:       "/dev/sda1",
		Filesystem: "ntfs3",
		UUID:       "E26A883E6A881189",
		Serial:     "WD-WX52A946D6VL",
		TotalBytes: 2000363192320,
	}
}

func unmountedDirtyState() State {
	d := elementsDevice()
	d.Mountpoint = ""
	return State{Device: d, Mounted: false, Dirty: TristateYes, Evidence: "/proc/fs/ntfs3/sda1/volinfo"}
}

func linuxService(t *testing.T, obs Observer, runner Runner) *Service {
	t.Helper()
	return New(Options{
		Observer: obs,
		Runner:   runner,
		GOOS:     "linux",
		LookPath: func(tool string) (string, error) {
			switch tool {
			case "udisksctl", "busctl", "mount", "umount":
				return "/usr/bin/" + tool, nil
			}
			return "", errors.New("not found")
		},
	})
}

func TestRepairRefusesWhileTheVolumeIsMounted(t *testing.T) {
	mounted := unmountedDirtyState()
	mounted.Mounted = true
	mounted.ReadOnly = true
	mounted.Device.Mountpoint = "/media/user/Elements"
	runner := &fakeRunner{}
	svc := linuxService(t, &fakeObserver{state: mounted}, runner)

	res, err := svc.Execute(context.Background(), Request{Action: ActionRepair, Device: elementsDevice(), AcknowledgeDataLoss: true})

	if err == nil {
		t.Fatal("expected a refusal for repairing a mounted volume")
	}
	if res.Status != StatusRefused {
		t.Fatalf("status = %q, want %q", res.Status, StatusRefused)
	}
	if len(runner.argv) != 0 {
		t.Fatalf("a refused repair must not execute anything, ran %v", runner.argv)
	}
}

func TestRepairRequiresDataLossAcknowledgement(t *testing.T) {
	runner := &fakeRunner{}
	svc := linuxService(t, &fakeObserver{state: unmountedDirtyState()}, runner)

	_, err := svc.Execute(context.Background(), Request{Action: ActionRepair, Device: elementsDevice()})

	var refused ErrRefused
	if !errors.As(err, &refused) {
		t.Fatalf("error = %v, want a refusal", err)
	}
	if !strings.Contains(refused.Reason, "data-loss acknowledgement") {
		t.Fatalf("reason = %q, want the acknowledgement gate", refused.Reason)
	}
	if len(runner.argv) != 0 {
		t.Fatal("nothing may execute without acknowledgement")
	}
}

// Without a UUID or serial there is no way to prove the disk in front of us is
// the one that was approved, so a writing action must refuse.
func TestRepairRefusesWithoutAStableIdentity(t *testing.T) {
	device := Device{Path: "/dev/sda1", Filesystem: "ntfs3"}
	state := State{Device: device, Dirty: TristateYes}
	svc := linuxService(t, &fakeObserver{state: state}, &fakeRunner{})

	_, err := svc.Execute(context.Background(), Request{Action: ActionRepair, Device: device, AcknowledgeDataLoss: true})

	var refused ErrRefused
	if !errors.As(err, &refused) {
		t.Fatalf("error = %v, want a refusal", err)
	}
	if !strings.Contains(refused.Reason, "UUID or serial") {
		t.Fatalf("reason = %q, want the identity gate", refused.Reason)
	}
}

// The approval is bound to a disk, not to a device path. A different disk that
// happens to land on the same path must not inherit it.
func TestExecuteRefusesWhenTheDiskWasSwapped(t *testing.T) {
	observed := unmountedDirtyState()
	observed.Device.UUID = "SOME-OTHER-UUID"
	observed.Device.Serial = "OTHER-SERIAL"
	runner := &fakeRunner{}
	svc := linuxService(t, &fakeObserver{state: observed}, runner)

	res, err := svc.Execute(context.Background(), Request{Action: ActionRepair, Device: elementsDevice(), AcknowledgeDataLoss: true})

	if !errors.Is(err, ErrDeviceChanged) {
		t.Fatalf("error = %v, want ErrDeviceChanged", err)
	}
	if res.Status != StatusRefused {
		t.Fatalf("status = %q, want refused", res.Status)
	}
	if len(runner.argv) != 0 {
		t.Fatal("a swapped disk must not be touched")
	}
}

func TestExecuteRefusesSystemVolumes(t *testing.T) {
	for _, mountpoint := range []string{"/", "/boot/efi", "/home", "/usr/local"} {
		t.Run(mountpoint, func(t *testing.T) {
			device := Device{Path: "/dev/nvme0n1p2", Filesystem: "ext4", UUID: "root-uuid", Mountpoint: mountpoint}
			runner := &fakeRunner{}
			svc := linuxService(t, &fakeObserver{state: State{Device: device, Mounted: true}}, runner)

			_, err := svc.Execute(context.Background(), Request{Action: ActionUnmount, Device: device})

			var refused ErrRefused
			if !errors.As(err, &refused) {
				t.Fatalf("error = %v, want a refusal for %s", err, mountpoint)
			}
			if len(runner.argv) != 0 {
				t.Fatal("a system volume must never be touched")
			}
		})
	}
}

func TestRepairRunsTheExpectedCommandAndReObserves(t *testing.T) {
	repaired := unmountedDirtyState()
	repaired.Dirty = TristateNo
	obs := &fakeObserver{state: unmountedDirtyState(), after: &repaired}
	runner := &fakeRunner{out: []byte("Volume is dirty flag cleared")}
	svc := linuxService(t, obs, runner)

	res, err := svc.Execute(context.Background(), Request{Action: ActionRepair, Device: elementsDevice(), AcknowledgeDataLoss: true})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if res.Status != StatusChanged || !res.Changed {
		t.Fatalf("status = %q changed = %v, want changed", res.Status, res.Changed)
	}
	if len(runner.argv) != 1 {
		t.Fatalf("expected exactly one command, got %v", runner.argv)
	}
	if res.State.Dirty != TristateNo {
		t.Fatalf("result must carry the post-action state, got dirty=%q", res.State.Dirty)
	}
	if res.Detail == "" {
		t.Fatal("tool output must be recorded as evidence")
	}
}

// A dry run must validate everything and still touch nothing.
func TestDryRunReportsTheCommandWithoutRunningIt(t *testing.T) {
	runner := &fakeRunner{}
	svc := linuxService(t, &fakeObserver{state: unmountedDirtyState()}, runner)

	res, err := svc.Execute(context.Background(), Request{Action: ActionRepair, Device: elementsDevice(), AcknowledgeDataLoss: true, DryRun: true})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !res.DryRun || res.Changed {
		t.Fatalf("dry run reported as changed: %+v", res)
	}
	if len(res.Command) == 0 {
		t.Fatal("a dry run must report the command it would run")
	}
	if len(runner.argv) != 0 {
		t.Fatalf("a dry run executed %v", runner.argv)
	}
}

// Re-running a completed step is normal in a recovery sequence and must not be
// an error.
func TestIdempotentStepsShortCircuit(t *testing.T) {
	t.Run("unmount when already unmounted", func(t *testing.T) {
		runner := &fakeRunner{}
		svc := linuxService(t, &fakeObserver{state: unmountedDirtyState()}, runner)
		res, err := svc.Execute(context.Background(), Request{Action: ActionUnmount, Device: elementsDevice()})
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if res.Status != StatusAlreadySatisfied || len(runner.argv) != 0 {
			t.Fatalf("status = %q ran = %v", res.Status, runner.argv)
		}
	})
	t.Run("mount when already read/write", func(t *testing.T) {
		state := unmountedDirtyState()
		state.Mounted = true
		state.ReadOnly = false
		state.Device.Mountpoint = "/media/user/Elements"
		runner := &fakeRunner{}
		svc := linuxService(t, &fakeObserver{state: state}, runner)
		res, err := svc.Execute(context.Background(), Request{Action: ActionMountReadWrite, Device: elementsDevice()})
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if res.Status != StatusAlreadySatisfied || len(runner.argv) != 0 {
			t.Fatalf("status = %q ran = %v", res.Status, runner.argv)
		}
	})
}

// Mounting a still-dirty volume read/write would either fail or silently mask
// that the repair never worked.
func TestMountReadWriteRefusesAStillDirtyVolume(t *testing.T) {
	runner := &fakeRunner{}
	svc := linuxService(t, &fakeObserver{state: unmountedDirtyState()}, runner)

	_, err := svc.Execute(context.Background(), Request{Action: ActionMountReadWrite, Device: elementsDevice()})

	var refused ErrRefused
	if !errors.As(err, &refused) {
		t.Fatalf("error = %v, want a refusal", err)
	}
	if !strings.Contains(refused.Reason, "dirty") {
		t.Fatalf("reason = %q, want the dirty gate", refused.Reason)
	}
	if len(runner.argv) != 0 {
		t.Fatal("nothing may run for a refused mount")
	}
}

func TestUnsupportedFilesystemIsRefusedNotGuessedAt(t *testing.T) {
	device := Device{Path: "/dev/sda1", Filesystem: "zfs", UUID: "u", Serial: "s"}
	svc := linuxService(t, &fakeObserver{state: State{Device: device}}, &fakeRunner{})

	_, err := svc.Execute(context.Background(), Request{Action: ActionCheck, Device: device})

	var refused ErrRefused
	if !errors.As(err, &refused) {
		t.Fatalf("error = %v, want a refusal", err)
	}
	if !strings.Contains(refused.Reason, "no repair adapter") {
		t.Fatalf("reason = %q", refused.Reason)
	}
}

func TestBackendSelection(t *testing.T) {
	cases := []struct {
		name  string
		goos  string
		tools map[string]bool
		want  Backend
		error bool
	}{
		{name: "linux prefers udisks", goos: "linux", tools: map[string]bool{"udisksctl": true, "busctl": true, "mount": true, "umount": true}, want: BackendUDisks},
		{name: "linux without udisks falls back", goos: "linux", tools: map[string]bool{"mount": true, "umount": true}, want: BackendNativeTools},
		{name: "linux with nothing is unsupported", goos: "linux", tools: map[string]bool{}, error: true},
		{name: "darwin", goos: "darwin", tools: map[string]bool{"diskutil": true}, want: BackendDiskutil},
		{name: "windows", goos: "windows", tools: map[string]bool{"powershell": true}, want: BackendRepairVolume},
		{name: "unknown platform", goos: "plan9", tools: map[string]bool{}, error: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := New(Options{GOOS: tc.goos, Observer: &fakeObserver{}, Runner: &fakeRunner{}, LookPath: func(tool string) (string, error) {
				if tc.tools[tool] {
					return "/usr/bin/" + tool, nil
				}
				return "", errors.New("not found")
			}})
			got, err := svc.Backend()
			if tc.error {
				if err == nil {
					t.Fatalf("expected an unsupported error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Backend: %v", err)
			}
			if got != tc.want {
				t.Fatalf("backend = %q, want %q", got, tc.want)
			}
		})
	}
}

// udisks2 must not be routed through the privilege broker: polkit already
// authorises it without elevation, and adding one would be a regression.
func TestOnlyNativeToolsNeedElevation(t *testing.T) {
	if BackendUDisks.NeedsElevation() {
		t.Fatal("udisks2 must not require elevation")
	}
	if !BackendNativeTools.NeedsElevation() {
		t.Fatal("direct filesystem tools require the privilege broker")
	}
	if BackendDiskutil.NeedsElevation() || BackendRepairVolume.NeedsElevation() {
		t.Fatal("platform-native backends carry their own authorisation")
	}
}

type fakeElevated struct {
	available bool
	status    string
	detail    string
	err       error
	calls     []Action
}

func (f *fakeElevated) Available() bool { return f.available }

func (f *fakeElevated) CheckFilesystem(context.Context, Device) (string, string, error) {
	f.calls = append(f.calls, ActionCheck)
	return f.status, f.detail, f.err
}

func (f *fakeElevated) RepairFilesystem(context.Context, Device) (string, string, error) {
	f.calls = append(f.calls, ActionRepair)
	return f.status, f.detail, f.err
}

// nativeToolsService models a Linux host with no udisks2 client.
func nativeToolsService(t *testing.T, obs Observer, runner Runner, elevated Elevated) *Service {
	t.Helper()
	return New(Options{
		Observer: obs, Runner: runner, Elevated: elevated, GOOS: "linux",
		LookPath: func(tool string) (string, error) {
			switch tool {
			case "mount", "umount":
				return "/usr/bin/" + tool, nil
			}
			return "", errors.New("not found")
		},
	})
}

// Without udisks2 the repair must go through the broker, never through a direct
// unprivileged exec that would simply fail with EACCES.
func TestWithoutUDisksRepairRoutesThroughTheBroker(t *testing.T) {
	elevated := &fakeElevated{available: true, status: StatusChanged, detail: "Clearing the dirty flag"}
	runner := &fakeRunner{}
	svc := nativeToolsService(t, &fakeObserver{state: unmountedDirtyState()}, runner, elevated)

	res, err := svc.Execute(context.Background(), Request{Action: ActionRepair, Device: elementsDevice(), AcknowledgeDataLoss: true})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if res.Backend != string(BackendNativeTools) {
		t.Fatalf("backend = %q, want %q", res.Backend, BackendNativeTools)
	}
	if len(elevated.calls) != 1 || elevated.calls[0] != ActionRepair {
		t.Fatalf("elevated calls = %v", elevated.calls)
	}
	if len(runner.argv) != 0 {
		t.Fatalf("privileged work must not run through the plain runner, ran %v", runner.argv)
	}
	if !res.Changed {
		t.Fatalf("result = %+v, want changed", res)
	}
}

// A mount through the broker would run inside a private mount namespace and
// silently not affect the host. Saying so beats appearing to succeed.
func TestWithoutUDisksMountIsUnsupportedButNamesTheOperatorCommand(t *testing.T) {
	state := unmountedDirtyState()
	state.Dirty = TristateNo
	elevated := &fakeElevated{available: true}
	svc := nativeToolsService(t, &fakeObserver{state: state}, &fakeRunner{}, elevated)

	res, err := svc.Execute(context.Background(), Request{
		Action: ActionMountReadWrite, Device: elementsDevice(), DesiredMountpoint: "/media/user/Elements",
	})

	var unsupported ErrUnsupported
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v, want ErrUnsupported", err)
	}
	if !strings.Contains(unsupported.OperatorCommand, "mount") {
		t.Fatalf("operator command = %q, want a mount command", unsupported.OperatorCommand)
	}
	if res.Status != StatusUnsupported {
		t.Fatalf("status = %q", res.Status)
	}
	if len(elevated.calls) != 0 {
		t.Fatal("no elevated call may be made for an action the broker cannot perform")
	}
}

// With neither udisks2 nor a broker, the operator is pointed at the one command
// that restores the capability.
func TestWithoutAnyPrivilegedPathTheOperatorIsPointedAtSetup(t *testing.T) {
	elevated := &fakeElevated{available: false}
	svc := nativeToolsService(t, &fakeObserver{state: unmountedDirtyState()}, &fakeRunner{}, elevated)

	_, err := svc.Execute(context.Background(), Request{Action: ActionRepair, Device: elementsDevice(), AcknowledgeDataLoss: true})

	var unsupported ErrUnsupported
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v, want ErrUnsupported", err)
	}
	if unsupported.OperatorCommand != "sudo vrooli setup" {
		t.Fatalf("operator command = %q, want the setup command", unsupported.OperatorCommand)
	}
}

// A dry run must not reach the broker either.
func TestElevatedDryRunDoesNotCallTheBroker(t *testing.T) {
	elevated := &fakeElevated{available: true, status: StatusChanged}
	svc := nativeToolsService(t, &fakeObserver{state: unmountedDirtyState()}, &fakeRunner{}, elevated)

	res, err := svc.Execute(context.Background(), Request{Action: ActionRepair, Device: elementsDevice(), AcknowledgeDataLoss: true, DryRun: true})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(elevated.calls) != 0 {
		t.Fatalf("dry run called the broker: %v", elevated.calls)
	}
	if len(res.Command) == 0 {
		t.Fatal("dry run must still report the command")
	}
}

// A check that ran and found an inconsistent filesystem is a successful action
// with a bad answer. Reporting only "verified" would read as "the volume is
// fine" — which is exactly backwards.
func TestCheckReportsTheConsistencyVerdictSeparatelyFromStatus(t *testing.T) {
	cases := []struct {
		name   string
		output string
		want   Tristate
	}{
		{name: "udisks reports inconsistent", output: "b false", want: TristateNo},
		{name: "udisks reports consistent", output: "b true", want: TristateYes},
		{name: "udisks output unrecognised", output: "something else", want: TristateUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := unmountedDirtyState()
			runner := &fakeRunner{out: []byte(tc.output)}
			svc := linuxService(t, &fakeObserver{state: state}, runner)

			res, err := svc.Execute(context.Background(), Request{Action: ActionCheck, Device: elementsDevice()})
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if res.Status != StatusVerified || res.Changed {
				t.Fatalf("a check must report verified without a change, got %+v", res)
			}
			if res.Consistent != tc.want {
				t.Fatalf("consistent = %q, want %q", res.Consistent, tc.want)
			}
		})
	}
}

// The tool-driven backends answer with exit status rather than a payload.
func TestCheckVerdictFromToolExitStatus(t *testing.T) {
	if got := checkVerdict(BackendNativeTools, nil, nil); got != TristateYes {
		t.Fatalf("clean exit = %q, want yes", got)
	}
	if got := checkVerdict(BackendNativeTools, nil, errors.New("exit status 4")); got != TristateNo {
		t.Fatalf("failing exit = %q, want no", got)
	}
	if got := checkVerdict(BackendDiskutil, nil, nil); got != TristateYes {
		t.Fatalf("diskutil clean = %q, want yes", got)
	}
}

// Only a check produces a consistency verdict; other actions must not imply one.
func TestNonCheckActionsCarryNoConsistencyVerdict(t *testing.T) {
	repaired := unmountedDirtyState()
	repaired.Dirty = TristateNo
	svc := linuxService(t, &fakeObserver{state: unmountedDirtyState(), after: &repaired}, &fakeRunner{})

	res, err := svc.Execute(context.Background(), Request{Action: ActionRepair, Device: elementsDevice(), AcknowledgeDataLoss: true})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.Consistent != "" {
		t.Fatalf("consistent = %q, want empty for a repair", res.Consistent)
	}
}
