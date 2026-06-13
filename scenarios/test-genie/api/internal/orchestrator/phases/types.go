package phases

import (
	"context"
	"io"
	"strings"
	"time"

	"test-genie/internal/orchestrator/runnability"
	"test-genie/internal/orchestrator/workspace"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// Name identifies a single orchestrator phase.
type Name string

// Canonical phase names implemented by the Go orchestrator.
const (
	Structure    Name = "structure"
	Contracts    Name = "contracts"
	UIHealth     Name = "ui-health"
	Standards    Name = "standards"
	Architecture Name = "architecture"
	Dependencies Name = "dependencies"
	Lint         Name = "lint"
	Docs         Name = "docs"
	Smoke        Name = "smoke"
	Unit         Name = "unit"
	Integration  Name = "integration"
	Playbooks    Name = "playbooks"
	Business     Name = "business"
	Performance  Name = "performance"
	Coverage     Name = "coverage"
	Tidiness     Name = "tidiness"
	Security     Name = "security"
	Measures     Name = "measures"
	Proto        Name = "proto"
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
	DocPath               string `json:"docPath,omitempty"`
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
	// Findings carries the phase's native findings normalized into the
	// shared ArchitectureFinding contract. Observations remain the human
	// view; Findings is the machine seam the cartographer campaign
	// tracker ingests and reconciles by stable ID. Pointers (not values)
	// because proto messages embed a no-copy MessageState.
	Findings []*architecturev1.ArchitectureFinding
}

// Runner is the function signature every Go-native phase must satisfy.
type Runner func(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport

// Definition is the normalized runner metadata used during plan selection.
type Definition struct {
	Name     Name
	Runner   Runner
	Timeout  time.Duration
	Optional bool
	// Capabilities is the phase's runnability contract (surfaces, lifecycle
	// mutation, DB isolation, resources). Sourced from the catalog Spec; the
	// runnability gate reads it to decide RUN/RUN_DEGRADED/SKIP.
	Capabilities runnability.PhaseCapabilities
	// FindingSource is the architecture-finding channel this phase emits into
	// (FINDING_SOURCE_UNSPECIFIED for phases that produce no findings, e.g.
	// unit/integration/lint). Carried from the catalog Spec so the orchestrator
	// can stamp the per-phase findingSource token onto each ExecutionResult.
	FindingSource architecturev1.FindingSource
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
	// Doc is the repo-relative documentation path for the phase. When empty at
	// registration it is auto-derived by convention, keeping doc lookups in
	// lockstep with the catalog instead of a separate hand-maintained map.
	Doc string
	// Capabilities is the phase's runnability contract. Register normalizes the
	// embedded Phase/Optional fields so every catalog entry carries a complete
	// manifest; the anti-drift guard asserts surface-bearing phases declare one.
	Capabilities runnability.PhaseCapabilities
	// FindingSource is the architecture-finding channel this phase emits into.
	// Leave UNSPECIFIED for phases that produce no findings (unit, integration,
	// lint, smoke, performance, playbooks, dependencies). The orchestrator
	// stamps the lower-case token onto each ExecutionResult so a downstream
	// campaign reaudit can derive which sources a partial run actually covered.
	FindingSource architecturev1.FindingSource
}

// ExecutionResult captures per-phase outcome information.
type ExecutionResult struct {
	Name            string `json:"name"`
	Status          string `json:"status"`
	DurationSeconds int    `json:"durationSeconds"`
	LogPath         string `json:"logPath"`
	Error           string `json:"error,omitempty"`
	Classification  string `json:"classification,omitempty"`
	Remediation     string `json:"remediation,omitempty"`
	// RunnabilityVerdict records the runnability gate's decision for this phase
	// ("run", "run_degraded", or "skip") and RunnabilityReason its rationale.
	// For a skipped phase these explain why it could not run in this
	// environment; for a degraded run they note the less-preferred path taken.
	RunnabilityVerdict string        `json:"runnabilityVerdict,omitempty"`
	RunnabilityReason  string        `json:"runnabilityReason,omitempty"`
	Observations       []Observation `json:"observations,omitempty"`
	// FindingSource is the lower-case source token (findingid vocabulary) for
	// the channel this phase emits into; empty for phases that produce no
	// findings. Its presence even on a zero-finding phase is what lets a
	// campaign reaudit know the source WAS covered by this run.
	FindingSource string `json:"findingSource,omitempty"`
	// Findings is the normalized, machine-ingestable finding set for this
	// phase (see RunReport.Findings). Serialized in the suite `--json`
	// report so `architecture-cartographer campaign create --from-audit`
	// can ingest it. Enum fields marshal as their proto integer values —
	// a stable seam since both sides share this contract.
	Findings []*architecturev1.ArchitectureFinding `json:"findings,omitempty"`
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
