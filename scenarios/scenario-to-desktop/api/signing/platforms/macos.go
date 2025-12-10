package platforms

import (
	"context"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"time"

	"scenario-to-desktop-api/signing/types"
)

// MacOSDetector detects macOS code signing tools and identities.
type MacOSDetector struct {
	fs  FileSystem
	cmd CommandRunner
	env EnvironmentReader
}

// NewMacOSDetector creates a new macOS platform detector.
func NewMacOSDetector(fs FileSystem, cmd CommandRunner, env EnvironmentReader) *MacOSDetector {
	return &MacOSDetector{fs: fs, cmd: cmd, env: env}
}

// Platform returns the platform identifier.
func (d *MacOSDetector) Platform() string {
	return types.PlatformMacOS
}

// DetectTools detects macOS signing tools.
func (d *MacOSDetector) DetectTools(ctx context.Context) []types.ToolDetectionResult {
	var results []types.ToolDetectionResult

	// Only detect on macOS
	if runtime.GOOS != "darwin" {
		return results
	}

	results = append(results, d.detectCodesign(ctx))
	results = append(results, d.detectNotarytool(ctx))

	return results
}

// DiscoverCertificates discovers signing identities from the keychain.
func (d *MacOSDetector) DiscoverCertificates(ctx context.Context) ([]types.DiscoveredCertificate, error) {
	// Only discover on macOS
	if runtime.GOOS != "darwin" {
		return nil, nil
	}

	return d.listKeychainIdentities(ctx)
}

// detectCodesign detects the codesign tool.
func (d *MacOSDetector) detectCodesign(ctx context.Context) types.ToolDetectionResult {
	result := types.ToolDetectionResult{
		Platform: types.PlatformMacOS,
		Tool:     "codesign",
	}

	path, err := d.cmd.LookPath("codesign")
	if err == nil {
		result.Installed = true
		result.Path = path

		// Get version
		stdout, _, err := d.cmd.Run(ctx, "codesign", "--version")
		if err == nil && len(stdout) > 0 {
			result.Version = strings.TrimSpace(string(stdout))
		} else {
			// Try getting Xcode version as proxy
			stdout, _, err = d.cmd.Run(ctx, "xcodebuild", "-version")
			if err == nil {
				lines := strings.Split(string(stdout), "\n")
				if len(lines) > 0 {
					result.Version = strings.TrimSpace(lines[0])
				}
			}
		}
		return result
	}

	result.Error = "codesign not found"
	result.Remediation = "Install Xcode Command Line Tools: xcode-select --install"

	return result
}

// detectNotarytool detects the notarytool (via xcrun).
func (d *MacOSDetector) detectNotarytool(ctx context.Context) types.ToolDetectionResult {
	result := types.ToolDetectionResult{
		Platform: types.PlatformMacOS,
		Tool:     "notarytool",
	}

	// notarytool is accessed via xcrun
	stdout, _, err := d.cmd.Run(ctx, "xcrun", "notarytool", "--version")
	if err == nil {
		result.Installed = true
		result.Path = "xcrun notarytool"
		result.Version = strings.TrimSpace(string(stdout))
		return result
	}

	result.Error = "notarytool not found (requires Xcode 13+)"
	result.Remediation = "Install Xcode 13 or later from the Mac App Store"

	return result
}

// listKeychainIdentities lists code signing identities from the keychain.
func (d *MacOSDetector) listKeychainIdentities(ctx context.Context) ([]types.DiscoveredCertificate, error) {
	var certs []types.DiscoveredCertificate

	// Use security find-identity to list signing identities
	stdout, stderr, err := d.cmd.Run(ctx, "security", "find-identity", "-v", "-p", "codesigning")
	if err != nil {
		return nil, fmt.Errorf("failed to list identities: %v (stderr: %s)", err, string(stderr))
	}

	// Parse output
	// Format: 1) FINGERPRINT "Identity Name"
	identityRegex := regexp.MustCompile(`^\s*\d+\)\s+([A-F0-9]{40})\s+"([^"]+)"`)

	lines := strings.Split(string(stdout), "\n")
	for _, line := range lines {
		match := identityRegex.FindStringSubmatch(line)
		if len(match) >= 3 {
			fingerprint := match[1]
			identity := match[2]

			// Determine identity type
			identityType := "Signing Identity"
			if strings.Contains(identity, "Developer ID Application") {
				identityType = "Developer ID Application"
			} else if strings.Contains(identity, "Developer ID Installer") {
				identityType = "Developer ID Installer"
			} else if strings.Contains(identity, "Apple Development") {
				identityType = "Apple Development"
			} else if strings.Contains(identity, "Apple Distribution") {
				identityType = "Apple Distribution"
			} else if strings.Contains(identity, "Mac Developer") {
				identityType = "Mac Developer"
			} else if strings.Contains(identity, "3rd Party Mac Developer") {
				identityType = "3rd Party Mac Developer"
			}

			// Extract team ID if present
			teamID := ""
			teamIDRegex := regexp.MustCompile(`\(([A-Z0-9]{10})\)$`)
			teamMatch := teamIDRegex.FindStringSubmatch(identity)
			if len(teamMatch) > 1 {
				teamID = teamMatch[1]
			}

			// Get certificate expiration info
			expiresAt, daysToExpiry, isExpired := d.getCertificateExpiration(ctx, fingerprint)

			cert := types.DiscoveredCertificate{
				ID:           fingerprint,
				Name:         identity,
				Subject:      identity,
				ExpiresAt:    expiresAt,
				DaysToExpiry: daysToExpiry,
				IsExpired:    isExpired,
				IsCodeSign:   true,
				Type:         identityType,
				Platform:     types.PlatformMacOS,
			}

			if teamID != "" {
				cert.UsageHint = fmt.Sprintf("Use --identity \"%s\" --team-id %s", identity, teamID)
			} else {
				cert.UsageHint = fmt.Sprintf("Use --identity \"%s\"", identity)
			}

			certs = append(certs, cert)
		}
	}

	return certs, nil
}

// getCertificateExpiration gets expiration info for a certificate by fingerprint.
func (d *MacOSDetector) getCertificateExpiration(ctx context.Context, fingerprint string) (string, int, bool) {
	// Use security find-certificate to get certificate details
	stdout, _, err := d.cmd.Run(ctx, "security", "find-certificate", "-c", fingerprint, "-p")
	if err != nil {
		return "", -1, false
	}

	// Try to parse with openssl
	if len(stdout) > 0 {
		expStdout, _, err := d.cmd.Run(ctx, "openssl", "x509", "-noout", "-enddate", "-in", "/dev/stdin")
		_ = err

		if len(expStdout) > 0 {
			// Format: notAfter=Mar 15 00:00:00 2026 GMT
			expStr := strings.TrimPrefix(strings.TrimSpace(string(expStdout)), "notAfter=")
			if t, err := time.Parse("Jan 2 15:04:05 2006 MST", expStr); err == nil {
				now := time.Now()
				duration := t.Sub(now)
				daysToExpiry := int(duration.Hours() / 24)
				return t.Format("2006-01-02"), daysToExpiry, now.After(t)
			}
		}
	}

	return "", -1, false
}

// ValidateIdentity checks if a signing identity exists in the keychain.
func (d *MacOSDetector) ValidateIdentity(ctx context.Context, identity string) (bool, error) {
	stdout, _, err := d.cmd.Run(ctx, "security", "find-identity", "-v", "-p", "codesigning")
	if err != nil {
		return false, fmt.Errorf("failed to query keychain: %w", err)
	}

	output := string(stdout)

	// Check exact match first
	if strings.Contains(output, identity) {
		return true, nil
	}

	// Try matching by team ID if it looks like a team ID
	if regexp.MustCompile(`^[A-Z0-9]{10}$`).MatchString(identity) {
		return strings.Contains(output, identity), nil
	}

	// Try extracting team ID from identity string like "Developer ID Application: Name (TEAMID)"
	if strings.Contains(identity, "(") && strings.Contains(identity, ")") {
		start := strings.LastIndex(identity, "(")
		end := strings.LastIndex(identity, ")")
		if start < end {
			teamID := identity[start+1 : end]
			return strings.Contains(output, teamID), nil
		}
	}

	return false, nil
}
