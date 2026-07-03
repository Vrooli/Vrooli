package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func buildProjectionChecks(scenarioRoot string, workspaces []Workspace) []ProjectionCheck {
	var checks []ProjectionCheck
	for _, ws := range workspaces {
		switch ws.Language {
		case "go":
			checks = append(checks, goProjectionChecks(scenarioRoot, ws)...)
		case "typescript":
			checks = append(checks, tsProjectionChecks(scenarioRoot, ws)...)
		}
	}
	sort.SliceStable(checks, func(i, j int) bool {
		if checks[i].WorkspaceID != checks[j].WorkspaceID {
			return checks[i].WorkspaceID < checks[j].WorkspaceID
		}
		return checks[i].Key < checks[j].Key
	})
	return checks
}

func goProjectionChecks(scenarioRoot string, ws Workspace) []ProjectionCheck {
	class := policyClassForWorkspace(scenarioRoot, ws)
	requiredRoots := class.TestUtils.RequiredRoots
	if len(requiredRoots) == 0 {
		requiredRoots = []string{filepath.Join(filepath.Base(ws.RootPath), "internal", "testutil")}
	}
	checks := make([]ProjectionCheck, 0, 2)
	for _, root := range requiredRoots {
		path := resolveProjectionPath(scenarioRoot, ws.RootPath, root)
		checks = append(checks, projectionCheck(ws, "testutil.root", "unit.policy_profile test_utils.required_roots", path, root, boolNative(fileOrDirExists(path)), fileOrDirExists(path),
			"Create the shared testutil root declared by the unit policy profile."))
	}
	if class.TestUtils.ProductionImportBan {
		want := "true"
		path := filepath.Join(ws.RootPath, "internal", "testutil", "no_prod_import_test.go")
		checks = append(checks, projectionCheck(ws, "testutil.production_import_ban", "go test native guard", path, want, boolNative(fileExists(path)), fileExists(path),
			"Add internal/testutil/no_prod_import_test.go so production code cannot import test helpers."))
	}
	return checks
}

func tsProjectionChecks(scenarioRoot string, ws Workspace) []ProjectionCheck {
	expect := resolveProjectionExpectation(ws)
	manifest, _ := loadNodeManifest(ws.RootPath)
	cfg := readFileString(filepath.Join(ws.RootPath, "vite.config.ts"))
	vitePath := filepath.Join(ws.RootPath, "vite.config.ts")
	if cfg == "" {
		cfg = readFileString(filepath.Join(ws.RootPath, "vite.config.js"))
		vitePath = filepath.Join(ws.RootPath, "vite.config.js")
	}
	eslintPath := filepath.Join(ws.RootPath, "eslint.config.js")
	eslint := readFileString(eslintPath)
	proj := parseViteProjection(cfg, eslint)
	class := policyClassForWorkspace(scenarioRoot, ws)

	checks := []ProjectionCheck{
		projectionCheck(ws, "runner.dependency", "package.json dependencies", filepath.Join(ws.RootPath, "package.json"), "vitest", boolNative(manifest.hasDep("vitest")), manifest.hasDep("vitest"),
			"Add vitest through Scenario Dependency Analyzer and keep the test scripts on Vitest."),
		projectionCheck(ws, "script.test", "package.json scripts", filepath.Join(ws.RootPath, "package.json"), "contains vitest", manifest.Scripts["test"], strings.Contains(manifest.Scripts["test"], "vitest"),
			"Set scripts.test to a Vitest command such as \"vitest run\"."),
		projectionCheck(ws, "script.coverage", "package.json scripts", filepath.Join(ws.RootPath, "package.json"), "contains vitest and coverage", manifest.Scripts["test:coverage"], strings.Contains(manifest.Scripts["test:coverage"], "vitest") && strings.Contains(manifest.Scripts["test:coverage"], "coverage"),
			"Set scripts.test:coverage to a Vitest coverage command such as \"vitest run --coverage\"."),
		projectionCheck(ws, "vitest.environment", "vite.config.ts test", vitePath, expect.vitestEnv, proj.environment, proj.environment == expect.vitestEnv,
			"Set test.environment to the policy-declared jsdom environment."),
		projectionCheck(ws, "vitest.setup_files", "vite.config.ts test", vitePath, strings.Join(expect.setupFiles, ","), strings.Join(proj.setupFiles, ","), containsAllSetupFiles(proj.setupFiles, expect.setupFiles),
			"Register the policy-declared setup file(s) in test.setupFiles."),
		projectionCheck(ws, "coverage.provider", "vite.config.ts test.coverage", vitePath, expect.coverageProvider, proj.coverageProvider, proj.coverageProvider == expect.coverageProvider,
			"Set coverage.provider to the policy-declared provider."),
		projectionCheck(ws, "coverage.reporters", "vite.config.ts test.coverage", vitePath, strings.Join(expect.coverageReporters, ","), strings.Join(proj.coverageReporters, ","), containsAllStrings(proj.coverageReporters, expect.coverageReporters),
			"Include every policy-declared reporter in coverage.reporter."),
		projectionCheck(ws, "coverage.include", "vite.config.ts test.coverage", vitePath, strings.Join(expect.coverageInclude, ","), strings.Join(proj.coverageInclude, ","), containsAllStrings(proj.coverageInclude, expect.coverageInclude),
			"Set coverage.include so coverage denominators stay scoped to production source files."),
		projectionCheck(ws, "coverage.exclude", "vite.config.ts test.coverage", vitePath, strings.Join(expect.coverageExclude, ","), strings.Join(proj.coverageExclude, ","), containsAllStrings(proj.coverageExclude, expect.coverageExclude),
			"Restore the canonical coverage.exclude entries for test scaffolding, generated files, boot files, and locale catalogs."),
		projectionCheck(ws, "coverage.report_on_failure", "vite.config.ts test.coverage", vitePath, "true", boolNative(proj.hasReportOnFailure && proj.reportOnFailure), proj.hasReportOnFailure && proj.reportOnFailure,
			"Set coverage.reportOnFailure to true so failed coverage runs remain interpretable."),
		projectionCheck(ws, "eslint.production_import_ban", "eslint.config.js no-restricted-imports", eslintPath, "test-utils and features/*/mocks banned from production", boolNative(proj.hasImportBanRule), proj.hasImportBanRule,
			"Restore no-restricted-imports patterns for src/test-utils and feature mocks."),
	}

	for _, key := range []string{"lines", "functions", "branches", "statements"} {
		value, ok := proj.thresholds[key]
		checks = append(checks, projectionCheck(ws, "coverage.threshold."+key, "vite.config.ts test.coverage.thresholds", vitePath, projectionFormatCoverageFloor(expect.coverageFloor), thresholdNative(value, ok), ok && value >= expect.coverageFloor,
			fmt.Sprintf("Restore the %s threshold to %.1f or higher.", key, expect.coverageFloor)))
	}
	for _, root := range class.TestUtils.RequiredRoots {
		path := resolveProjectionPath(scenarioRoot, ws.RootPath, root)
		checks = append(checks, projectionCheck(ws, "testutil.root", "unit.policy_profile test_utils.required_roots", path, root, boolNative(fileOrDirExists(path)), fileOrDirExists(path),
			"Create the shared UI test-utils root declared by the unit policy profile."))
	}
	if helper := strings.TrimSpace(class.TestUtils.CanonicalRenderHelper); helper != "" {
		path := resolveProjectionPath(scenarioRoot, ws.RootPath, helper)
		checks = append(checks, projectionCheck(ws, "testutil.canonical_render_helper", "unit.policy_profile test_utils.canonical_render_helper", path, helper, boolNative(fileExists(path)), fileExists(path),
			"Add src/test-utils/renderWithProviders.tsx and re-export it from src/test-utils/index.ts."))
	}
	return checks
}

func policyClassForWorkspace(scenarioRoot string, ws Workspace) unitPolicyClass {
	if scenarioRoot == "" {
		scenarioRoot = scenarioRootForWorkspace(ws.RootPath)
	}
	profile, _, ok, _ := loadUnitPolicyProfile("", scenarioRoot, "")
	if !ok {
		return defaultPolicyClass(ws)
	}
	for _, role := range profile.RequiredRoles {
		class, exists := profile.PolicyClasses[role.PolicyClass]
		if !exists || !pathMatches(role.Match.Path, ws.RootPath, scenarioRoot) {
			continue
		}
		return class
	}
	return defaultPolicyClass(ws)
}

func defaultPolicyClass(ws Workspace) unitPolicyClass {
	switch ws.Language {
	case "go":
		return unitPolicyClass{
			Language:  "go",
			Framework: "go test",
			TestUtils: unitTestUtilsPolicy{RequiredRoots: []string{filepath.Join(filepath.Base(ws.RootPath), "internal", "testutil")}, ProductionImportBan: true},
		}
	case "typescript":
		return unitPolicyClass{
			Language:  "typescript",
			Framework: "vitest",
			TestUtils: unitTestUtilsPolicy{RequiredRoots: []string{filepath.Join(filepath.Base(ws.RootPath), "src", "test-utils")}, ProductionImportBan: true, CanonicalRenderHelper: filepath.Join(filepath.Base(ws.RootPath), "src", "test-utils", "renderWithProviders.tsx")},
		}
	default:
		return unitPolicyClass{}
	}
}

func projectionCheck(ws Workspace, key, owner, file, policyValue, nativeValue string, pass bool, remediation string) ProjectionCheck {
	status := "pass"
	if !pass {
		status = "drift"
		if strings.TrimSpace(nativeValue) == "" || nativeValue == "false" {
			status = "missing"
		}
	}
	finding := ""
	if !pass {
		finding = codeUnitProjectionDrift
	}
	return ProjectionCheck{
		ID:          ws.ID + ":" + key,
		WorkspaceID: ws.ID,
		SurfaceID:   ws.ID,
		Key:         key,
		Owner:       owner,
		FilePath:    file,
		PolicyValue: strings.TrimSpace(policyValue),
		NativeValue: strings.TrimSpace(nativeValue),
		Status:      status,
		Remediation: remediationForStatus(status, remediation),
		FindingCode: finding,
	}
}

func remediationForStatus(status, remediation string) string {
	if status == "pass" {
		return ""
	}
	return remediation
}

func resolveProjectionPath(scenarioRoot, workspaceRoot, rel string) string {
	if rel == "" {
		return workspaceRoot
	}
	if filepath.IsAbs(rel) {
		return filepath.Clean(rel)
	}
	if scenarioRoot != "" {
		return filepath.Clean(filepath.Join(scenarioRoot, rel))
	}
	return filepath.Clean(filepath.Join(filepath.Dir(workspaceRoot), rel))
}

func fileOrDirExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func boolNative(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func thresholdNative(value float64, ok bool) string {
	if !ok {
		return ""
	}
	return projectionFormatCoverageFloor(value)
}

func projectionFormatCoverageFloor(floor float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", floor), "0"), ".")
}
