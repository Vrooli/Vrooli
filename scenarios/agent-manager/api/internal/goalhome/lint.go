package goalhome

import (
	"fmt"
	"sort"
	"strings"
)

// ResumeBudgetBytes caps the files an orchestrator reads on every resume:
// CHARTER.md, STATUS.md and open FEEDBACK (legacy import counts WORKAROUNDS separately).
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
	// V2 is true for a Phase-2 home, so text output names CHARTER/STATUS.
	V2 bool `json:"v2,omitempty"`
}

// Lint checks the goal-home rules the orchestrator owns. It is separate from
// epoch-check so goal-home state never fails a worker.
func Lint(home *Home) *LintReport {
	report := &LintReport{Findings: []Finding{}}
	add := func(finding Finding) {
		report.Findings = append(report.Findings, finding)
		report.Blocking = report.Blocking || finding.Blocking
	}
	// The required-file and queue-section anchors name CHARTER/STATUS for a
	// Phase-2 v2 home, GOAL/QUEUE for a v1 home. Same thresholds, new anchors.
	goalFile, queueFile := GoalFile, QueueFile
	if home.V2 {
		goalFile, queueFile = CharterFile, StatusFile
	}
	report.V2 = home.V2
	for _, missing := range home.Missing {
		add(Finding{Code: "missing-file", Blocking: missing == goalFile || missing == queueFile, File: missing, Detail: missing + " is missing from the goal home"})
	}
	files := make([]string, 0, len(home.FileBytes))
	for file := range home.FileBytes {
		files = append(files, file)
	}
	sort.Strings(files)
	for _, file := range files {
		size := home.FileBytes[file]
		if size > DefaultBoardFileBytes {
			add(Finding{
				Code:   "board-file-size-advisory",
				File:   file,
				Detail: fmt.Sprintf("%s is %d bytes, above the supervisor board default per-file read cap of %d bytes; configured board limits are authoritative", file, size, DefaultBoardFileBytes),
			})
		}
	}

	resume := ResumeSet{Goal: home.Goal.Bytes, Queue: home.Queue.Bytes, OpenFeedback: home.Feedback.OpenBytes, OpenWorkarounds: home.Workarounds.OpenBytes, Budget: ResumeBudgetBytes}
	if home.V2 {
		// STATUS already includes its workaround section; count each byte once.
		resume.OpenWorkarounds = 0
	}
	resume.Total = resume.Goal + resume.Queue + resume.OpenFeedback + resume.OpenWorkarounds
	report.ResumeSet = resume
	if resume.Total > ResumeBudgetBytes {
		add(Finding{Code: "resume-budget", Blocking: true, Detail: fmt.Sprintf("the resume set is %d bytes against %d (%s %d, %s %d, open FEEDBACK %d, open WORKAROUNDS %d); move done slices, resolved entries and census evidence to archive/", resume.Total, ResumeBudgetBytes, goalFile, resume.Goal, queueFile, resume.Queue, resume.OpenFeedback, resume.OpenWorkarounds)})
	}

	handoffs := home.Queue.Handoffs
	var handoff *HandoffSection
	switch {
	case len(handoffs) == 0:
		add(Finding{Code: "handoff-count", Blocking: true, File: queueFile, Detail: queueFile + " has no ## Handoff section; write one with `agent-manager effort handoff set`"})
	case len(handoffs) > 1:
		add(Finding{Code: "handoff-count", Blocking: true, File: queueFile, Line: handoffs[1].Line, Detail: fmt.Sprintf("%s has %d handoff sections, want 1: %s. Move the old ones to archive/ and replace the one with `effort handoff set`", queueFile, len(handoffs), describeHandoffs(handoffs))})
	default:
		handoff = &handoffs[0]
	}
	if handoff != nil && len(handoff.Body) > HandoffMaxBytes {
		add(Finding{Code: "handoff-size", Blocking: true, File: queueFile, Line: handoff.Line, Detail: fmt.Sprintf("the handoff is %d bytes against %d", len(handoff.Body), HandoffMaxBytes)})
	}

	if len(home.Queue.ReadyNext()) == 0 && len(home.Queue.NeedsOperator) == 0 {
		add(Finding{Code: "idle-queue", Blocking: true, File: queueFile, Detail: "no [ready] item under ## Next and no item under ## Needs operator: admit or plan a slice and mark it [ready], or record a decision under ## Needs operator"})
	}

	for _, entry := range unacknowledgedFeedback(home) {
		add(Finding{Code: "feedback-unacknowledged", Blocking: true, File: FeedbackFile, Line: entry.Line, Detail: entry.ID + " is open " + feedbackSourceLabel(entry) + " feedback that the handoff does not name; act on it and name it in the handoff, or resolve it"})
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
			add(Finding{Code: "census-format", File: queueFile, Line: census.Line, Detail: "a census line is \"- <ISO time> | <scope> | admissible=<slice>|none\""})
		}
	}
	return report
}

func unacknowledgedFeedback(home *Home) []FeedbackEntry {
	var handoff strings.Builder
	for _, h := range home.Queue.Handoffs {
		handoff.WriteString(h.Body)
		handoff.WriteByte('\n')
	}
	var pending []FeedbackEntry
	for _, entry := range home.Feedback.Entries {
		if entry.Open() && entry.Steering() && !namedIn(handoff.String(), entry.ID) {
			pending = append(pending, entry)
		}
	}
	return pending
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

// projectionFindings detects owner-authored shapes that would otherwise look
// healthy while silently dropping progress. Unknown is an explicit valid reading;
// prose and malformed keys are not measurements or ownership identities.
func projectionFindings(goal Goal, status []byte) []Finding {
	doc := parseDocument(status)
	var findings []Finding
	add := func(code, detail string, line int, blocking bool) {
		findings = append(findings, Finding{Code: code, Detail: detail, File: StatusFile, Line: line, Blocking: blocking})
	}
	currentValue := parseStatusCurrent(doc)
	if goal.Destination != nil && currentValue != nil && currentValue.At.IsZero() {
		add("projection-current", "numeric Current needs an ISO measurement date", 0, true)
	}
	if goal.Destination != nil && currentValue == nil {
		current, ok := doc.section("current")
		raw := strings.ToLower(strings.TrimSpace(current.Fields()["current"]))
		if ok && raw == "unknown" {
			add("current-unknown", "current measurement is explicitly unknown; no completion forecast is available", current.Line, false)
		} else {
			add("projection-current", "use ## Current: <number> @ <ISO date>, or ## Current with '- Current: unknown'; prose is not a measurement", current.Line, true)
		}
	}
	if s, ok := doc.section("scoreboard"); ok {
		ownerColumn := -1
		for _, l := range s.Body {
			cells, ok := tableRow(l.Text)
			if !ok || isSeparatorRow(cells) {
				continue
			}
			if len(cells) > 0 && strings.EqualFold(cells[0], "surface") {
				for i, c := range cells {
					if strings.EqualFold(c, "owner") {
						ownerColumn = i
					}
				}
				if ownerColumn < 0 {
					add("projection-scoreboard", "scoreboard needs an owner column; evidence belongs in its own column", l.Num, true)
				}
				continue
			}
			if len(cells) < 3 {
				add("projection-scoreboard", "scoreboard row needs surface, target and state", l.Num, true)
				continue
			}
			switch strings.ToLower(cells[2]) {
			case "unassessed", "queued", "rough", "close", "accepted":
			default:
				add("projection-scoreboard", "unknown surface state: "+cells[2], l.Num, true)
			}
			if ownerColumn >= 0 && len(cells) > ownerColumn {
				owner := cells[ownerColumn]
				ids := WorkerRunIDs(owner)
				if owner != "" && owner != "-" && owner != "—" && (len(ids) != 1 || ids[0] != owner) {
					add("projection-owner", "owner must be one run UUID or '-'; do not put evidence prose in owner", l.Num, true)
				}
			}
		}
	}
	if s, ok := doc.section("active"); ok {
		for _, item := range s.items() {
			fields := inlineFields(item.Text)
			unit := fields["unit"]
			if unit == "" || strings.ContainsAny(unit, " \t;`|") {
				add("projection-active", "active unit needs a stable unit ID, status and workers fields; put handoff prose outside the unit field", item.Line, true)
			}
		}
	}
	return findings
}
