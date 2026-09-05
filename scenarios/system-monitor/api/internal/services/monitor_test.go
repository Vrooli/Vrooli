package services

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/collectors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/infrastructure"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/memory"
)

type selfMetricsCollector struct {
	collectors.BaseCollector
}

func newSelfMetricsCollector() *selfMetricsCollector {
	return &selfMetricsCollector{BaseCollector: collectors.NewBaseCollector("self-test", 10*time.Second)}
}

func (c *selfMetricsCollector) Collect(context.Context) (*collectors.MetricData, error) {
	return &collectors.MetricData{
		CollectorName: c.GetName(),
		Timestamp:     time.Now(),
		Type:          "test",
		Values:        map[string]interface{}{"ok": true},
	}, nil
}

func TestMonitorService_GetCurrentMetrics(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			MetricsInterval: 10 * time.Second,
		},
	}
	repo := memory.NewRepository()

	svc := NewMonitorService(cfg, repo, infrastructure.NewStaticProvider()) // Pass nil for alert service in tests

	// Test
	ctx := context.Background()
	metrics, err := svc.GetCurrentMetrics(ctx)
	// Assertions
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if metrics == nil {
		t.Fatal("Expected metrics, got nil")
	}

	if metrics.CPUUsage < 0 || metrics.CPUUsage > 100 {
		t.Errorf("Invalid CPU usage: %f", metrics.CPUUsage)
	}

	if metrics.MemoryUsage < 0 || metrics.MemoryUsage > 100 {
		t.Errorf("Invalid memory usage: %f", metrics.MemoryUsage)
	}
}

func TestOnDemandMetricsDoNotResampleStatefulCollectors(t *testing.T) {
	cfg := &config.Config{Monitoring: config.MonitoringConfig{MetricsInterval: time.Second}}
	repo := memory.NewRepository()
	var calls atomic.Int32
	collector := &countingMetricCollector{BaseCollector: collectors.NewBaseCollector("cpu", time.Second), calls: &calls}
	svc := NewMonitorService(cfg, repo, infrastructure.NewStaticProvider(), WithCollectors(collector))

	before, err := svc.GetCurrentMetricsFresh(context.Background())
	if err != nil {
		t.Fatalf("initial current metrics: %v", err)
	}
	if before.CPUState.Status != "not_yet_sampled" {
		t.Fatalf("initial CPU state = %q, want not_yet_sampled", before.CPUState.Status)
	}
	if calls.Load() != 0 {
		t.Fatalf("on-demand read invoked collector %d times", calls.Load())
	}

	svc.collectMetrics()
	after, err := svc.GetCurrentMetricsFresh(context.Background())
	if err != nil {
		t.Fatalf("current metrics after cycle: %v", err)
	}
	if after.CPUState.Status != "measured" || after.CPUUsage != 42 {
		t.Fatalf("after cycle CPU = %#v, want measured 42", after.CPUState)
	}
	if calls.Load() != 1 {
		t.Fatalf("scheduler invoked collector %d times, want 1", calls.Load())
	}
	if _, err := svc.GetDetailedMetrics(context.Background()); err != nil {
		t.Fatalf("detailed metrics: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("detailed read resampled collector %d times, want scheduler-only sampling", calls.Load())
	}
}

func TestUpdateLatestSnapshotPreservesCollectorsSkippedByCadence(t *testing.T) {
	cfg := &config.Config{Monitoring: config.MonitoringConfig{MetricsInterval: time.Second}}
	svc := NewMonitorService(cfg, memory.NewRepository(), infrastructure.NewStaticProvider())
	firstObserved := time.Date(2026, time.August, 21, 12, 0, 0, 0, time.UTC)
	secondObserved := firstObserved.Add(10 * time.Second)

	disk := &collectors.MetricData{
		CollectorName: "disk",
		Timestamp:     firstObserved,
		Values: map[string]interface{}{"usage": map[string]interface{}{
			"used": int64(70), "total": int64(100), "free": int64(30), "percent": float64(70),
		}},
		Tags: map[string]string{"source": "test disk"},
	}
	svc.updateLatestSnapshot("cycle-disk", firstObserved, []*collectors.MetricData{disk})

	// The next scheduler tick only collects CPU. Disk must remain the last
	// measured value instead of falling back to the response type's zero value.
	cpu := &collectors.MetricData{
		CollectorName: "cpu",
		Timestamp:     secondObserved,
		Values:        map[string]interface{}{"status": "measured", "usage_percent": float64(23)},
		Tags:          map[string]string{"source": "test cpu"},
	}
	svc.updateLatestSnapshot("cycle-cpu", secondObserved, []*collectors.MetricData{cpu})

	got, err := svc.GetCurrentMetricsFresh(context.Background())
	if err != nil {
		t.Fatalf("read merged snapshot: %v", err)
	}
	if got.CPUState.Status != "measured" || got.CPUUsage != 23 {
		t.Fatalf("CPU snapshot = %#v, want measured 23", got.CPUState)
	}
	if got.DiskState.Status != "measured" || got.DiskUsage != 70 {
		t.Fatalf("disk snapshot = %#v, want preserved measured 70", got.DiskState)
	}
	if got.DiskState.CycleID != "cycle-disk" || !got.DiskState.ObservedAt.Equal(firstObserved) {
		t.Fatalf("disk provenance = %#v, want original observation", got.DiskState)
	}
}

type countingMetricCollector struct {
	collectors.BaseCollector
	calls *atomic.Int32
}

func (c *countingMetricCollector) Collect(context.Context) (*collectors.MetricData, error) {
	c.calls.Add(1)
	return &collectors.MetricData{
		CollectorName: c.GetName(),
		Timestamp:     time.Now(),
		Values:        map[string]interface{}{"usage_percent": float64(42)},
	}, nil
}

func TestMonitorService_CollectorRegistration(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			MetricsInterval: 10 * time.Second,
		},
	}
	repo := memory.NewRepository()

	svc := NewMonitorService(cfg, repo, infrastructure.NewStaticProvider())

	// Test that collectors are registered
	if svc.collectors == nil {
		t.Error("Collectors not initialized")
	}

	// Check for expected collectors
	expectedCollectors := []string{"cpu", "memory", "network", "disk", "process"}
	for _, name := range expectedCollectors {
		collector, exists := svc.collectors.Get(name)
		if !exists {
			t.Errorf("Expected collector %s not registered", name)
		}
		if collector == nil {
			t.Errorf("Collector %s is nil", name)
		}
	}
}

func TestCollectorCostBudgetsCoverDefaultCollectorsWithHeadroom(t *testing.T) {
	names := []string{"cpu", "memory", "network", "disk", "process", "pressure", "gpu"}
	if err := ValidateCollectorCostBudgets(names, 10*time.Second); err != nil {
		t.Fatalf("default collector cost policy is not headroom-safe: %v", err)
	}
	if _, ok := collectorBudget("disk"); !ok {
		t.Fatal("disk collector has no declared cost budget")
	}
}

func TestCollectorCostBudgetsRejectUndeclaredCollector(t *testing.T) {
	if err := ValidateCollectorCostBudgets([]string{"cpu", "new-probe"}, 10*time.Second); err == nil {
		t.Fatal("expected undeclared collector to fail cost policy")
	}
}

func TestMonitorSelfMetricsFlagCycleWithoutHeadroom(t *testing.T) {
	cfg := &config.Config{Monitoring: config.MonitoringConfig{MetricsInterval: 10 * time.Second}}
	svc := NewMonitorService(cfg, memory.NewRepository(), infrastructure.NewStaticProvider())
	svc.recordCycleSelfMetrics(6*time.Second, 0, time.Now())
	metrics := svc.SelfMetrics()
	if metrics["headroom_ok"] != false {
		t.Fatalf("headroom_ok = %#v, want false", metrics["headroom_ok"])
	}
	if metrics["headroom_reason"] == "" {
		t.Fatal("headroom breach did not include a reason")
	}
}

func TestMonitorService_GetMetricsTimeline_Empty(t *testing.T) {
	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			MetricsInterval: 10 * time.Second,
		},
	}
	repo := memory.NewRepository()
	svc := NewMonitorService(cfg, repo, infrastructure.NewStaticProvider())

	ctx := context.Background()
	timeline, err := svc.GetMetricsTimeline(ctx, 120, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if timeline == nil {
		t.Fatal("expected non-nil timeline")
	}
	if timeline.WindowSeconds != 120 {
		t.Errorf("expected WindowSeconds=120, got %d", timeline.WindowSeconds)
	}
	if timeline.SampleIntervalSeconds != 5 {
		t.Errorf("expected SampleIntervalSeconds=5, got %d", timeline.SampleIntervalSeconds)
	}
	if len(timeline.Samples) != 0 {
		t.Errorf("expected 0 samples for empty repo, got %d", len(timeline.Samples))
	}
}

func TestMonitorService_GetMetricsTimeline_WithData(t *testing.T) {
	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			MetricsInterval: 10 * time.Second,
		},
	}
	repo := memory.NewRepository()
	svc := NewMonitorService(cfg, repo, infrastructure.NewStaticProvider())

	ctx := context.Background()

	// Seed metrics
	_ = repo.SaveMetricCycle(ctx, "timeline-cpu", time.Now(), []repository.MetricObservation{{CollectorName: "cpu", Values: map[string]interface{}{"usage_percent": 42.5}}})
	_ = repo.SaveMetricCycle(ctx, "timeline-memory", time.Now(), []repository.MetricObservation{{CollectorName: "memory", Values: map[string]interface{}{"usage_percent": 65.3}}})
	_ = repo.SaveMetricCycle(ctx, "timeline-network", time.Now(), []repository.MetricObservation{{CollectorName: "network", Values: map[string]interface{}{"tcp_connections": 120}}})

	timeline, err := svc.GetMetricsTimeline(ctx, 120, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if timeline == nil {
		t.Fatal("expected non-nil timeline")
	}
	if len(timeline.Samples) == 0 {
		t.Error("expected at least 1 sample after seeding metrics")
	}
}

func TestMonitorService_GetMetricsTimeline_DefaultsForZeroParams(t *testing.T) {
	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			MetricsInterval: 10 * time.Second,
		},
	}
	repo := memory.NewRepository()
	svc := NewMonitorService(cfg, repo, infrastructure.NewStaticProvider())

	ctx := context.Background()
	timeline, err := svc.GetMetricsTimeline(ctx, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if timeline.WindowSeconds != 120 {
		t.Errorf("expected default WindowSeconds=120, got %d", timeline.WindowSeconds)
	}
	if timeline.SampleIntervalSeconds != 5 {
		t.Errorf("expected default SampleIntervalSeconds=5, got %d", timeline.SampleIntervalSeconds)
	}
}

func TestMonitorService_StartStop(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			MetricsInterval: 100 * time.Millisecond, // Short interval for testing
		},
	}
	repo := memory.NewRepository()

	svc := NewMonitorService(cfg, repo, infrastructure.NewStaticProvider())

	// Start service
	err := svc.Start()
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}

	// Stop service
	svc.Stop()

	// Service should be stopped (context canceled)
	if svc.ctx == nil {
		t.Fatal("Service context not initialized after Start()")
	}

	select {
	case <-svc.ctx.Done():
		// Expected - context canceled synchronously by Stop.
	case <-time.After(500 * time.Millisecond):
		t.Error("service context not canceled after Stop()")
	}
}

func TestMonitorServiceSelfMetricsRecordedAfterCollection(t *testing.T) {
	cfg := &config.Config{Monitoring: config.MonitoringConfig{MetricsInterval: 10 * time.Second}}
	svc := NewMonitorService(
		cfg,
		memory.NewRepository(),
		infrastructure.NewStaticProvider(),
		WithCollectors(newSelfMetricsCollector()),
		WithProcessSampling(nil, nil, nil),
	)

	svc.collectMetrics()
	self := svc.SelfMetrics()

	durations, ok := self["collector_duration_ms"].(map[string]float64)
	if !ok {
		t.Fatalf("collector_duration_ms type = %T", self["collector_duration_ms"])
	}
	if _, ok := durations["self-test"]; !ok {
		t.Fatalf("self-test duration missing from %+v", durations)
	}

	forks, ok := self["collector_forks"].(map[string]uint64)
	if !ok {
		t.Fatalf("collector_forks type = %T", self["collector_forks"])
	}
	if got := forks["self-test"]; got != 0 {
		t.Fatalf("self-test forks = %d, want 0", got)
	}

	if self["recorded_at"] == "0001-01-01T00:00:00Z" {
		t.Fatal("recorded_at was not updated")
	}
}

func TestHighestDiskPressureUsesPeakPartitionUsage(t *testing.T) {
	pressure := highestDiskPressure([]models.DiskPartitionInfo{
		{MountPoint: "/", UsePercent: 72.5},
		{MountPoint: "/var", UsePercent: 91.25},
		{MountPoint: "/tmp", UsePercent: 12},
	})
	if pressure != 91.25 {
		t.Fatalf("pressure = %v, want 91.25", pressure)
	}
}
