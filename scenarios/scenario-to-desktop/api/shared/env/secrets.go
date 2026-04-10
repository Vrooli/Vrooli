package env

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// SecretResolution describes where a secret value was resolved from.
type SecretResolution struct {
	Value      string
	Source     string
	SourcePath string
}

// ResolveSecret resolves a secret using the standard order:
// 1) environment variable, 2) nearest .vrooli/secrets.json, 3) empty string.
func ResolveSecret(key string) string {
	return ResolveSecretWithSource(key).Value
}

// ResolveSecretWithSource resolves a secret and reports where it came from.
func ResolveSecretWithSource(key string) SecretResolution {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return SecretResolution{
			Value:  value,
			Source: "env",
		}
	}

	if file := findSecretsFile(); file != "" {
		if value := readSecretFromJSON(file, key); value != "" {
			return SecretResolution{
				Value:      value,
				Source:     "file",
				SourcePath: file,
			}
		}
	}
	return SecretResolution{
		Source: "missing",
	}
}

func findSecretsFile() string {
	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" {
		candidate := filepath.Join(root, ".vrooli", "secrets.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := cwd
	for {
		candidate := filepath.Join(dir, ".vrooli", "secrets.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func readSecretFromJSON(path, key string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return ""
	}
	if value, ok := raw[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}
