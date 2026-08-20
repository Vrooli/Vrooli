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
}

type platformCPUReading struct {
	usage           float64
	status          string
	reason          string
	provenance      string
	loadAverage     []float64
	contextSwitches *int64
}

func deltaCPUUsage(state *platformCPUState, counters []uint64, now time.Time, idleIndexes ...int) (float64, bool) {
	if len(counters) == 0 {
		return 0, false
	}
	first := !state.initialized || len(state.previous) != len(counters)
	if first {
		state.previous = append(state.previous[:0], counters...)
		state.previousAt = now
		state.initialized = true
		return 0, false
	}

	var totalDelta uint64
	var idleDelta uint64
	for i, current := range counters {
		if current < state.previous[i] {
			state.previous = append(state.previous[:0], counters...)
			state.previousAt = now
			return 0, false
		}
		delta := current - state.previous[i]
		totalDelta += delta
		for _, idleIndex := range idleIndexes {
			if i == idleIndex {
				idleDelta += delta
			}
		}
	}
	state.previous = append(state.previous[:0], counters...)
	state.previousAt = now
	if totalDelta == 0 {
		return 0, true
	}
	used := totalDelta - idleDelta
	return float64(used) / float64(totalDelta) * 100, true
}

// platformCPUUnsupported is used by the fallback build and is also kept here
// as the single shape all native implementations return.
func platformCPUUnsupported(reason string) platformCPUReading {
	return platformCPUReading{status: "unsupported", reason: reason, provenance: "platform backend"}
}
