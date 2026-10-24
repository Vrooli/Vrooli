package sketch

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type CandidateRefinement struct {
	Round               int      `json:"round"`
	Budget              int      `json:"budget"`
	Reason              string   `json:"reason"`
	RequestedRegions    []string `json:"requestedRegions"`
	ChangedRegions      []string `json:"changedRegions"`
	BroaderChangeReason string   `json:"broaderChangeReason,omitempty"`
}
type RefinementRequest struct {
	Document                    Document
	RequestedRegions            []string
	Reason, BroaderChangeReason string
	AdditionalRounds            int
}
type RefinementResult struct {
	Candidate     Candidate
	Status        string
	Round, Budget int
}

func validateRefinement(r CandidateRefinement) error {
	if r.Round < 1 || r.Budget < r.Round || r.Budget > 1000 || strings.TrimSpace(r.Reason) == "" || len(r.Reason) > 4000 || len(r.BroaderChangeReason) > 4000 || len(r.RequestedRegions) == 0 || len(r.RequestedRegions) > 64 || len(r.ChangedRegions) == 0 || len(r.ChangedRegions) > 256 {
		return fmt.Errorf("invalid candidate refinement record")
	}
	return nil
}
func refinementRegions(d Document) map[string]bool {
	result := map[string]bool{}
	for _, r := range d.Regions {
		result[r.ID] = true
	}
	for _, p := range d.Placements {
		result[p.Region] = true
	}
	if d.Render != nil {
		for _, r := range d.Render.Regions {
			result[r.ID] = true
		}
	}
	return result
}
func refinementContent(d Document, id string) string {
	if id == "$page" {
		value := map[string]any{"template": d.Template, "viewport": d.Viewport, "intent": d.Intent, "unplaced": d.Unplaced}
		known := refinementRegions(d)
		var notes []Note
		for _, n := range d.Notes {
			if !known[n.Scope] {
				notes = append(notes, n)
			}
		}
		value["notes"] = notes
		if d.Render != nil {
			value["hasRender"] = true
			var fixtures []RenderFixture
			for _, f := range d.Render.Fixtures {
				if f.Target == "$template" {
					fixtures = append(fixtures, f)
				}
			}
			value["templateFixtures"] = fixtures
			reserved := map[string]any{}
			for key, value := range d.Render.Bindings {
				if strings.HasPrefix(key, "$") {
					reserved[key] = value
				}
			}
			value["reservedBindings"] = reserved
			value["templateExport"] = d.Render.TemplateExport
			value["templateBindings"] = d.Render.Bindings["$template"]
			value["labels"] = d.Render.Bindings["$labels"]
			value["interactions"] = d.Render.Bindings["$preview"]
		}
		raw, _ := json.Marshal(value)
		return string(raw)
	}
	locked := false
	for _, r := range d.Regions {
		if r.ID == id {
			locked = r.Locked
		}
	}
	d.Template = nil
	d.Viewport = ""
	d.Intent = nil
	if d.Render != nil {
		copy := *d.Render
		copy.TemplateExport = ""
		copy.Bindings = map[string]any{}
		for key, value := range d.Render.Bindings {
			if !strings.HasPrefix(key, "$") {
				copy.Bindings[key] = value
			}
		}
		copy.Fixtures = nil
		for _, f := range d.Render.Fixtures {
			if !strings.HasPrefix(f.Target, "$") {
				copy.Fixtures = append(copy.Fixtures, f)
			}
		}
		d.Render = &copy
	}
	raw, _ := json.Marshal(struct {
		Content json.RawMessage
		Locked  bool
	}{regionContent(d, id), locked})
	return string(raw)
}
func (s *Store) RefineCandidate(scenario, designID, parentHash string, request RefinementRequest) (RefinementResult, error) {
	parent, err := s.ReadCandidate(scenario, designID, parentHash)
	if err != nil {
		return RefinementResult{}, err
	}
	round, budget := 0, 3
	if parent.Refinement != nil {
		round = parent.Refinement.Round
		budget = parent.Refinement.Budget
	}
	result := RefinementResult{Candidate: parent, Round: round, Budget: budget}
	if request.AdditionalRounds < 0 || request.AdditionalRounds > 10 || budget+request.AdditionalRounds > 1000 {
		return result, fmt.Errorf("continuation permits one to ten additional rounds")
	}
	if request.AdditionalRounds > 0 {
		if round < budget {
			return result, fmt.Errorf("extend only an exhausted refinement budget")
		}
		budget += request.AdditionalRounds
	}
	if round >= budget {
		result.Status = "budget_exhausted"
		return result, nil
	}
	if strings.TrimSpace(request.Reason) == "" || len(request.Reason) > 4000 || len(request.BroaderChangeReason) > 4000 || len(request.RequestedRegions) == 0 || len(request.RequestedRegions) > 64 {
		return result, fmt.Errorf("bounded refinement reason and requested regions are required")
	}
	before, err := parent.Snapshot()
	if err != nil {
		return result, err
	}
	known := refinementRegions(before.Document)
	for id := range refinementRegions(request.Document) {
		known[id] = true
	}
	known["$page"] = true
	requested := map[string]bool{}
	for _, id := range request.RequestedRegions {
		if !known[id] || requested[id] {
			return result, fmt.Errorf("unknown or duplicate requested region %q", id)
		}
		requested[id] = true
	}
	changed := []string{}
	for id := range known {
		if refinementContent(before.Document, id) != refinementContent(request.Document, id) {
			changed = append(changed, id)
			if !requested[id] && strings.TrimSpace(request.BroaderChangeReason) == "" {
				return result, fmt.Errorf("region %s changed outside requested scope; explain the broader change", id)
			}
		}
	}
	sort.Strings(changed)
	if len(changed) == 0 {
		result.Status = "unchanged"
		return result, nil
	}
	requestedIDs := append([]string(nil), request.RequestedRegions...)
	sort.Strings(requestedIDs)
	record := &CandidateRefinement{Round: round + 1, Budget: budget, Reason: request.Reason, RequestedRegions: requestedIDs, ChangedRegions: changed, BroaderChangeReason: request.BroaderChangeReason}
	if err := validateRefinement(*record); err != nil {
		return result, err
	}
	candidate, err := s.deriveCandidate(scenario, designID, parentHash, request.Document, record)
	if err != nil {
		return result, err
	}
	return RefinementResult{Candidate: candidate, Status: "refined", Round: record.Round, Budget: budget}, nil
}
