package updates

import (
	"context"
	"fmt"

	"scenario-to-desktop-api/generation"
)

// ManifestGenerator orchestrates update manifest generation across platforms.
// It uses the ProviderFactory to create the appropriate provider and then
// generates manifests for all artifacts.
type ManifestGenerator struct {
	factory ProviderFactory
	logger  Logger
}

// ManifestGeneratorOption configures a ManifestGenerator.
type ManifestGeneratorOption func(*ManifestGenerator)

// WithManifestLogger sets the logger for the generator.
func WithManifestLogger(l Logger) ManifestGeneratorOption {
	return func(g *ManifestGenerator) {
		g.logger = l
	}
}

// WithProviderFactory sets the provider factory.
func WithProviderFactory(f ProviderFactory) ManifestGeneratorOption {
	return func(g *ManifestGenerator) {
		g.factory = f
	}
}

// NewManifestGenerator creates a new manifest generator.
func NewManifestGenerator(opts ...ManifestGeneratorOption) *ManifestGenerator {
	g := &ManifestGenerator{}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

// GenerateManifestsRequest contains the inputs for manifest generation.
type GenerateManifestsRequest struct {
	// Config is the update configuration from the pipeline.
	Config *generation.UpdateConfig

	// Version is the application version.
	Version string

	// Artifacts maps platform names to artifact file paths.
	Artifacts map[string]string

	// OutputDir is where to write manifest files.
	// If empty, manifests are returned in-memory only.
	OutputDir string

	// Channel overrides the channel in Config if set.
	Channel string
}

// GenerateManifestsResult contains the generated manifests and any warnings.
type GenerateManifestsResult struct {
	// Manifests maps manifest filename to content.
	Manifests map[string][]byte

	// ManifestPaths maps manifest filename to output path.
	ManifestPaths map[string]string

	// Warnings contains non-fatal issues.
	Warnings []ValidationWarning

	// Provider is the provider that was used.
	Provider Provider

	// RequiresUpload is true if manifests need to be uploaded separately.
	RequiresUpload bool
}

// GenerateManifests creates update manifests for the given artifacts.
// Returns nil result (not error) if the provider doesn't require manifests.
func (g *ManifestGenerator) GenerateManifests(ctx context.Context, req *GenerateManifestsRequest) (*GenerateManifestsResult, error) {
	if req == nil {
		return nil, fmt.Errorf("generate manifests request is required")
	}

	// Create provider using factory
	provider, factoryWarnings, err := g.createProvider(req.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create update provider: %w", err)
	}

	result := &GenerateManifestsResult{
		Warnings:       factoryWarnings,
		Provider:       provider,
		RequiresUpload: provider.RequiresManifestUpload(),
	}

	// If provider doesn't need manifests, we're done
	if !provider.RequiresManifestUpload() {
		if g.logger != nil {
			g.logger.Info("Provider does not require manifest upload", "provider", provider.Name())
		}
		return result, nil
	}

	// Skip if no artifacts
	if len(req.Artifacts) == 0 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Code:    "EMPTY_ARTIFACTS",
			Message: "No artifacts provided for manifest generation",
		})
		return result, nil
	}

	// Validate provider configuration
	if err := provider.Validate(); err != nil {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Code:    "PROVIDER_INVALID",
			Message: fmt.Sprintf("Provider validation failed: %v", err),
		})
		// Don't return error - this is a warning, not a fatal error
		return result, nil
	}

	// Determine channel
	channel := req.Channel
	if channel == "" && req.Config != nil {
		channel = req.Config.Channel
	}
	if channel == "" {
		channel = "stable"
	}

	// Build base URL
	publishConfig, err := provider.GetPublishConfig(channel)
	if err != nil {
		return nil, fmt.Errorf("failed to get publish config: %w", err)
	}

	baseURL := ""
	if publishConfig != nil {
		if url, ok := publishConfig["url"].(string); ok {
			baseURL = url
		}
	}

	// Generate manifests
	manifestReq := &ManifestRequest{
		Version:   req.Version,
		Channel:   channel,
		Artifacts: req.Artifacts,
		BaseURL:   baseURL,
		OutputDir: req.OutputDir,
	}

	manifestResult, err := provider.GenerateManifest(ctx, manifestReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate manifests: %w", err)
	}

	if manifestResult != nil {
		result.Manifests = manifestResult.Manifests
		result.ManifestPaths = manifestResult.ManifestPaths
		result.Warnings = append(result.Warnings, manifestResult.Warnings...)
	}

	if g.logger != nil {
		manifestCount := 0
		if result.Manifests != nil {
			manifestCount = len(result.Manifests)
		}
		g.logger.Info("Manifest generation complete",
			"provider", provider.Name(),
			"manifests", manifestCount,
			"warnings", len(result.Warnings))
	}

	return result, nil
}

// createProvider creates a provider using the factory or falls back to defaults.
func (g *ManifestGenerator) createProvider(config *generation.UpdateConfig) (Provider, []ValidationWarning, error) {
	if g.factory != nil {
		return g.factory.Create(config)
	}

	// Fallback to inline creation if no factory configured
	return createDefaultProvider(config)
}

// createDefaultProvider creates a provider without a factory.
// This is used as a fallback when no factory is configured.
func createDefaultProvider(config *generation.UpdateConfig) (Provider, []ValidationWarning, error) {
	// Import from providers package would create a cycle, so we implement inline
	// This should rarely be used - prefer injecting a factory
	var warnings []ValidationWarning

	if config == nil {
		config = &generation.UpdateConfig{
			Provider: "generic",
		}
	}

	providerType := config.Provider
	if providerType == "" {
		providerType = "generic"
	}

	switch providerType {
	case "generic":
		if config.Generic == nil || config.Generic.URL == "" {
			warnings = append(warnings, ValidationWarning{
				Code:    "MISSING_URL",
				Message: "Generic update provider configured without URL. Auto-updates will not work.",
				Field:   "update_config.generic.url",
			})
		}
		// Return a minimal generic provider - note this won't have custom options
		return &minimalGenericProvider{config: config}, warnings, nil

	case "github":
		return &minimalGitHubProvider{config: config}, warnings, nil

	case "none":
		return &minimalNoneProvider{}, warnings, nil

	default:
		return nil, nil, fmt.Errorf("unknown update provider: %s", providerType)
	}
}

// minimalGenericProvider is a fallback implementation when no factory is configured.
type minimalGenericProvider struct {
	config *generation.UpdateConfig
}

func (p *minimalGenericProvider) Name() string { return "generic" }

func (p *minimalGenericProvider) Validate() error {
	if p.config == nil || p.config.Generic == nil || p.config.Generic.URL == "" {
		return fmt.Errorf("generic provider requires URL configuration")
	}
	return nil
}

func (p *minimalGenericProvider) GetPublishConfig(channel string) (map[string]interface{}, error) {
	if p.config == nil || p.config.Generic == nil || p.config.Generic.URL == "" {
		return nil, nil
	}
	return map[string]interface{}{
		"provider": "generic",
		"url":      p.config.Generic.URL + "/" + channel,
	}, nil
}

func (p *minimalGenericProvider) GenerateManifest(ctx context.Context, req *ManifestRequest) (*ManifestResult, error) {
	// Minimal provider can't generate manifests - use factory with full provider
	return nil, fmt.Errorf("manifest generation requires full provider (use factory)")
}

func (p *minimalGenericProvider) RequiresManifestUpload() bool { return true }

// minimalGitHubProvider is a fallback implementation.
type minimalGitHubProvider struct {
	config *generation.UpdateConfig
}

func (p *minimalGitHubProvider) Name() string { return "github" }

func (p *minimalGitHubProvider) Validate() error { return nil }

func (p *minimalGitHubProvider) GetPublishConfig(channel string) (map[string]interface{}, error) {
	config := map[string]interface{}{
		"provider": "github",
	}
	if p.config != nil && p.config.GitHub != nil {
		if p.config.GitHub.Owner != "" {
			config["owner"] = p.config.GitHub.Owner
		}
		if p.config.GitHub.Repo != "" {
			config["repo"] = p.config.GitHub.Repo
		}
	}
	return config, nil
}

func (p *minimalGitHubProvider) GenerateManifest(ctx context.Context, req *ManifestRequest) (*ManifestResult, error) {
	return nil, nil
}

func (p *minimalGitHubProvider) RequiresManifestUpload() bool { return false }

// minimalNoneProvider is a fallback implementation.
type minimalNoneProvider struct{}

func (p *minimalNoneProvider) Name() string    { return "none" }
func (p *minimalNoneProvider) Validate() error { return nil }
func (p *minimalNoneProvider) GetPublishConfig(string) (map[string]interface{}, error) {
	return nil, nil
}

func (p *minimalNoneProvider) GenerateManifest(context.Context, *ManifestRequest) (*ManifestResult, error) {
	return nil, nil
}
func (p *minimalNoneProvider) RequiresManifestUpload() bool { return false }
