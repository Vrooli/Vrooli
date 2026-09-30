package registry

import (
	"sort"
	"sync"
	"time"
)

const defaultWindowSize = 32

type window struct {
	mu      sync.RWMutex
	size    int
	samples []Sample
	next    int
	full    bool
}

func newWindow(size int) *window {
	if size <= 0 {
		size = defaultWindowSize
	}
	return &window{size: size, samples: make([]Sample, 0, size)}
}

func (w *window) add(s Sample) RollingStats {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.samples) < w.size {
		w.samples = append(w.samples, s)
	} else {
		w.samples[w.next] = s
		w.full = true
	}
	w.next = (w.next + 1) % w.size
	return statsFromSamples(orderedSamples(w.samples, w.next, w.full))
}

func orderedSamples(samples []Sample, next int, full bool) []Sample {
	if len(samples) == 0 {
		return nil
	}
	if !full || next == len(samples) {
		out := make([]Sample, len(samples))
		copy(out, samples)
		return out
	}
	out := make([]Sample, 0, len(samples))
	out = append(out, samples[next:]...)
	out = append(out, samples[:next]...)
	return out
}

func statsFromSamples(samples []Sample) RollingStats {
	if len(samples) == 0 {
		return RollingStats{}
	}
	latencies := make([]time.Duration, 0, len(samples))
	var errors, degraded int
	var lastOK time.Time
	consecutiveOK := 0
	lastReason := samples[len(samples)-1].Reason
	for _, sample := range samples {
		latencies = append(latencies, sample.Latency)
		if !sample.OK {
			errors++
			consecutiveOK = 0
		} else {
			lastOK = sample.At
			consecutiveOK++
		}
		if sample.Degraded {
			degraded++
		}
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	count := len(samples)
	return RollingStats{
		LatencyP50:    percentile(latencies, 0.50),
		LatencyP95:    percentile(latencies, 0.95),
		ErrorRate:     float64(errors) / float64(count),
		DegradedRate:  float64(degraded) / float64(count),
		LastOKAt:      lastOK,
		SampleCount:   int64(count),
		ConsecutiveOK: consecutiveOK,
		LastReason:    lastReason,
	}
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	idx := int(float64(len(sorted)-1) * p)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
