package cliruntime

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// vrooliRootHelp mirrors the shape of `vrooli --help`: top-level section
// headers ending in `:`, indented `<name>  <description>` rows.
const vrooliRootHelp = `Vrooli CLI

Usage:
  vrooli <command> [options]

Lifecycle Commands:
    setup              Initialize the development environment
    develop            Start development servers
    build              Build project-level binaries

Scenario Management:
    scenario           Manage scenarios from their source locations

Options:
  --help, -h           Show help
  --json               Emit JSON output
`

const vrooliScenarioHelp = `Vrooli Scenario Commands

Usage:
  vrooli scenario <subcommand> [options]

Read-only Commands:
    list               List discovered scenarios
    info               Show scenario metadata

Lifecycle and Utility Commands:
    start              Start a scenario
    stop               Stop a running scenario
`

const vrooliScenarioStartHelp = `Usage:
  vrooli scenario start <scenario-name>... [options]

Start a scenario

Options:
  --path <path>
  --best-effort
  --json               Emit JSON output
`

// promptManagerHelp uses the cli-core "Commands:" with category-subheader
// shape — category lines have no description and must be tolerated.
const promptManagerHelp = `prompt-manager CLI

Usage:
  prompt-manager <command> [options]

Global Options:
  --api-base <url>   Override API base URL
  --no-color         Disable ANSI color output

Commands:
  Meta
    help               Show this help message
    version            Show CLI version

  Health
    status             Check API health

  Skills
    skill              Manage skills (list|show|read|add)
`

const promptManagerSkillHelp = `prompt-manager skill - Manage skills (list|show|read|add|sync)

Usage:
  prompt-manager skill <subcommand> [args]

Subcommands:
  list, ls              List all skills
  show, get <id>        Show skill details
  read <identifier>...  Read skills (content or combined output)
  sync                  Sync skills with hash-based change detection
`

func staticRunner(t *testing.T, table map[string]string) (Runner, *int) {
	t.Helper()
	calls := 0
	run := func(_ context.Context, bin string, args []string) ([]byte, error) {
		calls++
		key := strings.TrimSpace(bin + " " + strings.Join(args, " "))
		if out, ok := table[key]; ok {
			return []byte(out), nil
		}
		return nil, errors.New("no help fixture for: " + key)
	}
	return run, &calls
}

func TestParseHelpTree_VrooliRecurses(t *testing.T) {
	run, _ := staticRunner(t, map[string]string{
		"vrooli":                vrooliRootHelp,
		"vrooli scenario":       vrooliScenarioHelp,
		"vrooli scenario start": vrooliScenarioStartHelp,
		"vrooli scenario list":  "Usage: vrooli scenario list\n\nList discovered scenarios\n",
		"vrooli scenario info":  "Usage: vrooli scenario info\n\nShow scenario metadata\n",
		"vrooli scenario stop":  "Usage: vrooli scenario stop\n\nStop a running scenario\n",
		"vrooli setup":          "Usage: vrooli setup\n\nInitialize the dev environment\n",
		"vrooli develop":        "Usage: vrooli develop\n\nStart dev servers\n",
		"vrooli build":          "Usage: vrooli build\n\nBuild project\n",
	})
	records := ParseHelpTree(context.Background(), run, "vrooli", HelpTreeOptions{Origin: "vrooli"})

	paths := make(map[string]Command, len(records))
	for _, r := range records {
		paths[r.FullPath] = r
	}

	wantLeaves := []string{
		"vrooli setup",
		"vrooli develop",
		"vrooli build",
		"vrooli scenario list",
		"vrooli scenario info",
		"vrooli scenario start",
		"vrooli scenario stop",
	}
	for _, want := range wantLeaves {
		if _, ok := paths[want]; !ok {
			t.Errorf("missing leaf %q; got: %v", want, keys(paths))
		}
	}

	// No options/flags should sneak in as command names.
	for path := range paths {
		if strings.Contains(path, "--") {
			t.Errorf("leaf path contains flag-looking token: %q", path)
		}
	}

	// Origin is set on every record.
	for _, r := range records {
		if r.Origin != "vrooli" {
			t.Errorf("record %q has Origin=%q, want %q", r.FullPath, r.Origin, "vrooli")
		}
		if r.Source != SourceHelp {
			t.Errorf("record %q has Source=%q, want %q", r.FullPath, r.Source, SourceHelp)
		}
	}

	// Group is populated for depth>=2 leaves.
	if got := paths["vrooli scenario start"]; got.Group != "scenario" {
		t.Errorf("vrooli scenario start: Group=%q, want %q", got.Group, "scenario")
	}
	// Group is empty for depth=1 leaves.
	if got := paths["vrooli setup"]; got.Group != "" {
		t.Errorf("vrooli setup: Group=%q, want empty", got.Group)
	}
}

func TestParseHelpTree_DepthLimit(t *testing.T) {
	run, _ := staticRunner(t, map[string]string{
		"vrooli":                vrooliRootHelp,
		"vrooli scenario":       vrooliScenarioHelp,
		"vrooli scenario start": vrooliScenarioStartHelp,
		"vrooli setup":          "Usage: vrooli setup\n\nInitialize\n",
		"vrooli develop":        "Usage: vrooli develop\n\nStart servers\n",
		"vrooli build":          "Usage: vrooli build\n\nBuild\n",
	})
	// MaxDepth=1 means: do not recurse past the root. Every entry from
	// `vrooli --help` becomes a leaf, including `scenario` (no recursion into
	// `vrooli scenario --help`).
	records := ParseHelpTree(context.Background(), run, "vrooli", HelpTreeOptions{Origin: "vrooli", MaxDepth: 1})
	paths := make(map[string]bool, len(records))
	for _, r := range records {
		paths[r.FullPath] = true
		if tokens := len(strings.Fields(r.FullPath)); tokens > 2 {
			t.Errorf("MaxDepth=1 but emitted leaf at %d tokens: %q", tokens, r.FullPath)
		}
	}
	for _, want := range []string{"vrooli setup", "vrooli develop", "vrooli build", "vrooli scenario"} {
		if !paths[want] {
			t.Errorf("missing depth-1 leaf %q; got %v", want, recordPaths(records))
		}
	}
}

func TestParseHelpTree_SkipsOptionsAndFlags(t *testing.T) {
	run, _ := staticRunner(t, map[string]string{
		"vrooli":          vrooliRootHelp,
		"vrooli setup":    "Usage: vrooli setup\n\nInitialize\n",
		"vrooli develop":  "Usage: vrooli develop\n\nStart servers\n",
		"vrooli build":    "Usage: vrooli build\n\nBuild\n",
		"vrooli scenario": "no subcommands here\n",
	})
	records := ParseHelpTree(context.Background(), run, "vrooli", HelpTreeOptions{Origin: "vrooli"})
	for _, r := range records {
		// `--help`, `--json`, etc. must never appear as a command name.
		if strings.HasPrefix(r.Name, "-") {
			t.Errorf("flag leaked as command: %q", r.Name)
		}
	}
}

func TestParseHelpTree_CategoryLabelsIgnored(t *testing.T) {
	run, _ := staticRunner(t, map[string]string{
		"prompt-manager":         promptManagerHelp,
		"prompt-manager help":    "show help",
		"prompt-manager version": "1.0.0",
		"prompt-manager status":  "ok",
		"prompt-manager skill":   "manage skills",
	})
	records := ParseHelpTree(context.Background(), run, "prompt-manager", HelpTreeOptions{Origin: "prompt-manager"})
	names := make(map[string]bool)
	for _, r := range records {
		names[r.Name] = true
	}
	// Category subheaders (Meta, Health, Skills) must not become records.
	for _, label := range []string{"Meta", "Health", "Skills"} {
		if names[label] {
			t.Errorf("category label %q leaked as command record", label)
		}
	}
	// The cli-core `help` pseudo-command ("Show this help message") is filtered
	// out — it is not a real operation and pollutes the index.
	if names["help"] {
		t.Errorf("help pseudo-command leaked as a command record; got names=%v", names)
	}
	// Actual commands must remain.
	for _, want := range []string{"version", "status", "skill"} {
		if !names[want] {
			t.Errorf("missing command %q; got names=%v", want, names)
		}
	}
}

func TestParseHelpTree_ParsesCommandRowsWithUsageTail(t *testing.T) {
	run, _ := staticRunner(t, map[string]string{
		"prompt-manager":            promptManagerHelp,
		"prompt-manager version":    "1.0.0",
		"prompt-manager status":     "ok",
		"prompt-manager skill":      promptManagerSkillHelp,
		"prompt-manager skill list": "Usage: prompt-manager skill list\n\nList all skills\n",
		"prompt-manager skill show": "Usage: prompt-manager skill show <id>\n\nShow skill details\n",
		"prompt-manager skill read": "Usage: prompt-manager skill read <identifier>...\n\nRead skills\n",
		"prompt-manager skill sync": "Usage: prompt-manager skill sync\n\nSync skills\n",
	})

	records := ParseHelpTree(context.Background(), run, "prompt-manager", HelpTreeOptions{Origin: "prompt-manager"})
	paths := make(map[string]Command, len(records))
	for _, r := range records {
		paths[r.FullPath] = r
	}

	for _, want := range []string{
		"prompt-manager skill list",
		"prompt-manager skill show",
		"prompt-manager skill read",
		"prompt-manager skill sync",
	} {
		if _, ok := paths[want]; !ok {
			t.Errorf("missing command %q; got %v", want, keys(paths))
		}
	}
}

func TestParseHelpTree_IgnoresIndentedNotesProse(t *testing.T) {
	run, _ := staticRunner(t, map[string]string{
		"demo": `demo

Subcommands:
  records  Manage narrative records
`,
		"demo records": `demo records

Subcommands:
  create  Create a record

Notes:
  - A record already exists after a successful review.
`,
		"demo records create": `demo records create - Create a record`,
	})
	records := ParseHelpTree(context.Background(), run, "demo", HelpTreeOptions{Origin: "demo"})
	if len(records) != 1 || records[0].Group != "records" || records[0].Name != "create" {
		t.Fatalf("notes prose must not be parsed as a command: %+v", records)
	}
}

// TestParseHelpEntries_HelpPseudoCommandFiltered pins the precise filter: the
// cli-core `help` entry (description "Show this help message") is dropped, but a
// command named `help` with a genuinely different, non-help-printing description
// is preserved.
func TestParseHelpEntries_HelpPseudoCommandFiltered(t *testing.T) {
	const withPseudo = `myapp CLI

Commands:
    help               Show this help message
    deploy             Deploy the app
`
	got := parseHelpEntries([]byte(withPseudo))
	for _, e := range got {
		if e.Name == "help" {
			t.Errorf("help pseudo-command not filtered: %+v", got)
		}
	}
	if len(got) != 1 || got[0].Name != "deploy" {
		t.Errorf("expected only deploy to survive, got %+v", got)
	}

	// A real subcommand named `help` that does something else is kept.
	const realHelp = `support CLI

Commands:
    help               Open a support ticket for assistance
`
	got2 := parseHelpEntries([]byte(realHelp))
	if len(got2) != 1 || got2[0].Name != "help" {
		t.Errorf("real help subcommand was wrongly filtered, got %+v", got2)
	}
}

// TestParseHelpEntries_LongCommandWithSingleSpaceBeforeDescription protects
// the aligned-help edge case where the longest command has only one separating
// space before its description. This is common in cli-core output and must not
// make runtime manifest validation report a healthy command as missing.
func TestParseHelpEntries_LongCommandWithSingleSpaceBeforeDescription(t *testing.T) {
	const help = `example CLI

Subcommands:
  short-command          A shorter command with padded alignment.
  element-at-coordinate Probe an element at a coordinate.
`

	got := parseHelpEntries([]byte(help))
	if len(got) != 2 {
		t.Fatalf("expected two command entries, got %+v", got)
	}
	if got[1].Name != "element-at-coordinate" || got[1].Description != "Probe an element at a coordinate." {
		t.Fatalf("long command with one separator was not parsed correctly: %+v", got[1])
	}
}

// TestParseHelpTree_AliasSubcommands covers the cli-core "Subcommands:" listing
// where each row is a comma-separated alias list (`create, add  <desc>`). Before
// the commandLineRE fix the whole row failed to match, so `action` collapsed to
// a leaf and its real subcommand `prompt-manager action create` was never indexed
// — a concrete REQ-P0-004 recall miss.
func TestParseHelpTree_AliasSubcommands(t *testing.T) {
	const pmRoot = `prompt-manager CLI

Commands:
    action             Manage Actions (list|show|create)
`
	const actionHelp = `prompt-manager action - Manage Actions

Usage: prompt-manager action <subcommand> [args]

Subcommands:
  create, add          Create an Action from a --command
  list, ls             List Actions
`
	run, _ := staticRunner(t, map[string]string{
		"prompt-manager":               pmRoot,
		"prompt-manager action":        actionHelp,
		"prompt-manager action create": "Create an Action from a --command\n",
		"prompt-manager action list":   "List Actions\n",
	})
	records := ParseHelpTree(context.Background(), run, "prompt-manager", HelpTreeOptions{Origin: "prompt-manager"})
	paths := make(map[string]bool, len(records))
	for _, r := range records {
		paths[r.FullPath] = true
	}
	for _, want := range []string{"prompt-manager action create", "prompt-manager action list"} {
		if !paths[want] {
			t.Errorf("missing aliased subcommand %q; got %v", want, recordPaths(records))
		}
	}
	// The canonical (first) alias is used, not the secondary one.
	if paths["prompt-manager action add"] {
		t.Errorf("secondary alias 'add' should not be a separate record; got %v", recordPaths(records))
	}
}

func TestParseHelpTree_BinaryMissing(t *testing.T) {
	run := func(context.Context, string, []string) ([]byte, error) {
		return nil, errors.New("exec: file not found")
	}
	records := ParseHelpTree(context.Background(), run, "ghost", HelpTreeOptions{Origin: "ghost"})
	if len(records) != 1 {
		t.Fatalf("want 1 stub, got %d", len(records))
	}
	if records[0].Source != SourceHelpFailed {
		t.Errorf("Source=%q, want %q", records[0].Source, SourceHelpFailed)
	}
	if records[0].FullPath != "ghost" {
		t.Errorf("FullPath=%q, want %q", records[0].FullPath, "ghost")
	}
}

func TestParseHelpTree_EmptyOutput(t *testing.T) {
	run := func(context.Context, string, []string) ([]byte, error) {
		return []byte("   \n\n"), nil
	}
	records := ParseHelpTree(context.Background(), run, "blank", HelpTreeOptions{Origin: "blank"})
	if len(records) != 1 {
		t.Fatalf("want 1 stub, got %d", len(records))
	}
	if records[0].Source != SourceHelpFailed {
		t.Errorf("Source=%q, want %q", records[0].Source, SourceHelpFailed)
	}
}

func TestParseHelpTree_CycleDetection(t *testing.T) {
	// A buggy CLI where `<bin> <sub> --help` returns the same text as
	// `<bin> --help`. The parser must treat <sub> as a leaf, not infinite-loop.
	const looped = `Usage:
  loopy <command>

Commands:
    inner              Inner command
`
	run, _ := staticRunner(t, map[string]string{
		"loopy":       looped,
		"loopy inner": looped, // same signature as root
	})
	records := ParseHelpTree(context.Background(), run, "loopy", HelpTreeOptions{Origin: "loopy"})
	// `loopy inner` should be a single leaf record; the parser must not
	// re-walk the same help blob.
	count := 0
	for _, r := range records {
		if r.FullPath == "loopy inner" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("loopy inner emitted %d times, want 1; records: %v", count, recordPaths(records))
	}
}

// TestLeafRecordDescriptionNoDuplicateSummary guards WS4: a leaf command whose
// --help body leads with its summary line must not store that line twice in
// Description (which pollutes both the embedded vector and the displayed
// snippet). The leaf is reached when a subcommand's --help reveals no further
// subcommands, so its full help body becomes the Description.
func TestLeafRecordDescriptionNoDuplicateSummary(t *testing.T) {
	const providerLogsHelp = "audio-tools provider logs - Stream provider logs\n\nUsage:\n  audio-tools provider logs [options]\n\nOptions:\n  --follow\n"
	run, _ := staticRunner(t, map[string]string{
		"audio-tools":               "audio-tools CLI\n\nUsage:\n  audio-tools <command>\n\nCommands:\n    provider           Manage providers\n",
		"audio-tools provider":      "Usage:\n  audio-tools provider <subcommand>\n\nCommands:\n    logs               Stream provider logs\n",
		"audio-tools provider logs": providerLogsHelp,
	})
	records := ParseHelpTree(context.Background(), run, "audio-tools", HelpTreeOptions{Origin: "audio-tools"})

	var leaf *Command
	for i := range records {
		if records[i].FullPath == "audio-tools provider logs" {
			leaf = &records[i]
			break
		}
	}
	if leaf == nil {
		t.Fatalf("missing leaf 'audio-tools provider logs'; got %v", recordPaths(records))
	}

	summary := "audio-tools provider logs - Stream provider logs"
	if !strings.HasPrefix(leaf.Description, summary) {
		t.Fatalf("Description must lead with the summary line, got %q", leaf.Description)
	}
	if strings.Count(leaf.Description, summary) != 1 {
		t.Fatalf("summary line must appear exactly once, got %d in %q", strings.Count(leaf.Description, summary), leaf.Description)
	}
	// No adjacent duplicate of any line (the precise old defect).
	lines := strings.Split(leaf.Description, "\n")
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "" && lines[i] == lines[i-1] {
			t.Fatalf("adjacent duplicate line %q in Description %q", lines[i], leaf.Description)
		}
	}
}

func keys(m map[string]Command) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func recordPaths(records []Command) []string {
	out := make([]string, 0, len(records))
	for _, r := range records {
		out = append(out, r.FullPath)
	}
	return out
}
