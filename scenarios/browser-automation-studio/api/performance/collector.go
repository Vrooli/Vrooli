// Package performance provides types and utilities for debug performance mode.
package performance

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// Collector collects and aggregates frame timing data.
// Thread-safe for concurrent access.
type Collector struct {
	sessionID  string
	targetFps  int
	bufferSize int

	mu          sync.RWMutex
	timings     []FrameTimings
	frameCount  int
	skipCount   int
	windowStart time.Time
}

// NewCollector creates a new performance collector.
func NewCollector(sessionID string, targetFps, bufferSize int) *Collector {
	return &Collector{
		sessionID:   sessionID,
		targetFps:   targetFps,
		bufferSize:  bufferSize,
		timings:     make([]FrameTimings, 0, bufferSize),
		windowStart: time.Now(),
	}
}

// Record adds a frame timing to the collector.
func (c *Collector) Record(t *FrameTimings) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.frameCount++
	if t.Skipped {
		c.skipCount++
	}

	// Ring buffer: evict oldest if full
	if len(c.timings) >= c.bufferSize {
		c.timings = c.timings[1:]
	}
	c.timings = append(c.timings, *t)
}

// GetFrameCount returns the total number of frames recorded.
func (c *Collector) GetFrameCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.frameCount
}

// ShouldBroadcast returns true if stats should be broadcast (every 60 frames).
func (c *Collector) ShouldBroadcast() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.frameCount > 0 && c.frameCount%60 == 0
}

// GetAggregated returns aggregated statistics from the ring buffer.
func (c *Collector) GetAggregated() FrameStatsAggregated {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()
	windowDuration := now.Sub(c.windowStart)

	if len(c.timings) == 0 {
		return FrameStatsAggregated{
			SessionID:             c.sessionID,
			WindowStartTime:       c.windowStart,
			WindowDurationMs:      windowDuration.Milliseconds(),
			FrameCount:            0,
			SkippedCount:          0,
			TargetFps:             c.targetFps,
			PrimaryBottleneck:     BottleneckNone,
			BottleneckDescription: "No frames recorded yet",
		}
	}

	// Extract timing arrays for percentile calculation
	captureTimes := make([]float64, 0, len(c.timings))
	e2eTimes := make([]float64, 0, len(c.timings))
	var totalBytes int64

	for _, t := range c.timings {
		captureTimes = append(captureTimes, t.DriverCaptureMs)
		e2e := t.DriverTotalMs + t.APITotalMs
		e2eTimes = append(e2eTimes, e2e)
		if !t.Skipped {
			totalBytes += int64(t.FrameBytes)
		}
	}

	// Sort for percentile calculation
	sort.Float64s(captureTimes)
	sort.Float64s(e2eTimes)

	// Calculate percentiles
	captureP50 := percentile(captureTimes, 0.5)
	captureP90 := percentile(captureTimes, 0.9)
	captureP99 := percentile(captureTimes, 0.99)
	captureMax := captureTimes[len(captureTimes)-1]

	e2eP50 := percentile(e2eTimes, 0.5)
	e2eP90 := percentile(e2eTimes, 0.9)
	e2eP99 := percentile(e2eTimes, 0.99)
	e2eMax := e2eTimes[len(e2eTimes)-1]

	// Calculate throughput
	windowSec := windowDuration.Seconds()
	actualFps := 0.0
	if windowSec > 0 {
		actualFps = float64(c.frameCount) / windowSec
	}

	nonSkipped := len(c.timings) - c.skipCount
	avgFrameBytes := 0
	if nonSkipped > 0 {
		avgFrameBytes = int(totalBytes / int64(nonSkipped))
	}

	bandwidthBytesPerSec := int64(0)
	if windowSec > 0 {
		bandwidthBytesPerSec = int64(float64(totalBytes) / windowSec)
	}

	// Identify bottleneck
	bottleneck, description := identifyBottleneck(captureP50, captureP90, e2eP90, c.targetFps)

	return FrameStatsAggregated{
		SessionID:             c.sessionID,
		WindowStartTime:       c.windowStart,
		WindowDurationMs:      windowDuration.Milliseconds(),
		FrameCount:            c.frameCount,
		SkippedCount:          c.skipCount,
		CaptureP50Ms:          round2(captureP50),
		CaptureP90Ms:          round2(captureP90),
		CaptureP99Ms:          round2(captureP99),
		CaptureMaxMs:          round2(captureMax),
		E2EP50Ms:              round2(e2eP50),
		E2EP90Ms:              round2(e2eP90),
		E2EP99Ms:              round2(e2eP99),
		E2EMaxMs:              round2(e2eMax),
		ActualFps:             round2(actualFps),
		TargetFps:             c.targetFps,
		AvgFrameBytes:         avgFrameBytes,
		BandwidthBytesPerSec:  bandwidthBytesPerSec,
		PrimaryBottleneck:     bottleneck,
		BottleneckDescription: description,
	}
}

// GetRecentFrames returns the most recent frame timings.
func (c *Collector) GetRecentFrames(limit int) []FrameTimings {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if limit <= 0 || limit > len(c.timings) {
		limit = len(c.timings)
	}

	start := len(c.timings) - limit
	if start < 0 {
		start = 0
	}

	result := make([]FrameTimings, limit)
	copy(result, c.timings[start:])
	return result
}

// Reset clears all collected data.
func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.timings = c.timings[:0]
	c.frameCount = 0
	c.skipCount = 0
	c.windowStart = time.Now()
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

	// If E2E P90 > 150% of target frame time and capture is fine, it's likely network
	if e2eP90 > targetFrameTime*1.5 && captureP90 < targetFrameTime*0.5 {
		networkTime := e2eP90 - captureP90
		return BottleneckNetwork, fmt.Sprintf(
			"End-to-end P90 (%.1fms) indicates network latency. Estimated network overhead: %.1fms.",
			e2eP90, networkTime,
		)
	}

	return BottleneckNone, "No significant bottlenecks detected. Performance is within expected bounds."
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
