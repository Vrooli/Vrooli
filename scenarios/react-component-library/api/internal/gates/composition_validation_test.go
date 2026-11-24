package gates

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/storage/storagetest"
	_ "modernc.org/sqlite"
)

// compositionFixture creates the smallest real catalog/library layout accepted
// by the public validators. Each asset gets a versioned implementation so the
// test exercises source discovery as well as composition policy.
func compositionFixture(t *testing.T, assets map[string]compositionAssetFixture) Scope {
	t.Helper()
	root := t.TempDir()
	for id, asset := range assets {
		separator := strings.IndexByte(id, '.')
		if separator < 1 || separator == len(id)-1 {
			t.Fatalf("fixture asset id %q must have kind.name form", id)
		}
		kind, name := id[:separator], id[separator+1:]
		catalog := filepath.Join(root, "scenarios/react-component-library/catalog/assets", kind, name+".json")
		if err := os.MkdirAll(filepath.Dir(catalog), 0o755); err != nil {
			t.Fatal(err)
		}
		doc := map[string]any{"asset": map[string]any{"id": id, "name": name, "kind": "component"}}
		if asset.provides != nil {
			doc["provides"] = asset.provides
		}
		if asset.consumes != nil {
			doc["consumes"] = asset.consumes
		}
		data, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(catalog, data, 0o644); err != nil {
			t.Fatal(err)
		}
		library := filepath.Join(root, "scenarios/react-component-library/library/components", name)
		if err := os.MkdirAll(filepath.Join(library, "versions/1.0.0"), 0o755); err != nil {
			t.Fatal(err)
		}
		manifest := map[string]any{"catalogId": id, "latest": "1.0.0"}
		data, err = json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(library, "component.json"), data, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(library, "versions/1.0.0", name+".tsx"), []byte(asset.source), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return Scope{Context: context.Background(), Root: root}
}

type compositionAssetFixture struct {
	source             string
	provides, consumes map[string]any
}

func TestValidateCompositionContractAcceptsMatchingSharedContext(t *testing.T) {
	scope := compositionFixture(t, map[string]compositionAssetFixture{
		"components.provider": {source: `export function Provider(){ return <SurfaceProvider value={{elevation:"raised"}} /> }`, provides: map[string]any{"elevation": "raised"}},
		"components.consumer": {source: `export function Consumer(){ const value = useSurfaceContext("elevation"); return <div>{value}</div> }`, consumes: map[string]any{"elevation": "flat"}},
	})
	result, err := ValidateCompositionContract(scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("findings = %+v, want no findings", result.Findings)
	}
	if result.Inspected != 2 {
		t.Fatalf("inspected = %d, want 2", result.Inspected)
	}
}

func TestValidateCompositionContractReportsOrphanAndSourceMismatches(t *testing.T) {
	scope := compositionFixture(t, map[string]compositionAssetFixture{
		"components.provider": {source: `export const Provider = () => <div />`, provides: map[string]any{"orphan": "x", "elevation": "raised"}},
		"components.consumer": {source: `export const Consumer = () => <div style={{boxShadow:"var(--elev-raised)"}} />`, consumes: map[string]any{"missing": "x", "elevation": "flat"}},
	})
	result, err := ValidateCompositionContract(scope)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, finding := range result.Findings {
		got[finding.Code] = true
	}
	for _, code := range []string{"catalog.composition_contract_provider_not_shared", "catalog.composition_contract_context_missing", "catalog.composition_contract_contexts_differ", "catalog.composition_contract_no_consumer", "catalog.composition_contract_orphan_consumer"} {
		if !got[code] {
			t.Errorf("missing finding %q; findings=%+v", code, result.Findings)
		}
	}
}

func TestValidateCompositionReportsMissingCaptureAsUnmeasured(t *testing.T) {
	storagetest.IsolateStorage(t)
	scope := compositionFixture(t, map[string]compositionAssetFixture{"components.card": {source: `export const Card = () => <div />`}})
	result, err := ValidateComposition(scope)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected != 1 || len(result.UnmeasuredAssets) != 1 || result.UnmeasuredAssets[0] != "components.card" {
		t.Fatalf("capture census = inspected:%d unmeasured:%v, want one unmeasured asset", result.Inspected, result.UnmeasuredAssets)
	}
	if result.SurfaceCounts["measured"] != 0 {
		t.Fatalf("surface counts = %v, want zero measured", result.SurfaceCounts)
	}
}

func TestStampedNodeFindsNestedIdentityAndRejectsUnstampedTree(t *testing.T) {
	tree := axObservation{DOM: axDOM{Tag: "main"}, Children: []axObservation{{DOM: axDOM{Tag: "section", Attributes: map[string]string{"data-rcl-asset": "components.card"}}}}}
	if node, ok := stampedNode(&tree, "components.card"); !ok || node.DOM.Tag != "section" {
		t.Fatalf("nested stamped node = %#v, %v", node, ok)
	}
	if _, ok := stampedNode(&tree, "components.missing"); ok {
		t.Fatal("unstamped identity unexpectedly found")
	}
}

func TestQueryCompositionCapturesKeepsNewestValidTreePerAsset(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE reconcile_evidence (id INTEGER PRIMARY KEY, scenario TEXT, document_kind TEXT, ax_node_json TEXT, checked_at TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	rows := []struct{ checked, tree string }{
		{"2026-01-01", `{"dom":{"tag":"div","attributes":{"data-rcl-asset":"components.card"}}}`},
		{"2026-02-01", `{"dom":{"tag":"section","attributes":{"data-rcl-asset":"components.card"}}}`},
		{"2026-03-01", `not json`},
		{"2026-01-15", `{"dom":{"tag":"article","attributes":{"data-rcl-asset":"components.other"}}}`},
	}
	for _, row := range rows {
		if _, err := db.Exec(`INSERT INTO reconcile_evidence (scenario, document_kind, ax_node_json, checked_at) VALUES ('react-component-library','component',?,?)`, row.tree, row.checked); err != nil {
			t.Fatal(err)
		}
	}
	captures, err := queryCompositionCaptures(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(captures) != 2 {
		t.Fatalf("capture ids = %v, want newest valid row for both assets", captures)
	}
	if got := captures["components.card"]; got.CheckedAt != "2026-02-01" || got.Tree.DOM.Tag != "section" {
		t.Fatalf("card capture = %+v, want newest valid tree", got)
	}
}
