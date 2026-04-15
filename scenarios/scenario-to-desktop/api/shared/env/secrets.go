package env

import (
	"os"
	"strings"

	apisecrets "github.com/vrooli/api-core/secrets"
)

// SecretResolution describes where a secret value was resolved from.
type SecretResolution struct {
	Value      string
	Source     string
	SourcePath string
}

// ResolveSecret resolves a secret using the standard order:
// 1) environment variable, 2) ~/.vrooli/secrets.json, 3) empty string.
func ResolveSecret(key string) string {
	return ResolveSecretWithSource(key).Value
}

// ResolveSecretWithSource resolves a secret and reports where it came from.
func ResolveSecretWithSource(key string) SecretResolution {
	store, err := apisecrets.NewUserStore(apisecrets.Config{
		EnvLookup: os.Getenv,
	})
	if err != nil {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return SecretResolution{
				Value:  value,
				Source: apisecrets.SourceEnv,
			}
		}
		return SecretResolution{Source: apisecrets.SourceMissing}
	}
	resolved, err := store.Resolve(key)
	if err != nil {
		return SecretResolution{Source: apisecrets.SourceMissing}
	}
	return SecretResolution{
		Value:      resolved.Value,
		Source:     resolved.Source,
		SourcePath: resolved.SourcePath,
	}
}
