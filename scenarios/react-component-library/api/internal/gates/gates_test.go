package gates

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

func liveRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "..", "..", "..", "..", ".."))
}

func TestReleasedVersionImmutableRejectsSyntheticDrift(t *testing.T) {
	root := t.TempDir()
	dbDir := filepath.Join(root, "scenarios", "react-component-library", "data")
	sourcePath := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Fixture", "versions", "1.0.0", "Fixture.tsx")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	released := []byte("export const Fixture = () => null;")
	drifted := []byte("export const Fixture = () => <div>drifted</div>;")
	if err := os.WriteFile(sourcePath, drifted, 0o644); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(released)
	db, err := database.Open(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: "file:" + filepath.Join(dbDir, "react-component-library.db") + "?_pragma=foreign_keys(ON)"})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.ExecContext(context.Background(), `CREATE TABLE component_versions (status TEXT NOT NULL, source_path TEXT NOT NULL, content_sha256 TEXT NOT NULL)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(context.Background(), `INSERT INTO component_versions(status, source_path, content_sha256) VALUES ('released', ?, ?)`, "components/Fixture/versions/1.0.0/Fixture.tsx", hex.EncodeToString(hash[:]))
	if err != nil {
		t.Fatal(err)
	}
	result, err := ValidateReleasedVersionImmutable(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 || len(result.Findings) != 1 {
		t.Fatalf("result = %+v, want one synthetic drift finding", result)
	}
	if result.Findings[0].Code != "catalog.released_version_immutable" {
		t.Fatalf("finding = %+v", result.Findings[0])
	}
}

func TestLiveTokenGate(t *testing.T) {
	result, err := ValidateTokens(liveRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected < 170 {
		t.Fatalf("live token gate inspected %d active implementation files; expected the indexed active corpus", result.Inspected)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("live token gate reported findings in the active corpus: %+v", result.Findings)
	}
}

func TestLiveAPIGateInspectsClosureImplementations(t *testing.T) {
	result, err := ValidateAPI(liveRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected == 0 {
		t.Fatal("live api gate inspected zero closure implementations")
	}
}

func TestAPIGateRejectsUndeclaredImplementationVocabulary(t *testing.T) {
	root := t.TempDir()
	assetDir := filepath.Join(root, "scenarios", "react-component-library", "catalog", "assets", "controls")
	manifestDir := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Button", "versions", "1.0.0")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	asset := `{"asset":{"id":"controls.button","kind":"component"},"api":{"variants":{"tone":["danger"]},"modes":["controlled"],"parts":["icon"]}}`
	if err := os.WriteFile(filepath.Join(assetDir, "button.json"), []byte(asset), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Button", "component.json"), []byte(`{"catalogId":"controls.button","latest":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(manifestDir, "Button.tsx"), []byte(`export function Button() { return <button>save</button>; }`), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ValidateAPI(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 || len(result.Findings) != 3 {
		t.Fatalf("result = %+v, want three vocabulary findings", result)
	}
}

func TestFixtureGateRejectsMissingAdversarialShape(t *testing.T) {
	root := t.TempDir()
	assets := filepath.Join(root, "scenarios", "react-component-library", "catalog", "assets", "fixtures")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assets, "one.json"), []byte(`{"kind":"catalog-asset","asset":{"id":"fixtures.one","kind":"fixture"},"fixture":{"dataShapes":["typical"]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ValidateFixtures(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 || len(result.Findings) != 1 || result.Findings[0].Code != "catalog.fixture_adversarial" {
		t.Fatalf("result = %+v", result)
	}
}

func TestTokenGateRejectsLiteralDimensionInImplementation(t *testing.T) {
	root := t.TempDir()
	for _, kit := range []string{"one", "two", "three"} {
		path := filepath.Join(root, "templates", "design", kit, "adapters", "react-vite-tailwind")
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "tokens.css"), []byte(strings.Join([]string{
			"--space-3xs: 4px; --space-2xs: 8px; --space-xs: 12px; --space-sm: 16px; --space-md: 24px; --space-lg: 32px; --space-xl: 40px; --space-2xl: 48px;",
			"--text-display: x; --text-title: x; --text-heading: x; --text-body: x; --text-subheading: x; --text-body-sm: x; --text-label: x; --text-caption: x;",
			"--elev-flat: x; --elev-raised: x; --elev-overlay: x; --elev-modal: x; --layer-base: x; --layer-dropdown: x; --layer-sticky: x; --layer-overlay: x; --layer-modal: x; --layer-toast: x; --layer-tooltip: x; --border-hairline: x; --border-strong: x; --opacity-disabled: x; --opacity-muted: x; --opacity-scrim: x; --dur-instant: x;",
		}, "\n")), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	source := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Button", "versions", "1.0.0")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "Button.tsx"), []byte("export const Button = () => <button className=\"px-3\" />"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ValidateTokens(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 || len(result.Findings) != 1 || result.Findings[0].Code != "catalog.tokens_literal" {
		t.Fatalf("result = %+v", result)
	}
}

func TestLifecycleGateRejectsMissingCleanup(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Watcher", "versions", "1.0.0")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "Watcher.tsx"), []byte(`export const Watcher = () => { if (typeof window !== "undefined") window.addEventListener("resize", () => {}); return null; };`), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ValidateLifecycle(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 || len(result.Findings) != 1 || result.Findings[0].Code != "catalog.lifecycle_cleanup" {
		t.Fatalf("result = %+v", result)
	}
}

func TestLifecycleGateIgnoresStoriesAndEffectScopedBrowserAccess(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Watcher", "versions", "1.0.0")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	implementation := `import { useEffect } from "react";
export const Watcher = () => {
  useEffect(() => {
    const timer = window.setTimeout(() => {}, 10);
    return () => window.clearTimeout(timer);
  }, []);
  return null;
};`
	if err := os.WriteFile(filepath.Join(source, "Watcher.tsx"), []byte(implementation), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "story.tsx"), []byte(`export const Story = () => window.setTimeout(() => {}, 10);`), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ValidateLifecycle(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 || len(result.Findings) != 0 {
		t.Fatalf("result = %+v, want only runtime source inspected with no findings", result)
	}
}

func TestLifecycleGateRejectsRenderTimeBrowserAccess(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Watcher", "versions", "1.0.0")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	implementation := `export const Watcher = () => <output>{window.innerWidth}</output>;`
	if err := os.WriteFile(filepath.Join(source, "Watcher.tsx"), []byte(implementation), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ValidateLifecycle(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 || len(result.Findings) != 1 || result.Findings[0].Code != "catalog.lifecycle_ssr" {
		t.Fatalf("result = %+v, want render-time SSR finding", result)
	}
}

func TestLifecycleGateAcceptsDocumentGuard(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "scenarios", "react-component-library", "library", "hooks", "useLocale", "versions", "1.0.0")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	implementation := `export function useLocale() {
  return typeof document === "undefined" ? "en" : document.documentElement.lang || "en";
}`
	if err := os.WriteFile(filepath.Join(source, "useLocale.ts"), []byte(implementation), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ValidateLifecycle(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 || len(result.Findings) != 0 {
		t.Fatalf("result = %+v, want guarded document access to pass", result)
	}
}

// TestEveryFindingCarriesRemediation is the contract that keeps gate output
// actionable. A finding states what is wrong; without a remediation the reader
// is left to infer the fix, and inference is how "use a declared semantic
// token" survived for as long as it did against a ramp that publishes no such
// token. Any new gate that emits a bare message fails here.
func TestEveryFindingCarriesRemediation(t *testing.T) {
	root := t.TempDir()
	runners := map[string]func(string) (Result, error){
		"api":         ValidateAPI,
		"tokens":      ValidateTokens,
		"conformance": ValidateConformance,
		"lifecycle":   ValidateLifecycle,
		"fixtures":    ValidateFixtures,
		"examples":    ValidateExamples,
		"rtl":         ValidateRTL,
		"stress":      ValidateStress,
		"reduced":     ValidateReducedMotion,
		"integrate":   ValidateIntegration,
		"vocabulary":  ValidateTokenVocabulary,
	}
	for name, run := range runners {
		t.Run(name, func(t *testing.T) {
			result, err := run(root)
			if err != nil {
				t.Skipf("runner needs inputs this fixture does not supply: %v", err)
			}
			if len(result.Findings) == 0 && len(result.RunnerError) == 0 {
				t.Fatalf("%s produced no findings; the zero-input contract should have emitted one", name)
			}
			for _, finding := range append(result.Findings, result.RunnerError...) {
				if strings.TrimSpace(finding.Remediation) == "" {
					t.Errorf("finding %q has no remediation; state what to do about it, not only what is wrong", finding.Code)
				}
				if strings.TrimSpace(finding.Message) == "" {
					t.Errorf("finding %q has no message", finding.Code)
				}
			}
		})
	}
}

// TestLiteralDimensionFindingsNameTheRightFix guards the classification that
// makes the tokens gate trustworthy. Spacing has a ramp to point at; sizing
// does not, and pointing an author at a nonexistent icon-size token is worse
// than silence because it sends them looking for something they cannot find.
func TestLiteralDimensionFindingsNameTheRightFix(t *testing.T) {
	source := []byte("const a = <Icon className=\"h-4 w-4\" />;\nconst b = <div className=\"gap-3\" />;\nconst c = <hr className=\"w-[13px]\" />;\n")
	findings := literalDimensionFindings("/repo", "/repo/library/components/X/versions/1.0.0/X.tsx", source)
	byLine := map[int]Finding{}
	for _, finding := range findings {
		byLine[finding.Line] = finding
	}
	sizing, okSizing := byLine[1]
	if !okSizing || !strings.Contains(sizing.Remediation, "Icon primitive's size scale") {
		t.Errorf("sizing finding should point at the Icon size scale, got %q", sizing.Remediation)
	}
	if strings.Contains(sizing.Remediation, "semantic token") {
		t.Errorf("sizing finding must not promise a token the ramp does not publish: %q", sizing.Remediation)
	}
	spacing, okSpacing := byLine[2]
	if !okSpacing || !strings.Contains(spacing.Remediation, "gap-space-") {
		t.Errorf("spacing finding should name the ramp utilities, got %q", spacing.Remediation)
	}
	arbitrary, okArbitrary := byLine[3]
	if !okArbitrary || !strings.Contains(arbitrary.Remediation, "ramp is missing a rung") {
		t.Errorf("arbitrary-px finding should offer adding a ramp step, got %q", arbitrary.Remediation)
	}
	for _, finding := range findings {
		if finding.File != "library/components/X/versions/1.0.0/X.tsx" {
			t.Errorf("finding file should be repo-relative, got %q", finding.File)
		}
		if finding.Line == 0 {
			t.Errorf("finding %q resolved no line", finding.Message)
		}
	}
}

func TestEveryGateRejectsZeroInspectedInputs(t *testing.T) {
	root := t.TempDir()
	tests := []struct {
		name string
		run  func(string) (Result, error)
	}{
		{name: "api", run: ValidateAPI},
		{name: "tokens", run: ValidateTokens},
		{name: "conformance", run: ValidateConformance},
		{name: "lifecycle", run: ValidateLifecycle},
		{name: "fixtures", run: ValidateFixtures},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.run(root)
			if err != nil {
				t.Fatal(err)
			}
			if result.Inspected != 0 || len(result.Findings)+len(result.RunnerError) == 0 {
				t.Fatalf("result = %+v, want zero-input finding", result)
			}
		})
	}
}

func TestConformanceMeasuresStaySeparateAndNonVacuous(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "scenarios", "react-component-library", "ui", "src", "Fixture.tsx")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`
const Fixture = () => <div className="text-xs text-xs text-xs text-xs text-xs text-lg gap-3 h-4 w-4 rounded-[13px] shadow-[0_0_4px_#000]" />;
export default Fixture;
`), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ValidateConformance(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 {
		t.Fatalf("inspected = %d, want one workbench source", result.Inspected)
	}
	want := map[string]bool{
		"conformance.type-scale":       false,
		"conformance.ramp-adherence":   false,
		"conformance.icon-scale":       false,
		"conformance.elevation-radius": false,
	}
	for _, finding := range result.Findings {
		if finding.Category != "conformance" {
			t.Errorf("finding category = %q, want conformance", finding.Category)
		}
		if _, ok := want[finding.Code]; ok {
			want[finding.Code] = true
		}
	}
	for code, found := range want {
		if !found {
			t.Errorf("missing conformance measure %q in %+v", code, result.Findings)
		}
	}
}

func TestFixtureGateRejectsDataSourceWithoutTypeArgument(t *testing.T) {
	root := t.TempDir()
	assets := filepath.Join(root, "scenarios", "react-component-library", "catalog", "assets", "fixtures")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := `{"kind":"catalog-asset","asset":{"id":"fixtures.one","kind":"fixture"},"fixture":{"dataShapes":["typical","failure"],"satisfies":{"capability":"data-source","typeArguments":[]}}}`
	if err := os.WriteFile(filepath.Join(assets, "one.json"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ValidateFixtures(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 1 || result.Findings[0].Code != "catalog.fixture_data_source" {
		t.Fatalf("result = %+v", result)
	}
}
