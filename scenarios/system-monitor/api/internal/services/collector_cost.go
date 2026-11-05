package services

import (
	"fmt"
	"sort"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// CollectionProfile makes the host-cost tradeoff visible in health and
// self-metrics. Low-power hosts collect the same metric families more slowly
// and skip optional GPU probing.
type CollectionProfile string

const (
	CollectionProfileStandard CollectionProfile = "standard"
	CollectionProfileLowPower CollectionProfile = "low-power"
)

// CollectionProfileForHost classifies hosts below the plan's low-power
// thresholds without pretending that a missing inventory probe is a signal.
func CollectionProfileForHost(snapshot hostinventory.Snapshot) CollectionProfile {
	if (snapshot.CPU.Cores > 0 && snapshot.CPU.Cores < 4) ||
		(snapshot.Memory.TotalBytes > 0 && snapshot.Memory.TotalBytes < 2*1024*1024*1024) {
		return CollectionProfileLowPower
	}
	return CollectionProfileStandard
}

// CollectorCostBudget is the declared upper bound for one collector
// invocation. Runtime telemetry reports when an invocation exceeds it.
type CollectorCostBudget struct {
	MaxDuration time.Duration
	MaxForks    uint64
}

// Adding a default collector without adding a budget is a test failure. This
// keeps a new probe from silently consuming the monitor's host headroom.
var collectorCostBudgets = map[string]CollectorCostBudget{
	"cpu":      {MaxDuration: 250 * time.Millisecond, MaxForks: 0},
	"memory":   {MaxDuration: 250 * time.Millisecond, MaxForks: 0},
	"network":  {MaxDuration: 500 * time.Millisecond, MaxForks: 4},
	"disk":     {MaxDuration: 750 * time.Millisecond, MaxForks: 4},
	"process":  {MaxDuration: 500 * time.Millisecond, MaxForks: 0},
	"pressure": {MaxDuration: 250 * time.Millisecond, MaxForks: 4},
	"gpu":      {MaxDuration: 1 * time.Second, MaxForks: 4},
}

const collectorCycleHeadroom = 0.50

// CollectorCostBudgets returns a copy so callers cannot mutate the policy.
func CollectorCostBudgets() map[string]CollectorCostBudget {
	copyOf := make(map[string]CollectorCostBudget, len(collectorCostBudgets))
	for name, budget := range collectorCostBudgets {
		copyOf[name] = budget
	}
	return copyOf
}

// ValidateCollectorCostBudgets verifies that every registered collector has a
// declared budget and that the declared cycle leaves host headroom.
func ValidateCollectorCostBudgets(names []string, interval time.Duration) error {
	if interval <= 0 {
		return fmt.Errorf("collector interval must be positive")
	}
	missing := make([]string, 0)
	var declared time.Duration
	for _, name := range names {
		budget, ok := collectorCostBudgets[name]
		if !ok {
			missing = append(missing, name)
			continue
		}
		declared += budget.MaxDuration
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("collectors missing cost budgets: %v", missing)
	}
	if declared > time.Duration(float64(interval)*collectorCycleHeadroom) {
		return fmt.Errorf("declared collector cost %s exceeds %.0f%% of interval %s", declared, collectorCycleHeadroom*100, interval)
	}
	return nil
}

func collectorBudget(name string) (CollectorCostBudget, bool) {
	budget, ok := collectorCostBudgets[name]
	return budget, ok
}
