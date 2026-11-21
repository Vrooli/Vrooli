//go:build windows

package platform

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var windowsWrapperCustody struct {
	sync.Mutex
	payload *windowsOwnedPayload
	job     windows.Handle
	image   *os.File
}

// EstablishWindowsOwnedServiceCustody runs in the maintained trusted wrapper,
// before capsule lookup/witnesses. Its sole Job handle survives the submitting
// CLI. No vendor child inherits that handle. Repeated calls validate custody.
func EstablishWindowsOwnedServiceCustody() error { _, e := currentWindowsOwnedJob(); return e }

func bootstrapWindowsWrapperCustody() (windowsOwnedPayload, error) {
	windowsWrapperCustody.Lock()
	defer windowsWrapperCustody.Unlock()
	if windowsWrapperCustody.payload != nil {
		return *windowsWrapperCustody.payload, nil
	}
	return windowsOwnedPayload{}, ErrOperationIdentity
}

func readWindowsOwnedCustody(i OperationIntent) (windowsOwnedCustodyRecord, error) {
	var c windowsOwnedCustodyRecord
	k, e := windowsOperatorLedgerKey(i, registry.QUERY_VALUE)
	if e != nil {
		return c, e
	}
	defer k.Close()
	marker, _, e := k.GetStringValue("CustodyReady")
	if errors.Is(e, registry.ErrNotExist) {
		return c, nil
	}
	if e != nil {
		return c, e
	}
	raw, _, e := k.GetStringValue("Custody")
	if errors.Is(e, registry.ErrNotExist) {
		return c, ErrOperationIdentity
	}
	if e != nil {
		return c, e
	}
	if len(raw) > 8192 || !windowsCustodyReadyMatches(raw, marker) {
		return c, ErrOperationIdentity
	}
	d := json.NewDecoder(strings.NewReader(raw))
	d.DisallowUnknownFields()
	if e = d.Decode(&c); e != nil {
		return c, e
	}
	if e = d.Decode(new(any)); e != io.EOF {
		return c, ErrOperationIdentity
	}
	return c, nil
}
func verifyWindowsOwnedCustody(p windowsOwnedPayload, root windowsOwnedRootRecord) error {
	if _, e := verifyWindowsOwnedTaskAction(p, root); e != nil {
		return e
	}
	c, e := readWindowsOwnedCustody(p.Intent)
	if e != nil {
		return e
	}
	if !c.matches(p, root) {
		return ErrOperationIdentity
	}
	process, e := windowsServingProcess(root.PID, root.Incarnation)
	if e != nil {
		return e
	}
	defer windows.CloseHandle(process)
	if _, e = windowsProcessOperatorSecurityProof(process, p.Spec.NativeAccount); e != nil {
		return e
	}
	image, e := windowsServingBirthImage(process, root.ImageHandle, root.ImageID)
	if e != nil {
		return e
	}
	image.Close()
	var job windows.Handle
	if e = windows.DuplicateHandle(process, windows.Handle(root.JobHandle), windows.CurrentProcess(), &job, windows.READ_CONTROL|0x4, false, 0); e != nil {
		return e
	}
	// Even query handles keep a Job alive. This lease is closed on every return.
	defer windows.CloseHandle(job)
	if e = validateWindowsOwnedJobHandle(job, p); e != nil {
		return e
	}
	inside, e := windowsProcessInExactJob(process, job)
	if e != nil {
		return e
	}
	if !inside {
		return ErrOperationIdentity
	}
	return windowsServingProcessAlive(process)
}
func validateWindowsOwnedJobHandle(job windows.Handle, p windowsOwnedPayload) error {
	if job == 0 {
		return ErrOperationIdentity
	}
	sd, e := windows.GetSecurityInfo(job, windows.SE_KERNEL_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if e != nil {
		return e
	}
	expected, e := windows.SecurityDescriptorFromString("D:P(A;;GA;;;SY)(A;;GA;;;" + strings.TrimPrefix(p.Spec.NativeAccount, "sid:") + ")(A;;RC;;;OW)")
	if e != nil {
		return e
	}
	if sd.String() != expected.String() {
		return ErrOperationIdentity
	}
	var size uint32
	query := windows.NewLazySystemDLL("ntdll.dll").NewProc("NtQueryObject")
	status, _, _ := query.Call(uintptr(job), 1, 0, 0, uintptr(unsafe.Pointer(&size)))
	if size == 0 || size > 65536 {
		return ErrOperationIdentity
	}
	data := make([]byte, size)
	status, _, _ = query.Call(uintptr(job), 1, uintptr(unsafe.Pointer(&data[0])), uintptr(size), uintptr(unsafe.Pointer(&size)))
	if int32(status) < 0 {
		return windows.NTStatus(status).Errno()
	}
	name := (*windows.NTUnicodeString)(unsafe.Pointer(&data[0]))
	if name.Length == 0 || name.Length%2 != 0 || name.Length > name.MaximumLength || name.Buffer == nil {
		return ErrOperationIdentity
	}
	object := windows.UTF16ToString(unsafe.Slice(name.Buffer, int(name.Length/2)))
	expectedName := `\BaseNamedObjects\` + strings.TrimPrefix(windowsOwnedJob(p.Intent), `Global\`)
	if object != expectedName {
		return ErrOperationIdentity
	}
	var limits windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	if e = windows.QueryInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits)), nil); e != nil {
		return e
	}
	expectedFlags := uint32(windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE | windows.JOB_OBJECT_LIMIT_JOB_MEMORY | windows.JOB_OBJECT_LIMIT_ACTIVE_PROCESS)
	if limits.BasicLimitInformation.LimitFlags != expectedFlags || uint64(limits.JobMemoryLimit) != uint64(p.Spec.Limits.MemoryMaxBytes) || uint64(limits.BasicLimitInformation.ActiveProcessLimit) != uint64(p.Spec.Limits.TasksMax) {
		return ErrOperationIdentity
	}
	var rate jobObjectCPURateControlInformation
	if e = windows.QueryInformationJobObject(job, jobObjectCPURateControlInformationClass, uintptr(unsafe.Pointer(&rate)), uint32(unsafe.Sizeof(rate)), nil); e != nil {
		return e
	}
	if rate.ControlFlags != jobObjectCPURateControlEnable|jobObjectCPURateControlWeightBased || rate.Value != uint32(jobWeightForWeight(p.Spec.Limits.CPUWeight)) {
		return ErrOperationIdentity
	}
	return nil
}
