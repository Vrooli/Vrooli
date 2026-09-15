package commandtree

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestValidateSpecsRejectsDuplicateNameAndAlias(t *testing.T) {
	err := ValidateSpecs([]Spec[string]{
		{Name: "status", Aliases: []string{"st"}},
		{Name: "stop", Aliases: []string{"status"}},
	})
	if err == nil || !strings.Contains(err.Error(), `duplicate command name or alias "status"`) {
		t.Fatalf("ValidateSpecs error = %v", err)
	}
}

func TestBindSpecsPreservesCommandMetadata(t *testing.T) {
	source := []Spec[string]{
		{
			Name:        "status",
			Aliases:     []string{"st"},
			Group:       "Lifecycle",
			Summary:     "Show status",
			Hidden:      true,
			Suggestable: true,
			RootPolicy:  RootPolicy{RequiresRoot: true},
			Help:        Help{Title: "Status"},
			Args: ArgSchema{
				Positionals: []PositionalArg{{Name: "name", Required: true, Description: "Target name"}},
				Options:     []OptionArg{{Name: "--json", Description: "Emit JSON"}},
			},
			Handler: "status-id",
		},
	}

	bound := BindSpecs(source, map[string]int{"status-id": 7})
	if len(bound) != 1 {
		t.Fatalf("len(bound) = %d, want 1", len(bound))
	}
	spec := bound[0]
	if spec.Name != "status" || spec.Handler != 7 {
		t.Fatalf("spec = %#v", spec)
	}
	if got := strings.Join(spec.Aliases, ","); got != "st" {
		t.Fatalf("aliases = %q", got)
	}
	if spec.Group != "Lifecycle" || spec.Summary != "Show status" || !spec.Hidden || !spec.Suggestable {
		t.Fatalf("spec metadata = %#v", spec)
	}
	if !spec.RootPolicy.RequiresRoot || spec.Help.Title != "Status" {
		t.Fatalf("spec contracts = %#v", spec)
	}
	if len(spec.Args.Positionals) != 1 || spec.Args.Positionals[0].Name != "name" {
		t.Fatalf("spec args = %#v", spec.Args)
	}
	if len(spec.Args.Options) != 1 || spec.Args.Options[0].Name != "--json" {
		t.Fatalf("spec options = %#v", spec.Args)
	}
}

func TestBuildHandlerMapIncludesAliases(t *testing.T) {
	specs := []Spec[string]{
		{Name: "status", Aliases: []string{"st"}, Handler: "handler"},
	}

	items := BuildHandlerMap(specs)
	if items["status"] != "handler" {
		t.Fatalf("primary handler missing")
	}
	if items["st"] != "handler" {
		t.Fatalf("alias handler missing")
	}
}

func TestBuildHandlerMapPanicsOnDuplicateAliases(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for duplicate aliases")
		}
	}()

	_ = BuildHandlerMap([]Spec[string]{
		{Name: "status", Aliases: []string{"st"}, Handler: "status"},
		{Name: "stop", Aliases: []string{"st"}, Handler: "stop"},
	})
}

func TestSuggestableNamesSorted(t *testing.T) {
	specs := []Spec[string]{
		{Name: "beta", Suggestable: true},
		{Name: "alpha", Suggestable: true},
		{Name: "hidden", Suggestable: false},
	}

	names := SuggestableNames(specs)
	if got := strings.Join(names, ","); got != "alpha,beta" {
		t.Fatalf("names = %q", got)
	}
}

func TestRenderHelpUsesVisibleEntriesAndDefaultGroup(t *testing.T) {
	specs := []Spec[string]{
		{Name: "status", Summary: "Show status"},
		{Name: "hidden", Summary: "Ignore me", Hidden: true},
	}

	var output bytes.Buffer
	RenderHelp(&output, Help{
		Title:        "Commands",
		Usage:        "vrooli test <subcommand>",
		DefaultGroup: "General",
	}, specs)

	text := output.String()
	if !strings.Contains(text, "Commands") {
		t.Fatalf("missing title: %q", text)
	}
	if !strings.Contains(text, "vrooli test <subcommand>") {
		t.Fatalf("missing usage: %q", text)
	}
	if !strings.Contains(text, "General:") {
		t.Fatalf("missing default group: %q", text)
	}
	if !strings.Contains(text, "status") {
		t.Fatalf("missing visible command: %q", text)
	}
	if strings.Contains(text, "hidden") {
		t.Fatalf("rendered hidden command: %q", text)
	}
}

func TestUsageLineBuildsFromSchema(t *testing.T) {
	got := UsageLine("vrooli demo", ArgSchema{
		Positionals: []PositionalArg{
			{Name: "name", Required: true},
			{Name: "extra", Repeatable: true},
		},
		Options: []OptionArg{{Name: "--json"}},
	})
	if got != "vrooli demo <name> [extra...] [options]" {
		t.Fatalf("usage = %q", got)
	}
}

func TestSpecHelpTextIncludesDescriptionAndOptions(t *testing.T) {
	text := SpecHelpText("", "vrooli demo status", Spec[string]{
		Name:    "status",
		Summary: "Show status",
		Help: Help{
			Description: "Display status details.",
			Examples:    []string{"vrooli demo status alpha"},
		},
		Args: ArgSchema{
			Positionals: []PositionalArg{{Name: "name", Required: true}},
			Options: []OptionArg{
				{Name: "--json", Description: "Emit JSON output"},
				{Name: "--env", Aliases: []string{"-e"}, ValueName: "name", Description: "Select environment"},
			},
		},
	})
	for _, want := range []string{
		"Usage:\n  vrooli demo status <name> [options]",
		"Display status details.",
		"--json",
		"--env, -e <name>",
		"Show help for this command",
		"Examples:\n  vrooli demo status alpha",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in text:\n%s", want, text)
		}
	}
}

func TestRenderHelpIncludesOptionsAndNotes(t *testing.T) {
	var output bytes.Buffer
	RenderHelp(&output, Help{
		Title:        "Demo",
		Usage:        "vrooli demo <subcommand> [options]",
		DefaultGroup: "General",
		Options: []OptionArg{
			{Name: "--verbose", Description: "Enable verbose output"},
		},
		Notes: []string{"Documentation: docs/"},
	}, []Spec[string]{{Name: "status", Summary: "Show status"}})

	text := output.String()
	for _, want := range []string{
		"Options:",
		"--verbose",
		"Enable verbose output",
		"Documentation: docs/",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in text:\n%s", want, text)
		}
	}
}

type fakeHelpError struct {
	text string
}

func (e fakeHelpError) Error() string    { return e.text }
func (e fakeHelpError) HelpText() string { return e.text }

func TestExecuteActionPrintsHelpText(t *testing.T) {
	var output bytes.Buffer
	err := ExecuteAction(&output, nil, Action[struct{}, struct{}]{
		Parse: func(args []string) (struct{}, error) {
			return struct{}{}, fakeHelpError{text: "Usage: vrooli test"}
		},
		Execute: func(req struct{}) (struct{}, error) {
			t.Fatalf("execute should not run")
			return struct{}{}, nil
		},
		Render: func(w io.Writer, resp struct{}) error {
			return nil
		},
	})
	if err != nil {
		t.Fatalf("ExecuteAction: %v", err)
	}
	if got := output.String(); got != "Usage: vrooli test\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestExecuteActionRunsParseExecuteRender(t *testing.T) {
	var output bytes.Buffer
	err := ExecuteAction(&output, []string{"a"}, Action[string, string]{
		Parse: func(args []string) (string, error) {
			return args[0], nil
		},
		Execute: func(req string) (string, error) {
			return strings.ToUpper(req), nil
		},
		Render: func(w io.Writer, resp string) error {
			_, err := io.WriteString(w, resp)
			return err
		},
	})
	if err != nil {
		t.Fatalf("ExecuteAction: %v", err)
	}
	if got := output.String(); got != "A" {
		t.Fatalf("output = %q", got)
	}
}

func TestExecuteActionReturnsParseError(t *testing.T) {
	want := errors.New("boom")
	err := ExecuteAction(&bytes.Buffer{}, nil, Action[struct{}, struct{}]{
		Parse: func(args []string) (struct{}, error) {
			return struct{}{}, want
		},
		Execute: func(req struct{}) (struct{}, error) {
			return struct{}{}, nil
		},
		Render: func(w io.Writer, resp struct{}) error {
			return nil
		},
	})
	if !errors.Is(err, want) {
		t.Fatalf("err = %v", err)
	}
}
