//go:build testing
// +build testing

package main

import (
	"os"
	"testing"
)

func TestGetEnv(t *testing.T) {
	t.Run("ExistingVariable", func(t *testing.T) {
		os.Setenv("TEST_VAR", "test_value")
		defer os.Unsetenv("TEST_VAR")

		result := getEnv("TEST_VAR", "default")
		if result != "test_value" {
			t.Errorf("Expected 'test_value', got '%s'", result)
		}
	})

	t.Run("MissingVariable", func(t *testing.T) {
		result := getEnv("NONEXISTENT_VAR", "default_value")
		if result != "default_value" {
			t.Errorf("Expected 'default_value', got '%s'", result)
		}
	})

	t.Run("EmptyVariable", func(t *testing.T) {
		os.Setenv("EMPTY_VAR", "")
		defer os.Unsetenv("EMPTY_VAR")

		result := getEnv("EMPTY_VAR", "default")
		if result != "default" {
			t.Errorf("Empty variable should use default, got '%s'", result)
		}
	})
}

func TestGetEnvRequired(t *testing.T) {
	t.Run("ExistingVariable", func(t *testing.T) {
		os.Setenv("REQUIRED_VAR", "value")
		defer os.Unsetenv("REQUIRED_VAR")

		result, err := getEnvRequired("REQUIRED_VAR")
		if err != nil {
			t.Fatalf("Should not error for existing var: %v", err)
		}
		if result != "value" {
			t.Errorf("Expected 'value', got '%s'", result)
		}
	})

	t.Run("MissingVariable", func(t *testing.T) {
		_, err := getEnvRequired("MISSING_REQUIRED_VAR")
		if err == nil {
			t.Fatal("Should error for missing required variable")
		}
	})
}

func TestGetEnvBool(t *testing.T) {
	testCases := []struct {
		name         string
		value        string
		defaultValue bool
		expected     bool
	}{
		{"TrueString", "true", false, true},
		{"FalseString", "false", true, false},
		{"OneString", "1", false, true},
		{"ZeroString", "0", true, false},
		{"InvalidValue", "invalid", true, true},
		{"EmptyValue", "", false, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.value != "" {
				os.Setenv("TEST_BOOL", tc.value)
				defer os.Unsetenv("TEST_BOOL")
			}

			result := getEnvBool("TEST_BOOL", tc.defaultValue)
			if result != tc.expected {
				t.Errorf("For value '%s', expected %v, got %v", tc.value, tc.expected, result)
			}
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	testCases := []struct {
		name         string
		value        string
		defaultValue int
		expected     int
	}{
		{"ValidInteger", "123", 0, 123},
		{"NegativeInteger", "-456", 0, -456},
		{"Zero", "0", 999, 0},
		{"InvalidValue", "not_a_number", 100, 100},
		{"EmptyValue", "", 42, 42},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.value != "" {
				os.Setenv("TEST_INT", tc.value)
				defer os.Unsetenv("TEST_INT")
			} else {
				os.Unsetenv("TEST_INT")
			}

			result := getEnvInt("TEST_INT", tc.defaultValue)
			if result != tc.expected {
				t.Errorf("For value '%s', expected %d, got %d", tc.value, tc.expected, result)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	t.Run("DefaultConfiguration", func(t *testing.T) {
		// Clear any existing environment variables
		envVars := []string{
			"POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER",
			"POSTGRES_PASSWORD", "POSTGRES_DB", "QDRANT_URL",
			"OLLAMA_URL", "API_PORT", "VROOLI_LIFECYCLE_MANAGED",
			"USE_MOCK_TESTING",
		}
		for _, v := range envVars {
			os.Unsetenv(v)
		}

		config, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig should not error with defaults: %v", err)
		}

		// Verify defaults
		if config.PostgresHost != "localhost" {
			t.Errorf("Expected PostgresHost='localhost', got '%s'", config.PostgresHost)
		}
		if config.PostgresPort != "5432" {
			t.Errorf("Expected PostgresPort='5432', got '%s'", config.PostgresPort)
		}
		if config.PostgresDB != "prompt_injection_arena" {
			t.Errorf("Expected PostgresDB='prompt_injection_arena', got '%s'", config.PostgresDB)
		}
		if config.APIPort != "16018" {
			t.Errorf("Expected APIPort='16018', got '%s'", config.APIPort)
		}
	})

	t.Run("CustomConfiguration", func(t *testing.T) {
		os.Setenv("POSTGRES_HOST", "custom-host")
		os.Setenv("POSTGRES_PORT", "5433")
		os.Setenv("POSTGRES_DB", "custom_db")
		os.Setenv("API_PORT", "8080")
		defer func() {
			os.Unsetenv("POSTGRES_HOST")
			os.Unsetenv("POSTGRES_PORT")
			os.Unsetenv("POSTGRES_DB")
			os.Unsetenv("API_PORT")
		}()

		config, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig should not error: %v", err)
		}

		if config.PostgresHost != "custom-host" {
			t.Errorf("Expected custom PostgresHost, got '%s'", config.PostgresHost)
		}
		if config.PostgresPort != "5433" {
			t.Errorf("Expected custom PostgresPort, got '%s'", config.PostgresPort)
		}
		if config.PostgresDB != "custom_db" {
			t.Errorf("Expected custom PostgresDB, got '%s'", config.PostgresDB)
		}
		if config.APIPort != "8080" {
			t.Errorf("Expected custom APIPort, got '%s'", config.APIPort)
		}
	})

	t.Run("BooleanFlags", func(t *testing.T) {
		os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
		os.Setenv("USE_MOCK_TESTING", "true")
		defer func() {
			os.Unsetenv("VROOLI_LIFECYCLE_MANAGED")
			os.Unsetenv("USE_MOCK_TESTING")
		}()

		config, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig should not error: %v", err)
		}

		if !config.LifecycleManaged {
			t.Error("Expected LifecycleManaged=true")
		}
		if !config.UseMockTesting {
			t.Error("Expected UseMockTesting=true")
		}
	})
}

func TestConfigConnectionString(t *testing.T) {
	config := &Config{
		PostgresHost:     "testhost",
		PostgresPort:     "5432",
		PostgresUser:     "testuser",
		PostgresPassword: "testpass",
		PostgresDB:       "testdb",
	}

	connStr := config.GetDatabaseConnectionString()

	expectedParts := []string{
		"host=testhost",
		"port=5432",
		"user=testuser",
		"password=testpass",
		"dbname=testdb",
		"sslmode=disable",
	}

	for _, part := range expectedParts {
		if !stringContains(connStr, part) {
			t.Errorf("Connection string should contain '%s'", part)
		}
	}
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestConfigEdgeCases(t *testing.T) {
	t.Run("EmptyPassword", func(t *testing.T) {
		os.Unsetenv("POSTGRES_PASSWORD")

		config, err := LoadConfig()
		if err != nil {
			t.Fatalf("Should handle empty password: %v", err)
		}

		// Empty env var should use default value "postgres"
		if config.PostgresPassword != "postgres" {
			t.Errorf("Expected default password 'postgres', got '%s'", config.PostgresPassword)
		}
	})

	t.Run("SpecialCharactersInPassword", func(t *testing.T) {
		specialPass := "p@ss!w0rd#$%"
		os.Setenv("POSTGRES_PASSWORD", specialPass)
		defer os.Unsetenv("POSTGRES_PASSWORD")

		config, err := LoadConfig()
		if err != nil {
			t.Fatalf("Should handle special characters: %v", err)
		}

		if config.PostgresPassword != specialPass {
			t.Errorf("Password with special chars not preserved correctly")
		}
	})
}
