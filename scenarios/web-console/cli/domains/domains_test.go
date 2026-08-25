package domains

import (
	"os"
	"testing"
)

func TestSubcommandGroupsLoadManifest(t *testing.T) {
	manifest, err := os.ReadFile("../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	groups, err := SubcommandGroups(nil, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 13 {
		t.Fatalf("got %d command groups, want 13", len(groups))
	}
	if CommandGroups(nil) != nil {
		t.Fatal("CommandGroups should be empty after manifest migration")
	}
}
