package launch

import (
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliapp"

	"research-desk/cli/domains/render"

	activityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/activity"
	launchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/launch"
)

func optionsReport(_ cliapp.OperationContext, msg *launchv1.GetLaunchOptionsResponse) cliapp.ListReport {
	automatic := "Automatic launch unavailable: " + msg.AutomaticReason + "."
	if msg.AutomaticAvailable {
		automatic = "Automatic launch: Web Console reachable " + accessWords(msg.WebConsoleAccessKind) + "; whether it accepts the launch shows at launch."
	}
	results := []string{
		fmt.Sprintf("supervisor model: %s (choices %s)", msg.DefaultSupervisorModel, strings.Join(msg.SupervisorModels, ", ")),
		fmt.Sprintf("supervisor effort: %s (choices %s)", msg.DefaultSupervisorEffort, strings.Join(msg.SupervisorEfforts, ", ")),
		fmt.Sprintf("owner model tier: %s (choices %s)", msg.DefaultOwnerModelTier, strings.Join(msg.OwnerModelTiers, ", ")),
		fmt.Sprintf("gates: charter %d bytes, stance %d characters, kernel %d characters", msg.CharterBytesLimit, msg.StanceCharsLimit, msg.KernelCharsLimit),
	}
	if msg.StanceQuotesReason != "" {
		results = append(results, "stance quotes: "+msg.StanceQuotesReason)
	}
	for _, q := range msg.StanceQuotes {
		results = append(results, "    \""+q.Text+"\"")
	}
	for _, t := range msg.TopicWorkstreams {
		if !t.GetEvidence().GetKnown() {
			results = append(results, "workstreams of "+t.Topic+": "+render.Evidence(t.GetEvidence()))
			continue
		}
		results = append(results, fmt.Sprintf("workstreams of %s (%d):", t.Topic, len(t.Workstreams)))
		for _, w := range t.Workstreams {
			line := fmt.Sprintf("    %s %s — %s", w.Id, orDash(w.Status), orDash(w.Title))
			if w.Showcase != "" {
				line += " · showcase: " + w.Showcase
			}
			results = append(results, line)
		}
	}
	return cliapp.ListReport{Summary: []string{automatic, "From " + msg.LimitsPath + "."}, Results: results, ResultsHeading: "Defaults"}
}

// packetLines renders packet.py's result: gates, cards and refusals.
func packetLines(p *launchv1.PacketResult) []string {
	if p == nil {
		return nil
	}
	var out []string
	for _, g := range p.Gates {
		mark := "ok  "
		if !g.Ok {
			mark = "FAIL"
		}
		out = append(out, fmt.Sprintf("%s gate %s: %s (limit %s)", mark, g.Name, g.Value, g.Limit))
	}
	for _, c := range p.Cards {
		line := fmt.Sprintf("card %s: %s, order %s", c.Topic, orDash(c.Mode), orDash(strings.Join(c.Order, " > ")))
		if c.OrderSource != "" {
			line += " (from " + c.OrderSource + ")"
		}
		if c.ReadFirst != "" {
			line += "; read first " + c.ReadFirst
		}
		out = append(out, line)
	}
	for _, e := range p.Errors {
		out = append(out, "refused "+e.Code+": "+e.Message)
	}
	return out
}

func verdictLines(vs []*launchv1.TopicVerdict) []string {
	var out []string
	for _, v := range vs {
		mark := "    "
		if v.Blocks {
			mark = "BLOCK"
		}
		if v.Quiet {
			mark = "quiet"
		}
		out = append(out, fmt.Sprintf("%s %s · %s (%s): %s", mark, v.Topic, v.CommissionId, v.State, v.Message))
		if len(v.Choices) > 0 {
			rec := "none: " + v.RecommendedReason
			if v.Recommended != launchv1.Resolution_RESOLUTION_UNSPECIFIED {
				rec = resolutionWord[v.Recommended] + " (" + v.RecommendedReason + ")"
			}
			out = append(out, fmt.Sprintf("      choose --supersede %s or --alongside %s; recommended: %s", v.CommissionId, v.CommissionId, rec))
		}
		if v.ReadFirst != "" {
			out = append(out, "      read_first: "+v.ReadFirst)
		}
	}
	return out
}

var resolutionWord = map[launchv1.Resolution]string{
	launchv1.Resolution_RESOLUTION_SUPERSEDE:     "supersede",
	launchv1.Resolution_RESOLUTION_RUN_ALONGSIDE: "run alongside",
}

func preflightReport(_ cliapp.OperationContext, msg *launchv1.PreflightResponse) cliapp.ListReport {
	summary := []string{"Launchable: the packet passes and no earlier commission blocks it."}
	if !msg.Launchable {
		summary = []string{fmt.Sprintf("Not launchable (%d blocker(s)): %s.", len(msg.Blockers), strings.Join(msg.Blockers, "; "))}
	}
	if n := msg.NeverStarted; n != nil {
		if n.Assertable {
			summary = append(summary, "Unknown, and the operator may assert it never started (--assert-never-started): "+n.Reason+".")
		} else {
			summary = append(summary, "Unknown, and it cannot be asserted never started: "+n.Reason+".")
		}
	}
	results := append(packetLines(msg.Packet), verdictLines(msg.Topics)...)
	for _, set := range msg.OwnerReading {
		results = append(results, fmt.Sprintf("%s reads %d bytes at start, about %d tokens, %d file(s)", set.Role, set.Bytes, set.EstimatedTokens, len(set.Files)))
		for _, flag := range set.Flags {
			results = append(results, "    flag: "+flag)
		}
	}
	if e := msg.PacketState; e != nil {
		if e.Text == "" {
			results = append(results, "its own "+e.Path+": "+e.Reason)
		} else {
			more := ""
			if e.Truncated {
				more = " (start only)"
			}
			results = append(results, "its own "+e.Path+more+", verbatim:")
			for _, line := range strings.Split(strings.TrimRight(e.Text, "\n"), "\n") {
				results = append(results, "    | "+line)
			}
		}
	}
	return cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Packet and topics",
		Results:        results,
		RetrievalHints: []string{"`build <spec-file>` — write the packet", "`launch <commission-id>` — launch a prepared packet"},
	}
}

func buildReport(_ cliapp.OperationContext, msg *launchv1.BuildResponse) cliapp.ListReport {
	p := msg.GetPacket()
	summary := "Nothing written: packet.py refused the spec."
	if p.GetWritten() {
		summary = "Wrote " + p.GetFolder() + "."
	}
	return cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: "Packet and topics",
		Results:        append(packetLines(p), verdictLines(msg.Topics)...),
		RetrievalHints: []string{"`preflight " + orDash(p.GetCommissionId()) + "` — gate it again before launch", "`launch-messages " + orDash(p.GetCommissionId()) + "` — the two messages to paste"},
	}
}

func messagesReport(_ cliapp.OperationContext, msg *launchv1.GetLaunchMessagesResponse) cliapp.ListReport {
	return cliapp.ListReport{
		Summary: []string{"Nothing was written.", "In a new Codex session, paste the stance and wait for \"ready\", then send /goal followed by the kernel.", "Charter: " + msg.CharterPath},
		Results: []string{"--- stance ---", strings.TrimRight(msg.Stance, "\n"), "--- goal ---", "/goal " + msg.Kernel},
	}
}

func launchReport(_ cliapp.OperationContext, msg *launchv1.LaunchResponse) cliapp.ListReport {
	r := msg.GetLaunch()
	id := r.GetCommissionId()
	summary := []string{"Launch of " + id + " started in the background; each step is recorded in LAUNCH.json."}
	if len(r.GetSupersedes()) > 0 {
		summary = append(summary, "It supersedes "+strings.Join(r.GetSupersedes(), ", ")+": each is stopped through its own session first.")
	}
	if len(r.GetRunAlongside()) > 0 {
		summary = append(summary, "It runs alongside "+strings.Join(r.GetRunAlongside(), ", ")+".")
	}
	if n := r.GetNeverStarted(); n != nil {
		summary = append(summary, "Launched on the operator's word that it never started: "+n.Evidence+".")
	}
	return cliapp.ListReport{Summary: summary, RetrievalHints: []string{"`launch-status " + id + "` — follow its steps"}}
}

func recordLines(r *launchv1.LaunchRecord) []string {
	var out []string
	for _, s := range r.Steps {
		out = append(out, render.When(s.At)+"  "+s.Step)
	}
	if r.FailedStep != "" {
		out = append(out, "FAILED at "+r.FailedStep+": "+r.FailedReason)
	}
	for _, c := range r.Controls {
		line := render.When(c.At) + "  control " + c.Action
		if c.Outcome != "" {
			line += ": " + c.Outcome
		}
		out = append(out, line)
	}
	if r.DiscardedAt != nil {
		out = append(out, render.When(r.DiscardedAt)+"  discarded: "+r.DiscardedReason)
	}
	if len(r.Supersedes) > 0 {
		out = append(out, "supersedes "+strings.Join(r.Supersedes, ", "))
	}
	if len(r.RunAlongside) > 0 {
		out = append(out, "runs alongside "+strings.Join(r.RunAlongside, ", "))
	}
	if r.SupersededBy != "" {
		out = append(out, "superseded by "+r.SupersededBy)
	}
	if n := r.NeverStarted; n != nil {
		out = append(out, render.When(n.At)+"  launched on the operator's word that it never started: "+n.Evidence)
	}
	return out
}

func statusReport(_ cliapp.OperationContext, msg *launchv1.GetLaunchResponse) cliapp.ListReport {
	r := msg.GetLaunch()
	if r.GetPath() == "" {
		return cliapp.ListReport{Summary: []string{r.GetCommissionId() + " has no LAUNCH.json: it was launched by hand or not at all (`commission " + r.GetCommissionId() + "` shows its state)."}}
	}
	summary := fmt.Sprintf("%s, launched by %s; session %s.", r.Path, r.LaunchedBy, orDash(r.WebConsoleSessionId))
	if r.Running {
		summary += " A launch or control is running now."
	}
	return cliapp.ListReport{Summary: []string{summary}, ResultsHeading: "Steps and controls", Results: recordLines(r)}
}

func controlReport(_ cliapp.OperationContext, msg *launchv1.ControlResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{msg.Message + ".", "Recorded in " + msg.GetLaunch().GetPath() + "."}}
}

func testReport(_ cliapp.OperationContext, msg *launchv1.LaunchTestReport) cliapp.ListReport {
	if msg.ReportPath == "" {
		return cliapp.ListReport{Summary: []string{"No launch test has run; `launch-test` starts one."}}
	}
	state := "finished"
	switch {
	case msg.Running:
		state = "running"
	case msg.FailedStep != "":
		state = "failed at " + msg.FailedStep + ": " + msg.FailedReason
	}
	stopped := "its Codex processes are stopped"
	if !msg.Stopped {
		stopped = "its Codex processes are not confirmed stopped"
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Launch test of %s %s; session %s, %s.", render.When(msg.At), state, orDash(msg.SessionId), stopped),
			fmt.Sprintf("Goal status after the goal %s, after pause %s, after resume %s, at the end %s; kernel %d characters.", orDash(msg.StatusAfterGoal), orDash(msg.StatusAfterPause), orDash(msg.StatusAfterResume), orDash(msg.StatusFinal), msg.KernelChars),
			"Report: " + msg.ReportPath,
		},
		ResultsHeading: "Steps",
	}
	for _, s := range msg.Steps {
		report.Results = append(report.Results, render.When(s.At)+"  "+s.Step)
	}
	if msg.Running {
		report.RetrievalHints = []string{"`launch-test-status` — read it again"}
	}
	return report
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// accessWords says how the desk reaches Web Console, from the access kind.
func accessWords(kind activityv1.WebConsoleAccess) string {
	switch kind {
	case activityv1.WebConsoleAccess_WEB_CONSOLE_ACCESS_SERVICE_CREDENTIAL:
		return "with the desk's service credential"
	case activityv1.WebConsoleAccess_WEB_CONSOLE_ACCESS_AVAILABILITY_POLICY:
		return "under its local availability policy, the desk having no credential"
	}
	return "by an access the desk could not settle"
}
