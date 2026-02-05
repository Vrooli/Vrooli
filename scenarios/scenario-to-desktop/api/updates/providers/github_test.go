package providers

import (
	"context"
	"testing"

	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/updates"
)

func TestGitHubProvider_Name(t *testing.T) {
	p := NewGitHubProvider(&generation.UpdateConfig{})
	if p.Name() != "github" {
		t.Errorf("expected name 'github', got '%s'", p.Name())
	}
}

func TestGitHubProvider_Validate(t *testing.T) {
	// GitHub provider doesn't require configuration
	p := NewGitHubProvider(&generation.UpdateConfig{})
	if err := p.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGitHubProvider_GetPublishConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *generation.UpdateConfig
		channel     string
		wantOwner   string
		wantRepo    string
		wantRelease string
		wantPrivate bool
	}{
		{
			name: "basic config",
			config: &generation.UpdateConfig{
				GitHub: &generation.GitHubUpdateConfig{
					Owner: "myorg",
					Repo:  "myapp",
				},
			},
			channel:     "stable",
			wantOwner:   "myorg",
			wantRepo:    "myapp",
			wantRelease: "release",
		},
		{
			name: "prerelease channel",
			config: &generation.UpdateConfig{
				GitHub: &generation.GitHubUpdateConfig{
					Owner: "myorg",
					Repo:  "myapp",
				},
			},
			channel:     "beta",
			wantOwner:   "myorg",
			wantRepo:    "myapp",
			wantRelease: "prerelease",
		},
		{
			name: "private repo",
			config: &generation.UpdateConfig{
				GitHub: &generation.GitHubUpdateConfig{
					Owner:   "myorg",
					Repo:    "private-app",
					Private: true,
				},
			},
			channel:     "stable",
			wantOwner:   "myorg",
			wantRepo:    "private-app",
			wantRelease: "release",
			wantPrivate: true,
		},
		{
			name: "nil github config",
			config: &generation.UpdateConfig{
				GitHub: nil,
			},
			channel:     "stable",
			wantRelease: "release",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewGitHubProvider(tt.config)
			cfg, err := p.GetPublishConfig(tt.channel)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg == nil {
				t.Fatal("expected non-nil config")
			}

			if provider, ok := cfg["provider"].(string); !ok || provider != "github" {
				t.Errorf("expected provider=github, got %v", cfg["provider"])
			}

			if tt.wantOwner != "" {
				if owner, ok := cfg["owner"].(string); !ok || owner != tt.wantOwner {
					t.Errorf("owner = %v, want %s", cfg["owner"], tt.wantOwner)
				}
			}

			if tt.wantRepo != "" {
				if repo, ok := cfg["repo"].(string); !ok || repo != tt.wantRepo {
					t.Errorf("repo = %v, want %s", cfg["repo"], tt.wantRepo)
				}
			}

			if releaseType, ok := cfg["releaseType"].(string); !ok || releaseType != tt.wantRelease {
				t.Errorf("releaseType = %v, want %s", cfg["releaseType"], tt.wantRelease)
			}

			if tt.wantPrivate {
				if private, ok := cfg["private"].(bool); !ok || !private {
					t.Error("expected private=true")
				}
			}
		})
	}
}

func TestGitHubProvider_GenerateManifest(t *testing.T) {
	p := NewGitHubProvider(&generation.UpdateConfig{})

	// GitHub provider should return nil (electron-builder handles manifests)
	result, err := p.GenerateManifest(context.Background(), &updates.ManifestRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil result from GitHub provider")
	}
}

func TestGitHubProvider_RequiresManifestUpload(t *testing.T) {
	p := NewGitHubProvider(&generation.UpdateConfig{})
	if p.RequiresManifestUpload() {
		t.Error("github provider should not require manifest upload")
	}
}
