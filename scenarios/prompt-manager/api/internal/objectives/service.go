package objectives

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// TeamLookup reports whether a team id is known to the team domain. It is
// optional; when nil, team existence is not checked.
type TeamLookup func(teamID string) bool

// Service is the single owner of objective validation and mutation. Every
// caller (HTTP, Connect, CLI or the editor's typed client) goes through it, so
// all clients observe the same rules.
type Service struct {
	repo      Repository
	teamKnown TeamLookup
}

// NewService builds the objective application service.
func NewService(repo Repository, teamKnown TeamLookup) *Service {
	return &Service{repo: repo, teamKnown: teamKnown}
}

// ListObjectives returns every objective in persisted global order.
func (s *Service) ListObjectives(ctx context.Context) ([]Objective, error) {
	return s.repo.ListObjectives(ctx)
}

// GetObjective returns one objective by id.
func (s *Service) GetObjective(ctx context.Context, id string) (Objective, bool, error) {
	return s.repo.GetObjective(ctx, strings.TrimSpace(id))
}

// ListTeamIDs returns the distinct team ids that currently hold an attachment,
// in stable order. It is the read seam for a whole-authority projection.
func (s *Service) ListTeamIDs(ctx context.Context) ([]string, error) {
	return s.repo.ListTeamIDs(ctx)
}

// UpsertObjective creates or updates an objective. A new objective requires an
// empty expectedMeaningRevision; an update requires the caller's view of the
// current meaning revision so concurrent edits cannot silently overwrite.
func (s *Service) UpsertObjective(ctx context.Context, o Objective, expectedMeaningRevision string) (Objective, error) {
	o.ID = strings.TrimSpace(o.ID)
	o.Title = strings.TrimSpace(o.Title)
	o.EvidenceSource = strings.TrimSpace(o.EvidenceSource)
	o.GapMarker = strings.TrimSpace(o.GapMarker)
	o.Class = Class(strings.ToLower(strings.TrimSpace(string(o.Class))))
	if o.ID == "" {
		return Objective{}, &ValidationError{Err: ErrUnknownObjective, Field: "id", Msg: "is required"}
	}
	if !validClass(o.Class) {
		return Objective{}, &ValidationError{Err: ErrInvalidClass, Field: "class", Msg: fmt.Sprintf("%q is not terminal or instrumental", o.Class)}
	}
	if o.Title == "" {
		return Objective{}, &ValidationError{Err: ErrInvalidClass, Field: "title", Msg: "is required"}
	}
	o.HasEvidence = o.EvidenceSource != ""
	o.MeaningRevision = ComputeMeaningRevision(o)

	var result Objective
	err := s.repo.WithTx(ctx, func(tx Repository) error {
		existing, ok, err := tx.GetObjective(ctx, o.ID)
		if err != nil {
			return err
		}
		if ok {
			if strings.TrimSpace(expectedMeaningRevision) != existing.MeaningRevision {
				return &ValidationError{Err: ErrConflict, Field: "meaningRevision", Msg: fmt.Sprintf("expected %q but current is %q", expectedMeaningRevision, existing.MeaningRevision)}
			}
			o.GlobalOrder = existing.GlobalOrder
		} else {
			if strings.TrimSpace(expectedMeaningRevision) != "" {
				return &ValidationError{Err: ErrNotFound, Field: "id", Msg: fmt.Sprintf("%q does not exist", o.ID)}
			}
			objs, err := tx.ListObjectives(ctx)
			if err != nil {
				return err
			}
			maxOrder := -1
			for _, e := range objs {
				if e.GlobalOrder > maxOrder {
					maxOrder = e.GlobalOrder
				}
			}
			o.GlobalOrder = maxOrder + 1
		}
		if err := tx.PutObjective(ctx, o); err != nil {
			return err
		}
		result = o
		return nil
	})
	if err != nil {
		return Objective{}, err
	}
	return result, nil
}

// DeleteObjective removes an objective. An objective with active team
// attachments or relation endpoints is refused until those references are
// removed.
func (s *Service) DeleteObjective(ctx context.Context, id, expectedMeaningRevision string) error {
	id = strings.TrimSpace(id)
	return s.repo.WithTx(ctx, func(tx Repository) error {
		o, ok, err := tx.GetObjective(ctx, id)
		if err != nil {
			return err
		}
		if !ok {
			return &ValidationError{Err: ErrNotFound, Field: "id", Msg: fmt.Sprintf("%q does not exist", id)}
		}
		if strings.TrimSpace(expectedMeaningRevision) != o.MeaningRevision {
			return &ValidationError{Err: ErrConflict, Field: "meaningRevision", Msg: fmt.Sprintf("expected %q but current is %q", expectedMeaningRevision, o.MeaningRevision)}
		}
		atts, err := tx.ListAttachments(ctx, "")
		if err != nil {
			return err
		}
		for _, a := range atts {
			if a.ObjectiveID == id {
				return &ValidationError{Err: ErrActiveReferences, Field: "id", Msg: fmt.Sprintf("%q is attached to team %q", id, a.TeamID)}
			}
		}
		rels, err := tx.ListRelations(ctx)
		if err != nil {
			return err
		}
		for _, rel := range rels {
			if rel.FromObjectiveID == id || rel.ToObjectiveID == id {
				return &ValidationError{Err: ErrActiveReferences, Field: "id", Msg: fmt.Sprintf("%q is a relation endpoint", id)}
			}
		}
		return tx.DeleteObjective(ctx, id)
	})
}

// ReorderObjectives rewrites global priority. The supplied id set must match
// the persisted objective set exactly.
func (s *Service) ReorderObjectives(ctx context.Context, orderedIDs []string) ([]Objective, error) {
	var result []Objective
	err := s.repo.WithTx(ctx, func(tx Repository) error {
		objs, err := tx.ListObjectives(ctx)
		if err != nil {
			return err
		}
		index := map[string]Objective{}
		for _, o := range objs {
			index[o.ID] = o
		}
		if len(orderedIDs) != len(objs) {
			return &ValidationError{Err: ErrOrderMismatch, Field: "objectiveIds", Msg: fmt.Sprintf("expected %d ids, got %d", len(objs), len(orderedIDs))}
		}
		seen := map[string]bool{}
		for i, raw := range orderedIDs {
			id := strings.TrimSpace(raw)
			o, ok := index[id]
			if !ok || seen[id] {
				return &ValidationError{Err: ErrOrderMismatch, Field: "objectiveIds", Msg: fmt.Sprintf("%q is missing or repeated", id)}
			}
			seen[id] = true
			o.GlobalOrder = i
			if err := tx.PutObjective(ctx, o); err != nil {
				return err
			}
		}
		result, err = tx.ListObjectives(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Attach links a team to an objective. A duplicate (objective, team) pair is
// refused; use UpdateAttachment to edit an existing link.
func (s *Service) Attach(ctx context.Context, a Attachment, expectedTeamRevision string) (Attachment, error) {
	normalizeAttachment(&a)
	if err := s.validateAttachmentInput(a); err != nil {
		return Attachment{}, err
	}
	var result Attachment
	err := s.repo.WithTx(ctx, func(tx Repository) error {
		obj, ok, err := tx.GetObjective(ctx, a.ObjectiveID)
		if err != nil {
			return err
		}
		if !ok {
			return &ValidationError{Err: ErrUnknownObjective, Field: "objectiveId", Msg: fmt.Sprintf("%q does not exist", a.ObjectiveID)}
		}
		_, exists, err := tx.GetAttachment(ctx, a.ObjectiveID, a.TeamID)
		if err != nil {
			return err
		}
		if exists {
			return &ValidationError{Err: ErrDuplicateLink, Field: "objectiveId", Msg: fmt.Sprintf("team %q already has %q", a.TeamID, a.ObjectiveID)}
		}
		if err := s.checkTeamRevision(ctx, tx, a.TeamID, expectedTeamRevision); err != nil {
			return err
		}
		teamAtts, err := tx.ListAttachments(ctx, a.TeamID)
		if err != nil {
			return err
		}
		a.Priority = len(teamAtts)
		if err := tx.PutAttachment(ctx, a); err != nil {
			return err
		}
		rev, err := s.persistTeamRevision(ctx, tx, a.TeamID)
		if err != nil {
			return err
		}
		result = decorate(a, obj, rev)
		return nil
	})
	if err != nil {
		return Attachment{}, err
	}
	return result, nil
}

// UpdateAttachment edits role, coverage or note on an existing link. Priority
// and acknowledgement are preserved.
func (s *Service) UpdateAttachment(ctx context.Context, a Attachment, expectedTeamRevision string) (Attachment, error) {
	normalizeAttachment(&a)
	if err := s.validateAttachmentInput(a); err != nil {
		return Attachment{}, err
	}
	var result Attachment
	err := s.repo.WithTx(ctx, func(tx Repository) error {
		obj, ok, err := tx.GetObjective(ctx, a.ObjectiveID)
		if err != nil {
			return err
		}
		if !ok {
			return &ValidationError{Err: ErrUnknownObjective, Field: "objectiveId", Msg: fmt.Sprintf("%q does not exist", a.ObjectiveID)}
		}
		existing, ok, err := tx.GetAttachment(ctx, a.ObjectiveID, a.TeamID)
		if err != nil {
			return err
		}
		if !ok {
			return &ValidationError{Err: ErrNotFound, Field: "objectiveId", Msg: fmt.Sprintf("team %q is not attached to %q", a.TeamID, a.ObjectiveID)}
		}
		if err := s.checkTeamRevision(ctx, tx, a.TeamID, expectedTeamRevision); err != nil {
			return err
		}
		a.Priority = existing.Priority
		a.AcknowledgedRevision = existing.AcknowledgedRevision
		if err := tx.PutAttachment(ctx, a); err != nil {
			return err
		}
		rev, err := s.persistTeamRevision(ctx, tx, a.TeamID)
		if err != nil {
			return err
		}
		result = decorate(a, obj, rev)
		return nil
	})
	if err != nil {
		return Attachment{}, err
	}
	return result, nil
}

// Detach removes a team's link to an objective and closes the priority gap.
func (s *Service) Detach(ctx context.Context, objectiveID, teamID, expectedTeamRevision string) error {
	objectiveID = strings.TrimSpace(objectiveID)
	teamID = strings.TrimSpace(teamID)
	return s.repo.WithTx(ctx, func(tx Repository) error {
		if err := s.checkTeamRevision(ctx, tx, teamID, expectedTeamRevision); err != nil {
			return err
		}
		_, ok, err := tx.GetAttachment(ctx, objectiveID, teamID)
		if err != nil {
			return err
		}
		if !ok {
			return &ValidationError{Err: ErrNotFound, Field: "objectiveId", Msg: fmt.Sprintf("team %q is not attached to %q", teamID, objectiveID)}
		}
		if err := tx.DeleteAttachment(ctx, objectiveID, teamID); err != nil {
			return err
		}
		remaining, err := tx.ListAttachments(ctx, teamID)
		if err != nil {
			return err
		}
		if err := tx.ReplaceAttachmentOrder(ctx, teamID, orderedObjectiveIDs(remaining)); err != nil {
			return err
		}
		_, err = s.persistTeamRevision(ctx, tx, teamID)
		return err
	})
}

// ReorderTeamAttachments rewrites a team's objective priority. Reordering does
// not change objective meaning or acknowledgement state.
func (s *Service) ReorderTeamAttachments(ctx context.Context, teamID string, orderedObjectiveIDs []string, expectedTeamRevision string) ([]Attachment, error) {
	teamID = strings.TrimSpace(teamID)
	var result []Attachment
	err := s.repo.WithTx(ctx, func(tx Repository) error {
		if err := s.checkTeamRevision(ctx, tx, teamID, expectedTeamRevision); err != nil {
			return err
		}
		current, err := tx.ListAttachments(ctx, teamID)
		if err != nil {
			return err
		}
		if len(orderedObjectiveIDs) != len(current) {
			return &ValidationError{Err: ErrOrderMismatch, Field: "objectiveIds", Msg: fmt.Sprintf("expected %d ids, got %d", len(current), len(orderedObjectiveIDs))}
		}
		present := map[string]bool{}
		for _, c := range current {
			present[c.ObjectiveID] = true
		}
		seen := map[string]bool{}
		normalized := make([]string, 0, len(orderedObjectiveIDs))
		for _, raw := range orderedObjectiveIDs {
			id := strings.TrimSpace(raw)
			if !present[id] || seen[id] {
				return &ValidationError{Err: ErrOrderMismatch, Field: "objectiveIds", Msg: fmt.Sprintf("%q is missing or repeated", id)}
			}
			seen[id] = true
			normalized = append(normalized, id)
		}
		if err := tx.ReplaceAttachmentOrder(ctx, teamID, normalized); err != nil {
			return err
		}
		rev, err := s.persistTeamRevision(ctx, tx, teamID)
		if err != nil {
			return err
		}
		atts, err := tx.ListAttachments(ctx, teamID)
		if err != nil {
			return err
		}
		result, err = s.decorateTeamAttachments(ctx, tx, atts, rev)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ListTeamAttachments returns a team's attachments in priority order with
// current meaning and restatement state.
func (s *Service) ListTeamAttachments(ctx context.Context, teamID string) ([]Attachment, error) {
	teamID = strings.TrimSpace(teamID)
	rev, _, err := s.repo.GetTeamAttachmentRevision(ctx, teamID)
	if err != nil {
		return nil, err
	}
	atts, err := s.repo.ListAttachments(ctx, teamID)
	if err != nil {
		return nil, err
	}
	return s.decorateTeamAttachments(ctx, s.repo, atts, rev)
}

// TeamAttachmentRevision returns a team's current links/order revision.
func (s *Service) TeamAttachmentRevision(ctx context.Context, teamID string) (string, error) {
	rev, _, err := s.repo.GetTeamAttachmentRevision(ctx, strings.TrimSpace(teamID))
	return rev, err
}

// Acknowledge records that a team has confirmed its obligations against an
// objective's current meaning revision. A stale revision is refused.
func (s *Service) Acknowledge(ctx context.Context, objectiveID, teamID, revision string) (Attachment, error) {
	objectiveID = strings.TrimSpace(objectiveID)
	teamID = strings.TrimSpace(teamID)
	revision = strings.TrimSpace(revision)
	obj, ok, err := s.repo.GetObjective(ctx, objectiveID)
	if err != nil {
		return Attachment{}, err
	}
	if !ok {
		return Attachment{}, &ValidationError{Err: ErrUnknownObjective, Field: "objectiveId", Msg: fmt.Sprintf("%q does not exist", objectiveID)}
	}
	if revision != obj.MeaningRevision {
		return Attachment{}, &ValidationError{Err: ErrConflict, Field: "acknowledgedRevision", Msg: fmt.Sprintf("expected %q but current is %q", revision, obj.MeaningRevision)}
	}
	a, ok, err := s.repo.GetAttachment(ctx, objectiveID, teamID)
	if err != nil {
		return Attachment{}, err
	}
	if !ok {
		return Attachment{}, &ValidationError{Err: ErrNotFound, Field: "objectiveId", Msg: fmt.Sprintf("team %q is not attached to %q", teamID, objectiveID)}
	}
	a.AcknowledgedRevision = revision
	if err := s.repo.PutAttachment(ctx, a); err != nil {
		return Attachment{}, err
	}
	rev, _, err := s.repo.GetTeamAttachmentRevision(ctx, teamID)
	if err != nil {
		return Attachment{}, err
	}
	return decorate(a, obj, rev), nil
}

// ListRelations returns every objective support edge.
func (s *Service) ListRelations(ctx context.Context) ([]Relation, error) {
	return s.repo.ListRelations(ctx)
}

// AddRelation creates an instrumental support edge. Unknown endpoints, a
// non-instrumental source and any edge that would close a cycle are refused.
func (s *Service) AddRelation(ctx context.Context, from, to string) error {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if from == "" || to == "" {
		return &ValidationError{Err: ErrUnknownObjective, Msg: "both relation endpoints are required"}
	}
	if from == to {
		return &ValidationError{Err: ErrCycle, Field: "relation", Msg: "an objective cannot support itself"}
	}
	return s.repo.WithTx(ctx, func(tx Repository) error {
		source, ok, err := tx.GetObjective(ctx, from)
		if err != nil {
			return err
		}
		if !ok {
			return &ValidationError{Err: ErrUnknownObjective, Field: "fromObjectiveId", Msg: fmt.Sprintf("%q does not exist", from)}
		}
		if _, ok, err := tx.GetObjective(ctx, to); err != nil {
			return err
		} else if !ok {
			return &ValidationError{Err: ErrUnknownObjective, Field: "toObjectiveId", Msg: fmt.Sprintf("%q does not exist", to)}
		}
		if source.Class != ClassInstrumental {
			return &ValidationError{Err: ErrInvalidClass, Field: "fromObjectiveId", Msg: fmt.Sprintf("%q is %s; only instrumental objectives support others", from, source.Class)}
		}
		rels, err := tx.ListRelations(ctx)
		if err != nil {
			return err
		}
		if pathExists(to, from, rels) {
			return &ValidationError{Err: ErrCycle, Field: "relation", Msg: fmt.Sprintf("%s -> %s would close a cycle", from, to)}
		}
		return tx.PutRelation(ctx, Relation{FromObjectiveID: from, ToObjectiveID: to})
	})
}

// DeleteRelation removes a support edge.
func (s *Service) DeleteRelation(ctx context.Context, from, to string) error {
	return s.repo.DeleteRelation(ctx, strings.TrimSpace(from), strings.TrimSpace(to))
}

// Validate runs the domain rules over current state. It is the read-side
// counterpart to the mutation guards and lets every client surface the same
// findings.
func (s *Service) Validate(ctx context.Context) (ValidationResult, error) {
	objs, err := s.repo.ListObjectives(ctx)
	if err != nil {
		return ValidationResult{}, err
	}
	atts, err := s.repo.ListAttachments(ctx, "")
	if err != nil {
		return ValidationResult{}, err
	}
	rels, err := s.repo.ListRelations(ctx)
	if err != nil {
		return ValidationResult{}, err
	}

	result := ValidationResult{Findings: []Finding{}}
	byID := map[string]Objective{}
	for _, o := range objs {
		byID[o.ID] = o
	}
	attached := map[string]bool{}
	for _, a := range atts {
		attached[a.ObjectiveID] = true
		if !validRole(a.Role) {
			result.add("objective_role_invalid", "error", a.ObjectiveID, a.TeamID, fmt.Sprintf("role %q is invalid", a.Role))
		}
		if !validCoverage(a.Coverage) {
			result.add("objective_role_invalid", "error", a.ObjectiveID, a.TeamID, fmt.Sprintf("coverage %q is invalid", a.Coverage))
		}
		obj, ok := byID[a.ObjectiveID]
		if !ok {
			result.add("objective_unknown_id", "error", a.ObjectiveID, a.TeamID, "attachment references an undefined objective")
			continue
		}
		if a.AcknowledgedRevision != obj.MeaningRevision {
			result.add("objective_restatement_pending", "warning", a.ObjectiveID, a.TeamID, "team has not confirmed the current meaning revision")
		}
	}
	for _, o := range objs {
		if !attached[o.ID] && strings.TrimSpace(o.GapMarker) == "" {
			result.add("objective_unserved", "warning", o.ID, "", "no team serves this objective and no gap marker is declared")
		}
		if !o.HasEvidence {
			result.add("objective_unmeasurable", "warning", o.ID, "", "no evidence source is declared")
		}
	}
	for _, rel := range rels {
		if _, ok := byID[rel.FromObjectiveID]; !ok {
			result.add("objective_unknown_id", "error", rel.FromObjectiveID, "", "relation source is undefined")
		}
		if _, ok := byID[rel.ToObjectiveID]; !ok {
			result.add("objective_unknown_id", "error", rel.ToObjectiveID, "", "relation target is undefined")
		}
	}
	if cycle := firstCycleNode(objs, rels); cycle != "" {
		result.add("objective_relation_cycle", "error", cycle, "", "objective support edges contain a cycle")
	}
	sort.SliceStable(result.Findings, func(i, j int) bool {
		if result.Findings[i].Severity != result.Findings[j].Severity {
			return result.Findings[i].Severity == "error"
		}
		if result.Findings[i].Rule != result.Findings[j].Rule {
			return result.Findings[i].Rule < result.Findings[j].Rule
		}
		if result.Findings[i].ObjectiveID != result.Findings[j].ObjectiveID {
			return result.Findings[i].ObjectiveID < result.Findings[j].ObjectiveID
		}
		return result.Findings[i].TeamID < result.Findings[j].TeamID
	})
	return result, nil
}

func (r *ValidationResult) add(rule, severity, objectiveID, teamID, detail string) {
	r.Findings = append(r.Findings, Finding{Rule: rule, Severity: severity, ObjectiveID: objectiveID, TeamID: teamID, Detail: detail})
	if severity == "error" {
		r.Errors++
	} else {
		r.Warnings++
	}
}

func (s *Service) validateAttachmentInput(a Attachment) error {
	if a.TeamID == "" {
		return &ValidationError{Err: ErrUnknownTeam, Field: "teamId", Msg: "is required"}
	}
	if a.ObjectiveID == "" {
		return &ValidationError{Err: ErrUnknownObjective, Field: "objectiveId", Msg: "is required"}
	}
	if s.teamKnown != nil && !s.teamKnown(a.TeamID) {
		return &ValidationError{Err: ErrUnknownTeam, Field: "teamId", Msg: fmt.Sprintf("%q is not a known team", a.TeamID)}
	}
	if !validRole(a.Role) {
		return &ValidationError{Err: ErrInvalidRole, Field: "role", Msg: fmt.Sprintf("%q is not primary, supporting or empty", a.Role)}
	}
	if !validCoverage(a.Coverage) {
		return &ValidationError{Err: ErrInvalidCoverage, Field: "coverage", Msg: fmt.Sprintf("%q is not full, partial or empty", a.Coverage)}
	}
	return nil
}

func (s *Service) checkTeamRevision(ctx context.Context, repo Repository, teamID, expected string) error {
	current, ok, err := repo.GetTeamAttachmentRevision(ctx, teamID)
	if err != nil {
		return err
	}
	if !ok {
		current = ""
	}
	if strings.TrimSpace(expected) != current {
		return &ValidationError{Err: ErrConflict, Field: "attachmentRevision", Msg: fmt.Sprintf("expected %q but current is %q", expected, current)}
	}
	return nil
}

func (s *Service) persistTeamRevision(ctx context.Context, repo Repository, teamID string) (string, error) {
	atts, err := repo.ListAttachments(ctx, teamID)
	if err != nil {
		return "", err
	}
	revision := ComputeAttachmentRevision(atts)
	if err := repo.PutTeamAttachmentRevision(ctx, teamID, revision); err != nil {
		return "", err
	}
	return revision, nil
}

func (s *Service) decorateTeamAttachments(ctx context.Context, repo Repository, atts []Attachment, teamRevision string) ([]Attachment, error) {
	out := make([]Attachment, 0, len(atts))
	for _, a := range atts {
		obj, ok, err := repo.GetObjective(ctx, a.ObjectiveID)
		if err != nil {
			return nil, err
		}
		if !ok {
			a.AttachmentRevision = teamRevision
			out = append(out, a)
			continue
		}
		out = append(out, decorate(a, obj, teamRevision))
	}
	return out, nil
}

func decorate(a Attachment, o Objective, teamRevision string) Attachment {
	a.AttachmentRevision = teamRevision
	a.RestatementPending = strings.TrimSpace(o.MeaningRevision) != "" && a.AcknowledgedRevision != o.MeaningRevision
	return a
}

func normalizeAttachment(a *Attachment) {
	a.ObjectiveID = strings.TrimSpace(a.ObjectiveID)
	a.TeamID = strings.TrimSpace(a.TeamID)
	a.Note = strings.TrimSpace(a.Note)
	a.Role = strings.ToLower(strings.TrimSpace(a.Role))
	a.Coverage = strings.ToLower(strings.TrimSpace(a.Coverage))
}

func validClass(c Class) bool {
	return c == ClassTerminal || c == ClassInstrumental
}

func validRole(r string) bool {
	return r == "" || r == RolePrimary || r == RoleSupporting
}

func validCoverage(c string) bool {
	return c == "" || c == CoverageFull || c == CoveragePartial
}

func orderedObjectiveIDs(atts []Attachment) []string {
	ordered := append([]Attachment(nil), atts...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Priority != ordered[j].Priority {
			return ordered[i].Priority < ordered[j].Priority
		}
		return ordered[i].ObjectiveID < ordered[j].ObjectiveID
	})
	out := make([]string, 0, len(ordered))
	for _, a := range ordered {
		out = append(out, a.ObjectiveID)
	}
	return out
}

func pathExists(from, target string, rels []Relation) bool {
	adjacency := map[string][]string{}
	for _, r := range rels {
		adjacency[r.FromObjectiveID] = append(adjacency[r.FromObjectiveID], r.ToObjectiveID)
	}
	stack := []string{from}
	visited := map[string]bool{}
	for len(stack) > 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if node == target {
			return true
		}
		if visited[node] {
			continue
		}
		visited[node] = true
		stack = append(stack, adjacency[node]...)
	}
	return false
}

func firstCycleNode(objs []Objective, rels []Relation) string {
	adjacency := map[string][]string{}
	for _, r := range rels {
		adjacency[r.FromObjectiveID] = append(adjacency[r.FromObjectiveID], r.ToObjectiveID)
	}
	const (
		white = 0
		gray  = 1
		black = 2
	)
	state := map[string]int{}
	var visit func(string) string
	visit = func(node string) string {
		state[node] = gray
		for _, next := range adjacency[node] {
			switch state[next] {
			case gray:
				return next
			case white:
				if found := visit(next); found != "" {
					return found
				}
			}
		}
		state[node] = black
		return ""
	}
	for _, o := range objs {
		if state[o.ID] == white {
			if found := visit(o.ID); found != "" {
				return found
			}
		}
	}
	return ""
}
