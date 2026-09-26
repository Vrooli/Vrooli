package rootcli

import (
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/cli/topcli"
	"github.com/vrooli/vrooli/internal/vroolierr"
)

// [REQ:CLI-ARGV-001] --no-color consumes exactly one token.
func TestParseArgsNoColorConsumesOneToken(t *testing.T) {
	parsed, err := ParseArgs([]string{"--no-color", "scenario", "list"})
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if parsed.Command != "scenario" {
		t.Fatalf("command = %q, want scenario", parsed.Command)
	}
	if got := strings.Join(parsed.Args, "|"); got != "list" {
		t.Fatalf("args = %q, want list", got)
	}
	if !parsed.Globals.NoColor {
		t.Fatalf("globals = %#v, want NoColor", parsed.Globals)
	}
}

// [REQ:CLI-ARGV-001] --no-color alone must not panic and must fall through to help.
func TestParseArgsNoColorAloneShowsHelp(t *testing.T) {
	parsed, err := ParseArgs([]string{"--no-color"})
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if parsed.Command != "help" {
		t.Fatalf("command = %q, want help", parsed.Command)
	}
}

// [REQ:CLI-ARGV-002] A retired global is consumed with one warning, never a usage error.
func TestParseArgsRetiredGlobalIsToleratedWithWarning(t *testing.T) {
	parsed, err := ParseArgs([]string{"--no-stale-check", "version"})
	if err != nil {
		t.Fatalf("ParseArgs returned error for a retired global: %v", err)
	}
	if parsed.Command != "version" {
		t.Fatalf("command = %q, want version", parsed.Command)
	}
	if len(parsed.Warnings) != 1 {
		t.Fatalf("warnings = %v, want exactly one", parsed.Warnings)
	}
	if !strings.Contains(parsed.Warnings[0], "--no-stale-check") || !strings.Contains(parsed.Warnings[0], "retired") {
		t.Fatalf("warning = %q, want it to name the flag and say retired", parsed.Warnings[0])
	}
	globals, args, warnings := ConsumeInlineGlobalFlags(GlobalOptions{}, []string{"list", "--no-stale-check", "--json"})
	if !globals.JSON || len(args) != 1 || args[0] != "list" || len(warnings) != 1 {
		t.Fatalf("inline consume = globals %+v args %v warnings %v", globals, args, warnings)
	}
}

// [REQ:CLI-ARGV-002] The tolerance table must keep --no-stale-check until its removal date.
// Guard against the 2026-09-02 09:26 deletion: an empty table turns every
// pre-retirement supervisor and scenario CLI into a boot-time no-op.
func TestRetiredGlobalsTableCarriesNoStaleCheckUntilRemoval(t *testing.T) {
	entry, ok := retiredGlobals["--no-stale-check"]
	if !ok {
		t.Fatalf("retiredGlobals lacks --no-stale-check; see the comment on the table before removing it")
	}
	removeAfter, err := time.Parse("2006-01-02", entry.RemoveAfter)
	if err != nil {
		t.Fatalf("RemoveAfter %q is not a date: %v", entry.RemoveAfter, err)
	}
	if time.Now().Before(removeAfter) {
		return
	}
	t.Logf("--no-stale-check passed its removal date %s; the entry may now be deleted", entry.RemoveAfter)
}

// [REQ:CLI-ARGV-002] Tolerance is a closed table: an unknown leading flag stays a hard error.
func TestResolveArgvRejectsUnknownCommand(t *testing.T) {
	_, err := ResolveArgv([]string{"--bogus-flag", "version"})
	if err == nil {
		t.Fatalf("ResolveArgv accepted an unknown leading flag")
	}
	if code := vroolierr.Code(err, ""); code != "unknown_command" {
		t.Fatalf("error code = %q, want unknown_command (err=%v)", code, err)
	}
	if _, err := ResolveArgv([]string{"scenario", "no-such-verb"}); err == nil {
		t.Fatalf("ResolveArgv accepted an unknown scenario subcommand")
	}
}

// [REQ:CLI-ARGV-003] Every registered command path resolves without executing.
func TestResolveArgvAcceptsEveryRegisteredCommand(t *testing.T) {
	for _, spec := range topcli.CommandSpecs() {
		names := append([]string{spec.Name}, spec.Aliases...)
		for _, name := range names {
			res, err := ResolveArgv([]string{name})
			if err != nil {
				t.Fatalf("ResolveArgv(%q) error = %v", name, err)
			}
			if res.Command != spec.Name {
				t.Fatalf("ResolveArgv(%q).Command = %q, want %q", name, res.Command, spec.Name)
			}
		}
	}
	for _, spec := range scenariocli.CommandSpecs() {
		res, err := ResolveArgv([]string{"scenario", spec.Name, "--json"})
		if err != nil {
			t.Fatalf("ResolveArgv(scenario %q) error = %v", spec.Name, err)
		}
		if res.Subcommand != spec.Name {
			t.Fatalf("ResolveArgv(scenario %q).Subcommand = %q", spec.Name, res.Subcommand)
		}
		if !res.Globals.JSON {
			t.Fatalf("ResolveArgv(scenario %q) lost the inline --json global", spec.Name)
		}
	}
	for _, builtin := range []string{"help", "version", "--help", "-h", "--version", "-v"} {
		if _, err := ResolveArgv([]string{builtin}); err != nil {
			t.Fatalf("ResolveArgv(%q) error = %v", builtin, err)
		}
	}
	if res, err := ResolveArgv(nil); err != nil || res.Command != "help" {
		t.Fatalf("ResolveArgv(nil) = %#v, %v; want help", res, err)
	}
}
