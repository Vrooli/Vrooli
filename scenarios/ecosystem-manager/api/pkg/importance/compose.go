package importance

import (
	"math"
	"sort"
	"strings"
)

func Compose(facts []ScenarioFact, centrality []CentralityMetric, recency map[string]int, cfg Config, degraded []string) Report {
	if cfg == (Config{}) {
		cfg = DefaultConfig()
	}
	totalWeight := cfg.CentralityWeight + cfg.CoreProximityWeight + cfg.RecencyWeight
	if totalWeight <= 0 {
		cfg = DefaultConfig()
		totalWeight = cfg.CentralityWeight + cfg.CoreProximityWeight + cfg.RecencyWeight
	}

	centralityByScenario := map[string]CentralityMetric{}
	maxCentrality := 0.0
	for _, metric := range centrality {
		name := normalizeName(metric.Scenario)
		if name == "" {
			continue
		}
		centralityByScenario[name] = metric
		if metric.RequiredEdgeWeightedScore > maxCentrality {
			maxCentrality = metric.RequiredEdgeWeightedScore
		}
	}

	globalDegraded := normalizeDegraded(degraded)
	scores := make([]Score, 0, len(facts))
	for _, fact := range facts {
		name := normalizeName(fact.Name)
		if name == "" {
			continue
		}

		metric, hasCentrality := centralityByScenario[name]
		scoreDegraded := append([]string(nil), globalDegraded...)
		centralityScore := cfg.NeutralScore
		coreScore := cfg.NeutralScore
		if hasCentrality {
			centralityScore = normalizeCentrality(metric.RequiredEdgeWeightedScore, maxCentrality, cfg.NeutralScore)
			coreScore = normalizeCoreProximity(metric.DistanceToCoreSeed, cfg.NeutralScore)
		} else {
			scoreDegraded = appendMissing(scoreDegraded, "centrality_unavailable")
		}

		recentCount, hasRecency := recency[name]
		recencyScore := cfg.NeutralScore
		if hasRecency {
			recencyScore = normalizeRecency(recentCount)
		} else {
			scoreDegraded = appendMissing(scoreDegraded, "recency_unavailable")
		}

		combined := ((centralityScore * cfg.CentralityWeight) +
			(coreScore * cfg.CoreProximityWeight) +
			(recencyScore * cfg.RecencyWeight)) / totalWeight
		if fact.SystemRequired && combined < cfg.SystemRequiredFloor {
			combined = cfg.SystemRequiredFloor
		}

		scores = append(scores, Score{
			Scenario:       name,
			Score:          round4(clamp01(combined)),
			SystemRequired: fact.SystemRequired,
			Components: ComponentScores{
				Centrality:    round4(centralityScore),
				CoreProximity: round4(coreScore),
				Recency:       round4(recencyScore),
			},
			Signals: ScoreSignals{
				DirectReverseDependencyCount:     metric.DirectReverseDependencyCount,
				TransitiveReverseDependencyCount: metric.TransitiveReverseDependencyCount,
				RequiredReverseDependencyCount:   metric.RequiredReverseDependencyCount,
				RequiredEdgeWeightedScore:        metric.RequiredEdgeWeightedScore,
				DistanceToCoreSeed:               metric.DistanceToCoreSeed,
				NearestCoreSeed:                  metric.NearestCoreSeed,
				RecentActivityCount:              recentCount,
			},
			Degraded: scoreDegraded,
		})
	}

	sort.SliceStable(scores, func(i, j int) bool {
		if scores[i].Score == scores[j].Score {
			return scores[i].Scenario < scores[j].Scenario
		}
		return scores[i].Score > scores[j].Score
	})

	return Report{
		Scores:   scores,
		Degraded: globalDegraded,
		Config:   cfg,
	}
}

func normalizeCentrality(value, maxValue, neutral float64) float64 {
	if maxValue <= 0 {
		return neutral
	}
	return clamp01(value / maxValue)
}

func normalizeCoreProximity(distance int, neutral float64) float64 {
	if distance < 0 {
		return neutral
	}
	return 1.0 / float64(distance+1)
}

func normalizeRecency(count int) float64 {
	if count <= 0 {
		return 0
	}
	if count >= 5 {
		return 1
	}
	return float64(count) / 5.0
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func round4(value float64) float64 {
	return math.Round(value*10000) / 10000
}

func normalizeName(value string) string {
	return strings.TrimSpace(value)
}

func normalizeDegraded(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = appendMissing(out, value)
	}
	return out
}

func appendMissing(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
