package errors

import (
	"strings"
	"testing"
)

func TestRedactRemovesCredentialsFromHumanReadableErrors(t *testing.T) {
	accessKey := "AKIA" + "1234567890ABCDEF"
	input := "Bearer abc.def-123 api_key: exposed password=unsafe " + accessKey
	got := Redact(input)
	for _, sensitive := range []string{"abc.def-123", "exposed", "unsafe", accessKey} {
		if strings.Contains(got, sensitive) {
			t.Fatalf("Redact leaked %q in %q", sensitive, got)
		}
	}
	if !strings.Contains(got, "Bearer REDACTED") || !strings.Contains(got, "api_key=REDACTED") {
		t.Fatalf("Redact = %q", got)
	}
}

func TestRedactNeutralizesPresignedURLQueryValues(t *testing.T) {
	input := "upload failed: https://bucket.example/file?X-Amz-Signature=signature&AWSAccessKeyId=access&token=token-value&safe=value"
	got := Redact(input)
	for _, sensitive := range []string{"signature", "access", "token-value"} {
		if strings.Contains(got, sensitive) {
			t.Fatalf("Redact leaked %q in %q", sensitive, got)
		}
	}
	if !strings.Contains(got, "REDACTED") {
		t.Fatalf("Redact = %q", got)
	}
}

func TestRedactHandlesEmptyAndMalformedURLText(t *testing.T) {
	if got := Redact("  \t\n"); got != "" {
		t.Fatalf("empty Redact = %q", got)
	}
	if got := Redact("https://[bad X-Amz-Signature=still-secret"); strings.Contains(got, "still-secret") {
		t.Fatalf("malformed Redact leaked secret: %q", got)
	}
}
