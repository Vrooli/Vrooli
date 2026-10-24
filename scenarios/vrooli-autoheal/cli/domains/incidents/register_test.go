package incidents

import "testing"

func TestRegisterProvidesIncidentRemediationCommands(t *testing.T) {
	group := Register(nil)
	want := []string{"list", "latest", "show", "remediations", "remediation", "acknowledge", "resolve", "ignore"}
	if group.Name != "incidents" {
		t.Fatalf("Register().Name = %q, want incidents", group.Name)
	}
	if len(group.Subcommands) != len(want) {
		t.Fatalf("Register() subcommand count = %d, want %d", len(group.Subcommands), len(want))
	}
	for i, name := range want {
		if group.Subcommands[i].Name != name {
			t.Fatalf("Register() subcommand[%d] = %q, want %q", i, group.Subcommands[i].Name, name)
		}
	}
}
