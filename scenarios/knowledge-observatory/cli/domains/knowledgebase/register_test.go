package knowledgebase

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

// [REQ:KO-KB-001]
func TestManifestLoadsOnlyGovernedReadOperations(t *testing.T) {
	raw, err := os.ReadFile("../../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := cliapp.ParseManifest(raw)
	if err != nil {
		t.Fatal(err)
	}
	group := manifest.FindGroup("knowledge-base")
	if group == nil || len(group.Commands) != 5 {
		t.Fatal("missing governed operations")
	}
	var data struct {
		Groups []struct {
			Commands []struct {
				Governance struct {
					Effect      string `json:"effect"`
					RunEligible bool   `json:"run_eligible"`
				} `json:"governance"`
			} `json:"commands"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	for _, g := range data.Groups {
		for _, c := range g.Commands {
			if c.Governance.Effect != "read" || !c.Governance.RunEligible {
				t.Fatal("unexpected mutation or unbound operation")
			}
		}
	}
	for _, command := range group.Commands {
		if command.Binding.Kind != "connect-rpc" {
			t.Fatal("ungoverned command", command.Name)
		}
	}
}
