package validation

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// excessiveSnapshotThreshold is the per-file snapshot count above which a UI
// test file is flagged as snapshot-heavy.
const excessiveSnapshotThreshold = 5

// analyzeQuality retains snapshot advice. Requirement reconciliation consumes
// the registry owner and typed adapter evidence through the validation service.
// Assertion and focused-test observations belong to registered adapters.
// Semantic edge-case adequacy requires behavioral review, not keywords.
func analyzeQuality(scenario, scenarioRoot string, workspaces []Workspace, now string) []Finding {
	var findings []Finding
	for _, ws := range workspaces {
		switch ws.Language {
		case "typescript":
			findings = append(findings, analyzeTSQuality(scenario, ws, now)...)
		}
	}
	return findings
}

func qualityFinding(scenario string, ws Workspace, code, file, message, evidence, expected, observed, why, remediation, now string) Finding {
	return Finding{
		ID:           code + "-" + ws.ID,
		Scenario:     scenario,
		WorkspaceID:  ws.ID,
		Language:     ws.Language,
		Code:         code,
		Category:     "quality",
		Severity:     codeSeverity[code],
		FilePath:     file,
		Message:      message,
		Evidence:     evidence,
		Expected:     expected,
		Observed:     observed,
		WhyItMatters: why,
		Remediation:  remediation,
		CreatedAt:    now,
	}
}

var (
	tsSnapshotRe   = regexp.MustCompile(`\btoMatch(Inline)?Snapshot\s*\(`)
	tsLineComment  = regexp.MustCompile(`//[^\n]*`)
	tsBlockComment = regexp.MustCompile(`(?s)/\*.*?\*/`)
)

func analyzeTSQuality(scenario string, ws Workspace, now string) []Finding {
	var snapshotHeavy []string

	walkSourceFiles(ws.RootPath, func(path string) {
		if !isTSTestFile(path) {
			return
		}
		raw := readFileString(path)
		if raw == "" {
			return
		}
		// Strip comments so an `expect`/`error` mentioned only in a comment is
		// not counted as a real assertion (B4).
		src := stripTSComments(raw)
		base := filepath.Base(path)
		if n := len(tsSnapshotRe.FindAllString(src, -1)); n > excessiveSnapshotThreshold {
			snapshotHeavy = append(snapshotHeavy, fmt.Sprintf("%s (%d)", base, n))
		}
	})

	var findings []Finding
	if len(snapshotHeavy) > 0 {
		sort.Strings(snapshotHeavy)
		findings = append(findings, qualityFinding(scenario, ws, codeTestExcessiveSnapshots, ws.RootPath,
			fmt.Sprintf("%d UI test file(s) rely heavily on snapshots.", len(snapshotHeavy)),
			"snapshot-heavy: "+strings.Join(truncateList(snapshotHeavy, 10), ", "),
			fmt.Sprintf("Targeted assertions rather than more than %d snapshots per file.", excessiveSnapshotThreshold),
			"snapshot-heavy files",
			"Large snapshot suites are brittle and rubber-stamped on update, hiding real regressions.",
			"Replace broad snapshots with targeted assertions on the meaningful output.",
			now))
	}
	return findings
}

// stripTSComments removes line and block comments so heuristics do not match
// tokens that appear only in commentary. It is intentionally simple (it does
// not parse string literals); that is acceptable for these advisory signals.
func stripTSComments(src string) string {
	src = tsBlockComment.ReplaceAllString(src, " ")
	src = tsLineComment.ReplaceAllString(src, " ")
	return src
}

func containsAny(haystack string, needles []string) bool {
	for _, n := range needles {
		if strings.Contains(haystack, n) {
			return true
		}
	}
	return false
}

func truncateList(items []string, max int) []string {
	if len(items) <= max {
		return items
	}
	out := append([]string{}, items[:max]...)
	return append(out, fmt.Sprintf("… (+%d more)", len(items)-max))
}
