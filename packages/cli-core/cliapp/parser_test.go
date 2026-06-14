package cliapp

import (
	"errors"
	"strings"
	"testing"
)

func TestParseArgsNoArgs(t *testing.T) {
	ctx, err := parseArgs(ArgSchema{}, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("parseArgs: %v", err)
	}
	if len(ctx.Args()) != 0 {
		t.Errorf("expected empty args, got %v", ctx.Args())
	}
}

func TestParseArgsSimpleFlags(t *testing.T) {
	schema := ArgSchema{Flags: []Flag{
		{Name: "title", Required: true},
		{Name: "body"},
		{Name: "verbose", Bool: true},
	}}

	tests := []struct {
		name     string
		args     []string
		title    string
		body     string
		verbose  bool
		bodySet  bool
		titleSet bool
	}{
		{"separated values", []string{"--title", "hi", "--body", "bye"}, "hi", "bye", false, true, true},
		{"equals form", []string{"--title=hello", "--body=world"}, "hello", "world", false, true, true},
		{"bool flag", []string{"--title", "x", "--verbose"}, "x", "", true, false, true},
		{"mixed equals+separated", []string{"--title=x", "--body", "y"}, "x", "y", false, true, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, err := parseArgs(schema, tc.args, nil, nil, nil)
			if err != nil {
				t.Fatalf("parseArgs: %v", err)
			}
			if got := ctx.Flag("title"); got != tc.title {
				t.Errorf("title: got %q, want %q", got, tc.title)
			}
			if got := ctx.Flag("body"); got != tc.body {
				t.Errorf("body: got %q, want %q", got, tc.body)
			}
			if got := ctx.BoolFlag("verbose"); got != tc.verbose {
				t.Errorf("verbose: got %v, want %v", got, tc.verbose)
			}
		})
	}
}

func TestParseArgsRequiredFlagMissing(t *testing.T) {
	schema := ArgSchema{Flags: []Flag{{Name: "title", Required: true}}}
	_, err := parseArgs(schema, nil, nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "missing required flag --title") {
		t.Fatalf("expected missing-required error, got: %v", err)
	}
}

func TestParseArgsUnknownOption(t *testing.T) {
	_, err := parseArgs(ArgSchema{}, []string{"--bogus"}, nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "unknown option") {
		t.Fatalf("expected unknown-option error, got: %v", err)
	}
}

func TestParseArgsMissingValue(t *testing.T) {
	schema := ArgSchema{Flags: []Flag{{Name: "title"}}}
	_, err := parseArgs(schema, []string{"--title"}, nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "missing value for --title") {
		t.Fatalf("expected missing-value error, got: %v", err)
	}
}

func TestParseArgsBoolWithValueRejected(t *testing.T) {
	schema := ArgSchema{Flags: []Flag{{Name: "verbose", Bool: true}}}
	_, err := parseArgs(schema, []string{"--verbose=true"}, nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "does not accept a value") {
		t.Fatalf("expected bool-with-value error, got: %v", err)
	}
}

func TestParseArgsHelp(t *testing.T) {
	for _, tok := range []string{"--help", "-h"} {
		t.Run(tok, func(t *testing.T) {
			_, err := parseArgs(ArgSchema{}, []string{tok}, nil, nil, nil)
			if !errors.Is(err, ErrHelpRequested) {
				t.Fatalf("expected ErrHelpRequested, got: %v", err)
			}
		})
	}
}

func TestParseArgsJSONFlag(t *testing.T) {
	ctx, err := parseArgs(ArgSchema{}, []string{"--json"}, nil, nil, nil)
	if err != nil {
		t.Fatalf("parseArgs: %v", err)
	}
	if !ctx.JSON() {
		t.Error("expected JSON()=true after --json")
	}
}

func TestParseArgsAlias(t *testing.T) {
	schema := ArgSchema{Flags: []Flag{{Name: "verbose", Aliases: []string{"v"}, Bool: true}}}
	ctx, err := parseArgs(schema, []string{"-v"}, nil, nil, nil)
	if err != nil {
		t.Fatalf("parseArgs: %v", err)
	}
	if !ctx.BoolFlag("verbose") {
		t.Error("expected verbose to be set via -v alias")
	}
}

func TestParseArgsDefault(t *testing.T) {
	schema := ArgSchema{Flags: []Flag{{Name: "color", Default: "auto"}}}
	ctx, err := parseArgs(schema, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("parseArgs: %v", err)
	}
	if got := ctx.Flag("color"); got != "auto" {
		t.Errorf("default: got %q, want %q", got, "auto")
	}
	if ctx.BoolFlag("color") {
		t.Error("default-only flag should not be marked set")
	}
	if got := ctx.FlagValues("color"); strings.Join(got, ",") != "auto" {
		t.Errorf("default values: got %v, want [auto]", got)
	}
}

func TestParseArgsRepeatedValuedFlag(t *testing.T) {
	schema := ArgSchema{Flags: []Flag{{Name: "endpoint"}}}
	ctx, err := parseArgs(schema, []string{"--endpoint", "health", "--endpoint=notes_attach"}, nil, nil, nil)
	if err != nil {
		t.Fatalf("parseArgs: %v", err)
	}
	if got := ctx.Flag("endpoint"); got != "notes_attach" {
		t.Errorf("Flag returns last value: got %q, want notes_attach", got)
	}
	if got := ctx.FlagValues("endpoint"); strings.Join(got, ",") != "health,notes_attach" {
		t.Errorf("FlagValues: got %v, want [health notes_attach]", got)
	}
}

func TestParseArgsRepeatedValuedFlagOverridesDefault(t *testing.T) {
	schema := ArgSchema{Flags: []Flag{{Name: "include", Default: "all"}}}
	ctx, err := parseArgs(schema, []string{"--include", "imports", "--include", "calls"}, nil, nil, nil)
	if err != nil {
		t.Fatalf("parseArgs: %v", err)
	}
	if got := ctx.FlagValues("include"); strings.Join(got, ",") != "imports,calls" {
		t.Errorf("FlagValues: got %v, want [imports calls]", got)
	}
}

func TestParseArgsRequiredPositional(t *testing.T) {
	schema := ArgSchema{Positionals: []Positional{{Name: "id", Required: true}}}

	ctx, err := parseArgs(schema, []string{"abc"}, nil, nil, nil)
	if err != nil {
		t.Fatalf("parseArgs: %v", err)
	}
	if got := ctx.Positional("id"); got != "abc" {
		t.Errorf("id: got %q, want %q", got, "abc")
	}

	_, err = parseArgs(schema, nil, nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "missing required positional <id>") {
		t.Fatalf("expected missing-required-positional, got: %v", err)
	}
}

func TestParseArgsOptionalPositional(t *testing.T) {
	schema := ArgSchema{Positionals: []Positional{{Name: "filter"}}}

	ctx, err := parseArgs(schema, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("parseArgs: %v", err)
	}
	if got := ctx.Positional("filter"); got != "" {
		t.Errorf("optional missing: got %q, want empty", got)
	}

	ctx, err = parseArgs(schema, []string{"todo"}, nil, nil, nil)
	if err != nil {
		t.Fatalf("parseArgs: %v", err)
	}
	if got := ctx.Positional("filter"); got != "todo" {
		t.Errorf("optional present: got %q, want %q", got, "todo")
	}
}

func TestParseArgsRepeatedPositional(t *testing.T) {
	schema := ArgSchema{Positionals: []Positional{{Name: "ids", Required: true, Repeated: true}}}

	ctx, err := parseArgs(schema, []string{"a", "b", "c"}, nil, nil, nil)
	if err != nil {
		t.Fatalf("parseArgs: %v", err)
	}
	if got := ctx.Positionals("ids"); strings.Join(got, ",") != "a,b,c" {
		t.Errorf("repeated: got %v, want [a b c]", got)
	}

	_, err = parseArgs(schema, nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error: required repeated positional missing")
	}
}

func TestParseArgsMixedPositionals(t *testing.T) {
	schema := ArgSchema{Positionals: []Positional{
		{Name: "id", Required: true},
		{Name: "fields", Repeated: true},
	}}
	ctx, err := parseArgs(schema, []string{"42", "title", "body"}, nil, nil, nil)
	if err != nil {
		t.Fatalf("parseArgs: %v", err)
	}
	if got := ctx.Positional("id"); got != "42" {
		t.Errorf("id: got %q, want 42", got)
	}
	if got := ctx.Positionals("fields"); strings.Join(got, ",") != "title,body" {
		t.Errorf("fields: got %v", got)
	}
}

func TestParseArgsExtraPositionalsRejected(t *testing.T) {
	schema := ArgSchema{Positionals: []Positional{{Name: "id"}}}
	_, err := parseArgs(schema, []string{"a", "b"}, nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "unexpected positional") {
		t.Fatalf("expected unexpected-positional error, got: %v", err)
	}
}

func TestParseArgsDoubleDashEndsFlags(t *testing.T) {
	schema := ArgSchema{Positionals: []Positional{{Name: "args", Repeated: true}}}
	ctx, err := parseArgs(schema, []string{"--", "--not-a-flag", "-x"}, nil, nil, nil)
	if err != nil {
		t.Fatalf("parseArgs: %v", err)
	}
	got := ctx.Positionals("args")
	if strings.Join(got, "|") != "--not-a-flag|-x" {
		t.Errorf("after --: got %v", got)
	}
}
