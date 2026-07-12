// Package execute provides shared execution types used across the execute domain.
package execute

import (
	"encoding/json"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// Request represents an execution request.
type Request struct {
	ScenarioName string   `json:"scenarioName"`
	Preset       string   `json:"preset,omitempty"`
	Phases       []string `json:"phases,omitempty"`
	Skip         []string `json:"skip,omitempty"`
	FailFast     bool     `json:"failFast"`

	// DiagnosticsPreset ("none"|"light"|"full") overrides the playbooks
	// diagnostics config for this run (richer BAS artifact capture).
	DiagnosticsPreset string `json:"diagnosticsPreset,omitempty"`

	// Runtime URLs for phases that need to connect to running services.
	// UIURL/APIURL are optional overrides; when omitted, Test Genie manages the
	// target scenario lifecycle and discovers URLs from lifecycle process metadata.
	UIURL  string `json:"uiUrl,omitempty"`
	APIURL string `json:"apiUrl,omitempty"`

	// ScenarioPath is the absolute physical path to the scenario directory that
	// test-genie should read and write. When empty, the API resolves the physical
	// path from ScenarioName.
	ScenarioPath string `json:"scenarioPath,omitempty"`
	// LogicalRepoRoot and LogicalScenarioRelPath describe where the physical
	// scenario should be treated as living for repo-relative validation.
	LogicalRepoRoot        string `json:"logicalRepoRoot,omitempty"`
	LogicalScenarioRelPath string `json:"logicalScenarioRelPath,omitempty"`
}

// PlanPhase represents a selected phase before execution begins.
type PlanPhase struct {
	Name                     string `json:"name"`
	Description              string `json:"description,omitempty"`
	Optional                 bool   `json:"optional"`
	EstimatedDurationSeconds int    `json:"estimatedDurationSeconds"`
	TimeoutSeconds           int    `json:"timeoutSeconds"`
	EstimateSource           string `json:"estimateSource"`
	EstimateConfidence       string `json:"estimateConfidence"`
	EstimateSampleSize       int    `json:"estimateSampleSize"`
}

// PlanSummary provides aggregate timing guidance for the selected plan.
type PlanSummary struct {
	PhaseCount               int `json:"phaseCount"`
	EstimatedDurationSeconds int `json:"estimatedDurationSeconds"`
	TimeoutSeconds           int `json:"timeoutSeconds"`
}

// PlanPreview is the API preflight response for execution planning.
type PlanPreview struct {
	ScenarioName string      `json:"scenarioName"`
	PresetUsed   string      `json:"presetUsed"`
	Phases       []PlanPhase `json:"phases"`
	Summary      PlanSummary `json:"summary"`
	Warnings     []string    `json:"warnings"`
}

// Response represents the execution response.
type Response struct {
	Success        bool           `json:"success"`
	Verdict        string         `json:"verdict"`
	ExecutionID    string         `json:"executionId"`
	PresetUsed     string         `json:"presetUsed"`
	StartedAt      string         `json:"startedAt"`
	CompletedAt    string         `json:"completedAt"`
	PhaseSummary   PhaseSummary   `json:"phaseSummary"`
	Phases         []Phase        `json:"phases"`
	Warnings       []string       `json:"warnings"`
	WarningSummary WarningSummary `json:"warningSummary"`
	Error          string         `json:"error"`
	ErrorMessages  []string       `json:"errors"`
	Links          map[string]any `json:"links"`
	Metadata       map[string]any `json:"metadata"`
	CampaignNudge  *CampaignNudge `json:"campaignNudge,omitempty"`
	// RunHandle carries the durable, server-owned run identity plus the reattach
	// and follow commands, so a --json consumer can reattach to or audit the run
	// from the terminal object alone — without parsing human stderr. Additive:
	// ExecutionID stays the canonical run id for existing consumers. Populated on
	// the durable machine-output path.
	RunHandle *RunHandle `json:"runHandle,omitempty"`
	// Requirements summarizes PRD operational-target and requirement status for
	// the scenario. Present whenever the scenario has a requirements/ tree. Nil
	// otherwise. Mirrors the API's orchestrator.requirements.SyncOutcome.
	Requirements *RequirementsSummary `json:"requirements,omitempty"`
}

// RunStandingView is the curated terminal run payload shared by human and
// machine renderers. The JSON flag serializes this view directly; human output
// renders the same phases, completeness, verdict, and top-priority fields.
type RunStandingView struct {
	Success                       bool                 `json:"success"`
	Verdict                       string               `json:"verdict,omitempty"`
	Status                        string               `json:"status,omitempty"`
	Scenario                      string               `json:"scenario,omitempty"`
	RunID                         string               `json:"runId,omitempty"`
	ExecutionID                   string               `json:"executionId,omitempty"`
	PhaseSummary                  PhaseSummary         `json:"phaseSummary"`
	Phases                        []Phase              `json:"phases"`
	Completeness                  *CompletenessSummary `json:"completeness,omitempty"`
	TopPriority                   *RunTopPriority      `json:"topPriority,omitempty"`
	RunHandle                     *RunHandle           `json:"runHandle,omitempty"`
	RecommendedNextCheckSeconds   int32                `json:"recommendedNextCheckSeconds,omitempty"`
	TimedOut                      bool                 `json:"timedOut,omitempty"`
	Error                         string               `json:"error,omitempty"`
	TerminalSnapshotSchemaVersion int32                `json:"terminalSnapshotSchemaVersion,omitempty"`
	DegradedReasons               []string             `json:"degradedReasons,omitempty"`
}

// RunTopPriority is the single cross-phase next move selected from all phase
// maturity standings. Nil means every standing is at its ceiling or no provider
// emitted a maturity standing.
type RunTopPriority struct {
	Phase                   string   `json:"phase"`
	Provider                string   `json:"provider,omitempty"`
	CurrentLevel            string   `json:"currentLevel,omitempty"`
	CurrentLevelLabel       string   `json:"currentLevelLabel,omitempty"`
	NextLevel               string   `json:"nextLevel,omitempty"`
	NextMove                string   `json:"nextMove"`
	NextMoveReason          string   `json:"nextMoveReason,omitempty"`
	PriorityCapabilityID    string   `json:"priorityCapabilityId,omitempty"`
	PriorityCapabilityLabel string   `json:"priorityCapabilityLabel,omitempty"`
	DocSearchTopic          string   `json:"docSearchTopic,omitempty"`
	BlockingFindingCodes    []string `json:"blockingFindingCodes,omitempty"`
}

// CompletenessSummary is the cached scenario-completeness-scoring supplement
// projected into the shared terminal view.
type CompletenessSummary struct {
	Score           int32                        `json:"score"`
	Classification  string                       `json:"classification,omitempty"`
	WorkingRung     string                       `json:"workingRung,omitempty"`
	LadderClean     bool                         `json:"ladderClean,omitempty"`
	Trend           *CompletenessTrend           `json:"trend,omitempty"`
	Recommendations []CompletenessRecommendation `json:"recommendations,omitempty"`
	StaleEvidence   []string                     `json:"staleEvidence,omitempty"`
	RefreshCommand  string                       `json:"refreshCommand,omitempty"`
}

type CompletenessTrend struct {
	PreviousScore int32  `json:"previousScore"`
	PreviousDate  string `json:"previousDate,omitempty"`
	Delta         int32  `json:"delta"`
}

type CompletenessRecommendation struct {
	Priority     string  `json:"priority,omitempty"`
	Description  string  `json:"description"`
	ImpactPoints float64 `json:"impactPoints,omitempty"`
}

// RequirementsSummary mirrors the API's requirements sync outcome. It is
// rendered in the execute report on every run so PRD operational-target and
// requirement status stays visible regardless of which phases were selected.
type RequirementsSummary struct {
	// Synced is true when these counts were refreshed this run (full suite).
	// False means they are the last persisted values (a partial/gated run).
	Synced bool `json:"synced"`
	// SkipReason explains why sync did not run, when Synced is false.
	SkipReason string `json:"skipReason,omitempty"`
	// LastSyncedAt is the timestamp of the most recent persisted sync (RFC3339).
	LastSyncedAt string `json:"lastSyncedAt,omitempty"`

	OTComplete   int                `json:"otComplete"`
	OTTotal      int                `json:"otTotal"`
	OTByPriority map[string]OTCount `json:"otByPriority,omitempty"`

	ReqComplete int            `json:"reqComplete"`
	ReqTotal    int            `json:"reqTotal"`
	ReqByStatus map[string]int `json:"reqByStatus,omitempty"`

	Changes []RequirementChange `json:"changes,omitempty"`
}

// OTCount is the complete/total tally for one operational-target priority band.
type OTCount struct {
	Complete int `json:"complete"`
	Total    int `json:"total"`
}

// RequirementChange is one requirement-level status transition. Kind is
// "promotion" (… → complete), "regression" (complete → …), or "other".
type RequirementChange struct {
	ID     string `json:"id"`
	PRDRef string `json:"prdRef,omitempty"`
	From   string `json:"from"`
	To     string `json:"to"`
	Kind   string `json:"kind"`
}

// Change-kind constants, mirroring the API.
const (
	ChangeKindPromotion  = "promotion"
	ChangeKindRegression = "regression"
	ChangeKindOther      = "other"
)

// Regressions returns the changes classified as regressions.
func (r *RequirementsSummary) Regressions() []RequirementChange {
	if r == nil {
		return nil
	}
	var out []RequirementChange
	for _, c := range r.Changes {
		if c.Kind == ChangeKindRegression {
			out = append(out, c)
		}
	}
	return out
}

// Promotions returns the changes classified as promotions.
func (r *RequirementsSummary) Promotions() []RequirementChange {
	if r == nil {
		return nil
	}
	var out []RequirementChange
	for _, c := range r.Changes {
		if c.Kind == ChangeKindPromotion {
			out = append(out, c)
		}
	}
	return out
}

// CampaignNudge mirrors the API's campaign-nudge steer. Present only when an
// audit's finding load exceeded the single-pass threshold, recommending a
// tracked improvement campaign over ad-hoc fixing.
type CampaignNudge struct {
	Triggered    bool           `json:"triggered"`
	Total        int            `json:"total"`
	Severe       int            `json:"severe"`
	BySeverity   map[string]int `json:"bySeverity"`
	Reason       string         `json:"reason"`
	ArtifactPath string         `json:"artifactPath"`
	Command      string         `json:"command"`
}

// WarningDetail captures a non-fatal warning emitted by a phase.
type WarningDetail struct {
	Message      string `json:"message"`
	Source       string `json:"source,omitempty"`
	LogPath      string `json:"logPath,omitempty"`
	ArtifactPath string `json:"artifactPath,omitempty"`
}

// PhaseWarningSummary groups warnings by phase.
type PhaseWarningSummary struct {
	Name     string          `json:"name"`
	Count    int             `json:"count"`
	Warnings []WarningDetail `json:"warnings,omitempty"`
}

// WarningSummary aggregates phase warning observations.
type WarningSummary struct {
	Total  int                   `json:"total"`
	Phases []PhaseWarningSummary `json:"phases,omitempty"`
}

// PhaseToggle mirrors the API payload for global phase toggles.
type PhaseToggle struct {
	Disabled bool   `json:"disabled"`
	Reason   string `json:"reason,omitempty"`
	Owner    string `json:"owner,omitempty"`
	AddedAt  string `json:"addedAt,omitempty"`
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
	Name               string          `json:"name"`
	Status             string          `json:"status"`
	DurationSeconds    float64         `json:"durationSeconds"`
	LogPath            string          `json:"logPath"`
	Error              string          `json:"error"`
	Classification     string          `json:"classification"`
	Remediation        string          `json:"remediation"`
	RunnabilityVerdict string          `json:"runnabilityVerdict"`
	RunnabilityReason  string          `json:"runnabilityReason"`
	Observations       ObservationList `json:"observations"`
	// PhasePresentation is the provider-owned canonical phase presentation. Both
	// human and JSON output render this exact object; nil is explicit degraded or
	// native evidence, never an invented maturity claim.
	PhasePresentation *commonv1.PhasePresentation `json:"phasePresentation,omitempty"`
	// PresentationState makes absent canonical presentation explicit. It is
	// "legacy_maturity_standing" only when an historical run carries the retired
	// run-wire object; consumers must not treat that object as v1 presentation.
	PresentationState string `json:"presentationState,omitempty"`
	// FindingsSummary is the per-severity finding tally for the phase.
	FindingsSummary *FindingsSummary `json:"findingsSummary,omitempty"`
}

// FindingsSummary is the per-severity finding tally for a phase.
type FindingsSummary struct {
	Blockers int32 `json:"blockers,omitempty"`
	Errors   int32 `json:"errors,omitempty"`
	Warnings int32 `json:"warnings,omitempty"`
	Infos    int32 `json:"infos,omitempty"`
	Total    int32 `json:"total,omitempty"`
}

// RunHandle is the durable, server-owned run identity a machine consumer needs
// to reattach to, follow, or audit a run without parsing human stderr. It rides
// on the final --json Response and is also emitted as the early start-handle so
// a long --json run is not opaque until completion (see execute.StartHandle).
type RunHandle struct {
	// RunID is the durable run id (also mirrored to Response.ExecutionID).
	RunID string `json:"runId"`
	// Scenario is the scenario the run belongs to.
	Scenario string `json:"scenario"`
	// Reattach is the exact `runs wait --json` command that blocks quietly and
	// exits with the verdict — the agent-safe reattach breadcrumb.
	Reattach string `json:"reattach"`
	// Follow is the `runs follow` command for live human progress.
	Follow string `json:"follow"`
	// Coalesced reports that this request rode an already-in-flight identical run
	// (the one-run-per-scenario guard) instead of starting a second suite.
	Coalesced bool `json:"coalesced,omitempty"`
}

// PhaseSummary provides aggregate phase statistics.
type PhaseSummary struct {
	Total            int `json:"total"`
	Passed           int `json:"passed"`
	Failed           int `json:"failed"`
	Skipped          int `json:"skipped"`
	DurationSeconds  int `json:"durationSeconds"`
	ObservationCount int `json:"observationCount"`
}
