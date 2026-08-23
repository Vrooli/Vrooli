// Package bootstrap composes every singleton the audio-tools API needs
// (env, sqlite, stores, chains, modules) so main.go stays a 3-call shell.
package bootstrap

import (
	"os"
	"strings"
	"time"

	intsumm "audio-tools/internal/summarize"
)

// Bool parses a boolean env var with conventional truthy/falsy spellings.
func Bool(key string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "":
		return def
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	return def
}

// Or returns the env var value if set (after trim), else def.
func Or(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// Optional returns a trimmed optional environment value. The boolean is
// intentionally discarded because callers use an empty value as "not
// configured" rather than treating it as a startup failure.
func Optional(key string) string {
	v, _ := os.LookupEnv(key)
	return strings.TrimSpace(v)
}

// ResourceURL lets scenario-specific overrides win while keeping the resource
// runtime's configured endpoint authoritative. Resource URLs are injected by
// the control plane, so this package does not duplicate local host defaults.
func ResourceURL(overrideKey, resourceKey string) string {
	return Or(overrideKey, Optional(resourceKey))
}

// Duration parses a time.Duration env var, falling back to def on
// empty or parse error.
func Duration(key string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

// Env captures every environment variable the audio-tools API consults
// at boot. One Load() call replaces the previously scattered os.Getenv
// reads in main.go.
type Env struct {
	WhisperURL            string
	KyutaiURL             string
	SherpaURL             string
	OllamaURL             string
	OpenRouterURL         string
	OpenRouterAPIKey      string
	LPBSBaseURL           string
	LPBSAppBundleKey      string
	SummarizeDefaultModel string

	AvailTTLBYOK   time.Duration
	AvailTTLVrooli time.Duration

	EnableBYOK   bool
	EnableVrooli bool
	EnableLocal  bool
	DBKeyPath    string
}

// Load reads all audio-tools env vars at process start. No side effects.
func Load() Env {
	return Env{
		WhisperURL:            ResourceURL("AUDIO_WHISPER_URL", "WHISPER_URL"),
		KyutaiURL:             ResourceURL("AUDIO_KYUTAI_URL", "KYUTAI_URL"),
		SherpaURL:             ResourceURL("AUDIO_SHERPA_URL", "SHERPA_ONNX_URL"),
		OllamaURL:             ResourceURL("AUDIO_OLLAMA_URL", "OLLAMA_URL"),
		OpenRouterURL:         Or("AUDIO_OPENROUTER_URL", "https://openrouter.ai"),
		OpenRouterAPIKey:      Optional("OPENROUTER_API_KEY"),
		LPBSBaseURL:           Or("AUDIO_LPBS_BASE_URL", ""),
		LPBSAppBundleKey:      Or("AUDIO_LPBS_APP_BUNDLE_KEY", ""),
		SummarizeDefaultModel: intsumm.CoerceUnsafeStoredModel(Or("AUDIO_SUMMARIZE_DEFAULT_MODEL", intsumm.DefaultSummarizeModel), nil).Model,
		AvailTTLBYOK:          Duration("AUDIO_AVAIL_TTL_BYOK", 5*time.Minute),
		AvailTTLVrooli:        Duration("AUDIO_AVAIL_TTL_VROOLI", 30*time.Second),
		EnableBYOK:            Bool("AUDIO_AI_ENABLE_BYOK", true),
		EnableVrooli:          Bool("AUDIO_AI_ENABLE_VROOLI", false),
		EnableLocal:           Bool("AUDIO_AI_ENABLE_LOCAL", true),
		DBKeyPath:             Or("AUDIO_TOOLS_DB_KEY_PATH", ""),
	}
}
