package validation

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"unit-health/internal/discovery"
)

func TestResolveUnitPolicyProfileValidReactViteShape(t *testing.T) {
	root := t.TempDir()
	writeUnitPolicyProfile(t, root, reactViteUnitPolicyProfile())
	inv := discovery.Inventory{
		Scenario: "demo",
		RootPath: root,
		Surfaces: []discovery.Surface{
			{ID: "api", Kind: "api", Language: "go", RootPath: filepath.Join(root, "api")},
			{ID: "cli", Kind: "cli", Language: "go", RootPath: filepath.Join(root, "cli")},
			{ID: "ui", Kind: "ui", Language: "typescript", Framework: "vite", RootPath: filepath.Join(root, "ui")},
		},
	}

	findings := resolveUnitPolicyFindings("demo", inv, fixedNowStr)
	if len(findings) != 0 {
		t.Fatalf("expected valid profile to be clean, got %v", codes(findings))
	}
}

func TestResolveUnitPolicyProfileAcceptsCodeFactsReactViteFrameworkAlias(t *testing.T) {
	root := t.TempDir()
	writeUnitPolicyProfile(t, root, reactViteUnitPolicyProfile())
	inv := discovery.Inventory{
		Scenario: "demo",
		RootPath: root,
		Surfaces: []discovery.Surface{
			{ID: "api", Kind: "api", Language: "go", RootPath: filepath.Join(root, "api")},
			{ID: "cli", Kind: "cli", Language: "go", RootPath: filepath.Join(root, "cli")},
			{ID: "ui", Kind: "ui", Language: "typescript", Framework: "react-vite", RootPath: filepath.Join(root, "ui")},
		},
	}

	findings := resolveUnitPolicyFindings("demo", inv, fixedNowStr)
	if _, ok := findingByCode(findings, codeUnitRequiredRoleMissing); ok {
		t.Fatalf("react-vite should satisfy the Vite UI policy role, got %+v", findings)
	}
}

func TestResolveUnitPolicyProfileMissingRequiredRole(t *testing.T) {
	root := t.TempDir()
	writeUnitPolicyProfile(t, root, reactViteUnitPolicyProfile())
	inv := discovery.Inventory{
		Scenario: "demo",
		RootPath: root,
		Surfaces: []discovery.Surface{
			{ID: "api", Kind: "api", Language: "go", RootPath: filepath.Join(root, "api")},
			{ID: "ui", Kind: "ui", Language: "typescript", Framework: "vite", RootPath: filepath.Join(root, "ui")},
		},
	}

	findings := resolveUnitPolicyFindings("demo", inv, fixedNowStr)
	if _, ok := findingByCode(findings, codeUnitRequiredRoleMissing); !ok {
		t.Fatalf("expected UNIT_REQUIRED_ROLE_MISSING, got %v", codes(findings))
	}
}

func TestResolveUnitPolicyProfileWeakenedCoverage(t *testing.T) {
	root := t.TempDir()
	profile := reactViteUnitPolicyProfile()
	ui := profile.PolicyClasses["react_vite_ui"]
	ui.Coverage.MinimumPercent = 70
	profile.PolicyClasses["react_vite_ui"] = ui
	writeUnitPolicyProfile(t, root, profile)
	inv := discovery.Inventory{Scenario: "demo", RootPath: root}

	findings := resolveUnitPolicyFindings("demo", inv, fixedNowStr)
	if _, ok := findingByCode(findings, codeUnitPolicyWeakened); !ok {
		t.Fatalf("expected UNIT_POLICY_WEAKENED, got %v", codes(findings))
	}
}

func TestResolveUnitPolicyProfileInvalidWaiver(t *testing.T) {
	root := t.TempDir()
	profile := reactViteUnitPolicyProfile()
	profile.Customization.Waivers = []unitPolicyWaiver{{Finding: codeUnitPolicyWeakened, Reason: "temporary"}}
	writeUnitPolicyProfile(t, root, profile)
	inv := discovery.Inventory{Scenario: "demo", RootPath: root}

	findings := resolveUnitPolicyFindings("demo", inv, fixedNowStr)
	if _, ok := findingByCode(findings, codeUnitWaiverInvalid); !ok {
		t.Fatalf("expected UNIT_POLICY_WAIVER_INVALID, got %v", codes(findings))
	}
}

func TestResolveUnitPolicyProfileUngovernedUnsupportedSurface(t *testing.T) {
	root := t.TempDir()
	writeUnitPolicyProfile(t, root, reactViteUnitPolicyProfile())
	inv := discovery.Inventory{
		Scenario: "demo",
		RootPath: root,
		Surfaces: []discovery.Surface{
			{ID: "api", Kind: "api", Language: "go", RootPath: filepath.Join(root, "api")},
			{ID: "cli", Kind: "cli", Language: "go", RootPath: filepath.Join(root, "cli")},
			{ID: "ui", Kind: "ui", Language: "typescript", Framework: "vite", RootPath: filepath.Join(root, "ui")},
			{ID: "native", Kind: "worker", Language: "rust", RootPath: filepath.Join(root, "native")},
		},
	}

	findings := resolveUnitPolicyFindings("demo", inv, fixedNowStr)
	if _, ok := findingByCode(findings, codeUnitSurfaceUngoverned); !ok {
		t.Fatalf("expected UNIT_SURFACE_UNGOVERNED, got %v", codes(findings))
	}
}

func TestResolveUnitPolicyProfileAddedPythonSurfaceUsesUnitHealthDefault(t *testing.T) {
	root := t.TempDir()
	writeUnitPolicyProfile(t, root, reactViteUnitPolicyProfile())
	inv := discovery.Inventory{
		Scenario: "demo",
		RootPath: root,
		Surfaces: []discovery.Surface{
			{ID: "api", Kind: "api", Language: "go", RootPath: filepath.Join(root, "api")},
			{ID: "cli", Kind: "cli", Language: "go", RootPath: filepath.Join(root, "cli")},
			{ID: "ui", Kind: "ui", Language: "typescript", Framework: "vite", RootPath: filepath.Join(root, "ui")},
			{ID: "worker", Kind: "worker", Language: "python", RootPath: filepath.Join(root, "worker")},
		},
	}

	findings := resolveUnitPolicyFindings("demo", inv, fixedNowStr)
	if _, ok := findingByCode(findings, codeUnitSurfaceUngoverned); ok {
		t.Fatalf("python surfaces should use the Unit Health default instead of becoming ungoverned, got %+v", findings)
	}
}

func TestResolveUnitPolicyProfileJestUIDoesNotSatisfyReactViteRole(t *testing.T) {
	root := t.TempDir()
	writeUnitPolicyProfile(t, root, reactViteUnitPolicyProfile())
	inv := discovery.Inventory{
		Scenario: "demo",
		RootPath: root,
		Surfaces: []discovery.Surface{
			{ID: "api", Kind: "api", Language: "go", RootPath: filepath.Join(root, "api")},
			{ID: "cli", Kind: "cli", Language: "go", RootPath: filepath.Join(root, "cli")},
			{ID: "ui", Kind: "ui", Language: "typescript", Framework: "jest", RootPath: filepath.Join(root, "ui")},
		},
	}

	findings := resolveUnitPolicyFindings("demo", inv, fixedNowStr)
	f, ok := findingByCode(findings, codeUnitRequiredRoleMissing)
	if !ok {
		t.Fatalf("expected UNIT_REQUIRED_ROLE_MISSING for the React/Vite UI role, got %v", codes(findings))
	}
	if !strings.Contains(f.Evidence, "role=ui") {
		t.Fatalf("expected UI role evidence, got %+v", f)
	}
}

func TestResolveUnitPolicyProfileMissingPolicyProfileIsInvalid(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".vrooli", "testing.json"), `{
  "version": "1.0.0",
  "unit": {}
}`)
	inv := discovery.Inventory{
		Scenario: "demo",
		RootPath: root,
		Surfaces: []discovery.Surface{
			{ID: "api", Kind: "api", Language: "go", RootPath: filepath.Join(root, "api")},
			{ID: "ui", Kind: "ui", Language: "typescript", Framework: "vite", RootPath: filepath.Join(root, "ui")},
		},
	}

	findings := resolveUnitPolicyFindings("demo", inv, fixedNowStr)
	f, ok := findingByCode(findings, codeUnitPolicyInvalid)
	if !ok {
		t.Fatalf("expected UNIT_POLICY_PROFILE_INVALID, got %v", codes(findings))
	}
	if !strings.Contains(f.Observed, "missing unit.policy_profile") {
		t.Fatalf("expected missing profile evidence, got %+v", f)
	}
}

func TestResolveUnitPolicyProfileReactViteTemplateClean(t *testing.T) {
	repoRoot := findRepoRoot(t)
	root := filepath.Join(repoRoot, "templates", "scenarios", "react-vite")
	inv := discovery.Inventory{
		Scenario: "react-vite",
		RootPath: root,
		Surfaces: []discovery.Surface{
			{ID: "api", Kind: "api", Language: "go", RootPath: filepath.Join(root, "api")},
			{ID: "cli", Kind: "cli", Language: "go", RootPath: filepath.Join(root, "cli")},
			{ID: "ui", Kind: "ui", Language: "typescript", Framework: "vite", RootPath: filepath.Join(root, "ui")},
		},
	}

	findings := resolveUnitPolicyFindings("react-vite", inv, fixedNowStr)
	if len(findings) != 0 {
		t.Fatalf("react-vite template policy profile should be clean, got %+v", findings)
	}
}

func writeUnitPolicyProfile(t *testing.T, root string, profile unitPolicyProfile) {
	t.Helper()
	doc := unitPolicyDocument{}
	doc.Unit.PolicyProfile = profile
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatalf("marshal unit policy profile: %v", err)
	}
	writeFile(t, filepath.Join(root, ".vrooli", "testing.json"), string(raw))
}

func reactViteUnitPolicyProfile() unitPolicyProfile {
	return unitPolicyProfile{
		Version: "1.0.0",
		Template: unitPolicyTemplate{
			ID:            "react-vite",
			ScenarioClass: "react-vite",
		},
		RequiredRoles: []unitPolicyRequiredRole{
			{
				Role:        "api",
				PolicyClass: "go_service",
				Match:       unitPolicyMatch{SurfaceID: "api", Kind: "api", Path: "api", Language: "go"},
			},
			{
				Role:        "cli",
				PolicyClass: "go_cli",
				Match:       unitPolicyMatch{SurfaceID: "cli", Kind: "cli", Path: "cli", Language: "go"},
			},
			{
				Role:        "ui",
				PolicyClass: "react_vite_ui",
				Match:       unitPolicyMatch{SurfaceID: "ui", Kind: "ui", Path: "ui", Language: "typescript", Framework: "vite"},
			},
		},
		PolicyClasses: map[string]unitPolicyClass{
			"go_service": {
				Language:       "go",
				Framework:      "go test",
				PackageManager: "go",
				Coverage:       unitCoveragePolicy{MinimumPercent: 75, Mode: "total"},
				TestUtils:      unitTestUtilsPolicy{RequiredRoots: []string{"api/internal/testutil"}, ProductionImportBan: true},
				Projection:     unitProjectionPolicy{RequiredFiles: []string{"api/internal/testutil/no_prod_import_test.go"}},
			},
			"go_cli": {
				Language:       "go",
				Framework:      "go test",
				PackageManager: "go",
				Coverage:       unitCoveragePolicy{MinimumPercent: 75, Mode: "total"},
				TestUtils:      unitTestUtilsPolicy{RequiredRoots: []string{"cli/internal/testutil"}, ProductionImportBan: true},
				Projection:     unitProjectionPolicy{RequiredFiles: []string{"cli/app_test.go", "cli/internal/testutil/no_prod_import_test.go"}},
			},
			"react_vite_ui": {
				Language:       "typescript",
				Framework:      "vitest",
				PackageManager: "pnpm",
				Coverage: unitCoveragePolicy{
					MinimumPercent: 85,
					Mode:           "both",
					Provider:       "v8",
					Reporters:      []string{"text", "json-summary", "json"},
				},
				TestUtils: unitTestUtilsPolicy{
					RequiredRoots:         []string{"ui/src/test-utils"},
					ProductionImportBan:   true,
					CanonicalRenderHelper: "ui/src/test-utils/renderWithProviders.tsx",
				},
				Projection: unitProjectionPolicy{
					RequiredFiles:   []string{"ui/vite.config.ts", "ui/src/test-setup.ts", "ui/src/test-utils/renderWithProviders.tsx"},
					RequiredScripts: []string{"test", "test:coverage"},
					Vitest: unitVitestProjection{
						Environment:      "jsdom",
						SetupFiles:       []string{"./src/test-setup.ts"},
						CoverageProvider: "v8",
					},
				},
			},
		},
		Customization: unitPolicyCustomization{Mode: "monotonic", Waivers: []unitPolicyWaiver{}},
	}
}
