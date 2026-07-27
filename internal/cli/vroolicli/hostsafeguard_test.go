package vroolicli

import "testing"

func TestHostSafeguardSpecDeclaresSudoAndDryRun(t *testing.T) {
	spec := hostSafeguardSpec()
	if spec.Name != "safeguard" || spec.Handler != "safeguard" {
		t.Fatalf("unexpected spec: %#v", spec)
	}
	if len(spec.Args.Positionals) != 1 || spec.Args.Positionals[0].Name != "name" {
		t.Fatalf("safeguard must require a name: %#v", spec.Args)
	}
}
