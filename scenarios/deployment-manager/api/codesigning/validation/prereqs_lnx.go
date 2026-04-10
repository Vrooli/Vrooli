package validation

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"deployment-manager/codesigning"
)

// checkLinuxPrerequisites validates Linux GPG signing tools and keys.
func (c *prerequisiteChecker) checkLinuxPrerequisites(ctx context.Context, config *codesigning.LinuxSigningConfig, result *codesigning.ValidationResult) {
	pv := codesigning.PlatformValidation{
		Configured: true,
		Errors:     []string{},
		Warnings:   []string{},
	}

	gpgResult := c.detectGPG(ctx)
	pv.ToolInstalled = gpgResult.Installed
	pv.ToolPath = gpgResult.Path
	pv.ToolVersion = gpgResult.Version

	if !gpgResult.Installed {
		result.AddError(codesigning.ValidationError{
			Code:        "LINUX_GPG_NOT_FOUND",
			Platform:    codesigning.PlatformLinux,
			Message:     "gpg not found",
			Remediation: gpgResult.Remediation,
		})
		pv.Errors = append(pv.Errors, "gpg not found")
	} else if config.GPGKeyID != "" {
		c.checkGPGKey(ctx, config.GPGKeyID, config.GPGHomedir, &pv, result)
	}

	if config.GPGPassphraseEnv != "" {
		if _, exists := c.env.LookupEnv(config.GPGPassphraseEnv); !exists {
			result.AddWarning(codesigning.ValidationWarning{
				Code:     "LINUX_GPG_PASSPHRASE_ENV_NOT_SET",
				Platform: codesigning.PlatformLinux,
				Message:  fmt.Sprintf("Environment variable %s is not set", config.GPGPassphraseEnv),
			})
		}
	}

	result.Platforms[codesigning.PlatformLinux] = pv
}

func (c *prerequisiteChecker) checkGPGKey(ctx context.Context, keyID, homedir string, pv *codesigning.PlatformValidation, result *codesigning.ValidationResult) {
	args := []string{"--list-secret-keys", keyID}
	if homedir != "" {
		args = append([]string{"--homedir", homedir}, args...)
	}

	_, _, err := c.cmd.Run(ctx, "gpg", args...)
	if err != nil {
		result.AddError(codesigning.ValidationError{
			Code:        "LINUX_KEY_NOT_FOUND",
			Platform:    codesigning.PlatformLinux,
			Field:       "gpg_key_id",
			Message:     "GPG key not found: " + keyID,
			Remediation: "Import the GPG key or verify the key ID is correct. List available keys with: gpg --list-secret-keys",
		})
		pv.Errors = append(pv.Errors, "GPG key not found")
	}
}

func (c *prerequisiteChecker) detectGPG(ctx context.Context) codesigning.ToolDetectionResult {
	result := codesigning.ToolDetectionResult{
		Platform: codesigning.PlatformLinux,
		Tool:     "gpg",
	}

	path, err := c.cmd.LookPath("gpg")
	if err == nil {
		result.Installed = true
		result.Path = path

		stdout, _, err := c.cmd.Run(ctx, "gpg", "--version")
		if err == nil {
			lines := strings.Split(string(stdout), "\n")
			if len(lines) > 0 {
				result.Version = strings.TrimSpace(lines[0])
			}
		}
		return result
	}

	result.Error = "gpg not found"
	result.Remediation = "Install GnuPG: sudo apt-get install gnupg (Debian/Ubuntu) or sudo yum install gnupg2 (RHEL/CentOS)"

	return result
}

// discoverGPGKeys lists GPG secret keys.
func (c *prerequisiteChecker) discoverGPGKeys(ctx context.Context) ([]codesigning.DiscoveredCertificate, error) {
	var certs []codesigning.DiscoveredCertificate

	stdout, stderr, err := c.cmd.Run(ctx, "gpg", "--list-secret-keys", "--keyid-format", "long")
	if err != nil {
		if strings.Contains(string(stderr), "no secret key") {
			return certs, nil
		}
		return nil, fmt.Errorf("failed to list GPG keys: %v (stderr: %s)", err, string(stderr))
	}

	output := string(stdout)
	lines := strings.Split(output, "\n")

	var currentKey *codesigning.DiscoveredCertificate
	keyIDRegex := regexp.MustCompile(`sec\s+\w+/([A-F0-9]+)\s+(\d{4}-\d{2}-\d{2})`)
	expiresRegex := regexp.MustCompile(`\[expires:\s*(\d{4}-\d{2}-\d{2})\]`)
	uidRegex := regexp.MustCompile(`uid\s+\[.+\]\s+(.+)`)
	fingerprintRegex := regexp.MustCompile(`^\s+([A-F0-9]{40})\s*$`)

	for _, line := range lines {
		if match := keyIDRegex.FindStringSubmatch(line); len(match) >= 3 {
			if currentKey != nil {
				certs = append(certs, *currentKey)
			}

			keyID := match[1]
			currentKey = &codesigning.DiscoveredCertificate{
				ID:         keyID,
				IsCodeSign: true,
				Type:       "GPG Secret Key",
				Platform:   codesigning.PlatformLinux,
				UsageHint:  fmt.Sprintf("Use --gpg-key %s", keyID),
			}

			if expMatch := expiresRegex.FindStringSubmatch(line); len(expMatch) > 1 {
				currentKey.ExpiresAt = expMatch[1]
				if t, err := parseDate(expMatch[1]); err == nil {
					now := c.time.Now()
					duration := t.Sub(now)
					currentKey.DaysToExpiry = int(duration.Hours() / 24)
					currentKey.IsExpired = now.After(t)
				}
			} else {
				currentKey.DaysToExpiry = -1
				currentKey.ExpiresAt = "never"
			}
			continue
		}

		if currentKey != nil {
			if match := fingerprintRegex.FindStringSubmatch(line); len(match) > 1 {
				currentKey.ID = match[1]
				currentKey.UsageHint = fmt.Sprintf("Use --gpg-key %s", match[1])
			}
		}

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

	if currentKey != nil {
		certs = append(certs, *currentKey)
	}

	return certs, nil
}
