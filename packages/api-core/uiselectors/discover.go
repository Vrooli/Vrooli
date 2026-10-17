package uiselectors

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/uimanifest"
)

func Load(root string) (map[string]interface{}, string, error) {
	if strings.TrimSpace(root) == "" {
		return nil, "", fmt.Errorf("selector target root is required")
	}
	root = filepath.Clean(root)
	if filepath.Base(root) == "bas" {
		root = filepath.Dir(root)
	}
	layout, err := uimanifest.LoadAt(root)
	if err != nil {
		return nil, "", err
	}
	path := filepath.Join(root, layout.SelectorManifestPath())
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	var manifest map[string]interface{}
	if err = json.Unmarshal(raw, &manifest); err != nil {
		return nil, "", err
	}
	if version, ok := manifest["schemaVersion"]; ok && version != float64(1) {
		return nil, path, fmt.Errorf("unsupported selector manifest version %v", version)
	}
	return manifest, path, nil
}

// ReferenceFor returns a symbolic reference only when a literal locator has
// exactly one declared identity. Recording keeps its original locator otherwise.
func ReferenceFor(manifest map[string]interface{}, selector string) string {
	match := ""
	entries, _ := manifest["selectors"].(map[string]interface{})
	for key, raw := range entries {
		entry, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		id, _ := entry["testId"].(string)
		if selector != entry["selector"] && (id == "" || selector != `[data-testid="`+id+`"]` && selector != `[data-testid='`+id+`']`) {
			continue
		}
		if match != "" {
			return ""
		}
		match = "@selector/" + key
	}
	return match
}
