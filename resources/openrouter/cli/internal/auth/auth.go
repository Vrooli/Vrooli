package auth

import (
	"context"
	"errors"
	"os"
	"strings"
)

const (
	APIKeyEnv = "OPENROUTER_API_KEY"
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

// Resolver accepts only the process-scoped credential injected by the control
// plane. Vault and user files are not credential authorities; runtime injection
// is the sole way a value reaches this CLI.
type Resolver struct {
	LookupEnv func(string) (string, bool)
}

// NewResolver returns a Resolver with standard process and environment hooks.
func NewResolver() Resolver {
	return Resolver{
		LookupEnv: os.LookupEnv,
	}
}

// Resolve returns the ephemeral credential injected into this process. Context
// remains part of the method contract for provider callers, but this resolver
// intentionally makes no subprocess or filesystem credential request.
func (r Resolver) Resolve(ctx context.Context) (Credentials, error) {
	_ = ctx
	if creds, ok := r.resolveFromEnv(); ok {
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
		Source: "ephemeral-injection",
	}
	return creds, creds.Valid()
}
