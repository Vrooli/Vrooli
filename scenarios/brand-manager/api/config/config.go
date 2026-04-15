// Package config provides centralized, tunable configuration for brand-manager.
//
// Every lever has a sane default, can be overridden via environment variable,
// and is validated at load time. The Config struct is the single source of
// truth for runtime behavior across the API, database, and contrast subsystems.
//
// DOC: docs/reference/configuration.md
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/vrooli/api-core/storage"
)

// Config holds all tunable levers for the brand-manager service.
type Config struct {
	// --- Database ---

	// SQLitePath is the filesystem path for the SQLite database file.
	// Env: BM_SQLITE_PATH, SQLITE_PATH, SQLITE_DB
	// Default: ~/.vrooli/brand-manager/brand-manager.db
	SQLitePath string

	// BusyTimeoutMS is how long SQLite waits before returning SQLITE_BUSY (ms).
	// Higher values tolerate more write contention; lower values fail faster.
	// Env: BM_BUSY_TIMEOUT_MS
	// Default: 10000 (10 seconds)
	BusyTimeoutMS int

	// CacheSizeKB is the SQLite in-memory page cache size in KB.
	// Negative values in pragma = KB. Higher = more memory, faster reads.
	// Env: BM_CACHE_SIZE_KB
	// Default: 2000 (~2 MB)
	CacheSizeKB int

	// MaxOpenConns limits SQLite concurrent connections.
	// SQLite supports only one writer; changing this above 1 is not recommended.
	// Default: 1
	MaxOpenConns int

	// --- API ---

	// APIVersion is reported in the /health endpoint.
	// Default: "1.0.0"
	APIVersion string

	// DefaultListLimit caps the number of items returned by list endpoints
	// when the caller does not provide an explicit ?limit= parameter.
	// Env: BM_DEFAULT_LIST_LIMIT
	// Default: 100
	DefaultListLimit int

	// MaxListLimit is the absolute ceiling for ?limit= values.
	// Requests exceeding this are silently capped.
	// Env: BM_MAX_LIST_LIMIT
	// Default: 1000
	MaxListLimit int

	// --- WCAG Contrast ---

	// ContrastAANormal is the minimum contrast ratio for WCAG AA normal text.
	// Per WCAG 2.1 spec this is 4.5:1. Override only for AAA testing (7.0).
	// Env: BM_CONTRAST_AA_NORMAL
	// Default: 4.5
	ContrastAANormal float64

	// ContrastAALarge is the minimum contrast ratio for WCAG AA large text.
	// Per WCAG 2.1 spec this is 3.0:1.
	// Env: BM_CONTRAST_AA_LARGE
	// Default: 3.0
	ContrastAALarge float64

	// ContrastPrecision is the number of decimal places for contrast ratio display.
	// Env: BM_CONTRAST_PRECISION
	// Default: 2
	ContrastPrecision int

	// --- Assets ---

	// AssetBasePath is the root directory for brand asset files.
	// Env: BM_ASSET_BASE_PATH
	// Default: ~/.vrooli/brand-manager/assets
	AssetBasePath string

	// ScenariosDir is the root directory for Vrooli scenarios.
	// Used by the inline validation scanner.
	// Env: BM_SCENARIOS_DIR, SCENARIOS_DIR
	// Default: ./scenarios (relative to working directory)
	ScenariosDir string

	// --- AI Provider ---

	// OllamaURL is the base URL for a local Ollama instance.
	// Env: OLLAMA_URL, OLLAMA_BASE_URL
	OllamaURL string

	// OllamaModel overrides the default Ollama model for text generation.
	// Env: BM_OLLAMA_MODEL
	OllamaModel string

	// OpenRouterAPIKey enables the OpenRouter cloud provider.
	// Env: OPENROUTER_API_KEY
	OpenRouterAPIKey string

	// OpenRouterTextModel overrides the default OpenRouter text model.
	// Env: BM_OPENROUTER_TEXT_MODEL
	OpenRouterTextModel string

	// OpenRouterImageModel overrides the default OpenRouter image model.
	// Env: BM_OPENROUTER_IMAGE_MODEL
	OpenRouterImageModel string
}

// Default returns a Config with all default values applied.
func Default() Config {
	return Config{
		SQLitePath:        defaultDBPath(),
		BusyTimeoutMS:     10000,
		CacheSizeKB:       2000,
		MaxOpenConns:      1,
		APIVersion:        "1.0.0",
		DefaultListLimit:  100,
		MaxListLimit:      1000,
		ContrastAANormal:  4.5,
		ContrastAALarge:   3.0,
		ContrastPrecision: 2,
		AssetBasePath:     defaultAssetPath(),
		ScenariosDir:      defaultScenariosDir(),
	}
}

// Load returns a Config populated from environment variables, falling back to defaults.
func Load() Config {
	cfg := Default()

	// Database path: resolution chain
	for _, key := range []string{"BM_SQLITE_PATH", "SQLITE_PATH", "SQLITE_DB"} {
		if v := os.Getenv(key); v != "" {
			cfg.SQLitePath = v
			break
		}
	}

	cfg.BusyTimeoutMS = envInt("BM_BUSY_TIMEOUT_MS", cfg.BusyTimeoutMS)
	cfg.CacheSizeKB = envInt("BM_CACHE_SIZE_KB", cfg.CacheSizeKB)
	cfg.DefaultListLimit = envInt("BM_DEFAULT_LIST_LIMIT", cfg.DefaultListLimit)
	cfg.MaxListLimit = envInt("BM_MAX_LIST_LIMIT", cfg.MaxListLimit)
	cfg.ContrastAANormal = envFloat("BM_CONTRAST_AA_NORMAL", cfg.ContrastAANormal)
	cfg.ContrastAALarge = envFloat("BM_CONTRAST_AA_LARGE", cfg.ContrastAALarge)
	cfg.ContrastPrecision = envInt("BM_CONTRAST_PRECISION", cfg.ContrastPrecision)

	if v := os.Getenv("BM_ASSET_BASE_PATH"); v != "" {
		cfg.AssetBasePath = v
	}
	for _, key := range []string{"BM_SCENARIOS_DIR", "SCENARIOS_DIR"} {
		if v := os.Getenv(key); v != "" {
			cfg.ScenariosDir = v
			break
		}
	}

	// AI provider config
	for _, key := range []string{"OLLAMA_URL", "OLLAMA_BASE_URL"} {
		if v := os.Getenv(key); v != "" {
			cfg.OllamaURL = v
			break
		}
	}
	if v := os.Getenv("BM_OLLAMA_MODEL"); v != "" {
		cfg.OllamaModel = v
	}
	if v := os.Getenv("OPENROUTER_API_KEY"); v != "" {
		cfg.OpenRouterAPIKey = v
	}
	if v := os.Getenv("BM_OPENROUTER_TEXT_MODEL"); v != "" {
		cfg.OpenRouterTextModel = v
	}
	if v := os.Getenv("BM_OPENROUTER_IMAGE_MODEL"); v != "" {
		cfg.OpenRouterImageModel = v
	}

	return cfg.validated()
}

// DSN returns the SQLite DSN string with pragmas derived from config values.
func (c Config) DSN() string {
	return fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(%d)&_pragma=cache_size(-%d)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)",
		c.SQLitePath, c.BusyTimeoutMS, c.CacheSizeKB,
	)
}

// ClampLimit applies DefaultListLimit and MaxListLimit to a caller-provided limit.
// If limit <= 0, DefaultListLimit is used. Values above MaxListLimit are capped.
func (c Config) ClampLimit(limit int) int {
	if limit <= 0 {
		return c.DefaultListLimit
	}
	if limit > c.MaxListLimit {
		return c.MaxListLimit
	}
	return limit
}

// validated applies guardrails to prevent unsafe configurations.
func (c Config) validated() Config {
	if c.BusyTimeoutMS < 0 {
		c.BusyTimeoutMS = 0
	}
	if c.CacheSizeKB < 64 {
		c.CacheSizeKB = 64
	}
	if c.MaxOpenConns < 1 {
		c.MaxOpenConns = 1
	}
	if c.DefaultListLimit < 1 {
		c.DefaultListLimit = 1
	}
	if c.MaxListLimit < c.DefaultListLimit {
		c.MaxListLimit = c.DefaultListLimit
	}
	if c.ContrastAANormal < 1.0 {
		c.ContrastAANormal = 1.0
	}
	if c.ContrastAALarge < 1.0 {
		c.ContrastAALarge = 1.0
	}
	if c.ContrastPrecision < 0 {
		c.ContrastPrecision = 0
	}
	if c.ContrastPrecision > 6 {
		c.ContrastPrecision = 6
	}
	return c
}

func defaultDBPath() string {
	if path, err := resolveStoragePath(storage.ClassData, "brand-manager.db"); err == nil {
		return path
	}
	return legacyDBPath()
}

func defaultAssetPath() string {
	if path, err := resolveStoragePath(storage.ClassData, "assets"); err == nil {
		return path
	}
	return legacyAssetPath()
}

func defaultScenariosDir() string {
	if v := os.Getenv("VROOLI_ROOT"); v != "" {
		return filepath.Join(v, "scenarios")
	}
	return "scenarios"
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

func resolveStoragePath(class storage.Class, rel string) (string, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", err
	}
	return resolver.Path(storage.Options{ScenarioID: "brand-manager"}, class, rel)
}

func legacyDBPath() string {
	home := os.Getenv("HOME")
	if home == "" {
		home = "."
	}
	return filepath.Join(home, ".vrooli", "brand-manager", "brand-manager.db")
}

func legacyAssetPath() string {
	home := os.Getenv("HOME")
	if home == "" {
		home = "."
	}
	return filepath.Join(home, ".vrooli", "brand-manager", "assets")
}
