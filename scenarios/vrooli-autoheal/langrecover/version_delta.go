package langrecover

import (
	"sort"
	"strings"
)

// VersionDelta records one module whose selected version changed as a result
// of a recovery command.
//
// Why this exists: `go mod tidy` re-runs minimal version selection, so healing
// a missing go.sum entry can silently bump an unrelated direct dependency to a
// higher version required by a shared package. Observed 2026-09-01: syncing
// scenario modules after an api-core change bumped app-monitor's direct
// gorilla/websocket from v1.5.1 to v1.5.3. That is correct MVS behaviour, but
// an unattended healer must not perform it invisibly -- the operator needs the
// delta in the incident record, not just "healed".
type VersionDelta struct {
	// Module is the module path (e.g. "github.com/gorilla/websocket").
	Module string
	// From is the version before recovery; empty when the module was added.
	From string
	// To is the version after recovery; empty when the module was dropped.
	To string
}

// Added reports whether the module was newly introduced by the recovery.
func (d VersionDelta) Added() bool { return d.From == "" && d.To != "" }

// Removed reports whether the module was dropped by the recovery.
func (d VersionDelta) Removed() bool { return d.From != "" && d.To == "" }

// Changed reports whether an already-present module moved to a new version.
// This is the case that matters most for review: it alters what actually gets
// compiled, unlike an added indirect entry that merely records an existing
// transitive requirement.
func (d VersionDelta) Changed() bool { return d.From != "" && d.To != "" && d.From != d.To }

// String renders the delta compactly for incident output.
func (d VersionDelta) String() string {
	switch {
	case d.Added():
		return d.Module + " +" + d.To
	case d.Removed():
		return d.Module + " -" + d.From
	default:
		return d.Module + " " + d.From + " -> " + d.To
	}
}

// diffGoModVersions compares two go.mod bodies and returns the set of modules
// whose recorded version changed, sorted by module path for stable output.
//
// The parse is deliberately tolerant: it reads "<module> <version>" pairs from
// require blocks and single-line require directives, ignoring comments and the
// "// indirect" marker. It is not a full go.mod parser and does not need to be
// -- callers use the result for reporting, never for rewriting the file.
func diffGoModVersions(before, after string) []VersionDelta {
	pre := parseGoModRequires(before)
	post := parseGoModRequires(after)

	seen := map[string]struct{}{}
	var deltas []VersionDelta
	for module, fromVersion := range pre {
		seen[module] = struct{}{}
		toVersion := post[module]
		if fromVersion != toVersion {
			deltas = append(deltas, VersionDelta{Module: module, From: fromVersion, To: toVersion})
		}
	}
	for module, toVersion := range post {
		if _, ok := seen[module]; ok {
			continue
		}
		deltas = append(deltas, VersionDelta{Module: module, To: toVersion})
	}

	sort.Slice(deltas, func(i, j int) bool { return deltas[i].Module < deltas[j].Module })
	return deltas
}

// parseGoModRequires extracts module -> version pairs from a go.mod body.
func parseGoModRequires(body string) map[string]string {
	requires := map[string]string{}
	inBlock := false
	for _, rawLine := range strings.Split(body, "\n") {
		line := strings.TrimSpace(rawLine)
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if line == "" {
			continue
		}
		switch {
		case line == "require (":
			inBlock = true
			continue
		case inBlock && line == ")":
			inBlock = false
			continue
		case strings.HasPrefix(line, "require "):
			// Single-line form: `require example.com/mod v1.2.3`.
			if module, version, ok := splitModuleVersion(strings.TrimPrefix(line, "require ")); ok {
				requires[module] = version
			}
			continue
		}
		if !inBlock {
			continue
		}
		if module, version, ok := splitModuleVersion(line); ok {
			requires[module] = version
		}
	}
	return requires
}

// splitModuleVersion splits a "<module> <version>" pair, rejecting anything
// that does not look like a versioned requirement.
func splitModuleVersion(line string) (string, string, bool) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return "", "", false
	}
	module, version := fields[0], fields[1]
	if module == "" || !strings.HasPrefix(version, "v") {
		return "", "", false
	}
	return module, version, true
}

// ChangedVersionDeltas returns only the deltas that moved an already-present
// module to a different version. These are the ones worth escalating: added
// indirect entries are usually just the missing bookkeeping being restored.
func ChangedVersionDeltas(deltas []VersionDelta) []VersionDelta {
	var changed []VersionDelta
	for _, d := range deltas {
		if d.Changed() {
			changed = append(changed, d)
		}
	}
	return changed
}

// FormatVersionDeltas renders deltas as a single comma-separated line.
func FormatVersionDeltas(deltas []VersionDelta) string {
	if len(deltas) == 0 {
		return ""
	}
	parts := make([]string, 0, len(deltas))
	for _, d := range deltas {
		parts = append(parts, d.String())
	}
	return strings.Join(parts, ", ")
}
