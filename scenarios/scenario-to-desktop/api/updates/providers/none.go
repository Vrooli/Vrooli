package providers

import (
	"context"

	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/updates"
)

// NoneProvider implements the Provider interface for disabled updates.
// Use this when the application should not check for or perform updates.
type NoneProvider struct {
	fullConfig *generation.UpdateConfig
}

// NewNoneProvider creates a new disabled updates provider.
func NewNoneProvider(config *generation.UpdateConfig) *NoneProvider {
	return &NoneProvider{
		fullConfig: config,
	}
}

// Name returns the provider identifier.
func (p *NoneProvider) Name() string {
	return "none"
}

// Validate always returns nil because "none" is always valid.
func (p *NoneProvider) Validate() error {
	return nil
}

// GetPublishConfig returns nil because updates are disabled.
func (p *NoneProvider) GetPublishConfig(channel string) (map[string]interface{}, error) {
	return nil, nil
}

// GenerateManifest returns nil because updates are disabled.
func (p *NoneProvider) GenerateManifest(ctx context.Context, req *updates.ManifestRequest) (*updates.ManifestResult, error) {
	return nil, nil
}

// RequiresManifestUpload returns false because updates are disabled.
func (p *NoneProvider) RequiresManifestUpload() bool {
	return false
}
