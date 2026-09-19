package scenariohandlers

import (
	"testing"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
)

const (
	jsonFlag    = "--json"
	verboseFlag = "--verbose"
)

func TestBuildScenarioCompletenessArgsAddsJSONWhenFormatMissing(t *testing.T) {
	args := BuildScenarioCompletenessArgs(rootcli.GlobalOptions{JSON: true}, []string{"alpha"})
	if len(args) != 2 || args[0] != "alpha" || args[1] != jsonFlag {
		t.Fatalf("args = %#v", args)
	}
}

func TestBuildScenarioCompletenessArgsPreservesExplicitFormat(t *testing.T) {
	args := BuildScenarioCompletenessArgs(rootcli.GlobalOptions{JSON: true}, []string{"alpha", "--format", "json"})
	if len(args) != 3 || args[2] != "json" {
		t.Fatalf("args = %#v", args)
	}
	for _, arg := range args {
		if arg == jsonFlag {
			t.Fatalf("args = %#v", args)
		}
	}
}

func TestBuildScenarioCompletenessArgsPreservesFormatEquals(t *testing.T) {
	args := BuildScenarioCompletenessArgs(rootcli.GlobalOptions{JSON: true}, []string{"alpha", "--format=yaml"})
	for _, arg := range args {
		if arg == jsonFlag {
			t.Fatalf("--format=yaml should suppress auto-appended --json; args=%#v", args)
		}
	}
}

func TestBuildScenarioCompletenessArgsPrependsNoColor(t *testing.T) {
	args := BuildScenarioCompletenessArgs(rootcli.GlobalOptions{NoColor: true}, []string{"score", "get", "alpha"})
	if len(args) == 0 || args[0] != "--no-color" {
		t.Fatalf("expected --no-color first, got %#v", args)
	}
	// Existing flags must not be duplicated.
	again := BuildScenarioCompletenessArgs(rootcli.GlobalOptions{NoColor: true}, []string{"--no-color", "score", "get", "alpha"})
	count := 0
	for _, arg := range again {
		if arg == "--no-color" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected single --no-color, got %#v", again)
	}
}

func TestBuildScenarioCompletenessArgsAppendsVerbose(t *testing.T) {
	args := BuildScenarioCompletenessArgs(rootcli.GlobalOptions{Verbose: true}, []string{"score", "get", "alpha"})
	if len(args) == 0 || args[len(args)-1] != verboseFlag {
		t.Fatalf("expected --verbose last, got %#v", args)
	}
	// Don't add twice if user already passed --verbose.
	again := BuildScenarioCompletenessArgs(rootcli.GlobalOptions{Verbose: true}, []string{"score", "get", "alpha", verboseFlag})
	count := 0
	for _, arg := range again {
		if arg == verboseFlag {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected single --verbose, got %#v", again)
	}
	// Also respect the short form -v.
	short := BuildScenarioCompletenessArgs(rootcli.GlobalOptions{Verbose: true}, []string{"score", "get", "alpha", "-v"})
	for _, arg := range short {
		if arg == verboseFlag {
			t.Fatalf("should not add --verbose when -v already present: %#v", short)
		}
	}
}
