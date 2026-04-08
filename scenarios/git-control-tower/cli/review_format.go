package main

import (
	"fmt"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Shared output formatting -- Operational contract: Status -> Triage -> Next Steps
// ---------------------------------------------------------------------------

// reviewTriage collects needs-attention and passing dimension summaries.
type reviewTriage struct {
	needsAttention []string
	passing        []string
}

func printReviewSummary(resp *reviewSummaryResponse) {
	label := readinessLabel(resp.Readiness)
	fmt.Printf("=== Readiness Review: %s ===\n", resp.ScenarioName)
	fmt.Printf("Status: %s\n", label)
	fmt.Println()

	triage := triageDimensions(&resp.Dimensions)
	printTriageSections(&triage)
	printUntracedFiles(resp.Dimensions.Provenance)
	printNextSteps(resp.Readiness, resp.ScenarioName)
}

func triageDimensions(d *reviewDimensions) reviewTriage {
	var t reviewTriage
	triageStandards(d.Standards, &t)
	triageTests(d.Tests, &t)
	triageCodeQuality(d.CodeQuality, &t)
	triageVisual(d.Visual, &t)
	triageProvenance(d.Provenance, &t)
	return t
}

func triageStandards(s *standardsDimension, t *reviewTriage) {
	if s == nil || !s.Available {
		return
	}
	if s.BlockingViolations > 0 || s.Warnings > 0 {
		line := fmt.Sprintf("  Standards -- %d blocking, %d warnings (%d total)",
			s.BlockingViolations, s.Warnings, s.TotalViolations)
		for _, v := range s.TopViolations {
			line += formatViolationDetail(&v)
		}
		t.needsAttention = append(t.needsAttention, line)
	} else {
		t.passing = append(t.passing, "  Standards -- 0 violations")
	}
}

func formatViolationDetail(v *standardsViolationDetail) string {
	line := fmt.Sprintf("\n    %s:%d  %s (%s)", v.FilePath, v.LineNumber, v.Title, v.Severity)
	if v.Recommendation != "" {
		line += fmt.Sprintf("\n      -> %s", v.Recommendation)
	}
	return line
}

func triageTests(ts *testsDimension, t *reviewTriage) {
	if ts == nil || !ts.Available {
		return
	}
	if !ts.Passed && ts.Total > 0 {
		line := fmt.Sprintf("  Tests -- %d of %d failed", ts.FailedCount, ts.Total)
		for _, f := range ts.Failures {
			line += formatTestFailure(&f)
		}
		t.needsAttention = append(t.needsAttention, line)
	} else if ts.Total > 0 {
		line := fmt.Sprintf("  Tests -- %d/%d passed", ts.PassedCount, ts.Total)
		if ts.LastRun != "" {
			line += fmt.Sprintf(" (last run: %s)", formatTimestamp(ts.LastRun))
		}
		t.passing = append(t.passing, line)
	} else {
		t.needsAttention = append(t.needsAttention, "  Tests -- no test runs found")
	}
}

func formatTestFailure(f *testFailure) string {
	line := fmt.Sprintf("\n    %s: %s", f.Phase, f.Error)
	if f.Classification != "" {
		line += fmt.Sprintf(" (classification: %s)", f.Classification)
	}
	if f.Remediation != "" {
		line += fmt.Sprintf("\n      -> %s", f.Remediation)
	}
	return line
}

func triageCodeQuality(cq *codeQualityDimension, t *reviewTriage) {
	if cq == nil || !cq.Available {
		return
	}
	if cq.Score < 60 || cq.Stale {
		issue := fmt.Sprintf("  Code quality -- %.0f/100", cq.Score)
		if cq.Stale {
			issue += " (stale)"
		}
		for _, i := range cq.TopIssues {
			issue += fmt.Sprintf("\n    %s: %d", i.Category, i.Count)
		}
		t.needsAttention = append(t.needsAttention, issue)
	} else {
		line := fmt.Sprintf("  Code quality -- %.0f/100", cq.Score)
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

func triageVisual(v *visualDimension, t *reviewTriage) {
	if v == nil || !v.Available {
		return
	}
	line := fmt.Sprintf("  Visual -- %d screenshots", v.ScreenshotCount)
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

func triageProvenance(p *provenanceDimension, t *reviewTriage) {
	if p == nil || !p.Available {
		return
	}
	t.passing = append(t.passing, fmt.Sprintf("  Provenance -- %d traced files", p.TracedFiles))
	if len(p.UntracedFiles) > 0 {
		t.needsAttention = append(t.needsAttention,
			fmt.Sprintf("  Provenance -- %d untraced files", len(p.UntracedFiles)))
	}
}

func printTriageSections(t *reviewTriage) {
	if len(t.needsAttention) > 0 {
		fmt.Println("Needs attention:")
		for _, line := range t.needsAttention {
			fmt.Println(line)
		}
		fmt.Println()
	}

	if len(t.passing) > 0 {
		fmt.Println("Passing:")
		for _, line := range t.passing {
			fmt.Println(line)
		}
		fmt.Println()
	}
}

func printUntracedFiles(p *provenanceDimension) {
	if p == nil || len(p.UntracedFiles) == 0 {
		return
	}
	fmt.Println("Untraced files:")
	for _, f := range p.UntracedFiles {
		fmt.Printf("  %s\n", f)
	}
	fmt.Println()
}

func printNextSteps(readiness, scenarioName string) {
	fmt.Println("Next steps:")
	switch readiness {
	case "green":
		fmt.Println("  All checks pass -- scenario appears ready for commit review")
	case "yellow":
		fmt.Printf("  git-control-tower review-run %s    # re-check after fixes\n", scenarioName)
	case "red":
		fmt.Printf("  git-control-tower review-run %s    # re-check after fixes\n", scenarioName)
		fmt.Println("  Address the issues above before proceeding")
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
