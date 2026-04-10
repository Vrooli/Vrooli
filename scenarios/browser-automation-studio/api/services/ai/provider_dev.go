package ai

import (
	"context"
	"os/exec"

	"github.com/sirupsen/logrus"
)

// DevProvider implements AIProvider using the local resource-openrouter.
// This provider does not charge credits - it's for development and users
// who have resource-openrouter installed locally.
type DevProvider struct {
	log    *logrus.Logger
	client *OpenRouterClient

	// Cache availability check
	available    bool
	availableSet bool
}

// DevProviderOptions configures the dev mode provider.
type DevProviderOptions struct {
	Logger *logrus.Logger
	Model  string // Optional - defaults to BAS_OPENROUTER_MODEL or gpt-4o-mini
}

// NewDevProvider creates a new dev mode provider using resource-openrouter.
func NewDevProvider(opts DevProviderOptions) *DevProvider {
	client := NewOpenRouterClient(opts.Logger)
	if opts.Model != "" {
		// Override the model if specified
		client.model = opts.Model
	}

	return &DevProvider{
		log:    opts.Logger,
		client: client,
	}
}

// Type implements AIProvider.
func (p *DevProvider) Type() ProviderType {
	return ProviderTypeDev
}

// IsAvailable implements AIProvider.
// Checks if resource-openrouter is available in the PATH.
func (p *DevProvider) IsAvailable(ctx context.Context) bool {
	// Cache the check since it's unlikely to change during runtime
	if p.availableSet {
		return p.available
	}

	// Check if resource-openrouter command is available
	_, err := exec.LookPath(openRouterCommand)
	p.available = err == nil
	p.availableSet = true

	if !p.available {
		p.log.Debug("resource-openrouter not found in PATH, dev mode unavailable")
	}

	return p.available
}

// ExecutePrompt implements AIProvider.
func (p *DevProvider) ExecutePrompt(ctx context.Context, prompt string) (string, error) {
	if !p.IsAvailable(ctx) {
		return "", ErrProviderUnavailable
	}

	return p.client.ExecutePrompt(ctx, prompt)
}

// Model implements AIProvider.
func (p *DevProvider) Model() string {
	return p.client.Model()
}

// Compile-time interface check
var _ AIProvider = (*DevProvider)(nil)
