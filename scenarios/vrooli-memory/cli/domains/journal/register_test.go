package journal

import "testing"

func TestGroupNameMatchesManifestDomain(t *testing.T) {
	if GroupName != "journal" {
		t.Fatalf("GroupName = %q, want journal", GroupName)
	}
}
