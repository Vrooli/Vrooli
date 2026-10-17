package objectives

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"prompt-manager/internal/memberflow"
)

// This file owns the authority-backed coverage read for plan
// aquila-objective-authority-and-team-editor Phase 3 step 6 (decision
// 0b153c9b-3a07-451c-883d-31f367d20572). It replaces the previous reader that
// joined the OBJECTIVES.md parse against team.json::objectivesServed.
//
// The objective authority is the only owner of current objective state. The
// operator statement and each team.json `objectivesServed` block stay authored
// inputs: they are read here for declaration context and drift, exactly as
// CompareProjection does, and are never treated as a second writable authority.

// CoverageTeamRef is one team's relationship to an objective in the coverage
// read: its role and coverage qualifier from the authority (ServedBy) or from
// the retained declaration (DeclaredBy). AcknowledgedRevision and
// RestatementPending carry the authority's acknowledgement signal so a
// coverage reader resolves the same restatement state as the runtime contract
// instead of re-deriving it.
type CoverageTeamRef struct {
	TeamID               string `json:"teamId"`
	Role                 string `json:"role,omitempty"`
	Coverage             string `json:"coverage,omitempty"`
	AcknowledgedRevision string `json:"acknowledgedRevision,omitempty"`
	RestatementPending   bool   `json:"restatementPending,omitempty"`
}

// CoverageObjective is one objective row in the coverage read. Served is true
// when the authority has at least one team attachment for the objective.
// GlobalOrder and MeaningRevision are the authority's identity, ordering and
// revision facts, exposed so every consumer resolves the same objective
// ordering and meaning revision rather than re-deriving them.
type CoverageObjective struct {
	ID              string            `json:"id"`
	Title           string            `json:"title"`
	Class           string            `json:"class"`
	GlobalOrder     int               `json:"globalOrder"`
	MeaningRevision string            `json:"meaningRevision,omitempty"`
	ServedBy        []CoverageTeamRef `json:"servedBy,omitempty"`
	DeclaredBy      []CoverageTeamRef `json:"declaredBy,omitempty"`
	GapMarker       string            `json:"gapMarker,omitempty"`
	EvidenceSource  string            `json:"evidenceSource,omitempty"`
	HasEvidence     bool              `json:"hasEvidence"`
	Served          bool              `json:"served"`
}

// CoverageFinding mirrors the objective authority's validation finding in the
// shape the graph/CLI consumers already read. Team is the authority TeamID and
// ObjectiveID is carried separately so no consumer has to guess which side of
// the join a finding names.
type CoverageFinding struct {
	Rule        string `json:"rule"`
	Severity    string `json:"severity"`
	Team        string `json:"team,omitempty"`
	ObjectiveID string `json:"objectiveId,omitempty"`
	Detail      string `json:"detail"`
}

// CoverageValidation is the authority's validation tally.
type CoverageValidation struct {
	Findings []CoverageFinding `json:"findings"`
	Errors   int               `json:"errors"`
	Warnings int               `json:"warnings"`
}

// Coverage is the authority-backed replacement for the parser join. It keeps
// the legacy field names (sourcePath, rows, unattachedTeams, unserved,
// undeclaredHoles, validation) so existing CLI/audit consumers decode it
// unchanged, and adds `drift` so a reader can see any disagreement between the
// retained declarations and the authority instead of silently resolving it.
type Coverage struct {
	SourcePath string              `json:"sourcePath"`
	Rows       []CoverageObjective `json:"rows"`
	// Teams is the ordered runtime read, team-major: each tracked team's
	// attachments in priority order with objective identity, meaning revision
	// and restatement state. It is the same BuildTeamObjectiveContext contract
	// the runtime child consumes, surfaced here so the CLI and the runtime
	// resolve identical ordering and revisions.
	Teams           []TeamObjectiveContext `json:"teams,omitempty"`
	UnattachedTeams []string               `json:"unattachedTeams,omitempty"`
	Unserved        int                    `json:"unserved"`
	Undeclared      int                    `json:"undeclaredHoles"`
	Validation      CoverageValidation     `json:"validation"`
	Drift           ProjectionDrift        `json:"drift"`
}

// BuildCoverage renders coverage from the objective authority. repoRoot and
// configDir locate the retained operator declarations (OBJECTIVES.md and the
// store/teams directory) used only for declaration context and drift.
func BuildCoverage(ctx context.Context, svc *Service, repoRoot, configDir string) (Coverage, error) {
	projection, err := BuildProjection(ctx, svc)
	if err != nil {
		return Coverage{}, err
	}
	preview, err := PreviewImportFromRepo(repoRoot, configDir)
	if err != nil {
		return Coverage{}, err
	}
	// The declaration roster is needed to name teams that trace to no
	// objective at all; the preview only carries teams that declared one.
	declared, _, err := memberflow.LoadTeamObjectives(configDir)
	if err != nil {
		return Coverage{}, fmt.Errorf("objectives: coverage load team declarations: %w", err)
	}

	servedBy := map[string][]CoverageTeamRef{}
	teamsWithAttachment := map[string]bool{}
	for _, team := range projection.Teams {
		for _, a := range team.Attachments {
			key := normalizeKey(a.ObjectiveID)
			servedBy[key] = append(servedBy[key], CoverageTeamRef{
				TeamID:               a.TeamID,
				Role:                 a.Role,
				Coverage:             a.Coverage,
				AcknowledgedRevision: a.AcknowledgedRevision,
				RestatementPending:   a.RestatementPending,
			})
			teamsWithAttachment[a.TeamID] = true
		}
	}
	sortRefs(servedBy)

	declaredBy := map[string][]CoverageTeamRef{}
	for _, a := range preview.Attachments {
		key := normalizeKey(a.ObjectiveID)
		declaredBy[key] = append(declaredBy[key], CoverageTeamRef{
			TeamID:               a.TeamID,
			Role:                 a.Role,
			Coverage:             a.Coverage,
			AcknowledgedRevision: a.NewAcknowledgedRevision,
			RestatementPending:   a.RestatementPending,
		})
	}
	sortRefs(declaredBy)

	out := Coverage{
		SourcePath: memberflow.ObjectivesDocPath,
		Rows:       []CoverageObjective{},
		Drift:      CompareProjection(preview, projection),
	}
	for _, o := range projection.Objectives {
		key := normalizeKey(o.ID)
		row := CoverageObjective{
			ID:              o.ID,
			Title:           o.Title,
			Class:           o.Class,
			GlobalOrder:     o.GlobalOrder,
			MeaningRevision: o.MeaningRevision,
			ServedBy:        servedBy[key],
			DeclaredBy:      declaredBy[key],
			GapMarker:       o.GapMarker,
			EvidenceSource:  o.EvidenceSource,
			HasEvidence:     o.HasEvidence,
		}
		row.Served = len(row.ServedBy) > 0
		if !row.Served {
			out.Unserved++
			if strings.TrimSpace(o.GapMarker) == "" {
				out.Undeclared++
			}
		}
		out.Rows = append(out.Rows, row)
	}

	for teamID, decls := range declared {
		if len(decls) == 0 && !teamsWithAttachment[teamID] {
			out.UnattachedTeams = append(out.UnattachedTeams, teamID)
		}
	}
	sort.Strings(out.UnattachedTeams)

	out.Validation = CoverageValidation{Findings: []CoverageFinding{}}
	for _, f := range projection.Validation.Findings {
		out.Validation.Findings = append(out.Validation.Findings, CoverageFinding{
			Rule:        f.Rule,
			Severity:    f.Severity,
			Team:        f.TeamID,
			ObjectiveID: f.ObjectiveID,
			Detail:      f.Detail,
		})
	}
	out.Validation.Errors = projection.Validation.Errors
	out.Validation.Warnings = projection.Validation.Warnings
	out.Teams = BuildTeamObjectiveContexts(projection)
	return out, nil
}

func sortRefs(refs map[string][]CoverageTeamRef) {
	for key := range refs {
		list := refs[key]
		sort.SliceStable(list, func(i, j int) bool { return list[i].TeamID < list[j].TeamID })
	}
}
