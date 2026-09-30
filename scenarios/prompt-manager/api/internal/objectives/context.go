package objectives

import "sort"

// This file owns the runtime read contract for plan
// aquila-objective-authority-and-team-editor Phase 6 (Align objective read
// consumers and restatement signals).
//
// The runtime consumer (the heartbeat prompt builder and any team-scoped
// agent) needs one ordered, revision-stamped view of a team's objective
// obligations. Before this read model the only consumer surfaces were the
// coverage join (objective-major) and ListTeamAttachments (attachment-major,
// without objective identity/class/revision). A runtime that read either one
// had to re-join them and could disagree with the other about identity,
// ordering or revision. TeamObjectiveContext is that single read: a team's
// attachments in priority order, each carrying the objective's current meaning
// revision and the team's acknowledgement/restatement state.
//
// It is a read model. It owns no state and writes nothing; the objective
// domain remains the only owner. BuildTeamObjectiveContext is pure over the
// authority projection, so the coverage read and the future runtime consumer
// observe byte-identical ordering and revisions.

// TeamObjectiveReference is one objective attached to a team in the runtime
// read, in the team's priority order.
type TeamObjectiveReference struct {
	ObjectiveID     string `json:"objectiveId"`
	Title           string `json:"title"`
	Class           string `json:"class"`
	GlobalOrder     int    `json:"globalOrder"`
	MeaningRevision string `json:"meaningRevision"`
	Role            string `json:"role,omitempty"`
	Coverage        string `json:"coverage,omitempty"`
	Note            string `json:"note,omitempty"`
	Priority        int    `json:"priority"`
	// AcknowledgedRevision is the meaning revision the team last confirmed.
	// RestatementPending is true when it does not equal MeaningRevision, so a
	// strict meaning change re-pends while a pure reorder never does.
	AcknowledgedRevision string `json:"acknowledgedRevision,omitempty"`
	RestatementPending   bool   `json:"restatementPending"`
}

// TeamObjectiveContext is the ordered runtime read of one team's objective
// obligations. AttachmentRevision moves on attachment, order, role, coverage
// and acknowledgement changes; it is deliberately independent of any
// objective's meaning revision so an order change cannot look like a
// restatement obligation.
type TeamObjectiveContext struct {
	TeamID             string                   `json:"teamId"`
	AttachmentRevision string                   `json:"attachmentRevision,omitempty"`
	Objectives         []TeamObjectiveReference `json:"objectives"`
}

// BuildTeamObjectiveContext renders one team's runtime read from the authority
// projection. Objective facts come from the projection index, so identity,
// ordering and meaning revision cannot drift between this read and the
// objective-major coverage rows. An attachment whose objective is absent from
// the index is skipped rather than fabricated; the projection's validation
// reports that dangling attachment as objective_unknown_id.
func BuildTeamObjectiveContext(team ProjectedTeam, objectives map[string]ProjectedObjective) TeamObjectiveContext {
	out := TeamObjectiveContext{
		TeamID:             team.TeamID,
		AttachmentRevision: team.AttachmentRevision,
		Objectives:         make([]TeamObjectiveReference, 0, len(team.Attachments)),
	}
	for _, a := range team.Attachments {
		obj, ok := objectives[normalizeKey(a.ObjectiveID)]
		if !ok {
			continue
		}
		out.Objectives = append(out.Objectives, TeamObjectiveReference{
			ObjectiveID:          a.ObjectiveID,
			Title:                obj.Title,
			Class:                obj.Class,
			GlobalOrder:          obj.GlobalOrder,
			MeaningRevision:      obj.MeaningRevision,
			Role:                 a.Role,
			Coverage:             a.Coverage,
			Note:                 a.Note,
			Priority:             a.Priority,
			AcknowledgedRevision: a.AcknowledgedRevision,
			RestatementPending:   a.RestatementPending,
		})
	}
	return out
}

// BuildTeamObjectiveContexts renders the same read for every team the
// authority tracks, sorted by team id for deterministic output. It is the
// team-major half of the coverage read and the contract the runtime child
// consumes.
func BuildTeamObjectiveContexts(p Projection) []TeamObjectiveContext {
	index := make(map[string]ProjectedObjective, len(p.Objectives))
	for _, o := range p.Objectives {
		index[normalizeKey(o.ID)] = o
	}
	out := make([]TeamObjectiveContext, 0, len(p.Teams))
	for _, team := range p.Teams {
		out = append(out, BuildTeamObjectiveContext(team, index))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].TeamID < out[j].TeamID })
	return out
}
