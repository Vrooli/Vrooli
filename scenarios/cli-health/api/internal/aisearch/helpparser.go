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

	records := walkHelpTree(ctx, run, bin, origin, nil, rootOut, 1, maxDepth, seen)
	if len(records) == 0 {
		// Root had no parseable subcommands — treat the CLI itself as a leaf.
		return []CommandRecord{rootLeafRecord(origin, rootOut)}
	}
	return records
}

func walkHelpTree(ctx context.Context, run helpRunner, bin, origin string, parents []string, helpOut []byte, depth, maxDepth int, seen map[string]struct{}) []CommandRecord {
	entries := parseHelpEntries(helpOut)
	if len(entries) == 0 {
		return []CommandRecord{leafRecord(origin, parents, helpOut)}
	}
	if depth >= maxDepth {
		out := make([]CommandRecord, 0, len(entries))
		for _, e := range entries {
			out = append(out, leafRecordFromEntry(origin, parents, e))
		}
		return out
	}

	out := make([]CommandRecord, 0, len(entries))
	for _, e := range entries {
		childPath := append(append([]string{}, parents...), e.Name)
		childOut, err := run(ctx, bin, childPath)
		if err != nil || strings.TrimSpace(string(childOut)) == "" {
			out = append(out, leafRecordFromEntry(origin, parents, e))
			continue
		}
		sig := helpSignature(childOut)
		if _, dup := seen[sig]; dup {
			// Same help text as an ancestor — likely a leaf whose --help echoes
			// the parent. Treat as leaf.
			out = append(out, leafRecordFromEntry(origin, parents, e))
			continue
		}
		seen[sig] = struct{}{}

		childRecords := walkHelpTree(ctx, run, bin, origin, childPath, childOut, depth+1, maxDepth, seen)
		if len(childRecords) == 0 {
			out = append(out, leafRecordFromEntry(origin, parents, e))
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
	// We also accept lines without a description (visual category labels) so the
	// caller can filter them out.
	commandLineRE = regexp.MustCompile(`^(\s+)(\S+)(?:\s{2,}(.+))?\s*$`)

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

func leafRecord(origin string, parents []string, helpOut []byte) CommandRecord {
	rec := newRecord(origin, parents)
	rec.Description = helpDescription(helpOut)
	return rec
}

func leafRecordFromEntry(origin string, parents []string, e helpEntry) CommandRecord {
	path := append(append([]string{}, parents...), e.Name)
	rec := newRecord(origin, path)
	rec.Description = e.Description
	return rec
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

// helpDescription returns the help body trimmed to start at its first non-empty
// line. The body already leads with the command's summary line, so it is used
// verbatim — prepending firstNonEmptyLine again (the old behavior) duplicated
// the summary in both the embedded vector and the displayed snippet.
func helpDescription(helpOut []byte) string {
	return truncateForEmbedding(strings.TrimLeft(string(helpOut), " \t\r\n"), 1800)
}
