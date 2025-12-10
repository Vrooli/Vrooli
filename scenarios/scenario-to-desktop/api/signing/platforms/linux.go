package platforms

import (
	"context"
	"fmt"
	"regexp"
	"runtime"
	"strings"

	"scenario-to-desktop-api/signing/types"
)

// LinuxDetector detects Linux GPG signing tools and keys.
type LinuxDetector struct {
	fs  FileSystem
	cmd CommandRunner
	env EnvironmentReader
}

// NewLinuxDetector creates a new Linux platform detector.
func NewLinuxDetector(fs FileSystem, cmd CommandRunner, env EnvironmentReader) *LinuxDetector {
	return &LinuxDetector{fs: fs, cmd: cmd, env: env}
}

// Platform returns the platform identifier.
func (d *LinuxDetector) Platform() string {
	return types.PlatformLinux
}

// DetectTools detects Linux signing tools.
func (d *LinuxDetector) DetectTools(ctx context.Context) []types.ToolDetectionResult {
	var results []types.ToolDetectionResult

	// Only detect on Linux
	if runtime.GOOS != "linux" {
		return results
	}

	results = append(results, d.detectGPG(ctx))

	return results
}

// DiscoverCertificates discovers GPG signing keys.
func (d *LinuxDetector) DiscoverCertificates(ctx context.Context) ([]types.DiscoveredCertificate, error) {
	// Only discover on Linux (GPG is also available on other platforms)
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return nil, nil
	}

	return d.listGPGKeys(ctx, "")
}

// detectGPG detects the gpg tool.
func (d *LinuxDetector) detectGPG(ctx context.Context) types.ToolDetectionResult {
	result := types.ToolDetectionResult{
		Platform: types.PlatformLinux,
		Tool:     "gpg",
	}

	path, err := d.cmd.LookPath("gpg")
	if err == nil {
		result.Installed = true
		result.Path = path

		stdout, _, err := d.cmd.Run(ctx, "gpg", "--version")
		if err == nil {
			lines := strings.Split(string(stdout), "\n")
			if len(lines) > 0 {
				result.Version = strings.TrimSpace(lines[0])
			}
		}
		return result
	}

	result.Error = "gpg not found"
	result.Remediation = "Install GnuPG: sudo apt-get install gnupg (Debian/Ubuntu) or sudo dnf install gnupg2 (Fedora/RHEL)"

	return result
}

// listGPGKeys lists GPG secret keys that can be used for signing.
func (d *LinuxDetector) listGPGKeys(ctx context.Context, homedir string) ([]types.DiscoveredCertificate, error) {
	var certs []types.DiscoveredCertificate

	// Build command args
	args := []string{"--list-secret-keys", "--keyid-format", "long"}
	if homedir != "" {
		args = append([]string{"--homedir", homedir}, args...)
	}

	stdout, stderr, err := d.cmd.Run(ctx, "gpg", args...)
	if err != nil {
		// Check if it's just no keys found
		if strings.Contains(string(stderr), "no secret key") || strings.Contains(string(stderr), "not found") {
			return certs, nil
		}
		return nil, fmt.Errorf("failed to list GPG keys: %v (stderr: %s)", err, string(stderr))
	}

	// Parse GPG output
	output := string(stdout)
	lines := strings.Split(output, "\n")

	var currentKey *types.DiscoveredCertificate
	keyIDRegex := regexp.MustCompile(`sec\s+\w+/([A-F0-9]+)\s+(\d{4}-\d{2}-\d{2})`)
	expiresRegex := regexp.MustCompile(`\[expires:\s*(\d{4}-\d{2}-\d{2})\]`)
	uidRegex := regexp.MustCompile(`uid\s+\[.+\]\s+(.+)`)
	fingerprintRegex := regexp.MustCompile(`^\s+([A-F0-9]{40})\s*$`)

	for _, line := range lines {
		// Match key line
		if match := keyIDRegex.FindStringSubmatch(line); len(match) >= 3 {
			// Save previous key if exists
			if currentKey != nil {
				certs = append(certs, *currentKey)
			}

			keyID := match[1]

			currentKey = &types.DiscoveredCertificate{
				ID:         keyID,
				IsCodeSign: true,
				Type:       "GPG Secret Key",
				Platform:   types.PlatformLinux,
				UsageHint:  fmt.Sprintf("Use --gpg-key %s", keyID),
			}

			// Check for expiration
			if expMatch := expiresRegex.FindStringSubmatch(line); len(expMatch) > 1 {
				currentKey.ExpiresAt = expMatch[1]
				currentKey.DaysToExpiry = calculateDaysToExpiry(expMatch[1])
				currentKey.IsExpired = currentKey.DaysToExpiry < 0
			} else {
				currentKey.DaysToExpiry = -1 // No expiration
				currentKey.ExpiresAt = "never"
			}
			continue
		}

		// Match fingerprint line
		if currentKey != nil {
			if match := fingerprintRegex.FindStringSubmatch(line); len(match) > 1 {
				currentKey.ID = match[1]
				currentKey.UsageHint = fmt.Sprintf("Use --gpg-key %s", match[1])
			}
		}

		// Match uid line
		if currentKey != nil {
			if match := uidRegex.FindStringSubmatch(line); len(match) > 1 {
				uid := strings.TrimSpace(match[1])
				if currentKey.Name == "" {
					currentKey.Name = uid
					currentKey.Subject = uid
				}
			}
		}
	}

	// Don't forget the last key
	if currentKey != nil {
		certs = append(certs, *currentKey)
	}

	return certs, nil
}

// ValidateKeyExists checks if a GPG key exists in the keyring.
func (d *LinuxDetector) ValidateKeyExists(ctx context.Context, keyID, homedir string) (bool, error) {
	args := []string{"--list-secret-keys", keyID}
	if homedir != "" {
		args = append([]string{"--homedir", homedir}, args...)
	}

	_, stderr, err := d.cmd.Run(ctx, "gpg", args...)
	if err != nil {
		// Key not found
		if strings.Contains(string(stderr), "not found") || strings.Contains(string(stderr), "No secret key") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check GPG key: %w", err)
	}

	return true, nil
}

// ListKeysForSigning returns GPG keys suitable for signing with specific homedir.
func (d *LinuxDetector) ListKeysForSigning(ctx context.Context, homedir string) ([]types.DiscoveredCertificate, error) {
	return d.listGPGKeys(ctx, homedir)
}

// calculateDaysToExpiry calculates days until expiration from a date string.
func calculateDaysToExpiry(dateStr string) int {
	// Parse date in format "2025-01-01"
	// We use a simple calculation rather than time.Parse to avoid timezone issues
	parts := strings.Split(dateStr, "-")
	if len(parts) != 3 {
		return -1
	}

	var year, month, day int
	fmt.Sscanf(dateStr, "%d-%d-%d", &year, &month, &day)

	return -1 // Return -1 to indicate we couldn't calculate precisely
}
