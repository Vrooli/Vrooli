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
	w                    io.Writer
	color                *Color
	scenario             string
	requestedPreset      string
	requestedPhases      []string
	requestedSkip        []string
	failFast             bool
	descriptorMap        map[string]phases.Descriptor
	targetDurationByKey  map[string]time.Duration
	streamedObservations bool // true if observations were already streamed via SSE (live output shown)
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
	if !p.streamedObservations {
		p.printPhaseProgress(resp.Phases)
	}
	p.printPhaseResults(resp.Phases)
	p.printSummary(resp)
	p.printFailureDigest(resp.Phases)
	p.printQuickFixGuide(resp.Phases)
	p.printDebugGuides(resp.Phases)
	p.printArtifacts(resp)
	p.printDocs(resp.Phases)
}

// PrintPreExecution prints the header and test plan BEFORE the API call starts.
// This provides immediate feedback to users while waiting for tests to run.
func (p *Printer) PrintPreExecution(phaseNames []string) {
	p.printPreHeader(phaseNames)
	p.printPrePlan(phaseNames)
}

// SetStreamedObservations marks that observations were already streamed via SSE,
// so printPhaseProgress should skip re-rendering them.
func (p *Printer) SetStreamedObservations(streamed bool) {
	p.streamedObservations = streamed
}

// PrintResults prints only the results portion (after API call completes).
// Used in conjunction with PrintPreExecution for streaming-style output.
func (p *Printer) PrintResults(resp execTypes.Response) {
	// If we already streamed observations via SSE, skip the detailed phase replay
	// and go straight to the condensed results.
	if !p.streamedObservations {
		p.printPhaseProgress(resp.Phases)
	}
	p.printPhaseResults(resp.Phases)
	p.printSummary(resp)
	p.printFailureDigest(resp.Phases)
	p.printQuickFixGuide(resp.Phases)
	p.printDebugGuides(resp.Phases)
	p.printArtifacts(resp)
	p.printDocs(resp.Phases)
}

func (p *Printer) printPreHeader(phaseNames []string) {
	title := fmt.Sprintf("%s COMPREHENSIVE TEST SUITE", strings.ToUpper(p.scenario))
	paths := repo.DiscoverScenarioPaths(p.scenario)
	estimated := p.estimateTotalFromNames(phaseNames)
	startText := time.Now().Format("15:04:05")
	scenarioPath := p.scenario
	if paths.ScenarioDir != "" {
		scenarioPath = paths.ScenarioDir
	}

	fmt.Fprintln(p.w, p.color.Cyan("╔═══════════════════════════════════════════════════════════════╗"))
	fmt.Fprintf(p.w, "%s  %-61s%s\n", p.color.Cyan("║"), p.color.BoldCyan(title), p.color.Cyan("║"))
	fmt.Fprintln(p.w, p.color.Cyan("╠═══════════════════════════════════════════════════════════════╣"))
	fmt.Fprintf(p.w, "%s  %-61s%s\n", p.color.Cyan("║"), fmt.Sprintf("Scenario: %s", scenarioPath), p.color.Cyan("║"))
	if p.requestedPreset != "" {
		fmt.Fprintf(p.w, "%s  %-61s%s\n", p.color.Cyan("║"), fmt.Sprintf("Preset: %s", p.requestedPreset), p.color.Cyan("║"))
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
	if estimated != "" {
		fmt.Fprintf(p.w, "%s  %-61s%s\n", p.color.Cyan("║"), fmt.Sprintf("Estimated: ~%s", estimated), p.color.Cyan("║"))
	}
	fmt.Fprintf(p.w, "%s  Phases: %-54d%s\n", p.color.Cyan("║"), len(phaseNames), p.color.Cyan("║"))
	fmt.Fprintln(p.w, p.color.Cyan("╚═══════════════════════════════════════════════════════════════╝"))
	fmt.Fprintln(p.w)
}

func (p *Printer) printPrePlan(phaseNames []string) {
	fmt.Fprintln(p.w, p.color.Bold("Test Plan:"))
	if len(phaseNames) == 0 {
		fmt.Fprintln(p.w, "  • (no phases specified)")
		fmt.Fprintln(p.w)
		return
	}
	for idx, name := range phaseNames {
		target := p.targetDuration(name)
		desc := p.lookupPhaseDescription(name)
		doc := p.phaseDocHint(name)
		targetText := ""
		if target != "" {
			targetText = fmt.Sprintf("(±%s)", target)
		}
		line := fmt.Sprintf("  [%d/%d] %-14s %-10s", idx+1, len(phaseNames), name, targetText)
		if desc != "" {
			line = fmt.Sprintf("%s → %s", line, desc)
		}
		if doc != "" {
			line = fmt.Sprintf("%s (docs: %s)", line, doc)
		}
		fmt.Fprintln(p.w, p.color.Cyan(line))
	}
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, p.color.Cyan("Starting execution..."))
	fmt.Fprintln(p.w)
}

func (p *Printer) estimateTotalFromNames(phaseNames []string) string {
	if len(phaseNames) == 0 {
		return ""
	}
	var total time.Duration
	for _, name := range phaseNames {
		if d, ok := p.targetDurationByKey[NormalizeName(name)]; ok && d > 0 {
			total += d
		}
	}
	if total == 0 {
		return ""
	}
	return total.Truncate(time.Second).String()
}

func (p *Printer) printHeader(resp execTypes.Response) {
	title := fmt.Sprintf("%s COMPREHENSIVE TEST SUITE", strings.ToUpper(p.scenario))
	startText := FormatTimestampShort(resp.StartedAt, "unknown")
	finishText := FormatTimestampShort(resp.CompletedAt, "pending")
	duration := FormatRunDuration(resp.PhaseSummary.DurationSeconds, resp.StartedAt, resp.CompletedAt)
	preset := DefaultValue(resp.PresetUsed, p.requestedPreset)
	paths := repo.DiscoverScenarioPaths(p.scenario)
	estimated := p.estimateTotal(resp.Phases)
	scenarioPath := p.scenario
	if paths.ScenarioDir != "" {
		scenarioPath = paths.ScenarioDir
	}

	fmt.Fprintln(p.w, p.color.Cyan("╔═══════════════════════════════════════════════════════════════╗"))
	fmt.Fprintf(p.w, "%s  %-61s%s\n", p.color.Cyan("║"), p.color.BoldCyan(title), p.color.Cyan("║"))
	fmt.Fprintln(p.w, p.color.Cyan("╠═══════════════════════════════════════════════════════════════╣"))
	fmt.Fprintf(p.w, "%s  %-61s%s\n", p.color.Cyan("║"), fmt.Sprintf("Scenario: %s", scenarioPath), p.color.Cyan("║"))
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
		doc := p.phaseDocHint(phase.Name)
		// Format: [1/6] structure       (±120s)  → Description
		targetText := ""
		if target != "" {
			targetText = fmt.Sprintf("(±%s)", target)
		}
		line := fmt.Sprintf("  [%d/%d] %-14s %-10s", idx+1, len(phasesData), phase.Name, targetText)
		if desc != "" {
			line = fmt.Sprintf("%s → %s", line, desc)
		}
		if doc != "" {
			line = fmt.Sprintf("%s (docs: %s)", line, doc)
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

		// Show any observations from the phase (skip if already streamed via SSE)
		if len(phase.Observations) > 0 && !p.streamedObservations {
			for _, obs := range phase.Observations {
				text := strings.TrimSpace(obs.String())
				if text == "" {
					continue
				}
				if obs.IsSection() {
					// Section headers get a blank line before them for visual grouping
					fmt.Fprintf(p.w, "\n%s\n", text)
				} else if obs.Prefix != "" {
					// Prefixed observations already include status icon via String()
					fmt.Fprintf(p.w, "%s\n", text)
				} else {
					// Plain observations get a checkmark
					fmt.Fprintf(p.w, "%s %s\n", p.color.Green("✅"), text)
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
		headline := p.phaseHeadline(phase)
		warnings := WarningObservations(phase.Observations, 3)

		phaseLine := fmt.Sprintf("  %s phase-%-14s •  %-6s", icon, phase.Name, durationText)
		if !strings.EqualFold(status, "PASSED") && headline != "" {
			phaseLine = fmt.Sprintf("%s — %s", phaseLine, headline)
		}
		fmt.Fprintln(p.w, p.color.StatusColor(phase.Status, phaseLine))

		// Always show the log path for quick navigation
		if phase.LogPath != "" {
			fmt.Fprintf(p.w, "     log: %s\n", DescribeLogPath(phase.LogPath))
		}
		if strings.EqualFold(status, "PASSED") && len(warnings) > 0 {
			fmt.Fprintf(p.w, "     warnings: %s\n", strings.Join(warnings, " | "))
		}

		// Failed/error phases get a single-line fix hint and doc pointer
		if !strings.EqualFold(status, "PASSED") {
			if fix := p.phaseFixHint(phase); fix != "" {
				fmt.Fprintf(p.w, "     fix: %s\n", fix)
			}
			if doc := p.phaseDocHint(phase.Name); doc != "" {
				fmt.Fprintf(p.w, "     docs: %s\n", doc)
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
			fmt.Fprintln(p.w, p.color.Cyan("    Quick checks: open coverage/lighthouse/*.html | pnpm run analyze | inspect ui/dist/assets"))
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

	if paths := repo.DiscoverScenarioPaths(p.scenario); paths.ScenarioDir != "" {
		logsDir := filepath.Join(paths.ScenarioDir, "coverage", "logs")
		if repo.Exists(logsDir) {
			fmt.Fprintf(p.w, "Directory: %s\n", p.color.Cyan(logsDir))
		}
	}

	if artifactLines := DescribeArtifacts(resp.Phases); len(artifactLines) > 0 {
		fmt.Fprintln(p.w)
		for _, line := range artifactLines {
			fmt.Fprintf(p.w, "  • %s\n", line)
		}
	}
	for _, line := range DescribeCoverage(p.scenario, resp.Phases) {
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

// phaseDocMapping maps phase names to relevant documentation files.
var phaseDocMapping = map[string][]string{
	"structure":    {"scenarios/test-genie/docs/phases/structure/README.md"},
	"dependencies": {"scenarios/test-genie/docs/phases/dependencies/README.md"},
	"lint":         {"scenarios/test-genie/docs/phases/lint/README.md"},
	"docs":         {"scenarios/test-genie/docs/phases/docs/README.md"},
	"smoke":        {"scenarios/test-genie/docs/phases/smoke/README.md"},
	"unit":         {"scenarios/test-genie/docs/phases/unit/README.md"},
	"integration":  {"scenarios/test-genie/docs/phases/integration/README.md"},
	"playbooks":    {"scenarios/test-genie/docs/phases/playbooks/README.md"},
	"business":     {"scenarios/test-genie/docs/phases/business/README.md"},
	"performance":  {"scenarios/test-genie/docs/phases/performance/README.md"},
}

func phaseDocs(name string) []string {
	rawDocs, ok := phaseDocMapping[NormalizeName(name)]
	if !ok {
		return nil
	}
	var resolved []string
	for _, doc := range rawDocs {
		resolved = append(resolved, repo.AbsPath(doc))
	}
	return resolved
}

func (p *Printer) printDocs(phases []execTypes.Phase) {
	// Collect failed phases
	var failedPhases []string
	for _, phase := range phases {
		if phase.Status == "failed" {
			failedPhases = append(failedPhases, strings.ToLower(phase.Name))
		}
	}

	// Only show docs if there are failures
	if len(failedPhases) == 0 {
		return
	}

	// Collect unique docs relevant to failed phases
	docsSet := make(map[string]struct{})
	for _, phaseName := range failedPhases {
		for _, doc := range phaseDocs(phaseName) {
			docsSet[doc] = struct{}{}
		}
	}

	// Convert to sorted slice
	var docs []string
	for doc := range docsSet {
		docs = append(docs, doc)
	}
	sort.Strings(docs)

	if len(docs) == 0 {
		return
	}

	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, p.color.Bold("📚 DOCUMENTATION:"))
	for _, doc := range docs {
		fmt.Fprintf(p.w, "   • %s\n", p.color.Cyan(doc))
	}
	if paths := repo.DiscoverScenarioPaths(p.scenario); paths.TestDir != "" {
		fmt.Fprintf(p.w, "   • Test files in: %s\n", p.color.Cyan(paths.TestDir))
	}
	fmt.Fprintln(p.w)
}

// phaseHeadline selects a single-line headline for a phase, prioritizing explicit
// errors, then the first interesting line from the log, then classification/remediation.
func (p *Printer) phaseHeadline(phase execTypes.Phase) string {
	if text := strings.TrimSpace(phase.Error); text != "" {
		return firstLine(text)
	}
	if content := ReadLogSnippet(phase.LogPath, 4000); content != "" {
		if line := firstInterestingLine(content); line != "" {
			return line
		}
	}
	if phase.Classification != "" {
		return phase.Classification
	}
	if phase.Remediation != "" {
		return phase.Remediation
	}
	return ""
}

// phaseFixHint returns a concise fix hint for the phase.
func (p *Printer) phaseFixHint(phase execTypes.Phase) string {
	if fix := strings.TrimSpace(phase.Remediation); fix != "" {
		return fix
	}
	if strings.TrimSpace(phase.Error) != "" {
		return "Fix the errors shown above, then rerun execute"
	}
	if phase.Classification != "" {
		return fmt.Sprintf("Resolve %s issue, then rerun execute", phase.Classification)
	}
	return ""
}

// phaseDocHint returns the first doc link for the phase, if any.
func (p *Printer) phaseDocHint(name string) string {
	docs := phaseDocs(name)
	if len(docs) == 0 {
		return ""
	}
	return docs[0]
}

// firstInterestingLine finds the first error-like line in content, falling back to
// the first non-empty line.
func firstInterestingLine(content string) string {
	lines := strings.Split(content, "\n")
	var fallback string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if fallback == "" {
			fallback = trimmed
		}
		lower := strings.ToLower(trimmed)
		if strings.Contains(lower, "error") ||
			strings.Contains(lower, "fail") ||
			strings.Contains(lower, "missing") ||
			strings.Contains(trimmed, "❌") {
			return trimmed
		}
	}
	return fallback
}

// firstLine returns the first non-empty line from a multi-line string.
func firstLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return ""
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
