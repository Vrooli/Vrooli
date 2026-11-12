package operator

import (
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliapp"

	"research-desk/cli/domains/render"

	activityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/activity"
	operatorv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/operator"
)

func needsYouReport(_ cliapp.OperationContext, msg *operatorv1.ListNeedsYouResponse) cliapp.ListReport {
	c := msg.GetCourier()
	hub := "Notification Hub read."
	if !c.GetHubRead() {
		hub = "Notification Hub not read: " + strings.TrimSuffix(c.GetHubReason(), ".") + "."
	}
	writer := "This instance runs the courier."
	if !c.GetWriter() {
		writer = "This instance is read-only: " + c.GetWriterReason()
	}
	on := []string{}
	for _, s := range c.GetCommissions() {
		if s.On {
			on = append(on, s.CommissionId)
		}
	}
	global := "off"
	if c.GetGlobalOn() {
		global = "on"
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("%d item(s) need you; %d legacy section(s) are shown, never sent.", len(msg.Items), len(msg.Legacy)),
			fmt.Sprintf("Courier on for: %s; global ATTENTION.md items %s. %s %s", orNone(on), global, hub, writer),
		},
		ResultsHeading: "Needs you",
		ListShaped:     true,
		ResultCount:    len(msg.Items),
		RetrievalHints: []string{"`answer <commission-id> <item-id> <option-key>` — answer into OPERATOR.md", "`courier on <commission-id>` — send this commission's items to Notification Hub"},
	}
	for _, it := range msg.Items {
		owner := it.CommissionId
		if owner == "" {
			owner = "global"
		}
		report.Results = append(report.Results, fmt.Sprintf("%s · %s — %s", owner, it.Id, it.Title))
		if it.Recommended != nil {
			report.Results = append(report.Results, "    recommended: "+it.Recommended.Key+" — "+it.Recommended.Label)
		}
		keys := make([]string, 0, len(it.Options))
		for _, o := range it.Options {
			keys = append(keys, o.Key)
		}
		if it.Answerable {
			report.Results = append(report.Results, "    options: "+strings.Join(keys, ", ")+", other")
		} else {
			report.Results = append(report.Results, "    not answerable here: "+it.NotAnswerableReason)
			if it.Instruction != "" {
				report.Results = append(report.Results, "    it asks: "+it.Instruction)
			}
			for _, l := range it.Links {
				report.Results = append(report.Results, "    link: "+l.Label+" — "+l.Target)
			}
		}
		if it.Answer != nil {
			report.Results = append(report.Results, "    answered: "+render.Entry(it.Answer)+" · seen: "+render.Flag(it.Seen, "a unit has run since", "no unit has run since"))
		}
		step := it.GetCourier()
		courier := fmt.Sprintf("    courier: %s", step.GetAction())
		if step.GetWhy() != "" {
			courier += " — " + step.GetWhy()
		}
		if step.GetAskId() != "" {
			courier += fmt.Sprintf(" (ask %s %s)", step.GetAskId(), step.GetAskState())
		}
		report.Results = append(report.Results, courier, "    in: "+strings.Join(it.Sources, ", "))
	}
	for _, o := range c.GetOrphans() {
		report.Results = append(report.Results, fmt.Sprintf("hub ask %s (%s): %s — %s", o.AskId, o.Key, o.Action, o.Why))
	}
	for _, l := range msg.Legacy {
		report.Results = append(report.Results, "legacy, not sent: "+l.Source)
	}
	if len(msg.Unreadable) > 0 {
		report.Summary = append(report.Summary, fmt.Sprintf("%d file(s) cannot be read: their items and answers are unknown, and the courier sends nothing for them meanwhile.", len(msg.Unreadable)))
		report.Results = append(report.Results, render.Unreadable(msg.Unreadable)...)
	}
	return report
}

func orNone(list []string) string {
	if len(list) == 0 {
		return "no commission"
	}
	return strings.Join(list, ", ")
}

// written reports an append, or with written false the exact text a dry run
// would append; reading says when the supervisor reads it.
func written(path string, done bool, text, warning string, reading *activityv1.EntryReading) cliapp.ListReport {
	summary := "Written to " + path + "."
	if !done {
		summary = "Dry run: nothing written. This text would be appended to " + path + ":"
	}
	report := cliapp.ListReport{Summary: []string{summary}, ResultsHeading: "Text", Results: []string{strings.TrimRight(text, "\n")}}
	if reading != nil {
		report.Summary = append(report.Summary, "Read: "+reading.GetMessage()+".")
	}
	if warning != "" {
		report.Summary = append(report.Summary, "Warning: "+warning)
	}
	return report
}

func answerReport(_ cliapp.OperationContext, msg *operatorv1.AnswerResponse) cliapp.ListReport {
	return written(msg.Path, msg.Written, msg.Text, msg.Warning, msg.Reading)
}

func entryReport(_ cliapp.OperationContext, msg *operatorv1.AddEntryResponse) cliapp.ListReport {
	return written(msg.Path, msg.Written, msg.Text, msg.Warning, msg.Reading)
}

func verdictReport(_ cliapp.OperationContext, msg *operatorv1.AddVerdictResponse) cliapp.ListReport {
	return written(msg.Path, msg.Written, msg.Text, "", nil)
}

func steeringReport(_ cliapp.OperationContext, msg *operatorv1.EnableSteeringResponse) cliapp.ListReport {
	if !msg.Written {
		return cliapp.ListReport{Summary: []string{msg.Path + " already tells the scheduler to read OPERATOR.md; nothing written."}}
	}
	return cliapp.ListReport{Summary: []string{"Appended the Steering section to " + msg.Path + "; the scheduler reads OPERATOR.md from its next unit."}}
}

func reviewsReport(_ cliapp.OperationContext, msg *operatorv1.ListReviewsResponse) cliapp.ListReport {
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d review(s), newest REVIEW.md first.", len(msg.Reviews))},
		ResultsHeading: "Reviews",
		ListShaped:     true,
		ResultCount:    len(msg.Reviews),
		RetrievalHints: []string{"`verdict <review-id> <topic> \"<words>\"` — record a verdict verbatim"},
	}
	for _, r := range msg.Reviews {
		report.Results = append(report.Results, fmt.Sprintf("%s — %s", r.Id, render.Text(r.Title)))
		for _, f := range []*operatorv1.ReviewFile{r.Review, r.ReviewHtml} {
			if f != nil {
				report.Results = append(report.Results, fmt.Sprintf("    %s %s · %s", f.GetFile().GetName(), render.When(f.GetFile().GetModified()), f.Url))
			}
		}
		if len(r.Topics) > 0 {
			report.Results = append(report.Results, "    topics: "+strings.Join(r.Topics, ", "))
		}
		report.Results = append(report.Results, fmt.Sprintf("    %d proposal(s), %d verdict(s)", len(r.Proposals), len(r.Verdicts)))
		for _, line := range render.Unreadable(r.Unreadable) {
			report.Results = append(report.Results, "    "+line)
		}
	}
	report.Results = append(report.Results, render.Unreadable(msg.Unreadable)...)
	return report
}

func courierReport(_ cliapp.OperationContext, msg *operatorv1.SetCourierResponse) cliapp.ListReport {
	state := "off"
	if msg.On {
		state = "on"
	}
	return cliapp.ListReport{Summary: []string{"Courier " + state + ": " + msg.Reason + ".", "Switches: " + msg.SwitchesFile}}
}

func markSeenReport(_ cliapp.OperationContext, msg *operatorv1.MarkSeenResponse) cliapp.ListReport {
	if msg.Path == "" {
		return cliapp.ListReport{Summary: []string{fmt.Sprintf("Caught up at %s: a look recorded at %d commission(s) and %d topic(s); `changes` reports from now.", render.When(msg.At), msg.Commissions, msg.Topics)}}
	}
	return cliapp.ListReport{Summary: []string{"Look recorded at " + render.When(msg.At) + "; `changes` reports from now.", "Look file: " + msg.Path}}
}

func courierTestReport(_ cliapp.OperationContext, msg *operatorv1.TestCourierResponse) cliapp.ListReport {
	delivered := fmt.Sprintf("no device received it within %d s", msg.WaitedSeconds)
	if msg.FirstDeliveredAt != nil {
		delivered = "a device received it at " + render.When(msg.FirstDeliveredAt)
	}
	ask := msg.AskId
	if ask == "" {
		ask = "not yet in Notification Hub"
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Test ask %s (%s): %s; it expires unanswered at %s.", ask, msg.Key, delivered, render.When(msg.ExpiresAt))}}
}
