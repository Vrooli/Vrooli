package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	pkg "github.com/vrooli/ai-go/search"
)

// repoRootForTuningTest walks up from the test's working directory until it finds
// the committed KO search.json, returning the repo root. Mirrors the other
// repo-root helpers in this package (no shared helper exists yet).
func repoRootForTuningTest(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	for {
		candidate := filepath.Join(dir, "scenarios", "knowledge-observatory", ".vrooli", "search.json")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return dir
		}
		next := filepath.Dir(dir)
		if next == dir {
			t.Fatal("repo root (with scenarios/knowledge-observatory/.vrooli/search.json) not found")
		}
		dir = next
	}
}

// TestLoadDocsTuningMatchesPreset is a non-live per-build guard: the committed
// `.vrooli/search.json` must resolve to exactly the DocCorpusTuning() fallback
// preset. Without this, a divergence between the file and the preset (e.g. someone
// flips rerank_enabled in one but not the other) would only surface in the
// env-gated live recall gate — never in CI. It also exercises the real
// loadDocsTuning() path (file resolution + engine guard) end to end.
func TestLoadDocsTuningMatchesPreset(t *testing.T) {
	root := repoRootForTuningTest(t)
	s := &Server{config: &Config{ScenariosRoot: filepath.Join(root, "scenarios")}}

	got := s.loadDocsTuning()
	want := pkg.DocCorpusTuning().WithDefaults()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("committed search.json tuning drifted from DocCorpusTuning() preset:\n got: %+v\nwant: %+v", got, want)
	}
	if got.Engine != pkg.EngineHybrid {
		t.Errorf("KO docs engine must be hybrid (hybrid by construction), got %q", got.Engine)
	}
}

// TestLoadDocsTuningPinsEngineToHybrid proves the invariant guard: KO's read+index
// path is hybrid-only (internal/aisearch always builds a NewHybridBinding), so a
// search.json that declares a non-hybrid engine — including the WithDefaults
// "dense" fallback for an omitted field — is corrected to hybrid rather than
// silently honored against an engine that cannot satisfy it.
func TestLoadDocsTuningPinsEngineToHybrid(t *testing.T) {
	scenariosRoot := t.TempDir()
	dir := filepath.Join(scenariosRoot, "knowledge-observatory", ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// A minimal-but-valid provider that (wrongly) asks for the dense engine.
	const denseSearchJSON = `{
  "version": "1.0.0",
  "providers": [{
    "provider_id": "knowledge-observatory.docs",
    "bucket": "BUCKET_KNOW",
    "type": "doc",
    "scope": "SCOPE_PROJECT",
    "endpoint": {"http_json": {"path": "/docs"}},
    "result_mapping": {"id_field": "path"},
    "tuning": {"engine": "dense"}
  }]
}`
	if err := os.WriteFile(filepath.Join(dir, "search.json"), []byte(denseSearchJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	s := &Server{config: &Config{ScenariosRoot: scenariosRoot}}
	got := s.loadDocsTuning()
	if got.Engine != pkg.EngineHybrid {
		t.Errorf("engine guard did not pin to hybrid: got %q", got.Engine)
	}
}
