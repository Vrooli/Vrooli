package templatecontracts

import (
	"fmt"
	"regexp"
	"strings"
)

// Detemplate removes a template's illustrative example domain (the `notes`
// worked vertical slice) from a generated scenario. The example domain is
// marked with a single vocabulary, `EXAMPLE-DOMAIN:<marker>`, placed three
// ways — fenced doc blocks, whole files/dirs (manifest-listed), and trailing
// registration-line comments. This file owns the pure, dependency-free text
// transforms for the fenced-block and registration-line forms; the
// orchestration (loading the scenario, resolving the template manifest,
// deleting paths, running finalizers) lives in the scenariohandlers package.

// exampleDomainMarkerToken returns the literal marker token for a domain
// marker, e.g. "EXAMPLE-DOMAIN:notes".
func exampleDomainMarkerToken(marker string) string {
	return "EXAMPLE-DOMAIN:" + marker
}

// blockFenceRegexp builds a regexp that matches one complete fenced example
// block for the given marker, inclusive of both comment sentinels and any
// trailing newline. The block is matched non-greedily so multiple blocks in
// one file are stripped independently. The fence is comment-syntax-agnostic so
// the same vocabulary fences Markdown prose and multi-line code blocks alike:
//
//	<!-- EXAMPLE-DOMAIN:<marker> START [optional words] -->   (Markdown)
//	... content ...
//	<!-- EXAMPLE-DOMAIN:<marker> END -->
//
//	// EXAMPLE-DOMAIN:<marker> START                          (Go / TS)
//	... code ...
//	// EXAMPLE-DOMAIN:<marker> END
//
// A START line is any line opening with a comment punctuator (<!--, //, or #)
// followed by `EXAMPLE-DOMAIN:<marker> START`; the block runs to the next line
// carrying `EXAMPLE-DOMAIN:<marker> END`. The rest of the END line (e.g. a
// trailing `-->`) is consumed, plus the END line's newline and a single
// following blank line if present (so a paragraph-separated fence collapses to
// one blank line rather than two).
func blockFenceRegexp(marker string) *regexp.Regexp {
	tok := regexp.QuoteMeta(exampleDomainMarkerToken(marker))
	pattern := `(?s)[ \t]*(?:<!--|//|#)[ \t]*` + tok + `[ \t]+START\b.*?` + tok + `[ \t]+END\b[^\n]*(?:\n[ \t]*\n|\n)?`
	return regexp.MustCompile(pattern)
}

// lineMarkerRegexp matches a single source line that carries a trailing
// EXAMPLE-DOMAIN:<marker> comment marker (any comment syntax). The marker
// token must be followed by a word boundary so `:notes` does not match
// `:notes-archive`.
func lineMarkerRegexp(marker string) *regexp.Regexp {
	tok := regexp.QuoteMeta(exampleDomainMarkerToken(marker))
	return regexp.MustCompile(tok + `(?:[^A-Za-z0-9_-]|$)`)
}

// StripExampleDomainBlocks removes every fenced EXAMPLE-DOMAIN:<marker> block
// from content and returns the rewritten content plus the number of blocks
// removed. It is a no-op (removed == 0) when no fenced block is present.
func StripExampleDomainBlocks(content []byte, marker string) ([]byte, int) {
	re := blockFenceRegexp(marker)
	matches := re.FindAll(content, -1)
	if len(matches) == 0 {
		return content, 0
	}
	return re.ReplaceAll(content, nil), len(matches)
}

// StripExampleDomainLines removes every line that carries a trailing
// EXAMPLE-DOMAIN:<marker> registration marker and returns the rewritten
// content plus the number of lines removed. Call StripExampleDomainBlocks
// first so fenced-block sentinel lines are already gone; any remaining marked
// line is a registration line (import, route, nav item, schema/CLI/selector
// entry) that should be deleted whole.
func StripExampleDomainLines(content []byte, marker string) ([]byte, int) {
	re := lineMarkerRegexp(marker)
	if !re.Match(content) {
		return content, 0
	}
	hadTrailingNewline := len(content) > 0 && content[len(content)-1] == '\n'
	lines := strings.Split(string(content), "\n")
	kept := make([]string, 0, len(lines))
	removed := 0
	for i, line := range lines {
		// strings.Split yields a trailing empty element when content ends
		// with "\n"; preserve that so the trailing newline is reconstructed
		// faithfully rather than counted as a removable line.
		if i == len(lines)-1 && line == "" && hadTrailingNewline {
			kept = append(kept, line)
			continue
		}
		if re.MatchString(line) {
			removed++
			continue
		}
		kept = append(kept, line)
	}
	if removed == 0 {
		return content, 0
	}
	return []byte(strings.Join(kept, "\n")), removed
}

// FileMarkerSummary records what a single file contributed to a detemplate
// pass: how many fenced blocks and how many registration lines were removed.
type FileMarkerSummary struct {
	Path          string `json:"path"`
	BlocksRemoved int    `json:"blocksRemoved,omitempty"`
	LinesStripped int    `json:"linesStripped,omitempty"`
}

// StripExampleDomainFile applies both transforms (blocks then lines) to file
// content and returns the rewritten bytes plus a summary. changed reports
// whether anything was removed.
func StripExampleDomainFile(path string, content []byte, marker string) (out []byte, summary FileMarkerSummary, changed bool) {
	out, blocks := StripExampleDomainBlocks(content, marker)
	out, lines := StripExampleDomainLines(out, marker)
	summary = FileMarkerSummary{Path: path, BlocksRemoved: blocks, LinesStripped: lines}
	return out, summary, blocks > 0 || lines > 0
}

// ContainsExampleDomainMarker reports whether content carries any
// EXAMPLE-DOMAIN:<marker> marker in any of its three forms. Used by the
// residue gate to assert a detemplated scenario is clean.
func ContainsExampleDomainMarker(content []byte, marker string) bool {
	return lineMarkerRegexp(marker).Match(content)
}

// DetemplateFinalizer records one post-strip finalizer command and its
// outcome.
type DetemplateFinalizer struct {
	Description string `json:"description"`
	Command     string `json:"command"`
	Ran         bool   `json:"ran"`
	OK          bool   `json:"ok"`
	Message     string `json:"message,omitempty"`
}

// DetemplateRequest selects a generated scenario to strip.
type DetemplateRequest struct {
	Name   string
	DryRun bool
	JSON   bool
}

// DetemplateResult is the outcome of a `template-manager detemplate` run.
type DetemplateResult struct {
	Scenario      string                `json:"scenario"`
	ScenarioPath  string                `json:"scenarioPath,omitempty"`
	Marker        string                `json:"marker,omitempty"`
	DryRun        bool                  `json:"dryRun,omitempty"`
	BlocksRemoved int                   `json:"blocksRemoved"`
	LinesStripped int                   `json:"linesStripped"`
	FilesEdited   []FileMarkerSummary   `json:"filesEdited,omitempty"`
	PathsDeleted  []string              `json:"pathsDeleted,omitempty"`
	Finalizers    []DetemplateFinalizer `json:"finalizers,omitempty"`
	Message       string                `json:"message,omitempty"`
}

// DetemplateDanglingRefError is returned when a kept (non-example) file still
// references an example-domain path that detemplate would delete. Detemplate
// refuses rather than leave a non-building scenario; the error names the
// blocking references so the author can resolve them.
type DetemplateDanglingRefError struct {
	Marker     string
	References []DetemplateDanglingRef
}

// DetemplateDanglingRef is one kept-file reference to to-be-deleted example
// code.
type DetemplateDanglingRef struct {
	File      string `json:"file"`
	Reference string `json:"reference"`
}

func (e *DetemplateDanglingRefError) Error() string {
	refs := make([]string, 0, len(e.References))
	for _, r := range e.References {
		refs = append(refs, fmt.Sprintf("%s -> %s", r.File, r.Reference))
	}
	return fmt.Sprintf("refusing to detemplate: %d non-example file(s) still reference the %q example domain after marker removal:\n  %s",
		len(e.References), e.Marker, strings.Join(refs, "\n  "))
}
