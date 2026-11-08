//go:build linux

package platform

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

var queryServiceScopeUnit = queryServiceScopeUnitSystemd

func queryServiceScopeUnitSystemd(ctx context.Context, unit string) (ownedUnitState, error) {
	ctx, cancel := context.WithTimeout(ctx, operationCommandWait)
	defer cancel()
	cmd := exec.CommandContext(ctx, "systemctl", "--user", "show", "--property=LoadState", "--property=ActiveState", "--property=SubState", "--property=ControlGroup", "--property=InvocationID", "--property=Job", "--", unit)
	cmd.Env = userManagerEnv(nil)
	cmd.WaitDelay = time.Second
	data, runErr := cmd.CombinedOutput()
	if err := ctx.Err(); err != nil {
		return ownedUnitState{}, err
	}
	state, parseErr := parseOwnedNativeUnitState(string(data), true)
	if parseErr != nil {
		if runErr != nil {
			return state, fmt.Errorf("scope manager read failed: %w (invalid response: %v)", runErr, parseErr)
		}
		return state, parseErr
	}
	if runErr != nil && state.LoadState != "not-found" {
		return state, fmt.Errorf("scope manager read failed: %w", runErr)
	}
	return state, nil
}

type pinnedServiceScope struct{ dir *os.File }

func openPinnedServiceScope(scope ScopeRef) (*pinnedServiceScope, uint64, uint64, error) {
	fd, err := unix.Open(filepath.Join(operationCgroupRoot, strings.TrimPrefix(scope.Path, "/")), unix.O_RDONLY|unix.O_CLOEXEC|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, 0, 0, err
	}
	file := os.NewFile(uintptr(fd), scope.Path)
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		file.Close()
		return nil, 0, 0, err
	}
	var fs unix.Statfs_t
	if err := unix.Fstatfs(fd, &fs); err != nil {
		file.Close()
		return nil, 0, 0, err
	}
	if fs.Type != unix.CGROUP2_SUPER_MAGIC || stat.Dev == 0 || stat.Ino == 0 {
		file.Close()
		return nil, 0, 0, fmt.Errorf("%w: dedicated service scope is not a native cgroup v2 directory", ErrUnsupported)
	}
	return &pinnedServiceScope{dir: file}, uint64(stat.Dev), stat.Ino, nil
}

func (p *pinnedServiceScope) read(name string) ([]byte, error) {
	fd, err := unix.Openat(int(p.dir.Fd()), name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), name)
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 64<<10+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 64<<10 {
		return nil, fmt.Errorf("native service-scope observation exceeds its bound")
	}
	return data, nil
}

func (p *pinnedServiceScope) empty() (bool, error) {
	data, err := p.read("cgroup.events")
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, unix.ENODEV) {
		var stat unix.Stat_t
		if e := unix.Fstat(int(p.dir.Fd()), &stat); e != nil {
			return false, e
		}
		// Kernel removal of this exact held cgroup requires it to be empty.
		if stat.Nlink == 0 {
			return true, nil
		}
		// kernfs can retain a positive link count on a removed directory.
		// Inspect the kernel's link for this held FD, never the reused path.
		// Valid scope names cannot themselves end with this deleted suffix.
		link, linkErr := os.Readlink("/proc/self/fd/" + strconv.FormatUint(uint64(p.dir.Fd()), 10))
		if linkErr == nil && strings.HasSuffix(link, " (deleted)") {
			return true, nil
		}
	}
	if err != nil {
		return false, fmt.Errorf("pinned service-scope population unavailable: %w", err)
	}
	populated, err := parseCgroupPopulated(string(data))
	return !populated, err
}

func (p *pinnedServiceScope) kill() error {
	fd, err := unix.Openat(int(p.dir.Fd()), "cgroup.kill", unix.O_WRONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	n, err := unix.Write(fd, []byte("1"))
	if err == nil && n != 1 {
		err = io.ErrShortWrite
	}
	return err
}

func captureServiceScope(ctx context.Context, scope ScopeRef, pid int, startTime string) (ServiceScopeRef, error) {
	// Validate namespace before any manager or filesystem access.
	shape := ServiceScopeRef{Scope: scope, RootPID: pid, RootStartTime: startTime, NativeInvocationID: strings.Repeat("0", 32), ScopeDevice: 1, ScopeInode: 1}
	if err := shape.ValidateIdentity(); err != nil {
		return ServiceScopeRef{}, err
	}
	root, err := unix.PidfdOpen(pid, 0)
	if err != nil {
		return ServiceScopeRef{}, err
	}
	defer unix.Close(root)
	if err := verifyServiceScopeRoot(pid, startTime, scope.Path, root); err != nil {
		return ServiceScopeRef{}, err
	}
	state, err := queryServiceScopeUnit(ctx, filepath.Base(scope.Path))
	if err != nil {
		return ServiceScopeRef{}, err
	}
	if state.LoadState != "loaded" || state.ActiveState != "active" || state.ControlGroup != scope.Path || state.HasPendingJob || !operationInvocation.MatchString(state.InvocationID) {
		return ServiceScopeRef{}, fmt.Errorf("%w: service scope is not a stable native invocation", ErrOperationIdentity)
	}
	held, device, inode, err := openPinnedServiceScope(scope)
	if err != nil {
		return ServiceScopeRef{}, err
	}
	defer held.dir.Close()
	data, err := held.read("cgroup.procs")
	if err != nil {
		return ServiceScopeRef{}, err
	}
	if strings.TrimSpace(string(data)) != strconv.Itoa(pid) {
		return ServiceScopeRef{}, fmt.Errorf("%w: service scope is not an exclusive singleton root", ErrOperationIdentity)
	}
	children, err := held.dir.ReadDir(-1)
	if err != nil {
		return ServiceScopeRef{}, err
	}
	for _, entry := range children {
		if entry.IsDir() {
			return ServiceScopeRef{}, fmt.Errorf("%w: service scope has nested groups", ErrOperationIdentity)
		}
	}
	ref := ServiceScopeRef{Scope: scope, RootPID: pid, RootStartTime: startTime, NativeInvocationID: state.InvocationID, ScopeDevice: device, ScopeInode: inode}
	if err := verifyServiceScopeRoot(pid, startTime, scope.Path, root); err != nil {
		return ServiceScopeRef{}, err
	}
	if err := verifyServiceScopeManager(ctx, ref); err != nil {
		return ServiceScopeRef{}, err
	}
	return ref, ref.ValidateIdentity()
}

func verifyServiceScopeRoot(pid int, startTime, scope string, fd int) error {
	actual, alive, err := linuxProcessStartTime(pid)
	if err != nil {
		return err
	}
	actualScope, scopeErr := processScope(pid)
	if !alive || actual != startTime || scopeErr != nil || actualScope != scope {
		return fmt.Errorf("%w: dedicated service root changed", ErrOperationIdentity)
	}
	alive, err = serviceScopeRootAlive(fd)
	if err != nil {
		return err
	}
	if !alive {
		return fmt.Errorf("%w: dedicated service root exited during identity check", ErrOperationIdentity)
	}
	return nil
}

func serviceScopeRootAlive(fd int) (bool, error) {
	fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	for {
		_, err := unix.Poll(fds, 0)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			return false, err
		}
		if fds[0].Revents&(unix.POLLNVAL|unix.POLLERR) != 0 {
			return false, fmt.Errorf("invalid dedicated service root handle")
		}
		return fds[0].Revents&(unix.POLLIN|unix.POLLHUP) == 0, nil
	}
}

func verifyServiceScopeManager(ctx context.Context, ref ServiceScopeRef) error {
	state, err := queryServiceScopeUnit(ctx, filepath.Base(ref.Scope.Path))
	if err != nil {
		return err
	}
	if state.LoadState != "loaded" || state.InvocationID != ref.NativeInvocationID || state.ControlGroup != ref.Scope.Path || state.HasPendingJob {
		return fmt.Errorf("%w: dedicated service scope native invocation changed", ErrOperationIdentity)
	}
	return nil
}

func stopServiceScope(ctx context.Context, ref ServiceScopeRef) (ServiceScopeStopReceipt, error) {
	observed, err := inspectServiceScope(ctx, ref)
	if err != nil {
		return ServiceScopeStopReceipt{}, err
	}
	if !observed.RootAlive && !observed.TreePopulated {
		return ServiceScopeStopReceipt{Ref: ref, RootGone: true, TreeQuiescent: true}, nil
	}
	held, device, inode, err := openPinnedServiceScope(ref.Scope)
	if err != nil {
		return ServiceScopeStopReceipt{}, err
	}
	defer held.dir.Close()
	if device != ref.ScopeDevice || inode != ref.ScopeInode {
		return ServiceScopeStopReceipt{}, fmt.Errorf("%w: dedicated service cgroup changed", ErrOperationIdentity)
	}
	if err := verifyServiceScopeManager(ctx, ref); err != nil {
		return ServiceScopeStopReceipt{}, err
	}
	if !observed.RootAlive {
		if err := ctx.Err(); err != nil {
			return ServiceScopeStopReceipt{}, err
		}
		if err := held.kill(); err != nil {
			return ServiceScopeStopReceipt{}, err
		}
		if err := waitPinnedServiceScopeEmpty(ctx, held, -1, ref, operationStopWait); err != nil {
			return ServiceScopeStopReceipt{}, err
		}
		return ServiceScopeStopReceipt{Ref: ref, RootGone: true, TreeQuiescent: true}, nil
	}
	root, err := unix.PidfdOpen(ref.RootPID, 0)
	if err != nil {
		return ServiceScopeStopReceipt{}, fmt.Errorf("native service root handle unavailable; reconcile its exact outcome: %w", err)
	}
	defer unix.Close(root)
	if err := verifyServiceScopeRoot(ref.RootPID, ref.RootStartTime, ref.Scope.Path, root); err != nil {
		return ServiceScopeStopReceipt{}, err
	}
	if err := ctx.Err(); err != nil {
		return ServiceScopeStopReceipt{}, err
	}
	if err := unix.PidfdSendSignal(root, unix.SIGTERM, nil, 0); err != nil && !errors.Is(err, unix.ESRCH) {
		return ServiceScopeStopReceipt{}, err
	}
	if err := waitPinnedServiceScopeEmpty(ctx, held, root, ref, operationTermGrace); err == nil {
		return ServiceScopeStopReceipt{Ref: ref, RootGone: true, TreeQuiescent: true}, nil
	} else if !errors.Is(err, ErrOperationNotQuiescent) {
		return ServiceScopeStopReceipt{}, err
	}
	if err := ctx.Err(); err != nil {
		return ServiceScopeStopReceipt{}, err
	}
	if err := held.kill(); err != nil {
		return ServiceScopeStopReceipt{}, err
	}
	if err := waitPinnedServiceScopeEmpty(ctx, held, root, ref, operationStopWait); err != nil {
		return ServiceScopeStopReceipt{}, err
	}
	return ServiceScopeStopReceipt{Ref: ref, RootGone: true, TreeQuiescent: true}, nil
}

func waitPinnedServiceScopeEmpty(ctx context.Context, held *pinnedServiceScope, root int, ref ServiceScopeRef, maximum time.Duration) error {
	deadline := time.NewTimer(maximum)
	defer deadline.Stop()
	ticker := time.NewTicker(operationPoll)
	defer ticker.Stop()
	for {
		empty, err := held.empty()
		if err != nil {
			return err
		}
		var alive bool
		if root >= 0 {
			alive, err = serviceScopeRootAlive(root)
		} else {
			alive, err = serviceScopeRootMatches(ref)
		}
		if err != nil {
			return err
		}
		if empty && !alive {
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

func serviceScopeRootMatches(ref ServiceScopeRef) (bool, error) {
	stamp, alive, err := linuxProcessStartTime(ref.RootPID)
	if err != nil {
		return false, err
	}
	if !alive {
		return false, nil
	}
	scope, err := processScope(ref.RootPID)
	if err != nil {
		return false, err
	}
	if stamp != ref.RootStartTime {
		if scope == ref.Scope.Path {
			return false, fmt.Errorf("%w: live service-scope root disagrees with its recorded incarnation", ErrOperationIdentity)
		}
		return false, nil
	}
	if scope != ref.Scope.Path {
		return false, fmt.Errorf("%w: recorded service root moved outside its scope", ErrOperationIdentity)
	}
	return true, nil
}

func inspectServiceScope(ctx context.Context, ref ServiceScopeRef) (ServiceScopeObservation, error) {
	state, err := queryServiceScopeUnit(ctx, filepath.Base(ref.Scope.Path))
	if err != nil {
		return ServiceScopeObservation{}, err
	}
	rootAlive, err := serviceScopeRootMatches(ref)
	if err != nil {
		return ServiceScopeObservation{}, err
	}
	held, device, inode, err := openPinnedServiceScope(ref.Scope)
	if errors.Is(err, os.ErrNotExist) && state.LoadState == "not-found" && !rootAlive {
		return ServiceScopeObservation{Ref: ref, RootAlive: false, TreePopulated: false, NativePresent: false}, nil
	}
	if err != nil {
		return ServiceScopeObservation{}, err
	}
	defer held.dir.Close()
	if device != ref.ScopeDevice || inode != ref.ScopeInode {
		return ServiceScopeObservation{}, fmt.Errorf("%w: dedicated service cgroup changed", ErrOperationIdentity)
	}
	if state.LoadState != "loaded" || state.InvocationID != ref.NativeInvocationID || state.ControlGroup != ref.Scope.Path || state.HasPendingJob {
		return ServiceScopeObservation{}, fmt.Errorf("%w: dedicated native service scope changed or is ambiguous", ErrOperationIdentity)
	}
	empty, err := held.empty()
	if err != nil {
		return ServiceScopeObservation{}, err
	}
	finalRoot, err := serviceScopeRootMatches(ref)
	if err != nil {
		return ServiceScopeObservation{}, err
	}
	if finalRoot != rootAlive {
		return ServiceScopeObservation{}, ErrOperationOutcomeLost
	}
	if err := ctx.Err(); err != nil {
		return ServiceScopeObservation{}, err
	}
	return ServiceScopeObservation{Ref: ref, RootAlive: rootAlive, TreePopulated: !empty, NativePresent: true}, nil
}
