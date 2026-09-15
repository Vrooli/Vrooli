package support

import "encoding/json"

// Pattern mirrors the shape returned by /api/v1/patterns/{id} and appears
// inside search/recipes responses.
type Pattern struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Chapter       string   `json:"chapter"`
	Section       string   `json:"section"`
	MaturityLevel string   `json:"maturity_level"`
	Tags          []string `json:"tags"`
	WhatAndWhy    string   `json:"what_and_why,omitempty"`
	WhenToUse     string   `json:"when_to_use,omitempty"`
	Tradeoffs     string   `json:"tradeoffs,omitempty"`
	RefPatterns   []string `json:"reference_patterns,omitempty"`
	FailureModes  string   `json:"failure_modes,omitempty"`
	CostLevers    string   `json:"cost_levers,omitempty"`
	RecipeCount   int      `json:"recipe_count,omitempty"`
	ImplCount     int      `json:"implementation_count,omitempty"`
	Languages     []string `json:"languages,omitempty"`
	CreatedAt     string   `json:"created_at,omitempty"`
	UpdatedAt     string   `json:"updated_at,omitempty"`
}

// PatternSearchResponse is the envelope wrapping /api/v1/patterns/search.
type PatternSearchResponse struct {
	Patterns []Pattern              `json:"patterns"`
	Total    int                    `json:"total"`
	Limit    int                    `json:"limit"`
	Offset   int                    `json:"offset"`
	Facets   map[string]interface{} `json:"facets,omitempty"`
}

// Recipe mirrors the shape returned by /api/v1/recipes/{id} and appears inside
// pattern recipes responses.
type Recipe struct {
	ID               string                   `json:"id"`
	PatternID        string                   `json:"pattern_id"`
	Title            string                   `json:"title"`
	Type             string                   `json:"type"`
	Prerequisites    []string                 `json:"prerequisites,omitempty"`
	Steps            []map[string]interface{} `json:"steps,omitempty"`
	ConfigSnippets   map[string]interface{}   `json:"config_snippets,omitempty"`
	ValidationChecks []string                 `json:"validation_checks,omitempty"`
	Artifacts        []string                 `json:"artifacts,omitempty"`
	Metrics          []string                 `json:"metrics,omitempty"`
	Rollbacks        []string                 `json:"rollbacks,omitempty"`
	Prompts          []string                 `json:"prompts,omitempty"`
	TimeoutSec       int                      `json:"timeout_sec,omitempty"`
}

// PatternRecipesResponse wraps /api/v1/patterns/{id}/recipes.
type PatternRecipesResponse struct {
	Pattern Pattern  `json:"pattern"`
	Recipes []Recipe `json:"recipes"`
}

// Chapter is one row in /api/v1/patterns/chapters.
type Chapter struct {
	Name         string `json:"name"`
	PatternCount int    `json:"pattern_count"`
}

// StatsResponse wraps /api/v1/patterns/stats.
type StatsResponse struct {
	Statistics struct {
		TotalPatterns        int `json:"total_patterns"`
		TotalRecipes         int `json:"total_recipes"`
		TotalImplementations int `json:"total_implementations"`
		TotalChapters        int `json:"total_chapters"`
	} `json:"statistics"`
	MaturityLevels map[string]int `json:"maturity_levels"`
	Languages      map[string]int `json:"languages"`
}

// Implementation mirrors the shape returned by /api/v1/implementations.
type Implementation struct {
	ID           string   `json:"id"`
	RecipeID     string   `json:"recipe_id"`
	Language     string   `json:"language"`
	Code         string   `json:"code"`
	FilePath     string   `json:"file_path,omitempty"`
	Description  string   `json:"description,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	TestCode     string   `json:"test_code,omitempty"`
}

// GenerationResult wraps POST /api/v1/recipes/generate.
type GenerationResult struct {
	GeneratedCode     string          `json:"generated_code"`
	FileStructure     json.RawMessage `json:"file_structure,omitempty"`
	Dependencies      []string        `json:"dependencies,omitempty"`
	SetupInstructions []string        `json:"setup_instructions,omitempty"`
}
