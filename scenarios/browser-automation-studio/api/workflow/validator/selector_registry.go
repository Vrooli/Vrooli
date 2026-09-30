package validator

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
)

const manifestFilename = "selectors.manifest.json"

func loadSelectorManifest(selectorFile string) (map[string]string, string, error) {
	resolved := strings.TrimSpace(selectorFile)
	if resolved == "" {
		resolved = detectSelectorManifest()
	}
	if resolved == "" {
		return nil, "", errors.New("selector manifest not found")
	}
	content, err := os.ReadFile(resolved)
	if err != nil {
		return nil, resolved, err
	}
	manifest, parseErr := parseSelectorManifest(content)
	return manifest, resolved, parseErr
}

func parseSelectorManifest(content []byte) (map[string]string, error) {
	type selectorEntry struct {
		TestID string `json:"testId"`
	}
	var payload struct {
		Selectors map[string]selectorEntry `json:"selectors"`
	}
	if err := json.Unmarshal(content, &payload); err != nil {
		return nil, err
	}
	manifest := make(map[string]string, len(payload.Selectors))
	for key, entry := range payload.Selectors {
		if entry.TestID == "" {
			continue
		}
		manifest[entry.TestID] = key
	}
	return manifest, nil
}

func detectSelectorManifest() string {
	if env := strings.TrimSpace(os.Getenv("BAS_SELECTOR_FILE")); env != "" {
		if info, err := os.Stat(env); err == nil && !info.IsDir() {
			return env
		}
	}
	return ""
}
