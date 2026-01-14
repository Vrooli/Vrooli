package ai

import (
	"context"

	"github.com/sirupsen/logrus"
)

// VrooliProvider implements AIProvider using the Vrooli AI API.
// This provider charges credits when executing prompts.
//
// NOTE: This is a placeholder implementation. The actual Vrooli AI API
// service has not been built yet. This provider always returns unavailable
// until the external service is implemented.
type VrooliProvider struct {
	log    *logrus.Logger
	apiURL string
	model  string
}

// VrooliProviderOptions configures the Vrooli API provider.
type VrooliProviderOptions struct {
	Logger *logrus.Logger
	APIURL string // The Vrooli AI API endpoint
	Model  string
}

// NewVrooliProvider creates a new Vrooli API provider.
func NewVrooliProvider(opts VrooliProviderOptions) *VrooliProvider {
	return &VrooliProvider{
		log:    opts.Logger,
		apiURL: opts.APIURL,
		model:  opts.Model,
	}
}

// Type implements AIProvider.
func (p *VrooliProvider) Type() ProviderType {
	return ProviderTypeVrooli
}

// IsAvailable implements AIProvider.
// Currently always returns false as the Vrooli API is not yet implemented.
func (p *VrooliProvider) IsAvailable(ctx context.Context) bool {
	// TODO: When Vrooli API is implemented:
	// 1. Check if apiURL is configured
	// 2. Check if the API is reachable (health check)
	// 3. Check if user has sufficient credits
	return false
}

// ExecutePrompt implements AIProvider.
// Currently always returns ErrVrooliAPIUnavailable.
func (p *VrooliProvider) ExecutePrompt(ctx context.Context, prompt string) (string, error) {
	// TODO: When Vrooli API is implemented:
	// 1. Make request to Vrooli AI API with user's auth token
	// 2. The API will handle credit checking and charging
	// 3. Return the AI response
	return "", ErrVrooliAPIUnavailable
}

// Model implements AIProvider.
func (p *VrooliProvider) Model() string {
	if p.model == "" {
		return "vrooli-default"
	}
	return p.model
}

// Compile-time interface check
var _ AIProvider = (*VrooliProvider)(nil)
