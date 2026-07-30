package validation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"unit-health/internal/discovery"
	"unit-health/internal/hostbin"
)

// Unit Health finding codes. These are the contract between the engine and
// `.vrooli/maturity.json`; every code emitted here has a mapping there.
const (
	codeTestSurfaceAbsent       = "TEST_SURFACE_ABSENT"
	codeUnsupportedParseUnit    = "UNSUPPORTED_PARSE_UNIT"
	codeTestFrameworkMissing    = "TEST_FRAMEWORK_MISSING"
	codeTestFrameworkNoncanon   = "TEST_FRAMEWORK_NONCANONICAL"
	codeCoverageConfigMissing   = "COVERAGE_CONFIG_MISSING"
	codePackageManagerMismatch  = "PACKAGE_MANAGER_MISMATCH"
	codeTestMisconfiguration    = "TEST_MISCONFIGURATION"
	codeTestExecutionFailure    = "TEST_EXECUTION_FAILURE"
	codeTestDependencyMissing   = "TEST_DEPENDENCY_MISSING"
	codeTestTimeoutHang         = "TEST_TIMEOUT_HANG"
	codeUnitPolicyInvalid       = "UNIT_POLICY_PROFILE_INVALID"
	codeUnitRequiredRoleMissing = "UNIT_REQUIRED_ROLE_MISSING"
	codeUnitSurfaceUngoverned   = "UNIT_SURFACE_UNGOVERNED"
	codeUnitPolicyWeakened      = "UNIT_POLICY_WEAKENED"
	codeUnitWaiverInvalid       = "UNIT_POLICY_WAIVER_INVALID"
	codeUnitProjectionDrift     = "UNIT_POLICY_PROJECTION_DRIFT"
)

// codeSeverity mirrors the severity_default of each code in
// `.vrooli/maturity.json`. Findings carry their own severity so renderers and
// counts do not need the spec; the maturity assessor cross-checks the mapping.
var codeSeverity = map[string]string{
	codeTestSurfaceAbsent:       "error",
	codeUnsupportedParseUnit:    "info",
	codeTestFrameworkMissing:    "error",
	codeTestFrameworkNoncanon:   "error",
	codeCoverageConfigMissing:   "error",
	codePackageManagerMismatch:  "warning",
	codeTestMisconfiguration:    "warning",
	codeTestExecutionFailure:    "error",
	codeTestDependencyMissing:   "error",
	codeTestTimeoutHang:         "error",
	codeUnitPolicyInvalid:       "error",
	codeUnitRequiredRoleMissing: "error",
	codeUnitSurfaceUngoverned:   "warning",
	codeUnitPolicyWeakened:      "error",
	codeUnitWaiverInvalid:       "error",
	codeUnitProjectionDrift:     "error",

	// Phase 5 analyzer codes (coverage, architecture, quality, diagnostics).
	codeLowCoverage:             "warning",
	codeCoverageAbsent:          "info",
	codeTestNotColocated:        "warning",
	codeTestUtilMissing:         "warning",
	codeTestHelperFromProd:      "error",
	codeMissingInjectableSeam:   "warning",
	codeTestSkippedOrOnly:       "warning",
	codeTestNoAssertion:         "warning",
	codeTestRenderOnly:          "warning",
	codeTestExcessiveSnapshots:  "info",
	codeTestMissingEdgeCases:    "warning",
	codeTestFlakeSuspected:      "warning",
	codeTestRuntimeGrowth:       "info",
	codeTestUntaggedRequirement: "warning",
}

// defaultWorkspaceTimeoutSeconds bounds each planned command. The bounded
// executor (Phase 4) enforces it; the plan surfaces it so operators see the cap.
const defaultWorkspaceTimeoutSeconds = 600

// nodeManifest is the subset of package.json Unit Health needs to choose a
// canonical framework and detect degraded states.
type nodeManifest struct {
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	PackageManager  string            `json:"packageManager"`
}

func (m nodeManifest) hasDep(name string) bool {
	if _, ok := m.Dependencies[name]; ok {
		return true
	}
	_, ok := m.DevDependencies[name]
	return ok
}

func (m nodeManifest) hasScript(name string) bool {
	_, ok := m.Scripts[name]
	return ok
}

// buildPlan turns a discovery inventory into the surface list, the testable
// workspace list, the dry-run execution plan, and the discovery/config-gap
// findings. It performs no execution.
func buildPlan(scenario string, inv discovery.Inventory, now string) ([]Surface, []Workspace, ExecutionPlan, []Finding) {
	surfaces := make([]Surface, 0, len(inv.Surfaces))
	for _, s := range inv.Surfaces {
		if strings.EqualFold(s.Status, "missing") {
			continue
		}
		surfaces = append(surfaces, Surface{
			ID:             s.ID,
			Kind:           s.Kind,
			Language:       s.Language,
			Framework:      s.Framework,
			RootPath:       s.RootPath,
			PackageManager: s.PackageManager,
			Status:         s.Status,
			Confidence:     s.Confidence,
		})
	}

	findings := resolveUnitPolicyFindings(scenario, inv, now)
	if len(inv.Surfaces) == 0 {
		reason := inv.DegradedReason
		if reason == "" {
			reason = "no testable surface discovered"
		}
		findings = append(findings, Finding{
			ID:           codeTestSurfaceAbsent,
			Scenario:     scenario,
			Code:         codeTestSurfaceAbsent,
			Category:     "discovery",
			Severity:     codeSeverity[codeTestSurfaceAbsent],
			Message:      "No testable workspace (Go module, TypeScript/Vite UI, or Python package) could be discovered for the scenario.",
			Evidence:     reason,
			Expected:     "At least one discoverable, runnable test workspace.",
			Observed:     reason,
			WhyItMatters: "Without a discoverable test surface the scenario cannot be validated or hardened.",
			Remediation:  "Add a Go module, a Vitest-configured UI, or a Python package with tests, and ensure Code Facts can describe it.",
			CreatedAt:    now,
		})
		return surfaces, nil, ExecutionPlan{Notes: "No workspaces to plan."}, findings
	}

	workspaces := make([]Workspace, 0, len(inv.Surfaces))
	seen := map[string]struct{}{}
	for _, s := range inv.Surfaces {
		if strings.EqualFold(s.Status, "missing") {
			continue
		}
		lang := normalizeLanguage(s.Language, s.RootPath)
		key := lang + "|" + filepath.Clean(s.RootPath)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		ws, wsFindings := resolveWorkspace(scenario, s, lang, now)
		workspaces = append(workspaces, ws)
		findings = append(findings, wsFindings...)
	}

	sort.SliceStable(workspaces, func(i, j int) bool {
		if workspaces[i].Language != workspaces[j].Language {
			return workspaces[i].Language < workspaces[j].Language
		}
		return workspaces[i].RootPath < workspaces[j].RootPath
	})

	plan := buildExecutionPlan(workspaces)
	return surfaces, workspaces, plan, findings
}

func normalizeLanguage(lang, root string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	switch lang {
	case "go", "golang":
		return "go"
	case "typescript", "ts", "tsx", "javascript", "js", "jsx", "node":
		return "typescript"
	case "python", "py":
		return "python"
	case "bash", "sh", "shell":
		return "bash"
	case "", "unknown":
		// Fall back to filesystem evidence.
		switch {
		case fileExists(filepath.Join(root, "go.mod")):
			return "go"
		case fileExists(filepath.Join(root, "package.json")):
			return "typescript"
		case hasPythonFiles(root):
			return "python"
		case hasShellFiles(root):
			return "bash"
		}
		return "unknown"
	default:
		return lang
	}
}

// resolveWorkspace derives a testable workspace and its config-gap findings for
// one surface, choosing the canonical Vrooli framework per language.
func resolveWorkspace(scenario string, s discovery.Surface, lang, now string) (Workspace, []Finding) {
	ws := Workspace{
		ID:             s.ID,
		Language:       lang,
		RootPath:       s.RootPath,
		Framework:      s.Framework,
		PackageManager: s.PackageManager,
	}
	switch lang {
	case "go":
		ws.CanonicalFramework = "go test"
		ws.Framework = "go test"
		ws.TestCommand = "go test ./..."
		ws.CoverageCommand = "go test -covermode=atomic -coverprofile=coverage.out ./..."
		ws.Status = "ready"
		return ws, nil
	case "typescript":
		return resolveNodeWorkspace(scenario, s, ws, now)
	case "bash":
		return resolveBashWorkspace(scenario, s, ws, now)
	case "python":
		ws.CanonicalFramework = "pytest"
		ws.Framework = "pytest"
		ws.TestCommand = "python3 -m pytest -q"
		ws.Status = "degraded"
		ws.DegradedReason = "Code Facts does not yet provide Python parse units; using fallback discovery."
		return ws, []Finding{{
			ID:           codeUnsupportedParseUnit + "-" + s.ID,
			Scenario:     scenario,
			SurfaceID:    s.ID,
			WorkspaceID:  s.ID,
			Language:     lang,
			Code:         codeUnsupportedParseUnit,
			Category:     "discovery",
			Severity:     codeSeverity[codeUnsupportedParseUnit],
			FilePath:     s.RootPath,
			Message:      "Python workspace discovered through filesystem fallback; Code Facts does not yet describe Python parse units.",
			Evidence:     "language=python, code-facts parse units unavailable",
			Expected:     "Code Facts parse-unit coverage for Python.",
			Observed:     "fallback discovery only",
			WhyItMatters: "Fallback discovery is coarser than Code Facts and may miss workspaces or misclassify frameworks.",
			Remediation:  "Track Code Facts Python parse-unit support; until then verify the pytest plan manually.",
			CreatedAt:    now,
		}}
	default:
		ws.CanonicalFramework = ""
		ws.Status = "unsupported"
		ws.DegradedReason = "No canonical test framework is known for this language."
		return ws, []Finding{{
			ID:           codeUnsupportedParseUnit + "-" + s.ID,
			Scenario:     scenario,
			SurfaceID:    s.ID,
			WorkspaceID:  s.ID,
			Language:     lang,
			Code:         codeUnsupportedParseUnit,
			Category:     "discovery",
			Severity:     codeSeverity[codeUnsupportedParseUnit],
			FilePath:     s.RootPath,
			Message:      fmt.Sprintf("Surface %q has an unsupported or unknown language (%q); Unit Health cannot plan tests for it.", s.ID, lang),
			Evidence:     "language=" + lang,
			Expected:     "A Go, TypeScript/Vite, or Python workspace.",
			Observed:     "unsupported language",
			WhyItMatters: "Unsupported surfaces cannot be validated, so their maturity is unknown rather than proven.",
			Remediation:  "Add support upstream in Code Facts/Unit Health or convert the surface to a supported language.",
			CreatedAt:    now,
		}}
	}
}

// resolveNodeWorkspace inspects package.json to choose Vitest as canonical and
// emit findings for missing test scripts, noncanonical frameworks, missing
// coverage config, and package-manager mismatches.
func resolveNodeWorkspace(scenario string, s discovery.Surface, ws Workspace, now string) (Workspace, []Finding) {
	ws.CanonicalFramework = "vitest"
	pm := s.PackageManager
	if pm == "" {
		pm = "pnpm"
	}
	ws.PackageManager = pm

	mkFinding := func(code, message, evidence, expected, observed, why, remediation string) Finding {
		return Finding{
			ID:           code + "-" + s.ID,
			Scenario:     scenario,
			SurfaceID:    s.ID,
			WorkspaceID:  s.ID,
			Language:     "typescript",
			Framework:    s.Framework,
			Code:         code,
			Category:     "config",
			Severity:     codeSeverity[code],
			FilePath:     filepath.Join(s.RootPath, "package.json"),
			Message:      message,
			Evidence:     evidence,
			Expected:     expected,
			Observed:     observed,
			WhyItMatters: why,
			Remediation:  remediation,
			CreatedAt:    now,
		}
	}

	manifest, err := loadNodeManifest(s.RootPath)
	if err != nil {
		ws.Status = "degraded"
		ws.DegradedReason = "package.json could not be read or parsed."
		return ws, []Finding{mkFinding(codeTestMisconfiguration,
			"package.json for the UI surface could not be read or parsed.",
			err.Error(),
			"A readable, valid package.json.",
			"unreadable/invalid package.json",
			"Without a manifest Unit Health cannot determine the test framework or commands.",
			"Fix package.json so the workspace can be parsed.",
		)}
	}

	var findings []Finding

	hasVitest := manifest.hasDep("vitest")
	hasJest := manifest.hasDep("jest")
	hasTestScript := manifest.hasScript("test")
	hasCoverageScript := manifest.hasScript("test:coverage")
	hasCoverageConfig := hasCoverageScript || viteCoverageConfigured(s.RootPath)

	switch {
	case !hasTestScript && !hasVitest && !hasJest:
		findings = append(findings, mkFinding(codeTestFrameworkMissing,
			"UI surface has no test script and no test framework configured.",
			"no \"test\" script; neither vitest nor jest in dependencies",
			"A Vitest test script (e.g. \"test\": \"vitest run\").",
			"missing test framework",
			"Without a test framework the UI cannot be unit-tested at all.",
			"Add Vitest and a \"test\" script to package.json.",
		))
		ws.Status = "degraded"
		ws.DegradedReason = "no test framework configured"
	case hasJest && !hasVitest:
		findings = append(findings, mkFinding(codeTestFrameworkNoncanon,
			"UI surface uses Jest; Vrooli React/Vite scenarios should use Vitest.",
			"jest present in dependencies; vitest absent",
			"Vitest as the canonical React/Vite test framework.",
			"jest (noncanonical)",
			"Fragmenting between Jest and Vitest blocks shared tooling and Vite-native coverage.",
			"Migrate the UI test suite to Vitest (see the test skill).",
		))
		ws.Framework = "jest"
		ws.TestCommand = pm + " test"
		ws.Status = "degraded"
		ws.DegradedReason = "noncanonical test framework (jest)"
	case !hasTestScript:
		findings = append(findings, mkFinding(codeTestFrameworkMissing,
			"UI surface has Vitest available but no \"test\" script.",
			"vitest present; no \"test\" script",
			"A \"test\" script wired to Vitest.",
			"missing test script",
			"Without a test script the canonical runner cannot be invoked.",
			"Add \"test\": \"vitest run\" to package.json scripts.",
		))
		ws.Framework = "vitest"
		ws.Status = "degraded"
		ws.DegradedReason = "missing test script"
	default:
		ws.Framework = "vitest"
		ws.TestCommand = pm + " test"
		ws.Status = "ready"
	}

	// A runnable node workspace needs its dependencies installed before vitest;
	// a clean checkout has no node_modules. Resolve a lockfile-frozen install
	// cross-platform so a missing install classifies as a dependency gap rather
	// than a test misconfiguration. If no package-manager binary is available on
	// this host we leave InstallCommand empty and note the degrade (the test may
	// still run against pre-existing node_modules) rather than block.
	if ws.TestCommand != "" {
		if install, ok := nodeInstallCommand(pm); ok {
			ws.InstallCommand = install
		} else {
			ws.DegradedReason = strings.TrimSpace(ws.DegradedReason + " (no pnpm/yarn/npm binary found to install dependencies before tests)")
		}
	}

	if ws.TestCommand != "" {
		if hasCoverageScript {
			ws.CoverageCommand = pm + " test:coverage"
		} else if !hasCoverageConfig {
			findings = append(findings, mkFinding(codeCoverageConfigMissing,
				"UI surface has no coverage script or Vite coverage configuration.",
				"no \"test:coverage\" script; no coverage block in vite config",
				"A coverage-capable Vitest configuration (e.g. \"test:coverage\": \"vitest run --coverage\").",
				"missing coverage config",
				"Coverage cannot be measured, so hardening depth is invisible.",
				"Add a \"test:coverage\" script and configure V8 coverage in vite.config.",
			))
		}
	}

	if declared := normalizePackageManager(manifest.PackageManager); declared != "" && s.PackageManager != "" && declared != s.PackageManager {
		findings = append(findings, mkFinding(codePackageManagerMismatch,
			fmt.Sprintf("Declared package manager (%s) does not match the lockfile (%s).", declared, s.PackageManager),
			fmt.Sprintf("packageManager=%s; lockfile implies %s", declared, s.PackageManager),
			"Declared package manager matching the committed lockfile.",
			fmt.Sprintf("declared=%s, lockfile=%s", declared, s.PackageManager),
			"A mismatch causes inconsistent installs between environments and CI.",
			"Align the packageManager field with the committed lockfile.",
		))
	}

	return ws, findings
}

// installBinaryResolver picks the first available install binary among the
// candidate package managers, cross-platform (it probes the invoking user's
// ~/.local/bin, ~/go/bin, ~/bin too, not just the sudo PATH). It is a seam so
// tests can drive the resolution deterministically.
var installBinaryResolver = hostbin.Resolve

// nodeInstallCommand resolves a lockfile-frozen dependency install for a node
// workspace, preferring the lockfile's package manager and falling back through
// pnpm→yarn→npm. It returns false when no package-manager binary is available.
func nodeInstallCommand(preferred string) (string, bool) {
	bin, ok := installBinaryResolver(installCandidateOrder(preferred))
	if !ok {
		return "", false
	}
	switch bin {
	case "pnpm":
		// --ignore-scripts: a unit-health install must not run arbitrary
		// postinstall scripts from the workspace under validation.
		return "pnpm install --frozen-lockfile --ignore-scripts", true
	case "yarn":
		return "yarn install --frozen-lockfile --ignore-scripts", true
	case "npm":
		return "npm ci --ignore-scripts", true
	default:
		return "", false
	}
}

// installCandidateOrder yields the preferred package manager first (when known
// and supported) followed by the canonical fallback chain, de-duplicated.
func installCandidateOrder(preferred string) []string {
	order := make([]string, 0, 3)
	seen := map[string]struct{}{}
	add := func(name string) {
		switch name {
		case "pnpm", "yarn", "npm":
			if _, ok := seen[name]; !ok {
				seen[name] = struct{}{}
				order = append(order, name)
			}
		}
	}
	add(normalizePackageManager(preferred))
	add(preferred)
	add("pnpm")
	add("yarn")
	add("npm")
	return order
}

// batsBinaryResolver resolves the bats binary cross-platform. Seam for tests.
var batsBinaryResolver = hostbin.Resolve

// resolveBashWorkspace plans a Bash test surface. The canonical Vrooli shell
// unit-test framework is bats. It emits TEST_FRAMEWORK_MISSING when shell
// sources exist but no `*.bats` tests do, and TEST_DEPENDENCY_MISSING when bats
// itself is not installed (an explicit degrade, never a false pass — including
// on platforms without bats such as bare Windows).
func resolveBashWorkspace(scenario string, s discovery.Surface, ws Workspace, now string) (Workspace, []Finding) {
	ws.CanonicalFramework = "bats"
	ws.Framework = "bats"

	mkFinding := func(code, message, evidence, expected, observed, why, remediation string) Finding {
		return Finding{
			ID:           code + "-" + s.ID,
			Scenario:     scenario,
			SurfaceID:    s.ID,
			WorkspaceID:  s.ID,
			Language:     "bash",
			Framework:    "bats",
			Code:         code,
			Category:     "config",
			Severity:     codeSeverity[code],
			FilePath:     s.RootPath,
			Message:      message,
			Evidence:     evidence,
			Expected:     expected,
			Observed:     observed,
			WhyItMatters: why,
			Remediation:  remediation,
			CreatedAt:    now,
		}
	}

	if !hasBatsFiles(s.RootPath) {
		ws.Status = "degraded"
		ws.DegradedReason = "shell scripts present but no bats test files (*.bats) found"
		return ws, []Finding{mkFinding(codeTestFrameworkMissing,
			"Shell surface has shell scripts but no bats (*.bats) tests.",
			"shell sources present; no *.bats files under "+s.RootPath,
			"At least one bats test file exercising the shell scripts.",
			"no bats tests",
			"Untested shell scripts (CLI scaffolding, lifecycle helpers) silently break; bats is the canonical Vrooli shell unit-test framework.",
			"Add bats tests (e.g. test/*.bats) covering the shell entrypoints.",
		)}
	}

	bin, ok := batsBinaryResolver([]string{"bats"})
	if !ok {
		ws.Status = "degraded"
		ws.DegradedReason = "bats is not installed on this host"
		return ws, []Finding{mkFinding(codeTestDependencyMissing,
			"bats test files exist but the bats runner is not installed.",
			"*.bats files present; no bats binary on PATH or in the user's bin dirs",
			"The bats runner installed and resolvable.",
			"bats not installed",
			"Without bats the shell tests cannot run, so the shell surface is unvalidated.",
			"Install bats (a registered Vrooli tool) so unit-health can run the shell tests.",
		)}
	}

	// bats --recursive runs every *.bats under the working directory. The
	// executor runs this with Dir set to the surface root, so target ".".
	ws.TestCommand = bin + " --recursive ."
	ws.Status = "ready"
	return ws, nil
}

func buildExecutionPlan(workspaces []Workspace) ExecutionPlan {
	plan := ExecutionPlan{}
	for _, ws := range workspaces {
		command := ws.CoverageCommand
		if command == "" {
			command = ws.TestCommand
		}
		if command == "" {
			continue
		}
		if ws.InstallCommand != "" {
			plan.Commands = append(plan.Commands, PlannedCommand{
				WorkspaceID:      ws.ID,
				Name:             ws.Language + " install",
				Command:          ws.InstallCommand,
				WorkingDirectory: ws.RootPath,
				TimeoutSeconds:   defaultWorkspaceTimeoutSeconds,
				Kind:             kindInstall,
			})
		}
		plan.Commands = append(plan.Commands, PlannedCommand{
			WorkspaceID:      ws.ID,
			Name:             ws.Language + " test",
			Command:          command,
			WorkingDirectory: ws.RootPath,
			TimeoutSeconds:   defaultWorkspaceTimeoutSeconds,
			Kind:             kindTest,
		})
	}
	switch len(plan.Commands) {
	case 0:
		plan.Notes = "No runnable test commands; resolve the configuration findings first."
	default:
		plan.Notes = fmt.Sprintf("Dry run: %d command(s) planned. Execution is bounded per-workspace and runs only with --include-execution.", len(plan.Commands))
	}
	return plan
}

func loadNodeManifest(root string) (nodeManifest, error) {
	raw, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return nodeManifest{}, err
	}
	var m nodeManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nodeManifest{}, err
	}
	return m, nil
}

// viteCoverageConfigured reports whether a vite/vitest config mentions coverage.
func viteCoverageConfigured(root string) bool {
	for _, name := range []string{"vite.config.ts", "vite.config.js", "vitest.config.ts", "vitest.config.js"} {
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			continue
		}
		if strings.Contains(string(raw), "coverage") {
			return true
		}
	}
	return false
}

func normalizePackageManager(declared string) string {
	declared = strings.TrimSpace(strings.ToLower(declared))
	if declared == "" {
		return ""
	}
	// packageManager is "pnpm@9.1.0" style; take the tool name.
	if idx := strings.IndexByte(declared, '@'); idx > 0 {
		declared = declared[:idx]
	}
	switch declared {
	case "pnpm", "npm", "yarn":
		return declared
	default:
		return ""
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func hasPythonFiles(root string) bool {
	for _, name := range []string{"pyproject.toml", "setup.py", "setup.cfg", "requirements.txt"} {
		if fileExists(filepath.Join(root, name)) {
			return true
		}
	}
	return false
}

// hasShellFiles reports whether the tree rooted at root contains any .sh source.
func hasShellFiles(root string) bool { return hasFilesWithExt(root, ".sh") }

// hasBatsFiles reports whether the tree rooted at root contains any .bats test.
func hasBatsFiles(root string) bool { return hasFilesWithExt(root, ".bats") }

func hasFilesWithExt(root, ext string) bool {
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "dist", "build", ".cache", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ext) {
			found = true
		}
		return nil
	})
	return found
}
