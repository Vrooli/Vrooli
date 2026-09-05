package measures

import (
	"os"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestRegisterLoadsManifestMeasuresWithVerifiedProtoListEvidence(t *testing.T) {
	manifest, err := os.ReadFile("../../manifest.json")
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	group, err := Register(nil, manifest)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if group.Name != "measures" || len(group.Subcommands) != 26 {
		t.Fatalf("group = %+v, want twenty-six measures subcommands", group)
	}
	for _, command := range group.Subcommands {
		if got := command.PrimitiveEvidence(); got != cliapp.PrimitiveProtoList {
			t.Errorf("%s evidence = %q, want %q", command.Name, got, cliapp.PrimitiveProtoList)
		}
	}
}
