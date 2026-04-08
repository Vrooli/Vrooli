package validation

import (
	"context"
	"fmt"
	"strings"

	"deployment-manager/codesigning"
)

// checkWindowsPrerequisites validates Windows signing tools and certificates.
func (c *prerequisiteChecker) checkWindowsPrerequisites(ctx context.Context, config *codesigning.WindowsSigningConfig, result *codesigning.ValidationResult) {
	pv := codesigning.PlatformValidation{
		Configured: true,
		Errors:     []string{},
		Warnings:   []string{},
	}

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

	switch config.CertificateSource {
	case codesigning.CertSourceFile:
		c.checkWindowsFileCertificate(ctx, config, &pv, result)
	case codesigning.CertSourceStore:
		result.AddWarning(codesigning.ValidationWarning{
			Code:     "WIN_STORE_CERT_RUNTIME_CHECK",
			Platform: codesigning.PlatformWindows,
			Message:  "Certificate store availability will be verified at signing time",
		})
	}

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

	password := ""
	if config.CertificatePasswordEnv != "" {
		password = c.env.GetEnv(config.CertificatePasswordEnv)
	}

	certInfo, err := c.parsePKCS12Certificate(certData, password)
	if err != nil {
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

	c.checkCertificateExpiration(certInfo, codesigning.PlatformWindows, pv, result)
}

func (c *prerequisiteChecker) detectSigntool(ctx context.Context) codesigning.ToolDetectionResult {
	result := codesigning.ToolDetectionResult{
		Platform: codesigning.PlatformWindows,
		Tool:     "signtool.exe",
	}

	path, err := c.cmd.LookPath("signtool.exe")
	if err == nil {
		result.Installed = true
		result.Path = path

		stdout, _, err := c.cmd.Run(ctx, "signtool.exe")
		if err == nil {
			if ver := extractSigntoolVersion(string(stdout)); ver != "" {
				result.Version = ver
			}
		}
		return result
	}

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

// discoverWindowsCertificates lists code signing certificates from Windows Certificate Store.
func (c *prerequisiteChecker) discoverWindowsCertificates(ctx context.Context) ([]codesigning.DiscoveredCertificate, error) {
	if runtimeGOOS() != "windows" {
		return nil, nil
	}

	var certs []codesigning.DiscoveredCertificate

	psScript := `Get-ChildItem Cert:\CurrentUser\My -CodeSigningCert | ForEach-Object {
		$cert = $_
		Write-Output ("CERT:" + $cert.Thumbprint + "|" + $cert.Subject + "|" + $cert.Issuer + "|" + $cert.NotAfter.ToString("yyyy-MM-dd") + "|" + $cert.FriendlyName)
	}`

	stdout, stderr, err := c.cmd.Run(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	if err != nil {
		return nil, fmt.Errorf("failed to list certificates: %v (stderr: %s)", err, string(stderr))
	}

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

		daysToExpiry := -1
		isExpired := false
		if t, err := parseDate(expiresStr); err == nil {
			now := c.time.Now()
			duration := t.Sub(now)
			daysToExpiry = int(duration.Hours() / 24)
			isExpired = now.After(t)
		}

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
