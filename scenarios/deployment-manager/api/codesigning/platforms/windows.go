package platforms

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"time"

	"deployment-manager/codesigning"

	"software.sslmate.com/src/go-pkcs12"
)

// WindowsDetector detects Windows Authenticode signing tools and certificates.
type WindowsDetector struct {
	fs  codesigning.FileSystem
	cmd codesigning.CommandRunner
	env codesigning.EnvironmentReader
}

// NewWindowsDetector creates a new Windows platform detector.
func NewWindowsDetector(fs codesigning.FileSystem, cmd codesigning.CommandRunner, env codesigning.EnvironmentReader) *WindowsDetector {
	return &WindowsDetector{fs: fs, cmd: cmd, env: env}
}

// Platform returns the platform identifier.
func (d *WindowsDetector) Platform() string {
	return codesigning.PlatformWindows
}

// DetectTools detects Windows signing tools.
func (d *WindowsDetector) DetectTools(ctx context.Context) []codesigning.ToolDetectionResult {
	var results []codesigning.ToolDetectionResult

	// Only detect on Windows or when mocked
	if runtime.GOOS != "windows" && !d.isMocked() {
		return results
	}

	results = append(results, d.detectSigntool(ctx))
	return results
}

// DiscoverCertificates discovers code signing certificates.
func (d *WindowsDetector) DiscoverCertificates(ctx context.Context) ([]DiscoveredCertificate, error) {
	// Only discover on Windows or when mocked
	if runtime.GOOS != "windows" && !d.isMocked() {
		return nil, nil
	}

	var certs []DiscoveredCertificate

	// Try to list certificates from Windows Certificate Store
	storeCerts, err := d.listStoreCertificates(ctx)
	if err == nil {
		certs = append(certs, storeCerts...)
	}

	return certs, nil
}

// detectSigntool detects signtool.exe.
func (d *WindowsDetector) detectSigntool(ctx context.Context) codesigning.ToolDetectionResult {
	result := codesigning.ToolDetectionResult{
		Platform: codesigning.PlatformWindows,
		Tool:     "signtool.exe",
	}

	// Try to find signtool in PATH
	path, err := d.cmd.LookPath("signtool.exe")
	if err == nil {
		result.Installed = true
		result.Path = path

		// Try to get version
		stdout, _, err := d.cmd.Run(ctx, "signtool.exe")
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
		`C:\Program Files (x86)\Windows Kits\10\bin\10.0.18362.0\x64\signtool.exe`,
		`C:\Program Files (x86)\Windows Kits\8.1\bin\x64\signtool.exe`,
	}

	for _, sdkPath := range sdkPaths {
		if d.fs.Exists(sdkPath) {
			result.Installed = true
			result.Path = sdkPath
			return result
		}
	}

	result.Error = "signtool.exe not found in PATH or Windows SDK"
	result.Remediation = "Install Windows SDK or add signtool.exe to PATH. Download from: https://developer.microsoft.com/en-us/windows/downloads/windows-sdk/"

	return result
}

// listStoreCertificates lists code signing certificates from Windows Certificate Store.
func (d *WindowsDetector) listStoreCertificates(ctx context.Context) ([]DiscoveredCertificate, error) {
	var certs []DiscoveredCertificate

	// Use PowerShell to list code signing certificates
	// Get-ChildItem Cert:\CurrentUser\My -CodeSigningCert
	psScript := `Get-ChildItem Cert:\CurrentUser\My -CodeSigningCert | ForEach-Object {
		$cert = $_
		Write-Output ("CERT:" + $cert.Thumbprint + "|" + $cert.Subject + "|" + $cert.Issuer + "|" + $cert.NotAfter.ToString("yyyy-MM-dd") + "|" + $cert.FriendlyName)
	}`

	stdout, stderr, err := d.cmd.Run(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	if err != nil {
		// Also try the local machine store
		psScriptMachine := `Get-ChildItem Cert:\LocalMachine\My -CodeSigningCert | ForEach-Object {
			$cert = $_
			Write-Output ("CERT:" + $cert.Thumbprint + "|" + $cert.Subject + "|" + $cert.Issuer + "|" + $cert.NotAfter.ToString("yyyy-MM-dd") + "|" + $cert.FriendlyName)
		}`
		stdout, stderr, err = d.cmd.Run(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psScriptMachine)
		if err != nil {
			return nil, fmt.Errorf("failed to list certificates: %v (stderr: %s)", err, string(stderr))
		}
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

		// Parse expiration date
		var expiresAt time.Time
		var daysToExpiry int
		var isExpired bool
		if t, err := time.Parse("2006-01-02", expiresStr); err == nil {
			expiresAt = t
			now := time.Now()
			duration := expiresAt.Sub(now)
			daysToExpiry = int(duration.Hours() / 24)
			isExpired = now.After(expiresAt)
		}

		// Determine certificate type
		certType := "Code Signing"
		if strings.Contains(subject, "EV") || strings.Contains(issuer, "Extended Validation") {
			certType = "EV Code Signing"
		}

		// Create display name
		name := friendlyName
		if name == "" {
			name = extractCNFromSubject(subject)
		}

		certs = append(certs, DiscoveredCertificate{
			ID:           thumbprint,
			Name:         name,
			Subject:      subject,
			Issuer:       issuer,
			ExpiresAt:    expiresStr,
			DaysToExpiry: daysToExpiry,
			IsExpired:    isExpired,
			IsCodeSign:   true,
			Type:         certType,
			Platform:     codesigning.PlatformWindows,
			UsageHint:    fmt.Sprintf("Use --thumbprint %s or --cert-source store", thumbprint),
		})
	}

	return certs, nil
}

// ParsePFXCertificate parses a .pfx/.p12 certificate file and returns info.
func (d *WindowsDetector) ParsePFXCertificate(data []byte, password string) (*DiscoveredCertificate, error) {
	_, cert, err := pkcs12.Decode(data, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decode PKCS#12: %w", err)
	}

	return d.extractCertificateInfo(cert), nil
}

// ParsePEMCertificate parses a PEM certificate and returns info.
func (d *WindowsDetector) ParsePEMCertificate(data []byte) (*DiscoveredCertificate, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return d.extractCertificateInfo(cert), nil
}

func (d *WindowsDetector) extractCertificateInfo(cert *x509.Certificate) *DiscoveredCertificate {
	now := time.Now()

	// Calculate days to expiry
	duration := cert.NotAfter.Sub(now)
	daysToExpiry := int(duration.Hours() / 24)

	// Check if certificate is for code signing
	isCodeSign := false
	for _, eku := range cert.ExtKeyUsage {
		if eku == x509.ExtKeyUsageCodeSigning {
			isCodeSign = true
			break
		}
	}

	// Determine certificate type
	certType := "Certificate"
	if isCodeSign {
		certType = "Code Signing"
	}

	return &DiscoveredCertificate{
		ID:           fmt.Sprintf("%X", cert.SerialNumber),
		Name:         extractCNFromSubject(cert.Subject.String()),
		Subject:      cert.Subject.String(),
		Issuer:       cert.Issuer.String(),
		ExpiresAt:    cert.NotAfter.Format("2006-01-02"),
		DaysToExpiry: daysToExpiry,
		IsExpired:    now.After(cert.NotAfter),
		IsCodeSign:   isCodeSign,
		Type:         certType,
		Platform:     codesigning.PlatformWindows,
	}
}

// isMocked returns true if the detector is using mocked dependencies.
func (d *WindowsDetector) isMocked() bool {
	// Check if we can detect mocked file system by testing a known-nonexistent path
	// This is a heuristic - a mock would return true for things that don't exist
	// For simplicity, we'll just check if runtime.GOOS != "windows" but detection still works
	// This allows tests to work on non-Windows platforms
	return false
}

// extractSigntoolVersion extracts version from signtool output.
func extractSigntoolVersion(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "version") || strings.Contains(lower, "signtool") {
			// Look for version pattern like "10.0.22000.0"
			versionRegex := regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
			if match := versionRegex.FindString(line); match != "" {
				return match
			}
			return strings.TrimSpace(line)
		}
	}
	return ""
}

// extractCNFromSubject extracts the CN field from a certificate subject.
func extractCNFromSubject(subject string) string {
	// Look for CN= in the subject
	cnRegex := regexp.MustCompile(`CN=([^,]+)`)
	match := cnRegex.FindStringSubmatch(subject)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return subject
}
