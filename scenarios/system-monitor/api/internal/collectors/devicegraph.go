package collectors

import (
	"context"
	"sync"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/devicegraph"
)

// DeviceGraphProvider returns the graded hardware device graph. It mirrors the
// SnapshotProvider seam: the device graph is one host probe per cycle, shared
// by every consumer, rather than a second uncached walk per reader.
type DeviceGraphProvider interface {
	DeviceGraph(ctx context.Context) devicegraph.Graph
}

// CachedDeviceGraphProvider memoizes the most recent device graph for a short
// TTL and owns the trend tracker. Trends belong to the observation, not to a
// consumer: folding them in here means every reader sees the same rung-five
// grades and no reader can double-count a sample.
type CachedDeviceGraphProvider struct {
	ttl     time.Duration
	env     devicegraph.Env
	collect func(ctx context.Context, env devicegraph.Env) devicegraph.Graph
	trends  *devicegraph.TrendTracker
	now     func() time.Time

	mu       sync.Mutex
	cached   devicegraph.Graph
	cachedAt time.Time
	hasValue bool
}

// NewCachedDeviceGraphProvider builds a provider backed by the real host. A
// non-positive ttl falls back to 30s: enumerating buses, disks and sensors is
// heavier than reading /proc, and the topology changes far more slowly than
// utilization does.
func NewCachedDeviceGraphProvider(ttl time.Duration) *CachedDeviceGraphProvider {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &CachedDeviceGraphProvider{
		ttl:     ttl,
		collect: devicegraph.Collect,
		trends:  devicegraph.NewTrendTracker(),
		now:     time.Now,
	}
}

// DeviceGraph returns a cached graph when fresh, otherwise walks the host once.
func (p *CachedDeviceGraphProvider) DeviceGraph(ctx context.Context) devicegraph.Graph {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.hasValue && p.now().Sub(p.cachedAt) < p.ttl {
		return p.cached
	}
	graph := p.collect(ctx, p.env)
	p.trends.Observe(&graph)
	p.cached = graph
	p.cachedAt = p.now()
	p.hasValue = true
	return graph
}

// DeviceGraphCollector publishes the graded device graph as a metric payload.
// Every device carries a state for all five ladder rungs, so a consumer can
// always distinguish "this part is healthy" from "this part cannot be read".
type DeviceGraphCollector struct {
	BaseCollector
	graphs DeviceGraphProvider
}

// NewDeviceGraphCollector constructs the collector with its own provider. Wire
// a shared provider with SetDeviceGraphProvider so the API and the collection
// loop reuse one host walk.
func NewDeviceGraphCollector() *DeviceGraphCollector {
	return &DeviceGraphCollector{
		BaseCollector: NewBaseCollector("device_graph", 60*time.Second),
		graphs:        NewCachedDeviceGraphProvider(0),
	}
}

// SetDeviceGraphProvider injects the shared device-graph provider.
func (c *DeviceGraphCollector) SetDeviceGraphProvider(provider DeviceGraphProvider) {
	if provider != nil {
		c.graphs = provider
	}
}

// Collect returns the current device graph together with its ladder readout.
func (c *DeviceGraphCollector) Collect(ctx context.Context) (*MetricData, error) {
	graph := c.graphs.DeviceGraph(ctx)

	rungCounts := make(map[string]map[string]int, len(devicegraph.Rungs))
	for rung, states := range graph.RungCounts() {
		converted := make(map[string]int, len(states))
		for state, count := range states {
			converted[string(state)] = count
		}
		rungCounts[string(rung)] = converted
	}

	values := map[string]interface{}{
		"platform":                   graph.Platform,
		"devices":                    graph.Devices,
		"subsystems":                 graph.Subsystems,
		"virtual_network_interfaces": graph.VirtualNetworkInterfaces,
		"device_count":               len(graph.Devices),
		"rung_counts":                rungCounts,
	}
	if len(graph.Warnings) > 0 {
		values["warnings"] = graph.Warnings
	}
	// A graph that graded nothing is reported as a failure to observe, not as
	// a host with no hardware.
	if len(graph.Devices) == 0 && len(graph.Subsystems) == 0 {
		values["status"] = "failed"
		values["reason"] = "the device-graph walk produced neither a device nor a graded subsystem"
	} else {
		values["status"] = "measured"
	}
	if err := graph.Validate(); err != nil {
		values["status"] = "failed"
		values["reason"] = "device graph failed its structural invariants: " + err.Error()
	}

	return &MetricData{
		CollectorName: c.GetName(),
		Timestamp:     graph.CollectedAt,
		Type:          "device_graph",
		Values:        values,
		Tags: map[string]string{
			"os":     collectorOS,
			"source": "device graph: sysfs, bus enumeration and SMART",
		},
	}, nil
}
