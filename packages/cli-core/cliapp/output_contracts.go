package cliapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// TriageGroup represents one remediation-oriented grouping in the operational
// output contract.
type TriageGroup struct {
	Heading string   `json:"heading"`
	Items   []string `json:"items,omitempty"`
}

// OperationalReport renders the canonical human-first contract for diagnostic
// commands: Status -> Triage -> Next Steps.
type OperationalReport struct {
	Status    []string      `json:"status,omitempty"`
	Triage    []TriageGroup `json:"triage,omitempty"`
	NextSteps []string      `json:"next_steps,omitempty"`
}

// ListReport renders the canonical human-first contract for data retrieval:
// Summary -> Results -> Retrieval Hints.
type ListReport struct {
	Summary        []string `json:"summary,omitempty"`
	ResultsHeading string   `json:"results_heading,omitempty"`
	Results        []string `json:"results,omitempty"`
	RetrievalHints []string `json:"retrieval_hints,omitempty"`
}

// MutationReport renders the canonical human-first contract for mutations:
// Result -> What Changed -> Next Command.
type MutationReport struct {
	Result      []string `json:"result,omitempty"`
	Changes     []string `json:"changes,omitempty"`
	NextCommand []string `json:"next_command,omitempty"`
}

// APIRecoveryReportOptions captures the common recovery context for commands
// that could not reach their scenario API.
type APIRecoveryReportOptions struct {
	AppName           string
	CommandName       string
	ResolvedAPIBase   string
	ConfiguredAPIBase string
	DetectedAPIBase   string
	Cause             string
	MissingAPIBase    bool
	RuntimeStatus     string
}

func RenderOperationalReport(w io.Writer, report OperationalReport) error {
	if err := printSectionLines(w, "Status", report.Status, "(no status reported)"); err != nil {
		return err
	}
	if len(report.Triage) > 0 {
		if _, err := fmt.Fprintln(w, "\nTriage:"); err != nil {
			return err
		}
		for _, group := range report.Triage {
			heading := strings.TrimSpace(group.Heading)
			if heading == "" {
				heading = "General"
			}
			if _, err := fmt.Fprintf(w, "  %s:\n", heading); err != nil {
				return err
			}
			items := sanitizedLines(group.Items)
			if len(items) == 0 {
				if _, err := fmt.Fprintln(w, "    (none)"); err != nil {
					return err
				}
				continue
			}
			for _, item := range items {
				if err := writeIndentedLines(w, item, "    "); err != nil {
					return err
				}
			}
		}
	}
	return printSectionLines(w, "Next Steps", report.NextSteps, "(none)")
}

func RenderListReport(w io.Writer, report ListReport) error {
	if err := printSectionLines(w, "Summary", report.Summary, "(no summary available)"); err != nil {
		return err
	}
	heading := strings.TrimSpace(report.ResultsHeading)
	if heading == "" {
		heading = "Results"
	}
	if err := printSectionLines(w, heading, report.Results, "(none)"); err != nil {
		return err
	}
	return printSectionLines(w, "Retrieval Hints", report.RetrievalHints, "(none)")
}

func RenderMutationReport(w io.Writer, report MutationReport) error {
	if err := printSectionLines(w, "Result", report.Result, "(no result reported)"); err != nil {
		return err
	}
	if err := printSectionLines(w, "What Changed", report.Changes, "(none)"); err != nil {
		return err
	}
	return printSectionLines(w, "Next Command", report.NextCommand, "(none)")
}

func PrintReportJSON(w io.Writer, report interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func RenderOperationalReportString(report OperationalReport) (string, error) {
	var out bytes.Buffer
	if err := RenderOperationalReport(&out, report); err != nil {
		return "", err
	}
	return out.String(), nil
}

func NewAPIRecoveryReport(opts APIRecoveryReportOptions) OperationalReport {
	appName := strings.TrimSpace(opts.AppName)
	commandName := strings.TrimSpace(opts.CommandName)
	noRuntimeDetected := strings.TrimSpace(opts.DetectedAPIBase) == ""
	stoppedBeforeBaseResolution := opts.MissingAPIBase && noRuntimeDetected
	confirmedStopped := strings.EqualFold(strings.TrimSpace(opts.RuntimeStatus), "stopped")

	report := OperationalReport{}
	switch {
	case confirmedStopped:
		// When no API port exists, "API base is empty" is merely an internal
		// consequence of the scenario being stopped. Lead with the operator's
		// actionable condition instead of sending them toward configuration.
		report.Status = append(report.Status, fmt.Sprintf("%s is stopped, so its local API is unavailable.", appName))
	case stoppedBeforeBaseResolution:
		report.Status = append(report.Status, fmt.Sprintf("No local %s API was detected.", appName))
	case opts.MissingAPIBase:
		report.Status = append(report.Status, fmt.Sprintf("Unable to resolve the %s API base.", appName))
	default:
		report.Status = append(report.Status, fmt.Sprintf("Unable to reach the %s API.", appName))
	}
	if resolved := strings.TrimSpace(opts.ResolvedAPIBase); resolved != "" {
		report.Status = append(report.Status, fmt.Sprintf("Resolved API base: %s", resolved))
	}
	if cause := strings.TrimSpace(opts.Cause); cause != "" && !stoppedBeforeBaseResolution {
		report.Status = append(report.Status, fmt.Sprintf("Last error: %s", cause))
	}

	runtimeItems := make([]string, 0, 2)
	if detected := strings.TrimSpace(opts.DetectedAPIBase); detected != "" {
		runtimeItems = append(runtimeItems, fmt.Sprintf("Detected running API base: %s", detected))
	} else {
		if confirmedStopped {
			runtimeItems = append(runtimeItems, "Scenario lifecycle status: stopped.")
		} else {
			runtimeItems = append(runtimeItems, fmt.Sprintf("No running API port was detected for %s. Start the scenario, then retry the command.", appName))
		}
	}
	report.Triage = append(report.Triage, TriageGroup{
		Heading: "Runtime",
		Items:   runtimeItems,
	})

	configItems := make([]string, 0, 2)
	if configured := strings.TrimSpace(opts.ConfiguredAPIBase); configured != "" {
		configItems = append(configItems, fmt.Sprintf("Saved config api_base: %s", configured))
	}
	if configured := strings.TrimSpace(opts.ConfiguredAPIBase); configured != "" && strings.TrimSpace(opts.DetectedAPIBase) != "" && !sameAPIBase(configured, opts.DetectedAPIBase) {
		configItems = append(configItems, "Saved api_base does not match the currently detected running API and may be stale.")
	}
	if len(configItems) > 0 {
		report.Triage = append(report.Triage, TriageGroup{
			Heading: "Configuration",
			Items:   configItems,
		})
	}

	if stoppedBeforeBaseResolution || confirmedStopped {
		report.NextSteps = append(report.NextSteps, fmt.Sprintf("vrooli scenario start %s", appName))
		if commandName != "" {
			report.NextSteps = append(report.NextSteps, fmt.Sprintf("%s --auto-start %s", appName, commandName))
		}
		report.NextSteps = append(report.NextSteps, fmt.Sprintf("vrooli scenario status %s", appName))
		return report
	}

	if commandName != "" {
		report.NextSteps = append(report.NextSteps, fmt.Sprintf("%s --auto-start %s", appName, commandName))
	}
	report.NextSteps = append(report.NextSteps, fmt.Sprintf("vrooli scenario status %s", appName))
	report.NextSteps = append(report.NextSteps, fmt.Sprintf("vrooli scenario start %s", appName))
	if detected := strings.TrimSpace(opts.DetectedAPIBase); detected != "" {
		report.NextSteps = append(report.NextSteps, fmt.Sprintf("%s configure api_base %s", appName, detected))
	}

	return report
}

func printSectionLines(w io.Writer, heading string, lines []string, empty string) error {
	heading = strings.TrimSpace(heading)
	if heading == "" {
		return nil
	}
	if _, err := fmt.Fprintf(w, "%s:\n", heading); err != nil {
		return err
	}
	lines = sanitizedLines(lines)
	if len(lines) == 0 {
		if _, err := fmt.Fprintf(w, "  %s\n", strings.TrimSpace(empty)); err != nil {
			return err
		}
		return nil
	}
	for _, line := range lines {
		if err := writeIndentedLines(w, line, "  "); err != nil {
			return err
		}
	}
	return nil
}

func writeIndentedLines(w io.Writer, value string, indent string) error {
	lines := strings.Split(strings.TrimRight(value, "\n"), "\n")
	for _, line := range lines {
		if _, err := fmt.Fprintf(w, "%s%s\n", indent, strings.TrimRight(line, "\r")); err != nil {
			return err
		}
	}
	return nil
}

func sanitizedLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func sameAPIBase(a, b string) bool {
	return normalizeAPIBaseComparison(a) == normalizeAPIBaseComparison(b)
}

func normalizeAPIBaseComparison(value string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(value), "/")
	trimmed = strings.TrimSuffix(trimmed, "/api/v1")
	return strings.TrimRight(trimmed, "/")
}
