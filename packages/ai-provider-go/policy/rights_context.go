package policy

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const RightsContextPathEnv = "VROOLI_MODEL_RIGHTS_CONTEXT_PATH"

// ResolveRightsContextPath returns only the operator-owned context location.
// Deployment configuration may point at an equivalent managed copy; absent
// that override, source-root variables or repository discovery select the
// canonical AI Gateway context. Request data must never choose this path.
func ResolveRightsContextPath(getenv func(string) string) (string, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	if path := strings.TrimSpace(getenv(RightsContextPathEnv)); path != "" {
		return path, nil
	}
	for _, key := range []string{"VROOLI_SOURCE_ROOT", "VROOLI_CLI_SOURCE_ROOT", "RESOURCE_ROOT"} {
		if root := strings.TrimSpace(getenv(key)); root != "" {
			if found := rightsContextUnderRoot(root); found != "" {
				return found, nil
			}
			return filepath.Join(root, "scenarios", "ai-gateway", "config", "model-rights-context.json"), nil
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		if found := rightsContextUnderRoot(dir); found != "" {
			return found, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", errors.New("locate trusted model rights context: set VROOLI_MODEL_RIGHTS_CONTEXT_PATH or VROOLI_SOURCE_ROOT")
}

// LoadDefaultRightsContext resolves and reads the shared operator-owned record.
func LoadDefaultRightsContext(getenv func(string) string) (RightsContext, error) {
	path, err := ResolveRightsContextPath(getenv)
	if err != nil {
		return RightsContext{}, err
	}
	context, err := LoadRightsContextFile(path)
	if err != nil {
		return RightsContext{}, err
	}
	return context, nil
}

func rightsContextUnderRoot(root string) string {
	root = filepath.Clean(root)
	for dir := root; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "scenarios", "ai-gateway", "config", "model-rights-context.json")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
	}
}

// LoadRightsContextFile reads the versioned operator-owned business context
// used by local model rights decisions. Request payloads must never provide
// this value.
func LoadRightsContextFile(path string) (RightsContext, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RightsContext{}, err
	}
	var document struct {
		SchemaVersion string        `json:"schema_version"`
		Context       RightsContext `json:"context"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		return RightsContext{}, err
	}
	c := document.Context
	if document.SchemaVersion != "1" || c.ID == "" || c.Version == "" || c.Source == "" || c.AsOf == "" || c.RevenueStatus == "" {
		return RightsContext{}, errors.New("trusted model rights context is incomplete or unsupported")
	}
	return c, nil
}
