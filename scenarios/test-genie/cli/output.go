package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

type executionPrinter struct {
	w               io.Writer
	scenario        string
	requestedPreset string
	requestedPhases []string
	descriptorMap   map[string]PhaseDescriptor
}

func newExecutionPrinter(w io.Writer, scenario, requestedPreset string, requestedPhases []string, descriptors []PhaseDescriptor) *executionPrinter {
	descMap := make(map[string]PhaseDescriptor, len(descriptors))
	for _, desc := range descriptors {
		key := strings.ToLower(strings.TrimSpace(desc.Name))
		if key == "" {
			continue
		}
		if _, exists := descMap[key]; !exists {
			descMap[key] = desc
		}
	}
	return &executionPrinter{
		w:               w,
		scenario:        scenario,
		requestedPreset: requestedPreset,
		requestedPhases: requestedPhases,
		descriptorMap:   descMap,
	}
}

func (p *executionPrinter) Print(resp ExecuteResponse) {
	p.printHeader(resp)
	p.printPlan(resp.Phases)
	p.printPhaseResults(resp.Phases)
	p.printSummary(resp)
	p.printFailureDigest(resp.Phases)
	p.printArtifacts(resp)
	p.printDocs()
}

func (p *executionPrinter) printHeader(resp ExecuteResponse) {
	title := fmt.Sprintf("%s TEST EXECUTION", strings.ToUpper(p.scenario))
	startText := defaultValue(resp.StartedAt, "unknown")
	finishText := defaultValue(resp.CompletedAt, "pending")
	duration := formatRunDuration(resp.PhaseSummary.DurationSeconds, resp.StartedAt, resp.CompletedAt)
	preset := defaultValue(resp.PresetUsed, p.requestedPreset)

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
	fmt.Fprintf(p.w, "║  %-61s║\n", fmt.Sprintf("Started: %s", startText))
	fmt.Fprintf(p.w, "║  %-61s║\n", fmt.Sprintf("Finished: %s", finishText))
	fmt.Fprintf(p.w, "║  %-61s║\n", fmt.Sprintf("Duration: %s", duration))
	fmt.Fprintf(p.w, "║  %-61s║\n", fmt.Sprintf("Phases executed: %d", len(resp.Phases)))
	fmt.Fprintln(p.w, "╚═══════════════════════════════════════════════════════════════╝")
	fmt.Fprintln(p.w)
}

func (p *executionPrinter) printPlan(phases []ExecutePhase) {
	fmt.Fprintln(p.w, "Execution plan:")
	if len(phases) == 0 {
		fmt.Fprintln(p.w, "  • (planner returned no phases)")
		fmt.Fprintln(p.w)
		return
	}
	for idx, phase := range phases {
		desc := p.lookupPhaseDescription(phase.Name)
		line := fmt.Sprintf("  [%d/%d] %-12s", idx+1, len(phases), phase.Name)
		if desc != "" {
			line = fmt.Sprintf("%s → %s", line, desc)
		}
		fmt.Fprintln(p.w, line)
	}
	fmt.Fprintln(p.w)
}

func (p *executionPrinter) printPhaseResults(phases []ExecutePhase) {
	fmt.Fprintln(p.w, "Phase results:")
	if len(phases) == 0 {
		fmt.Fprintln(p.w, "  (no phases recorded)")
		return
	}
	for _, phase := range phases {
		icon := statusIcon(phase.Status)
		status := strings.ToUpper(defaultValue(phase.Status, "unknown"))
		fmt.Fprintf(p.w, "  %s %-12s status=%-8s duration=%s\n", icon, phase.Name, status, formatPhaseDuration(phase.DurationSeconds))
		if phase.LogPath != "" {
			fmt.Fprintf(p.w, "      log: %s\n", phase.LogPath)
		}
		if len(phase.Observations) > 0 {
			for _, obs := range phase.Observations {
				if strings.TrimSpace(obs) == "" {
					continue
				}
				fmt.Fprintf(p.w, "      • %s\n", strings.TrimSpace(obs))
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

func (p *executionPrinter) printSummary(resp ExecuteResponse) {
	total := resp.PhaseSummary.Total
	passed := resp.PhaseSummary.Passed
	failed := resp.PhaseSummary.Failed
	duration := formatRunDuration(resp.PhaseSummary.DurationSeconds, resp.StartedAt, resp.CompletedAt)
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

func (p *executionPrinter) printFailureDigest(phases []ExecutePhase) {
	var failed []ExecutePhase
	for _, phase := range phases {
		if !strings.EqualFold(phase.Status, "passed") && !strings.EqualFold(phase.Status, "skipped") {
			failed = append(failed, phase)
		}
	}
	if len(failed) == 0 {
		return
	}
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, "Failure analysis:")
	for _, phase := range failed {
		fmt.Fprintf(p.w, "  ❌ %s\n", strings.ToUpper(phase.Name))
		if phase.Error != "" {
			fmt.Fprintf(p.w, "     error: %s\n", phase.Error)
		}
		if phase.Classification != "" {
			fmt.Fprintf(p.w, "     classification: %s\n", phase.Classification)
		}
		if phase.Remediation != "" {
			fmt.Fprintf(p.w, "     remediation: %s\n", phase.Remediation)
		}
		if phase.LogPath != "" {
			fmt.Fprintf(p.w, "     log: %s\n", phase.LogPath)
		}
		if len(phase.Observations) > 0 {
			fmt.Fprintf(p.w, "     observations:\n")
			for _, obs := range phase.Observations {
				if strings.TrimSpace(obs) == "" {
					continue
				}
				fmt.Fprintf(p.w, "       - %s\n", strings.TrimSpace(obs))
			}
		}
	}
}

func (p *executionPrinter) printArtifacts(resp ExecuteResponse) {
	if len(resp.Links) == 0 && len(resp.Metadata) == 0 {
		return
	}
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, "Artifacts & metadata:")
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
}

func (p *executionPrinter) printDocs() {
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, "Reference docs:")
	fmt.Fprintln(p.w, "  • docs/testing/architecture/PHASED_TESTING.md")
	fmt.Fprintln(p.w, "  • docs/testing/guides/requirement-tracking-quick-start.md")
	fmt.Fprintln(p.w, "  • docs/testing/guides/ui-automation-with-bas.md")
	fmt.Fprintln(p.w)
}

func (p *executionPrinter) lookupPhaseDescription(name string) string {
	desc, ok := p.descriptorMap[strings.ToLower(name)]
	if !ok {
		return ""
	}
	return desc.Description
}

func statusIcon(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "passed", "success":
		return "✓"
	case "skipped":
		return "↷"
	case "failed", "error":
		return "✗"
	default:
		return "•"
	}
}

func formatPhaseDuration(seconds float64) string {
	if seconds <= 0 {
		return "0s"
	}
	return fmt.Sprintf("%.1fs", seconds)
}

func formatRunDuration(summarySeconds int, started, completed string) string {
	if summarySeconds > 0 {
		return (time.Duration(summarySeconds) * time.Second).String()
	}
	start, okStart := parseTime(started)
	complete, okComplete := parseTime(completed)
	if okStart && okComplete && complete.After(start) {
		return complete.Sub(start).String()
	}
	return "n/a"
}

func parseTime(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, true
	}
	return time.Time{}, false
}
