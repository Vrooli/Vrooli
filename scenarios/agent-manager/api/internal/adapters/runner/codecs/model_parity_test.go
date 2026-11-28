package codecs

import (
	"testing"

	"agent-manager/internal/rolepolicy"
)

// TestRoleCatalogCodecConformance ensures role-policy candidates name actual
// codecs without making codec capabilities a second static model catalog.
func TestRoleCatalogCodecConformance(t *testing.T) {
	revision, err := rolepolicy.Load(rolepolicy.ResolvePath())
	if err != nil {
		t.Fatalf("load role policy catalog: %v", err)
	}
	catalog := revision.Catalog()
	if catalog == nil {
		t.Fatal("role policy catalog is nil")
	}

	codecTypes := make(map[string]struct{})
	for _, codec := range []Codec{NewClaudeForTest(), NewCodexForTest(), NewOpenCodeForTest(), NewGrokForTest(), NewAntigravityForTest()} {
		if !codec.Type().IsValid() {
			t.Errorf("codec has invalid runner type %q", codec.Type())
		}
		codecTypes[string(codec.Type())] = struct{}{}
	}
	for roleRef, role := range catalog.Roles {
		if len(role.Candidates) == 0 {
			t.Errorf("role %q has no candidates", roleRef)
		}
		for _, candidate := range role.Candidates {
			if _, ok := codecTypes[string(candidate.Runner)]; !ok {
				t.Errorf("role %q names runner %q with no codec", roleRef, candidate.Runner)
			}
			if candidate.ResourceRole == "" {
				t.Errorf("role %q candidate %q has no resource role", roleRef, candidate.Runner)
			}
		}
	}
}
