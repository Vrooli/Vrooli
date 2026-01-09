package engine

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// SelectionConfig.Resolve Tests
// =============================================================================

func TestResolvePrefersOverride(t *testing.T) {
	cfg := SelectionConfig{DefaultEngine: "browserless", Override: "desktop"}
	if got := cfg.Resolve("request"); got != "desktop" {
		t.Fatalf("expected override to win, got %q", got)
	}
}

func TestResolveUsesRequestedWhenNoOverride(t *testing.T) {
	cfg := SelectionConfig{DefaultEngine: "browserless"}
	if got := cfg.Resolve("desktop"); got != "desktop" {
		t.Fatalf("expected requested engine to be used, got %q", got)
	}
}

func TestResolveFallsBackToDefault(t *testing.T) {
	cfg := SelectionConfig{DefaultEngine: "browserless"}
	if got := cfg.Resolve(""); got != "browserless" {
		t.Fatalf("expected default engine, got %q", got)
	}
}

// Additional Resolve tests

func TestResolveTrimsWhitespaceFromRequested(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] trims leading whitespace", func(t *testing.T) {
		cfg := SelectionConfig{DefaultEngine: "browserless"}
		got := cfg.Resolve("  desktop")
		assert.Equal(t, "desktop", got)
	})

	t.Run("trims trailing whitespace", func(t *testing.T) {
		cfg := SelectionConfig{DefaultEngine: "browserless"}
		got := cfg.Resolve("desktop  ")
		assert.Equal(t, "desktop", got)
	})

	t.Run("trims both sides", func(t *testing.T) {
		cfg := SelectionConfig{DefaultEngine: "browserless"}
		got := cfg.Resolve("  desktop  ")
		assert.Equal(t, "desktop", got)
	})

	t.Run("whitespace-only falls back to default", func(t *testing.T) {
		cfg := SelectionConfig{DefaultEngine: "browserless"}
		got := cfg.Resolve("   ")
		assert.Equal(t, "browserless", got)
	})

	t.Run("tabs and newlines are trimmed", func(t *testing.T) {
		cfg := SelectionConfig{DefaultEngine: "browserless"}
		got := cfg.Resolve("\tdesktop\n")
		assert.Equal(t, "desktop", got)
	})
}

func TestResolveOverridePrecedence(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] override beats both requested and default", func(t *testing.T) {
		cfg := SelectionConfig{
			DefaultEngine: "browserless",
			Override:      "playwright",
		}
		got := cfg.Resolve("desktop")
		assert.Equal(t, "playwright", got)
	})

	t.Run("override wins even with empty requested", func(t *testing.T) {
		cfg := SelectionConfig{
			DefaultEngine: "browserless",
			Override:      "desktop",
		}
		got := cfg.Resolve("")
		assert.Equal(t, "desktop", got)
	})

	t.Run("override wins even with whitespace-only requested", func(t *testing.T) {
		cfg := SelectionConfig{
			DefaultEngine: "browserless",
			Override:      "playwright",
		}
		got := cfg.Resolve("   ")
		assert.Equal(t, "playwright", got)
	})
}

func TestResolveDefaultBehavior(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] empty config returns empty string", func(t *testing.T) {
		cfg := SelectionConfig{}
		got := cfg.Resolve("")
		assert.Equal(t, "", got)
	})

	t.Run("empty default with request returns request", func(t *testing.T) {
		cfg := SelectionConfig{}
		got := cfg.Resolve("desktop")
		assert.Equal(t, "desktop", got)
	})
}

func TestResolveSpecialEngineNames(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] handles browserless engine", func(t *testing.T) {
		cfg := SelectionConfig{DefaultEngine: "playwright"}
		got := cfg.Resolve("browserless")
		assert.Equal(t, "browserless", got)
	})

	t.Run("handles playwright engine", func(t *testing.T) {
		cfg := SelectionConfig{DefaultEngine: "browserless"}
		got := cfg.Resolve("playwright")
		assert.Equal(t, "playwright", got)
	})

	t.Run("handles desktop engine", func(t *testing.T) {
		cfg := SelectionConfig{DefaultEngine: "browserless"}
		got := cfg.Resolve("desktop")
		assert.Equal(t, "desktop", got)
	})

	t.Run("handles unknown engine names", func(t *testing.T) {
		cfg := SelectionConfig{DefaultEngine: "browserless"}
		got := cfg.Resolve("unknown-engine")
		assert.Equal(t, "unknown-engine", got)
	})
}

// =============================================================================
// FromEnv Tests
// =============================================================================

func TestFromEnv(t *testing.T) {
	// Save original env vars
	originalEngine := os.Getenv("ENGINE")
	originalOverride := os.Getenv("ENGINE_OVERRIDE")
	defer func() {
		os.Setenv("ENGINE", originalEngine)
		os.Setenv("ENGINE_OVERRIDE", originalOverride)
	}()

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] reads ENGINE env var", func(t *testing.T) {
		os.Setenv("ENGINE", "playwright")
		os.Unsetenv("ENGINE_OVERRIDE")

		cfg := FromEnv()
		assert.Equal(t, "playwright", cfg.DefaultEngine)
		assert.Equal(t, "", cfg.Override)
	})

	t.Run("reads ENGINE_OVERRIDE env var", func(t *testing.T) {
		os.Setenv("ENGINE", "browserless")
		os.Setenv("ENGINE_OVERRIDE", "desktop")

		cfg := FromEnv()
		assert.Equal(t, "browserless", cfg.DefaultEngine)
		assert.Equal(t, "desktop", cfg.Override)
	})

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] trims whitespace from ENGINE", func(t *testing.T) {
		os.Setenv("ENGINE", "  playwright  ")
		os.Unsetenv("ENGINE_OVERRIDE")

		cfg := FromEnv()
		assert.Equal(t, "playwright", cfg.DefaultEngine)
	})

	t.Run("trims whitespace from ENGINE_OVERRIDE", func(t *testing.T) {
		os.Setenv("ENGINE", "browserless")
		os.Setenv("ENGINE_OVERRIDE", "  desktop  ")

		cfg := FromEnv()
		assert.Equal(t, "desktop", cfg.Override)
	})

	t.Run("handles empty env vars", func(t *testing.T) {
		os.Unsetenv("ENGINE")
		os.Unsetenv("ENGINE_OVERRIDE")

		cfg := FromEnv()
		assert.Equal(t, "", cfg.DefaultEngine)
		assert.Equal(t, "", cfg.Override)
	})

	t.Run("handles whitespace-only env vars", func(t *testing.T) {
		os.Setenv("ENGINE", "   ")
		os.Setenv("ENGINE_OVERRIDE", "\t\n")

		cfg := FromEnv()
		assert.Equal(t, "", cfg.DefaultEngine)
		assert.Equal(t, "", cfg.Override)
	})
}

func TestFromEnvIntegration(t *testing.T) {
	// Save original env vars
	originalEngine := os.Getenv("ENGINE")
	originalOverride := os.Getenv("ENGINE_OVERRIDE")
	defer func() {
		os.Setenv("ENGINE", originalEngine)
		os.Setenv("ENGINE_OVERRIDE", originalOverride)
	}()

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] FromEnv + Resolve integration", func(t *testing.T) {
		os.Setenv("ENGINE", "browserless")
		os.Setenv("ENGINE_OVERRIDE", "playwright")

		cfg := FromEnv()
		got := cfg.Resolve("desktop")

		// Override should win
		assert.Equal(t, "playwright", got)
	})

	t.Run("FromEnv without override uses requested", func(t *testing.T) {
		os.Setenv("ENGINE", "browserless")
		os.Unsetenv("ENGINE_OVERRIDE")

		cfg := FromEnv()
		got := cfg.Resolve("desktop")

		assert.Equal(t, "desktop", got)
	})

	t.Run("FromEnv without override or request uses default", func(t *testing.T) {
		os.Setenv("ENGINE", "browserless")
		os.Unsetenv("ENGINE_OVERRIDE")

		cfg := FromEnv()
		got := cfg.Resolve("")

		assert.Equal(t, "browserless", got)
	})
}

// =============================================================================
// SelectionConfig Structure Tests
// =============================================================================

func TestSelectionConfigZeroValue(t *testing.T) {
	t.Run("zero value has empty strings", func(t *testing.T) {
		var cfg SelectionConfig
		assert.Equal(t, "", cfg.DefaultEngine)
		assert.Equal(t, "", cfg.Override)
	})

	t.Run("zero value resolves to empty with empty request", func(t *testing.T) {
		var cfg SelectionConfig
		got := cfg.Resolve("")
		assert.Equal(t, "", got)
	})
}

func TestSelectionConfigCopy(t *testing.T) {
	t.Run("config is value type (copy safe)", func(t *testing.T) {
		original := SelectionConfig{
			DefaultEngine: "browserless",
			Override:      "desktop",
		}
		copied := original

		// Modify the copy
		copied.DefaultEngine = "playwright"
		copied.Override = ""

		// Original should be unchanged
		assert.Equal(t, "browserless", original.DefaultEngine)
		assert.Equal(t, "desktop", original.Override)
	})
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestResolveEdgeCases(t *testing.T) {
	t.Run("handles very long engine name", func(t *testing.T) {
		longName := "this-is-a-very-long-engine-name-that-should-still-work-correctly"
		cfg := SelectionConfig{DefaultEngine: "browserless"}
		got := cfg.Resolve(longName)
		assert.Equal(t, longName, got)
	})

	t.Run("handles engine name with special characters", func(t *testing.T) {
		cfg := SelectionConfig{DefaultEngine: "browserless"}
		got := cfg.Resolve("engine-v2.0")
		assert.Equal(t, "engine-v2.0", got)
	})

	t.Run("handles engine name with underscores", func(t *testing.T) {
		cfg := SelectionConfig{DefaultEngine: "browserless"}
		got := cfg.Resolve("my_custom_engine")
		assert.Equal(t, "my_custom_engine", got)
	})

	t.Run("preserves case sensitivity", func(t *testing.T) {
		cfg := SelectionConfig{DefaultEngine: "browserless"}
		got := cfg.Resolve("Playwright")
		assert.Equal(t, "Playwright", got)
	})
}

// =============================================================================
// Concurrency Safety Tests
// =============================================================================

func TestResolveConcurrencySafety(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] concurrent Resolve calls are safe", func(t *testing.T) {
		cfg := SelectionConfig{
			DefaultEngine: "browserless",
			Override:      "",
		}

		done := make(chan string, 100)
		for i := 0; i < 100; i++ {
			go func() {
				done <- cfg.Resolve("playwright")
			}()
		}

		// All should return the same value
		for i := 0; i < 100; i++ {
			got := <-done
			require.Equal(t, "playwright", got)
		}
	})
}
