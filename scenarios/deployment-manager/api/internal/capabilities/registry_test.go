package capabilities

import (
	"testing"
)

func TestKnownCapabilitiesDoNotDuplicateManifestDependencies(t *testing.T) {
	if len(Known) != 0 {
		t.Fatalf("manifest dependencies must not be repeated in Known: %#v", Known)
	}
}
