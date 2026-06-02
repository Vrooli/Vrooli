package services

import (
	"testing"
	"time"
)

// TestSettingsManager_RoundTrip proves that settings persisted through the
// canonical wrapper shape load back into Settings exactly, including the new
// lifecycle fields and a deliberately-false RetentionRunOnStartup.
func TestSettingsManager_RoundTrip(t *testing.T) {
	store := NewMemoryConfigStore()
	clock := NewStubClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	sm := NewSettingsManager(WithSettingsConfigStore(store), WithSettingsClock(clock))

	want := Settings{
		Active:                        true,
		MetricCollectionInterval:      15,
		AnomalyDetectionInterval:      45,
		ThresholdCheckInterval:        25,
		CooldownPeriodSeconds:         600,
		CPUThreshold:                  70,
		MemoryThreshold:               75,
		DiskThreshold:                 80,
		MetricsRetentionDays:          14,
		RetentionCheckIntervalSeconds: 1800,
		RetentionRunOnStartup:         false,
		CompactAfterRetention:         true,
	}
	if err := sm.UpdateSettings(want); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	// A fresh manager reading the same store must reproduce the settings.
	reloaded := NewSettingsManager(WithSettingsConfigStore(store), WithSettingsClock(clock))
	if got := reloaded.GetSettings(); got != want {
		t.Errorf("round-trip mismatch:\n got=%+v\nwant=%+v", got, want)
	}
}

// TestSettingsManager_LoadsWrapperShape is the regression test for the original
// defect: a wrapper-shaped file ({version, metadata, settings}) was unmarshaled
// directly into Settings, silently dropping every persisted value. It must now
// load the inner settings (proving Active=true survives) and backfill lifecycle
// defaults for a legacy file that predates those fields.
func TestSettingsManager_LoadsWrapperShape(t *testing.T) {
	store := NewMemoryConfigStore()
	legacy := `{
		"version": "1.0.0",
		"metadata": {"config_version": "1.0.0"},
		"settings": {
			"active": true,
			"metric_collection_interval": 10,
			"anomaly_detection_interval": 30,
			"threshold_check_interval": 20,
			"cooldown_period_seconds": 300,
			"cpu_threshold": 85,
			"memory_threshold": 90,
			"disk_threshold": 85
		}
	}`
	if err := store.WriteConfig("system-monitor-settings.json", []byte(legacy)); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	sm := NewSettingsManager(WithSettingsConfigStore(store))
	got := sm.GetSettings()

	if !got.Active {
		t.Error("Active should load as true from the wrapper shape (regression)")
	}
	if got.MetricCollectionInterval != 10 {
		t.Errorf("MetricCollectionInterval = %d, want 10", got.MetricCollectionInterval)
	}
	if got.MetricsRetentionDays != 30 {
		t.Errorf("MetricsRetentionDays = %d, want default 30", got.MetricsRetentionDays)
	}
	if got.RetentionCheckIntervalSeconds != 3600 {
		t.Errorf("RetentionCheckIntervalSeconds = %d, want default 3600", got.RetentionCheckIntervalSeconds)
	}
	if !got.RetentionRunOnStartup {
		t.Error("RetentionRunOnStartup should default to true for a legacy file")
	}
}

// TestSettingsManager_SettingsChangedCallback verifies the new callback fires
// on update with the sanitized settings.
func TestSettingsManager_SettingsChangedCallback(t *testing.T) {
	store := NewMemoryConfigStore()
	sm := NewSettingsManager(WithSettingsConfigStore(store))

	got := make(chan Settings, 1)
	sm.SetSettingsChangedCallback(func(next Settings) { got <- next })

	want := defaultSettings
	want.MetricCollectionInterval = 42
	if err := sm.UpdateSettings(want); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	select {
	case s := <-got:
		if s.MetricCollectionInterval != 42 {
			t.Errorf("callback interval = %d, want 42", s.MetricCollectionInterval)
		}
	case <-time.After(time.Second):
		t.Fatal("settings-changed callback did not fire")
	}
}
