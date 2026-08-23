package adapters

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"unit-health/internal/discovery"
	"unit-health/internal/hostbin"
)

// WorkspaceInput is the observed surface plus the small set of host seams an
// adapter needs to resolve a runnable workspace. It contains facts and intent
// hints, never a shell command.
type WorkspaceInput struct {
	Surface           discovery.Surface
	Language          string
	ResolveExecutable func([]string) (string, bool)
}

// WorkspaceResolution is the adapter-owned projection of a discovered
// surface. Validation maps it into its response model and owns no framework
// selection logic.
type WorkspaceResolution struct {
	Framework          string
	CanonicalFramework string
	PackageManager     string
	TestCommand        string
	CoverageCommand    string
	CoverageRequested  bool
	TestExecutable     string
	TestArgs           []string
	CoverageExecutable string
	CoverageArgs       []string
	TestArtifacts      []Artifact
	TestPath           string
	Status             string
	DegradedReason     string
}

// Diagnostic is an adapter finding before it is normalized into Unit Health's
// response contract.
type Diagnostic struct {
	Code, Category, Severity  string
	File, Message, Evidence   string
	Expected, Observed        string
	WhyItMatters, Remediation string
}

const (
	diagnosticSurfaceMissing   = "TEST_FRAMEWORK_MISSING"
	diagnosticNoncanonical     = "TEST_FRAMEWORK_NONCANONICAL"
	diagnosticCoverage         = "COVERAGE_CONFIG_MISSING"
	diagnosticPackageManager   = "PACKAGE_MANAGER_MISMATCH"
	diagnosticMisconfiguration = "TEST_MISCONFIGURATION"
	diagnosticDependency       = "TEST_DEPENDENCY_MISSING"
	diagnosticUnsupported      = "UNSUPPORTED_PARSE_UNIT"
)

// ResolveWorkspace dispatches only through adapter-owned resolution rules.
func ResolveWorkspace(input WorkspaceInput) (WorkspaceResolution, []Diagnostic) {
	var resolution WorkspaceResolution
	var diagnostics []Diagnostic
	add := func(code, file, message, evidence, expected, observed, why, remediation string) {
		diagnostics = append(diagnostics, Diagnostic{
			Code: code, Category: "config", Severity: severity(code), File: file,
			Message: message, Evidence: evidence, Expected: expected, Observed: observed,
			WhyItMatters: why, Remediation: remediation,
		})
	}
	resolution.PackageManager = input.Surface.PackageManager
	if resolution.PackageManager == "" {
		resolution.PackageManager = "pnpm"
	}

	switch input.Language {
	case "go":
		resolution.Framework = "go test"
		resolution.CanonicalFramework = "go test"
		resolution.Status = "ready"
		resolution.CoverageRequested = true
		if input.ResolveExecutable != nil {
			if executable, ok := input.ResolveExecutable([]string{"go"}); ok {
				resolution.TestExecutable = executable
			} else {
				resolution.Status = "degraded"
				resolution.DegradedReason = "Go surface was discovered, but the Go toolchain is not resolvable on this host."
				diagnostics = append(diagnostics, Diagnostic{Code: diagnosticDependency, Category: "config", Severity: severity(diagnosticDependency), File: input.Surface.RootPath, Message: "Go workspace was discovered but the Go toolchain is not resolvable.", Evidence: "go not found", Expected: "Go installed and resolvable.", Observed: "missing Go toolchain", WhyItMatters: "Without the Go toolchain the Go tests and coverage command cannot run.", Remediation: "Install or expose Go through Scenario Dependency Analyzer, then rerun Unit Health."})
			}
		}
	case "typescript", "javascript", "node":
		resolveNodeWorkspace(input, &resolution, &diagnostics)
	case "bash":
		resolveBashWorkspace(input, &resolution, add)
	case "python":
		resolution.Framework = "pytest"
		resolution.CanonicalFramework = "pytest"
		resolution.Status = "degraded"
		resolution.DegradedReason = "Code Facts does not yet provide Python parse units; using fallback discovery."
		add(diagnosticUnsupported, input.Surface.RootPath,
			"Python workspace discovered through filesystem fallback; Code Facts does not yet describe Python parse units.",
			"language=python, code-facts parse units unavailable", "Code Facts parse-unit coverage for Python.",
			"fallback discovery only", "Fallback discovery is coarser than Code Facts and may miss workspaces or misclassify frameworks.",
			"Track Code Facts Python parse-unit support; until then verify the pytest plan manually.")
		if input.ResolveExecutable != nil {
			candidates := []string{"python3", "python"}
			if runtime.GOOS == "windows" {
				candidates = []string{"py", "python"}
			}
			if executable, ok := input.ResolveExecutable(candidates); ok {
				resolution.TestExecutable = executable
			} else {
				resolution.DegradedReason = "Python fallback discovery succeeded, but no Python launcher is available on this host."
				diagnostics = append(diagnostics, Diagnostic{Code: diagnosticDependency, Category: "config", Severity: severity(diagnosticDependency), File: input.Surface.RootPath, Message: "Python workspace was discovered but no Python launcher is resolvable.", Evidence: "pytest surface; python3/python/py not found", Expected: "A Python launcher capable of importing pytest.", Observed: "missing Python launcher", WhyItMatters: "Without a launcher the degraded pytest plan cannot execute.", Remediation: "Install or expose Python through Scenario Dependency Analyzer, then rerun Unit Health."})
			}
		}
		if pythonCoverageConfigured(input.Surface.RootPath) {
			resolution.CoverageRequested = true
		}
	case "rust":
		resolution.Framework = "cargo"
		resolution.CanonicalFramework = "cargo"
		resolution.Status = "ready"
		if input.ResolveExecutable != nil {
			if executable, ok := input.ResolveExecutable([]string{"cargo"}); ok {
				resolution.TestExecutable = executable
			} else {
				resolution.Status = "degraded"
				resolution.DegradedReason = "Rust surface was discovered, but Cargo is not installed on this host."
				diagnostics = append(diagnostics, Diagnostic{Code: diagnosticDependency, Category: "config", Severity: severity(diagnosticDependency), File: input.Surface.RootPath, Message: "Rust workspace was discovered but Cargo is not resolvable.", Evidence: "cargo not found", Expected: "Cargo installed and resolvable.", Observed: "missing Cargo", WhyItMatters: "Without Cargo the Rust tests cannot run.", Remediation: "Install or expose Cargo through Scenario Dependency Analyzer, then rerun Unit Health."})
			}
			if executable, ok := input.ResolveExecutable([]string{"cargo-llvm-cov"}); ok {
				resolution.CoverageExecutable = executable
				resolution.CoverageRequested = true
			}
		}
	case "powershell":
		resolution.Framework = "pester"
		resolution.CanonicalFramework = "pester"
		resolution.TestPath = firstTestFile(input.Surface.RootPath, ".Tests.ps1")
		if resolution.TestPath == "" {
			resolution.Status = "degraded"
			resolution.DegradedReason = "PowerShell surface has no *Tests.ps1 file for Pester."
			add(diagnosticSurfaceMissing, input.Surface.RootPath, "PowerShell surface has no Pester test script.",
				"no *Tests.ps1 file", "A Pester *Tests.ps1 test script.", "missing Pester test path",
				"Without a test script the PowerShell surface cannot be validated.",
				"Add a Pester *Tests.ps1 test script and ensure Pester is governed by Scenario Dependency Analyzer.")
		} else {
			resolution.Status = "ready"
			if input.ResolveExecutable != nil {
				if executable, ok := input.ResolveExecutable([]string{"pwsh", "powershell"}); ok {
					resolution.TestExecutable = executable
				} else {
					resolution.Status = "degraded"
					resolution.DegradedReason = "Pester tests were found, but no PowerShell launcher is available on this host."
					diagnostics = append(diagnostics, Diagnostic{Code: diagnosticDependency, Category: "config", Severity: severity(diagnosticDependency), File: resolution.TestPath, Message: "Pester tests were found but no PowerShell launcher is resolvable.", Evidence: "pwsh/powershell not found", Expected: "PowerShell installed and resolvable.", Observed: "missing PowerShell launcher", WhyItMatters: "Without PowerShell the Pester tests cannot run.", Remediation: "Run this adapter on Windows with PowerShell provisioned through Scenario Dependency Analyzer."})
				}
			}
		}
	default:
		resolution.Status = "unsupported"
		resolution.DegradedReason = "No canonical test framework is known for this language."
		add(diagnosticUnsupported, input.Surface.RootPath,
			fmt.Sprintf("Surface %q has an unsupported or unknown language (%q); Unit Health cannot plan tests for it.", input.Surface.ID, input.Language),
			"language="+input.Language, "A registered adapter for the observed language/framework.", "unsupported language",
			"Unsupported surfaces cannot be validated, so their maturity is unknown rather than proven.",
			"Add support upstream in Code Facts/Unit Health or convert the surface to a supported language.")
	}
	return resolution, diagnostics
}

func resolveNodeWorkspace(input WorkspaceInput, resolution *WorkspaceResolution, diagnostics *[]Diagnostic) {
	resolution.CanonicalFramework = "vitest"
	genericNode := strings.EqualFold(input.Language, "javascript") || strings.EqualFold(input.Language, "node")
	manifest, err := loadNodeManifest(input.Surface.RootPath)
	if err != nil {
		resolution.Status = "degraded"
		resolution.DegradedReason = "package.json could not be read or parsed."
		*diagnostics = append(*diagnostics, Diagnostic{Code: diagnosticMisconfiguration, Category: "config", Severity: severity(diagnosticMisconfiguration), File: filepath.Join(input.Surface.RootPath, "package.json"), Message: "package.json for the UI surface could not be read or parsed.", Evidence: err.Error(), Expected: "A readable, valid package.json.", Observed: "unreadable/invalid package.json", WhyItMatters: "Without a manifest Unit Health cannot determine the test framework or commands.", Remediation: "Fix package.json so the workspace can be parsed."})
		return
	}
	pm := input.Surface.PackageManager
	if pm == "" {
		pm = "pnpm"
	} else if normalized := normalizePackageManager(pm); normalized != "" {
		// Code Facts may report the manifest's versioned package-manager
		// identity (for example, pnpm@9.12.1).  Test execution needs the
		// executable name, while discovery deliberately preserves the raw
		// observation for evidence and display.
		pm = normalized
	}
	resolution.PackageManager = pm
	hasVitest, hasJest := manifest.hasDependency("vitest"), manifest.hasDependency("jest")
	hasTest, hasCoverage := manifest.hasScript("test"), manifest.hasScript("test:coverage")
	hasCoverageConfig := hasCoverage || viteCoverageConfigured(input.Surface.RootPath)
	add := func(code, message, evidence, expected, observed, why, remediation string) {
		*diagnostics = append(*diagnostics, Diagnostic{Code: code, Category: "config", Severity: severity(code), File: filepath.Join(input.Surface.RootPath, "package.json"), Message: message, Evidence: evidence, Expected: expected, Observed: observed, WhyItMatters: why, Remediation: remediation})
	}
	switch {
	case !hasTest && !hasVitest && !hasJest:
		add(diagnosticSurfaceMissing, "UI surface has no test script and no test framework configured.", "no \"test\" script; neither vitest nor jest in dependencies", "A Vitest test script (e.g. \"test\": \"vitest run\").", "missing test framework", "Without a test framework the UI cannot be unit-tested at all.", "Add Vitest and a \"test\" script to package.json.")
		resolution.Status, resolution.DegradedReason = "degraded", "no test framework configured"
	case hasJest && !hasVitest:
		add(diagnosticNoncanonical, "UI surface uses Jest; Vrooli React/Vite scenarios should use Vitest.", "jest present in dependencies; vitest absent", "Vitest as the canonical React/Vite test framework.", "jest (noncanonical)", "Fragmenting between Jest and Vitest blocks shared tooling and Vite-native coverage.", "Migrate the UI test suite to Vitest (see the test skill).")
		resolution.Framework, resolution.TestCommand = "jest", pm+" test"
		if genericNode {
			*diagnostics = (*diagnostics)[:len(*diagnostics)-1]
			resolution.CanonicalFramework, resolution.Status = "jest", "ready"
		} else {
			resolution.Status, resolution.DegradedReason = "degraded", "noncanonical test framework (jest)"
		}
	case !hasTest:
		add(diagnosticSurfaceMissing, "UI surface has Vitest available but no \"test\" script.", "vitest present; no \"test\" script", "A \"test\" script wired to Vitest.", "missing test script", "Without a test script the canonical runner cannot be invoked.", "Add \"test\": \"vitest run\" to package.json scripts.")
		resolution.Framework, resolution.Status, resolution.DegradedReason = "vitest", "degraded", "missing test script"
	default:
		resolution.Framework, resolution.Status = "vitest", "ready"
		resolution.TestCommand = pm + " test"
	}
	if resolution.TestCommand != "" && !hasCoverageConfig && !genericNode {
		add(diagnosticCoverage, "UI surface has no coverage script or Vite coverage configuration.", "no \"test:coverage\" script; no coverage block in vite config", "A coverage-capable Vitest configuration (e.g. \"test:coverage\": \"vitest run --coverage\").", "missing coverage config", "Coverage cannot be measured, so hardening depth is invisible.", "Add a \"test:coverage\" script and configure V8 coverage in vite.config.")
	}
	if hasCoverage {
		resolution.CoverageRequested = true
		resolution.CoverageCommand = pm + " test:coverage"
	}
	if resolution.TestCommand != "" && input.Surface.PackageManager != "" && normalizePackageManager(manifest.PackageManager) != "" && normalizePackageManager(manifest.PackageManager) != pm {
		declared, observed := normalizePackageManager(manifest.PackageManager), pm
		add(diagnosticPackageManager, fmt.Sprintf("Declared package manager (%s) does not match the lockfile (%s).", declared, observed), fmt.Sprintf("packageManager=%s; lockfile implies %s", declared, observed), "Declared package manager matching the committed lockfile.", fmt.Sprintf("declared=%s, lockfile=%s", declared, observed), "A mismatch causes inconsistent installs between environments and CI.", "Align the packageManager field with the committed lockfile.")
	}
	if resolution.TestCommand != "" && input.ResolveExecutable != nil {
		if executable, ok := input.ResolveExecutable([]string{pm}); ok {
			resolution.TestExecutable = executable
		} else {
			resolution.Status = "degraded"
			resolution.DegradedReason = "The declared Node package manager is not resolvable on this host."
			add(diagnosticDependency, "Node workspace was discovered but its package manager is not resolvable.", "package manager="+pm+"; executable not found", "The declared package manager installed and resolvable.", "missing "+pm, "Without the package manager the Node test plan cannot execute.", "Install or expose the declared package manager through Scenario Dependency Analyzer, then rerun Unit Health.")
		}
	}
}

func resolveBashWorkspace(input WorkspaceInput, resolution *WorkspaceResolution, add func(string, string, string, string, string, string, string, string)) {
	resolution.Framework, resolution.CanonicalFramework = "bats", "bats"
	if !hasFilesWithExt(input.Surface.RootPath, ".bats") {
		resolution.Status, resolution.DegradedReason = "degraded", "shell scripts present but no bats test files (*.bats) found"
		add(diagnosticSurfaceMissing, input.Surface.RootPath, "Shell surface has no bats (*.bats) tests.", "shell sources present; no *.bats files under "+input.Surface.RootPath, "At least one bats test file exercising the shell scripts.", "no bats tests", "Untested shell scripts silently break; bats is the canonical Vrooli shell unit-test framework.", "Add bats tests (e.g. test/*.bats) covering the shell entrypoints.")
		return
	}
	resolve := input.ResolveExecutable
	if resolve == nil {
		resolve = hostbin.Resolve
	}
	bin, ok := resolve([]string{"bats"})
	if !ok {
		resolution.Status, resolution.DegradedReason = "degraded", "bats is not installed on this host"
		add(diagnosticDependency, input.Surface.RootPath, "bats test files exist but the bats runner is not installed.", "*.bats files present; no bats binary on PATH or in the user's bin dirs", "The bats runner installed and resolvable.", "bats not installed", "Without bats the shell tests cannot run, so the shell surface is unvalidated.", "Request governed bats provisioning through Scenario Dependency Analyzer, then rerun Unit Health; Unit Health does not install tools.")
		return
	}
	resolution.Status, resolution.TestExecutable = "ready", bin
}

type nodeManifest struct {
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	PackageManager  string            `json:"packageManager"`
}

func (m nodeManifest) hasDependency(name string) bool {
	_, a := m.Dependencies[name]
	_, b := m.DevDependencies[name]
	return a || b
}
func (m nodeManifest) hasScript(name string) bool { _, ok := m.Scripts[name]; return ok }

func loadNodeManifest(root string) (nodeManifest, error) {
	raw, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return nodeManifest{}, err
	}
	var manifest nodeManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nodeManifest{}, err
	}
	return manifest, nil
}

func viteCoverageConfigured(root string) bool {
	for _, name := range []string{"vite.config.ts", "vite.config.js", "vitest.config.ts", "vitest.config.js"} {
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err == nil && strings.Contains(string(raw), "coverage") {
			return true
		}
	}
	return false
}

func pythonCoverageConfigured(root string) bool {
	for _, name := range []string{"pyproject.toml", "pytest.ini", "setup.cfg", "tox.ini", "requirements.txt", "requirements-dev.txt"} {
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err == nil {
			contents := strings.ToLower(string(raw))
			if strings.Contains(contents, "pytest-cov") || strings.Contains(contents, "--cov") {
				return true
			}
		}
	}
	return false
}

func normalizePackageManager(declared string) string {
	declared = strings.ToLower(strings.TrimSpace(declared))
	if at := strings.IndexByte(declared, '@'); at > 0 {
		declared = declared[:at]
	}
	switch declared {
	case "pnpm", "npm", "yarn":
		return declared
	default:
		return ""
	}
}

func firstTestFile(root, suffix string) string {
	var found string
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || found != "" {
			return nil
		}
		if entry.IsDir() {
			if ignoredDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(strings.ToLower(filepath.Base(path)), strings.ToLower(suffix)) {
			found = path
		}
		return nil
	})
	return found
}

func hasFilesWithExt(root, ext string) bool {
	found := false
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if entry.IsDir() {
			if ignoredDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(strings.ToLower(filepath.Base(path)), strings.ToLower(ext)) {
			found = true
		}
		return nil
	})
	return found
}

func ignoredDir(name string) bool {
	switch name {
	case ".git", "node_modules", "dist", "build", ".cache", "coverage", "vendor":
		return true
	default:
		return false
	}
}

func severity(code string) string {
	switch code {
	case diagnosticSurfaceMissing, diagnosticMisconfiguration, diagnosticDependency:
		return "error"
	case diagnosticNoncanonical, diagnosticCoverage:
		return "error"
	case diagnosticPackageManager:
		return "warning"
	default:
		return "info"
	}
}
