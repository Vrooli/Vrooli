package validation

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"unit-health/internal/testquality"
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

// rollupFindings projects typed per-test violations into the scenario-level
// findings consumed by the maturity assessor. The typed result remains the
// source of truth; unknown and not-applicable observations never become a
// rollup. One bounded finding per workspace keeps the scenario response useful
// without losing the individual identities in the evidence.
func rollupFindings(scenario string, workspaces []Workspace, results []testquality.Result, now string) []Finding {
	type violation struct {
		identity string
		rule     string
	}
	type rollupKey struct {
		workspace string
		code      string
	}
	byRollup := map[rollupKey][]violation{}
	for _, result := range results {
		if testquality.NormalizeStatus(result.Status) != testquality.Violation {
			continue
		}
		code := ""
		switch result.RuleID {
		case "focused-test", "skip-declaration":
			code = codeTestSkippedOrOnly
		case "requirement-link":
			code = codeTestUntaggedRequirement
		case "assertion-observation":
			if result.Enforcement != testquality.Blocking {
				continue
			}
			code = codeTestQualityViolation
		default:
			continue
		}
		identity := result.Target.File
		if result.Target.TestID != "" {
			identity += "#" + result.Target.TestID
		}
		if identity == "" {
			identity = result.Target.Workspace
		}
		key := rollupKey{workspace: result.Target.Workspace, code: code}
		byRollup[key] = append(byRollup[key], violation{identity: identity, rule: result.RuleID})
	}

	var findings []Finding
	for _, ws := range workspaces {
		for _, code := range []string{codeTestSkippedOrOnly, codeTestUntaggedRequirement, codeTestQualityViolation} {
			violations := byRollup[rollupKey{workspace: ws.ID, code: code}]
			if len(violations) == 0 {
				continue
			}
			// Results from one workspace can contain multiple rule observations. Keep
			// evidence deterministic and bounded, while retaining the rule name for
			// each identity so remediation is actionable.
			sort.SliceStable(violations, func(i, j int) bool {
				if violations[i].identity != violations[j].identity {
					return violations[i].identity < violations[j].identity
				}
				return violations[i].rule < violations[j].rule
			})
			items := make([]string, 0, minInt(len(violations), 10))
			for _, item := range violations[:minInt(len(violations), 10)] {
				items = append(items, item.identity+" ("+item.rule+")")
			}
			message := "Typed test-quality violations were found in this workspace."
			remediation := "Inspect the typed test-quality results and repair the reported rule violations."
			if code == codeTestSkippedOrOnly {
				message = "Focused or skipped test declarations were found in this workspace."
				remediation = "Remove focused declarations and replace unconditional skips with executable assertions, or document a supported exception."
			} else if code == codeTestUntaggedRequirement {
				message = "Tests without a registered requirement link were found in this workspace."
				remediation = "Add an exact requirement tag and a matching registry validation responsibility to each reported test."
			} else {
				message = "An owner-approved blocking test-quality rule violation was found in this workspace."
				remediation = "Repair the reported test-quality violation, then rerun the unit phase under the approved rule profile."
			}
			findings = append(findings, Finding{
				ID:           code + "-" + ws.ID,
				Scenario:     scenario,
				WorkspaceID:  ws.ID,
				Language:     ws.Language,
				Framework:    ws.Framework,
				Code:         code,
				Category:     "quality",
				Severity:     codeSeverity[code],
				FilePath:     ws.RootPath,
				Message:      message,
				Evidence:     "violating tests: " + strings.Join(items, ", "),
				Expected:     "No typed violations for the rolled-up rule.",
				Observed:     fmt.Sprintf("%d typed violation(s)", len(violations)),
				WhyItMatters: "A scenario-level summary lets maturity consumers act on typed quality evidence without losing its per-test scope.",
				Remediation:  remediation,
				CreatedAt:    now,
			})
		}
	}
	return findings
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
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
