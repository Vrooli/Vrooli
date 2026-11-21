package gates

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLibraryAssetForPathIdentifiesTheOwningAsset(t *testing.T) {
	libraryRoot := filepath.Join("/repo", "scenarios", "react-component-library", "library")
	for _, testCase := range []struct{ file, want string }{
		{filepath.Join(libraryRoot, "components", "IconButton", "versions", "3.1.1", "IconButton.tsx"), "IconButton"},
		{filepath.Join(libraryRoot, "foundations", "ClassMerge", "versions", "1.0.2", "ClassMerge.tsx"), "ClassMerge"},
		{filepath.Join(libraryRoot, "hooks", "useLocale", "versions", "1.0.1", "useLocale.ts"), "useLocale"},
		{filepath.Join("/repo", "scenarios", "react-component-library", "ui", "src", "App.tsx"), ""},
		{filepath.Join(libraryRoot, "vacuous-allowlist.json"), ""},
	} {
		if got := libraryAssetForPath(libraryRoot, testCase.file); got != testCase.want {
			t.Fatalf("libraryAssetForPath(%q) = %q, want %q", testCase.file, got, testCase.want)
		}
	}
}

// A failing catalog:check used to produce one finding with an empty AssetID.
// The evidence mapper matches a finding to an asset by exact id, so nothing
// matched and every asset recorded `types: pass` for a run that failed.
func TestAttributeCatalogDiagnosticsBindsErrorsToAssets(t *testing.T) {
	root := t.TempDir()
	libraryRoot := filepath.Join(root, "scenarios", "react-component-library", "library")
	report := filepath.Join(root, "report.json")
	write := func(body string) {
		if err := os.WriteFile(report, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write(`{"schemaVersion":1,"diagnostics":[
		{"file":"` + filepath.Join(libraryRoot, "components", "IconButton", "versions", "3.1.1", "IconButton.tsx") + `","line":195,"severity":"error","message":"TS2322: bad","source":"tsc"},
		{"file":"` + filepath.Join(libraryRoot, "components", "IconButton", "versions", "3.1.1", "story.tsx") + `","line":170,"severity":"error","message":"TS2322: also bad","source":"tsc"},
		{"file":"` + filepath.Join(libraryRoot, "foundations", "ClassMerge", "versions", "1.0.2", "ClassMerge.tsx") + `","line":43,"severity":"warning","message":"unsafe","source":"eslint"}
	]}`)
	findings, unattributed := attributeCatalogDiagnostics(root, report)
	if unattributed {
		t.Fatal("every error names a library file, so none should be unattributed")
	}
	if len(findings) != 1 || findings[0].AssetID != "IconButton" {
		t.Fatalf("findings = %+v, want one finding for IconButton (two errors, one asset)", findings)
	}
	if findings[0].Line != 195 {
		t.Fatalf("finding line = %d, want the first error's line", findings[0].Line)
	}

	// A warning alone is not a failure and must not fail an asset.
	write(`{"schemaVersion":1,"diagnostics":[{"file":"` + filepath.Join(libraryRoot, "components", "Card", "versions", "1.0.0", "Card.tsx") + `","line":3,"severity":"warning","message":"unsafe","source":"eslint"}]}`)
	findings, unattributed = attributeCatalogDiagnostics(root, report)
	if len(findings) != 0 || unattributed {
		t.Fatalf("warnings must not attribute a failure: findings=%+v unattributed=%v", findings, unattributed)
	}

	// An error outside library/ belongs to no asset, which must force the
	// fail-closed corpus path rather than silently vanishing.
	write(`{"schemaVersion":1,"diagnostics":[{"file":"` + filepath.Join(root, "scenarios", "react-component-library", "ui", "src", "App.tsx") + `","line":9,"severity":"error","message":"TS2322: bad","source":"tsc"}]}`)
	findings, unattributed = attributeCatalogDiagnostics(root, report)
	if len(findings) != 0 || !unattributed {
		t.Fatalf("an unowned error must report unattributed: findings=%+v unattributed=%v", findings, unattributed)
	}

	// An unreadable or unparseable report can never be read as "no problems".
	if _, unattributed := attributeCatalogDiagnostics(root, filepath.Join(root, "absent.json")); !unattributed {
		t.Fatal("a missing report must fail closed")
	}
	write(`not json`)
	if _, unattributed := attributeCatalogDiagnostics(root, report); !unattributed {
		t.Fatal("an unparseable report must fail closed")
	}
}

func TestInspectedFromReportPrefersTheRealCountAndFailsSafe(t *testing.T) {
	root := t.TempDir()
	report := filepath.Join(root, "report.json")

	if err := os.WriteFile(report, []byte(`{"schemaVersion":1,"inspected":3,"diagnostics":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if inspected, ok := inspectedFromReport(report); !ok || inspected != 3 {
		t.Fatalf("inspected = %d, ok = %v; want 3, true", inspected, ok)
	}

	// A scoped run that legitimately checked nothing is impossible — the script
	// refuses an empty scope — but a zero must still be reported as a zero
	// rather than silently replaced, so the caller's own guard can fire.
	if err := os.WriteFile(report, []byte(`{"schemaVersion":1,"inspected":0,"diagnostics":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if inspected, ok := inspectedFromReport(report); !ok || inspected != 0 {
		t.Fatalf("inspected = %d, ok = %v; want 0, true", inspected, ok)
	}

	// An unusable report must leave the caller's corpus count alone rather than
	// substitute a zero, which would read as "this gate inspected nothing".
	for _, body := range []string{`not json`, `{"schemaVersion":1,"diagnostics":[]}`} {
		if err := os.WriteFile(report, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, ok := inspectedFromReport(report); ok {
			t.Fatalf("report %q should be unusable", body)
		}
	}
	if _, ok := inspectedFromReport(filepath.Join(root, "absent.json")); ok {
		t.Fatal("a missing report should be unusable")
	}
}
