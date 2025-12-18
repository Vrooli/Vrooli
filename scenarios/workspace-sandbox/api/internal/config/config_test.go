package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	t.Run("Server defaults", func(t *testing.T) {
		if cfg.Server.ReadTimeout != 30*time.Second {
			t.Errorf("expected ReadTimeout 30s, got %v", cfg.Server.ReadTimeout)
		}
		if cfg.Server.WriteTimeout != 30*time.Second {
			t.Errorf("expected WriteTimeout 30s, got %v", cfg.Server.WriteTimeout)
		}
		if cfg.Server.IdleTimeout != 120*time.Second {
			t.Errorf("expected IdleTimeout 120s, got %v", cfg.Server.IdleTimeout)
		}
		if cfg.Server.ShutdownTimeout != 10*time.Second {
			t.Errorf("expected ShutdownTimeout 10s, got %v", cfg.Server.ShutdownTimeout)
		}
		if cfg.Server.CORSAllowedOrigins != nil {
			t.Errorf("expected nil CORSAllowedOrigins, got %v", cfg.Server.CORSAllowedOrigins)
		}
	})

	t.Run("Limits defaults", func(t *testing.T) {
		if cfg.Limits.MaxSandboxes != 1000 {
			t.Errorf("expected MaxSandboxes 1000, got %d", cfg.Limits.MaxSandboxes)
		}
		if cfg.Limits.MaxSandboxSizeMB != 10240 {
			t.Errorf("expected MaxSandboxSizeMB 10240, got %d", cfg.Limits.MaxSandboxSizeMB)
		}
		if cfg.Limits.MaxTotalSizeMB != 102400 {
			t.Errorf("expected MaxTotalSizeMB 102400, got %d", cfg.Limits.MaxTotalSizeMB)
		}
		if cfg.Limits.DefaultListLimit != 100 {
			t.Errorf("expected DefaultListLimit 100, got %d", cfg.Limits.DefaultListLimit)
		}
		if cfg.Limits.MaxListLimit != 1000 {
			t.Errorf("expected MaxListLimit 1000, got %d", cfg.Limits.MaxListLimit)
		}
	})

	t.Run("Lifecycle defaults", func(t *testing.T) {
		if cfg.Lifecycle.DefaultTTL != 24*time.Hour {
			t.Errorf("expected DefaultTTL 24h, got %v", cfg.Lifecycle.DefaultTTL)
		}
		if cfg.Lifecycle.IdleTimeout != 4*time.Hour {
			t.Errorf("expected IdleTimeout 4h, got %v", cfg.Lifecycle.IdleTimeout)
		}
		if cfg.Lifecycle.GCInterval != 15*time.Minute {
			t.Errorf("expected GCInterval 15m, got %v", cfg.Lifecycle.GCInterval)
		}
		if !cfg.Lifecycle.AutoCleanupTerminal {
			t.Error("expected AutoCleanupTerminal true")
		}
		if cfg.Lifecycle.TerminalCleanupDelay != 1*time.Hour {
			t.Errorf("expected TerminalCleanupDelay 1h, got %v", cfg.Lifecycle.TerminalCleanupDelay)
		}
	})

	t.Run("Policy defaults", func(t *testing.T) {
		if !cfg.Policy.RequireHumanApproval {
			t.Error("expected RequireHumanApproval true")
		}
		if cfg.Policy.AutoApproveThresholdFiles != 10 {
			t.Errorf("expected AutoApproveThresholdFiles 10, got %d", cfg.Policy.AutoApproveThresholdFiles)
		}
		if cfg.Policy.AutoApproveThresholdLines != 500 {
			t.Errorf("expected AutoApproveThresholdLines 500, got %d", cfg.Policy.AutoApproveThresholdLines)
		}
		if cfg.Policy.CommitMessageTemplate != "Apply sandbox changes ({{.FileCount}} files)" {
			t.Errorf("unexpected CommitMessageTemplate: %s", cfg.Policy.CommitMessageTemplate)
		}
		if cfg.Policy.CommitAuthorMode != "agent" {
			t.Errorf("expected CommitAuthorMode 'agent', got %s", cfg.Policy.CommitAuthorMode)
		}
	})

	t.Run("Driver defaults", func(t *testing.T) {
		// BaseDir should use XDG data directory (~/.local/share/workspace-sandbox)
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("failed to get home dir: %v", err)
		}
		expectedBaseDir := filepath.Join(home, ".local", "share", "workspace-sandbox")
		if cfg.Driver.BaseDir != expectedBaseDir {
			t.Errorf("expected BaseDir %s, got %s", expectedBaseDir, cfg.Driver.BaseDir)
		}
		if cfg.Driver.UseFuseOverlayfs {
			t.Error("expected UseFuseOverlayfs false")
		}
	})

	t.Run("Database defaults", func(t *testing.T) {
		if cfg.Database.Schema != "workspace-sandbox" {
			t.Errorf("expected Schema 'workspace-sandbox', got %s", cfg.Database.Schema)
		}
		if cfg.Database.SSLMode != "disable" {
			t.Errorf("expected SSLMode 'disable', got %s", cfg.Database.SSLMode)
		}
	})
}

func TestLoadFromEnv(t *testing.T) {
	// Save original environment and restore after test
	originalEnv := map[string]string{}
	envVars := []string{
		"API_PORT", "WORKSPACE_SANDBOX_READ_TIMEOUT", "WORKSPACE_SANDBOX_WRITE_TIMEOUT",
		"WORKSPACE_SANDBOX_MAX_SANDBOXES", "WORKSPACE_SANDBOX_CORS_ORIGINS",
		"WORKSPACE_SANDBOX_USE_FUSE", "WORKSPACE_SANDBOX_DEFAULT_TTL",
		"WORKSPACE_SANDBOX_COMMIT_TEMPLATE", "WORKSPACE_SANDBOX_COMMIT_AUTHOR_MODE",
		"PROJECT_ROOT", "SANDBOX_BASE_DIR", "DATABASE_URL",
		"POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD",
		"POSTGRES_DB", "POSTGRES_SCHEMA",
	}
	for _, key := range envVars {
		originalEnv[key] = os.Getenv(key)
	}
	defer func() {
		for key, val := range originalEnv {
			if val == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, val)
			}
		}
	}()

	// Clear all environment variables first
	for _, key := range envVars {
		os.Unsetenv(key)
	}

	t.Run("fails without required API_PORT", func(t *testing.T) {
		_, err := LoadFromEnv()
		if err == nil {
			t.Error("expected error without API_PORT")
		}
		if !strings.Contains(err.Error(), "API_PORT") {
			t.Errorf("error should mention API_PORT: %v", err)
		}
	})

	t.Run("loads with minimum required config", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		cfg, err := LoadFromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Server.Port != "8080" {
			t.Errorf("expected Port 8080, got %s", cfg.Server.Port)
		}
		os.Unsetenv("API_PORT")
	})

	t.Run("loads optional server config", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		os.Setenv("WORKSPACE_SANDBOX_READ_TIMEOUT", "60s")
		os.Setenv("WORKSPACE_SANDBOX_WRITE_TIMEOUT", "45s")
		os.Setenv("WORKSPACE_SANDBOX_CORS_ORIGINS", "http://localhost,http://example.com")

		cfg, err := LoadFromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Server.ReadTimeout != 60*time.Second {
			t.Errorf("expected ReadTimeout 60s, got %v", cfg.Server.ReadTimeout)
		}
		if cfg.Server.WriteTimeout != 45*time.Second {
			t.Errorf("expected WriteTimeout 45s, got %v", cfg.Server.WriteTimeout)
		}
		if len(cfg.Server.CORSAllowedOrigins) != 2 {
			t.Errorf("expected 2 CORS origins, got %d", len(cfg.Server.CORSAllowedOrigins))
		}
	})

	t.Run("loads limits config", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		os.Setenv("WORKSPACE_SANDBOX_MAX_SANDBOXES", "500")

		cfg, err := LoadFromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Limits.MaxSandboxes != 500 {
			t.Errorf("expected MaxSandboxes 500, got %d", cfg.Limits.MaxSandboxes)
		}
	})

	t.Run("loads driver config", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		os.Setenv("SANDBOX_BASE_DIR", "/custom/path")
		os.Setenv("WORKSPACE_SANDBOX_USE_FUSE", "true")
		os.Setenv("PROJECT_ROOT", "/my/project")

		cfg, err := LoadFromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Driver.BaseDir != "/custom/path" {
			t.Errorf("expected BaseDir /custom/path, got %s", cfg.Driver.BaseDir)
		}
		if !cfg.Driver.UseFuseOverlayfs {
			t.Error("expected UseFuseOverlayfs true")
		}
		if cfg.Driver.ProjectRoot != "/my/project" {
			t.Errorf("expected ProjectRoot /my/project, got %s", cfg.Driver.ProjectRoot)
		}
	})

	t.Run("loads policy config", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		os.Setenv("WORKSPACE_SANDBOX_COMMIT_TEMPLATE", "Custom: {{.FileCount}}")
		os.Setenv("WORKSPACE_SANDBOX_COMMIT_AUTHOR_MODE", "coauthored")

		cfg, err := LoadFromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Policy.CommitMessageTemplate != "Custom: {{.FileCount}}" {
			t.Errorf("unexpected CommitMessageTemplate: %s", cfg.Policy.CommitMessageTemplate)
		}
		if cfg.Policy.CommitAuthorMode != "coauthored" {
			t.Errorf("expected CommitAuthorMode 'coauthored', got %s", cfg.Policy.CommitAuthorMode)
		}
	})

	t.Run("loads database config", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		os.Setenv("DATABASE_URL", "postgres://localhost/test")
		os.Setenv("POSTGRES_HOST", "db.example.com")
		os.Setenv("POSTGRES_SCHEMA", "custom_schema")

		cfg, err := LoadFromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Database.URL != "postgres://localhost/test" {
			t.Errorf("expected DATABASE_URL, got %s", cfg.Database.URL)
		}
		if cfg.Database.Host != "db.example.com" {
			t.Errorf("expected Host db.example.com, got %s", cfg.Database.Host)
		}
		if cfg.Database.Schema != "custom_schema" {
			t.Errorf("expected Schema custom_schema, got %s", cfg.Database.Schema)
		}
	})
}

func TestValidate(t *testing.T) {
	t.Run("valid config passes", func(t *testing.T) {
		cfg := Default()
		cfg.Server.Port = "8080"
		if err := cfg.Validate(); err != nil {
			t.Errorf("expected valid config to pass: %v", err)
		}
	})

	t.Run("missing port fails", func(t *testing.T) {
		cfg := Default()
		cfg.Server.Port = ""
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for missing port")
		}
		if !strings.Contains(err.Error(), "server.port") {
			t.Errorf("error should mention server.port: %v", err)
		}
	})

	t.Run("ReadTimeout too low fails", func(t *testing.T) {
		cfg := Default()
		cfg.Server.Port = "8080"
		cfg.Server.ReadTimeout = 500 * time.Millisecond
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for low ReadTimeout")
		}
		if !strings.Contains(err.Error(), "readTimeout") {
			t.Errorf("error should mention readTimeout: %v", err)
		}
	})

	t.Run("WriteTimeout too low fails", func(t *testing.T) {
		cfg := Default()
		cfg.Server.Port = "8080"
		cfg.Server.WriteTimeout = 500 * time.Millisecond
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for low WriteTimeout")
		}
	})

	t.Run("MaxSandboxes below 1 fails", func(t *testing.T) {
		cfg := Default()
		cfg.Server.Port = "8080"
		cfg.Limits.MaxSandboxes = 0
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for MaxSandboxes 0")
		}
		if !strings.Contains(err.Error(), "maxSandboxes") {
			t.Errorf("error should mention maxSandboxes: %v", err)
		}
	})

	t.Run("MaxSandboxes above safe limit fails", func(t *testing.T) {
		cfg := Default()
		cfg.Server.Port = "8080"
		cfg.Limits.MaxSandboxes = 200000
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for MaxSandboxes > 100000")
		}
	})

	t.Run("MaxSandboxSizeMB below 1 fails", func(t *testing.T) {
		cfg := Default()
		cfg.Server.Port = "8080"
		cfg.Limits.MaxSandboxSizeMB = 0
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for MaxSandboxSizeMB 0")
		}
	})

	t.Run("DefaultListLimit out of range fails", func(t *testing.T) {
		cfg := Default()
		cfg.Server.Port = "8080"
		cfg.Limits.DefaultListLimit = 0
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for DefaultListLimit 0")
		}

		cfg = Default()
		cfg.Server.Port = "8080"
		cfg.Limits.DefaultListLimit = cfg.Limits.MaxListLimit + 1
		err = cfg.Validate()
		if err == nil {
			t.Error("expected error for DefaultListLimit > MaxListLimit")
		}
	})

	t.Run("DefaultTTL too low fails", func(t *testing.T) {
		cfg := Default()
		cfg.Server.Port = "8080"
		cfg.Lifecycle.DefaultTTL = 30 * time.Second
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for low DefaultTTL")
		}
	})

	t.Run("GCInterval too low fails", func(t *testing.T) {
		cfg := Default()
		cfg.Server.Port = "8080"
		cfg.Lifecycle.GCInterval = 30 * time.Second
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for low GCInterval")
		}
	})

	t.Run("invalid CommitAuthorMode fails", func(t *testing.T) {
		cfg := Default()
		cfg.Server.Port = "8080"
		cfg.Policy.CommitAuthorMode = "invalid"
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for invalid CommitAuthorMode")
		}
		if !strings.Contains(err.Error(), "commitAuthorMode") {
			t.Errorf("error should mention commitAuthorMode: %v", err)
		}
	})

	t.Run("valid CommitAuthorMode values pass", func(t *testing.T) {
		for _, mode := range []string{"agent", "reviewer", "coauthored"} {
			cfg := Default()
			cfg.Server.Port = "8080"
			cfg.Policy.CommitAuthorMode = mode
			if err := cfg.Validate(); err != nil {
				t.Errorf("expected mode %q to be valid: %v", mode, err)
			}
		}
	})

	t.Run("missing BaseDir fails", func(t *testing.T) {
		cfg := Default()
		cfg.Server.Port = "8080"
		cfg.Driver.BaseDir = ""
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for missing BaseDir")
		}
	})
}

func TestEnvHelpers(t *testing.T) {
	t.Run("envInt returns default for empty", func(t *testing.T) {
		os.Unsetenv("TEST_INT")
		if got := envInt("TEST_INT", 42); got != 42 {
			t.Errorf("expected 42, got %d", got)
		}
	})

	t.Run("envInt returns default for invalid", func(t *testing.T) {
		os.Setenv("TEST_INT", "notanumber")
		defer os.Unsetenv("TEST_INT")
		if got := envInt("TEST_INT", 42); got != 42 {
			t.Errorf("expected 42 for invalid, got %d", got)
		}
	})

	t.Run("envInt parses valid value", func(t *testing.T) {
		os.Setenv("TEST_INT", "123")
		defer os.Unsetenv("TEST_INT")
		if got := envInt("TEST_INT", 42); got != 123 {
			t.Errorf("expected 123, got %d", got)
		}
	})

	t.Run("envDuration returns default for empty", func(t *testing.T) {
		os.Unsetenv("TEST_DUR")
		if got := envDuration("TEST_DUR", 5*time.Minute); got != 5*time.Minute {
			t.Errorf("expected 5m, got %v", got)
		}
	})

	t.Run("envDuration returns default for invalid", func(t *testing.T) {
		os.Setenv("TEST_DUR", "notaduration")
		defer os.Unsetenv("TEST_DUR")
		if got := envDuration("TEST_DUR", 5*time.Minute); got != 5*time.Minute {
			t.Errorf("expected 5m for invalid, got %v", got)
		}
	})

	t.Run("envDuration parses valid value", func(t *testing.T) {
		os.Setenv("TEST_DUR", "30s")
		defer os.Unsetenv("TEST_DUR")
		if got := envDuration("TEST_DUR", 5*time.Minute); got != 30*time.Second {
			t.Errorf("expected 30s, got %v", got)
		}
	})

	t.Run("envBool returns default for empty", func(t *testing.T) {
		os.Unsetenv("TEST_BOOL")
		if got := envBool("TEST_BOOL", true); !got {
			t.Error("expected true for empty with default true")
		}
		if got := envBool("TEST_BOOL", false); got {
			t.Error("expected false for empty with default false")
		}
	})

	t.Run("envBool parses true values", func(t *testing.T) {
		for _, val := range []string{"true", "1", "yes", "on", "TRUE", "Yes", "ON"} {
			os.Setenv("TEST_BOOL", val)
			if got := envBool("TEST_BOOL", false); !got {
				t.Errorf("expected true for %q", val)
			}
		}
		os.Unsetenv("TEST_BOOL")
	})

	t.Run("envBool parses false values", func(t *testing.T) {
		for _, val := range []string{"false", "0", "no", "off", "FALSE", "No", "OFF"} {
			os.Setenv("TEST_BOOL", val)
			if got := envBool("TEST_BOOL", true); got {
				t.Errorf("expected false for %q", val)
			}
		}
		os.Unsetenv("TEST_BOOL")
	})

	t.Run("envBool returns default for unknown value", func(t *testing.T) {
		os.Setenv("TEST_BOOL", "maybe")
		defer os.Unsetenv("TEST_BOOL")
		if got := envBool("TEST_BOOL", true); !got {
			t.Error("expected default true for unknown value")
		}
		if got := envBool("TEST_BOOL", false); got {
			t.Error("expected default false for unknown value")
		}
	})
}

func TestRequireEnv(t *testing.T) {
	t.Run("returns value when set", func(t *testing.T) {
		os.Setenv("TEST_REQUIRED", "myvalue")
		defer os.Unsetenv("TEST_REQUIRED")
		var errs []string
		val := requireEnv("TEST_REQUIRED", &errs)
		if val != "myvalue" {
			t.Errorf("expected 'myvalue', got %q", val)
		}
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %v", errs)
		}
	})

	t.Run("adds error when missing", func(t *testing.T) {
		os.Unsetenv("TEST_REQUIRED")
		var errs []string
		val := requireEnv("TEST_REQUIRED", &errs)
		if val != "" {
			t.Errorf("expected empty string, got %q", val)
		}
		if len(errs) != 1 || errs[0] != "TEST_REQUIRED" {
			t.Errorf("expected [TEST_REQUIRED], got %v", errs)
		}
	})

	t.Run("trims whitespace", func(t *testing.T) {
		os.Setenv("TEST_REQUIRED", "  trimmed  ")
		defer os.Unsetenv("TEST_REQUIRED")
		var errs []string
		val := requireEnv("TEST_REQUIRED", &errs)
		if val != "trimmed" {
			t.Errorf("expected 'trimmed', got %q", val)
		}
	})

	t.Run("treats whitespace-only as missing", func(t *testing.T) {
		os.Setenv("TEST_REQUIRED", "   ")
		defer os.Unsetenv("TEST_REQUIRED")
		var errs []string
		val := requireEnv("TEST_REQUIRED", &errs)
		if val != "" {
			t.Errorf("expected empty string, got %q", val)
		}
		if len(errs) != 1 {
			t.Errorf("expected error for whitespace-only value")
		}
	})
}
