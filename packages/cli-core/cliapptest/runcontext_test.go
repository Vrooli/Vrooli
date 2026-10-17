package cliapptest

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestNewTestRunContextMatchesCliapp(t *testing.T) {
	schema := cliapp.ArgSchema{
		Positionals: []cliapp.Positional{
			{Name: "id"},
			{Name: "rest", Repeated: true},
		},
		Flags: []cliapp.Flag{
			{Name: "title", Default: "untitled"},
			{Name: "verbose", Bool: true},
		},
	}
	opts := TestRunContextOptions{
		Schema:      schema,
		Flags:       map[string]string{"title": "hello"},
		BoolFlags:   map[string]bool{"verbose": true},
		Positionals: map[string]string{"id": "42"},
		Repeated:    map[string][]string{"rest": {"a", "b"}},
		RawArgs:     []string{"--title", "hello", "42", "a", "b"},
		JSON:        true,
	}

	got := NewTestRunContext(opts)
	want := cliapp.NewTestRunContext(opts)

	if got.Flag("title") != want.Flag("title") {
		t.Fatalf("Flag(title) = %q, want %q", got.Flag("title"), want.Flag("title"))
	}
	if got.BoolFlag("verbose") != want.BoolFlag("verbose") {
		t.Fatalf("BoolFlag(verbose) = %v, want %v", got.BoolFlag("verbose"), want.BoolFlag("verbose"))
	}
	if got.Positional("id") != want.Positional("id") {
		t.Fatalf("Positional(id) = %q, want %q", got.Positional("id"), want.Positional("id"))
	}
	if strings.Join(got.Positionals("rest"), ",") != strings.Join(want.Positionals("rest"), ",") {
		t.Fatalf("Positionals(rest) = %v, want %v", got.Positionals("rest"), want.Positionals("rest"))
	}
	if strings.Join(got.Args(), ",") != strings.Join(want.Args(), ",") {
		t.Fatalf("Args = %v, want %v", got.Args(), want.Args())
	}
	if got.JSON() != want.JSON() {
		t.Fatalf("JSON = %v, want %v", got.JSON(), want.JSON())
	}
}

func TestNewTestRunContextFromArgsMatchesCliapp(t *testing.T) {
	schema := cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "id", Required: true}},
		Flags: []cliapp.Flag{
			{Name: "title", Required: true},
			{Name: "verbose", Bool: true},
		},
	}
	args := []string{"--json", "--title", "hello", "--verbose", "42"}

	got, err := NewTestRunContextFromArgs(schema, args, nil, nil, nil)
	if err != nil {
		t.Fatalf("cliapptest parse: %v", err)
	}
	want, err := cliapp.NewTestRunContextFromArgs(schema, args, nil, nil, nil)
	if err != nil {
		t.Fatalf("cliapp parse: %v", err)
	}

	if got.Flag("title") != want.Flag("title") {
		t.Fatalf("Flag(title) = %q, want %q", got.Flag("title"), want.Flag("title"))
	}
	if got.Positional("id") != want.Positional("id") {
		t.Fatalf("Positional(id) = %q, want %q", got.Positional("id"), want.Positional("id"))
	}
	if got.JSON() != want.JSON() {
		t.Fatalf("JSON = %v, want %v", got.JSON(), want.JSON())
	}
}

func TestNewTestRunContextFromArgsForwardsErrors(t *testing.T) {
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "title", Required: true}}}

	_, err := NewTestRunContextFromArgs(schema, []string{}, nil, nil, nil)
	if err == nil {
		t.Fatal("expected required flag error")
	}
	if !strings.Contains(err.Error(), "missing required flag --title") {
		t.Fatalf("error = %v", err)
	}

	_, err = NewTestRunContextFromArgs(schema, []string{"--help"}, nil, nil, nil)
	if !errors.Is(err, cliapp.ErrHelpRequested) {
		t.Fatalf("help error = %v, want ErrHelpRequested", err)
	}
}

func TestNewTestRunContextForwardsWriters(t *testing.T) {
	var gotOut, wantOut bytes.Buffer

	got := NewTestRunContext(TestRunContextOptions{JSON: true, Stdout: &gotOut})
	want := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{JSON: true, Stdout: &wantOut})

	report := cliapp.ListReport{Summary: []string{"ok"}}
	if err := got.RenderList(report); err != nil {
		t.Fatalf("cliapptest RenderList: %v", err)
	}
	if err := want.RenderList(report); err != nil {
		t.Fatalf("cliapp RenderList: %v", err)
	}
	if gotOut.String() != wantOut.String() {
		t.Fatalf("stdout = %q, want %q", gotOut.String(), wantOut.String())
	}
}
