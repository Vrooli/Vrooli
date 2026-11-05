package hostinventory

import (
	"context"
	"os"
	"testing"
)

// TestLiveAcceleratorFactsOnThisHost prints the accelerator facts this host
// actually reports. It is evidence capture, not an assertion: it is skipped
// unless VROOLI_HOSTINVENTORY_LIVE is set, so it never fails on a host without
// an accelerator.
func TestLiveAcceleratorFactsOnThisHost(t *testing.T) {
	if os.Getenv("VROOLI_HOSTINVENTORY_LIVE") == "" {
		t.Skip("set VROOLI_HOSTINVENTORY_LIVE to capture live accelerator facts")
	}
	snap, err := SystemCollector().CollectGPUFacts(context.Background())
	if err != nil {
		t.Fatalf("CollectGPUFacts: %v", err)
	}
	for _, line := range snap.AcceleratorFactSummary() {
		t.Log(line)
	}
	t.Logf("rocm nodes=%v vulkan icds=%d probe statuses=%v", snap.ROCmDeviceNodes, len(snap.VulkanICDs), snap.ProbeStatuses)
}
