package config

import (
	"os"
	"testing"
	"time"
)

// TestDefaultDatabaseConfig verifies database defaults.
func TestDefaultDatabaseConfig(t *testing.T) {
	cfg := DefaultDatabaseConfig()

	if cfg.MaxOpenConns != 1 {
		t.Errorf("Expected MaxOpenConns=1, got %d", cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns != 1 {
		t.Errorf("Expected MaxIdleConns=1, got %d", cfg.MaxIdleConns)
	}
	if cfg.BusyTimeout != 5*time.Second {
		t.Errorf("Expected BusyTimeout=5s, got %v", cfg.BusyTimeout)
	}
	if cfg.RetryMaxAttempts != 3 {
		t.Errorf("Expected RetryMaxAttempts=3, got %d", cfg.RetryMaxAttempts)
	}
}

// TestDefaultQueryConfig verifies query defaults.
func TestDefaultQueryConfig(t *testing.T) {
	cfg := DefaultQueryConfig()

	if cfg.DefaultEventLimit != 100 {
		t.Errorf("Expected DefaultEventLimit=100, got %d", cfg.DefaultEventLimit)
	}
	if cfg.MaxEventLimit != 1000 {
		t.Errorf("Expected MaxEventLimit=1000, got %d", cfg.MaxEventLimit)
	}
	if cfg.DefaultTimelineDays != 7 {
		t.Errorf("Expected DefaultTimelineDays=7, got %d", cfg.DefaultTimelineDays)
	}
	if cfg.MaxTimelineDays != 365 {
		t.Errorf("Expected MaxTimelineDays=365, got %d", cfg.MaxTimelineDays)
	}
}

// TestQueryConfig_EnvOverride verifies environment variable overrides.
func TestQueryConfig_EnvOverride(t *testing.T) {
	// Set environment variables
	os.Setenv("LD_DEFAULT_EVENT_LIMIT", "50")
	os.Setenv("LD_DEFAULT_TIMELINE_DAYS", "14")
	defer func() {
		os.Unsetenv("LD_DEFAULT_EVENT_LIMIT")
		os.Unsetenv("LD_DEFAULT_TIMELINE_DAYS")
	}()

	cfg := DefaultQueryConfig()

	if cfg.DefaultEventLimit != 50 {
		t.Errorf("Expected DefaultEventLimit=50 from env, got %d", cfg.DefaultEventLimit)
	}
	if cfg.DefaultTimelineDays != 14 {
		t.Errorf("Expected DefaultTimelineDays=14 from env, got %d", cfg.DefaultTimelineDays)
	}
}

// TestQueryConfig_InvalidEnv verifies invalid env values are ignored.
func TestQueryConfig_InvalidEnv(t *testing.T) {
	os.Setenv("LD_DEFAULT_EVENT_LIMIT", "invalid")
	os.Setenv("LD_DEFAULT_TIMELINE_DAYS", "-5")
	defer func() {
		os.Unsetenv("LD_DEFAULT_EVENT_LIMIT")
		os.Unsetenv("LD_DEFAULT_TIMELINE_DAYS")
	}()

	cfg := DefaultQueryConfig()

	// Should fall back to defaults on invalid values
	if cfg.DefaultEventLimit != 100 {
		t.Errorf("Expected DefaultEventLimit=100 (fallback), got %d", cfg.DefaultEventLimit)
	}
	// Negative values should be ignored
	if cfg.DefaultTimelineDays != 7 {
		t.Errorf("Expected DefaultTimelineDays=7 (fallback), got %d", cfg.DefaultTimelineDays)
	}
}

// TestDefaultHealthCheckConfig verifies health check defaults.
func TestDefaultHealthCheckConfig(t *testing.T) {
	cfg := DefaultHealthCheckConfig()

	if cfg.Timeout != 5*time.Second {
		t.Errorf("Expected Timeout=5s, got %v", cfg.Timeout)
	}
	if cfg.UnhealthyThreshold != 300 {
		t.Errorf("Expected UnhealthyThreshold=300, got %d", cfg.UnhealthyThreshold)
	}
}

// TestHealthCheckConfig_EnvOverride verifies timeout env override.
func TestHealthCheckConfig_EnvOverride(t *testing.T) {
	os.Setenv("LD_HEALTH_CHECK_TIMEOUT_SECS", "10")
	defer os.Unsetenv("LD_HEALTH_CHECK_TIMEOUT_SECS")

	cfg := DefaultHealthCheckConfig()

	if cfg.Timeout != 10*time.Second {
		t.Errorf("Expected Timeout=10s from env, got %v", cfg.Timeout)
	}
}

// TestDefaultCORSConfig verifies CORS defaults.
func TestDefaultCORSConfig(t *testing.T) {
	cfg := DefaultCORSConfig()

	if len(cfg.AllowedOrigins) != 0 {
		t.Errorf("Expected empty AllowedOrigins, got %v", cfg.AllowedOrigins)
	}
	if !cfg.AllowCredentials {
		t.Error("Expected AllowCredentials=true")
	}
}

// TestLoad verifies full config loading.
func TestLoad(t *testing.T) {
	cfg := Load()

	// Verify all sections are populated
	if cfg.Database.MaxOpenConns == 0 {
		t.Error("Database config not loaded")
	}
	if cfg.Query.DefaultEventLimit == 0 {
		t.Error("Query config not loaded")
	}
	if cfg.HealthCheck.Timeout == 0 {
		t.Error("HealthCheck config not loaded")
	}
	if !cfg.CORS.AllowCredentials {
		t.Error("CORS config not loaded")
	}
}
