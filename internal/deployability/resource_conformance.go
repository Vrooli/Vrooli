package deployability

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ResourceDeclarationOverride is the deliberately explicit escape hatch for
// a resource that still needs the closed compose-service driver. It is not a
// portability claim: it is an operator-owned exception with a bounded review
// date, so an exception cannot become an unowned permanent dependency.
type ResourceDeclarationOverride struct {
	Reason   string `json:"reason"`
	ReviewBy string `json:"review_by"`
}

// CheckResourceDeclarations enforces the resource fleet's closed-driver
// policy. Docker observation remains a control-plane capability, but resource
// manifests may not silently acquire a Docker daemon through compose-service.
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
			Driver                 string                       `json:"driver"`
			ComposeServiceOverride *ResourceDeclarationOverride `json:"compose_service_override,omitempty"`
		}
		if err := json.Unmarshal(data, &declaration); err != nil {
			return nil, fmt.Errorf("parse resource declaration %s: %w", path, err)
		}
		if strings.TrimSpace(declaration.Driver) != "compose-service" {
			continue
		}
		if declaration.ComposeServiceOverride == nil {
			findings = append(findings, ConformanceFinding{
				ManifestPath: relativePath(root, path),
				Rule:         "closed-compose-service",
				Message:      "compose-service is closed for resources; migrate to managed-service or provide compose_service_override.reason and review_by",
			})
			continue
		}
		if strings.TrimSpace(declaration.ComposeServiceOverride.Reason) == "" {
			findings = append(findings, resourceDeclarationFinding(root, path, "compose_service_override.reason is required"))
		}
		if _, err := time.Parse("2006-01-02", strings.TrimSpace(declaration.ComposeServiceOverride.ReviewBy)); err != nil {
			findings = append(findings, resourceDeclarationFinding(root, path, "compose_service_override.review_by must be an ISO-8601 date"))
		}
	}
	sort.Slice(findings, func(i, j int) bool { return findings[i].ManifestPath < findings[j].ManifestPath })
	return findings, nil
}

func resourceDeclarationFinding(root, path, message string) ConformanceFinding {
	return ConformanceFinding{ManifestPath: relativePath(root, path), Rule: "closed-compose-service", Message: message}
}
