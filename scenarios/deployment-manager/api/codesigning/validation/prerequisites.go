package validation

import (
	"context"
	"crypto/x509"
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
	fs   codesigning.FileSystem
	cmd  codesigning.CommandRunner
	env  codesigning.EnvironmentReader
	time codesigning.TimeProvider
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

	switch runtimeGOOS() {
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

// runtimeGOOS returns the current GOOS. Extracted for testability.
func runtimeGOOS() string {
	return runtime.GOOS
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

// parsePKCS12Certificate decodes a PKCS#12 certificate.
func (c *prerequisiteChecker) parsePKCS12Certificate(data []byte, password string) (*codesigning.CertificateInfo, error) {
	_, cert, err := pkcs12.Decode(data, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decode PKCS#12: %w", err)
	}
	return c.extractCertificateInfo(cert), nil
}

func (c *prerequisiteChecker) extractCertificateInfo(cert *x509.Certificate) *codesigning.CertificateInfo {
	now := c.time.Now()

	return &codesigning.CertificateInfo{
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

// getCertificateType returns the certificate type based on file extension.
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
