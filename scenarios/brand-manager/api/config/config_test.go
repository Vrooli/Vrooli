package config

import (
	"os"
	"testing"
)

func TestDefaultValues(t *testing.T) {
	cfg := Default()

	if cfg.BusyTimeoutMS != 10000 {
		t.Errorf("BusyTimeoutMS = %d, want 10000", cfg.BusyTimeoutMS)
	}
	if cfg.CacheSizeKB != 2000 {
		t.Errorf("CacheSizeKB = %d, want 2000", cfg.CacheSizeKB)
	}
	if cfg.MaxOpenConns != 1 {
		t.Errorf("MaxOpenConns = %d, want 1", cfg.MaxOpenConns)
	}
	if cfg.APIVersion != "1.0.0" {
		t.Errorf("APIVersion = %q, want %q", cfg.APIVersion, "1.0.0")
	}
	if cfg.DefaultListLimit != 100 {
		t.Errorf("DefaultListLimit = %d, want 100", cfg.DefaultListLimit)
	}
	if cfg.MaxListLimit != 1000 {
		t.Errorf("MaxListLimit = %d, want 1000", cfg.MaxListLimit)
	}
	if cfg.ContrastAANormal != 4.5 {
		t.Errorf("ContrastAANormal = %f, want 4.5", cfg.ContrastAANormal)
	}
	if cfg.ContrastAALarge != 3.0 {
		t.Errorf("ContrastAALarge = %f, want 3.0", cfg.ContrastAALarge)
	}
	if cfg.ContrastPrecision != 2 {
		t.Errorf("ContrastPrecision = %d, want 2", cfg.ContrastPrecision)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("BM_BUSY_TIMEOUT_MS", "5000")
	t.Setenv("BM_CACHE_SIZE_KB", "4000")
	t.Setenv("BM_DEFAULT_LIST_LIMIT", "50")
	t.Setenv("BM_MAX_LIST_LIMIT", "500")
	t.Setenv("BM_CONTRAST_AA_NORMAL", "7.0")
	t.Setenv("BM_CONTRAST_PRECISION", "3")

	cfg := Load()

	if cfg.BusyTimeoutMS != 5000 {
		t.Errorf("BusyTimeoutMS = %d, want 5000", cfg.BusyTimeoutMS)
	}
	if cfg.CacheSizeKB != 4000 {
		t.Errorf("CacheSizeKB = %d, want 4000", cfg.CacheSizeKB)
	}
	if cfg.DefaultListLimit != 50 {
		t.Errorf("DefaultListLimit = %d, want 50", cfg.DefaultListLimit)
	}
	if cfg.MaxListLimit != 500 {
		t.Errorf("MaxListLimit = %d, want 500", cfg.MaxListLimit)
	}
	if cfg.ContrastAANormal != 7.0 {
		t.Errorf("ContrastAANormal = %f, want 7.0", cfg.ContrastAANormal)
	}
	if cfg.ContrastPrecision != 3 {
		t.Errorf("ContrastPrecision = %d, want 3", cfg.ContrastPrecision)
	}
}

func TestLoadDBPathResolutionChain(t *testing.T) {
	// Clear all DB path vars first
	for _, key := range []string{"BM_SQLITE_PATH", "SQLITE_PATH", "SQLITE_DB"} {
		os.Unsetenv(key)
	}

	// BM_SQLITE_PATH takes priority
	t.Setenv("BM_SQLITE_PATH", "/custom/path.db")
	t.Setenv("SQLITE_PATH", "/fallback/path.db")
	cfg := Load()
	if cfg.SQLitePath != "/custom/path.db" {
		t.Errorf("SQLitePath = %q, want /custom/path.db", cfg.SQLitePath)
	}

	// SQLITE_PATH is second priority
	os.Unsetenv("BM_SQLITE_PATH")
	cfg = Load()
	if cfg.SQLitePath != "/fallback/path.db" {
		t.Errorf("SQLitePath = %q, want /fallback/path.db", cfg.SQLitePath)
	}
}

func TestLoadInvalidEnvFallsBackToDefault(t *testing.T) {
	t.Setenv("BM_BUSY_TIMEOUT_MS", "not-a-number")
	t.Setenv("BM_CONTRAST_AA_NORMAL", "bad")

	cfg := Load()

	if cfg.BusyTimeoutMS != 10000 {
		t.Errorf("BusyTimeoutMS = %d, want 10000 (default on bad env)", cfg.BusyTimeoutMS)
	}
	if cfg.ContrastAANormal != 4.5 {
		t.Errorf("ContrastAANormal = %f, want 4.5 (default on bad env)", cfg.ContrastAANormal)
	}
}

func TestValidationGuardrails(t *testing.T) {
	cfg := Config{
		BusyTimeoutMS:     -1,
		CacheSizeKB:       10,
		MaxOpenConns:      0,
		DefaultListLimit:  0,
		MaxListLimit:      0,
		ContrastAANormal:  0.5,
		ContrastAALarge:   0.2,
		ContrastPrecision: -1,
	}

	v := cfg.validated()

	if v.BusyTimeoutMS != 0 {
		t.Errorf("BusyTimeoutMS = %d, want 0 (clamped from -1)", v.BusyTimeoutMS)
	}
	if v.CacheSizeKB != 64 {
		t.Errorf("CacheSizeKB = %d, want 64 (min)", v.CacheSizeKB)
	}
	if v.MaxOpenConns != 1 {
		t.Errorf("MaxOpenConns = %d, want 1 (min)", v.MaxOpenConns)
	}
	if v.DefaultListLimit != 1 {
		t.Errorf("DefaultListLimit = %d, want 1 (min)", v.DefaultListLimit)
	}
	if v.MaxListLimit != 1 {
		t.Errorf("MaxListLimit = %d, want 1 (clamped to DefaultListLimit)", v.MaxListLimit)
	}
	if v.ContrastAANormal != 1.0 {
		t.Errorf("ContrastAANormal = %f, want 1.0 (min)", v.ContrastAANormal)
	}
	if v.ContrastAALarge != 1.0 {
		t.Errorf("ContrastAALarge = %f, want 1.0 (min)", v.ContrastAALarge)
	}
	if v.ContrastPrecision != 0 {
		t.Errorf("ContrastPrecision = %d, want 0 (min)", v.ContrastPrecision)
	}
}

func TestValidationPrecisionMax(t *testing.T) {
	cfg := Config{
		BusyTimeoutMS:     10000,
		CacheSizeKB:       2000,
		MaxOpenConns:      1,
		DefaultListLimit:  100,
		MaxListLimit:      1000,
		ContrastAANormal:  4.5,
		ContrastAALarge:   3.0,
		ContrastPrecision: 10,
	}
	v := cfg.validated()
	if v.ContrastPrecision != 6 {
		t.Errorf("ContrastPrecision = %d, want 6 (max)", v.ContrastPrecision)
	}
}

func TestDSN(t *testing.T) {
	cfg := Config{
		SQLitePath:    "/tmp/test.db",
		BusyTimeoutMS: 5000,
		CacheSizeKB:   4000,
	}

	dsn := cfg.DSN()

	if dsn == "" {
		t.Fatal("DSN is empty")
	}
	// Verify key pragma values are embedded
	want := "busy_timeout(5000)"
	if !contains(dsn, want) {
		t.Errorf("DSN missing %q: %s", want, dsn)
	}
	want = "cache_size(-4000)"
	if !contains(dsn, want) {
		t.Errorf("DSN missing %q: %s", want, dsn)
	}
}

func TestClampLimit(t *testing.T) {
	cfg := Config{DefaultListLimit: 100, MaxListLimit: 1000}

	tests := []struct {
		input int
		want  int
	}{
		{0, 100},     // zero → default
		{-5, 100},    // negative → default
		{50, 50},     // within range → pass through
		{100, 100},   // exactly default → pass through
		{1000, 1000}, // exactly max → pass through
		{2000, 1000}, // above max → capped
	}

	for _, tt := range tests {
		got := cfg.ClampLimit(tt.input)
		if got != tt.want {
			t.Errorf("ClampLimit(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
