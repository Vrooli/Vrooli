// Package execute provides shared execution types used across the execute domain.
package execute

import (
	"encoding/json"
)

// Request represents an execution request.
type Request struct {
	ScenarioName   string   `json:"scenarioName"`
	Preset         string   `json:"preset,omitempty"`
	Phases         []string `json:"phases,omitempty"`
	Skip           []string `json:"skip,omitempty"`
	FailFast       bool     `json:"failFast"`
	SuiteRequestID string   `json:"suiteRequestId,omitempty"`

	// Runtime URLs for phases that need to connect to running services.
	// UIURL is required for Lighthouse audits; if empty, Lighthouse is skipped.
	// BrowserlessURL falls back to BROWSERLESS_URL env var or default if not specified.
	UIURL          string `json:"uiUrl,omitempty"`
	APIURL         string `json:"apiUrl,omitempty"`
	BrowserlessURL string `json:"browserlessUrl,omitempty"`
}

// Response represents the execution response.
type Response struct {
	Success       bool           `json:"success"`
	ExecutionID   string         `json:"executionId"`
	SuiteRequest  string         `json:"suiteRequestId"`
	PresetUsed    string         `json:"presetUsed"`
	StartedAt     string         `json:"startedAt"`
	CompletedAt   string         `json:"completedAt"`
	PhaseSummary  PhaseSummary   `json:"phaseSummary"`
	Phases        []Phase        `json:"phases"`
	Error         string         `json:"error"`
	ErrorMessages []string       `json:"errors"`
	Links         map[string]any `json:"links"`
	Metadata      map[string]any `json:"metadata"`
}

// Observation represents a single test observation with optional rich formatting.
type Observation struct {
	Icon    string `json:"icon,omitempty"`    // Emoji indicator (🔍, 🏗️, 🔗, 🧪, etc.)
	Prefix  string `json:"prefix,omitempty"`  // Status prefix (SUCCESS, WARNING, ERROR)
	Section string `json:"section,omitempty"` // Section header for grouping
	Text    string `json:"text"`              // The actual observation message
}

// String returns the observation as a formatted string for display.
func (o Observation) String() string {
	if o.Section != "" {
		if o.Icon != "" {
			return o.Icon + " " + o.Section
		}
		return o.Section
	}
	if o.Text != "" {
		prefix := ""
		if o.Prefix != "" {
			switch o.Prefix {
			case "SUCCESS":
				prefix = "[SUCCESS] ✅ "
			case "WARNING":
				prefix = "[WARNING] ⚠️ "
			case "ERROR":
				prefix = "[ERROR] ❌ "
			default:
				prefix = "[" + o.Prefix + "] "
			}
		}
		return prefix + o.Text
	}
	return ""
}

// IsSection returns true if this observation is a section header.
func (o Observation) IsSection() bool {
	return o.Section != ""
}

// ObservationList is a slice of observations that can unmarshal from both
// string arrays (legacy) and object arrays (new format).
type ObservationList []Observation

// UnmarshalJSON handles both legacy string arrays and new Observation objects.
func (ol *ObservationList) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as new format (array of objects)
	var observations []Observation
	if err := json.Unmarshal(data, &observations); err == nil {
		*ol = observations
		return nil
	}

	// Fall back to legacy format (array of strings)
	var strings []string
	if err := json.Unmarshal(data, &strings); err != nil {
		return err
	}

	// Convert strings to Observation objects
	*ol = make([]Observation, len(strings))
	for i, s := range strings {
		(*ol)[i] = Observation{Text: s}
	}
	return nil
}

// Phase represents a single execution phase result.
type Phase struct {
	Name            string          `json:"name"`
	Status          string          `json:"status"`
	DurationSeconds float64         `json:"durationSeconds"`
	LogPath         string          `json:"logPath"`
	Error           string          `json:"error"`
	Classification  string          `json:"classification"`
	Remediation     string          `json:"remediation"`
	Observations    ObservationList `json:"observations"`
}

// PhaseSummary provides aggregate phase statistics.
type PhaseSummary struct {
	Total            int `json:"total"`
	Passed           int `json:"passed"`
	Failed           int `json:"failed"`
	DurationSeconds  int `json:"durationSeconds"`
	ObservationCount int `json:"observationCount"`
}
