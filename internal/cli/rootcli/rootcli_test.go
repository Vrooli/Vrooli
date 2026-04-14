package rootcli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/cli/topcli"
)

func TestParseArgsSeparatesGlobalsAndCommand(t *testing.T) {
	parsed, err := ParseArgs([]string{"--json", "--verbose", "scenario", "list"})
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if parsed.Command != "scenario" {
		t.Fatalf("command = %q, want scenario", parsed.Command)
	}
	if !parsed.Globals.JSON || !parsed.Globals.Verbose {
		t.Fatalf("globals = %#v, want json and verbose", parsed.Globals)
	}
	if got := strings.Join(parsed.Args, "|"); got != "list" {
		t.Fatalf("args = %q, want list", got)
	}
}

func TestConsumeInlineGlobalFlagsPreservesPassthroughTail(t *testing.T) {
	globals, args := ConsumeInlineGlobalFlags(GlobalOptions{}, []string{"list", "--json", "--", "--json"})
	if !globals.JSON {
		t.Fatalf("globals = %#v, want JSON enabled", globals)
	}
	if got := strings.Join(args, "|"); got != "list|--|--json" {
		t.Fatalf("args = %q", got)
	}
}

func TestRegistryCanRunWithoutRoot(t *testing.T) {
	registry := NewRegistry(
		map[topcli.CommandID]Handler[struct{}]{
			topcli.CommandSetup:    func(struct{}, []string) error { return nil },
			topcli.CommandScenario: func(struct{}, []string) error { return nil },
		},
		map[scenariocli.CommandID]Handler[struct{}]{
			scenariocli.CommandList: func(struct{}, []string) error { return nil },
		},
	)
	if !registry.CanRunWithoutRoot(ParsedArgs{Command: "help"}) {
		t.Fatal("help should run without root")
	}
	if registry.CanRunWithoutRoot(ParsedArgs{Command: "setup", Args: []string{"--dry-run"}}) {
		t.Fatal("setup should require root resolution")
	}
	if !registry.CanRunWithoutRoot(ParsedArgs{Command: "scenario", Args: []string{"--help"}}) {
		t.Fatal("scenario help should run without root")
	}
}

func TestPrintErrorWithContextUnknownCommandIncludesSuggestions(t *testing.T) {
	err := NewUnknownCommandError("statsu", []string{"status"})
	var stderr bytes.Buffer
	PrintErrorWithContext(&stderr, err)

	output := stderr.String()
	if !strings.Contains(output, clipolicy.UnknownCommandLabel+": statsu") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "status") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, clipolicy.MainHelpHint) {
		t.Fatalf("output = %q", output)
	}
}

func TestPrintErrorWithContextUnknownScenarioCommandIncludesSuggestions(t *testing.T) {
	err := NewUnknownScenarioCommandError("statsu", []string{"status"})
	var stderr bytes.Buffer
	PrintErrorWithContext(&stderr, err)

	output := stderr.String()
	if !strings.Contains(output, clipolicy.UnknownScenarioCommandLabel+": statsu") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "status") {
		t.Fatalf("output = %q", output)
	}
}

func TestPrintErrorWithContextCategorizedRuntimeError(t *testing.T) {
	err := NewErrorWithCategory(
		errors.New("pipeline failed"),
		ErrorCategoryRuntime,
		"Check the command inputs and try again.",
		[]string{"setup", "status"},
	)

	var stderr bytes.Buffer
	PrintErrorWithContext(&stderr, err)

	output := stderr.String()
	if !strings.Contains(output, "Runtime error: pipeline failed") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "Check the command inputs and try again.") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "Did you mean one of these?") {
		t.Fatalf("output = %q", output)
	}
}

func TestPrintErrorWithContextPreservesPlainErrors(t *testing.T) {
	var stderr bytes.Buffer
	PrintErrorWithContext(&stderr, errors.New("plain failure"))

	if got := strings.TrimSpace(stderr.String()); got != "plain failure" {
		t.Fatalf("output = %q", got)
	}
}

func TestExitCodePrefersTypedExitCodes(t *testing.T) {
	if code := ExitCode(nil); code != 0 {
		t.Fatalf("ExitCode(nil) = %d", code)
	}
	if code := ExitCode(ExitCodeError{Code: 23}); code != 23 {
		t.Fatalf("ExitCode(typed) = %d", code)
	}
	if code := ExitCode(errors.New("boom")); code != 1 {
		t.Fatalf("ExitCode(default) = %d", code)
	}
}

func TestExitCodeErrorFormatting(t *testing.T) {
	if got := (ExitCodeError{Code: 7, Message: "boom"}).Error(); got != "boom" {
		t.Fatalf("ExitCodeError message = %q", got)
	}
	if got := (ExitCodeError{Code: 7}).Error(); got != "exit code 7" {
		t.Fatalf("ExitCodeError default = %q", got)
	}
}

func TestNewErrorWithCategoryPreservesExitCode(t *testing.T) {
	err := NewErrorWithCategory(ExitCodeError{Code: 23, Message: "wrapped"}, ErrorCategoryRuntime, "", nil)
	if got := ExitCode(err); got != 23 {
		t.Fatalf("ExitCode(err) = %d, want 23", got)
	}
}

func TestPassthroughFlagsOmitsExistingFlags(t *testing.T) {
	flags := PassthroughFlags(GlobalOptions{JSON: true, Verbose: true, NoColor: true}, []string{"--json", "scenario"})
	if got, want := strings.Join(flags, ","), "--verbose,--no-color"; got != want {
		t.Fatalf("flags = %q, want %q", got, want)
	}
}

func TestContainsArgDoesNotMatchAbsentFlag(t *testing.T) {
	if ContainsArg([]string{"alpha", "beta"}, "--json") {
		t.Fatal("ContainsArg() should not match absent flag")
	}
}
