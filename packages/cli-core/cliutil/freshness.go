package cliutil

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/buildinfo"
)

// FreshnessSpec defines the exact source contract used to compute a CLI
// freshness fingerprint.
type FreshnessSpec struct {
	SourceRoot  string
	ContextRoot string
	Inputs      []string
	SkipFiles   []string
}

// CanonicalScenarioGoModuleFreshnessSpec returns the freshness contract used by
// cli-core's NewStandardScenarioApp (SourceContextPath="..", FreshnessInputs=
// ["<moduleDir>/**", ".vrooli/service.json"]). Both the installer and the
// runtime StaleChecker must use this same spec to produce matching
// fingerprints, otherwise the installed binary is perpetually considered stale.
//
// scenarioRoot is the absolute path to the scenario directory (one level above
// the module). modulePath is the absolute path to the CLI's Go module. Custom
// inputs override the default when non-empty.
func CanonicalScenarioGoModuleFreshnessSpec(scenarioRoot, modulePath, binaryName string, customInputs []string) FreshnessSpec {
	moduleDir := strings.TrimSpace(filepath.Base(modulePath))
	if moduleDir == "" || moduleDir == "." || moduleDir == string(filepath.Separator) {
		moduleDir = "cli"
	}
	inputs := []string{moduleDir + "/**", ".vrooli/service.json"}
	if trimmed := trimNonEmpty(customInputs); len(trimmed) > 0 {
		inputs = trimmed
	}
	return FreshnessSpec{
		SourceRoot:  modulePath,
		ContextRoot: scenarioRoot,
		Inputs:      inputs,
		SkipFiles:   []string{binaryName},
	}
}

// CanonicalResourceGoModuleFreshnessSpec returns the freshness contract used
// by cli-core's NewResourceApp (SourceContextPath="..", FreshnessInputs=
// ["<moduleDir>/**", "resource.json"]). See CanonicalScenarioGoModuleFreshnessSpec.
func CanonicalResourceGoModuleFreshnessSpec(resourceRoot, modulePath, binaryName string, customInputs []string) FreshnessSpec {
	moduleDir := strings.TrimSpace(filepath.Base(modulePath))
	if moduleDir == "" || moduleDir == "." || moduleDir == string(filepath.Separator) {
		moduleDir = "cli"
	}
	inputs := []string{moduleDir + "/**", "resource.json"}
	if trimmed := trimNonEmpty(customInputs); len(trimmed) > 0 {
		inputs = trimmed
	}
	return FreshnessSpec{
		SourceRoot:  modulePath,
		ContextRoot: resourceRoot,
		Inputs:      inputs,
		SkipFiles:   []string{binaryName},
	}
}

// CanonicalShellScriptFreshnessSpec returns the freshness contract for
// shell_script-adapter CLIs. ownerRoot is the scenario or resource directory
// that contains the script and manifest; customInputs overrides the default
// [scriptPath, installScript, manifestRelPath] list when non-empty.
func CanonicalShellScriptFreshnessSpec(ownerRoot, scriptPath, installScript, manifestRelPath, binaryName string, customInputs []string) FreshnessSpec {
	inputs := []string{scriptPath, installScript, filepath.ToSlash(manifestRelPath)}
	if trimmed := trimNonEmpty(customInputs); len(trimmed) > 0 {
		inputs = trimmed
	}
	return FreshnessSpec{
		SourceRoot:  ownerRoot,
		ContextRoot: ownerRoot,
		Inputs:      inputs,
		SkipFiles:   []string{binaryName},
	}
}

// ComputeFreshnessFingerprint computes a deterministic fingerprint from one
// canonical freshness contract. When Inputs are declared, they are resolved
// from ContextRoot. Otherwise the entire SourceRoot is fingerprinted.
func ComputeFreshnessFingerprint(spec FreshnessSpec) (string, error) {
	sourceRoot := filepath.Clean(strings.TrimSpace(spec.SourceRoot))
	if sourceRoot == "" {
		return "", fmt.Errorf("freshness source root must not be empty")
	}

	skipFiles := append([]string(nil), spec.SkipFiles...)
	inputs := trimNonEmpty(spec.Inputs)
	if len(inputs) == 0 {
		return buildinfo.ComputeFingerprint(sourceRoot, skipFiles...)
	}

	contextRoot := filepath.Clean(strings.TrimSpace(spec.ContextRoot))
	if contextRoot == "" {
		contextRoot = sourceRoot
	}
	return computeFingerprintFromDeclaredInputs(contextRoot, inputs, skipFiles...)
}

func trimNonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}
