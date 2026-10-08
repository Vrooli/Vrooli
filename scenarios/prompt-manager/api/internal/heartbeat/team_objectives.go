// Team objectives — the setpoint a team is measured against.
//
// A team's prompt used to say nothing about which objectives the team was
// accountable to. A member could execute its lane perfectly and still be
// unable to notice that its work no longer followed from the objective it was
// attached to, because the prompt never named the objective or its revision.
//
// The objective authority is the single owner of objective definitions and
// ordered team attachments. This section consumes its ordered, revision-
// stamped runtime read so the prompt and every other consumer cannot disagree
// about identity, order or revision. It renders nothing when a team has no
// attachments and renders an explicit unavailable section when the read fails:
// a member must never mistake "could not read" for "no obligations".
package heartbeat

import (
	"context"
	"fmt"
	"strings"
)

// TeamObjective is one objective attached to a team in the runtime read, in the
// team's priority order. It is a narrow consumer shape: the adapter that fills
// it from the objective authority lives with the owner wiring, so this package
// stays independent of the authority's storage.
type TeamObjective struct {
	ObjectiveID          string
	Title                string
	Class                string
	GlobalOrder          int
	MeaningRevision      string
	Role                 string
	Note                 string
	Priority             int
	AcknowledgedRevision string
	RestatementPending   bool
}

// TeamObjectiveContext is the ordered, revision-stamped runtime read of one
// team's objective obligations. AttachmentRevision moves on attachment, order,
// role, coverage and acknowledgement changes; it is independent of any single
// objective's meaning revision so an order change cannot look like a
// restatement obligation.
type TeamObjectiveContext struct {
	TeamID             string
	AttachmentRevision string
	Objectives         []TeamObjective
}

// TeamObjectiveProvider supplies the canonical objective authority read for one
// team. Implementations are expected to be cheap to call once per prompt build.
type TeamObjectiveProvider interface {
	TeamObjectives(ctx context.Context, teamID string) (TeamObjectiveContext, error)
}

// SetTeamObjectiveProvider wires the objective authority read. When unset, the
// builder omits the section rather than claiming a setpoint it never read.
func (b *PromptBuilder) SetTeamObjectiveProvider(provider TeamObjectiveProvider) {
	if b == nil {
		return
	}
	b.teamObjectives = provider
}

func (b *PromptBuilder) buildTeamObjectivesSection(ctx context.Context, teamID string) string {
	if b == nil || b.teamObjectives == nil {
		return ""
	}
	oc, err := b.teamObjectives.TeamObjectives(ctx, teamID)
	if err != nil {
		return renderTeamObjectivesUnavailable(teamID, err)
	}
	if len(oc.Objectives) == 0 {
		return ""
	}
	return renderTeamObjectives(oc)
}

func renderTeamObjectives(oc TeamObjectiveContext) string {
	var b strings.Builder
	b.WriteString(promptHeading(promptSectionKindTeamObjectives) + "\n\n")
	b.WriteString("Your team is accountable to these objectives, in priority order. This is the canonical objective authority read; treat it as the setpoint your instrument measures against and do not restate or edit the list in team documents.\n\n")

	if rev := strings.TrimSpace(oc.AttachmentRevision); rev != "" {
		b.WriteString(fmt.Sprintf("Attachment revision: `%s`.\n\n", rev))
	}

	for i, o := range oc.Objectives {
		title := strings.TrimSpace(o.Title)
		if title == "" {
			title = o.ObjectiveID
		}
		b.WriteString(fmt.Sprintf("%d. **%s** (`%s`)", i+1, title, o.ObjectiveID))
		if class := strings.TrimSpace(o.Class); class != "" {
			b.WriteString(" — class `" + class + "`")
		}
		if rev := strings.TrimSpace(o.MeaningRevision); rev != "" {
			b.WriteString(", meaning `" + rev + "`")
		}
		if o.RestatementPending {
			if ack := strings.TrimSpace(o.AcknowledgedRevision); ack != "" {
				b.WriteString(", restatement pending (last acknowledged `" + ack + "`)")
			} else {
				b.WriteString(", restatement pending")
			}
		}
		if role := strings.TrimSpace(o.Role); role != "" {
			b.WriteString("; role `" + role + "`")
		}
		b.WriteString("\n")
		if note := strings.TrimSpace(o.Note); note != "" {
			b.WriteString("   " + note + "\n")
		}
	}

	b.WriteString("\nA changed **meaning revision** means the objective's meaning changed; re-read your commitments against it. `restatement pending` means the team has not yet acknowledged the current meaning.\n")
	return strings.TrimRight(b.String(), "\n")
}

func renderTeamObjectivesUnavailable(teamID string, err error) string {
	var b strings.Builder
	b.WriteString(promptHeading(promptSectionKindTeamObjectives) + "\n\n")
	b.WriteString(fmt.Sprintf("The objective authority could not be read for team `%s` this run", teamID))
	if err != nil {
		b.WriteString(fmt.Sprintf(" (%v)", err))
	}
	b.WriteString(". Treat your team's objective obligations as unknown, not empty: do not claim coverage of a setpoint you could not read, and report the failure in your continuity record.\n")
	return strings.TrimRight(b.String(), "\n")
}
