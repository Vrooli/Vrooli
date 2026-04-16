package env

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultAPIBaseURL = "https://api.cloudflare.com/client/v4/accounts"
	resourceSlug      = "cloudflare-ai-gateway"
)

// Runtime holds derived local storage and endpoint configuration for the
// Cloudflare AI Gateway resource.
type Runtime struct {
	DataRoot   string
	ConfigsDir string
	LogsDir    string
	ConfigFile string
	StateFile  string
	APIBaseURL string
}

// Load derives runtime paths from standard env vars with XDG-based defaults.
func Load() Runtime {
	dataRoot := firstNonEmpty(
		os.Getenv("CLOUDFLARE_AI_GATEWAY_DATA_DIR"),
		os.Getenv("RESOURCE_DATA_DIR"),
		filepath.Join(xdgDataHome(), "vrooli", "resources", resourceSlug),
	)

	return Runtime{
		DataRoot:   dataRoot,
		ConfigsDir: filepath.Join(dataRoot, "configs"),
		LogsDir:    filepath.Join(dataRoot, "logs"),
		ConfigFile: filepath.Join(dataRoot, "config.json"),
		StateFile:  filepath.Join(dataRoot, "state.json"),
		APIBaseURL: firstNonEmpty(
			os.Getenv("CLOUDFLARE_AI_GATEWAY_API_BASE_URL"),
			defaultAPIBaseURL,
		),
	}
}

// EnsureDirectories creates the standard repo-external storage locations for
// the resource.
func (r Runtime) EnsureDirectories() error {
	for _, path := range []string{r.DataRoot, r.ConfigsDir, r.LogsDir} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}
	return nil
}

// Export returns a minimal derived env snapshot for future command wiring.
func (r Runtime) Export() map[string]string {
	return map[string]string{
		"RESOURCE_DATA_DIR":                  r.DataRoot,
		"CLOUDFLARE_AI_GATEWAY_DATA_DIR":     r.DataRoot,
		"CLOUDFLARE_AI_GATEWAY_CONFIG_DIR":   r.ConfigsDir,
		"CLOUDFLARE_AI_GATEWAY_LOG_DIR":      r.LogsDir,
		"CLOUDFLARE_AI_GATEWAY_CONFIG_FILE":  r.ConfigFile,
		"CLOUDFLARE_AI_GATEWAY_STATE_FILE":   r.StateFile,
		"CLOUDFLARE_AI_GATEWAY_API_BASE_URL": r.APIBaseURL,
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
