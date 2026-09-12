package heartbeat

import "strings"

// maxMarkdownHeadingLevel is the deepest ATX heading Markdown defines. A shift
// that would exceed it clamps here rather than emitting `#######`, which
// renders as literal text instead of a heading.
const maxMarkdownHeadingLevel = 6

// shiftMarkdownHeadings re-levels an injected document so that its own headings
// nest underneath the prompt section that carries it.
//
// Every document the prompt injects — TEAM.md, RESPONSIBILITIES.md,
// HEARTBEAT.md, SOUL.md, AGENTS.md, TOOLS.md, a stored handoff, an inbox
// message — carries its own level-one title. Markdown has no section nesting,
// so that title terminates the prompt section it was meant to fill and opens a
// new top-level one. The measured effect was a `# Operating Policy` holding 87
// bytes while the 7268-byte team charter sat under a heading no prompt rule
// named, and a `# Heartbeat Task (HEARTBEAT.md)` holding nothing at all.
//
// The whole document shifts by one constant amount rather than only its
// level-one headings, because flattening `# Title` to `## Title` while leaving
// its `## Section` children at level two would turn children into siblings.
// Shifting preserves the document's internal shape and only changes where it
// hangs.
//
// The source files on disk are never rewritten. Demotion happens at injection
// time, so a document reads the same to a human opening it and nests correctly
// when a prompt embeds it.
func shiftMarkdownHeadings(body string, topLevel int) string {
	if strings.TrimSpace(body) == "" || topLevel < 1 {
		return body
	}
	lines := strings.Split(body, "\n")

	minLevel := 0
	forEachHeading(lines, func(_ int, level int, _ string) {
		if minLevel == 0 || level < minLevel {
			minLevel = level
		}
	})
	if minLevel == 0 || minLevel >= topLevel {
		return body
	}

	shift := topLevel - minLevel
	forEachHeading(lines, func(index int, level int, text string) {
		shifted := level + shift
		if shifted > maxMarkdownHeadingLevel {
			shifted = maxMarkdownHeadingLevel
		}
		lines[index] = strings.Repeat("#", shifted) + " " + text
	})
	return strings.Join(lines, "\n")
}

// forEachHeading visits every ATX heading outside a fenced code block. Fenced
// content is skipped because `# comment` is ordinary shell syntax, and the
// injected documents are full of shell examples that must survive verbatim.
func forEachHeading(lines []string, visit func(index int, level int, text string)) {
	fence := ""
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if marker := fenceMarker(trimmed); marker != "" {
			switch {
			case fence == "":
				fence = marker
			case strings.HasPrefix(marker, fence[:1]) && len(marker) >= len(fence):
				fence = ""
			}
			continue
		}
		if fence != "" {
			continue
		}
		// CommonMark allows up to three leading spaces before an ATX marker;
		// a fourth makes the line an indented code block instead.
		if len(line)-len(strings.TrimLeft(line, " ")) > 3 || strings.HasPrefix(line, "\t") {
			continue
		}
		level := 0
		for level < len(trimmed) && trimmed[level] == '#' {
			level++
		}
		if level == 0 || level > maxMarkdownHeadingLevel {
			continue
		}
		rest := trimmed[level:]
		if rest != "" && rest[0] != ' ' && rest[0] != '\t' {
			continue
		}
		visit(i, level, strings.TrimSpace(rest))
	}
}

// fenceMarker returns the backtick or tilde run that opens or closes a fenced
// code block, or empty when the line is not a fence.
func fenceMarker(trimmed string) string {
	for _, char := range []byte{'`', '~'} {
		run := 0
		for run < len(trimmed) && trimmed[run] == char {
			run++
		}
		if run >= 3 {
			return trimmed[:run]
		}
	}
	return ""
}
