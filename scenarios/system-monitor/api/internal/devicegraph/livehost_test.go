//go:build linux

package devicegraph

import (
	"context"
	"encoding/json"
	"os"
	"testing"
)

// TestLiveHostDeviceGraph is an opt-in observation of the machine the suite is
// running on. It is skipped unless DEVICEGRAPH_LIVE_HOST is set, so the suite
// stays hermetic; it exists so the collectors can be exercised against real
// hardware without running a scenario binary.
func TestLiveHostDeviceGraph(t *testing.T) {
	if os.Getenv("DEVICEGRAPH_LIVE_HOST") == "" {
		t.Skip("set DEVICEGRAPH_LIVE_HOST to observe the running machine")
	}
	graph := Collect(context.Background(), Env{})
	if err := graph.Validate(); err != nil {
		t.Fatalf("live graph failed its invariants: %v", err)
	}
	encoded, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if path := os.Getenv("DEVICEGRAPH_LIVE_OUT"); path != "" {
		if err := os.WriteFile(path, encoded, 0o644); err != nil {
			t.Fatalf("write live graph: %v", err)
		}
	}
	t.Logf("devices=%d subsystems=%d virtual_interfaces=%d",
		len(graph.Devices), len(graph.Subsystems), len(graph.VirtualNetworkInterfaces))
}
