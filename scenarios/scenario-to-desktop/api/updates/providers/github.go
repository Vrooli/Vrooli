package providers

import (
	"context"

	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/updates"
)

// GitHubProvider implements the Provider interface for GitHub Releases.
// Unlike the generic provider, GitHub provider doesn't generate manifests because
// electron-builder handles manifest generation and upload automatically when
// using the github provider type.
type GitHubProvider struct {
	config     *generation.GitHubUpdateConfig
	fullConfig *generation.UpdateConfig
	channel    string
}

// NewGitHubProvider creates a new GitHub releases provider.
func NewGitHubProvider(config *generation.UpdateConfig) *GitHubProvider {
	channel := config.Channel
	if channel == "" {
		channel = "stable"
	}

	return &GitHubProvider{
		config:     config.GitHub,
		fullConfig: config,
		channel:    channel,
	}
}

// Name returns the provider identifier.
func (p *GitHubProvider) Name() string {
	return "github"
}

// Validate checks if the provider configuration is valid.
func (p *GitHubProvider) Validate() error {
	// GitHub provider doesn't require configuration - electron-builder can
	// auto-detect repository info from package.json or git remote.
	return nil
}

// GetPublishConfig returns the electron-builder publish configuration.
func (p *GitHubProvider) GetPublishConfig(channel string) (map[string]interface{}, error) {
	config := map[string]interface{}{
		"provider": "github",
	}

	// Add owner/repo if configured
	if p.config != nil {
		if p.config.Owner != "" {
			config["owner"] = p.config.Owner
		}
		if p.config.Repo != "" {
			config["repo"] = p.config.Repo
		}
		if p.config.Private {
			config["private"] = true
		}
	}

	// Set release type based on channel
	if channel == "stable" {
		config["releaseType"] = "release"
	} else {
		config["releaseType"] = "prerelease"
	}

	return config, nil
}

// GenerateManifest returns nil because electron-builder handles manifest generation
// for GitHub releases automatically.
func (p *GitHubProvider) GenerateManifest(ctx context.Context, req *updates.ManifestRequest) (*updates.ManifestResult, error) {
	// GitHub provider doesn't generate manifests - electron-builder does this
	// automatically when publishing to GitHub releases.
	return nil, nil
}

// RequiresManifestUpload returns false because GitHub provider doesn't need
// manual manifest upload - electron-builder handles it.
func (p *GitHubProvider) RequiresManifestUpload() bool {
	return false
}
