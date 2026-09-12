// Objective coverage service: loads all three surfaces and returns the joined
// view plus its findings.
//
// DOC: docs/director-swarm/strategy/OBJECTIVES.md § The coverage rule
package memberflow

import (
	"net/http"
	"sort"
	"strings"
)

// ObjectiveCoverageRow is one objective joined against the roster.
type ObjectiveCoverageRow struct {
	Objective
	// DeclaredBy names the teams whose team.json declares this objective.
	// It is derived from the roster, not from the table, so the two can be
	// compared rather than conflated.
	DeclaredBy []ObjectiveTeamRef `json:"declaredBy,omitempty"`
	// Served is true when at least one team both appears in the table and
	// declares the objective back. A one-sided link is not coverage.
	Served bool `json:"served"`
}

// ObjectiveCoverageResponse is the JSON shape for GET /objectives.
type ObjectiveCoverageResponse struct {
	SourcePath string                 `json:"sourcePath"`
	Rows       []ObjectiveCoverageRow `json:"rows"`
	// UnattachedTeams are teams that declare no objective at all — the
	// upward half of the coverage rule.
	UnattachedTeams []string `json:"unattachedTeams,omitempty"`
	// Unserved counts objectives no team serves, whether or not the hole is
	// declared. A declared hole still counts: it is reported every cycle
	// until it closes.
	Unserved   int                            `json:"unserved"`
	Undeclared int                            `json:"undeclaredHoles"`
	Validation OperatingGraphValidationResult `json:"validation"`
}

// ObjectiveCoverage loads every surface and returns the joined view.
func (s OperatingModelService) ObjectiveCoverage() (ObjectiveCoverageResponse, error) {
	resp := ObjectiveCoverageResponse{SourcePath: ObjectivesDocPath, Rows: []ObjectiveCoverageRow{}}

	registry, err := LoadObjectives(s.RepoRoot)
	if err != nil {
		return resp, err
	}
	declared, paths, err := LoadTeamObjectives(s.StoreDir)
	if err != nil {
		return resp, err
	}
	// Operating models supply the prose half. A failure to load them is not
	// fatal: the declaration join is the load-bearing check and must still
	// run when a model document is unparseable.
	models, _ := LoadOperatingModelDocuments(s.RepoRoot)

	resp.Validation = ValidateObjectives(ObjectiveValidationInput{
		Registry:        registry,
		Declared:        declared,
		TeamSourcePaths: paths,
		Models:          models,
	})

	byObjective := map[string][]ObjectiveTeamRef{}
	for teamID, decls := range declared {
		for _, d := range decls {
			id := strings.ToUpper(strings.TrimSpace(d.ID))
			byObjective[id] = append(byObjective[id], ObjectiveTeamRef{
				TeamID:   teamID,
				Role:     d.Role,
				Coverage: d.Coverage,
			})
		}
	}
	for id := range byObjective {
		sort.Slice(byObjective[id], func(i, j int) bool { return byObjective[id][i].TeamID < byObjective[id][j].TeamID })
	}

	for _, obj := range registry.Objectives {
		row := ObjectiveCoverageRow{Objective: obj, DeclaredBy: byObjective[strings.ToUpper(obj.ID)]}
		for _, ref := range obj.ServedBy {
			if declaresObjective(declared[ref.TeamID], obj.ID) {
				row.Served = true
				break
			}
		}
		if !row.Served {
			resp.Unserved++
			if obj.GapMarker == "" {
				resp.Undeclared++
			}
		}
		resp.Rows = append(resp.Rows, row)
	}

	for _, teamID := range sortedMapKeys(declared) {
		if len(declared[teamID]) == 0 {
			resp.UnattachedTeams = append(resp.UnattachedTeams, teamID)
		}
	}
	return resp, nil
}

// GetObjectives handles GET /objectives.
func (h *Handlers) GetObjectives(w http.ResponseWriter, r *http.Request) {
	resp, err := h.operatingModelService().ObjectiveCoverage()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
