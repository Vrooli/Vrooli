package platforms

import (
	"time"
)

// PlatformIntegration defines the interface that all platform integrations must implement
type PlatformIntegration interface {
	// GetBookmarks retrieves bookmarks from the platform
	GetBookmarks(limit int) ([]StandardBookmark, error)
	
	// ProcessBookmark converts platform-specific bookmark to standard format
	ProcessBookmark(rawData interface{}) StandardBookmark
	
	// ValidateConfig validates the platform configuration
	ValidateConfig() error
	
	// TestConnection tests the platform API connection
	TestConnection() error
	
	// GetPlatformInfo returns metadata about this platform
	GetPlatformInfo() PlatformInfo
	
	// EstimateCategories suggests categories based on content
	EstimateCategories(bookmark StandardBookmark) []CategorySuggestion
	
	// GetSuggestedActions returns suggested actions for a bookmark
	GetSuggestedActions(bookmark StandardBookmark) []ActionSuggestion
}

// StandardBookmark represents a bookmark in our unified format
type StandardBookmark struct {
	ID              string                 `json:"id"`
	Platform        string                 `json:"platform"`
	OriginalURL     string                 `json:"original_url"`
	Title          string                 `json:"title"`
	Content        string                 `json:"content"`
	Author         string                 `json:"author"`
	AuthorUsername string                 `json:"author_username,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	BookmarkedAt   time.Time              `json:"bookmarked_at"`
	ContentType    string                 `json:"content_type"` // text, link, image, video
	Metadata       map[string]interface{} `json:"metadata"`
	HashSignature  string                 `json:"hash_signature"`
}

// PlatformInfo contains metadata about a platform integration
type PlatformInfo struct {
	Name                string            `json:"name"`                 // Internal name (reddit, twitter, tiktok)
	DisplayName         string            `json:"display_name"`         // User-friendly name
	Icon               string            `json:"icon"`                 // FontAwesome icon class
	Color              string            `json:"color"`                // Brand color hex code
	SupportedFeatures  []string          `json:"supported_features"`   // What this integration can do
	AuthMethod         string            `json:"auth_method"`          // oauth2, session, api_key
	RateLimits         map[string]int    `json:"rate_limits"`          // Rate limiting info
	RequiredCredentials []string          `json:"required_credentials"` // Required config fields
	OptionalCredentials []string          `json:"optional_credentials"` // Optional config fields
	DefaultSyncInterval time.Duration     `json:"default_sync_interval"`
	ContentTypes       []string          `json:"content_types"`        // Types of content supported
	Categories         []string          `json:"categories"`           // Common categories for this platform
	Notes              string            `json:"notes,omitempty"`      // Implementation notes
}

// CategorySuggestion represents a suggested category for a bookmark
type CategorySuggestion struct {
	Category    string  `json:"category"`    // Suggested category name
	Confidence  float64 `json:"confidence"`  // Confidence score (0.0 to 1.0)
	Reason      string  `json:"reason"`      // Human-readable explanation
	Method      string  `json:"method"`      // How this suggestion was generated
	Keywords    []string `json:"keywords,omitempty"` // Keywords that triggered this suggestion
}

// ActionSuggestion represents a suggested action for a bookmark
type ActionSuggestion struct {
	ActionType      string                 `json:"action_type"`      // Type of action (add_to_recipe_book, etc.)
	TargetScenario  string                 `json:"target_scenario"`  // Which scenario to invoke
	Confidence      float64                `json:"confidence"`       // Confidence score (0.0 to 1.0)
	ActionData      map[string]interface{} `json:"action_data"`      // Data to pass to the target scenario
	Description     string                 `json:"description"`      // Human-readable description
	Priority        int                    `json:"priority"`         // Priority order (higher = more important)
	RequiredFields  []string               `json:"required_fields,omitempty"` // Fields needed for execution
}

// PlatformConfig holds configuration for a platform integration
type PlatformConfig struct {
	PlatformName     string                 `json:"platform_name"`
	Enabled         bool                   `json:"enabled"`
	SyncInterval    time.Duration          `json:"sync_interval"`
	Credentials     map[string]string      `json:"credentials"`
	Settings        map[string]interface{} `json:"settings"`
	LastSyncAt      *time.Time             `json:"last_sync_at,omitempty"`
	Status          string                 `json:"status"` // active, inactive, error, rate_limited
	ErrorCount      int                    `json:"error_count"`
	LastError       string                 `json:"last_error,omitempty"`
}

// BookmarkProcessingResult represents the result of processing bookmarks
type BookmarkProcessingResult struct {
	ProcessedCount      int                    `json:"processed_count"`
	SuccessCount        int                    `json:"success_count"`
	ErrorCount          int                    `json:"error_count"`
	DuplicateCount      int                    `json:"duplicate_count"`
	CategorizedCount    int                    `json:"categorized_count"`
	ActionsGenerated    int                    `json:"actions_generated"`
	ProcessingTimeMs    int64                  `json:"processing_time_ms"`
	Errors              []string               `json:"errors,omitempty"`
	Categories          map[string]int         `json:"categories"` // Category -> count
	BookmarkIDs         []string               `json:"bookmark_ids"`
}

// SyncStatus represents the current sync status for a platform
type SyncStatus struct {
	PlatformName    string     `json:"platform_name"`
	Status          string     `json:"status"` // syncing, idle, error
	LastSyncAt      *time.Time `json:"last_sync_at"`
	NextSyncAt      *time.Time `json:"next_sync_at"`
	ItemsProcessed  int        `json:"items_processed"`
	ItemsRemaining  int        `json:"items_remaining"`
	ErrorMessage    string     `json:"error_message,omitempty"`
	ProgressPercent int        `json:"progress_percent"`
}

// ContentAnalysis represents analysis results for bookmark content
type ContentAnalysis struct {
	Language           string            `json:"language"`
	SentimentScore     float64           `json:"sentiment_score"` // -1.0 to 1.0
	KeyPhrases         []string          `json:"key_phrases"`
	Topics             []string          `json:"topics"`
	Entities           []NamedEntity     `json:"entities"`
	ReadingTimeMinutes int               `json:"reading_time_minutes"`
	ComplexityScore    float64           `json:"complexity_score"` // 0.0 to 1.0
	Tags               []string          `json:"tags"`
}

// NamedEntity represents a named entity found in content
type NamedEntity struct {
	Text       string  `json:"text"`
	Type       string  `json:"type"` // PERSON, ORGANIZATION, LOCATION, etc.
	Confidence float64 `json:"confidence"`
}

// PlatformHealth represents the health status of a platform integration
type PlatformHealth struct {
	PlatformName       string     `json:"platform_name"`
	IsHealthy         bool       `json:"is_healthy"`
	LastCheckAt       time.Time  `json:"last_check_at"`
	ResponseTimeMs    int64      `json:"response_time_ms"`
	ErrorRate         float64    `json:"error_rate"` // Percentage of failed requests
	RateLimitStatus   string     `json:"rate_limit_status"` // ok, warning, exceeded
	RemainingRequests int        `json:"remaining_requests"`
	ResetTime         *time.Time `json:"reset_time,omitempty"`
	Issues            []string   `json:"issues,omitempty"`
}

// BookmarkFilter defines filtering options for bookmark queries
type BookmarkFilter struct {
	ProfileID     string     `json:"profile_id,omitempty"`
	Platforms     []string   `json:"platforms,omitempty"`
	Categories    []string   `json:"categories,omitempty"`
	Authors       []string   `json:"authors,omitempty"`
	DateFrom      *time.Time `json:"date_from,omitempty"`
	DateTo        *time.Time `json:"date_to,omitempty"`
	SearchQuery   string     `json:"search_query,omitempty"`
	HasActions    *bool      `json:"has_actions,omitempty"`
	ActionStatus  string     `json:"action_status,omitempty"` // pending, approved, rejected
	MinConfidence *float64   `json:"min_confidence,omitempty"`
	MaxConfidence *float64   `json:"max_confidence,omitempty"`
	ContentTypes  []string   `json:"content_types,omitempty"`
	Tags          []string   `json:"tags,omitempty"`
	Limit         int        `json:"limit,omitempty"`
	Offset        int        `json:"offset,omitempty"`
	SortBy        string     `json:"sort_by,omitempty"`    // created_at, confidence, title
	SortOrder     string     `json:"sort_order,omitempty"` // asc, desc
}

// IntegrationMetrics represents metrics for a platform integration
type IntegrationMetrics struct {
	PlatformName          string    `json:"platform_name"`
	PeriodStart          time.Time `json:"period_start"`
	PeriodEnd            time.Time `json:"period_end"`
	TotalBookmarks       int       `json:"total_bookmarks"`
	NewBookmarks         int       `json:"new_bookmarks"`
	ProcessedBookmarks   int       `json:"processed_bookmarks"`
	FailedBookmarks      int       `json:"failed_bookmarks"`
	AverageConfidence    float64   `json:"average_confidence"`
	CategoryBreakdown    map[string]int `json:"category_breakdown"`
	ActionsGenerated     int       `json:"actions_generated"`
	ActionsApproved      int       `json:"actions_approved"`
	ActionsRejected      int       `json:"actions_rejected"`
	AverageProcessingMs  int64     `json:"average_processing_ms"`
	ErrorRate            float64   `json:"error_rate"`
	UserSatisfaction     float64   `json:"user_satisfaction"` // Based on feedback
}

// Common constants for platform integrations
const (
	// Status constants
	StatusActive      = "active"
	StatusInactive    = "inactive"
	StatusError       = "error"
	StatusRateLimited = "rate_limited"
	StatusSyncing     = "syncing"
	StatusIdle        = "idle"
	
	// Content types
	ContentTypeText  = "text"
	ContentTypeLink  = "link"
	ContentTypeImage = "image"
	ContentTypeVideo = "video"
	ContentTypeAudio = "audio"
	
	// Action types
	ActionAddToRecipeBook    = "add_to_recipe_book"
	ActionScheduleWorkout    = "schedule_workout"
	ActionAddToCodeLibrary   = "add_to_code_library"
	ActionAddToTravelList    = "add_to_travel_list"
	ActionAddToResearch      = "add_to_research"
	ActionCreateReminder     = "create_reminder"
	ActionShareContent       = "share_content"
	ActionArchiveContent     = "archive_content"
	
	// Approval statuses
	ApprovalPending  = "pending"
	ApprovalApproved = "approved"
	ApprovalRejected = "rejected"
	ApprovalExecuted = "executed"
	ApprovalFailed   = "failed"
	
	// Processing statuses
	ProcessingPending   = "pending"
	ProcessingInProgress = "in_progress"
	ProcessingCompleted = "completed"
	ProcessingFailed    = "failed"
)