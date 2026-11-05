package openrouter

import (
	"errors"
	"os"
	"strings"
)

const (
	apiKeyEnv  = "OPENROUTER_API_KEY" // #nosec G101 -- environment variable name, not a hardcoded credential.
	baseURLEnv = "OPENROUTER_BASE_URL"
)

var ErrAPIKeyMissing = errors.New("OPENROUTER_API_KEY not configured")

type Config struct {
	APIKey  string
	BaseURL string
}

type Status struct {
	Configured bool
	Error      string
}

func ResolveConfig() (Config, error) {
	apiKey := strings.TrimSpace(os.Getenv(apiKeyEnv))
	if apiKey == "" {
		return Config{}, ErrAPIKeyMissing
	}
	return Config{APIKey: apiKey, BaseURL: strings.TrimSpace(os.Getenv(baseURLEnv))}, nil
}

func CheckConfig() Status {
	if _, err := ResolveConfig(); err != nil {
		return Status{Configured: false, Error: err.Error()}
	}
	return Status{Configured: true}
}
