package cliapp

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestRunContextTypedAccessors(t *testing.T) {
	schema := ArgSchema{
		Positionals: []Positional{{Name: "id", Required: true}},
		Flags: []Flag{
			{Name: "title"},
			{Name: "verbose", Bool: true},
		},
	}
	ctx := NewTestRunContext(TestRunContextOptions{
		Schema:      schema,
		Flags:       map[string]string{"title": "hello"},
		BoolFlags:   map[string]bool{"verbose": true},
		Positionals: map[string]string{"id": "42"},
	})

	if got := ctx.Flag("title"); got != "hello" {
		t.Errorf("Flag(title): %q", got)
	}
	if !ctx.BoolFlag("verbose") {
		t.Error("BoolFlag(verbose): false")
	}
	if got := ctx.Positional("id"); got != "42" {
		t.Errorf("Positional(id): %q", got)
	}
}

func TestRunContextUndeclaredFlagPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on undeclared flag")
		}
	}()
	ctx := NewTestRunContext(TestRunContextOptions{Schema: ArgSchema{}})
	_ = ctx.Flag("nonexistent")
}

func TestRunContextRenderListHuman(t *testing.T) {
	var buf bytes.Buffer
	ctx := NewTestRunContext(TestRunContextOptions{
		Stdout: &buf,
	})
	if err := ctx.RenderList(ListReport{Summary: []string{"hello"}}); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	if !strings.Contains(buf.String(), "hello") {
		t.Errorf("expected human output, got: %s", buf.String())
	}
	if strings.HasPrefix(strings.TrimSpace(buf.String()), "{") {
		t.Errorf("expected human output, got JSON: %s", buf.String())
	}
}

func TestRunContextRenderListJSON(t *testing.T) {
	var buf bytes.Buffer
	ctx := NewTestRunContext(TestRunContextOptions{
		JSON:   true,
		Stdout: &buf,
	})
	if err := ctx.RenderList(ListReport{Summary: []string{"hello"}}); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	if !strings.HasPrefix(out, "{") || !strings.Contains(out, `"summary"`) {
		t.Errorf("expected JSON output, got: %s", out)
	}
}

func TestRunContextRenderMutationRoutesByJSON(t *testing.T) {
	var human, jsonOut bytes.Buffer

	ctxHuman := NewTestRunContext(TestRunContextOptions{Stdout: &human})
	if err := ctxHuman.RenderMutation(MutationReport{Result: []string{"done"}}); err != nil {
		t.Fatal(err)
	}
	ctxJSON := NewTestRunContext(TestRunContextOptions{JSON: true, Stdout: &jsonOut})
	if err := ctxJSON.RenderMutation(MutationReport{Result: []string{"done"}}); err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(strings.TrimSpace(human.String()), "{") {
		t.Errorf("human output is JSON: %s", human.String())
	}
	if !strings.HasPrefix(strings.TrimSpace(jsonOut.String()), "{") {
		t.Errorf("json output not JSON: %s", jsonOut.String())
	}
}

func TestRunContextArgsCopy(t *testing.T) {
	ctx := NewTestRunContext(TestRunContextOptions{
		RawArgs: []string{"a", "b"},
	})
	got := ctx.Args()
	got[0] = "MUTATED"
	again := ctx.Args()
	if again[0] != "a" {
		t.Errorf("Args() must return a copy; got %v", again)
	}
}

func TestNewTestRunContextFromArgs_PopulatesValues(t *testing.T) {
	schema := ArgSchema{
		Positionals: []Positional{{Name: "id", Required: true}},
		Flags: []Flag{
			{Name: "title", Required: true},
			{Name: "verbose", Bool: true},
		},
	}
	ctx, err := NewTestRunContextFromArgs(
		schema,
		[]string{"--title", "hello", "--verbose", "42"},
		nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := ctx.Flag("title"); got != "hello" {
		t.Errorf("Flag(title): %q", got)
	}
	if !ctx.BoolFlag("verbose") {
		t.Error("BoolFlag(verbose): false")
	}
	if got := ctx.Positional("id"); got != "42" {
		t.Errorf("Positional(id): %q", got)
	}
}

func TestNewTestRunContextFromArgs_EnforcesRequiredFlag(t *testing.T) {
	schema := ArgSchema{Flags: []Flag{{Name: "title", Required: true}}}
	_, err := NewTestRunContextFromArgs(schema, []string{}, nil, nil, nil)
	if err == nil {
		t.Fatal("expected required-flag error, got nil")
	}
	if !strings.Contains(err.Error(), "missing required flag --title") {
		t.Errorf("error message: %v", err)
	}
}

func TestNewTestRunContextFromArgs_EnforcesRequiredPositional(t *testing.T) {
	schema := ArgSchema{Positionals: []Positional{{Name: "id", Required: true}}}
	_, err := NewTestRunContextFromArgs(schema, []string{}, nil, nil, nil)
	if err == nil {
		t.Fatal("expected required-positional error, got nil")
	}
	if !strings.Contains(err.Error(), "missing required positional <id>") {
		t.Errorf("error message: %v", err)
	}
}

func TestNewTestRunContextFromArgs_HelpReturnsSentinel(t *testing.T) {
	schema := ArgSchema{Flags: []Flag{{Name: "title", Required: true}}}
	_, err := NewTestRunContextFromArgs(schema, []string{"--help"}, nil, nil, nil)
	if !errors.Is(err, ErrHelpRequested) {
		t.Errorf("expected ErrHelpRequested, got %v", err)
	}
}

func TestRunContextRepeatedRequiresRepeatedSchema(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on Positionals() over non-repeated")
		}
	}()
	ctx := NewTestRunContext(TestRunContextOptions{
		Schema: ArgSchema{Positionals: []Positional{{Name: "id"}}},
	})
	_ = ctx.Positionals("id")
}
