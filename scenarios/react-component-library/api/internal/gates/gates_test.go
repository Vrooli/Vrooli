package gates

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
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

func TestValidateNoUtilityClassesDistinguishesClassBearingSource(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		wantDefect bool
	}{
		{name: "token-bound", source: `export const X = ({ className }) => <div className={className} />`},
		{name: "responsive", source: `export const X = () => <div className="md:inset-x-8" />`, wantDefect: true},
		{name: "foreign-palette", source: `export const X = () => <div className="bg-wc-backdrop" />`, wantDefect: true},
		{name: "custom-utility", source: `export const X = () => <div className="touch-target" />`, wantDefect: true},
		{name: "stylesheet-literal", source: "const styles = `[data-x] .bg-wc-backdrop { display: grid; }`\nexport const X = ({ className }) => <div className={className} />"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "X", "versions", "1.0.0", "X.tsx")
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(test.source), 0o600); err != nil {
				t.Fatal(err)
			}
			result, err := ValidateNoUtilityClasses(Scope{Root: root})
			if err != nil {
				t.Fatal(err)
			}
			if got := len(result.Findings) > 0; got != test.wantDefect {
				t.Fatalf("findings = %+v, want defect %v", result.Findings, test.wantDefect)
			}
		})
	}
}

func TestLiveUtilityClassAllowlistIsExactAndShrinkOnly(t *testing.T) {
	root := liveRoot(t)
	result, err := ValidateNoUtilityClasses(Scope{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("unallowlisted or stale utility debt: %+v", result.Findings)
	}
	data, err := os.ReadFile(filepath.Join(root, "scenarios", "react-component-library", "library", "utility-class-allowlist.json"))
	if err != nil {
		t.Fatal(err)
	}
	var allowlist utilityClassAllowance
	if err := json.Unmarshal(data, &allowlist); err != nil {
		t.Fatal(err)
	}
	maxData, err := os.ReadFile(filepath.Join(root, "scenarios", "react-component-library", "library", "utility-class-allowlist.max"))
	if err != nil {
		t.Fatal(err)
	}
	maximum, err := strconv.Atoi(strings.TrimSpace(string(maxData)))
	if err != nil {
		t.Fatal(err)
	}
	if len(allowlist.Entries) > maximum {
		t.Fatalf("utility allowlist grew to %d entries above shrink-only maximum %d", len(allowlist.Entries), maximum)
	}
	if len(result.InformationalFindings) != len(allowlist.Entries) {
		t.Fatalf("observed allowlisted debt = %d, entries = %d", len(result.InformationalFindings), len(allowlist.Entries))
	}
}

func TestValidateConsumerPinsNamesEveryDefectAndConsumer(t *testing.T) {
	root := t.TempDir()
	assetRoot := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Fixture")
	versionRoot := filepath.Join(assetRoot, "versions", "1.0.0")
	if err := os.MkdirAll(versionRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"libraryId":"react-component-library:Fixture","catalogId":"calibration.fixture","latest":"2.0.0","deprecatedVersions":["1.0.0"]}`
	if err := os.WriteFile(filepath.Join(assetRoot, "component.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(versionRoot, "Fixture.tsx"), []byte(`export const Fixture = () => <div className="md:inset-x-8" />`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"alpha", "beta"} {
		path := filepath.Join(root, "scenarios", scenario, "ui", "src", "App.tsx")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		source := `import { Fixture } from "@vrooli/react-component-library/Fixture/1.0.0"; export const App = Fixture;`
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	missingPath := filepath.Join(root, "scenarios", "alpha", "ui", "src", "Missing.tsx")
	if err := os.WriteFile(missingPath, []byte(`import "@vrooli/react-component-library/Missing/9.9.9";`), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := ValidateConsumerPins(Scope{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	codes := map[string]bool{}
	for _, finding := range result.Findings {
		codes[finding.Code] = true
		if strings.Contains(finding.Message, "Fixture@1.0.0") && (!strings.Contains(finding.Message, "alpha") || !strings.Contains(finding.Message, "beta")) {
			t.Fatalf("grouped finding omitted a consumer: %+v", finding)
		}
	}
	for _, code := range []string{"catalog.consumer-pin.deprecated", "catalog.consumer-pin.stale-major", "catalog.consumer-pin.utility-class", "catalog.consumer-pin.missing"} {
		if !codes[code] {
			t.Fatalf("missing %s in %+v", code, result.Findings)
		}
	}
}

func TestValidateSelectorCoverageLiveCorpusIsMeasuredAcrossExportedVersions(t *testing.T) {
	result, err := ValidateSelectorCoverage(Scope{Root: liveRoot(t)})
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected == 0 || result.Inspected != len(result.InspectedAssets) || result.InspectedVersions <= result.Inspected || len(result.Skipped) != 0 || len(result.RunnerError) != 0 {
		t.Fatalf("selector coverage = %+v, want all active assets and exported versions measured", result)
	}
}

func TestScopeSelectsOnlyRequestedAssets(t *testing.T) {
	scope := Scope{Root: "/repo", Assets: []string{"controls.button", "forms.select"}}
	selected := map[string]bool{}
	for _, assetID := range scope.Assets {
		selected[assetID] = true
	}
	for _, test := range []struct {
		assetID string
		want    bool
	}{
		{assetID: "controls.button", want: true},
		{assetID: "forms.select", want: true},
		{assetID: "controls.checkbox", want: false},
	} {
		got := scope.IsFullCorpus() || selected[test.assetID]
		if got != test.want {
			t.Fatalf("asset %q selected = %v, want %v", test.assetID, got, test.want)
		}
	}
}

func TestVersionLivenessIgnoresRetiredMaterializedVersions(t *testing.T) {
	root := t.TempDir()
	assetRoot := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Fixture")
	for _, version := range []string{"1.0.0", "2.0.0"} {
		versionRoot := filepath.Join(assetRoot, "versions", version)
		require.NoError(t, os.MkdirAll(versionRoot, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(versionRoot, "Fixture.tsx"), []byte(`import "../../../Other/versions/1.0.0/Other"; export const Fixture = () => null;`), 0o600))
	}
	require.NoError(t, os.WriteFile(filepath.Join(assetRoot, "component.json"), []byte(`{"catalogId":"calibration.fixture","libraryId":"react-component-library:Fixture","latest":"2.0.0","deprecatedVersions":["1.0.0"]}`), 0o600))

	result, err := ValidateVersionLiveness(Scope{Root: root})
	require.NoError(t, err)
	require.Len(t, result.Findings, 1)
	require.Contains(t, result.Findings[0].File, "versions/2.0.0/Fixture.tsx")
}

func TestStoryDistinctnessRejectsOneSpecimenForManyFrames(t *testing.T) {
	root := t.TempDir()
	storyDir := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Fixture", "versions", "1.0.0")
	if err := os.MkdirAll(storyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	contract := `{"schemaVersion":5,"kind":"component","args":{"fields":[]},"environment":{"fixtures":[]},"stories":[{"id":"one","name":"One","role":"anatomy","composition":{"specimen":{"module":"./story.tsx","export":"Shared"}},"expect":[{"kind":"visible","selector":"body"}]},{"id":"two","name":"Two","role":"boundary","composition":{"specimen":{"module":"./story.tsx","export":"Shared"}},"expect":[{"kind":"visible","selector":"body"}]},{"id":"three","name":"Three","role":"boundary","composition":{"specimen":{"module":"./story.tsx","export":"Shared"}},"expect":[{"kind":"visible","selector":"body"}]}]}`
	if err := os.WriteFile(filepath.Join(storyDir, "story.json"), []byte(contract), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ValidateStoryDistinctness(Scope{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	foundRepeatedSpecimen := false
	for _, finding := range result.Findings {
		if strings.Contains(finding.Message, "reuse specimen") {
			foundRepeatedSpecimen = true
		}
	}
	if !foundRepeatedSpecimen {
		t.Fatalf("result = %+v, want repeated-specimen finding", result)
	}
}

func TestAnalyzeRestyleSourceEnforcesClassRefAndInlineStyleRules(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{name: "inline token style", source: `export function Sample({ className }: { className?: string }) { return <div className={className} style={{ color: "red" }} />; }`, want: "inline style"},
		{name: "nested layout style is not a root override", source: `export const Sample = forwardRef<HTMLDivElement, { className?: string }>(function Sample({ className }, ref) { return <div ref={ref} className={className}><span style={{ display: "grid" }} /></div>; });`, want: ""},
		{name: "missing ref", source: `export function Sample({ className }: { className?: string }) { return <div className={className} />; }`, want: "ref"},
		{name: "overloaded style", source: `export function Sample({ className, style }: { className?: string; style?: object }) { return <div className={className} />; }`, want: "style prop"},
		{name: "clean forward ref", source: `export const Sample = forwardRef<HTMLDivElement, { className?: string }>(function Sample({ className }, ref) { return <div ref={ref} className={cn("sample", className)} />; });`, want: ""},
		{name: "hoisted inline token style", source: `const controlStyle = { color: "red" }; export const Sample = forwardRef<HTMLDivElement, { className?: string }>(function Sample({ className }, ref) { return <div ref={ref} className={className} style={controlStyle} />; });`, want: "inline style"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			finding := analyzeRestyleSource(test.source)
			if test.want == "" {
				if finding.Message != "" {
					t.Fatalf("unexpected finding: %+v", finding)
				}
				return
			}
			if !strings.Contains(finding.Message, test.want) {
				t.Fatalf("finding=%+v, want message containing %q", finding, test.want)
			}
		})
	}
}

func TestValidateManifestIdentityRejectsOmittedCatalogID(t *testing.T) {
	root := t.TempDir()
	manifestDir := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Fixture")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(manifestDir, "component.json"), []byte(`{"libraryId":"react-component-library:Fixture"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ValidateManifestIdentity(Scope{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 1 || result.Findings[0].Code != "catalog.manifest_identity" {
		t.Fatalf("manifest identity result = %+v, want omitted catalogId finding", result)
	}
}

func TestValidateManifestMetadataLiveCorpus(t *testing.T) {
	root, err := filepath.Abs("../../../../../")
	if err != nil {
		t.Fatal(err)
	}
	result, err := ValidateManifestMetadata(Scope{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("manifest metadata findings = %+v", result.Findings)
	}
}

func TestValidateOverlaySurfaceCompositionAcceptsSharedCoreAndReasonedOptOut(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "scenarios", "react-component-library", "library", "components")
	fixtures := []struct {
		name     string
		manifest string
		source   string
	}{
		{name: "Composed", manifest: `{"libraryId":"react-component-library:Composed","catalogId":"overlays.composed","category":"overlays","latest":"1.0.0"}`, source: `import { useOverlaySurface } from "x"; export function Composed(){ useOverlaySurface({open:true}); return <section role="dialog" /> }`},
		{name: "OptOut", manifest: `{"libraryId":"react-component-library:OptOut","catalogId":"overlays.opt-out","category":"overlays","latest":"1.0.0","overlaySurfaceOptOutReason":"Static semantics-only example; it never opens or dismisses."}`, source: `export function OptOut(){ return <section role="dialog" /> }`},
	}
	for _, fixture := range fixtures {
		dir := filepath.Join(base, fixture.name, "versions", "1.0.0")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(base, fixture.name, "component.json"), []byte(fixture.manifest), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, fixture.name+".tsx"), []byte(fixture.source), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	result, err := ValidateOverlaySurfaceComposition(Scope{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("overlay composition findings = %+v", result.Findings)
	}
}

// The behaviour under test is that the AST pass sees a role assembled at
// runtime, which a text match over the source cannot. It is deliberately not
// pinned to a corpus path: the cold-version tier legitimately moves released
// version folders into the durable mirror, and a test that reads one directly
// fails for a storage decision rather than a behaviour change.
func TestStructuredSourceFactsDetectComputedOverlayRole(t *testing.T) {
	root := liveRoot(t)
	source := []byte(`export const Shell = ({ modal }: { modal: boolean }) => (
  <aside role={modal ? "dialog" : "complementary"} data-rcl-shell="sidebar" />
);
`)
	path := filepath.Join(t.TempDir(), "Shell.tsx")
	if err := os.WriteFile(path, source, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := sourceHasOverlayRole(root, path, source)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Fatal("structured AST facts did not detect the computed dialog role")
	}
}

// The same pass must not invent a role that is nowhere in the source.
func TestStructuredSourceFactsRejectSourceWithNoOverlayRole(t *testing.T) {
	root := liveRoot(t)
	source := []byte(`export const Plain = () => <div data-rcl-shell="plain" />;
`)
	path := filepath.Join(t.TempDir(), "Plain.tsx")
	if err := os.WriteFile(path, source, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := sourceHasOverlayRole(root, path, source)
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Fatal("structured AST facts reported an overlay role for source that declares none")
	}
}

func TestLibraryManifestIdentitiesResolvesVersionedSourceManifest(t *testing.T) {
	root := t.TempDir()
	assetDir := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Fixture")
	sourcePath := filepath.Join(assetDir, "versions", "1.0.0", "Fixture.tsx")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "component.json"), []byte(`{"libraryId":"react-component-library:Fixture","catalogId":"react-component-library:Fixture"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	libraryID, catalogID := libraryManifestIdentities(sourcePath)
	if libraryID != "react-component-library:Fixture" || catalogID != "react-component-library:Fixture" {
		t.Fatalf("identities = %q, %q; want the owning asset identities", libraryID, catalogID)
	}
}

func TestValidateRestyleContractLiveCorpusIsMeasuredAcrossExportedVersions(t *testing.T) {
	root, err := filepath.Abs("../../../../../")
	if err != nil {
		t.Fatal(err)
	}
	result, err := ValidateRestyleContract(Scope{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected == 0 || result.Inspected != len(result.InspectedAssets) || result.InspectedVersions <= result.Inspected || len(result.Skipped) != 0 || len(result.RunnerError) != 0 {
		t.Fatalf("restyle contract result = %+v, want all active assets and exported versions measured", result)
	}
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
	db, err := openGateDB(context.Background(), filepath.Join(dbDir, "react-component-library.db"))
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
	result, err := ValidateReleasedVersionImmutable(Scope{Context: context.Background(), Root: root, DB: db.Primary()})
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

func TestReleasedVersionImmutableRejectsEmptyIndexWithoutEvidence(t *testing.T) {
	root := t.TempDir()
	result, err := ValidateReleasedVersionImmutable(Scope{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 0 || len(result.RunnerError) == 0 {
		t.Fatalf("result = %+v, want explicit zero-evidence runner error", result)
	}
}

func TestEvidenceFreshnessSkipsSupplementalImplementations(t *testing.T) {
	root := t.TempDir()
	storyPath := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "AuthClient", "versions", "1.0.0", "story.json")
	manifestPath := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "AuthClient", "component.json")
	if err := os.MkdirAll(filepath.Dir(storyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(storyPath, []byte(`{"kind":"component","stories":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, []byte(`{"libraryId":"react-component-library:AuthClient","latest":"1.0.0","supplemental":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	normalStory := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Button", "versions", "1.0.0", "story.json")
	normalManifest := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Button", "component.json")
	if err := os.MkdirAll(filepath.Dir(normalStory), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(normalStory, []byte(`{"kind":"component","stories":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(normalManifest, []byte(`{"libraryId":"react-component-library:Button","latest":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	db, err := openGateDB(context.Background(), filepath.Join(root, "react-component-library.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(context.Background(), `CREATE TABLE component_test_reports (root_library_id TEXT, root_version TEXT, source_revision TEXT, created_at TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), `INSERT INTO component_test_reports(root_library_id, root_version, source_revision, created_at) VALUES ('react-component-library:Button', '1.0.0', 'revision', '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	result, err := ValidateEvidenceFreshness(Scope{Context: context.Background(), Root: root, DB: db.Primary(), Revision: func(string, string) (string, error) {
		return "revision", nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 || len(result.Findings) != 0 || len(result.RunnerError) != 0 {
		t.Fatalf("supplemental freshness result = %+v, want only the normal asset measured", result)
	}
}

func TestReleasedVersionImmutableFallsBackToHashLedgerAndDetectsOneByteMutation(t *testing.T) {
	root := t.TempDir()
	libraryRoot := filepath.Join(root, "scenarios", "react-component-library", "library")
	sourcePath := filepath.Join(libraryRoot, "components", "Fixture", "versions", "1.0.0", "Fixture.tsx")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatal(err)
	}
	released := []byte("export const Fixture = () => null;\n")
	if err := os.WriteFile(sourcePath, released, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(released)
	ledger := fmt.Sprintf(`{"schemaVersion":1,"entries":[{"path":"components/Fixture/versions/1.0.0/Fixture.tsx","sha256":"%s"}]}`, hex.EncodeToString(digest[:]))
	if err := os.WriteFile(filepath.Join(libraryRoot, "released-version-hashes.json"), []byte(ledger), 0o600); err != nil {
		t.Fatal(err)
	}
	clean, err := ValidateReleasedVersionImmutable(Scope{Root: root})
	if err != nil || clean.Inspected != 1 || len(clean.Findings) != 0 {
		t.Fatalf("clean fallback = %+v, err=%v", clean, err)
	}
	if err := os.WriteFile(sourcePath, append(released, ' '), 0o600); err != nil {
		t.Fatal(err)
	}
	drifted, err := ValidateReleasedVersionImmutable(Scope{Root: root})
	if err != nil || len(drifted.Findings) != 1 {
		t.Fatalf("drifted fallback = %+v, err=%v", drifted, err)
	}
}

func TestReleasedVersionImmutableAcceptsCanonicalTerminalLF(t *testing.T) {
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
	if err := os.WriteFile(sourcePath, append(append([]byte(nil), released...), '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(released)
	db, err := openGateDB(context.Background(), filepath.Join(dbDir, "react-component-library.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.ExecContext(context.Background(), `CREATE TABLE component_versions (status TEXT NOT NULL, source_path TEXT NOT NULL, content_sha256 TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(context.Background(), `INSERT INTO component_versions(status, source_path, content_sha256) VALUES ('released', ?, ?)`, "components/Fixture/versions/1.0.0/Fixture.tsx", hex.EncodeToString(hash[:])); err != nil {
		t.Fatal(err)
	}
	result, err := ValidateReleasedVersionImmutable(Scope{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("result = %+v, want terminal LF to be canonical", result)
	}
}

func TestLiveTokenGate(t *testing.T) {
	result, err := ValidateTokens(Scope{Root: liveRoot(t)})
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
	result, err := ValidateAPI(Scope{Root: liveRoot(t)})
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
	result, err := ValidateAPI(Scope{Root: root})
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
	result, err := ValidateFixtures(Scope{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 || len(result.Findings) != 1 || result.Findings[0].Code != "catalog.fixture_adversarial" {
		t.Fatalf("result = %+v", result)
	}
}

func TestTokenGateRejectsLiteralDimensionInImplementation(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "templates", "design", "_base", "tokens.css")
	if err := os.MkdirAll(filepath.Dir(base), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(base, []byte(":root {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
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
	result, err := ValidateTokens(Scope{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 || len(result.Findings) != 1 || result.Findings[0].Code != "catalog.tokens_literal" {
		t.Fatalf("result = %+v", result)
	}
}

func TestScenarioTokenRequirementsGateRejectsMissingImportedProperty(t *testing.T) {
	root := t.TempDir()
	write := func(relative, content string) {
		t.Helper()
		path := filepath.Join(root, relative)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}
	write("templates/design/_base/tokens.css", ":root {\n  /* @tier Expression */\n  --color-needed: blue;\n  /* @tier Contract */\n  --layer-modal: 400;\n}\n")
	write("scenarios/react-component-library/library/components/Fixture/component.json", `{"libraryId":"react-component-library:Fixture"}`)
	write("scenarios/react-component-library/library/components/Fixture/versions/1.0.0/Fixture.tsx", `export const Fixture = () => <div style={{ color: "var(--color-needed)", zIndex: "var(--layer-modal)" }} />;`)
	write("scenarios/fixture/ui/src/App.tsx", `import { Fixture } from "@vrooli/react-component-library/Fixture"; export const App = Fixture;`)
	write("scenarios/fixture/ui/src/design-tokens.css", ":root {\n/* rcl:tokens:begin */\n/* rcl:tokens:end */\n}\n")

	result, err := ValidateScenarioTokenRequirements(Scope{Root: root})
	require.NoError(t, err)
	require.Len(t, result.Findings, 1)
	require.Equal(t, "catalog.scenario_token_requirements", result.Findings[0].Code)
	require.Contains(t, result.Findings[0].Message, "--color-needed")
	require.NotContains(t, result.Findings[0].Message, "--layer-modal")
}

func TestFallbackParityReportsOnlyDisagreeingExternalFallback(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "templates", "design", "_base")
	require.NoError(t, os.MkdirAll(base, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(base, "tokens.css"), []byte(":root {\n  --space-md: 24px;\n  --radius-panel: 0.5rem;\n}\n"), 0o644))
	version := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Panel", "versions", "1.0.0")
	require.NoError(t, os.MkdirAll(version, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(version, "Panel.tsx"), []byte("const css = `padding: var(--space-md, 1rem); border-radius: var(--radius-panel, 0.5rem); color: var(--local, red); --local: blue;`;"), 0o644))

	result, err := ValidateFallbackParity(Scope{Root: root})
	require.NoError(t, err)
	require.Len(t, result.Findings, 1)
	require.Equal(t, "catalog.fallback_parity", result.Findings[0].Code)
	require.Contains(t, result.Findings[0].Message, "--space-md")
}

func TestTokenVocabularyAllowsCompatibilityAliasDeclarationsButRejectsReferences(t *testing.T) {
	root := t.TempDir()
	version := filepath.Join(root, "scenarios", "react-component-library", "library", "foundations", "BaseStyles", "versions", "1.0.0")
	require.NoError(t, os.MkdirAll(version, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(version, "BaseStyles.ts"), []byte(`const css = ":root { --app-surface: var(--color-surface); }";`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "scenarios", "react-component-library", "library", "foundations", "BaseStyles", "component.json"), []byte(`{"libraryId":"react-component-library:BaseStyles","latest":"1.0.0"}`), 0o644))

	result, err := ValidateTokenVocabulary(Scope{Root: root})
	require.NoError(t, err)
	require.Empty(t, result.Findings)

	require.NoError(t, os.WriteFile(filepath.Join(version, "BaseStyles.ts"), []byte(`const css = ":root { --app-surface: var(--color-surface); color: var(--app-surface); }";`), 0o644))
	result, err = ValidateTokenVocabulary(Scope{Root: root})
	require.NoError(t, err)
	require.Len(t, result.Findings, 1)
	require.Equal(t, "catalog.token_vocabulary", result.Findings[0].Code)
}

func TestLiveFallbackParity(t *testing.T) {
	result, err := ValidateFallbackParity(Scope{Root: liveRoot(t)})
	require.NoError(t, err)
	for _, finding := range result.Findings {
		t.Logf("%s:%d: %s", finding.File, finding.Line, finding.Message)
	}
	require.Empty(t, result.Findings)
}

func TestCompatibilityGatesRejectBadVerdictsAndOverclaims(t *testing.T) {
	census := Census{
		ComponentsScanned: 4,
		Components: []ComponentTokenCensus{
			{LibraryID: "react-component-library:Universal", Verdict: CompatibilityUniversal},
			{LibraryID: "react-component-library:Restricted", Verdict: CompatibilityRestricted},
			{LibraryID: "react-component-library:Impossible", Verdict: CompatibilityUnsatisfiable},
			{LibraryID: "react-component-library:Undefined", Verdict: CompatibilityUndefinedVocabulary, RequiredTokens: []string{"--missing"}},
		},
		AffinityOverclaims: []AffinityOverclaim{{LibraryID: "react-component-library:Restricted", StyleID: "kit-b"}},
	}
	require.Len(t, compatibilityGateResult(census, false).Findings, 2)
	require.Len(t, compatibilityGateResult(census, true).Findings, 1)
}

func TestLiveKitCompatibility(t *testing.T) {
	result, err := ValidateKitCompatibility(Scope{Root: liveRoot(t)})
	require.NoError(t, err)
	require.Empty(t, result.Findings)
}

func TestLiveAffinityCompatibility(t *testing.T) {
	result, err := ValidateAffinityNotBroaderThanCompatibility(Scope{Root: liveRoot(t)})
	require.NoError(t, err)
	require.Empty(t, result.Findings)
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
	result, err := ValidateLifecycle(Scope{Root: root})
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
	result, err := ValidateLifecycle(Scope{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 || len(result.Findings) != 0 {
		t.Fatalf("result = %+v, want only runtime source inspected with no findings", result)
	}
}

func TestLifecycleGateIgnoresBehaviorTests(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "scenarios", "react-component-library", "library", "hooks", "useViewport", "versions", "1.0.0")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	implementation := `export function useViewport() {
  return typeof window === "undefined" || typeof document === "undefined" ? 0 : window.innerHeight;
}`
	if err := os.WriteFile(filepath.Join(source, "useViewport.ts"), []byte(implementation), 0o644); err != nil {
		t.Fatal(err)
	}
	testSource := `it("subscribes", () => {
  window.addEventListener("resize", () => {});
  document.body.append(document.createElement("input"));
});`
	if err := os.WriteFile(filepath.Join(source, "useViewport.behavior.test.tsx"), []byte(testSource), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ValidateLifecycle(Scope{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 || len(result.Findings) != 0 {
		t.Fatalf("result = %+v, want behavior test excluded from runtime lifecycle findings", result)
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
	result, err := ValidateLifecycle(Scope{Root: root})
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
	result, err := ValidateLifecycle(Scope{Root: root})
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
	runners := map[string]Runner{
		"api":           ValidateAPI,
		"tokens":        ValidateTokens,
		"conformance":   ValidateConformance,
		"lifecycle":     ValidateLifecycle,
		"fixtures":      ValidateFixtures,
		"examples":      ValidateExamples,
		"rtl":           ValidateRTL,
		"documentation": ValidateDocumentation,
		"reduced":       ValidateReducedMotion,
		"integrate":     ValidateIntegration,
		"vocabulary":    ValidateTokenVocabulary,
	}
	for name, run := range runners {
		t.Run(name, func(t *testing.T) {
			result, err := run(Scope{Root: root})
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
		run  Runner
	}{
		{name: "api", run: ValidateAPI},
		{name: "tokens", run: ValidateTokens},
		{name: "conformance", run: ValidateConformance},
		{name: "lifecycle", run: ValidateLifecycle},
		{name: "fixtures", run: ValidateFixtures},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.run(Scope{Root: root})
			if err != nil {
				t.Fatal(err)
			}
			if result.Inspected != 0 || len(result.Findings)+len(result.RunnerError) == 0 {
				t.Fatalf("result = %+v, want zero-input finding", result)
			}
		})
	}
}

func TestUnmeasuredGateDoesNotReadSourceAsEvidence(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Button", "component.json")
	if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, []byte(`{"catalogId":"controls.button","latest":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := UnmeasuredGate(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "unmeasured" || result.Inspected != 1 || len(result.InspectedAssets) != 1 {
		t.Fatalf("result = %+v, want one explicitly unmeasured asset", result)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("unmeasured asset selection produced findings: %+v", result.Findings)
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
	result, err := ValidateConformance(Scope{Root: root})
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

func TestNormalizeResultRemovesCorpusPseudoAssets(t *testing.T) {
	result := NormalizeResult(t.TempDir(), Result{Findings: []Finding{{Code: "conformance.type-scale", AssetID: "workbench.conformance"}}})
	if len(result.Findings) != 1 || len(result.RunnerError) != 0 {
		t.Fatalf("corpus finding normalization = %+v, want one finding and no runner errors", result)
	}
	if result.Findings[0].Scope != FindingScopeCorpus || result.Findings[0].Severity != FindingSeverityWarning {
		t.Fatalf("corpus finding attribution = %+v, want corpus/warning", result.Findings[0])
	}
	if result.Findings[0].AssetID != "" || result.Findings[0].CatalogID != "" {
		t.Fatalf("corpus finding retains pseudo identity: %+v", result.Findings[0])
	}
}

func TestNormalizeResultAssignsTypedAssetAttribution(t *testing.T) {
	root := t.TempDir()
	assetDir := filepath.Join(root, "scenarios", "react-component-library", "catalog", "assets", "components")
	require.NoError(t, os.MkdirAll(assetDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(assetDir, "dialog.json"), []byte(`{"asset":{"id":"react-component-library:Dialog"}}`), 0o644))
	result := NormalizeResult(root, Result{Findings: []Finding{{Code: "catalog.example", AssetID: "react-component-library:Dialog", File: "library/components/Dialog/component.json"}}})
	require.Len(t, result.Findings, 1)
	require.Empty(t, result.RunnerError)
	require.Equal(t, FindingScopeAsset, result.Findings[0].Scope)
	require.Equal(t, FindingSeverityBlocking, result.Findings[0].Severity)
	require.Equal(t, "react-component-library:Dialog", result.Findings[0].CatalogID)
	require.Equal(t, "react-component-library:Dialog", result.Findings[0].LibraryID)
	require.Equal(t, "library/components/Dialog/component.json", result.Findings[0].Owner)
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
	result, err := ValidateFixtures(Scope{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 1 || result.Findings[0].Code != "catalog.fixture_data_source" {
		t.Fatalf("result = %+v", result)
	}
}

func TestValidateDependencyRankRejectsPrimitiveImportingComponent(t *testing.T) {
	root := t.TempDir()
	library := filepath.Join(root, "scenarios", "react-component-library", "library")
	primitive := filepath.Join(library, "primitives", "Stack")
	component := filepath.Join(library, "components", "Dialog")
	require.NoError(t, os.MkdirAll(filepath.Join(primitive, "versions", "1.0.0"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(component, "versions", "1.0.0"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(primitive, "component.json"), []byte(`{"libraryId":"react-component-library:Stack"}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(component, "component.json"), []byte(`{"libraryId":"react-component-library:Dialog"}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(primitive, "versions", "1.0.0", "dependencies.json"), []byte(`{"libraryId":"react-component-library:Stack","version":"1.0.0","dependencies":[{"libraryId":"react-component-library:Dialog","version":"1.0.0"}]}`), 0o600))

	result, err := ValidateDependencyRank(Scope{Root: root})
	require.NoError(t, err)
	require.Len(t, result.Findings, 1)
	require.Equal(t, "catalog.dependency_rank", result.Findings[0].Code)
}

func TestLiveDependencyLocksObeyCompositionRank(t *testing.T) {
	result, err := ValidateDependencyRank(Scope{Root: liveRoot(t)})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) > 0 {
		t.Fatalf("dependency-rank findings: %+v", result.Findings)
	}
}

func TestValidateReleaseProvenanceRejectsHandCreatedRelease(t *testing.T) {
	root := t.TempDir()
	library := filepath.Join(root, "scenarios", "react-component-library", "library")
	asset := filepath.Join(library, "components", "Dialog")
	require.NoError(t, os.MkdirAll(filepath.Join(asset, "versions", "1.0.0"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(asset, "component.json"), []byte(`{"libraryId":"react-component-library:Dialog"}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(library, "release-provenance.json"), []byte(`{"schemaVersion":1,"entries":[]}`), 0o600))

	result, err := ValidateReleaseProvenance(Scope{Root: root})
	require.NoError(t, err)
	require.Len(t, result.Findings, 1)
	require.Equal(t, "catalog.release_provenance_missing", result.Findings[0].Code)
	require.Equal(t, "react-component-library:Dialog", result.Findings[0].AssetID)
}

func TestLiveReleaseProvenanceCoversEveryMaterializedRelease(t *testing.T) {
	result, err := ValidateReleaseProvenance(Scope{Root: liveRoot(t)})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.RunnerError) > 0 || len(result.Findings) > 0 {
		t.Fatalf("release-provenance result: %+v", result)
	}
}

func TestLiveSelfHostingDoesNotRegressBelowReviewedFloor(t *testing.T) {
	result, err := ValidateSelfHosting(Scope{Root: liveRoot(t)})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.RunnerError) > 0 || len(result.Findings) > 0 {
		t.Fatalf("self-hosting result: %+v", result)
	}
}

func TestLiveBASIsCapabilityDriven(t *testing.T) {
	result, err := ValidateBASGenericity(Scope{Root: liveRoot(t)})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.RunnerError) > 0 || len(result.Findings) > 0 {
		t.Fatalf("bas-genericity result: %+v", result)
	}
}

func TestBASGenericityRejectsAssetKnowledgeAndVersionPins(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Button")
	caseDir := filepath.Join(root, "scenarios", "react-component-library", "bas", "cases")
	calibrationDir := filepath.Join(root, "scenarios", "react-component-library", "bas", "calibration")
	require.NoError(t, os.MkdirAll(manifest, 0o755))
	require.NoError(t, os.MkdirAll(caseDir, 0o755))
	require.NoError(t, os.MkdirAll(calibrationDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(manifest, "component.json"), []byte(`{"displayName":"Button"}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(caseDir, "bad.json"), []byte(`{"url":"/story?version=1.0.0","asset":"Button"}`), 0o600))
	result, err := ValidateBASGenericity(Scope{Root: root})
	require.NoError(t, err)
	require.Len(t, result.Findings, 2)
}
