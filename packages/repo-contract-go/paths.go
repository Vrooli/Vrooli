package repocontract

import (
	"path/filepath"
	"strings"
)

func ScenarioRoot(repoRoot, scenario string) string {
	repoRoot = filepath.Clean(repoRoot)
	if contract, err := LoadDefault(repoRoot); err == nil {
		if resolved, err := contract.ScenarioRoot(repoRoot, scenario); err == nil {
			return resolved
		}
	}
	return filepath.Join(repoRoot, "scenarios", filepath.Clean(scenario))
}

func (c *Contract) ScenarioFile(repoRoot, scenario, key string) (string, error) {
	root, err := c.ScenarioRoot(repoRoot, scenario)
	if err != nil {
		return "", err
	}
	rel, ok := c.doc.Scenario.WellKnownPaths[strings.TrimSpace(key)]
	if !ok {
		return "", &Error{Kind: ErrNotFound, Message: "scenario well-known path not found", Details: key}
	}
	return filepath.Join(root, filepath.FromSlash(rel)), nil
}

func (c *Contract) ResourceFile(repoRoot, resource, key string) (string, error) {
	root, err := c.ResourceRoot(repoRoot, resource)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(key) == "manifest" {
		return filepath.Join(root, filepath.FromSlash(c.doc.Resource.Manifest)), nil
	}
	rel, ok := c.doc.Resource.WellKnownPaths[strings.TrimSpace(key)]
	if !ok {
		return "", &Error{Kind: ErrNotFound, Message: "resource well-known path not found", Details: key}
	}
	return filepath.Join(root, filepath.FromSlash(rel)), nil
}

func (c *Contract) TopLevelDir(repoRoot, key string) (string, error) {
	var rel string
	switch strings.TrimSpace(key) {
	case "project_config":
		rel = c.doc.Layout.ProjectConfigDir
	case "scenarios":
		rel = c.doc.Layout.ScenarioDir
	case "resources":
		rel = c.doc.Layout.ResourceDir
	case "packages":
		rel = c.doc.Layout.PackageDir
	case "cmd":
		rel = c.doc.Layout.CommandDir
	case "internal":
		rel = c.doc.Layout.InternalDir
	case "docs":
		rel = c.doc.Layout.DocsDir
	default:
		return "", &Error{Kind: ErrNotFound, Message: "top-level dir not found", Details: key}
	}
	return filepath.Join(filepath.Clean(repoRoot), filepath.FromSlash(rel)), nil
}
