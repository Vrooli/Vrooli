package config

import (
	"os"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	// Pagination defaults
	if cfg.Pagination.DefaultLimit != 20 {
		t.Errorf("expected DefaultLimit 20, got %d", cfg.Pagination.DefaultLimit)
	}
	if cfg.Pagination.MaxLimit != 100 {
		t.Errorf("expected MaxLimit 100, got %d", cfg.Pagination.MaxLimit)
	}

	// Validation defaults
	if cfg.Validation.TaskTitleMaxLength != 255 {
		t.Errorf("expected TaskTitleMaxLength 255, got %d", cfg.Validation.TaskTitleMaxLength)
	}
	if cfg.Validation.ProjectNameMaxLength != 100 {
		t.Errorf("expected ProjectNameMaxLength 100, got %d", cfg.Validation.ProjectNameMaxLength)
	}
	if cfg.Validation.NoteContentMaxLength != 10000 {
		t.Errorf("expected NoteContentMaxLength 10000, got %d", cfg.Validation.NoteContentMaxLength)
	}

	// CORS defaults
	if cfg.CORS.AllowedOrigins != "http://localhost:*" {
		t.Errorf("expected AllowedOrigins 'http://localhost:*', got %q", cfg.CORS.AllowedOrigins)
	}
	if cfg.CORS.MaxAge != 86400 {
		t.Errorf("expected MaxAge 86400, got %d", cfg.CORS.MaxAge)
	}

	// Server defaults
	if cfg.Server.HealthVersion != "1.0.0" {
		t.Errorf("expected HealthVersion '1.0.0', got %q", cfg.Server.HealthVersion)
	}
}

func TestLoadFromEnv_UsesDefaults(t *testing.T) {
	// Clear any existing env vars
	os.Unsetenv("PAGINATION_DEFAULT_LIMIT")
	os.Unsetenv("PAGINATION_MAX_LIMIT")
	os.Unsetenv("CORS_ALLOWED_ORIGINS")

	cfg := LoadFromEnv()

	if cfg.Pagination.DefaultLimit != 20 {
		t.Errorf("expected default DefaultLimit 20, got %d", cfg.Pagination.DefaultLimit)
	}
	if cfg.CORS.AllowedOrigins != "http://localhost:*" {
		t.Errorf("expected default AllowedOrigins, got %q", cfg.CORS.AllowedOrigins)
	}
}

func TestLoadFromEnv_OverridesPagination(t *testing.T) {
	os.Setenv("PAGINATION_DEFAULT_LIMIT", "50")
	os.Setenv("PAGINATION_MAX_LIMIT", "200")
	defer func() {
		os.Unsetenv("PAGINATION_DEFAULT_LIMIT")
		os.Unsetenv("PAGINATION_MAX_LIMIT")
	}()

	cfg := LoadFromEnv()

	if cfg.Pagination.DefaultLimit != 50 {
		t.Errorf("expected DefaultLimit 50, got %d", cfg.Pagination.DefaultLimit)
	}
	if cfg.Pagination.MaxLimit != 200 {
		t.Errorf("expected MaxLimit 200, got %d", cfg.Pagination.MaxLimit)
	}
}

func TestLoadFromEnv_OverridesValidation(t *testing.T) {
	os.Setenv("VALIDATION_TASK_TITLE_MAX", "500")
	os.Setenv("VALIDATION_PROJECT_NAME_MAX", "150")
	os.Setenv("VALIDATION_NOTE_CONTENT_MAX", "5000")
	defer func() {
		os.Unsetenv("VALIDATION_TASK_TITLE_MAX")
		os.Unsetenv("VALIDATION_PROJECT_NAME_MAX")
		os.Unsetenv("VALIDATION_NOTE_CONTENT_MAX")
	}()

	cfg := LoadFromEnv()

	if cfg.Validation.TaskTitleMaxLength != 500 {
		t.Errorf("expected TaskTitleMaxLength 500, got %d", cfg.Validation.TaskTitleMaxLength)
	}
	if cfg.Validation.ProjectNameMaxLength != 150 {
		t.Errorf("expected ProjectNameMaxLength 150, got %d", cfg.Validation.ProjectNameMaxLength)
	}
	if cfg.Validation.NoteContentMaxLength != 5000 {
		t.Errorf("expected NoteContentMaxLength 5000, got %d", cfg.Validation.NoteContentMaxLength)
	}
}

func TestLoadFromEnv_OverridesCORS(t *testing.T) {
	os.Setenv("CORS_ALLOWED_ORIGINS", "https://example.com,https://api.example.com")
	os.Setenv("CORS_MAX_AGE", "3600")
	defer func() {
		os.Unsetenv("CORS_ALLOWED_ORIGINS")
		os.Unsetenv("CORS_MAX_AGE")
	}()

	cfg := LoadFromEnv()

	if cfg.CORS.AllowedOrigins != "https://example.com,https://api.example.com" {
		t.Errorf("expected custom AllowedOrigins, got %q", cfg.CORS.AllowedOrigins)
	}
	if cfg.CORS.MaxAge != 3600 {
		t.Errorf("expected MaxAge 3600, got %d", cfg.CORS.MaxAge)
	}
}

func TestLoadFromEnv_OverridesServer(t *testing.T) {
	os.Setenv("HEALTH_VERSION", "2.0.0")
	defer os.Unsetenv("HEALTH_VERSION")

	cfg := LoadFromEnv()

	if cfg.Server.HealthVersion != "2.0.0" {
		t.Errorf("expected HealthVersion '2.0.0', got %q", cfg.Server.HealthVersion)
	}
}

func TestLoadFromEnv_IgnoresInvalidValues(t *testing.T) {
	os.Setenv("PAGINATION_DEFAULT_LIMIT", "invalid")
	os.Setenv("PAGINATION_MAX_LIMIT", "-5")
	os.Setenv("CORS_MAX_AGE", "not-a-number")
	defer func() {
		os.Unsetenv("PAGINATION_DEFAULT_LIMIT")
		os.Unsetenv("PAGINATION_MAX_LIMIT")
		os.Unsetenv("CORS_MAX_AGE")
	}()

	cfg := LoadFromEnv()

	// Should use defaults when invalid values are provided
	if cfg.Pagination.DefaultLimit != 20 {
		t.Errorf("expected default DefaultLimit when invalid, got %d", cfg.Pagination.DefaultLimit)
	}
	if cfg.Pagination.MaxLimit != 100 {
		t.Errorf("expected default MaxLimit when negative, got %d", cfg.Pagination.MaxLimit)
	}
	if cfg.CORS.MaxAge != 86400 {
		t.Errorf("expected default MaxAge when invalid, got %d", cfg.CORS.MaxAge)
	}
}

func TestLoadFromEnv_ZeroMaxAgeAllowed(t *testing.T) {
	os.Setenv("CORS_MAX_AGE", "0")
	defer os.Unsetenv("CORS_MAX_AGE")

	cfg := LoadFromEnv()

	if cfg.CORS.MaxAge != 0 {
		t.Errorf("expected MaxAge 0, got %d", cfg.CORS.MaxAge)
	}
}
