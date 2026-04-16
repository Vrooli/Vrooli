package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	APIKeyEnv = "OPENROUTER_API_KEY"

	defaultVaultCommand = "resource-vault"
	defaultSecretPath   = "resources/openrouter/api/main"
	defaultSecretKey    = "value"
	legacySecretPath    = "vrooli/openrouter"
	legacySecretKey     = "api_key"
)

var ErrCredentialsNotConfigured = errors.New("openrouter credentials not configured")

// Credentials holds the API key required for OpenRouter requests.
type Credentials struct {
	APIKey string
	Source string
}

// Valid reports whether a usable OpenRouter API key is configured.
func (c Credentials) Valid() bool {
	return strings.TrimSpace(c.APIKey) != ""
}

// RedactedAPIKey returns a log-safe preview of the configured API key.
func (c Credentials) RedactedAPIKey() string {
	key := strings.TrimSpace(c.APIKey)
	switch {
	case key == "":
		return ""
	case len(key) <= 8:
		return "********"
	default:
		return key[:6] + "..." + key[len(key)-4:]
	}
}

// Resolver owns OpenRouter-specific credential lookup across Vault, env vars,
// and optional compatibility files.
type Resolver struct {
	LookupEnv    func(string) (string, bool)
	LookPathFunc func(string) (string, error)
	RunCommand   func(context.Context, string, ...string) (string, error)

	VaultCommand        string
	SecretPath          string
	SecretKey           string
	LegacySecretPath    string
	LegacySecretKey     string
	CredentialsFilePath string
}

// NewResolver returns a Resolver with standard process and environment hooks.
func NewResolver() Resolver {
	return Resolver{
		LookupEnv:        os.LookupEnv,
		LookPathFunc:     exec.LookPath,
		RunCommand:       runCommand,
		VaultCommand:     defaultVaultCommand,
		SecretPath:       defaultSecretPath,
		SecretKey:        defaultSecretKey,
		LegacySecretPath: legacySecretPath,
		LegacySecretKey:  legacySecretKey,
	}
}

// Resolve returns the first complete credential set found from Vault, the
// process environment, or the optional compatibility credentials file.
func (r Resolver) Resolve(ctx context.Context) (Credentials, error) {
	if creds, ok := r.resolveFromVault(ctx); ok {
		return creds, nil
	}
	if creds, ok := r.resolveFromEnv(); ok {
		return creds, nil
	}
	if creds, ok := r.resolveFromCredentialsFile(); ok {
		return creds, nil
	}
	return Credentials{}, ErrCredentialsNotConfigured
}

func (r Resolver) resolveFromEnv() (Credentials, bool) {
	lookup := r.LookupEnv
	if lookup == nil {
		lookup = os.LookupEnv
	}
	value, _ := lookup(APIKeyEnv)
	creds := Credentials{
		APIKey: strings.TrimSpace(value),
		Source: "env",
	}
	return creds, creds.Valid()
}

func (r Resolver) resolveFromVault(ctx context.Context) (Credentials, bool) {
	command := strings.TrimSpace(r.VaultCommand)
	if command == "" {
		command = defaultVaultCommand
	}

	lookPath := r.LookPathFunc
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	if _, err := lookPath(command); err != nil {
		return Credentials{}, false
	}

	run := r.RunCommand
	if run == nil {
		run = runCommand
	}

	candidates := []struct {
		path   string
		key    string
		source string
	}{
		{path: firstNonEmpty(r.SecretPath, defaultSecretPath), key: firstNonEmpty(r.SecretKey, defaultSecretKey), source: "vault"},
		{path: firstNonEmpty(r.LegacySecretPath, legacySecretPath), key: firstNonEmpty(r.LegacySecretKey, legacySecretKey), source: "vault-legacy"},
	}

	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.path) == "" || strings.TrimSpace(candidate.key) == "" {
			continue
		}
		value, err := run(ctx, command, "content", "get", "--path", candidate.path, "--key", candidate.key, "--format", "raw")
		if err != nil {
			continue
		}
		creds := Credentials{
			APIKey: strings.TrimSpace(value),
			Source: candidate.source,
		}
		if creds.Valid() {
			return creds, true
		}
	}

	return Credentials{}, false
}

func (r Resolver) resolveFromCredentialsFile() (Credentials, bool) {
	path := strings.TrimSpace(r.CredentialsFilePath)
	if path == "" {
		return Credentials{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Credentials{}, false
	}
	var payload struct {
		Data struct {
			APIKey string `json:"apiKey"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return Credentials{}, false
	}
	creds := Credentials{
		APIKey: strings.TrimSpace(payload.Data.APIKey),
		Source: "file",
	}
	return creds, creds.Valid()
}

func runCommand(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(output)), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
