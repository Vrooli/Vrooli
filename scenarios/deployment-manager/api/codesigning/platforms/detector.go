// Package platforms provides platform-specific signing tool detection and certificate discovery.
//
// This package abstracts the differences between Windows (Authenticode),
// macOS (Developer ID + Notarization), and Linux (GPG) signing workflows.
package platforms

import (
	"context"

	"deployment-manager/codesigning"
)

// PlatformDetector detects signing tools and certificates for a specific platform.
type PlatformDetector interface {
	// Platform returns the platform this detector handles.
	Platform() string

	// DetectTools returns information about available signing tools.
	DetectTools(ctx context.Context) []codesigning.ToolDetectionResult

	// DiscoverCertificates finds available signing certificates/identities.
	DiscoverCertificates(ctx context.Context) ([]DiscoveredCertificate, error)
}

// DiscoveredCertificate represents a signing certificate or identity found on the system.
type DiscoveredCertificate struct {
	// ID is the unique identifier (thumbprint, identity name, or key ID)
	ID string `json:"id"`

	// Name is the human-readable display name
	Name string `json:"name"`

	// Subject is the certificate subject (CN, O, etc.)
	Subject string `json:"subject,omitempty"`

	// Issuer is the certificate issuer
	Issuer string `json:"issuer,omitempty"`

	// ExpiresAt is the expiration date (empty for GPG keys)
	ExpiresAt string `json:"expires_at,omitempty"`

	// DaysToExpiry is days until expiration (-1 if N/A or already expired)
	DaysToExpiry int `json:"days_to_expiry"`

	// IsExpired indicates if the certificate is expired
	IsExpired bool `json:"is_expired"`

	// IsCodeSign indicates if this can be used for code signing
	IsCodeSign bool `json:"is_code_sign"`

	// Type describes the certificate type (e.g., "Developer ID", "EV Code Signing", "GPG")
	Type string `json:"type,omitempty"`

	// Platform is the platform this certificate is for
	Platform string `json:"platform"`

	// UsageHint provides guidance on how to use this certificate
	UsageHint string `json:"usage_hint,omitempty"`
}

// DiscoveryResult contains the result of platform discovery.
type DiscoveryResult struct {
	Platform     string                           `json:"platform"`
	Tools        []codesigning.ToolDetectionResult `json:"tools"`
	Certificates []DiscoveredCertificate          `json:"certificates"`
	Errors       []string                         `json:"errors,omitempty"`
}

// MultiPlatformDetector runs detection across multiple platforms.
type MultiPlatformDetector struct {
	detectors []PlatformDetector
}

// NewMultiPlatformDetector creates a detector that runs on all platforms.
func NewMultiPlatformDetector(opts ...DetectorOption) *MultiPlatformDetector {
	cfg := &detectorConfig{
		fs:  codesigning.NewRealFileSystem(),
		cmd: newRealCommandRunner(),
		env: codesigning.NewRealEnvironmentReader(),
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

// DetectorOption configures detector creation.
type DetectorOption func(*detectorConfig)

type detectorConfig struct {
	fs  codesigning.FileSystem
	cmd codesigning.CommandRunner
	env codesigning.EnvironmentReader
}

// WithFileSystem sets a custom file system for detectors.
func WithFileSystem(fs codesigning.FileSystem) DetectorOption {
	return func(c *detectorConfig) {
		c.fs = fs
	}
}

// WithCommandRunner sets a custom command runner for detectors.
func WithCommandRunner(cmd codesigning.CommandRunner) DetectorOption {
	return func(c *detectorConfig) {
		c.cmd = cmd
	}
}

// WithEnvironmentReader sets a custom environment reader for detectors.
func WithEnvironmentReader(env codesigning.EnvironmentReader) DetectorOption {
	return func(c *detectorConfig) {
		c.env = env
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
func (m *MultiPlatformDetector) DiscoverCertificates(ctx context.Context, platform string) ([]DiscoveredCertificate, error) {
	for _, d := range m.detectors {
		if d.Platform() == platform {
			return d.DiscoverCertificates(ctx)
		}
	}
	return nil, nil
}
