//go:build darwin

package platform

import (
	"errors"
	"fmt"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

// darwinProcessZombie is SZOMB from <sys/proc.h>: exited, not yet reaped.
const darwinProcessZombie = 5

// pidIsAlive treats a zombie as gone, as Linux does. kill(pid, 0) succeeds for
// an exited child its parent has not reaped, so without the state check a
// teardown that already killed a process would wait on it until its deadline.
func pidIsAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	if err := syscall.Kill(pid, 0); err != nil && !errors.Is(err, syscall.EPERM) {
		return false
	}
	info, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil || info == nil {
		// kill already proved the pid exists; an unreadable state is not death.
		return true
	}
	return info.Proc.P_stat != darwinProcessZombie
}

// readProcArgs reads the kern.procargs2 block. The kernel serves it for the
// caller's own processes (and to root); another user's process is an error.
func readProcArgs(pid int) (procArgs, error) {
	if pid <= 0 {
		return procArgs{}, fmt.Errorf("platform: invalid pid %d", pid)
	}
	before, err := darwinObservationStamp(pid)
	if err != nil {
		return procArgs{}, err
	}
	data, err := unix.SysctlRaw("kern.procargs2", pid)
	if err != nil {
		return procArgs{}, fmt.Errorf("platform: read process arguments for pid %d: %w", pid, err)
	}
	after, err := darwinObservationStamp(pid)
	if err != nil || before != after {
		return procArgs{}, ErrOperationIdentity
	}
	return parseProcArgs2(data)
}

func readProcessEnvironment(pid int) (map[string]string, error) {
	args, err := readProcArgs(pid)
	if err != nil {
		return nil, err
	}
	if args.environmentErr != nil {
		return nil, args.environmentErr
	}
	return args.Env, nil
}

func processExecutablePath(pid int) (string, error) {
	args, err := readProcArgs(pid)
	if err != nil {
		return "", err
	}
	return args.Executable, nil
}

func processName(pid int) (string, error) {
	path, err := processExecutablePath(pid)
	if err != nil {
		return "", err
	}
	return filepath.Base(path), nil
}

func processWorkingDir(pid int) (string, error) { return darwinWorkingDir(pid) }

func processHasChildren(pid int) (bool, error) {
	if pid <= 0 {
		return false, fmt.Errorf("platform: invalid pid %d", pid)
	}
	processes, err := unix.SysctlKinfoProcSlice("kern.proc.all")
	if err != nil {
		return false, err
	}
	for _, p := range processes {
		if int(p.Eproc.Ppid) == pid {
			return true, nil
		}
	}
	return false, nil
}
