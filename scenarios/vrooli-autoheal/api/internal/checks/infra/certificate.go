// Package infra provides infrastructure health checks
// [REQ:INFRA-CERT-001] [REQ:HEAL-ACTION-001]
package infra

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// FileReader abstracts file system operations for testing.
// [REQ:TEST-SEAM-001]
type FileReader interface {
	ReadFile(path string) ([]byte, error)
	Stat(path string) (os.FileInfo, error)
}

// defaultFileReader implements FileReader using the real filesystem.
type defaultFileReader struct{}

func (d *defaultFileReader) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (d *defaultFileReader) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// CertificateCheck monitors SSL/TLS certificate expiration.
// Currently focuses on cloudflared tunnel certificates but can be extended.
type CertificateCheck struct {
	warningDays  int                    // Days before expiry to warn
	criticalDays int                    // Days before expiry to go critical
	certPaths    []string               // Paths to check for certificates
	fileReader   FileReader             // Injectable file reader for testing
	executor     checks.CommandExecutor // Injectable executor for recovery actions
	caps         *platform.Capabilities // Platform capabilities for recovery actions
}

// CertificateCheckOption configures a CertificateCheck.
type CertificateCheckOption func(*CertificateCheck)

// WithCertWarningDays sets the warning threshold in days.
func WithCertWarningDays(days int) CertificateCheckOption {
	return func(c *CertificateCheck) {
		c.warningDays = days
	}
}

// WithCertCriticalDays sets the critical threshold in days.
func WithCertCriticalDays(days int) CertificateCheckOption {
	return func(c *CertificateCheck) {
		c.criticalDays = days
	}
}

// WithCertPaths sets additional certificate paths to check.
func WithCertPaths(paths []string) CertificateCheckOption {
	return func(c *CertificateCheck) {
		c.certPaths = append(c.certPaths, paths...)
	}
}

// WithFileReader sets the file reader for testing.
// [REQ:TEST-SEAM-001]
func WithFileReader(fr FileReader) CertificateCheckOption {
	return func(c *CertificateCheck) {
		c.fileReader = fr
	}
}

// WithCertExecutor sets the command executor for recovery actions.
// [REQ:TEST-SEAM-001]
func WithCertExecutor(executor checks.CommandExecutor) CertificateCheckOption {
	return func(c *CertificateCheck) {
		c.executor = executor
	}
}

// WithCertPlatformCaps sets the platform capabilities for recovery actions.
func WithCertPlatformCaps(caps *platform.Capabilities) CertificateCheckOption {
	return func(c *CertificateCheck) {
		c.caps = caps
	}
}

// NewCertificateCheck creates a certificate expiration check.
// Default thresholds: warning at 7 days, critical at 3 days.
func NewCertificateCheck(opts ...CertificateCheckOption) *CertificateCheck {
	c := &CertificateCheck{
		warningDays:  7,
		criticalDays: 3,
		certPaths:    getDefaultCertPaths(),
		fileReader:   &defaultFileReader{},
		executor:     checks.DefaultExecutor,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *CertificateCheck) ID() string    { return "infra-certificate" }
func (c *CertificateCheck) Title() string { return "Certificate Expiration" }
func (c *CertificateCheck) Description() string {
	return "Monitors SSL/TLS certificate expiration dates"
}
func (c *CertificateCheck) Importance() string {
	return "Required for secure connections - expired certificates break HTTPS, tunnels, and API access"
}
func (c *CertificateCheck) Category() checks.Category  { return checks.CategoryInfrastructure }
func (c *CertificateCheck) IntervalSeconds() int       { return 3600 } // Check hourly
func (c *CertificateCheck) Platforms() []platform.Type { return nil }  // All platforms

func (c *CertificateCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	// Track certificates found and their status
	var certsChecked []certInfo
	var worstStatus checks.Status = checks.StatusOK
	var worstMessage string
	soonestExpiry := time.Time{}
	var validCertCount, skippedCount, errorCount int

	for _, certPath := range c.certPaths {
		// Expand home directory
		expandedPath := expandPath(certPath)

		// Handle glob patterns (e.g., "*.crt", "*.pem")
		// If it's a glob pattern, expand it; otherwise use as-is
		var paths []string
		if matches, err := filepath.Glob(expandedPath); err == nil && len(matches) > 0 {
			paths = matches
		} else {
			// Not a glob or no matches - try as literal path
			paths = []string{expandedPath}
		}

		for _, path := range paths {
			// Check if file exists
			if _, err := c.fileReader.Stat(path); os.IsNotExist(err) {
				continue // Skip missing files silently
			}

			// Read and parse certificate
			info, err := c.checkCertificate(path)
			if err != nil {
				certsChecked = append(certsChecked, certInfo{
					Path:   path,
					Error:  err.Error(),
					Status: "error",
				})
				errorCount++
				// Parse errors should be reported as warnings
				if worstStatus == checks.StatusOK {
					worstStatus = checks.StatusWarning
					worstMessage = fmt.Sprintf("Failed to parse certificate %s", path)
				}
				continue
			}

			certsChecked = append(certsChecked, info)

			// Handle skipped files (non-certificate PEM files like tokens)
			if info.Status == "skipped" {
				skippedCount++
				continue // Don't count skipped files in expiry tracking or status
			}

			validCertCount++

			// Track soonest expiry (only for valid certificates)
			if soonestExpiry.IsZero() || info.ExpiresAt.Before(soonestExpiry) {
				soonestExpiry = info.ExpiresAt
			}

			// Update worst status based on certificate expiry
			if info.Status == "critical" && worstStatus != checks.StatusCritical {
				worstStatus = checks.StatusCritical
				worstMessage = fmt.Sprintf("Certificate %s expires in %d days", info.Path, info.DaysUntilExpiry)
			} else if info.Status == "warning" && worstStatus == checks.StatusOK {
				worstStatus = checks.StatusWarning
				worstMessage = fmt.Sprintf("Certificate %s expires in %d days", info.Path, info.DaysUntilExpiry)
			}
		}
	}

	result.Details["certificates"] = certsChecked
	result.Details["warningThresholdDays"] = c.warningDays
	result.Details["criticalThresholdDays"] = c.criticalDays
	result.Details["certificatesChecked"] = len(certsChecked)
	result.Details["validCertificates"] = validCertCount
	result.Details["skippedFiles"] = skippedCount
	result.Details["parseErrors"] = errorCount

	if !soonestExpiry.IsZero() {
		result.Details["soonestExpiry"] = soonestExpiry.Format(time.RFC3339)
		result.Details["soonestExpiryDays"] = int(time.Until(soonestExpiry).Hours() / 24)
	}

	// Calculate score based on soonest expiry
	if !soonestExpiry.IsZero() {
		daysUntil := int(time.Until(soonestExpiry).Hours() / 24)
		score := 100
		if daysUntil <= c.criticalDays {
			score = 0
		} else if daysUntil <= c.warningDays {
			score = 50
		} else if daysUntil <= 30 {
			// Gradually reduce score as expiry approaches
			score = 70 + (daysUntil-c.warningDays)*30/(30-c.warningDays)
		}
		result.Metrics = &checks.HealthMetrics{
			Score: &score,
		}
	}

	// Set final status
	if len(certsChecked) == 0 {
		result.Status = checks.StatusOK
		result.Message = "No certificates found to monitor"
		result.Details["note"] = "This is normal if cloudflared is not installed or using different cert paths"
		return result
	}

	// If we only found non-certificate files (all skipped), that's OK
	if validCertCount == 0 && skippedCount > 0 && errorCount == 0 {
		result.Status = checks.StatusOK
		result.Message = fmt.Sprintf("No X.509 certificates found (%d non-certificate files skipped)", skippedCount)
		result.Details["note"] = "Files checked were not X.509 certificates (e.g., tunnel tokens)"
		return result
	}

	result.Status = worstStatus
	if worstStatus == checks.StatusOK {
		if skippedCount > 0 {
			result.Message = fmt.Sprintf("All %d certificates are valid (%d non-certificate files skipped)", validCertCount, skippedCount)
		} else {
			result.Message = fmt.Sprintf("All %d certificates are valid", validCertCount)
		}
	} else {
		result.Message = worstMessage
	}

	return result
}

// certInfo holds information about a certificate
type certInfo struct {
	Path            string    `json:"path"`
	Subject         string    `json:"subject,omitempty"`
	Issuer          string    `json:"issuer,omitempty"`
	ExpiresAt       time.Time `json:"expiresAt,omitempty"`
	DaysUntilExpiry int       `json:"daysUntilExpiry,omitempty"`
	Status          string    `json:"status"` // ok, warning, critical, error
	Error           string    `json:"error,omitempty"`
}

// knownNonCertTypes are PEM block types that are NOT certificates and should be skipped
var knownNonCertTypes = map[string]bool{
	"ARGO TUNNEL TOKEN":     true, // Cloudflare tunnel authentication token
	"PRIVATE KEY":           true,
	"RSA PRIVATE KEY":       true,
	"EC PRIVATE KEY":        true,
	"ENCRYPTED PRIVATE KEY": true,
	"PUBLIC KEY":            true,
	"RSA PUBLIC KEY":        true,
}

// checkCertificate reads and validates a certificate file
func (c *CertificateCheck) checkCertificate(path string) (certInfo, error) {
	info := certInfo{
		Path: path,
	}

	// Read file using injectable file reader
	data, err := c.fileReader.ReadFile(path)
	if err != nil {
		return info, fmt.Errorf("failed to read certificate: %w", err)
	}

	// Parse PEM block
	block, _ := pem.Decode(data)
	if block == nil {
		return info, fmt.Errorf("failed to parse PEM block")
	}

	// Check if this is a known non-certificate PEM type
	if knownNonCertTypes[block.Type] {
		info.Status = "skipped"
		info.Error = fmt.Sprintf("File is a %s, not a certificate", block.Type)
		return info, nil // Return without error - this is expected for token files
	}

	// Parse certificate
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return info, fmt.Errorf("failed to parse certificate (PEM type: %s): %w", block.Type, err)
	}

	// Extract info
	info.Subject = cert.Subject.CommonName
	info.Issuer = cert.Issuer.CommonName
	info.ExpiresAt = cert.NotAfter
	info.DaysUntilExpiry = int(time.Until(cert.NotAfter).Hours() / 24)

	// Determine status based on expiry
	if info.DaysUntilExpiry <= 0 {
		info.Status = "critical"
		info.Error = "Certificate has expired"
	} else if info.DaysUntilExpiry <= c.criticalDays {
		info.Status = "critical"
	} else if info.DaysUntilExpiry <= c.warningDays {
		info.Status = "warning"
	} else {
		info.Status = "ok"
	}

	return info, nil
}

// getDefaultCertPaths returns the default certificate paths to check
func getDefaultCertPaths() []string {
	var paths []string

	// Cloudflared certificate paths
	home, _ := os.UserHomeDir()
	if home != "" {
		paths = append(paths,
			filepath.Join(home, ".cloudflared", "cert.pem"),
			filepath.Join(home, ".cloudflared", "*.crt"),
		)
	}

	// System-wide cloudflared paths
	if runtime.GOOS == "linux" {
		paths = append(paths,
			"/etc/cloudflared/cert.pem",
			"/etc/ssl/certs/cloudflared.crt",
		)
	} else if runtime.GOOS == "darwin" {
		paths = append(paths,
			"/usr/local/etc/cloudflared/cert.pem",
		)
	}

	return paths
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}

// extractCertificatesFromDetails safely extracts certificate info from Details map.
// Handles both direct []certInfo (in-memory) and []interface{} (after JSON round-trip).
func extractCertificatesFromDetails(details map[string]interface{}) []certInfo {
	if details == nil {
		return nil
	}

	rawCerts, ok := details["certificates"]
	if !ok {
		return nil
	}

	// Try direct type assertion first (in-memory results)
	if certs, ok := rawCerts.([]certInfo); ok {
		return certs
	}

	// Handle JSON-unmarshaled []interface{} (persisted results)
	rawSlice, ok := rawCerts.([]interface{})
	if !ok {
		return nil
	}

	var certs []certInfo
	for _, item := range rawSlice {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		cert := certInfo{}
		if v, ok := itemMap["path"].(string); ok {
			cert.Path = v
		}
		if v, ok := itemMap["subject"].(string); ok {
			cert.Subject = v
		}
		if v, ok := itemMap["issuer"].(string); ok {
			cert.Issuer = v
		}
		if v, ok := itemMap["status"].(string); ok {
			cert.Status = v
		}
		if v, ok := itemMap["error"].(string); ok {
			cert.Error = v
		}
		// Handle daysUntilExpiry (may be float64 from JSON)
		if v, ok := itemMap["daysUntilExpiry"].(float64); ok {
			cert.DaysUntilExpiry = int(v)
		} else if v, ok := itemMap["daysUntilExpiry"].(int); ok {
			cert.DaysUntilExpiry = v
		}
		// Handle expiresAt (time.Time from JSON becomes string)
		if v, ok := itemMap["expiresAt"].(string); ok {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				cert.ExpiresAt = t
			}
		}

		certs = append(certs, cert)
	}

	return certs
}

// RecoveryActions returns available recovery actions for certificate issues
// [REQ:HEAL-ACTION-001]
func (c *CertificateCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	isLinux := runtime.GOOS == "linux"
	hasSystemd := c.caps != nil && c.caps.SupportsSystemd
	hasCriticalCert := false
	hasExpiredCert := false

	// Check if we have critical or expired certificates
	if lastResult != nil {
		certs := extractCertificatesFromDetails(lastResult.Details)
		for _, cert := range certs {
			if cert.Status == "critical" {
				hasCriticalCert = true
				if cert.DaysUntilExpiry <= 0 {
					hasExpiredCert = true
				}
			}
		}
	}

	return []checks.RecoveryAction{
		{
			ID:          "renew-cloudflared",
			Name:        "Renew Cloudflared Cert",
			Description: "Re-authenticate cloudflared to renew tunnel certificate",
			Dangerous:   false,
			Available:   isLinux && (hasCriticalCert || hasExpiredCert),
		},
		{
			ID:          "restart-cloudflared",
			Name:        "Restart Cloudflared",
			Description: "Restart cloudflared service to pick up renewed certificates",
			Dangerous:   false,
			Available:   isLinux && hasSystemd,
		},
		{
			ID:          "check-expiry",
			Name:        "Check Expiry Details",
			Description: "Show detailed certificate expiration information",
			Dangerous:   false,
			Available:   true,
		},
		{
			ID:          "list-certs",
			Name:        "List Certificates",
			Description: "List all certificate files being monitored",
			Dangerous:   false,
			Available:   true,
		},
	}
}

// ExecuteAction runs the specified recovery action
// [REQ:HEAL-ACTION-001]
func (c *CertificateCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.ID(),
		Timestamp: start,
	}

	switch actionID {
	case "renew-cloudflared":
		return c.executeRenewCloudflared(ctx, result, start)

	case "restart-cloudflared":
		output, err := c.executor.CombinedOutput(ctx, "sudo", "systemctl", "restart", "cloudflared")
		result.Duration = time.Since(start)
		result.Output = string(output)

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to restart cloudflared service"
			return result
		}

		result.Success = true
		result.Message = "Cloudflared service restarted - certificates will be reloaded"
		return result

	case "check-expiry":
		return c.executeCheckExpiry(ctx, result, start)

	case "list-certs":
		return c.executeListCerts(result, start)

	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		result.Duration = time.Since(start)
		return result
	}
}

// executeRenewCloudflared attempts to renew cloudflared certificates
func (c *CertificateCheck) executeRenewCloudflared(ctx context.Context, result checks.ActionResult, start time.Time) checks.ActionResult {
	var outputBuilder strings.Builder

	outputBuilder.WriteString("=== Cloudflared Certificate Renewal ===\n\n")

	// Step 1: Check current tunnel status
	outputBuilder.WriteString("Step 1: Checking tunnel status...\n")
	tunnelOutput, err := c.executor.CombinedOutput(ctx, "cloudflared", "tunnel", "list")
	if err != nil {
		outputBuilder.WriteString(fmt.Sprintf("Warning: Could not list tunnels: %v\n", err))
	} else {
		outputBuilder.Write(tunnelOutput)
		outputBuilder.WriteString("\n")
	}

	// Step 2: Run cloudflared login to refresh credentials
	outputBuilder.WriteString("\nStep 2: To renew certificates, run:\n")
	outputBuilder.WriteString("  cloudflared tunnel login\n")
	outputBuilder.WriteString("\nThis will open a browser for authentication.\n")
	outputBuilder.WriteString("After authentication, run:\n")
	outputBuilder.WriteString("  sudo systemctl restart cloudflared\n")

	// Note: We can't run cloudflared tunnel login non-interactively
	// So we provide instructions instead
	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "Certificate renewal instructions provided - manual steps required"
	return result
}

// executeCheckExpiry shows detailed certificate information
func (c *CertificateCheck) executeCheckExpiry(ctx context.Context, result checks.ActionResult, start time.Time) checks.ActionResult {
	var outputBuilder strings.Builder

	outputBuilder.WriteString("=== Certificate Expiration Details ===\n\n")

	for _, certPath := range c.certPaths {
		expandedPath := expandPath(certPath)

		// Handle glob patterns
		paths, err := filepath.Glob(expandedPath)
		if err != nil || len(paths) == 0 {
			// Not a glob or no matches, try as literal path
			paths = []string{expandedPath}
		}

		for _, path := range paths {
			if _, err := c.fileReader.Stat(path); os.IsNotExist(err) {
				continue
			}

			info, err := c.checkCertificate(path)
			if err != nil {
				outputBuilder.WriteString(fmt.Sprintf("Certificate: %s\n", path))
				outputBuilder.WriteString(fmt.Sprintf("  Error: %v\n\n", err))
				continue
			}

			outputBuilder.WriteString(fmt.Sprintf("Certificate: %s\n", path))
			outputBuilder.WriteString(fmt.Sprintf("  Subject: %s\n", info.Subject))
			outputBuilder.WriteString(fmt.Sprintf("  Issuer: %s\n", info.Issuer))
			outputBuilder.WriteString(fmt.Sprintf("  Expires: %s\n", info.ExpiresAt.Format(time.RFC3339)))
			outputBuilder.WriteString(fmt.Sprintf("  Days Until Expiry: %d\n", info.DaysUntilExpiry))
			outputBuilder.WriteString(fmt.Sprintf("  Status: %s\n\n", strings.ToUpper(info.Status)))
		}
	}

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "Certificate expiry details retrieved"
	return result
}

// executeListCerts lists all certificate paths being monitored
func (c *CertificateCheck) executeListCerts(result checks.ActionResult, start time.Time) checks.ActionResult {
	var outputBuilder strings.Builder

	outputBuilder.WriteString("=== Monitored Certificate Paths ===\n\n")
	outputBuilder.WriteString(fmt.Sprintf("Warning Threshold: %d days\n", c.warningDays))
	outputBuilder.WriteString(fmt.Sprintf("Critical Threshold: %d days\n\n", c.criticalDays))

	foundCount := 0
	for _, certPath := range c.certPaths {
		expandedPath := expandPath(certPath)

		// Handle glob patterns
		paths, err := filepath.Glob(expandedPath)
		if err != nil || len(paths) == 0 {
			// Not a glob or no matches, try as literal path
			paths = []string{expandedPath}
		}

		for _, path := range paths {
			exists := "NOT FOUND"
			if _, err := c.fileReader.Stat(path); err == nil {
				exists = "FOUND"
				foundCount++
			}
			outputBuilder.WriteString(fmt.Sprintf("[%s] %s\n", exists, path))
		}
	}

	outputBuilder.WriteString(fmt.Sprintf("\nTotal certificates found: %d\n", foundCount))

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = fmt.Sprintf("Listed %d certificates", foundCount)
	return result
}

// Ensure CertificateCheck implements HealableCheck
var _ checks.HealableCheck = (*CertificateCheck)(nil)
