// Package platforms provides platform-specific signing tool detection and certificate discovery.
//
// This package abstracts the differences between Windows (Authenticode),
// macOS (Developer ID + Notarization), and Linux (GPG) signing workflows.
package platforms

import (
	"context"

	"scenario-to-desktop-api/signing/types"
)

// PlatformDetector detects signing tools and certificates for a specific platform.
type PlatformDetector interface {
	// Platform returns the platform this detector handles.
	Platform() string

	// DetectTools returns information about available signing tools.
	DetectTools(ctx context.Context) []types.ToolDetectionResult

	// DiscoverCertificates finds available signing certificates/identities.
	DiscoverCertificates(ctx context.Context) ([]types.DiscoveredCertificate, error)
}

// DiscoveryResult contains the result of platform discovery.
type DiscoveryResult struct {
	Platform     string                       `json:"platform"`
	Tools        []types.ToolDetectionResult  `json:"tools"`
	Certificates []types.DiscoveredCertificate `json:"certificates"`
	Errors       []string                     `json:"errors,omitempty"`
}

// MultiPlatformDetector runs detection across multiple platforms.
type MultiPlatformDetector struct {
	detectors []PlatformDetector
}

// DetectorOption configures detector creation.
type DetectorOption func(*detectorConfig)

type detectorConfig struct {
	fs  FileSystem
	cmd CommandRunner
	env EnvironmentReader
}

// FileSystem provides file system operations for platform detection.
type FileSystem interface {
	Exists(path string) bool
}

// CommandRunner executes system commands for platform detection.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) (stdout []byte, stderr []byte, err error)
	LookPath(name string) (string, error)
}

// EnvironmentReader reads environment variables.
type EnvironmentReader interface {
	GetEnv(key string) string
	LookupEnv(key string) (string, bool)
}

// WithFileSystem sets a custom file system for detectors.
func WithFileSystem(fs FileSystem) DetectorOption {
	return func(c *detectorConfig) {
		c.fs = fs
	}
}

// WithCommandRunner sets a custom command runner for detectors.
func WithCommandRunner(cmd CommandRunner) DetectorOption {
	return func(c *detectorConfig) {
		c.cmd = cmd
	}
}

// WithEnvironmentReader sets a custom environment reader for detectors.
func WithEnvironmentReader(env EnvironmentReader) DetectorOption {
	return func(c *detectorConfig) {
		c.env = env
	}
}

// NewMultiPlatformDetector creates a detector that runs on all platforms.
func NewMultiPlatformDetector(opts ...DetectorOption) *MultiPlatformDetector {
	cfg := &detectorConfig{
		fs:  NewRealFileSystem(),
		cmd: NewRealCommandRunner(),
		env: NewRealEnvironmentReader(),
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return &MultiPlatformDetector{
		detectors: []PlatformDetector{
			NewWindowsDetector(cfg.fs, cfg.cmd, cfg.env),
			NewMacOSDetector(cfg.fs, cfg.cmd, cfg.env),
			NewLinuxDetector(cfg.fs, cfg.cmd, cfg.env),
		},
	}
}

// DetectAll runs detection on all platforms and returns combined results.
func (m *MultiPlatformDetector) DetectAll(ctx context.Context) []DiscoveryResult {
	results := make([]DiscoveryResult, 0, len(m.detectors))

	for _, d := range m.detectors {
		result := DiscoveryResult{
			Platform: d.Platform(),
			Tools:    d.DetectTools(ctx),
		}

		certs, err := d.DiscoverCertificates(ctx)
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
		} else {
			result.Certificates = certs
		}

		results = append(results, result)
	}

	return results
}

// DetectPlatform runs detection for a specific platform.
func (m *MultiPlatformDetector) DetectPlatform(ctx context.Context, platform string) *DiscoveryResult {
	for _, d := range m.detectors {
		if d.Platform() == platform {
			result := &DiscoveryResult{
				Platform: d.Platform(),
				Tools:    d.DetectTools(ctx),
			}

			certs, err := d.DiscoverCertificates(ctx)
			if err != nil {
				result.Errors = append(result.Errors, err.Error())
			} else {
				result.Certificates = certs
			}

			return result
		}
	}

	return nil
}

// DiscoverCertificates discovers certificates for a specific platform.
func (m *MultiPlatformDetector) DiscoverCertificates(ctx context.Context, platform string) ([]types.DiscoveredCertificate, error) {
	for _, d := range m.detectors {
		if d.Platform() == platform {
			return d.DiscoverCertificates(ctx)
		}
	}
	return nil, nil
}
