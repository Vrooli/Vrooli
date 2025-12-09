package phases

import (
	"context"
	"io"
	"strings"
	"time"

	"test-genie/internal/orchestrator/workspace"
)

// Name identifies a single orchestrator phase.
type Name string

// Canonical phase names implemented by the Go orchestrator.
const (
	Structure    Name = "structure"
	Dependencies Name = "dependencies"
	Lint         Name = "lint"
	Docs         Name = "docs"
	Smoke        Name = "smoke"
	Unit         Name = "unit"
	Integration  Name = "integration"
	Playbooks    Name = "playbooks"
	Business     Name = "business"
	Performance  Name = "performance"
)

const (
	// DefaultTimeout defines the baseline duration budget for runners unless overridden.
	DefaultTimeout = 15 * time.Minute
)

const (
	FailureClassMisconfiguration  = "misconfiguration"
	FailureClassMissingDependency = "missing_dependency"
	FailureClassTimeout           = "timeout"
	FailureClassSystem            = "system"
)

// Descriptor surfaces metadata about registered phases so the UI/CLI can
// describe the orchestration flow without scraping bash scripts.
type Descriptor struct {
	Name                  string `json:"name"`
	Optional              bool   `json:"optional"`
	Description           string `json:"description,omitempty"`
	Source                string `json:"source"`
	DefaultTimeoutSeconds int    `json:"defaultTimeoutSeconds,omitempty"`
}

// Observation represents a single test observation with optional rich formatting.
// When marshaled to JSON, if only Text is set, it produces a simple string for backwards compat.
type Observation struct {
	Icon    string `json:"icon,omitempty"`    // Emoji indicator (🔍, 🏗️, 🔗, 🧪, etc.)
	Prefix  string `json:"prefix,omitempty"`  // Status prefix (SUCCESS, WARNING, ERROR)
	Section string `json:"section,omitempty"` // Section header for grouping
	Text    string `json:"text"`              // The actual observation message
}

// NewObservation creates a simple text observation.
func NewObservation(text string) Observation {
	return Observation{Text: text}
}

// NewSectionObservation creates a section header observation.
func NewSectionObservation(icon, section string) Observation {
	return Observation{Icon: icon, Section: section}
}

// NewSuccessObservation creates a success observation.
func NewSuccessObservation(text string) Observation {
	return Observation{Prefix: "SUCCESS", Text: text}
}

// NewWarningObservation creates a warning observation.
func NewWarningObservation(text string) Observation {
	return Observation{Prefix: "WARNING", Text: text}
}

// NewSkipObservation creates a skip observation (not a failure, just skipped).
func NewSkipObservation(text string) Observation {
	return Observation{Prefix: "SKIP", Text: text}
}

// NewInfoObservation creates an informational observation (not success/warning/error).
func NewInfoObservation(text string) Observation {
	return Observation{Prefix: "INFO", Text: text}
}

// NewErrorObservation creates an error observation.
func NewErrorObservation(text string) Observation {
	return Observation{Prefix: "ERROR", Text: text}
}

// String returns the observation as a formatted string for logging.
func (o Observation) String() string {
	var parts []string
	if o.Section != "" {
		if o.Icon != "" {
			parts = append(parts, o.Icon+" "+o.Section)
		} else {
			parts = append(parts, o.Section)
		}
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
			case "SKIP":
				prefix = "[SKIP] ⏭️ "
			case "INFO":
				prefix = "[INFO] ℹ️ "
			default:
				prefix = "[" + o.Prefix + "] "
			}
		}
		parts = append(parts, prefix+o.Text)
	}
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

// StringsToObservations converts a slice of strings to observations.
// This is a convenience function for phases that don't need rich formatting.
func StringsToObservations(strs []string) []Observation {
	obs := make([]Observation, len(strs))
	for i, s := range strs {
		obs[i] = NewObservation(s)
	}
	return obs
}

// ObservationsToStrings converts observations to strings for backwards compatibility.
func ObservationsToStrings(obs []Observation) []string {
	strs := make([]string, len(obs))
	for i, o := range obs {
		strs[i] = o.String()
	}
	return strs
}

// RunReport captures per-phase execution context that a runner returns.
type RunReport struct {
	Err                   error
	Observations          []Observation
	FailureClassification string
	Remediation           string
}

// Runner is the function signature every Go-native phase must satisfy.
type Runner func(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport

// Definition is the normalized runner metadata used during plan selection.
type Definition struct {
	Name     Name
	Runner   Runner
	Timeout  time.Duration
	Optional bool
}

// Spec captures metadata for a catalog entry.
type Spec struct {
	Name           Name
	Runner         Runner
	Optional       bool
	DefaultTimeout time.Duration
	Weight         int
	Description    string
	Source         string
}

// ExecutionResult captures per-phase outcome information.
type ExecutionResult struct {
	Name            string        `json:"name"`
	Status          string        `json:"status"`
	DurationSeconds int           `json:"durationSeconds"`
	LogPath         string        `json:"logPath"`
	Error           string        `json:"error,omitempty"`
	Classification  string        `json:"classification,omitempty"`
	Remediation     string        `json:"remediation,omitempty"`
	Observations    []Observation `json:"observations,omitempty"`
}

// NormalizeName standardizes arbitrary input into a canonical Name.
func NormalizeName(raw string) (Name, bool) {
	normalized := Name(strings.ToLower(strings.TrimSpace(raw)))
	if normalized == "" {
		return "", false
	}
	return normalized, true
}

// String returns the canonical lowercase phase name.
func (n Name) String() string {
	return string(n)
}

// Key returns a safe map key for the phase.
func (n Name) Key() string {
	return strings.ToLower(strings.TrimSpace(n.String()))
}

// IsZero reports whether the name is empty.
func (n Name) IsZero() bool {
	return n == ""
}
