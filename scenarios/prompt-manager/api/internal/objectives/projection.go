package objectives

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// This file owns the read-only documentation projection for plan
// aquila-objective-authority-and-team-editor Phase 3 step 7 (decision
// 02e4dd6b-2645-484c-b0e0-7c0ceb790d5d).
//
// The objective authority is the single owner of objective state. The operator's
// statement (docs/director-swarm/strategy/OBJECTIVES.md) and each team.json
// `objectivesServed` block stay authored inputs. This file renders the authority
// as an explicit projection and reports drift against those retained
// declarations. It never writes the statement, never writes the declarations,
// and never creates an objectives.json: D9/D10 keep Prompt Manager on the state
// side of the boundary.

// ProjectedObjective is one objective in the read-only authority projection.
type ProjectedObjective struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Class           string `json:"class"`
	EvidenceSource  string `json:"evidenceSource,omitempty"`
	HasEvidence     bool   `json:"hasEvidence"`
	GapMarker       string `json:"gapMarker,omitempty"`
	GlobalOrder     int    `json:"globalOrder"`
	MeaningRevision string `json:"meaningRevision"`
}

// ProjectedTeam is one team's authority attachments in priority order.
type ProjectedTeam struct {
	TeamID             string       `json:"teamId"`
	AttachmentRevision string       `json:"attachmentRevision,omitempty"`
	Attachments        []Attachment `json:"attachments"`
}

// Projection is the deterministic, read-only rendering of the current objective
// authority. It is a view, not a second source of truth.
type Projection struct {
	Objectives []ProjectedObjective `json:"objectives"`
	Teams      []ProjectedTeam      `json:"teams"`
	Relations  []Relation           `json:"relations"`
	Validation ValidationResult     `json:"validation"`
}

// Drift kinds reported by CompareProjection. They describe a disagreement
// between the retained declarations and the authority; the comparison does not
// decide which side is correct.
const (
	DriftObjectiveMissingInAuthority  = "objective-missing-in-authority"
	DriftObjectiveUntrackedInSource   = "objective-untracked-in-source"
	DriftObjectiveMeaningChanged      = "objective-meaning-changed"
	DriftObjectiveOrderChanged        = "objective-order-changed"
	DriftAttachmentMissingInAuthority = "attachment-missing-in-authority"
	DriftAttachmentUntrackedInSource  = "attachment-untracked-in-source"
	DriftAttachmentRoleChanged        = "attachment-role-changed"
	DriftAttachmentCoverageChanged    = "attachment-coverage-changed"
	DriftAttachmentNoteChanged        = "attachment-note-changed"
	DriftAttachmentPriorityChanged    = "attachment-priority-changed"
	DriftAcknowledgementChanged       = "acknowledgement-changed"
	DriftRestatementPendingChanged    = "restatement-pending-changed"
)

// DriftItem is one declaration-to-authority disagreement.
type DriftItem struct {
	Kind        string `json:"kind"`
	ObjectiveID string `json:"objectiveId,omitempty"`
	TeamID      string `json:"teamId,omitempty"`
	Declared    string `json:"declared,omitempty"`
	Authority   string `json:"authority,omitempty"`
	Detail      string `json:"detail"`
}

// ProjectionDrift is the complete drift report between the retained declarations
// and the authority projection.
type ProjectionDrift struct {
	HasDrift bool        `json:"hasDrift"`
	Items    []DriftItem `json:"items,omitempty"`
	// CoverageGapsDeclared are gaps the declarations themselves report (the
	// table names a team that does not declare the objective back). They are
	// surfaced for context, not counted as drift.
	CoverageGapsDeclared []string `json:"coverageGapsDeclared,omitempty"`
}

// BuildProjection renders the current authority without mutating it. It reads
// objectives, relations, every team's attachments and the domain validation.
func BuildProjection(ctx context.Context, svc *Service) (Projection, error) {
	objs, err := svc.ListObjectives(ctx)
	if err != nil {
		return Projection{}, fmt.Errorf("objectives: projection list objectives: %w", err)
	}
	out := Projection{}
	for _, o := range objs {
		out.Objectives = append(out.Objectives, ProjectedObjective{
			ID:              o.ID,
			Title:           o.Title,
			Class:           string(o.Class),
			EvidenceSource:  o.EvidenceSource,
			HasEvidence:     o.HasEvidence,
			GapMarker:       o.GapMarker,
			GlobalOrder:     o.GlobalOrder,
			MeaningRevision: o.MeaningRevision,
		})
	}
	sort.SliceStable(out.Objectives, func(i, j int) bool {
		if out.Objectives[i].GlobalOrder != out.Objectives[j].GlobalOrder {
			return out.Objectives[i].GlobalOrder < out.Objectives[j].GlobalOrder
		}
		return out.Objectives[i].ID < out.Objectives[j].ID
	})

	rels, err := svc.ListRelations(ctx)
	if err != nil {
		return Projection{}, fmt.Errorf("objectives: projection list relations: %w", err)
	}
	out.Relations = rels

	teamIDs, err := svc.ListTeamIDs(ctx)
	if err != nil {
		return Projection{}, fmt.Errorf("objectives: projection list teams: %w", err)
	}
	for _, teamID := range teamIDs {
		atts, err := svc.ListTeamAttachments(ctx, teamID)
		if err != nil {
			return Projection{}, fmt.Errorf("objectives: projection list attachments %q: %w", teamID, err)
		}
		rev, err := svc.TeamAttachmentRevision(ctx, teamID)
		if err != nil {
			return Projection{}, fmt.Errorf("objectives: projection team revision %q: %w", teamID, err)
		}
		out.Teams = append(out.Teams, ProjectedTeam{TeamID: teamID, AttachmentRevision: rev, Attachments: atts})
	}

	validation, err := svc.Validate(ctx)
	if err != nil {
		return Projection{}, fmt.Errorf("objectives: projection validate: %w", err)
	}
	out.Validation = validation
	return out, nil
}

// CompareProjection reports where the retained declarations and the authority
// disagree. It is pure: both inputs are read-only views.
func CompareProjection(declared ImportPreview, authority Projection) ProjectionDrift {
	drift := ProjectionDrift{CoverageGapsDeclared: declared.CoverageGaps}

	authorityByID := map[string]ProjectedObjective{}
	for _, o := range authority.Objectives {
		authorityByID[normalizeKey(o.ID)] = o
	}
	declaredIDs := map[string]bool{}
	for _, o := range declared.Objectives {
		key := normalizeKey(o.ID)
		declaredIDs[key] = true
		auth, ok := authorityByID[key]
		if !ok {
			drift.Items = append(drift.Items, DriftItem{
				Kind:        DriftObjectiveMissingInAuthority,
				ObjectiveID: o.ID,
				Declared:    o.NewMeaningRevision,
				Detail:      fmt.Sprintf("declaration defines %q but the authority does not track it", o.ID),
			})
			continue
		}
		if auth.MeaningRevision != o.NewMeaningRevision {
			drift.Items = append(drift.Items, DriftItem{
				Kind:        DriftObjectiveMeaningChanged,
				ObjectiveID: o.ID,
				Declared:    o.NewMeaningRevision,
				Authority:   auth.MeaningRevision,
				Detail:      fmt.Sprintf("meaning revision for %q differs between the declaration and the authority", o.ID),
			})
		}
		if auth.GlobalOrder != o.GlobalOrder {
			drift.Items = append(drift.Items, DriftItem{
				Kind:        DriftObjectiveOrderChanged,
				ObjectiveID: o.ID,
				Declared:    fmt.Sprintf("%d", o.GlobalOrder),
				Authority:   fmt.Sprintf("%d", auth.GlobalOrder),
				Detail:      fmt.Sprintf("global order for %q differs", o.ID),
			})
		}
	}
	for _, o := range authority.Objectives {
		if !declaredIDs[normalizeKey(o.ID)] {
			drift.Items = append(drift.Items, DriftItem{
				Kind:        DriftObjectiveUntrackedInSource,
				ObjectiveID: o.ID,
				Authority:   o.MeaningRevision,
				Detail:      fmt.Sprintf("authority tracks %q which the retained declarations do not define", o.ID),
			})
		}
	}

	authorityAtts := map[string]Attachment{}
	for _, team := range authority.Teams {
		for _, a := range team.Attachments {
			authorityAtts[attachmentKey(a.TeamID, a.ObjectiveID)] = a
		}
	}
	declaredAtts := map[string]bool{}
	for _, a := range declared.Attachments {
		key := attachmentKey(a.TeamID, a.ObjectiveID)
		declaredAtts[key] = true
		auth, ok := authorityAtts[key]
		if !ok {
			drift.Items = append(drift.Items, DriftItem{
				Kind:        DriftAttachmentMissingInAuthority,
				ObjectiveID: a.ObjectiveID,
				TeamID:      a.TeamID,
				Detail:      fmt.Sprintf("team %q declares %q but the authority has no attachment", a.TeamID, a.ObjectiveID),
			})
			continue
		}
		compareAttachment(&drift, a, auth)
	}
	for key, a := range authorityAtts {
		if declaredAtts[key] {
			continue
		}
		drift.Items = append(drift.Items, DriftItem{
			Kind:        DriftAttachmentUntrackedInSource,
			ObjectiveID: a.ObjectiveID,
			TeamID:      a.TeamID,
			Detail:      fmt.Sprintf("authority attaches %q to team %q but the declarations do not", a.ObjectiveID, a.TeamID),
		})
	}

	sort.SliceStable(drift.Items, func(i, j int) bool {
		if drift.Items[i].Kind != drift.Items[j].Kind {
			return drift.Items[i].Kind < drift.Items[j].Kind
		}
		if drift.Items[i].TeamID != drift.Items[j].TeamID {
			return drift.Items[i].TeamID < drift.Items[j].TeamID
		}
		return drift.Items[i].ObjectiveID < drift.Items[j].ObjectiveID
	})
	drift.HasDrift = len(drift.Items) > 0
	return drift
}

func compareAttachment(drift *ProjectionDrift, declared ImportedAttachment, auth Attachment) {
	if auth.Role != declared.Role {
		drift.Items = append(drift.Items, DriftItem{
			Kind:        DriftAttachmentRoleChanged,
			ObjectiveID: declared.ObjectiveID,
			TeamID:      declared.TeamID,
			Declared:    declared.Role,
			Authority:   auth.Role,
			Detail:      fmt.Sprintf("role for %s/%s differs", declared.TeamID, declared.ObjectiveID),
		})
	}
	if auth.Coverage != declared.Coverage {
		drift.Items = append(drift.Items, DriftItem{
			Kind:        DriftAttachmentCoverageChanged,
			ObjectiveID: declared.ObjectiveID,
			TeamID:      declared.TeamID,
			Declared:    declared.Coverage,
			Authority:   auth.Coverage,
			Detail:      fmt.Sprintf("coverage for %s/%s differs", declared.TeamID, declared.ObjectiveID),
		})
	}
	if auth.Note != declared.Note {
		drift.Items = append(drift.Items, DriftItem{
			Kind:        DriftAttachmentNoteChanged,
			ObjectiveID: declared.ObjectiveID,
			TeamID:      declared.TeamID,
			Declared:    declared.Note,
			Authority:   auth.Note,
			Detail:      fmt.Sprintf("note for %s/%s differs", declared.TeamID, declared.ObjectiveID),
		})
	}
	if auth.Priority != declared.Priority {
		drift.Items = append(drift.Items, DriftItem{
			Kind:        DriftAttachmentPriorityChanged,
			ObjectiveID: declared.ObjectiveID,
			TeamID:      declared.TeamID,
			Declared:    fmt.Sprintf("%d", declared.Priority),
			Authority:   fmt.Sprintf("%d", auth.Priority),
			Detail:      fmt.Sprintf("priority for %s/%s differs", declared.TeamID, declared.ObjectiveID),
		})
	}
	if auth.AcknowledgedRevision != declared.NewAcknowledgedRevision {
		drift.Items = append(drift.Items, DriftItem{
			Kind:        DriftAcknowledgementChanged,
			ObjectiveID: declared.ObjectiveID,
			TeamID:      declared.TeamID,
			Declared:    declared.NewAcknowledgedRevision,
			Authority:   auth.AcknowledgedRevision,
			Detail:      fmt.Sprintf("acknowledged revision for %s/%s differs", declared.TeamID, declared.ObjectiveID),
		})
	}
	if auth.RestatementPending != declared.RestatementPending {
		drift.Items = append(drift.Items, DriftItem{
			Kind:        DriftRestatementPendingChanged,
			ObjectiveID: declared.ObjectiveID,
			TeamID:      declared.TeamID,
			Declared:    fmt.Sprintf("%t", declared.RestatementPending),
			Authority:   fmt.Sprintf("%t", auth.RestatementPending),
			Detail:      fmt.Sprintf("restatement-pending state for %s/%s differs", declared.TeamID, declared.ObjectiveID),
		})
	}
}

// RenderProjection renders the authority as deterministic markdown. It is a
// generated view: callers may print or retain it, but it must never overwrite
// the operator statement.
func RenderProjection(p Projection) string {
	var b strings.Builder
	b.WriteString("# Objective authority projection (read-only)\n\n")
	b.WriteString("Generated from the Prompt Manager objective authority. This is a projection, not the operator statement; do not edit it. ")
	b.WriteString("The operator statement remains docs/director-swarm/strategy/OBJECTIVES.md.\n\n")

	fmt.Fprintf(&b, "Objectives: %d | Teams: %d | Relations: %d | Validation: %d error(s), %d warning(s)\n\n",
		len(p.Objectives), len(p.Teams), len(p.Relations), p.Validation.Errors, p.Validation.Warnings)

	b.WriteString("## Objectives\n\n")
	b.WriteString("| Order | ID | Class | Meaning revision | Measurable | Evidence | Title |\n")
	b.WriteString("|---|---|---|---|---|---|---|\n")
	for _, o := range p.Objectives {
		fmt.Fprintf(&b, "| %d | %s | %s | %s | %t | %s | %s |\n",
			o.GlobalOrder, escapeCell(o.ID), escapeCell(o.Class), escapeCell(o.MeaningRevision),
			o.HasEvidence, escapeCell(o.EvidenceSource), escapeCell(o.Title))
	}

	b.WriteString("\n## Team attachments\n\n")
	for _, t := range p.Teams {
		fmt.Fprintf(&b, "### %s (attachment revision %s)\n\n", t.TeamID, t.AttachmentRevision)
		if len(t.Attachments) == 0 {
			b.WriteString("_No objectives attached._\n\n")
			continue
		}
		for _, a := range t.Attachments {
			fmt.Fprintf(&b, "- %s (priority %d, role=%s, coverage=%s, ack=%s, restatement-pending=%t)\n",
				a.ObjectiveID, a.Priority, emptyDash(a.Role), emptyDash(a.Coverage), emptyDash(a.AcknowledgedRevision), a.RestatementPending)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// RenderDrift renders the drift report as deterministic markdown or a
// confirming line when there is none.
func RenderDrift(d ProjectionDrift) string {
	var b strings.Builder
	b.WriteString("## Declaration drift\n\n")
	if !d.HasDrift {
		b.WriteString("No drift between the retained declarations and the authority.\n")
	} else {
		fmt.Fprintf(&b, "%d drift item(s):\n\n", len(d.Items))
		for _, item := range d.Items {
			fmt.Fprintf(&b, "- [%s] %s: %s (declared=%q authority=%q)\n",
				item.Kind, firstNonEmpty(item.TeamID+"/"+item.ObjectiveID, item.ObjectiveID), item.Detail, item.Declared, item.Authority)
		}
	}
	if len(d.CoverageGapsDeclared) > 0 {
		fmt.Fprintf(&b, "\nDeclared coverage gaps (context, not drift): %s\n", strings.Join(d.CoverageGapsDeclared, ", "))
	}
	return b.String()
}

func normalizeKey(id string) string { return strings.ToUpper(strings.TrimSpace(id)) }

func attachmentKey(teamID, objectiveID string) string {
	return strings.TrimSpace(teamID) + "\x00" + normalizeKey(objectiveID)
}

func escapeCell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

func emptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
