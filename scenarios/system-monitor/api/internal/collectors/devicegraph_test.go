package collectors

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/devicegraph"
)

type stubDeviceGraphProvider struct {
	graph devicegraph.Graph
	calls int
}

func (s *stubDeviceGraphProvider) DeviceGraph(context.Context) devicegraph.Graph {
	s.calls++
	return s.graph
}

func gradedDevice(id string, class devicegraph.Class, telemetry devicegraph.State) devicegraph.Device {
	rungs := map[devicegraph.Rung]devicegraph.RungState{}
	for _, rung := range devicegraph.Rungs {
		state := devicegraph.RungState{Rung: rung, State: devicegraph.StateMeasured}
		if rung == devicegraph.RungTelemetry && telemetry != devicegraph.StateMeasured {
			state.State = telemetry
			state.Reason = "the fixture host refused this reading"
		}
		rungs[rung] = state
	}
	return devicegraph.Device{ID: id, Class: class, Rungs: rungs}
}

func TestDeviceGraphCollectorPublishesRungCounts(t *testing.T) {
	provider := &stubDeviceGraphProvider{graph: devicegraph.Graph{
		CollectedAt: time.Unix(1_700_000_000, 0).UTC(),
		Platform:    "linux",
		Devices: []devicegraph.Device{
			gradedDevice("block:nvme0n1", devicegraph.ClassBlockDevice, devicegraph.StateMeasured),
			gradedDevice("block:sdb", devicegraph.ClassBlockDevice, devicegraph.StateUnmeasurable),
		},
		VirtualNetworkInterfaces: []string{"docker0"},
	}}
	collector := NewDeviceGraphCollector()
	collector.SetDeviceGraphProvider(provider)

	data, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if data.Values["status"] != "measured" {
		t.Fatalf("status = %v, want measured", data.Values["status"])
	}
	if data.Values["device_count"] != 2 {
		t.Errorf("device_count = %v, want 2", data.Values["device_count"])
	}
	counts, ok := data.Values["rung_counts"].(map[string]map[string]int)
	if !ok {
		t.Fatalf("rung_counts has type %T", data.Values["rung_counts"])
	}
	telemetry := counts[string(devicegraph.RungTelemetry)]
	if telemetry["measured"] != 1 || telemetry["unmeasurable"] != 1 {
		t.Errorf("telemetry counts = %v, want one measured and one unmeasurable", telemetry)
	}
	if !data.Timestamp.Equal(provider.graph.CollectedAt) {
		t.Errorf("timestamp = %v, want the observation time", data.Timestamp)
	}
}

// A graph that graded nothing is a failure to observe the host, not a host
// with no hardware.
func TestDeviceGraphCollectorReportsAnEmptyGraphAsFailed(t *testing.T) {
	collector := NewDeviceGraphCollector()
	collector.SetDeviceGraphProvider(&stubDeviceGraphProvider{})

	data, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if data.Values["status"] != "failed" {
		t.Fatalf("status = %v, want failed", data.Values["status"])
	}
	if data.Values["reason"] == "" || data.Values["reason"] == nil {
		t.Error("a failed device graph must carry a reason")
	}
}

// A graph that breaks its own invariants is reported as failed rather than
// published as if it were sound.
func TestDeviceGraphCollectorReportsInvalidGraphsAsFailed(t *testing.T) {
	broken := gradedDevice("block:sda", devicegraph.ClassBlockDevice, devicegraph.StateMeasured)
	delete(broken.Rungs, devicegraph.RungControl)
	collector := NewDeviceGraphCollector()
	collector.SetDeviceGraphProvider(&stubDeviceGraphProvider{graph: devicegraph.Graph{
		Devices: []devicegraph.Device{broken},
	}})

	data, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if data.Values["status"] != "failed" {
		t.Fatalf("status = %v, want failed", data.Values["status"])
	}
}

// One collection cycle walks the host once: the cached provider is the single
// probe path, exactly like the host-snapshot seam.
func TestCachedDeviceGraphProviderProbesOncePerTTL(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	walks := 0
	provider := NewCachedDeviceGraphProvider(30 * time.Second)
	provider.now = func() time.Time { return now }
	provider.collect = func(context.Context, devicegraph.Env) devicegraph.Graph {
		walks++
		return devicegraph.Graph{CollectedAt: now, Devices: []devicegraph.Device{
			gradedDevice("block:sda", devicegraph.ClassBlockDevice, devicegraph.StateMeasured),
		}}
	}

	for i := 0; i < 4; i++ {
		provider.DeviceGraph(context.Background())
	}
	if walks != 1 {
		t.Fatalf("host walks = %d, want 1 within the TTL", walks)
	}

	now = now.Add(31 * time.Second)
	provider.DeviceGraph(context.Background())
	if walks != 2 {
		t.Fatalf("host walks = %d, want a second walk after the TTL expired", walks)
	}
}
