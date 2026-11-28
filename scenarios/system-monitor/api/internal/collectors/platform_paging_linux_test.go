//go:build linux

package collectors

import (
	"os"
	"testing"
)

func TestReadPlatformPagingUsesTheKernelPageSize(t *testing.T) {
	vmstat := map[string]uint64{"pswpin": 10, "pswpout": 4, "pgmajfault": 7}

	got := readPlatformPaging(vmstat)

	if !got.supported || got.provenance != "/proc/vmstat" {
		t.Fatalf("reading supported=%v provenance=%q, want a supported /proc/vmstat reading", got.supported, got.provenance)
	}
	if want := uint64(os.Getpagesize()); got.pageSize != want {
		t.Fatalf("page size = %d, want the kernel page size %d so swap pages convert to bytes correctly", got.pageSize, want)
	}
	if got.counters["pswpin"] != 10 || got.counters["pswpout"] != 4 {
		t.Fatalf("counters = %v, want the vmstat swap counters passed through", got.counters)
	}
}
