package depgraph

import (
	"reflect"
	"testing"
)

func TestTransitiveClosure(t *testing.T) {
	g := New()
	// a -> b -> c ; a -> d ; d -> c ; isolated e
	g.AddNode("a", []string{"b", "d"})
	g.AddNode("b", []string{"c"})
	g.AddNode("c", nil)
	g.AddNode("d", []string{"c"})
	g.AddNode("e", nil)

	got := g.TransitiveClosure([]string{"a"})
	want := []string{"a", "b", "c", "d"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("TransitiveClosure(a) = %v, want %v", got, want)
	}

	// Multiple roots de-dupe.
	got = g.TransitiveClosure([]string{"b", "d"})
	want = []string{"b", "c", "d"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("TransitiveClosure(b,d) = %v, want %v", got, want)
	}
}

func TestTransitiveClosure_CycleSafe(t *testing.T) {
	g := New()
	g.AddNode("a", []string{"b"})
	g.AddNode("b", []string{"c"})
	g.AddNode("c", []string{"a"}) // cycle a->b->c->a
	got := g.TransitiveClosure([]string{"a"})
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("cycle closure = %v, want %v (must terminate)", got, want)
	}
}

func TestTransitiveClosure_UnknownDepIncludedAsLeaf(t *testing.T) {
	g := New()
	g.AddNode("a", []string{"external/x"}) // external/x is not a node
	got := g.TransitiveClosure([]string{"a"})
	want := []string{"a", "external/x"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("closure with unknown dep = %v, want %v", got, want)
	}
}

func TestTransitiveDependents(t *testing.T) {
	g := New()
	g.AddNode("a", []string{"b", "d"})
	g.AddNode("b", []string{"c"})
	g.AddNode("c", nil)
	g.AddNode("d", []string{"c"})
	// Who transitively depends on c? b (b->c), d (d->c), a (a->b, a->d).
	got := g.TransitiveDependents([]string{"c"})
	want := []string{"a", "b", "d"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("TransitiveDependents(c) = %v, want %v", got, want)
	}
}
