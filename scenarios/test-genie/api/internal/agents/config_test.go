package agents

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	t.Run("lock timeout defaults to 20 minutes", func(t *testing.T) {
		if cfg.LockTimeoutMinutes != 20 {
			t.Errorf("Expected LockTimeoutMinutes=20, got %d", cfg.LockTimeoutMinutes)
		}
	})

	t.Run("heartbeat interval defaults to 5 minutes", func(t *testing.T) {
		if cfg.HeartbeatIntervalMinutes != 5 {
			t.Errorf("Expected HeartbeatIntervalMinutes=5, got %d", cfg.HeartbeatIntervalMinutes)
		}
	})

	t.Run("retention days defaults to 7", func(t *testing.T) {
		if cfg.RetentionDays != 7 {
			t.Errorf("Expected RetentionDays=7, got %d", cfg.RetentionDays)
		}
	})

	t.Run("max concurrent agents defaults to 10", func(t *testing.T) {
		if cfg.MaxConcurrentAgents != 10 {
			t.Errorf("Expected MaxConcurrentAgents=10, got %d", cfg.MaxConcurrentAgents)
		}
	})

	t.Run("default concurrency defaults to 3", func(t *testing.T) {
		if cfg.DefaultConcurrency != 3 {
			t.Errorf("Expected DefaultConcurrency=3, got %d", cfg.DefaultConcurrency)
		}
	})

	// Execution defaults tests
	t.Run("default timeout defaults to 900 seconds (15 min)", func(t *testing.T) {
		if cfg.DefaultTimeoutSeconds != 900 {
			t.Errorf("Expected DefaultTimeoutSeconds=900, got %d", cfg.DefaultTimeoutSeconds)
		}
	})

	t.Run("default max turns defaults to 50", func(t *testing.T) {
		if cfg.DefaultMaxTurns != 50 {
			t.Errorf("Expected DefaultMaxTurns=50, got %d", cfg.DefaultMaxTurns)
		}
	})

	t.Run("default max files changed defaults to 50", func(t *testing.T) {
		if cfg.DefaultMaxFilesChanged != 50 {
			t.Errorf("Expected DefaultMaxFilesChanged=50, got %d", cfg.DefaultMaxFilesChanged)
		}
	})

	t.Run("default max bytes written defaults to 1MB", func(t *testing.T) {
		if cfg.DefaultMaxBytesWritten != 1024*1024 {
			t.Errorf("Expected DefaultMaxBytesWritten=1048576, got %d", cfg.DefaultMaxBytesWritten)
		}
	})

	t.Run("default network enabled defaults to false", func(t *testing.T) {
		if cfg.DefaultNetworkEnabled != false {
			t.Errorf("Expected DefaultNetworkEnabled=false, got %v", cfg.DefaultNetworkEnabled)
		}
	})
}

func TestConfigDurations(t *testing.T) {
	cfg := DefaultConfig()

	t.Run("LockTimeout returns duration", func(t *testing.T) {
		expected := 20 * time.Minute
		if cfg.LockTimeout() != expected {
			t.Errorf("Expected LockTimeout=%v, got %v", expected, cfg.LockTimeout())
		}
	})

	t.Run("HeartbeatInterval returns duration", func(t *testing.T) {
		expected := 5 * time.Minute
		if cfg.HeartbeatInterval() != expected {
			t.Errorf("Expected HeartbeatInterval=%v, got %v", expected, cfg.HeartbeatInterval())
		}
	})

	t.Run("CleanupInterval returns duration", func(t *testing.T) {
		expected := 60 * time.Minute
		if cfg.CleanupInterval() != expected {
			t.Errorf("Expected CleanupInterval=%v, got %v", expected, cfg.CleanupInterval())
		}
	})

	t.Run("RetentionDuration returns duration in days", func(t *testing.T) {
		expected := 7 * 24 * time.Hour
		if cfg.RetentionDuration() != expected {
			t.Errorf("Expected RetentionDuration=%v, got %v", expected, cfg.RetentionDuration())
		}
	})
}

func TestLoadConfigFromEnv(t *testing.T) {
	// Clean up any env vars after test
	defer func() {
		os.Unsetenv("AGENT_LOCK_TIMEOUT_MINUTES")
		os.Unsetenv("AGENT_RETENTION_DAYS")
		os.Unsetenv("AGENT_MAX_PROMPTS")
		os.Unsetenv("AGENT_MAX_CONCURRENT")
		os.Unsetenv("AGENT_DEFAULT_TIMEOUT_SECONDS")
		os.Unsetenv("AGENT_DEFAULT_MAX_TURNS")
		os.Unsetenv("AGENT_DEFAULT_MAX_FILES")
		os.Unsetenv("AGENT_DEFAULT_MAX_BYTES")
		os.Unsetenv("AGENT_DEFAULT_NETWORK_ENABLED")
	}()

	t.Run("loads from environment", func(t *testing.T) {
		os.Setenv("AGENT_LOCK_TIMEOUT_MINUTES", "45")
		os.Setenv("AGENT_RETENTION_DAYS", "30")
		os.Setenv("AGENT_MAX_PROMPTS", "50")
		os.Setenv("AGENT_MAX_CONCURRENT", "15")

		cfg := LoadConfigFromEnv()

		if cfg.LockTimeoutMinutes != 45 {
			t.Errorf("Expected LockTimeoutMinutes=45, got %d", cfg.LockTimeoutMinutes)
		}
		if cfg.RetentionDays != 30 {
			t.Errorf("Expected RetentionDays=30, got %d", cfg.RetentionDays)
		}
		if cfg.MaxPromptsPerSpawn != 50 {
			t.Errorf("Expected MaxPromptsPerSpawn=50, got %d", cfg.MaxPromptsPerSpawn)
		}
		if cfg.MaxConcurrentAgents != 15 {
			t.Errorf("Expected MaxConcurrentAgents=15, got %d", cfg.MaxConcurrentAgents)
		}
	})

	t.Run("loads execution defaults from environment", func(t *testing.T) {
		os.Setenv("AGENT_DEFAULT_TIMEOUT_SECONDS", "1800")
		os.Setenv("AGENT_DEFAULT_MAX_TURNS", "100")
		os.Setenv("AGENT_DEFAULT_MAX_FILES", "200")
		os.Setenv("AGENT_DEFAULT_MAX_BYTES", "5242880") // 5MB
		os.Setenv("AGENT_DEFAULT_NETWORK_ENABLED", "true")

		cfg := LoadConfigFromEnv()

		if cfg.DefaultTimeoutSeconds != 1800 {
			t.Errorf("Expected DefaultTimeoutSeconds=1800, got %d", cfg.DefaultTimeoutSeconds)
		}
		if cfg.DefaultMaxTurns != 100 {
			t.Errorf("Expected DefaultMaxTurns=100, got %d", cfg.DefaultMaxTurns)
		}
		if cfg.DefaultMaxFilesChanged != 200 {
			t.Errorf("Expected DefaultMaxFilesChanged=200, got %d", cfg.DefaultMaxFilesChanged)
		}
		if cfg.DefaultMaxBytesWritten != 5242880 {
			t.Errorf("Expected DefaultMaxBytesWritten=5242880, got %d", cfg.DefaultMaxBytesWritten)
		}
		if !cfg.DefaultNetworkEnabled {
			t.Errorf("Expected DefaultNetworkEnabled=true, got %v", cfg.DefaultNetworkEnabled)
		}
	})

	t.Run("clamps values to valid ranges", func(t *testing.T) {
		os.Setenv("AGENT_LOCK_TIMEOUT_MINUTES", "200") // Max is 120
		os.Setenv("AGENT_RETENTION_DAYS", "500")       // Max is 365
		os.Setenv("AGENT_MAX_PROMPTS", "200")          // Max is 100
		os.Setenv("AGENT_MAX_CONCURRENT", "50")        // Max is 20

		cfg := LoadConfigFromEnv()

		if cfg.LockTimeoutMinutes != 120 {
			t.Errorf("Expected LockTimeoutMinutes clamped to 120, got %d", cfg.LockTimeoutMinutes)
		}
		if cfg.RetentionDays != 365 {
			t.Errorf("Expected RetentionDays clamped to 365, got %d", cfg.RetentionDays)
		}
		if cfg.MaxPromptsPerSpawn != 100 {
			t.Errorf("Expected MaxPromptsPerSpawn clamped to 100, got %d", cfg.MaxPromptsPerSpawn)
		}
		if cfg.MaxConcurrentAgents != 20 {
			t.Errorf("Expected MaxConcurrentAgents clamped to 20, got %d", cfg.MaxConcurrentAgents)
		}
	})

	t.Run("clamps execution defaults to valid ranges", func(t *testing.T) {
		os.Setenv("AGENT_DEFAULT_TIMEOUT_SECONDS", "5000") // Max is 3600
		os.Setenv("AGENT_DEFAULT_MAX_TURNS", "300")        // Max is 200
		os.Setenv("AGENT_DEFAULT_MAX_FILES", "1000")       // Max is 500
		os.Setenv("AGENT_DEFAULT_MAX_BYTES", "200000000")  // Max is 104857600

		cfg := LoadConfigFromEnv()

		if cfg.DefaultTimeoutSeconds != 3600 {
			t.Errorf("Expected DefaultTimeoutSeconds clamped to 3600, got %d", cfg.DefaultTimeoutSeconds)
		}
		if cfg.DefaultMaxTurns != 200 {
			t.Errorf("Expected DefaultMaxTurns clamped to 200, got %d", cfg.DefaultMaxTurns)
		}
		if cfg.DefaultMaxFilesChanged != 500 {
			t.Errorf("Expected DefaultMaxFilesChanged clamped to 500, got %d", cfg.DefaultMaxFilesChanged)
		}
		if cfg.DefaultMaxBytesWritten != 104857600 {
			t.Errorf("Expected DefaultMaxBytesWritten clamped to 104857600, got %d", cfg.DefaultMaxBytesWritten)
		}
	})

	t.Run("clamps execution defaults minimum values", func(t *testing.T) {
		os.Setenv("AGENT_DEFAULT_TIMEOUT_SECONDS", "10") // Min is 60
		os.Setenv("AGENT_DEFAULT_MAX_TURNS", "1")        // Min is 5
		os.Setenv("AGENT_DEFAULT_MAX_FILES", "0")        // Min is 1
		os.Setenv("AGENT_DEFAULT_MAX_BYTES", "100")      // Min is 1024

		cfg := LoadConfigFromEnv()

		if cfg.DefaultTimeoutSeconds != 60 {
			t.Errorf("Expected DefaultTimeoutSeconds clamped to 60, got %d", cfg.DefaultTimeoutSeconds)
		}
		if cfg.DefaultMaxTurns != 5 {
			t.Errorf("Expected DefaultMaxTurns clamped to 5, got %d", cfg.DefaultMaxTurns)
		}
		// 0 is clamped to minimum (1)
		if cfg.DefaultMaxFilesChanged != 1 {
			t.Errorf("Expected DefaultMaxFilesChanged clamped to 1, got %d", cfg.DefaultMaxFilesChanged)
		}
		if cfg.DefaultMaxBytesWritten != 1024 {
			t.Errorf("Expected DefaultMaxBytesWritten clamped to 1024, got %d", cfg.DefaultMaxBytesWritten)
		}
	})

	t.Run("clamps minimum values", func(t *testing.T) {
		os.Setenv("AGENT_LOCK_TIMEOUT_MINUTES", "1") // Min is 5
		os.Setenv("AGENT_RETENTION_DAYS", "0")       // Min is 1
		os.Setenv("AGENT_MAX_PROMPTS", "0")          // Min is 1
		os.Setenv("AGENT_MAX_CONCURRENT", "0")       // Min is 1

		cfg := LoadConfigFromEnv()

		if cfg.LockTimeoutMinutes != 5 {
			t.Errorf("Expected LockTimeoutMinutes clamped to 5, got %d", cfg.LockTimeoutMinutes)
		}
		// 0 is clamped to minimum (1)
		if cfg.RetentionDays != 1 {
			t.Errorf("Expected RetentionDays clamped to 1, got %d", cfg.RetentionDays)
		}
	})

	t.Run("derives heartbeat interval from lock timeout", func(t *testing.T) {
		os.Unsetenv("AGENT_HEARTBEAT_INTERVAL_MINUTES")
		os.Setenv("AGENT_LOCK_TIMEOUT_MINUTES", "40")

		cfg := LoadConfigFromEnv()

		// Should be 1/4 of lock timeout
		if cfg.HeartbeatIntervalMinutes != 10 {
			t.Errorf("Expected HeartbeatIntervalMinutes=10 (40/4), got %d", cfg.HeartbeatIntervalMinutes)
		}
	})
}

func TestConfigValidate(t *testing.T) {
	t.Run("validation is non-destructive", func(t *testing.T) {
		cfg := DefaultConfig()
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}

func TestConfigValidateWithReport(t *testing.T) {
	t.Run("default config has no warnings", func(t *testing.T) {
		cfg := DefaultConfig()
		result := cfg.ValidateWithReport()

		if result.HasChanges() {
			t.Errorf("Expected no changes for default config, got warnings: %v", result.Warnings)
		}
	})

	t.Run("warns when network enabled by default", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.DefaultNetworkEnabled = true
		result := cfg.ValidateWithReport()

		found := false
		for _, w := range result.Warnings {
			if w == "Network access is enabled by default. Agents can make outbound requests." {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected warning about network access")
		}
	})

	t.Run("warns about high max files changed", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.DefaultMaxFilesChanged = 300
		result := cfg.ValidateWithReport()

		found := false
		for _, w := range result.Warnings {
			if w == "DefaultMaxFilesChanged is very high (>200). Agents can modify many files." {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected warning about high max files")
		}
	})

	t.Run("warns about high max bytes written", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.DefaultMaxBytesWritten = 20 * 1024 * 1024 // 20MB
		result := cfg.ValidateWithReport()

		found := false
		for _, w := range result.Warnings {
			if w == "DefaultMaxBytesWritten is very high (>10MB). Agents can write large amounts of data." {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected warning about high max bytes")
		}
	})

	t.Run("warns about heartbeat interval too close to lock timeout", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.LockTimeoutMinutes = 10
		cfg.HeartbeatIntervalMinutes = 6 // >= 10/2
		result := cfg.ValidateWithReport()

		found := false
		for _, w := range result.Warnings {
			if w == "HeartbeatIntervalMinutes should be less than half of LockTimeoutMinutes for reliable lock renewal" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected warning about heartbeat interval")
		}
	})
}
