package scenariocli

import (
	"testing"

	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
)

// scenario logs must accept --instance like its sibling lifecycle commands.
// Before this was wired, the flag was rejected as an unknown option and the
// parser's own remediation ("place it BEFORE the command") was rejected too,
// leaving no working spelling for reading a variant's logs.
func TestParseLogsArgsResolvesInstanceFlag(t *testing.T) {
	name, opts, err := ParseLogsArgs([]string{"web-console", "--instance", "presentation", "--tail", "20"})
	if err != nil {
		t.Fatalf("parse with --instance: %v", err)
	}
	if name != "web-console@presentation" {
		t.Errorf("name = %q, want %q", name, "web-console@presentation")
	}
	if opts.Tail != 20 {
		t.Errorf("tail = %d, want 20", opts.Tail)
	}

	// The addressed form stays equivalent to the flag form.
	addressed, _, err := ParseLogsArgs([]string{"web-console@presentation"})
	if err != nil {
		t.Fatalf("parse addressed form: %v", err)
	}
	if addressed != name {
		t.Errorf("addressed form = %q, flag form = %q; they must agree", addressed, name)
	}

	// A live instance keeps the bare slug.
	live, _, err := ParseLogsArgs([]string{"web-console"})
	if err != nil {
		t.Fatalf("parse live: %v", err)
	}
	if live != "web-console" {
		t.Errorf("live name = %q, want %q", live, "web-console")
	}
}

// Logs are read from this host's log directory, so a node address has no
// reader. It must be refused by name rather than reaching the path builder,
// which would report it as an invalid selector.
func TestParseLogsArgsRefusesNodeAddress(t *testing.T) {
	_, _, err := ParseLogsArgs([]string{"minimouse/web-console"})
	if err == nil {
		t.Fatal("expected a node address to be refused")
	}
}

// A scenario's brand declaration must survive the mapping onto the proto info
// message. It is the one hop that decides whether `scenario info --json`
// answers the "which brand does this scenario ship?" question in one call
// instead of sending a reader to parse .vrooli/service.json.
func TestScenarioInfoDataCarriesBranding(t *testing.T) {
	msg := scenarioInfoData(InfoScenarioData{
		Name:     "web-console",
		Branding: &scenariomodel.Branding{Brand: "aquila", Targets: []string{"web-public-v1", "electron-v1"}},
	})
	if msg.GetBranding() == nil {
		t.Fatal("branding was dropped on the way to the proto message")
	}
	if msg.GetBranding().GetBrand() != "aquila" {
		t.Errorf("brand = %q, want %q", msg.GetBranding().GetBrand(), "aquila")
	}
	if len(msg.GetBranding().GetTargets()) != 2 {
		t.Errorf("targets = %v, want 2 entries", msg.GetBranding().GetTargets())
	}

	// An unbranded scenario must stay nil rather than gaining an empty brand.
	if plain := scenarioInfoData(InfoScenarioData{Name: "plain"}); plain.GetBranding() != nil {
		t.Errorf("unbranded scenario gained a branding message: %+v", plain.GetBranding())
	}
}
