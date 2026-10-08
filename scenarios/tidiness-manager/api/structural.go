package main

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// logicTokenRe matches control-flow and branching tokens that indicate a block
// contains real logic rather than declarative data. String literals and
// comments are stripped before this is applied, so English prose in a
// descriptor's Description field (e.g. "run if ready") does not count as logic.
var logicTokenRe = regexp.MustCompile(`\b(if|else|for|switch|select|case|go|defer|range|func)\b|&&|\|\||==|!=|<=|>=`)

// stringLiteralRe matches double-quoted, single-quoted, and backtick string
// literals (non-greedy) so their contents are removed before token scanning.
var stringLiteralRe = regexp.MustCompile("\"[^\"]*\"|'[^']*'|`[^`]*`")

// IsStructuralBlock reports whether a block of source lines is overwhelmingly
// declarative data (struct literals, field assignments, string/const literals,
// list elements) with near-zero branching, as opposed to real logic.
//
// It is deterministic: each line is stripped of string-literal contents and
// line comments, then scanned for control-flow tokens. A block is structural
// only when the fraction of code lines carrying a logic token is at or below
// structuralLogicRatio. The default is conservative — when a block is ambiguous
// or carries meaningful branching, it is treated as logic (returns false) so a
// genuine logic block in a role-named file is never masked.
func IsStructuralBlock(lines []string) bool {
	const structuralLogicRatio = 0.15

	codeLines := 0
	logicLines := 0
	for _, raw := range lines {
		stripped := stripLiteralsAndComments(raw)
		if strings.TrimSpace(stripped) == "" {
			continue
		}
		codeLines++
		if logicTokenRe.MatchString(stripped) {
			logicLines++
		}
	}

	// Too little signal to confirm structure — default to logic (do not cap).
	if codeLines < 3 {
		return false
	}
	return float64(logicLines)/float64(codeLines) <= structuralLogicRatio
}

// stripLiteralsAndComments removes string-literal contents and line comments so
// that prose and identifiers inside data do not register as control flow.
func stripLiteralsAndComments(line string) string {
	line = stringLiteralRe.ReplaceAllString(line, `""`)
	// Drop Go/TS line comments and Python comments.
	if idx := strings.Index(line, "//"); idx >= 0 {
		line = line[:idx]
	}
	if idx := strings.Index(line, "#"); idx >= 0 {
		line = line[:idx]
	}
	return line
}

// readBlockLines reads numLines source lines from absPath starting at startLine
// (1-based). It returns whatever lines are available, even if the file is
// shorter than startLine+numLines, and an error only when the file cannot be
// opened.
func readBlockLines(absPath string, startLine, numLines int) ([]string, error) {
	file, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if startLine < 1 {
		startLine = 1
	}
	end := startLine + numLines

	out := make([]string, 0, numLines)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		if lineNo < startLine {
			continue
		}
		if lineNo >= end {
			break
		}
		out = append(out, scanner.Text())
	}
	return out, scanner.Err()
}
