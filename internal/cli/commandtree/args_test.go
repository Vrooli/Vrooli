package commandtree

import (
	"strings"
	"testing"
)

func TestParseArgsSupportsBooleanAndValuedOptions(t *testing.T) {
	parsed, err := ParseArgs("demo", "Usage: demo", ArgSchema{
		Positionals: []PositionalArg{
			{Name: "name", Required: true},
			{Name: "extra", Repeatable: true},
		},
		Options: []OptionArg{
			{Name: "--json"},
			{Name: "--env", Aliases: []string{"-e"}, ValueName: "name"},
		},
	}, []string{"alpha", "--json", "-e", "prod", "beta"})
	if err != nil {
		t.Fatalf("ParseArgs: %v", err)
	}
	if got := strings.Join(parsed.Positionals, ","); got != "alpha,beta" {
		t.Fatalf("positionals = %q", got)
	}
	if !parsed.HasFlag("--json") {
		t.Fatal("expected --json flag")
	}
	if got := parsed.FlagValue("--env"); got != "prod" {
		t.Fatalf("env = %q", got)
	}
}

func TestParseArgsRejectsUnknownOption(t *testing.T) {
	_, err := ParseArgs("demo", "Usage: demo", ArgSchema{}, []string{"--bogus"})
	if err == nil || !strings.Contains(err.Error(), "unknown option for demo") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseArgsRejectsMissingRequiredPositional(t *testing.T) {
	_, err := ParseArgs("demo", "Usage: demo", ArgSchema{
		Positionals: []PositionalArg{{Name: "scenario name", Required: true}},
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "requires exactly one scenario name") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseArgsRejectsExtraOptionalPositional(t *testing.T) {
	_, err := ParseArgs("demo", "Usage: demo", ArgSchema{
		Positionals: []PositionalArg{{Name: "scenario name"}},
	}, []string{"alpha", "beta"})
	if err == nil || !strings.Contains(err.Error(), "accepts at most one scenario name") {
		t.Fatalf("err = %v", err)
	}
}
