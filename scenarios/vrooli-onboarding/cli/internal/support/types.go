package support

import (
	"encoding/json"
	"time"
)

// Resource is the shape returned by /api/v1/resources.
type Resource struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Category  string `json:"category"`
	Installed bool   `json:"installed"`
}

// ResourceList wraps /api/v1/resources.
type ResourceList struct {
	Resources []Resource `json:"resources"`
	Count     int        `json:"count"`
	LoadedAt  string     `json:"loaded_at"`
}

// ResourceHealthStatus is one entry in /api/v1/resources/health.
type ResourceHealthStatus struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Category    string `json:"category"`
	Available   bool   `json:"available"`
	LastChecked string `json:"last_checked"`
}

// ResourceHealthResponse is the envelope for /api/v1/resources/health.
type ResourceHealthResponse struct {
	Resources    []ResourceHealthStatus `json:"resources"`
	Total        int                    `json:"total"`
	HealthyCount int                    `json:"healthy_count"`
	CheckedAt    string                 `json:"checked_at"`
}

// OnboardingProgress mirrors the API progress row.
type OnboardingProgress struct {
	ID             int             `json:"id"`
	UserID         string          `json:"user_id"`
	CurrentStep    int             `json:"current_step"`
	CompletedSteps json.RawMessage `json:"completed_steps"`
	ConfigData     json.RawMessage `json:"config_data"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// CompleteResponse is the envelope for POST /api/v1/complete.
type CompleteResponse struct {
	Status      string `json:"status"`
	UserID      string `json:"user_id"`
	CompletedAt string `json:"completed_at"`
	ConfigPath  string `json:"config_path"`
}

// ConfigResourceEntry is one entry inside a generated service.json snippet.
type ConfigResourceEntry struct {
	Enabled bool   `json:"enabled"`
	Name    string `json:"name"`
}

// GeneratedConfigSnippet is the snippet returned by /api/v1/config/generate.
type GeneratedConfigSnippet struct {
	Resources map[string]ConfigResourceEntry `json:"resources"`
}

// ConfigGenerateResponse wraps /api/v1/config/generate.
type ConfigGenerateResponse struct {
	Config   GeneratedConfigSnippet `json:"config"`
	Warnings []string               `json:"warnings,omitempty"`
}

// ValidationResult is one per-resource validation outcome.
type ValidationResult struct {
	Resource string   `json:"resource"`
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// ConfigValidateResponse wraps /api/v1/config/validate.
type ConfigValidateResponse struct {
	Valid   bool               `json:"valid"`
	Results []ValidationResult `json:"results"`
}

// OrderedResource is one entry from /api/v1/setup-order.
type OrderedResource struct {
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	Order        int      `json:"order"`
	Dependencies []string `json:"dependencies"`
}

// SetupOrderResponse wraps /api/v1/setup-order.
type SetupOrderResponse struct {
	SetupOrder []OrderedResource `json:"setup_order"`
	Total      int               `json:"total"`
}

// GlossaryEntry is a single glossary term.
type GlossaryEntry struct {
	Term        string `json:"term"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// GlossaryResponse wraps /api/v1/glossary.
type GlossaryResponse struct {
	Entries []GlossaryEntry `json:"entries"`
	Count   int             `json:"count"`
	Query   string          `json:"query,omitempty"`
}
