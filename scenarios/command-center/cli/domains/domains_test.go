package domains

import (
	"os"
	"testing"
)

func TestCommandGroupsRegisterInstrumentAndIntegrationVerbs(t *testing.T) {
	got := CommandGroups(nil)
	if len(got) != 1 || len(got[0].Commands) != 9 {
		t.Fatalf("unexpected flat command registration: %#v", got)
	}
}

func TestWalkReadIsRegistered(t *testing.T) {
	manifest, err := os.ReadFile("../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := SubcommandGroups(nil, manifest); len(got) != 1 {
		t.Fatalf("missing walk group: %#v", got)
	}
}
