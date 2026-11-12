package activity

import (
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliapp"

	"research-desk/cli/domains/render"

	activityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/activity"
)

var stateWords = []string{"live", "paused", "limited", "stopped", "prepared", "unknown", "ended"}

func stateWord(s activityv1.CommissionState) string {
	return strings.ToLower(strings.TrimPrefix(s.String(), "COMMISSION_STATE_"))
}

func stateFromWord(word string) (activityv1.CommissionState, bool) {
	value, ok := activityv1.CommissionState_value["COMMISSION_STATE_"+strings.ToUpper(strings.TrimSpace(word))]
	return activityv1.CommissionState(value), ok && value != 0
}

func topicList(c *activityv1.CommissionSummary) string {
	if len(c.Topics) == 0 {
		return render.Evidence(c.TopicsEvidence)
	}
	names := make([]string, 0, len(c.Topics))
	for _, ref := range c.Topics {
		names = append(names, ref.Topic)
	}
	return strings.Join(names, ", ")
}

func stateLabel(c *activityv1.CommissionSummary) string {
	label := stateWord(c.State)
	if c.Stalled {
		label += " (stalled)"
	}
	return label
}

// commissionLine is one commission on one line.
func commissionLine(c *activityv1.CommissionSummary) string {
	last := "-"
	if c.LastWrite.GetEvidence().GetKnown() {
		last = render.When(c.LastWrite.Value)
	}
	return fmt.Sprintf("%-40s %-18s last write %s · topics: %s", c.Id, stateLabel(c), last, topicList(c))
}

func (h *handlers) nowReport(_ cliapp.OperationContext, msg *activityv1.GetOverviewResponse) cliapp.ListReport {
	counts := make([]string, 0, len(msg.StateCounts))
	for _, sc := range msg.StateCounts {
		counts = append(counts, fmt.Sprintf("%d %s", sc.Count, stateWord(sc.State)))
	}
	report := cliapp.ListReport{
		Summary: []string{
			"Commissions: " + strings.Join(counts, ", ") + ".",
			"Research home: " + msg.ResearchHome + " (read " + render.When(msg.GeneratedAt) + ")",
		},
		ResultsHeading: "Supervisor windows open (live, paused or limited)",
		RetrievalHints: []string{"`commission <id>` — one commission with its evidence", "`commissions --state <state>` — every commission in a state"},
	}
	for _, c := range msg.Running {
		report.Results = append(report.Results, commissionLine(c), "    "+c.StateRule+" · claim: "+render.Text(c.Claim))
	}
	for _, o := range msg.Overlaps {
		report.Results = append(report.Results, fmt.Sprintf("overlap on %s: %s", o.Topic, strings.Join(o.CommissionIds, ", ")))
	}
	report.Results = append(report.Results, render.Unreadable(msg.Unreadable)...)
	for _, lens := range msg.Lenses {
		switch {
		case lens.Name == "web-console access":
			report.Summary = append(report.Summary, lens.Reason)
		case lens.Name == "courier":
			report.Summary = append(report.Summary, "Courier: "+lens.Reason)
		case !lens.Available:
			report.Results = append(report.Results, fmt.Sprintf("lens %s unavailable: %s", lens.Name, lens.Reason))
		}
	}
	return report
}

func (h *handlers) commissionsReport(_ cliapp.OperationContext, msg *activityv1.ListCommissionsResponse) cliapp.ListReport {
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d commission(s), newest last write first.", len(msg.Commissions))},
		ResultsHeading: "Commissions",
		ListShaped:     true,
		ResultCount:    len(msg.Commissions),
		RetrievalHints: []string{"`commission <id>` — sources, sessions and read sets"},
	}
	for _, c := range msg.Commissions {
		report.Results = append(report.Results, commissionLine(c))
	}
	return report
}

func (h *handlers) commissionReport(_ cliapp.OperationContext, msg *activityv1.GetCommissionResponse) cliapp.ListReport {
	c := msg.Commission
	lines := []string{
		"title:          " + render.Text(c.Title),
		"folder:         " + c.Folder,
		"state:          " + stateLabel(c) + " — " + c.StateRule,
	}
	for _, e := range c.StateEvidence {
		lines = append(lines, "  evidence:     "+render.Evidence(e))
	}
	lines = append(lines,
		"claim:          "+render.Text(c.Claim),
		"started:        "+render.Time(c.Started),
		"allowance end:  "+render.Time(c.AllowanceEnd),
		"last write:     "+render.Time(c.LastWrite),
	)
	if c.LaunchedBy != "" {
		lines = append(lines, "launched by:    "+c.LaunchedBy)
	}
	if len(c.Topics) == 0 {
		lines = append(lines, "topics:         "+render.Evidence(c.TopicsEvidence))
	}
	for _, ref := range c.Topics {
		lines = append(lines, fmt.Sprintf("topic:          %s  (%s: %s)", ref.Topic, ref.Basis, ref.Source))
	}
	for _, m := range msg.Matches {
		kind := strings.ToLower(strings.TrimPrefix(m.Kind.String(), "MATCH_KIND_"))
		window := "window closed"
		if m.WindowOpen {
			window = "window open"
		}
		goal := m.GoalStatus
		if goal == "" {
			goal = "no goal row"
		}
		lines = append(lines, fmt.Sprintf("session:        %s %s %s thread %s · goal %s · %s · %s  [%s]", kind, m.Basis, m.WebConsoleSessionId, m.ThreadId, goal, window, render.When(m.At), m.Source))
	}
	if len(msg.Matches) == 0 {
		lines = append(lines, "session:        none names this commission")
	}
	for _, run := range msg.Runs {
		checkpoint := "no CHECKPOINT.md"
		if run.Checkpoint != nil {
			checkpoint = fmt.Sprintf("CHECKPOINT.md %s", render.When(run.Checkpoint.Modified))
		}
		lines = append(lines, fmt.Sprintf("run folder:     %s · %s", run.Path, checkpoint))
	}
	lines = append(lines, "ceilings:       "+render.Evidence(msg.Ceilings))
	for _, set := range msg.ReadSets {
		lines = append(lines, fmt.Sprintf("read set:       %s %d bytes, about %d tokens, %d file(s)", set.Role, set.Bytes, set.EstimatedTokens, len(set.Files)))
		for _, flag := range set.Flags {
			lines = append(lines, "  flag:         "+flag)
		}
	}
	lines = append(lines, "steering:       "+render.Flag(msg.Steering, "the charter tells the scheduler to read OPERATOR.md", "not enabled; direction will not be seen until `enable-steering "+c.Id+"`"))
	lines = append(lines, "decisions:      "+render.Flag(msg.Decisions, "the charter leaves delivery to the desk", "its scheduler delivers needs-you items itself; the courier waits two hours for `Delivered:`"))
	for _, e := range msg.OperatorEntries {
		lines = append(lines, "operator:       "+render.Entry(e))
	}
	if msg.OperatorUnparsed > 0 {
		lines = append(lines, fmt.Sprintf("operator:       %d `### ` line(s) in OPERATOR.md do not match the entry grammar", msg.OperatorUnparsed))
	}
	for _, e := range msg.RecordErrors {
		lines = append(lines, "record error:   "+e)
	}
	lines = append(lines, render.Unreadable(msg.Unreadable)...)
	for _, f := range msg.Files {
		lines = append(lines, fmt.Sprintf("file:           %-24s %8d bytes  %s", f.Name, f.Bytes, render.When(f.Modified)))
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Commission %s is %s.", c.Id, stateLabel(c))},
		ResultsHeading: "Evidence",
		Results:        lines,
	}
}

func (h *handlers) topicsReport(_ cliapp.OperationContext, msg *activityv1.ListTopicsResponse) cliapp.ListReport {
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d topic(s), newest research write (run folders) first.", len(msg.Topics))},
		ResultsHeading: "Topics",
		ListShaped:     true,
		ResultCount:    len(msg.Topics),
	}
	for _, t := range msg.Topics {
		last := researchWrite(t.LastResearchWrite)
		check := render.Evidence(t.MissionCheck.GetEvidence())
		if mc := t.MissionCheck; mc.GetEvidence().GetKnown() {
			check = fmt.Sprintf("%s · %d open, %d active, %d waiting on operator, %d landed", mc.Mode, mc.Open, mc.Active, mc.WaitingOperator, mc.Landed)
			if len(mc.Errors) > 0 {
				check += fmt.Sprintf(" · %d mission error(s)", len(mc.Errors))
			}
		}
		running := ""
		for _, group := range []struct {
			word string
			ids  []string
		}{{"live", t.LiveCommissionIds}, {"paused", t.PausedCommissionIds}, {"limited", t.LimitedCommissionIds}} {
			if len(group.ids) > 0 {
				running += " · " + group.word + ": " + strings.Join(group.ids, ", ")
			}
		}
		report.Results = append(report.Results, fmt.Sprintf("%-40s research written %s · %d commission(s)%s", t.Topic, last, len(t.CommissionIds), running), "    mission: "+check)
		if m := t.Mission; m != nil {
			report.Results = append(report.Results, "    mission edited "+render.When(m.Modified)+" · "+m.Path)
		}
		if r := t.LastReview; r != nil {
			report.Results = append(report.Results, "    reviewed "+render.When(r.GetFile().GetModified())+" in "+r.ReviewId+" · "+r.GetFile().GetPath())
		}
	}
	return report
}

func (h *handlers) changesReport(_ cliapp.OperationContext, msg *activityv1.GetChangesResponse) cliapp.ListReport {
	report := cliapp.ListReport{Summary: []string{"Since " + render.When(msg.Since) + ": " + msg.Baseline + "."}, ResultsHeading: "Changes"}
	for _, l := range msg.Ledger {
		cut := ""
		if l.Truncated {
			cut = " (cut at 16 KiB)"
		}
		report.Results = append(report.Results, fmt.Sprintf("ledger %s: %d new bytes%s", l.Path, len(l.Text), cut))
	}
	for _, line := range msg.ShowcaseLines {
		report.Results = append(report.Results, "showcase: "+line)
	}
	for _, s := range msg.StatusChanges {
		report.Results = append(report.Results, fmt.Sprintf("workstream %s %s: %s -> %s", s.Topic, s.Workstream, s.Before, s.After))
	}
	for _, f := range msg.Checkpoints {
		report.Results = append(report.Results, "checkpoint: "+f.Path+" "+render.When(f.Modified))
	}
	report.Results = append(report.Results, fmt.Sprintf("%d file(s) written", len(msg.Files)))
	report.Results = append(report.Results, render.Unreadable(msg.Unreadable)...)
	if msg.Truncated {
		report.Results = append(report.Results, "the file walk stopped at its cap; more files may have changed")
	}
	return report
}

// researchWrite says when a topic's run folders were last written, by which
// commission, and what that means when it is paused or ended (rulings P8).
func researchWrite(w *activityv1.ResearchWrite) string {
	at := w.GetAt()
	if !at.GetEvidence().GetKnown() {
		return render.Evidence(at.GetEvidence())
	}
	out := render.When(at.GetValue())
	if w.GetCommissionId() != "" {
		out += " by " + w.GetCommissionId()
	}
	if w.GetNote() != "" {
		out += " (" + w.GetNote() + ")"
	}
	return out
}
