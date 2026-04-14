package scenariohandlers

import (
	"strings"
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
	got := FormatTemplateRequiredFlags(scenariocli.TemplateManifest{
		RequiredVars: map[string]scenariocli.TemplateVar{
			"SCENARIO_ID":           {Flag: "id"},
			"SCENARIO_DISPLAY_NAME": {Flag: "display-name"},
		},
	})
	if got == "" {
		t.Fatal("expected formatted required flags")
	}
	for _, want := range []string{"--id <scenario_id>", "--display-name <scenario_display_name>"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in summary %q", want, got)
		}
	}
}

func TestLooksLikeTextFileRejectsBinaryData(t *testing.T) {
	if LooksLikeTextFile([]byte{0}) {
		t.Fatal("LooksLikeTextFile() should reject binary content")
	}
}
