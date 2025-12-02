package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type executionPrinter struct {
	w                   io.Writer
	scenario            string
	requestedPreset     string
	requestedPhases     []string
	requestedSkip       []string
	failFast            bool
	descriptorMap       map[string]PhaseDescriptor
	targetDurationByKey map[string]time.Duration
}

func newExecutionPrinter(
	w io.Writer,
	scenario,
	requestedPreset string,
	requestedPhases,
	requestedSkip []string,
	failFast bool,
	descriptors []PhaseDescriptor,
) *executionPrinter {
	descMap, targets := makeDescriptorMaps(descriptors)
	return &executionPrinter{
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

func (p *executionPrinter) Print(resp ExecuteResponse) {
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

func (p *executionPrinter) printHeader(resp ExecuteResponse) {
	title := fmt.Sprintf("%s TEST EXECUTION", strings.ToUpper(p.scenario))
	startText := defaultValue(resp.StartedAt, "unknown")
	finishText := defaultValue(resp.CompletedAt, "pending")
	duration := formatRunDuration(resp.PhaseSummary.DurationSeconds, resp.StartedAt, resp.CompletedAt)
	preset := defaultValue(resp.PresetUsed, p.requestedPreset)
	paths := discoverScenarioPaths(p.scenario)
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

func (p *executionPrinter) printPlan(phases []ExecutePhase) {
	fmt.Fprintln(p.w, "Execution plan:")
	if len(phases) == 0 {
		fmt.Fprintln(p.w, "  • (planner returned no phases)")
		fmt.Fprintln(p.w)
		return
	}
	for idx, phase := range phases {
		desc := p.lookupPhaseDescription(phase.Name)
		target := p.targetDuration(phase.Name)
		line := fmt.Sprintf("  [%d/%d] %-12s", idx+1, len(phases), phase.Name)
		if desc != "" {
			line = fmt.Sprintf("%s → %s", line, desc)
		}
		if target != "" {
			line = fmt.Sprintf("%s (target: %s)", line, target)
		}
		fmt.Fprintln(p.w, line)
	}
	if est := p.estimateTotal(phases); est != "" {
		fmt.Fprintf(p.w, "  • Estimated total time: %s\n", est)
	}
	fmt.Fprintln(p.w)
}

func (p *executionPrinter) printPhaseProgress(phases []ExecutePhase) {
	if len(phases) == 0 {
		return
	}
	fmt.Fprintln(p.w, "Phase progress:")
	var cumulative time.Duration
	for idx, phase := range phases {
		key := normalizeName(phase.Name)
		duration := time.Duration(phase.DurationSeconds * float64(time.Second))
		if duration < 0 {
			duration = 0
		}
		target := p.targetDuration(key)
		startMark := fmt.Sprintf("t+%s", cumulative.Truncate(time.Millisecond).String())
		cumulative += duration
		endMark := fmt.Sprintf("t+%s", cumulative.Truncate(time.Millisecond).String())
		status := strings.ToUpper(defaultValue(phase.Status, "pending"))
		fmt.Fprintf(p.w, "  [%d] %-12s %s (%s → %s", idx+1, key, status, startMark, endMark)
		if target != "" {
			fmt.Fprintf(p.w, ", target %s", target)
		}
		fmt.Fprintln(p.w, ")")
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
		warnings := ""
		if phase.LogPath != "" {
			if warns := countLinesContaining(phase.LogPath, "[WARNING"); warns > 0 {
				warnings = fmt.Sprintf(" warnings=%d", warns)
			}
		}
		target := p.targetDuration(phase.Name)
		targetText := ""
		if target != "" {
			targetText = fmt.Sprintf(" target=%s", target)
		}
		fmt.Fprintf(p.w, "  %s %-12s status=%-8s duration=%s%s%s\n", icon, phase.Name, status, formatPhaseDuration(phase.DurationSeconds), targetText, warnings)
		if phase.LogPath != "" {
			exists, empty := fileState(phase.LogPath)
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
			if snippet := readLogSnippet(phase.LogPath, 2000); snippet != "" {
				fmt.Fprintf(p.w, "      log snippet:\n")
				for _, line := range tailLines(snippet, 12) {
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
	insights := analyzePhaseFailures(failed)
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

	if cp := summarizeCriticalPath(insights); cp != nil {
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

func (p *executionPrinter) printArtifacts(resp ExecuteResponse) {
	if len(resp.Links) == 0 && len(resp.Metadata) == 0 {
		paths := collectArtifactRoots(resp.Phases)
		if len(paths) == 0 {
			return
		}
	}
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, "Artifacts & metadata:")

	if paths := discoverScenarioPaths(p.scenario); paths.TestDir != "" {
		artifactDir := filepath.Join(paths.TestDir, "artifacts")
		if exists(artifactDir) {
			fmt.Fprintf(p.w, "  • phase logs: %s\n", artifactDir)
		}
	}

	if artifactLines := p.describeArtifacts(resp.Phases); len(artifactLines) > 0 {
		for _, line := range artifactLines {
			fmt.Fprintf(p.w, "  • %s\n", line)
		}
	}
	for _, line := range describeCoverage(p.scenario) {
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

	if diag := diagnoseFailures(resp.Phases); diag != nil {
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

func (p *executionPrinter) printDocs() {
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, "Reference docs:")
	fmt.Fprintln(p.w, "  • docs/testing/architecture/PHASED_TESTING.md")
	fmt.Fprintln(p.w, "  • docs/testing/guides/requirement-tracking-quick-start.md")
	fmt.Fprintln(p.w, "  • docs/testing/guides/ui-automation-with-bas.md")
	fmt.Fprintln(p.w, "  • docs/testing/guides/writing-testable-uis.md")
	fmt.Fprintln(p.w)
}

func (p *executionPrinter) lookupPhaseDescription(name string) string {
	desc, ok := p.descriptorMap[strings.ToLower(name)]
	if !ok {
		return ""
	}
	return desc.Description
}

func (p *executionPrinter) targetDuration(name string) string {
	if d, ok := p.targetDurationByKey[normalizeName(name)]; ok && d > 0 {
		return d.Truncate(time.Second).String()
	}
	return ""
}

func (p *executionPrinter) estimateTotal(phases []ExecutePhase) string {
	if len(phases) == 0 {
		return ""
	}
	var total time.Duration
	for _, phase := range phases {
		if d, ok := p.targetDurationByKey[normalizeName(phase.Name)]; ok && d > 0 {
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

func fileState(path string) (exists bool, empty bool) {
	tryPath := func(candidate string) (bool, bool) {
		info, err := os.Stat(candidate)
		if err != nil {
			return false, false
		}
		if info.IsDir() {
			return true, true
		}
		return true, info.Size() == 0
	}
	if exists, empty := tryPath(path); exists {
		return exists, empty
	}
	if filepath.IsAbs(path) {
		return false, false
	}
	if root := detectRepoRoot(); root != "" {
		return tryPath(filepath.Join(root, path))
	}
	return false, false
}

func collectArtifactRoots(phases []ExecutePhase) []string {
	roots := make(map[string]struct{})
	for _, phase := range phases {
		if phase.LogPath == "" {
			continue
		}
		dir := filepath.Dir(phase.LogPath)
		if dir == "." || dir == "/" {
			continue
		}
		roots[dir] = struct{}{}
	}
	result := make([]string, 0, len(roots))
	for dir := range roots {
		result = append(result, dir)
	}
	sort.Strings(result)
	return result
}

func (p *executionPrinter) describeArtifacts(phases []ExecutePhase) []string {
	roots := collectArtifactRoots(phases)
	if len(roots) == 0 {
		return nil
	}
	var lines []string
	lines = append(lines, fmt.Sprintf("artifact roots: %s", strings.Join(roots, ", ")))

	totalLogs := 0
	presentLogs := 0
	emptyLogs := 0
	warningCount := 0
	for _, phase := range phases {
		if phase.LogPath == "" {
			continue
		}
		totalLogs++
		exists, empty := fileState(phase.LogPath)
		if exists {
			presentLogs++
		}
		if empty {
			emptyLogs++
		}
		if exists && !empty {
			warningCount += countLinesContaining(phase.LogPath, "[WARNING")
		}
	}
	if totalLogs > 0 {
		lines = append(lines, fmt.Sprintf("logs: %d total • %d present • %d empty • %d warnings", totalLogs, presentLogs, emptyLogs, warningCount))
	}
	return lines
}

func describeCoverage(scenario string) []string {
	paths := discoverScenarioPaths(scenario)
	if paths.ScenarioDir == "" {
		return nil
	}
	var lines []string
	coverageDirs := []string{
		filepath.Join(paths.ScenarioDir, "coverage", "unit", "go"),
		filepath.Join(paths.ScenarioDir, "coverage", "integration"),
		filepath.Join(paths.ScenarioDir, "coverage", "test-genie"),
	}
	for _, dir := range coverageDirs {
		if exists(dir) {
			lines = append(lines, fmt.Sprintf("coverage: %s", dir))
		}
	}
	lighthouse := filepath.Join(paths.ScenarioDir, "test", "artifacts", "lighthouse")
	if exists(lighthouse) {
		lines = append(lines, fmt.Sprintf("lighthouse: %s", lighthouse))
	}
	reqIndex := filepath.Join(paths.ScenarioDir, "requirements", "index.json")
	if exists(reqIndex) {
		lines = append(lines, fmt.Sprintf("requirements index: %s", reqIndex))
	}
	goHTML := filepath.Join(paths.ScenarioDir, "coverage", "unit", "go", "coverage.html")
	if exists(goHTML) {
		lines = append(lines, fmt.Sprintf("go coverage html: %s", goHTML))
	}
	nodeHTML := filepath.Join(paths.ScenarioDir, "ui", "coverage", "lcov-report", "index.html")
	if exists(nodeHTML) {
		lines = append(lines, fmt.Sprintf("node coverage html: %s", nodeHTML))
	}
	return lines
}

func countLinesContaining(path, substr string) int {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return 0
	}
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, substr) {
			count++
		}
	}
	return count
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func describeLogPath(path string) string {
	if path == "" {
		return ""
	}
	exists, empty := fileState(path)
	switch {
	case !exists:
		return fmt.Sprintf("%s (missing)", path)
	case empty:
		return fmt.Sprintf("%s (empty)", path)
	default:
		return path
	}
}

func cleanObservations(obs []string) []string {
	var cleaned []string
	for _, o := range obs {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		cleaned = append(cleaned, o)
	}
	return cleaned
}

func makeDescriptorMaps(descriptors []PhaseDescriptor) (map[string]PhaseDescriptor, map[string]time.Duration) {
	descMap := make(map[string]PhaseDescriptor, len(descriptors))
	targets := make(map[string]time.Duration, len(descriptors))
	for _, desc := range descriptors {
		key := strings.ToLower(strings.TrimSpace(desc.Name))
		if key == "" {
			continue
		}
		if _, exists := descMap[key]; !exists {
			descMap[key] = desc
		}
		if desc.DefaultTimeoutSeconds > 0 {
			targets[key] = time.Duration(desc.DefaultTimeoutSeconds) * time.Second
		}
	}
	if len(targets) == 0 {
		targets = map[string]time.Duration{
			"structure":    120 * time.Second,
			"dependencies": 60 * time.Second,
			"unit":         120 * time.Second,
			"integration":  600 * time.Second,
			"business":     120 * time.Second,
			"performance":  60 * time.Second,
		}
	}
	return descMap, targets
}

var (
	repoRootOnce sync.Once
	repoRootPath string
)

func detectRepoRoot() string {
	repoRootOnce.Do(func() {
		dir, _ := os.Getwd()
		repoRootPath = locateRepoRoot(dir)
	})
	return repoRootPath
}

func locateRepoRoot(start string) string {
	dir := start
	for i := 0; i < 8 && dir != "" && dir != string(filepath.Separator); i++ {
		if dir == "" {
			break
		}
		if exists(filepath.Join(dir, ".git")) {
			return dir
		}
		if exists(filepath.Join(dir, "pnpm-workspace.yaml")) {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	return ""
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

type scenarioPaths struct {
	ScenarioDir string
	TestDir     string
}

func discoverScenarioPaths(scenario string) scenarioPaths {
	root := detectRepoRoot()
	if root == "" {
		return scenarioPaths{}
	}
	scenarioDir := filepath.Join(root, "scenarios", scenario)
	info, err := os.Stat(scenarioDir)
	if err != nil || !info.IsDir() {
		return scenarioPaths{}
	}
	testDir := filepath.Join(scenarioDir, "test")
	if _, err := os.Stat(testDir); err != nil {
		testDir = ""
	}
	return scenarioPaths{ScenarioDir: scenarioDir, TestDir: testDir}
}

type failureDiagnosis struct {
	Primary         string
	PrimaryPhase    string
	Details         string
	ImpactedPhases  []string
	SecondaryIssues []string
	QuickFixes      []string
}

type phaseInsight struct {
	Phase        string
	Cause        string
	Detail       string
	Impact       []string
	Fixes        []string
	Log          string
	Observations []string
}

type criticalPath struct {
	PrimaryPhase  string
	PrimaryCause  string
	Detail        string
	BlockedPhases []string
	QuickFixes    []string
}

func analyzePhaseFailures(phases []ExecutePhase) []phaseInsight {
	patterns := map[string]*regexp.Regexp{
		"typescript": regexp.MustCompile(`(?i)typescript|error TS[0-9]+|Cannot find name`),
		"ui_bundle":  regexp.MustCompile(`(?i)bundle.*stale|UI bundle.*outdated|Bundle Status:.*stale`),
		"api_port":   regexp.MustCompile(`(?i)API_PORT|HTTP 502|Bad Gateway|not responding`),
		"build":      regexp.MustCompile(`(?i)build .*failed|compilation .*failed|ELIFECYCLE`),
		"timeout":    regexp.MustCompile(`(?i)timed out|timeout|exceeded .*time`),
		"tests":      regexp.MustCompile(`(?i)❌|test failed|tests failed`),
		"missing":    regexp.MustCompile(`(?i)required (file|directory) missing|no such file or directory|expected file but found directory|expected directory but found file`),
	}

	var insights []phaseInsight
	for _, phase := range phases {
		content := readLogSnippet(phase.LogPath, 48_000)
		insight := phaseInsight{
			Phase:        phase.Name,
			Log:          describeLogPath(phase.LogPath),
			Observations: cleanObservations(phase.Observations),
		}

		switch {
		case patterns["typescript"].MatchString(content):
			insight.Cause = "TypeScript compilation error"
			insight.Detail = extractLine(content, patterns["typescript"])
			insight.Impact = []string{"integration", "performance"}
			insight.Fixes = append(insight.Fixes,
				"Fix TypeScript compiler errors (see log)",
				"Rebuild UI and rerun: test-genie execute <scenario>")
		case patterns["ui_bundle"].MatchString(content):
			insight.Cause = "Stale or missing UI bundle"
			insight.Detail = extractLine(content, patterns["ui_bundle"])
			insight.Impact = []string{"integration", "performance"}
			insight.Fixes = append(insight.Fixes,
				"Rebuild the UI bundle",
				"Restart the scenario then rerun execute")
		case patterns["api_port"].MatchString(content):
			insight.Cause = "Scenario API unreachable"
			insight.Detail = extractLine(content, patterns["api_port"])
			insight.Impact = []string{"integration"}
			insight.Fixes = append(insight.Fixes,
				"Restart the scenario: vrooli scenario restart <scenario>",
				"Verify API_PORT resolution and retry")
		case patterns["build"].MatchString(content):
			insight.Cause = "Build failure"
			insight.Detail = extractLine(content, patterns["build"])
			insight.Fixes = append(insight.Fixes, "Fix build errors shown in the log, then rerun execute")
		case patterns["timeout"].MatchString(content):
			insight.Cause = "Phase timeout"
			insight.Detail = extractLine(content, patterns["timeout"])
			insight.Fixes = append(insight.Fixes, "Inspect long-running task in the log, rerun with --fail-fast to isolate")
		case patterns["tests"].MatchString(content):
			insight.Cause = "Tests failed"
			insight.Detail = extractLine(content, patterns["tests"])
			insight.Fixes = append(insight.Fixes, "Open the phase log and fix failing tests, then rerun execute")
		case patterns["missing"].MatchString(content):
			insight.Cause = "Required file/directory missing"
			insight.Detail = extractLine(content, patterns["missing"])
			insight.Fixes = append(insight.Fixes,
				"Restore the missing path referenced in the error",
				"Update .vrooli/testing.json structure expectations if the layout is intentionally different")
		default:
			// Fall back to error fields when the log lacks the error line.
			if patterns["missing"].MatchString(phase.Error) {
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

func summarizeCriticalPath(insights []phaseInsight) *criticalPath {
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

	return &criticalPath{
		PrimaryPhase:  primary.Phase,
		PrimaryCause:  defaultValue(primary.Cause, "Unknown issue"),
		Detail:        primary.Detail,
		BlockedPhases: blocked,
		QuickFixes:    fixes,
	}
}

func (p *executionPrinter) printQuickFixGuide(phases []ExecutePhase) {
	failed := filterFailedPhases(phases)
	if len(failed) == 0 {
		return
	}
	insights := analyzePhaseFailures(failed)
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

func (p *executionPrinter) printDebugGuides(phases []ExecutePhase) {
	failed := filterFailedPhases(phases)
	if len(failed) < 2 {
		return
	}
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, "Phase-specific debug guides:")
	for _, phase := range failed {
		switch normalizeName(phase.Name) {
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

func filterFailedPhases(phases []ExecutePhase) []ExecutePhase {
	var failed []ExecutePhase
	for _, phase := range phases {
		if strings.EqualFold(phase.Status, "passed") || strings.EqualFold(phase.Status, "skipped") {
			continue
		}
		failed = append(failed, phase)
	}
	return failed
}

func diagnoseFailures(phases []ExecutePhase) *failureDiagnosis {
	var failed []ExecutePhase
	for _, phase := range phases {
		if strings.EqualFold(phase.Status, "passed") || strings.EqualFold(phase.Status, "skipped") {
			continue
		}
		failed = append(failed, phase)
	}
	if len(failed) == 0 {
		return nil
	}

	diag := &failureDiagnosis{}
	patterns := map[string]*regexp.Regexp{
		"typescript": regexp.MustCompile(`(?i)typescript|error TS[0-9]+|Cannot find name`),
		"ui_bundle":  regexp.MustCompile(`(?i)bundle.*stale|UI bundle.*outdated`),
		"api_port":   regexp.MustCompile(`(?i)API_PORT|HTTP 502|Bad Gateway|not responding`),
		"build":      regexp.MustCompile(`(?i)build .*failed|compilation .*failed|ELIFECYCLE`),
		"timeout":    regexp.MustCompile(`(?i)timed out|timeout|exceeded .*time`),
		"tests":      regexp.MustCompile(`(?i)❌|test failed|tests failed`),
		"missing":    regexp.MustCompile(`(?i)required (file|directory) missing|no such file or directory|expected file but found directory|expected directory but found file`),
	}

	for _, phase := range failed {
		content := readLogSnippet(phase.LogPath, 48_000)
		switch {
		case diag.Primary == "" && patterns["typescript"].MatchString(content):
			diag.Primary = "TypeScript compilation error"
			diag.PrimaryPhase = phase.Name
			diag.Details = extractLine(content, patterns["typescript"])
			diag.ImpactedPhases = append(diag.ImpactedPhases, "integration", "performance")
			diag.QuickFixes = append(diag.QuickFixes,
				"Fix TypeScript compiler errors (see log snippet)",
				"Rebuild/restart scenario, then re-run: test-genie execute <scenario>")
		case diag.Primary == "" && patterns["ui_bundle"].MatchString(content):
			diag.Primary = "Stale or missing UI bundle"
			diag.PrimaryPhase = phase.Name
			diag.Details = extractLine(content, patterns["ui_bundle"])
			diag.ImpactedPhases = append(diag.ImpactedPhases, "integration", "performance")
			diag.QuickFixes = append(diag.QuickFixes,
				"Rebuild the UI bundle and restart the scenario",
				"Re-run: test-genie execute <scenario> --preset quick")
		case diag.Primary == "" && patterns["api_port"].MatchString(content):
			diag.Primary = "Scenario API unreachable"
			diag.PrimaryPhase = phase.Name
			diag.Details = extractLine(content, patterns["api_port"])
			diag.ImpactedPhases = append(diag.ImpactedPhases, "integration")
			diag.QuickFixes = append(diag.QuickFixes,
				"Restart the scenario: vrooli scenario restart <scenario>",
				"Verify API_PORT resolution and retry execute")
		case diag.Primary == "" && patterns["build"].MatchString(content):
			diag.Primary = "Build failure"
			diag.PrimaryPhase = phase.Name
			diag.Details = extractLine(content, patterns["build"])
			diag.QuickFixes = append(diag.QuickFixes,
				"Fix build errors shown in the log and rerun execute")
		case diag.Primary == "" && patterns["timeout"].MatchString(content):
			diag.Primary = "Phase timeout"
			diag.PrimaryPhase = phase.Name
			diag.Details = extractLine(content, patterns["timeout"])
			diag.QuickFixes = append(diag.QuickFixes,
				"Investigate long-running tasks in the phase log, rerun with --fail-fast to isolate")
		case diag.Primary == "" && (patterns["missing"].MatchString(content) || patterns["missing"].MatchString(phase.Error)):
			diag.Primary = "Required file/directory missing"
			diag.PrimaryPhase = phase.Name
			diag.Details = extractLine(content, patterns["missing"])
			if diag.Details == "" {
				diag.Details = strings.TrimSpace(phase.Error)
			}
			diag.QuickFixes = append(diag.QuickFixes,
				"Restore the missing path referenced in the log",
				"Update .vrooli/testing.json structure expectations if the layout is intentionally different")
		}

		if patterns["tests"].MatchString(content) {
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

func readLogSnippet(path string, maxBytes int) string {
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return ""
	}
	if len(data) > maxBytes {
		data = data[len(data)-maxBytes:]
	}
	return string(data)
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

func tailLines(content string, max int) []string {
	if max <= 0 {
		return nil
	}
	lines := strings.Split(content, "\n")
	if len(lines) <= max {
		return lines
	}
	return lines[len(lines)-max:]
}

type logTailer struct {
	dir     string
	w       io.Writer
	offsets map[string]int64
	stop    chan struct{}
	wg      sync.WaitGroup
}

func startLogTailer(w io.Writer, dir string) (*logTailer, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, fmt.Errorf("log directory is empty")
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create log directory: %w", err)
		}
	}
	t := &logTailer{
		dir:     dir,
		w:       w,
		offsets: make(map[string]int64),
		stop:    make(chan struct{}),
	}
	t.wg.Add(1)
	go t.run()
	return t, nil
}

func (t *logTailer) Stop() {
	close(t.stop)
	t.wg.Wait()
}

func (t *logTailer) run() {
	defer t.wg.Done()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-t.stop:
			return
		case <-ticker.C:
			t.tailNewest()
		}
	}
}

func (t *logTailer) tailNewest() {
	path := t.findNewestLog()
	if path == "" {
		return
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return
	}
	prev := t.offsets[path]
	size := info.Size()
	if size == prev {
		return
	}
	if size < prev {
		prev = 0
	}
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	if _, err := f.Seek(prev, io.SeekStart); err != nil {
		return
	}
	buf := make([]byte, size-prev)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return
	}
	if n > 0 {
		rel := path
		if root := detectRepoRoot(); root != "" {
			if relPath, err := filepath.Rel(root, path); err == nil {
				rel = relPath
			}
		}
		fmt.Fprintf(t.w, "\n[stream] %s\n%s", rel, string(buf[:n]))
	}
	t.offsets[path] = size
}

func (t *logTailer) findNewestLog() string {
	entries, err := os.ReadDir(t.dir)
	if err != nil || len(entries) == 0 {
		return ""
	}
	type candidate struct {
		name    string
		modTime time.Time
	}
	var files []candidate
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, candidate{
			name:    filepath.Join(t.dir, entry.Name()),
			modTime: info.ModTime(),
		})
	}
	if len(files) == 0 {
		return ""
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})
	return files[0].name
}
