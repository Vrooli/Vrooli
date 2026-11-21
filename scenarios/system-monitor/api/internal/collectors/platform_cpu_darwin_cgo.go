//go:build darwin && cgo

package collectors

/*
#include <mach/mach.h>
#include <mach/processor_info.h>

// mach_task_self can be a function-like SDK macro; keep it inside C.
static mach_port_t vrooli_task_self(void) { return mach_task_self(); }
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// readDarwinCPUCounters owns both Mach allocations: the host send right and
// the out-of-line processor-info array. Copy into Go memory before releasing.
func readDarwinCPUCounters() ([][]uint64, string, error) {
	host := C.mach_host_self()
	defer C.mach_port_deallocate(C.vrooli_task_self(), host)
	var count C.natural_t
	var words C.mach_msg_type_number_t
	var info C.processor_info_array_t
	result := C.host_processor_info(host, C.PROCESSOR_CPU_LOAD_INFO, &count, &info, &words)
	if result != C.KERN_SUCCESS {
		return nil, "failed", fmt.Errorf("host_processor_info: Mach error %d", result)
	}
	defer C.vm_deallocate(C.vrooli_task_self(), C.vm_address_t(uintptr(unsafe.Pointer(info))), C.vm_size_t(words)*C.vm_size_t(C.sizeof_integer_t))
	if count == 0 || uint64(words) != uint64(count)*uint64(C.CPU_STATE_MAX) || info == nil {
		return nil, "failed", fmt.Errorf("host_processor_info returned invalid processor/count shape: %d/%d", count, words)
	}
	raw := unsafe.Slice(info, int(words))
	cores := make([][]uint64, int(count))
	for i := range cores {
		offset := i * int(C.CPU_STATE_MAX)
		cores[i] = []uint64{uint64(uint32(raw[offset+int(C.CPU_STATE_USER)])), uint64(uint32(raw[offset+int(C.CPU_STATE_SYSTEM)])), uint64(uint32(raw[offset+int(C.CPU_STATE_IDLE)])), uint64(uint32(raw[offset+int(C.CPU_STATE_NICE)]))}
	}
	return cores, "measured", nil
}
