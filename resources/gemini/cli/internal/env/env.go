package env

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAPIBaseURL = "https://generativelanguage.googleapis.com/v1beta"
	resourceSlug      = "gemini"
)

// Runtime holds derived Gemini runtime settings and repo-external storage paths.
type Runtime struct {
	DataRoot           string
	ContentRoot        string
	PromptsDir         string
	TemplatesDir       string
	FunctionsDir       string
	LogsDir            string
	CredentialsFile    string
	TokenLogFile       string
	APIBaseURL         string
	DefaultModel       string
	Timeout            time.Duration
	HealthCheckTimeout time.Duration
	HealthCheckModel   string
	RateLimitRPM       int
	RateLimitTPM       int
	CacheEnabled       bool
	CacheTTL           int
	CachePrefix        string
	RedisHost          string
	RedisPort          string
	TokenTracking      bool
}

// Load derives the Gemini runtime from env vars with XDG-based defaults.
func Load() Runtime {
	dataRoot := firstNonEmpty(
		os.Getenv("GEMINI_DATA_DIR"),
		os.Getenv("RESOURCE_DATA_DIR"),
		filepath.Join(xdgDataHome(), "vrooli", "resources", resourceSlug),
	)
	contentRoot := filepath.Join(dataRoot, "content")
	logsDir := filepath.Join(dataRoot, "logs")

	return Runtime{
		DataRoot:           dataRoot,
		ContentRoot:        contentRoot,
		PromptsDir:         filepath.Join(contentRoot, "prompts"),
		TemplatesDir:       filepath.Join(contentRoot, "templates"),
		FunctionsDir:       filepath.Join(contentRoot, "functions"),
		LogsDir:            logsDir,
		CredentialsFile:    firstNonEmpty(os.Getenv("GEMINI_CREDENTIALS_FILE"), filepath.Join(dataRoot, "credentials.json")),
		TokenLogFile:       firstNonEmpty(os.Getenv("GEMINI_TOKEN_LOG_FILE"), filepath.Join(logsDir, "token_usage.log")),
		APIBaseURL:         firstNonEmpty(os.Getenv("GEMINI_API_BASE"), defaultAPIBaseURL),
		DefaultModel:       firstNonEmpty(os.Getenv("GEMINI_DEFAULT_MODEL"), "gemini-pro"),
		Timeout:            time.Duration(envInt("GEMINI_TIMEOUT", 30)) * time.Second,
		HealthCheckTimeout: time.Duration(envInt("GEMINI_HEALTH_CHECK_TIMEOUT", 5)) * time.Second,
		HealthCheckModel:   firstNonEmpty(os.Getenv("GEMINI_HEALTH_CHECK_MODEL"), "gemini-pro"),
		RateLimitRPM:       envInt("GEMINI_RATE_LIMIT_RPM", 60),
		RateLimitTPM:       envInt("GEMINI_RATE_LIMIT_TPM", 1000000),
		CacheEnabled:       envBool("GEMINI_CACHE_ENABLED", true),
		CacheTTL:           envInt("GEMINI_CACHE_TTL", 3600),
		CachePrefix:        firstNonEmpty(os.Getenv("GEMINI_CACHE_PREFIX"), "gemini:cache"),
		RedisHost:          firstNonEmpty(os.Getenv("REDIS_HOST"), "localhost"),
		RedisPort:          firstNonEmpty(os.Getenv("REDIS_PORT"), "6379"),
		TokenTracking:      envBool("GEMINI_TOKEN_TRACKING_ENABLED", true),
	}
}

// EnsureDirectories creates the standard repo-external storage locations.
func (r Runtime) EnsureDirectories() error {
	for _, path := range []string{r.DataRoot, r.ContentRoot, r.PromptsDir, r.TemplatesDir, r.FunctionsDir, r.LogsDir} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}
	return nil
}

// Export returns a derived env snapshot for future command wiring and docs.
func (r Runtime) Export() map[string]string {
	return map[string]string{
		"RESOURCE_DATA_DIR":         r.DataRoot,
		"GEMINI_DATA_DIR":           r.DataRoot,
		"GEMINI_API_BASE":           r.APIBaseURL,
		"GEMINI_DEFAULT_MODEL":      r.DefaultModel,
		"GEMINI_HEALTH_CHECK_MODEL": r.HealthCheckModel,
		"GEMINI_CACHE_PREFIX":       r.CachePrefix,
		"GEMINI_TOKEN_LOG_FILE":     r.TokenLogFile,
		"GEMINI_CREDENTIALS_FILE":   r.CredentialsFile,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func envInt(key string, fallback int) int {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if value := strings.TrimSpace(strings.ToLower(os.Getenv(key))); value != "" {
		return value == "1" || value == "true" || value == "yes" || value == "on"
	}
	return fallback
}

func userHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}

func xdgDataHome() string {
	if value := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); value != "" {
		return value
	}
	return filepath.Join(userHomeDir(), ".local", "share")
}
