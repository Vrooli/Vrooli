package phases

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/shared"
)

const (
	defaultStandardsFailOn      = "high"
	defaultStandardsSummaryTopN = 20
	defaultStandardsMinDisplay  = "medium"
)

type auditorAuditEnvelope struct {
	Standards *auditorSummaryResponse `json:"standards"`
}

type auditorSummaryResponse struct {
	Summary auditorViolationSummary `json:"summary"`
}

type auditorScanArtifactRef struct {
	Path string `json:"path"`
}

type auditorRuleCount struct {
	RuleID   string `json:"rule_id"`
	Count    int    `json:"count"`
	Title    string `json:"title,omitempty"`
	Severity string `json:"severity,omitempty"`
}

type auditorViolationExcerpt struct {
	Severity   string `json:"severity"`
	RuleID     string `json:"rule_id,omitempty"`
	Title      string `json:"title,omitempty"`
	FilePath   string `json:"file_path,omitempty"`
	LineNumber int    `json:"line_number,omitempty"`
}

type auditorViolationSummary struct {
	Total           int                       `json:"total"`
	BySeverity      map[string]int            `json:"by_severity"`
	ByRule          []auditorRuleCount        `json:"by_rule"`
	HighestSeverity string                    `json:"highest_severity"`
	TopViolations   []auditorViolationExcerpt `json:"top_violations"`
	Artifact        *auditorScanArtifactRef   `json:"artifact,omitempty"`
	Recommended     []string                  `json:"recommended_steps,omitempty"`
}

func runStandardsPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if report := CheckContext(ctx); report != nil {
		return *report
	}

	cleanLog := wrapLogSansANSI(logWriter)

	auditorBin, err := resolveScenarioAuditorBinary(env)
	if err != nil {
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassMissingDependency,
			Remediation:           "Install the scenario-auditor CLI (or ensure it is on PATH) so Test Genie can enforce standards.",
			Observations: []Observation{
				NewErrorObservation("scenario-auditor CLI is missing"),
			},
		}
	}

	failOn := normalizeSeverity(os.Getenv("TEST_GENIE_STANDARDS_FAIL_ON"))
	if failOn == "" {
		failOn = defaultStandardsFailOn
	}
	summaryLimit := envInt("TEST_GENIE_STANDARDS_LIMIT", defaultStandardsSummaryTopN)
	minDisplay := normalizeSeverity(os.Getenv("TEST_GENIE_STANDARDS_MIN_SEVERITY"))
	if minDisplay == "" {
		minDisplay = defaultStandardsMinDisplay
	}

	timeoutSeconds := auditTimeoutSeconds(ctx, 60)
	if timeoutSeconds <= 0 {
		timeoutSeconds = 60
	}

	shared.LogStep(cleanLog, "running standards audit via scenario-auditor (timeout=%ds, fail_on=%s)", timeoutSeconds, failOn)
	output, err := phaseCommandCapture(
		ctx,
		env.AppRoot,
		cleanLog,
		auditorBin,
		"audit",
		env.ScenarioName,
		"--standards-only",
		"--timeout",
		strconv.Itoa(timeoutSeconds),
		"--limit",
		strconv.Itoa(summaryLimit),
		"--min-severity",
		"info",
		"--json",
	)

	summary, parseErr := parseAuditorStandardsSummary(output)
	observations := buildStandardsObservations(summary, failOn, minDisplay)

	if parseErr != nil {
		if err == nil {
			err = parseErr
		}
		classification, remediation := classifyAuditorError(err)
		writePhasePointer(env, "standards", RunReport{
			Err:                   err,
			FailureClassification: classification,
			Remediation:           remediation,
			Observations:          observations,
		}, nil, cleanLog)
		return RunReport{
			Err:                   err,
			FailureClassification: classification,
			Remediation:           remediation,
			Observations:          observations,
		}
	}

	failedThreshold := violatesFailOn(summary.HighestSeverity, failOn)
	if err != nil || failedThreshold {
		if err == nil && failedThreshold {
			err = fmt.Errorf("standards violations exceed fail_on=%s (highest=%s)", failOn, summary.HighestSeverity)
		}
		classification, remediation := classifyAuditorError(err)
		if failedThreshold {
			classification = FailureClassMisconfiguration
			remediation = fmt.Sprintf("Run `scenario-auditor audit %s --standards-only --timeout %d` and address %s+ findings.", env.ScenarioName, timeoutSeconds, strings.ToUpper(failOn))
		}

		extras := map[string]any{
			"summary": map[string]any{
				"total":            summary.Total,
				"highest_severity": summary.HighestSeverity,
			},
		}
		writePhasePointer(env, "standards", RunReport{
			Err:                   err,
			FailureClassification: classification,
			Remediation:           remediation,
			Observations:          observations,
		}, extras, cleanLog)
		return RunReport{
			Err:                   err,
			FailureClassification: classification,
			Remediation:           remediation,
			Observations:          observations,
		}
	}

	extras := map[string]any{
		"summary": map[string]any{
			"total":            summary.Total,
			"highest_severity": summary.HighestSeverity,
		},
	}
	report := RunReport{Observations: observations}
	writePhasePointer(env, "standards", report, extras, cleanLog)
	return report
}

func resolveScenarioAuditorBinary(env workspace.Environment) (string, error) {
	if _, err := commandLookup("scenario-auditor"); err == nil {
		return "scenario-auditor", nil
	}

	appRoot := strings.TrimSpace(env.AppRoot)
	if appRoot == "" && strings.TrimSpace(env.ScenarioDir) != "" {
		appRoot = workspace.AppRootFromScenario(env.ScenarioDir)
	}
	if appRoot == "" {
		return "", fmt.Errorf("required command 'scenario-auditor' is not available (and app root is unknown)")
	}

	candidates := []string{
		filepath.Join(appRoot, "scenarios", "scenario-auditor", "cli", "scenario-auditor"),
		filepath.Join(appRoot, "scenarios", "scenario-auditor", "cli", "scenario-auditor.sh"),
	}
	for _, candidate := range candidates {
		if err := ensureExecutable(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("required command 'scenario-auditor' is not available on PATH and no repo-local CLI was found under %s", filepath.Join(appRoot, "scenarios", "scenario-auditor", "cli"))
}

func parseAuditorStandardsSummary(raw string) (*auditorViolationSummary, error) {
	payload := strings.TrimSpace(raw)
	if payload == "" {
		return nil, fmt.Errorf("scenario-auditor produced no output")
	}

	var envelope auditorAuditEnvelope
	if err := ParseJSON(payload, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse scenario-auditor JSON: %w", err)
	}
	if envelope.Standards == nil {
		return nil, fmt.Errorf("scenario-auditor JSON missing 'standards' payload")
	}

	summary := envelope.Standards.Summary
	if summary.BySeverity == nil {
		summary.BySeverity = map[string]int{}
	}
	summary.HighestSeverity = normalizeSeverity(summary.HighestSeverity)
	if summary.HighestSeverity == "" && summary.Total > 0 {
		summary.HighestSeverity = "info"
	}
	if len(summary.BySeverity) > 0 {
		normalized := make(map[string]int, len(summary.BySeverity))
		for k, v := range summary.BySeverity {
			key := normalizeSeverity(k)
			if key == "" {
				key = strings.ToLower(strings.TrimSpace(k))
				if key == "" {
					continue
				}
			}
			normalized[key] += v
		}
		summary.BySeverity = normalized
	}
	for i := range summary.ByRule {
		summary.ByRule[i].Severity = normalizeSeverity(summary.ByRule[i].Severity)
	}
	for i := range summary.TopViolations {
		summary.TopViolations[i].Severity = normalizeSeverity(summary.TopViolations[i].Severity)
	}

	return &summary, nil
}

func buildStandardsObservations(summary *auditorViolationSummary, failOn, minDisplay string) []Observation {
	obs := []Observation{
		NewSectionObservation("📏", "Standards"),
	}
	if summary == nil {
		return append(obs, NewErrorObservation("No standards summary available"))
	}

	highest := summary.HighestSeverity
	if highest == "" {
		highest = "none"
	}
	obs = append(obs, NewInfoObservation(fmt.Sprintf("Total violations: %d (highest=%s, fail_on=%s+)", summary.Total, highest, failOn)))

	if len(summary.BySeverity) > 0 {
		obs = append(obs, NewInfoObservation("By severity: "+formatSeverityCounts(summary.BySeverity)))
	}

	if summary.Artifact != nil && strings.TrimSpace(summary.Artifact.Path) != "" {
		obs = append(obs, NewInfoObservation("Artifact: "+strings.TrimSpace(summary.Artifact.Path)))
	}

	if len(summary.ByRule) > 0 {
		limit := 5
		if len(summary.ByRule) < limit {
			limit = len(summary.ByRule)
		}
		var parts []string
		for _, rc := range summary.ByRule[:limit] {
			label := rc.RuleID
			if strings.TrimSpace(rc.Title) != "" {
				label = fmt.Sprintf("%s (%s)", rc.RuleID, strings.TrimSpace(rc.Title))
			}
			parts = append(parts, fmt.Sprintf("%s=%d", label, rc.Count))
		}
		obs = append(obs, NewInfoObservation("Top rules: "+strings.Join(parts, ", ")))
	}

	if len(summary.TopViolations) > 0 {
		obs = append(obs, NewSectionObservation("🔎", "Top Violations"))
		for _, v := range summary.TopViolations {
			if !shouldDisplaySeverity(v.Severity, minDisplay) {
				continue
			}
			line := v.FilePath
			if v.LineNumber > 0 {
				line = fmt.Sprintf("%s:%d", line, v.LineNumber)
			}
			title := strings.TrimSpace(v.Title)
			if title == "" {
				title = v.RuleID
			}
			msg := fmt.Sprintf("[%s] %s -> %s", strings.ToUpper(v.Severity), title, line)
			if violatesFailOn(v.Severity, failOn) {
				obs = append(obs, NewErrorObservation(msg))
			} else {
				obs = append(obs, NewWarningObservation(msg))
			}
		}
	}

	if violatesFailOn(summary.HighestSeverity, failOn) {
		obs = append(obs, NewErrorObservation(fmt.Sprintf("Standards violations include %s+ severity findings", strings.ToUpper(failOn))))
	} else if summary.Total > 0 {
		obs = append(obs, NewWarningObservation("Standards violations detected (below fail threshold)"))
	} else {
		obs = append(obs, NewSuccessObservation("No standards violations detected"))
	}

	return obs
}

func auditTimeoutSeconds(ctx context.Context, fallback int) int {
	if fallback <= 0 {
		fallback = 60
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return fallback
	}
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return 0
	}
	seconds := int(remaining.Round(time.Second).Seconds())
	if seconds <= 0 {
		seconds = 1
	}
	return seconds
}

func envInt(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func normalizeSeverity(raw string) string {
	sev := strings.ToLower(strings.TrimSpace(raw))
	switch sev {
	case "critical", "high", "medium", "low", "info":
		return sev
	case "informational":
		return "info"
	default:
		return ""
	}
}

func severityWeight(sev string) int {
	switch normalizeSeverity(sev) {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium":
		return 3
	case "low":
		return 2
	case "info":
		return 1
	default:
		return 0
	}
}

func violatesFailOn(severity, failOn string) bool {
	highest := severityWeight(severity)
	threshold := severityWeight(failOn)
	if threshold == 0 {
		threshold = severityWeight(defaultStandardsFailOn)
	}
	return highest >= threshold && highest > 0
}

func shouldDisplaySeverity(severity, minDisplay string) bool {
	if minDisplay == "" {
		minDisplay = defaultStandardsMinDisplay
	}
	return severityWeight(severity) >= severityWeight(minDisplay)
}

func formatSeverityCounts(counts map[string]int) string {
	if len(counts) == 0 {
		return ""
	}
	order := []string{"critical", "high", "medium", "low", "info"}
	var parts []string
	seen := map[string]struct{}{}
	for _, sev := range order {
		if n, ok := counts[sev]; ok && n > 0 {
			parts = append(parts, fmt.Sprintf("%s=%d", sev, n))
			seen[sev] = struct{}{}
		}
	}
	var extras []string
	for k, v := range counts {
		if _, ok := seen[k]; ok || v <= 0 {
			continue
		}
		extras = append(extras, fmt.Sprintf("%s=%d", k, v))
	}
	sort.Strings(extras)
	parts = append(parts, extras...)
	return strings.Join(parts, ", ")
}

func classifyAuditorError(err error) (classification string, remediation string) {
	if err == nil {
		return "", ""
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return FailureClassTimeout, "Increase the standards phase timeout via `.vrooli/testing.json` (phases.standards.timeout) or reduce the audit scope."
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		switch exitErr.ExitCode() {
		case 124:
			return FailureClassTimeout, "Increase the standards phase timeout via `.vrooli/testing.json` (phases.standards.timeout) or reduce the audit scope."
		case 127:
			return FailureClassMissingDependency, "Ensure `scenario-auditor` (and `vrooli`) are installed and accessible on PATH."
		default:
			return FailureClassSystem, "Re-run with `scenario-auditor audit <scenario> --standards-only --json` and inspect logs for errors."
		}
	}

	return FailureClassSystem, "Re-run the standards phase and inspect logs for errors."
}
