package services

import (
	"context"
	"testing"
	"time"

	"system-monitor-api/internal/config"
	"system-monitor-api/internal/repository"
)


func TestMonitorService_GetCurrentMetrics(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			MetricsInterval: 10 * time.Second,
		},
	}
	repo := repository.NewMemoryRepository()
	
	svc := NewMonitorService(cfg, repo, nil) // Pass nil for alert service in tests
	
	// Test
	ctx := context.Background()
	metrics, err := svc.GetCurrentMetrics(ctx)
	
	// Assertions
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if metrics == nil {
		t.Error("Expected metrics, got nil")
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
	repo := repository.NewMemoryRepository()
	
	svc := NewMonitorService(cfg, repo, nil)
	
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

func TestMonitorService_StartStop(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			MetricsInterval: 100 * time.Millisecond, // Short interval for testing
		},
	}
	repo := repository.NewMemoryRepository()
	
	svc := NewMonitorService(cfg, repo, nil)
	
	// Start service
	err := svc.Start()
	if err != nil {
		t.Errorf("Failed to start service: %v", err)
	}
	
	// Let it run briefly
	time.Sleep(200 * time.Millisecond)
	
	// Stop service
	svc.Stop()
	
	// Give it time to stop
	time.Sleep(100 * time.Millisecond)
	
	// Service should be stopped (context canceled)
	select {
	case <-svc.ctx.Done():
		// Expected - context should be canceled
	default:
		t.Error("Service context not canceled after Stop()")
	}
}