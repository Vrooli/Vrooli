package main

import "time"

// FactCheckResult represents the result of fact-checking an article
type FactCheckResult struct {
	ArticleID   string    `json:"article_id"`
	CheckedAt   time.Time `json:"checked_at"`
	Claims      []Claim   `json:"claims"`
	OverallScore float64  `json:"overall_score"`
}

// Claim represents a single claim that was fact-checked
type Claim struct {
	Text       string `json:"text"`
	Verifiable bool   `json:"verifiable"`
	Verdict    string `json:"verdict"` // true, false, misleading, unverifiable
	Reasoning  string `json:"reasoning"`
	Sources    []string `json:"sources"`
}

// PerspectiveAggregation represents aggregated perspectives on a topic
type PerspectiveAggregation struct {
	Topic                string                `json:"topic"`
	ArticleCount         int                   `json:"article_count"`
	Perspectives         map[string][]Article  `json:"perspectives"`
	CommonGround         []string              `json:"common_ground"`
	Disagreements        []string              `json:"disagreements"`
	UniquePoints         interface{}           `json:"unique_points"`
	NarrativeDifferences string                `json:"narrative_differences"`
	AnalyzedAt           time.Time             `json:"analyzed_at"`
}

// BiasAnalysis represents detailed bias analysis
type BiasAnalysis struct {
	Score             float64  `json:"score"`
	Category          string   `json:"category"` // left, center-left, center, center-right, right
	LoadedLanguage    []string `json:"loaded_language"`
	MissingContext    []string `json:"missing_context"`
	SourceCredibility float64  `json:"source_credibility"`
	Explanation       string   `json:"explanation"`
}

// NewsEnrichment represents enriched news data
type NewsEnrichment struct {
	ArticleID      string    `json:"article_id"`
	Summary        string    `json:"summary"`
	KeyTopics      []string  `json:"key_topics"`
	Entities       []Entity  `json:"entities"`
	Sentiment      string    `json:"sentiment"`
	Category       string    `json:"category"`
	RelatedArticles []string `json:"related_articles"`
	EnrichedAt     time.Time `json:"enriched_at"`
}

// Entity represents a named entity extracted from text
type Entity struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // person, organization, location, etc.
	Mentions int    `json:"mentions"`
}