package channelmanager

import (
	"path/filepath"
	"testing"
)

// [REQ:CHANMGR-P0-003] Shipped declarative descriptors load as one valid,
// structurally varied set rather than accepting ad-hoc platform state.
func TestShippedDescriptorsAreValidAndStructurallyDifferent(t *testing.T) {
	p, w, e := LoadDescriptors(filepath.Join("..", "..", "..", "data"))
	if e != nil {
		t.Fatal(e)
	}
	if len(p) != 4 || len(w) != 2 {
		t.Fatalf("want four platform/two program descriptors, got %d/%d", len(p), len(w))
	}
	if _, e = New(p, w); e != nil {
		t.Fatal(e)
	}
	for _, platform := range p {
		if len(platform.Formats) == 0 {
			t.Fatalf("platform %q has no format constraints", platform.ID)
		}
		if len(platform.PostTypes) == 0 {
			t.Fatalf("platform %q has no explicit post-type contract", platform.ID)
		}
		if !platform.Provenance.Valid() {
			t.Fatalf("platform %q has no complete source provenance", platform.ID)
		}
	}
	if p[0].ActionKinds[0] == p[1].ActionKinds[0] && len(p[0].ActionKinds) == len(p[1].ActionKinds) {
		t.Fatal("descriptors must exercise different shapes")
	}
}
