package graphreconcile

import (
	"context"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompareVerdictPrecedenceAndDeterminism(t *testing.T) {
	ids := []string{"z", "a", "empty", "phantom", "missing"}
	built := map[string]bool{"z": true, "a": true, "empty": true, "phantom": true, "missing": true}
	got := Compare(ids,
		map[string][]string{"a": {"dep"}, "empty": {"dep"}, "phantom": {"dep"}, "missing": {"dep"}},
		map[string][]string{"a": {"dep"}, "phantom": {"dep"}, "missing": {"other"}},
		map[string][]string{"a": {"dep"}, "empty": {"dep"}, "phantom": {}, "missing": {}}, built, true)
	if got.Assets[0].AssetID != "a" || got.Assets[0].Verdict != Reconciled {
		t.Fatalf("unexpected sorted result: %#v", got.Assets)
	}
	want := map[string]Verdict{"a": Reconciled, "empty": ManifestEmpty, "phantom": PhantomEdge, "missing": UndeclaredInManifest, "z": Reconciled}
	for _, row := range got.Assets {
		if row.Verdict != want[row.AssetID] {
			t.Errorf("%s: got %s want %s", row.AssetID, row.Verdict, want[row.AssetID])
		}
	}
	if Compare([]string{"a"}, nil, nil, nil, map[string]bool{"a": true}, false).Assets[0].Verdict != ImportsUnavailable {
		t.Fatal("unavailable import source must be typed")
	}
}

// An asset the catalog declares but nobody has built has one dependency view,
// not three. It must not be reported as manifest drift, and that must hold
// even when the import graph is missing, because it is knowable without it.
func TestCompareSeparatesUnbuiltAssetsFromManifestDrift(t *testing.T) {
	catalog := map[string][]string{"built": {"dep"}, "unbuilt": {"dep"}}
	for _, importsAvailable := range []bool{true, false} {
		got := Compare([]string{"built", "unbuilt"}, catalog, nil, nil, map[string]bool{"built": true}, importsAvailable)
		verdicts := map[string]Verdict{}
		for _, row := range got.Assets {
			verdicts[row.AssetID] = row.Verdict
		}
		if verdicts["unbuilt"] != NotImplemented {
			t.Errorf("importsAvailable=%v: unbuilt asset verdict=%s", importsAvailable, verdicts["unbuilt"])
		}
		if verdicts["built"] == NotImplemented {
			t.Errorf("importsAvailable=%v: built asset must not report not-implemented", importsAvailable)
		}
	}
}

func TestCompareUnpinnedImport(t *testing.T) {
	got := Compare([]string{"asset"}, map[string][]string{"asset": {"catalog"}}, map[string][]string{"asset": {"manifest"}}, map[string][]string{"asset": {"import"}}, map[string]bool{"asset": true}, true)
	if got.Assets[0].Verdict != UnpinnedImport {
		t.Fatalf("got %#v", got.Assets[0])
	}
}

// The extraction project is what makes the third dependency view exist at all.
// It regressed silently once by pointing at a directory with no tsconfig, so
// this asserts the two properties that made the old graph useless: it must
// reach the library corpus, and it must carry only published versions, because
// every version of an asset collapses onto one catalog ID.
func TestGraphSourceFilesCoverPublishedVersionsOnly(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	scenarioDir := filepath.Join(root, "scenarios", "react-component-library")
	libraryDir := filepath.Join(scenarioDir, "library")
	impls, err := loadImplementations(libraryDir)
	if err != nil {
		t.Fatal(err)
	}
	files, err := graphSourceFiles(libraryDir, filepath.Join(scenarioDir, "ui"), impls)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < len(impls) {
		t.Fatalf("resolved %d sources for %d implementations", len(files), len(impls))
	}
	published := map[string]bool{}
	for _, impl := range impls {
		for _, version := range []string{impl.Latest, impl.Draft} {
			if version = strings.TrimSpace(version); version != "" {
				published[path.Join("library", impl.Root, impl.Name, "versions", version)] = true
			}
		}
	}
	for _, file := range files {
		if !strings.HasPrefix(file, "../library/") {
			t.Fatalf("source escapes the library corpus: %s", file)
		}
		dir := path.Dir(strings.TrimPrefix(file, "../"))
		if !published[dir] {
			t.Fatalf("retired version entered the extraction project: %s", file)
		}
	}
}

func TestLiveReconcileReportsEveryCatalogAsset(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	report, err := Reconcile(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Assets) != 428 {
		t.Fatalf("assets=%d", len(report.Assets))
	}
	if len(report.Distribution) == 0 {
		t.Fatal("empty verdict distribution")
	}
	for _, row := range report.Assets {
		if row.AssetID == "data-display.data-table" {
			found := false
			for _, edge := range row.CatalogEdges {
				if edge == "services.selection-store" {
					found = true
				}
			}
			if !found {
				t.Fatal("data table oracle edge missing")
			}
			if row.Verdict == ImportsUnavailable {
				return
			}
			if row.Verdict != UndeclaredInManifest && row.Verdict != UnpinnedImport {
				t.Fatalf("data table verdict=%s", row.Verdict)
			}
			return
		}
	}
	t.Fatal("data table row missing")
}
