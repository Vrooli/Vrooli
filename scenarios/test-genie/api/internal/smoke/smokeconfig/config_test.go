package smokeconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultUISmokeConfig(t *testing.T) {
	cfg := DefaultUISmokeConfig()

	if !cfg.Enabled {
		t.Error("expected Enabled to default to true")
	}
	if cfg.TimeoutMs != 0 {
		t.Errorf("expected TimeoutMs to default to 0, got %d", cfg.TimeoutMs)
	}
	if cfg.HandshakeTimeoutMs != 0 {
		t.Errorf("expected HandshakeTimeoutMs to default to 0, got %d", cfg.HandshakeTimeoutMs)
	}
	if len(cfg.HandshakeSignals) != 0 {
		t.Errorf("expected HandshakeSignals to default to empty, got %v", cfg.HandshakeSignals)
	}
}

func TestLoadUISmokeConfig_FileNotExist(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := LoadUISmokeConfig(tmpDir)

	// Should return defaults when file doesn't exist
	if !cfg.Enabled {
		t.Error("expected Enabled to default to true when file missing")
	}
}

func TestLoadUISmokeConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	// Write invalid JSON
	configPath := filepath.Join(vrooliDir, "testing.json")
	if err := os.WriteFile(configPath, []byte("not valid json{"), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg := LoadUISmokeConfig(tmpDir)

	// Should return defaults when JSON is invalid
	if !cfg.Enabled {
		t.Error("expected Enabled to default to true when JSON invalid")
	}
}

func TestLoadUISmokeConfig_EmptyStructure(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	// Write valid JSON but without ui_smoke section
	configPath := filepath.Join(vrooliDir, "testing.json")
	if err := os.WriteFile(configPath, []byte(`{"structure": {}}`), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg := LoadUISmokeConfig(tmpDir)

	// Should return defaults when ui_smoke section missing
	if !cfg.Enabled {
		t.Error("expected Enabled to default to true when section missing")
	}
}

func TestLoadUISmokeConfig_EnabledFalse(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	configPath := filepath.Join(vrooliDir, "testing.json")
	content := `{"structure": {"ui_smoke": {"enabled": false}}}`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg := LoadUISmokeConfig(tmpDir)

	if cfg.Enabled {
		t.Error("expected Enabled to be false when explicitly set")
	}
}

func TestLoadUISmokeConfig_EnabledTrue(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	configPath := filepath.Join(vrooliDir, "testing.json")
	content := `{"structure": {"ui_smoke": {"enabled": true}}}`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg := LoadUISmokeConfig(tmpDir)

	if !cfg.Enabled {
		t.Error("expected Enabled to be true when explicitly set")
	}
}

func TestLoadUISmokeConfig_CustomTimeouts(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	configPath := filepath.Join(vrooliDir, "testing.json")
	content := `{
		"structure": {
			"ui_smoke": {
				"timeout_ms": 30000,
				"handshake_timeout_ms": 5000
			}
		}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg := LoadUISmokeConfig(tmpDir)

	if cfg.TimeoutMs != 30000 {
		t.Errorf("expected TimeoutMs=30000, got %d", cfg.TimeoutMs)
	}
	if cfg.HandshakeTimeoutMs != 5000 {
		t.Errorf("expected HandshakeTimeoutMs=5000, got %d", cfg.HandshakeTimeoutMs)
	}
}

func TestLoadUISmokeConfig_ZeroTimeoutsIgnored(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	configPath := filepath.Join(vrooliDir, "testing.json")
	// Zero values should not override defaults (though defaults are also 0)
	content := `{
		"structure": {
			"ui_smoke": {
				"timeout_ms": 0,
				"handshake_timeout_ms": 0
			}
		}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg := LoadUISmokeConfig(tmpDir)

	// Zero values should be ignored, keeping defaults
	if cfg.TimeoutMs != 0 {
		t.Errorf("expected TimeoutMs=0 (default), got %d", cfg.TimeoutMs)
	}
}

func TestLoadUISmokeConfig_CustomHandshakeSignals(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	configPath := filepath.Join(vrooliDir, "testing.json")
	content := `{
		"structure": {
			"ui_smoke": {
				"handshake_signals": ["window.APP_READY", "window.HYDRATED"]
			}
		}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg := LoadUISmokeConfig(tmpDir)

	if len(cfg.HandshakeSignals) != 2 {
		t.Fatalf("expected 2 handshake signals, got %d", len(cfg.HandshakeSignals))
	}
	if cfg.HandshakeSignals[0] != "window.APP_READY" {
		t.Errorf("expected first signal 'window.APP_READY', got %s", cfg.HandshakeSignals[0])
	}
	if cfg.HandshakeSignals[1] != "window.HYDRATED" {
		t.Errorf("expected second signal 'window.HYDRATED', got %s", cfg.HandshakeSignals[1])
	}
}

func TestLoadUISmokeConfig_EmptyHandshakeSignalsIgnored(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	configPath := filepath.Join(vrooliDir, "testing.json")
	content := `{
		"structure": {
			"ui_smoke": {
				"handshake_signals": []
			}
		}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg := LoadUISmokeConfig(tmpDir)

	// Empty array should not override defaults
	if len(cfg.HandshakeSignals) != 0 {
		t.Errorf("expected empty handshake signals, got %v", cfg.HandshakeSignals)
	}
}

func TestLoadUISmokeConfig_FullConfig(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	configPath := filepath.Join(vrooliDir, "testing.json")
	content := `{
		"structure": {
			"ui_smoke": {
				"enabled": false,
				"timeout_ms": 60000,
				"handshake_timeout_ms": 10000,
				"handshake_signals": ["window.READY"]
			}
		}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg := LoadUISmokeConfig(tmpDir)

	if cfg.Enabled {
		t.Error("expected Enabled=false")
	}
	if cfg.TimeoutMs != 60000 {
		t.Errorf("expected TimeoutMs=60000, got %d", cfg.TimeoutMs)
	}
	if cfg.HandshakeTimeoutMs != 10000 {
		t.Errorf("expected HandshakeTimeoutMs=10000, got %d", cfg.HandshakeTimeoutMs)
	}
	if len(cfg.HandshakeSignals) != 1 || cfg.HandshakeSignals[0] != "window.READY" {
		t.Errorf("expected HandshakeSignals=['window.READY'], got %v", cfg.HandshakeSignals)
	}
}
