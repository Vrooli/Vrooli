package validation

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"deployment-manager/codesigning"
)

// checkMacOSPrerequisites validates macOS signing tools and identities.
func (c *prerequisiteChecker) checkMacOSPrerequisites(ctx context.Context, config *codesigning.MacOSSigningConfig, result *codesigning.ValidationResult) {
	pv := codesigning.PlatformValidation{
		Configured: true,
		Errors:     []string{},
		Warnings:   []string{},
	}

	codesignResult := c.detectCodesign(ctx)
	pv.ToolInstalled = codesignResult.Installed
	pv.ToolPath = codesignResult.Path
	pv.ToolVersion = codesignResult.Version

	if !codesignResult.Installed {
		result.AddError(codesigning.ValidationError{
			Code:        "MACOS_CODESIGN_NOT_FOUND",
			Platform:    codesigning.PlatformMacOS,
			Message:     "codesign not found",
			Remediation: codesignResult.Remediation,
		})
		pv.Errors = append(pv.Errors, "codesign not found")
	}

	if config.Notarize {
		notarytoolResult := c.detectNotarytool(ctx)
		if !notarytoolResult.Installed {
			result.AddError(codesigning.ValidationError{
				Code:        "MACOS_NOTARYTOOL_NOT_FOUND",
				Platform:    codesigning.PlatformMacOS,
				Message:     "notarytool not found (required for notarization)",
				Remediation: notarytoolResult.Remediation,
			})
			pv.Errors = append(pv.Errors, "notarytool not found")
		}
	}

	if config.Identity != "" && runtimeGOOS() == "darwin" {
		c.checkMacOSIdentity(ctx, config.Identity, &pv, result)
	}

	if config.AppleAPIKeyFile != "" {
		if !c.fs.Exists(config.AppleAPIKeyFile) {
			result.AddError(codesigning.ValidationError{
				Code:        "MACOS_API_KEY_NOT_FOUND",
				Platform:    codesigning.PlatformMacOS,
				Field:       "apple_api_key_file",
				Message:     "API key file not found: " + config.AppleAPIKeyFile,
				Remediation: "Download your API key from App Store Connect and place it at the specified path",
			})
			pv.Errors = append(pv.Errors, "API key file not found")
		}
	}

	if config.AppleIDEnv != "" {
		if _, exists := c.env.LookupEnv(config.AppleIDEnv); !exists {
			result.AddWarning(codesigning.ValidationWarning{
				Code:     "MACOS_APPLE_ID_ENV_NOT_SET",
				Platform: codesigning.PlatformMacOS,
				Message:  fmt.Sprintf("Environment variable %s is not set", config.AppleIDEnv),
			})
		}
	}
	if config.AppleIDPasswordEnv != "" {
		if _, exists := c.env.LookupEnv(config.AppleIDPasswordEnv); !exists {
			result.AddWarning(codesigning.ValidationWarning{
				Code:     "MACOS_APPLE_PASSWORD_ENV_NOT_SET",
				Platform: codesigning.PlatformMacOS,
				Message:  fmt.Sprintf("Environment variable %s is not set", config.AppleIDPasswordEnv),
			})
		}
	}

	if config.EntitlementsFile != "" {
		if !c.fs.Exists(config.EntitlementsFile) {
			result.AddError(codesigning.ValidationError{
				Code:        "MACOS_ENTITLEMENTS_NOT_FOUND",
				Platform:    codesigning.PlatformMacOS,
				Field:       "entitlements_file",
				Message:     "Entitlements file not found: " + config.EntitlementsFile,
				Remediation: "Create the entitlements.plist file or update the path",
			})
			pv.Errors = append(pv.Errors, "Entitlements file not found")
		}
	}

	result.Platforms[codesigning.PlatformMacOS] = pv
}

func (c *prerequisiteChecker) checkMacOSIdentity(ctx context.Context, identity string, pv *codesigning.PlatformValidation, result *codesigning.ValidationResult) {
	stdout, _, err := c.cmd.Run(ctx, "security", "find-identity", "-v", "-p", "codesigning")
	if err != nil {
		result.AddWarning(codesigning.ValidationWarning{
			Code:     "MACOS_IDENTITY_CHECK_FAILED",
			Platform: codesigning.PlatformMacOS,
			Message:  "Could not check keychain for signing identities: " + err.Error(),
		})
		return
	}

	output := string(stdout)
	if !strings.Contains(output, identity) {
		teamIDMatch := false
		if len(identity) >= 10 {
			if regexp.MustCompile(`^[A-Z0-9]{10}$`).MatchString(identity) {
				teamIDMatch = strings.Contains(output, identity)
			} else if strings.Contains(identity, "(") && strings.Contains(identity, ")") {
				start := strings.LastIndex(identity, "(")
				end := strings.LastIndex(identity, ")")
				if start < end {
					teamID := identity[start+1 : end]
					teamIDMatch = strings.Contains(output, teamID)
				}
			}
		}

		if !teamIDMatch {
			result.AddError(codesigning.ValidationError{
				Code:        "MACOS_IDENTITY_NOT_FOUND",
				Platform:    codesigning.PlatformMacOS,
				Field:       "identity",
				Message:     "Signing identity not found in keychain: " + identity,
				Remediation: "Import the Developer ID certificate into your login keychain, or run 'security find-identity -v -p codesigning' to list available identities",
			})
			pv.Errors = append(pv.Errors, "Signing identity not found")
		}
	}
}

func (c *prerequisiteChecker) detectCodesign(ctx context.Context) codesigning.ToolDetectionResult {
	result := codesigning.ToolDetectionResult{
		Platform: codesigning.PlatformMacOS,
		Tool:     "codesign",
	}

	path, err := c.cmd.LookPath("codesign")
	if err == nil {
		result.Installed = true
		result.Path = path

		stdout, _, err := c.cmd.Run(ctx, "codesign", "--version")
		if err == nil {
			result.Version = strings.TrimSpace(string(stdout))
		}
		return result
	}

	result.Error = "codesign not found"
	result.Remediation = "Install Xcode Command Line Tools: xcode-select --install"

	return result
}

func (c *prerequisiteChecker) detectNotarytool(ctx context.Context) codesigning.ToolDetectionResult {
	result := codesigning.ToolDetectionResult{
		Platform: codesigning.PlatformMacOS,
		Tool:     "notarytool",
	}

	stdout, _, err := c.cmd.Run(ctx, "xcrun", "notarytool", "--version")
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

// discoverMacOSIdentities lists code signing identities from the keychain.
func (c *prerequisiteChecker) discoverMacOSIdentities(ctx context.Context) ([]codesigning.DiscoveredCertificate, error) {
	if runtimeGOOS() != "darwin" {
		return nil, nil
	}

	var certs []codesigning.DiscoveredCertificate

	stdout, stderr, err := c.cmd.Run(ctx, "security", "find-identity", "-v", "-p", "codesigning")
	if err != nil {
		return nil, fmt.Errorf("failed to list identities: %v (stderr: %s)", err, string(stderr))
	}

	identityRegex := regexp.MustCompile(`^\s*\d+\)\s+([A-F0-9]{40})\s+"([^"]+)"`)
	teamIDRegex := regexp.MustCompile(`\(([A-Z0-9]{10})\)$`)

	lines := strings.Split(string(stdout), "\n")
	for _, line := range lines {
		match := identityRegex.FindStringSubmatch(line)
		if len(match) < 3 {
			continue
		}

		fingerprint := match[1]
		identity := match[2]

		identityType := "Signing Identity"
		if strings.Contains(identity, "Developer ID Application") {
			identityType = "Developer ID Application"
		} else if strings.Contains(identity, "Developer ID Installer") {
			identityType = "Developer ID Installer"
		} else if strings.Contains(identity, "Apple Development") {
			identityType = "Apple Development"
		} else if strings.Contains(identity, "Apple Distribution") {
			identityType = "Apple Distribution"
		}

		teamID := ""
		teamMatch := teamIDRegex.FindStringSubmatch(identity)
		if len(teamMatch) > 1 {
			teamID = teamMatch[1]
		}

		usageHint := fmt.Sprintf("Use --identity \"%s\"", identity)
		if teamID != "" {
			usageHint = fmt.Sprintf("Use --identity \"%s\" --team-id %s", identity, teamID)
		}

		certs = append(certs, codesigning.DiscoveredCertificate{
			ID:           fingerprint,
			Name:         identity,
			Subject:      identity,
			ExpiresAt:    "",
			DaysToExpiry: -1,
			IsExpired:    false,
			IsCodeSign:   true,
			Type:         identityType,
			Platform:     codesigning.PlatformMacOS,
			UsageHint:    usageHint,
		})
	}

	return certs, nil
}
