package providers

import (
	"testing"

	"scenario-to-desktop-api/generation"
)

func TestDefaultProviderFactory_Create(t *testing.T) {
	tests := []struct {
		name             string
		config           *generation.UpdateConfig
		wantProviderName string
		wantWarnings     int
		wantErr          bool
	}{
		{
			name:             "nil config defaults to generic with warning",
			config:           nil,
			wantProviderName: "generic",
			wantWarnings:     1, // Missing URL warning
		},
		{
			name:             "empty provider defaults to generic",
			config:           &generation.UpdateConfig{Provider: ""},
			wantProviderName: "generic",
			wantWarnings:     1, // Missing URL warning
		},
		{
			name: "explicit generic with URL",
			config: &generation.UpdateConfig{
				Provider: "generic",
				Generic: &generation.GenericUpdateConfig{
					URL: "https://updates.example.com",
				},
			},
			wantProviderName: "generic",
			wantWarnings:     0,
		},
		{
			name:             "explicit generic without URL",
			config:           &generation.UpdateConfig{Provider: "generic"},
			wantProviderName: "generic",
			wantWarnings:     1,
		},
		{
			name:             "github provider",
			config:           &generation.UpdateConfig{Provider: "github"},
			wantProviderName: "github",
			wantWarnings:     0,
		},
		{
			name:             "none provider",
			config:           &generation.UpdateConfig{Provider: "none"},
			wantProviderName: "none",
			wantWarnings:     0,
		},
		{
			name:    "unknown provider",
			config:  &generation.UpdateConfig{Provider: "unknown"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewProviderFactory()
			provider, warnings, err := f.Create(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if provider.Name() != tt.wantProviderName {
				t.Errorf("provider name = %s, want %s", provider.Name(), tt.wantProviderName)
			}

			if len(warnings) != tt.wantWarnings {
				t.Errorf("warnings count = %d, want %d", len(warnings), tt.wantWarnings)
				for _, w := range warnings {
					t.Logf("  warning: [%s] %s", w.Code, w.Message)
				}
			}
		})
	}
}

func TestDefaultProviderFactory_CreateWithGenericOptions(t *testing.T) {
	mockHash := &mockHashCalculator{hash: "test"}

	f := NewProviderFactory(
		WithGenericOptions(WithHashCalculator(mockHash)),
	)

	config := &generation.UpdateConfig{
		Provider: "generic",
		Generic: &generation.GenericUpdateConfig{
			URL: "https://example.com",
		},
	}

	provider, _, err := f.Create(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the option was applied by checking the provider is a GenericProvider
	gp, ok := provider.(*GenericProvider)
	if !ok {
		t.Fatal("expected GenericProvider")
	}

	// The mock hash calculator should be set
	hash, _ := gp.hashCalc.CalculateSHA512("")
	if hash != "test" {
		t.Error("expected mock hash calculator to be used")
	}
}

func TestCreateProvider(t *testing.T) {
	// Test the convenience function
	provider, warnings, err := CreateProvider(&generation.UpdateConfig{
		Provider: "github",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.Name() != "github" {
		t.Errorf("expected github provider, got %s", provider.Name())
	}

	if len(warnings) != 0 {
		t.Errorf("expected no warnings for github, got %d", len(warnings))
	}
}

func TestCreateProvider_MissingURLWarning(t *testing.T) {
	provider, warnings, err := CreateProvider(&generation.UpdateConfig{
		Provider: "generic",
		// No generic.url configured
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.Name() != "generic" {
		t.Errorf("expected generic provider, got %s", provider.Name())
	}

	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}

	if warnings[0].Code != "MISSING_URL" {
		t.Errorf("expected MISSING_URL warning, got %s", warnings[0].Code)
	}

	if warnings[0].Field != "update_config.generic.url" {
		t.Errorf("expected field update_config.generic.url, got %s", warnings[0].Field)
	}
}
