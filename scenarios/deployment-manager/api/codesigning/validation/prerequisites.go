package validation

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"deployment-manager/codesigning"

	"software.sslmate.com/src/go-pkcs12"
)

// prerequisiteChecker implements PrerequisiteChecker using injected dependencies.
type prerequisiteChecker struct {
	fs      codesigning.FileSystem
	cmd     codesigning.CommandRunner
	env     codesigning.EnvironmentReader
	time    codesigning.TimeProvider
}

// PrerequisiteCheckerOption configures a prerequisite checker.
type PrerequisiteCheckerOption func(*prerequisiteChecker)

// WithFileSystem sets a custom file system.
func WithFileSystem(fs codesigning.FileSystem) PrerequisiteCheckerOption {
	return func(c *prerequisiteChecker) {
		c.fs = fs
	}
}

// WithCommandRunner sets a custom command runner.
func WithCommandRunner(cmd codesigning.CommandRunner) PrerequisiteCheckerOption {
	return func(c *prerequisiteChecker) {
		c.cmd = cmd
	}
}

// WithEnvironmentReader sets a custom environment reader.
func WithEnvironmentReader(env codesigning.EnvironmentReader) PrerequisiteCheckerOption {
	return func(c *prerequisiteChecker) {
		c.env = env
	}
}

// WithTimeProvider sets a custom time provider.
func WithTimeProvider(time codesigning.TimeProvider) PrerequisiteCheckerOption {
	return func(c *prerequisiteChecker) {
		c.time = time
	}
}

// NewPrerequisiteChecker creates a new prerequisite checker with the given options.
// By default, it uses real implementations for all dependencies.
func NewPrerequisiteChecker(opts ...PrerequisiteCheckerOption) PrerequisiteChecker {
	c := &prerequisiteChecker{
		fs:   codesigning.NewRealFileSystem(),
		cmd:  newRealCommandRunner(),
		env:  codesigning.NewRealEnvironmentReader(),
		time: codesigning.NewRealTimeProvider(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// CheckPrerequisites validates tools and certificates are available.
func (c *prerequisiteChecker) CheckPrerequisites(ctx context.Context, config *codesigning.SigningConfig) *codesigning.ValidationResult {
	result := codesigning.NewValidationResult()

	if config == nil || !config.Enabled {
		return result
	}

	// Check each configured platform
	if config.Windows != nil {
		c.checkWindowsPrerequisites(ctx, config.Windows, result)
	}
	if config.MacOS != nil {
		c.checkMacOSPrerequisites(ctx, config.MacOS, result)
	}
	if config.Linux != nil {
		c.checkLinuxPrerequisites(ctx, config.Linux, result)
	}

	return result
}

// CheckPlatformPrerequisites checks prerequisites for a specific platform.
func (c *prerequisiteChecker) CheckPlatformPrerequisites(ctx context.Context, config *codesigning.SigningConfig, platform string) *codesigning.ValidationResult {
	result := codesigning.NewValidationResult()

	if config == nil || !config.Enabled {
		return result
	}

	switch platform {
	case codesigning.PlatformWindows:
		if config.Windows != nil {
			c.checkWindowsPrerequisites(ctx, config.Windows, result)
		}
	case codesigning.PlatformMacOS:
		if config.MacOS != nil {
			c.checkMacOSPrerequisites(ctx, config.MacOS, result)
		}
	case codesigning.PlatformLinux:
		if config.Linux != nil {
			c.checkLinuxPrerequisites(ctx, config.Linux, result)
		}
	}

	return result
}

// DetectTools returns available signing tools on the current system.
func (c *prerequisiteChecker) DetectTools(ctx context.Context) ([]codesigning.ToolDetectionResult, error) {
	var results []codesigning.ToolDetectionResult

	// Always detect tools for the current platform
	currentPlatform := runtime.GOOS

	switch currentPlatform {
	case "windows":
		results = append(results, c.detectSigntool(ctx))
	case "darwin":
		results = append(results, c.detectCodesign(ctx))
		results = append(results, c.detectNotarytool(ctx))
	case "linux":
		results = append(results, c.detectGPG(ctx))
	}

	return results, nil
}

// DiscoverCertificates discovers available certificates/identities for a platform.
// Implements codesigning.CertificateDiscoverer interface.
func (c *prerequisiteChecker) DiscoverCertificates(ctx context.Context, platform string) ([]codesigning.DiscoveredCertificate, error) {
	switch platform {
	case codesigning.PlatformWindows:
		return c.discoverWindowsCertificates(ctx)
	case codesigning.PlatformMacOS:
		return c.discoverMacOSIdentities(ctx)
	case codesigning.PlatformLinux:
		return c.discoverGPGKeys(ctx)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
}

// discoverWindowsCertificates lists code signing certificates from Windows Certificate Store.
func (c *prerequisiteChecker) discoverWindowsCertificates(ctx context.Context) ([]codesigning.DiscoveredCertificate, error) {
	if runtime.GOOS != "windows" {
		return nil, nil
	}

	var certs []codesigning.DiscoveredCertificate

	// Use PowerShell to list code signing certificates
	psScript := `Get-ChildItem Cert:\CurrentUser\My -CodeSigningCert | ForEach-Object {
		$cert = $_
		Write-Output ("CERT:" + $cert.Thumbprint + "|" + $cert.Subject + "|" + $cert.Issuer + "|" + $cert.NotAfter.ToString("yyyy-MM-dd") + "|" + $cert.FriendlyName)
	}`

	stdout, stderr, err := c.cmd.Run(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	if err != nil {
		return nil, fmt.Errorf("failed to list certificates: %v (stderr: %s)", err, string(stderr))
	}

	// Parse output
	lines := strings.Split(string(stdout), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "CERT:") {
			continue
		}

		parts := strings.Split(strings.TrimPrefix(line, "CERT:"), "|")
		if len(parts) < 4 {
			continue
		}

		thumbprint := parts[0]
		subject := parts[1]
		issuer := parts[2]
		expiresStr := parts[3]
		friendlyName := ""
		if len(parts) > 4 {
			friendlyName = parts[4]
		}

		// Calculate days to expiry
		daysToExpiry := -1
		isExpired := false
		if t, err := parseDate(expiresStr); err == nil {
			now := c.time.Now()
			duration := t.Sub(now)
			daysToExpiry = int(duration.Hours() / 24)
			isExpired = now.After(t)
		}

		// Create display name
		name := friendlyName
		if name == "" {
			name = extractCNFromSubjectLine(subject)
		}

		certs = append(certs, codesigning.DiscoveredCertificate{
			ID:           thumbprint,
			Name:         name,
			Subject:      subject,
			Issuer:       issuer,
			ExpiresAt:    expiresStr,
			DaysToExpiry: daysToExpiry,
			IsExpired:    isExpired,
			IsCodeSign:   true,
			Type:         "Code Signing",
			Platform:     codesigning.PlatformWindows,
			UsageHint:    fmt.Sprintf("Use --thumbprint %s or --cert-source store", thumbprint),
		})
	}

	return certs, nil
}

// discoverMacOSIdentities lists code signing identities from the keychain.
func (c *prerequisiteChecker) discoverMacOSIdentities(ctx context.Context) ([]codesigning.DiscoveredCertificate, error) {
	if runtime.GOOS != "darwin" {
		return nil, nil
	}

	var certs []codesigning.DiscoveredCertificate

	// Use security find-identity to list signing identities
	stdout, stderr, err := c.cmd.Run(ctx, "security", "find-identity", "-v", "-p", "codesigning")
	if err != nil {
		return nil, fmt.Errorf("failed to list identities: %v (stderr: %s)", err, string(stderr))
	}

	// Parse output
	identityRegex := regexp.MustCompile(`^\s*\d+\)\s+([A-F0-9]{40})\s+"([^"]+)"`)
	teamIDRegex := regexp.MustCompile(`\(([A-Z0-9]{10})\)$`)

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
			}

			// Extract team ID if present
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
				ExpiresAt:    "", // Would need additional parsing
				DaysToExpiry: -1,
				IsExpired:    false,
				IsCodeSign:   true,
				Type:         identityType,
				Platform:     codesigning.PlatformMacOS,
				UsageHint:    usageHint,
			})
		}
	}

	return certs, nil
}

// discoverGPGKeys lists GPG secret keys.
func (c *prerequisiteChecker) discoverGPGKeys(ctx context.Context) ([]codesigning.DiscoveredCertificate, error) {
	var certs []codesigning.DiscoveredCertificate

	// List GPG secret keys
	stdout, stderr, err := c.cmd.Run(ctx, "gpg", "--list-secret-keys", "--keyid-format", "long")
	if err != nil {
		if strings.Contains(string(stderr), "no secret key") {
			return certs, nil
		}
		return nil, fmt.Errorf("failed to list GPG keys: %v (stderr: %s)", err, string(stderr))
	}

	// Parse GPG output
	output := string(stdout)
	lines := strings.Split(output, "\n")

	var currentKey *codesigning.DiscoveredCertificate
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

			currentKey = &codesigning.DiscoveredCertificate{
				ID:         keyID,
				IsCodeSign: true,
				Type:       "GPG Secret Key",
				Platform:   codesigning.PlatformLinux,
				UsageHint:  fmt.Sprintf("Use --gpg-key %s", keyID),
			}

			// Check for expiration
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

// parseDate parses a date string in YYYY-MM-DD format.
func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

// extractCNFromSubjectLine extracts the CN field from a certificate subject.
func extractCNFromSubjectLine(subject string) string {
	cnRegex := regexp.MustCompile(`CN=([^,]+)`)
	match := cnRegex.FindStringSubmatch(subject)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return subject
}

// --- Windows Prerequisites ---

func (c *prerequisiteChecker) checkWindowsPrerequisites(ctx context.Context, config *codesigning.WindowsSigningConfig, result *codesigning.ValidationResult) {
	pv := codesigning.PlatformValidation{
		Configured: true,
		Errors:     []string{},
		Warnings:   []string{},
	}

	// Check signtool availability
	toolResult := c.detectSigntool(ctx)
	pv.ToolInstalled = toolResult.Installed
	pv.ToolPath = toolResult.Path
	pv.ToolVersion = toolResult.Version

	if !toolResult.Installed {
		result.AddError(codesigning.ValidationError{
			Code:        "WIN_SIGNTOOL_NOT_FOUND",
			Platform:    codesigning.PlatformWindows,
			Message:     "signtool.exe not found",
			Remediation: toolResult.Remediation,
		})
		pv.Errors = append(pv.Errors, "signtool.exe not found")
	}

	// Check certificate based on source
	switch config.CertificateSource {
	case codesigning.CertSourceFile:
		c.checkWindowsFileCertificate(ctx, config, &pv, result)
	case codesigning.CertSourceStore:
		// Store-based certificates are checked at signing time
		result.AddWarning(codesigning.ValidationWarning{
			Code:     "WIN_STORE_CERT_RUNTIME_CHECK",
			Platform: codesigning.PlatformWindows,
			Message:  "Certificate store availability will be verified at signing time",
		})
	}

	// Check password environment variable
	if config.CertificatePasswordEnv != "" {
		if _, exists := c.env.LookupEnv(config.CertificatePasswordEnv); !exists {
			result.AddWarning(codesigning.ValidationWarning{
				Code:     "WIN_CERT_PASSWORD_ENV_NOT_SET",
				Platform: codesigning.PlatformWindows,
				Message:  fmt.Sprintf("Environment variable %s is not set", config.CertificatePasswordEnv),
			})
		}
	}

	result.Platforms[codesigning.PlatformWindows] = pv
}

func (c *prerequisiteChecker) checkWindowsFileCertificate(ctx context.Context, config *codesigning.WindowsSigningConfig, pv *codesigning.PlatformValidation, result *codesigning.ValidationResult) {
	if config.CertificateFile == "" {
		return
	}

	// Check file exists
	if !c.fs.Exists(config.CertificateFile) {
		result.AddError(codesigning.ValidationError{
			Code:        "WIN_CERT_FILE_NOT_FOUND",
			Platform:    codesigning.PlatformWindows,
			Field:       "certificate_file",
			Message:     "Certificate file not found: " + config.CertificateFile,
			Remediation: "Verify the certificate file path is correct and the file exists",
		})
		pv.Errors = append(pv.Errors, "Certificate file not found")
		return
	}

	// Try to parse the certificate
	certData, err := c.fs.ReadFile(config.CertificateFile)
	if err != nil {
		result.AddError(codesigning.ValidationError{
			Code:        "WIN_CERT_FILE_UNREADABLE",
			Platform:    codesigning.PlatformWindows,
			Field:       "certificate_file",
			Message:     "Cannot read certificate file: " + err.Error(),
			Remediation: "Verify file permissions and that the file is not corrupted",
		})
		pv.Errors = append(pv.Errors, "Cannot read certificate file")
		return
	}

	// Get password from environment if available
	password := ""
	if config.CertificatePasswordEnv != "" {
		password = c.env.GetEnv(config.CertificatePasswordEnv)
	}

	// Parse the PKCS#12 certificate
	certInfo, err := c.parsePKCS12Certificate(certData, password)
	if err != nil {
		// If parsing fails due to password, it's a warning not error (might work at build time)
		if strings.Contains(err.Error(), "password") {
			result.AddWarning(codesigning.ValidationWarning{
				Code:     "WIN_CERT_PASSWORD_NEEDED",
				Platform: codesigning.PlatformWindows,
				Message:  "Certificate parsing failed - may need correct password at build time",
			})
		} else {
			result.AddError(codesigning.ValidationError{
				Code:        "WIN_CERT_INVALID",
				Platform:    codesigning.PlatformWindows,
				Field:       "certificate_file",
				Message:     "Invalid certificate file: " + err.Error(),
				Remediation: "Ensure the file is a valid PKCS#12 (.pfx/.p12) certificate",
			})
			pv.Errors = append(pv.Errors, "Invalid certificate file")
		}
		return
	}

	pv.Certificate = certInfo

	// Check if certificate is for code signing
	if !certInfo.IsCodeSign {
		result.AddError(codesigning.ValidationError{
			Code:        "WIN_CERT_NOT_CODE_SIGN",
			Platform:    codesigning.PlatformWindows,
			Field:       "certificate_file",
			Message:     "Certificate is not valid for code signing",
			Remediation: "Use a certificate with the Code Signing extended key usage",
		})
		pv.Errors = append(pv.Errors, "Certificate not valid for code signing")
	}

	// Check certificate expiration
	c.checkCertificateExpiration(certInfo, codesigning.PlatformWindows, pv, result)
}

func (c *prerequisiteChecker) detectSigntool(ctx context.Context) codesigning.ToolDetectionResult {
	result := codesigning.ToolDetectionResult{
		Platform: codesigning.PlatformWindows,
		Tool:     "signtool.exe",
	}

	// Try to find signtool in PATH
	path, err := c.cmd.LookPath("signtool.exe")
	if err == nil {
		result.Installed = true
		result.Path = path

		// Try to get version
		stdout, _, err := c.cmd.Run(ctx, "signtool.exe")
		if err == nil {
			// Extract version from output
			if ver := extractSigntoolVersion(string(stdout)); ver != "" {
				result.Version = ver
			}
		}
		return result
	}

	// Try common Windows SDK paths
	sdkPaths := []string{
		`C:\Program Files (x86)\Windows Kits\10\bin\x64\signtool.exe`,
		`C:\Program Files (x86)\Windows Kits\10\bin\10.0.22000.0\x64\signtool.exe`,
		`C:\Program Files (x86)\Windows Kits\10\bin\10.0.19041.0\x64\signtool.exe`,
		`C:\Program Files (x86)\Windows Kits\8.1\bin\x64\signtool.exe`,
	}

	for _, sdkPath := range sdkPaths {
		if c.fs.Exists(sdkPath) {
			result.Installed = true
			result.Path = sdkPath
			return result
		}
	}

	result.Error = "signtool.exe not found in PATH or Windows SDK"
	result.Remediation = "Install Windows SDK or add signtool.exe to PATH. Download from: https://developer.microsoft.com/en-us/windows/downloads/windows-sdk/"

	return result
}

// --- macOS Prerequisites ---

func (c *prerequisiteChecker) checkMacOSPrerequisites(ctx context.Context, config *codesigning.MacOSSigningConfig, result *codesigning.ValidationResult) {
	pv := codesigning.PlatformValidation{
		Configured: true,
		Errors:     []string{},
		Warnings:   []string{},
	}

	// Check codesign availability
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

	// Check notarytool if notarization is enabled
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

	// Check signing identity exists in keychain
	if config.Identity != "" && runtime.GOOS == "darwin" {
		c.checkMacOSIdentity(ctx, config.Identity, &pv, result)
	}

	// Check API key file if using API key auth
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

	// Check environment variables
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

	// Check entitlements file if specified
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
	// Use security find-identity to check if the identity exists
	stdout, _, err := c.cmd.Run(ctx, "security", "find-identity", "-v", "-p", "codesigning")
	if err != nil {
		result.AddWarning(codesigning.ValidationWarning{
			Code:     "MACOS_IDENTITY_CHECK_FAILED",
			Platform: codesigning.PlatformMacOS,
			Message:  "Could not check keychain for signing identities: " + err.Error(),
		})
		return
	}

	// Check if the identity is in the output
	output := string(stdout)
	if !strings.Contains(output, identity) {
		// Try matching by team ID if full identity not found
		teamIDMatch := false
		if len(identity) >= 10 {
			// Extract team ID from identity or use as-is if it looks like a team ID
			if regexp.MustCompile(`^[A-Z0-9]{10}$`).MatchString(identity) {
				teamIDMatch = strings.Contains(output, identity)
			} else if strings.Contains(identity, "(") && strings.Contains(identity, ")") {
				// Extract team ID from "Developer ID Application: Name (TEAMID)"
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

		// Get version from codesign --version or xcodebuild
		stdout, _, err := c.cmd.Run(ctx, "codesign", "--version")
		if err == nil {
			result.Version = strings.TrimSpace(string(stdout))
		}
		return result
	}

	// codesign should always be available on macOS with Xcode Command Line Tools
	result.Error = "codesign not found"
	result.Remediation = "Install Xcode Command Line Tools: xcode-select --install"

	return result
}

func (c *prerequisiteChecker) detectNotarytool(ctx context.Context) codesigning.ToolDetectionResult {
	result := codesigning.ToolDetectionResult{
		Platform: codesigning.PlatformMacOS,
		Tool:     "notarytool",
	}

	// notarytool is accessed via xcrun
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

// --- Linux Prerequisites ---

func (c *prerequisiteChecker) checkLinuxPrerequisites(ctx context.Context, config *codesigning.LinuxSigningConfig, result *codesigning.ValidationResult) {
	pv := codesigning.PlatformValidation{
		Configured: true,
		Errors:     []string{},
		Warnings:   []string{},
	}

	// Check GPG availability
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
		// Check if the key exists
		c.checkGPGKey(ctx, config.GPGKeyID, config.GPGHomedir, &pv, result)
	}

	// Check passphrase environment variable
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

	_, stderr, err := c.cmd.Run(ctx, "gpg", args...)
	if err != nil {
		errMsg := strings.TrimSpace(string(stderr))
		if errMsg == "" {
			errMsg = err.Error()
		}

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
			// Extract version from first line
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

// --- Certificate Parsing ---

func (c *prerequisiteChecker) parsePKCS12Certificate(data []byte, password string) (*codesigning.CertificateInfo, error) {
	// Decode the PKCS#12 data
	_, cert, err := pkcs12.Decode(data, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decode PKCS#12: %w", err)
	}

	return c.extractCertificateInfo(cert), nil
}

func (c *prerequisiteChecker) parsePEMCertificate(data []byte) (*codesigning.CertificateInfo, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return c.extractCertificateInfo(cert), nil
}

func (c *prerequisiteChecker) extractCertificateInfo(cert *x509.Certificate) *codesigning.CertificateInfo {
	now := c.time.Now()

	info := &codesigning.CertificateInfo{
		Subject:      cert.Subject.String(),
		Issuer:       cert.Issuer.String(),
		SerialNumber: cert.SerialNumber.String(),
		NotBefore:    cert.NotBefore,
		NotAfter:     cert.NotAfter,
		IsExpired:    codesigning.IsCertificateExpired(cert.NotAfter, now),
		DaysToExpiry: codesigning.CalculateDaysToExpiry(cert.NotAfter, now),
		KeyUsage:     extractKeyUsage(cert),
		IsCodeSign:   isCodeSigningCert(cert),
	}

	return info
}

func (c *prerequisiteChecker) checkCertificateExpiration(certInfo *codesigning.CertificateInfo, platform string, pv *codesigning.PlatformValidation, result *codesigning.ValidationResult) {
	if certInfo.IsExpired {
		result.AddError(codesigning.ValidationError{
			Code:        platform[:3] + "_CERT_EXPIRED",
			Platform:    platform,
			Message:     "Certificate has expired",
			Remediation: "Renew your code signing certificate",
		})
		pv.Errors = append(pv.Errors, "Certificate expired")
		return
	}

	now := c.time.Now()

	if codesigning.IsCertificateExpiringCritical(certInfo.NotAfter, now) {
		result.AddWarning(codesigning.ValidationWarning{
			Code:     platform[:3] + "_CERT_EXPIRING_SOON",
			Platform: platform,
			Message:  fmt.Sprintf("Certificate expires in %d days (CRITICAL)", certInfo.DaysToExpiry),
		})
		pv.Warnings = append(pv.Warnings, fmt.Sprintf("Certificate expires in %d days", certInfo.DaysToExpiry))
	} else if codesigning.IsCertificateExpiringWarning(certInfo.NotAfter, now) {
		result.AddWarning(codesigning.ValidationWarning{
			Code:     platform[:3] + "_CERT_EXPIRING",
			Platform: platform,
			Message:  fmt.Sprintf("Certificate expires in %d days", certInfo.DaysToExpiry),
		})
		pv.Warnings = append(pv.Warnings, fmt.Sprintf("Certificate expires in %d days", certInfo.DaysToExpiry))
	}
}

// --- Helper Functions ---

func extractKeyUsage(cert *x509.Certificate) []string {
	var usages []string

	// Key usage
	if cert.KeyUsage&x509.KeyUsageDigitalSignature != 0 {
		usages = append(usages, "DigitalSignature")
	}
	if cert.KeyUsage&x509.KeyUsageContentCommitment != 0 {
		usages = append(usages, "ContentCommitment")
	}
	if cert.KeyUsage&x509.KeyUsageKeyEncipherment != 0 {
		usages = append(usages, "KeyEncipherment")
	}
	if cert.KeyUsage&x509.KeyUsageDataEncipherment != 0 {
		usages = append(usages, "DataEncipherment")
	}
	if cert.KeyUsage&x509.KeyUsageKeyAgreement != 0 {
		usages = append(usages, "KeyAgreement")
	}
	if cert.KeyUsage&x509.KeyUsageCertSign != 0 {
		usages = append(usages, "CertSign")
	}
	if cert.KeyUsage&x509.KeyUsageCRLSign != 0 {
		usages = append(usages, "CRLSign")
	}

	// Extended key usage
	for _, eku := range cert.ExtKeyUsage {
		switch eku {
		case x509.ExtKeyUsageCodeSigning:
			usages = append(usages, "CodeSigning")
		case x509.ExtKeyUsageServerAuth:
			usages = append(usages, "ServerAuth")
		case x509.ExtKeyUsageClientAuth:
			usages = append(usages, "ClientAuth")
		case x509.ExtKeyUsageEmailProtection:
			usages = append(usages, "EmailProtection")
		case x509.ExtKeyUsageTimeStamping:
			usages = append(usages, "TimeStamping")
		}
	}

	return usages
}

func isCodeSigningCert(cert *x509.Certificate) bool {
	for _, eku := range cert.ExtKeyUsage {
		if eku == x509.ExtKeyUsageCodeSigning {
			return true
		}
	}
	return false
}

func extractSigntoolVersion(output string) string {
	// signtool.exe output typically includes version info
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "version") {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

// realCommandRunner implements CommandRunner using os/exec.
type realCommandRunner struct{}

func newRealCommandRunner() codesigning.CommandRunner {
	return &realCommandRunner{}
}

func (r *realCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	return runCommand(ctx, name, args...)
}

func (r *realCommandRunner) LookPath(name string) (string, error) {
	return lookupPath(name)
}

// Check file extension for certificate type
func getCertificateType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".pfx", ".p12":
		return "pkcs12"
	case ".pem", ".crt", ".cer":
		return "pem"
	default:
		return "unknown"
	}
}
