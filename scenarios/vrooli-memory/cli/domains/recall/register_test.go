package recall

import "testing"

func TestGroupNameMatchesManifestDomain(t *testing.T) {
	if GroupName != "recall" {
		t.Fatalf("GroupName = %q, want recall", GroupName)
	}
}
