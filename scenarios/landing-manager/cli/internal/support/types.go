package support

import (
	"encoding/json"
	"time"
)

// Template mirrors the shape returned by /api/v1/templates.
type Template struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Category    string          `json:"category,omitempty"`
	Version     string          `json:"version,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
	Extra       json.RawMessage `json:"-"`
}

// Persona mirrors the shape returned by /api/v1/personas.
type Persona struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	UseCases    []string `json:"use_cases,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Prompt      string   `json:"prompt,omitempty"`
}

// GeneratedScenario is an entry returned by /api/v1/generated.
type GeneratedScenario struct {
	ID         string     `json:"id,omitempty"`
	ScenarioID string     `json:"scenario_id,omitempty"`
	Name       string     `json:"name,omitempty"`
	Slug       string     `json:"slug,omitempty"`
	TemplateID string     `json:"template_id,omitempty"`
	Status     string     `json:"status,omitempty"`
	CreatedAt  *time.Time `json:"created_at,omitempty"`
	Path       string     `json:"path,omitempty"`
}

// PreviewLinks mirrors the response from /api/v1/preview/{scenario_id}.
type PreviewLinks struct {
	ScenarioID   string            `json:"scenario_id"`
	Path         string            `json:"path,omitempty"`
	BaseURL      string            `json:"base_url,omitempty"`
	Links        map[string]string `json:"links,omitempty"`
	Instructions []string          `json:"instructions,omitempty"`
	Notes        string            `json:"notes,omitempty"`
}
