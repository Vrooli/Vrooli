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
	Unit         Name = "unit"
	Integration  Name = "integration"
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
	Name        string `json:"name"`
	Optional    bool   `json:"optional"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source"`
}

// RunReport captures per-phase execution context that a runner returns.
type RunReport struct {
	Err                   error
	Observations          []string
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
	Name            string   `json:"name"`
	Status          string   `json:"status"`
	DurationSeconds int      `json:"durationSeconds"`
	LogPath         string   `json:"logPath"`
	Error           string   `json:"error,omitempty"`
	Classification  string   `json:"classification,omitempty"`
	Remediation     string   `json:"remediation,omitempty"`
	Observations    []string `json:"observations,omitempty"`
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
