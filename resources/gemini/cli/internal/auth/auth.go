package auth

import (
	"context"
	"errors"
	"os"
	"strings"
)

const (
	APIKeyEnv = "GEMINI_API_KEY"
)

var ErrCredentialsNotConfigured = errors.New("gemini credentials not configured")

// Credentials holds the API key required for Gemini requests.
type Credentials struct {
	APIKey string
	Source string
}

// Valid reports whether a usable Gemini API key is configured.
func (c Credentials) Valid() bool {
	return strings.TrimSpace(c.APIKey) != ""
}

// RedactedAPIKey returns a log-safe preview of the configured API key.
func (c Credentials) RedactedAPIKey() string {
	token := strings.TrimSpace(c.APIKey)
	switch {
	case token == "":
		return ""
	case len(token) <= 8:
		return "********"
	default:
		return token[:4] + "..." + token[len(token)-4:]
	}
}

// Resolver owns Gemini-specific credential lookup from the process contract.
// The control plane resolves the canonical credential-authority reference and
// injects GEMINI_API_KEY only into this process before it starts.
type Resolver struct {
	LookupEnv func(string) (string, bool)
}

// NewResolver returns a Resolver with standard process and environment hooks.
func NewResolver() Resolver {
	return Resolver{
		LookupEnv: os.LookupEnv,
	}
}

// Resolve returns the credential injected into the process environment by the
// canonical credential authority.
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
		Source: "env",
	}
	return creds, creds.Valid()
}
