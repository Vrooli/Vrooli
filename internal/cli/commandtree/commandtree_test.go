package commandtree

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

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
