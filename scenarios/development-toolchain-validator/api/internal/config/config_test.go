// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md
package config_test

import (
	"os"
	"testing"

	"development-toolchain-validator/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	t.Run("pagination defaults", func(t *testing.T) {
		if cfg.Pagination.DefaultLimit != 20 {
			t.Errorf("DefaultLimit = %d, want 20", cfg.Pagination.DefaultLimit)
		}
		if cfg.Pagination.MaxLimit != 100 {
			t.Errorf("MaxLimit = %d, want 100", cfg.Pagination.MaxLimit)
		}
	})

	t.Run("validation defaults", func(t *testing.T) {
		if cfg.Validation.SlugMinLength != 2 {
			t.Errorf("SlugMinLength = %d, want 2", cfg.Validation.SlugMinLength)
		}
		if cfg.Validation.SlugMaxLength != 100 {
			t.Errorf("SlugMaxLength = %d, want 100", cfg.Validation.SlugMaxLength)
		}
	})

	t.Run("CORS defaults", func(t *testing.T) {
		if len(cfg.CORS.AllowedOrigins) != 4 {
			t.Errorf("AllowedOrigins count = %d, want 4", len(cfg.CORS.AllowedOrigins))
		}
	})
}

func TestLoadFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		validate func(t *testing.T, cfg config.Config)
	}{
		{
			name:    "no env vars uses defaults",
			envVars: nil,
			validate: func(t *testing.T, cfg config.Config) {
				if cfg.Pagination.DefaultLimit != 20 {
					t.Errorf("DefaultLimit = %d, want 20", cfg.Pagination.DefaultLimit)
				}
			},
		},
		{
			name: "pagination limit override",
			envVars: map[string]string{
				"DTV_PAGINATION_DEFAULT_LIMIT": "50",
				"DTV_PAGINATION_MAX_LIMIT":     "200",
			},
			validate: func(t *testing.T, cfg config.Config) {
				if cfg.Pagination.DefaultLimit != 50 {
					t.Errorf("DefaultLimit = %d, want 50", cfg.Pagination.DefaultLimit)
				}
				if cfg.Pagination.MaxLimit != 200 {
					t.Errorf("MaxLimit = %d, want 200", cfg.Pagination.MaxLimit)
				}
			},
		},
		{
			name: "max limit capped at 1000",
			envVars: map[string]string{
				"DTV_PAGINATION_MAX_LIMIT": "5000",
			},
			validate: func(t *testing.T, cfg config.Config) {
				// Should remain default since 5000 > 1000
				if cfg.Pagination.MaxLimit != 100 {
					t.Errorf("MaxLimit = %d, want 100 (default)", cfg.Pagination.MaxLimit)
				}
			},
		},
		{
			name: "slug length override",
			envVars: map[string]string{
				"DTV_SLUG_MIN_LENGTH": "3",
				"DTV_SLUG_MAX_LENGTH": "50",
			},
			validate: func(t *testing.T, cfg config.Config) {
				if cfg.Validation.SlugMinLength != 3 {
					t.Errorf("SlugMinLength = %d, want 3", cfg.Validation.SlugMinLength)
				}
				if cfg.Validation.SlugMaxLength != 50 {
					t.Errorf("SlugMaxLength = %d, want 50", cfg.Validation.SlugMaxLength)
				}
			},
		},
		{
			name: "min > max is corrected",
			envVars: map[string]string{
				"DTV_SLUG_MIN_LENGTH": "100",
				"DTV_SLUG_MAX_LENGTH": "50",
			},
			validate: func(t *testing.T, cfg config.Config) {
				if cfg.Validation.SlugMinLength != 50 {
					t.Errorf("SlugMinLength = %d, want 50 (corrected to max)", cfg.Validation.SlugMinLength)
				}
			},
		},
		{
			name: "CORS origins override",
			envVars: map[string]string{
				"CORS_ALLOWED_ORIGINS": "https://example.com,https://api.example.com",
			},
			validate: func(t *testing.T, cfg config.Config) {
				if len(cfg.CORS.AllowedOrigins) != 2 {
					t.Errorf("AllowedOrigins count = %d, want 2", len(cfg.CORS.AllowedOrigins))
				}
				if cfg.CORS.AllowedOrigins[0] != "https://example.com" {
					t.Errorf("First origin = %q, want https://example.com", cfg.CORS.AllowedOrigins[0])
				}
			},
		},
		{
			name: "invalid values use defaults",
			envVars: map[string]string{
				"DTV_PAGINATION_DEFAULT_LIMIT": "not-a-number",
				"DTV_PAGINATION_MAX_LIMIT":     "-10",
			},
			validate: func(t *testing.T, cfg config.Config) {
				if cfg.Pagination.DefaultLimit != 20 {
					t.Errorf("DefaultLimit = %d, want 20 (default)", cfg.Pagination.DefaultLimit)
				}
				if cfg.Pagination.MaxLimit != 100 {
					t.Errorf("MaxLimit = %d, want 100 (default)", cfg.Pagination.MaxLimit)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear relevant env vars
			envKeys := []string{
				"DTV_PAGINATION_DEFAULT_LIMIT",
				"DTV_PAGINATION_MAX_LIMIT",
				"DTV_SLUG_MIN_LENGTH",
				"DTV_SLUG_MAX_LENGTH",
				"CORS_ALLOWED_ORIGINS",
			}
			for _, k := range envKeys {
				os.Unsetenv(k)
			}

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			cfg := config.LoadFromEnv()
			tt.validate(t, cfg)
		})
	}
}

func TestApplyPaginationLimit(t *testing.T) {
	cfg := config.PaginationConfig{
		DefaultLimit: 20,
		MaxLimit:     100,
	}

	tests := []struct {
		name  string
		input int
		want  int
	}{
		{"negative uses default", -1, 20},
		{"zero uses default", 0, 20},
		{"valid limit passes through", 50, 50},
		{"exceeds max uses default", 200, 20},
		{"at max is valid", 100, 100},
		{"one is valid", 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.ApplyPaginationLimit(tt.input)
			if got != tt.want {
				t.Errorf("ApplyPaginationLimit(%d) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsValidSlugLength(t *testing.T) {
	cfg := config.ValidationConfig{
		SlugMinLength: 2,
		SlugMaxLength: 100,
	}

	tests := []struct {
		name   string
		length int
		want   bool
	}{
		{"too short", 1, false},
		{"at min", 2, true},
		{"valid middle", 50, true},
		{"at max", 100, true},
		{"too long", 101, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.IsValidSlugLength(tt.length)
			if got != tt.want {
				t.Errorf("IsValidSlugLength(%d) = %v, want %v", tt.length, got, tt.want)
			}
		})
	}
}

func TestIsOriginAllowed(t *testing.T) {
	cfg := config.CORSConfig{
		AllowedOrigins: []string{
			"http://localhost:3000",
			"https://example.com",
		},
	}

	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		{"allowed origin", "http://localhost:3000", true},
		{"another allowed", "https://example.com", true},
		{"not allowed", "http://malicious.com", false},
		{"empty", "", false},
		{"partial match not allowed", "http://localhost:300", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.IsOriginAllowed(tt.origin)
			if got != tt.want {
				t.Errorf("IsOriginAllowed(%q) = %v, want %v", tt.origin, got, tt.want)
			}
		})
	}
}
