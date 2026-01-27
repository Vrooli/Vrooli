package tlsinfo

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"testing"
	"time"
)

func TestBuildProbeResultFullValidationError(t *testing.T) {
	now := time.Now()
	cert := &x509.Certificate{
		Issuer:       pkix.Name{CommonName: "Test CA"},
		Subject:      pkix.Name{CommonName: "example.com"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(time.Hour),
		SerialNumber: big.NewInt(12345),
	}
	result := buildProbeResult("example.com", cert, now, "full", errors.New("verify failed"))
	if result.Valid {
		t.Fatalf("expected invalid when full validation fails")
	}
	if result.Validation != "full" {
		t.Fatalf("expected validation full, got %q", result.Validation)
	}
	if result.ValidationError == "" {
		t.Fatalf("expected validation error to be set")
	}
}

func TestBuildProbeResultTimeOnlyValidation(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name      string
		notBefore time.Time
		notAfter  time.Time
		wantValid bool
	}{
		{
			name:      "valid certificate within time window",
			notBefore: now.Add(-24 * time.Hour),
			notAfter:  now.Add(30 * 24 * time.Hour),
			wantValid: true,
		},
		{
			name:      "expired certificate",
			notBefore: now.Add(-30 * 24 * time.Hour),
			notAfter:  now.Add(-time.Hour),
			wantValid: false,
		},
		{
			name:      "not yet valid certificate",
			notBefore: now.Add(time.Hour),
			notAfter:  now.Add(30 * 24 * time.Hour),
			wantValid: false,
		},
		{
			name:      "certificate expiring today",
			notBefore: now.Add(-24 * time.Hour),
			notAfter:  now.Add(12 * time.Hour),
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cert := &x509.Certificate{
				Issuer:       pkix.Name{CommonName: "Test CA"},
				Subject:      pkix.Name{CommonName: "example.com"},
				NotBefore:    tt.notBefore,
				NotAfter:     tt.notAfter,
				SerialNumber: big.NewInt(12345),
			}
			result := buildProbeResult("example.com", cert, now, "time_only", nil)
			if result.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v", result.Valid, tt.wantValid)
			}
			if result.Validation != "time_only" {
				t.Errorf("Validation = %q, want %q", result.Validation, "time_only")
			}
		})
	}
}

func TestBuildProbeResultNilCertificate(t *testing.T) {
	now := time.Now()
	result := buildProbeResult("example.com", nil, now, "full", nil)

	if result.Valid {
		t.Error("expected invalid when certificate is nil")
	}
	if result.Domain != "example.com" {
		t.Errorf("Domain = %q, want %q", result.Domain, "example.com")
	}
}

func TestBuildProbeResultEmptyValidation(t *testing.T) {
	now := time.Now()
	cert := &x509.Certificate{
		Issuer:       pkix.Name{CommonName: "Test CA"},
		Subject:      pkix.Name{CommonName: "example.com"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(30 * 24 * time.Hour),
		SerialNumber: big.NewInt(12345),
	}

	// Empty validation should default to "time_only"
	result := buildProbeResult("example.com", cert, now, "", nil)
	if result.Validation != "time_only" {
		t.Errorf("Validation = %q, want %q (should default)", result.Validation, "time_only")
	}

	// Whitespace-only validation should also default
	result = buildProbeResult("example.com", cert, now, "   ", nil)
	if result.Validation != "time_only" {
		t.Errorf("Validation = %q, want %q (should default for whitespace)", result.Validation, "time_only")
	}
}

func TestBuildProbeResultDaysRemaining(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name              string
		notAfter          time.Time
		wantDaysRemaining int
	}{
		{
			name:              "30 days remaining",
			notAfter:          now.Add(30 * 24 * time.Hour),
			wantDaysRemaining: 30,
		},
		{
			name:              "1 day remaining",
			notAfter:          now.Add(24 * time.Hour),
			wantDaysRemaining: 1,
		},
		{
			name:              "expired (negative days)",
			notAfter:          now.Add(-24 * time.Hour),
			wantDaysRemaining: 0, // Should be clamped to 0
		},
		{
			name:              "90 days remaining",
			notAfter:          now.Add(90 * 24 * time.Hour),
			wantDaysRemaining: 90,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cert := &x509.Certificate{
				Issuer:       pkix.Name{CommonName: "Test CA"},
				Subject:      pkix.Name{CommonName: "example.com"},
				NotBefore:    now.Add(-time.Hour),
				NotAfter:     tt.notAfter,
				SerialNumber: big.NewInt(12345),
			}
			result := buildProbeResult("example.com", cert, now, "time_only", nil)
			if result.DaysRemaining != tt.wantDaysRemaining {
				t.Errorf("DaysRemaining = %d, want %d", result.DaysRemaining, tt.wantDaysRemaining)
			}
		})
	}
}

func TestBuildProbeResultCertificateFields(t *testing.T) {
	now := time.Now()
	cert := &x509.Certificate{
		Issuer:       pkix.Name{CommonName: "Let's Encrypt Authority X3"},
		Subject:      pkix.Name{CommonName: "example.com", Organization: []string{"Example Inc"}},
		NotBefore:    now.Add(-24 * time.Hour),
		NotAfter:     now.Add(90 * 24 * time.Hour),
		SerialNumber: big.NewInt(123456789),
		DNSNames:     []string{"example.com", "www.example.com", "api.example.com"},
	}

	result := buildProbeResult("example.com", cert, now, "time_only", nil)

	if result.Issuer != "Let's Encrypt Authority X3" {
		t.Errorf("Issuer = %q, want %q", result.Issuer, "Let's Encrypt Authority X3")
	}
	if result.Domain != "example.com" {
		t.Errorf("Domain = %q, want %q", result.Domain, "example.com")
	}
	if len(result.SANs) != 3 {
		t.Errorf("SANs length = %d, want 3", len(result.SANs))
	}
	if result.NotBefore == "" {
		t.Error("NotBefore should not be empty")
	}
	if result.NotAfter == "" {
		t.Error("NotAfter should not be empty")
	}
	if result.SerialNumber == "" {
		t.Error("SerialNumber should not be empty")
	}
}

func TestBuildProbeResultFullValidationSuccess(t *testing.T) {
	now := time.Now()
	cert := &x509.Certificate{
		Issuer:       pkix.Name{CommonName: "Test CA"},
		Subject:      pkix.Name{CommonName: "example.com"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(30 * 24 * time.Hour),
		SerialNumber: big.NewInt(12345),
	}

	// Full validation with no error should be valid
	result := buildProbeResult("example.com", cert, now, "full", nil)
	if !result.Valid {
		t.Error("expected valid when full validation passes with no error")
	}
	if result.Validation != "full" {
		t.Errorf("Validation = %q, want %q", result.Validation, "full")
	}
	if result.ValidationError != "" {
		t.Errorf("ValidationError = %q, want empty", result.ValidationError)
	}
}

func TestBuildProbeResultSANsCopy(t *testing.T) {
	now := time.Now()
	dnsNames := []string{"example.com", "www.example.com"}
	cert := &x509.Certificate{
		Issuer:       pkix.Name{CommonName: "Test CA"},
		Subject:      pkix.Name{CommonName: "example.com"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(30 * 24 * time.Hour),
		SerialNumber: big.NewInt(12345),
		DNSNames:     dnsNames,
	}

	result := buildProbeResult("example.com", cert, now, "time_only", nil)

	// Modify original slice - should not affect result
	dnsNames[0] = "modified.com"

	if result.SANs[0] == "modified.com" {
		t.Error("SANs should be a copy, not a reference to original DNSNames")
	}
}

func TestVerifyCertificateChainEmptyCerts(t *testing.T) {
	now := time.Now()
	err := verifyCertificateChain("example.com", []*x509.Certificate{}, now)
	if err == nil {
		t.Error("expected error for empty certificate slice")
	}
}

func TestValidationErrorString(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "nil error returns empty",
			err:  nil,
			want: "",
		},
		{
			name: "error returns message",
			err:  errors.New("certificate expired"),
			want: "certificate expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validationErrorString(tt.err)
			if got != tt.want {
				t.Errorf("validationErrorString(%v) = %q, want %q", tt.err, got, tt.want)
			}
		})
	}
}
