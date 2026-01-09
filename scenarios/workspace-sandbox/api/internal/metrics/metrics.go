// Package metrics provides Prometheus-compatible metrics for workspace-sandbox.
// [OT-P1-008] Metrics/Observability
package metrics

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Collector collects metrics for the workspace-sandbox service.
type Collector struct {
	mu sync.RWMutex

	// Operation counters
	sandboxCreatedTotal   int64
	sandboxStoppedTotal   int64
	sandboxApprovedTotal  int64
	sandboxRejectedTotal  int64
	sandboxDeletedTotal   int64
	sandboxErrorsTotal    int64
	gcRunsTotal           int64
	gcSandboxesDeleted    int64
	diffGenerationsTotal  int64
	patchApplicationTotal int64

	// Latency histograms (simplified - store last N values for avg)
	createLatencies []float64
	diffLatencies   []float64
	maxLatencyItems int

	// Gauge values
	activeSandboxes     int64
	totalDiskUsageBytes int64
	processCount        int64
}

// NewCollector creates a new metrics collector.
func NewCollector() *Collector {
	return &Collector{
		maxLatencyItems: 100,
		createLatencies: make([]float64, 0, 100),
		diffLatencies:   make([]float64, 0, 100),
	}
}

// Default global collector
var defaultCollector = NewCollector()

// Default returns the default global metrics collector.
func Default() *Collector {
	return defaultCollector
}

// --- Counter Increments ---

// IncSandboxCreated increments the sandbox created counter.
func (c *Collector) IncSandboxCreated() {
	atomic.AddInt64(&c.sandboxCreatedTotal, 1)
}

// IncSandboxStopped increments the sandbox stopped counter.
func (c *Collector) IncSandboxStopped() {
	atomic.AddInt64(&c.sandboxStoppedTotal, 1)
}

// IncSandboxApproved increments the sandbox approved counter.
func (c *Collector) IncSandboxApproved() {
	atomic.AddInt64(&c.sandboxApprovedTotal, 1)
}

// IncSandboxRejected increments the sandbox rejected counter.
func (c *Collector) IncSandboxRejected() {
	atomic.AddInt64(&c.sandboxRejectedTotal, 1)
}

// IncSandboxDeleted increments the sandbox deleted counter.
func (c *Collector) IncSandboxDeleted() {
	atomic.AddInt64(&c.sandboxDeletedTotal, 1)
}

// IncSandboxErrors increments the sandbox errors counter.
func (c *Collector) IncSandboxErrors() {
	atomic.AddInt64(&c.sandboxErrorsTotal, 1)
}

// IncGCRuns increments the GC runs counter.
func (c *Collector) IncGCRuns() {
	atomic.AddInt64(&c.gcRunsTotal, 1)
}

// AddGCSandboxesDeleted adds to the GC sandboxes deleted counter.
func (c *Collector) AddGCSandboxesDeleted(n int64) {
	atomic.AddInt64(&c.gcSandboxesDeleted, n)
}

// IncDiffGenerations increments the diff generations counter.
func (c *Collector) IncDiffGenerations() {
	atomic.AddInt64(&c.diffGenerationsTotal, 1)
}

// IncPatchApplications increments the patch applications counter.
func (c *Collector) IncPatchApplications() {
	atomic.AddInt64(&c.patchApplicationTotal, 1)
}

// --- Gauge Updates ---

// SetActiveSandboxes sets the active sandbox gauge.
func (c *Collector) SetActiveSandboxes(count int64) {
	atomic.StoreInt64(&c.activeSandboxes, count)
}

// SetTotalDiskUsage sets the total disk usage gauge.
func (c *Collector) SetTotalDiskUsage(bytes int64) {
	atomic.StoreInt64(&c.totalDiskUsageBytes, bytes)
}

// SetProcessCount sets the process count gauge.
func (c *Collector) SetProcessCount(count int64) {
	atomic.StoreInt64(&c.processCount, count)
}

// --- Latency Recording ---

// RecordCreateLatency records a sandbox creation latency.
func (c *Collector) RecordCreateLatency(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ms := float64(d.Milliseconds())
	if len(c.createLatencies) >= c.maxLatencyItems {
		c.createLatencies = c.createLatencies[1:]
	}
	c.createLatencies = append(c.createLatencies, ms)
}

// RecordDiffLatency records a diff generation latency.
func (c *Collector) RecordDiffLatency(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ms := float64(d.Milliseconds())
	if len(c.diffLatencies) >= c.maxLatencyItems {
		c.diffLatencies = c.diffLatencies[1:]
	}
	c.diffLatencies = append(c.diffLatencies, ms)
}

// --- Export ---

// ExportPrometheus exports all metrics in Prometheus text format.
func (c *Collector) ExportPrometheus() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var b strings.Builder

	// Counters
	b.WriteString("# HELP workspace_sandbox_created_total Total number of sandboxes created\n")
	b.WriteString("# TYPE workspace_sandbox_created_total counter\n")
	b.WriteString(fmt.Sprintf("workspace_sandbox_created_total %d\n", atomic.LoadInt64(&c.sandboxCreatedTotal)))

	b.WriteString("# HELP workspace_sandbox_stopped_total Total number of sandboxes stopped\n")
	b.WriteString("# TYPE workspace_sandbox_stopped_total counter\n")
	b.WriteString(fmt.Sprintf("workspace_sandbox_stopped_total %d\n", atomic.LoadInt64(&c.sandboxStoppedTotal)))

	b.WriteString("# HELP workspace_sandbox_approved_total Total number of sandboxes approved\n")
	b.WriteString("# TYPE workspace_sandbox_approved_total counter\n")
	b.WriteString(fmt.Sprintf("workspace_sandbox_approved_total %d\n", atomic.LoadInt64(&c.sandboxApprovedTotal)))

	b.WriteString("# HELP workspace_sandbox_rejected_total Total number of sandboxes rejected\n")
	b.WriteString("# TYPE workspace_sandbox_rejected_total counter\n")
	b.WriteString(fmt.Sprintf("workspace_sandbox_rejected_total %d\n", atomic.LoadInt64(&c.sandboxRejectedTotal)))

	b.WriteString("# HELP workspace_sandbox_deleted_total Total number of sandboxes deleted\n")
	b.WriteString("# TYPE workspace_sandbox_deleted_total counter\n")
	b.WriteString(fmt.Sprintf("workspace_sandbox_deleted_total %d\n", atomic.LoadInt64(&c.sandboxDeletedTotal)))

	b.WriteString("# HELP workspace_sandbox_errors_total Total number of sandbox errors\n")
	b.WriteString("# TYPE workspace_sandbox_errors_total counter\n")
	b.WriteString(fmt.Sprintf("workspace_sandbox_errors_total %d\n", atomic.LoadInt64(&c.sandboxErrorsTotal)))

	b.WriteString("# HELP workspace_sandbox_gc_runs_total Total number of garbage collection runs\n")
	b.WriteString("# TYPE workspace_sandbox_gc_runs_total counter\n")
	b.WriteString(fmt.Sprintf("workspace_sandbox_gc_runs_total %d\n", atomic.LoadInt64(&c.gcRunsTotal)))

	b.WriteString("# HELP workspace_sandbox_gc_sandboxes_deleted_total Total sandboxes deleted by GC\n")
	b.WriteString("# TYPE workspace_sandbox_gc_sandboxes_deleted_total counter\n")
	b.WriteString(fmt.Sprintf("workspace_sandbox_gc_sandboxes_deleted_total %d\n", atomic.LoadInt64(&c.gcSandboxesDeleted)))

	b.WriteString("# HELP workspace_sandbox_diff_generations_total Total diff generations\n")
	b.WriteString("# TYPE workspace_sandbox_diff_generations_total counter\n")
	b.WriteString(fmt.Sprintf("workspace_sandbox_diff_generations_total %d\n", atomic.LoadInt64(&c.diffGenerationsTotal)))

	b.WriteString("# HELP workspace_sandbox_patch_applications_total Total patch applications\n")
	b.WriteString("# TYPE workspace_sandbox_patch_applications_total counter\n")
	b.WriteString(fmt.Sprintf("workspace_sandbox_patch_applications_total %d\n", atomic.LoadInt64(&c.patchApplicationTotal)))

	// Gauges
	b.WriteString("# HELP workspace_sandbox_active Active sandboxes count\n")
	b.WriteString("# TYPE workspace_sandbox_active gauge\n")
	b.WriteString(fmt.Sprintf("workspace_sandbox_active %d\n", atomic.LoadInt64(&c.activeSandboxes)))

	b.WriteString("# HELP workspace_sandbox_disk_usage_bytes Total disk usage in bytes\n")
	b.WriteString("# TYPE workspace_sandbox_disk_usage_bytes gauge\n")
	b.WriteString(fmt.Sprintf("workspace_sandbox_disk_usage_bytes %d\n", atomic.LoadInt64(&c.totalDiskUsageBytes)))

	b.WriteString("# HELP workspace_sandbox_processes Active process count\n")
	b.WriteString("# TYPE workspace_sandbox_processes gauge\n")
	b.WriteString(fmt.Sprintf("workspace_sandbox_processes %d\n", atomic.LoadInt64(&c.processCount)))

	// Latency summaries
	if len(c.createLatencies) > 0 {
		avg := average(c.createLatencies)
		b.WriteString("# HELP workspace_sandbox_create_latency_ms Average sandbox creation latency in ms\n")
		b.WriteString("# TYPE workspace_sandbox_create_latency_ms gauge\n")
		b.WriteString(fmt.Sprintf("workspace_sandbox_create_latency_ms %.2f\n", avg))
	}

	if len(c.diffLatencies) > 0 {
		avg := average(c.diffLatencies)
		b.WriteString("# HELP workspace_sandbox_diff_latency_ms Average diff generation latency in ms\n")
		b.WriteString("# TYPE workspace_sandbox_diff_latency_ms gauge\n")
		b.WriteString(fmt.Sprintf("workspace_sandbox_diff_latency_ms %.2f\n", avg))
	}

	return b.String()
}

// Snapshot returns a copy of the current metrics values for JSON export.
func (c *Collector) Snapshot() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	snapshot := map[string]interface{}{
		"sandbox_created_total":      atomic.LoadInt64(&c.sandboxCreatedTotal),
		"sandbox_stopped_total":      atomic.LoadInt64(&c.sandboxStoppedTotal),
		"sandbox_approved_total":     atomic.LoadInt64(&c.sandboxApprovedTotal),
		"sandbox_rejected_total":     atomic.LoadInt64(&c.sandboxRejectedTotal),
		"sandbox_deleted_total":      atomic.LoadInt64(&c.sandboxDeletedTotal),
		"sandbox_errors_total":       atomic.LoadInt64(&c.sandboxErrorsTotal),
		"gc_runs_total":              atomic.LoadInt64(&c.gcRunsTotal),
		"gc_sandboxes_deleted_total": atomic.LoadInt64(&c.gcSandboxesDeleted),
		"diff_generations_total":     atomic.LoadInt64(&c.diffGenerationsTotal),
		"patch_applications_total":   atomic.LoadInt64(&c.patchApplicationTotal),
		"active_sandboxes":           atomic.LoadInt64(&c.activeSandboxes),
		"total_disk_usage_bytes":     atomic.LoadInt64(&c.totalDiskUsageBytes),
		"process_count":              atomic.LoadInt64(&c.processCount),
		"create_latency_samples":     len(c.createLatencies),
		"diff_latency_samples":       len(c.diffLatencies),
	}

	if len(c.createLatencies) > 0 {
		snapshot["avg_create_latency_ms"] = average(c.createLatencies)
	}
	if len(c.diffLatencies) > 0 {
		snapshot["avg_diff_latency_ms"] = average(c.diffLatencies)
	}

	return snapshot
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}
