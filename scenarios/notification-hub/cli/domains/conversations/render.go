package conversations

// Human renderings of asks. --json always prints the proto wire shape, so
// everything here is for people and agents reading a terminal.

import (
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

// askOutcomeLine says what the hub did with an ask. Only an ask with a
// --subject-key can be anything but newly created.
func askOutcomeLine(resp *v1.AskResponse) string {
	id := resp.GetAskId()
	switch resp.GetOutcome() {
	case "reasserted":
		return fmt.Sprintf("Ask %s is already open with these terms; nothing new was sent.", id)
	case "updated":
		return fmt.Sprintf("Updated open ask %s in place; no new notification was sent.", id)
	case "replaced":
		return fmt.Sprintf("Created ask %s; it replaced the open ask for this subject.", id)
	case "suppressed":
		return fmt.Sprintf("Ask %s was dismissed with these exact terms; no new ask was created.", id)
	case "returned":
		return fmt.Sprintf("Ask %s was just answered; returning it instead of creating another.", id)
	}
	return fmt.Sprintf("Created ask %s.", id)
}

// askSummary is the one-line form at the top of get and the core of a list row.
func askSummary(ask *v1.Ask) string {
	summary := fmt.Sprintf("%s [%s] %s", ask.GetId(), ask.GetState(), ask.GetQuestion())
	if ask.GetAnswer() != "" {
		summary += " -> " + ask.GetAnswer()
	}
	if ask.GetReason() != "" {
		summary += " (" + ask.GetReason() + ")"
	}
	if packet := packetCounts(ask); packet != "" {
		summary += " {" + packet + "}"
	}
	return summary
}

// askRow is a row of `list`: the summary, then what tells asks apart.
func askRow(ask *v1.Ask) string {
	if who := askWho(ask); who != "" {
		return askSummary(ask) + " | " + who
	}
	return askSummary(ask)
}

// askWho is the part of a list row that tells asks apart: who is asking, how
// urgent it is (only when it is not the ordinary "normal"), and the deadline of
// an ask that still waits. A sort by due date is only legible with the deadline
// printed.
func askWho(ask *v1.Ask) string {
	var parts []string
	if ask.GetSender() != "" {
		parts = append(parts, "from "+ask.GetSender())
	}
	if urgency := ask.GetUrgency(); urgency != "" && urgency != "normal" {
		parts = append(parts, urgency)
	}
	if awaiting(ask) && ask.GetDeadline() != "" {
		parts = append(parts, "due "+ask.GetDeadline())
	}
	return strings.Join(parts, ", ")
}

// awaiting is an ask no answer has closed.
func awaiting(ask *v1.Ask) bool {
	return ask.GetState() == "pending" || ask.GetState() == "escalated"
}

// packetCounts names the decision-packet parts of a list row.
func packetCounts(ask *v1.Ask) string {
	var parts []string
	if n := ask.GetAttachmentCount(); n > 0 {
		parts = append(parts, plural(int(n), "attachment"))
	}
	if n := ask.GetQuestionCount(); n > 0 {
		parts = append(parts, plural(int(n), "question"))
	}
	if n := ask.GetOpenContextRequestCount(); n > 0 {
		parts = append(parts, plural(int(n), "open context request"))
	}
	return strings.Join(parts, ", ")
}

func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// askDetailReport is `conversations get`: the ask, then each part of its
// decision packet that exists.
func askDetailReport(ask *v1.Ask) cliapp.ListReport {
	summary := []string{askSummary(ask)}
	if ask.GetSubjectKey() != "" {
		line := "Subject key: " + ask.GetSubjectKey()
		if n := ask.GetRevisionCount(); n > 0 {
			line += fmt.Sprintf(" (revised %s)", plural(int(n), "time"))
		}
		summary = append(summary, line)
	}
	if ask.GetSender() != "" {
		summary = append(summary, "From: "+ask.GetSender())
	}
	if ask.GetDeadline() != "" {
		summary = append(summary, "Deadline: "+ask.GetDeadline())
	}
	var details []string
	add := func(heading string, lines []string) {
		if len(lines) > 0 {
			details = append(details, heading+":\n"+indentLines(lines))
		}
	}
	add("Options", askOptionLines(ask))
	add("Attachments", attachmentLines(ask.GetAttachments()))
	add("Questions", questionLines(ask.GetQuestions()))
	add("Responses", responseLines(ask.GetResponses()))
	add("Attachment notes", attachmentNoteLines(ask.GetAttachmentNotes()))
	if summaryText := strings.TrimSpace(ask.GetAnswerSummary()); summaryText != "" {
		add("Answer summary", strings.Split(summaryText, "\n"))
	}
	add("Context requests", contextRequestLines(ask.GetContextRequests()))
	return cliapp.ListReport{Summary: summary, ResultsHeading: "Details", Results: details}
}

func indentLines(lines []string) string {
	indented := make([]string, len(lines))
	for i, line := range lines {
		indented[i] = "  " + strings.ReplaceAll(line, "\n", "\n  ")
	}
	return strings.Join(indented, "\n")
}

func askOptionLines(ask *v1.Ask) []string {
	lines := make([]string, 0, len(ask.GetOptions()))
	for _, option := range ask.GetOptions() {
		var marks []string
		if option.GetKey() == ask.GetRecommended() {
			marks = append(marks, "recommended")
		}
		if option.GetKey() == ask.GetDefaultAnswer() {
			marks = append(marks, "default")
		}
		line := option.GetKey() + " = " + option.GetLabel()
		if len(marks) > 0 {
			line += " (" + strings.Join(marks, ", ") + ")"
		}
		lines = append(lines, line)
	}
	return lines
}

func attachmentLines(attachments []*v1.AskAttachment) []string {
	lines := make([]string, 0, len(attachments))
	for _, att := range attachments {
		var facts []string
		if att.GetRole() == v1.AskAttachment_ROLE_EXPLAINER {
			facts = append(facts, "explainer")
		}
		var size int64
		if att.GetKind() == v1.AskAttachment_KIND_FOLDER {
			for _, entry := range att.GetEntries() {
				size += entry.GetSizeBytes()
			}
			facts = append(facts, "folder", plural(len(att.GetEntries()), "file"))
			if att.GetTruncated() {
				facts = append(facts, "truncated")
			}
		} else {
			size = att.GetSizeBytes()
			facts = append(facts, "file")
			if att.GetMediaType() != "" {
				facts = append(facts, att.GetMediaType())
			}
		}
		facts = append(facts, formatBytes(size))
		if att.GetPreviewKind() != "" {
			facts = append(facts, "preview: "+att.GetPreviewKind())
		}
		line := fmt.Sprintf("%s (%s)", att.GetName(), strings.Join(facts, ", "))
		if att.GetLabel() != "" && att.GetLabel() != att.GetName() {
			line += " " + att.GetLabel()
		}
		status := att.GetSourceStatus()
		if status == "" {
			status = "unknown"
		}
		line += " - source " + status
		if att.GetSourcePath() != "" {
			line += ": " + att.GetSourcePath()
		}
		lines = append(lines, line)
	}
	return lines
}

func questionLines(questions []*v1.AskQuestion) []string {
	lines := make([]string, 0, len(questions))
	for _, q := range questions {
		head := fmt.Sprintf("%s [%s] %s", q.GetId(), q.GetType(), q.GetPrompt())
		if q.GetRequired() {
			head += " (required)"
		}
		if v := q.GetUnansweredVerdict(); v != "" {
			head += " (sections left alone: " + v + ")"
		}
		block := []string{head}
		for _, c := range q.GetChoices() {
			line := "  " + c.GetKey() + " = " + c.GetLabel()
			if c.GetAttachment() != "" {
				line += " [" + c.GetAttachment() + "]"
			}
			block = append(block, line)
		}
		for _, s := range q.GetSections() {
			block = append(block, "  "+s.GetKey()+": "+s.GetTitle())
		}
		if rec := q.GetRecommended(); rec != nil {
			line := "  recommended: " + responseText(rec)
			if q.GetRecommendationReason() != "" {
				line += " - " + q.GetRecommendationReason()
			}
			block = append(block, line)
		}
		lines = append(lines, strings.Join(block, "\n"))
	}
	return lines
}

func responseLines(responses []*v1.AskQuestionResponse) []string {
	lines := make([]string, 0, len(responses))
	for _, resp := range responses {
		lines = append(lines, resp.GetQuestionId()+": "+responseText(resp))
	}
	return lines
}

// responseText renders one AskQuestionResponse on a line.
func responseText(resp *v1.AskQuestionResponse) string {
	var parts []string
	if len(resp.GetSelected()) > 0 {
		parts = append(parts, strings.Join(resp.GetSelected(), ", "))
	}
	for _, s := range resp.GetSections() {
		part := s.GetSectionKey() + " " + s.GetVerdict()
		if s.GetNote() != "" {
			part += " (" + s.GetNote() + ")"
		}
		parts = append(parts, part)
	}
	if resp.GetText() != "" {
		parts = append(parts, fmt.Sprintf("%q", resp.GetText()))
	}
	return strings.Join(parts, "; ")
}

// attachmentNoteLines renders the operator's notes pinned to attachments, one
// per line: "character.glb: Hair is too glossy." or "shots/a.png: ..." for a
// file inside a folder.
func attachmentNoteLines(notes []*v1.AskAttachmentNote) []string {
	lines := make([]string, 0, len(notes))
	for _, n := range notes {
		target := n.GetAttachment()
		if n.GetPath() != "" {
			target += "/" + n.GetPath()
		}
		lines = append(lines, target+": "+n.GetNote())
	}
	return lines
}

func contextRequestLines(requests []*v1.AskContextRequest) []string {
	lines := make([]string, 0, len(requests))
	for _, req := range requests {
		state := "unaddressed"
		if req.GetAddressedAt() != "" {
			state = fmt.Sprintf("addressed in revision %d", req.GetAddressedRevision())
		}
		line := fmt.Sprintf("%q (%s, asked %s)", req.GetQuestion(), state, req.GetRequestedAt())
		if req.GetResponse() != "" {
			line += fmt.Sprintf(" answered: %q", req.GetResponse())
		}
		lines = append(lines, line)
	}
	return lines
}
