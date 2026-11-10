package intents

import (
	"sort"
	"strings"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

// ownedBy reports whether a result is a leaf of the scenario. It mirrors the
// router's `--group <scenario>` scope (api/internal/routing/scenario_scope.go):
// the CLI is a separate module, so the rule is restated rather than imported,
// and both are pinned by tests against the same identifier shapes.
func ownedBy(hit *routingv1.SearchHit, scenario string) bool {
	if hit == nil || scenario == "" {
		return false
	}
	if hit.GetProviderGroup() == scenario {
		return true
	}
	if strings.Contains(hit.GetPath(), "/scenarios/"+scenario+"/") {
		return true
	}
	for _, name := range []string{hit.GetId(), hit.GetTitle()} {
		if name == scenario || strings.HasPrefix(name, scenario+".") || strings.HasPrefix(name, scenario+" ") {
			return true
		}
	}
	return hit.GetType() == "skill" && strings.HasPrefix(hit.GetId(), scenario+"-")
}

// orderedHits returns hits in the order an agent reads them: the reranked list
// when present, otherwise provider groups merged by descending score.
func orderedHits(resp *routingv1.QueryResponse) []*routingv1.SearchHit {
	if resp == nil {
		return nil
	}
	if len(resp.GetRanked()) > 0 {
		return resp.GetRanked()
	}
	var hits []*routingv1.SearchHit
	for _, group := range resp.GetGroups() {
		if group != nil {
			hits = append(hits, group.GetHits()...)
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].GetScore() > hits[j].GetScore() })
	return hits
}

// caseResult is the verdict for one case.
type caseResult struct {
	ID       string   `json:"id"`
	Scenario string   `json:"scenario"`
	Query    string   `json:"query"`
	Tags     []string `json:"tags,omitempty"`
	// Pass: a qualifying owned leaf is within the top-k.
	Pass bool `json:"pass"`
	// Rank is the 1-based rank of the first qualifying leaf (0 = absent).
	Rank int `json:"rank"`
	// AnyLeafRank is the rank of the first leaf of the scenario of any type
	// (0 = absent); it separates "scenario invisible" from "wrong leaf kind".
	AnyLeafRank int `json:"any_leaf_rank"`
	// Top lists the leaves shown, for diagnosis.
	Top []string `json:"top"`
	// Unavailable is set when the query itself failed; such cases are excluded
	// from rates and reported separately instead of counting as misses.
	Unavailable string `json:"unavailable,omitempty"`
}

// grade evaluates one case against the hits shown to the agent.
func grade(c loadedCase, hits []*routingv1.SearchHit, k int) caseResult {
	result := caseResult{ID: c.ID, Scenario: c.Scenario, Query: c.Query, Tags: c.Tags}
	if k > 0 && len(hits) > k {
		hits = hits[:k]
	}
	for index, hit := range hits {
		result.Top = append(result.Top, hit.GetType()+":"+hit.GetId())
		owned := ownedBy(hit, c.Scenario)
		if owned && result.AnyLeafRank == 0 {
			result.AnyLeafRank = index + 1
		}
		if result.Rank == 0 && qualifies(c, hit, owned) {
			result.Rank = index + 1
		}
	}
	result.Pass = result.Rank > 0
	return result
}

// qualifies decides whether a hit satisfies the case. A case that names exact
// leaf ids is satisfied by one of them whatever its name looks like (a skill id
// need not carry its scenario's name); otherwise the hit must be a leaf the
// scenario owns, of an accepted type.
func qualifies(c loadedCase, hit *routingv1.SearchHit, owned bool) bool {
	if len(c.ExpectTypes) > 0 && !contains(c.ExpectTypes, hit.GetType()) {
		return false
	}
	if len(c.ExpectIDs) > 0 {
		return contains(c.ExpectIDs, hit.GetId())
	}
	return owned
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

// scenarioRecall is one owner's rolled-up result.
type scenarioRecall struct {
	Scenario    string `json:"scenario"`
	Cases       int    `json:"cases"`
	Passed      int    `json:"passed"`
	AnyLeaf     int    `json:"any_leaf"`
	Unavailable int    `json:"unavailable"`
}

// rollUp aggregates per-case results by owning scenario, sorted by name.
func rollUp(results []caseResult) []scenarioRecall {
	byScenario := make(map[string]*scenarioRecall)
	for _, r := range results {
		row := byScenario[r.Scenario]
		if row == nil {
			row = &scenarioRecall{Scenario: r.Scenario}
			byScenario[r.Scenario] = row
		}
		if r.Unavailable != "" {
			row.Unavailable++
			continue
		}
		row.Cases++
		if r.Pass {
			row.Passed++
		}
		if r.AnyLeafRank > 0 {
			row.AnyLeaf++
		}
	}
	rows := make([]scenarioRecall, 0, len(byScenario))
	for _, row := range byScenario {
		rows = append(rows, *row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Scenario < rows[j].Scenario })
	return rows
}
