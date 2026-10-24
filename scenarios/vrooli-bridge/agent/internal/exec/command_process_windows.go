//go:build windows

package exec

import (
	"fmt"
	"os/exec"
	"unsafe"

	"golang.org/x/sys/windows"
)

type processController struct{ job windows.Handle }

func prepareCommand(*exec.Cmd) (processController, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return processController{}, fmt.Errorf("create Windows process job: %w", err)
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	); err != nil {
		_ = windows.CloseHandle(job)
		return processController{}, fmt.Errorf("configure Windows process job: %w", err)
	}
	return processController{job: job}, nil
}

func (c processController) attach(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return fmt.Errorf("Windows process job cannot attach before process start")
	}
	process, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err != nil {
		return fmt.Errorf("open Windows child process for job: %w", err)
	}
	defer windows.CloseHandle(process)
	if err := windows.AssignProcessToJobObject(c.job, process); err != nil {
		return fmt.Errorf("attach Windows child process to job: %w", err)
	}
	return nil
}

func (c processController) close() {
	if c.job != 0 {
		_ = windows.CloseHandle(c.job)
	}
}

func (c processController) terminate(cmd *exec.Cmd) {
	if c.job != 0 {
		_ = windows.TerminateJobObject(c.job, 1)
	}
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
