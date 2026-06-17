package validation

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"unit-health/internal/executor"
	"unit-health/internal/runhistory"
)

// runtimePressureRatio flags a command whose duration reaches this fraction of
// its timeout: the suite is approaching the bound and likely to start timing out.
const runtimePressureRatio = 0.75

// Runtime-growth and flake thresholds, computed from persisted cross-run
// history (B6). Growth needs a few historical samples to form a stable
// baseline; flake needs at least one prior run to observe a status flip.
const (
	minRuntimeHistory  = 3
	runtimeGrowthRatio = 1.5
	minFlakeObservs    = 2
	historyRunWindow   = runhistory.DefaultRetention
)

// flakeMarkerRe matches deliberate source signals that tests self-identify as
// flaky. Detecting flake without reruns is best-effort, so Unit Health only
// reports the strongest intentional markers (the word "flaky"/"flake", a
// flaky-gated skip, or a flaky-suppressing lint directive) rather than broad
// words like "retry" or "eventually" that appear in healthy async tests.
var flakeMarkerRe = regexp.MustCompile(`(?i)\bflak(?:e|y)\b`)

// analyzeDiagnostics derives flake/runtime/hang diagnostics. Hang and
// runtime-pressure come from the current run's executed commands; runtime-growth
// and cross-run flake come from persisted history (B6); the static "flaky"
// source marker survives only as a weak supplementary signal. results and
// history are empty when execution was not requested.
func analyzeDiagnostics(scenario string, workspaces []Workspace, plan ExecutionPlan, results []CommandResult, history []runhistory.CommandSample, now string) ([]Diagnostic, []Finding) {
	var diagnostics []Diagnostic
	var findings []Finding
	hist := groupCommandHistory(history)

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

	// Runtime pressure (single-run): a command whose duration approaches its
	// timeout. This is a diagnostic only; the TEST_RUNTIME_GROWTH finding below
	// is now history-based growth, not single-run near-timeout.
	for _, r := range results {
		timeout := timeoutSecondsFor(plan, r)
		if timeout <= 0 || r.DurationMS <= 0 {
			continue
		}
		ratio := float64(r.DurationMS) / float64(int64(timeout)*1000)
		if ratio < runtimePressureRatio {
			continue
		}
		diagnostics = append(diagnostics, Diagnostic{
			Kind:        "runtime",
			WorkspaceID: workspaceForCommand(plan, r),
			Message:     fmt.Sprintf("Command %q ran for %dms, %.0f%% of its %ds timeout.", r.Command, r.DurationMS, ratio*100, timeout),
			Evidence:    fmt.Sprintf("duration=%dms timeout=%ds", r.DurationMS, timeout),
			Severity:    "info",
		})
	}

	// Runtime growth (cross-run): current duration vs the rolling median of the
	// same command's recent passing runs.
	for _, r := range results {
		if r.DurationMS <= 0 || r.Status != executor.StatusPassed {
			continue
		}
		ws := workspaceForCommand(plan, r)
		prior := pastDurations(hist[ws+"|"+r.Command])
		if len(prior) < minRuntimeHistory {
			continue
		}
		baseline := medianInt64(prior)
		if baseline <= 0 || float64(r.DurationMS) < float64(baseline)*runtimeGrowthRatio {
			continue
		}
		growth := float64(r.DurationMS) / float64(baseline)
		diagnostics = append(diagnostics, Diagnostic{
			Kind:        "runtime",
			WorkspaceID: ws,
			Message:     fmt.Sprintf("Command %q runtime grew %.1f× over its rolling baseline.", r.Command, growth),
			Evidence:    fmt.Sprintf("current=%dms baseline(median of %d runs)=%dms", r.DurationMS, len(prior), baseline),
			Severity:    "warning",
		})
		findings = append(findings, Finding{
			ID:            codeTestRuntimeGrowth + "-" + ws,
			Scenario:      scenario,
			WorkspaceID:   ws,
			Code:          codeTestRuntimeGrowth,
			Category:      "diagnostics",
			Severity:      codeSeverity[codeTestRuntimeGrowth],
			Message:       fmt.Sprintf("Workspace %q test runtime grew %.1f× over its rolling baseline.", ws, growth),
			Evidence:      fmt.Sprintf("current=%dms vs baseline=%dms (median of %d prior runs)", r.DurationMS, baseline, len(prior)),
			Expected:      fmt.Sprintf("Runtime within %.0f%% of the rolling baseline.", runtimeGrowthRatio*100),
			Observed:      fmt.Sprintf("%.1f× baseline", growth),
			WhyItMatters:  "A test suite whose runtime is climbing run-over-run trends toward timeouts and flake as load varies.",
			Remediation:   "Profile what slowed down since the baseline (new tests, fixtures, I/O) and bring the runtime back down.",
			SourceCommand: r.Command,
			CreatedAt:     now,
		})
	}

	// Flake (cross-run): the same command flips between pass and fail across
	// recent runs (current + history).
	for _, r := range results {
		ws := workspaceForCommand(plan, r)
		statuses := append([]string{r.Status}, pastStatuses(hist[ws+"|"+r.Command])...)
		if len(statuses) < minFlakeObservs {
			continue
		}
		if !statusesFlipFlop(statuses) {
			continue
		}
		passes, fails := countStatuses(statuses)
		findings = append(findings, Finding{
			ID:            codeTestFlakeSuspected + "-" + ws,
			Scenario:      scenario,
			WorkspaceID:   ws,
			Code:          codeTestFlakeSuspected,
			Category:      "diagnostics",
			Severity:      codeSeverity[codeTestFlakeSuspected],
			Message:       fmt.Sprintf("Command %q flips between pass and fail across recent runs.", r.Command),
			Evidence:      fmt.Sprintf("%d pass / %d fail across %d recent run(s)", passes, fails, len(statuses)),
			Expected:      "Deterministic, stable pass/fail across runs.",
			Observed:      "pass/fail flip-flop",
			WhyItMatters:  "A command with inconsistent results across runs is flaky; it erodes trust and lets real regressions hide behind a passing retry.",
			Remediation:   "Make the test deterministic (inject time/randomness, await real conditions) rather than relying on reruns.",
			SourceCommand: r.Command,
			CreatedAt:     now,
		})
		diagnostics = append(diagnostics, Diagnostic{
			Kind:        "flake",
			WorkspaceID: ws,
			Message:     fmt.Sprintf("Command %q has an inconsistent pass/fail history.", r.Command),
			Evidence:    fmt.Sprintf("%d pass / %d fail across %d run(s)", passes, fails, len(statuses)),
			Severity:    "warning",
		})
	}

	// Supplementary (weak) signal: tests that self-identify as flaky in source.
	// Kept as a diagnostic only — it is not durable evidence of flake, unlike
	// the cross-run variance above.
	for _, ws := range workspaces {
		markers := flakeMarkers(ws)
		if len(markers) == 0 {
			continue
		}
		sort.Strings(markers)
		diagnostics = append(diagnostics, Diagnostic{
			Kind:        "flake",
			WorkspaceID: ws.ID,
			Message:     "Supplementary signal: tests reference flake/retry handling in source.",
			Evidence:    "static markers (weak): " + strings.Join(truncateList(markers, 10), "; "),
			Severity:    "info",
		})
	}

	return diagnostics, findings
}

func groupCommandHistory(samples []runhistory.CommandSample) map[string][]runhistory.CommandSample {
	m := map[string][]runhistory.CommandSample{}
	for _, s := range samples {
		k := s.WorkspaceID + "|" + s.Command
		m[k] = append(m[k], s)
	}
	return m
}

// pastDurations returns the durations of prior passing runs of a command.
func pastDurations(samples []runhistory.CommandSample) []int64 {
	var out []int64
	for _, s := range samples {
		if s.Status == executor.StatusPassed && s.DurationMS > 0 {
			out = append(out, s.DurationMS)
		}
	}
	return out
}

func pastStatuses(samples []runhistory.CommandSample) []string {
	out := make([]string, 0, len(samples))
	for _, s := range samples {
		out = append(out, s.Status)
	}
	return out
}

func medianInt64(in []int64) int64 {
	if len(in) == 0 {
		return 0
	}
	cp := append([]int64(nil), in...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	mid := len(cp) / 2
	if len(cp)%2 == 1 {
		return cp[mid]
	}
	return (cp[mid-1] + cp[mid]) / 2
}

// statusesFlipFlop reports whether a command both passed and failed across runs.
func statusesFlipFlop(statuses []string) bool {
	passes, fails := countStatuses(statuses)
	return passes > 0 && fails > 0
}

func countStatuses(statuses []string) (passes, fails int) {
	for _, s := range statuses {
		switch s {
		case executor.StatusPassed:
			passes++
		case executor.StatusFailed, executor.StatusTimeout, executor.StatusError:
			fails++
		}
	}
	return passes, fails
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
