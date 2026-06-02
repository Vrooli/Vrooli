package services

import (
	"testing"
	"time"

	"system-monitor-api/internal/config"
	"system-monitor-api/internal/infrastructure"
	"system-monitor-api/internal/repository/memory"
)

// TestMonitorService_ApplySettingsChangesCadence proves that a settings change
// affects the effective collection interval and subsequent collection
// decisions without restarting the service.
func TestMonitorService_ApplySettingsChangesCadence(t *testing.T) {
	cfg := &config.Config{Monitoring: config.MonitoringConfig{MetricsInterval: 10 * time.Second}}
	svc := NewMonitorService(cfg, memory.NewRepository(), infrastructure.NewStaticProvider())

	if got := svc.EffectiveCollectionInterval(); got != 10*time.Second {
		t.Fatalf("initial interval = %v, want 10s", got)
	}

	svc.ApplySettings(Settings{MetricCollectionInterval: 30})
	if got := svc.EffectiveCollectionInterval(); got != 30*time.Second {
		t.Fatalf("after ApplySettings interval = %v, want 30s", got)
	}

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	svc.markCollected("cpu", base)

	// interval=0 forces the collector to fall back to the live setting (30s).
	if svc.shouldCollect("cpu", 0, base.Add(20*time.Second)) {
		t.Error("should NOT collect 20s after last run with a 30s interval")
	}
	if !svc.shouldCollect("cpu", 0, base.Add(31*time.Second)) {
		t.Error("should collect 31s after last run with a 30s interval")
	}
}

// TestMonitorService_ApplySettingsIgnoresNonPositive ensures a zero interval
// does not clobber the existing cadence.
func TestMonitorService_ApplySettingsIgnoresNonPositive(t *testing.T) {
	cfg := &config.Config{Monitoring: config.MonitoringConfig{MetricsInterval: 10 * time.Second}}
	svc := NewMonitorService(cfg, memory.NewRepository(), infrastructure.NewStaticProvider())

	svc.ApplySettings(Settings{MetricCollectionInterval: 0})
	if got := svc.EffectiveCollectionInterval(); got != 10*time.Second {
		t.Errorf("interval = %v, want unchanged 10s", got)
	}
}
