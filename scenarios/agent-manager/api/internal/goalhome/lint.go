package goalhome

import (
	"fmt"
	"strings"
)

// ResumeBudgetBytes caps the files an orchestrator reads on every resume:
// GOAL.md, QUEUE.md, open FEEDBACK and open WORKAROUNDS (decision P-06).
const ResumeBudgetBytes = 30000

// ResumeSet reports the resume-set sizes in bytes.
type ResumeSet struct {
	Goal            int `json:"goal"`
	Queue           int `json:"queue"`
	OpenFeedback    int `json:"open_feedback"`
	OpenWorkarounds int `json:"open_workarounds"`
	Total           int `json:"total"`
	Budget          int `json:"budget"`
}

// LintReport is the goal-home lint result. Blocking is true when any finding
// blocks.
type LintReport struct {
	ResumeSet ResumeSet `json:"resume_set"`
	Findings  []Finding `json:"findings"`
	Forecast  *Forecast `json:"forecast"`
	Blocking  bool      `json:"blocking"`
}

// Lint checks the goal-home rules the orchestrator owns. It is separate from
// epoch-check so goal-home state never fails a worker.
func Lint(home *Home) *LintReport {
	report := &LintReport{Findings: []Finding{}}
	add := func(finding Finding) {
		report.Findings = append(report.Findings, finding)
		report.Blocking = report.Blocking || finding.Blocking
	}
	for _, missing := range home.Missing {
		add(Finding{Code: "missing-file", Blocking: missing == GoalFile || missing == QueueFile, File: missing, Detail: missing + " is missing from the goal home"})
	}

	resume := ResumeSet{Goal: home.Goal.Bytes, Queue: home.Queue.Bytes, OpenFeedback: home.Feedback.OpenBytes, OpenWorkarounds: home.Workarounds.OpenBytes, Budget: ResumeBudgetBytes}
	resume.Total = resume.Goal + resume.Queue + resume.OpenFeedback + resume.OpenWorkarounds
	report.ResumeSet = resume
	if resume.Total > ResumeBudgetBytes {
		add(Finding{Code: "resume-budget", Blocking: true, Detail: fmt.Sprintf("the resume set is %d bytes against %d (GOAL %d, QUEUE %d, open FEEDBACK %d, open WORKAROUNDS %d); move done slices, resolved entries and census evidence to archive/", resume.Total, ResumeBudgetBytes, resume.Goal, resume.Queue, resume.OpenFeedback, resume.OpenWorkarounds)})
	}

	handoffs := home.Queue.Handoffs
	var handoff *HandoffSection
	switch {
	case len(handoffs) == 0:
		add(Finding{Code: "handoff-count", Blocking: true, File: QueueFile, Detail: "QUEUE.md has no ## Handoff section; write one with `agent-manager effort handoff set`"})
	case len(handoffs) > 1:
		add(Finding{Code: "handoff-count", Blocking: true, File: QueueFile, Line: handoffs[1].Line, Detail: fmt.Sprintf("QUEUE.md has %d handoff sections, want 1: %s. Move the old ones to archive/ and replace the one with `effort handoff set`", len(handoffs), describeHandoffs(handoffs))})
	default:
		handoff = &handoffs[0]
	}
	if handoff != nil && len(handoff.Body) > HandoffMaxBytes {
		add(Finding{Code: "handoff-size", Blocking: true, File: QueueFile, Line: handoff.Line, Detail: fmt.Sprintf("the handoff is %d bytes against %d", len(handoff.Body), HandoffMaxBytes)})
	}

	if len(home.Queue.ReadyNext()) == 0 && len(home.Queue.NeedsOperator) == 0 {
		add(Finding{Code: "idle-queue", Blocking: true, File: QueueFile, Detail: "no [ready] item under ## Next and no item under ## Needs operator: admit or plan a slice and mark it [ready], or record a decision under ## Needs operator"})
	}

	handoffText := ""
	for _, h := range handoffs {
		handoffText += h.Body + "\n"
	}
	for _, entry := range home.Feedback.Entries {
		if entry.Open() && entry.Steering() && !namedIn(handoffText, entry.ID) {
			add(Finding{Code: "feedback-unacknowledged", Blocking: true, File: FeedbackFile, Line: entry.Line, Detail: entry.ID + " is open " + feedbackSourceLabel(entry) + " feedback that the handoff does not name; act on it and name it in the handoff, or resolve it"})
		}
	}

	report.Forecast = ForecastOf(home)
	if finding := report.Forecast.Finding(); finding != nil {
		add(*finding)
	}
	for _, warning := range home.Warnings {
		add(warning)
	}
	for _, census := range home.Queue.Censuses {
		if census.At.IsZero() {
			add(Finding{Code: "census-format", File: QueueFile, Line: census.Line, Detail: "a census line is \"- <ISO time> | <scope> | admissible=<slice>|none\""})
		}
	}
	return report
}

func feedbackSourceLabel(entry FeedbackEntry) string {
	if entry.Source == "" {
		return "operator"
	}
	return entry.Source
}

// Summary is a one-line verdict for text output.
func (r *LintReport) Summary() string {
	blocking := 0
	for _, finding := range r.Findings {
		if finding.Blocking {
			blocking++
		}
	}
	if blocking == 0 {
		return fmt.Sprintf("goal home ok: resume set %d/%d bytes, %d warnings", r.ResumeSet.Total, r.ResumeSet.Budget, len(r.Findings))
	}
	codes := make([]string, 0, blocking)
	for _, finding := range r.Findings {
		if finding.Blocking {
			codes = append(codes, finding.Code)
		}
	}
	return fmt.Sprintf("goal home fails %d rules: %s", blocking, strings.Join(codes, ", "))
}
