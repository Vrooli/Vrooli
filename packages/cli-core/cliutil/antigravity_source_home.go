package cliutil

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func antigravityModelLabel(source ModelSource) string {
	return "Vrooli credits (" + source.Model + ")"
}

func antigravityModelURI(source ModelSource) string {
	base, err := url.Parse(source.BaseURL)
	if err != nil || base.Host == "" {
		return ""
	}
	// Antigravity's URI parser treats the authority as the Gemini API host and
	// the path after /models/ as the model ID. GOOGLE_GEMINI_BASE_URL carries
	// the selected endpoint including its path, so embedding the full URL here
	// would turn `http:/host/...` into part of the model ID.
	return "gemini-api://" + base.Host + "/models/" + url.PathEscape(source.Model)
}

// prepareAntigravitySourceHome creates a short-lived HOME overlay because the
// supported Antigravity CLI reads customModelsConfig only from settings.json.
// It copies user preferences into the overlay, overrides only model routing,
// and leaves the original settings and credential files untouched.
func prepareAntigravitySourceHome(source ModelSource, environment []string) ([]string, func(), error) {
	originalHome := environmentValue(environment, "HOME")
	if originalHome == "" {
		var err error
		originalHome, err = os.UserHomeDir()
		if err != nil {
			return nil, nil, err
		}
	}
	root, err := os.MkdirTemp("", "vrooli-antigravity-source-")
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { _ = os.RemoveAll(root) }
	settingsDir := filepath.Join(root, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(settingsDir, 0700); err != nil {
		cleanup()
		return nil, nil, err
	}
	settings := map[string]any{}
	originalSettings := filepath.Join(originalHome, ".gemini", "antigravity-cli", "settings.json")
	if data, err := os.ReadFile(originalSettings); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			cleanup()
			return nil, nil, errors.New("Antigravity settings are invalid JSON")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		cleanup()
		return nil, nil, err
	}
	settings["modelProvider"] = "gemini"
	customConfig, _ := settings["customModelsConfig"].(map[string]any)
	if customConfig == nil {
		customConfig = map[string]any{}
	}
	models, _ := customConfig["customModels"].([]any)
	modelName := antigravityModelURI(source)
	label := antigravityModelLabel(source)
	filtered := make([]any, 0, len(models)+1)
	for _, value := range models {
		model, _ := value.(map[string]any)
		if model["label"] != label {
			filtered = append(filtered, value)
		}
	}
	customConfig["customModels"] = append(filtered, map[string]any{"modelName": modelName, "label": label})
	settings["customModelsConfig"] = customConfig
	data, err := json.Marshal(settings)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	if err := os.WriteFile(filepath.Join(settingsDir, "settings.json"), data, 0600); err != nil {
		cleanup()
		return nil, nil, err
	}
	if sourceKeys := filepath.Join(originalHome, ".gemini", "antigravity-cli", "keybindings.json"); fileExists(sourceKeys) {
		if contents, readErr := os.ReadFile(sourceKeys); readErr == nil {
			_ = os.WriteFile(filepath.Join(settingsDir, "keybindings.json"), contents, 0600)
		}
	}
	environment = withEnvironmentValue(environment, "HOME", root)
	return environment, cleanup, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// antigravitySourceEnvironment routes Gemini API-key requests through the
// selected endpoint. Antigravity uses GEMINI_API_KEY only after
// modelProvider=gemini, and its custom-endpoint selector reads
// GOOGLE_GEMINI_BASE_URL.
func antigravitySourceEnvironment(source ModelSource, environment []string) []string {
	environment = withEnvironmentValue(environment, "GEMINI_API_KEY", environmentValue(environment, source.SecretEnv))
	return withEnvironmentValue(environment, "GOOGLE_GEMINI_BASE_URL", strings.TrimRight(source.BaseURL, "/"))
}
