package channelmanager

import (
	"path/filepath"
	"testing"
)

func TestShippedDescriptorsAreValidAndStructurallyDifferent(t *testing.T) {
	p, w, e := LoadDescriptors(filepath.Join("..", "..", "..", "data"))
	if e != nil {
		t.Fatal(e)
	}
	if len(p) != 2 || len(w) != 2 {
		t.Fatalf("want two platform/program descriptors, got %d/%d", len(p), len(w))
	}
	if _, e = New(p, w); e != nil {
		t.Fatal(e)
	}
	for _, platform := range p {
		if len(platform.Formats) == 0 {
			t.Fatalf("platform %q has no format constraints", platform.ID)
		}
	}
	if p[0].ActionKinds[0] == p[1].ActionKinds[0] && len(p[0].ActionKinds) == len(p[1].ActionKinds) {
		t.Fatal("descriptors must exercise different shapes")
	}
}
