package validation

import (
	"bytes"
	"context"
	"crypto/x509"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"scenario-to-desktop-api/shared/env"
	"scenario-to-desktop-api/signing/types"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"software.sslmate.com/src/go-pkcs12"
)

// --- macOS Prerequisites ---

func (c *PrerequisiteChecker) checkMacOSPrerequisites(ctx context.Context, config *types.MacOSSigningConfig, result *types.ValidationResult) {
	pv := types.PlatformValidation{
		Configured: true,
		Errors:     []string{},
		Warnings:   []string{},
	}

	// Check codesign availability (only on macOS)
	if runtime.GOOS == "darwin" {
		codesignResult := c.detectCodesign(ctx)
		pv.ToolInstalled = codesignResult.Installed
		pv.ToolPath = codesignResult.Path
		pv.ToolVersion = codesignResult.Version

		if !codesignResult.Installed {
			addError(result, types.ValidationError{
				Code:        "MACOS_CODESIGN_NOT_FOUND",
				Platform:    types.PlatformMacOS,
				Message:     "codesign not found",
				Remediation: codesignResult.Remediation,
			})
			pv.Errors = append(pv.Errors, "codesign not found")
		}

		// Check notarytool if notarization is enabled
		if config.Notarize {
			notarytoolResult := c.detectNotarytool(ctx)
			if !notarytoolResult.Installed {
				addError(result, types.ValidationError{
					Code:        "MACOS_NOTARYTOOL_NOT_FOUND",
					Platform:    types.PlatformMacOS,
					Message:     "notarytool not found (required for notarization)",
					Remediation: notarytoolResult.Remediation,
				})
				pv.Errors = append(pv.Errors, "notarytool not found")
			}
		}

		// Check signing identity exists in keychain
		if config.Identity != "" {
			c.checkMacOSIdentity(ctx, config.Identity, &pv, result)
		}
	}

	// Check API key bindings if using API key auth. The generated hook needs
	// all three public identifiers plus the private file path; a partial
	// binding must not look like a usable notarization method.
	hasAPIKeyBinding := config.AppleAPIKeyID != "" || config.AppleAPIKeyFile != "" || config.AppleAPIIssuerID != ""
	hasCompleteAPIKey := config.AppleAPIKeyID != "" && config.AppleAPIKeyFile != "" && config.AppleAPIIssuerID != ""
	if hasAPIKeyBinding && !hasCompleteAPIKey {
		addError(result, types.ValidationError{
			Code:        "MACOS_API_KEY_BINDING_INCOMPLETE",
			Platform:    types.PlatformMacOS,
			Message:     "Apple API-key notarization requires a key ID, issuer ID, and private .p8 file path",
			Remediation: "Set apple_api_key_id, apple_api_issuer_id, and apple_api_key_file together, or use the Apple ID app-password method",
		})
		pv.Errors = append(pv.Errors, "Apple API key binding incomplete")
	}
	if config.AppleAPIKeyFile != "" {
		if !c.fs.Exists(config.AppleAPIKeyFile) {
			addError(result, types.ValidationError{
				Code:        "MACOS_API_KEY_NOT_FOUND",
				Platform:    types.PlatformMacOS,
				Field:       "apple_api_key_file",
				Message:     "API key file not found: " + config.AppleAPIKeyFile,
				Remediation: "Download your API key from App Store Connect and place it at the specified path",
			})
			pv.Errors = append(pv.Errors, "API key file not found")
		}
	}

	// Check app-password environment variables when API-key notarization is not
	// selected. Empty config fields intentionally use the generator defaults.
	if (config.Notarize || config.AppleIDEnv != "" || config.AppleIDPasswordEnv != "") && !hasCompleteAPIKey {
		appleIDEnv := config.AppleIDEnv
		if appleIDEnv == "" {
			appleIDEnv = types.DefaultAppleIDEnv
		}
		appleIDPasswordEnv := config.AppleIDPasswordEnv
		if appleIDPasswordEnv == "" {
			appleIDPasswordEnv = types.DefaultAppleIDPasswordEnv
		}
		if _, exists := c.env.LookupEnv(appleIDEnv); !exists {
			addWarning(result, types.ValidationWarning{
				Code:     "MACOS_APPLE_ID_ENV_NOT_SET",
				Platform: types.PlatformMacOS,
				Message:  fmt.Sprintf("Environment variable %s is not set", appleIDEnv),
			})
		}
		if _, exists := c.env.LookupEnv(appleIDPasswordEnv); !exists {
			addWarning(result, types.ValidationWarning{
				Code:     "MACOS_APPLE_PASSWORD_ENV_NOT_SET",
				Platform: types.PlatformMacOS,
				Message:  fmt.Sprintf("Environment variable %s is not set", appleIDPasswordEnv),
			})
		}
	}

	// Check entitlements file if specified
	if config.EntitlementsFile != "" {
		if !c.fs.Exists(config.EntitlementsFile) {
			addError(result, types.ValidationError{
				Code:        "MACOS_ENTITLEMENTS_NOT_FOUND",
				Platform:    types.PlatformMacOS,
				Field:       "entitlements_file",
				Message:     "Entitlements file not found: " + config.EntitlementsFile,
				Remediation: "Create the entitlements.plist file or update the path",
			})
			pv.Errors = append(pv.Errors, "Entitlements file not found")
		}
	}

	result.Platforms[types.PlatformMacOS] = pv
}

func (c *PrerequisiteChecker) checkMacOSIdentity(ctx context.Context, identity string, pv *types.PlatformValidation, result *types.ValidationResult) {
	// Use security find-identity to check if the identity exists
	stdout, _, err := c.cmd.Run(ctx, "security", "find-identity", "-v", "-p", "codesigning")
	if err != nil {
		addWarning(result, types.ValidationWarning{
			Code:     "MACOS_IDENTITY_CHECK_FAILED",
			Platform: types.PlatformMacOS,
			Message:  "Could not check keychain for signing identities: " + err.Error(),
		})
		return
	}

	output := string(stdout)
	if !strings.Contains(output, identity) {
		// Try matching by team ID
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
			addError(result, types.ValidationError{
				Code:        "MACOS_IDENTITY_NOT_FOUND",
				Platform:    types.PlatformMacOS,
				Field:       "identity",
				Message:     "Signing identity not found in keychain: " + identity,
				Remediation: "Import the Developer ID certificate into your login keychain, or run 'security find-identity -v -p codesigning' to list available identities",
			})
			pv.Errors = append(pv.Errors, "Signing identity not found")
		}
	}
}

func (c *PrerequisiteChecker) detectCodesign(ctx context.Context) types.ToolDetectionResult {
	result := types.ToolDetectionResult{
		Platform: types.PlatformMacOS,
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

func (c *PrerequisiteChecker) detectNotarytool(ctx context.Context) types.ToolDetectionResult {
	result := types.ToolDetectionResult{
		Platform: types.PlatformMacOS,
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

func (c *PrerequisiteChecker) checkLinuxPrerequisites(ctx context.Context, config *types.LinuxSigningConfig, result *types.ValidationResult) {
	pv := types.PlatformValidation{
		Configured: true,
		Errors:     []string{},
		Warnings:   []string{},
	}

	// Check GPG availability
	gpgResult := c.detectGPG(ctx)
	pv.ToolInstalled = gpgResult.Installed
	pv.ToolPath = gpgResult.Path
	pv.ToolVersion = gpgResult.Version

	switch {
	case !gpgResult.Installed:
		addError(result, types.ValidationError{
			Code:        "LINUX_GPG_NOT_FOUND",
			Platform:    types.PlatformLinux,
			Message:     "gpg not found",
			Remediation: gpgResult.Remediation,
		})
		pv.Errors = append(pv.Errors, "gpg not found")
	case config.ManagedKey != nil:
		c.checkManagedGPGKey(config.ManagedKey, &pv, result)
	case config.GPGKeyID != "":
		// Check if the key exists
		c.checkGPGKey(ctx, config.GPGKeyID, config.GPGHomedir, &pv, result)
	}

	// A managed key resolves its passphrase from the credential authority at
	// build time, so an absent ambient variable is not a warning.
	if config.ManagedKey == nil && config.GPGPassphraseEnv != "" {
		if _, exists := c.env.LookupEnv(config.GPGPassphraseEnv); !exists {
			addWarning(result, types.ValidationWarning{
				Code:     "LINUX_GPG_PASSPHRASE_ENV_NOT_SET",
				Platform: types.PlatformLinux,
				Message:  fmt.Sprintf("Environment variable %s is not set", config.GPGPassphraseEnv),
			})
		}
	}

	result.Platforms[types.PlatformLinux] = pv
}

// checkManagedGPGKey verifies the credential authority holds both halves of a
// managed signing key. It never reads the values.
func (c *PrerequisiteChecker) checkManagedGPGKey(managed *types.ManagedSigningKey, pv *types.PlatformValidation, result *types.ValidationResult) {
	identity, err := parseManagedIdentity(managed.LogicalID)
	if err != nil {
		addError(result, types.ValidationError{
			Code:        "LINUX_MANAGED_KEY_IDENTITY_INVALID",
			Platform:    types.PlatformLinux,
			Field:       "managed_key.logical_id",
			Message:     fmt.Sprintf("invalid managed key identity: %v", err),
			Remediation: "Use a namespaced identity such as vrooli/desktop-signing",
		})
		pv.Errors = append(pv.Errors, "invalid managed key identity")
		return
	}

	authority := c.authority
	if authority == nil {
		opened, openErr := openCredentialAuthority()
		if openErr != nil {
			addError(result, types.ValidationError{
				Code:        "LINUX_MANAGED_KEY_AUTHORITY_UNAVAILABLE",
				Platform:    types.PlatformLinux,
				Field:       "managed_key.logical_id",
				Message:     fmt.Sprintf("credential authority unavailable: %v", openErr),
				Remediation: "Run 'vrooli credentials doctor' to diagnose the host credential store",
			})
			pv.Errors = append(pv.Errors, "credential authority unavailable")
			return
		}
		authority = opened
	}

	if availabilityErr := authority.Availability(); availabilityErr != nil {
		addError(result, types.ValidationError{
			Code:        "LINUX_MANAGED_KEY_AUTHORITY_UNAVAILABLE",
			Platform:    types.PlatformLinux,
			Field:       "managed_key.logical_id",
			Message:     fmt.Sprintf("credential authority unavailable: %v", availabilityErr),
			Remediation: "Unlock the host credential store or run 'vrooli credentials doctor'",
		})
		pv.Errors = append(pv.Errors, "credential authority unavailable")
		return
	}

	for _, check := range []struct {
		field   string
		message string
		missing string
	}{
		{managed.ResolvedPrivateKeyField(), "managed signing private key is not configured", "private key not configured"},
		{managed.ResolvedPassphraseField(), "managed signing passphrase is not configured", "passphrase not configured"},
	} {
		if !authority.Status(identity, check.field).Configured {
			addError(result, types.ValidationError{
				Code:        "LINUX_MANAGED_KEY_NOT_CONFIGURED",
				Platform:    types.PlatformLinux,
				Field:       "managed_key.logical_id",
				Message:     fmt.Sprintf("%s: %s:%s", check.message, identity, check.field),
				Remediation: fmt.Sprintf("Provision the key with 'vrooli credentials provision --identity %s --field %s'", identity, check.field),
			})
			pv.Errors = append(pv.Errors, check.missing)
		}
	}
}

func parseManagedIdentity(logicalID string) (credentialauthority.Identity, error) {
	return credentialauthority.ParseIdentity(logicalID)
}

func (c *PrerequisiteChecker) checkGPGKey(ctx context.Context, keyID, homedir string, pv *types.PlatformValidation, result *types.ValidationResult) {
	args := []string{"--list-secret-keys", keyID}
	if homedir != "" {
		args = append([]string{"--homedir", homedir}, args...)
	}

	_, _, err := c.cmd.Run(ctx, "gpg", args...)
	if err != nil {
		addError(result, types.ValidationError{
			Code:        "LINUX_KEY_NOT_FOUND",
			Platform:    types.PlatformLinux,
			Field:       "gpg_key_id",
			Message:     "GPG key not found: " + keyID,
			Remediation: "Import the GPG key or verify the key ID is correct. List available keys with: gpg --list-secret-keys",
		})
		pv.Errors = append(pv.Errors, "GPG key not found")
	}
}

func (c *PrerequisiteChecker) detectGPG(ctx context.Context) types.ToolDetectionResult {
	result := types.ToolDetectionResult{
		Platform: types.PlatformLinux,
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
	result.Remediation = "Install GnuPG: sudo apt-get install gnupg (Debian/Ubuntu) or sudo dnf install gnupg2 (Fedora/RHEL)"

	return result
}

// --- Certificate Parsing ---

func (c *PrerequisiteChecker) parsePKCS12Certificate(data []byte, password string) (*types.CertificateInfo, error) {
	_, cert, err := pkcs12.Decode(data, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decode PKCS#12: %w", err)
	}

	return c.extractCertificateInfo(cert), nil
}

func (c *PrerequisiteChecker) extractCertificateInfo(cert *x509.Certificate) *types.CertificateInfo {
	now := c.time.Now()

	info := &types.CertificateInfo{
		Subject:      cert.Subject.String(),
		Issuer:       cert.Issuer.String(),
		SerialNumber: cert.SerialNumber.String(),
		NotBefore:    cert.NotBefore,
		NotAfter:     cert.NotAfter,
		IsExpired:    isCertificateExpired(cert.NotAfter, now),
		DaysToExpiry: calculateDaysToExpiry(cert.NotAfter, now),
		KeyUsage:     extractKeyUsage(cert),
		IsCodeSign:   isCodeSigningCert(cert),
	}

	return info
}

func (c *PrerequisiteChecker) checkCertificateExpiration(certInfo *types.CertificateInfo, platform string, pv *types.PlatformValidation, result *types.ValidationResult) {
	if certInfo.IsExpired {
		addError(result, types.ValidationError{
			Code:        platform[:3] + "_CERT_EXPIRED",
			Platform:    platform,
			Message:     "Certificate has expired",
			Remediation: "Renew your code signing certificate",
		})
		pv.Errors = append(pv.Errors, "Certificate expired")
		return
	}

	now := c.time.Now()

	if isCertificateExpiringCritical(certInfo.NotAfter, now) {
		addWarning(result, types.ValidationWarning{
			Code:     platform[:3] + "_CERT_EXPIRING_SOON",
			Platform: platform,
			Message:  fmt.Sprintf("Certificate expires in %d days (CRITICAL)", certInfo.DaysToExpiry),
		})
		pv.Warnings = append(pv.Warnings, fmt.Sprintf("Certificate expires in %d days", certInfo.DaysToExpiry))
	} else if isCertificateExpiringWarning(certInfo.NotAfter, now) {
		addWarning(result, types.ValidationWarning{
			Code:     platform[:3] + "_CERT_EXPIRING",
			Platform: platform,
			Message:  fmt.Sprintf("Certificate expires in %d days", certInfo.DaysToExpiry),
		})
		pv.Warnings = append(pv.Warnings, fmt.Sprintf("Certificate expires in %d days", certInfo.DaysToExpiry))
	}
}

// --- Helper Functions ---

func isCertificateExpired(notAfter, now time.Time) bool {
	return now.After(notAfter)
}

func isCertificateExpiringWarning(notAfter, now time.Time) bool {
	warningDate := now.AddDate(0, 0, CertExpiryWarningDays)
	return notAfter.Before(warningDate)
}

func isCertificateExpiringCritical(notAfter, now time.Time) bool {
	criticalDate := now.AddDate(0, 0, CertExpiryCriticalDays)
	return notAfter.Before(criticalDate)
}

func calculateDaysToExpiry(notAfter, now time.Time) int {
	duration := notAfter.Sub(now)
	return int(duration.Hours() / 24)
}

func extractKeyUsage(cert *x509.Certificate) []string {
	var usages []string

	if cert.KeyUsage&x509.KeyUsageDigitalSignature != 0 {
		usages = append(usages, "DigitalSignature")
	}
	if cert.KeyUsage&x509.KeyUsageContentCommitment != 0 {
		usages = append(usages, "ContentCommitment")
	}
	if cert.KeyUsage&x509.KeyUsageKeyEncipherment != 0 {
		usages = append(usages, "KeyEncipherment")
	}

	for _, eku := range cert.ExtKeyUsage {
		switch eku {
		case x509.ExtKeyUsageCodeSigning:
			usages = append(usages, "CodeSigning")
		case x509.ExtKeyUsageServerAuth:
			usages = append(usages, "ServerAuth")
		case x509.ExtKeyUsageClientAuth:
			usages = append(usages, "ClientAuth")
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
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "version") || strings.Contains(lower, "signtool") {
			versionRegex := regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
			if match := versionRegex.FindString(line); match != "" {
				return match
			}
			return strings.TrimSpace(line)
		}
	}
	return ""
}

// --- Real Implementations ---

type realFileSystem struct{}

func (f *realFileSystem) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (f *realFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

type realCommandRunner struct{}

func (r *realCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func (r *realCommandRunner) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

// realEnvironmentReader adapts env.OSReader to the local EnvironmentReader interface.
type realEnvironmentReader struct {
	reader env.Reader
}

func newRealEnvironmentReader() *realEnvironmentReader {
	return &realEnvironmentReader{reader: env.NewOSReader()}
}

func (e *realEnvironmentReader) GetEnv(key string) string {
	return e.reader.GetEnv(key)
}

func (e *realEnvironmentReader) LookupEnv(key string) (string, bool) {
	return e.reader.LookupEnv(key)
}

type realTimeProvider struct{}

func (t *realTimeProvider) Now() time.Time {
	return time.Now()
}
