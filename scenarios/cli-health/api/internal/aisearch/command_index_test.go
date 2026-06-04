package aisearch

import (
	"context"
	"testing"
)

// TestLoadAll_ExcludesHelpFailedStubs asserts WS3: help-failed stubs stay in
// discovery but never enter the vector index, so they cannot surface as
// semantic hits (and the reconciler ghost-deletes any previously-indexed ones).
func TestLoadAll_ExcludesHelpFailedStubs(t *testing.T) {
	disc := &staticDiscovery{
		scenarios: map[string][]CommandRecord{
			"real-scn": {{Origin: "real-scn", Name: "do", FullPath: "real-scn do", Source: SourceManifest}},
			"empty-scn": {{
				Origin: "empty-scn", Name: "empty-scn", FullPath: "empty-scn",
				Description: "Scenario empty-scn has no CLI manifest", Source: SourceHelpFailed,
			}},
		},
		externalCLIs: []ExternalCLI{{Name: "extcli", Binary: "extcli"}},
		external: map[string][]CommandRecord{
			"extcli": {{Origin: "extcli", Name: "extcli", FullPath: "extcli", Source: SourceHelpFailed}},
		},
	}
	docs, err := newCommandSource(disc).LoadAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected only the real command, got %d: %+v", len(docs), docs)
	}
	if docs[0].ID != "real-scn do" {
		t.Fatalf("unexpected doc: %+v", docs[0])
	}
}

func TestCommandContentHash_StableAndChanges(t *testing.T) {
	r := CommandRecord{
		Origin: "demo", Group: "x", Name: "y", FullPath: "demo x y",
		Description: "list things", Flags: []string{"json"}, Tags: []string{"effect:read"},
		Binding: "Svc.M", Source: SourceManifest,
	}
	h1 := commandContentHash(r)
	h2 := commandContentHash(r)
	if h1 == "" {
		t.Fatalf("content hash missing")
	}
	if h1 != h2 {
		t.Errorf("hash changed across no-op rebuild: %q vs %q", h1, h2)
	}

	edited := r
	edited.Description = "now with a different description"
	if commandContentHash(edited) == h1 {
		t.Errorf("hash unchanged after field edit (%q)", h1)
	}
}

func TestCommandToSourceDoc_Shape(t *testing.T) {
	r := CommandRecord{
		Origin: "demo", Group: "x", Name: "y", FullPath: "demo x y",
		Description: "list things", Source: SourceManifest,
	}
	doc := commandToSourceDoc(r)
	if doc.ID != "demo x y" {
		t.Errorf("ID = %q, want command full path", doc.ID)
	}
	if doc.Kind != commandKind {
		t.Errorf("Kind = %q, want %q", doc.Kind, commandKind)
	}
	if doc.ContentHash == "" {
		t.Error("ContentHash empty; source-level drift skip would never fire")
	}
	if doc.Body != composeCommandEmbeddingText(r) {
		t.Error("Body must be the pre-composed embedding text (identity composer embeds it verbatim)")
	}
	if got, _ := doc.Meta["full_path"].(string); got != "demo x y" {
		t.Errorf("Meta[full_path] = %q, want %q", got, "demo x y")
	}
}

func TestPayloadToHit_FullProjection(t *testing.T) {
	r := CommandRecord{
		Origin: "demo", Group: "x", Name: "y", FullPath: "demo x y",
		Description: "d", Flags: []string{"json"}, Tags: []string{"effect:read"},
		Binding: "Svc.M", Source: SourceManifest,
	}
	p := commandMeta(r)
	hit := payloadToHit("pt-1", 0.83, p)
	if hit.ID != "pt-1" || hit.Origin != "demo" || hit.FullPath != "demo x y" {
		t.Errorf("bad identity: %+v", hit)
	}
	if hit.Score != 0.83 || hit.ScorePercent != 83 {
		t.Errorf("score = %v / %d", hit.Score, hit.ScorePercent)
	}
	if hit.Binding != "Svc.M" || hit.Source != SourceManifest {
		t.Errorf("missing binding/source: %+v", hit)
	}
	if len(hit.Tags) != 1 || hit.Tags[0] != "effect:read" {
		t.Errorf("tags not projected: %+v", hit.Tags)
	}
}

func TestPointIDForCommand_StableAndUnique(t *testing.T) {
	a := pointIDForCommand("demo x list")
	b := pointIDForCommand("demo x list")
	c := pointIDForCommand("demo x show")
	if a != b {
		t.Errorf("not stable: %q vs %q", a, b)
	}
	if a == c {
		t.Errorf("collision: %q == %q", a, c)
	}
	if len(a) != 36 {
		t.Errorf("not UUID-shaped (len=%d): %q", len(a), a)
	}
}
