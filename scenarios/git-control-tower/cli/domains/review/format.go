package review

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

type triage struct {
	needsAttention []string
	passing        []string
}

func renderSummary(w io.Writer, resp *summaryResponse) error {
	if resp == nil {
		return nil
	}

	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Scenario: %s", resp.ScenarioName),
			fmt.Sprintf("Readiness: %s", readinessLabel(resp.Readiness)),
		},
		NextSteps: nextStepCommands(resp.Readiness, resp.ScenarioName),
	}

	triage := triageDimensions(&resp.Dimensions)
	if len(triage.needsAttention) > 0 {
		report.Triage = append(report.Triage, cliapp.TriageGroup{
			Heading: "Needs attention",
			Items:   triage.needsAttention,
		})
	}
	if len(triage.passing) > 0 {
		report.Triage = append(report.Triage, cliapp.TriageGroup{
			Heading: "Passing",
			Items:   triage.passing,
		})
	}

	if items := untracedFileItems(resp.Dimensions.Provenance); len(items) > 0 {
		report.Triage = append(report.Triage, cliapp.TriageGroup{
			Heading: "Untraced files",
			Items:   items,
		})
	}

	return cliapp.RenderOperationalReport(w, report)
}

func printSummary(resp *summaryResponse) {
	var out bytes.Buffer
	if err := renderSummary(&out, resp); err != nil {
		fmt.Print(out.String())
		return
	}
	fmt.Print(out.String())
}

func triageDimensions(d *dimensions) triage {
	var t triage
	triageStandards(d.Standards, &t)
	triageTests(d.Tests, &t)
	triageCodeQuality(d.CodeQuality, &t)
	triageVisual(d.Visual, &t)
	triageProvenance(d.Provenance, &t)
	return t
}

func triageStandards(s *standardsDimension, t *triage) {
	if s == nil || !s.Available {
		return
	}
	if s.BlockingViolations > 0 || s.Warnings > 0 {
		line := fmt.Sprintf("Standards: %d blocking, %d warnings (%d total)", s.BlockingViolations, s.Warnings, s.TotalViolations)
		for _, v := range s.TopViolations {
			line += formatViolationDetail(&v)
		}
		t.needsAttention = append(t.needsAttention, line)
	} else {
		t.passing = append(t.passing, "Standards: 0 violations")
	}
}

func formatViolationDetail(v *standardsViolationDetail) string {
	line := fmt.Sprintf("\n%s:%d  %s (%s)", v.FilePath, v.LineNumber, v.Title, v.Severity)
	if v.Recommendation != "" {
		line += fmt.Sprintf("\n-> %s", v.Recommendation)
	}
	return line
}

func triageTests(ts *testsDimension, t *triage) {
	if ts == nil || !ts.Available {
		return
	}
	if !ts.Passed && ts.Total > 0 {
		line := fmt.Sprintf("Tests: %d of %d failed", ts.FailedCount, ts.Total)
		for _, f := range ts.Failures {
			line += formatTestFailure(&f)
		}
		t.needsAttention = append(t.needsAttention, line)
	} else if ts.Total > 0 {
		line := fmt.Sprintf("Tests: %d/%d passed", ts.PassedCount, ts.Total)
		if ts.LastRun != "" {
			line += fmt.Sprintf(" (last run: %s)", formatTimestamp(ts.LastRun))
		}
		t.passing = append(t.passing, line)
	} else {
		t.needsAttention = append(t.needsAttention, "Tests: no test runs found")
	}
}

func formatTestFailure(f *testFailure) string {
	line := fmt.Sprintf("\n%s: %s", f.Phase, f.Error)
	if f.Classification != "" {
		line += fmt.Sprintf(" (classification: %s)", f.Classification)
	}
	if f.Remediation != "" {
		line += fmt.Sprintf("\n-> %s", f.Remediation)
	}
	return line
}

func triageCodeQuality(cq *codeQualityDimension, t *triage) {
	if cq == nil || !cq.Available {
		return
	}
	if cq.Score < 60 || cq.Stale {
		issue := fmt.Sprintf("Code quality: %.0f/100", cq.Score)
		if cq.Stale {
			issue += " (stale)"
		}
		for _, i := range cq.TopIssues {
			issue += fmt.Sprintf("\n%s: %d", i.Category, i.Count)
		}
		t.needsAttention = append(t.needsAttention, issue)
	} else {
		line := fmt.Sprintf("Code quality: %.0f/100", cq.Score)
		if len(cq.TopIssues) > 0 {
			line += " (" + formatCodeQualityIssues(cq.TopIssues) + ")"
		}
		t.passing = append(t.passing, line)
	}
}

func formatCodeQualityIssues(issues []codeQualityIssue) string {
	var parts []string
	for _, i := range issues {
		parts = append(parts, fmt.Sprintf("%s: %d", i.Category, i.Count))
	}
	return strings.Join(parts, ", ")
}

func triageVisual(v *visualDimension, t *triage) {
	if v == nil || !v.Available {
		return
	}
	line := fmt.Sprintf("Visual: %d screenshots", v.ScreenshotCount)
	line += formatVisualCapture(v.LatestCapture)
	if v.ScreenshotCount == 0 {
		t.needsAttention = append(t.needsAttention, line)
	} else {
		t.passing = append(t.passing, line)
	}
}

func formatVisualCapture(cap *visualCaptureMeta) string {
	if cap == nil {
		return ""
	}
	line := fmt.Sprintf(" (latest: %s", formatTimestamp(cap.CapturedAt))
	if cap.CommitHash != "" {
		hash := cap.CommitHash
		if len(hash) > 7 {
			hash = hash[:7]
		}
		line += fmt.Sprintf(", commit %s", hash)
	}
	line += ")"
	return line
}

func triageProvenance(p *provenanceDimension, t *triage) {
	if p == nil || !p.Available {
		return
	}
	t.passing = append(t.passing, fmt.Sprintf("Provenance: %d traced files", p.TracedFiles))
	if len(p.UntracedFiles) > 0 {
		t.needsAttention = append(t.needsAttention, fmt.Sprintf("Provenance: %d untraced files", len(p.UntracedFiles)))
	}
}

func untracedFileItems(p *provenanceDimension) []string {
	if p == nil || len(p.UntracedFiles) == 0 {
		return nil
	}
	items := make([]string, 0, len(p.UntracedFiles))
	for _, f := range p.UntracedFiles {
		items = append(items, f)
	}
	return items
}

func nextStepCommands(readiness, scenarioName string) []string {
	switch readiness {
	case "green":
		return []string{"All checks pass; scenario appears ready for commit review."}
	case "yellow", "red":
		steps := []string{fmt.Sprintf("git-control-tower review run %s", scenarioName)}
		if readiness == "red" {
			steps = append(steps, "Address the issues above before proceeding.")
		}
		return steps
	default:
		return nil
	}
}

func readinessLabel(r string) string {
	switch r {
	case "green":
		return "GREEN (ready)"
	case "yellow":
		return "YELLOW (needs attention)"
	case "red":
		return "RED (not ready)"
	default:
		return strings.ToUpper(r)
	}
}

func formatTimestamp(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return t.Format("2006-01-02 15:04")
}
