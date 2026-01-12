package tlsinfo

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"testing"
	"time"
)

func TestBuildProbeResultFullValidationError(t *testing.T) {
	now := time.Now()
	cert := &x509.Certificate{
		Issuer:    pkix.Name{CommonName: "Test CA"},
		Subject:   pkix.Name{CommonName: "example.com"},
		NotBefore: now.Add(-time.Hour),
		NotAfter:  now.Add(time.Hour),
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
