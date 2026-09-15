package main

import (
	"encoding/base64"
	"strings"
	"testing"
)

// [REQ] The secret has no external issuer, so the scenario must be able to
// produce one rather than asking an operator for a value they cannot obtain.
func TestGenerateJWTSecretProducesADistinctFullLengthValue(t *testing.T) {
	first, err := generateJWTSecret()
	if err != nil {
		t.Fatal(err)
	}
	second, err := generateJWTSecret()
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("two generated secrets are identical")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(first)
	if err != nil {
		t.Fatalf("generated secret is not decodable: %v", err)
	}
	if len(decoded) != jwtSecretBytes {
		t.Fatalf("generated secret is %d bytes, want %d", len(decoded), jwtSecretBytes)
	}
	if strings.TrimSpace(first) != first || first == "" {
		t.Fatalf("generated secret is not a clean token: %q", first)
	}
}

// The resolver is the only path to the secret, and it must not read the
// environment: a value there is readable at /proc/<pid>/environ and inherited
// by every subprocess this scenario spawns.
func TestResolveJWTSecretIgnoresTheEnvironment(t *testing.T) {
	t.Setenv("JWT_SECRET", "value-from-the-environment")
	previous := jwtSecretResolver
	jwtSecretResolver = func() (string, error) { return "value-from-the-authority", nil }
	t.Cleanup(func() { jwtSecretResolver = previous })

	secret, err := resolveJWTSecret()
	if err != nil {
		t.Fatal(err)
	}
	if secret != "value-from-the-authority" {
		t.Fatalf("secret = %q, want the authority's value", secret)
	}
}
