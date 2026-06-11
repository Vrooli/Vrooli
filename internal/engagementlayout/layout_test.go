package engagementlayout

import "testing"

// allLayouts is every supported arrangement. New layouts must be added here so
// the property-based checks below cover them automatically.
var allLayouts = []Layout{StandardLayout, InvertedLayout}

func TestLayoutsValidate(t *testing.T) {
	for _, l := range allLayouts {
		if err := l.Validate(); err != nil {
			t.Errorf("layout %q invalid: %v", l.Name, err)
		}
	}
}

func TestStandardLayoutDerivations(t *testing.T) {
	l := StandardLayout

	if got := l.RoleForVariant(Live); got != Baseline {
		t.Errorf("live serves %q, want baseline", got)
	}
	if got := l.RoleForVariant(Shadow); got != Candidate {
		t.Errorf("shadow serves %q, want candidate", got)
	}

	// No engagement: every variant runs from the working tree (no copy exists).
	if got := l.LocationForVariant(Live, false); got != WorkingTree {
		t.Errorf("live unengaged at %q, want working-tree", got)
	}
	if got := l.LocationForVariant(Shadow, false); got != WorkingTree {
		t.Errorf("shadow unengaged at %q, want working-tree", got)
	}

	// Engaged: live (baseline) → copy; shadow (candidate) → working tree.
	if got := l.LocationForVariant(Live, true); got != RestorePointCopy {
		t.Errorf("live engaged at %q, want restore-point-copy", got)
	}
	if got := l.LocationForVariant(Shadow, true); got != WorkingTree {
		t.Errorf("shadow engaged at %q, want working-tree", got)
	}

	if got := l.EditedLocation(); got != WorkingTree {
		t.Errorf("standard EditedLocation %q, want working-tree", got)
	}
}

func TestInvertedLayoutFlipsLocationsOnly(t *testing.T) {
	l := InvertedLayout

	// Roles per variant are unchanged by the flip — only locations move.
	if got := l.RoleForVariant(Live); got != Baseline {
		t.Errorf("inverted live serves %q, want baseline", got)
	}
	if got := l.RoleForVariant(Shadow); got != Candidate {
		t.Errorf("inverted shadow serves %q, want candidate", got)
	}

	// Engaged: baseline now in the working tree, candidate in the copy.
	if got := l.LocationForVariant(Live, true); got != WorkingTree {
		t.Errorf("inverted live engaged at %q, want working-tree", got)
	}
	if got := l.LocationForVariant(Shadow, true); got != RestorePointCopy {
		t.Errorf("inverted shadow engaged at %q, want restore-point-copy", got)
	}

	if got := l.EditedLocation(); got != RestorePointCopy {
		t.Errorf("inverted EditedLocation %q, want restore-point-copy", got)
	}
}

// TestServingInstanceIsolated is the property-based safety invariant. It must
// hold for EVERY supported layout: the instance serving the Baseline role never
// runs from the location receiving the agent's merge. A directionality flip that
// would violate this fails here at one place rather than corrupting live.
func TestServingInstanceIsolated(t *testing.T) {
	for _, l := range allLayouts {
		if !l.ServingInstanceIsolated() {
			t.Errorf("layout %q violates isolation: baseline runs from EditedLocation %q",
				l.Name, l.EditedLocation())
		}
		// Stated the long way too, so the property is legible at the assertion site.
		if l.LocationForRole(Baseline) == l.EditedLocation() {
			t.Errorf("layout %q: baseline location %q == edited location %q",
				l.Name, l.LocationForRole(Baseline), l.EditedLocation())
		}
	}
}

func TestDefaultIsStandard(t *testing.T) {
	if Default().Name != StandardLayout.Name {
		t.Errorf("Default() = %q, want %q", Default().Name, StandardLayout.Name)
	}
}

func TestValidateRejectsCollapsedLayout(t *testing.T) {
	bad := Layout{
		Name:         "collapsed",
		variantRole:  map[Variant]Role{Live: Baseline, Shadow: Candidate},
		roleLocation: map[Role]Location{Baseline: WorkingTree, Candidate: WorkingTree},
	}
	if err := bad.Validate(); err == nil {
		t.Fatal("expected validate error for both roles sharing a location")
	}
}
