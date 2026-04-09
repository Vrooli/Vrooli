package validation

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"scenario-to-desktop-api/signing/types"
)

// FileSystem abstracts file operations for testing.
type FileSystem interface {
	Exists(path string) bool
	ReadFile(path string) ([]byte, error)
}

// CommandRunner abstracts command execution for testing.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error)
	LookPath(name string) (string, error)
}

// EnvironmentReader abstracts environment variable access.
type EnvironmentReader interface {
	GetEnv(key string) string
	LookupEnv(key string) (string, bool)
}

// TimeProvider abstracts time for testing.
type TimeProvider interface {
	Now() time.Time
}

// Certificate expiration thresholds (in days)
const (
	CertExpiryWarningDays  = 60
	CertExpiryCriticalDays = 14
)

// PrerequisiteChecker implements signing.PrerequisiteChecker.
type PrerequisiteChecker struct {
	fs   FileSystem
	cmd  CommandRunner
	env  EnvironmentReader
	time TimeProvider
}

// PrerequisiteCheckerOption configures a prerequisite checker.
type PrerequisiteCheckerOption func(*PrerequisiteChecker)

// WithFileSystem sets a custom file system.
func WithFileSystem(fs FileSystem) PrerequisiteCheckerOption {
	return func(c *PrerequisiteChecker) {
		c.fs = fs
	}
}

// WithCommandRunner sets a custom command runner.
func WithCommandRunner(cmd CommandRunner) PrerequisiteCheckerOption {
	return func(c *PrerequisiteChecker) {
		c.cmd = cmd
	}
}

// WithEnvironmentReader sets a custom environment reader.
func WithEnvironmentReader(env EnvironmentReader) PrerequisiteCheckerOption {
	return func(c *PrerequisiteChecker) {
		c.env = env
	}
}

// WithTimeProvider sets a custom time provider.
func WithTimeProvider(tp TimeProvider) PrerequisiteCheckerOption {
	return func(c *PrerequisiteChecker) {
		c.time = tp
	}
}

// NewPrerequisiteChecker creates a new prerequisite checker.
func NewPrerequisiteChecker(opts ...PrerequisiteCheckerOption) *PrerequisiteChecker {
	c := &PrerequisiteChecker{
		fs:   &realFileSystem{},
		cmd:  &realCommandRunner{},
		env:  newRealEnvironmentReader(),
		time: &realTimeProvider{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// CheckPrerequisites validates tools and certificates are available.
func (c *PrerequisiteChecker) CheckPrerequisites(ctx context.Context, config *types.SigningConfig) *types.ValidationResult {
	result := NewValidationResult()

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
func (c *PrerequisiteChecker) CheckPlatformPrerequisites(ctx context.Context, config *types.SigningConfig, platform string) *types.ValidationResult {
	result := NewValidationResult()

	if config == nil || !config.Enabled {
		return result
	}

	switch platform {
	case types.PlatformWindows:
		if config.Windows != nil {
			c.checkWindowsPrerequisites(ctx, config.Windows, result)
		}
	case types.PlatformMacOS:
		if config.MacOS != nil {
			c.checkMacOSPrerequisites(ctx, config.MacOS, result)
		}
	case types.PlatformLinux:
		if config.Linux != nil {
			c.checkLinuxPrerequisites(ctx, config.Linux, result)
		}
	}

	return result
}

// DetectTools returns available signing tools on the current system.
func (c *PrerequisiteChecker) DetectTools(ctx context.Context) ([]types.ToolDetectionResult, error) {
	var results []types.ToolDetectionResult

	// Detect tools based on current platform
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

// --- Windows Prerequisites ---

func (c *PrerequisiteChecker) checkWindowsPrerequisites(ctx context.Context, config *types.WindowsSigningConfig, result *types.ValidationResult) {
	pv := types.PlatformValidation{
		Configured: true,
		Errors:     []string{},
		Warnings:   []string{},
	}

	// Check signtool availability (only on Windows)
	if runtime.GOOS == "windows" {
		toolResult := c.detectSigntool(ctx)
		pv.ToolInstalled = toolResult.Installed
		pv.ToolPath = toolResult.Path
		pv.ToolVersion = toolResult.Version

		if !toolResult.Installed {
			addError(result, types.ValidationError{
				Code:        "WIN_SIGNTOOL_NOT_FOUND",
				Platform:    types.PlatformWindows,
				Message:     "signtool.exe not found",
				Remediation: toolResult.Remediation,
			})
			pv.Errors = append(pv.Errors, "signtool.exe not found")
		}
	}

	// Check certificate based on source
	if config.CertificateSource == types.CertSourceFile && config.CertificateFile != "" {
		c.checkWindowsFileCertificate(ctx, config, &pv, result)
	}

	// Check password environment variable
	if config.CertificatePasswordEnv != "" {
		if _, exists := c.env.LookupEnv(config.CertificatePasswordEnv); !exists {
			addWarning(result, types.ValidationWarning{
				Code:     "WIN_CERT_PASSWORD_ENV_NOT_SET",
				Platform: types.PlatformWindows,
				Message:  fmt.Sprintf("Environment variable %s is not set", config.CertificatePasswordEnv),
			})
		}
	}

	result.Platforms[types.PlatformWindows] = pv
}

func (c *PrerequisiteChecker) checkWindowsFileCertificate(ctx context.Context, config *types.WindowsSigningConfig, pv *types.PlatformValidation, result *types.ValidationResult) {
	if !c.fs.Exists(config.CertificateFile) {
		addError(result, types.ValidationError{
			Code:        "WIN_CERT_FILE_NOT_FOUND",
			Platform:    types.PlatformWindows,
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
		addError(result, types.ValidationError{
			Code:        "WIN_CERT_FILE_UNREADABLE",
			Platform:    types.PlatformWindows,
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
		if strings.Contains(err.Error(), "password") {
			addWarning(result, types.ValidationWarning{
				Code:     "WIN_CERT_PASSWORD_NEEDED",
				Platform: types.PlatformWindows,
				Message:  "Certificate parsing failed - may need correct password at build time",
			})
		} else {
			addError(result, types.ValidationError{
				Code:        "WIN_CERT_INVALID",
				Platform:    types.PlatformWindows,
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
		addError(result, types.ValidationError{
			Code:        "WIN_CERT_NOT_CODE_SIGN",
			Platform:    types.PlatformWindows,
			Field:       "certificate_file",
			Message:     "Certificate is not valid for code signing",
			Remediation: "Use a certificate with the Code Signing extended key usage",
		})
		pv.Errors = append(pv.Errors, "Certificate not valid for code signing")
	}

	// Check certificate expiration
	c.checkCertificateExpiration(certInfo, types.PlatformWindows, pv, result)
}

func (c *PrerequisiteChecker) detectSigntool(ctx context.Context) types.ToolDetectionResult {
	result := types.ToolDetectionResult{
		Platform: types.PlatformWindows,
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
			if ver := extractSigntoolVersion(string(stdout)); ver != "" {
				result.Version = ver
			}
		}
		return result
	}

	// Try common Windows SDK paths
	sdkPaths := []string{
		`C:\Program Files (x86)\Windows Kits\10\bin\x64\signtool.exe`,
		`C:\Program Files (x86)\Windows Kits\10\bin\10.0.22621.0\x64\signtool.exe`,
		`C:\Program Files (x86)\Windows Kits\10\bin\10.0.22000.0\x64\signtool.exe`,
		`C:\Program Files (x86)\Windows Kits\10\bin\10.0.19041.0\x64\signtool.exe`,
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
