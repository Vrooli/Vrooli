package repocontract

import (
	"path/filepath"
	"strings"
)

// Manifest-path helpers resolve well-known manifest file locations through
// the repository contract. Callers MUST use these helpers rather than joining
// literal strings like "docs/manifest.json" or "cli/manifest.json" — the
// audit_test.go enforces this for the whole repo.

const (
	// schemasDirName is the leaf directory under the project_config_dir
	// where shared JSON Schemas live (e.g., .vrooli/schemas/).
	schemasDirName = "schemas"

	// Canonical relative paths used as fallbacks if the contract cannot
	// be loaded. These literals live here (allowlisted by audit_test.go)
	// as the single source of truth.
	defaultDocsManifestRel = "docs/manifest.json"
	defaultCLIManifestRel  = "cli/manifest.json"
	defaultServiceRel      = ".vrooli/service.json"
)

// ScenarioWellKnownPath identifies a scenario well-known path key from the
// repository contract.
type ScenarioWellKnownPath string

const (
	ScenarioPathService      ScenarioWellKnownPath = "service"
	ScenarioPathDocsManifest ScenarioWellKnownPath = "docs_manifest"
	ScenarioPathCLIManifest  ScenarioWellKnownPath = "cli_manifest"
)

// ScenarioServiceManifestRel returns the slash-normalized scenario-relative
// path at which the service manifest is expected (e.g.,
// ".vrooli/service.json").
func ScenarioServiceManifestRel(repoRoot string) (string, error) {
	return ScenarioWellKnownRel(repoRoot, ScenarioPathService)
}

// ScenarioDocsManifestRel returns the slash-normalized scenario-relative path
// at which the docs manifest is expected (e.g., "docs/manifest.json"). Use
// this when a caller needs the path string itself rather than the absolute
// path resolved against a scenario root — e.g., a finding's display value,
// a config default, or a test fixture writing into a temp scenario.
func ScenarioDocsManifestRel(repoRoot string) (string, error) {
	return ScenarioWellKnownRel(repoRoot, ScenarioPathDocsManifest)
}

// ScenarioCLIManifestRel returns the slash-normalized scenario-relative path
// at which the CLI manifest is expected (e.g., "cli/manifest.json").
func ScenarioCLIManifestRel(repoRoot string) (string, error) {
	return ScenarioWellKnownRel(repoRoot, ScenarioPathCLIManifest)
}

// ScenarioCLIManifestRepoRel returns the slash-normalized repo-relative path
// to a scenario's CLI manifest.
func ScenarioCLIManifestRepoRel(repoRoot, scenario string) (string, error) {
	return ScenarioWellKnownRepoRel(repoRoot, scenario, ScenarioPathCLIManifest)
}

// ScenarioWellKnownRel returns the slash-normalized scenario-relative path for
// a well-known scenario path key.
func ScenarioWellKnownRel(repoRoot string, key ScenarioWellKnownPath) (string, error) {
	fallback, err := defaultScenarioWellKnownRel(key)
	if err != nil {
		return "", err
	}
	repoRoot = filepath.Clean(repoRoot)
	if contract, err := LoadDefault(repoRoot); err == nil {
		if rel, ok := contract.doc.Scenario.WellKnownPaths[string(key)]; ok && strings.TrimSpace(rel) != "" {
			return rel, nil
		}
	}
	return fallback, nil
}

// ScenarioWellKnownRepoRel returns a slash-normalized repo-relative path for a
// scenario well-known path key.
func ScenarioWellKnownRepoRel(repoRoot, scenario string, key ScenarioWellKnownPath) (string, error) {
	if strings.TrimSpace(scenario) == "" {
		return "", &Error{Kind: ErrInvalidInput, Message: "scenario name is required"}
	}
	rel, err := ScenarioWellKnownRel(repoRoot, key)
	if err != nil {
		return "", err
	}
	scenarioDir := "scenarios"
	repoRoot = filepath.Clean(repoRoot)
	if contract, err := LoadDefault(repoRoot); err == nil && strings.TrimSpace(contract.doc.Layout.ScenarioDir) != "" {
		scenarioDir = contract.doc.Layout.ScenarioDir
	}
	return filepath.ToSlash(filepath.Join(scenarioDir, filepath.Clean(scenario), filepath.FromSlash(rel))), nil
}

func defaultScenarioWellKnownRel(key ScenarioWellKnownPath) (string, error) {
	switch key {
	case ScenarioPathService:
		return defaultServiceRel, nil
	case ScenarioPathDocsManifest:
		return defaultDocsManifestRel, nil
	case ScenarioPathCLIManifest:
		return defaultCLIManifestRel, nil
	default:
		return "", &Error{Kind: ErrInvalidInput, Message: "unknown scenario well-known path key", Details: string(key)}
	}
}

// ScenarioDocsManifestPath returns the absolute path to a scenario's docs
// manifest (`docs/manifest.json` by default), resolved through the repo
// contract's `scenario.well_known_paths.docs_manifest` entry. Falls back to
// the canonical default if the contract cannot be loaded.
func ScenarioDocsManifestPath(repoRoot, scenario string) (string, error) {
	return ScenarioWellKnownPathForScenario(repoRoot, scenario, ScenarioPathDocsManifest)
}

// ScenarioCLIManifestPath returns the absolute path to a scenario's CLI
// manifest (`cli/manifest.json` by default), resolved through the repo
// contract's `scenario.well_known_paths.cli_manifest` entry. Falls back to
// the canonical default if the contract cannot be loaded.
func ScenarioCLIManifestPath(repoRoot, scenario string) (string, error) {
	return ScenarioWellKnownPathForScenario(repoRoot, scenario, ScenarioPathCLIManifest)
}

// ScenarioServiceManifestPath returns the absolute path to a scenario's service
// manifest (`.vrooli/service.json` by default), resolved through the repo
// contract's `scenario.well_known_paths.service` entry. Falls back to the
// canonical default if the contract cannot be loaded.
func ScenarioServiceManifestPath(repoRoot, scenario string) (string, error) {
	return ScenarioWellKnownPathForScenario(repoRoot, scenario, ScenarioPathService)
}

// ScenarioWellKnownPathForScenario returns the absolute path to a scenario's
// well-known path entry, resolved through the repository contract.
func ScenarioWellKnownPathForScenario(repoRoot, scenario string, key ScenarioWellKnownPath) (string, error) {
	fallback, err := defaultScenarioWellKnownRel(key)
	if err != nil {
		return "", err
	}
	return scenarioManifestPath(repoRoot, scenario, string(key), fallback)
}

// SchemaPath returns the absolute path to a shared JSON Schema file under the
// project's schemas directory (e.g., `.vrooli/schemas/<name>`). It resolves
// the project_config_dir through the contract and joins `schemas/<name>`.
// Falls back to `.vrooli/schemas/<name>` if the contract cannot be loaded.
func SchemaPath(repoRoot, schemaName string) (string, error) {
	schemaName = strings.TrimSpace(schemaName)
	if schemaName == "" {
		return "", &Error{Kind: ErrInvalidInput, Message: "schema name is required"}
	}
	if strings.ContainsAny(schemaName, `\/`) || strings.Contains(schemaName, "..") {
		return "", &Error{Kind: ErrInvalidInput, Message: "schema name must not contain path separators or traversal", Details: schemaName}
	}
	repoRoot = filepath.Clean(repoRoot)
	if contract, err := LoadDefault(repoRoot); err == nil {
		if projectCfg, err := contract.TopLevelDir(repoRoot, "project_config"); err == nil {
			return filepath.Join(projectCfg, schemasDirName, schemaName), nil
		}
	}
	return filepath.Join(repoRoot, ".vrooli", schemasDirName, schemaName), nil
}

// scenarioManifestPath resolves an absolute scenario-relative manifest path
// through the contract, falling back to <scenarioRoot>/<fallbackRel>.
func scenarioManifestPath(repoRoot, scenario, key, fallbackRel string) (string, error) {
	if strings.TrimSpace(scenario) == "" {
		return "", &Error{Kind: ErrInvalidInput, Message: "scenario name is required"}
	}
	repoRoot = filepath.Clean(repoRoot)
	if contract, err := LoadDefault(repoRoot); err == nil {
		if resolved, err := contract.ScenarioFile(repoRoot, scenario, key); err == nil {
			return resolved, nil
		}
	}
	return filepath.Join(ScenarioRoot(repoRoot, scenario), filepath.FromSlash(fallbackRel)), nil
}
