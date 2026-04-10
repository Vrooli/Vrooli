package services

import (
	"context"
	"testing"
	"time"

	"system-monitor-api/internal/config"
	"system-monitor-api/internal/infrastructure"
	"system-monitor-api/internal/repository/memory"
)

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
	_ = repo.SaveMetrics(ctx, "cpu", map[string]interface{}{"usage_percent": 42.5})
	_ = repo.SaveMetrics(ctx, "memory", map[string]interface{}{"usage_percent": 65.3})
	_ = repo.SaveMetrics(ctx, "network", map[string]interface{}{"tcp_connections": 120})

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
