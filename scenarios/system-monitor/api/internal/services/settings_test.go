package services

import (
	"reflect"
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
		CPUHighPercent:                92,
		CPUCriticalPercent:            97,
		CPUEscalationCooldownSeconds:  1800,
		CPUEscalationDebounceTicks:    2,
		CPUSustainedWindowTicks:       3,
		CPUPressureThreshold:          10,
		MemoryThreshold:               75,
		MemoryCriticalPercent:         85,
		TCPConnectionsWarning:         3500,
		TCPConnectionsCritical:        12000,
		TCPGrowthWindowSeconds:        900,
		TCPGrowthFloor:                400,
		TCPGrowthMinSamples:           6,
		TCPGrowthFactor:               2.5,
		GPUUsageWarningPercent:        88,
		MetricsSnapshotStaleIntervals: 12,
		OrphanedAgentAgeSeconds:       7200,
		OrphanedAgentMemoryBytes:      3 * 1024 * 1024 * 1024,
		AutoInvestigateCriticalRules: map[string]bool{
			RuleMetricsSnapshot: true,
			RuleCPUUsage:        false,
			RuleMemoryUsage:     true,
			RuleTCPConnections:  true,
			RuleGPUUsage:        true,
			RuleMemoryPressure:  true,
			RuleOrphanedAgents:  true,
		},
		DiskThreshold:                 80,
		DiskHighPercent:               90,
		DiskCriticalPercent:           95,
		DiskEscalationCooldownSeconds: 1800,
		DiskEscalationDebounceTicks:   2,
		DiskFastFillJumpPercent:       5,
		AlertSustainSeconds:           120,

		MemoryPressureSomeAvg60ElevatedPercent:     15,
		MemoryPressureFullAvg60CriticalPercent:     7,
		MemoryPressureSwapInElevatedBytesPerSecond: 8 * 1024 * 1024,
		MemoryPressureSwapInCriticalBytesPerSecond: 40 * 1024 * 1024,

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
	if got := reloaded.GetSettings(); !reflect.DeepEqual(got, want) {
		t.Errorf("round-trip mismatch:\n got=%+v\nwant=%+v", got, want)
	}
}

func TestSettingsManager_AutoInvestigationPolicyDefaultsAndClones(t *testing.T) {
	sm := NewSettingsManager(WithSettingsConfigStore(NewMemoryConfigStore()))
	got := sm.GetSettings()
	if !got.AutoInvestigateCriticalRules[RuleCPUUsage] {
		t.Fatal("CPU auto-investigation should be enabled by default")
	}
	got.AutoInvestigateCriticalRules[RuleCPUUsage] = false
	if !sm.GetSettings().AutoInvestigateCriticalRules[RuleCPUUsage] {
		t.Fatal("mutating returned policy changed the manager's settings")
	}
	got = sm.GetSettings()
	got.AutoInvestigateCriticalRules = map[string]bool{RuleCPUUsage: false, "unknown_rule": true}
	if err := sm.UpdateSettings(got); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	saved := sm.GetSettings().AutoInvestigateCriticalRules
	if saved[RuleCPUUsage] || !saved[RuleMemoryUsage] {
		t.Fatalf("policy defaults/explicit false not preserved: %#v", saved)
	}
	if _, exists := saved["unknown_rule"]; exists {
		t.Fatalf("unknown rule retained in policy: %#v", saved)
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
			"metric_collection_interval": 20,
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
	if got.MetricCollectionInterval != 20 {
		t.Errorf("MetricCollectionInterval = %d, want 20", got.MetricCollectionInterval)
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

func TestSettingsManager_PreservesUnreadableSettingsForRecovery(t *testing.T) {
	store := NewMemoryConfigStore()
	corrupt := []byte(`{"version":`)
	if err := store.WriteConfig(settingsFileName, corrupt); err != nil {
		t.Fatalf("seed corrupt settings: %v", err)
	}

	sm := NewSettingsManager(WithSettingsConfigStore(store))
	if got := sm.GetSettings(); got.MetricCollectionInterval != defaultSettings.MetricCollectionInterval {
		t.Fatalf("in-memory fallback interval = %d, want default %d", got.MetricCollectionInterval, defaultSettings.MetricCollectionInterval)
	}
	got, err := store.ReadConfig(settingsFileName)
	if err != nil {
		t.Fatalf("read preserved settings: %v", err)
	}
	if string(got) != string(corrupt) {
		t.Fatalf("corrupt settings were overwritten: got %q, want original %q", got, corrupt)
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
