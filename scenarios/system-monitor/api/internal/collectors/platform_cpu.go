package collectors

import "time"

// platformCPUState is deliberately shared by the platform implementations:
// CPU usage is a delta, so every implementation needs one previous sample.
// The platform-specific files only decide how the cumulative counters are
// obtained.
type platformCPUState struct {
	previous    []uint64
	previousAt  time.Time
	initialized bool
	perCore     map[string][]uint64
}

type platformCPUReading struct {
	usage      float64
	status     string
	reason     string
	provenance string
	values     map[string]interface{}
}

// deltaCPUValues applies one consistent definition on every platform: idle
// and iowait are capacity not doing runnable work; all other accounted modes
// are used time. The returned map contains mode percentages for diagnostics.
func deltaCPUValues(state *platformCPUState, counters []uint64, now time.Time, names []string, idleIndexes ...int) (float64, map[string]float64, bool) {
	if len(counters) == 0 || len(counters) != len(names) {
		return 0, nil, false
	}
	first := !state.initialized || len(state.previous) != len(counters)
	if first {
		state.previous = append(state.previous[:0], counters...)
		state.previousAt = now
		state.initialized = true
		return 0, nil, false
	}
	deltas := make([]uint64, len(counters))
	var total uint64
	for i, current := range counters {
		if current < state.previous[i] {
			state.previous = append(state.previous[:0], counters...)
			state.previousAt = now
			return 0, nil, false
		}
		deltas[i] = current - state.previous[i]
		total += deltas[i]
	}
	state.previous = append(state.previous[:0], counters...)
	state.previousAt = now
	if total == 0 {
		return 0, nil, false
	}
	modes := make(map[string]float64, len(names))
	idle := uint64(0)
	for _, i := range idleIndexes {
		if i >= 0 && i < len(deltas) {
			idle += deltas[i]
		}
	}
	for i, name := range names {
		modes[name] = float64(deltas[i]) * 100 / float64(total)
	}
	return float64(total-idle) * 100 / float64(total), modes, true
}

// platformCPUUnsupported is used by the fallback build and is also kept here
// as the single shape all native implementations return.
func platformCPUUnsupported(reason string) platformCPUReading {
	return platformCPUReading{status: "unsupported", reason: reason, provenance: "platform backend"}
}
