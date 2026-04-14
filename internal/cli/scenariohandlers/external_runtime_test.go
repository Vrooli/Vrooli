package scenariohandlers

import (
	"testing"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cli/scenariocli"
)

func TestBuildUISmokeArgsAppendsPassthroughFlags(t *testing.T) {
	args := BuildUISmokeArgs(rootcli.GlobalOptions{JSON: true}, []string{"alpha"})
	if len(args) != 3 {
		t.Fatalf("args = %#v", args)
	}
	if args[0] != "ui-smoke" || args[1] != "alpha" || args[2] != "--json" {
		t.Fatalf("args = %#v", args)
	}
}

func TestBuildScenarioCompletenessArgsAddsJSONWhenFormatMissing(t *testing.T) {
	args := BuildScenarioCompletenessArgs(rootcli.GlobalOptions{JSON: true}, []string{"alpha"})
	if len(args) != 2 || args[0] != "alpha" || args[1] != "--json" {
		t.Fatalf("args = %#v", args)
	}
}

func TestBuildScenarioCompletenessArgsPreservesExplicitFormat(t *testing.T) {
	args := BuildScenarioCompletenessArgs(rootcli.GlobalOptions{JSON: true}, []string{"alpha", "--format", "json"})
	if len(args) != 3 || args[2] != "json" {
		t.Fatalf("args = %#v", args)
	}
	for _, arg := range args {
		if arg == "--json" {
			t.Fatalf("args = %#v", args)
		}
	}
}

func TestFormatTemplateRequiredFlagsReturnsNonEmptySummary(t *testing.T) {
	if FormatTemplateRequiredFlags(scenariocli.TemplateManifest{
		RequiredVars: map[string]scenariocli.TemplateVar{
			"SCENARIO_ID": {Flag: "id"},
		},
	}) == "" {
		t.Fatal("expected formatted required flags")
	}
}
