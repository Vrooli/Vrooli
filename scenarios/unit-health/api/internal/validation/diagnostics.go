package validation

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"unit-health/internal/executor"
)

// runtimePressureRatio flags a command whose duration reaches this fraction of
// its timeout: the suite is approaching the bound and likely to start timing out.
const runtimePressureRatio = 0.75

// flakeMarkerRe matches deliberate source signals that tests self-identify as
// flaky. Detecting flake without reruns is best-effort, so Unit Health only
// reports the strongest intentional markers (the word "flaky"/"flake", a
// flaky-gated skip, or a flaky-suppressing lint directive) rather than broad
// words like "retry" or "eventually" that appear in healthy async tests.
var flakeMarkerRe = regexp.MustCompile(`(?i)\bflak(?:e|y)\b`)

// analyzeDiagnostics derives flake/runtime/hang diagnostics. Flake markers are
// static; runtime-pressure and hang diagnostics come from the executed
// commands (results is empty when execution was not requested).
func analyzeDiagnostics(scenario string, workspaces []Workspace, plan ExecutionPlan, results []CommandResult, now string) ([]Diagnostic, []Finding) {
	var diagnostics []Diagnostic
	var findings []Finding

	for _, r := range results {
		if r.FailureClass == executor.ClassTimeoutHang || r.FailureClass == executor.ClassNoOutputStall {
			// The blocking TEST_TIMEOUT_HANG finding is emitted by the executor
			// path; the diagnostic adds the likely-culprit log tail.
			diagnostics = append(diagnostics, Diagnostic{
				Kind:        "hang",
				WorkspaceID: workspaceForCommand(plan, r),
				Message:     fmt.Sprintf("Command %q did not finish within its bounds (%s).", r.Command, r.FailureClass),
				Evidence:    lastLines(r),
				Severity:    "error",
			})
		}
	}

	// Runtime pressure: a (passing or failing) command whose duration approaches
	// its timeout.
	for _, r := range results {
		timeout := timeoutSecondsFor(plan, r)
		if timeout <= 0 || r.DurationMS <= 0 {
			continue
		}
		ratio := float64(r.DurationMS) / float64(int64(timeout)*1000)
		if ratio < runtimePressureRatio {
			continue
		}
		ws := workspaceForCommand(plan, r)
		diagnostics = append(diagnostics, Diagnostic{
			Kind:        "runtime",
			WorkspaceID: ws,
			Message:     fmt.Sprintf("Command %q ran for %dms, %.0f%% of its %ds timeout.", r.Command, r.DurationMS, ratio*100, timeout),
			Evidence:    fmt.Sprintf("duration=%dms timeout=%ds", r.DurationMS, timeout),
			Severity:    "info",
		})
		findings = append(findings, Finding{
			ID:            codeTestRuntimeGrowth + "-" + ws,
			Scenario:      scenario,
			WorkspaceID:   ws,
			Code:          codeTestRuntimeGrowth,
			Category:      "diagnostics",
			Severity:      codeSeverity[codeTestRuntimeGrowth],
			Message:       fmt.Sprintf("Workspace %q test runtime is approaching its timeout.", ws),
			Evidence:      fmt.Sprintf("%dms of a %ds budget (%.0f%%)", r.DurationMS, timeout, ratio*100),
			Expected:      "Test runtime comfortably under the per-workspace timeout.",
			Observed:      fmt.Sprintf("%.0f%% of timeout", ratio*100),
			WhyItMatters:  "A suite running near its timeout flakes intermittently as load varies and will eventually fail outright.",
			Remediation:   "Profile and speed up the slowest tests, or split the workspace; avoid simply raising the timeout.",
			SourceCommand: r.Command,
			CreatedAt:     now,
		})
	}

	// Static flake markers.
	for _, ws := range workspaces {
		markers := flakeMarkers(ws)
		if len(markers) == 0 {
			continue
		}
		sort.Strings(markers)
		diagnostics = append(diagnostics, Diagnostic{
			Kind:        "flake",
			WorkspaceID: ws.ID,
			Message:     "Tests self-identify as flaky or retry-prone.",
			Evidence:    strings.Join(truncateList(markers, 10), "; "),
			Severity:    "warning",
		})
		findings = append(findings, Finding{
			ID:           codeTestFlakeSuspected + "-" + ws.ID,
			Scenario:     scenario,
			WorkspaceID:  ws.ID,
			Language:     ws.Language,
			Code:         codeTestFlakeSuspected,
			Category:     "diagnostics",
			Severity:     codeSeverity[codeTestFlakeSuspected],
			FilePath:     ws.RootPath,
			Message:      "Tests reference flake/retry handling, suggesting known nondeterminism.",
			Evidence:     strings.Join(truncateList(markers, 10), "; "),
			Expected:     "Deterministic tests with no flake/retry workarounds.",
			Observed:     fmt.Sprintf("%d flake/retry marker(s)", len(markers)),
			WhyItMatters: "Flaky tests erode trust in the suite and let real regressions hide behind retries.",
			Remediation:  "Make the underlying test deterministic (inject time/randomness, await real conditions) instead of retrying.",
			CreatedAt:    now,
		})
	}

	return diagnostics, findings
}

func flakeMarkers(ws Workspace) []string {
	seen := map[string]struct{}{}
	var out []string
	walkSourceFiles(ws.RootPath, func(path string) {
		if !isGoTestFile(path) && !isTSTestFile(path) {
			return
		}
		if flakeMarkerRe.MatchString(readFileString(path)) {
			base := relTo(ws.RootPath, path)
			if _, ok := seen[base]; !ok {
				seen[base] = struct{}{}
				out = append(out, base)
			}
		}
	})
	return out
}

func workspaceForCommand(plan ExecutionPlan, r CommandResult) string {
	for _, c := range plan.Commands {
		if c.Command == r.Command && c.WorkingDirectory == r.WorkingDirectory {
			return c.WorkspaceID
		}
	}
	return ""
}

func timeoutSecondsFor(plan ExecutionPlan, r CommandResult) int {
	if r.TimeoutSeconds > 0 {
		return r.TimeoutSeconds
	}
	for _, c := range plan.Commands {
		if c.Command == r.Command && c.WorkingDirectory == r.WorkingDirectory {
			return c.TimeoutSeconds
		}
	}
	return 0
}

func lastLines(r CommandResult) string {
	tail := strings.TrimSpace(r.StderrExcerpt)
	if tail == "" {
		tail = strings.TrimSpace(r.StdoutExcerpt)
	}
	if tail == "" {
		return r.FailureReason
	}
	return r.FailureReason + "\n--- last output ---\n" + tail
}
