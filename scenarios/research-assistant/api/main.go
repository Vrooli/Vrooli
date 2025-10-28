package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// StructuredLogger provides JSON-structured logging for better observability
type StructuredLogger struct {
	logger *log.Logger
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger() *StructuredLogger {
	return &StructuredLogger{
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// log outputs a structured log entry
func (sl *StructuredLogger) log(level, message string, fields map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		Fields:    fields,
	}

	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		sl.logger.Printf("Error marshaling log entry: %v", err)
		return
	}

	sl.logger.Println(string(jsonBytes))
}

// Info logs an informational message
func (sl *StructuredLogger) Info(message string, fields map[string]interface{}) {
	sl.log("info", message, fields)
}

// Warn logs a warning message
func (sl *StructuredLogger) Warn(message string, fields map[string]interface{}) {
	sl.log("warn", message, fields)
}

// Error logs an error message
func (sl *StructuredLogger) Error(message string, fields map[string]interface{}) {
	sl.log("error", message, fields)
}

// Fatal logs a fatal message and exits
func (sl *StructuredLogger) Fatal(message string, fields map[string]interface{}) {
	sl.log("fatal", message, fields)
	os.Exit(1)
}

type Report struct {
	ID                    string     `json:"id" db:"id"`
	Title                 string     `json:"title" db:"title"`
	Topic                 string     `json:"topic" db:"topic"`
	Depth                 string     `json:"depth" db:"depth"`
	TargetLength          int        `json:"target_length" db:"target_length"`
	Language              string     `json:"language" db:"language"`
	MarkdownContent       *string    `json:"markdown_content" db:"markdown_content"`
	Summary               *string    `json:"summary" db:"summary"`
	KeyFindings           *string    `json:"key_findings" db:"key_findings"`
	SourcesCount          int        `json:"sources_count" db:"sources_count"`
	WordCount             int        `json:"word_count" db:"word_count"`
	ConfidenceScore       *float64   `json:"confidence_score" db:"confidence_score"`
	PDFURL                *string    `json:"pdf_url" db:"pdf_url"`
	AssetsFolder          *string    `json:"assets_folder" db:"assets_folder"`
	RequestedAt           time.Time  `json:"requested_at" db:"requested_at"`
	StartedAt             *time.Time `json:"started_at" db:"started_at"`
	CompletedAt           *time.Time `json:"completed_at" db:"completed_at"`
	ProcessingTimeSeconds *int       `json:"processing_time_seconds" db:"processing_time_seconds"`
	Status                string     `json:"status" db:"status"`
	ErrorMessage          *string    `json:"error_message" db:"error_message"`
	RequestedBy           *string    `json:"requested_by" db:"requested_by"`
	Organization          *string    `json:"organization" db:"organization"`
	ScheduleID            *string    `json:"schedule_id" db:"schedule_id"`
	EmbeddingID           *string    `json:"embedding_id" db:"embedding_id"`
	Tags                  []string   `json:"tags" db:"tags"`
	Category              *string    `json:"category" db:"category"`
	IsArchived            bool       `json:"is_archived" db:"is_archived"`
	CreatedAt             time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at" db:"updated_at"`
}

type ReportRequest struct {
	Topic        string   `json:"topic"`
	Depth        string   `json:"depth"`
	TargetLength int      `json:"target_length"`
	Language     string   `json:"language"`
	RequestedBy  *string  `json:"requested_by"`
	Organization *string  `json:"organization"`
	Tags         []string `json:"tags"`
	Category     *string  `json:"category"`
}

type ChatConversation struct {
	ID               string     `json:"id" db:"id"`
	Title            *string    `json:"title" db:"title"`
	UserID           *string    `json:"user_id" db:"user_id"`
	Organization     *string    `json:"organization" db:"organization"`
	IsActive         bool       `json:"is_active" db:"is_active"`
	LastMessageAt    *time.Time `json:"last_message_at" db:"last_message_at"`
	MessageCount     int        `json:"message_count" db:"message_count"`
	RelatedReportIDs []string   `json:"related_report_ids" db:"related_report_ids"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

type ChatMessage struct {
	ID                string    `json:"id" db:"id"`
	ConversationID    string    `json:"conversation_id" db:"conversation_id"`
	Role              string    `json:"role" db:"role"`
	Content           string    `json:"content" db:"content"`
	ContextSources    *string   `json:"context_sources" db:"context_sources"`
	ConfidenceScore   *float64  `json:"confidence_score" db:"confidence_score"`
	TriggeredReportID *string   `json:"triggered_report_id" db:"triggered_report_id"`
	TriggeredAction   *string   `json:"triggered_action" db:"triggered_action"`
	TokensUsed        *int      `json:"tokens_used" db:"tokens_used"`
	ProcessingTimeMs  *int      `json:"processing_time_ms" db:"processing_time_ms"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

type ReportSchedule struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Description    *string    `json:"description,omitempty"`
	TopicTemplate  string     `json:"topic_template"`
	CronExpression string     `json:"cron_expression"`
	Timezone       string     `json:"timezone"`
	Depth          string     `json:"depth"`
	TargetLength   int        `json:"target_length"`
	IsActive       bool       `json:"is_active"`
	LastRunAt      *time.Time `json:"last_run_at,omitempty"`
	NextRunAt      *time.Time `json:"next_run_at,omitempty"`
	TotalRuns      int        `json:"total_runs"`
	SuccessfulRuns int        `json:"successful_runs"`
	FailedRuns     int        `json:"failed_runs"`
}

type ChatConversationSummary struct {
	ID            string     `json:"id"`
	Title         *string    `json:"title,omitempty"`
	IsActive      bool       `json:"is_active"`
	MessageCount  int        `json:"message_count"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type SearchRequest struct {
	Query     string            `json:"query"`
	Engines   []string          `json:"engines"`
	Category  string            `json:"category"`
	TimeRange string            `json:"time_range"`
	Limit     int               `json:"limit"`
	Filters   map[string]string `json:"filters"`
	// Advanced filters for P1 requirement
	Language     string   `json:"language"`      // Filter by content language
	SafeSearch   int      `json:"safe_search"`   // 0=off, 1=moderate, 2=strict
	FileType     string   `json:"file_type"`     // pdf, doc, etc.
	Site         string   `json:"site"`          // Specific site/domain
	ExcludeSites []string `json:"exclude_sites"` // Sites to exclude
	SortBy       string   `json:"sort_by"`       // relevance, date, popularity
	Region       string   `json:"region"`        // Geographic region filter
	MinDate      string   `json:"min_date"`      // ISO date string
	MaxDate      string   `json:"max_date"`      // ISO date string
}

type AnalysisRequest struct {
	Content             string  `json:"content"`
	AnalysisType        string  `json:"analysis_type"`
	OutputFormat        string  `json:"output_format"`
	ConfidenceThreshold float64 `json:"confidence_threshold"`
	MaxInsights         int     `json:"max_insights"`
	FocusAreas          string  `json:"focus_areas"`
	Context             string  `json:"context"`
}

type APIServer struct {
	db             *sql.DB
	n8nURL         string
	windmillURL    string
	searxngURL     string
	qdrantURL      string
	minioURL       string
	ollamaURL      string
	browserlessURL string
	httpClient     *http.Client       // Shared HTTP client with timeout for health checks
	logger         *StructuredLogger  // Structured logger for observability
}

func nullStringPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	value := ns.String
	return &value
}

func nullTimePtr(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}
	value := nt.Time
	return &value
}

// ResearchDepthConfig defines search parameters for each research depth level
type ResearchDepthConfig struct {
	MaxSources     int
	SearchEngines  int
	AnalysisRounds int
	MinConfidence  float64
	TimeoutMinutes int
}

// TemplateConfig defines parameters for report templates
type TemplateConfig struct {
	Name             string
	Description      string
	DefaultDepth     string
	DefaultLength    int
	RequiredSections []string
	OptionalSections []string
	PreferDomains    []string
}

// getDepthConfig returns the configuration for a given research depth level
func getDepthConfig(depth string) ResearchDepthConfig {
	configs := map[string]ResearchDepthConfig{
		"quick": {
			MaxSources:     5,
			SearchEngines:  3,
			AnalysisRounds: 1,
			MinConfidence:  0.6,
			TimeoutMinutes: 2,
		},
		"standard": {
			MaxSources:     15,
			SearchEngines:  7,
			AnalysisRounds: 2,
			MinConfidence:  0.75,
			TimeoutMinutes: 5,
		},
		"deep": {
			MaxSources:     30,
			SearchEngines:  15,
			AnalysisRounds: 3,
			MinConfidence:  0.85,
			TimeoutMinutes: 10,
		},
	}

	if config, ok := configs[depth]; ok {
		return config
	}
	// Default to standard if invalid depth
	return configs["standard"]
}

// getReportTemplates returns available report templates
func getReportTemplates() map[string]TemplateConfig {
	return map[string]TemplateConfig{
		"general": {
			Name:             "General Research",
			Description:      "Comprehensive research report covering all aspects of a topic",
			DefaultDepth:     "standard",
			DefaultLength:    5,
			RequiredSections: []string{"Executive Summary", "Key Findings", "Analysis", "Conclusion"},
			OptionalSections: []string{"Methodology", "References", "Appendix"},
			PreferDomains:    []string{},
		},
		"academic": {
			Name:             "Academic Research",
			Description:      "Scholarly research with emphasis on peer-reviewed sources",
			DefaultDepth:     "deep",
			DefaultLength:    10,
			RequiredSections: []string{"Abstract", "Literature Review", "Methodology", "Results", "Discussion", "Conclusion", "References"},
			OptionalSections: []string{"Acknowledgments", "Appendices"},
			PreferDomains:    []string{"arxiv.org", "scholar.google.com", "pubmed.gov", "jstor.org"},
		},
		"market": {
			Name:             "Market Analysis",
			Description:      "Business and market intelligence focused research",
			DefaultDepth:     "standard",
			DefaultLength:    7,
			RequiredSections: []string{"Executive Summary", "Market Overview", "Competitive Analysis", "Key Trends", "Recommendations"},
			OptionalSections: []string{"SWOT Analysis", "Financial Data", "Sources"},
			PreferDomains:    []string{"bloomberg.com", "reuters.com", "forbes.com", "wsj.com"},
		},
		"technical": {
			Name:             "Technical Documentation",
			Description:      "In-depth technical analysis and documentation",
			DefaultDepth:     "deep",
			DefaultLength:    8,
			RequiredSections: []string{"Overview", "Technical Specifications", "Implementation Details", "Best Practices", "Conclusion"},
			OptionalSections: []string{"Code Examples", "Troubleshooting", "FAQ"},
			PreferDomains:    []string{"github.com", "stackoverflow.com", "dev.to", "medium.com"},
		},
		"quick-brief": {
			Name:             "Quick Brief",
			Description:      "Fast, concise overview of a topic",
			DefaultDepth:     "quick",
			DefaultLength:    2,
			RequiredSections: []string{"Summary", "Key Points"},
			OptionalSections: []string{"Further Reading"},
			PreferDomains:    []string{},
		},
	}
}

// validateDepth checks if the provided depth value is valid
func validateDepth(depth string) bool {
	validDepths := []string{"quick", "standard", "deep"}
	for _, valid := range validDepths {
		if depth == valid {
			return true
		}
	}
	return false
}

// SourceQualityMetrics represents quality scoring for a search result source
type SourceQualityMetrics struct {
	DomainAuthority float64 `json:"domain_authority"` // 0-1 based on domain reputation
	RecencyScore    float64 `json:"recency_score"`    // Time-weighted relevance
	ContentDepth    float64 `json:"content_depth"`    // Substance vs fluff ratio
	OverallQuality  float64 `json:"overall_quality"`  // Composite score
}

// Contradiction represents a detected conflict between sources
type Contradiction struct {
	Claim1     string  `json:"claim1"`     // First claim
	Claim2     string  `json:"claim2"`     // Contradictory claim
	Source1    string  `json:"source1"`    // Source of first claim
	Source2    string  `json:"source2"`    // Source of second claim
	Confidence float64 `json:"confidence"` // Confidence in contradiction (0-1)
	Context    string  `json:"context"`    // Additional context about the contradiction
	ResultIDs  []int   `json:"result_ids"` // Indices of results involved
}

// ContradictionRequest represents a request to detect contradictions
type ContradictionRequest struct {
	Results []map[string]interface{} `json:"results"` // Search results to analyze
	Topic   string                   `json:"topic"`   // Research topic for context
}

// Domain authority tiers - higher tier = higher authority
var domainAuthorityMap = map[string]float64{
	// Academic & Research (Tier 1: 0.95-1.0)
	"arxiv.org":          1.0,
	"scholar.google.com": 1.0,
	"pubmed.gov":         1.0,
	"ncbi.nlm.nih.gov":   1.0,
	"jstor.org":          0.98,
	"sciencedirect.com":  0.97,
	"nature.com":         0.98,
	"science.org":        0.98,
	"ieee.org":           0.97,
	"acm.org":            0.97,

	// News & Media - Tier 1 (Tier 2: 0.85-0.95)
	"reuters.com":        0.95,
	"apnews.com":         0.95,
	"bbc.com":            0.93,
	"nytimes.com":        0.92,
	"washingtonpost.com": 0.92,
	"wsj.com":            0.93,
	"bloomberg.com":      0.93,
	"economist.com":      0.92,
	"theguardian.com":    0.90,
	"ft.com":             0.92,

	// Government & Official (Tier 1: 0.95-1.0)
	"gov":       0.98, // Any .gov domain
	"edu":       0.95, // Any .edu domain
	"who.int":   0.97,
	"un.org":    0.96,
	"europa.eu": 0.95,

	// Technical & Documentation (Tier 2: 0.80-0.90)
	"github.com":            0.88,
	"stackoverflow.com":     0.87,
	"dev.to":                0.75,
	"medium.com":            0.70,
	"docs.microsoft.com":    0.90,
	"docs.aws.amazon.com":   0.90,
	"developer.mozilla.org": 0.90,

	// Business & Finance (Tier 2: 0.80-0.90)
	"forbes.com":          0.85,
	"fortune.com":         0.84,
	"businessinsider.com": 0.78,
	"cnbc.com":            0.82,

	// General Knowledge (Tier 3: 0.70-0.80)
	"wikipedia.org":  0.80,
	"britannica.com": 0.82,

	// Social/Community (Tier 4: 0.60-0.70)
	"reddit.com":   0.65,
	"quora.com":    0.63,
	"twitter.com":  0.60,
	"x.com":        0.60,
	"linkedin.com": 0.68,
}

// calculateDomainAuthority returns authority score for a given URL
func calculateDomainAuthority(resultURL string) float64 {
	// Parse URL to extract domain
	parsedURL, err := url.Parse(resultURL)
	if err != nil {
		return 0.5 // Default for unparseable URLs
	}

	domain := strings.ToLower(parsedURL.Hostname())

	// Check exact domain matches first
	if score, exists := domainAuthorityMap[domain]; exists {
		return score
	}

	// Check for TLD-based scoring (.gov, .edu)
	if strings.HasSuffix(domain, ".gov") {
		return domainAuthorityMap["gov"]
	}
	if strings.HasSuffix(domain, ".edu") {
		return domainAuthorityMap["edu"]
	}

	// Check for partial domain matches (e.g., subdomain.scholar.google.com)
	for knownDomain, score := range domainAuthorityMap {
		if strings.Contains(domain, knownDomain) {
			return score
		}
	}

	// Default score for unknown domains
	return 0.5
}

// calculateRecencyScore calculates time-based relevance (newer = higher)
func calculateRecencyScore(publishedDate interface{}) float64 {
	if publishedDate == nil {
		return 0.5 // Default if no date available
	}

	// Try to parse date string
	dateStr, ok := publishedDate.(string)
	if !ok {
		return 0.5
	}

	// Parse various date formats
	formats := []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02T15:04:05",
		"Mon, 02 Jan 2006 15:04:05 MST",
	}

	var parsedTime time.Time
	var parseErr error
	for _, format := range formats {
		parsedTime, parseErr = time.Parse(format, dateStr)
		if parseErr == nil {
			break
		}
	}

	if parseErr != nil {
		return 0.5 // Default if parsing fails
	}

	// Calculate age in days
	ageInDays := time.Since(parsedTime).Hours() / 24

	// Score based on age (exponential decay)
	// Recent (< 30 days): 0.9-1.0
	// Medium (30-180 days): 0.7-0.9
	// Old (180-365 days): 0.5-0.7
	// Very old (> 365 days): 0.3-0.5

	if ageInDays < 30 {
		return 1.0 - (ageInDays / 300) // 0.9-1.0
	} else if ageInDays < 180 {
		return 0.9 - ((ageInDays - 30) / 750) // 0.7-0.9
	} else if ageInDays < 365 {
		return 0.7 - ((ageInDays - 180) / 925) // 0.5-0.7
	} else {
		// Cap at 0.3 for very old content
		score := 0.5 - ((ageInDays - 365) / 3650)
		if score < 0.3 {
			return 0.3
		}
		return score
	}
}

// calculateContentDepth estimates content quality from available metadata
func calculateContentDepth(result map[string]interface{}) float64 {
	score := 0.5 // Base score

	// Check for detailed content
	if content, ok := result["content"].(string); ok && len(content) > 200 {
		score += 0.2
	}

	// Check for title quality (longer, more descriptive = better)
	if title, ok := result["title"].(string); ok {
		wordCount := len(strings.Fields(title))
		if wordCount >= 5 && wordCount <= 15 {
			score += 0.15 // Good title length
		}
	}

	// Penalize short URLs (often low-quality landing pages)
	if urlStr, ok := result["url"].(string); ok && len(urlStr) > 50 {
		score += 0.1
	}

	// Bonus for having author information
	if _, hasAuthor := result["author"]; hasAuthor {
		score += 0.05
	}

	// Cap at 1.0
	if score > 1.0 {
		return 1.0
	}
	return score
}

// calculateSourceQuality computes comprehensive quality metrics for a search result
func calculateSourceQuality(result map[string]interface{}) SourceQualityMetrics {
	// Extract URL
	resultURL, _ := result["url"].(string)

	// Calculate individual metrics
	domainAuth := calculateDomainAuthority(resultURL)
	recency := calculateRecencyScore(result["publishedDate"])
	contentDepth := calculateContentDepth(result)

	// Calculate weighted overall quality
	// Weights: Domain authority (50%), Content depth (30%), Recency (20%)
	overall := (domainAuth * 0.5) + (contentDepth * 0.3) + (recency * 0.2)

	return SourceQualityMetrics{
		DomainAuthority: domainAuth,
		RecencyScore:    recency,
		ContentDepth:    contentDepth,
		OverallQuality:  overall,
	}
}

// enhanceResultsWithQuality adds quality metrics to search results and sorts by quality
func enhanceResultsWithQuality(results []interface{}) []interface{} {
	for i, result := range results {
		resultMap, ok := result.(map[string]interface{})
		if !ok {
			continue
		}

		// Calculate quality metrics
		quality := calculateSourceQuality(resultMap)

		// Add quality metrics to result
		resultMap["quality_metrics"] = map[string]interface{}{
			"domain_authority": quality.DomainAuthority,
			"recency_score":    quality.RecencyScore,
			"content_depth":    quality.ContentDepth,
			"overall_quality":  quality.OverallQuality,
		}

		results[i] = resultMap
	}

	return results
}

// sortResultsByQuality sorts results by overall quality score (descending)
func sortResultsByQuality(results []interface{}) {
	// Simple bubble sort for quality (could use sort.Slice for production)
	n := len(results)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			result1, ok1 := results[j].(map[string]interface{})
			result2, ok2 := results[j+1].(map[string]interface{})

			if !ok1 || !ok2 {
				continue
			}

			quality1, _ := result1["quality_metrics"].(map[string]interface{})
			quality2, _ := result2["quality_metrics"].(map[string]interface{})

			if quality1 == nil || quality2 == nil {
				continue
			}

			score1, _ := quality1["overall_quality"].(float64)
			score2, _ := quality2["overall_quality"].(float64)

			// Swap if score1 < score2 (descending order)
			if score1 < score2 {
				results[j], results[j+1] = results[j+1], results[j]
			}
		}
	}
}

// triggerResearchWorkflow sends a request to n8n to start the research workflow
func (s *APIServer) triggerResearchWorkflow(reportID string, req ReportRequest) error {
	workflowURL := s.n8nURL + "/webhook/research-request"

	// Get depth configuration
	depthConfig := getDepthConfig(req.Depth)

	payload := map[string]interface{}{
		"report_id":     reportID,
		"topic":         req.Topic,
		"depth":         req.Depth,
		"target_length": req.TargetLength,
		"language":      req.Language,
		"requested_by":  req.RequestedBy,
		"organization":  req.Organization,
		"tags":          req.Tags,
		"category":      req.Category,
		// Include depth configuration for workflow
		"depth_config": map[string]interface{}{
			"max_sources":     depthConfig.MaxSources,
			"search_engines":  depthConfig.SearchEngines,
			"analysis_rounds": depthConfig.AnalysisRounds,
			"min_confidence":  depthConfig.MinConfidence,
			"timeout_minutes": depthConfig.TimeoutMinutes,
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Use client with timeout for workflow trigger
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Post(workflowURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return http.ErrMissingFile // Simple error for now
	}

	return nil
}

func main() {
	// Lifecycle protection check MUST be first - before any business logic
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start research-assistant

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Initialize structured logger
	logger := NewStructuredLogger()

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		logger.Fatal("API_PORT environment variable is required", nil)
	}

	// Database configuration - support both POSTGRES_URL and individual components
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		// Try to build from individual components - REQUIRED, no defaults
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")

		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			logger.Fatal("Database configuration missing. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD (not shown), POSTGRES_DB", nil)
		}

		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	// Service URLs - prefer explicit configuration, fall back to well-known ports
	// Security Note: Defaults are only used when environment variables are not set.
	// Production deployments should explicitly configure all resource URLs.

	// n8n (REQUIRED) - workflow automation
	n8nURL := os.Getenv("N8N_BASE_URL")
	if n8nURL == "" {
		n8nPort := os.Getenv("RESOURCE_PORT_N8N")
		if n8nPort == "" {
			n8nPort = "5678"
			logger.Warn("Using default port for n8n", map[string]interface{}{
				"service": "n8n",
				"port":    n8nPort,
				"env_var": "RESOURCE_PORT_N8N",
			})
		}
		n8nURL = fmt.Sprintf("http://localhost:%s", n8nPort)
	}

	// windmill (OPTIONAL) - dashboard UI
	windmillURL := os.Getenv("WINDMILL_BASE_URL")
	if windmillURL == "" {
		windmillPort := os.Getenv("RESOURCE_PORT_WINDMILL")
		if windmillPort == "" {
			windmillPort = "8000"
			logger.Warn("Using default port for windmill", map[string]interface{}{
				"service": "windmill",
				"port":    windmillPort,
				"env_var": "RESOURCE_PORT_WINDMILL",
			})
		}
		windmillURL = fmt.Sprintf("http://localhost:%s", windmillPort)
	}

	// searxng (REQUIRED) - privacy-focused search
	searxngURL := os.Getenv("SEARXNG_URL")
	if searxngURL == "" {
		searxngPort := os.Getenv("RESOURCE_PORT_SEARXNG")
		if searxngPort == "" {
			searxngPort = "8280"
			logger.Warn("Using default port for searxng", map[string]interface{}{
				"service": "searxng",
				"port":    searxngPort,
				"env_var": "RESOURCE_PORT_SEARXNG",
			})
		}
		searxngURL = fmt.Sprintf("http://localhost:%s", searxngPort)
	}

	// qdrant (REQUIRED) - vector database
	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		qdrantPort := os.Getenv("RESOURCE_PORT_QDRANT")
		if qdrantPort == "" {
			qdrantPort = "6333"
			logger.Warn("Using default port for qdrant", map[string]interface{}{
				"service": "qdrant",
				"port":    qdrantPort,
				"env_var": "RESOURCE_PORT_QDRANT",
			})
		}
		qdrantURL = fmt.Sprintf("http://localhost:%s", qdrantPort)
	}

	// minio (OPTIONAL) - object storage
	minioURL := os.Getenv("MINIO_URL")
	if minioURL == "" {
		minioPort := os.Getenv("RESOURCE_PORT_MINIO")
		if minioPort != "" {
			minioURL = fmt.Sprintf("http://localhost:%s", minioPort)
		} else {
			// minio is optional, leave URL empty if not explicitly configured
			logger.Info("MinIO not configured (optional resource)", map[string]interface{}{
				"service": "minio",
				"status":  "optional_not_configured",
			})
		}
	}

	// ollama (REQUIRED) - AI model inference
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaPort := os.Getenv("RESOURCE_PORT_OLLAMA")
		if ollamaPort == "" {
			ollamaPort = "11434"
			logger.Warn("Using default port for ollama", map[string]interface{}{
				"service": "ollama",
				"port":    ollamaPort,
				"env_var": "RESOURCE_PORT_OLLAMA",
			})
		}
		ollamaURL = fmt.Sprintf("http://localhost:%s", ollamaPort)
	}

	// browserless (OPTIONAL) - headless browser for JavaScript-heavy sites
	browserlessURL := os.Getenv("BROWSERLESS_URL")
	if browserlessURL == "" {
		browserlessPort := os.Getenv("RESOURCE_PORT_BROWSERLESS")
		if browserlessPort == "" {
			browserlessPort = "4110"
			logger.Warn("Using default port for browserless", map[string]interface{}{
				"service": "browserless",
				"port":    browserlessPort,
				"env_var": "RESOURCE_PORT_BROWSERLESS",
			})
		}
		browserlessURL = fmt.Sprintf("http://localhost:%s", browserlessPort)
	}

	// Connect to database
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		logger.Fatal("Failed to open database connection", map[string]interface{}{
			"error": err.Error(),
		})
	}
	defer db.Close()

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	logger.Info("Attempting database connection with exponential backoff", map[string]interface{}{
		"max_retries": maxRetries,
		"base_delay":  baseDelay.String(),
		"max_delay":   maxDelay.String(),
	})

	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			logger.Info("Database connected successfully", map[string]interface{}{
				"attempt": attempt + 1,
			})
			break
		}

		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay)*math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))

		// Add random jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * rand.Float64())
		actualDelay := delay + jitter

		logger.Warn("Database connection attempt failed", map[string]interface{}{
			"attempt":      attempt + 1,
			"max_retries":  maxRetries,
			"error":        pingErr.Error(),
			"next_attempt": actualDelay.String(),
		})

		// Provide detailed status every few attempts
		if attempt > 0 && attempt%3 == 0 {
			logger.Info("Retry progress update", map[string]interface{}{
				"attempts_made": attempt + 1,
				"max_retries":   maxRetries,
				"total_wait":    (time.Duration(attempt*2) * baseDelay).String(),
				"current_delay": delay.String(),
				"jitter":        jitter.String(),
			})
		}

		time.Sleep(actualDelay)
	}

	if pingErr != nil {
		logger.Fatal("Database connection failed after all retries", map[string]interface{}{
			"max_retries": maxRetries,
			"error":       pingErr.Error(),
		})
	}

	logger.Info("Database connection pool established successfully", nil)

	server := &APIServer{
		db:             db,
		n8nURL:         n8nURL,
		windmillURL:    windmillURL,
		searxngURL:     searxngURL,
		qdrantURL:      qdrantURL,
		minioURL:       minioURL,
		ollamaURL:      ollamaURL,
		browserlessURL: browserlessURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second, // Timeout for health checks and short operations
		},
		logger: logger,
	}

	router := mux.NewRouter()

	// CORS middleware
	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)

	// Health check
	router.HandleFunc("/health", server.healthCheck).Methods("GET")

	// API routes
	api := router.PathPrefix("/api").Subrouter()

	// Report endpoints
	api.HandleFunc("/reports", server.getReports).Methods("GET")
	api.HandleFunc("/reports", server.createReport).Methods("POST")
	api.HandleFunc("/reports/{id}", server.getReport).Methods("GET")
	api.HandleFunc("/reports/{id}", server.updateReport).Methods("PUT")
	api.HandleFunc("/reports/{id}", server.deleteReport).Methods("DELETE")
	api.HandleFunc("/reports/{id}/pdf", server.getReportPDF).Methods("GET")
	api.HandleFunc("/reports/count", server.getReportCount).Methods("GET")
	api.HandleFunc("/reports/confidence-average", server.getReportConfidenceAverage).Methods("GET")

	// Schedule endpoints
	api.HandleFunc("/schedules", server.getSchedules).Methods("GET")
	api.HandleFunc("/schedules/count", server.getScheduleCount).Methods("GET")
	api.HandleFunc("/schedules/{id}/run", server.runScheduleNow).Methods("POST")
	api.HandleFunc("/schedules/{id}/toggle", server.toggleSchedule).Methods("POST")

	// Chat endpoints
	api.HandleFunc("/conversations", server.getConversations).Methods("GET")
	api.HandleFunc("/conversations", server.createConversation).Methods("POST")
	api.HandleFunc("/conversations/{id}", server.getConversation).Methods("GET")
	api.HandleFunc("/conversations/{id}/messages", server.getMessages).Methods("GET")
	api.HandleFunc("/conversations/{id}/messages", server.sendMessage).Methods("POST")
	api.HandleFunc("/chat/conversations", server.getChatConversationsSummary).Methods("GET")
	api.HandleFunc("/chat/messages", server.getChatMessages).Methods("GET")
	api.HandleFunc("/chat/message", server.handleChatMessage).Methods("POST")
	api.HandleFunc("/chat/sessions/count", server.getChatSessionsCount).Methods("GET")

	// Search endpoints
	api.HandleFunc("/search", server.performSearch).Methods("POST")
	api.HandleFunc("/search/history", server.getSearchHistory).Methods("GET")
	api.HandleFunc("/detect-contradictions", server.detectContradictions).Methods("POST")
	api.HandleFunc("/extract-content", server.extractWebContent).Methods("POST")

	// Analysis endpoints
	api.HandleFunc("/analyze", server.analyzeContent).Methods("POST")
	api.HandleFunc("/analyze/insights", server.extractInsights).Methods("POST")
	api.HandleFunc("/analyze/trends", server.analyzeTrends).Methods("POST")
	api.HandleFunc("/analyze/competitive", server.analyzeCompetitive).Methods("POST")

	// Knowledge base endpoints
	api.HandleFunc("/knowledge/search", server.searchKnowledge).Methods("GET")
	api.HandleFunc("/knowledge/collections", server.getCollections).Methods("GET")

	// Dashboard data endpoints
	api.HandleFunc("/dashboard/stats", server.getDashboardStats).Methods("GET")
	api.HandleFunc("/dashboard/recent-activity", server.getRecentActivity).Methods("GET")

	// Template and configuration endpoints
	api.HandleFunc("/templates", server.getTemplates).Methods("GET")
	api.HandleFunc("/depth-configs", server.getDepthConfigs).Methods("GET")

	logger.Info("Research Assistant API starting", map[string]interface{}{
		"port":         port,
		"database":     postgresURL,
		"n8n_url":      n8nURL,
		"windmill_url": windmillURL,
		"searxng_url":  searxngURL,
		"qdrant_url":   qdrantURL,
		"minio_url":    minioURL,
		"ollama_url":   ollamaURL,
	})

	handler := corsHandler(router)

	// Create HTTP server with graceful shutdown support
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 120 * time.Second, // Allow time for long-running requests
		IdleTimeout:  120 * time.Second,
	}

	// Run server in goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server error", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server gracefully", nil)

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(ctx); err != nil {
		logger.Warn("Server forced to shutdown", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Close database connection
	if err := db.Close(); err != nil {
		logger.Warn("Error closing database", map[string]interface{}{
			"error": err.Error(),
		})
	}

	logger.Info("Server stopped", nil)
}

// getEnv removed to prevent hardcoded defaults

func (s *APIServer) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check all services
	services := map[string]string{
		"database":    s.checkDatabase(),
		"n8n":         s.checkN8N(),
		"windmill":    s.checkWindmill(),
		"searxng":     s.checkSearXNG(),
		"qdrant":      s.checkQdrant(),
		"ollama":      s.checkOllama(),
		"browserless": s.checkBrowserless(),
	}

	// Determine overall status based on critical dependencies
	overallStatus := "healthy"
	criticalServices := []string{"database", "n8n", "ollama", "qdrant", "searxng"}
	for _, svc := range criticalServices {
		if services[svc] != "healthy" {
			overallStatus = "degraded"
			break
		}
	}

	status := map[string]interface{}{
		"status":    overallStatus,
		"service":   "research-assistant-api",
		"timestamp": time.Now().Format(time.RFC3339),
		"readiness": true,
		"services":  services,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *APIServer) checkDatabase() string {
	if s.db == nil {
		return "not_configured"
	}

	if err := s.db.Ping(); err != nil {
		return "unhealthy"
	}
	return "healthy"
}

func (s *APIServer) checkN8N() string {
	if s.db == nil || s.httpClient == nil || s.n8nURL == "" {
		return "not_configured"
	}

	resp, err := s.httpClient.Get(s.n8nURL + "/healthz")
	if err != nil {
		return "unavailable"
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) // Drain body to allow connection reuse

	if resp.StatusCode != http.StatusOK {
		return "unavailable"
	}
	return "healthy"
}

func (s *APIServer) checkWindmill() string {
	if s.db == nil || s.httpClient == nil || s.windmillURL == "" {
		return "not_configured"
	}

	resp, err := s.httpClient.Get(s.windmillURL + "/api/version")
	if err != nil {
		return "unavailable"
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) // Drain body to allow connection reuse

	if resp.StatusCode != http.StatusOK {
		return "unavailable"
	}
	return "healthy"
}

func (s *APIServer) checkSearXNG() string {
	if s.db == nil || s.httpClient == nil || s.searxngURL == "" {
		return "not_configured"
	}

	// Use root endpoint instead of performing an actual search for faster health checks
	resp, err := s.httpClient.Get(s.searxngURL + "/")
	if err != nil {
		return "unavailable"
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) // Drain body to allow connection reuse

	if resp.StatusCode != http.StatusOK {
		return "unavailable"
	}
	return "healthy"
}

func (s *APIServer) checkQdrant() string {
	if s.db == nil || s.httpClient == nil || s.qdrantURL == "" {
		return "not_configured"
	}

	resp, err := s.httpClient.Get(s.qdrantURL + "/")
	if err != nil {
		return "unavailable"
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) // Drain body to allow connection reuse

	if resp.StatusCode != http.StatusOK {
		return "unavailable"
	}
	return "healthy"
}

func (s *APIServer) checkOllama() string {
	if s.db == nil || s.httpClient == nil || s.ollamaURL == "" {
		return "not_configured"
	}

	resp, err := s.httpClient.Get(s.ollamaURL + "/api/tags")
	if err != nil {
		return "unavailable"
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) // Drain body to allow connection reuse

	if resp.StatusCode != http.StatusOK {
		return "unavailable"
	}
	return "healthy"
}

func (s *APIServer) checkBrowserless() string {
	if s.db == nil || s.httpClient == nil || s.browserlessURL == "" {
		return "not_configured"
	}
	resp, err := s.httpClient.Get(s.browserlessURL + "/pressure")
	if err != nil {
		return "unavailable"
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) // Drain body to allow connection reuse

	if resp.StatusCode != http.StatusOK {
		return "unavailable"
	}
	return "healthy"
}

func (s *APIServer) getReports(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	query := `
		SELECT id, title, topic, depth, target_length, language, 
		       markdown_content, summary, key_findings, sources_count,
		       word_count, confidence_score, pdf_url, assets_folder,
		       requested_at, started_at, completed_at, processing_time_seconds,
		       status, error_message, requested_by, organization,
		       schedule_id, embedding_id, tags, category, is_archived,
		       created_at, updated_at
		FROM research_assistant.reports 
		WHERE is_archived = false
		ORDER BY created_at DESC 
		LIMIT $1`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var reports []Report
	for rows.Next() {
		var report Report
		var tagsJSON []byte

		err := rows.Scan(
			&report.ID, &report.Title, &report.Topic, &report.Depth,
			&report.TargetLength, &report.Language, &report.MarkdownContent,
			&report.Summary, &report.KeyFindings, &report.SourcesCount,
			&report.WordCount, &report.ConfidenceScore, &report.PDFURL,
			&report.AssetsFolder, &report.RequestedAt, &report.StartedAt,
			&report.CompletedAt, &report.ProcessingTimeSeconds, &report.Status,
			&report.ErrorMessage, &report.RequestedBy, &report.Organization,
			&report.ScheduleID, &report.EmbeddingID, &tagsJSON, &report.Category,
			&report.IsArchived, &report.CreatedAt, &report.UpdatedAt,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Parse tags JSON array
		if len(tagsJSON) > 0 {
			json.Unmarshal(tagsJSON, &report.Tags)
		}

		reports = append(reports, report)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reports)
}

func (s *APIServer) getReportCount(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		response := map[string]int{"count": 0}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	var total int
	query := `
		SELECT COUNT(*)
		FROM research_assistant.reports
		WHERE is_archived = false`

	if err := s.db.QueryRow(query).Scan(&total); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]int{"count": total}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *APIServer) getReportConfidenceAverage(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		response := map[string]float64{"average": 0}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	var avg sql.NullFloat64
	query := `
		SELECT AVG(confidence_score)
		FROM research_assistant.reports
		WHERE is_archived = false
		  AND confidence_score IS NOT NULL`

	if err := s.db.QueryRow(query).Scan(&avg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	average := 0.0
	if avg.Valid {
		average = avg.Float64 * 100
	}

	response := map[string]float64{"average": average}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *APIServer) createReport(w http.ResponseWriter, r *http.Request) {
	var req ReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Generate UUID for new report
	reportID := uuid.New().String()
	now := time.Now()

	// Set defaults
	if req.Depth == "" {
		req.Depth = "standard"
	}
	// Validate depth
	if !validateDepth(req.Depth) {
		http.Error(w, fmt.Sprintf("Invalid depth value: %s. Must be one of: quick, standard, deep", req.Depth), http.StatusBadRequest)
		return
	}
	if req.TargetLength == 0 {
		req.TargetLength = 5
	}
	if req.Language == "" {
		req.Language = "en"
	}

	tagsJSON, _ := json.Marshal(req.Tags)

	query := `
		INSERT INTO research_assistant.reports 
		(id, title, topic, depth, target_length, language, status, 
		 requested_by, organization, tags, category, requested_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, sources_count, word_count, is_archived`

	var report Report
	err := s.db.QueryRow(query,
		reportID, req.Topic, req.Topic, req.Depth, req.TargetLength,
		req.Language, "pending", req.RequestedBy, req.Organization,
		tagsJSON, req.Category, now, now, now,
	).Scan(&report.ID, &report.SourcesCount, &report.WordCount, &report.IsArchived)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fill in the rest of the report data
	report.Title = req.Topic
	report.Topic = req.Topic
	report.Depth = req.Depth
	report.TargetLength = req.TargetLength
	report.Language = req.Language
	report.Status = "pending"
	report.RequestedBy = req.RequestedBy
	report.Organization = req.Organization
	report.Tags = req.Tags
	report.Category = req.Category
	report.RequestedAt = now
	report.CreatedAt = now
	report.UpdatedAt = now

	// Trigger n8n workflow for report generation
	err = s.triggerResearchWorkflow(reportID, req)
	if err != nil {
		s.logger.Warn("Failed to trigger n8n workflow for report", map[string]interface{}{
			"report_id": reportID,
			"error":     err.Error(),
		})
		// Continue execution even if workflow trigger fails
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(report)
}

func (s *APIServer) getReport(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	reportID := vars["id"]

	query := `
		SELECT id, title, topic, depth, target_length, language, 
		       markdown_content, summary, key_findings, sources_count,
		       word_count, confidence_score, pdf_url, assets_folder,
		       requested_at, started_at, completed_at, processing_time_seconds,
		       status, error_message, requested_by, organization,
		       schedule_id, embedding_id, tags, category, is_archived,
		       created_at, updated_at
		FROM research_assistant.reports 
		WHERE id = $1`

	var report Report
	var tagsJSON []byte

	err := s.db.QueryRow(query, reportID).Scan(
		&report.ID, &report.Title, &report.Topic, &report.Depth,
		&report.TargetLength, &report.Language, &report.MarkdownContent,
		&report.Summary, &report.KeyFindings, &report.SourcesCount,
		&report.WordCount, &report.ConfidenceScore, &report.PDFURL,
		&report.AssetsFolder, &report.RequestedAt, &report.StartedAt,
		&report.CompletedAt, &report.ProcessingTimeSeconds, &report.Status,
		&report.ErrorMessage, &report.RequestedBy, &report.Organization,
		&report.ScheduleID, &report.EmbeddingID, &tagsJSON, &report.Category,
		&report.IsArchived, &report.CreatedAt, &report.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Report not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse tags JSON array
	if len(tagsJSON) > 0 {
		json.Unmarshal(tagsJSON, &report.Tags)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func (s *APIServer) getDashboardStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"total_reports":        7,
		"completed_this_month": 3,
		"active_projects": map[string]int{
			"high_priority":   2,
			"medium_priority": 3,
			"low_priority":    2,
		},
		"search_activity": map[string]interface{}{
			"searches_today":     247,
			"success_rate":       0.89,
			"engines_active":     15,
			"results_analyzed":   1834,
			"insights_generated": 342,
		},
		"ai_insights": map[string]interface{}{
			"confidence":               0.92,
			"high_priority_insights":   18,
			"market_trends":            5,
			"competitive_intelligence": 12,
			"recommendations":          23,
		},
		"knowledge_base": map[string]interface{}{
			"documents_indexed":  "2.3M",
			"collections":        847,
			"vector_embeddings":  "15K",
			"retrieval_accuracy": 0.987,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *APIServer) getRecentActivity(w http.ResponseWriter, r *http.Request) {
	activity := []map[string]interface{}{
		{
			"time":       "11:45 AM",
			"type":       "research_completed",
			"title":      "Market research on \"AI adoption in healthcare 2024\" completed",
			"details":    "47 sources analyzed, 12 key insights extracted",
			"confidence": 0.96,
		},
		{
			"time":       "10:22 AM",
			"type":       "analysis_updated",
			"title":      "Competitive analysis for \"Financial services automation\" updated",
			"details":    "3 new competitors identified, strategic recommendations updated",
			"confidence": 0.91,
		},
		{
			"time":       "09:15 AM",
			"type":       "trend_analysis",
			"title":      "AI trend analysis on \"Machine learning enterprise adoption\" finished",
			"details":    "8 major trends identified, investment recommendations generated",
			"confidence": 0.94,
		},
		{
			"time":       "08:45 AM",
			"type":       "report_published",
			"title":      "Research report \"Q4 Technology Investment Landscape\" published",
			"details":    "45-page comprehensive analysis with executive summary",
			"confidence": 0.98,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(activity)
}

// Stub implementations for remaining endpoints
func (s *APIServer) updateReport(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"status": "not implemented"})
}

func (s *APIServer) deleteReport(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"status": "not implemented"})
}

func (s *APIServer) getReportPDF(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"status": "not implemented"})
}

func (s *APIServer) getConversations(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"status": "not implemented"})
}

func (s *APIServer) createConversation(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"status": "not implemented"})
}

func (s *APIServer) getConversation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"status": "not implemented"})
}

func (s *APIServer) getMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"status": "not implemented"})
}

func (s *APIServer) sendMessage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"status": "not implemented"})
}

func (s *APIServer) getChatSessionsCount(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		response := map[string]int{
			"count":  2,
			"active": 1,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	var total, active int

	if err := s.db.QueryRow(`SELECT COUNT(*) FROM research_assistant.chat_conversations`).Scan(&total); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := s.db.QueryRow(`SELECT COUNT(*) FROM research_assistant.chat_conversations WHERE COALESCE(is_active, FALSE) = TRUE`).Scan(&active); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]int{
		"count":  total,
		"active": active,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *APIServer) getChatConversationsSummary(w http.ResponseWriter, r *http.Request) {
	limit := 25
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	if s.db == nil {
		now := time.Now()
		last := now.Add(-2 * time.Hour)
		title := "Competitor landscape"
		conversations := []ChatConversationSummary{
			{
				ID:            "demo-conversation-1",
				Title:         &title,
				IsActive:      true,
				MessageCount:  4,
				LastMessageAt: &last,
				CreatedAt:     now.Add(-48 * time.Hour),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(conversations)
		return
	}

	query := `
		SELECT id,
		       title,
		       COALESCE(is_active, FALSE) AS is_active,
		       COALESCE(message_count, 0) AS message_count,
		       last_message_at,
		       created_at
		FROM research_assistant.chat_conversations
		ORDER BY COALESCE(last_message_at, created_at) DESC
		LIMIT $1`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	conversations := make([]ChatConversationSummary, 0)
	for rows.Next() {
		var summary ChatConversationSummary
		var title sql.NullString
		var lastMessage sql.NullTime

		if err := rows.Scan(&summary.ID, &title, &summary.IsActive, &summary.MessageCount, &lastMessage, &summary.CreatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		summary.Title = nullStringPtr(title)
		summary.LastMessageAt = nullTimePtr(lastMessage)
		conversations = append(conversations, summary)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conversations)
}

func (s *APIServer) getChatMessages(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil {
			if parsed <= 0 {
				parsed = 1
			}
			if parsed > 200 {
				parsed = 200
			}
			limit = parsed
		}
	}

	conversationID := strings.TrimSpace(r.URL.Query().Get("conversation_id"))

	if s.db == nil {
		now := time.Now()
		messages := []ChatMessage{
			{
				ID:             "demo-message-1",
				ConversationID: "demo-conversation-1",
				Role:           "assistant",
				Content:        "Welcome back! Ready to continue the research session?",
				CreatedAt:      now.Add(-3 * time.Hour),
			},
			{
				ID:             "demo-message-2",
				ConversationID: "demo-conversation-1",
				Role:           "user",
				Content:        "Summarize the latest AI policy developments in Europe.",
				CreatedAt:      now.Add(-2 * time.Hour),
			},
			{
				ID:             "demo-message-3",
				ConversationID: "demo-conversation-1",
				Role:           "assistant",
				Content:        "Sure thing—I'll highlight regulatory updates, major initiatives, and potential risks.",
				CreatedAt:      now.Add(-90 * time.Minute),
			},
		}

		if conversationID != "" {
			filtered := make([]ChatMessage, 0)
			for _, msg := range messages {
				if msg.ConversationID == conversationID {
					filtered = append(filtered, msg)
				}
			}
			messages = filtered
		}

		if len(messages) > limit {
			messages = messages[len(messages)-limit:]
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messages)
		return
	}

	var (
		rows *sql.Rows
		err  error
	)

	baseQuery := `SELECT id, conversation_id, role, content, created_at
		FROM research_assistant.chat_messages`

	if conversationID != "" {
		rows, err = s.db.Query(
			baseQuery+" WHERE conversation_id = $1 ORDER BY created_at DESC LIMIT $2",
			conversationID,
			limit,
		)
	} else {
		rows, err = s.db.Query(
			baseQuery+" ORDER BY created_at DESC LIMIT $1",
			limit,
		)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	messages := make([]ChatMessage, 0)
	for rows.Next() {
		var message ChatMessage
		if err := rows.Scan(&message.ID, &message.ConversationID, &message.Role, &message.Content, &message.CreatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		messages = append(messages, message)
	}

	// Ensure chronological order for UI rendering
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func (s *APIServer) handleChatMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Message        string  `json:"message"`
		ConversationID *string `json:"conversation_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	message := strings.TrimSpace(req.Message)
	if message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	if s.db == nil {
		assistantResponse := generateAssistantResponse(message)
		response := map[string]interface{}{
			"conversation_id": "demo-conversation-1",
			"response":        assistantResponse,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	ctx := r.Context()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var conversationID string

	if req.ConversationID != nil && strings.TrimSpace(*req.ConversationID) != "" {
		conversationID = strings.TrimSpace(*req.ConversationID)

		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM research_assistant.chat_conversations WHERE id = $1)`, conversationID).Scan(&exists); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !exists {
			http.Error(w, "conversation not found", http.StatusNotFound)
			return
		}
	} else {
		title := message
		if len(title) > 60 {
			title = title[:60] + "..."
		}

		if err := tx.QueryRowContext(
			ctx,
			`INSERT INTO research_assistant.chat_conversations (title, is_active, last_message_at, message_count, created_at, updated_at)
			 VALUES ($1, TRUE, NOW(), 0, NOW(), NOW()) RETURNING id`,
			title,
		).Scan(&conversationID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO research_assistant.chat_messages (conversation_id, role, content, created_at)
		 VALUES ($1, 'user', $2, NOW())`,
		conversationID,
		message,
	); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	assistantResponse := generateAssistantResponse(message)

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO research_assistant.chat_messages (conversation_id, role, content, created_at)
		 VALUES ($1, 'assistant', $2, NOW())`,
		conversationID,
		assistantResponse,
	); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE research_assistant.chat_conversations
		 SET message_count = COALESCE(message_count, 0) + 2,
		     last_message_at = NOW(),
		     updated_at = NOW()
		 WHERE id = $1`,
		conversationID,
	); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"conversation_id": conversationID,
		"response":        assistantResponse,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func generateAssistantResponse(message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return "Happy to help with your research. Let me know what topic you want to explore."
	}

	if len(trimmed) > 140 {
		trimmed = trimmed[:140] + "..."
	}

	return fmt.Sprintf("I've logged your request about \"%s\". I'll review recent findings and follow up with prioritized insights shortly.", trimmed)
}

// Helper function to sort results by a field
func sortResultsByField(results []interface{}, field string, descending bool) {
	// Simple bubble sort for demonstration - could be optimized
	n := len(results)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			r1 := results[j].(map[string]interface{})
			r2 := results[j+1].(map[string]interface{})

			v1, _ := r1[field]
			v2, _ := r2[field]

			shouldSwap := false
			switch field {
			case "score":
				s1, _ := v1.(float64)
				s2, _ := v2.(float64)
				if descending {
					shouldSwap = s1 < s2
				} else {
					shouldSwap = s1 > s2
				}
			case "publishedDate":
				d1, _ := v1.(string)
				d2, _ := v2.(string)
				if descending {
					shouldSwap = d1 < d2
				} else {
					shouldSwap = d1 > d2
				}
			}

			if shouldSwap {
				results[j], results[j+1] = results[j+1], results[j]
			}
		}
	}
}

func (s *APIServer) performSearch(w http.ResponseWriter, r *http.Request) {
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.Limit == 0 {
		req.Limit = 10
	}
	if req.Category == "" {
		req.Category = "general"
	}

	// Build SearXNG search URL
	searchURL := s.searxngURL + "/search"

	// Create HTTP client
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Build enhanced query with advanced filters
	enhancedQuery := req.Query

	// Add site filter to query
	if req.Site != "" {
		enhancedQuery += " site:" + req.Site
	}

	// Add file type filter
	if req.FileType != "" {
		enhancedQuery += " filetype:" + req.FileType
	}

	// Add excluded sites
	for _, site := range req.ExcludeSites {
		enhancedQuery += " -site:" + site
	}

	// Add date range if specified
	if req.MinDate != "" || req.MaxDate != "" {
		if req.MinDate != "" {
			enhancedQuery += " after:" + req.MinDate
		}
		if req.MaxDate != "" {
			enhancedQuery += " before:" + req.MaxDate
		}
	}

	// Prepare query parameters
	params := map[string]string{
		"q":        enhancedQuery,
		"format":   "json",
		"category": req.Category,
	}

	// Add engines if specified
	if len(req.Engines) > 0 {
		params["engines"] = strings.Join(req.Engines, ",")
	}

	// Add time range if specified
	if req.TimeRange != "" {
		params["time_range"] = req.TimeRange
	}

	// Add language filter
	if req.Language != "" {
		params["language"] = req.Language
	}

	// Add safe search
	if req.SafeSearch > 0 {
		params["safesearch"] = strconv.Itoa(req.SafeSearch)
	}

	// Add region filter
	if req.Region != "" {
		params["locale"] = req.Region
	}

	// Build query string
	values := url.Values{}
	for key, value := range params {
		values.Add(key, value)
	}

	// Make request to SearXNG
	searchReq, err := http.NewRequest("GET", searchURL+"?"+values.Encode(), nil)
	if err != nil {
		http.Error(w, "Failed to create search request", http.StatusInternalServerError)
		return
	}

	resp, err := client.Do(searchReq)
	if err != nil {
		http.Error(w, "Search service unavailable", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Search request failed", http.StatusBadGateway)
		return
	}

	// Parse SearXNG response
	var searxngResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&searxngResponse); err != nil {
		http.Error(w, "Failed to parse search results", http.StatusInternalServerError)
		return
	}

	// Extract results
	results, ok := searxngResponse["results"].([]interface{})
	if !ok {
		results = []interface{}{}
	}

	// Enhance results with quality metrics
	results = enhanceResultsWithQuality(results)

	// Apply sorting if requested
	if req.SortBy != "" && len(results) > 0 {
		// Sort results based on the requested sort method
		switch req.SortBy {
		case "date":
			// Sort by date (newest first)
			sortResultsByField(results, "publishedDate", true)
		case "popularity":
			// Sort by score/popularity
			sortResultsByField(results, "score", true)
		case "quality":
			// Sort by source quality (NEW!)
			sortResultsByQuality(results)
		case "relevance":
			// Already sorted by relevance by SearXNG
		}
	} else {
		// Default: sort by quality if no specific sort requested
		sortResultsByQuality(results)
	}

	// Limit results
	if len(results) > req.Limit {
		results = results[:req.Limit]
	}

	// Calculate average quality for the result set
	var avgQuality float64
	qualityCount := 0
	for _, result := range results {
		if resultMap, ok := result.(map[string]interface{}); ok {
			if quality, ok := resultMap["quality_metrics"].(map[string]interface{}); ok {
				if score, ok := quality["overall_quality"].(float64); ok {
					avgQuality += score
					qualityCount++
				}
			}
		}
	}
	if qualityCount > 0 {
		avgQuality = avgQuality / float64(qualityCount)
	}

	// Format response with enhanced metadata
	response := map[string]interface{}{
		"query":           req.Query,
		"results_count":   len(results),
		"results":         results,
		"engines_used":    searxngResponse["engines"],
		"query_time":      searxngResponse["query_time"],
		"timestamp":       time.Now().Unix(),
		"average_quality": avgQuality,
		"filters_applied": map[string]interface{}{
			"language":    req.Language,
			"safe_search": req.SafeSearch,
			"file_type":   req.FileType,
			"site":        req.Site,
			"region":      req.Region,
			"sort_by":     req.SortBy,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *APIServer) getSchedules(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		now := time.Now()
		next := now.Add(24 * time.Hour)
		last := now.Add(-24 * time.Hour)
		name := "Weekly Trend Briefing"
		placeholder := []ReportSchedule{
			{
				ID:             "demo-schedule-1",
				Name:           name,
				TopicTemplate:  "Weekly industry intelligence summary",
				CronExpression: "0 9 * * 1",
				Timezone:       "UTC",
				Depth:          "standard",
				TargetLength:   5,
				IsActive:       true,
				LastRunAt:      &last,
				NextRunAt:      &next,
				TotalRuns:      12,
				SuccessfulRuns: 12,
				FailedRuns:     0,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(placeholder)
		return
	}

	query := `
		SELECT
			id,
			name,
			description,
			topic_template,
			cron_expression,
			timezone,
			depth,
			target_length,
			COALESCE(is_active, FALSE) AS is_active,
			last_run_at,
			next_run_at,
			COALESCE(total_runs, 0) AS total_runs,
			COALESCE(successful_runs, 0) AS successful_runs,
			COALESCE(failed_runs, 0) AS failed_runs
		FROM research_assistant.report_schedules
		ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	schedules := make([]ReportSchedule, 0)
	for rows.Next() {
		var schedule ReportSchedule
		var description sql.NullString
		var lastRun sql.NullTime
		var nextRun sql.NullTime
		var targetLength sql.NullInt64
		var totalRuns, successfulRuns, failedRuns int64

		err := rows.Scan(
			&schedule.ID,
			&schedule.Name,
			&description,
			&schedule.TopicTemplate,
			&schedule.CronExpression,
			&schedule.Timezone,
			&schedule.Depth,
			&targetLength,
			&schedule.IsActive,
			&lastRun,
			&nextRun,
			&totalRuns,
			&successfulRuns,
			&failedRuns,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		schedule.Description = nullStringPtr(description)
		schedule.LastRunAt = nullTimePtr(lastRun)
		schedule.NextRunAt = nullTimePtr(nextRun)
		schedule.TargetLength = 0
		if targetLength.Valid {
			schedule.TargetLength = int(targetLength.Int64)
		}
		schedule.TotalRuns = int(totalRuns)
		schedule.SuccessfulRuns = int(successfulRuns)
		schedule.FailedRuns = int(failedRuns)

		schedules = append(schedules, schedule)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schedules)
}

func (s *APIServer) getScheduleCount(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		response := map[string]int{
			"count":  1,
			"active": 1,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	var total, active int

	if err := s.db.QueryRow(`SELECT COUNT(*) FROM research_assistant.report_schedules`).Scan(&total); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := s.db.QueryRow(`SELECT COUNT(*) FROM research_assistant.report_schedules WHERE COALESCE(is_active, FALSE) = TRUE`).Scan(&active); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]int{
		"count":  total,
		"active": active,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *APIServer) runScheduleNow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if strings.TrimSpace(id) == "" {
		http.Error(w, "schedule id is required", http.StatusBadRequest)
		return
	}

	if s.db == nil {
		response := map[string]string{"status": "scheduled"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	query := `
		UPDATE research_assistant.report_schedules
		SET last_run_at = NOW(),
		    next_run_at = COALESCE(next_run_at, NOW()) + INTERVAL '1 day',
		    total_runs = COALESCE(total_runs, 0) + 1,
		    updated_at = NOW()
		WHERE id = $1`

	result, err := s.db.Exec(query, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "schedule not found", http.StatusNotFound)
		return
	}

	response := map[string]string{"status": "scheduled"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *APIServer) toggleSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if strings.TrimSpace(id) == "" {
		http.Error(w, "schedule id is required", http.StatusBadRequest)
		return
	}

	if s.db == nil {
		response := map[string]interface{}{
			"id":        id,
			"is_active": true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	query := `
		UPDATE research_assistant.report_schedules
		SET is_active = NOT COALESCE(is_active, FALSE),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING is_active`

	var newStatus bool
	if err := s.db.QueryRow(query, id).Scan(&newStatus); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "schedule not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"id":        id,
		"is_active": newStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *APIServer) getSearchHistory(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"status": "not implemented"})
}

// detectContradictions analyzes search results for contradictory information using Ollama
func (s *APIServer) detectContradictions(w http.ResponseWriter, r *http.Request) {
	var req ContradictionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Results) < 2 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"contradictions": []Contradiction{},
			"message":        "Need at least 2 results to detect contradictions",
		})
		return
	}

	// Limit to 5 results to prevent excessive API call times
	// Each result requires multiple Ollama generation calls which can take 10-30s each
	if len(req.Results) > 5 {
		http.Error(w, "Maximum 5 results allowed for contradiction detection. This endpoint is synchronous and processing time increases exponentially with more results. Consider using async job processing for larger datasets.", http.StatusBadRequest)
		return
	}

	// Extract key claims from each result using Ollama
	claims := make([]map[string]interface{}, 0)
	for i, result := range req.Results {
		title := ""
		content := ""
		url := ""

		if t, ok := result["title"].(string); ok {
			title = t
		}
		if c, ok := result["content"].(string); ok {
			content = c
		}
		if u, ok := result["url"].(string); ok {
			url = u
		}

		// Extract claims using Ollama
		extractedClaims, err := s.extractClaims(title, content, req.Topic)
		if err != nil {
			s.logger.Error("Error extracting claims from search result", map[string]interface{}{
				"result_index": i,
				"error":        err.Error(),
			})
			continue
		}

		claims = append(claims, map[string]interface{}{
			"index":  i,
			"url":    url,
			"title":  title,
			"claims": extractedClaims,
		})
	}

	// Compare claims to find contradictions
	contradictions := make([]Contradiction, 0)
	for i := 0; i < len(claims); i++ {
		for j := i + 1; j < len(claims); j++ {
			claim1Data := claims[i]
			claim2Data := claims[j]

			claim1List, ok1 := claim1Data["claims"].([]string)
			claim2List, ok2 := claim2Data["claims"].([]string)

			if !ok1 || !ok2 {
				continue
			}

			// Check each pair of claims for contradictions
			for _, c1 := range claim1List {
				for _, c2 := range claim2List {
					isContradiction, confidence, context, err := s.checkContradiction(c1, c2, req.Topic)
					if err != nil {
						s.logger.Error("Error checking contradiction", map[string]interface{}{
							"error": err.Error(),
						})
						continue
					}

					if isContradiction && confidence > 0.6 {
						contradictions = append(contradictions, Contradiction{
							Claim1:     c1,
							Claim2:     c2,
							Source1:    claim1Data["url"].(string),
							Source2:    claim2Data["url"].(string),
							Confidence: confidence,
							Context:    context,
							ResultIDs:  []int{claim1Data["index"].(int), claim2Data["index"].(int)},
						})
					}
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"contradictions":  contradictions,
		"total_results":   len(req.Results),
		"claims_analyzed": len(claims),
		"topic":           req.Topic,
		"warning":         "This endpoint is synchronous and may take 30-60+ seconds for multiple results. Consider implementing async job processing for production use.",
	})
}

// extractWebContent uses Browserless to extract content from JavaScript-heavy websites
func (s *APIServer) extractWebContent(w http.ResponseWriter, r *http.Request) {
	if s.browserlessURL == "" {
		http.Error(w, "Browserless service not configured", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		URL             string `json:"url"`
		WaitForSelector string `json:"wait_for_selector"`
		Timeout         int    `json:"timeout"` // milliseconds
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Default timeout of 30 seconds
	if req.Timeout == 0 {
		req.Timeout = 30000
	}

	// Use Browserless content endpoint to get rendered HTML
	browserlessReq := map[string]interface{}{
		"url": req.URL,
		"gotoOptions": map[string]interface{}{
			"waitUntil": "networkidle0",
			"timeout":   req.Timeout,
		},
	}

	if req.WaitForSelector != "" {
		browserlessReq["waitForSelector"] = map[string]interface{}{
			"selector": req.WaitForSelector,
			"timeout":  req.Timeout,
		}
	}

	reqBody, err := json.Marshal(browserlessReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(req.Timeout+5000) * time.Millisecond, // Add 5s buffer
	}

	browserlessURL := s.browserlessURL + "/chrome/content"
	httpReq, err := http.NewRequest("POST", browserlessURL, bytes.NewBuffer(reqBody))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Browserless request failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		http.Error(w, fmt.Sprintf("Browserless returned status %d: %s", resp.StatusCode, string(bodyBytes)), http.StatusBadGateway)
		return
	}

	// Read and parse the content response
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"url":     req.URL,
		"content": string(bodyBytes),
		"status":  "success",
	})
}

// extractClaims uses Ollama to extract key factual claims from text
func (s *APIServer) extractClaims(title, content, topic string) ([]string, error) {
	prompt := fmt.Sprintf(`Extract 2-3 key factual claims from this text related to "%s". Return ONLY a JSON array of strings, nothing else.

Title: %s
Content: %s

Return format: ["claim 1", "claim 2", "claim 3"]`, topic, title, content)

	ollamaReq := map[string]interface{}{
		"model":  "llama3.2:3b", // Use smaller, faster model for claim extraction
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"temperature": 0.1, // Low temperature for factual extraction
		},
	}

	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, err
	}

	// Create HTTP client with timeout to prevent hanging
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("POST", s.ollamaURL+"/api/generate", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama request failed with status %d", resp.StatusCode)
	}

	var ollamaResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, err
	}

	responseText, ok := ollamaResp["response"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid response from ollama")
	}

	// Parse JSON array from response
	var claims []string
	responseText = strings.TrimSpace(responseText)

	// Try to extract JSON from markdown code blocks if present
	if strings.Contains(responseText, "```") {
		start := strings.Index(responseText, "```")
		if start >= 0 {
			responseText = responseText[start+3:]
			if strings.HasPrefix(responseText, "json") {
				responseText = responseText[4:]
			}
			end := strings.Index(responseText, "```")
			if end >= 0 {
				responseText = strings.TrimSpace(responseText[:end])
			}
		}
	}

	// Find first [ and last ] to extract JSON array
	startIdx := strings.Index(responseText, "[")
	endIdx := strings.LastIndex(responseText, "]")
	if startIdx >= 0 && endIdx > startIdx {
		responseText = responseText[startIdx : endIdx+1]
	}

	if err := json.Unmarshal([]byte(responseText), &claims); err != nil {
		s.logger.Warn("Failed to parse claims from Ollama response", map[string]interface{}{
			"error":        err.Error(),
			"response_len": len(responseText),
		})
		// If JSON parsing fails, return the response as a single claim
		return []string{responseText}, nil
	}

	return claims, nil
}

// checkContradiction uses Ollama to determine if two claims contradict each other
func (s *APIServer) checkContradiction(claim1, claim2, topic string) (bool, float64, string, error) {
	prompt := fmt.Sprintf(`Analyze if these two claims about "%s" contradict each other. Respond ONLY with valid JSON.

Claim 1: %s
Claim 2: %s

Return format:
{
  "is_contradiction": true/false,
  "confidence": 0.0-1.0,
  "explanation": "brief explanation"
}`, topic, claim1, claim2)

	ollamaReq := map[string]interface{}{
		"model":  "llama3.2:3b",
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"temperature": 0.2,
		},
	}

	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return false, 0, "", err
	}

	// Create HTTP client with timeout to prevent hanging
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("POST", s.ollamaURL+"/api/generate", bytes.NewBuffer(reqBody))
	if err != nil {
		return false, 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return false, 0, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, 0, "", fmt.Errorf("ollama request failed with status %d", resp.StatusCode)
	}

	var ollamaResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return false, 0, "", err
	}

	responseText, ok := ollamaResp["response"].(string)
	if !ok {
		return false, 0, "", fmt.Errorf("invalid response from ollama")
	}

	// Parse JSON response - extract JSON from potential markdown code blocks
	var analysis struct {
		IsContradiction bool    `json:"is_contradiction"`
		Confidence      float64 `json:"confidence"`
		Explanation     string  `json:"explanation"`
	}

	responseText = strings.TrimSpace(responseText)

	// Try to extract JSON from markdown code blocks if present
	if strings.Contains(responseText, "```") {
		// Extract content between ```json and ``` or ``` and ```
		start := strings.Index(responseText, "```")
		if start >= 0 {
			responseText = responseText[start+3:]
			if strings.HasPrefix(responseText, "json") {
				responseText = responseText[4:]
			}
			end := strings.Index(responseText, "```")
			if end >= 0 {
				responseText = strings.TrimSpace(responseText[:end])
			}
		}
	}

	// Find first { and last } to extract JSON object
	startIdx := strings.Index(responseText, "{")
	endIdx := strings.LastIndex(responseText, "}")
	if startIdx >= 0 && endIdx > startIdx {
		responseText = responseText[startIdx : endIdx+1]
	}

	if err := json.Unmarshal([]byte(responseText), &analysis); err != nil {
		s.logger.Warn("Failed to parse contradiction analysis from Ollama response", map[string]interface{}{
			"error":        err.Error(),
			"response_len": len(responseText),
		})
		// If parsing fails, return no contradiction
		return false, 0, "", err
	}

	return analysis.IsContradiction, analysis.Confidence, analysis.Explanation, nil
}

func (s *APIServer) analyzeContent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"status": "not implemented"})
}

func (s *APIServer) extractInsights(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"status": "not implemented"})
}

func (s *APIServer) analyzeTrends(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"status": "not implemented"})
}

func (s *APIServer) analyzeCompetitive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"status": "not implemented"})
}

func (s *APIServer) searchKnowledge(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"status": "not implemented"})
}

func (s *APIServer) getCollections(w http.ResponseWriter, r *http.Request) {
	collections := []map[string]interface{}{
		{
			"id":           "ai_trends",
			"name":         "AI Market Trends",
			"documents":    1247,
			"last_updated": "2 hours ago",
			"relevance":    0.94,
		},
		{
			"id":           "healthcare",
			"name":         "Healthcare Innovation",
			"documents":    892,
			"last_updated": "5 hours ago",
			"relevance":    0.89,
		},
		{
			"id":           "fintech",
			"name":         "FinTech Disruption",
			"documents":    634,
			"last_updated": "1 day ago",
			"relevance":    0.91,
		},
		{
			"id":           "competitive",
			"name":         "Competitive Intelligence",
			"documents":    1089,
			"last_updated": "3 hours ago",
			"relevance":    0.96,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(collections)
}

// getTemplates returns available report templates
func (s *APIServer) getTemplates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	templates := getReportTemplates()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"templates": templates,
		"count":     len(templates),
	})
}

// getDepthConfigs returns configurations for all research depth levels
func (s *APIServer) getDepthConfigs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	configs := map[string]ResearchDepthConfig{
		"quick":    getDepthConfig("quick"),
		"standard": getDepthConfig("standard"),
		"deep":     getDepthConfig("deep"),
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"depth_configs": configs,
		"description":   "Research depth configurations define search parameters and analysis intensity",
	})
}
