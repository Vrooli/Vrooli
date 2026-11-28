//go:build windows

package platform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func verifyWindowsOwnedTaskDescriptor(raw, account string) error {
	sd, e := windows.SecurityDescriptorFromString(raw)
	if e != nil {
		return e
	}
	sid := strings.TrimPrefix(account, "sid:")
	owner, _, e := sd.Owner()
	if e != nil {
		return e
	}
	if owner == nil || owner.String() != sid {
		return ErrOperationIdentity
	}
	control, _, e := sd.Control()
	if e != nil {
		return e
	}
	if control&windows.SE_DACL_PROTECTED == 0 {
		return ErrOperationIdentity
	}
	acl, _, e := sd.DACL()
	if e != nil {
		return e
	}
	if acl == nil || acl.AceCount != 2 {
		return ErrOperationIdentity
	}
	seen := map[string]bool{}
	for n := uint32(0); n < uint32(acl.AceCount); n++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if e = windows.GetAce(acl, n, &ace); e != nil {
			return e
		}
		who := (*windows.SID)(unsafe.Pointer(&ace.SidStart)).String()
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || ace.Header.AceFlags != 0 || seen[who] || (who != "S-1-5-18" && who != sid) || (ace.Mask != windows.GENERIC_ALL && ace.Mask != (windows.STANDARD_RIGHTS_REQUIRED|windows.SYNCHRONIZE|0x1ff)) {
			return ErrOperationIdentity
		}
		seen[who] = true
	}
	return nil
}
func windowsOwnedTaskSnapshot(p windowsOwnedPayload) (WindowsTaskSnapshot, error) {
	s, e := WindowsTaskStatus(windowsOwnedTaskName(p.Intent))
	if e != nil {
		return s, e
	}
	if e = windowsOwnedTaskDefinitionMatches(p, s); e != nil {
		return s, e
	}
	return s, verifyWindowsOwnedTaskDescriptor(s.SecurityDescriptor, p.Spec.NativeAccount)
}
func registerWindowsOwnedTask(p windowsOwnedPayload) error {
	return withTaskScheduler(func(folder *taskDispatch) error {
		var values []taskVariant
		defer func() {
			for n := range values {
				taskClear(&values[n])
			}
		}()
		sid := strings.TrimPrefix(p.Spec.NativeAccount, "sid:")
		for _, text := range []string{windowsOwnedTaskName(p.Intent), windowsOwnedTaskXML(p), sid, "O:" + sid + "D:P(A;;GA;;;SY)(A;;GA;;;" + sid + ")"} {
			v, e := taskString(text)
			if e != nil {
				return e
			}
			values = append(values, v)
		}
		// TASK_CREATE, DONT_ADD_PRINCIPAL_ACE, IGNORE_REGISTRATION_TRIGGERS.
		// Never update a same-name task, including after uncertain registration.
		task, e := folder.object("RegisterTask", 1, values[0], values[1], taskVariant{kind: 3, value: 0x32}, values[2], taskVariant{}, taskVariant{kind: 3, value: 2}, values[3])
		if e != nil {
			return e
		}
		defer task.release()
		s := WindowsTaskSnapshot{Name: windowsOwnedTaskName(p.Intent)}
		if e = readWindowsTaskSnapshot(task, &s); e != nil {
			return e
		}
		if e = windowsOwnedTaskDefinitionMatches(p, s); e != nil {
			return e
		}
		return verifyWindowsOwnedTaskDescriptor(s.SecurityDescriptor, p.Spec.NativeAccount)
	})
}
func setWindowsOwnedRunIssued(i OperationIntent) error {
	k, e := windowsOperatorLedgerKey(i, registry.QUERY_VALUE|registry.SET_VALUE)
	if e != nil {
		return e
	}
	defer k.Close()
	if _, _, e = k.GetIntegerValue(windowsOwnedRunIssuedValue); e == nil {
		return ErrOperationExists
	} else if !errors.Is(e, registry.ErrNotExist) {
		return e
	}
	if e = k.SetDWordValue(windowsOwnedRunIssuedValue, windowsOwnedMarker); e != nil {
		return e
	}
	return flushWindowsOwnedKey(k)
}
func verifyWindowsOwnedRunIssued(i OperationIntent) error {
	k, e := windowsOperatorLedgerKey(i, registry.QUERY_VALUE)
	if e != nil {
		return e
	}
	defer k.Close()
	value, _, e := k.GetIntegerValue(windowsOwnedRunIssuedValue)
	if e != nil {
		return e
	}
	if value != uint64(windowsOwnedMarker) {
		return ErrOperationIdentity
	}
	return nil
}

// PrepareWindowsOwnedServiceSubmission is called only by the maintained early
// capsule reader, after a non-effecting typed capsule read and before any
// witness, owner IPC, hardware access or vendor execution. The scheduler owns
// THIS wrapper's lifetime; no relay exits leaving scheduler-owned descendants.
func PrepareWindowsOwnedServiceSubmission(ctx context.Context, i OperationIntent) error {
	if ctx == nil {
		return ErrOperationIdentity
	}
	if e := ctx.Err(); e != nil {
		return e
	}
	windowsWrapperCustody.Lock()
	defer windowsWrapperCustody.Unlock()
	if windowsWrapperCustody.payload != nil {
		if windowsWrapperCustody.payload.Intent != i {
			return ErrOperationIdentity
		}
		return nil
	}
	unlock, e := lockWindowsOwnedSubmission(ctx, i)
	if e != nil {
		return e
	}
	defer unlock()
	p, e := readWindowsOperatorPayload(i)
	if e != nil {
		return e
	}
	if e = validateWindowsMaintainedLaunchShape(i, p.Spec); e != nil {
		return e
	}
	if e = verifyWindowsOwnedNotRevoked(i); e != nil {
		return e
	}
	if e = verifyWindowsOwnedRunIssued(i); e != nil {
		return e
	}
	root, e := readWindowsOwnedRecord(i)
	if e != nil {
		return e
	}
	if root.PID != 0 {
		return ErrOperationExists
	}
	if _, e = resolveOwnedServiceAccount(ctx, p.Spec.NativeAccount); e != nil {
		return e
	}
	if _, e = windowsProcessOperatorSecurityProof(windows.CurrentProcess(), p.Spec.NativeAccount); e != nil {
		return e
	}
	boot, e := windowsOwnedBootID()
	if e != nil {
		return e
	}
	if boot != p.BootID {
		return ErrOperationIdentity
	}
	args, e := processArguments(os.Getpid())
	if e != nil {
		return e
	}
	if len(args) != len(p.Spec.Args)+1 || !strings.EqualFold(args[0], p.Spec.Path) || !slices.Equal(args[1:], p.Spec.Args) {
		return ErrOperationIdentity
	}
	matches, e := processWorkingDirectoryMatches(os.Getpid(), p.Spec.Dir)
	if e != nil {
		return e
	}
	if !matches {
		return ErrOperationIdentity
	}
	image, e := openWindowsOwnedServiceSpecHost(ctx, p.Spec)
	if e != nil {
		return e
	}
	keep := false
	defer func() {
		if !keep {
			image.Close()
		}
	}()
	if e = verifyWindowsMappedImageHandle(windows.CurrentProcess(), image); e != nil {
		return e
	}
	stamp, alive, e := windowsOwnedProcessStart(os.Getpid())
	if e != nil {
		return e
	}
	if !alive {
		return ErrOperationIdentity
	}
	ticks, e := strconv.ParseUint(stamp, 10, 64)
	if e != nil {
		return e
	}
	creation := (&windows.Filetime{HighDateTime: uint32(ticks >> 32), LowDateTime: uint32(ticks)}).Nanoseconds()
	imageID, e := windowsImageID(image)
	if e != nil {
		return e
	}
	root = windowsOwnedRootRecord{PID: os.Getpid(), Start: stamp, Incarnation: fmt.Sprintf("windows:%d:%d", os.Getpid(), creation), ImageID: imageID, ImageHandle: uint64(image.Fd())}
	instance, e := verifyWindowsOwnedTaskAction(p, root)
	if e != nil {
		return e
	}
	root.TaskInstance = instance
	job, e := createWindowsOwnedJob(p)
	if e != nil {
		return e
	}
	defer func() {
		if !keep {
			windows.CloseHandle(job)
		}
	}()
	// Joining happens before custody/any untrusted code. Children inherit this Job
	// atomically; the caller's scenario Job is not an ancestor of this task action.
	if e = windows.AssignProcessToJobObject(job, windows.CurrentProcess()); e != nil {
		return e
	}
	root.JobHandle = uint64(job)
	if e = writeWindowsOwnedRecord(i, root, p.Spec.ObserverPrincipal); e != nil {
		return e
	}
	// Scheduler ambient configuration never reaches the vendor. Exact explicit
	// argv remain OS-observable; environment is normalized in this trusted code.
	os.Clearenv()
	for _, entry := range p.Spec.Env {
		key, value, _ := strings.Cut(entry, "=")
		if e = os.Setenv(key, value); e != nil {
			return e
		}
	}
	k, e := windowsOperatorLedgerKey(i, windowsCustodyRegistryWriteAccess)
	if e != nil {
		return e
	}
	defer k.Close()
	raw, e := json.Marshal(windowsOwnedCustodyRecord{Version: 1, Intent: i, BootID: p.BootID, Root: root})
	if e != nil {
		return e
	}
	if e = commitWindowsOwnedCustody(string(raw), windowsCustodyCommitEffects{WritePending: func(raw string) error { return k.SetStringValue("Custody", raw) }, Flush: func() error { return flushWindowsOwnedKey(k) }, WriteReady: func(marker string) error { return k.SetStringValue("CustodyReady", marker) }}); e != nil {
		return e
	}
	windowsWrapperCustody.payload = &p
	windowsWrapperCustody.job = job
	windowsWrapperCustody.image = image
	keep = true
	return nil
}

func createWindowsOwnedJob(p windowsOwnedPayload) (windows.Handle, error) {
	if uint64(uintptr(p.Spec.Limits.MemoryMaxBytes)) != uint64(p.Spec.Limits.MemoryMaxBytes) || uint64(p.Spec.Limits.TasksMax) > uint64(^uint32(0)) {
		return 0, ErrOperationIdentity
	}
	sd, e := windows.SecurityDescriptorFromString("D:P(A;;GA;;;SY)(A;;GA;;;" + strings.TrimPrefix(p.Spec.NativeAccount, "sid:") + ")(A;;RC;;;OW)")
	if e != nil {
		return 0, e
	}
	sa := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: sd}
	name, _ := windows.UTF16PtrFromString(windowsOwnedJob(p.Intent))
	h, _, e := windows.NewLazySystemDLL("kernel32.dll").NewProc("CreateJobObjectW").Call(uintptr(unsafe.Pointer(&sa)), uintptr(unsafe.Pointer(name)))
	job := windows.Handle(h)
	if job == 0 {
		return 0, e
	}
	if errors.Is(e, windows.ERROR_ALREADY_EXISTS) {
		windows.CloseHandle(job)
		return 0, ErrOperationExists
	}
	ok := false
	defer func() {
		if !ok {
			windows.CloseHandle(job)
		}
	}()
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE | windows.JOB_OBJECT_LIMIT_JOB_MEMORY | windows.JOB_OBJECT_LIMIT_ACTIVE_PROCESS
	limits.JobMemoryLimit = uintptr(p.Spec.Limits.MemoryMaxBytes)
	limits.BasicLimitInformation.ActiveProcessLimit = uint32(p.Spec.Limits.TasksMax)
	if _, e = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); e != nil {
		return 0, e
	}
	rate := jobObjectCPURateControlInformation{ControlFlags: jobObjectCPURateControlEnable | jobObjectCPURateControlWeightBased, Value: uint32(jobWeightForWeight(p.Spec.Limits.CPUWeight))}
	if _, e = windows.SetInformationJobObject(job, jobObjectCPURateControlInformationClass, uintptr(unsafe.Pointer(&rate)), uint32(unsafe.Sizeof(rate))); e != nil {
		return 0, e
	}
	ok = true
	return job, nil
}

func verifyWindowsOwnedTaskAction(p windowsOwnedPayload, root windowsOwnedRootRecord) (string, error) {
	before, e := windowsOwnedTaskSnapshot(p)
	if e != nil {
		return "", e
	}
	if !before.Enabled || before.State != 4 || len(before.Instances) != 1 || (root.TaskInstance != "" && before.Instances[0].GUID != root.TaskInstance) {
		return "", ErrOperationIdentity
	}
	instance := before.Instances[0]
	process, e := windowsServingProcess(root.PID, root.Incarnation)
	if e != nil {
		return "", e
	}
	defer windows.CloseHandle(process)
	engine, e := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, instance.EnginePID)
	if e != nil {
		return "", e
	}
	defer windows.CloseHandle(engine)
	var birth, exit, kernel, user windows.Filetime
	if e = windows.GetProcessTimes(engine, &birth, &exit, &kernel, &user); e != nil {
		return "", e
	}
	query, e := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.SYNCHRONIZE, false, uint32(root.PID))
	if e != nil {
		return "", e
	}
	defer windows.CloseHandle(query)
	var actionBirth, actionExit, actionKernel, actionUser windows.Filetime
	if e = windows.GetProcessTimes(query, &actionBirth, &actionExit, &actionKernel, &actionUser); e != nil {
		return "", e
	}
	if root.Incarnation != fmt.Sprintf("windows:%d:%d", root.PID, actionBirth.Nanoseconds()) {
		return "", ErrOperationIdentity
	}
	var basic windows.PROCESS_BASIC_INFORMATION
	if e = windows.NtQueryInformationProcess(query, windows.ProcessBasicInformation, unsafe.Pointer(&basic), uint32(unsafe.Sizeof(basic)), nil); e != nil {
		return "", e
	}
	ticks, e := strconv.ParseUint(root.Start, 10, 64)
	if e != nil {
		return "", e
	}
	if !windowsTaskActionRelation(uint32(root.PID), uint32(basic.InheritedFromUniqueProcessId), ticks, instance, uint64(birth.HighDateTime)<<32|uint64(birth.LowDateTime)) {
		return "", ErrOperationIdentity
	}
	if status, e := windows.WaitForSingleObject(engine, 0); e != nil || status != uint32(windows.WAIT_TIMEOUT) {
		return "", ErrOperationIdentity
	}
	after, e := windowsOwnedTaskSnapshot(p)
	if e != nil {
		return "", e
	}
	if !windowsTaskSameIncarnation(before, after) {
		return "", ErrOperationIdentity
	}
	return instance.GUID, nil
}

func revokeWindowsOwnedSubmission(i OperationIntent) error {
	k, e := windowsOperatorLedgerKey(i, registry.QUERY_VALUE|registry.SET_VALUE)
	if e != nil {
		return e
	}
	defer k.Close()
	if e = k.SetDWordValue(windowsOwnedRevokedValue, windowsOwnedMarker); e != nil {
		return e
	}
	return flushWindowsOwnedKey(k)
}
func verifyWindowsOwnedNotRevoked(i OperationIntent) error {
	k, e := windowsOperatorLedgerKey(i, registry.QUERY_VALUE)
	if e != nil {
		return e
	}
	defer k.Close()
	_, _, e = k.GetIntegerValue(windowsOwnedRevokedValue)
	if errors.Is(e, registry.ErrNotExist) {
		return nil
	}
	if e != nil {
		return e
	}
	return ErrOperationIdentity
}
func stopWindowsOwnedTask(ctx context.Context, p windowsOwnedPayload, root windowsOwnedRootRecord) error {
	s, e := windowsOwnedTaskSnapshot(p)
	if taskNativeErrorMissing(e) {
		return nil
	}
	if e != nil {
		return e
	}
	if len(s.Instances) > 1 {
		return ErrOperationIdentity
	}
	if root.TaskInstance != "" && len(s.Instances) == 1 && s.Instances[0].GUID != root.TaskInstance {
		return ErrOperationIdentity
	}
	if e = windowsSetTaskEnabled(s.Name, false); e != nil {
		return e
	}
	// Disable prevents new Run; it does not prove an already queued dispatch
	// was canceled. Stop the UUID-exact registered task itself, not merely the
	// instances GetInstances happens to expose. Native failure stays unknown.
	if e = windowsStopRegisteredTask(s.Name); e != nil {
		return e
	}
	// Admission and Run are serialized by the same mutex as revocation.
	// GetInstances covers running instances; queued-without-instance is unknown.
	for {
		s, e = windowsOwnedTaskSnapshot(p)
		if taskNativeErrorMissing(e) {
			return nil
		}
		if e != nil {
			return e
		}
		if s.Enabled {
			return ErrOperationIdentity
		}
		if len(s.Instances) == 0 {
			quiet, e := windowsTaskQuiescent(s)
			if e != nil {
				return e
			}
			if !quiet {
				return ErrOperationNotQuiescent
			}
			return nil
		}
		if len(s.Instances) > 1 {
			return ErrOperationIdentity
		}
		if root.TaskInstance != "" && s.Instances[0].GUID != root.TaskInstance {
			return ErrOperationIdentity
		}
		if e = WindowsStopTaskInstance(s.Name, s.Instances[0].GUID); e != nil && !errors.Is(e, ErrOperationNotFound) {
			return e
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(operationWaitPoll):
		}
	}
}
func deleteWindowsOwnedTask(p windowsOwnedPayload) error {
	return deleteWindowsQuiescentTask(func() (WindowsTaskSnapshot, error) {
		return windowsOwnedTaskSnapshot(p)
	}, windowsDeleteTask, taskNativeErrorMissing)
}
