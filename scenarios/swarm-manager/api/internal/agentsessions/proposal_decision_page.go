package agentsessions

import (
	"fmt"
	"strings"

	"github.com/vrooli/api-core/operatordecision"
)

// proposalCover is a proposal decision in the operator's words: what would be
// added or changed, under what, and what each answer does.
type proposalCover struct {
	subject, question, bigPicture, reason string
	options                               []operatordecision.Option
	addOnly                               bool
}

// firstLine is a long target name cut to its first line: capture targets are
// whole notes, and the question names them, not quotes them.
func firstLine(text string, limit int) string {
	line, _, _ := strings.Cut(strings.TrimSpace(text), "\n")
	return operatordecision.Clip(strings.TrimRight(strings.TrimSpace(line), "."), limit)
}

func itemTitle(m mutationSummary) string {
	if m.Item != nil {
		if title := strings.TrimSpace(m.Item.Title); title != "" {
			return title
		}
		if name := strings.TrimSpace(m.Item.Name); name != "" {
			return name
		}
	}
	return strings.TrimSpace(m.Target)
}

func newProposalCover(proposal Proposal, target string, mutations []mutationSummary, parseErr error) proposalCover {
	where := firstLine(target, 90)
	n := len(mutations)
	c := proposalCover{addOnly: parseErr == nil && n > 0}
	for _, m := range mutations {
		if !strings.EqualFold(strings.TrimSpace(m.Op), "add_item") {
			c.addOnly = false
		}
	}
	switch {
	case c.addOnly && n == 1:
		c.question = fmt.Sprintf("Add the backlog item “%s” to “%s”?", operatordecision.Clip(itemTitle(mutations[0]), 90), where)
	case c.addOnly:
		c.question = fmt.Sprintf("Add %d proposed backlog items to “%s”?", n, where)
	case parseErr == nil && n > 0:
		c.question = fmt.Sprintf("Apply %s to “%s”?", operatordecision.Count(n, "proposed change", "proposed changes"), where)
	default:
		c.question = fmt.Sprintf("Apply the proposed changes to “%s”?", where)
	}
	c.subject = operatordecision.Clip("Review: "+strings.TrimSuffix(c.question, "?"), 110)
	if c.addOnly {
		c.bigPicture = fmt.Sprintf("An agent reviewed “%s” and proposes %s. Approving records %s for later implementation; it does not implement that work.", where, operatordecision.Count(n, "backlog item", "backlog items"), operatordecision.Plural(n, "it", "them"))
		c.options = []operatordecision.Option{
			{Key: "approve", Label: fmt.Sprintf("Add %s: record %s in the backlog; no work starts yet", operatordecision.Plural(n, "it", "them"), operatordecision.Plural(n, "this item", fmt.Sprintf("these %d items", n)))},
			{Key: "reject", Label: "Reject: discard the proposal and add nothing"},
			{Key: "request_changes", Label: "Request changes: say what to change in the note, and the agent revises it"},
		}
		c.reason = "It passed validation, and adding backlog items only records work; nothing is built until someone picks it up."
	} else {
		if parseErr == nil && n > 0 {
			c.bigPicture = fmt.Sprintf("An agent proposes %s to “%s”. Approving applies %s to the work graph.", operatordecision.Count(n, "change", "changes"), where, operatordecision.Plural(n, "it", "them"))
		} else {
			c.bigPicture = fmt.Sprintf("An agent proposes changes to “%s”. The review link shows them exactly.", where)
		}
		c.options = []operatordecision.Option{
			{Key: "approve", Label: "Apply the listed changes"},
			{Key: "reject", Label: "Reject: discard the proposal and change nothing"},
			{Key: "request_changes", Label: "Request changes: say what to change in the note, and the agent revises it"},
		}
		c.reason = "It passed validation, and a change to the work graph can itself be reviewed and reverted later."
	}
	c.bigPicture += " If you reject it, nothing changes. If you request changes, the agent revises the proposal and asks again. Nothing is applied without your answer."
	return c
}

var opWords = map[string]string{
	"add_item": "Add", "update_item": "Change", "remove_item": "Remove", "link": "Link", "unlink": "Unlink",
}

// page shows each proposed item as a card with its purpose, size and what
// "done" means, so the operator judges the work itself.
func (c proposalCover) page(facts operatordecision.Facts, proposal Proposal, mutations []mutationSummary) string {
	page := operatordecision.NewPage(c.question)
	if summary := strings.TrimSpace(proposal.Summary); summary != "" {
		page.Lead(summary)
	}
	rows := []operatordecision.Row{{Label: "Under", Value: firstLine(targetName(proposal), 200)}}
	if len(mutations) > 0 {
		rows = append(rows, operatordecision.Row{Label: "Proposed", Value: operatordecision.Count(len(mutations), "change", "changes")})
	}
	page.Facts(rows...)
	for i, m := range mutations {
		if i == 25 {
			page.Callout("", "More on the review page:", fmt.Sprintf("%d more changes are not shown here.", len(mutations)-i))
			break
		}
		verb := opWords[strings.ToLower(strings.TrimSpace(m.Op))]
		if verb == "" {
			verb = strings.TrimSpace(m.Op)
		}
		text := strings.TrimSpace(m.Rationale)
		if m.Item != nil && strings.TrimSpace(m.Item.Description) != "" {
			text = strings.TrimSpace(m.Item.Description) + "\n\nWhy: " + text
		}
		page.Section(fmt.Sprintf("%s: %s", verb, itemTitle(m)), text)
		if m.Item == nil {
			continue
		}
		var meta []operatordecision.Row
		if kind := strings.TrimSpace(m.Item.Kind); kind != "" {
			meta = append(meta, operatordecision.Row{Label: "Kind", Value: kind})
		}
		if m.Item.Priority != nil {
			meta = append(meta, operatordecision.Row{Label: "Priority", Value: fmt.Sprint(*m.Item.Priority)})
		}
		if effort := strings.TrimSpace(m.Item.Effort); effort != "" {
			meta = append(meta, operatordecision.Row{Label: "Effort", Value: effort})
		}
		page.Facts(meta...)
		var done [][]string
		for _, criterion := range m.Item.AcceptanceAllow {
			if criterion = strings.TrimSpace(criterion); criterion != "" {
				done = append(done, []string{"Must", criterion})
			}
		}
		for _, criterion := range m.Item.AcceptanceDeny {
			if criterion = strings.TrimSpace(criterion); criterion != "" {
				done = append(done, []string{"Must not", criterion})
			}
		}
		page.Table([]string{"Done means", "What"}, done)
	}
	effects := map[string]string{
		"reject":          "The proposal is discarded. Nothing changes.",
		"request_changes": "Write what should change in the note. The agent revises the proposal and you get a new question.",
		"approve":         "The listed changes are applied to the work graph.",
	}
	if c.addOnly {
		effects["approve"] = "The items are recorded in the backlog. Nothing is built until someone picks them up."
	}
	return page.Options(facts, effects).
		IfNoAnswer("Nothing changes. The proposal waits in Swarm Manager's review queue.").
		HTML()
}

func targetName(proposal Proposal) string {
	if proposal.Target == nil {
		return ""
	}
	if name := strings.TrimSpace(proposal.Target.Name); name != "" {
		return name
	}
	return proposal.Target.Ref
}
