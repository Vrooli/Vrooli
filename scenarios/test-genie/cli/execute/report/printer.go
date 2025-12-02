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

// Separator line used for phase boundaries (matches legacy output).
const phaseSeparator = "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

// Printer renders a structured execution report to an io.Writer.
type Printer struct {
	w                   io.Writer
	color               *Color
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
		color:               NewColor(w),
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
	title := fmt.Sprintf("%s COMPREHENSIVE TEST SUITE", strings.ToUpper(p.scenario))
	startText := FormatTimestampShort(resp.StartedAt, "unknown")
	finishText := FormatTimestampShort(resp.CompletedAt, "pending")
	duration := FormatRunDuration(resp.PhaseSummary.DurationSeconds, resp.StartedAt, resp.CompletedAt)
	preset := DefaultValue(resp.PresetUsed, p.requestedPreset)
	paths := repo.DiscoverScenarioPaths(p.scenario)
	estimated := p.estimateTotal(resp.Phases)

	fmt.Fprintln(p.w, p.color.Cyan("╔═══════════════════════════════════════════════════════════════╗"))
	fmt.Fprintf(p.w, "%s  %-61s%s\n", p.color.Cyan("║"), p.color.BoldCyan(title), p.color.Cyan("║"))
	fmt.Fprintln(p.w, p.color.Cyan("╠═══════════════════════════════════════════════════════════════╣"))
	fmt.Fprintf(p.w, "%s  %-61s%s\n", p.color.Cyan("║"), fmt.Sprintf("Scenario: %s", p.scenario), p.color.Cyan("║"))
	if preset != "" {
		fmt.Fprintf(p.w, "%s  %-61s%s\n", p.color.Cyan("║"), fmt.Sprintf("Preset: %s", preset), p.color.Cyan("║"))
	}
	if len(p.requestedPhases) > 0 {
		fmt.Fprintf(p.w, "%s  %-61s%s\n", p.color.Cyan("║"), fmt.Sprintf("Requested phases: %s", strings.Join(p.requestedPhases, ", ")), p.color.Cyan("║"))
	}
	if len(p.requestedSkip) > 0 {
		fmt.Fprintf(p.w, "%s  %-61s%s\n", p.color.Cyan("║"), fmt.Sprintf("Skip: %s", strings.Join(p.requestedSkip, ", ")), p.color.Cyan("║"))
	}
	if p.failFast {
		fmt.Fprintf(p.w, "%s  %-61s%s\n", p.color.Cyan("║"), "Fail-fast: enabled", p.color.Cyan("║"))
	}
	fmt.Fprintf(p.w, "%s  %-61s%s\n", p.color.Cyan("║"), fmt.Sprintf("Started: %s", startText), p.color.Cyan("║"))
	fmt.Fprintf(p.w, "%s  %-61s%s\n", p.color.Cyan("║"), fmt.Sprintf("Finished: %s", finishText), p.color.Cyan("║"))
	fmt.Fprintf(p.w, "%s  %-61s%s\n", p.color.Cyan("║"), fmt.Sprintf("Duration: %s", duration), p.color.Cyan("║"))
	if estimated != "" {
		fmt.Fprintf(p.w, "%s  %-61s%s\n", p.color.Cyan("║"), fmt.Sprintf("Estimated plan time: %s", estimated), p.color.Cyan("║"))
	}
	fmt.Fprintf(p.w, "%s  Phases: %-54d%s\n", p.color.Cyan("║"), len(resp.Phases), p.color.Cyan("║"))
	if paths.ScenarioDir != "" {
		fmt.Fprintf(p.w, "%s  %-61s%s\n", p.color.Cyan("║"), fmt.Sprintf("Test directory: %s/test", p.scenario), p.color.Cyan("║"))
	}
	fmt.Fprintln(p.w, p.color.Cyan("╚═══════════════════════════════════════════════════════════════╝"))
	fmt.Fprintln(p.w)
}

func (p *Printer) printPlan(phasesData []execTypes.Phase) {
	fmt.Fprintln(p.w, p.color.Bold("Test Plan:"))
	if len(phasesData) == 0 {
		fmt.Fprintln(p.w, "  • (planner returned no phases)")
		fmt.Fprintln(p.w)
		return
	}
	for idx, phase := range phasesData {
		target := p.targetDuration(phase.Name)
		desc := p.lookupPhaseDescription(phase.Name)
		// Format: [1/6] structure       (±120s)  → Description
		targetText := ""
		if target != "" {
			targetText = fmt.Sprintf("(±%s)", target)
		}
		line := fmt.Sprintf("  [%d/%d] %-14s %-10s", idx+1, len(phasesData), phase.Name, targetText)
		if desc != "" {
			line = fmt.Sprintf("%s → %s", line, desc)
		}
		fmt.Fprintln(p.w, p.color.Cyan(line))
	}
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, p.color.Cyan("Starting execution..."))
	fmt.Fprintln(p.w)
}

func (p *Printer) printPhaseProgress(phasesData []execTypes.Phase) {
	if len(phasesData) == 0 {
		return
	}
	total := len(phasesData)
	for idx, phase := range phasesData {
		key := NormalizeName(phase.Name)
		duration := time.Duration(phase.DurationSeconds * float64(time.Second))
		if duration < 0 {
			duration = 0
		}
		durationSec := int(duration.Seconds())
		target := p.targetDuration(key)
		status := strings.ToLower(DefaultValue(phase.Status, "pending"))
		timestamp := time.Now().Unix()

		// Count observations/tests (estimate based on available data)
		testCount := len(phase.Observations)
		if testCount == 0 {
			testCount = 1 // At minimum, 1 test per phase
		}
		errorCount := 0
		warningCount := 0
		if phase.Error != "" {
			errorCount = 1
		}
		if phase.LogPath != "" {
			warningCount = CountLinesContaining(phase.LogPath, "[WARNING")
		}

		// Machine-parseable marker: PHASE_START
		fmt.Fprintf(p.w, "[PHASE_START:%s:%d/%d:%d]\n", key, idx+1, total, timestamp)

		// Visual separator and header
		fmt.Fprintln(p.w, phaseSeparator)
		targetText := ""
		if target != "" {
			targetText = fmt.Sprintf("Timeout: %s • ", target)
		}
		headerLine := fmt.Sprintf("[%d/%d] %s • %sStarted: %s",
			idx+1, total, strings.ToUpper(key), targetText,
			time.Now().Format("15:04:05"))
		fmt.Fprintln(p.w, p.color.Bold(headerLine))

		// Show log path (like legacy shows script path)
		if phase.LogPath != "" {
			fmt.Fprintf(p.w, "   %s %s\n", p.color.Cyan("Log:"), phase.LogPath)
		}
		fmt.Fprintln(p.w, phaseSeparator)

		// Show any observations from the phase
		if len(phase.Observations) > 0 {
			for _, obs := range phase.Observations {
				if obs = strings.TrimSpace(obs); obs != "" {
					fmt.Fprintf(p.w, "%s %s\n", p.color.Green("✅"), obs)
				}
			}
		}

		// Machine-parseable marker: PHASE_END
		fmt.Fprintf(p.w, "[PHASE_END:%s:%s:%ds:%dtests:%derrors:%dwarnings]\n",
			key, status, durationSec, testCount, errorCount, warningCount)

		// Visual footer with status
		fmt.Fprintln(p.w, phaseSeparator)
		icon := StatusIcon(phase.Status)
		statusText := fmt.Sprintf("%s [%d/%d] %s %s • %ds",
			icon, idx+1, total, strings.ToUpper(key), strings.ToUpper(status), durationSec)
		if testCount > 0 {
			statusText = fmt.Sprintf("%s (%d tests", statusText, testCount)
			if warningCount > 0 {
				statusText = fmt.Sprintf("%s, %d warnings", statusText, warningCount)
			}
			statusText = statusText + ")"
		}
		fmt.Fprintln(p.w, p.color.StatusColor(status, statusText))
		fmt.Fprintln(p.w, phaseSeparator)
		fmt.Fprintln(p.w)
	}
}

func (p *Printer) printPhaseResults(phasesData []execTypes.Phase) {
	// Phase results summary in a box (matches legacy TEST SUITE COMPLETE box)
	passed := 0
	failed := 0
	var totalDuration time.Duration
	for _, phase := range phasesData {
		if strings.EqualFold(phase.Status, "passed") {
			passed++
		} else if strings.EqualFold(phase.Status, "failed") || strings.EqualFold(phase.Status, "error") {
			failed++
		}
		totalDuration += time.Duration(phase.DurationSeconds * float64(time.Second))
	}

	fmt.Fprintln(p.w, p.color.Cyan("╔═══════════════════════════════════════════════════════════════╗"))
	fmt.Fprintf(p.w, "%s  %-61s%s\n", p.color.Cyan("║"), p.color.BoldCyan("TEST SUITE COMPLETE"), p.color.Cyan("║"))
	fmt.Fprintln(p.w, p.color.Cyan("╠═══════════════════════════════════════════════════════════════╣"))
	resultText := fmt.Sprintf("Results: %d passed • %d failed • Duration: %s",
		passed, failed, totalDuration.Truncate(time.Second).String())
	fmt.Fprintf(p.w, "%s  %-61s%s\n", p.color.Cyan("║"), resultText, p.color.Cyan("║"))
	fmt.Fprintln(p.w, p.color.Cyan("╚═══════════════════════════════════════════════════════════════╝"))
	fmt.Fprintln(p.w)

	// Phase results breakdown
	fmt.Fprintln(p.w, p.color.Bold("PHASE RESULTS:"))
	if len(phasesData) == 0 {
		fmt.Fprintln(p.w, "  (no phases recorded)")
		return
	}
	for _, phase := range phasesData {
		icon := StatusIcon(phase.Status)
		status := strings.ToUpper(DefaultValue(phase.Status, "unknown"))
		durationText := FormatPhaseDuration(phase.DurationSeconds)

		// Format: ✅ phase-structure      •   8s
		phaseLine := fmt.Sprintf("  %s phase-%-14s •  %6s", icon, phase.Name, durationText)
		fmt.Fprintln(p.w, p.color.StatusColor(phase.Status, phaseLine))

		// Show log path if present
		if phase.LogPath != "" {
			exists, empty := repo.FileState(phase.LogPath)
			logLine := fmt.Sprintf("     log: %s", phase.LogPath)
			switch {
			case !exists:
				logLine = fmt.Sprintf("%s (missing)", logLine)
			case empty:
				logLine = fmt.Sprintf("%s (empty)", logLine)
			}
			fmt.Fprintln(p.w, logLine)
		}

		// For failed phases, show error details
		if !strings.EqualFold(status, "PASSED") {
			if phase.Error != "" {
				fmt.Fprintf(p.w, "     %s %s\n", p.color.Red("error:"), phase.Error)
			}
			if phase.Classification != "" {
				fmt.Fprintf(p.w, "     classification: %s\n", phase.Classification)
			}
			if phase.Remediation != "" {
				fmt.Fprintf(p.w, "     %s %s\n", p.color.Yellow("remediation:"), phase.Remediation)
			}
			// Show log snippet for failed phases
			if phase.LogPath != "" {
				if snippet := ReadLogSnippet(phase.LogPath, 2000); snippet != "" {
					fmt.Fprintln(p.w, "     log snippet:")
					for _, line := range TailLines(snippet, 8) {
						fmt.Fprintf(p.w, "       %s\n", line)
					}
				}
			}
		}
	}
	fmt.Fprintln(p.w)
}

func (p *Printer) printSummary(resp execTypes.Response) {
	if resp.Success {
		// Success celebration message (matches legacy output)
		fmt.Fprintln(p.w, p.color.BoldGreen("🎉 All tests passed successfully!"))
		fmt.Fprintf(p.w, "%s %s testing infrastructure is working correctly\n",
			p.color.Green("✅"), p.scenario)
		fmt.Fprintln(p.w)
	} else {
		// Failure summary
		total := resp.PhaseSummary.Total
		passed := resp.PhaseSummary.Passed
		failed := resp.PhaseSummary.Failed
		duration := FormatRunDuration(resp.PhaseSummary.DurationSeconds, resp.StartedAt, resp.CompletedAt)

		fmt.Fprintln(p.w)
		fmt.Fprintln(p.w, p.color.Bold("Summary:"))
		fmt.Fprintf(p.w, "  • Results: %s • %s • %d total\n",
			p.color.Green(fmt.Sprintf("%d passed", passed)),
			p.color.Red(fmt.Sprintf("%d failed", failed)),
			total)
		fmt.Fprintf(p.w, "  • Duration: %s\n", duration)
		if resp.PhaseSummary.ObservationCount > 0 {
			fmt.Fprintf(p.w, "  • Observations recorded: %d\n", resp.PhaseSummary.ObservationCount)
		}
		fmt.Fprintf(p.w, "  • Status: %s failures detected (see analysis below)\n", p.color.Yellow("⚠"))
	}
}

func (p *Printer) printFailureDigest(phasesData []execTypes.Phase) {
	failed := FilterFailedPhases(phasesData)
	if len(failed) == 0 {
		return
	}
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, p.color.BoldRed("ERROR DIGEST:"))
	insights := AnalyzePhaseFailures(failed)
	for _, insight := range insights {
		fmt.Fprintf(p.w, "  %s %s\n", p.color.Red("❌"), p.color.BoldRed(strings.ToUpper(insight.Phase)))
		if insight.Cause != "" {
			fmt.Fprintf(p.w, "     %s %s\n", p.color.Red("cause:"), insight.Cause)
		}
		if insight.Detail != "" {
			fmt.Fprintf(p.w, "     detail: %s\n", insight.Detail)
		}
		if len(insight.Impact) > 0 {
			fmt.Fprintf(p.w, "     %s blocks %s\n", p.color.Yellow("impact:"), strings.Join(insight.Impact, ", "))
		}
		if len(insight.Fixes) > 0 {
			fmt.Fprintf(p.w, "     %s\n", p.color.Cyan("quick fixes:"))
			for i, fix := range insight.Fixes {
				fmt.Fprintf(p.w, "       %d) %s\n", i+1, fix)
			}
		}
		if insight.Log != "" {
			fmt.Fprintf(p.w, "     log: %s\n", insight.Log)
		}
		if len(insight.Observations) > 0 {
			fmt.Fprintln(p.w, "     observations:")
			for _, obs := range insight.Observations {
				fmt.Fprintf(p.w, "       • %s\n", obs)
			}
		}
	}

	if cp := SummarizeCriticalPath(insights); cp != nil {
		fmt.Fprintln(p.w)
		fmt.Fprintln(p.w, p.color.Bold("CRITICAL PATH ANALYSIS:"))
		fmt.Fprintf(p.w, "  %s Primary: %s — %s\n",
			p.color.BoldRed("🔴"),
			p.color.Bold(cp.PrimaryPhase),
			cp.PrimaryCause)
		if cp.Detail != "" {
			fmt.Fprintf(p.w, "     Detail: %s\n", cp.Detail)
		}
		if len(cp.BlockedPhases) > 0 {
			fmt.Fprintf(p.w, "     %s %s\n", p.color.Yellow("Blocks:"), strings.Join(cp.BlockedPhases, ", "))
		}
		if len(cp.QuickFixes) > 0 {
			fmt.Fprintf(p.w, "  %s Quick fix guide:\n", p.color.Cyan("🧭"))
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
	fmt.Fprintln(p.w, p.color.BoldCyan("QUICK FIX GUIDE:"))
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
			fmt.Fprintf(p.w, "  %s %s (%s)\n",
				p.color.Cyan(fmt.Sprintf("%d)", step)),
				fix,
				p.color.Yellow(insight.Phase))
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
	fmt.Fprintln(p.w, p.color.BoldYellow("PHASE-SPECIFIC DEBUG GUIDES:"))
	for _, phase := range failed {
		switch NormalizeName(phase.Name) {
		case "unit":
			fmt.Fprintf(p.w, "  • %s: common issues → missing migrations, stale module cache, missing deps\n", p.color.Bold("UNIT"))
			fmt.Fprintln(p.w, p.color.Cyan("    Quick checks: go clean -testcache | reinstall UI deps | rerun go/node tests directly"))
		case "integration":
			fmt.Fprintf(p.w, "  • %s: common issues → scenario not running (API_PORT), stale UI bundle, missing BAS CLI\n", p.color.Bold("INTEGRATION"))
			fmt.Fprintln(p.w, p.color.Cyan("    Quick checks: vrooli scenario restart <scenario> | vrooli scenario status <scenario> | install BAS CLI"))
		case "performance":
			fmt.Fprintf(p.w, "  • %s: common issues → Lighthouse scores, bundle too large, slow page load\n", p.color.Bold("PERFORMANCE"))
			fmt.Fprintln(p.w, p.color.Cyan("    Quick checks: open test/artifacts/lighthouse/*.html | pnpm run analyze | inspect ui/dist/assets"))
		case "structure":
			fmt.Fprintf(p.w, "  • %s: common issues → UI smoke failing, missing files, invalid JSON config\n", p.color.Bold("STRUCTURE"))
			fmt.Fprintln(p.w, p.color.Cyan("    Quick checks: tail logs/<scenario>-api.log | validate .vrooli/service.json | restart scenario"))
		case "business":
			fmt.Fprintf(p.w, "  • %s: common issues → API contract changes, CLI drift, websocket issues\n", p.color.Bold("BUSINESS"))
			fmt.Fprintln(p.w, p.color.Cyan("    Quick checks: rerun CLI against API | inspect API logs | rebuild UI bundle"))
		case "dependencies":
			fmt.Fprintf(p.w, "  • %s: common issues → missing resources, install drift, pinned versions outdated\n", p.color.Bold("DEPENDENCIES"))
			fmt.Fprintln(p.w, p.color.Cyan("    Quick checks: verify postgres/redis availability | reinstall deps | rerun install scripts"))
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
	fmt.Fprintln(p.w, p.color.Bold("📊 Test Artifacts Summary"))
	fmt.Fprintln(p.w, "════════════════════════════════════════")

	if paths := repo.DiscoverScenarioPaths(p.scenario); paths.TestDir != "" {
		artifactDir := filepath.Join(paths.TestDir, "artifacts")
		if repo.Exists(artifactDir) {
			fmt.Fprintf(p.w, "Directory: %s\n", p.color.Cyan(artifactDir))
		}
	}

	if artifactLines := DescribeArtifacts(resp.Phases); len(artifactLines) > 0 {
		fmt.Fprintln(p.w)
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
		fmt.Fprintln(p.w, p.color.Bold("Root-cause summary:"))
		if diag.Primary != "" {
			fmt.Fprintf(p.w, "  %s Primary blocker: %s\n", p.color.BoldRed("🔴"), diag.Primary)
			if diag.PrimaryPhase != "" {
				fmt.Fprintf(p.w, "     Phase: %s\n", p.color.Bold(diag.PrimaryPhase))
			}
			if diag.Details != "" {
				fmt.Fprintf(p.w, "     Details: %s\n", diag.Details)
			}
			if len(diag.ImpactedPhases) > 0 {
				fmt.Fprintf(p.w, "     %s blocks %s\n", p.color.Yellow("Impact:"), strings.Join(diag.ImpactedPhases, ", "))
			}
		}
		if len(diag.SecondaryIssues) > 0 {
			fmt.Fprintf(p.w, "  %s Secondary issues:\n", p.color.BoldYellow("🟡"))
			for _, issue := range diag.SecondaryIssues {
				fmt.Fprintf(p.w, "     • %s\n", issue)
			}
		}
		if len(diag.QuickFixes) > 0 {
			fmt.Fprintf(p.w, "  %s Quick fixes:\n", p.color.Cyan("🧭"))
			for idx, fix := range diag.QuickFixes {
				fmt.Fprintf(p.w, "     %d) %s\n", idx+1, fix)
			}
		}
	}
}

func (p *Printer) printDocs() {
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, p.color.Bold("📚 DOCUMENTATION:"))
	fmt.Fprintf(p.w, "   • %s\n", p.color.Cyan("docs/testing/architecture/PHASED_TESTING.md"))
	fmt.Fprintf(p.w, "   • %s\n", p.color.Cyan("docs/testing/guides/requirement-tracking-quick-start.md"))
	fmt.Fprintf(p.w, "   • %s\n", p.color.Cyan("docs/testing/guides/ui-automation-with-bas.md"))
	fmt.Fprintf(p.w, "   • %s\n", p.color.Cyan("docs/testing/guides/writing-testable-uis.md"))
	if paths := repo.DiscoverScenarioPaths(p.scenario); paths.TestDir != "" {
		fmt.Fprintf(p.w, "   • Test files in: %s\n", p.color.Cyan(paths.TestDir))
	}
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
