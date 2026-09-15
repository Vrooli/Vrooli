package support

import "time"

// InjectionTechnique mirrors the payload returned by /api/v1/injections/library.
type InjectionTechnique struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Category          string    `json:"category"`
	Description       string    `json:"description,omitempty"`
	ExamplePrompt     string    `json:"example_prompt,omitempty"`
	DifficultyScore   float64   `json:"difficulty_score,omitempty"`
	SuccessRate       float64   `json:"success_rate,omitempty"`
	SourceAttribution string    `json:"source_attribution,omitempty"`
	IsActive          bool      `json:"is_active,omitempty"`
	CreatedAt         time.Time `json:"created_at,omitempty"`
	UpdatedAt         time.Time `json:"updated_at,omitempty"`
	CreatedBy         string    `json:"created_by,omitempty"`
}

// InjectionLibraryResponse wraps /api/v1/injections/library.
type InjectionLibraryResponse struct {
	Techniques []InjectionTechnique `json:"techniques"`
	TotalCount int                  `json:"total_count"`
	Categories []string             `json:"categories"`
}

// AddInjectionResponse wraps the 201 payload from POST /api/v1/injections.
type AddInjectionResponse struct {
	ID        string             `json:"id"`
	Message   string             `json:"message,omitempty"`
	Technique InjectionTechnique `json:"technique"`
}

// SimilarInjectionResult is a single similar-technique hit.
type SimilarInjectionResult struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name,omitempty"`
	Category string                 `json:"category,omitempty"`
	Score    float64                `json:"score,omitempty"`
	Payload  map[string]interface{} `json:"payload,omitempty"`
}

// SimilarInjectionsResponse wraps /api/v1/injections/similar.
type SimilarInjectionsResponse struct {
	Query   string                   `json:"query"`
	Limit   int                      `json:"limit"`
	Results []SimilarInjectionResult `json:"results"`
}

// LeaderboardEntry mirrors /api/v1/leaderboards/* entries.
type LeaderboardEntry struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Score          float64                `json:"score"`
	TestsRun       int                    `json:"tests_run"`
	TestsPassed    int                    `json:"tests_passed"`
	PassPercentage float64                `json:"pass_percentage"`
	LastTested     time.Time              `json:"last_tested"`
	AdditionalInfo map[string]interface{} `json:"additional_info,omitempty"`
}

// LeaderboardResponse wraps /api/v1/leaderboards/{agents,injections}.
type LeaderboardResponse struct {
	Leaderboard  []LeaderboardEntry `json:"leaderboard"`
	TotalEntries int                `json:"total_entries"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

// TestAgentResponse wraps /api/v1/security/test-agent.
type TestAgentResponse struct {
	RobustnessScore float64                  `json:"robustness_score"`
	TestResults     []map[string]interface{} `json:"test_results"`
	Recommendations []string                 `json:"recommendations"`
	Summary         map[string]interface{}   `json:"summary"`
}

// StatisticsResponse wraps /api/v1/statistics.
type StatisticsResponse struct {
	Totals              map[string]int `json:"totals"`
	InjectionCategories map[string]int `json:"injection_categories"`
	AgentModels         map[string]int `json:"agent_models"`
	RecentActivity      map[string]int `json:"recent_activity"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

// VectorSearchResult is one hit from /api/v1/vector/search.
type VectorSearchResult struct {
	ID       string                 `json:"id"`
	Score    float64                `json:"score,omitempty"`
	Payload  map[string]interface{} `json:"payload,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Category string                 `json:"category,omitempty"`
}

// VectorSearchResponse wraps /api/v1/vector/search.
type VectorSearchResponse struct {
	Query   string               `json:"query"`
	Results []VectorSearchResult `json:"results"`
	Count   int                  `json:"count"`
}

// Tournament mirrors a row from /api/v1/tournaments.
type Tournament struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
}

// TournamentListResponse wraps /api/v1/tournaments.
type TournamentListResponse struct {
	Tournaments []Tournament `json:"tournaments"`
	Count       int          `json:"count"`
}

// TournamentResult is one row from /api/v1/tournaments/:id/results.
type TournamentResult struct {
	AgentID         string                 `json:"agent_id"`
	InjectionID     string                 `json:"injection_id"`
	Success         bool                   `json:"success"`
	ExecutionTimeMS int                    `json:"execution_time_ms"`
	Score           float64                `json:"score"`
	Details         map[string]interface{} `json:"details,omitempty"`
	TestedAt        time.Time              `json:"tested_at"`
}

// TournamentResultsResponse wraps /api/v1/tournaments/:id/results.
type TournamentResultsResponse struct {
	TournamentID string             `json:"tournament_id"`
	Name         string             `json:"name"`
	Status       string             `json:"status"`
	Results      []TournamentResult `json:"results"`
	Count        int                `json:"count"`
	CompletedAt  *time.Time         `json:"completed_at,omitempty"`
}

// ExportFormat is one entry from /api/v1/export/formats.
type ExportFormat struct {
	Format      string `json:"format"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ContentType string `json:"content_type"`
}

// ExportFormatsResponse wraps /api/v1/export/formats.
type ExportFormatsResponse struct {
	Formats []ExportFormat `json:"formats"`
	Default string         `json:"default"`
}

// AdminCleanupResponse wraps POST /api/v1/admin/cleanup-test-data.
type AdminCleanupResponse struct {
	Message               string `json:"message"`
	TestResultsDeleted    int64  `json:"test_results_deleted"`
	TestInjectionsDeleted int64  `json:"test_injections_deleted"`
}
