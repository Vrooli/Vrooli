//go:build darwin && cgo

package collectors

import (
	"os"
	"testing"
)

func TestDarwinNativeCPUCounters(t *testing.T) {
	if os.Getenv("VROOLI_DARWIN_TELEMETRY_LIVE") != "1" {
		t.Skip("native host qualification requires VROOLI_DARWIN_TELEMETRY_LIVE=1")
	}
	cores, status, err := readDarwinCPUCounters()
	if err != nil || status != "measured" || len(cores) == 0 {
		t.Fatalf("Mach counters: %v %s %v", cores, status, err)
	}
	for i, core := range cores {
		if len(core) != 4 {
			t.Fatalf("cpu%d has %d states", i, len(core))
		}
	}
}
