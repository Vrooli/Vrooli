package main

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Save original env vars
	originalEnv := map[string]string{
		"API_PORT":                      os.Getenv("API_PORT"),
		"API_HOST":                      os.Getenv("API_HOST"),
		"WEB_CONSOLE_DEFAULT_COMMAND":   os.Getenv("WEB_CONSOLE_DEFAULT_COMMAND"),
		"WEB_CONSOLE_SESSION_TTL":       os.Getenv("WEB_CONSOLE_SESSION_TTL"),
		"WEB_CONSOLE_IDLE_TIMEOUT":      os.Getenv("WEB_CONSOLE_IDLE_TIMEOUT"),
		"WEB_CONSOLE_MAX_CONCURRENT":    os.Getenv("WEB_CONSOLE_MAX_CONCURRENT"),
		"WEB_CONSOLE_EXPECT_PROXY":      os.Getenv("WEB_CONSOLE_EXPECT_PROXY"),
		"WEB_CONSOLE_PANIC_GRACE":       os.Getenv("WEB_CONSOLE_PANIC_GRACE"),
		"WEB_CONSOLE_READ_BUFFER":       os.Getenv("WEB_CONSOLE_READ_BUFFER"),
		"WEB_CONSOLE_TTY_ROWS":          os.Getenv("WEB_CONSOLE_TTY_ROWS"),
		"WEB_CONSOLE_TTY_COLS":          os.Getenv("WEB_CONSOLE_TTY_COLS"),
		"WEB_CONSOLE_STORAGE_PATH":      os.Getenv("WEB_CONSOLE_STORAGE_PATH"),
		"WEB_CONSOLE_WORKING_DIR":       os.Getenv("WEB_CONSOLE_WORKING_DIR"),
	}

	// Restore env vars after test
	defer func() {
		for key, value := range originalEnv {
			if value != "" {
				os.Setenv(key, value)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	t.Run("ValidConfig", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		os.Setenv("API_HOST", "127.0.0.1")
		os.Setenv("WEB_CONSOLE_DEFAULT_COMMAND", "/bin/echo")
		os.Setenv("WEB_CONSOLE_SESSION_TTL", "10m")
		os.Setenv("WEB_CONSOLE_IDLE_TIMEOUT", "2m")
		os.Setenv("WEB_CONSOLE_MAX_CONCURRENT", "5")

		cfg, err := loadConfig()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if cfg.addr != "127.0.0.1:8080" {
			t.Errorf("Expected addr '127.0.0.1:8080', got '%s'", cfg.addr)
		}
		if cfg.defaultCommand != "/bin/echo" {
			t.Errorf("Expected command '/bin/echo', got '%s'", cfg.defaultCommand)
		}
		if cfg.sessionTTL != 10*time.Minute {
			t.Errorf("Expected TTL 10m, got %v", cfg.sessionTTL)
		}
		if cfg.idleTimeout != 2*time.Minute {
			t.Errorf("Expected idle timeout 2m, got %v", cfg.idleTimeout)
		}
		if cfg.maxConcurrent != 5 {
			t.Errorf("Expected max concurrent 5, got %d", cfg.maxConcurrent)
		}
	})

	t.Run("MissingAPIPort", func(t *testing.T) {
		os.Unsetenv("API_PORT")

		_, err := loadConfig()
		if err == nil {
			t.Error("Expected error when API_PORT missing")
		}
	})

	t.Run("DefaultHost", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		os.Unsetenv("API_HOST")
		os.Setenv("WEB_CONSOLE_DEFAULT_COMMAND", "/bin/bash")

		cfg, err := loadConfig()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if cfg.addr != "0.0.0.0:8080" {
			t.Errorf("Expected default host '0.0.0.0:8080', got '%s'", cfg.addr)
		}
	})

	t.Run("DefaultValues", func(t *testing.T) {
		// Clear all optional env vars to test defaults
		os.Unsetenv("WEB_CONSOLE_SESSION_TTL")
		os.Unsetenv("WEB_CONSOLE_IDLE_TIMEOUT")
		os.Unsetenv("WEB_CONSOLE_MAX_CONCURRENT")
		os.Unsetenv("WEB_CONSOLE_EXPECT_PROXY")

		os.Setenv("API_PORT", "8080")
		os.Setenv("WEB_CONSOLE_DEFAULT_COMMAND", "/bin/bash")

		cfg, err := loadConfig()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if cfg.sessionTTL != 30*time.Minute {
			t.Errorf("Expected default TTL 30m, got %v", cfg.sessionTTL)
		}
		if cfg.idleTimeout != 5*time.Minute {
			t.Errorf("Expected default idle timeout 5m, got %v", cfg.idleTimeout)
		}
		if cfg.maxConcurrent != 20 {
			t.Errorf("Expected default max concurrent 20, got %d", cfg.maxConcurrent)
		}
		if cfg.enableProxyGuard != true {
			t.Error("Expected proxy guard enabled by default")
		}
	})

	t.Run("InvalidSessionTTL", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		os.Setenv("WEB_CONSOLE_DEFAULT_COMMAND", "/bin/bash")
		os.Setenv("WEB_CONSOLE_SESSION_TTL", "0s")

		_, err := loadConfig()
		if err == nil {
			t.Error("Expected error for invalid session TTL")
		}
	})

	t.Run("InvalidIdleTimeout", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		os.Setenv("WEB_CONSOLE_DEFAULT_COMMAND", "/bin/bash")
		os.Setenv("WEB_CONSOLE_IDLE_TIMEOUT", "-1s")

		_, err := loadConfig()
		if err == nil {
			t.Error("Expected error for invalid idle timeout")
		}
	})

	t.Run("InvalidMaxConcurrent", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		os.Setenv("WEB_CONSOLE_DEFAULT_COMMAND", "/bin/bash")
		os.Setenv("WEB_CONSOLE_MAX_CONCURRENT", "0")

		_, err := loadConfig()
		if err == nil {
			t.Error("Expected error for invalid max concurrent")
		}
	})

	t.Run("InvalidPanicGrace", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		os.Setenv("WEB_CONSOLE_DEFAULT_COMMAND", "/bin/bash")
		os.Setenv("WEB_CONSOLE_PANIC_GRACE", "100ms")

		_, err := loadConfig()
		if err == nil {
			t.Error("Expected error for panic grace < 500ms")
		}
	})

	t.Run("InvalidReadBuffer", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		os.Setenv("WEB_CONSOLE_DEFAULT_COMMAND", "/bin/bash")
		os.Setenv("WEB_CONSOLE_READ_BUFFER", "100")

		_, err := loadConfig()
		if err == nil {
			t.Error("Expected error for read buffer < 512")
		}
	})
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"   ", nil},
		{"arg1", []string{"arg1"}},
		{"arg1 arg2", []string{"arg1", "arg2"}},
		{"arg1  arg2  arg3", []string{"arg1", "arg2", "arg3"}},
		{"  arg1  arg2  ", []string{"arg1", "arg2"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseArgs(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d args, got %d", len(tt.expected), len(result))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("Expected arg[%d] '%s', got '%s'", i, tt.expected[i], result[i])
				}
			}
		})
	}
}

func TestParseDurationOrDefault(t *testing.T) {
	tests := []struct {
		input    string
		fallback time.Duration
		expected time.Duration
	}{
		{"", 5 * time.Minute, 5 * time.Minute},
		{"10m", 5 * time.Minute, 10 * time.Minute},
		{"1h30m", 5 * time.Minute, 90 * time.Minute},
		{"invalid", 5 * time.Minute, 5 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseDurationOrDefault(tt.input, tt.fallback)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestParseIntOrDefault(t *testing.T) {
	tests := []struct {
		input    string
		fallback int
		expected int
	}{
		{"", 10, 10},
		{"5", 10, 5},
		{"100", 10, 100},
		{"invalid", 10, 10},
		{"-5", 10, -5},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseIntOrDefault(tt.input, tt.fallback)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestParseBoolOrDefault(t *testing.T) {
	tests := []struct {
		input    string
		fallback bool
		expected bool
	}{
		{"", true, true},
		{"default", false, false},
		{"true", false, true},
		{"TRUE", false, true},
		{"1", false, true},
		{"yes", false, true},
		{"false", true, false},
		{"FALSE", true, false},
		{"0", true, false},
		{"no", true, false},
		{"invalid", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseBoolOrDefault(tt.input, tt.fallback)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFirstNonEmpty(t *testing.T) {
	tests := []struct {
		values   []string
		expected string
	}{
		{[]string{}, ""},
		{[]string{""}, ""},
		{[]string{"", ""}, ""},
		{[]string{"first"}, "first"},
		{[]string{"", "second"}, "second"},
		{[]string{"first", "second"}, "first"},
		{[]string{"", "", "third"}, "third"},
	}

	for i, tt := range tests {
		t.Run(string(rune(i)), func(t *testing.T) {
			result := firstNonEmpty(tt.values...)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestUpDirectory(t *testing.T) {
	t.Run("SingleLevel", func(t *testing.T) {
		result := upDirectory(1)
		if result == "" {
			t.Error("Expected non-empty result")
		}
	})

	t.Run("MultipleLevels", func(t *testing.T) {
		result := upDirectory(3)
		if result == "" {
			t.Error("Expected non-empty result")
		}
	})

	t.Run("ZeroLevels", func(t *testing.T) {
		cwd, _ := os.Getwd()
		result := upDirectory(0)
		if result != cwd {
			t.Errorf("Expected current directory, got '%s'", result)
		}
	})
}
