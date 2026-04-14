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
