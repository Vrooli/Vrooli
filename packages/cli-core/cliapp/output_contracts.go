package cliapp

import (
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
