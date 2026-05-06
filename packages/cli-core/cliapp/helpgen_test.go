package cliapp

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderHelpNoArgs(t *testing.T) {
	var buf bytes.Buffer
	cmd := Command{Name: "list", Description: "List things"}
	if err := renderHelp("demo", cmd, &buf); err != nil {
		t.Fatalf("renderHelp: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"demo list - List things",
		"Usage:\n  demo list",
		"--json",
		"--help, -h",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("help missing %q\n--- output ---\n%s", want, out)
		}
	}
}

func TestRenderHelpFlagsAndPositionals(t *testing.T) {
	var buf bytes.Buffer
	cmd := Command{
		Name:        "create",
		Description: "Create a note",
		Args: ArgSchema{
			Positionals: []Positional{{Name: "id", Required: true, Description: "Note id"}},
			Flags: []Flag{
				{Name: "title", Required: true, Description: "Note title"},
				{Name: "verbose", Aliases: []string{"v"}, Bool: true, Description: "Verbose output"},
				{Name: "color", Default: "auto", Description: "Color mode"},
			},
		},
	}
	if err := renderHelp("demo notes", cmd, &buf); err != nil {
		t.Fatalf("renderHelp: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"demo notes create - Create a note",
		"<id>",
		"--title <value>",
		"(required)",
		"--verbose, -v",
		"--color <value>",
		"(default: auto)",
		"--json",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("help missing %q\n--- output ---\n%s", want, out)
		}
	}
}

func TestUsageLine(t *testing.T) {
	tests := []struct {
		name   string
		schema ArgSchema
		want   string
	}{
		{"empty", ArgSchema{}, "demo cmd"},
		{"required pos", ArgSchema{Positionals: []Positional{{Name: "id", Required: true}}}, "demo cmd <id>"},
		{"optional pos", ArgSchema{Positionals: []Positional{{Name: "id"}}}, "demo cmd [id]"},
		{"flags only", ArgSchema{Flags: []Flag{{Name: "title"}}}, "demo cmd [options]"},
		{"mixed", ArgSchema{
			Positionals: []Positional{{Name: "id", Required: true}},
			Flags:       []Flag{{Name: "title"}},
		}, "demo cmd <id> [options]"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := usageLine("demo cmd", tc.schema); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
