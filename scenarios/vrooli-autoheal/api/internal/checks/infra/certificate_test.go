// Package infra provides tests for certificate health checks
// [REQ:INFRA-CERT-001] [REQ:TEST-SEAM-001]
package infra

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"testing"
	"time"

	"vrooli-autoheal/internal/checks"
)

// mockFileReader implements FileReader for testing
type mockFileReader struct {
	files map[string][]byte
	stats map[string]os.FileInfo
	errs  map[string]error
}

func newMockFileReader() *mockFileReader {
	return &mockFileReader{
		files: make(map[string][]byte),
		stats: make(map[string]os.FileInfo),
		errs:  make(map[string]error),
	}
}

func (m *mockFileReader) ReadFile(path string) ([]byte, error) {
	if err, ok := m.errs[path]; ok {
		return nil, err
	}
	if data, ok := m.files[path]; ok {
		return data, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockFileReader) Stat(path string) (os.FileInfo, error) {
	if err, ok := m.errs[path]; ok {
		return nil, err
	}
	if info, ok := m.stats[path]; ok {
		return info, nil
	}
	// If we have file data, return a mock stat
	if _, ok := m.files[path]; ok {
		return &mockFileInfo{name: path}, nil
	}
	return nil, os.ErrNotExist
}

// mockFileInfo implements os.FileInfo for testing
type mockFileInfo struct {
	name string
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return 0 }
func (m *mockFileInfo) Mode() os.FileMode  { return 0644 }
func (m *mockFileInfo) ModTime() time.Time { return time.Now() }
func (m *mockFileInfo) IsDir() bool        { return false }
func (m *mockFileInfo) Sys() interface{}   { return nil }

// generateTestCert creates a test PEM certificate with the given expiration
func generateTestCert(notBefore, notAfter time.Time, subject string) []byte {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: subject,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, _ := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	return certPEM
}

// TestCertificateCheckInterface verifies CertificateCheck implements Check
// [REQ:INFRA-CERT-001]
func TestCertificateCheckInterface(t *testing.T) {
	var _ checks.Check = (*CertificateCheck)(nil)

	check := NewCertificateCheck()
	if check.ID() != "infra-certificate" {
		t.Errorf("ID() = %q, want %q", check.ID(), "infra-certificate")
	}
	if check.Title() == "" {
		t.Error("Title() is empty")
	}
	if check.Description() == "" {
		t.Error("Description() is empty")
	}
	if check.Importance() == "" {
		t.Error("Importance() is empty")
	}
	if check.IntervalSeconds() <= 0 {
		t.Error("IntervalSeconds() should be positive")
	}
	if check.Category() != checks.CategoryInfrastructure {
		t.Error("Category should be infrastructure")
	}
	// Should run on all platforms
	if check.Platforms() != nil {
		t.Error("CertificateCheck should run on all platforms")
	}
}

// TestCertificateCheckOptions verifies configuration options
// [REQ:INFRA-CERT-001]
func TestCertificateCheckOptions(t *testing.T) {
	check := NewCertificateCheck(
		WithCertWarningDays(14),
		WithCertCriticalDays(5),
		WithCertPaths([]string{"/custom/path/cert.pem"}),
	)

	if check.warningDays != 14 {
		t.Errorf("warningDays = %d, want %d", check.warningDays, 14)
	}
	if check.criticalDays != 5 {
		t.Errorf("criticalDays = %d, want %d", check.criticalDays, 5)
	}
	// Check that custom path was added (in addition to defaults)
	found := false
	for _, p := range check.certPaths {
		if p == "/custom/path/cert.pem" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Custom cert path was not added")
	}
}

// TestCertificateCheckRunNoCertificates verifies behavior when no certificates exist
// [REQ:INFRA-CERT-001] [REQ:TEST-SEAM-001]
func TestCertificateCheckRunNoCertificates(t *testing.T) {
	mockReader := newMockFileReader()
	// Don't add any files - simulate no certificates found

	check := NewCertificateCheck(
		WithFileReader(mockReader),
		WithCertPaths([]string{"/nonexistent/cert.pem"}),
	)
	// Override default paths to only check our nonexistent path
	check.certPaths = []string{"/nonexistent/cert.pem"}

	result := check.Run(context.Background())

	if result.CheckID != check.ID() {
		t.Errorf("CheckID = %q, want %q", result.CheckID, check.ID())
	}
	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want OK (no certificates = OK)", result.Status)
	}
	if result.Message != "No certificates found to monitor" {
		t.Errorf("Message = %q, unexpected", result.Message)
	}
}

// TestCertificateCheckRunValidCertificate verifies behavior with valid certificate
// [REQ:INFRA-CERT-001] [REQ:TEST-SEAM-001]
func TestCertificateCheckRunValidCertificate(t *testing.T) {
	mockReader := newMockFileReader()

	// Create a certificate valid for 30 days
	certData := generateTestCert(
		time.Now().Add(-24*time.Hour),
		time.Now().Add(30*24*time.Hour),
		"test.example.com",
	)
	mockReader.files["/test/cert.pem"] = certData

	check := NewCertificateCheck(
		WithFileReader(mockReader),
	)
	check.certPaths = []string{"/test/cert.pem"}

	result := check.Run(context.Background())

	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want OK for valid certificate", result.Status)
	}
	if result.Details == nil {
		t.Fatal("Details should not be nil")
	}
	if result.Details["certificatesChecked"].(int) != 1 {
		t.Errorf("certificatesChecked = %v, want 1", result.Details["certificatesChecked"])
	}
}

// TestCertificateCheckRunWarningCertificate verifies behavior when certificate is near expiry
// [REQ:INFRA-CERT-001] [REQ:TEST-SEAM-001]
func TestCertificateCheckRunWarningCertificate(t *testing.T) {
	mockReader := newMockFileReader()

	// Create a certificate expiring in 5 days (within warning threshold of 7 days)
	certData := generateTestCert(
		time.Now().Add(-24*time.Hour),
		time.Now().Add(5*24*time.Hour),
		"expiring.example.com",
	)
	mockReader.files["/test/cert.pem"] = certData

	check := NewCertificateCheck(
		WithFileReader(mockReader),
		WithCertWarningDays(7),
		WithCertCriticalDays(3),
	)
	check.certPaths = []string{"/test/cert.pem"}

	result := check.Run(context.Background())

	if result.Status != checks.StatusWarning {
		t.Errorf("Status = %v, want Warning for certificate expiring in 5 days", result.Status)
	}
}

// TestCertificateCheckRunCriticalCertificate verifies behavior when certificate is critically near expiry
// [REQ:INFRA-CERT-001] [REQ:TEST-SEAM-001]
func TestCertificateCheckRunCriticalCertificate(t *testing.T) {
	mockReader := newMockFileReader()

	// Create a certificate expiring in 2 days (within critical threshold of 3 days)
	certData := generateTestCert(
		time.Now().Add(-24*time.Hour),
		time.Now().Add(2*24*time.Hour),
		"critical.example.com",
	)
	mockReader.files["/test/cert.pem"] = certData

	check := NewCertificateCheck(
		WithFileReader(mockReader),
		WithCertWarningDays(7),
		WithCertCriticalDays(3),
	)
	check.certPaths = []string{"/test/cert.pem"}

	result := check.Run(context.Background())

	if result.Status != checks.StatusCritical {
		t.Errorf("Status = %v, want Critical for certificate expiring in 2 days", result.Status)
	}
}

// TestCertificateCheckRunExpiredCertificate verifies behavior with expired certificate
// [REQ:INFRA-CERT-001] [REQ:TEST-SEAM-001]
func TestCertificateCheckRunExpiredCertificate(t *testing.T) {
	mockReader := newMockFileReader()

	// Create an expired certificate
	certData := generateTestCert(
		time.Now().Add(-30*24*time.Hour),
		time.Now().Add(-1*24*time.Hour),
		"expired.example.com",
	)
	mockReader.files["/test/cert.pem"] = certData

	check := NewCertificateCheck(
		WithFileReader(mockReader),
	)
	check.certPaths = []string{"/test/cert.pem"}

	result := check.Run(context.Background())

	if result.Status != checks.StatusCritical {
		t.Errorf("Status = %v, want Critical for expired certificate", result.Status)
	}
}

// TestCertificateCheckRunInvalidPEM verifies behavior with invalid PEM data
// [REQ:INFRA-CERT-001] [REQ:TEST-SEAM-001]
func TestCertificateCheckRunInvalidPEM(t *testing.T) {
	mockReader := newMockFileReader()
	mockReader.files["/test/invalid.pem"] = []byte("not a valid PEM file")

	check := NewCertificateCheck(
		WithFileReader(mockReader),
	)
	check.certPaths = []string{"/test/invalid.pem"}

	result := check.Run(context.Background())

	// Should handle gracefully without crashing
	if result.Details == nil {
		t.Fatal("Details should not be nil")
	}
	certs := result.Details["certificates"].([]certInfo)
	if len(certs) != 1 {
		t.Errorf("Expected 1 certificate entry, got %d", len(certs))
	}
	if certs[0].Status != "error" {
		t.Errorf("Expected certificate status 'error', got %q", certs[0].Status)
	}
}

// TestCertificateCheckRunReadError verifies behavior when file read fails
// [REQ:INFRA-CERT-001] [REQ:TEST-SEAM-001]
func TestCertificateCheckRunReadError(t *testing.T) {
	mockReader := newMockFileReader()
	// File exists (stat succeeds) but read fails
	mockReader.stats["/test/unreadable.pem"] = &mockFileInfo{name: "unreadable.pem"}
	mockReader.errs["/test/unreadable.pem"] = errors.New("permission denied")

	check := NewCertificateCheck(
		WithFileReader(mockReader),
	)
	check.certPaths = []string{"/test/unreadable.pem"}

	result := check.Run(context.Background())

	// Should handle read errors gracefully
	if result.Details == nil {
		t.Fatal("Details should not be nil")
	}
}

// TestCertificateCheckRunMultipleCertificates verifies behavior with multiple certificates
// [REQ:INFRA-CERT-001] [REQ:TEST-SEAM-001]
func TestCertificateCheckRunMultipleCertificates(t *testing.T) {
	mockReader := newMockFileReader()

	// Valid certificate (30 days)
	validCert := generateTestCert(
		time.Now().Add(-24*time.Hour),
		time.Now().Add(30*24*time.Hour),
		"valid.example.com",
	)
	mockReader.files["/test/valid.pem"] = validCert

	// Warning certificate (5 days)
	warningCert := generateTestCert(
		time.Now().Add(-24*time.Hour),
		time.Now().Add(5*24*time.Hour),
		"warning.example.com",
	)
	mockReader.files["/test/warning.pem"] = warningCert

	check := NewCertificateCheck(
		WithFileReader(mockReader),
		WithCertWarningDays(7),
		WithCertCriticalDays(3),
	)
	check.certPaths = []string{"/test/valid.pem", "/test/warning.pem"}

	result := check.Run(context.Background())

	// Should return worst status (Warning)
	if result.Status != checks.StatusWarning {
		t.Errorf("Status = %v, want Warning (worst of valid + warning)", result.Status)
	}
	if result.Details["certificatesChecked"].(int) != 2 {
		t.Errorf("certificatesChecked = %v, want 2", result.Details["certificatesChecked"])
	}
}

// TestCertificateCheckScoreCalculation verifies health score is calculated correctly
// [REQ:INFRA-CERT-001]
func TestCertificateCheckScoreCalculation(t *testing.T) {
	tests := []struct {
		name       string
		daysUntil  int
		wantScore  int
		wantStatus checks.Status
	}{
		{"30 days", 30, 100, checks.StatusOK},
		{"15 days", 15, 80, checks.StatusOK}, // Between warning and 30: 70 + (15-7)*30/(30-7) ≈ 80
		{"5 days (warning)", 5, 50, checks.StatusWarning},
		{"2 days (critical)", 2, 0, checks.StatusCritical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReader := newMockFileReader()
			certData := generateTestCert(
				time.Now().Add(-24*time.Hour),
				time.Now().Add(time.Duration(tt.daysUntil)*24*time.Hour),
				"test.example.com",
			)
			mockReader.files["/test/cert.pem"] = certData

			check := NewCertificateCheck(
				WithFileReader(mockReader),
				WithCertWarningDays(7),
				WithCertCriticalDays(3),
			)
			check.certPaths = []string{"/test/cert.pem"}

			result := check.Run(context.Background())

			if result.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", result.Status, tt.wantStatus)
			}
			if result.Metrics == nil || result.Metrics.Score == nil {
				t.Fatal("Expected metrics with score")
			}
			// Allow some variance due to time calculations
			score := *result.Metrics.Score
			if score < tt.wantScore-10 || score > tt.wantScore+10 {
				t.Errorf("Score = %d, want ~%d", score, tt.wantScore)
			}
		})
	}
}

// TestExpandPath verifies path expansion works correctly
func TestExpandPath(t *testing.T) {
	// Test non-tilde path (should remain unchanged)
	path := "/etc/ssl/cert.pem"
	expanded := expandPath(path)
	if expanded != path {
		t.Errorf("expandPath(%q) = %q, want unchanged", path, expanded)
	}
}
