package darwinsys

import (
	"testing"
	"unsafe"
)

// The kernel copies HOST_VM_INFO64_COUNT natural_t words and libproc copies a
// fixed rusage_info_v0; a Go layout that drifts from the C layout would read
// every counter from the wrong offset.
func TestNativeStructLayoutsMatchMacOSHeaders(t *testing.T) {
	if got := unsafe.Sizeof(VMStatistics64{}); got != 152 {
		t.Fatalf("vm_statistics64 size = %d, want 152 (HOST_VM_INFO64_COUNT=38)", got)
	}
	if got := unsafe.Offsetof(VMStatistics64{}.Swapins); got != 112 {
		t.Fatalf("vm_statistics64.swapins offset = %d, want 112", got)
	}
	if got := unsafe.Sizeof(RusageInfoV0{}); got != 96 {
		t.Fatalf("rusage_info_v0 size = %d, want 96", got)
	}
	if got := unsafe.Offsetof(RusageInfoV0{}.PageIns); got != 48 {
		t.Fatalf("rusage_info_v0.ri_pageins offset = %d, want 48", got)
	}
}
