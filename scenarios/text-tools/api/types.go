package main

import (
	"fmt"
)

// Configuration holds application configuration
type Config struct {
	Port         string
	DatabaseURL  string
	MinIOURL     string
	RedisURL     string
	OllamaURL    string
	QdrantURL    string
}

// Health Response
type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp int64                  `json:"timestamp"`
	Database  string                 `json:"database"`
	Resources map[string]interface{} `json:"resources"`
	Version   string                 `json:"version"`
}

// Error Responses
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Version string `json:"version"`
}

type ErrorResponseV2 struct {
	Error     string `json:"error"`
	Code      int    `json:"code"`
	Version   string `json:"version"`
	Timestamp int64  `json:"timestamp"`
	RequestID string `json:"request_id"`
}

// API v1 Types

// DiffRequest represents a request to compare two texts
type DiffRequest struct {
	Text1   interface{} `json:"text1"`
	Text2   interface{} `json:"text2"`
	Options DiffOptions `json:"options,omitempty"`
}

// DiffOptions configures diff behavior
type DiffOptions struct {
	Type             string `json:"type,omitempty"`
	IgnoreWhitespace bool   `json:"ignore_whitespace,omitempty"`
	IgnoreCase       bool   `json:"ignore_case,omitempty"`
	GeneratePatch    bool   `json:"generate_patch,omitempty"`
	IncludeMetrics   bool   `json:"include_metrics,omitempty"`
}

// DiffResponse contains diff results
type DiffResponse struct {
	Changes         []Change `json:"changes"`
	SimilarityScore float64  `json:"similarity_score"`
	Summary         string   `json:"summary"`
}

// Change represents a diff change
type Change struct {
	Type      string `json:"type"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
	Content   string `json:"content"`
}

// SearchRequest represents a text search request
type SearchRequest struct {
	Text    interface{}   `json:"text"`
	Pattern string        `json:"pattern"`
	Options SearchOptions `json:"options,omitempty"`
}

// SearchOptions configures search behavior
type SearchOptions struct {
	Regex         bool `json:"regex,omitempty"`
	CaseSensitive bool `json:"case_sensitive,omitempty"`
	WholeWord     bool `json:"whole_word,omitempty"`
	Fuzzy         bool `json:"fuzzy,omitempty"`
	Semantic      bool `json:"semantic,omitempty"`
}

// SearchResponse contains search results
type SearchResponse struct {
	Matches      []Match `json:"matches"`
	TotalMatches int     `json:"total_matches"`
}

// Match represents a search match
type Match struct {
	Line    int     `json:"line"`
	Column  int     `json:"column"`
	Length  int     `json:"length"`
	Context string  `json:"context"`
	Score   float64 `json:"score,omitempty"`
}

// TransformRequest represents a text transformation request
type TransformRequest struct {
	Text            string           `json:"text"`
	Transformations []Transformation `json:"transformations"`
}

// Transformation defines a text transformation
type Transformation struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// TransformResponse contains transformation results
type TransformResponse struct {
	Result                 string   `json:"result"`
	TransformationsApplied []string `json:"transformations_applied"`
	Warnings               []string `json:"warnings,omitempty"`
}

// ExtractRequest represents a text extraction request
type ExtractRequest struct {
	Source  interface{}     `json:"source"`
	Format  string          `json:"format,omitempty"`
	Options ExtractOptions  `json:"options,omitempty"`
}

// ExtractOptions configures extraction behavior
type ExtractOptions struct {
	OCR                bool `json:"ocr,omitempty"`
	PreserveFormatting bool `json:"preserve_formatting,omitempty"`
	ExtractMetadata    bool `json:"extract_metadata,omitempty"`
	ExtractStructured  bool `json:"extract_structured,omitempty"`
}

// ExtractResponse contains extraction results
type ExtractResponse struct {
	Text     string                 `json:"text"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Warnings []string               `json:"warnings,omitempty"`
}

// AnalyzeRequest represents a text analysis request
type AnalyzeRequest struct {
	Text     string          `json:"text"`
	Analyses []string        `json:"analyses"`
	Options  AnalyzeOptions  `json:"options,omitempty"`
}

// AnalyzeOptions configures analysis behavior
type AnalyzeOptions struct {
	SummaryLength int    `json:"summary_length,omitempty"`
	EntityTypes   []string `json:"entity_types,omitempty"`
	UseAI         bool   `json:"use_ai,omitempty"`
}

// AnalyzeResponse contains analysis results
type AnalyzeResponse struct {
	Entities   []Entity               `json:"entities,omitempty"`
	Sentiment  Sentiment              `json:"sentiment,omitempty"`
	Summary    string                 `json:"summary,omitempty"`
	Keywords   []Keyword              `json:"keywords,omitempty"`
	Language   Language               `json:"language,omitempty"`
	Statistics TextStatistics         `json:"statistics,omitempty"`
}

// TextStatistics represents basic text statistics
type TextStatistics struct {
	WordCount      int     `json:"word_count"`
	CharCount      int     `json:"character_count"`
	LineCount      int     `json:"line_count"`
	SentenceCount  int     `json:"sentence_count"`
	ParagraphCount int     `json:"paragraph_count"`
	AvgWordLength  float64 `json:"avg_word_length"`
	ReadingTime    int     `json:"reading_time_seconds"`
}

// Entity represents an extracted entity
type Entity struct {
	Type       string  `json:"type"`
	Value      string  `json:"value"`
	Confidence float64 `json:"confidence"`
}

// Sentiment represents sentiment analysis result
type Sentiment struct {
	Score float64 `json:"score"`
	Label string  `json:"label"`
}

// Keyword represents an extracted keyword
type Keyword struct {
	Word  string  `json:"word"`
	Score float64 `json:"score"`
}

// Language represents language detection result
type Language struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
}

// API v2 Types (Extended versions with additional features)

// DiffRequestV2 extends DiffRequest with v2 features
type DiffRequestV2 struct {
	DiffRequest
	Options DiffOptionsV2 `json:"options,omitempty"`
}

// DiffOptionsV2 extends DiffOptions with v2 features
type DiffOptionsV2 struct {
	DiffOptions
	GeneratePatch  bool `json:"generate_patch,omitempty"`
	IncludeMetrics bool `json:"include_metrics,omitempty"`
}

// DiffResponseV2 extends DiffResponse with v2 features
type DiffResponseV2 struct {
	DiffResponse
	RequestID      string                 `json:"request_id"`
	ProcessingTime int64                  `json:"processing_time_ms"`
	Patch          string                 `json:"patch,omitempty"`
	Metrics        map[string]interface{} `json:"metrics,omitempty"`
}

// SearchRequestV2 extends SearchRequest with v2 features
type SearchRequestV2 struct {
	SearchRequest
	Options SearchOptionsV2 `json:"options,omitempty"`
}

// SearchOptionsV2 extends SearchOptions with v2 features
type SearchOptionsV2 struct {
	SearchOptions
	MaxResults int `json:"max_results,omitempty"`
	Offset     int `json:"offset,omitempty"`
}

// SearchResponseV2 extends SearchResponse with v2 features
type SearchResponseV2 struct {
	SearchResponse
	RequestID       string         `json:"request_id"`
	SemanticMatches []SemanticMatch `json:"semantic_matches,omitempty"`
}

// SemanticMatch represents a semantic search match
type SemanticMatch struct {
	Text       string  `json:"text"`
	Score      float64 `json:"score"`
	Similarity float64 `json:"similarity"`
}

// TransformRequestV2 extends TransformRequest with v2 features
type TransformRequestV2 struct {
	TransformRequest
	Options TransformOptionsV2 `json:"options,omitempty"`
}

// TransformOptionsV2 adds v2 specific options
type TransformOptionsV2 struct {
	TrackIntermediates bool `json:"track_intermediates,omitempty"`
	ValidateOutput     bool `json:"validate_output,omitempty"`
}

// TransformResponseV2 extends TransformResponse with v2 features
type TransformResponseV2 struct {
	TransformResponse
	RequestID           string   `json:"request_id"`
	IntermediateResults []string `json:"intermediate_results,omitempty"`
}

// ExtractRequestV2 extends ExtractRequest with v2 features
type ExtractRequestV2 struct {
	ExtractRequest
	Options ExtractOptionsV2 `json:"options,omitempty"`
}

// ExtractOptionsV2 extends ExtractOptions with v2 features
type ExtractOptionsV2 struct {
	ExtractOptions
	Stream bool `json:"stream,omitempty"`
}

// ExtractResponseV2 extends ExtractResponse with v2 features
type ExtractResponseV2 struct {
	ExtractResponse
	RequestID      string                 `json:"request_id"`
	StructuredData map[string]interface{} `json:"structured_data,omitempty"`
}

// AnalyzeRequestV2 extends AnalyzeRequest with v2 features
type AnalyzeRequestV2 struct {
	AnalyzeRequest
	Options AnalyzeOptionsV2 `json:"options,omitempty"`
}

// AnalyzeOptionsV2 extends AnalyzeOptions with v2 features
type AnalyzeOptionsV2 struct {
	AnalyzeOptions
	DeepAnalysis bool `json:"deep_analysis,omitempty"`
}

// AnalyzeResponseV2 extends AnalyzeResponse with v2 features
type AnalyzeResponseV2 struct {
	AnalyzeResponse
	RequestID  string                 `json:"request_id"`
	AIInsights map[string]interface{} `json:"ai_insights,omitempty"`
}

// Pipeline types (v2 only)

// PipelineRequest represents a text processing pipeline
type PipelineRequest struct {
	Input string         `json:"input"`
	Steps []PipelineStep `json:"steps"`
}

// PipelineStep represents a single pipeline step
type PipelineStep struct {
	Name       string                 `json:"name"`
	Operation  string                 `json:"operation"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// PipelineResponse contains pipeline results
type PipelineResponse struct {
	FinalOutput string               `json:"final_output"`
	Steps       []PipelineStepResult `json:"steps"`
	RequestID   string               `json:"request_id"`
}

// PipelineStepResult contains results from a pipeline step
type PipelineStepResult struct {
	StepName string                 `json:"step_name"`
	Output   string                 `json:"output"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Validation methods for v2 types

func (r *DiffRequestV2) Validate() error {
	if r.Text1 == nil || r.Text2 == nil {
		return fmt.Errorf("both text1 and text2 are required")
	}
	return nil
}

func (r *SearchRequestV2) Validate() error {
	if r.Pattern == "" {
		return fmt.Errorf("pattern is required")
	}
	if r.Options.MaxResults < 0 {
		return fmt.Errorf("max_results must be non-negative")
	}
	return nil
}

func (r *TransformRequestV2) Validate() error {
	if r.Text == "" {
		return fmt.Errorf("text is required")
	}
	if len(r.Transformations) == 0 {
		return fmt.Errorf("at least one transformation is required")
	}
	return nil
}

func (r *ExtractRequestV2) Validate() error {
	if r.Source == nil {
		return fmt.Errorf("source is required")
	}
	return nil
}

func (r *AnalyzeRequestV2) Validate() error {
	if r.Text == "" {
		return fmt.Errorf("text is required")
	}
	if len(r.Analyses) == 0 {
		return fmt.Errorf("at least one analysis type is required")
	}
	return nil
}

func (r *PipelineRequest) Validate() error {
	if r.Input == "" {
		return fmt.Errorf("input is required")
	}
	if len(r.Steps) == 0 {
		return fmt.Errorf("at least one pipeline step is required")
	}
	for i, step := range r.Steps {
		if step.Name == "" {
			return fmt.Errorf("step %d: name is required", i)
		}
		if step.Operation == "" {
			return fmt.Errorf("step %d: operation is required", i)
		}
	}
	return nil
}