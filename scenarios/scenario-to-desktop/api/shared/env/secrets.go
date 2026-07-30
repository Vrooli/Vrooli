package env

import (
	"os"
	"strings"
)

// SecretResolution describes where a secret value was resolved from.
type SecretResolution struct {
	Value      string
	Source     string
	SourcePath string
}

// ResolveSecret reads a process-scoped value injected by the deployment host.
// Durable credentials remain owned by the Vrooli control plane.
func ResolveSecret(key string) string {
	return ResolveSecretWithSource(key).Value
}

func ResolveSecretWithSource(key string) SecretResolution {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return SecretResolution{Source: "missing"}
	}
	return SecretResolution{Value: value, Source: "process"}
}
