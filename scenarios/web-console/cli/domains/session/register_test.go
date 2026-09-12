package session

import (
	"os"
	"testing"
)

func TestRegisterLoadsManifest(t *testing.T) {
	manifest, err := os.ReadFile("../../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	group, err := Register(nil, manifest)
	if err != nil || len(group.Subcommands) == 0 {
		t.Fatalf("Register() = %#v, %v", group, err)
	}
}
