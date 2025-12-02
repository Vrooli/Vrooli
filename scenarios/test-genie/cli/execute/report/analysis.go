package report

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	execTypes "test-genie/cli/internal/execute"
)

// FailureDiagnosis contains the root cause analysis for failed phases.
type FailureDiagnosis struct {
	Primary         string
	PrimaryPhase    string
	Details         string
	ImpactedPhases  []string
	SecondaryIssues []string
	QuickFixes      []string
}

// PhaseInsight contains analysis for a single failed phase.
type PhaseInsight struct {
	Phase        string
	Cause        string
	Detail       string
	Impact       []string
	Fixes        []string
	Log          string
	Observations []string
}

// CriticalPath identifies the primary blocker and downstream effects.
type CriticalPath struct {
	PrimaryPhase  string
	PrimaryCause  string
	Detail        string
	BlockedPhases []string
	QuickFixes    []string
}

// failurePatterns defines regex patterns for common failure types.
var failurePatterns = map[string]*regexp.Regexp{
	"typescript": regexp.MustCompile(`(?i)typescript|error TS[0-9]+|Cannot find name`),
	"ui_bundle":  regexp.MustCompile(`(?i)bundle.*stale|UI bundle.*outdated|Bundle Status:.*stale`),
	"api_port":   regexp.MustCompile(`(?i)API_PORT|HTTP 502|Bad Gateway|not responding`),
	"build":      regexp.MustCompile(`(?i)build .*failed|compilation .*failed|ELIFECYCLE`),
	"timeout":    regexp.MustCompile(`(?i)timed out|timeout|exceeded .*time`),
	"tests":      regexp.MustCompile(`(?i)❌|test failed|tests failed`),
	"missing":    regexp.MustCompile(`(?i)required (file|directory) missing|no such file or directory|expected file but found directory|expected directory but found file`),
}

// AnalyzePhaseFailures generates insights for all failed phases.
func AnalyzePhaseFailures(phases []execTypes.Phase) []PhaseInsight {
	var insights []PhaseInsight
	for _, phase := range phases {
		content := ReadLogSnippet(phase.LogPath, 48_000)
		insight := PhaseInsight{
			Phase:        phase.Name,
			Log:          DescribeLogPath(phase.LogPath),
			Observations: CleanObservations(phase.Observations),
		}

		switch {
		case failurePatterns["typescript"].MatchString(content):
			insight.Cause = "TypeScript compilation error"
			insight.Detail = extractLine(content, failurePatterns["typescript"])
			insight.Impact = []string{"integration", "performance"}
			insight.Fixes = append(insight.Fixes,
				"Fix TypeScript compiler errors (see log)",
				"Rebuild UI and rerun: test-genie execute <scenario>")
		case failurePatterns["ui_bundle"].MatchString(content):
			insight.Cause = "Stale or missing UI bundle"
			insight.Detail = extractLine(content, failurePatterns["ui_bundle"])
			insight.Impact = []string{"integration", "performance"}
			insight.Fixes = append(insight.Fixes,
				"Rebuild the UI bundle",
				"Restart the scenario then rerun execute")
		case failurePatterns["api_port"].MatchString(content):
			insight.Cause = "Scenario API unreachable"
			insight.Detail = extractLine(content, failurePatterns["api_port"])
			insight.Impact = []string{"integration"}
			insight.Fixes = append(insight.Fixes,
				"Restart the scenario: vrooli scenario restart <scenario>",
				"Verify API_PORT resolution and retry")
		case failurePatterns["build"].MatchString(content):
			insight.Cause = "Build failure"
			insight.Detail = extractLine(content, failurePatterns["build"])
			insight.Fixes = append(insight.Fixes, "Fix build errors shown in the log, then rerun execute")
		case failurePatterns["timeout"].MatchString(content):
			insight.Cause = "Phase timeout"
			insight.Detail = extractLine(content, failurePatterns["timeout"])
			insight.Fixes = append(insight.Fixes, "Inspect long-running task in the log, rerun with --fail-fast to isolate")
		case failurePatterns["tests"].MatchString(content):
			insight.Cause = "Tests failed"
			insight.Detail = extractLine(content, failurePatterns["tests"])
			insight.Fixes = append(insight.Fixes, "Open the phase log and fix failing tests, then rerun execute")
		case failurePatterns["missing"].MatchString(content):
			insight.Cause = "Required file/directory missing"
			insight.Detail = extractLine(content, failurePatterns["missing"])
			insight.Fixes = append(insight.Fixes,
				"Restore the missing path referenced in the error",
				"Update .vrooli/testing.json structure expectations if the layout is intentionally different")
		default:
			if failurePatterns["missing"].MatchString(phase.Error) {
				insight.Cause = "Required file/directory missing"
				insight.Detail = phase.Error
				insight.Fixes = append(insight.Fixes,
					"Restore the missing path referenced in the error",
					"Update .vrooli/testing.json structure expectations if the layout is intentionally different")
				insights = append(insights, insight)
				continue
			}
			if phase.Error != "" {
				insight.Cause = phase.Error
			} else if phase.Classification != "" {
				insight.Cause = phase.Classification
			} else {
				insight.Cause = "Unknown failure"
			}
		}

		insights = append(insights, insight)
	}

	return insights
}

// SummarizeCriticalPath identifies the primary blocker from phase insights.
func SummarizeCriticalPath(insights []PhaseInsight) *CriticalPath {
	if len(insights) == 0 {
		return nil
	}
	primary := insights[0]
	for _, in := range insights {
		if len(in.Impact) > 0 && in.Cause != "" {
			primary = in
			break
		}
	}

	blockedSet := make(map[string]struct{})
	for _, phase := range primary.Impact {
		blockedSet[phase] = struct{}{}
	}
	var blocked []string
	for phase := range blockedSet {
		blocked = append(blocked, phase)
	}
	sort.Strings(blocked)

	seenFix := make(map[string]struct{})
	var fixes []string
	for _, in := range insights {
		for _, fix := range in.Fixes {
			fix = strings.TrimSpace(fix)
			if fix == "" {
				continue
			}
			key := strings.ToLower(fix)
			if _, ok := seenFix[key]; ok {
				continue
			}
			seenFix[key] = struct{}{}
			fixes = append(fixes, fix)
		}
	}

	return &CriticalPath{
		PrimaryPhase:  primary.Phase,
		PrimaryCause:  DefaultValue(primary.Cause, "Unknown issue"),
		Detail:        primary.Detail,
		BlockedPhases: blocked,
		QuickFixes:    fixes,
	}
}

// DiagnoseFailures performs root cause analysis on failed phases.
func DiagnoseFailures(phases []execTypes.Phase) *FailureDiagnosis {
	failed := FilterFailedPhases(phases)
	if len(failed) == 0 {
		return nil
	}

	diag := &FailureDiagnosis{}

	for _, phase := range failed {
		content := ReadLogSnippet(phase.LogPath, 48_000)
		switch {
		case diag.Primary == "" && failurePatterns["typescript"].MatchString(content):
			diag.Primary = "TypeScript compilation error"
			diag.PrimaryPhase = phase.Name
			diag.Details = extractLine(content, failurePatterns["typescript"])
			diag.ImpactedPhases = append(diag.ImpactedPhases, "integration", "performance")
			diag.QuickFixes = append(diag.QuickFixes,
				"Fix TypeScript compiler errors (see log snippet)",
				"Rebuild/restart scenario, then re-run: test-genie execute <scenario>")
		case diag.Primary == "" && failurePatterns["ui_bundle"].MatchString(content):
			diag.Primary = "Stale or missing UI bundle"
			diag.PrimaryPhase = phase.Name
			diag.Details = extractLine(content, failurePatterns["ui_bundle"])
			diag.ImpactedPhases = append(diag.ImpactedPhases, "integration", "performance")
			diag.QuickFixes = append(diag.QuickFixes,
				"Rebuild the UI bundle and restart the scenario",
				"Re-run: test-genie execute <scenario> --preset quick")
		case diag.Primary == "" && failurePatterns["api_port"].MatchString(content):
			diag.Primary = "Scenario API unreachable"
			diag.PrimaryPhase = phase.Name
			diag.Details = extractLine(content, failurePatterns["api_port"])
			diag.ImpactedPhases = append(diag.ImpactedPhases, "integration")
			diag.QuickFixes = append(diag.QuickFixes,
				"Restart the scenario: vrooli scenario restart <scenario>",
				"Verify API_PORT resolution and retry execute")
		case diag.Primary == "" && failurePatterns["build"].MatchString(content):
			diag.Primary = "Build failure"
			diag.PrimaryPhase = phase.Name
			diag.Details = extractLine(content, failurePatterns["build"])
			diag.QuickFixes = append(diag.QuickFixes,
				"Fix build errors shown in the log and rerun execute")
		case diag.Primary == "" && failurePatterns["timeout"].MatchString(content):
			diag.Primary = "Phase timeout"
			diag.PrimaryPhase = phase.Name
			diag.Details = extractLine(content, failurePatterns["timeout"])
			diag.QuickFixes = append(diag.QuickFixes,
				"Investigate long-running tasks in the phase log, rerun with --fail-fast to isolate")
		case diag.Primary == "" && (failurePatterns["missing"].MatchString(content) || failurePatterns["missing"].MatchString(phase.Error)):
			diag.Primary = "Required file/directory missing"
			diag.PrimaryPhase = phase.Name
			diag.Details = extractLine(content, failurePatterns["missing"])
			if diag.Details == "" {
				diag.Details = strings.TrimSpace(phase.Error)
			}
			diag.QuickFixes = append(diag.QuickFixes,
				"Restore the missing path referenced in the log",
				"Update .vrooli/testing.json structure expectations if the layout is intentionally different")
		}

		if failurePatterns["tests"].MatchString(content) {
			diag.SecondaryIssues = append(diag.SecondaryIssues, fmt.Sprintf("%s: test failures detected", phase.Name))
			if len(diag.QuickFixes) == 0 {
				diag.QuickFixes = append(diag.QuickFixes, "Open the phase log and address failing tests, then rerun execute")
			}
		}
	}

	// Deduplicate quick fixes
	seenFix := make(map[string]struct{})
	var fixes []string
	for _, fix := range diag.QuickFixes {
		key := strings.TrimSpace(strings.ToLower(fix))
		if key == "" {
			continue
		}
		if _, ok := seenFix[key]; ok {
			continue
		}
		seenFix[key] = struct{}{}
		fixes = append(fixes, fix)
	}
	diag.QuickFixes = fixes
	return diag
}

func extractLine(content string, re *regexp.Regexp) string {
	if re == nil {
		return ""
	}
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if re.MatchString(line) {
			return strings.TrimSpace(line)
		}
	}
	return ""
}
