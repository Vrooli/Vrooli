package main

import (
	"testing"
)

func TestBuildUISmokeArgsAppendsPassthroughFlags(t *testing.T) {
	args := buildUISmokeArgs(globalOptions{json: true}, []string{"alpha"})
	if len(args) != 3 {
		t.Fatalf("args = %#v", args)
	}
	if args[0] != "ui-smoke" || args[1] != "alpha" || args[2] != "--json" {
		t.Fatalf("args = %#v", args)
	}
}

func TestBuildScenarioCompletenessArgsAddsJSONWhenFormatMissing(t *testing.T) {
	args := buildScenarioCompletenessArgs(globalOptions{json: true}, []string{"alpha"})
	if len(args) != 2 || args[0] != "alpha" || args[1] != "--json" {
		t.Fatalf("args = %#v", args)
	}
}

func TestBuildScenarioCompletenessArgsPreservesExplicitFormat(t *testing.T) {
	args := buildScenarioCompletenessArgs(globalOptions{json: true}, []string{"alpha", "--format", "json"})
	if len(args) != 3 || args[2] != "json" {
		t.Fatalf("args = %#v", args)
	}
	for _, arg := range args {
		if arg == "--json" {
			t.Fatalf("args should not append --json when format already supplied: %#v", args)
		}
	}
}
