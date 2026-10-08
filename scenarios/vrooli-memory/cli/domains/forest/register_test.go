package forest

import "testing"

func TestGroupNameMatchesManifestDomain(t *testing.T) {
	if GroupName != "forest" {
		t.Fatalf("GroupName = %q, want forest", GroupName)
	}
}
