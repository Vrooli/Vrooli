//go:build linux

package platform

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	operationStopWait    = 5 * time.Second
	operationCommandWait = 5 * time.Second
	operationPoll        = 25 * time.Millisecond
)

var (
	operationStartWait = 2 * time.Second
	operationTermGrace = 7 * time.Second
)

var operationCgroupRoot = "/sys/fs/cgroup"

var errOperationCgroupAbsent = errors.New("platform: operation cgroup directory is absent")

type ownedUnitState struct {
	LoadState, ActiveState, SubState, ControlGroup string
	InvocationID, Result, Job, JobType, JobPath    string
	MainPID, ExecMainCode, ExecMainStatus          int
	ExitKnown, HasPendingJob                       bool
}

var (
	queryOwnedUnit       = queryOwnedUnitSystemd
	queryOperationsSlice = queryOperationsSliceSystemd
	signalOwnedUnit      = signalOwnedUnitSystemd
	cancelOwnedUnit      = cancelOwnedUnitSystemd
	releaseOwnedUnit     = releaseOwnedUnitSystemd
	resetFailedOwnedUnit = resetFailedOwnedUnitSystemd
)

func startOwnedOperation(ctx context.Context, intent OperationIntent, spec OperationSpec) (*OwnedOperation, error) {
	unit := operationUnitName(intent)
	prior, err := queryOwnedUnit(ctx, unit)
	if err != nil {
		return nil, fmt.Errorf("platform: inspect operation unit before start: %w", err)
	}
	if prior.LoadState != "not-found" {
		return nil, ErrOperationExists
	}

	cmd, err := ownedServiceCommand(intent, spec)
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("platform: submit owned operation: %w", err)
	}
	waitErr := waitLauncher(ctx, cmd)
	if waitErr != nil {
		// Dispatch may already have reached the manager. Leave the caller's
		// persisted intent intact so it can reconcile by deterministic unit.
		return nil, fmt.Errorf("platform: operation dispatch result is ambiguous; reconcile persisted intent: %w", waitErr)
	}
	state, err := awaitOperationStart(ctx, unit)
	if err != nil {
		return nil, fmt.Errorf("platform: operation dispatch accepted but start state is ambiguous; reconcile persisted intent: %w", err)
	}
	if state.LoadState != "loaded" || state.InvocationID == "" {
		return nil, fmt.Errorf("%w: manager did not retain operation unit and invocation identity", ErrOperationOutcomeLost)
	}
	if state.ControlGroup != "" && !validOperationServicePath(state.ControlGroup, unit) {
		return nil, fmt.Errorf("%w: operation service is outside the dedicated slice", ErrOperationIdentity)
	}
	ref, err := makeOperationRef(ctx, intent, unit, state)
	if err != nil {
		return nil, err
	}
	return &OwnedOperation{Ref: ref, launcherReaped: true}, nil
}

// systemd-run --no-block only acknowledges submission. Do not return a
// handle until the manager has acknowledged exec or retained a terminal
// outcome. In particular, an empty cgroup while activating is not proof that
// the service will not start moments later.
func awaitOperationStart(ctx context.Context, unit string) (ownedUnitState, error) {
	deadline := time.NewTimer(operationStartWait)
	defer deadline.Stop()
	ticker := time.NewTicker(operationPoll)
	defer ticker.Stop()
	for {
		state, err := queryOwnedUnit(ctx, unit)
		if err == nil {
			if state.LoadState == "loaded" && state.InvocationID != "" && !state.HasPendingJob {
				if state.ActiveState == "active" && state.SubState == "running" && state.ControlGroup != "" {
					return state, nil
				}
				if state.ExitKnown && terminalOperationState(state) {
					return state, nil
				}
			}
		} else if ctx.Err() != nil {
			return ownedUnitState{}, ctx.Err()
		}
		select {
		case <-ctx.Done():
			return ownedUnitState{}, ctx.Err()
		case <-deadline.C:
			return ownedUnitState{}, ErrOperationOutcomeLost
		case <-ticker.C:
		}
	}
}

func ownedServiceCommand(intent OperationIntent, spec OperationSpec) (*exec.Cmd, error) {
	path, err := exec.LookPath("systemd-run")
	if err != nil {
		return nil, fmt.Errorf("%w: systemd-run is required for pre-exec operation ownership", ErrUnsupported)
	}
	unit := operationUnitName(intent)
	args := []string{
		"--user", "--quiet", "--no-block", "--remain-after-exit", "--service-type=exec", "--expand-environment=no", "--unit=" + unit, "--slice=" + OperationsSlice,
		"--working-directory=" + spec.Dir,
		"--property=CPUWeight=" + strconv.Itoa(spec.Limits.CPUWeight),
		"--property=MemoryMax=" + strconv.FormatInt(spec.Limits.MemoryMaxBytes, 10),
		"--property=TasksMax=" + strconv.Itoa(spec.Limits.TasksMax),
		"--property=RuntimeMaxSec=" + spec.Limits.WallTime.String(),
		"--property=ExitType=cgroup", "--property=KillMode=mixed", "--property=TimeoutStopSec=10s", "--property=SendSIGKILL=yes",
		"--property=LogRateLimitIntervalSec=30s", "--property=LogRateLimitBurst=1000",
		"--property=StandardOutput=journal", "--property=StandardError=journal", "--",
	}
	args = append(args, "/usr/bin/env", "-i")
	args = append(args, spec.Env...)
	args = append(args, spec.Path)
	args = append(args, spec.Args...)
	cmd := exec.Command(path, args...)
	cmd.Env = userManagerEnv(nil)
	cmd.Dir = "/"
	cmd.WaitDelay = time.Second
	return cmd, nil
}

func waitLauncher(ctx context.Context, cmd *exec.Cmd) error {
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	timer := time.NewTimer(operationCommandWait)
	defer timer.Stop()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		<-done
		return ctx.Err()
	case <-timer.C:
		_ = cmd.Process.Kill()
		<-done
		return errors.New("systemd-run request exceeded bounded launcher wait")
	}
}

func lookupOwnedOperation(ctx context.Context, intent OperationIntent) (OperationRef, error) {
	unit := operationUnitName(intent)
	state, err := queryOwnedUnit(ctx, unit)
	if err != nil {
		return OperationRef{}, fmt.Errorf("platform: inspect operation intent: %w", err)
	}
	if state.LoadState == "not-found" {
		return OperationRef{}, ErrOperationOutcomeLost
	}
	if state.LoadState != "loaded" || state.InvocationID == "" ||
		(state.ControlGroup != "" && !validOperationServicePath(state.ControlGroup, unit)) ||
		(state.ControlGroup == "" && !((terminalOperationState(state) || state.HasPendingJob) && state.MainPID == 0)) {
		return OperationRef{}, ErrOperationOutcomeLost
	}
	return makeOperationRef(ctx, intent, unit, state)
}

func makeOperationRef(ctx context.Context, intent OperationIntent, unit string, state ownedUnitState) (OperationRef, error) {
	cgroup := state.ControlGroup
	if cgroup == "" {
		if !(terminalOperationState(state) || state.HasPendingJob) || state.MainPID != 0 {
			return OperationRef{}, fmt.Errorf("%w: manager omitted control group before acknowledged or pending state", ErrOperationOutcomeLost)
		}
		base, err := queryOperationsSlice(ctx)
		if err != nil {
			return OperationRef{}, fmt.Errorf("platform: resolve retained operation slice identity: %w", err)
		}
		cgroup = filepath.Join(base, unit)
	}
	if !validOperationServicePath(cgroup, unit) {
		return OperationRef{}, fmt.Errorf("%w: operation service is outside the dedicated slice", ErrOperationIdentity)
	}
	ref := OperationRef{Intent: intent, Unit: unit, Scope: ScopeRef{Name: strings.TrimSuffix(unit, ".service"), Kind: ScopeKindCgroup, Path: cgroup}, NativeInvocationID: state.InvocationID}
	if state.MainPID > 0 {
		stamp, alive, err := linuxProcessStartTime(state.MainPID)
		if err != nil {
			return OperationRef{}, fmt.Errorf("platform: identify operation main process: %w", err)
		}
		if alive {
			ref.RootPID, ref.RootStartTime = state.MainPID, stamp
			device, inode, err := linuxScopeInode(cgroup)
			if err != nil {
				return OperationRef{}, fmt.Errorf("platform: identify operation service cgroup: %w", err)
			}
			ref.ScopeDevice, ref.ScopeInode = device, inode
		}
	}
	return ref, nil
}

func inspectOwnedOperation(ctx context.Context, expected OperationRef) (OperationObservation, error) {
	state, err := queryOwnedUnit(ctx, expected.Unit)
	if err != nil {
		return OperationObservation{}, fmt.Errorf("platform: inspect owned operation: %w", err)
	}
	if state.LoadState == "not-found" {
		return OperationObservation{}, ErrOperationOutcomeLost
	}
	if err := validateOwnedUnitRef(expected, state); err != nil {
		return OperationObservation{}, err
	}
	populated := false
	cgroupAbsent := false
	populated, err = linuxScopePopulated(expected.Scope.Path)
	if errors.Is(err, errOperationCgroupAbsent) {
		cgroupAbsent = true
	} else if err != nil {
		return OperationObservation{}, fmt.Errorf("platform: read recursive operation state: %w", err)
	}
	rootAlive := false
	if expected.RootPID > 0 {
		stamp, alive, err := linuxProcessStartTime(expected.RootPID)
		if err != nil {
			return OperationObservation{}, fmt.Errorf("platform: inspect operation root: %w", err)
		}
		rootAlive = alive && stamp == expected.RootStartTime
	}
	stateName := OperationOutcomeUnknown
	if populated || (state.ActiveState == "active" && state.SubState == "running" && !state.HasPendingJob) {
		stateName = OperationRunning
	}
	if state.ExitKnown && state.MainPID == 0 && (cgroupAbsent || !populated) {
		stateName = OperationStoppedOutcomeKnown
	}
	return OperationObservation{
		Ref: expected, State: stateName, Populated: populated, RootAlive: rootAlive,
		Result: state.Result, ExecMainCode: state.ExecMainCode, ExecMainStatus: state.ExecMainStatus, ExitKnown: state.ExitKnown && state.MainPID == 0 && (cgroupAbsent || !populated),
	}, nil
}

func stopOwnedOperation(ctx context.Context, expected OperationRef) (OperationStopReceipt, error) {
	state, err := queryOwnedUnit(ctx, expected.Unit)
	if err != nil {
		return OperationStopReceipt{}, fmt.Errorf("platform: verify exact operation before stop: %w", err)
	}
	if state.LoadState == "not-found" {
		return OperationStopReceipt{}, ErrOperationOutcomeLost
	}
	if err := validateOwnedUnitRef(expected, state); err != nil {
		return OperationStopReceipt{}, err
	}
	if (state.ActiveState == "activating" || state.HasPendingJob) && state.MainPID == 0 && !state.ExitKnown {
		// A pending start has no process to signal yet. Cancel the exact unit
		// through the manager, then require terminal manager and cgroup evidence.
		if err := cancelOwnedUnit(ctx, expected.Unit); err != nil {
			return OperationStopReceipt{}, fmt.Errorf("platform: cancel pending operation start: %w", err)
		}
		if err := waitOperationTreeEmpty(ctx, expected, operationStopWait); err != nil {
			return OperationStopReceipt{}, err
		}
		return OperationStopReceipt{Ref: expected, NativeQuiescent: true, LauncherReap: LauncherReapUnknown}, nil
	}
	if empty, err := operationTreeEmpty(state, expected.Scope.Path); err != nil {
		return OperationStopReceipt{}, err
	} else if empty {
		return OperationStopReceipt{Ref: expected, NativeQuiescent: true, LauncherReap: LauncherReapUnknown}, nil
	}
	if err := signalOwnedUnit(ctx, expected.Unit, "SIGTERM"); err != nil {
		return OperationStopReceipt{}, fmt.Errorf("platform: request operation tree termination: %w", err)
	}
	if waitErr := waitOperationTreeEmpty(ctx, expected, operationTermGrace); waitErr == nil {
		return OperationStopReceipt{Ref: expected, NativeQuiescent: true, LauncherReap: LauncherReapUnknown}, nil
	} else if !errors.Is(waitErr, ErrOperationNotQuiescent) {
		return OperationStopReceipt{}, waitErr
	}
	if err := signalOwnedUnit(ctx, expected.Unit, "SIGKILL"); err != nil {
		return OperationStopReceipt{}, fmt.Errorf("platform: force operation tree termination: %w", err)
	}
	if err := waitOperationTreeEmpty(ctx, expected, operationStopWait); err != nil {
		return OperationStopReceipt{}, err
	}
	return OperationStopReceipt{Ref: expected, NativeQuiescent: true, LauncherReap: LauncherReapUnknown}, nil
}

func releaseOwnedOperation(ctx context.Context, expected OperationRef) error {
	state, err := queryOwnedUnit(ctx, expected.Unit)
	if err != nil {
		return fmt.Errorf("platform: verify operation before release: %w", err)
	}
	if state.LoadState == "not-found" {
		return ErrOperationOutcomeLost
	}
	if err := validateOwnedUnitRef(expected, state); err != nil {
		return err
	}
	if !state.ExitKnown {
		return fmt.Errorf("platform: cannot release operation before manager exit evidence is known")
	}
	empty, err := operationTreeEmpty(state, expected.Scope.Path)
	if err != nil {
		return err
	}
	if !empty {
		return ErrOperationNotQuiescent
	}
	if err := releaseOwnedUnit(ctx, expected.Unit); err != nil {
		return fmt.Errorf("platform: release retained operation unit: %w", err)
	}
	current, err := queryOwnedUnit(ctx, expected.Unit)
	if err != nil {
		return fmt.Errorf("platform: verify operation release: %w", err)
	}
	if current.LoadState == "not-found" {
		return nil
	}
	if err := validateOwnedUnitRef(expected, current); err != nil {
		return err
	}
	if empty, err := operationTreeEmpty(current, expected.Scope.Path); err != nil {
		return err
	} else if !empty {
		return ErrOperationNotQuiescent
	}
	if current.ActiveState == "failed" {
		if err := resetFailedOwnedUnit(ctx, expected.Unit); err != nil {
			return fmt.Errorf("platform: clear exact failed operation unit: %w", err)
		}
		current, err = queryOwnedUnit(ctx, expected.Unit)
		if err != nil {
			return fmt.Errorf("platform: verify cleared operation unit: %w", err)
		}
		if current.LoadState == "not-found" {
			return nil
		}
	}
	if current.InvocationID != expected.NativeInvocationID || current.ActiveState != "inactive" {
		return fmt.Errorf("%w: operation unit remains loaded or changed after release", ErrOperationIdentity)
	}
	return errors.New("platform: manager retained inactive operation unit after release")
}

func operationTreeEmpty(state ownedUnitState, path string) (bool, error) {
	if path == "" || state.MainPID != 0 || !terminalOperationState(state) {
		return false, nil
	}
	populated, err := linuxScopePopulated(path)
	if errors.Is(err, errOperationCgroupAbsent) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("platform: read recursive operation population: %w", err)
	}
	return !populated, nil
}

func terminalOperationState(state ownedUnitState) bool {
	if state.LoadState != "loaded" || state.HasPendingJob {
		return false
	}
	return (state.ActiveState == "active" && state.SubState == "exited") ||
		(state.ActiveState == "failed" && state.SubState == "failed") ||
		(state.ActiveState == "inactive" && (state.SubState == "dead" || state.SubState == "inactive"))
}

func waitOperationTreeEmpty(ctx context.Context, expected OperationRef, maximum time.Duration) error {
	deadline := time.NewTimer(maximum)
	defer deadline.Stop()
	ticker := time.NewTicker(operationPoll)
	defer ticker.Stop()
	for {
		state, err := queryOwnedUnit(ctx, expected.Unit)
		if err != nil {
			return fmt.Errorf("platform: verify operation after signal: %w", err)
		}
		if state.LoadState == "not-found" {
			return ErrOperationOutcomeLost
		}
		if err := validateOwnedUnitRef(expected, state); err != nil {
			return err
		}
		empty, err := operationTreeEmpty(state, expected.Scope.Path)
		if err != nil {
			return err
		}
		if empty {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return ErrOperationNotQuiescent
		case <-ticker.C:
		}
	}
}

func validateNativeOperationRef(ref OperationRef) error {
	if ref.Scope.Kind != ScopeKindCgroup || !validOperationServicePath(ref.Scope.Path, ref.Unit) {
		return fmt.Errorf("%w: invalid Linux operation service scope", ErrOperationIdentity)
	}
	if (ref.ScopeDevice == 0) != (ref.ScopeInode == 0) {
		return fmt.Errorf("%w: partial cgroup identity", ErrOperationIdentity)
	}
	return nil
}

func validateOwnedUnitRef(expected OperationRef, state ownedUnitState) error {
	if state.LoadState != "loaded" || state.InvocationID == "" || state.InvocationID != expected.NativeInvocationID {
		return fmt.Errorf("%w: native invocation changed or unavailable", ErrOperationIdentity)
	}
	if state.ControlGroup == "" && !((terminalOperationState(state) || state.HasPendingJob) && state.MainPID == 0) {
		return fmt.Errorf("%w: manager omitted control group before acknowledged or pending state", ErrOperationOutcomeLost)
	}
	if state.ControlGroup != "" && state.ControlGroup != expected.Scope.Path {
		return fmt.Errorf("%w: operation control group changed", ErrOperationIdentity)
	}
	if state.ControlGroup != "" && expected.ScopeInode != 0 {
		device, inode, err := linuxScopeInode(state.ControlGroup)
		if err == nil && (device != expected.ScopeDevice || inode != expected.ScopeInode) {
			return fmt.Errorf("%w: operation cgroup inode changed", ErrOperationIdentity)
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func validOperationServicePath(path, unit string) bool {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return false
	}
	for _, part := range strings.Split(path, string(filepath.Separator)) {
		if part == ".." {
			return false
		}
	}
	return strings.HasSuffix(path, "/"+OperationsSlice+"/"+unit)
}

func queryOwnedUnitSystemd(ctx context.Context, unit string) (ownedUnitState, error) {
	cmd := exec.CommandContext(ctx, "systemctl", "--user", "show", "--property=LoadState", "--property=ActiveState", "--property=SubState", "--property=ControlGroup", "--property=InvocationID", "--property=Result", "--property=MainPID", "--property=ExecMainCode", "--property=ExecMainStatus", "--property=Job", "--property=JobType", "--property=JobPath", "--", unit)
	cmd.Env = userManagerEnv(nil)
	cmd.WaitDelay = time.Second
	data, runErr := cmd.CombinedOutput()
	state, parseErr := parseOwnedUnitState(string(data))
	if parseErr != nil {
		return state, parseErr
	}
	if runErr != nil && !(state.LoadState == "not-found" && state.ActiveState == "inactive") {
		return state, fmt.Errorf("systemctl show %s: %w: %s", unit, runErr, strings.TrimSpace(string(data)))
	}
	return state, nil
}

func queryOperationsSliceSystemd(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "systemctl", "--user", "show", "--property=LoadState", "--property=ControlGroup", "--", OperationsSlice)
	cmd.Env = userManagerEnv(nil)
	cmd.WaitDelay = time.Second
	data, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("systemctl show operations slice: %w", err)
	}
	values := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if ok {
			values[key] = value
		}
	}
	base := values["ControlGroup"]
	if values["LoadState"] != "loaded" || !validOperationSlicePath(base) {
		return "", fmt.Errorf("%w: operations slice manager identity is missing or invalid", ErrOperationOutcomeLost)
	}
	return base, nil
}

func validOperationSlicePath(path string) bool {
	return filepath.IsAbs(path) && filepath.Clean(path) == path && strings.HasSuffix(path, "/"+OperationsSlice)
}

func parseOwnedUnitState(data string) (ownedUnitState, error) {
	state := ownedUnitState{}
	seen := map[string]bool{}
	for _, line := range strings.Split(data, "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		seen[key] = true
		switch key {
		case "LoadState":
			state.LoadState = value
		case "ActiveState":
			state.ActiveState = value
		case "SubState":
			state.SubState = value
		case "ControlGroup":
			state.ControlGroup = value
		case "InvocationID":
			state.InvocationID = value
		case "Result":
			state.Result = value
		case "Job":
			state.Job = value
		case "JobType":
			state.JobType = value
		case "JobPath":
			state.JobPath = value
		case "MainPID", "ExecMainCode", "ExecMainStatus":
			n, err := strconv.Atoi(value)
			if err != nil {
				return state, fmt.Errorf("systemctl returned malformed %s", key)
			}
			switch key {
			case "MainPID":
				state.MainPID = n
			case "ExecMainCode":
				state.ExecMainCode = n
			case "ExecMainStatus":
				state.ExecMainStatus = n
			}
		}
	}
	for _, key := range []string{"LoadState", "ActiveState", "SubState", "ControlGroup", "InvocationID", "Result", "MainPID", "ExecMainCode", "ExecMainStatus", "Job"} {
		if !seen[key] {
			return state, fmt.Errorf("systemctl manager omitted required operation field %s", key)
		}
	}
	if state.LoadState != "loaded" && state.LoadState != "not-found" {
		return state, fmt.Errorf("systemctl returned unexpected operation load state %q", state.LoadState)
	}
	if state.LoadState == "not-found" {
		return state, nil
	}
	if state.ActiveState == "" || (state.LoadState == "loaded" && state.InvocationID == "") {
		return state, errors.New("systemctl manager omitted required operation identity/state")
	}
	state.HasPendingJob = state.Job != "" && state.Job != "0"
	terminalSubstate := terminalOperationState(state)
	state.ExitKnown = terminalSubstate && state.MainPID == 0 && state.Result != "" && state.Result != "start-limit-hit"
	return state, nil
}

func signalOwnedUnitSystemd(ctx context.Context, unit, signal string) error {
	killWhom, err := killWhomForSignal(signal)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "systemctl", "--user", "kill", "--kill-whom="+killWhom, "--signal="+signal, "--", unit)
	cmd.Env = userManagerEnv(nil)
	cmd.WaitDelay = time.Second
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl kill %s: %w: %s", unit, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func killWhomForSignal(signal string) (string, error) {
	switch signal {
	case "SIGTERM":
		return "main", nil
	case "SIGKILL":
		return "all", nil
	default:
		return "", fmt.Errorf("platform: unsupported operation stop signal %q", signal)
	}
}

func releaseOwnedUnitSystemd(ctx context.Context, unit string) error {
	cmd := exec.CommandContext(ctx, "systemctl", "--user", "stop", "--", unit)
	cmd.Env = userManagerEnv(nil)
	cmd.WaitDelay = time.Second
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl stop %s: %w: %s", unit, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func resetFailedOwnedUnitSystemd(ctx context.Context, unit string) error {
	cmd := exec.CommandContext(ctx, "systemctl", "--user", "reset-failed", "--", unit)
	cmd.Env = userManagerEnv(nil)
	cmd.WaitDelay = time.Second
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl reset failed operation %s: %w: %s", unit, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func cancelOwnedUnitSystemd(ctx context.Context, unit string) error {
	cmd := exec.CommandContext(ctx, "systemctl", "--user", "stop", "--", unit)
	cmd.Env = userManagerEnv(nil)
	cmd.WaitDelay = time.Second
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl cancel pending operation %s: %w: %s", unit, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func linuxScopeInode(path string) (uint64, uint64, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return 0, 0, fmt.Errorf("invalid cgroup path %q", path)
	}
	info, err := os.Stat(filepath.Join(operationCgroupRoot, strings.TrimPrefix(path, "/")))
	if err != nil {
		return 0, 0, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Dev == 0 || stat.Ino == 0 {
		return 0, 0, errors.New("cgroup stat omitted native device/inode identity")
	}
	return uint64(stat.Dev), stat.Ino, nil
}

func linuxScopePopulated(path string) (bool, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return false, fmt.Errorf("invalid cgroup path %q", path)
	}
	directory := filepath.Join(operationCgroupRoot, strings.TrimPrefix(path, "/"))
	if _, err := os.Stat(directory); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, errOperationCgroupAbsent
		}
		return false, err
	}
	data, err := os.ReadFile(filepath.Join(directory, "cgroup.events"))
	if err != nil {
		return false, fmt.Errorf("read cgroup.events from existing operation cgroup: %w", err)
	}
	return parseCgroupPopulated(string(data))
}

func parseCgroupPopulated(data string) (bool, error) {
	for _, line := range strings.Split(data, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "populated" {
			switch fields[1] {
			case "0":
				return false, nil
			case "1":
				return true, nil
			default:
				return false, fmt.Errorf("invalid populated value %q", fields[1])
			}
		}
	}
	return false, errors.New("cgroup.events omitted populated field")
}

func linuxProcessStartTime(pid int) (string, bool, error) {
	if pid <= 0 {
		return "", false, errors.New("invalid process ID")
	}
	data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}
	closing := strings.LastIndex(string(data), ")")
	if closing < 0 || closing+1 >= len(data) {
		return "", false, errors.New("malformed process stat")
	}
	fields := strings.Fields(string(data[closing+1:]))
	if len(fields) < 20 || fields[0] == "Z" || fields[0] == "X" {
		return "", false, nil
	}
	return fields[19], true, nil
}
