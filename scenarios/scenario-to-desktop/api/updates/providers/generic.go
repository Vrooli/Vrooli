// Package providers contains update provider implementations.
package providers

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/updates"
)

// GenericProvider implements the Provider interface for self-hosted update servers.
// It generates electron-updater compatible manifest files (latest.yml, etc.) that
// can be served from any HTTP server (S3, LPBS, nginx, etc.).
type GenericProvider struct {
	config     *generation.GenericUpdateConfig
	hashCalc   updates.HashCalculator
	fileStat   updates.FileStatProvider
	writer     updates.ManifestWriter
	logger     updates.Logger
	channel    string
	fullConfig *generation.UpdateConfig
}

// GenericProviderOption configures a GenericProvider.
type GenericProviderOption func(*GenericProvider)

// WithHashCalculator sets the hash calculator for testing.
func WithHashCalculator(h updates.HashCalculator) GenericProviderOption {
	return func(p *GenericProvider) {
		p.hashCalc = h
	}
}

// WithFileStatProvider sets the file stat provider for testing.
func WithFileStatProvider(fs updates.FileStatProvider) GenericProviderOption {
	return func(p *GenericProvider) {
		p.fileStat = fs
	}
}

// WithManifestWriter sets the manifest writer for testing.
func WithManifestWriter(w updates.ManifestWriter) GenericProviderOption {
	return func(p *GenericProvider) {
		p.writer = w
	}
}

// WithLogger sets the logger.
func WithLogger(l updates.Logger) GenericProviderOption {
	return func(p *GenericProvider) {
		p.logger = l
	}
}

// NewGenericProvider creates a new generic update provider.
func NewGenericProvider(config *generation.UpdateConfig, opts ...GenericProviderOption) *GenericProvider {
	p := &GenericProvider{
		config:     config.Generic,
		fullConfig: config,
		channel:    config.Channel,
		hashCalc:   &realHashCalculator{},
		fileStat:   &realFileStatProvider{},
		writer:     &realManifestWriter{},
	}

	if p.channel == "" {
		p.channel = "stable"
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Name returns the provider identifier.
func (p *GenericProvider) Name() string {
	return "generic"
}

// Validate checks if the provider configuration is valid.
func (p *GenericProvider) Validate() error {
	if p.config == nil || p.config.URL == "" {
		return fmt.Errorf("generic provider requires URL configuration")
	}
	return nil
}

// GetPublishConfig returns the electron-builder publish configuration.
func (p *GenericProvider) GetPublishConfig(channel string) (map[string]interface{}, error) {
	if p.config == nil || p.config.URL == "" {
		return nil, nil
	}

	url := p.buildChannelURL(channel)

	return map[string]interface{}{
		"provider":                "generic",
		"url":                     url,
		"useMultipleRangeRequest": false,
	}, nil
}

// GenerateManifest creates update manifest files for the given artifacts.
func (p *GenericProvider) GenerateManifest(ctx context.Context, req *updates.ManifestRequest) (*updates.ManifestResult, error) {
	if req == nil {
		return nil, fmt.Errorf("manifest request is required")
	}

	if len(req.Artifacts) == 0 {
		return &updates.ManifestResult{
			Manifests: make(map[string][]byte),
			Warnings: []updates.ValidationWarning{{
				Code:    "EMPTY_ARTIFACTS",
				Message: "No artifacts provided for manifest generation",
			}},
		}, nil
	}

	result := &updates.ManifestResult{
		Manifests:     make(map[string][]byte),
		ManifestPaths: make(map[string]string),
	}

	releaseDate := req.ReleaseDate
	if releaseDate.IsZero() {
		releaseDate = time.Now()
	}

	// Generate manifest for each platform
	for platform, artifactPath := range req.Artifacts {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		normalizedPlatform := updates.NormalizePlatform(platform)
		manifestFilename := updates.PlatformManifestFilename(normalizedPlatform)

		manifest, err := p.generatePlatformManifest(req, artifactPath, releaseDate)
		if err != nil {
			result.Warnings = append(result.Warnings, updates.ValidationWarning{
				Code:    "MANIFEST_GENERATION_FAILED",
				Message: fmt.Sprintf("Failed to generate manifest for %s: %v", platform, err),
				Field:   platform,
			})
			continue
		}

		manifestYAML, err := yaml.Marshal(manifest)
		if err != nil {
			result.Warnings = append(result.Warnings, updates.ValidationWarning{
				Code:    "YAML_MARSHAL_FAILED",
				Message: fmt.Sprintf("Failed to marshal manifest for %s: %v", platform, err),
				Field:   platform,
			})
			continue
		}

		result.Manifests[manifestFilename] = manifestYAML

		// Write to file if output directory specified
		if req.OutputDir != "" {
			outputPath := filepath.Join(req.OutputDir, manifestFilename)
			if err := p.writer.WriteFile(outputPath, manifestYAML); err != nil {
				result.Warnings = append(result.Warnings, updates.ValidationWarning{
					Code:    "MANIFEST_WRITE_FAILED",
					Message: fmt.Sprintf("Failed to write manifest %s: %v", outputPath, err),
					Field:   platform,
				})
				continue
			}
			result.ManifestPaths[manifestFilename] = outputPath

			if p.logger != nil {
				p.logger.Info("Generated manifest", "file", manifestFilename, "path", outputPath)
			}
		}
	}

	return result, nil
}

// RequiresManifestUpload returns true because generic provider needs manifests uploaded.
func (p *GenericProvider) RequiresManifestUpload() bool {
	return true
}

// generatePlatformManifest creates the electron-updater manifest for a single platform.
func (p *GenericProvider) generatePlatformManifest(req *updates.ManifestRequest, artifactPath string, releaseDate time.Time) (*updates.ElectronManifest, error) {
	// Get file info
	fileInfo, err := p.fileStat.Stat(artifactPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat artifact %s: %w", artifactPath, err)
	}

	// Calculate hash
	hash, err := p.hashCalc.CalculateSHA512(artifactPath)
	if err != nil {
		return nil, fmt.Errorf("failed to hash artifact %s: %w", artifactPath, err)
	}

	filename := fileInfo.Name
	if filename == "" {
		filename = filepath.Base(artifactPath)
	}

	// Build download URL
	var downloadURL string
	if req.BaseURL != "" {
		downloadURL = strings.TrimSuffix(req.BaseURL, "/") + "/" + filename
	} else {
		downloadURL = filename
	}

	manifest := &updates.ElectronManifest{
		Version:     req.Version,
		Path:        filename,
		Sha512:      hash,
		ReleaseDate: releaseDate.Format(time.RFC3339),
		Files: []updates.ElectronManifestFile{
			{
				URL:    downloadURL,
				Sha512: hash,
				Size:   fileInfo.Size,
			},
		},
	}

	return manifest, nil
}

// buildChannelURL constructs the update URL for a channel.
func (p *GenericProvider) buildChannelURL(channel string) string {
	if p.config == nil || p.config.URL == "" {
		return ""
	}

	url := strings.TrimSuffix(p.config.URL, "/")

	if p.config.ChannelPath != "" {
		// Use configured channel path template
		channelPath := strings.ReplaceAll(p.config.ChannelPath, "{channel}", channel)
		url = url + channelPath
	} else {
		// Default: append channel as path segment
		url = url + "/" + channel
	}

	return url
}

// realHashCalculator implements HashCalculator using crypto/sha512.
type realHashCalculator struct{}

func (r *realHashCalculator) CalculateSHA512(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha512.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

// realFileStatProvider implements FileStatProvider using os.Stat.
type realFileStatProvider struct{}

func (r *realFileStatProvider) Stat(path string) (updates.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return updates.FileInfo{}, err
	}

	return updates.FileInfo{
		Size: info.Size(),
		Name: info.Name(),
	}, nil
}

// realManifestWriter implements ManifestWriter using os.WriteFile.
type realManifestWriter struct{}

func (r *realManifestWriter) WriteFile(path string, content []byte) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}
