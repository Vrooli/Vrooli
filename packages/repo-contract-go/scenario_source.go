package repocontract

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveScenarioSourceRoot locates the repository that owns a scenario's
// configuration and declarations. It must not be used to locate executable
// artifacts or writable runtime state: lifecycle may retain those elsewhere.
//
// A caller-specific explicit root wins, followed by VROOLI_SOURCE_ROOT and
// VROOLI_ROOT. Invalid explicit inputs fail closed. Without an override, the
// lifecycle-owned SCENARIO_PATH/VROOLI_SCENARIO_DIR identify canonical source
// even when the process runs from a retained artifact. Their identity and paths
// must agree. Only an absent owner context permits cwd/executable discovery.
func ResolveScenarioSourceRoot(scenario, override string) (string, error) {
	if strings.TrimSpace(scenario) == "" {
		return "", fmt.Errorf("scenario is required for source root resolution")
	}
	if override = strings.TrimSpace(override); override != "" {
		absolute, err := filepath.Abs(override)
		if err != nil {
			return "", fmt.Errorf("resolve explicit source root: %w", err)
		}
		root, err := FindRepoRoot(absolute)
		if err != nil {
			return "", fmt.Errorf("resolve explicit source root: %w", err)
		}
		if root != absolute {
			return "", fmt.Errorf("explicit source root %q is not a repository root", override)
		}
		return root, nil
	}
	for _, key := range []string{defaultSourceRootEnvVar, defaultRepoRootEnvVar} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			absolute, err := filepath.Abs(value)
			if err != nil {
				return "", fmt.Errorf("resolve %s: %w", key, err)
			}
			root, err := FindRepoRoot(absolute)
			if err != nil {
				return "", fmt.Errorf("resolve %s: %w", key, err)
			}
			return root, nil
		}
	}
	var sourceRoot string
	for _, key := range []string{"SCENARIO_PATH", "VROOLI_SCENARIO_DIR"} {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" {
			continue
		}
		identityPresent := false
		for _, identityKey := range []string{"SCENARIO_NAME", "VROOLI_SCENARIO"} {
			if identity := strings.TrimSpace(os.Getenv(identityKey)); identity != "" {
				if identity != scenario {
					return "", fmt.Errorf("%s identifies %q, not %q", identityKey, identity, scenario)
				}
				identityPresent = true
			}
		}
		if !identityPresent {
			return "", fmt.Errorf("%s requires lifecycle scenario identity", key)
		}
		if !filepath.IsAbs(value) {
			return "", fmt.Errorf("%s must be an absolute canonical scenario path", key)
		}
		source, err := filepath.EvalSymlinks(value)
		if err != nil {
			return "", fmt.Errorf("resolve %s: %w", key, err)
		}
		root, err := FindRepoRoot(source)
		if err != nil {
			return "", fmt.Errorf("resolve %s repository: %w", key, err)
		}
		expected, err := ResolveScenarioPath(root, scenario)
		if err != nil {
			return "", err
		}
		expected, err = filepath.EvalSymlinks(expected)
		if err != nil {
			return "", fmt.Errorf("resolve canonical scenario path: %w", err)
		}
		if source != expected {
			return "", fmt.Errorf("%s %q is not canonical source for %q", key, value, scenario)
		}
		if sourceRoot != "" && root != sourceRoot {
			return "", fmt.Errorf("lifecycle scenario source paths disagree")
		}
		sourceRoot = root
	}
	if sourceRoot != "" {
		return sourceRoot, nil
	}
	return findRepoRootFromProcess()
}
