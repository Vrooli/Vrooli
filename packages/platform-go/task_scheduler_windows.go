//go:build windows

package platform

// Task Scheduler's automation interfaces use IDispatch so VARIANT values never
// cross the syscall boundary by value (whose ABI differs on amd64 and arm64).
import (
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"unsafe"
)

type taskDispatch struct{ vtable *[7]uintptr }
type taskVariant struct {
	kind     uint16
	reserved [3]uint16
	value    uint64
	extra    uint64
}

// taskVariantPointer reads the pointer member of the native union. Numeric
// variants are never passed here; native allocation lifetime remains explicit.
func taskVariantPointer(v *taskVariant) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&v.value))
}

// EXCEPINFO uses native pointer alignment on both supported 64-bit ABIs.
type taskExceptionInfo struct {
	Code, Reserved                uint16
	Source, Description, HelpFile uintptr
	HelpContext                   uint32
	Reserved2, Deferred           uintptr
	SCode                         int32
}

type taskDispatchParameters struct {
	args              *taskVariant
	named             *int32
	count, namedCount uint32
}

// VARIANT includes a two-pointer record union and is 24 bytes on 64-bit Windows.
var taskOle32 = windows.NewLazySystemDLL("ole32.dll")
var taskOleAut32 = windows.NewLazySystemDLL("oleaut32.dll")

type windowsTaskHRESULT uint32

func (e windowsTaskHRESULT) Error() string {
	return fmt.Sprintf("platform: Task Scheduler HRESULT 0x%08x", uint32(e))
}
func taskHRESULT(value uintptr) error {
	if int32(value) < 0 {
		return windowsTaskHRESULT(uint32(value))
	}
	return nil
}
func (d *taskDispatch) release() {
	if d != nil {
		syscall.SyscallN(d.vtable[2], uintptr(unsafe.Pointer(d)))
	}
}
func taskClear(v *taskVariant) { taskOleAut32.NewProc("VariantClear").Call(uintptr(unsafe.Pointer(v))) }
func taskString(s string) (taskVariant, error) {
	p, e := windows.UTF16PtrFromString(s)
	if e != nil {
		return taskVariant{}, e
	}
	b, _, _ := taskOleAut32.NewProc("SysAllocString").Call(uintptr(unsafe.Pointer(p)))
	runtime.KeepAlive(p)
	if b == 0 && s != "" {
		return taskVariant{}, windows.ERROR_NOT_ENOUGH_MEMORY
	}
	return taskVariant{kind: 8, value: uint64(b)}, nil
}
func taskText(v taskVariant) (string, error) {
	if v.kind != 8 {
		return "", ErrOperationIdentity
	}
	if v.value == 0 {
		return "", nil
	}
	n, _, _ := taskOleAut32.NewProc("SysStringLen").Call(uintptr(v.value))
	if n > 1<<20 {
		return "", ErrOperationIdentity
	}
	return windows.UTF16ToString(unsafe.Slice((*uint16)(taskVariantPointer(&v)), int(n))), nil
}
func (d *taskDispatch) invoke(name string, flags uint16, args ...taskVariant) (taskVariant, error) {
	p, e := windows.UTF16PtrFromString(name)
	if e != nil {
		return taskVariant{}, e
	}
	var id int32
	var iid windows.GUID
	hr, _, _ := syscall.SyscallN(d.vtable[5], uintptr(unsafe.Pointer(d)), uintptr(unsafe.Pointer(&iid)), uintptr(unsafe.Pointer(&p)), 1, 0, uintptr(unsafe.Pointer(&id)))
	if e = taskHRESULT(hr); e != nil {
		return taskVariant{}, e
	}
	for left, right := 0, len(args)-1; left < right; left, right = left+1, right-1 {
		args[left], args[right] = args[right], args[left]
	}
	params := taskDispatchParameters{count: uint32(len(args))}
	if len(args) > 0 {
		params.args = &args[0]
	}
	var propertyPutID int32 = -3
	if flags&4 != 0 {
		params.named = &propertyPutID
		params.namedCount = 1
	}
	var result taskVariant
	var exception taskExceptionInfo
	var argument uint32
	hr, _, _ = syscall.SyscallN(d.vtable[6], uintptr(unsafe.Pointer(d)), uintptr(uint32(id)), uintptr(unsafe.Pointer(&iid)), 0, uintptr(flags), uintptr(unsafe.Pointer(&params)), uintptr(unsafe.Pointer(&result)), uintptr(unsafe.Pointer(&exception)), uintptr(unsafe.Pointer(&argument)))
	// Automation can defer populating its error strings and SCODE. Fill it
	// before reading or freeing the BSTRs, while the apartment is still alive.
	if uint32(hr) == 0x80020009 && exception.Deferred != 0 {
		syscall.SyscallN(exception.Deferred, uintptr(unsafe.Pointer(&exception)))
	}
	for _, b := range []uintptr{exception.Source, exception.Description, exception.HelpFile} {
		if b != 0 {
			taskOleAut32.NewProc("SysFreeString").Call(b)
		}
	}
	runtime.KeepAlive(args)
	runtime.KeepAlive(p)
	if uint32(hr) == 0x80020009 && exception.SCode < 0 {
		hr = uintptr(uint32(exception.SCode))
	}
	if e = taskHRESULT(hr); e != nil {
		taskClear(&result)
		return taskVariant{}, fmt.Errorf("%s: %w", name, e)
	}
	return result, nil
}
func (d *taskDispatch) object(name string, flags uint16, args ...taskVariant) (*taskDispatch, error) {
	v, e := d.invoke(name, flags, args...)
	if e != nil {
		return nil, e
	}
	if v.kind != 9 || v.value == 0 {
		taskClear(&v)
		return nil, ErrOperationIdentity
	}
	return (*taskDispatch)(taskVariantPointer(&v)), nil
}
func withTaskScheduler(fn func(*taskDispatch) error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := taskOle32.NewProc("CoInitializeEx").Call(0, 0)
	if e := taskHRESULT(hr); e != nil {
		return e
	}
	defer taskOle32.NewProc("CoUninitialize").Call()
	clsid := windows.GUID{Data1: 0x0f87369f, Data2: 0xa4e5, Data3: 0x4cfc, Data4: [8]byte{0xbd, 0x3e, 0x73, 0xe6, 0x15, 0x45, 0x72, 0xdd}}
	iid := windows.GUID{Data1: 0x00020400, Data4: [8]byte{0xc0, 0, 0, 0, 0, 0, 0, 0x46}}
	var service *taskDispatch
	hr, _, _ = taskOle32.NewProc("CoCreateInstance").Call(uintptr(unsafe.Pointer(&clsid)), 0, 1, uintptr(unsafe.Pointer(&iid)), uintptr(unsafe.Pointer(&service)))
	if e := taskHRESULT(hr); e != nil {
		return e
	}
	defer service.release()
	v, e := service.invoke("Connect", 1)
	if e != nil {
		return e
	}
	taskClear(&v)
	root, e := taskString(`\`)
	if e != nil {
		return e
	}
	defer taskClear(&root)
	folder, e := service.object("GetFolder", 1, root)
	if e != nil {
		return e
	}
	defer folder.release()
	return fn(folder)
}
func taskRegistered(folder *taskDispatch, name string) (*taskDispatch, error) {
	v, e := taskString(name)
	if e != nil {
		return nil, e
	}
	defer taskClear(&v)
	return folder.object("GetTask", 1, v)
}

// EnginePID identifies the task engine, never the Exec action process. Callers
// must bind a separately birth-locked action registration before using a PID.
func taskInstances(task *taskDispatch) ([]WindowsTaskInstance, error) {
	collection, e := task.object("GetInstances", 1, taskVariant{kind: 3})
	if e != nil {
		return nil, e
	}
	defer collection.release()
	count, e := collection.invoke("Count", 2)
	if e != nil {
		return nil, e
	}
	defer taskClear(&count)
	if count.kind != 3 || count.value > 4096 {
		return nil, ErrOperationIdentity
	}
	out := make([]WindowsTaskInstance, 0, int(count.value))
	for n := uint64(1); n <= count.value; n++ {
		item, e := collection.object("Item", 2, taskVariant{kind: 3, value: n})
		if e != nil {
			return nil, e
		}
		instance, e := func() (WindowsTaskInstance, error) {
			defer item.release()
			id, e := item.invoke("InstanceGuid", 2)
			if e != nil {
				return WindowsTaskInstance{}, e
			}
			defer taskClear(&id)
			guid, e := taskText(id)
			if e != nil || guid == "" {
				return WindowsTaskInstance{}, ErrOperationIdentity
			}
			pid, e := item.invoke("EnginePID", 2)
			if e != nil {
				return WindowsTaskInstance{}, e
			}
			defer taskClear(&pid)
			state, e := item.invoke("State", 2)
			if e != nil {
				return WindowsTaskInstance{}, e
			}
			defer taskClear(&state)
			if (pid.kind != 3 && pid.kind != 19) || state.kind != 3 || !validWindowsTaskState(int32(state.value)) {
				return WindowsTaskInstance{}, ErrOperationIdentity
			}
			return WindowsTaskInstance{guid, uint32(pid.value), int32(state.value)}, nil
		}()
		if e != nil {
			return nil, e
		}
		out = append(out, instance)
	}
	return out, nil
}
func taskNativeErrorMissing(e error) bool {
	var hr windowsTaskHRESULT
	return errors.As(e, &hr) && windowsTaskHRESULTMissing(uint32(hr))
}

// WindowsTaskStatus returns manager evidence, never an inferred action PID.
func WindowsTaskStatus(name string) (WindowsTaskSnapshot, error) {
	result := WindowsTaskSnapshot{Name: name}
	e := withTaskScheduler(func(folder *taskDispatch) error {
		task, e := taskRegistered(folder, name)
		if e != nil {
			return e
		}
		defer task.release()
		return readWindowsTaskSnapshot(task, &result)
	})
	return result, e
}
func readWindowsTaskSnapshot(task *taskDispatch, result *WindowsTaskSnapshot) error {
	v, e := task.invoke("Xml", 2)
	if e != nil {
		return e
	}
	result.XML, e = taskText(v)
	taskClear(&v)
	if e != nil {
		return e
	}
	sd, e := task.invoke("GetSecurityDescriptor", 1, taskVariant{kind: 3, value: 7})
	if e != nil {
		return e
	}
	result.SecurityDescriptor, e = taskText(sd)
	taskClear(&sd)
	if e != nil {
		return e
	}
	state, e := task.invoke("State", 2)
	if e != nil {
		return e
	}
	defer taskClear(&state)
	enabled, e := task.invoke("Enabled", 2)
	if e != nil {
		return e
	}
	defer taskClear(&enabled)
	if state.kind != 3 || enabled.kind != 11 || !validWindowsTaskState(int32(state.value)) {
		return ErrOperationIdentity
	}
	result.State = int32(state.value)
	result.Enabled = uint16(enabled.value) != 0
	result.Instances, e = taskInstances(task)
	return e
}

// WindowsStopTaskInstance stops only the recorded scheduler incarnation.
func WindowsStopTaskInstance(name, guid string) error {
	if guid == "" {
		return ErrOperationIdentity
	}
	return withTaskScheduler(func(folder *taskDispatch) error {
		task, e := taskRegistered(folder, name)
		if e != nil {
			return e
		}
		defer task.release()
		collection, e := task.object("GetInstances", 1, taskVariant{kind: 3})
		if e != nil {
			return e
		}
		defer collection.release()
		count, e := collection.invoke("Count", 2)
		if e != nil {
			return e
		}
		defer taskClear(&count)
		if count.kind != 3 || count.value > 4096 {
			return ErrOperationIdentity
		}
		for n := uint64(1); n <= count.value; n++ {
			item, e := collection.object("Item", 2, taskVariant{kind: 3, value: n})
			if e != nil {
				return e
			}
			matched, e := func() (bool, error) {
				defer item.release()
				v, e := item.invoke("InstanceGuid", 2)
				if e != nil {
					return false, e
				}
				observed, e := taskText(v)
				taskClear(&v)
				if e != nil {
					return false, e
				}
				if !strings.EqualFold(observed, guid) {
					return false, nil
				}
				v, e = item.invoke("Stop", 1)
				if e == nil {
					taskClear(&v)
				}
				return true, e
			}()
			if e != nil {
				return e
			}
			if matched {
				return nil
			}
		}
		return ErrOperationNotFound
	})
}
func windowsTaskControl(name, verb string) error {
	if verb == "stop" {
		return windowsStopRegisteredTask(name)
	}
	return withTaskScheduler(func(folder *taskDispatch) error {
		task, e := taskRegistered(folder, name)
		if e != nil {
			return e
		}
		defer task.release()
		if verb == "start" {
			instance, e := task.object("Run", 1, taskVariant{})
			if e != nil {
				return e
			}
			defer instance.release()
			return nil
		}
		return ErrUnsupported
	})
}
func windowsDeleteTask(name string) error {
	return withTaskScheduler(func(folder *taskDispatch) error {
		v, e := taskString(name)
		if e != nil {
			return e
		}
		defer taskClear(&v)
		result, e := folder.invoke("DeleteTask", 1, v, taskVariant{kind: 3})
		if e == nil {
			taskClear(&result)
		}
		return e
	})
}

// registerWindowsTask uses passwordless S4U/LUA and a root-protected definition.
func registerWindowsTask(name, content string) error {
	d, e := parseWindowsTaskDefinition(content)
	if e != nil {
		return e
	}
	sid, e := resolveWindowsTaskUserSID(d.Principals[0].UserID)
	if e != nil {
		return e
	}
	descriptor := "O:SYG:SYD:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;GRGX;;;" + sid.String() + ")"
	return withTaskScheduler(func(folder *taskDispatch) error {
		values := make([]taskVariant, 0, 4)
		defer func() {
			for i := range values {
				taskClear(&values[i])
			}
		}()
		for _, s := range []string{name, content, d.Principals[0].UserID, descriptor} {
			v, e := taskString(s)
			if e != nil {
				return e
			}
			values = append(values, v)
		}
		task, e := folder.object("RegisterTask", 1, values[0], values[1], taskVariant{kind: 3, value: 0x3e}, values[2], taskVariant{}, taskVariant{kind: 3, value: 2}, values[3])
		if e != nil {
			return e
		}
		defer task.release()
		var observed WindowsTaskSnapshot
		if e = readWindowsTaskSnapshot(task, &observed); e != nil {
			return e
		}
		parsed, e := parseWindowsTaskDefinition(observed.XML)
		if e != nil {
			return e
		}
		actualSID, e := resolveWindowsTaskUserSID(parsed.Principals[0].UserID)
		if e != nil {
			return e
		}
		if !windows.EqualSid(actualSID, sid) || parsed.Principals[0].LogonType != d.Principals[0].LogonType || parsed.Principals[0].RunLevel != d.Principals[0].RunLevel || parsed.Actions[0] != d.Actions[0] || parsed.Settings != d.Settings || !slices.Equal(parsed.Boot, d.Boot) || !slices.Equal(parsed.Time, d.Time) {
			return ErrOperationIdentity
		}
		if e = verifyWindowsTaskDescriptor(observed.SecurityDescriptor, sid); e != nil {
			return e
		}
		enabled, e := task.invoke("Enabled", 4, taskVariant{kind: 11, value: 0xffff})
		if e == nil {
			taskClear(&enabled)
		}
		return e
	})
}

func verifyWindowsTaskDescriptor(raw string, operator *windows.SID) error {
	sd, e := windows.SecurityDescriptorFromString(raw)
	if e != nil {
		return e
	}
	owner, _, e := sd.Owner()
	if e != nil {
		return e
	}
	if owner == nil || owner.String() != "S-1-5-18" {
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
	if acl == nil || acl.AceCount != 3 {
		return ErrOperationIdentity
	}
	seen := map[string]bool{}
	for i := uint32(0); i < uint32(acl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if e = windows.GetAce(acl, i, &ace); e != nil {
			return e
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || ace.Header.AceFlags != 0 {
			return ErrOperationIdentity
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart)).String()
		if seen[sid] {
			return ErrOperationIdentity
		}
		seen[sid] = true
		switch sid {
		case "S-1-5-18", "S-1-5-32-544":
			if ace.Mask != windows.GENERIC_ALL && ace.Mask != (windows.STANDARD_RIGHTS_REQUIRED|windows.SYNCHRONIZE|0x1ff) {
				return ErrOperationIdentity
			}
		case operator.String():
			if ace.Mask != windows.GENERIC_READ|windows.GENERIC_EXECUTE && ace.Mask != windows.FILE_GENERIC_READ|windows.FILE_GENERIC_EXECUTE {
				return ErrOperationIdentity
			}
		default:
			return ErrOperationIdentity
		}
	}
	return nil
}

func windowsSetTaskEnabled(name string, enabled bool) error {
	return withTaskScheduler(func(folder *taskDispatch) error {
		task, e := taskRegistered(folder, name)
		if e != nil {
			return e
		}
		defer task.release()
		var value uint64
		if enabled {
			value = 0xffff
		}
		v, e := task.invoke("Enabled", 4, taskVariant{kind: 11, value: value})
		if e == nil {
			taskClear(&v)
		}
		return e
	})
}

func resolveWindowsTaskUserSID(value string) (*windows.SID, error) {
	if strings.HasPrefix(value, "S-") {
		sid, e := windows.StringToSid(value)
		if e != nil || sid.String() != value {
			return nil, ErrOperationIdentity
		}
		_, _, kind, e := sid.LookupAccount("")
		if e != nil {
			return nil, e
		}
		if kind != windows.SidTypeUser {
			return nil, ErrOperationIdentity
		}
		return sid, nil
	}
	sid, _, kind, e := windows.LookupSID("", value)
	if e != nil {
		return nil, e
	}
	if kind != windows.SidTypeUser {
		return nil, ErrOperationIdentity
	}
	return sid, nil
}

// IRegisteredTask.Stop stops the registered task immediately, including all its
// instances. This is scoped to the verified UUID-specific task; it is distinct
// from a snapshot/list loop that could omit an admitted queued dispatch.
// IRegisteredTask is queried explicitly: IDispatch::Invoke may hide a method's
// nonnegative S_FALSE. Its native Stop returns S_FALSE when instances could not
// be stopped, which is not a cancellation acknowledgement. Vtable order/IID:
// Microsoft win32metadata generation/WinSDK/RecompiledIdlHeaders/um/taskschd.h.
type windowsRegisteredTask struct{ vtable *[25]uintptr }

func windowsStopRegisteredTask(name string) error {
	err := withTaskScheduler(func(folder *taskDispatch) error {
		task, e := taskRegistered(folder, name)
		if e != nil {
			return e
		}
		defer task.release()
		iid := windows.GUID{Data1: 0x9c86f320, Data2: 0xdee3, Data3: 0x4dd1, Data4: [8]byte{0xb9, 0x72, 0xa3, 0x03, 0xf2, 0x6b, 0x06, 0x1e}}
		var native *windowsRegisteredTask
		hr, _, _ := syscall.SyscallN(task.vtable[0], uintptr(unsafe.Pointer(task)), uintptr(unsafe.Pointer(&iid)), uintptr(unsafe.Pointer(&native)))
		if uint32(hr) != 0 {
			return windowsTaskHRESULT(uint32(hr))
		}
		if native == nil {
			return ErrOperationIdentity
		}
		defer syscall.SyscallN(native.vtable[2], uintptr(unsafe.Pointer(native)))
		// IUnknown(3), IDispatch(4), properties/Run/RunEx/GetInstances/...,
		// GetSecurityDescriptor and SetSecurityDescriptor place native Stop at23.
		hr, _, _ = syscall.SyscallN(native.vtable[23], uintptr(unsafe.Pointer(native)), 0)
		if !windowsTaskStopAcknowledged(uint32(hr)) {
			return windowsTaskHRESULT(uint32(hr))
		}
		return nil
	})
	var hr windowsTaskHRESULT
	if errors.As(err, &hr) && windowsTaskHRESULTNotRunning(uint32(hr)) {
		// No running instance does not exclude queued work. Only an independent
		// known quiet snapshot makes an already-inactive Stop idempotent.
		snapshot, queryErr := WindowsTaskStatus(name)
		if queryErr != nil {
			return queryErr
		}
		quiet, queryErr := windowsTaskQuiescent(snapshot)
		if queryErr != nil {
			return queryErr
		}
		if quiet {
			return nil
		}
	}
	return err
}
