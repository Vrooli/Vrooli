//go:build windows

package platform

import (
	"golang.org/x/sys/windows"
	"testing"
	"unsafe"
)

func TestTaskAutomationABIUsesFullNativeVariantAndReversedArguments(t *testing.T) {
	if unsafe.Sizeof(taskVariant{}) != 24 || unsafe.Offsetof(taskVariant{}.value) != 8 {
		t.Fatal("VARIANT/VARIANTARG native ABI drift")
	}
	var table [7]uintptr
	table[5] = windows.NewCallback(func(self, iid, names unsafe.Pointer, count, locale uintptr, result *int32) uintptr {
		*result = 7
		return 0
	})
	calls := 0
	table[6] = windows.NewCallback(func(self unsafe.Pointer, id uintptr, iid unsafe.Pointer, locale, flags uintptr, params *taskDispatchParameters, result *taskVariant, exception, argument unsafe.Pointer) uintptr {
		calls++
		if flags == 1 {
			if params.count != 2 {
				t.Error("argument count lost")
			} else {
				args := unsafe.Slice(params.args, 2)
				if args[0].value != 2 || args[1].value != 1 {
					t.Error("COM argument order changed")
				}
			}
		}
		if flags == 4 && (params.namedCount != 1 || params.named == nil || *params.named != -3) {
			t.Error("property-put named DISPID missing")
		}
		*result = taskVariant{kind: 3, value: 42}
		return 0
	})
	dispatch := taskDispatch{vtable: &table}
	result, e := dispatch.invoke("Call", 1, taskVariant{kind: 3, value: 1}, taskVariant{kind: 3, value: 2})
	if e != nil {
		t.Fatal(e)
	}
	if result.kind != 3 || result.value != 42 {
		t.Fatal("native result lost")
	}
	if _, e = dispatch.invoke("Enabled", 4, taskVariant{kind: 11, value: 0xffff}); e != nil {
		t.Fatal(e)
	}
	if calls != 2 {
		t.Fatal("native dispatcher not invoked")
	}
}

func TestTaskNativeDescriptorAllowsOnlySetupAndSelectedOperator(t *testing.T) {
	operator, e := windows.StringToSid("S-1-5-21-100-200-300-1001")
	if e != nil {
		t.Fatal(e)
	}
	good := "O:SYG:SYD:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;GRGX;;;" + operator.String() + ")"
	if e = verifyWindowsTaskDescriptor(good, operator); e != nil {
		t.Fatal(e)
	}
	for _, raw := range []string{
		"O:BAG:BAD:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;GRGX;;;" + operator.String() + ")",
		"O:SYG:SYD:(A;;GA;;;SY)(A;;GA;;;BA)(A;;GRGX;;;" + operator.String() + ")",
		"O:SYG:SYD:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;GA;;;" + operator.String() + ")",
		good + "(A;;GRGX;;;WD)",
	} {
		if e = verifyWindowsTaskDescriptor(raw, operator); e == nil {
			t.Fatal("widened native definition authority accepted")
		}
	}
}

func TestTaskAutomationFillsDeferredExceptionBeforeReturningHRESULT(t *testing.T) {
	var table [7]uintptr
	table[5] = windows.NewCallback(func(self, iid, names unsafe.Pointer, count, locale uintptr, result *int32) uintptr {
		*result = 7
		return 0
	})
	filled := false
	deferred := windows.NewCallback(func(info *taskExceptionInfo) uintptr {
		filled = true
		info.SCode = -2147024894 // HRESULT_FROM_WIN32(ERROR_FILE_NOT_FOUND)
		return 0
	})
	table[6] = windows.NewCallback(func(self unsafe.Pointer, id uintptr, iid unsafe.Pointer, locale, flags uintptr, params *taskDispatchParameters, result *taskVariant, exception *taskExceptionInfo, argument unsafe.Pointer) uintptr {
		exception.Deferred = deferred
		return 0x80020009 // DISP_E_EXCEPTION
	})
	dispatch := taskDispatch{vtable: &table}
	_, e := dispatch.invoke("GetTask", 1)
	if !filled || !taskNativeErrorMissing(e) {
		t.Fatalf("deferred missing-task HRESULT lost: filled=%v err=%v", filled, e)
	}
	if unsafe.Sizeof(taskExceptionInfo{}) != 64 || unsafe.Offsetof(taskExceptionInfo{}.SCode) != 56 {
		t.Fatal("EXCEPINFO ABI drift")
	}
}
