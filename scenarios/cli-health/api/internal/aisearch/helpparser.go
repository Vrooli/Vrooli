package aisearch

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"regexp"
	"strings"
)

// helpRunner invokes `<bin> args... --help` and returns combined stdout.
// seam: helpRunner — production wires exec.CommandContext (discovery.go);
// tests inject a static map so the recursive parser can be exercised against
// captured help corpora without spawning processes.
type helpRunner func(ctx context.Context, bin string, args []string) ([]byte, error)

// HelpTreeOptions configure ParseHelpTree.
type HelpTreeOptions struct {
	// Origin is the value placed on every emitted CommandRecord. Typically the
	// binary name ("vrooli") for an ExternalCLI or the scenario name for a
	// scenario CLI.
	Origin string
	// MaxDepth bounds recursion. depth=1 walks `<bin> --help` only; depth=3 is
	// the production default and covers `vrooli scenario start`.
	MaxDepth int
}

const defaultHelpMaxDepth = 3

// ParseHelpTree recursively walks a CLI's `--help` tree and emits one
// CommandRecord per leaf (command whose --help reveals no further
// subcommands or whose depth hit MaxDepth). On any failure a single
// Source=help-failed stub is returned so the CLI is still searchable by
// name.
func ParseHelpTree(ctx context.Context, run helpRunner, bin string, opts HelpTreeOptions) []CommandRecord {
	origin := strings.TrimSpace(opts.Origin)
	if origin == "" {
		origin = bin
	}
	maxDepth := opts.MaxDepth
	if maxDepth <= 0 {
		maxDepth = defaultHelpMaxDepth
	}

	rootOut, err := run(ctx, bin, nil)
	if err != nil || strings.TrimSpace(string(rootOut)) == "" {
		return []CommandRecord{stubRecord(origin, err)}
	}

	seen := make(map[string]struct{})
	seen[helpSignature(rootOut)] = struct{}{}

	records := walkHelpTree(ctx, run, bin, origin, nil, "", rootOut, 1, maxDepth, seen)
	if len(records) == 0 {
		// Root had no parseable subcommands — treat the CLI itself as a leaf.
		return []CommandRecord{rootLeafRecord(origin, rootOut)}
	}
	return records
}

// walkHelpTree recurses the --help tree. groupDesc is the nearest enclosing
// group's one-line summary (the description of the parent entry we recursed
// through); it is threaded down so every leaf can fold its group's real-world
// vocabulary into its embedding text.
func walkHelpTree(ctx context.Context, run helpRunner, bin, origin string, parents []string, groupDesc string, helpOut []byte, depth, maxDepth int, seen map[string]struct{}) []CommandRecord {
	entries := parseHelpEntries(helpOut)
	if len(entries) == 0 {
		return []CommandRecord{leafRecord(origin, parents, groupDesc, helpOut)}
	}
	if depth >= maxDepth {
		out := make([]CommandRecord, 0, len(entries))
		for _, e := range entries {
			out = append(out, leafRecordFromEntry(origin, parents, groupDesc, e))
		}
		return out
	}

	out := make([]CommandRecord, 0, len(entries))
	for _, e := range entries {
		childPath := append(append([]string{}, parents...), e.Name)
		childOut, err := run(ctx, bin, childPath)
		if err != nil || strings.TrimSpace(string(childOut)) == "" {
			out = append(out, leafRecordFromEntry(origin, parents, groupDesc, e))
			continue
		}
		sig := helpSignature(childOut)
		if _, dup := seen[sig]; dup {
			// Same help text as an ancestor — likely a leaf whose --help echoes
			// the parent. Treat as leaf.
			out = append(out, leafRecordFromEntry(origin, parents, groupDesc, e))
			continue
		}
		seen[sig] = struct{}{}

		// e is the group we are descending into; its description becomes the
		// group context for the leaves below it (keep the nearest non-empty one).
		childGroupDesc := groupDesc
		if strings.TrimSpace(e.Description) != "" {
			childGroupDesc = e.Description
		}
		childRecords := walkHelpTree(ctx, run, bin, origin, childPath, childGroupDesc, childOut, depth+1, maxDepth, seen)
		if len(childRecords) == 0 {
			out = append(out, leafRecordFromEntry(origin, parents, groupDesc, e))
			continue
		}
		out = append(out, childRecords...)
	}
	return out
}

// helpEntry is one parsed command line: indented `<name><whitespace><desc>`.
type helpEntry struct {
	Name        string
	Description string
}

var (
	// Section headers end with `:` and live at column 0 (no leading whitespace
	// other than the trailing newline). We accept any line ending with `:`
	// regardless of trailing spaces.
	sectionHeaderRE = regexp.MustCompile(`^(\S[^:]*):\s*$`)

	// A command line is indented and looks like: `<spaces><name><2+ spaces><desc>`.
	// The name may be a comma-separated alias list (`create, add   <desc>`), the
	// cli-core convention — without this the whole line failed to match and the
	// command (and all its subcommands) was silently dropped, e.g.
	// `prompt-manager action create`. We also accept lines without a description
	// (visual category labels) so the caller can filter them out.
	commandLineRE = regexp.MustCompile(`^(\s+)([A-Za-z][A-Za-z0-9_.-]*(?:,\s*[A-Za-z0-9_.-]+)*)(?:\s{2,}(.+))?\s*$`)

	// Section header names that contain any of these substrings (case-insensitive)
	// hold flags/usage text, not subcommands. Skip them.
	skipSectionSubstrings = []string{
		"option", "flag", "global option", "example", "usage", "alias",
		"argument", "environment", "documentation",
	}
)

// parseHelpEntries extracts the subcommand entries from a help blob. It walks
// section by section; sections whose header signals "flags/options/etc." are
// skipped. Category sub-labels (a single token at lower indentation than the
// commands below it, with no description) are tolerated and ignored.
func parseHelpEntries(helpOut []byte) []helpEntry {
	lines := strings.Split(string(helpOut), "\n")
	var (
		out      []helpEntry
		inAccept bool
		seen     = make(map[string]struct{})
	)
	for _, raw := range lines {
		line := strings.TrimRight(raw, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if m := sectionHeaderRE.FindStringSubmatch(line); m != nil {
			inAccept = !isSkipSection(m[1])
			continue
		}
		if !inAccept {
			continue
		}
		m := commandLineRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := m[2]
		desc := strings.TrimSpace(m[3])
		// Skip flags (defensive — section header filter should catch most).
		if strings.HasPrefix(name, "-") {
			continue
		}
		// Skip category labels (name only, no description).
		if desc == "" {
			continue
		}
		// Some CLIs list aliases as `list, ls` — take the canonical first name.
		if comma := strings.IndexByte(name, ','); comma >= 0 {
			name = strings.TrimSpace(name[:comma])
		}
		// Skip names that look like usage patterns rather than identifiers.
		if !isCommandName(name) {
			continue
		}
		// Skip the cli-core "help" pseudo-command: every CLI emits a `help`
		// entry under its Meta section whose only purpose is to PRINT the help
		// text (description like "Show this help message"). It is not a real
		// operation — help is the `--help` flag — and indexing one near-identical
		// "Show this help message" record per scenario pollutes the vector index
		// with a generic semantic magnet that crowds out real commands. Real
		// subcommands that merely share the name `help` (with a different,
		// non-help-printing description) are kept.
		if isHelpPseudoCommand(name, desc) {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, helpEntry{Name: name, Description: desc})
	}
	return out
}

var commandNameRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.-]*$`)

func isCommandName(s string) bool {
	return commandNameRE.MatchString(s)
}

// isHelpPseudoCommand reports whether a parsed entry is the spurious `help`
// pseudo-command emitted by cli-core CLIs (a `help` entry whose description just
// says it prints/shows the help text). It is matched conservatively — name must
// be exactly "help" AND the description must signal "show … help" — so a real
// subcommand that happens to be named `help` but does something else is kept.
func isHelpPseudoCommand(name, desc string) bool {
	if strings.ToLower(strings.TrimSpace(name)) != "help" {
		return false
	}
	d := strings.ToLower(desc)
	return strings.Contains(d, "help") &&
		(strings.Contains(d, "show") || strings.Contains(d, "print") ||
			strings.Contains(d, "display") || strings.Contains(d, "this"))
}

func isSkipSection(header string) bool {
	h := strings.ToLower(strings.TrimSpace(header))
	for _, sub := range skipSectionSubstrings {
		if strings.Contains(h, sub) {
			return true
		}
	}
	return false
}

func helpSignature(b []byte) string {
	h := sha1.Sum(b)
	return hex.EncodeToString(h[:])
}

func stubRecord(origin string, err error) CommandRecord {
	desc := "Help invocation returned no output"
	if err != nil {
		desc = "Help invocation failed: " + err.Error()
	}
	return CommandRecord{
		Origin:      origin,
		Name:        origin,
		FullPath:    origin,
		Source:      SourceHelpFailed,
		Description: desc,
	}
}

func rootLeafRecord(origin string, helpOut []byte) CommandRecord {
	return CommandRecord{
		Origin:      origin,
		Name:        origin,
		FullPath:    origin,
		Source:      SourceHelp,
		Description: helpDescription(helpOut),
	}
}

func leafRecord(origin string, parents []string, groupDesc string, helpOut []byte) CommandRecord {
	rec := newRecord(origin, parents)
	rec.Description = helpDescription(helpOut)
	rec.GroupDescription = groupContext(groupDesc, rec.Description)
	return rec
}

func leafRecordFromEntry(origin string, parents []string, groupDesc string, e helpEntry) CommandRecord {
	path := append(append([]string{}, parents...), e.Name)
	rec := newRecord(origin, path)
	rec.Description = e.Description
	rec.GroupDescription = groupContext(groupDesc, rec.Description)
	return rec
}

// groupContext returns the parent-group summary to attach to a leaf, dropped
// when empty or when it merely repeats the leaf's own description (no signal
// gained, avoids duplicating text in the embedding).
func groupContext(groupDesc, leafDesc string) string {
	g := strings.TrimSpace(groupDesc)
	if g == "" || g == strings.TrimSpace(leafDesc) {
		return ""
	}
	return g
}

func newRecord(origin string, path []string) CommandRecord {
	rec := CommandRecord{
		Origin: origin,
		Source: SourceHelp,
	}
	switch len(path) {
	case 0:
		rec.Name = origin
		rec.FullPath = origin
	case 1:
		rec.Name = path[0]
		rec.FullPath = origin + " " + path[0]
	default:
		rec.Group = path[0]
		rec.Name = path[len(path)-1]
		rec.FullPath = origin + " " + strings.Join(path, " ")
	}
	return rec
}

// helpDescription returns the human-facing prose of a leaf command's --help
// body: the summary line plus any description paragraphs, with the Usage block,
// flag/option/example sections, and bare flag lines stripped. Genuine leaf
// commands (reached when a subcommand's --help reveals no further subcommands)
// embed their FULL --help output; without this cleanup the scaffolding that
// every cli-core help shares — "Usage:" lines and the ubiquitous
// "--help, -h  Show help for this command" flag — lands in every leaf's vector,
// diluting the real per-command vocabulary and turning generic queries like
// "show help" into a magnet that matches arbitrary commands. Stripping noise
// (not adding synthetic text — the inverse of the enriched-text experiment that
// HURT recall) keeps embeddings anchored to what each command actually does.
func helpDescription(helpOut []byte) string {
	return truncateForEmbedding(cleanHelpBody(string(helpOut)), 1800)
}

// cleanHelpBody removes Usage blocks, flag/option/example sections, and bare
// flag lines from a raw --help body, leaving the prose. It walks section by
// section (reusing the same header / skip-section classification as the
// subcommand parser) and drops any line that is a flag definition wherever it
// appears, since some CLIs list flags without a section header. Falls back to
// the trimmed raw body when cleaning would empty it (a command whose help is
// pure scaffolding still needs *some* description).
func cleanHelpBody(raw string) string {
	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	skipSection := false
	for _, l := range lines {
		line := strings.TrimRight(l, "\r")
		trimmed := strings.TrimSpace(line)
		// A section header toggles skip state and is itself dropped (scaffolding).
		if m := sectionHeaderRE.FindStringSubmatch(line); m != nil {
			skipSection = isSkipSection(m[1])
			continue
		}
		if trimmed == "" {
			// A blank line ends an implicit usage/flag run; collapse repeats.
			skipSection = false
			if len(out) > 0 && out[len(out)-1] != "" {
				out = append(out, "")
			}
			continue
		}
		if skipSection {
			continue
		}
		// Bare flag definition (e.g. "--help, -h  Show help for this command").
		if strings.HasPrefix(trimmed, "-") {
			continue
		}
		out = append(out, trimmed)
	}
	cleaned := strings.TrimSpace(strings.Join(out, "\n"))
	if cleaned == "" {
		return strings.TrimSpace(raw)
	}
	return cleaned
}
