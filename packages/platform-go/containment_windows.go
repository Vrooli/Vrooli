//go:build windows

package platform

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows contains a tree with a Job Object: kill-on-close for the tree
// boundary, a job-wide memory quota from MemoryMax, an active-process limit
// from TasksMax and a weight-based CPU rate from CPUWeight. There is no
// pause primitive that spans a job, so FreezeScope terminates it; the
// requirement module records that difference. Compile-verified only.

func containedCommand(spec ContainedSpec) (*Contained, error) {
	cmd := exec.Command(spec.Path, spec.Args...)
	cmd.Env = spec.Env
	cmd.Dir = spec.Dir
	cmd.Stdin = spec.Stdin
	cmd.Stdout = spec.Stdout
	cmd.Stderr = spec.Stderr
	cmd.SysProcAttr = detachedProcessAttrs()
	containment := spec.Containment
	return &Contained{Cmd: cmd, Method: MethodJob, after: func(c *Contained) error {
		job, err := assignJob(c.Cmd.Process, containment)
		if err != nil {
			return err
		}
		c.Scope = ScopeRef{Name: spec.Scope, Kind: ScopeKindJob, PID: c.Cmd.Process.Pid}
		c.cleanup = func() { _ = windows.CloseHandle(job) }
		return nil
	}}, nil
}

func containSelf(scope string, c Containment) (ScopeRef, string, error) {
	self, err := os.FindProcess(os.Getpid())
	if err != nil {
		return ScopeRef{Kind: ScopeKindNone}, MethodNone, err
	}
	if _, err := assignJob(self, c); err != nil {
		return ScopeRef{Kind: ScopeKindNone}, MethodNone, err
	}
	return ScopeRef{Name: scope, Kind: ScopeKindJob, PID: os.Getpid()}, MethodJob, nil
}

// assignJob creates a Job carrying the ceilings and assigns process to it.
func assignJob(process *os.Process, c Containment) (windows.Handle, error) {
	if process == nil || process.Pid <= 0 {
		return 0, errors.New("platform: invalid process for containment")
	}
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, err
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if bytes, err := memoryCeilingBytes(c.MemoryMax, physicalMemoryBytes()); err != nil {
		_ = windows.CloseHandle(job)
		return 0, err
	} else if bytes > 0 {
		info.BasicLimitInformation.LimitFlags |= windows.JOB_OBJECT_LIMIT_JOB_MEMORY
		info.JobMemoryLimit = uintptr(bytes)
	}
	if c.TasksMax > 0 {
		info.BasicLimitInformation.LimitFlags |= windows.JOB_OBJECT_LIMIT_ACTIVE_PROCESS
		info.BasicLimitInformation.ActiveProcessLimit = uint32(c.TasksMax)
	}
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		_ = windows.CloseHandle(job)
		return 0, err
	}
	if c.CPUWeight > 0 {
		rate := jobObjectCPURateControlInformation{
			ControlFlags: jobObjectCPURateControlEnable | jobObjectCPURateControlWeightBased,
			Value:        uint32(jobWeightForWeight(c.CPUWeight)),
		}
		if _, err := windows.SetInformationJobObject(job, jobObjectCPURateControlInformationClass, uintptr(unsafe.Pointer(&rate)), uint32(unsafe.Sizeof(rate))); err != nil {
			_ = windows.CloseHandle(job)
			return 0, err
		}
	}
	handle, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(process.Pid))
	if err != nil {
		_ = windows.CloseHandle(job)
		return 0, err
	}
	defer windows.CloseHandle(handle)
	if err := windows.AssignProcessToJobObject(job, handle); err != nil {
		_ = windows.CloseHandle(job)
		return 0, err
	}
	return job, nil
}

// JOBOBJECT_CPU_RATE_CONTROL_INFORMATION and its class and flags, from
// <winnt.h>; golang.org/x/sys/windows does not export them.
type jobObjectCPURateControlInformation struct {
	ControlFlags uint32
	Value        uint32
}

const (
	jobObjectCPURateControlInformationClass = 15
	jobObjectCPURateControlEnable           = 0x1
	jobObjectCPURateControlWeightBased      = 0x2
)

// Job CPU rate weights (1..9, 5 neutral).
const (
	jobWeightMin     = 1
	jobWeightLowered = 3
	jobWeightNeutral = 5
	jobWeightRaised  = 7
	jobWeightHigh    = 8
	jobWeightMax     = 9
)

// jobWeightForWeight maps CPUWeight onto the Job's 1..9 weight (5 neutral)
// through the shared priority table: priority 9 → 1, 7 → 5, 4 → 8, 2 → 9.
func jobWeightForWeight(weight int) int {
	switch windowsPriorityForWeight(weight) {
	case taskPriorityUrgent:
		return jobWeightMax
	case taskPriorityHigh:
		return jobWeightHigh
	case taskPriorityRaised:
		return jobWeightRaised
	case taskPriorityDefault:
		return jobWeightNeutral
	case taskPriorityLowered:
		return jobWeightLowered
	default:
		return jobWeightMin
	}
}

// memoryStatusEx is MEMORYSTATUSEX from <sysinfoapi.h>; x/sys/windows does
// not wrap GlobalMemoryStatusEx.
type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

var procGlobalMemoryStatusEx = windows.NewLazySystemDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx")

func physicalMemoryBytes() int64 {
	var status memoryStatusEx
	status.Length = uint32(unsafe.Sizeof(status))
	ok, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&status)))
	if ok == 0 {
		return 0
	}
	return int64(status.TotalPhys)
}

// freezeScope terminates the Job (see the package comment); thaw cannot
// bring a terminated tree back.
func freezeScope(ref ScopeRef) error {
	if ref.Kind != ScopeKindJob || ref.PID <= 0 {
		return fmt.Errorf("platform: %s is not a job scope", ref.String())
	}
	return signalPID(ref.PID, true)
}

func thawScope(ScopeRef) error { return ErrUnsupported }

func scopeFrozen(ref ScopeRef) (bool, error) {
	if ref.Kind != ScopeKindJob || ref.PID <= 0 {
		return false, fmt.Errorf("platform: %s is not a job scope", ref.String())
	}
	return !pidIsAlive(ref.PID), nil
}
