// DOC: docs/concepts/GRAPH.md#health-scoring
package graph

import (
	"math"
	"time"
)

// ScoreFn computes a named scoring factor for a node within the graph context.
type ScoreFn struct {
	Name   string
	Weight float64
	Fn     func(nodeID string, g Graph) float64
}

// ScoreAll computes health scores for all nodes using the provided score functions.
func ScoreAll(g Graph, fns []ScoreFn) []HealthScore {
	scores := make([]HealthScore, 0, len(g.Nodes))
	for _, n := range g.Nodes {
		hs := HealthScore{
			NodeID:  n.ID,
			Factors: make(map[string]float64, len(fns)),
		}
		var total float64
		var weightSum float64
		for _, fn := range fns {
			val := fn.Fn(n.ID, g)
			hs.Factors[fn.Name] = val
			total += val * fn.Weight
			weightSum += fn.Weight
		}
		if weightSum > 0 {
			hs.Score = total / weightSum
		}
		scores = append(scores, hs)
	}
	return scores
}

// DefaultScoreFns returns the default set of scoring functions.
func DefaultScoreFns() []ScoreFn {
	return []ScoreFn{
		{Name: "outgoing-edges", Weight: 1.0, Fn: outgoingEdgesScore},
		{Name: "incoming-edges", Weight: 1.0, Fn: incomingEdgesScore},
		{Name: "code-usage", Weight: 0.5, Fn: codeUsageScore},
		{Name: "recent-activity", Weight: 0.5, Fn: recentActivityScore},
	}
}

// outgoingEdgesScore: higher is better, capped at 1.0.
func outgoingEdgesScore(nodeID string, g Graph) float64 {
	count := 0
	for _, e := range g.Edges {
		if e.From == nodeID {
			count++
		}
	}
	// Normalize: 5+ outgoing edges is "full score"
	return math.Min(float64(count)/5.0, 1.0)
}

// incomingEdgesScore: higher is better (node is well-referenced).
func incomingEdgesScore(nodeID string, g Graph) float64 {
	count := 0
	for _, e := range g.Edges {
		if e.To == nodeID {
			count++
		}
	}
	return math.Min(float64(count)/5.0, 1.0)
}

// codeUsageScore rewards Vrooli-only tool use and penalizes external tools.
//
//	1.0 — only Vrooli CLI usage (incentivized)
//	0.5 — no tool usage detected (neutral)
//	0.1 — has external tool or script usage (penalty)
func codeUsageScore(nodeID string, g Graph) float64 {
	hasVrooli := false
	hasExternal := false
	for _, e := range g.Edges {
		if e.From == nodeID && e.Kind == EdgeCodeUsage {
			switch e.Category {
			case CodeScenarioCLI:
				hasVrooli = true
			case CodeExternalTool, CodeScript:
				hasExternal = true
			}
		}
	}
	if hasExternal {
		return 0.1
	}
	if hasVrooli {
		return 1.0
	}
	return 0.5
}

// recentActivityScore: based on the node's updatedAt timestamp.
// Nodes updated within 7 days get 1.0, decaying over 90 days to 0.
func recentActivityScore(nodeID string, g Graph) float64 {
	for _, n := range g.Nodes {
		if n.ID != nodeID {
			continue
		}
		// Nodes don't carry timestamps directly in the graph; return 0.5 as neutral.
		return 0.5
	}
	return 0.0
}

// RecentActivityScoreFromTimestamp computes the recency score from a timestamp string.
func RecentActivityScoreFromTimestamp(updatedAt string) float64 {
	t, err := time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return 0.0
	}
	daysSince := time.Since(t).Hours() / 24.0
	if daysSince <= 7 {
		return 1.0
	}
	if daysSince >= 90 {
		return 0.0
	}
	// Linear decay from 1.0 at 7 days to 0.0 at 90 days
	return 1.0 - (daysSince-7)/(90-7)
}
