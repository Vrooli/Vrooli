package deployability

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CheckResourceDeclarations rejects resource declarations outside the closed
// set of supported resource archetypes. Runtime-specific validation remains in
// internal/resources/manifest; this check keeps deployability independent of
// any particular runtime implementation.
func CheckResourceDeclarations(root string) ([]ConformanceFinding, error) {
	resourcesRoot := filepath.Join(root, "resources")
	entries, err := os.ReadDir(resourcesRoot)
	if err != nil {
		return nil, err
	}
	var findings []ConformanceFinding
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(resourcesRoot, entry.Name(), "resource.json")
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		var declaration struct {
			Driver string `json:"driver"`
		}
		if err := json.Unmarshal(data, &declaration); err != nil {
			return nil, fmt.Errorf("parse resource declaration %s: %w", path, err)
		}
		driver := strings.TrimSpace(declaration.Driver)
		if driver == "managed-service" || driver == "external-cli" || driver == "native-cli" || driver == "cloud-api" {
			continue
		}
		findings = append(findings, ConformanceFinding{ManifestPath: relativePath(root, path), Rule: "unsupported-driver", Message: fmt.Sprintf("resource driver %q is not supported", driver)})
	}
	sort.Slice(findings, func(i, j int) bool { return findings[i].ManifestPath < findings[j].ManifestPath })
	return findings, nil
}
