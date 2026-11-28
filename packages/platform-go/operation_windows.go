//go:build windows

package platform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func startOwnedOperation(ctx context.Context, i OperationIntent, s OperationSpec) (*OwnedOperation, error) {
	if err := validateWindowsMaintainedLaunchShape(i, s); err != nil {
		return nil, err
	}
	unlock, err := lockWindowsOwnedSubmission(ctx, i)
	if err != nil {
		return nil, err
	}
	locked := true
	defer func() {
		if locked {
			unlock()
		}
	}()
	if _, err = resolveOwnedServiceAccount(ctx, s.NativeAccount); err != nil {
		return nil, err
	}
	if _, err = windowsProcessOperatorSecurityProof(windows.CurrentProcess(), s.NativeAccount); err != nil {
		return nil, err
	}
	boot, err := windowsOwnedBootID()
	if err != nil {
		return nil, err
	}
	p := windowsOwnedPayload{Version: 2, Intent: i, Spec: s, BootID: boot}
	if err = validateWindowsPayload(p); err != nil {
		return nil, err
	}
	if prior, e := readWindowsOperatorPayload(i); e == nil {
		return nil, refuseWindowsOwnedReplay(p, prior)
	} else if !errors.Is(e, ErrOperationNotFound) {
		return nil, e
	}
	image, err := openWindowsOwnedServiceSpecHost(ctx, s)
	if err != nil {
		return nil, err
	}
	defer image.Close()
	if err = submitWindowsOwnedTask(p, windowsOwnedTaskSubmissionEffects{
		Reserve: prepareWindowsOperatorPayload, Register: registerWindowsOwnedTask, MarkIssued: setWindowsOwnedRunIssued,
		Run: func(name string) error { return windowsTaskControl(name, "start") },
	}); err != nil {
		return nil, err
	}
	unlock()
	locked = false
	settle, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for {
		root, e := readWindowsOwnedRecord(i)
		if e != nil {
			return nil, e
		}
		custody, e := readWindowsOwnedCustody(i)
		if e != nil {
			return nil, e
		}
		if custody.matches(p, root) {
			if e = verifyWindowsOwnedCustody(p, root); e != nil {
				return nil, e
			}
			ref := windowsRef(p)
			ref.RootPID = root.PID
			ref.RootStartTime = root.Start
			ref.Native.RootIncarnation = root.Incarnation
			return &OwnedOperation{Ref: ref}, nil
		}
		select {
		case <-settle.Done():
			return nil, settle.Err()
		case <-time.After(operationWaitPoll):
		}
	}
}

func lookupOwnedOperation(ctx context.Context, i OperationIntent) (OperationRef, error) {
	if err := ctx.Err(); err != nil {
		return OperationRef{}, err
	}
	p, err := readWindowsOperatorPayload(i)
	if err != nil {
		return OperationRef{}, err
	}
	r := windowsRef(p)
	root, err := readWindowsOwnedRecord(i)
	if err != nil {
		return OperationRef{}, err
	}
	custody, e := readWindowsOwnedCustody(i)
	if e != nil {
		return OperationRef{}, e
	}
	if _, e = windowsOwnedSubmissionMetadata(p, root, custody); e != nil {
		return OperationRef{}, e
	}
	r.RootPID = root.PID
	r.RootStartTime = root.Start
	r.Native.RootIncarnation = root.Incarnation
	return r, nil
}
func validateNativeOperationRef(r OperationRef) error {
	if r.Native == nil {
		return ErrOperationIdentity
	}
	expected := windowsRef(windowsOwnedPayload{Intent: r.Intent, BootID: r.Native.BootID, Spec: OperationSpec{ObserverPrincipal: r.Native.ObserverPrincipal}})
	if !windowsSameNative(expected, r) || r.Scope.Kind != ScopeKindNative || r.Scope.Path != expected.Scope.Path || r.Scope.PID != 0 || r.NativeInvocationID != r.Intent.InvocationID {
		return ErrOperationIdentity
	}
	return nil
}
func inspectOwnedOperation(ctx context.Context, r OperationRef) (OperationObservation, error) {
	actual, err := lookupOwnedOperation(ctx, r.Intent)
	if err != nil {
		return OperationObservation{}, err
	}
	if !windowsSameNative(actual, r) || (r.RootPID > 0 && !actual.Equal(r)) {
		return OperationObservation{}, ErrOperationIdentity
	}
	p, err := readWindowsOperatorPayload(r.Intent)
	if err != nil {
		return OperationObservation{}, err
	}
	populated, err := windowsOwnedJobPopulated(windowsOwnedJob(r.Intent))
	if err != nil {
		return OperationObservation{}, err
	}
	o := OperationObservation{Ref: actual, Populated: populated, State: OperationOutcomeUnknown}
	boot, err := windowsOwnedBootID()
	if err != nil {
		return o, err
	}
	if boot == p.BootID && actual.RootPID > 0 {
		stamp, alive, e := windowsOwnedProcessStart(actual.RootPID)
		if e != nil {
			return o, e
		}
		o.RootAlive = alive && stamp == actual.RootStartTime
	}
	if populated {
		root, e := readWindowsOwnedRecord(r.Intent)
		if e != nil {
			return o, e
		}
		custody, e := readWindowsOwnedCustody(r.Intent)
		if e != nil {
			return o, e
		}
		state, e := windowsOwnedSubmissionMetadata(p, root, custody)
		if e != nil {
			return o, e
		}
		if state != windowsSubmissionClaimed {
			return o, nil
		}
		if e = verifyWindowsOwnedCustody(p, root); e != nil {
			return o, e
		}
		o.State = OperationRunning
		return o, nil
	}
	terminal, err := readWindowsOwnedOutcome(r.Intent)
	if err != nil {
		return o, err
	}
	root, err := readWindowsOwnedRecord(r.Intent)
	if err != nil {
		return o, err
	}
	if terminal.Version != 0 && !terminal.matches(p, root) {
		return o, ErrOperationIdentity
	}
	if terminal.matches(p, root) {
		o.State = OperationStoppedOutcomeKnown
		o.ExitKnown = true
		o.ExecMainCode = 1
		o.ExecMainStatus = int(terminal.ExitCode)
		o.Result = "exit-code"
		if terminal.ExitCode == 0 {
			o.Result = "success"
		}
	}
	return o, nil
}
func stopOwnedOperation(ctx context.Context, r OperationRef) (OperationStopReceipt, error) {
	unlock, err := lockWindowsOwnedSubmission(ctx, r.Intent)
	if err != nil {
		return OperationStopReceipt{}, err
	}
	defer unlock()
	p, err := readWindowsOperatorPayload(r.Intent)
	if err != nil {
		return OperationStopReceipt{}, err
	}
	if !windowsSameNative(windowsRef(p), r) {
		return OperationStopReceipt{}, ErrOperationIdentity
	}
	root, err := readWindowsOwnedRecord(r.Intent)
	if err != nil {
		return OperationStopReceipt{}, err
	}
	if r.RootPID > 0 && (r.RootPID != root.PID || r.RootStartTime != root.Start || r.Native.RootIncarnation != root.Incarnation) {
		return OperationStopReceipt{}, ErrOperationIdentity
	}
	boot, err := windowsOwnedBootID()
	if err != nil {
		return OperationStopReceipt{}, err
	}
	absent, err := windowsOwnedJobAbsent(windowsOwnedJob(r.Intent))
	if err != nil {
		return OperationStopReceipt{}, err
	}
	taskRoot, err := windowsOwnedRetirementTaskRoot(p.BootID, boot, root, absent)
	if err != nil {
		return OperationStopReceipt{}, err
	}
	// Revocation precedes scheduler control. A delayed action cannot claim this
	// intent after Stop, even if Run was queued or its acknowledgement was lost.
	if err = revokeWindowsOwnedSubmission(r.Intent); err != nil {
		return OperationStopReceipt{}, err
	}
	if err = stopWindowsOwnedTask(ctx, p, taskRoot); err != nil {
		return OperationStopReceipt{}, err
	}
	if boot != p.BootID {
		// A reboot destroys the old kernel Job/process objects. Never signal an
		// old PID or a newly occupied same-name Job; only retire the exact task.
		absent, err = windowsOwnedJobAbsent(windowsOwnedJob(r.Intent))
		if err != nil {
			return OperationStopReceipt{}, err
		}
		if !absent {
			return OperationStopReceipt{}, ErrOperationIdentity
		}
		return OperationStopReceipt{Ref: r, NativeQuiescent: true, LauncherReap: LauncherReapUnknown}, nil
	}
	job, err := openWindowsOwnedJobAccess(windowsOwnedJob(r.Intent), windows.READ_CONTROL|0x4|0x8)
	if err != nil {
		return OperationStopReceipt{}, err
	}
	if job != 0 {
		if err = validateWindowsOwnedJobHandle(job, p); err != nil {
			windows.CloseHandle(job)
			return OperationStopReceipt{}, err
		}
		err = windows.TerminateJobObject(job, 1)
		windows.CloseHandle(job)
		if err != nil {
			return OperationStopReceipt{}, err
		}
	}
	for {
		populated, e := windowsOwnedJobPopulated(windowsOwnedJob(r.Intent))
		if e != nil {
			return OperationStopReceipt{}, e
		}
		if !populated {
			return OperationStopReceipt{Ref: r, NativeQuiescent: true, LauncherReap: LauncherReapUnknown}, nil
		}
		select {
		case <-ctx.Done():
			return OperationStopReceipt{}, ctx.Err()
		case <-time.After(operationWaitPoll):
		}
	}
}
func releaseOwnedOperation(ctx context.Context, r OperationRef) error {
	unlock, err := lockWindowsOwnedSubmission(ctx, r.Intent)
	if err != nil {
		return err
	}
	defer unlock()
	p, err := readWindowsOperatorPayload(r.Intent)
	if err != nil {
		return err
	}
	if !windowsSameNative(windowsRef(p), r) {
		return ErrOperationIdentity
	}
	root, err := readWindowsOwnedRecord(r.Intent)
	if err != nil {
		return err
	}
	if r.RootPID > 0 && (r.RootPID != root.PID || r.RootStartTime != root.Start || r.Native.RootIncarnation != root.Incarnation) {
		return ErrOperationIdentity
	}
	// Release never stops an established live role. Rootless queued task
	// admission is revoked only after the role Job namespace is already absent.
	absent, err := windowsOwnedJobAbsent(windowsOwnedJob(r.Intent))
	if err != nil {
		return err
	}
	if !absent {
		return ErrOperationNotQuiescent
	}
	boot, err := windowsOwnedBootID()
	if err != nil {
		return err
	}
	taskRoot, err := windowsOwnedRetirementTaskRoot(p.BootID, boot, root, absent)
	if err != nil {
		return err
	}
	if boot == p.BootID && root.PID > 0 {
		stamp, alive, e := windowsOwnedProcessStart(root.PID)
		if e != nil {
			return e
		}
		if alive && stamp == root.Start {
			return ErrOperationNotQuiescent
		}
	}
	if err = revokeWindowsOwnedSubmission(r.Intent); err != nil {
		return err
	}
	if err = stopWindowsOwnedTask(ctx, p, taskRoot); err != nil {
		return err
	}
	absent, err = windowsOwnedJobAbsent(windowsOwnedJob(r.Intent))
	if err != nil {
		return err
	}
	if !absent {
		return ErrOperationNotQuiescent
	}
	if err = deleteWindowsOwnedTask(p); err != nil {
		return err
	}
	return registry.DeleteKey(registry.USERS, windowsOwnedRecordKey(r.Intent))
}

var windowsOpenJob = windows.NewLazySystemDLL("kernel32.dll").NewProc("OpenJobObjectW")

func openWindowsOwnedJob(name string) (windows.Handle, error) {
	return openWindowsOwnedJobAccess(name, 0x4|windows.READ_CONTROL)
}
func openWindowsOwnedJobAccess(name string, access uint32) (windows.Handle, error) {
	p, e := windows.UTF16PtrFromString(name)
	if e != nil {
		return 0, e
	}
	h, _, e := windowsOpenJob.Call(uintptr(access), 0, uintptr(unsafe.Pointer(p)))
	if h == 0 {
		if errors.Is(e, windows.ERROR_FILE_NOT_FOUND) {
			return 0, nil
		}
		return 0, e
	}
	return windows.Handle(h), nil
}
func windowsOwnedJobPopulated(name string) (bool, error) {
	h, e := openWindowsOwnedJob(name)
	if e != nil || h == 0 {
		return false, e
	}
	defer windows.CloseHandle(h)
	var a windowsJobAccounting
	e = windows.QueryInformationJobObject(h, 1, uintptr(unsafe.Pointer(&a)), uint32(unsafe.Sizeof(a)), nil)
	return a.ActiveProcesses > 0, e
}

type windowsJobAccounting struct {
	TotalUserTime, TotalKernelTime, ThisPeriodTotalUserTime, ThisPeriodTotalKernelTime int64
	TotalPageFaultCount, TotalProcesses, ActiveProcesses, TotalTerminatedProcesses     uint32
}

var windowsBootQuery = windows.NewLazySystemDLL("ntdll.dll").NewProc("NtQuerySystemInformation")

func windowsOwnedBootID() (string, error) {
	var data struct {
		ID           windows.GUID
		FirmwareType uint32
		Flags        uint64
	}
	status, _, _ := windowsBootQuery.Call(90, uintptr(unsafe.Pointer(&data)), unsafe.Sizeof(data), 0)
	if int32(status) < 0 {
		return "", fmt.Errorf("platform: Windows boot identity query failed: NTSTATUS %#x", status)
	}
	return strings.ToLower(strings.Trim(data.ID.String(), "{}")), nil
}

func windowsOwnedRecordKey(i OperationIntent) string {
	return strings.TrimPrefix(i.NativeAccount, "sid:") + `\Software\Vrooli\OwnedJobs\` + i.InvocationID
}
func windowsOperatorLedgerKey(i OperationIntent, access uint32) (registry.Key, error) {
	if !validNativeObserverPrincipal("windows", i.NativeAccount) || i.NativeAccount == "" || !operationInvocation.MatchString(i.InvocationID) {
		return 0, ErrOperationIdentity
	}

	k, e := registry.OpenKey(registry.USERS, windowsOwnedRecordKey(i), access|windows.READ_CONTROL)
	if errors.Is(e, registry.ErrNotExist) {
		return 0, ErrOperationNotFound
	}
	if e != nil {
		return 0, e
	}
	sd, e := windows.GetSecurityInfo(windows.Handle(k), windows.SE_REGISTRY_KEY, windows.DACL_SECURITY_INFORMATION)
	if e != nil {
		k.Close()
		return 0, e
	}
	user, e := windows.GetCurrentProcessToken().GetTokenUser()
	if e != nil {
		k.Close()
		return 0, e
	}
	expected, e := windows.SecurityDescriptorFromString("D:P(A;;KA;;;SY)(A;;KA;;;" + user.User.Sid.String() + ")(A;;RC;;;OW)")
	if e != nil || sd.String() != expected.String() {
		k.Close()
		return 0, ErrOperationIdentity
	}
	return k, nil
}
func prepareWindowsOperatorPayload(p windowsOwnedPayload) error {
	if e := validateWindowsPayload(p); e != nil {
		return e
	}
	user, e := windows.GetCurrentProcessToken().GetTokenUser()
	if e != nil {
		return e
	}
	if p.Spec.NativeAccount != "sid:"+user.User.Sid.String() {
		return ErrOperationIdentity
	}
	hive, e := registry.OpenKey(registry.USERS, user.User.Sid.String(), registry.QUERY_VALUE)
	if e != nil {
		return fmt.Errorf("platform: operator SID hive must be explicitly loaded: %w", e)
	}
	hive.Close()
	sd, e := windows.SecurityDescriptorFromString("D:P(A;;KA;;;SY)(A;;KA;;;" + user.User.Sid.String() + ")(A;;RC;;;OW)")
	if e != nil {
		return e
	}
	// RegCreateKeyEx applies the DACL at creation: no inherited-access window.
	sa := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: sd}
	name, _ := windows.UTF16PtrFromString(windowsOwnedRecordKey(p.Intent))
	var raw windows.Handle
	var disposition uint32
	status, _, _ := windows.NewLazySystemDLL("advapi32.dll").NewProc("RegCreateKeyExW").Call(uintptr(registry.USERS), uintptr(unsafe.Pointer(name)), 0, 0, 0, uintptr(registry.ALL_ACCESS), uintptr(unsafe.Pointer(&sa)), uintptr(unsafe.Pointer(&raw)), uintptr(unsafe.Pointer(&disposition)))
	if status != 0 {
		return syscall.Errno(status)
	}
	k := registry.Key(raw)
	if disposition != 1 {
		k.Close()
		empty, inspectErr := windowsOwnedUnpublishedSlot(p.Intent)
		if inspectErr != nil {
			return inspectErr
		}
		if !empty {
			return ErrOperationExists
		}
		if e = registry.DeleteKey(registry.USERS, windowsOwnedRecordKey(p.Intent)); e != nil {
			return e
		}
		// No payload/native submission ever existed. Recreate only this empty slot;
		// complete or partial value-bearing records are never replayed/overwritten.
		return prepareWindowsOperatorPayload(p)
	}
	defer k.Close()
	b, e := json.Marshal(p)
	if e != nil {
		return e
	}
	if e = k.SetStringValue("Payload", string(b)); e != nil {
		return e
	}
	return flushWindowsOwnedKey(k)
}

// Read-only proof for a pre-Payload key. Never treats a value-bearing corrupt
// slot or retained native Job (even empty) as absence or replay authority.
func windowsOwnedUnpublishedSlot(i OperationIntent) (bool, error) {
	k, e := windowsOperatorLedgerKey(i, registry.QUERY_VALUE|registry.ENUMERATE_SUB_KEYS)
	if e != nil {
		return false, e
	}
	defer k.Close()
	info, e := k.Stat()
	if e != nil {
		return false, e
	}
	absent, e := windowsOwnedJobAbsent(windowsOwnedJob(i))
	if e != nil {
		return false, e
	}
	return windowsOwnedUnpublishedSlotRemovable(info.ValueCount, info.SubKeyCount, absent), nil
}

func readWindowsOperatorPayload(i OperationIntent) (windowsOwnedPayload, error) {
	var p windowsOwnedPayload
	k, e := windowsOperatorLedgerKey(i, registry.QUERY_VALUE)
	if e != nil {
		return p, e
	}
	defer k.Close()
	raw, _, e := k.GetStringValue("Payload")
	if errors.Is(e, registry.ErrNotExist) {
		empty, inspectErr := windowsOwnedUnpublishedSlot(i)
		if inspectErr != nil {
			return p, inspectErr
		}
		if empty {
			return p, ErrOperationNotFound
		}
		return p, ErrOperationIdentity
	}
	if e != nil {
		return p, e
	}
	return decodeWindowsOperatorPayloadJSON(raw, i)
}

func writeWindowsOwnedRecord(i OperationIntent, r windowsOwnedRootRecord, _ string) error {
	k, e := windowsOperatorLedgerKey(i, registry.QUERY_VALUE|registry.SET_VALUE)
	if e != nil {
		return e
	}
	defer k.Close()
	if _, _, e = k.GetStringValue("Root"); e == nil {
		return ErrOperationExists
	} else if !errors.Is(e, registry.ErrNotExist) {
		return e
	}
	b, e := json.Marshal(r)
	if e != nil {
		return e
	}
	if e = k.SetStringValue("Root", string(b)); e != nil {
		return e
	}
	return flushWindowsOwnedKey(k)
}
func readWindowsOwnedRecord(i OperationIntent) (windowsOwnedRootRecord, error) {
	var r windowsOwnedRootRecord
	k, e := windowsOperatorLedgerKey(i, registry.QUERY_VALUE)
	if e != nil {
		return r, e
	}
	defer k.Close()
	raw, _, e := k.GetStringValue("Root")
	if errors.Is(e, registry.ErrNotExist) {
		return r, nil
	}
	if e != nil {
		return r, e
	}
	if len(raw) > 4096 {
		return r, ErrOperationIdentity
	}
	if e = json.Unmarshal([]byte(raw), &r); e != nil {
		return r, e
	}
	if r.PID <= 0 || !nativeIncarnationShape("windows", r.PID, r.Incarnation) || !nativeStartMatches("windows", r.Start, r.Incarnation) || r.ImageID == "" || r.ImageHandle == 0 || r.JobHandle == 0 || r.TaskInstance == "" {
		return r, ErrOperationIdentity
	}
	return r, nil
}
func readWindowsOwnedOutcome(i OperationIntent) (windowsOwnedTerminalRecord, error) {
	var t windowsOwnedTerminalRecord
	k, e := windowsOperatorLedgerKey(i, registry.QUERY_VALUE)
	if e != nil {
		return t, e
	}
	defer k.Close()
	raw, _, e := k.GetStringValue("Terminal")
	if errors.Is(e, registry.ErrNotExist) {
		return t, nil
	}
	if e != nil {
		return t, e
	}
	if len(raw) > 8192 {
		return t, ErrOperationIdentity
	}
	e = json.Unmarshal([]byte(raw), &t)
	return t, e
}
func flushWindowsOwnedKey(k registry.Key) error {
	status, _, _ := windows.NewLazySystemDLL("advapi32.dll").NewProc("RegFlushKey").Call(uintptr(k))
	if status != 0 {
		return syscall.Errno(status)
	}
	return nil
}
func lockWindowsOwnedSubmission(ctx context.Context, i OperationIntent) (func(), error) {
	runtime.LockOSThread()
	locked := true
	defer func() {
		if locked {
			runtime.UnlockOSThread()
		}
	}()
	user, e := windows.GetCurrentProcessToken().GetTokenUser()
	if e != nil {
		return nil, e
	}
	sd, e := windows.SecurityDescriptorFromString("D:P(A;;GA;;;SY)(A;;GA;;;" + user.User.Sid.String() + ")(A;;RC;;;OW)")
	if e != nil {
		return nil, e
	}
	sa := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: sd}
	name, _ := windows.UTF16PtrFromString(`Global\VrooliSubmit-` + user.User.Sid.String() + "-" + windowsOwnedServiceName(i))
	h, e := windows.CreateMutexEx(&sa, name, 0, windows.SYNCHRONIZE|windows.MUTEX_MODIFY_STATE)
	if errors.Is(e, windows.ERROR_ALREADY_EXISTS) {
		e = nil
	}
	if e != nil {
		return nil, e
	}
	for {
		if e = ctx.Err(); e != nil {
			windows.CloseHandle(h)
			return nil, e
		}
		state, e := windows.WaitForSingleObject(h, 100)
		if e != nil {
			windows.CloseHandle(h)
			return nil, e
		}
		if state == uint32(windows.WAIT_OBJECT_0) || state == uint32(windows.WAIT_ABANDONED) {
			locked = false
			return func() { windows.ReleaseMutex(h); windows.CloseHandle(h); runtime.UnlockOSThread() }, nil
		}
		if state != uint32(windows.WAIT_TIMEOUT) {
			windows.CloseHandle(h)
			return nil, ErrOperationIdentity
		}
	}
}
func resolveOwnedServiceAccount(ctx context.Context, account string) (OwnedServiceAccount, error) {
	if e := ctx.Err(); e != nil {
		return OwnedServiceAccount{}, e
	}
	sid, e := resolveWindowsObserverSID(account)
	if e != nil || sid == nil {
		return OwnedServiceAccount{}, ErrOperationIdentity
	}
	user, e := windows.GetCurrentProcessToken().GetTokenUser()
	if e != nil {
		return OwnedServiceAccount{}, e
	}
	if !windows.EqualSid(user.User.Sid, sid) {
		return OwnedServiceAccount{}, ErrOperationIdentity
	}
	return OwnedServiceAccount{Name: account, Principal: account}, nil
}
func prepareOwnedServiceAccount(ctx context.Context, account string) (OwnedServiceAccount, error) {
	return resolveOwnedServiceAccount(ctx, account)
}

// Keep creation timestamps decimal for persisted identity consumers.
func windowsOwnedProcessStart(pid int) (string, bool, error) {
	if pid <= 0 {
		return "", false, nil
	}
	h, e := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, uint32(pid))
	if errors.Is(e, windows.ERROR_INVALID_PARAMETER) {
		return "", false, nil
	}
	if e != nil {
		return "", false, e
	}
	defer windows.CloseHandle(h)
	state, e := windows.WaitForSingleObject(h, 0)
	if e != nil {
		return "", false, e
	}
	if state != uint32(windows.WAIT_TIMEOUT) {
		return "", false, nil
	}
	var c, x, k, u windows.Filetime
	if e = windows.GetProcessTimes(h, &c, &x, &k, &u); e != nil {
		return "", false, e
	}
	return strconv.FormatUint(uint64(c.HighDateTime)<<32|uint64(c.LowDateTime), 10), true, nil
}

func openWindowsOwnedServiceHost(path string) (*os.File, error) {
	// The resource launcher is a sealed copy, not necessarily the installed
	// pathname/inode. Only independently approved installed bytes authorize it.
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || len(path) > 4096 {
		return nil, ErrOperationIdentity
	}
	approved, e := openInstalledNativeSupervisorImage(context.Background())
	if e != nil {
		return nil, e
	}
	defer approved.Close()
	h := sha256.New()
	n, e := io.Copy(h, io.LimitReader(approved, maxNativeSystemBrokerImageBytes+1))
	if e != nil {
		return nil, e
	}
	if n <= 0 || n > maxNativeSystemBrokerImageBytes {
		return nil, ErrOperationIdentity
	}
	held, e := windowsEnsureExistingBrokerParents(filepath.Dir(path))
	if e != nil {
		return nil, e
	}
	defer func() {
		for _, handle := range held {
			windows.CloseHandle(handle)
		}
	}()
	handle, e := windowsOpenBrokerPath(path, false, windows.GENERIC_READ)
	if e != nil {
		return nil, e
	}
	image := os.NewFile(uintptr(handle), path)
	if e = verifyNativeSystemBrokerImage(image, hex.EncodeToString(h.Sum(nil))); e == nil {
		e = verifyWindowsMaintainedLauncherPE(image, runtime.GOARCH)
	}
	if e != nil {
		image.Close()
		return nil, e
	}
	return image, nil
}

func grantWindowsOwnedHostRead(ctx context.Context, path, account string) error {
	unlock, e := lockWindowsOwnedSubmission(ctx, OperationIntent{Kind: ownedServiceKind, Owner: "host-image-acl"})
	if e != nil {
		return e
	}
	defer unlock()
	image, e := openWindowsOwnedServiceHost(path)
	if e != nil {
		return e
	}
	defer image.Close()
	sid, _, _, e := windows.LookupSID("", account)
	if e != nil {
		return e
	}
	// The validation handle pins the path while the administrative ACL handle
	// is opened; both handles must identify the same immutable image.
	handle, e := windowsOpenBrokerPath(path, false, windows.GENERIC_READ|windows.WRITE_DAC)
	if e != nil {
		return e
	}
	defer windows.CloseHandle(handle)
	var info windows.ByHandleFileInformation
	if e = windows.GetFileInformationByHandle(handle, &info); e != nil {
		return e
	}
	expectedID, e := windowsImageID(image)
	if e != nil {
		return e
	}
	actualID := fmt.Sprintf("windows-file:%08x:%08x%08x", info.VolumeSerialNumber, info.FileIndexHigh, info.FileIndexLow)
	if actualID != expectedID {
		return ErrOperationIdentity
	}
	sd, e := windows.GetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if e != nil {
		return e
	}
	entry := windows.EXPLICIT_ACCESS{AccessPermissions: windows.FILE_GENERIC_READ | windows.FILE_GENERIC_EXECUTE, AccessMode: windows.GRANT_ACCESS, Trustee: windows.TRUSTEE{TrusteeForm: windows.TRUSTEE_IS_SID, TrusteeType: windows.TRUSTEE_IS_USER, TrusteeValue: windows.TrusteeValueFromSID(sid)}}
	merged, e := windows.BuildSecurityDescriptor(nil, nil, []windows.EXPLICIT_ACCESS{entry}, nil, sd)
	if e != nil {
		return e
	}
	acl, _, e := merged.DACL()
	if e != nil {
		return e
	}
	return windows.SetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil)
}
