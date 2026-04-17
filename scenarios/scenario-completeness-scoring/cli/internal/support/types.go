package support

// ScoreListItem is a row in the `GET /api/v1/scores` response.
type ScoreListItem struct {
	Scenario       string  `json:"scenario"`
	Category       string  `json:"category"`
	Score          float64 `json:"score"`
	Classification string  `json:"classification"`
	Partial        bool    `json:"partial"`
}

// ScoreListResponse is the envelope for `GET /api/v1/scores`.
type ScoreListResponse struct {
	Scenarios []ScoreListItem `json:"scenarios"`
	Total     int             `json:"total"`
}

// Recommendation is a single improvement hint returned by
// GET /api/v1/recommendations/{scenario}. The server emits `priority`,
// `description`, `impact`; older callers that hand-rolled decoders against
// `message` will see an empty description, which is why the rendered list
// used to print " (impact N)" with no text.
type Recommendation struct {
	Priority    int     `json:"priority"`
	Description string  `json:"description"`
	Impact      float64 `json:"impact"`
}

// RecommendationsResponse is the envelope for `GET /api/v1/recommendations/{scenario}`.
type RecommendationsResponse struct {
	Scenario        string           `json:"scenario"`
	CurrentScore    interface{}      `json:"current_score"`
	TotalImpact     interface{}      `json:"total_impact"`
	PotentialScore  interface{}      `json:"potential_score"`
	Recommendations []Recommendation `json:"recommendations"`
}

// ScoreSummary captures the high-level fields of a scenario score. The full
// breakdown is intentionally left as a raw map so we do not drift from the API.
type ScoreSummary struct {
	Scenario          string                 `json:"scenario"`
	Category          string                 `json:"category"`
	Score             float64                `json:"score"`
	BaseScore         float64                `json:"base_score"`
	ValidationPenalty float64                `json:"validation_penalty"`
	Classification    string                 `json:"classification"`
	CalculatedAt      string                 `json:"calculated_at"`
	Breakdown         map[string]interface{} `json:"breakdown"`
	Metrics           map[string]interface{} `json:"metrics"`
}

// CollectorHealthResponse is the shape of `GET /api/v1/health/collectors`.
type CollectorHealthResponse struct {
	Status     string                 `json:"status"`
	Collectors map[string]interface{} `json:"collectors"`
	Summary    map[string]int         `json:"summary"`
}
