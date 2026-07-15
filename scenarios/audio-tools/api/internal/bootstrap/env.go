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
	SpeakerURL            string
	KokoroURL             string
	OllamaURL             string
	LPBSBaseURL           string
	LPBSAppBundleKey      string
	SummarizeDefaultModel string

	AvailTTLBYOK   time.Duration
	AvailTTLVrooli time.Duration

	EnableBYOK   bool
	EnableVrooli bool
	EnableLocal  bool
	// EnableStreamTestFaults is intentionally opt-in. It enables only
	// deterministic, request-scoped WebSocket faults whose request also carries
	// the test-mode header; it must never be enabled for ordinary deployments.
	EnableStreamTestFaults bool

	DBKeyPath  string
	SqlitePath string
	SqliteDB   string
}

// Load reads all audio-tools env vars at process start. No side effects.
func Load() Env {
	return Env{
		WhisperURL:             Or("AUDIO_WHISPER_URL", "http://localhost:8090"),
		KyutaiURL:              Or("AUDIO_KYUTAI_URL", "http://localhost:8094"),
		SpeakerURL:             Or("AUDIO_SPEAKER_URL", "http://localhost:11452"),
		KokoroURL:              Or("AUDIO_KOKORO_URL", "http://localhost:8880"),
		OllamaURL:              Or("AUDIO_OLLAMA_URL", "http://localhost:11434"),
		LPBSBaseURL:            Or("AUDIO_LPBS_BASE_URL", ""),
		LPBSAppBundleKey:       Or("AUDIO_LPBS_APP_BUNDLE_KEY", ""),
		SummarizeDefaultModel:  intsumm.CoerceUnsafeStoredModel(Or("AUDIO_SUMMARIZE_DEFAULT_MODEL", intsumm.DefaultSummarizeModel), nil).Model,
		AvailTTLBYOK:           Duration("AUDIO_AVAIL_TTL_BYOK", 5*time.Minute),
		AvailTTLVrooli:         Duration("AUDIO_AVAIL_TTL_VROOLI", 30*time.Second),
		EnableBYOK:             Bool("AUDIO_AI_ENABLE_BYOK", true),
		EnableVrooli:           Bool("AUDIO_AI_ENABLE_VROOLI", false),
		EnableLocal:            Bool("AUDIO_AI_ENABLE_LOCAL", true),
		EnableStreamTestFaults: Bool("AUDIO_TOOLS_ENABLE_STREAM_TEST_FAULTS", false),
		DBKeyPath:              Or("AUDIO_TOOLS_DB_KEY_PATH", ""),
		SqlitePath:             strings.TrimSpace(os.Getenv("SQLITE_PATH")),
		SqliteDB:               strings.TrimSpace(os.Getenv("SQLITE_DB")),
	}
}
