package config

import (
	"strings"
	"testing"
)

func TestReadSecretTrimsOnlyInputFraming(t *testing.T) {
	got, err := readSecret(strings.NewReader("  api-token\n"), "Cloudflare API token")
	if err != nil {
		t.Fatalf("readSecret() error = %v", err)
	}
	if got != "api-token" {
		t.Fatalf("readSecret() = %q, want %q", got, "api-token")
	}
}

func TestReadSecretRejectsEmptyAndOversizedInput(t *testing.T) {
	if _, err := readSecret(strings.NewReader("\n"), "Cloudflare API token"); err == nil {
		t.Fatal("readSecret() accepted empty input")
	}
	if _, err := readSecret(strings.NewReader(strings.Repeat("x", 4097)), "Cloudflare API token"); err == nil {
		t.Fatal("readSecret() accepted oversized input")
	}
}
