//go:build windows

package platform

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func detachedProcessAttrs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_PROCESS_GROUP}
}

func signalPID(pid int, _ bool) error {
	if pid <= 0 {
		return nil
	}
	handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	return windows.TerminateProcess(handle, 1)
}

func signalPIDWithSignal(pid int, signal os.Signal) error {
	if pid <= 0 {
		return nil
	}
	// Windows has no POSIX signal delivery. Interrupt is the portable request
	// for the backend's graceful console-control path; other graceful signals
	// retain the historical process-termination behavior.
	if signal == os.Interrupt {
		return gracefulStopProcess(&os.Process{Pid: pid})
	}
	return signalPID(pid, false)
}

func signalProcessGroup(groupID int, force bool) error { return signalPID(groupID, force) }

func reraiseSignal(signal os.Signal) error {
	if signal == nil {
		return nil
	}
	return signalPID(os.Getpid(), true)
}

func killProcess(pid int, force bool) error { return signalPID(pid, force) }

func replaceProcess(argv0 string, argv []string, env []string) error {
	args := []string(nil)
	if len(argv) > 1 {
		args = argv[1:]
	}
	cmd := exec.Command(argv0, args...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func assignProcessContainment(process *os.Process) (func(), error) {
	if process == nil || process.Pid <= 0 {
		return nil, errors.New("platform: invalid process for containment")
	}
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, err
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		_ = windows.CloseHandle(job)
		return nil, err
	}
	handle, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(process.Pid))
	if err != nil {
		_ = windows.CloseHandle(job)
		return nil, err
	}
	defer windows.CloseHandle(handle)
	if err := windows.AssignProcessToJobObject(job, handle); err != nil {
		_ = windows.CloseHandle(job)
		return nil, err
	}
	return func() { _ = windows.CloseHandle(job) }, nil
}

func gracefulStopProcess(process *os.Process) error {
	if process == nil {
		return nil
	}
	return windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, uint32(process.Pid))
}

func processGroupID(pid int) (int, error) { return pid, nil }

func terminationSignals() []os.Signal { return []os.Signal{os.Interrupt} }

func pidIsAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return errors.Is(err, windows.ERROR_ACCESS_DENIED)
	}
	defer windows.CloseHandle(handle)
	state, err := windows.WaitForSingleObject(handle, 0)
	return err == nil && state == uint32(windows.WAIT_TIMEOUT)
}

func processExecutablePath(pid int) (string, error) {
	h, err := openProcessForObservation(pid, windows.PROCESS_QUERY_LIMITED_INFORMATION)
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(h)
	buf := make([]uint16, 32768)
	n := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &n); err != nil {
		return "", err
	}
	if n == 0 {
		return "", fmt.Errorf("platform: empty process image path")
	}
	return windows.UTF16ToString(buf[:n]), nil
}

func processCommandLine(pid int) (string, error) {
	return readWindowsProcessString(pid, func(p *windows.RTL_USER_PROCESS_PARAMETERS) windows.NTUnicodeString { return p.CommandLine })
}

func processWorkingDir(pid int) (string, error) {
	return readWindowsProcessString(pid, func(p *windows.RTL_USER_PROCESS_PARAMETERS) windows.NTUnicodeString {
		return p.CurrentDirectory.DosPath
	})
}

func processHasChildren(pid int) (bool, error) {
	if pid <= 0 {
		return false, fmt.Errorf("platform: invalid pid %d", pid)
	}
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return false, err
	}
	defer windows.CloseHandle(snapshot)
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	if err := windows.Process32First(snapshot, &entry); err != nil {
		return false, err
	}
	for {
		if entry.ParentProcessID == uint32(pid) {
			return true, nil
		}
		if err := windows.Process32Next(snapshot, &entry); err != nil {
			if errors.Is(err, windows.ERROR_NO_MORE_FILES) {
				return false, nil
			}
			return false, err
		}
	}
}

// processScope reports empty: a Job Object has no stable name to record.
func processScope(int) (string, error) { return "", nil }

func processName(pid int) (string, error) {
	path, err := processExecutablePath(pid)
	if err != nil {
		return "", err
	}
	return filepath.Base(path), nil
}
