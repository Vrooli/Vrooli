package skillset

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Intent recall guards the failure that no declaration check can see: a scenario
// whose skill exists and parses, but whose description shares no vocabulary with
// how requests are phrased, so the skill never surfaces and agents say the
// capability does not exist. The check asks this scenario's own skill search the
// questions the scenario says it answers.

// IntentRecallDepth is how many skill results an agent reads: the top-5 contract.
const IntentRecallDepth = 5

// Recaller returns skill ids for a query, best first.
type Recaller func(ctx context.Context, query string, limit int) ([]string, error)

// IntentReport is the measured result of the intent-recall check.
type IntentReport struct {
	Cases  int
	Passed int
	Missed []string // missed queries
}

type intentFile struct {
	Scenario string `json:"scenario"`
	Cases    []struct {
		Query string `json:"query"`
		Owner string `json:"owner"`
	} `json:"cases"`
}

// intentQueries gathers the queries a scenario declares for itself: its own
// .vrooli/intents.json plus the shared Search Hub corpus files that name it.
func intentQueries(root, scenario string) []string {
	paths := []string{filepath.Join(root, ".vrooli", "intents.json")}
	shared, _ := filepath.Glob(filepath.Join(filepath.Dir(root), "search-hub", "intents", "*.json"))
	paths = append(paths, shared...)
	var queries []string
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var file intentFile
		if json.Unmarshal(raw, &file) != nil {
			continue
		}
		for _, c := range file.Cases {
			owner := c.Owner
			if owner == "" {
				owner = file.Scenario
			}
			if owner == scenario && strings.TrimSpace(c.Query) != "" {
				queries = append(queries, c.Query)
			}
		}
	}
	sort.Strings(queries)
	return queries
}

// ownSkillIDs lists the scenario's authored skill ids (skills/<dir>/SKILL.md).
func ownSkillIDs(root string) []string {
	matches, _ := filepath.Glob(filepath.Join(root, "skills", "*", "SKILL.md"))
	ids := make([]string, 0, len(matches))
	for _, match := range matches {
		ids = append(ids, filepath.Base(filepath.Dir(match)))
	}
	return ids
}

// CheckIntents measures recall for scenario and records advisory warnings:
//   - skill-set.intent_missing: the scenario ships a skill but declares no intent,
//     so nothing measures whether the skill can be found;
//   - skill-set.intent_recall: at least one declared intent does not surface the
//     scenario's own skill in the top IntentRecallDepth.
//
// Both are warnings: they never fail the phase, they make the defect visible.
// A nil recaller (search unavailable) records nothing rather than a false miss.
func CheckIntents(ctx context.Context, r *Result, root string, recall Recaller) {
	skills := ownSkillIDs(root)
	if len(skills) == 0 {
		return
	}
	queries := intentQueries(root, r.Scenario)
	if len(queries) == 0 {
		r.Warnings = append(r.Warnings, "skill-set.intent_missing:usage")
		return
	}
	if recall == nil {
		return
	}
	own := make(map[string]bool, len(skills))
	for _, id := range skills {
		own[id] = true
	}
	report := &IntentReport{}
	for _, query := range queries {
		ids, err := recall(ctx, query, IntentRecallDepth)
		if err != nil {
			return // an unavailable index is not evidence about the skill
		}
		report.Cases++
		hit := false
		for _, id := range ids {
			if own[id] || strings.HasPrefix(id, r.Scenario+"-") {
				hit = true
				break
			}
		}
		if hit {
			report.Passed++
		} else {
			report.Missed = append(report.Missed, query)
		}
	}
	r.Intents = report
	if len(report.Missed) > 0 {
		r.Warnings = append(r.Warnings, "skill-set.intent_recall:usage")
	}
}
