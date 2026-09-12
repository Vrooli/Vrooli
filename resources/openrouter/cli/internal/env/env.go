package env

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAPIBaseURL = "https://openrouter.ai/api/v1"
	resourceSlug      = "openrouter"
)

// Runtime holds derived OpenRouter runtime settings and repo-external paths.
type Runtime struct {
	DataRoot           string
	ConfigRoot         string
	StateRoot          string
	ContentRoot        string
	PromptsDir         string
	ConfigContentDir   string
	RoutesDir          string
	CredentialsFile    string
	ManualModelsFile   string
	APIBaseURL         string
	DefaultRole        string
	Timeout            time.Duration
	HealthCheckTimeout time.Duration
}

// Load derives the OpenRouter runtime from env vars with XDG-based defaults.
func Load() Runtime {
	dataRoot := firstNonEmpty(
		os.Getenv("OPENROUTER_DATA_DIR"),
		os.Getenv("RESOURCE_DATA_DIR"),
		filepath.Join(xdgDataHome(), "vrooli", "resources", resourceSlug),
	)
	configRoot := firstNonEmpty(
		os.Getenv("OPENROUTER_CONFIG_DIR"),
		os.Getenv("RESOURCE_CONFIG_DIR"),
		filepath.Join(xdgConfigHome(), "vrooli", "resources", resourceSlug),
	)
	stateRoot := firstNonEmpty(
		os.Getenv("OPENROUTER_STATE_DIR"),
		os.Getenv("RESOURCE_STATE_DIR"),
		filepath.Join(xdgStateHome(), "vrooli", "resources", resourceSlug),
	)
	contentRoot := filepath.Join(dataRoot, "content")

	return Runtime{
		DataRoot:           dataRoot,
		ConfigRoot:         configRoot,
		StateRoot:          stateRoot,
		ContentRoot:        contentRoot,
		PromptsDir:         filepath.Join(contentRoot, "prompts"),
		ConfigContentDir:   filepath.Join(contentRoot, "configs"),
		RoutesDir:          filepath.Join(contentRoot, "routes"),
		CredentialsFile:    firstNonEmpty(os.Getenv("OPENROUTER_CREDENTIALS_FILE"), filepath.Join(configRoot, "openrouter-credentials.json")),
		ManualModelsFile:   firstNonEmpty(os.Getenv("OPENROUTER_MANUAL_MODELS_FILE"), filepath.Join(configRoot, "manual-models.json")),
		APIBaseURL:         firstNonEmpty(os.Getenv("OPENROUTER_API_BASE"), defaultAPIBaseURL),
		DefaultRole:        firstNonEmpty(os.Getenv("OPENROUTER_DEFAULT_ROLE"), "chat.default"),
		Timeout:            time.Duration(envInt("OPENROUTER_TIMEOUT", 30)) * time.Second,
		HealthCheckTimeout: time.Duration(envInt("OPENROUTER_HEALTH_CHECK_TIMEOUT", 10)) * time.Second,
	}
}

// EnsureDirectories creates the standard repo-external storage locations.
func (r Runtime) EnsureDirectories() error {
	for _, path := range []string{r.DataRoot, r.ConfigRoot, r.StateRoot, r.ContentRoot, r.PromptsDir, r.ConfigContentDir, r.RoutesDir} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}
	return nil
}

// Export returns a derived env snapshot for future command wiring and docs.
func (r Runtime) Export() map[string]string {
	return map[string]string{
		"RESOURCE_DATA_DIR":             r.DataRoot,
		"RESOURCE_CONFIG_DIR":           r.ConfigRoot,
		"RESOURCE_STATE_DIR":            r.StateRoot,
		"OPENROUTER_DATA_DIR":           r.DataRoot,
		"OPENROUTER_CONFIG_DIR":         r.ConfigRoot,
		"OPENROUTER_STATE_DIR":          r.StateRoot,
		"OPENROUTER_API_BASE":           r.APIBaseURL,
		"OPENROUTER_DEFAULT_ROLE":       r.DefaultRole,
		"OPENROUTER_CREDENTIALS_FILE":   r.CredentialsFile,
		"OPENROUTER_MANUAL_MODELS_FILE": r.ManualModelsFile,
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

func xdgConfigHome() string {
	if value := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); value != "" {
		return value
	}
	return filepath.Join(userHomeDir(), ".config")
}

func xdgStateHome() string {
	if value := strings.TrimSpace(os.Getenv("XDG_STATE_HOME")); value != "" {
		return value
	}
	return filepath.Join(userHomeDir(), ".local", "state")
}
