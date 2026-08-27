package volumeremediation

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/vrooli/vrooli/internal/shell"
)

// Observer reports the current state of a device. It is a seam so the service's
// safety gates can be tested against exact host conditions.
type Observer interface {
	Observe(ctx context.Context, devicePath string) (State, error)
}

// Runner executes a fully-formed argv. Implementations receive an argv slice
// and never a command string, so no caller input can reach a shell.
type Runner interface {
	Run(ctx context.Context, argv []string) ([]byte, error)
}

// Elevated is the seam for actions that need root. The only implementation
// routes through the setup-installed privilege broker; nothing in this package
// spawns sudo, and no other elevation path is permitted.
type Elevated interface {
	Available() bool
	CheckFilesystem(ctx context.Context, device Device) (string, string, error)
	RepairFilesystem(ctx context.Context, device Device) (string, string, error)
}

// Service orchestrates volume remediation behind its safety gates.
type Service struct {
	observer Observer
	runner   Runner
	elevated Elevated
	goos     string
	// lookPath reports whether a tool is available, deciding which backend can
	// serve a request on this host.
	lookPath func(string) (string, error)
}

// Options configures a Service. Every field is optional; the zero value yields
// the production wiring for the current host.
type Options struct {
	Observer Observer
	Runner   Runner
	Elevated Elevated
	GOOS     string
	LookPath func(string) (string, error)
}

// New constructs a remediation service.
func New(opts Options) *Service {
	s := &Service{
		observer: opts.Observer,
		runner:   opts.Runner,
		elevated: opts.Elevated,
		goos:     opts.GOOS,
		lookPath: opts.LookPath,
	}
	if s.goos == "" {
		s.goos = runtime.GOOS
	}
	if s.lookPath == nil {
		s.lookPath = exec.LookPath
	}
	if s.observer == nil {
		s.observer = NewHostObserver(s.goos)
	}
	if s.runner == nil {
		s.runner = execRunner{}
	}
	if s.elevated == nil {
		s.elevated = NewBrokerElevated()
	}
	return s
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, argv []string) ([]byte, error) {
	if len(argv) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	return shell.NewCommandContext(ctx, argv[0], argv[1:]...).CombinedOutput()
}

// Backend reports which execution path this host offers, or an unsupported
// error naming the native command an operator can run instead.
func (s *Service) Backend() (Backend, error) {
	switch s.goos {
	case "linux":
		if s.has("udisksctl") && s.has("busctl") {
			return BackendUDisks, nil
		}
		if s.has("umount") && s.has("mount") {
			return BackendNativeTools, nil
		}
		return "", ErrUnsupported{Reason: "no udisks2 client and no mount tools on this Linux host"}
	case "darwin":
		if s.has("diskutil") {
			return BackendDiskutil, nil
		}
		return "", ErrUnsupported{Reason: "diskutil is unavailable"}
	case "windows":
		if s.has("powershell") {
			return BackendRepairVolume, nil
		}
		return "", ErrUnsupported{Reason: "powershell is unavailable", OperatorCommand: "Repair-Volume -DriveLetter <letter> -OfflineScanAndFix"}
	default:
		return "", ErrUnsupported{Reason: "no volume remediation adapter for " + s.goos}
	}
}

func (s *Service) has(tool string) bool {
	if s.lookPath == nil {
		return false
	}
	_, err := s.lookPath(tool)
	return err == nil
}

// Inspect observes a device without changing anything.
func (s *Service) Inspect(ctx context.Context, device Device) (State, error) {
	if err := validateDevicePath(device.Path); err != nil {
		return State{}, err
	}
	return s.observer.Observe(ctx, device.Path)
}

// Execute runs one remediation action behind every gate.
//
// The gates are applied against a *fresh* observation taken inside this call,
// never against state the caller supplied. A caller that observed a volume,
// paused, and then asked for a repair must not be able to act on a disk that
// has since been swapped, remounted, or replugged.
func (s *Service) Execute(ctx context.Context, req Request) (Result, error) {
	if err := validateRequest(req); err != nil {
		return Result{Action: req.Action, Device: req.Device, Status: StatusRefused}, err
	}
	backend, err := s.Backend()
	if err != nil {
		return Result{Action: req.Action, Device: req.Device, Status: StatusUnsupported}, err
	}

	observed, err := s.observer.Observe(ctx, req.Device.Path)
	if err != nil {
		return Result{Action: req.Action, Device: req.Device, Status: StatusFailed, Backend: string(backend)}, fmt.Errorf("observe device: %w", err)
	}
	if req.Action == ActionInspect {
		return Result{Action: req.Action, Device: observed.Device, Status: StatusVerified, Backend: string(backend), State: observed}, nil
	}
	if !req.Device.Matches(observed.Device) {
		return Result{Action: req.Action, Device: req.Device, Status: StatusRefused, Backend: string(backend), State: observed}, ErrDeviceChanged
	}
	if isSystemMountpoint(observed.Device.Mountpoint) {
		return Result{Action: req.Action, Device: observed.Device, Status: StatusRefused, Backend: string(backend), State: observed},
			ErrRefused{Reason: "refusing to act on a system volume mounted at " + observed.Device.Mountpoint}
	}

	if done, result := s.shortCircuit(req, observed, backend); done {
		return result, nil
	}
	if err := s.gateOnObservedState(req, observed); err != nil {
		return Result{Action: req.Action, Device: observed.Device, Status: StatusRefused, Backend: string(backend), State: observed}, err
	}

	if backend.NeedsElevation() {
		return s.executeElevated(ctx, req, observed, backend)
	}

	argv, err := commandFor(backend, req, observed)
	if err != nil {
		status := StatusUnsupported
		var refused ErrRefused
		if asRefused(err, &refused) {
			status = StatusRefused
		}
		return Result{Action: req.Action, Device: observed.Device, Status: status, Backend: string(backend)}, err
	}

	result := Result{
		Action:  req.Action,
		Device:  observed.Device,
		Backend: string(backend),
		Command: argv,
		DryRun:  req.DryRun,
		State:   observed,
	}
	if req.DryRun {
		result.Status = StatusVerified
		return result, nil
	}

	out, runErr := s.runner.Run(ctx, argv)
	result.Detail = boundedOutput(out)
	if runErr != nil {
		result.Status = StatusFailed
		return result, fmt.Errorf("%s %s: %w: %s", backend, req.Action, runErr, result.Detail)
	}
	result.Status = StatusChanged
	result.Changed = true
	if req.Action == ActionCheck {
		// A check proves state; it does not move it. Its verdict lives in
		// Consistent, not in Status: a check that ran successfully and found
		// an inconsistent filesystem still *ran* successfully, and reporting
		// only "verified" would read as "the volume is fine".
		result.Status = StatusVerified
		result.Changed = false
		result.Consistent = checkVerdict(backend, out, runErr)
	}
	// Re-observe so the result carries what the host looks like now rather than
	// what it looked like before the action.
	if after, obsErr := s.observer.Observe(ctx, req.Device.Path); obsErr == nil {
		result.State = after
	}
	return result, nil
}

// executeElevated serves the backend that has no unprivileged path. Only the
// block-device actions are available here: this host's elevation runs inside a
// service with mount-namespace isolation, so a mount performed through it would
// not propagate to the host. Rather than appear to succeed while changing
// nothing, the mount actions return the operator command instead.
func (s *Service) executeElevated(ctx context.Context, req Request, observed State, backend Backend) (Result, error) {
	result := Result{Action: req.Action, Device: observed.Device, Backend: string(backend), DryRun: req.DryRun, State: observed}

	switch req.Action {
	case ActionCheck, ActionRepair:
	default:
		return withStatus(result, StatusUnsupported), ErrUnsupported{
			Reason:          fmt.Sprintf("%s has no unprivileged path on this host and cannot be performed through the privilege broker", req.Action),
			OperatorCommand: operatorMountCommand(req, observed),
		}
	}

	if s.elevated == nil || !s.elevated.Available() {
		return withStatus(result, StatusUnsupported), ErrUnsupported{
			Reason:          "this host has no udisks2 client and the privilege broker is unavailable",
			OperatorCommand: "vrooli setup --sudo-mode=ask",
		}
	}

	argv, err := commandFor(backend, req, observed)
	if err != nil {
		return withStatus(result, StatusUnsupported), err
	}
	result.Command = argv
	if req.DryRun {
		return withStatus(result, StatusVerified), nil
	}

	action := s.elevated.CheckFilesystem
	if req.Action == ActionRepair {
		action = s.elevated.RepairFilesystem
	}
	status, detail, err := action(ctx, observed.Device)
	result.Detail = detail
	if err != nil {
		return withStatus(result, StatusFailed), err
	}
	result.Status = status
	result.Changed = status == StatusChanged
	if after, obsErr := s.observer.Observe(ctx, req.Device.Path); obsErr == nil {
		result.State = after
	}
	return result, nil
}

func withStatus(result Result, status string) Result {
	result.Status = status
	return result
}

// operatorMountCommand names the exact command an operator can run when this
// host offers no automated path, so an unsupported result is still actionable.
func operatorMountCommand(req Request, observed State) string {
	switch req.Action {
	case ActionUnmount:
		mountpoint := observed.Device.Mountpoint
		if mountpoint == "" {
			mountpoint = req.Device.Mountpoint
		}
		return "sudo umount " + mountpoint
	case ActionMountReadWrite:
		mountpoint := req.DesiredMountpoint
		if mountpoint == "" {
			mountpoint = observed.Device.Mountpoint
		}
		return fmt.Sprintf("sudo mount -t %s -o rw %s %s", strings.ToLower(req.Device.Filesystem), req.Device.Path, mountpoint)
	default:
		return ""
	}
}

// shortCircuit answers requests the host already satisfies, so a retried
// sequence is idempotent instead of failing on its second run.
func (s *Service) shortCircuit(req Request, observed State, backend Backend) (bool, Result) {
	base := Result{Action: req.Action, Device: observed.Device, Backend: string(backend), State: observed, DryRun: req.DryRun}
	switch req.Action {
	case ActionUnmount:
		if !observed.Mounted {
			base.Status = StatusAlreadySatisfied
			return true, base
		}
	case ActionMountReadWrite:
		if observed.Mounted && !observed.ReadOnly {
			base.Status = StatusAlreadySatisfied
			return true, base
		}
	}
	return false, Result{}
}

// gateOnObservedState applies the gates that depend on live host state.
func (s *Service) gateOnObservedState(req Request, observed State) error {
	switch req.Action {
	case ActionCheck, ActionRepair:
		if observed.Mounted {
			// Every filesystem checker in the supported set requires exclusive
			// access. Running one against a mounted volume risks corrupting the
			// very data the remediation exists to protect.
			return ErrRefused{Reason: fmt.Sprintf("%s requires the volume to be unmounted; it is mounted at %s", req.Action, observed.Device.Mountpoint)}
		}
	case ActionMountReadWrite:
		if observed.Mounted && observed.ReadOnly {
			// Remounting read/write on top of an existing read-only mount hides
			// whether the underlying cause was actually resolved.
			return ErrRefused{Reason: "volume is already mounted read-only; unmount it before mounting read/write"}
		}
		if observed.Dirty == TristateYes {
			return ErrRefused{Reason: "volume still reports a dirty filesystem; repair it before mounting read/write"}
		}
	}
	return nil
}

// checkVerdict reads a check's consistency answer from what the backend
// actually reported. udisks2 returns a D-Bus boolean payload ("b true" /
// "b false"); the tool-driven backends answer with their exit status. An
// unreadable answer stays unknown rather than defaulting to healthy.
func checkVerdict(backend Backend, out []byte, runErr error) Tristate {
	if backend == BackendUDisks {
		switch strings.TrimSpace(strings.ToLower(string(out))) {
		case "b true":
			return TristateYes
		case "b false":
			return TristateNo
		default:
			return TristateUnknown
		}
	}
	if runErr == nil {
		return TristateYes
	}
	return TristateNo
}

func asRefused(err error, target *ErrRefused) bool {
	if refused, ok := err.(ErrRefused); ok {
		*target = refused
		return true
	}
	return false
}

// boundedOutput trims tool output to a size safe to persist and log.
func boundedOutput(out []byte) string {
	const maxDetail = 2048
	text := strings.TrimSpace(string(out))
	if len(text) <= maxDetail {
		return text
	}
	return text[:maxDetail] + "… (truncated)"
}
