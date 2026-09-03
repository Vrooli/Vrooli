package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"unit-health/internal/adapterregistry"
	"unit-health/internal/adapters"
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
	codeUnsupportedTargetKind   = "UNSUPPORTED_TARGET_KIND"
	codeUnitTestKindOutOfScope  = "UNIT_TEST_KIND_OUT_OF_SCOPE"
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
	codeUnsupportedTargetKind:   "error",
	codeUnitTestKindOutOfScope:  "warning",

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
	codeSeamDuplicatedInPackage: "error",
	codeSeamReimplemented:       "error",
	codeCompanionReimplemented:  "error",
	codeCompanionAvailable:      "info",
}

// defaultWorkspaceTimeoutSeconds bounds each planned command. The bounded
// executor (Phase 4) enforces it; the plan surfaces it so operators see the cap.
const defaultWorkspaceTimeoutSeconds = 600

// buildPlan turns a discovery inventory into the surface list, the testable
// workspace list, the dry-run execution plan, and the discovery/config-gap
// findings. It performs no execution.
func buildPlan(scenario string, inv discovery.Inventory, now string) ([]Surface, []Workspace, ExecutionPlan, []Finding) {
	inv = normalizeTargetInventory(inv)
	surfaces := make([]Surface, 0, len(inv.Surfaces))
	for _, s := range inv.Surfaces {
		if strings.EqualFold(s.Status, "missing") {
			continue
		}
		surfaces = append(surfaces, Surface{
			ID:                s.ID,
			Kind:              s.Kind,
			Language:          s.Language,
			Framework:         s.Framework,
			RootPath:          s.RootPath,
			PackageManager:    s.PackageManager,
			Status:            s.Status,
			Confidence:        s.Confidence,
			ToolchainIdentity: s.Toolchain.ToolchainIdentity,
		})
	}

	var findings []Finding
	if inv.TargetKind == "scenario" || inv.TargetKind == "" {
		findings = resolveUnitPolicyFindings(scenario, inv, now)
	}
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
			Remediation:  "Add a supported test surface with governed dependencies and ensure Code Facts can describe it.",
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
		if ws.Status == "ready" && ws.AdapterID == "" {
			ws.Status = "unsupported"
			ws.DegradedReason = fmt.Sprintf("no registered adapter supports language=%s framework=%s on %s", ws.Language, ws.Framework, runtime.GOOS)
			ws.TestCommand = ""
			ws.CoverageCommand = ""
			wsFindings = append(wsFindings, Finding{ID: codeUnsupportedParseUnit + "-adapter-" + ws.ID, Scenario: scenario, SurfaceID: ws.ID, WorkspaceID: ws.ID, Language: ws.Language, Framework: ws.Framework, Code: codeUnsupportedParseUnit, Category: "adapter", Severity: codeSeverity[codeUnsupportedParseUnit], FilePath: ws.RootPath, Message: "No registered versioned test adapter supports this language/framework on the current host.", Evidence: ws.DegradedReason, Expected: "A registered adapter with matching platform support.", Observed: "adapter resolution failed", WhyItMatters: "Unsupported combinations must remain explicit rather than receiving unrelated runner rules.", Remediation: "Add or select a versioned adapter that declares support for this host combination.", CreatedAt: now})
		}
		workspaces = append(workspaces, ws)
		findings = append(findings, wsFindings...)
	}

	sort.SliceStable(workspaces, func(i, j int) bool {
		if workspaces[i].Language != workspaces[j].Language {
			return workspaces[i].Language < workspaces[j].Language
		}
		return workspaces[i].RootPath < workspaces[j].RootPath
	})
	applyRunnerProfiles(inv.RootPath, workspaces)
	for index := range workspaces {
		if workspaces[index].TestKind == "integration" || workspaces[index].TestKind == "workflow" {
			findings = append(findings, unitTestKindFinding(scenario, workspaces[index], now))
			workspaces[index].TestCommand = ""
			workspaces[index].CoverageCommand = ""
		}
	}

	plan := buildExecutionPlan(workspaces)
	return surfaces, workspaces, plan, findings
}

func unitTestKindFinding(scenario string, workspace Workspace, now string) Finding {
	return Finding{
		ID: codeUnitTestKindOutOfScope + "-" + workspace.ID, Scenario: scenario, WorkspaceID: workspace.ID,
		Language: workspace.Language, Framework: workspace.Framework, Code: codeUnitTestKindOutOfScope,
		Category: "test-kind", Severity: codeSeverity[codeUnitTestKindOutOfScope], FilePath: workspace.RootPath,
		Message:  fmt.Sprintf("Workspace %q is classified as %s work and is excluded from unit execution.", workspace.ID, workspace.TestKind),
		Evidence: "test_kind=" + workspace.TestKind, Expected: "Unit execution contains only unit, component, or isolated repository tests.", Observed: workspace.TestKind,
		WhyItMatters: "Running integration or workflow tests in the unit phase makes results slow, shared-state dependent, and misleading.",
		Remediation:  "Move this work to the integration or workflow phase and retain a focused unit surface.", CreatedAt: now,
	}
}

var defaultPlannerRegistry = adapters.DefaultPlannerRegistry()

// External runner resolution remains injectable for deterministic adapter
// conformance tests; adapters receive a resolved executable rather than a
// command string or package-install request.
var externalRunnerBinaryResolver = hostbin.Resolve

func applyAdapterPlan(workspace *Workspace) {
	if workspace == nil || workspace.Language == "" || workspace.Framework == "" {
		return
	}
	// Python's filesystem fallback is intentionally degraded but still has a
	// runnable plan. Other degraded surfaces (for example a UI with no test
	// script or a shell tree without its declared runner) must not acquire a command that
	// their own readiness checks deliberately withheld.
	if workspace.Status != "ready" && workspace.Language != "python" && workspace.TestCommand == "" {
		return
	}
	coverageScript := workspace.CoverageCommand != ""
	plan, err := defaultPlannerRegistry.Resolve(adapters.Facts{
		Language:           workspace.Language,
		Framework:          workspace.Framework,
		PackageManager:     workspace.PackageManager,
		CoverageScript:     coverageScript,
		Platform:           runtime.GOOS,
		TestPath:           workspace.TestPath,
		Executable:         workspace.TestExecutable,
		CoverageExecutable: workspace.CoverageExecutable,
	})
	if err != nil {
		return
	}
	workspace.AdapterID = plan.Adapter.ID
	workspace.AdapterVersion = plan.Adapter.Version
	workspace.TestKind = plan.TestKind
	workspace.TestExecutable = plan.Test.Executable
	workspace.TestArgs = append([]string(nil), plan.Test.Args...)
	workspace.TestCommand = plan.Test.Display
	if plan.Coverage != nil {
		workspace.CoverageExecutable = plan.Coverage.Executable
		workspace.CoverageArgs = append([]string(nil), plan.Coverage.Args...)
		workspace.CoverageCommand = plan.Coverage.Display
		workspace.TestArtifacts = nil
		for _, artifact := range plan.Coverage.Artifacts {
			workspace.TestArtifacts = append(workspace.TestArtifacts, Artifact{Label: artifact.Label, Kind: artifact.Kind, Reference: artifact.Path})
		}
	}
}

// normalizeTargetInventory gives non-scenario targets a stable package-shaped
// workspace. Code Facts may describe a package as a collection of nested parse
// units; unit-health's package contract is deliberately narrower: one package
// target means one Go module rooted at that package. Scenario discovery keeps
// its existing surface model unchanged.
func normalizeTargetInventory(inv discovery.Inventory) discovery.Inventory {
	targetKind := strings.ToLower(strings.TrimSpace(inv.TargetKind))
	if targetKind != "package" && targetKind != "control-plane" {
		return inv
	}
	if targetKind == "package" && fileExists(filepath.Join(inv.RootPath, "go.mod")) {
		inv.Surfaces = []discovery.Surface{{
			ID: "package", Kind: "package", Language: "go", RootPath: inv.RootPath,
			Status: "known", Confidence: 1,
		}}
		return inv
	}
	if targetKind == "control-plane" && fileExists(filepath.Join(inv.RootPath, "go.mod")) {
		inv.Surfaces = []discovery.Surface{{
			ID: "control-plane", Kind: "control-plane", Language: "go", RootPath: inv.RootPath,
			Status: "known", Confidence: 1,
		}}
	}
	return inv
}

func unsupportedTargetFinding(scenario, targetKind, now string) Finding {
	return Finding{
		ID:           codeUnsupportedTargetKind + "-" + targetKind,
		Scenario:     scenario,
		Code:         codeUnsupportedTargetKind,
		Category:     "target",
		Severity:     codeSeverity[codeUnsupportedTargetKind],
		Message:      fmt.Sprintf("Unit Health does not implement target kind %q.", targetKind),
		Evidence:     "target_kind=" + targetKind,
		Expected:     "A provider implementation that declares and supports this target kind.",
		Observed:     "no unit-health analyzer for target kind " + targetKind,
		WhyItMatters: "Reporting a clean maturity rung for an unanalyzed target would be misleading.",
		Remediation:  "Use a provider that supports this target kind or add an explicit target-aware analyzer before declaring support.",
		CreatedAt:    now,
	}
}

func normalizeLanguage(lang, root string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	switch lang {
	case "go", "golang":
		return "go"
	case "typescript", "ts", "tsx", "jsx":
		return "typescript"
	case "javascript", "js":
		return "javascript"
	case "node":
		return "node"
	case "python", "py":
		return "python"
	case "bash", "sh", "shell":
		return "bash"
	case "rust", "rs":
		return "rust"
	case "powershell", "pwsh":
		return "powershell"
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
		case hasRustFiles(root):
			return "rust"
		case hasPowerShellFiles(root):
			return "powershell"
		}
		return "unknown"
	default:
		return lang
	}
}

// resolveWorkspace derives a testable workspace and its config-gap findings for
// one surface, choosing the canonical Vrooli framework per language.
func resolveWorkspace(scenario string, s discovery.Surface, lang, now string) (ws Workspace, findings []Finding) {
	resolution, diagnostics := adapters.ResolveWorkspace(adapters.WorkspaceInput{
		Surface: s, Language: lang, ResolveExecutable: externalRunnerBinaryResolver,
	})
	ws = Workspace{
		ID: s.ID, Language: lang, RootPath: s.RootPath, Framework: resolution.Framework,
		CanonicalFramework: resolution.CanonicalFramework, PackageManager: resolution.PackageManager,
		ToolchainIdentity: s.Toolchain.ToolchainIdentity, TestPath: resolution.TestPath,
		TestCommand: resolution.TestCommand, CoverageCommand: resolution.CoverageCommand,
		TestExecutable: resolution.TestExecutable, TestArgs: append([]string(nil), resolution.TestArgs...),
		TypecheckCommand: resolution.TypecheckCommand, TypecheckExecutable: resolution.TypecheckExecutable,
		TypecheckArgs:      append([]string(nil), resolution.TypecheckArgs...),
		CoverageExecutable: resolution.CoverageExecutable, CoverageArgs: append([]string(nil), resolution.CoverageArgs...),
		Status: resolution.Status, DegradedReason: resolution.DegradedReason,
	}
	if resolution.CoverageRequested && ws.CoverageCommand == "" {
		ws.CoverageCommand = "adapter coverage requested"
	}
	for _, diagnostic := range diagnostics {
		findings = append(findings, Finding{
			ID: diagnostic.Code + "-" + s.ID, Scenario: scenario, SurfaceID: s.ID, WorkspaceID: s.ID,
			Language: lang, Framework: ws.Framework, Code: diagnostic.Code, Category: diagnostic.Category,
			Severity: diagnostic.Severity, FilePath: diagnostic.File, Message: diagnostic.Message,
			Evidence: diagnostic.Evidence, Expected: diagnostic.Expected, Observed: diagnostic.Observed,
			WhyItMatters: diagnostic.WhyItMatters, Remediation: diagnostic.Remediation, CreatedAt: now,
		})
	}
	applyAdapterPlan(&ws)
	return ws, findings
}

func buildExecutionPlan(workspaces []Workspace) ExecutionPlan {
	return buildExecutionPlanForMode(workspaces, false)
}

func buildExecutionPlanForMode(workspaces []Workspace, fastTestOnly bool) ExecutionPlan {
	plan := ExecutionPlan{}
	for _, ws := range workspaces {
		command := ws.TestCommand
		if !fastTestOnly && ws.CoverageCommand != "" {
			command = ws.CoverageCommand
		}
		timeout := ws.TimeoutSeconds
		if timeout <= 0 {
			timeout = defaultWorkspaceTimeoutSeconds
		}
		if command != "" {
			executable := ws.TestExecutable
			args := ws.TestArgs
			artifacts := ws.TestArtifacts
			if fastTestOnly {
				artifacts = nil
			}
			if !fastTestOnly && ws.CoverageCommand != "" {
				executable = ws.CoverageExecutable
				args = ws.CoverageArgs
			}
			plan.Commands = append(plan.Commands, PlannedCommand{
				WorkspaceID:            ws.ID,
				Name:                   ws.Language + " test",
				Command:                command,
				Executable:             executable,
				Args:                   append([]string(nil), args...),
				Artifacts:              append([]Artifact(nil), artifacts...),
				Resource:               ws.Resource,
				WorkingDirectory:       ws.RootPath,
				TimeoutSeconds:         timeout,
				NoOutputTimeoutSeconds: ws.NoOutputTimeoutSeconds,
				Kind:                   kindTest,
				TestKind:               ws.TestKind,
				Hermetic:               ws.Hermetic,
			})
		}
		if ws.TypecheckCommand != "" {
			plan.Commands = append(plan.Commands, PlannedCommand{
				WorkspaceID: ws.ID, Name: ws.Language + " typecheck", Command: ws.TypecheckCommand,
				Executable: ws.TypecheckExecutable, Args: append([]string(nil), ws.TypecheckArgs...),
				WorkingDirectory: ws.RootPath, TimeoutSeconds: timeout,
				NoOutputTimeoutSeconds: ws.NoOutputTimeoutSeconds, Kind: kindTest, TestKind: "typecheck", Hermetic: ws.Hermetic,
			})
		}
	}
	switch len(plan.Commands) {
	case 0:
		plan.Notes = "No runnable test commands; resolve the configuration findings first."
	default:
		mode := "coverage-capable"
		if fastTestOnly {
			mode = "fast-test-only"
		}
		plan.Notes = fmt.Sprintf("Dry run: %d %s command(s) planned. Execution is bounded per-workspace and runs only with --include-execution.", len(plan.Commands), mode)
	}
	return plan
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

func hasRustFiles(root string) bool { return adapterregistry.HasMarkerFile(root, "rust") }

func hasPowerShellFiles(root string) bool { return hasFilesWithExt(root, ".ps1") }

// hasShellFiles reports whether the tree rooted at root contains any .sh source.
func hasShellFiles(root string) bool { return hasFilesWithExt(root, ".sh") }

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
