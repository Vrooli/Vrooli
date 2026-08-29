// Package config owns the typed runtime settings used by Ollama health and
// gateway commands. It replaces the retired shell defaults surface.
package config

import (
	"os"
	"strings"

	"github.com/vrooli/vrooli/internal/tuning"
	"time"
)

type Runtime struct {
	BaseURL          string
	ReadinessTimeout time.Duration
	RequireModel     bool
}

func FromEnv(getenv func(string) string) Runtime {
	baseURL := strings.TrimRight(strings.TrimSpace(getenv("OLLAMA_BASE_URL")), "/")
	if baseURL == "" {
		host := strings.TrimSpace(getenv("OLLAMA_HOST"))
		if host == "" {
			host = "127.0.0.1:11434"
		} else if !strings.Contains(host, ":") {
			host += ":11434"
		}
		if !strings.Contains(host, "://") {
			baseURL = "http://" + host
		} else {
			baseURL = host
		}
	}
	return Runtime{BaseURL: baseURL, ReadinessTimeout: tuning.ServiceHealthTimeout(), RequireModel: true}
}

func Default() Runtime { return FromEnv(os.Getenv) }
