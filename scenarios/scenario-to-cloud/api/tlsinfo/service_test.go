package tlsinfo

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"
)

func TestBuildProbeResult(t *testing.T) {
	now := time.Date(2025, 1, 5, 12, 0, 0, 0, time.UTC)
	cert := &x509.Certificate{
		Issuer:       pkix.Name{CommonName: "Test CA"},
		Subject:      pkix.Name{CommonName: "example.com"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(48 * time.Hour),
		SerialNumber: big.NewInt(0x1a2b),
		DNSNames:     []string{"example.com", "www.example.com"},
	}

	result := buildProbeResult("example.com", cert, now)

	if !result.Valid {
		t.Fatalf("expected cert to be valid")
	}
	if result.Issuer != "Test CA" {
		t.Fatalf("expected issuer, got %q", result.Issuer)
	}
	if result.SerialNumber == "" {
		t.Fatalf("expected serial number")
	}
	if result.DaysRemaining == 0 {
		t.Fatalf("expected days remaining to be set")
	}
	if len(result.SANs) != 2 {
		t.Fatalf("expected SANs, got %v", result.SANs)
	}
}
