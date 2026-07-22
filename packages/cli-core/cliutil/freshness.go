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
	// SkipSuffixes excludes files whose slash-path ends with any listed suffix
	// from the freshness manifest (e.g. "_test.go" so test edits never mark a
	// compiled artifact stale). It is honored by the manifest engine
	// (ComputeFreshnessManifest / EvaluateFreshness) only; the legacy
	// ComputeFreshnessFingerprint deliberately ignores it so installed-CLI
	// fingerprints stay byte-stable.
	SkipSuffixes []string
	// CaseInsensitive case-folds rel-path comparison in EvaluateFreshness so a
	// manifest recorded on a case-insensitive volume (NTFS, default APFS/HFS+)
	// matches the live input set even when the OS reports a different casing.
	// The lifecycle layer sets this from a capability-flagged volume probe; when
	// the probe is unavailable it stays false (case-sensitive — correctness-safe,
	// it can only over-report staleness, never merge two distinct files).
	CaseInsensitive bool
}

// GoModuleInstallerArgs returns the canonical cli-installer invocation for a
// Go-module CLI. Callers execute it with their cli-core directory as Cmd.Dir.
// Keeping argument construction here prevents scenario, resource, and
// api-core source installs from drifting apart.
func GoModuleInstallerArgs(modulePath, manifestPath, binaryName, installDir string, spec FreshnessSpec) []string {
	args := []string{"run", "./cmd/cli-installer", "--module", modulePath, "--name", binaryName, "--install-dir", installDir}
	if strings.TrimSpace(manifestPath) != "" {
		args = append(args, "--manifest", manifestPath)
	}
	if strings.TrimSpace(spec.ContextRoot) != "" && filepath.Clean(spec.ContextRoot) != filepath.Clean(modulePath) {
		args = append(args, "--context-root", spec.ContextRoot)
	}
	for _, input := range trimNonEmpty(spec.Inputs) {
		args = append(args, "--freshness-input", input)
	}
	return args
}

// CanonicalScenarioGoModuleFreshnessSpec returns the freshness contract used by
// cli-core's NewStandardScenarioApp (SourceContextPath="..", FreshnessInputs=
// ["<moduleDir>/**", ".vrooli/service.json", "../../packages/cli-core"]).
// Both the installer and the runtime StaleChecker must use this same spec to
// produce matching fingerprints, otherwise the installed binary is perpetually
// considered stale.
//
// scenarioRoot is the absolute path to the scenario directory (one level above
// the module). modulePath is the absolute path to the CLI's Go module. Custom
// inputs override the default when non-empty.
func CanonicalScenarioGoModuleFreshnessSpec(scenarioRoot, modulePath, binaryName string, customInputs []string) FreshnessSpec {
	moduleDir := strings.TrimSpace(filepath.Base(modulePath))
	if moduleDir == "" || moduleDir == "." || moduleDir == string(filepath.Separator) {
		moduleDir = "cli"
	}
	inputs := []string{moduleDir + "/**", ".vrooli/service.json", "../../packages/cli-core"}
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
// ["<moduleDir>/**", "resource.json", "../../packages/cli-core"]). See
// CanonicalScenarioGoModuleFreshnessSpec.
func CanonicalResourceGoModuleFreshnessSpec(resourceRoot, modulePath, binaryName string, customInputs []string) FreshnessSpec {
	moduleDir := strings.TrimSpace(filepath.Base(modulePath))
	if moduleDir == "" || moduleDir == "." || moduleDir == string(filepath.Separator) {
		moduleDir = "cli"
	}
	inputs := []string{moduleDir + "/**", "resource.json", "../../packages/cli-core"}
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
