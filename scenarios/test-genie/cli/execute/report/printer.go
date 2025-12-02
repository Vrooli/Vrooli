package report

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	execTypes "test-genie/cli/internal/execute"
	"test-genie/cli/internal/phases"
	"test-genie/cli/internal/repo"
)

// Printer renders a structured execution report to an io.Writer.
type Printer struct {
	w                   io.Writer
	scenario            string
	requestedPreset     string
	requestedPhases     []string
	requestedSkip       []string
	failFast            bool
	descriptorMap       map[string]phases.Descriptor
	targetDurationByKey map[string]time.Duration
}

// New builds a printer instance.
func New(
	w io.Writer,
	scenario,
	requestedPreset string,
	requestedPhases,
	requestedSkip []string,
	failFast bool,
	descriptors []phases.Descriptor,
) *Printer {
	descMap, targets := phases.MakeDescriptorMaps(descriptors)
	return &Printer{
		w:                   w,
		scenario:            scenario,
		requestedPreset:     requestedPreset,
		requestedPhases:     requestedPhases,
		requestedSkip:       requestedSkip,
		failFast:            failFast,
		descriptorMap:       descMap,
		targetDurationByKey: targets,
	}
}

// Print renders the full execution report.
func (p *Printer) Print(resp execTypes.Response) {
	p.printHeader(resp)
	p.printPlan(resp.Phases)
	p.printPhaseProgress(resp.Phases)
	p.printPhaseResults(resp.Phases)
	p.printSummary(resp)
	p.printFailureDigest(resp.Phases)
	p.printQuickFixGuide(resp.Phases)
	p.printDebugGuides(resp.Phases)
	p.printArtifacts(resp)
	p.printDocs()
}

func (p *Printer) printHeader(resp execTypes.Response) {
	title := fmt.Sprintf("%s TEST EXECUTION", strings.ToUpper(p.scenario))
	startText := DefaultValue(resp.StartedAt, "unknown")
	finishText := DefaultValue(resp.CompletedAt, "pending")
	duration := FormatRunDuration(resp.PhaseSummary.DurationSeconds, resp.StartedAt, resp.CompletedAt)
	preset := DefaultValue(resp.PresetUsed, p.requestedPreset)
	paths := repo.DiscoverScenarioPaths(p.scenario)
	estimated := p.estimateTotal(resp.Phases)

	fmt.Fprintln(p.w, "╔═══════════════════════════════════════════════════════════════╗")
	fmt.Fprintf(p.w, "║  %-61s║\n", title)
	fmt.Fprintln(p.w, "╠═══════════════════════════════════════════════════════════════╣")
	fmt.Fprintf(p.w, "║  %-61s║\n", fmt.Sprintf("Scenario: %s", p.scenario))
	if preset != "" {
		fmt.Fprintf(p.w, "║  %-61s║\n", fmt.Sprintf("Preset: %s", preset))
	}
	if len(p.requestedPhases) > 0 {
		fmt.Fprintf(p.w, "║  %-61s║\n", fmt.Sprintf("Requested phases: %s", strings.Join(p.requestedPhases, ", ")))
	}
	if len(p.requestedSkip) > 0 {
		fmt.Fprintf(p.w, "║  %-61s║\n", fmt.Sprintf("Skip: %s", strings.Join(p.requestedSkip, ", ")))
	}
	if p.failFast {
		fmt.Fprintf(p.w, "║  %-61s║\n", "Fail-fast: enabled")
	}
	fmt.Fprintf(p.w, "║  %-61s║\n", fmt.Sprintf("Started: %s", startText))
	fmt.Fprintf(p.w, "║  %-61s║\n", fmt.Sprintf("Finished: %s", finishText))
	fmt.Fprintf(p.w, "║  %-61s║\n", fmt.Sprintf("Duration: %s", duration))
	if estimated != "" {
		fmt.Fprintf(p.w, "║  %-61s║\n", fmt.Sprintf("Estimated plan time: %s", estimated))
	}
	fmt.Fprintf(p.w, "║  %-61s║\n", fmt.Sprintf("Phases executed: %d", len(resp.Phases)))
	if paths.ScenarioDir != "" {
		fmt.Fprintf(p.w, "║  %-61s║\n", fmt.Sprintf("Scenario dir: %s", paths.ScenarioDir))
	}
	if paths.TestDir != "" {
		fmt.Fprintf(p.w, "║  %-61s║\n", fmt.Sprintf("Test dir: %s", paths.TestDir))
	}
	fmt.Fprintln(p.w, "╚═══════════════════════════════════════════════════════════════╝")
	fmt.Fprintln(p.w)
}

func (p *Printer) printPlan(phasesData []execTypes.Phase) {
	fmt.Fprintln(p.w, "Execution plan:")
	if len(phasesData) == 0 {
		fmt.Fprintln(p.w, "  • (planner returned no phases)")
		fmt.Fprintln(p.w)
		return
	}
	for idx, phase := range phasesData {
		desc := p.lookupPhaseDescription(phase.Name)
		target := p.targetDuration(phase.Name)
		line := fmt.Sprintf("  [%d/%d] %-12s", idx+1, len(phasesData), phase.Name)
		if desc != "" {
			line = fmt.Sprintf("%s → %s", line, desc)
		}
		if target != "" {
			line = fmt.Sprintf("%s (target: %s)", line, target)
		}
		fmt.Fprintln(p.w, line)
	}
	if est := p.estimateTotal(phasesData); est != "" {
		fmt.Fprintf(p.w, "  • Estimated total time: %s\n", est)
	}
	fmt.Fprintln(p.w)
}

func (p *Printer) printPhaseProgress(phasesData []execTypes.Phase) {
	if len(phasesData) == 0 {
		return
	}
	fmt.Fprintln(p.w, "Phase progress:")
	var cumulative time.Duration
	for idx, phase := range phasesData {
		key := NormalizeName(phase.Name)
		duration := time.Duration(phase.DurationSeconds * float64(time.Second))
		if duration < 0 {
			duration = 0
		}
		target := p.targetDuration(key)
		startMark := fmt.Sprintf("t+%s", cumulative.Truncate(time.Millisecond).String())
		cumulative += duration
		endMark := fmt.Sprintf("t+%s", cumulative.Truncate(time.Millisecond).String())
		status := strings.ToUpper(DefaultValue(phase.Status, "pending"))
		fmt.Fprintf(p.w, "  [%d] %-12s %s (%s → %s", idx+1, key, status, startMark, endMark)
		if target != "" {
			fmt.Fprintf(p.w, ", target %s", target)
		}
		fmt.Fprintln(p.w, ")")
	}
	fmt.Fprintln(p.w)
}

func (p *Printer) printPhaseResults(phasesData []execTypes.Phase) {
	fmt.Fprintln(p.w, "Phase results:")
	if len(phasesData) == 0 {
		fmt.Fprintln(p.w, "  (no phases recorded)")
		return
	}
	for _, phase := range phasesData {
		icon := StatusIcon(phase.Status)
		status := strings.ToUpper(DefaultValue(phase.Status, "unknown"))
		warnings := ""
		if phase.LogPath != "" {
			if warns := CountLinesContaining(phase.LogPath, "[WARNING"); warns > 0 {
				warnings = fmt.Sprintf(" warnings=%d", warns)
			}
		}
		target := p.targetDuration(phase.Name)
		targetText := ""
		if target != "" {
			targetText = fmt.Sprintf(" target=%s", target)
		}
		fmt.Fprintf(p.w, "  %s %-12s status=%-8s duration=%s%s%s\n", icon, phase.Name, status, FormatPhaseDuration(phase.DurationSeconds), targetText, warnings)
		if phase.LogPath != "" {
			exists, empty := repo.FileState(phase.LogPath)
			logLine := fmt.Sprintf("log: %s", phase.LogPath)
			switch {
			case !exists:
				logLine = fmt.Sprintf("%s (missing)", logLine)
			case empty:
				logLine = fmt.Sprintf("%s (empty)", logLine)
			}
			fmt.Fprintf(p.w, "      %s\n", logLine)
		}
		if len(phase.Observations) > 0 {
			for _, obs := range phase.Observations {
				if strings.TrimSpace(obs) == "" {
					continue
				}
				fmt.Fprintf(p.w, "      • %s\n", strings.TrimSpace(obs))
			}
		}
		if !strings.EqualFold(status, "PASSED") && phase.LogPath != "" {
			if snippet := ReadLogSnippet(phase.LogPath, 2000); snippet != "" {
				fmt.Fprintf(p.w, "      log snippet:\n")
				for _, line := range TailLines(snippet, 12) {
					fmt.Fprintf(p.w, "        %s\n", line)
				}
			}
		}
		if phase.Error != "" {
			fmt.Fprintf(p.w, "      error: %s\n", phase.Error)
		}
		if phase.Classification != "" {
			fmt.Fprintf(p.w, "      classification: %s\n", phase.Classification)
		}
		if phase.Remediation != "" {
			fmt.Fprintf(p.w, "      remediation: %s\n", phase.Remediation)
		}
	}
}

func (p *Printer) printSummary(resp execTypes.Response) {
	total := resp.PhaseSummary.Total
	passed := resp.PhaseSummary.Passed
	failed := resp.PhaseSummary.Failed
	duration := FormatRunDuration(resp.PhaseSummary.DurationSeconds, resp.StartedAt, resp.CompletedAt)
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, "Summary:")
	fmt.Fprintf(p.w, "  • Results: %d passed • %d failed • %d total\n", passed, failed, total)
	fmt.Fprintf(p.w, "  • Duration: %s\n", duration)
	if resp.PhaseSummary.ObservationCount > 0 {
		fmt.Fprintf(p.w, "  • Observations recorded: %d\n", resp.PhaseSummary.ObservationCount)
	}
	if resp.Success {
		fmt.Fprintln(p.w, "  • Status: ✅ all phases completed successfully")
	} else {
		fmt.Fprintln(p.w, "  • Status: ⚠ failures detected (see analysis below)")
	}
}

func (p *Printer) printFailureDigest(phasesData []execTypes.Phase) {
	failed := FilterFailedPhases(phasesData)
	if len(failed) == 0 {
		return
	}
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, "Failure analysis:")
	insights := AnalyzePhaseFailures(failed)
	for _, insight := range insights {
		fmt.Fprintf(p.w, "  ❌ %s\n", strings.ToUpper(insight.Phase))
		if insight.Cause != "" {
			fmt.Fprintf(p.w, "     cause: %s\n", insight.Cause)
		}
		if insight.Detail != "" {
			fmt.Fprintf(p.w, "     detail: %s\n", insight.Detail)
		}
		if len(insight.Impact) > 0 {
			fmt.Fprintf(p.w, "     impact: blocks %s\n", strings.Join(insight.Impact, ", "))
		}
		if len(insight.Fixes) > 0 {
			fmt.Fprintf(p.w, "     quick fixes:\n")
			for i, fix := range insight.Fixes {
				fmt.Fprintf(p.w, "       %d) %s\n", i+1, fix)
			}
		}
		if insight.Log != "" {
			fmt.Fprintf(p.w, "     log: %s\n", insight.Log)
		}
		if len(insight.Observations) > 0 {
			fmt.Fprintf(p.w, "     observations:\n")
			for _, obs := range insight.Observations {
				fmt.Fprintf(p.w, "       • %s\n", obs)
			}
		}
	}

	if cp := SummarizeCriticalPath(insights); cp != nil {
		fmt.Fprintln(p.w)
		fmt.Fprintln(p.w, "Critical path:")
		fmt.Fprintf(p.w, "  🔴 Primary: %s — %s\n", cp.PrimaryPhase, cp.PrimaryCause)
		if cp.Detail != "" {
			fmt.Fprintf(p.w, "     Detail: %s\n", cp.Detail)
		}
		if len(cp.BlockedPhases) > 0 {
			fmt.Fprintf(p.w, "     Blocks: %s\n", strings.Join(cp.BlockedPhases, ", "))
		}
		if len(cp.QuickFixes) > 0 {
			fmt.Fprintln(p.w, "  🧭 Quick fix guide:")
			for i, fix := range cp.QuickFixes {
				fmt.Fprintf(p.w, "     %d) %s\n", i+1, fix)
			}
		}
	}
}

func (p *Printer) printQuickFixGuide(phasesData []execTypes.Phase) {
	failed := FilterFailedPhases(phasesData)
	if len(failed) == 0 {
		return
	}
	insights := AnalyzePhaseFailures(failed)
	if len(insights) == 0 {
		return
	}
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, "Quick fix guide:")
	step := 1
	for _, insight := range insights {
		if len(insight.Fixes) == 0 {
			continue
		}
		for _, fix := range insight.Fixes {
			fix = strings.TrimSpace(fix)
			if fix == "" {
				continue
			}
			fmt.Fprintf(p.w, "  %d) %s (%s)\n", step, fix, insight.Phase)
			step++
		}
	}
	if step == 1 {
		fmt.Fprintln(p.w, "  (no quick fixes detected; check phase logs)")
	}
}

func (p *Printer) printDebugGuides(phasesData []execTypes.Phase) {
	failed := FilterFailedPhases(phasesData)
	if len(failed) < 2 {
		return
	}
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, "Phase-specific debug guides:")
	for _, phase := range failed {
		switch NormalizeName(phase.Name) {
		case "unit":
			fmt.Fprintln(p.w, "  • UNIT: common issues → missing migrations, stale module cache, missing deps")
			fmt.Fprintln(p.w, "    Quick checks: go clean -testcache | reinstall UI deps | rerun go/node tests directly")
		case "integration":
			fmt.Fprintln(p.w, "  • INTEGRATION: common issues → scenario not running (API_PORT), stale UI bundle, missing BAS CLI")
			fmt.Fprintln(p.w, "    Quick checks: vrooli scenario restart <scenario> | vrooli scenario status <scenario> | install BAS CLI")
		case "performance":
			fmt.Fprintln(p.w, "  • PERFORMANCE: common issues → Lighthouse scores, bundle too large, slow page load")
			fmt.Fprintln(p.w, "    Quick checks: open test/artifacts/lighthouse/*.html | pnpm run analyze | inspect ui/dist/assets")
		case "structure":
			fmt.Fprintln(p.w, "  • STRUCTURE: common issues → UI smoke failing, missing files, invalid JSON config")
			fmt.Fprintln(p.w, "    Quick checks: tail logs/<scenario>-api.log | validate .vrooli/service.json | restart scenario")
		case "business":
			fmt.Fprintln(p.w, "  • BUSINESS: common issues → API contract changes, CLI drift, websocket issues")
			fmt.Fprintln(p.w, "    Quick checks: rerun CLI against API | inspect API logs | rebuild UI bundle")
		case "dependencies":
			fmt.Fprintln(p.w, "  • DEPENDENCIES: common issues → missing resources, install drift, pinned versions outdated")
			fmt.Fprintln(p.w, "    Quick checks: verify postgres/redis availability | reinstall deps | rerun install scripts")
		}
	}
}

func (p *Printer) printArtifacts(resp execTypes.Response) {
	if len(resp.Links) == 0 && len(resp.Metadata) == 0 {
		paths := CollectArtifactRoots(resp.Phases)
		if len(paths) == 0 {
			return
		}
	}
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, "Artifacts & metadata:")

	if paths := repo.DiscoverScenarioPaths(p.scenario); paths.TestDir != "" {
		artifactDir := filepath.Join(paths.TestDir, "artifacts")
		if repo.Exists(artifactDir) {
			fmt.Fprintf(p.w, "  • phase logs: %s\n", artifactDir)
		}
	}

	if artifactLines := DescribeArtifacts(resp.Phases); len(artifactLines) > 0 {
		for _, line := range artifactLines {
			fmt.Fprintf(p.w, "  • %s\n", line)
		}
	}
	for _, line := range DescribeCoverage(p.scenario) {
		fmt.Fprintf(p.w, "  • %s\n", line)
	}

	printMap := func(m map[string]any) {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			value := fmt.Sprintf("%v", m[key])
			if strings.TrimSpace(value) == "" {
				continue
			}
			fmt.Fprintf(p.w, "  • %s: %s\n", key, value)
		}
	}
	if len(resp.Links) > 0 {
		printMap(resp.Links)
	}
	if len(resp.Metadata) > 0 {
		printMap(resp.Metadata)
	}

	if diag := DiagnoseFailures(resp.Phases); diag != nil {
		fmt.Fprintln(p.w)
		fmt.Fprintln(p.w, "Root-cause summary:")
		if diag.Primary != "" {
			fmt.Fprintf(p.w, "  🔴 Primary blocker: %s\n", diag.Primary)
			if diag.PrimaryPhase != "" {
				fmt.Fprintf(p.w, "     Phase: %s\n", diag.PrimaryPhase)
			}
			if diag.Details != "" {
				fmt.Fprintf(p.w, "     Details: %s\n", diag.Details)
			}
			if len(diag.ImpactedPhases) > 0 {
				fmt.Fprintf(p.w, "     Impact: blocks %s\n", strings.Join(diag.ImpactedPhases, ", "))
			}
		}
		if len(diag.SecondaryIssues) > 0 {
			fmt.Fprintln(p.w, "  🟡 Secondary issues:")
			for _, issue := range diag.SecondaryIssues {
				fmt.Fprintf(p.w, "     • %s\n", issue)
			}
		}
		if len(diag.QuickFixes) > 0 {
			fmt.Fprintln(p.w, "  🧭 Quick fixes:")
			for idx, fix := range diag.QuickFixes {
				fmt.Fprintf(p.w, "     %d) %s\n", idx+1, fix)
			}
		}
	}
}

func (p *Printer) printDocs() {
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, "Reference docs:")
	fmt.Fprintln(p.w, "  • docs/testing/architecture/PHASED_TESTING.md")
	fmt.Fprintln(p.w, "  • docs/testing/guides/requirement-tracking-quick-start.md")
	fmt.Fprintln(p.w, "  • docs/testing/guides/ui-automation-with-bas.md")
	fmt.Fprintln(p.w, "  • docs/testing/guides/writing-testable-uis.md")
	fmt.Fprintln(p.w)
}

func (p *Printer) lookupPhaseDescription(name string) string {
	desc, ok := p.descriptorMap[strings.ToLower(name)]
	if !ok {
		return ""
	}
	return desc.Description
}

func (p *Printer) targetDuration(name string) string {
	if d, ok := p.targetDurationByKey[NormalizeName(name)]; ok && d > 0 {
		return d.Truncate(time.Second).String()
	}
	return ""
}

func (p *Printer) estimateTotal(phasesData []execTypes.Phase) string {
	if len(phasesData) == 0 {
		return ""
	}
	var total time.Duration
	for _, phase := range phasesData {
		if d, ok := p.targetDurationByKey[NormalizeName(phase.Name)]; ok && d > 0 {
			total += d
			continue
		}
		if phase.DurationSeconds > 0 {
			total += time.Duration(phase.DurationSeconds * float64(time.Second))
		}
	}
	if total == 0 {
		return ""
	}
	return total.Truncate(time.Second).String()
}
