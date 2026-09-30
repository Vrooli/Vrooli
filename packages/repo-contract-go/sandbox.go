package repocontract

import (
	"path/filepath"
	"strings"
)

// IsFullRepoScope reports whether the sandbox scope covers the entire repo.
func (c *Contract) IsFullRepoScope(scope string) bool {
	scope = normalizeSandboxScope(scope)
	for _, candidate := range c.doc.Sandbox.FullRepoScopes {
		if scope == normalizeSandboxScope(candidate) {
			return true
		}
	}
	return false
}

// ScenarioScopeMatch reports whether a sandbox scope covers the named scenario.
func (c *Contract) ScenarioScopeMatch(scenario, scope string) bool {
	scenario, err := cleanIdentifier(scenario)
	if err != nil {
		return false
	}

	scope = normalizeSandboxScope(scope)
	if c.IsFullRepoScope(scope) {
		return true
	}

	scenarioDir := filepathToSlashTrimmed(c.doc.Layout.ScenarioDir)
	if scope == scenarioDir {
		return true
	}

	prefix := filepathToSlashTrimmed(strings.TrimSuffix(c.doc.Sandbox.ScenarioScopePrefix, "/"))
	if prefix == "" {
		prefix = scenarioDir
	}
	if !strings.HasPrefix(scope, prefix+"/") {
		return false
	}

	remainder := strings.TrimPrefix(scope, prefix+"/")
	scopedName := remainder
	if idx := strings.Index(remainder, "/"); idx >= 0 {
		scopedName = remainder[:idx]
	}
	return scopedName == scenario
}

// ResolveSandboxScenarioPath maps a scenario to its effective path inside a
// sandbox merged directory when the scope covers that scenario.
func (c *Contract) ResolveSandboxScenarioPath(merged, scope, scenario string) (string, bool, error) {
	scenario, err := cleanIdentifier(scenario)
	if err != nil {
		return "", false, err
	}

	scope = normalizeSandboxScope(scope)
	merged = filepath.Clean(merged)
	scenarioRel := filepath.Join(filepath.FromSlash(c.doc.Layout.ScenarioDir), scenario)

	if c.IsFullRepoScope(scope) {
		return filepath.Join(merged, scenarioRel), true, nil
	}
	if !c.ScenarioScopeMatch(scenario, scope) {
		return "", false, nil
	}
	if scope == filepathToSlashTrimmed(c.doc.Layout.ScenarioDir) {
		return filepath.Join(merged, scenario), true, nil
	}
	if scope == filepath.ToSlash(scenarioRel) {
		return merged, true, nil
	}
	if strings.HasPrefix(filepath.ToSlash(scenarioRel), scope+"/") {
		relative := strings.TrimPrefix(filepath.ToSlash(scenarioRel), scope+"/")
		return filepath.Join(merged, filepath.FromSlash(relative)), true, nil
	}

	return filepath.Join(merged, scenarioRel), true, nil
}

func normalizeSandboxScope(scope string) string {
	scope = filepathToSlashTrimmed(scope)
	scope = strings.TrimSuffix(scope, "/")
	return scope
}
