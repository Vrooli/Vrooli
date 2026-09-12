package identity

import "testing"

func TestIntersectScopesIsOneWayAndDeduplicated(t *testing.T) {
	got := IntersectScopes(
		[]string{"vrooli-bridge:read", "vrooli-bridge:write"},
		[]string{"vrooli-bridge:*"},
		[]string{"vrooli-bridge:read", "vrooli-bridge:read", "vrooli-bridge:destructive"},
	)
	if len(got) != 1 || got[0] != "vrooli-bridge:read" {
		t.Fatalf("intersection = %#v", got)
	}
}

func TestIntersectScopesExplicitEmptyRequestGrantsNothing(t *testing.T) {
	got := IntersectScopes([]string{"vrooli-bridge:read"}, []string{"vrooli-bridge:read"}, []string{})
	if len(got) != 0 {
		t.Fatalf("explicit empty request = %#v", got)
	}
}
