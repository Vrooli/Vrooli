package metrics

import (
	"strings"
	"testing"
	"time"
)

func TestCollector_Counters(t *testing.T) {
	c := NewCollector()

	// Increment counters
	c.IncSandboxCreated()
	c.IncSandboxCreated()
	c.IncSandboxStopped()
	c.IncSandboxApproved()
	c.IncSandboxRejected()
	c.IncSandboxDeleted()
	c.IncSandboxErrors()
	c.IncGCRuns()
	c.AddGCSandboxesDeleted(5)
	c.IncDiffGenerations()
	c.IncPatchApplications()

	snapshot := c.Snapshot()

	if got := snapshot["sandbox_created_total"].(int64); got != 2 {
		t.Errorf("sandbox_created_total = %d, want 2", got)
	}
	if got := snapshot["sandbox_stopped_total"].(int64); got != 1 {
		t.Errorf("sandbox_stopped_total = %d, want 1", got)
	}
	if got := snapshot["gc_sandboxes_deleted_total"].(int64); got != 5 {
		t.Errorf("gc_sandboxes_deleted_total = %d, want 5", got)
	}
}

func TestCollector_Gauges(t *testing.T) {
	c := NewCollector()

	c.SetActiveSandboxes(10)
	c.SetTotalDiskUsage(1024 * 1024 * 100) // 100 MB
	c.SetProcessCount(5)

	snapshot := c.Snapshot()

	if got := snapshot["active_sandboxes"].(int64); got != 10 {
		t.Errorf("active_sandboxes = %d, want 10", got)
	}
	if got := snapshot["total_disk_usage_bytes"].(int64); got != 104857600 {
		t.Errorf("total_disk_usage_bytes = %d, want 104857600", got)
	}
	if got := snapshot["process_count"].(int64); got != 5 {
		t.Errorf("process_count = %d, want 5", got)
	}
}

func TestCollector_Latencies(t *testing.T) {
	c := NewCollector()

	// Record some latencies
	c.RecordCreateLatency(100 * time.Millisecond)
	c.RecordCreateLatency(200 * time.Millisecond)
	c.RecordCreateLatency(300 * time.Millisecond)

	c.RecordDiffLatency(50 * time.Millisecond)
	c.RecordDiffLatency(150 * time.Millisecond)

	snapshot := c.Snapshot()

	avgCreate, ok := snapshot["avg_create_latency_ms"].(float64)
	if !ok {
		t.Fatal("avg_create_latency_ms not found in snapshot")
	}
	if avgCreate < 199 || avgCreate > 201 {
		t.Errorf("avg_create_latency_ms = %f, want ~200", avgCreate)
	}

	avgDiff, ok := snapshot["avg_diff_latency_ms"].(float64)
	if !ok {
		t.Fatal("avg_diff_latency_ms not found in snapshot")
	}
	if avgDiff < 99 || avgDiff > 101 {
		t.Errorf("avg_diff_latency_ms = %f, want ~100", avgDiff)
	}
}

func TestCollector_ExportPrometheus(t *testing.T) {
	c := NewCollector()

	c.IncSandboxCreated()
	c.SetActiveSandboxes(5)
	c.RecordCreateLatency(100 * time.Millisecond)

	output := c.ExportPrometheus()

	// Check it contains expected metrics
	expectedMetrics := []string{
		"workspace_sandbox_created_total 1",
		"workspace_sandbox_active 5",
		"workspace_sandbox_create_latency_ms",
		"# TYPE workspace_sandbox_created_total counter",
		"# HELP workspace_sandbox_active",
	}

	for _, expected := range expectedMetrics {
		if !strings.Contains(output, expected) {
			t.Errorf("Prometheus output missing: %s", expected)
		}
	}
}

func TestCollector_LatencyRingBuffer(t *testing.T) {
	c := NewCollector()
	c.maxLatencyItems = 3 // Small buffer for testing

	// Fill and overflow the buffer
	for i := 0; i < 5; i++ {
		c.RecordCreateLatency(time.Duration(i+1) * 100 * time.Millisecond)
	}

	snapshot := c.Snapshot()

	// Should only have 3 samples (the last 3: 300, 400, 500 ms)
	if got := snapshot["create_latency_samples"].(int); got != 3 {
		t.Errorf("create_latency_samples = %d, want 3", got)
	}

	// Average should be (300+400+500)/3 = 400
	avgCreate := snapshot["avg_create_latency_ms"].(float64)
	if avgCreate < 399 || avgCreate > 401 {
		t.Errorf("avg_create_latency_ms = %f, want ~400", avgCreate)
	}
}

func TestDefault(t *testing.T) {
	// Ensure default collector is available
	d := Default()
	if d == nil {
		t.Error("Default() returned nil")
	}

	// Operations should work on default
	d.IncSandboxCreated()
	snapshot := d.Snapshot()
	if snapshot["sandbox_created_total"] == nil {
		t.Error("Default collector not tracking metrics")
	}
}
