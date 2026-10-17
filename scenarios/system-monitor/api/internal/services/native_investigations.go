package services

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sort"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/collectors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/procsampler"
)

// NativeInvestigationRunner is the typed-query seam used by investigations
// whose facts already exist in the monitor's collectors. It intentionally
// returns JSON because the existing investigation result contract is JSON,
// while the data behind it is now typed and portable.
type NativeInvestigationRunner interface {
	RunNative(ctx context.Context, query string) ([]byte, error)
}

type NativeInvestigator struct {
	sampler procsampler.Sampler
}

func NewNativeInvestigator() *NativeInvestigator {
	return &NativeInvestigator{sampler: procsampler.NewCachedSampler(procsampler.NewSampler(), 500_000_000)}
}

func (n *NativeInvestigator) RunNative(ctx context.Context, query string) ([]byte, error) {
	switch query {
	case "cpu":
		collector := collectors.NewCPUCollector()
		data, err := collector.Collect(ctx)
		return marshalNative(data, err)
	case "memory":
		data, err := collectors.NewMemoryCollector().Collect(ctx)
		return marshalNative(data, err)
	case "network":
		data, err := collectors.NewNetworkCollector().Collect(ctx)
		return marshalNative(data, err)
	case "disk":
		data, err := collectors.NewDiskCollector().Collect(ctx)
		return marshalNative(data, err)
	case "process-genealogy":
		return n.processGenealogy(ctx)
	case "zombies":
		return n.zombies(ctx)
	case "service-health", "service-config":
		return marshalNative(map[string]any{
			"investigation": query,
			"platform":      runtime.GOOS,
			"status":        "native backend available; service detail is reported by the platform service manager",
		}, nil)
	case "network-anomalies", "resource-leaks", "processes", "system":
		data, err := collectors.NewProcessCollector().Collect(ctx)
		return marshalNative(map[string]any{"investigation": query, "data": data}, err)
	default:
		return nil, fmt.Errorf("unknown native investigation query %q", query)
	}
}

func marshalNative(value any, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{
		"execution_mode": "native",
		"data":           value,
	})
}

func (n *NativeInvestigator) processGenealogy(ctx context.Context) ([]byte, error) {
	samples, err := n.sampler.Sample(ctx)
	if err != nil {
		return nil, err
	}
	children := make(map[int][]map[string]any)
	for _, sample := range samples {
		children[sample.PPID] = append(children[sample.PPID], map[string]any{
			"pid": sample.PID, "name": sample.Comm, "threads": sample.Threads,
		})
	}
	parents := make([]int, 0, len(children))
	for parent := range children {
		parents = append(parents, parent)
	}
	sort.Ints(parents)
	families := make([]map[string]any, 0, len(parents))
	for _, parent := range parents {
		family := children[parent]
		if len(family) < 2 {
			continue
		}
		families = append(families, map[string]any{"parent_pid": parent, "children": family})
	}
	return json.Marshal(map[string]any{
		"execution_mode":   "native",
		"investigation":    "process-genealogy",
		"process_families": families,
		"sample_count":     len(samples),
	})
}

func (n *NativeInvestigator) zombies(ctx context.Context) ([]byte, error) {
	samples, err := n.sampler.Sample(ctx)
	if err != nil {
		return nil, err
	}
	zombies := make([]map[string]any, 0)
	for _, sample := range samples {
		if sample.State != "Z" && sample.State != "z" {
			continue
		}
		zombies = append(zombies, map[string]any{
			"pid": sample.PID, "name": sample.Comm, "parent_pid": sample.PPID,
		})
	}
	return json.Marshal(map[string]any{
		"execution_mode":   "native",
		"investigation":    "zombies",
		"zombie_processes": zombies,
		"count":            len(zombies),
	})
}
