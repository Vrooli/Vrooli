package support

import "encoding/json"

// Algorithm mirrors the shape returned by /api/v1/algorithms/search and
// /api/v1/algorithms/{id}.
type Algorithm struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	DisplayName      string   `json:"display_name"`
	Category         string   `json:"category"`
	Description      string   `json:"description"`
	ComplexityTime   string   `json:"complexity_time"`
	ComplexitySpace  string   `json:"complexity_space"`
	Difficulty       string   `json:"difficulty"`
	Tags             []string `json:"tags"`
	LanguageCount    int      `json:"language_count"`
	TestCaseCount    int      `json:"test_case_count"`
	HasValidatedImpl bool     `json:"has_validated_impl"`
}

// SearchResponse is the wrapper returned by /api/v1/algorithms/search.
type SearchResponse struct {
	Algorithms []Algorithm `json:"algorithms"`
	Total      int         `json:"total"`
}

// Implementation mirrors one entry in the implementations array.
type Implementation struct {
	ID               string  `json:"id"`
	AlgorithmID      string  `json:"algorithm_id"`
	Language         string  `json:"language"`
	Code             string  `json:"code"`
	Version          string  `json:"version"`
	IsPrimary        bool    `json:"is_primary"`
	Validated        bool    `json:"validated"`
	ValidationCount  int     `json:"validation_count"`
	PerformanceScore float64 `json:"performance_score"`
}

// ImplementationsResponse is returned by /api/v1/algorithms/{id}/implementations.
type ImplementationsResponse struct {
	Algorithm       Algorithm        `json:"algorithm"`
	Implementations []Implementation `json:"implementations"`
}

// TestResult is one entry in a validation response.
type TestResult struct {
	TestCaseID     string `json:"test_case_id"`
	Passed         bool   `json:"passed"`
	ExecutionTime  int    `json:"execution_time_ms"`
	ActualOutput   string `json:"actual_output"`
	ExpectedOutput string `json:"expected_output"`
	ErrorMessage   string `json:"error_message,omitempty"`
}

// ValidationResult mirrors /api/v1/algorithms/validate response.
type ValidationResult struct {
	Valid       bool                   `json:"valid"`
	TestResults []TestResult           `json:"test_results"`
	Performance map[string]interface{} `json:"performance"`
}

// Category describes one entry from /api/v1/algorithms/categories.
type Category struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// StatsResponse mirrors /api/v1/algorithms/stats.
type StatsResponse struct {
	Statistics struct {
		TotalAlgorithms      int `json:"total_algorithms"`
		TotalImplementations int `json:"total_implementations"`
		TotalTestCases       int `json:"total_test_cases"`
		ValidatedCount       int `json:"validated_implementations"`
	} `json:"statistics"`
	Languages map[string]int `json:"languages"`
}

// HealthResponse mirrors the shared /health envelope returned by api-core.
type HealthResponse struct {
	Status   string          `json:"status"`
	Service  string          `json:"service,omitempty"`
	Version  string          `json:"version,omitempty"`
	Database string          `json:"database,omitempty"`
	Checks   json.RawMessage `json:"checks,omitempty"`
}
