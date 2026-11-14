//go:build darwin && cgo

package hostinventory

/*
#include <mach/mach.h>
#include <mach/mach_host.h>
#include <mach/host_info.h>
#include <mach/vm_statistics.h>
#include <stdint.h>
#include <string.h>
static kern_return_t vrooli_host_memory(uint64_t *page, uint64_t *free_pages, uint64_t *inactive_pages) {
 mach_port_t host=mach_host_self();
 vm_size_t page_size=0;
 kern_return_t result=host_page_size(host,&page_size);
 vm_statistics64_data_t stats; memset(&stats,0,sizeof(stats));
 // REV0 contains the counters we consume and avoids future-SDK structure
 // extensions requiring a newer kernel than the supported deployment target.
 mach_msg_type_number_t count=HOST_VM_INFO64_REV0_COUNT;
 if(result==KERN_SUCCESS) result=host_statistics64(host,HOST_VM_INFO64,(host_info64_t)&stats,&count);
 mach_port_deallocate(mach_task_self(),host);
 if(result!=KERN_SUCCESS) return result;
 if(count<HOST_VM_INFO64_REV0_COUNT) return KERN_FAILURE;
 *page=(uint64_t)page_size; *free_pages=(uint64_t)stats.free_count; *inactive_pages=(uint64_t)stats.inactive_count;
 return KERN_SUCCESS;
}
*/
import "C"
import (
	"fmt"
	"golang.org/x/sys/unix"
)

func hostMemoryFacts() (HostMemory, error) {
	total, err := unix.SysctlUint64("hw.memsize")
	if err != nil {
		return HostMemory{}, err
	}
	var page, free, inactive C.uint64_t
	if code := C.vrooli_host_memory(&page, &free, &inactive); code != C.KERN_SUCCESS {
		return HostMemory{}, fmt.Errorf("native Mach memory counters unavailable (%d)", code)
	}
	// Free includes speculative pages in the Mach ABI: adding speculative again
	// overstates availability. Inactive pages retain the existing reclaimable
	// estimate; the capacity engine still applies its reserve and ledger budget.
	return hostMemoryFromDarwinPages(total, uint64(page), uint64(free), uint64(inactive))
}
