package containment

import (
	"os"
	"testing"
	"time"
)

func TestContainmentDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	t.Run("docker image defaults to ubuntu:22.04", func(t *testing.T) {
		if cfg.DockerImage != "ubuntu:22.04" {
			t.Errorf("Expected DockerImage='ubuntu:22.04', got %s", cfg.DockerImage)
		}
	})

	t.Run("max memory defaults to 2048MB", func(t *testing.T) {
		if cfg.MaxMemoryMB != 2048 {
			t.Errorf("Expected MaxMemoryMB=2048, got %d", cfg.MaxMemoryMB)
		}
	})

	t.Run("max CPU defaults to 200 percent", func(t *testing.T) {
		if cfg.MaxCPUPercent != 200 {
			t.Errorf("Expected MaxCPUPercent=200, got %d", cfg.MaxCPUPercent)
		}
	})

	t.Run("availability timeout defaults to 5 seconds", func(t *testing.T) {
		if cfg.AvailabilityTimeoutSeconds != 5 {
			t.Errorf("Expected AvailabilityTimeoutSeconds=5, got %d", cfg.AvailabilityTimeoutSeconds)
		}
	})

	t.Run("security defaults are enabled", func(t *testing.T) {
		if !cfg.DropAllCapabilities {
			t.Error("Expected DropAllCapabilities=true")
		}
		if !cfg.NoNewPrivileges {
			t.Error("Expected NoNewPrivileges=true")
		}
		if cfg.ReadOnlyRootFS {
			t.Error("Expected ReadOnlyRootFS=false")
		}
	})

	t.Run("fallback is allowed by default", func(t *testing.T) {
		if !cfg.AllowFallback {
			t.Error("Expected AllowFallback=true")
		}
	})
}

func TestContainmentConfigDurations(t *testing.T) {
	cfg := DefaultConfig()

	t.Run("AvailabilityTimeout returns duration", func(t *testing.T) {
		expected := 5 * time.Second
		if cfg.AvailabilityTimeout() != expected {
			t.Errorf("Expected AvailabilityTimeout=%v, got %v", expected, cfg.AvailabilityTimeout())
		}
	})
}

func TestContainmentLoadConfigFromEnv(t *testing.T) {
	// Clean up any env vars after test
	defer func() {
		os.Unsetenv("CONTAINMENT_DOCKER_IMAGE")
		os.Unsetenv("CONTAINMENT_MAX_MEMORY_MB")
		os.Unsetenv("CONTAINMENT_MAX_CPU_PERCENT")
		os.Unsetenv("CONTAINMENT_AVAILABILITY_TIMEOUT_SECONDS")
		os.Unsetenv("CONTAINMENT_PREFER_DOCKER")
		os.Unsetenv("CONTAINMENT_ALLOW_FALLBACK")
	}()

	t.Run("loads from environment", func(t *testing.T) {
		os.Setenv("CONTAINMENT_DOCKER_IMAGE", "alpine:latest")
		os.Setenv("CONTAINMENT_MAX_MEMORY_MB", "4096")
		os.Setenv("CONTAINMENT_MAX_CPU_PERCENT", "400")
		os.Setenv("CONTAINMENT_AVAILABILITY_TIMEOUT_SECONDS", "10")

		cfg := LoadConfigFromEnv()

		if cfg.DockerImage != "alpine:latest" {
			t.Errorf("Expected DockerImage='alpine:latest', got %s", cfg.DockerImage)
		}
		if cfg.MaxMemoryMB != 4096 {
			t.Errorf("Expected MaxMemoryMB=4096, got %d", cfg.MaxMemoryMB)
		}
		if cfg.MaxCPUPercent != 400 {
			t.Errorf("Expected MaxCPUPercent=400, got %d", cfg.MaxCPUPercent)
		}
		if cfg.AvailabilityTimeoutSeconds != 10 {
			t.Errorf("Expected AvailabilityTimeoutSeconds=10, got %d", cfg.AvailabilityTimeoutSeconds)
		}
	})

	t.Run("clamps values to valid ranges", func(t *testing.T) {
		os.Setenv("CONTAINMENT_MAX_MEMORY_MB", "20000")             // Max is 16384
		os.Setenv("CONTAINMENT_MAX_CPU_PERCENT", "1000")            // Max is 800
		os.Setenv("CONTAINMENT_AVAILABILITY_TIMEOUT_SECONDS", "60") // Max is 30

		cfg := LoadConfigFromEnv()

		if cfg.MaxMemoryMB != 16384 {
			t.Errorf("Expected MaxMemoryMB clamped to 16384, got %d", cfg.MaxMemoryMB)
		}
		if cfg.MaxCPUPercent != 800 {
			t.Errorf("Expected MaxCPUPercent clamped to 800, got %d", cfg.MaxCPUPercent)
		}
		if cfg.AvailabilityTimeoutSeconds != 30 {
			t.Errorf("Expected AvailabilityTimeoutSeconds clamped to 30, got %d", cfg.AvailabilityTimeoutSeconds)
		}
	})

	t.Run("clamps minimum values", func(t *testing.T) {
		os.Setenv("CONTAINMENT_MAX_MEMORY_MB", "100")  // Min is 256
		os.Setenv("CONTAINMENT_MAX_CPU_PERCENT", "10") // Min is 50

		cfg := LoadConfigFromEnv()

		if cfg.MaxMemoryMB != 256 {
			t.Errorf("Expected MaxMemoryMB clamped to 256, got %d", cfg.MaxMemoryMB)
		}
		if cfg.MaxCPUPercent != 50 {
			t.Errorf("Expected MaxCPUPercent clamped to 50, got %d", cfg.MaxCPUPercent)
		}
	})

	t.Run("parses boolean values", func(t *testing.T) {
		os.Setenv("CONTAINMENT_PREFER_DOCKER", "false")
		os.Setenv("CONTAINMENT_ALLOW_FALLBACK", "no")

		cfg := LoadConfigFromEnv()

		if cfg.PreferDocker {
			t.Error("Expected PreferDocker=false")
		}
		if cfg.AllowFallback {
			t.Error("Expected AllowFallback=false")
		}
	})
}

func TestContainmentConfigValidateWithReport(t *testing.T) {
	t.Run("default config warns about fallback", func(t *testing.T) {
		cfg := DefaultConfig()
		result := cfg.ValidateWithReport()

		// Default config has AllowFallback=true which should trigger a warning
		found := false
		for _, w := range result.Warnings {
			if w == "AllowFallback is enabled. Agents may run without containment if Docker is unavailable." {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected warning about fallback being enabled")
		}
	})

	t.Run("warns about disabled security features", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.DropAllCapabilities = false
		cfg.NoNewPrivileges = false
		cfg.AllowFallback = false // Disable to avoid that warning
		result := cfg.ValidateWithReport()

		foundCapabilities := false
		foundPrivileges := false
		for _, w := range result.Warnings {
			if w == "DropAllCapabilities is disabled. Containers retain default Linux capabilities." {
				foundCapabilities = true
			}
			if w == "NoNewPrivileges is disabled. Privilege escalation inside containers is possible." {
				foundPrivileges = true
			}
		}

		if !foundCapabilities {
			t.Error("Expected warning about DropAllCapabilities")
		}
		if !foundPrivileges {
			t.Error("Expected warning about NoNewPrivileges")
		}
	})

	t.Run("warns about high resource limits", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.MaxMemoryMB = 10240   // 10GB
		cfg.MaxCPUPercent = 500   // 5 cores
		cfg.AllowFallback = false // Disable to avoid that warning
		result := cfg.ValidateWithReport()

		foundMemory := false
		foundCPU := false
		for _, w := range result.Warnings {
			if w == "MaxMemoryMB is very high (>8GB). Agents can consume substantial memory." {
				foundMemory = true
			}
			if w == "MaxCPUPercent is very high (>4 cores). Agents can consume substantial CPU." {
				foundCPU = true
			}
		}

		if !foundMemory {
			t.Error("Expected warning about high memory")
		}
		if !foundCPU {
			t.Error("Expected warning about high CPU")
		}
	})

	t.Run("warns about latest Docker tag", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.DockerImage = "ubuntu:latest"
		cfg.AllowFallback = false // Disable to avoid that warning
		result := cfg.ValidateWithReport()

		found := false
		for _, w := range result.Warnings {
			if w == "Using 'latest' tag for Docker image. Consider using a specific tag for reproducibility." {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected warning about latest tag")
		}
	})
}
