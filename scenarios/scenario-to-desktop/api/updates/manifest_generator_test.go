package updates

import (
	"context"
	"testing"

	"scenario-to-desktop-api/generation"
)

// mockProviderFactory is a test double for ProviderFactory.
type mockProviderFactory struct {
	provider Provider
	warnings []ValidationWarning
	err      error
}

func (m *mockProviderFactory) Create(config *generation.UpdateConfig) (Provider, []ValidationWarning, error) {
	return m.provider, m.warnings, m.err
}

// mockProvider is a test double for Provider.
type mockProvider struct {
	name             string
	validateErr      error
	publishConfig    map[string]interface{}
	publishConfigErr error
	manifestResult   *ManifestResult
	manifestErr      error
	requiresUpload   bool
}

func (m *mockProvider) Name() string    { return m.name }
func (m *mockProvider) Validate() error { return m.validateErr }
func (m *mockProvider) GetPublishConfig(channel string) (map[string]interface{}, error) {
	return m.publishConfig, m.publishConfigErr
}

func (m *mockProvider) GenerateManifest(ctx context.Context, req *ManifestRequest) (*ManifestResult, error) {
	return m.manifestResult, m.manifestErr
}
func (m *mockProvider) RequiresManifestUpload() bool { return m.requiresUpload }

func TestManifestGenerator_GenerateManifests_NilRequest(t *testing.T) {
	g := NewManifestGenerator()
	_, err := g.GenerateManifests(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil request")
	}
}

func TestManifestGenerator_GenerateManifests_NoManifestRequired(t *testing.T) {
	provider := &mockProvider{
		name:           "github",
		requiresUpload: false,
	}

	factory := &mockProviderFactory{provider: provider}
	g := NewManifestGenerator(WithProviderFactory(factory))

	req := &GenerateManifestsRequest{
		Config: &generation.UpdateConfig{Provider: "github"},
	}

	result, err := g.GenerateManifests(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.RequiresUpload {
		t.Error("github provider should not require upload")
	}

	if result.Provider.Name() != "github" {
		t.Errorf("expected github provider, got %s", result.Provider.Name())
	}
}

func TestManifestGenerator_GenerateManifests_GenericProvider(t *testing.T) {
	manifestResult := &ManifestResult{
		Manifests: map[string][]byte{
			"latest.yml": []byte("version: 1.0.0"),
		},
		ManifestPaths: map[string]string{
			"latest.yml": "/output/latest.yml",
		},
	}

	provider := &mockProvider{
		name:           "generic",
		requiresUpload: true,
		publishConfig:  map[string]interface{}{"url": "https://updates.example.com/stable"},
		manifestResult: manifestResult,
	}

	factory := &mockProviderFactory{provider: provider}
	g := NewManifestGenerator(WithProviderFactory(factory))

	req := &GenerateManifestsRequest{
		Config: &generation.UpdateConfig{
			Provider: "generic",
			Generic: &generation.GenericUpdateConfig{
				URL: "https://updates.example.com",
			},
		},
		Version:   "1.0.0",
		Artifacts: map[string]string{"win": "/path/to/app.exe"},
		OutputDir: "/output",
	}

	result, err := g.GenerateManifests(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.RequiresUpload {
		t.Error("generic provider should require upload")
	}

	if len(result.Manifests) != 1 {
		t.Errorf("expected 1 manifest, got %d", len(result.Manifests))
	}

	if _, ok := result.Manifests["latest.yml"]; !ok {
		t.Error("expected latest.yml manifest")
	}
}

func TestManifestGenerator_GenerateManifests_FactoryWarnings(t *testing.T) {
	provider := &mockProvider{
		name:           "generic",
		requiresUpload: true,
		validateErr:    nil,
	}

	factory := &mockProviderFactory{
		provider: provider,
		warnings: []ValidationWarning{
			{Code: "MISSING_URL", Message: "URL not configured"},
		},
	}

	g := NewManifestGenerator(WithProviderFactory(factory))

	req := &GenerateManifestsRequest{
		Config:    &generation.UpdateConfig{Provider: "generic"},
		Artifacts: map[string]string{},
	}

	result, err := g.GenerateManifests(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Warnings) < 1 {
		t.Error("expected at least 1 warning from factory")
	}

	hasFactoryWarning := false
	for _, w := range result.Warnings {
		if w.Code == "MISSING_URL" {
			hasFactoryWarning = true
			break
		}
	}
	if !hasFactoryWarning {
		t.Error("expected MISSING_URL warning from factory")
	}
}

func TestManifestGenerator_GenerateManifests_EmptyArtifacts(t *testing.T) {
	provider := &mockProvider{
		name:           "generic",
		requiresUpload: true,
	}

	factory := &mockProviderFactory{provider: provider}
	g := NewManifestGenerator(WithProviderFactory(factory))

	req := &GenerateManifestsRequest{
		Config: &generation.UpdateConfig{
			Provider: "generic",
			Generic:  &generation.GenericUpdateConfig{URL: "https://example.com"},
		},
		Artifacts: map[string]string{},
	}

	result, err := g.GenerateManifests(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have an empty artifacts warning
	hasWarning := false
	for _, w := range result.Warnings {
		if w.Code == "EMPTY_ARTIFACTS" {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Error("expected EMPTY_ARTIFACTS warning")
	}
}

func TestManifestGenerator_GenerateManifests_ValidationFailed(t *testing.T) {
	provider := &mockProvider{
		name:           "generic",
		requiresUpload: true,
		validateErr:    errValidationFailed,
	}

	factory := &mockProviderFactory{provider: provider}
	g := NewManifestGenerator(WithProviderFactory(factory))

	req := &GenerateManifestsRequest{
		Config:    &generation.UpdateConfig{Provider: "generic"},
		Artifacts: map[string]string{"win": "/path/to/app.exe"},
	}

	result, err := g.GenerateManifests(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have a validation warning, not an error
	hasWarning := false
	for _, w := range result.Warnings {
		if w.Code == "PROVIDER_INVALID" {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Error("expected PROVIDER_INVALID warning")
	}
}

// errValidationFailed is a sentinel error for testing.
var errValidationFailed = &validationError{}

type validationError struct{}

func (e *validationError) Error() string { return "validation failed" }

func TestManifestGenerator_DefaultChannel(t *testing.T) {
	provider := &mockProvider{
		name:           "generic",
		requiresUpload: true,
		publishConfig:  map[string]interface{}{"url": "https://updates.example.com/stable"},
		manifestResult: &ManifestResult{Manifests: map[string][]byte{}},
	}

	factory := &mockProviderFactory{provider: provider}
	g := NewManifestGenerator(WithProviderFactory(factory))

	req := &GenerateManifestsRequest{
		Config: &generation.UpdateConfig{
			Provider: "generic",
			Generic:  &generation.GenericUpdateConfig{URL: "https://example.com"},
			// Channel not set - should default to "stable"
		},
		Artifacts: map[string]string{"win": "/path/to/app.exe"},
	}

	_, err := g.GenerateManifests(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The channel defaulting is tested implicitly - if it panics, test fails
}

// =============================================================================
// Tests for createDefaultProvider fallback path (no factory configured)
// =============================================================================

func TestCreateDefaultProvider_NilConfig(t *testing.T) {
	provider, warnings, err := createDefaultProvider(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.Name() != "generic" {
		t.Errorf("expected generic provider, got %s", provider.Name())
	}

	// Should have MISSING_URL warning
	hasWarning := false
	for _, w := range warnings {
		if w.Code == "MISSING_URL" {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Error("expected MISSING_URL warning for nil config")
	}
}

func TestCreateDefaultProvider_EmptyProvider(t *testing.T) {
	config := &generation.UpdateConfig{
		Provider: "",
	}

	provider, warnings, err := createDefaultProvider(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.Name() != "generic" {
		t.Errorf("expected generic provider, got %s", provider.Name())
	}

	// Should have MISSING_URL warning
	if len(warnings) == 0 {
		t.Error("expected MISSING_URL warning for empty provider")
	}
}

func TestCreateDefaultProvider_GenericWithURL(t *testing.T) {
	config := &generation.UpdateConfig{
		Provider: "generic",
		Generic: &generation.GenericUpdateConfig{
			URL: "https://updates.example.com",
		},
	}

	provider, warnings, err := createDefaultProvider(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.Name() != "generic" {
		t.Errorf("expected generic provider, got %s", provider.Name())
	}

	if len(warnings) != 0 {
		t.Errorf("expected no warnings with URL configured, got %d", len(warnings))
	}
}

func TestCreateDefaultProvider_GenericWithoutURL(t *testing.T) {
	config := &generation.UpdateConfig{
		Provider: "generic",
		Generic:  nil,
	}

	provider, warnings, err := createDefaultProvider(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.Name() != "generic" {
		t.Errorf("expected generic provider, got %s", provider.Name())
	}

	hasWarning := false
	for _, w := range warnings {
		if w.Code == "MISSING_URL" && w.Field == "update_config.generic.url" {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Error("expected MISSING_URL warning with correct field")
	}
}

func TestCreateDefaultProvider_GitHub(t *testing.T) {
	config := &generation.UpdateConfig{
		Provider: "github",
	}

	provider, warnings, err := createDefaultProvider(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.Name() != "github" {
		t.Errorf("expected github provider, got %s", provider.Name())
	}

	if len(warnings) != 0 {
		t.Errorf("expected no warnings for github provider, got %d", len(warnings))
	}
}

func TestCreateDefaultProvider_None(t *testing.T) {
	config := &generation.UpdateConfig{
		Provider: "none",
	}

	provider, warnings, err := createDefaultProvider(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.Name() != "none" {
		t.Errorf("expected none provider, got %s", provider.Name())
	}

	if len(warnings) != 0 {
		t.Errorf("expected no warnings for none provider, got %d", len(warnings))
	}
}

func TestCreateDefaultProvider_UnknownProvider(t *testing.T) {
	config := &generation.UpdateConfig{
		Provider: "unknown-provider",
	}

	_, _, err := createDefaultProvider(config)
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

// =============================================================================
// Tests for minimalGenericProvider
// =============================================================================

func TestMinimalGenericProvider_Name(t *testing.T) {
	p := &minimalGenericProvider{}
	if p.Name() != "generic" {
		t.Errorf("expected name 'generic', got '%s'", p.Name())
	}
}

func TestMinimalGenericProvider_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *generation.UpdateConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "nil generic config",
			config: &generation.UpdateConfig{
				Generic: nil,
			},
			wantErr: true,
		},
		{
			name: "empty URL",
			config: &generation.UpdateConfig{
				Generic: &generation.GenericUpdateConfig{URL: ""},
			},
			wantErr: true,
		},
		{
			name: "valid URL",
			config: &generation.UpdateConfig{
				Generic: &generation.GenericUpdateConfig{URL: "https://example.com"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &minimalGenericProvider{config: tt.config}
			err := p.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMinimalGenericProvider_GetPublishConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *generation.UpdateConfig
		channel string
		wantNil bool
		wantURL string
	}{
		{
			name:    "nil config returns nil",
			config:  nil,
			channel: "stable",
			wantNil: true,
		},
		{
			name: "nil generic config returns nil",
			config: &generation.UpdateConfig{
				Generic: nil,
			},
			channel: "stable",
			wantNil: true,
		},
		{
			name: "empty URL returns nil",
			config: &generation.UpdateConfig{
				Generic: &generation.GenericUpdateConfig{URL: ""},
			},
			channel: "stable",
			wantNil: true,
		},
		{
			name: "valid URL returns config",
			config: &generation.UpdateConfig{
				Generic: &generation.GenericUpdateConfig{URL: "https://updates.example.com"},
			},
			channel: "stable",
			wantNil: false,
			wantURL: "https://updates.example.com/stable",
		},
		{
			name: "beta channel",
			config: &generation.UpdateConfig{
				Generic: &generation.GenericUpdateConfig{URL: "https://updates.example.com"},
			},
			channel: "beta",
			wantNil: false,
			wantURL: "https://updates.example.com/beta",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &minimalGenericProvider{config: tt.config}
			cfg, err := p.GetPublishConfig(tt.channel)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantNil {
				if cfg != nil {
					t.Errorf("expected nil config, got %v", cfg)
				}
				return
			}

			if cfg == nil {
				t.Fatal("expected non-nil config")
			}

			url, ok := cfg["url"].(string)
			if !ok {
				t.Fatal("expected url to be a string")
			}
			if url != tt.wantURL {
				t.Errorf("url = %s, want %s", url, tt.wantURL)
			}

			provider, ok := cfg["provider"].(string)
			if !ok || provider != "generic" {
				t.Errorf("provider = %v, want 'generic'", cfg["provider"])
			}
		})
	}
}

func TestMinimalGenericProvider_GenerateManifest(t *testing.T) {
	p := &minimalGenericProvider{}
	_, err := p.GenerateManifest(context.Background(), &ManifestRequest{})
	if err == nil {
		t.Error("expected error - minimal provider cannot generate manifests")
	}
}

func TestMinimalGenericProvider_RequiresManifestUpload(t *testing.T) {
	p := &minimalGenericProvider{}
	if !p.RequiresManifestUpload() {
		t.Error("minimal generic provider should require manifest upload")
	}
}

// =============================================================================
// Tests for minimalGitHubProvider
// =============================================================================

func TestMinimalGitHubProvider_Name(t *testing.T) {
	p := &minimalGitHubProvider{}
	if p.Name() != "github" {
		t.Errorf("expected name 'github', got '%s'", p.Name())
	}
}

func TestMinimalGitHubProvider_Validate(t *testing.T) {
	p := &minimalGitHubProvider{}
	if err := p.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMinimalGitHubProvider_GetPublishConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *generation.UpdateConfig
		wantOwner string
		wantRepo  string
	}{
		{
			name:   "nil config",
			config: nil,
		},
		{
			name: "nil github config",
			config: &generation.UpdateConfig{
				GitHub: nil,
			},
		},
		{
			name: "with owner and repo",
			config: &generation.UpdateConfig{
				GitHub: &generation.GitHubUpdateConfig{
					Owner: "myorg",
					Repo:  "myrepo",
				},
			},
			wantOwner: "myorg",
			wantRepo:  "myrepo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &minimalGitHubProvider{config: tt.config}
			cfg, err := p.GetPublishConfig("stable")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg == nil {
				t.Fatal("expected non-nil config")
			}

			provider, ok := cfg["provider"].(string)
			if !ok || provider != "github" {
				t.Errorf("provider = %v, want 'github'", cfg["provider"])
			}

			if tt.wantOwner != "" {
				owner, _ := cfg["owner"].(string)
				if owner != tt.wantOwner {
					t.Errorf("owner = %s, want %s", owner, tt.wantOwner)
				}
			}

			if tt.wantRepo != "" {
				repo, _ := cfg["repo"].(string)
				if repo != tt.wantRepo {
					t.Errorf("repo = %s, want %s", repo, tt.wantRepo)
				}
			}
		})
	}
}

func TestMinimalGitHubProvider_GenerateManifest(t *testing.T) {
	p := &minimalGitHubProvider{}
	result, err := p.GenerateManifest(context.Background(), &ManifestRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil result from github provider")
	}
}

func TestMinimalGitHubProvider_RequiresManifestUpload(t *testing.T) {
	p := &minimalGitHubProvider{}
	if p.RequiresManifestUpload() {
		t.Error("github provider should not require manifest upload")
	}
}

// =============================================================================
// Tests for minimalNoneProvider
// =============================================================================

func TestMinimalNoneProvider_Name(t *testing.T) {
	p := &minimalNoneProvider{}
	if p.Name() != "none" {
		t.Errorf("expected name 'none', got '%s'", p.Name())
	}
}

func TestMinimalNoneProvider_Validate(t *testing.T) {
	p := &minimalNoneProvider{}
	if err := p.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMinimalNoneProvider_GetPublishConfig(t *testing.T) {
	p := &minimalNoneProvider{}
	cfg, err := p.GetPublishConfig("stable")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Error("expected nil config from none provider")
	}
}

func TestMinimalNoneProvider_GenerateManifest(t *testing.T) {
	p := &minimalNoneProvider{}
	result, err := p.GenerateManifest(context.Background(), &ManifestRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil result from none provider")
	}
}

func TestMinimalNoneProvider_RequiresManifestUpload(t *testing.T) {
	p := &minimalNoneProvider{}
	if p.RequiresManifestUpload() {
		t.Error("none provider should not require manifest upload")
	}
}

// =============================================================================
// Tests for ManifestGenerator without factory (uses fallback)
// =============================================================================

func TestManifestGenerator_NoFactory_UsesDefaultProvider(t *testing.T) {
	// No factory configured - should use createDefaultProvider fallback
	g := NewManifestGenerator()

	req := &GenerateManifestsRequest{
		Config: &generation.UpdateConfig{
			Provider: "github",
		},
		Artifacts: map[string]string{"win": "/path/to/app.exe"},
	}

	result, err := g.GenerateManifests(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// GitHub provider doesn't require upload, so result should reflect that
	if result.RequiresUpload {
		t.Error("github provider should not require upload")
	}

	if result.Provider == nil {
		t.Fatal("expected provider to be set")
	}

	if result.Provider.Name() != "github" {
		t.Errorf("expected github provider, got %s", result.Provider.Name())
	}
}

func TestManifestGenerator_NoFactory_GenericWithoutURL(t *testing.T) {
	g := NewManifestGenerator()

	req := &GenerateManifestsRequest{
		Config: &generation.UpdateConfig{
			Provider: "generic",
			// No URL configured
		},
		Artifacts: map[string]string{"win": "/path/to/app.exe"},
	}

	result, err := g.GenerateManifests(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have MISSING_URL warning
	hasWarning := false
	for _, w := range result.Warnings {
		if w.Code == "MISSING_URL" {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Error("expected MISSING_URL warning")
	}
}

func TestManifestGenerator_NoFactory_NoneProvider(t *testing.T) {
	g := NewManifestGenerator()

	req := &GenerateManifestsRequest{
		Config: &generation.UpdateConfig{
			Provider: "none",
		},
		Artifacts: map[string]string{"win": "/path/to/app.exe"},
	}

	result, err := g.GenerateManifests(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.RequiresUpload {
		t.Error("none provider should not require upload")
	}
}

func TestManifestGenerator_NoFactory_UnknownProvider(t *testing.T) {
	g := NewManifestGenerator()

	req := &GenerateManifestsRequest{
		Config: &generation.UpdateConfig{
			Provider: "unknown",
		},
		Artifacts: map[string]string{"win": "/path/to/app.exe"},
	}

	_, err := g.GenerateManifests(context.Background(), req)
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}
