// Package performance provides types and utilities for debug performance mode.
package performance

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// frameSample keeps the local observation interval separate from the driver's clock.
type frameSample struct {
	timing FrameTimings
	start  time.Time
}

// Collector owns a fixed-capacity sample ring. Lifetime count is only for cadence.
// All access is synchronized; caller-provided timing values are copied.
type Collector struct {
	sessionID    string
	targetFps    int
	mu           sync.RWMutex
	samples      []frameSample
	next         int
	frameCount   int
	lastRecorded time.Time
	now          func() time.Time
}

func NewCollector(sessionID string, targetFps, bufferSize int) *Collector {
	return &Collector{
		sessionID: sessionID, targetFps: targetFps,
		samples:      make([]frameSample, 0, max(1, bufferSize)),
		lastRecorded: time.Now(), now: time.Now,
	}
}

func (c *Collector) Record(t *FrameTimings) {
	c.mu.Lock()
	defer c.mu.Unlock()
	sample := frameSample{timing: *t, start: c.lastRecorded}
	c.lastRecorded = c.now()
	if len(c.samples) < cap(c.samples) {
		c.samples = append(c.samples, sample)
	} else {
		c.samples[c.next] = sample
	}
	c.next = (c.next + 1) % cap(c.samples)
	c.frameCount++
}

// GetFrameCount returns the lifetime count, including evicted samples.
func (c *Collector) GetFrameCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.frameCount
}

func (c *Collector) ShouldBroadcast() bool {
	count := c.GetFrameCount()
	return count > 0 && count%60 == 0
}

// GetAggregated uses the same retained window for every count, size and rate.
// The interval begins before its first sample and includes subsequent idle time.
func (c *Collector) GetAggregated() FrameStatsAggregated {
	c.mu.RLock()
	defer c.mu.RUnlock()
	start := c.lastRecorded
	if len(c.samples) > 0 {
		start = c.samples[c.next%len(c.samples)].start
	}
	duration := max(time.Duration(0), c.now().Sub(start))
	stats := FrameStatsAggregated{
		SessionID: c.sessionID, WindowStartTime: start, WindowDurationMs: duration.Milliseconds(),
		FrameCount: len(c.samples), TargetFps: c.targetFps, PrimaryBottleneck: BottleneckNone,
		BottleneckDescription: "No frames recorded yet",
	}
	if len(c.samples) == 0 {
		return stats
	}
	captureTimes := make([]float64, 0, len(c.samples))
	processingTimes := make([]float64, 0, len(c.samples))
	var totalBytes int64
	for _, sample := range c.samples {
		t := sample.timing
		captureTimes = append(captureTimes, t.DriverCaptureMs)
		if t.Skipped {
			stats.SkippedCount++
		} else {
			processingTimes = append(processingTimes, t.DriverTotalMs+t.APITotalMs)
			totalBytes += int64(t.FrameBytes)
		}
	}
	sort.Float64s(captureTimes)
	sort.Float64s(processingTimes)
	stats.CaptureP50Ms = round2(percentile(captureTimes, .5))
	stats.CaptureP90Ms = round2(percentile(captureTimes, .9))
	stats.CaptureP99Ms = round2(percentile(captureTimes, .99))
	stats.CaptureMaxMs = round2(percentile(captureTimes, 1))
	stats.E2EP50Ms = round2(percentile(processingTimes, .5))
	stats.E2EP90Ms = round2(percentile(processingTimes, .9))
	stats.E2EP99Ms = round2(percentile(processingTimes, .99))
	stats.E2EMaxMs = round2(percentile(processingTimes, 1))
	delivered := len(processingTimes)
	if delivered > 0 {
		stats.AvgFrameBytes = int(totalBytes / int64(delivered))
	}
	if duration > 0 {
		stats.ActualFps = round2(float64(delivered) / duration.Seconds())
		stats.BandwidthBytesPerSec = int64(float64(totalBytes) / duration.Seconds())
	}
	stats.PrimaryBottleneck, stats.BottleneckDescription = identifyBottleneck(
		stats.CaptureP50Ms, stats.CaptureP90Ms, stats.E2EP90Ms, c.targetFps)
	return stats
}

// GetRecentFrames returns detached values in chronological order.
func (c *Collector) GetRecentFrames(limit int) []FrameTimings {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if limit <= 0 || limit > len(c.samples) {
		limit = len(c.samples)
	}
	result := make([]FrameTimings, limit)
	for i := range result {
		result[i] = c.samples[(c.next+len(c.samples)-limit+i)%len(c.samples)].timing
	}
	return result
}

func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	clear(c.samples)
	c.samples = c.samples[:0]
	c.next = 0
	c.frameCount = 0
	c.lastRecorded = c.now()
}

// percentile calculates the p-th percentile of a sorted slice.
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}

	index := p * float64(len(sorted)-1)
	lower := int(index)
	upper := lower + 1
	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}

	fraction := index - float64(lower)
	return sorted[lower]*(1-fraction) + sorted[upper]*fraction
}

// round2 rounds to 2 decimal places.
func round2(n float64) float64 {
	return float64(int(n*100+0.5)) / 100
}

// identifyBottleneck determines the primary bottleneck based on timing data.
func identifyBottleneck(captureP50, captureP90, e2eP90 float64, targetFps int) (BottleneckType, string) {
	targetFrameTime := 1000.0 / float64(targetFps)

	// If capture P90 > 80% of target frame time, capture is the bottleneck
	if captureP90 > targetFrameTime*0.8 {
		return BottleneckCapture, fmt.Sprintf(
			"Screenshot capture P90 (%.1fms) exceeds 80%% of target frame time (%.1fms). Consider reducing quality or resolution.",
			captureP90, targetFrameTime,
		)
	}

	// If capture P50 > 100ms, capture is definitely slow
	if captureP50 > 100 {
		return BottleneckCapture, fmt.Sprintf(
			"Screenshot capture averaging %.1fms (>100ms threshold). The browser may be under heavy load.",
			captureP50,
		)
	}

	// Component processing sums do not measure transit or client paint.
	if e2eP90 > targetFrameTime*1.5 {
		return BottleneckProcessing, fmt.Sprintf(
			"Processing P90 (%.1fms) exceeds the frame budget. Network transit and client rendering are not measured.", e2eP90)
	}

	return BottleneckNone, "No significant bottlenecks in measured processing. Network transit and client rendering are not measured."
}

// CollectorRegistry manages collectors for multiple sessions.
type CollectorRegistry struct {
	mu         sync.RWMutex
	collectors map[string]*Collector
	targetFps  int
	bufferSize int
}

// NewCollectorRegistry creates a new registry for performance collectors.
func NewCollectorRegistry(targetFps, bufferSize int) *CollectorRegistry {
	return &CollectorRegistry{
		collectors: make(map[string]*Collector),
		targetFps:  targetFps,
		bufferSize: bufferSize,
	}
}

// GetOrCreate returns an existing collector or creates a new one for the session.
func (r *CollectorRegistry) GetOrCreate(sessionID string) *Collector {
	r.mu.Lock()
	defer r.mu.Unlock()

	if c, ok := r.collectors[sessionID]; ok {
		return c
	}

	c := NewCollector(sessionID, r.targetFps, r.bufferSize)
	r.collectors[sessionID] = c
	return c
}

// Get returns a collector for the session, or nil if not found.
func (r *CollectorRegistry) Get(sessionID string) *Collector {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.collectors[sessionID]
}

// Remove removes and returns the collector for a session.
func (r *CollectorRegistry) Remove(sessionID string) *Collector {
	r.mu.Lock()
	defer r.mu.Unlock()

	c := r.collectors[sessionID]
	delete(r.collectors, sessionID)
	return c
}

// Count returns the number of active collectors.
func (r *CollectorRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.collectors)
}
