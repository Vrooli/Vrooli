package deploy

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MonetizationManifest is the release-facing subset of a scenario's paid
// surface declaration. Deployment reads this declaration instead of accepting
// entitlement policy as a second, hand-typed input.
type MonetizationManifest struct {
	Version             int    `json:"version"`
	BundleKey           string `json:"bundle_key"`
	AppKey              string `json:"app_key"`
	RequiresEntitlement *bool  `json:"requires_entitlement"`
}

// LoadMonetizationManifest loads a scenario declaration when one exists.
// A scenario without a monetization declaration is still deployable; a
// malformed declaration is an error because it would make release policy
// ambiguous.
func LoadMonetizationManifest(root, scenarioName string) (*MonetizationManifest, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		root = strings.TrimSpace(os.Getenv("VROOLI_ROOT"))
	}
	if root == "" {
		root, _ = os.Getwd()
	}
	root = findVrooliRoot(root)
	if strings.TrimSpace(scenarioName) == "" {
		return nil, nil
	}
	path := filepath.Join(root, "scenarios", scenarioName, ".vrooli", "monetization.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read monetization manifest %s: %w", path, err)
	}
	var manifest MonetizationManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("decode monetization manifest %s: %w", path, err)
	}
	if manifest.Version != 2 || strings.TrimSpace(manifest.BundleKey) == "" || strings.TrimSpace(manifest.AppKey) == "" {
		return nil, fmt.Errorf("invalid monetization manifest %s: version 2, bundle_key, and app_key are required", path)
	}
	if manifest.RequiresEntitlement == nil {
		value := true
		manifest.RequiresEntitlement = &value
	}
	return &manifest, nil
}

func findVrooliRoot(start string) string {
	path := filepath.Clean(start)
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		path = filepath.Dir(path)
	}
	for {
		if _, err := os.Stat(filepath.Join(path, ".vrooli")); err == nil {
			return path
		}
		parent := filepath.Dir(path)
		if parent == path {
			return filepath.Clean(start)
		}
		path = parent
	}
}
