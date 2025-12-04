// Package infra provides infrastructure health checks
// [REQ:INFRA-CERT-001]
package infra

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// CertificateCheck monitors SSL/TLS certificate expiration.
// Currently focuses on cloudflared tunnel certificates but can be extended.
type CertificateCheck struct {
	warningDays  int      // Days before expiry to warn
	criticalDays int      // Days before expiry to go critical
	certPaths    []string // Paths to check for certificates
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

// NewCertificateCheck creates a certificate expiration check.
// Default thresholds: warning at 7 days, critical at 3 days.
func NewCertificateCheck(opts ...CertificateCheckOption) *CertificateCheck {
	c := &CertificateCheck{
		warningDays:  7,
		criticalDays: 3,
		certPaths:    getDefaultCertPaths(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *CertificateCheck) ID() string          { return "infra-certificate" }
func (c *CertificateCheck) Title() string       { return "Certificate Expiration" }
func (c *CertificateCheck) Description() string { return "Monitors SSL/TLS certificate expiration dates" }
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

	for _, certPath := range c.certPaths {
		// Expand home directory
		expandedPath := expandPath(certPath)

		// Check if file exists
		if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
			continue // Skip missing files silently
		}

		// Read and parse certificate
		info, err := c.checkCertificate(expandedPath)
		if err != nil {
			certsChecked = append(certsChecked, certInfo{
				Path:   certPath,
				Error:  err.Error(),
				Status: "error",
			})
			continue
		}

		certsChecked = append(certsChecked, info)

		// Track soonest expiry
		if soonestExpiry.IsZero() || info.ExpiresAt.Before(soonestExpiry) {
			soonestExpiry = info.ExpiresAt
		}

		// Update worst status
		if info.Status == "critical" && worstStatus != checks.StatusCritical {
			worstStatus = checks.StatusCritical
			worstMessage = fmt.Sprintf("Certificate %s expires in %d days", info.Path, info.DaysUntilExpiry)
		} else if info.Status == "warning" && worstStatus == checks.StatusOK {
			worstStatus = checks.StatusWarning
			worstMessage = fmt.Sprintf("Certificate %s expires in %d days", info.Path, info.DaysUntilExpiry)
		}
	}

	result.Details["certificates"] = certsChecked
	result.Details["warningThresholdDays"] = c.warningDays
	result.Details["criticalThresholdDays"] = c.criticalDays
	result.Details["certificatesChecked"] = len(certsChecked)

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

	result.Status = worstStatus
	if worstStatus == checks.StatusOK {
		result.Message = fmt.Sprintf("All %d certificates are valid", len(certsChecked))
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

// checkCertificate reads and validates a certificate file
func (c *CertificateCheck) checkCertificate(path string) (certInfo, error) {
	info := certInfo{
		Path: path,
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return info, fmt.Errorf("failed to read certificate: %w", err)
	}

	// Parse PEM block
	block, _ := pem.Decode(data)
	if block == nil {
		return info, fmt.Errorf("failed to parse PEM block")
	}

	// Parse certificate
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return info, fmt.Errorf("failed to parse certificate: %w", err)
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
