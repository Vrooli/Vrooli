package providers

import (
	"fmt"

	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/updates"
)

// DefaultProviderFactory implements ProviderFactory with standard provider creation.
type DefaultProviderFactory struct {
	// genericOpts are passed to GenericProvider on creation.
	genericOpts []GenericProviderOption
}

// FactoryOption configures a DefaultProviderFactory.
type FactoryOption func(*DefaultProviderFactory)

// WithGenericOptions sets options to pass to GenericProvider.
func WithGenericOptions(opts ...GenericProviderOption) FactoryOption {
	return func(f *DefaultProviderFactory) {
		f.genericOpts = opts
	}
}

// NewProviderFactory creates a new provider factory.
func NewProviderFactory(opts ...FactoryOption) *DefaultProviderFactory {
	f := &DefaultProviderFactory{}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// Create instantiates a Provider based on the update configuration.
// Behavior:
// - nil config: Returns GenericProvider with no URL (warning emitted)
// - Empty provider: Defaults to "generic"
// - "generic": Returns GenericProvider (warning if no URL)
// - "github": Returns GitHubProvider
// - "none": Returns NoneProvider
// - Unknown: Returns error
func (f *DefaultProviderFactory) Create(config *generation.UpdateConfig) (updates.Provider, []updates.ValidationWarning, error) {
	var warnings []updates.ValidationWarning

	// Handle nil config - default to generic with warning
	if config == nil {
		config = &generation.UpdateConfig{
			Provider: "generic",
		}
	}

	// Determine provider type, defaulting to generic
	providerType := config.Provider
	if providerType == "" {
		providerType = "generic"
	}

	switch providerType {
	case "generic":
		// Check for missing URL
		if config.Generic == nil || config.Generic.URL == "" {
			warnings = append(warnings, updates.ValidationWarning{
				Code:    "MISSING_URL",
				Message: "Generic update provider configured without URL. Auto-updates will not work until update_config.generic.url is set.",
				Field:   "update_config.generic.url",
			})
		}
		return NewGenericProvider(config, f.genericOpts...), warnings, nil

	case "github":
		return NewGitHubProvider(config), warnings, nil

	case "none":
		return NewNoneProvider(config), warnings, nil

	default:
		return nil, nil, fmt.Errorf("unknown update provider: %s (valid options: generic, github, none)", providerType)
	}
}

// CreateProvider is a convenience function that creates a provider without needing
// a factory instance. Useful for simple cases without custom options.
func CreateProvider(config *generation.UpdateConfig) (updates.Provider, []updates.ValidationWarning, error) {
	factory := NewProviderFactory()
	return factory.Create(config)
}
