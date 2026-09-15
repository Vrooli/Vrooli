package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"unit-health/internal/adapterregistry"
	"unit-health/internal/adapters"
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
	analyzer, ok := adapterregistry.Default().Resolve(ws.AdapterID, ws.Language, ws.Framework)
	if !ok {
		return nil
	}
	expect := resolveProjectionPolicy(ws)
	native := analyzer.AnalyzeProjectionChecks(adapters.ProjectionInput{RootPath: ws.RootPath, Policy: expect})
	checks := make([]ProjectionCheck, 0, len(native)+4)
	for _, check := range native {
		checks = append(checks, projectionCheck(ws, check.Key, check.Owner, check.File, check.PolicyValue, check.NativeValue, check.Pass, check.Remediation))
	}
	class := policyClassForWorkspace(scenarioRoot, ws)
	for _, root := range class.TestUtils.RequiredRoots {
		path := resolveProjectionPath(scenarioRoot, ws.RootPath, root)
		checks = append(checks, projectionCheck(ws, "testutil.root", "unit.policy_profile test_utils.required_roots", path, root, boolNative(fileOrDirExists(path)), fileOrDirExists(path),
			"Create the shared UI test-utils root declared by the unit policy profile."))
	}
	if helper := strings.TrimSpace(class.TestUtils.CanonicalRenderHelper); helper != "" {
		path := resolveProjectionPath(scenarioRoot, ws.RootPath, helper)
		if fileExists(path) {
			checks = append(checks, projectionCheck(ws, "testutil.canonical_render_helper", "unit.policy_profile test_utils.canonical_render_helper", path, helper, "true", true,
				"Add src/test-utils/renderWithProviders.tsx and re-export it from src/test-utils/index.ts."))
		} else if importsSharedRenderHelperFromWorkspace(ws.RootPath) {
			checks = append(checks, projectionCheck(ws, "testutil.canonical_render_helper", "unit.policy_profile test_utils.canonical_render_helper", filepath.Join(ws.RootPath, "package.json"), helper, "@vrooli/api-base/testing", true,
				"Import renderWithProviders from @vrooli/api-base/testing or provide a documented local adapter."))
		} else {
			checks = append(checks, projectionCheck(ws, "testutil.canonical_render_helper", "unit.policy_profile test_utils.canonical_render_helper", path, helper, "false", false,
				"Import renderWithProviders from @vrooli/api-base/testing or provide a documented local adapter."))
		}
	}
	return checks
}

func importsSharedRenderHelperFromWorkspace(root string) bool {
	found := false
	walkSourceFiles(root, func(path string) {
		if found || !isTSSourceFile(path) {
			return
		}
		found = strings.Contains(readFileString(path), "@vrooli/api-base/testing") && strings.Contains(readFileString(path), "renderWithProviders")
	})
	return found
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
			Framework: adapterregistry.DefaultFramework("typescript"),
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
