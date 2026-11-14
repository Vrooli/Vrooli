package cliapp

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestParseGlobalFlags(t *testing.T) {
	api := ""
	global := GlobalOptions{ColorEnabled: true}
	args := []string{"--api-base", "http://example.com", "--no-color", "run"}

	remaining, err := ParseGlobalFlags(args, &global, &api)
	if err != nil {
		t.Fatalf("ParseGlobalFlags: %v", err)
	}
	if api != "http://example.com" || global.APIBaseOverride != "http://example.com" {
		t.Fatalf("expected api override to propagate, got %q", api)
	}
	if global.ColorEnabled {
		t.Fatalf("expected color disabled")
	}
	if len(remaining) != 1 || remaining[0] != "run" {
		t.Fatalf("unexpected remaining args: %v", remaining)
	}
}

func TestAppRunWithWritersRoutesDeclarativeOutput(t *testing.T) {
	app := NewApp(AppOptions{
		Name: "demo",
		Commands: []CommandGroup{{
			Title: "Demo",
			Commands: []Command{{
				Name: "list",
				Args: ArgSchema{},
				RunCtx: func(ctx RunContext) error {
					return RenderListReport(ctx.Stdout(), ListReport{Summary: []string{"captured"}})
				},
			}},
		}},
	})
	var stdout, stderr bytes.Buffer
	if err := app.RunWithWriters([]string{"list"}, &stdout, &stderr); err != nil {
		t.Fatalf("RunWithWriters: %v", err)
	}
	if !strings.Contains(stdout.String(), "captured") {
		t.Fatalf("expected declarative output in supplied stdout, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestAppRunPassesNestedHelpToLegacyCommandWhenDeclared(t *testing.T) {
	var got []string
	app := NewApp(AppOptions{
		Name: "demo",
		Commands: []CommandGroup{{
			Title: "Demo",
			Commands: []Command{{
				Name:          "team",
				NeedsAPI:      true,
				PassHelpToRun: true,
				Run: func(args []string) error {
					got = append([]string(nil), args...)
					return nil
				},
			}},
		}},
	})
	var stdout, stderr bytes.Buffer
	if err := app.RunWithWriters([]string{"team", "bug-capture", "scenario-qa", "--help"}, &stdout, &stderr); err != nil {
		t.Fatalf("RunWithWriters: %v", err)
	}
	if gotText, wantText := strings.Join(got, "|"), "bug-capture|scenario-qa|--help"; gotText != wantText {
		t.Fatalf("Run args = %v, want %s", got, wantText)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("unexpected help output from dispatcher: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestAppRunPassesNestedHelpWithoutAPIOrPreflight(t *testing.T) {
	var got []string
	preflightCalled := false
	app := NewApp(AppOptions{
		Name: "demo",
		Commands: []CommandGroup{{
			Title: "Demo",
			Commands: []Command{{
				Name:          "team",
				NeedsAPI:      true,
				PassHelpToRun: true,
				Run: func(args []string) error {
					got = append([]string(nil), args...)
					return nil
				},
			}},
		}},
		Preflight: func(Command, GlobalOptions) error {
			preflightCalled = true
			return errors.New("API unavailable")
		},
	})
	if err := app.Run([]string{"team", "bug-capture", "scenario-qa", "--help"}); err != nil {
		t.Fatalf("nested help failed: %v", err)
	}
	if preflightCalled {
		t.Fatal("preflight must not run for nested help")
	}
	if gotText, wantText := strings.Join(got, "|"), "bug-capture|scenario-qa|--help"; gotText != wantText {
		t.Fatalf("Run args = %v, want %s", got, wantText)
	}
}

func TestParseGlobalFlagsMissingValue(t *testing.T) {
	global := GlobalOptions{}
	_, err := ParseGlobalFlags([]string{"--api-base"}, &global, nil)
	if err == nil {
		t.Fatalf("expected missing value error")
	}
}

func TestParseGlobalFlagsInstance(t *testing.T) {
	global := GlobalOptions{}
	remaining, err := ParseGlobalFlags([]string{"--instance", "shadow", "backlog", "list"}, &global, nil)
	if err != nil {
		t.Fatalf("ParseGlobalFlags: %v", err)
	}
	if global.Instance != "shadow" {
		t.Fatalf("expected instance=shadow, got %q", global.Instance)
	}
	if len(remaining) != 2 || remaining[0] != "backlog" || remaining[1] != "list" {
		t.Fatalf("unexpected remaining args: %v", remaining)
	}
}

func TestParseGlobalFlagsInstanceMissingValue(t *testing.T) {
	global := GlobalOptions{}
	if _, err := ParseGlobalFlags([]string{"--instance"}, &global, nil); err == nil {
		t.Fatalf("expected missing value error for --instance")
	}
}

func TestParseGlobalFlagsNode(t *testing.T) {
	global := GlobalOptions{}
	remaining, err := ParseGlobalFlags([]string{"--node", "minimouse", "run"}, &global, nil)
	if err != nil {
		t.Fatalf("ParseGlobalFlags: %v", err)
	}
	if global.Node != "minimouse" {
		t.Fatalf("expected node=minimouse, got %q", global.Node)
	}
	if len(remaining) != 1 || remaining[0] != "run" {
		t.Fatalf("unexpected remaining args: %v", remaining)
	}
}

func TestParseGlobalFlagsNodeMissingValue(t *testing.T) {
	global := GlobalOptions{}
	if _, err := ParseGlobalFlags([]string{"--node"}, &global, nil); err == nil {
		t.Fatalf("expected missing value error for --node")
	}
}

func TestParseGlobalFlagsStopsAtCommandBoundary(t *testing.T) {
	api := ""
	global := GlobalOptions{ColorEnabled: true}
	args := []string{"run", "--api-base", "http://command-local"}

	remaining, err := ParseGlobalFlags(args, &global, &api)
	if err != nil {
		t.Fatalf("ParseGlobalFlags: %v", err)
	}
	if api != "" || global.APIBaseOverride != "" {
		t.Fatalf("expected command-local --api-base to remain untouched, got %q", api)
	}
	if len(remaining) != 3 || remaining[0] != "run" || remaining[1] != "--api-base" || remaining[2] != "http://command-local" {
		t.Fatalf("unexpected remaining args: %v", remaining)
	}
}

func TestAppRoutesCommandsAndRunsStaleCheck(t *testing.T) {
	called := false
	stale := &cliutil.StaleChecker{
		BuildFingerprint: "fp",
		BuildSourceRoot:  t.TempDir(),
		FingerprintFunc: func(spec cliutil.FreshnessSpec) (string, error) {
			called = true
			return "fp", nil
		},
		LookPathFunc: func(file string) (string, error) {
			return "/usr/bin/go", nil
		},
	}

	group := CommandGroup{
		Title: "Demo",
		Commands: []Command{
			{Name: "run", Description: "Run demo", NeedsAPI: true, Run: func(args []string) error { return nil }},
		},
	}

	app := NewApp(AppOptions{
		Name:         "demo",
		Version:      "0.0.1",
		Commands:     []CommandGroup{group},
		ColorEnabled: DefaultColorEnabled(),
		OnColor:      func(enabled bool) {},
		StaleChecker: stale,
	})

	if err := app.Run([]string{"run"}); err != nil {
		t.Fatalf("app run: %v", err)
	}
	if !called {
		t.Fatalf("expected stale checker to run for NeedsAPI command")
	}
}

func TestAppRunWiresInstanceOverrideScopedToOwnScenario(t *testing.T) {
	cliutil.SetInstanceOverride("swarm-manager", "")
	cliutil.SetInstanceOverride("agent-manager", "")
	t.Cleanup(func() {
		cliutil.SetInstanceOverride("swarm-manager", "")
		cliutil.SetInstanceOverride("agent-manager", "")
	})

	group := CommandGroup{
		Title: "Demo",
		Commands: []Command{
			{Name: "run", Description: "Run demo", Run: func(args []string) error { return nil }},
		},
	}
	app := NewApp(AppOptions{
		Name:         "swarm-manager",
		Version:      "0.0.1",
		Commands:     []CommandGroup{group},
		ColorEnabled: DefaultColorEnabled(),
	})

	if err := app.Run([]string{"--instance", "shadow", "run"}); err != nil {
		t.Fatalf("app run: %v", err)
	}
	// The CLI's own scenario routes to shadow...
	if got := cliutil.ResolveShadowTarget("swarm-manager"); got != "swarm-manager@shadow" {
		t.Fatalf("own scenario target = %q, want swarm-manager@shadow", got)
	}
	// ...but an unrelated target is untouched by this CLI's --instance flag.
	if got := cliutil.ResolveShadowTarget("agent-manager"); got != "agent-manager" {
		t.Fatalf("unrelated target = %q, want agent-manager (bare)", got)
	}
}

func TestAppUnknownCommand(t *testing.T) {
	group := CommandGroup{
		Title: "Demo",
		Commands: []Command{
			{Name: "run", Description: "Run demo", Run: func(args []string) error { return nil }},
		},
	}
	app := NewApp(AppOptions{
		Name:         "demo",
		Version:      "0.0.1",
		Commands:     []CommandGroup{group},
		ColorEnabled: DefaultColorEnabled(),
	})

	err := app.Run([]string{"missing"})
	if err == nil {
		t.Fatalf("expected error for unknown command")
	}
	if !strings.Contains(err.Error(), "Unknown command") {
		t.Fatalf("expected unknown command error, got %v", err)
	}
}

func TestAppUnknownCommandChecksFreshnessBeforeReportingAbsence(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "packages", "cli-core"), 0o755); err != nil {
		t.Fatalf("mkdir cli-core fixture: %v", err)
	}
	sourceRoot := filepath.Join(repoRoot, "scenarios", "demo", "cli")
	if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
		t.Fatalf("mkdir source fixture: %v", err)
	}

	var rebuilt bool
	var reexecArgs []string
	stale := &cliutil.StaleChecker{
		AppName:          "demo",
		InstallKind:      "scenario",
		InstallName:      "demo",
		BuildFingerprint: "installed-old",
		BuildSourceRoot:  sourceRoot,
		FingerprintFunc: func(cliutil.FreshnessSpec) (string, error) {
			return "source-with-new-command", nil
		},
		LookPathFunc: func(string) (string, error) { return "/usr/bin/go", nil },
		CommandRunner: func(*exec.Cmd) error {
			rebuilt = true
			return nil
		},
		Reexec: func(_ string, args []string) error {
			reexecArgs = append([]string(nil), args...)
			return nil
		},
	}
	app := NewApp(AppOptions{
		Name:         "demo",
		Version:      "0.0.1",
		ColorEnabled: DefaultColorEnabled(),
		StaleChecker: stale,
	})

	if err := app.Run([]string{"new-command", "--json"}); err != nil {
		t.Fatalf("stale unknown command should rebuild instead of failing: %v", err)
	}
	if !rebuilt {
		t.Fatal("unknown command did not trigger the stale binary rebuild")
	}
	if got := strings.Join(reexecArgs, " "); got != "new-command --json" {
		t.Fatalf("reexec args = %q, want original command", got)
	}
}

func TestAppUnknownSubcommandChecksFreshnessBeforeReportingAbsence(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "packages", "cli-core"), 0o755); err != nil {
		t.Fatalf("mkdir cli-core fixture: %v", err)
	}
	sourceRoot := filepath.Join(repoRoot, "scenarios", "demo", "cli")
	if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
		t.Fatalf("mkdir source fixture: %v", err)
	}

	var reexecArgs []string
	stale := &cliutil.StaleChecker{
		AppName:          "demo",
		InstallKind:      "scenario",
		InstallName:      "demo",
		BuildFingerprint: "installed-old",
		BuildSourceRoot:  sourceRoot,
		FingerprintFunc: func(cliutil.FreshnessSpec) (string, error) {
			return "source-with-new-subcommand", nil
		},
		LookPathFunc:  func(string) (string, error) { return "/usr/bin/go", nil },
		CommandRunner: func(*exec.Cmd) error { return nil },
		Reexec: func(_ string, args []string) error {
			reexecArgs = append([]string(nil), args...)
			return nil
		},
	}
	app := NewApp(AppOptions{
		Name: "demo",
		SubcommandGroups: []SubcommandGroup{{
			Name: "events",
			Subcommands: []Command{{
				Name: "query",
				Run:  func([]string) error { return nil },
			}},
		}},
		StaleChecker: stale,
	})

	if err := app.Run([]string{"events", "aggregate"}); err != nil {
		t.Fatalf("stale unknown subcommand should rebuild instead of failing: %v", err)
	}
	if got := strings.Join(reexecArgs, " "); got != "events aggregate" {
		t.Fatalf("reexec args = %q, want original command", got)
	}
}

func TestAppKnownNonAPISubcommandChecksFreshnessBeforeDispatch(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "packages", "cli-core"), 0o755); err != nil {
		t.Fatalf("mkdir cli-core fixture: %v", err)
	}
	sourceRoot := filepath.Join(repoRoot, "resources", "demo", "cli")
	if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
		t.Fatalf("mkdir source fixture: %v", err)
	}
	var rebuilt bool
	var reexecArgs []string
	stale := &cliutil.StaleChecker{
		AppName:          "demo",
		InstallKind:      "resource",
		InstallName:      "demo",
		BuildFingerprint: "installed-old",
		BuildSourceRoot:  sourceRoot,
		FingerprintFunc: func(cliutil.FreshnessSpec) (string, error) {
			return "source-with-new-policy-command", nil
		},
		LookPathFunc:  func(string) (string, error) { return "/usr/bin/go", nil },
		CommandRunner: func(*exec.Cmd) error { rebuilt = true; return nil },
		Reexec: func(_ string, args []string) error {
			reexecArgs = append([]string(nil), args...)
			return nil
		},
	}
	dispatched := false
	app := NewApp(AppOptions{
		Name: "demo",
		SubcommandGroups: []SubcommandGroup{{
			Name: "policy",
			Subcommands: []Command{{
				Name: "resolve",
				Run:  func([]string) error { dispatched = true; return nil },
			}},
		}},
		StaleChecker: stale,
	})

	if err := app.Run([]string{"policy", "resolve", "--role", "code.delivery"}); err != nil {
		t.Fatalf("stale known subcommand should rebuild instead of dispatching: %v", err)
	}
	if !rebuilt {
		t.Fatal("known non-API subcommand did not trigger the stale binary rebuild")
	}
	if dispatched {
		t.Fatal("known non-API subcommand dispatched before stale rebuild")
	}
	if got := strings.Join(reexecArgs, " "); got != "policy resolve --role code.delivery" {
		t.Fatalf("reexec args = %q, want original command", got)
	}
}

func TestAppResolvesSingularPluralFormsAndSuggestsRealCommands(t *testing.T) {
	var ran []string
	record := func(name string) func(RunContext) error {
		return func(RunContext) error { ran = append(ran, name); return nil }
	}
	app := NewApp(AppOptions{
		Name: "demo",
		SubcommandGroups: []SubcommandGroup{
			{
				Name:        "records",
				Aliases:     []string{"rec"},
				Description: "records group",
				Subcommands: []Command{
					{Name: "create", Description: "Create a record", RunCtx: record("records create")},
					{Name: "get", Description: "Show one record", Args: ArgSchema{Positionals: []Positional{{Name: "id", Required: true}}}, RunCtx: record("records get")},
					{Name: "executions", Description: "List executions", RunCtx: record("records executions")},
				},
			},
			{
				Name:        "library",
				Description: "library group",
				Subcommands: []Command{
					{Name: "run", Description: "Run a library program", Args: ArgSchema{Positionals: []Positional{{Name: "name", Required: true}}, Flags: []Flag{{Name: "input", Repeated: true}}}, RunCtx: record("library run")},
				},
			},
		},
	})

	// A declared group alias, and its plural, is that group.
	if err := app.Run([]string{"rec", "create"}); err != nil {
		t.Fatalf("rec create: %v", err)
	}
	if err := app.Run([]string{"recs", "create"}); err != nil {
		t.Fatalf("recs create: %v", err)
	}
	ran = nil

	// A singular/plural form of exactly one name is that name.
	if err := app.Run([]string{"record", "create"}); err != nil {
		t.Fatalf("record create: %v", err)
	}
	if err := app.Run([]string{"records", "execution"}); err != nil {
		t.Fatalf("records execution: %v", err)
	}
	if got := strings.Join(ran, ","); got != "records create,records executions" {
		t.Fatalf("dispatched %q", got)
	}

	cases := []struct {
		args      []string
		want      []string
		forbidden string
	}{
		// The same leaf in another group, with its usage line.
		{args: []string{"run", "x"}, want: []string{`Unknown command: run (did you mean "library run"?)`, "demo library run <name> [options]", "Run a library program"}},
		// A sibling verb with the same meaning.
		{args: []string{"records", "show", "r1"}, want: []string{`Unknown subcommand: records show (did you mean "records get"?)`, "demo records get <id>"}},
		// A near spelling.
		{args: []string{"records", "creat"}, want: []string{`did you mean "records create"?`}},
		// Another group named as a leaf.
		{args: []string{"records", "library"}, want: []string{`did you mean "library"?`, "demo library <subcommand> [options]"}},
		// Distant garbage gets no suggestion.
		{args: []string{"zzzzzzzz"}, want: []string{"Unknown command: zzzzzzzz"}, forbidden: "did you mean"},
	}
	for _, tc := range cases {
		err := app.Run(tc.args)
		if err == nil {
			t.Fatalf("%v: expected an error", tc.args)
		}
		for _, want := range tc.want {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("%v: error %q lacks %q", tc.args, err.Error(), want)
			}
		}
		if tc.forbidden != "" && strings.Contains(err.Error(), tc.forbidden) {
			t.Errorf("%v: error %q contains %q", tc.args, err.Error(), tc.forbidden)
		}
	}
}

func TestSameWordForm(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want bool
	}{
		{"run", "runs", true},
		{"executions", "execution", true},
		{"process", "processes", true},
		{"policy", "policies", true},
		{"playbooks-seed", "playbooks-seeds", true},
		{"run", "run", false},
		{"ls", "lss", false},
		{"status", "state", false},
	} {
		if got := sameWordForm(tc.a, tc.b); got != tc.want {
			t.Errorf("sameWordForm(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestAppUsageErrorsCarryUsageAndAcceptNamedPositionals(t *testing.T) {
	var gotRun, gotArtifact, gotScenario string
	app := NewApp(AppOptions{
		Name: "demo",
		SubcommandGroups: []SubcommandGroup{{
			Name: "runs",
			Subcommands: []Command{{
				Name: "artifact",
				Args: ArgSchema{
					Positionals: []Positional{{Name: "run_id", Required: true}, {Name: "artifact_id", Required: true}},
					Flags:       []Flag{{Name: "scenario", Required: true}, {Name: "owner-id"}},
				},
				RunCtx: func(ctx RunContext) error {
					gotRun, gotArtifact, gotScenario = ctx.Positional("run_id"), ctx.Positional("artifact_id"), ctx.Flag("scenario")
					return nil
				},
			}},
		}},
	})

	if err := app.Run([]string{"runs", "artifact", "a1", "--scenario", "s", "--run-id", "r1"}); err != nil {
		t.Fatalf("named positional: %v", err)
	}
	if gotRun != "r1" || gotArtifact != "a1" || gotScenario != "s" {
		t.Fatalf("got run=%q artifact=%q scenario=%q", gotRun, gotArtifact, gotScenario)
	}

	err := app.Run([]string{"runs", "artifact", "r1", "a1", "--senario", "s"})
	var usageErr *UsageError
	if !errors.As(err, &usageErr) {
		t.Fatalf("expected a UsageError, got %v", err)
	}
	for _, want := range []string{
		"unknown option: --senario (did you mean --scenario?)",
		"Usage: demo runs artifact <run_id> <artifact_id> [options]",
		"Options: --scenario <value>, --owner-id <value>, --json",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q lacks %q", err.Error(), want)
		}
	}

	err = app.Run([]string{"runs", "artifact", "r1", "a1", "--scenario", "s", "--owner", "x"})
	if err == nil || !strings.Contains(err.Error(), "did you mean --owner-id?") {
		t.Fatalf("expected owner-id suggestion, got %v", err)
	}

	err = app.Run([]string{"runs", "artifact", "r1", "--scenario", "s"})
	if err == nil || !strings.Contains(err.Error(), "missing required positional <artifact_id>\nUsage: demo runs artifact") {
		t.Fatalf("expected missing positional with usage, got %v", err)
	}
}

func TestAppUnknownCommandUsesRecoveryHint(t *testing.T) {
	app := NewApp(AppOptions{
		Name: "demo",
		SubcommandGroups: []SubcommandGroup{{
			Name:        "log",
			Description: "log group",
			Subcommands: []Command{
				{Name: "decision-add", Description: "add", Run: func(args []string) error { return nil }},
			},
		}},
		UnknownCommandHint: func(args []string) string {
			if strings.Join(args, " ") == "log decision add" {
				return "Did you mean:\n  demo log decision-add"
			}
			return ""
		},
	})

	err := app.Run([]string{"log", "decision", "add"})
	if err == nil {
		t.Fatalf("expected unknown subcommand error")
	}
	if !strings.Contains(err.Error(), "Unknown subcommand: log decision") {
		t.Fatalf("expected unknown subcommand, got %v", err)
	}
	if !strings.Contains(err.Error(), "demo log decision-add") {
		t.Fatalf("expected recovery hint, got %v", err)
	}
}

func TestAppPreflightRuns(t *testing.T) {
	preflightCalled := false
	group := CommandGroup{
		Title: "Demo",
		Commands: []Command{
			{Name: "run", Description: "Run demo", Run: func(args []string) error { return nil }},
		},
	}
	app := NewApp(AppOptions{
		Name:     "demo",
		Commands: []CommandGroup{group},
		Preflight: func(cmd Command, global GlobalOptions) error {
			preflightCalled = true
			return nil
		},
	})

	if err := app.Run([]string{"run"}); err != nil {
		t.Fatalf("expected run to succeed: %v", err)
	}
	if !preflightCalled {
		t.Fatalf("expected preflight to be invoked")
	}
}

func TestAppPreflightErrorStopsRun(t *testing.T) {
	group := CommandGroup{
		Title: "Demo",
		Commands: []Command{
			{Name: "run", Description: "Run demo", Run: func(args []string) error {
				t.Fatalf("command should not execute when preflight fails")
				return nil
			}},
		},
	}
	app := NewApp(AppOptions{
		Name:     "demo",
		Commands: []CommandGroup{group},
		Preflight: func(cmd Command, global GlobalOptions) error {
			return errors.New("blocked")
		},
	})

	if err := app.Run([]string{"run"}); err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("expected preflight error, got %v", err)
	}
}

func TestAppPreflightReceivesFullSubcommandName(t *testing.T) {
	var gotName string
	app := NewApp(AppOptions{
		Name: "demo",
		SubcommandGroups: []SubcommandGroup{
			{
				Name: "campaigns",
				Subcommands: []Command{
					{Name: "list", NeedsAPI: true, Run: func(args []string) error { return nil }},
				},
			},
		},
		Preflight: func(cmd Command, global GlobalOptions) error {
			gotName = cmd.Name
			return nil
		},
	})

	if err := app.Run([]string{"campaigns", "list"}); err != nil {
		t.Fatalf("expected run to succeed: %v", err)
	}
	if gotName != "campaigns list" {
		t.Fatalf("preflight command name = %q, want %q", gotName, "campaigns list")
	}
}

func TestAppCommandHelpSkipsStaleCheckAndPreflight(t *testing.T) {
	var output bytes.Buffer
	staleCalled := false
	preflightCalled := false

	app := NewApp(AppOptions{
		Name: "demo",
		Commands: []CommandGroup{{
			Title: "Demo",
			Commands: []Command{{
				Name:        "status",
				Description: "Show health",
				Usage:       "demo status [--json]",
				HelpText:    "Use --json to print the raw payload.",
				NeedsAPI:    true,
				Run: func(args []string) error {
					t.Fatal("command should not run for help")
					return nil
				},
			}},
		}},
		StaleChecker: &cliutil.StaleChecker{
			BuildFingerprint: "fp",
			BuildSourceRoot:  t.TempDir(),
			FingerprintFunc: func(spec cliutil.FreshnessSpec) (string, error) {
				staleCalled = true
				return "fp", nil
			},
		},
		Preflight: func(cmd Command, global GlobalOptions) error {
			preflightCalled = true
			return nil
		},
	})

	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	done := make(chan struct{})
	go func() {
		_, _ = output.ReadFrom(r)
		close(done)
	}()

	runErr := app.Run([]string{"status", "--help"})
	_ = w.Close()
	os.Stdout = originalStdout
	<-done
	if runErr != nil {
		t.Fatalf("Run: %v", runErr)
	}
	if staleCalled {
		t.Fatalf("stale checker should not run for command help")
	}
	if preflightCalled {
		t.Fatalf("preflight should not run for command help")
	}
	text := output.String()
	if !strings.Contains(text, "demo status [--json]") {
		t.Fatalf("expected command usage in help output, got %q", text)
	}
}

func TestAppSubcommandHelpSkipsStaleCheckAndPreflight(t *testing.T) {
	staleCalled := false
	preflightCalled := false
	app := NewApp(AppOptions{
		Name: "demo",
		SubcommandGroups: []SubcommandGroup{{
			Name:        "chat",
			Description: "Chat operations",
			NeedsAPI:    true,
			Subcommands: []Command{{
				Name:        "list",
				Description: "List chats",
				Usage:       "demo chat list [--json]",
				Run: func(args []string) error {
					t.Fatal("subcommand should not run for help")
					return nil
				},
			}},
		}},
		StaleChecker: &cliutil.StaleChecker{
			BuildFingerprint: "fp",
			BuildSourceRoot:  t.TempDir(),
			FingerprintFunc: func(spec cliutil.FreshnessSpec) (string, error) {
				staleCalled = true
				return "fp", nil
			},
		},
		Preflight: func(cmd Command, global GlobalOptions) error {
			preflightCalled = true
			return nil
		},
	})

	if err := app.Run([]string{"chat", "list", "--help"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if staleCalled {
		t.Fatalf("stale checker should not run for subcommand help")
	}
	if preflightCalled {
		t.Fatalf("preflight should not run for subcommand help")
	}
}

// TestAppSubcommandGroup_DefaultSubcommandFallback verifies that when a
// SubcommandGroup declares DefaultSubcommand, unknown args[0] tokens are
// routed there with the full arg slice (not args[1:]) — preserving the
// shorthand `<group> <free-form args>` form.
func TestAppSubcommandGroup_DefaultSubcommandFallback(t *testing.T) {
	var capturedArgs []string
	app := NewApp(AppOptions{
		Name: "demo",
		SubcommandGroups: []SubcommandGroup{{
			Name:              "search",
			Description:       "search ops",
			DefaultSubcommand: "query",
			Subcommands: []Command{
				{
					Name:        "query",
					Description: "search corpus",
					Run: func(args []string) error {
						capturedArgs = args
						return nil
					},
				},
				{
					Name:        "status",
					Description: "backend status",
					Run: func(args []string) error {
						t.Fatal("status should not be invoked for unknown subcommand")
						return nil
					},
				},
			},
		}},
	})

	if err := app.Run([]string{"search", "validate", "manifest"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(capturedArgs) != 2 || capturedArgs[0] != "validate" || capturedArgs[1] != "manifest" {
		t.Fatalf("expected query to receive [validate manifest], got %v", capturedArgs)
	}

	// Known subcommand still wins over the default.
	statusCalled := false
	app2 := NewApp(AppOptions{
		Name: "demo",
		SubcommandGroups: []SubcommandGroup{{
			Name:              "search",
			DefaultSubcommand: "query",
			Subcommands: []Command{
				{Name: "query", Run: func([]string) error { t.Fatal("query should not run"); return nil }},
				{Name: "status", Run: func([]string) error { statusCalled = true; return nil }},
			},
		}},
	})
	if err := app2.Run([]string{"search", "status"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !statusCalled {
		t.Fatalf("explicit subcommand should still match before falling back to default")
	}
}

// --- dispatchCommand routing (RunCtx vs Run vs neither) -------------------
//
// The dispatcher routes a Command's execution through either the declarative
// RunCtx path (parser → RunContext → handler) or the legacy Run path
// ([]string → handler). These tests pin the routing rules; their coverage
// is the load-bearing assertion that iteration-1's substrate addition
// didn't break the legacy path or mis-handle the new one.

// TestDispatchCommand_RoutesToRunCtxWhenSet covers the canonical declarative
// path. The legacy Run field is set too — RunCtx must win.
func TestDispatchCommand_RoutesToRunCtxWhenSet(t *testing.T) {
	var runCtxCalled, runCalled bool
	var sawTitle string

	cmd := Command{
		Name: "create",
		Args: ArgSchema{Flags: []Flag{{Name: "title", Required: true}}},
		Run: func(args []string) error {
			runCalled = true
			return nil
		},
		RunCtx: func(ctx RunContext) error {
			runCtxCalled = true
			sawTitle = ctx.Flag("title")
			return nil
		},
	}

	app := NewApp(AppOptions{Name: "demo"})
	if err := app.dispatchCommand("demo", cmd, []string{"--title", "hello"}); err != nil {
		t.Fatalf("dispatchCommand: %v", err)
	}
	if !runCtxCalled {
		t.Error("RunCtx not invoked")
	}
	if runCalled {
		t.Error("legacy Run was invoked despite RunCtx being set")
	}
	if sawTitle != "hello" {
		t.Errorf("RunCtx saw title=%q, want %q", sawTitle, "hello")
	}
}

// TestDispatchCommand_HelpReturnsNilWithoutCallingHandler confirms the
// dispatcher swallows ErrHelpRequested into a successful exit. The handler
// must NOT run.
func TestDispatchCommand_HelpReturnsNilWithoutCallingHandler(t *testing.T) {
	var called bool
	cmd := Command{
		Name:        "create",
		Description: "Create a thing",
		Args:        ArgSchema{Flags: []Flag{{Name: "title", Required: true}}},
		RunCtx: func(ctx RunContext) error {
			called = true
			return nil
		},
	}

	app := NewApp(AppOptions{Name: "demo"})
	if err := app.dispatchCommand("demo", cmd, []string{"--help"}); err != nil {
		t.Fatalf("expected nil for --help, got %v", err)
	}
	if called {
		t.Error("handler ran on --help; help should short-circuit")
	}
}

// TestDispatchCommand_PropagatesParserErrors confirms the dispatcher does not
// swallow other parser errors. These must reach the user so the CLI exits non-zero.
func TestDispatchCommand_PropagatesParserErrors(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		contains string
	}{
		{
			name:     "missing required flag",
			args:     []string{},
			contains: "missing required flag --title",
		},
		{
			name:     "unknown option",
			args:     []string{"--nope"},
			contains: "unknown option: --nope",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := Command{
				Name:   "create",
				Args:   ArgSchema{Flags: []Flag{{Name: "title", Required: true}}},
				RunCtx: func(ctx RunContext) error { return nil },
			}
			app := NewApp(AppOptions{Name: "demo"})
			err := app.dispatchCommand("demo", cmd, tc.args)
			if err == nil {
				t.Fatal("expected parser error, got nil")
			}
			if errors.Is(err, ErrHelpRequested) {
				t.Errorf("ErrHelpRequested should not propagate, got: %v", err)
			}
			if !strings.Contains(err.Error(), tc.contains) {
				t.Errorf("error %q did not contain %q", err.Error(), tc.contains)
			}
		})
	}
}

// TestDispatchCommand_FallsBackToLegacyRun covers the additive-only contract:
// Commands without RunCtx still dispatch through the original Run path so
// scenarios that pre-date iteration-1 keep working unchanged.
func TestDispatchCommand_FallsBackToLegacyRun(t *testing.T) {
	var sawArgs []string
	cmd := Command{
		Name: "legacy",
		Run: func(args []string) error {
			sawArgs = args
			return nil
		},
	}

	app := NewApp(AppOptions{Name: "demo"})
	if err := app.dispatchCommand("demo", cmd, []string{"a", "b"}); err != nil {
		t.Fatalf("dispatchCommand: %v", err)
	}
	if len(sawArgs) != 2 || sawArgs[0] != "a" || sawArgs[1] != "b" {
		t.Errorf("legacy Run saw args=%v, want [a b]", sawArgs)
	}
}

// TestDispatchCommand_NeitherHandlerErrors guards against silently registering
// a Command that can never run. The dispatcher must surface the misconfiguration.
func TestDispatchCommand_NeitherHandlerErrors(t *testing.T) {
	cmd := Command{Name: "broken"}
	app := NewApp(AppOptions{Name: "demo"})
	err := app.dispatchCommand("demo", cmd, []string{})
	if err == nil {
		t.Fatal("expected error when neither Run nor RunCtx is set")
	}
	if !strings.Contains(err.Error(), "broken") {
		t.Errorf("error did not name the offending command: %v", err)
	}
}

// TestDispatchCommand_JSONFlagSurfacesViaRunContext covers the parser's
// reserved --json pseudo-flag.
func TestDispatchCommand_JSONFlagSurfacesViaRunContext(t *testing.T) {
	var sawJSON bool
	cmd := Command{
		Name: "list",
		RunCtx: func(ctx RunContext) error {
			sawJSON = ctx.JSON()
			return nil
		},
	}

	app := NewApp(AppOptions{Name: "demo"})
	if err := app.dispatchCommand("demo", cmd, []string{"--json"}); err != nil {
		t.Fatalf("dispatchCommand: %v", err)
	}
	if !sawJSON {
		t.Error("RunContext.JSON() = false despite --json being passed")
	}
}

// TestAppRun_DispatchesRunCtxThroughEntryPoint exercises the production
// callsite end-to-end and confirms RunCtx-style commands dispatch correctly.
func TestAppRun_DispatchesRunCtxThroughEntryPoint(t *testing.T) {
	var sawTitle string
	app := NewApp(AppOptions{
		Name: "demo",
		Commands: []CommandGroup{{
			Title: "Notes",
			Commands: []Command{{
				Name: "create",
				Args: ArgSchema{Flags: []Flag{{Name: "title", Required: true}}},
				RunCtx: func(ctx RunContext) error {
					sawTitle = ctx.Flag("title")
					return nil
				},
			}},
		}},
	})
	if err := app.Run([]string{"create", "--title", "hello"}); err != nil {
		t.Fatalf("App.Run: %v", err)
	}
	if sawTitle != "hello" {
		t.Errorf("RunCtx saw title=%q, want hello", sawTitle)
	}
}

// TestAppRun_RunCtxHelpDoesNotShortCircuitBeforeParser proves App.Run's
// pre-dispatch help short-circuit (which exists for legacy Run commands)
// does NOT swallow --help for RunCtx commands. The parser inside
// dispatchCommand handles --help so renderHelp gets the ArgSchema.
func TestAppRun_RunCtxHelpDoesNotShortCircuitBeforeParser(t *testing.T) {
	var called bool
	app := NewApp(AppOptions{
		Name: "demo",
		Commands: []CommandGroup{{
			Title: "Notes",
			Commands: []Command{{
				Name: "create",
				Args: ArgSchema{Flags: []Flag{{Name: "title", Required: true}}},
				RunCtx: func(ctx RunContext) error {
					called = true
					return nil
				},
			}},
		}},
	})
	if err := app.Run([]string{"create", "--help"}); err != nil {
		t.Fatalf("App.Run --help: %v", err)
	}
	if called {
		t.Error("handler ran on --help; help should short-circuit before RunCtx")
	}
}
