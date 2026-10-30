package measures

import (
	"strings"
	"testing"
)

func TestRegisterDeclaresGenericVerbsNotPerMeasureVerbs(t *testing.T) {
	names := map[string]bool{}
	for _, command := range Register().Subcommands {
		names[command.Name] = true
	}
	for _, want := range []string{"list", "get", "select-cohort", "cohort-report"} {
		if !names[want] {
			t.Errorf("measures must expose %q", want)
		}
	}
	for _, gone := range []string{"episode-cohort", "episode-compare", "run-volume", "retry-rate", "token-attribution", "runner-breakdown", "workload-efficiency"} {
		if names[gone] {
			t.Errorf("per-measure verb %q must come from `measures get`, not a hand-written subcommand", gone)
		}
	}
}

func TestGetCommandHelpDeclaresTheWindowAndParamFlags(t *testing.T) {
	var flags []string
	for _, command := range Register().Subcommands {
		if command.Name == "get" {
			if err := command.Args.Validate(); err != nil {
				t.Fatalf("get args invalid: %v", err)
			}
			for _, flag := range command.Args.Flags {
				flags = append(flags, flag.Name)
			}
		}
	}
	if strings.Join(flags, ",") != "window,param" {
		t.Fatalf("get flags = %v, want window,param", flags)
	}
}

func TestResolveDeclarationAcceptsFullIDOrBareName(t *testing.T) {
	declarations := []declaration{{Name: "throughput.run_volume"}, {Name: "friction.token_attribution"}, {Name: "usage.friction"}, {Name: "friction.tool_usage"}, {Name: "a.dup"}, {Name: "b.dup"}}
	for input, want := range map[string]string{
		"throughput.run_volume": "throughput.run_volume",
		"run-volume":            "throughput.run_volume",
		"token-attribution":     "friction.token_attribution",
		"friction":              "usage.friction",
	} {
		if got, err := resolveDeclaration(input, declarations); err != nil || got.Name != want {
			t.Errorf("resolveDeclaration(%q) = %q, %v; want %q", input, got.Name, err, want)
		}
	}
	if _, err := resolveDeclaration("dup", declarations); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("a bare name shared by two domains must be ambiguous, got %v", err)
	}
	if _, err := resolveDeclaration("nope", declarations); err == nil || !strings.Contains(err.Error(), "measures list") {
		t.Errorf("an unknown id must point at `measures list`, got %v", err)
	}
}

func TestBuildParamsFillsDeclaredDefaultsAndKeepsGivenValues(t *testing.T) {
	var chosen declaration
	chosen.Params = map[string]struct {
		Default string `json:"default"`
	}{"window": {Default: "this_week"}, "group_by": {Default: "capability"}, "view": {}}
	got, err := buildParams(chosen, "", []string{"group_by=executable"})
	if err != nil || got["window"] != "this_week" || got["group_by"] != "executable" {
		t.Fatalf("params = %v, %v; want the window default and the given group_by", got, err)
	}
	if _, present := got["view"]; present {
		t.Errorf("a parameter without a default must not be invented: %v", got)
	}
	if got, _ = buildParams(chosen, "last_7d", nil); got["window"] != "last_7d" {
		t.Errorf("--window must beat the default: %v", got)
	}
	if _, err := buildParams(chosen, "", []string{"novalue"}); err == nil || !strings.Contains(err.Error(), "name=value") {
		t.Errorf("a malformed --param must be refused, got %v", err)
	}
}
