package env

import (
	"os"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// SecretResolution describes where a secret value was resolved from.
type SecretResolution struct {
	Value      string
	Source     string
	SourcePath string
}

// requireCredential is a narrow seam for the authority lookup. Keeping the
// seam injectable makes the authority-first contract testable without
// changing the production resolution path or touching the live credential
// store.
var requireCredential = func(identity credentialauthority.Identity, field string) (string, error) {
	authority, err := credentialauthority.Default()
	if err != nil {
		return "", err
	}
	return authority.Require(identity, field)
}

// processBackedSecrets are the credentials whose final consumer is an
// external tool. Their descriptors intentionally retain an env field; all
// other durable credentials must resolve from the authority.
var processBackedSecrets = map[string]struct{}{
	"CSC_KEY_PASSWORD":            {},
	"APPLE_ID":                    {},
	"APPLE_APP_SPECIFIC_PASSWORD": {},
	"APPLE_API_KEY_FILE":          {},
}

// ResolveSecret reads a process-scoped value injected by the deployment host.
// Durable credentials remain owned by the Vrooli control plane.
func ResolveSecret(key string) string {
	return ResolveSecretWithSource(key).Value
}

func ResolveSecretWithSource(key string) SecretResolution {
	if key == "LPBS_SERVICE_SECRET" && os.Getenv("S2D_TEST_CREDENTIAL_FALLBACK") != "1" {
		identity, err := credentialauthority.ParseIdentity("vrooli/landing-page-business-suite")
		if err != nil {
			return SecretResolution{Source: "invalid-identity"}
		}
		value, err := requireCredential(identity, "service-secret")
		if err != nil {
			return SecretResolution{Source: "authority-unavailable"}
		}
		return SecretResolution{Value: strings.TrimSpace(value), Source: "authority"}
	}
	if _, allowed := processBackedSecrets[key]; !allowed && os.Getenv("S2D_TEST_CREDENTIAL_FALLBACK") != "1" {
		return SecretResolution{Source: "missing"}
	}
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return SecretResolution{Source: "missing"}
	}
	return SecretResolution{Value: value, Source: "process"}
}
