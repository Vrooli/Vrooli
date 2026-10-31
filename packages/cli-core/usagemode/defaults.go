package usagemode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const configFileMaxBytes = 1 << 20

// ConfiguredModel is the model the harness would start on without a mode: the
// top-level `model` of Codex's config.toml or of Claude's settings. ok is false
// when a config exists but cannot be read cheaply and safely (not a regular
// file, too large, malformed); callers then change nothing. An absent file or
// key is model "" with ok true. Project-level settings are not consulted.
func ConfiguredModel(runner string) (model string, ok bool) {
	switch NormalizeRunner(runner) {
	case "codex":
		dir := strings.TrimSpace(os.Getenv("CODEX_HOME"))
		if dir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", false
			}
			dir = filepath.Join(home, ".codex")
		}
		raw, err := readRegular(filepath.Join(dir, "config.toml"), configFileMaxBytes)
		if os.IsNotExist(err) {
			return "", true
		}
		if err != nil {
			return "", false
		}
		return tomlTopLevelString(string(raw), "model"), true
	case "claude":
		dir := strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_DIR"))
		if dir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", false
			}
			dir = filepath.Join(home, ".claude")
		}
		// settings.local.json overrides settings.json.
		for _, name := range []string{"settings.local.json", "settings.json"} {
			raw, err := readRegular(filepath.Join(dir, name), configFileMaxBytes)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return "", false
			}
			var settings struct {
				Model string            `json:"model"`
				Env   map[string]string `json:"env"`
			}
			if json.Unmarshal(raw, &settings) != nil {
				return "", false
			}
			if settings.Model != "" {
				return settings.Model, true
			}
			if settings.Env["ANTHROPIC_MODEL"] != "" {
				return settings.Env["ANTHROPIC_MODEL"], true
			}
		}
		return "", true
	}
	return "", false
}

// tomlTopLevelString reads `key = "value"` before the first table header.
func tomlTopLevelString(text, key string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			return ""
		}
		name, value, found := strings.Cut(line, "=")
		if !found || strings.TrimSpace(name) != key {
			continue
		}
		value = strings.TrimSpace(value)
		if value == "" || (value[0] != '"' && value[0] != '\'') {
			return ""
		}
		if end := strings.IndexByte(value[1:], value[0]); end >= 0 {
			return value[1 : 1+end]
		}
		return ""
	}
	return ""
}

// SessionModelAllowed reports whether a mode may set the session model: only
// when the harness has no configured default, or the default is one the mode
// lowers. A default that is already lower than every mapped model, or one that
// cannot be read, is left alone, so a mode never raises a model.
func SessionModelAllowed(runner string, s RunnerSettings) bool {
	if s.SessionModel == "" {
		return false
	}
	configured, ok := ConfiguredModel(runner)
	if !ok {
		return false
	}
	if configured == "" {
		return true
	}
	if i := strings.IndexByte(configured, '['); i > 0 { // "opus[1m]"
		configured = configured[:i]
	}
	configured = strings.ToLower(strings.TrimSpace(configured))
	for key := range s.ModelMap {
		if strings.ToLower(key) == configured {
			return true
		}
	}
	return false
}
