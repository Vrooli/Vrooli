package importance

import "time"

type Config struct {
	CentralityWeight    float64 `json:"centrality_weight"`
	CoreProximityWeight float64 `json:"core_proximity_weight"`
	RecencyWeight       float64 `json:"recency_weight"`
	SystemRequiredFloor float64 `json:"system_required_floor"`
	NeutralScore        float64 `json:"neutral_score"`
	CacheTTL            time.Duration
}

func DefaultConfig() Config {
	return Config{
		CentralityWeight:    0.45,
		CoreProximityWeight: 0.30,
		RecencyWeight:       0.25,
		SystemRequiredFloor: 0.90,
		NeutralScore:        0.50,
		CacheTTL:            5 * time.Minute,
	}
}

type ScenarioFact struct {
	Name           string `json:"name"`
	SystemRequired bool   `json:"system_required"`
}

type CentralityMetric struct {
	Scenario                         string  `json:"scenario"`
	DirectReverseDependencyCount     int     `json:"direct_reverse_dependency_count"`
	TransitiveReverseDependencyCount int     `json:"transitive_reverse_dependency_count"`
	RequiredReverseDependencyCount   int     `json:"required_reverse_dependency_count"`
	RequiredEdgeWeightedScore        float64 `json:"required_edge_weighted_score"`
	DistanceToCoreSeed               int     `json:"distance_to_core_seed"`
	NearestCoreSeed                  string  `json:"nearest_core_seed,omitempty"`
}

type ComponentScores struct {
	Centrality    float64 `json:"centrality"`
	CoreProximity float64 `json:"core_proximity"`
	Recency       float64 `json:"recency"`
}

type Score struct {
	Scenario       string          `json:"scenario"`
	Score          float64         `json:"score"`
	SystemRequired bool            `json:"system_required"`
	Components     ComponentScores `json:"components"`
	Signals        ScoreSignals    `json:"signals"`
	Degraded       []string        `json:"degraded,omitempty"`
}

type ScoreSignals struct {
	DirectReverseDependencyCount     int     `json:"direct_reverse_dependency_count"`
	TransitiveReverseDependencyCount int     `json:"transitive_reverse_dependency_count"`
	RequiredReverseDependencyCount   int     `json:"required_reverse_dependency_count"`
	RequiredEdgeWeightedScore        float64 `json:"required_edge_weighted_score"`
	DistanceToCoreSeed               int     `json:"distance_to_core_seed"`
	NearestCoreSeed                  string  `json:"nearest_core_seed,omitempty"`
	RecentActivityCount              int     `json:"recent_activity_count"`
}

type Report struct {
	GeneratedAt time.Time `json:"generated_at"`
	Scores      []Score   `json:"scores"`
	Degraded    []string  `json:"degraded,omitempty"`
	Config      Config    `json:"config"`
}
