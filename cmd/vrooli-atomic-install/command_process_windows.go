//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/vrooli/vrooli/internal/cliinstall"
	"golang.org/x/sys/windows"
)

const procThreadAttributeJobList = 0x0002000D

const managedCommandInterruptGrace = 5 * time.Second

func managedCommandSignals() []os.Signal { return []os.Signal{os.Interrupt} }

func waitManagedCommand(*exec.Cmd) error { return nil }

type managedJob struct {
	job     windows.Handle
	process windows.Handle
	pid     uint32
	mu      sync.Mutex
	stopBy  time.Time
	stopErr error
}

// Windows LockFileEx ownership stays with the helper process. The named job is
// the recovery witness that admission and pruning query if that process exits
// before its tree drains; the lock handle itself is never inherited.
func startManagedCommand(cmd *exec.Cmd, _ *os.File, leaseRoot, leaseSnapshot string) (func() error, func(os.Signal) error, func(), error) {
	if cmd == nil || cmd.Path == "" || len(cmd.Args) == 0 {
		return nil, nil, nil, fmt.Errorf("Windows snapshot command is invalid")
	}
	if !managedWindowsFile(cmd.Stdin) || !managedWindowsFile(cmd.Stdout) || !managedWindowsFile(cmd.Stderr) {
		return nil, nil, nil, fmt.Errorf("Windows snapshot command requires file-backed standard streams")
	}

	var jobName *uint16
	if leaseSnapshot != "" {
		name, err := cliinstall.SnapshotLeaseJobName(leaseRoot, leaseSnapshot)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("name Windows snapshot lease job: %w", err)
		}
		jobName, err = windows.UTF16PtrFromString(name)
		if err != nil {
			return nil, nil, nil, err
		}
	} else if leaseRoot != "" {
		name, err := cliinstall.SnapshotRootJobName(leaseRoot)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("name Windows snapshot root job: %w", err)
		}
		jobName, err = windows.UTF16PtrFromString(name)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	job, err := createSnapshotJobObject(jobName)
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return nil, nil, nil, fmt.Errorf("Windows snapshot lease job already exists")
	}
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create Windows snapshot command job: %w", err)
	}
	closeJob := true
	defer func() {
		if closeJob {
			_ = windows.CloseHandle(job)
		}
	}()
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		return nil, nil, nil, fmt.Errorf("configure Windows snapshot command job: %w", err)
	}

	var inherited []windows.Handle
	defer func() {
		for _, handle := range inherited {
			_ = windows.CloseHandle(handle)
		}
	}()
	duplicates := make(map[windows.Handle]windows.Handle)
	addInheritedDuplicate := func(handle windows.Handle) (windows.Handle, error) {
		if handle == 0 || handle == windows.InvalidHandle {
			return 0, nil
		}
		if duplicate, ok := duplicates[handle]; ok {
			return duplicate, nil
		}
		duplicate, err := duplicateWindowsHandleForInheritance(handle)
		if err != nil {
			return 0, err
		}
		duplicates[handle] = duplicate
		inherited = append(inherited, duplicate)
		return duplicate, nil
	}
	stdio := [3]windows.Handle{}
	files := [3]*os.File{}
	if f, ok := cmd.Stdin.(*os.File); ok {
		files[0] = f
	}
	if f, ok := cmd.Stdout.(*os.File); ok {
		files[1] = f
	}
	if f, ok := cmd.Stderr.(*os.File); ok {
		files[2] = f
	}
	for i, file := range files {
		if file == nil {
			continue // NULL standard handle, matching runQuietCommand semantics.
		}
		handle := windows.Handle(file.Fd())
		inheritedHandle, err := addInheritedDuplicate(handle)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("prepare inherited command stream: %w", err)
		}
		stdio[i] = inheritedHandle
	}

	attributes, err := windows.NewProcThreadAttributeList(2)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create Windows process attributes: %w", err)
	}
	defer attributes.Delete()
	jobHandles := []windows.Handle{job}
	if err := attributes.Update(procThreadAttributeJobList, unsafe.Pointer(&jobHandles[0]), unsafe.Sizeof(jobHandles[0])); err != nil {
		return nil, nil, nil, fmt.Errorf("attach snapshot command job at process creation: %w", err)
	}
	if len(inherited) > 0 {
		if err := attributes.Update(windows.PROC_THREAD_ATTRIBUTE_HANDLE_LIST, unsafe.Pointer(&inherited[0]), uintptr(len(inherited))*unsafe.Sizeof(inherited[0])); err != nil {
			return nil, nil, nil, fmt.Errorf("configure inherited command handles: %w", err)
		}
	}

	startup := windows.StartupInfoEx{StartupInfo: windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfoEx{})), Flags: windows.STARTF_USESTDHANDLES, StdInput: stdio[0], StdOutput: stdio[1], StdErr: stdio[2]}, ProcThreadAttributeList: attributes.List()}
	application, err := windows.UTF16PtrFromString(cmd.Path)
	if err != nil {
		return nil, nil, nil, err
	}
	args := make([]string, len(cmd.Args))
	for i, arg := range cmd.Args {
		args[i] = syscall.EscapeArg(arg)
	}
	commandLine, err := windows.UTF16PtrFromString(strings.Join(args, " "))
	if err != nil {
		return nil, nil, nil, err
	}
	var envBlock *uint16
	if env := cmd.Environ(); len(env) > 0 {
		envBlock, err = windows.UTF16PtrFromString(strings.Join(sortWindowsEnvironment(env), "\x00") + "\x00")
		if err != nil {
			return nil, nil, nil, err
		}
	}
	var directory *uint16
	if cmd.Dir != "" {
		directory, err = windows.UTF16PtrFromString(cmd.Dir)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	flags := uint32(windows.EXTENDED_STARTUPINFO_PRESENT | windows.CREATE_UNICODE_ENVIRONMENT | windows.CREATE_NEW_PROCESS_GROUP)
	var info windows.ProcessInformation
	if err := windows.CreateProcess(application, commandLine, nil, nil, true, flags, envBlock, directory, &startup.StartupInfo, &info); err != nil {
		return nil, nil, nil, fmt.Errorf("start Windows snapshot command in job: %w", err)
	}
	_ = windows.CloseHandle(info.Thread)
	closeJob = false
	managed := &managedJob{job: job, process: info.Process, pid: info.ProcessId}
	cleanup := func() {
		// Kill any residual descendants on every error path before dropping the
		// job handle. On success the job is already empty.
		_ = windows.TerminateJobObject(managed.job, 1)
		_ = windows.CloseHandle(managed.process)
		_ = windows.CloseHandle(managed.job)
	}
	wait := func() error { return waitWindowsManagedCommand(managed) }
	signal := func(os.Signal) error {
		managed.mu.Lock()
		if managed.stopBy.IsZero() {
			managed.stopBy = time.Now().Add(managedCommandInterruptGrace)
		}
		managed.mu.Unlock()
		err := windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, managed.pid)
		if err != nil {
			managed.mu.Lock()
			managed.stopBy = time.Now()
			managed.stopErr = fmt.Errorf("send CTRL_BREAK to Windows snapshot command: %w", err)
			managed.mu.Unlock()
		}
		return err
	}
	return wait, signal, cleanup, nil
}

func createSnapshotJobObject(name *uint16) (windows.Handle, error) {
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	setLastError := kernel32.NewProc("SetLastError")
	createJob := kernel32.NewProc("CreateJobObjectW")
	if err := setLastError.Find(); err != nil {
		return 0, err
	}
	if err := createJob.Find(); err != nil {
		return 0, err
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	_, _, _ = setLastError.Call(0)
	handle, _, callErr := createJob.Call(0, uintptr(unsafe.Pointer(name)))
	if handle == 0 {
		if callErr == windows.ERROR_SUCCESS {
			callErr = windows.ERROR_INVALID_HANDLE
		}
		return 0, callErr
	}
	job := windows.Handle(handle)
	if errors.Is(callErr, windows.ERROR_ALREADY_EXISTS) {
		_ = windows.CloseHandle(job)
		return 0, windows.ERROR_ALREADY_EXISTS
	}
	return job, nil
}

func waitWindowsManagedCommand(managed *managedJob) error {
	forced, terminateErr, queryErr := waitForWindowsJobDrain(func() (uint32, error) {
		var accounting jobAccountingInformation
		err := windows.QueryInformationJobObject(managed.job, windows.JobObjectBasicAccountingInformation, uintptr(unsafe.Pointer(&accounting)), uint32(unsafe.Sizeof(accounting)), nil)
		return accounting.ActiveProcesses, err
	}, managed.shutdownDeadline, func() error { return windows.TerminateJobObject(managed.job, 1) }, time.Sleep, time.Now)
	if queryErr != nil {
		return fmt.Errorf("query Windows snapshot command job: %w", queryErr)
	}
	if forced {
		// Forced termination is only requested after the bounded grace. The
		// helper waits for zero active members before allowing lock release.
		if terminateErr != nil {
			return fmt.Errorf("terminate Windows snapshot command job: %w", terminateErr)
		}
	}
	var exitCode uint32
	if err := windows.GetExitCodeProcess(managed.process, &exitCode); err != nil {
		return fmt.Errorf("read Windows snapshot command exit code: %w", err)
	}
	managed.mu.Lock()
	stopErr := managed.stopErr
	managed.mu.Unlock()
	if stopErr != nil {
		return stopErr
	}
	if exitCode != 0 {
		return managedCommandExitError(int(exitCode))
	}
	return nil
}

func duplicateWindowsHandleForInheritance(handle windows.Handle) (windows.Handle, error) {
	var duplicate windows.Handle
	if err := windows.DuplicateHandle(windows.CurrentProcess(), handle, windows.CurrentProcess(), &duplicate, 0, true, windows.DUPLICATE_SAME_ACCESS); err != nil {
		return 0, err
	}
	return duplicate, nil
}

func getWindowsHandleInformation(handle windows.Handle) (uint32, error) {
	var flags uint32
	proc := windows.NewLazySystemDLL("kernel32.dll").NewProc("GetHandleInformation")
	ok, _, callErr := proc.Call(uintptr(handle), uintptr(unsafe.Pointer(&flags)))
	if ok == 0 {
		if callErr == windows.ERROR_SUCCESS {
			callErr = windows.ERROR_INVALID_HANDLE
		}
		return 0, callErr
	}
	return flags, nil
}

func (managed *managedJob) shutdownDeadline() (time.Time, bool) {
	managed.mu.Lock()
	defer managed.mu.Unlock()
	return managed.stopBy, !managed.stopBy.IsZero()
}

type managedCommandExitError int

func (e managedCommandExitError) Error() string {
	return fmt.Sprintf("snapshot command exited with status %d", int(e))
}
func (e managedCommandExitError) ExitCode() int { return int(e) }

type jobAccountingInformation struct {
	TotalUserTime, TotalKernelTime                     int64
	ThisPeriodTotalUserTime, ThisPeriodTotalKernelTime int64
	TotalPageFaults, TotalProcesses, ActiveProcesses   uint32
	TotalTerminatedProcesses                           uint32
}

func waitForWindowsJobDrain(active func() (uint32, error), deadline func() (time.Time, bool), terminate func() error, sleep func(time.Duration), now func() time.Time) (bool, error, error) {
	// The interrupt path is bounded through TerminateJobObject. If querying the
	// Job or terminating it never makes progress, keep the lock owner alive
	// indefinitely: releasing the lock without proof that the tree is empty
	// would let another snapshot operation overlap a surviving command.
	forced := false
	var terminateErr error
	var queryErr error
	noDeadline := func() (time.Time, bool) { return time.Time{}, false }
	for {
		count, err := active()
		if err != nil {
			queryErr = err
			if !forced {
				forced = true
				terminateErr = terminate()
				deadline = noDeadline
			}
			sleep(10 * time.Millisecond)
			continue
		}
		if count == 0 {
			return forced, terminateErr, queryErr
		}
		if !forced && expired(deadline, now) {
			forced = true
			terminateErr = terminate()
			deadline = noDeadline
		}
		sleep(10 * time.Millisecond)
	}
}

func expired(deadline func() (time.Time, bool), now func() time.Time) bool {
	at, ok := deadline()
	return ok && !now().Before(at)
}

func managedWindowsFile(value any) bool {
	if value == nil {
		return true
	}
	_, ok := value.(*os.File)
	return ok
}

func sortWindowsEnvironment(env []string) []string {
	out := append([]string(nil), env...)
	sort.SliceStable(out, func(i, j int) bool {
		keyI, _, _ := strings.Cut(out[i], "=")
		keyJ, _, _ := strings.Cut(out[j], "=")
		return strings.ToUpper(keyI) < strings.ToUpper(keyJ)
	})
	return out
}
