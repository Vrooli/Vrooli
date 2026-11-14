package adapters

import "testing"

type testAdapter struct {
	id      Identity
	matches bool
}

func (a testAdapter) Identity() Identity { return a.id }
func (a testAdapter) Matches(Match) bool { return a.matches }

func TestRegistryFailsClosedOnDuplicateAndAmbiguousAdapters(t *testing.T) {
	r := NewRegistry()
	a := testAdapter{id: Identity{ID: "go", Version: "1.0.0"}, matches: true}
	if err := r.Register(a); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(a); err == nil {
		t.Fatal("duplicate registration unexpectedly succeeded")
	}
	if err := r.Register(testAdapter{id: Identity{ID: "go", Version: "2.0.0"}, matches: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Resolve(Identity{ID: "go"}, Match{}); err == nil {
		t.Fatal("ambiguous resolution unexpectedly succeeded")
	}
}

func TestRegistryRejectsUnsupportedVersionAndMatch(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(testAdapter{id: Identity{ID: "node", Version: "1.0.0"}, matches: false}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Resolve(Identity{ID: "node", Version: "2.0.0"}, Match{}); err == nil {
		t.Fatal("unknown version unexpectedly resolved")
	}
	if _, err := r.Resolve(Identity{ID: "node", Version: "1.0.0"}, Match{}); err == nil {
		t.Fatal("unsupported match unexpectedly resolved")
	}
}

func TestRegistryCoversNilValidationExplicitResolutionAndIdentityOrdering(t *testing.T) {
	if err := (*Registry)(nil).Register(testAdapter{}); err == nil {
		t.Fatal("nil registry accepted adapter")
	}
	r := NewRegistry()
	for _, adapter := range []testAdapter{{id: Identity{ID: "", Version: "1"}, matches: true}, {id: Identity{ID: "x", Version: ""}, matches: true}} {
		if err := r.Register(adapter); err == nil {
			t.Fatal("invalid identity accepted")
		}
	}
	if _, err := r.Resolve(Identity{ID: "missing"}, Match{}); err == nil {
		t.Fatal("missing adapter resolved")
	}
	if err := r.Register(testAdapter{id: Identity{ID: "go", Version: "1"}, matches: true}); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(testAdapter{id: Identity{ID: "go", Version: "2"}, matches: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Resolve(Identity{ID: "go", Version: "1"}, Match{}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Resolve(Identity{ID: "go", Version: "9"}, Match{}); err == nil {
		t.Fatal("unknown explicit version resolved")
	}
	if _, err := r.Resolve(Identity{ID: "go"}, Match{}); err == nil {
		t.Fatal("ambiguous/multiple resolution unexpectedly succeeded")
	}
	if ids := r.Identities(); len(ids) != 2 || ids[0].ID != "go" || ids[0].Version != "1" {
		t.Fatalf("identities = %+v", ids)
	}
	if (*Registry)(nil).Identities() != nil {
		t.Fatal("nil registry identities not nil")
	}
}
