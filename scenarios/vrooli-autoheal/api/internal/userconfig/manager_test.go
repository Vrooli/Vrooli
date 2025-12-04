package userconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", cfg.Version)
	}

	if cfg.Global.GracePeriodSeconds != 60 {
		t.Errorf("expected grace period 60, got %d", cfg.Global.GracePeriodSeconds)
	}

	if cfg.Global.TickIntervalSeconds != 60 {
		t.Errorf("expected tick interval 60, got %d", cfg.Global.TickIntervalSeconds)
	}

	if cfg.UI.Theme != "system" {
		t.Errorf("expected theme system, got %s", cfg.UI.Theme)
	}
}

func TestGetCheckDefaults(t *testing.T) {
	tests := []struct {
		checkID          string
		expectedEnabled  bool
		expectedAutoHeal bool
	}{
		{"infra-network", true, false},
		{"infra-dns", true, true},
		{"resource-postgres", true, true},
		{"infra-display", false, false},
		{"unknown-check", true, false}, // Generic defaults
	}

	for _, tc := range tests {
		t.Run(tc.checkID, func(t *testing.T) {
			defaults := GetCheckDefaults(tc.checkID)
			if defaults.Enabled != tc.expectedEnabled {
				t.Errorf("check %s: expected enabled=%v, got %v", tc.checkID, tc.expectedEnabled, defaults.Enabled)
			}
			if defaults.AutoHeal != tc.expectedAutoHeal {
				t.Errorf("check %s: expected autoHeal=%v, got %v", tc.checkID, tc.expectedAutoHeal, defaults.AutoHeal)
			}
		})
	}
}

func TestManagerLoadSave(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	schemaPath := filepath.Join(tmpDir, "schema.json")

	// Create a minimal schema file
	os.WriteFile(schemaPath, []byte(`{"type":"object"}`), 0644)

	mgr := NewManager(configPath, schemaPath)

	// Load should succeed even with no file (uses defaults)
	if err := mgr.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Get config should return defaults
	cfg := mgr.Get()
	if cfg.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", cfg.Version)
	}

	// Modify and save
	cfg.Global.GracePeriodSeconds = 120
	if err := mgr.Update(cfg); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Create new manager and load
	mgr2 := NewManager(configPath, schemaPath)
	if err := mgr2.Load(); err != nil {
		t.Fatalf("Load after save failed: %v", err)
	}

	cfg2 := mgr2.Get()
	if cfg2.Global.GracePeriodSeconds != 120 {
		t.Errorf("expected grace period 120 after reload, got %d", cfg2.Global.GracePeriodSeconds)
	}
}

func TestManagerValidation(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(filepath.Join(tmpDir, "config.json"), filepath.Join(tmpDir, "schema.json"))

	tests := []struct {
		name   string
		config Config
		valid  bool
	}{
		{
			name: "valid config",
			config: Config{
				Version: "1.0",
				Global: GlobalConfig{
					GracePeriodSeconds:     60,
					TickIntervalSeconds:    60,
					VerifyDelaySeconds:     30,
					MaxRestartAttempts:     3,
					RestartCooldownSeconds: 300,
					HistoryRetentionHours:  24,
				},
				UI: UIConfig{
					AutoRefreshSeconds: 30,
					Theme:              "system",
					DefaultTab:         "dashboard",
				},
			},
			valid: true,
		},
		{
			name: "invalid version",
			config: Config{
				Version: "2.0",
				Global:  DefaultGlobal(),
				UI:      DefaultUI(),
			},
			valid: false,
		},
		{
			name: "invalid tick interval - too low",
			config: Config{
				Version: "1.0",
				Global: GlobalConfig{
					GracePeriodSeconds:     60,
					TickIntervalSeconds:    5, // Too low
					VerifyDelaySeconds:     30,
					MaxRestartAttempts:     3,
					RestartCooldownSeconds: 300,
					HistoryRetentionHours:  24,
				},
				UI: DefaultUI(),
			},
			valid: false,
		},
		{
			name: "invalid theme",
			config: Config{
				Version: "1.0",
				Global:  DefaultGlobal(),
				UI: UIConfig{
					AutoRefreshSeconds: 30,
					Theme:              "invalid",
					DefaultTab:         "dashboard",
				},
			},
			valid: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := mgr.Validate(&tc.config)
			if result.Valid != tc.valid {
				t.Errorf("expected valid=%v, got %v. Errors: %v", tc.valid, result.Valid, result.Errors)
			}
		})
	}
}

func TestManagerGetCheck(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	schemaPath := filepath.Join(tmpDir, "schema.json")

	mgr := NewManager(configPath, schemaPath)
	mgr.Load()

	// Get default check config
	cfg := mgr.GetCheck("resource-postgres")
	if !cfg.Enabled {
		t.Error("postgres should be enabled by default")
	}
	if !cfg.AutoHeal {
		t.Error("postgres should have autoHeal enabled by default")
	}

	// Override via config
	mgr.SetCheckEnabled("resource-postgres", false)

	cfg = mgr.GetCheck("resource-postgres")
	if cfg.Enabled {
		t.Error("postgres should be disabled after override")
	}
}

func TestManagerSetCheckAutoHeal(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	schemaPath := filepath.Join(tmpDir, "schema.json")

	mgr := NewManager(configPath, schemaPath)
	mgr.Load()

	// Initially should be default (true for postgres)
	if !mgr.IsAutoHealEnabled("resource-postgres") {
		t.Error("postgres should have autoHeal enabled by default")
	}

	// Disable autoHeal
	if err := mgr.SetCheckAutoHeal("resource-postgres", false); err != nil {
		t.Fatalf("SetCheckAutoHeal failed: %v", err)
	}

	if mgr.IsAutoHealEnabled("resource-postgres") {
		t.Error("postgres autoHeal should be disabled after override")
	}

	// Re-enable
	if err := mgr.SetCheckAutoHeal("resource-postgres", true); err != nil {
		t.Fatalf("SetCheckAutoHeal failed: %v", err)
	}

	if !mgr.IsAutoHealEnabled("resource-postgres") {
		t.Error("postgres autoHeal should be enabled after re-enable")
	}
}

func TestManagerExportImport(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(filepath.Join(tmpDir, "config.json"), filepath.Join(tmpDir, "schema.json"))
	mgr.Load()

	// Modify config
	mgr.SetCheckEnabled("infra-network", false)

	// Export
	data, err := mgr.Export()
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Import into new manager
	mgr2 := NewManager(filepath.Join(tmpDir, "config2.json"), filepath.Join(tmpDir, "schema.json"))
	mgr2.Load()

	imported, err := mgr2.Import(data)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify import
	check, exists := imported.Checks["infra-network"]
	if !exists {
		t.Error("imported config should have infra-network check")
	} else if check.Enabled == nil || *check.Enabled {
		t.Error("imported infra-network should be disabled")
	}
}
