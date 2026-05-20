package phases

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"test-genie/internal/eligibility"
	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/shared"

	"github.com/vrooli/api-core/discovery"
)

const (
	defaultStandardsFailOn      = "high"
	defaultStandardsSummaryTopN = 20
	defaultStandardsMinDisplay  = "medium"
)

// Re-export eligibility types as the package-local types this file (and its
// tests) previously owned. This keeps the auditor wiring in a single place
// (the eligibility package) while preserving the existing test surface.
type (
	auditorSummaryResponse        = eligibility.SummaryResponse
	auditorScanArtifactRef        = eligibility.ScanArtifactRef
	auditorRuleCount              = eligibility.RuleCount
	auditorViolationExcerpt       = eligibility.ViolationExcerpt
	auditorViolationSummary       = eligibility.ViolationSummary
	auditorStandardsStartResponse = eligibility.StandardsStartResponse
	auditorStandardsStatus        = eligibility.StandardsStatus
)

// Re-bind the eligibility package's seam variables to package-level vars so
// tests can override them without importing eligibility.
var (
	resolveScenarioAuditorBaseURL = eligibility.ResolveBaseURL
)

// Test-friendly aliases for the moved auditor helpers.
var (
	parseAuditorStandardsSummary      = eligibility.ParseSummary
	startAuditorStandardsScan         = eligibility.StartScan
	fetchAuditorStandardsStatus       = eligibility.FetchStatus
	fetchAuditorStandardsSummaryByJob = eligibility.FetchSummaryByJob
)

func fetchAuditorStandardsSummary(ctx context.Context, logWriter io.Writer, baseURL, scenarioName string, mapping workspace.Mapping, summaryLimit int) (*auditorViolationSummary, error) {
	return eligibility.FetchSummary(ctx, logWriter, baseURL, scenarioName, mapping, summaryLimit)
}

func runStandardsPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if report := CheckContext(ctx); report != nil {
		return *report
	}

	cleanLog := wrapLogSansANSI(logWriter)

	if os.Getenv("TEST_GENIE_SKIP_STANDARDS") == "1" {
		shared.LogWarn(cleanLog, "standards phase disabled via TEST_GENIE_SKIP_STANDARDS")
		return RunReport{
			Observations: []Observation{
				NewSkipObservation("standards phase disabled via TEST_GENIE_SKIP_STANDARDS"),
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

	shared.LogStep(cleanLog, "running standards scan via scenario-auditor API (timeout=%ds, fail_on=%s)", timeoutSeconds, failOn)
	baseURL, err := resolveScenarioAuditorBaseURL(ctx)
	if err != nil {
		classification, remediation := classifyAuditorError(err)
		return RunReport{
			Err:                   err,
			FailureClassification: classification,
			Remediation:           remediation,
			Observations: []Observation{
				NewSectionObservation("📏", "Standards"),
				NewErrorObservation("scenario-auditor API unavailable"),
			},
		}
	}

	mapping := env.Mapping
	if strings.TrimSpace(mapping.PhysicalScenarioDir) == "" {
		mapping = workspace.Mapping{
			PhysicalScenarioDir: strings.TrimSpace(env.ScenarioDir),
			PhysicalAppRoot:     strings.TrimSpace(env.PhysicalAppRoot),
		}
	}
	summary, err := fetchAuditorStandardsSummary(ctx, cleanLog, baseURL, env.ScenarioName, mapping, summaryLimit)
	observations := buildStandardsObservations(summary, failOn, minDisplay)

	if err != nil {
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
			remediation = fmt.Sprintf("Run `scenario-auditor standards scan %s --wait --timeout %ds` and address %s+ findings.", env.ScenarioName, timeoutSeconds, strings.ToUpper(failOn))
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
	return eligibility.NormalizeSeverity(raw)
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

	var discoveryErr *discovery.Error
	if errors.As(err, &discoveryErr) {
		switch discoveryErr.Kind {
		case discovery.ErrTimeout:
			return FailureClassTimeout, "Increase the standards phase timeout via `.vrooli/testing.json` (phases.standards.timeout) or reduce the audit scope."
		case discovery.ErrVrooliNotFound:
			return FailureClassMissingDependency, "Ensure `vrooli` is installed and accessible so Test Genie can discover the scenario-auditor API."
		case discovery.ErrScenarioNotRunning:
			return FailureClassMissingDependency, "Start `scenario-auditor` so Test Genie can reach its API for standards checks."
		default:
			return FailureClassSystem, "Resolve scenario-auditor API discovery failures, then rerun the standards phase."
		}
	}

	return FailureClassSystem, "Re-run the standards phase after verifying the scenario-auditor API is healthy and reachable."
}
