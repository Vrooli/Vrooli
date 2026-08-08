package phases

import (
	"context"
	"io"
	"time"

	"test-genie/internal/orchestrator/phasekeys"
	"test-genie/internal/orchestrator/phasepolicy"
	"test-genie/internal/orchestrator/runnability"
	"test-genie/internal/orchestrator/workspace"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

// Name identifies a single orchestrator phase.
type Name string

// Canonical phase names implemented by the Go orchestrator.
const (
	Structure    Name = "structure"
	Contracts    Name = "contracts"
	UIHealth     Name = "ui-health"
	API          Name = "api"
	Architecture Name = "architecture"
	Dependencies Name = "dependencies"
	Quality      Name = "quality"
	Docs         Name = "docs"
	Unit         Name = "unit"
	Storage      Name = "storage"
	Workflow     Name = "workflow"
	Business     Name = "business"
	Performance  Name = "performance"
	Tidiness     Name = "tidiness"
	Security     Name = "security"
	Measures     Name = "measures"
	Proto        Name = "proto"
	Branding     Name = "branding"
	Search       Name = "search"
	// ProviderConformance validates scenarios that declare themselves as Test
	// Genie phase providers (.vrooli/test-genie.json); owned by test-genie itself.
	ProviderConformance Name = "provider-conformance"
)

const (
	// DefaultTimeout defines the baseline duration budget for runners unless overridden.
	DefaultTimeout = 15 * time.Minute
)

const (
	FailureClassMisconfiguration  = "misconfiguration"
	FailureClassMissingDependency = "missing_dependency"
	FailureClassTimeout           = "timeout"
	FailureClassMaturityContract  = "maturity_contract"
	FailureClassSystem            = "system"
)

// Descriptor surfaces metadata about registered phases so the UI/CLI can
// describe the orchestration flow from the catalog.
type Descriptor struct {
	Name                  string                        `json:"name"`
	DisplayName           string                        `json:"displayName,omitempty"`
	Optional              bool                          `json:"optional"`
	Description           string                        `json:"description,omitempty"`
	Source                string                        `json:"source"`
	Provider              string                        `json:"provider,omitempty"`
	DefaultTimeoutSeconds int                           `json:"defaultTimeoutSeconds,omitempty"`
	DocPath               string                        `json:"docPath,omitempty"`
	DescriptorPath        string                        `json:"descriptorPath,omitempty"`
	SkipEnvVar            string                        `json:"skipEnvVar,omitempty"`
	Comparable            bool                          `json:"comparable"`
	Advisory              bool                          `json:"advisory,omitempty"`
	ArtifactBacked        bool                          `json:"artifactBacked,omitempty"`
	NonComparable         bool                          `json:"nonComparable,omitempty"`
	Policy                phasepolicy.Policy            `json:"policy,omitempty"`
	Runnability           runnability.PhaseCapabilities `json:"runnability,omitempty"`
	FindingSource         string                        `json:"findingSource,omitempty"`
	ProfileMembership     []string                      `json:"profileMembership,omitempty"`
	FreshnessRequirement  string                        `json:"freshnessRequirement,omitempty"`
	PhaseClass            string                        `json:"phaseClass,omitempty"`
	RuntimeClass          string                        `json:"runtimeClass,omitempty"`
	Concurrency           Concurrency                   `json:"concurrency,omitempty"`
	Dimensions            []string                      `json:"dimensions,omitempty"`
}

// Concurrency describes the provider-owned isolation contract for a phase.
// An empty mode is resolved to exclusive by the scheduler.
type Concurrency struct {
	Mode   string `json:"mode,omitempty"`
	Reason string `json:"reason,omitempty"`
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
	// Findings carries phase findings normalized into the
	// shared ArchitectureFinding contract. Observations remain the human
	// view; Findings is the machine seam the cartographer campaign
	// tracker ingests and reconciles by stable ID. Pointers (not values)
	// because proto messages embed a no-copy MessageState.
	Findings []*architecturev1.ArchitectureFinding
	// Assessment preserves the provider-owned maturity contract, including
	// descriptor-owned recommended skill IDs, for phase evidence consumers.
	Assessment *commonv1.MaturityAssessment
	// Metrics carries the delegated provider's execution metrics when present.
	// nil for non-delegated phases and for providers that have not adopted the
	// metrics contract.
	Metrics *commonv1.ExecutionMetrics
	// FindingSource is the lower-case source token for the phase's finding
	// channel. This mirrors ExecutionResult.FindingSource and lets phase pointer
	// artifacts retain covered-source information.
	FindingSource string
	// PhasePresentation is the compact per-phase maturity standing projected from
	// the delegated provider's MaturityAssessment (Phase Capability Contract). nil
	// for native phases and providers that declare no ladder.
	PhasePresentation *commonv1.PhasePresentation
	// FindingsSummary is the per-severity finding tally for the phase (non-nil
	// whenever a delegated provider returned an assessment).
	FindingsSummary *runspb.PhaseFindingsSummary
}

// Runner is the function signature every catalog phase must satisfy.
type Runner func(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport

// Definition is the normalized runner metadata used during plan selection.
type Definition struct {
	Name     Name
	Runner   Runner
	Timeout  time.Duration
	Optional bool
	// DisplayName is descriptor-owned presentation metadata. Name remains the
	// stable machine key used in API payloads, run records, and commands.
	DisplayName string
	// ProviderScenario is set for delegated provider-backed phases. Empty means
	// the phase has no external provider readiness work.
	ProviderScenario string
	// Policy is the explicit internal replacement for the overloaded Optional
	// flag. Optional remains on external descriptors as a legacy projection.
	Policy phasepolicy.Policy
	// SkipEnvVar is the catalog-owned environment switch that disables this
	// phase during selection. Runners must not inspect it.
	SkipEnvVar string
	// Capabilities is the phase's runnability contract (surfaces, lifecycle
	// mutation, DB isolation, resources). Sourced from the catalog Spec; the
	// runnability gate reads it to decide RUN/RUN_DEGRADED/SKIP.
	Capabilities runnability.PhaseCapabilities
	// FindingSource is the architecture-finding channel this phase emits into
	// (FINDING_SOURCE_UNSPECIFIED for phases that produce no findings, e.g.
	// performance). Carried from the catalog Spec so the orchestrator
	// can stamp the per-phase findingSource token onto each ExecutionResult.
	FindingSource        architecturev1.FindingSource
	ProfileMembership    []string
	FreshnessRequirement string
	PhaseClass           string
	RuntimeClass         string
	Concurrency          Concurrency
	Dimensions           []string
}

// Spec captures metadata for a catalog entry.
type Spec struct {
	Name Name
	// DisplayName is presentation metadata sourced from provider descriptors.
	// Name remains the stable machine key.
	DisplayName    string
	Runner         Runner
	Optional       bool
	DefaultTimeout time.Duration
	// SkipEnvVar is the externally-observable environment switch that disables
	// this phase before runnability or execution. Empty values are derived from
	// the phase name at catalog registration.
	SkipEnvVar  string
	Description string
	Source      string
	// Doc is the repo-relative documentation path for the phase. When empty at
	// registration it is auto-derived by convention, keeping doc lookups in
	// lockstep with the catalog instead of a separate hand-maintained map.
	Doc string
	// Capabilities is the phase's runnability contract. Register normalizes the
	// embedded Phase/Optional fields so every catalog entry carries a complete
	// manifest; the anti-drift guard asserts surface-bearing phases declare one.
	Capabilities runnability.PhaseCapabilities
	// Policy is the explicit internal replacement for the overloaded Optional
	// flag. Descriptor-backed registry construction should populate it directly;
	// legacy catalog registration derives it from Optional/Advisory.
	Policy phasepolicy.Policy
	// FindingSource is the architecture-finding channel this phase emits into.
	// Leave UNSPECIFIED for phases that produce no findings (performance).
	// The orchestrator
	// stamps the lower-case token onto each ExecutionResult so a downstream
	// campaign reaudit can derive which sources a partial run actually covered.
	FindingSource        architecturev1.FindingSource
	ProfileMembership    []string
	FreshnessRequirement string
	PhaseClass           string
	RuntimeClass         string
	Concurrency          Concurrency
	Dimensions           []string
	// NonComparable opts a phase out of baseline/run comparison. The default is
	// comparable; catalog entries opt out only when their result cannot produce
	// a meaningful phase verdict.
	NonComparable bool
	Advisory      bool
	// ArtifactBacked marks phases whose primary comparison channel is artifact
	// metadata or a dedicated analyzer rather than phase pass/fail status.
	ArtifactBacked bool
	// Delegated is present when the phase delegates to another scenario through
	// ScenarioValidationService. It is catalog-owned metadata, not a second
	// provider registry.
	Delegated *Delegated
}

// ToDefinition projects a catalog Spec into the runner Definition consumed
// during plan selection. The three phase-metadata layers intentionally share
// field names — Spec is the catalog's registration record, Definition its
// selection-time projection, and Descriptor (see Catalog.Descriptors) its
// serialized view — so this single converter is the only place Spec→Definition
// fields are copied, keeping the layers from drifting.
func (s Spec) ToDefinition() Definition {
	def := Definition{
		Name:                 s.Name,
		Runner:               s.Runner,
		Timeout:              s.DefaultTimeout,
		Optional:             s.Optional,
		DisplayName:          s.DisplayName,
		Policy:               s.Policy,
		SkipEnvVar:           s.SkipEnvVar,
		Capabilities:         s.Capabilities,
		FindingSource:        s.FindingSource,
		ProfileMembership:    append([]string(nil), s.ProfileMembership...),
		FreshnessRequirement: s.FreshnessRequirement,
		PhaseClass:           s.PhaseClass,
		RuntimeClass:         s.RuntimeClass,
		Concurrency:          s.Concurrency,
		Dimensions:           append([]string(nil), s.Dimensions...),
	}
	if s.Delegated != nil {
		def.ProviderScenario = s.Delegated.ProviderScenario
	}
	return def
}

// ExecutionResult captures per-phase outcome information.
type ExecutionResult struct {
	Name                          string `json:"name"`
	Status                        string `json:"status"`
	DurationSeconds               int    `json:"durationSeconds"`
	DurationMilliseconds          int64  `json:"durationMilliseconds,omitempty"`
	PredictedDurationMilliseconds int64  `json:"predictedDurationMilliseconds,omitempty"`
	LogPath                       string `json:"logPath"`
	Error                         string `json:"error,omitempty"`
	Classification                string `json:"classification,omitempty"`
	Remediation                   string `json:"remediation,omitempty"`
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
	// Assessment is the unchanged provider maturity response for this phase.
	Assessment *commonv1.MaturityAssessment `json:"assessment,omitempty"`
	// Metrics is the delegated provider's reported execution metrics (timing,
	// stages, resources, host environment), persisted into immutable per-run
	// phase evidence and a fixed-width SQLite rollup. Absent for phases whose
	// provider has not adopted the contract.
	Metrics *commonv1.ExecutionMetrics `json:"metrics,omitempty"`
	// PhasePresentation is the compact per-phase maturity standing (Phase
	// Capability Contract) projected from the provider's MaturityAssessment. It is
	// carried into the phase-completed run event, the terminal response, and the
	// findings.json artifact so the human scorecard and --json output derive from
	// one server payload. nil for native phases / providers with no ladder.
	PhasePresentation *commonv1.PhasePresentation `json:"phasePresentation,omitempty"`
	// FindingsSummary is the per-severity finding tally for the phase.
	FindingsSummary *runspb.PhaseFindingsSummary `json:"findingsSummary,omitempty"`
}

// NormalizeName standardizes arbitrary input into a canonical Name.
func NormalizeName(raw string) (Name, bool) {
	normalized := Name(NormalizeKey(raw))
	if normalized == "" {
		return "", false
	}
	return normalized, true
}

// NormalizeKey standardizes arbitrary input into a canonical phase key.
func NormalizeKey(raw string) string {
	return phasekeys.NormalizeKey(raw)
}

// String returns the canonical lowercase phase name.
func (n Name) String() string {
	return string(n)
}

// Key returns a safe map key for the phase.
func (n Name) Key() string {
	return NormalizeKey(n.String())
}

// IsZero reports whether the name is empty.
func (n Name) IsZero() bool {
	return n == ""
}
