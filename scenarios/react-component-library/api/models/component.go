package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Component represents a React component in the library
type Component struct {
	ID                  uuid.UUID              `json:"id" db:"id"`
	Name                string                 `json:"name" db:"name" validate:"required,min=1,max=100"`
	Category            string                 `json:"category" db:"category" validate:"required"`
	Description         string                 `json:"description" db:"description" validate:"required,min=10,max=1000"`
	Code                string                 `json:"code" db:"code" validate:"required"`
	PropsSchema         json.RawMessage        `json:"props_schema" db:"props_schema"`
	CreatedAt           time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at" db:"updated_at"`
	Version             string                 `json:"version" db:"version"`
	Author              string                 `json:"author" db:"author"`
	UsageCount          int                    `json:"usage_count" db:"usage_count"`
	AccessibilityScore  *float64               `json:"accessibility_score" db:"accessibility_score"`
	PerformanceMetrics  json.RawMessage        `json:"performance_metrics" db:"performance_metrics"`
	Tags                []string               `json:"tags" db:"tags"`
	IsActive            bool                   `json:"is_active" db:"is_active"`
	Dependencies        []string               `json:"dependencies" db:"dependencies"`
	Screenshots         []string               `json:"screenshots" db:"screenshots"`
	ExampleUsage        string                 `json:"example_usage" db:"example_usage"`
}

// ComponentVersion represents a version of a component
type ComponentVersion struct {
	ID              uuid.UUID   `json:"id" db:"id"`
	ComponentID     uuid.UUID   `json:"component_id" db:"component_id"`
	Version         string      `json:"version" db:"version"`
	Code            string      `json:"code" db:"code"`
	Changelog       string      `json:"changelog" db:"changelog"`
	BreakingChanges []string    `json:"breaking_changes" db:"breaking_changes"`
	Deprecated      bool        `json:"deprecated" db:"deprecated"`
	CreatedAt       time.Time   `json:"created_at" db:"created_at"`
}

// TestResult represents the result of component testing
type TestResult struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	ComponentID  uuid.UUID       `json:"component_id" db:"component_id"`
	TestType     TestType        `json:"test_type" db:"test_type"`
	Results      json.RawMessage `json:"results" db:"results"`
	Passed       bool            `json:"passed" db:"passed"`
	Score        float64         `json:"score" db:"score"`
	TestedAt     time.Time       `json:"tested_at" db:"tested_at"`
	TestDuration int             `json:"test_duration_ms" db:"test_duration_ms"`
}

// TestType represents different types of component tests
type TestType string

const (
	TestTypeAccessibility TestType = "accessibility"
	TestTypePerformance   TestType = "performance"
	TestTypeVisual        TestType = "visual"
	TestTypeUnitTest      TestType = "unit_test"
	TestTypeLinting       TestType = "linting"
)

// UsageAnalytics represents component usage statistics
type UsageAnalytics struct {
	ID          uuid.UUID `json:"id" db:"id"`
	ComponentID uuid.UUID `json:"component_id" db:"component_id"`
	Scenario    string    `json:"scenario" db:"scenario"`
	Context     string    `json:"context" db:"context"`
	UsedAt      time.Time `json:"used_at" db:"used_at"`
	UserAgent   string    `json:"user_agent" db:"user_agent"`
}

// ComponentSearchRequest represents a search request
type ComponentSearchRequest struct {
	Query                 string   `json:"query" form:"query" validate:"required,min=1"`
	Category              string   `json:"category,omitempty" form:"category"`
	Tags                  []string `json:"tags,omitempty" form:"tags"`
	MinAccessibilityScore float64  `json:"min_accessibility_score,omitempty" form:"min_accessibility_score"`
	Limit                 int      `json:"limit,omitempty" form:"limit" validate:"min=1,max=100"`
	Offset                int      `json:"offset,omitempty" form:"offset" validate:"min=0"`
}

// ComponentSearchResponse represents a search response
type ComponentSearchResponse struct {
	Components   []Component `json:"components"`
	Total        int         `json:"total"`
	SearchTimeMs int64       `json:"search_time_ms"`
	Query        string      `json:"query"`
}

// ComponentGenerationRequest represents an AI generation request
type ComponentGenerationRequest struct {
	Description        string            `json:"description" validate:"required,min=10,max=1000"`
	Requirements       []string          `json:"requirements,omitempty"`
	StylePreferences   map[string]string `json:"style_preferences,omitempty"`
	AccessibilityLevel AccessibilityLevel `json:"accessibility_level,omitempty"`
	Category           string            `json:"category,omitempty"`
	Dependencies       []string          `json:"dependencies,omitempty"`
}

// ComponentGenerationResponse represents an AI generation response
type ComponentGenerationResponse struct {
	GeneratedCode string          `json:"generated_code"`
	ComponentName string          `json:"component_name"`
	PropsSchema   json.RawMessage `json:"props_schema"`
	Explanation   string          `json:"explanation"`
	Dependencies  []string        `json:"dependencies"`
	ExampleUsage  string          `json:"example_usage"`
}

// AccessibilityLevel represents WCAG compliance levels
type AccessibilityLevel string

const (
	AccessibilityLevelA   AccessibilityLevel = "A"
	AccessibilityLevelAA  AccessibilityLevel = "AA"
	AccessibilityLevelAAA AccessibilityLevel = "AAA"
)

// ComponentTestRequest represents a test request
type ComponentTestRequest struct {
	TestTypes  []TestType        `json:"test_types" validate:"required,min=1"`
	TestConfig map[string]string `json:"test_config,omitempty"`
}

// ComponentTestResponse represents a test response
type ComponentTestResponse struct {
	TestResults     []TestResult `json:"test_results"`
	OverallScore    float64      `json:"overall_score"`
	Recommendations []string     `json:"recommendations"`
	TestDurationMs  int64        `json:"test_duration_ms"`
}

// ComponentExportRequest represents an export request
type ComponentExportRequest struct {
	Format      ExportFormat `json:"format" validate:"required"`
	IncludeDeps bool         `json:"include_deps,omitempty"`
	Output      string       `json:"output,omitempty"`
}

// ExportFormat represents different export formats
type ExportFormat string

const (
	ExportFormatNPMPackage ExportFormat = "npm-package"
	ExportFormatCDN        ExportFormat = "cdn"
	ExportFormatRawCode    ExportFormat = "raw-code"
	ExportFormatZip        ExportFormat = "zip"
)

// ComponentExportResponse represents an export response
type ComponentExportResponse struct {
	ExportURL        string            `json:"export_url,omitempty"`
	Code             string            `json:"code,omitempty"`
	Dependencies     []string          `json:"dependencies"`
	UsageInstructions string           `json:"usage_instructions"`
	ExportType       ExportFormat      `json:"export_type"`
	FileSize         int64             `json:"file_size,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

// ComponentImprovementRequest represents an improvement request
type ComponentImprovementRequest struct {
	Focus []ImprovementFocus `json:"focus,omitempty" validate:"min=1"`
	Apply bool               `json:"apply,omitempty"`
}

// ImprovementFocus represents areas for improvement
type ImprovementFocus string

const (
	ImprovementFocusAccessibility ImprovementFocus = "accessibility"
	ImprovementFocusPerformance   ImprovementFocus = "performance"
	ImprovementFocusCodeQuality   ImprovementFocus = "code-quality"
	ImprovementFocusSecurity      ImprovementFocus = "security"
)

// ComponentImprovementResponse represents an improvement response
type ComponentImprovementResponse struct {
	Suggestions     []ImprovementSuggestion `json:"suggestions"`
	ImprovedCode    string                  `json:"improved_code,omitempty"`
	Applied         bool                    `json:"applied"`
	EstimatedImpact map[string]float64      `json:"estimated_impact"`
}

// ImprovementSuggestion represents a single improvement suggestion
type ImprovementSuggestion struct {
	Type        ImprovementFocus `json:"type"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	CodeDiff    string           `json:"code_diff,omitempty"`
	Impact      string           `json:"impact"`
	Effort      string           `json:"effort"`
}

// HealthStatus represents the API health status
type HealthStatus struct {
	Status      string            `json:"status"`
	Timestamp   time.Time         `json:"timestamp"`
	Version     string            `json:"version"`
	Database    string            `json:"database"`
	Resources   map[string]string `json:"resources"`
	Uptime      string            `json:"uptime"`
	RequestsTotal int64           `json:"requests_total,omitempty"`
}

// APIMetrics represents API metrics
type APIMetrics struct {
	ComponentCreations    int64             `json:"component_creations"`
	SearchQueries         int64             `json:"search_queries"`
	AccessibilityTests    int64             `json:"accessibility_tests"`
	PerformanceBenchmarks int64             `json:"performance_benchmarks"`
	AIGenerations         int64             `json:"ai_generations"`
	ComponentExports      int64             `json:"component_exports"`
	ResponseTimes         map[string]float64 `json:"avg_response_times_ms"`
	ErrorRates            map[string]float64 `json:"error_rates"`
}

// ComponentCategory represents component categories
type ComponentCategory string

const (
	CategoryForm       ComponentCategory = "form"
	CategoryLayout     ComponentCategory = "layout"
	CategoryDisplay    ComponentCategory = "display"
	CategoryNavigation ComponentCategory = "navigation"
	CategoryFeedback   ComponentCategory = "feedback"
	CategoryDataViz    ComponentCategory = "data-visualization"
	CategoryMedia      ComponentCategory = "media"
	CategoryUtility    ComponentCategory = "utility"
)

// GetValidCategories returns all valid component categories
func GetValidCategories() []ComponentCategory {
	return []ComponentCategory{
		CategoryForm,
		CategoryLayout,
		CategoryDisplay,
		CategoryNavigation,
		CategoryFeedback,
		CategoryDataViz,
		CategoryMedia,
		CategoryUtility,
	}
}