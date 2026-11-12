//go:build windows

package platform

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Qualification only. The native owner must supply an already provisioned,
// approved immutable host + isolated valid maintained capsule, whose SDK stays
// alive. This test does not provision a host, grant batch logon, create an
// account, or bypass the product launch guard. It can start the explicitly
// prepared capsule, so it is never part of an ordinary hermetic package run.
//
// Run separately for mode=task-stop and mode=custody-loss with FRESH capsules:
// VROOLI_WINDOWS_OWNED_LIVE=1
// VROOLI_WINDOWS_OWNED_LIVE_PAYLOAD=<JSON windowsOwnedPayload file>
// VROOLI_WINDOWS_OWNED_LIVE_MODE=task-stop|custody-loss
// go test -count=1 -run '^TestWindowsOwnedTaskNativeOuterJobLifetime$' .
func TestWindowsOwnedTaskNativeOuterJobLifetime(t *testing.T) {
	if os.Getenv("VROOLI_WINDOWS_OWNED_LIVE") != "1" {
		t.Skip("explicit native qualification only")
	}
	payloadPath := os.Getenv("VROOLI_WINDOWS_OWNED_LIVE_PAYLOAD")
	mode := os.Getenv("VROOLI_WINDOWS_OWNED_LIVE_MODE")
	if !filepath.IsAbs(payloadPath) || (mode != "task-stop" && mode != "custody-loss") {
		t.Fatal("absolute prepared payload and explicit mode required")
	}
	raw, e := os.ReadFile(payloadPath)
	if e != nil {
		t.Fatal(e)
	}
	var p windowsOwnedPayload
	if e = json.Unmarshal(raw, &p); e != nil {
		t.Fatal(e)
	}
	if e = validateWindowsPayload(p); e != nil {
		t.Fatal(e)
	}
	if e = validateWindowsMaintainedLaunchShape(p.Intent, p.Spec); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cleanupRef := windowsRef(p)
	defer func() {
		cleanup, done := context.WithTimeout(context.Background(), 30*time.Second)
		defer done()
		// A failed uncertain Start still has one durable intent. Retire that exact
		// slot through the real owner, never launch a replacement or use a PID guess.
		if ref, err := LookupOwnedOperation(cleanup, p.Intent); err == nil {
			cleanupRef = ref
		}
		if _, err := StopOwnedOperation(cleanup, cleanupRef); err != nil && !errors.Is(err, ErrOperationNotFound) {
			t.Errorf("native cleanup stop: %v", err)
			return
		}
		if err := ReleaseOwnedOperation(cleanup, cleanupRef); err != nil && !errors.Is(err, ErrOperationNotFound) {
			t.Errorf("native cleanup release: %v", err)
		}
	}()
	outer, e := windows.CreateJobObject(nil, nil)
	if e != nil {
		t.Fatal(e)
	}
	outerOpen := true
	defer func() {
		if outerOpen {
			windows.CloseHandle(outer)
		}
	}()
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, e = windows.SetInformationJobObject(outer, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); e != nil {
		t.Fatal(e)
	}
	attrs, e := windows.NewProcThreadAttributeList(1)
	if e != nil {
		t.Fatal(e)
	}
	defer attrs.Delete()
	if e = attrs.Update(0x0002000d, unsafe.Pointer(&outer), unsafe.Sizeof(outer)); e != nil {
		t.Fatal(e)
	}
	exe, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	app, _ := windows.UTF16PtrFromString(exe)
	command, _ := windows.UTF16PtrFromString(syscall.EscapeArg(exe) + " -test.run=^TestWindowsOwnedTaskNativeSubmitter$ -test.v")
	resultPath := filepath.Join(t.TempDir(), "submitted.json")
	environment := append(os.Environ(), "VROOLI_WINDOWS_OWNED_HELPER_PAYLOAD="+payloadPath, "VROOLI_WINDOWS_OWNED_HELPER_RESULT="+resultPath)
	var env []uint16
	for _, entry := range environment {
		value, err := windows.UTF16FromString(entry)
		if err != nil {
			t.Fatal(err)
		}
		env = append(env, value...)
	}
	env = append(env, 0)
	startup := windows.StartupInfoEx{StartupInfo: windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfoEx{}))}, ProcThreadAttributeList: attrs.List()}
	var child windows.ProcessInformation
	if e = windows.CreateProcess(app, command, nil, nil, false, windows.CREATE_UNICODE_ENVIRONMENT|windows.EXTENDED_STARTUPINFO_PRESENT|windows.CREATE_NO_WINDOW, &env[0], nil, &startup.StartupInfo, &child); e != nil {
		t.Fatal(e)
	}
	defer windows.CloseHandle(child.Process)
	defer windows.CloseHandle(child.Thread)
	var result windowsNativeSubmitterResult
	for {
		raw, e = os.ReadFile(resultPath)
		if e == nil {
			if e = json.Unmarshal(raw, &result); e != nil {
				t.Fatal(e)
			}
			break
		}
		if !os.IsNotExist(e) {
			t.Fatal(e)
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(50 * time.Millisecond):
		}
	}
	if result.Error != "" {
		t.Fatal(result.Error)
	}
	cleanupRef = result.Ref
	// Require a descendant in the exact role Job before testing termination;
	// a wrapper-only observation cannot qualify SDK lifetime.
	for {
		job, err := openWindowsOwnedJob(windowsOwnedJob(p.Intent))
		if err != nil {
			t.Fatal(err)
		}
		var accounting windowsJobAccounting
		if job != 0 {
			err = windows.QueryInformationJobObject(job, 1, uintptr(unsafe.Pointer(&accounting)), uint32(unsafe.Sizeof(accounting)), nil)
			windows.CloseHandle(job)
		}
		if err != nil {
			t.Fatal(err)
		}
		if accounting.ActiveProcesses >= 2 {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("no role SDK descendant before lifetime check")
		case <-time.After(50 * time.Millisecond):
		}
	}
	// The helper deliberately stays alive after receiving custody ACK. Closing
	// its kill-on-close outer Job must kill that caller and preserve the role.
	if state, err := windows.WaitForSingleObject(child.Process, 0); err != nil || state != uint32(windows.WAIT_TIMEOUT) {
		t.Fatal("submitter exited before outer Job close", state, err)
	}
	if e = windows.CloseHandle(outer); e != nil {
		t.Fatal(e)
	}
	outerOpen = false
	if state, err := windows.WaitForSingleObject(child.Process, 10000); err != nil || state != uint32(windows.WAIT_OBJECT_0) {
		t.Fatal("caller survived outer Job close", state, err)
	}
	observation, e := InspectOwnedOperation(ctx, result.Ref)
	if e != nil || !observation.RootAlive || !observation.Populated || observation.State != OperationRunning {
		t.Fatal("manager-owned role did not survive caller Job close", observation, e)
	}
	if mode == "task-stop" {
		if e = windowsStopRegisteredTask(windowsOwnedTaskName(p.Intent)); e != nil {
			t.Fatal(e)
		}
	} else {
		process, err := windows.OpenProcess(windows.PROCESS_TERMINATE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(result.Ref.RootPID))
		if err != nil {
			t.Fatal(err)
		}
		var birth, exit, kernel, user windows.Filetime
		err = windows.GetProcessTimes(process, &birth, &exit, &kernel, &user)
		if err == nil && strconv.FormatUint(uint64(birth.HighDateTime)<<32|uint64(birth.LowDateTime), 10) != result.Ref.RootStartTime {
			err = ErrOperationIdentity
		}
		if err == nil {
			err = windows.TerminateProcess(process, 1)
		}
		windows.CloseHandle(process)
		if err != nil {
			t.Fatal(err)
		}
	}
	for {
		populated, err := windowsOwnedJobPopulated(windowsOwnedJob(p.Intent))
		if err != nil {
			t.Fatal(err)
		}
		stamp, alive, err := windowsOwnedProcessStart(result.Ref.RootPID)
		if err != nil {
			t.Fatal(err)
		}
		if !populated && (!alive || stamp != result.Ref.RootStartTime) {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("task/custody loss retained wrapper or SDK", ctx.Err())
		case <-time.After(50 * time.Millisecond):
		}
	}
}

type windowsNativeSubmitterResult struct {
	Ref   OperationRef
	Error string
}

func TestWindowsOwnedTaskNativeSubmitter(t *testing.T) {
	path := os.Getenv("VROOLI_WINDOWS_OWNED_HELPER_PAYLOAD")
	resultPath := os.Getenv("VROOLI_WINDOWS_OWNED_HELPER_RESULT")
	if os.Getenv("VROOLI_WINDOWS_OWNED_LIVE") != "1" || path == "" || resultPath == "" {
		t.Skip("native qualification helper only")
	}
	raw, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	var p windowsOwnedPayload
	if e = json.Unmarshal(raw, &p); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	result := windowsNativeSubmitterResult{}
	op, e := StartOwnedOperation(ctx, p.Intent, p.Spec)
	if e != nil {
		result.Error = e.Error()
	} else {
		result.Ref = op.Ref
	}
	raw, e = json.Marshal(result)
	if e != nil {
		t.Fatal(e)
	}
	temp := resultPath + ".tmp"
	if e = os.WriteFile(temp, raw, 0600); e != nil {
		t.Fatal(e)
	}
	if e = os.Rename(temp, resultPath); e != nil {
		t.Fatal(e)
	}
	// Parent closes the fixture outer Job. Never exit early and mistake normal
	// helper completion for kill-on-close proof.
	<-ctx.Done()
	t.Fatal("parent did not close helper outer Job")
}

func TestWindowsSubmissionContextRefusedBeforeAnyBootstrapEffect(t *testing.T) {
	if e := PrepareWindowsOwnedServiceSubmission(nil, OperationIntent{}); !errors.Is(e, ErrOperationIdentity) {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if e := PrepareWindowsOwnedServiceSubmission(ctx, OperationIntent{}); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
}
