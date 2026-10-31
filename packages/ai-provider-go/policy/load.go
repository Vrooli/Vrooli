package policy

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PathOptions describes a provider's policy-file lookup conventions.
type PathOptions struct {
	// ProviderName is the resource id, e.g. "openrouter".
	ProviderName string
	// PolicyEnv overrides the policy path, e.g. OPENROUTER_MODEL_POLICY_PATH.
	PolicyEnv string
	// SourceEnv is an optional resource-root override, e.g. OLLAMA_CLI_SOURCE_ROOT.
	SourceEnv string
	// Validation selects the admission profile applied on load.
	Validation ValidationOptions
}

// LoadFile reads, parses, and validates a catalog.
func LoadFile(path string, opts ValidationOptions) (Catalog, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Catalog{}, err
	}
	return Parse(raw, opts)
}

// Parse parses and validates catalog bytes.
func Parse(raw []byte, opts ValidationOptions) (Catalog, error) {
	var catalog Catalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		return Catalog{}, fmt.Errorf("parse model policy: %w", err)
	}
	if err := Validate(catalog, opts); err != nil {
		return Catalog{}, err
	}
	return catalog, nil
}

// LoadDefaultFile resolves a provider's policy path and loads it.
func LoadDefaultFile(getenv func(string) string, opts PathOptions) (Catalog, string, error) {
	path, err := DefaultPolicyPath(getenv, opts)
	if err != nil {
		return Catalog{}, "", err
	}
	catalog, err := LoadFile(path, opts.Validation)
	if err != nil {
		return Catalog{}, "", err
	}
	return catalog, path, nil
}

// DefaultPolicyPath resolves the provider's model-policy.json location.
func DefaultPolicyPath(getenv func(string) string, opts PathOptions) (string, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	provider := strings.TrimSpace(opts.ProviderName)
	if provider == "" {
		return "", errors.New("provider name is required")
	}
	if env := strings.TrimSpace(opts.PolicyEnv); env != "" {
		if value := strings.TrimSpace(getenv(env)); value != "" {
			return value, nil
		}
	}
	for _, key := range []string{"RESOURCE_ROOT", strings.TrimSpace(opts.SourceEnv)} {
		if key == "" {
			continue
		}
		if value := strings.TrimSpace(getenv(key)); value != "" {
			candidate := filepath.Join(value, "model-policy.json")
			if fileExists(candidate) {
				return candidate, nil
			}
		}
	}
	for _, key := range []string{"VROOLI_SOURCE_ROOT", "VROOLI_CLI_SOURCE_ROOT"} {
		if value := strings.TrimSpace(getenv(key)); value != "" {
			candidate := filepath.Join(value, "resources", provider, "model-policy.json")
			if fileExists(candidate) {
				return candidate, nil
			}
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		for _, rel := range []string{
			filepath.Join("resources", provider, "model-policy.json"),
			filepath.Join("..", "..", "..", "model-policy.json"),
		} {
			candidate := filepath.Clean(filepath.Join(dir, rel))
			if fileExists(candidate) {
				return candidate, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", fmt.Errorf("locate %s model policy: set %s, RESOURCE_ROOT, or VROOLI_SOURCE_ROOT", provider, opts.PolicyEnv)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
