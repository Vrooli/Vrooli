package evidence

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"scenario-to-desktop-api/procmetrics"
)

// ManifestSchemaVersion is incremented when the JSON contract changes in an
// incompatible way. The manifest is producer-owned; deployment-manager only
// receives the references represented by the shared protobuf contract.
const ManifestSchemaVersion = 1

type Profile string

const (
	ProfileProtocol      Profile = "protocol"
	ProfileVisual        Profile = "visual"
	ProfileReleaseVisual Profile = "release_visual"
)

type RunState string

const (
	StateCreated            RunState = "created"
	StateProtocolReady      RunState = "protocol_ready"
	StateVisualLaunched     RunState = "visual_launched"
	StateJourneyPassed      RunState = "journey_passed"
	StateCaptureIntegrity   RunState = "capture_integrity_passed"
	StateArtifactsPersisted RunState = "artifacts_persisted"
	StateGovernanceReported RunState = "governance_reported"
	StatePassed             RunState = "passed"
	StateFailed             RunState = "failed"
	StateDegraded           RunState = "degraded"
	StateUnavailable        RunState = "unavailable"
)

type GateName string

const (
	GateProtocol    GateName = "protocol_readiness"
	GateVisual      GateName = "visual_launch"
	GateJourney     GateName = "semantic_journey"
	GateCapture     GateName = "capture_integrity"
	GatePersistence GateName = "artifact_persistence"
	GateGovernance  GateName = "governance_reporting"
)

type GateDisposition string

const (
	GatePassed      GateDisposition = "passed"
	GateFailed      GateDisposition = "failed"
	GateDegraded    GateDisposition = "degraded"
	GateUnavailable GateDisposition = "unavailable"
	GateUnverified  GateDisposition = "unverified"
)

type GateResult struct {
	Name        GateName        `json:"name"`
	Disposition GateDisposition `json:"disposition"`
	Required    bool            `json:"required"`
	Reason      string          `json:"reason,omitempty"`
	StartedAt   time.Time       `json:"started_at"`
	CompletedAt time.Time       `json:"completed_at"`
}

type Artifact struct {
	ImmutableRef string    `json:"immutable_ref"`
	LocalPath    string    `json:"local_path,omitempty"`
	Kind         string    `json:"kind"`
	Checksum     string    `json:"checksum"`
	SizeBytes    int64     `json:"size_bytes"`
	Width        int       `json:"width,omitempty"`
	Height       int       `json:"height,omitempty"`
	DurationMs   int64     `json:"duration_ms,omitempty"`
	Container    string    `json:"container,omitempty"`
	Codec        string    `json:"codec,omitempty"`
	UsefulFrames bool      `json:"useful_frames"`
	CreatedAt    time.Time `json:"created_at"`
}

type Target struct {
	ID         string `json:"id,omitempty"`
	Ramp       string `json:"ramp"`
	Platform   string `json:"platform"`
	OS         string `json:"os"`
	DeviceKind string `json:"device_kind"`
}

type WorkflowArtifactReference struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	URI       string `json:"uri"`
	MediaType string `json:"media_type,omitempty"`
	Checksum  string `json:"checksum"`
	Redacted  bool   `json:"redacted"`
}

// WorkflowReference is provider-neutral evidence linkage. It intentionally
// carries references, not provider-specific execution clients or semantics.
type WorkflowReference struct {
	Provider       string                      `json:"provider"`
	AssetID        string                      `json:"asset_id"`
	ExecutionID    string                      `json:"execution_id"`
	RunID          string                      `json:"run_id"`
	ArtifactDigest string                      `json:"artifact_digest"`
	TargetID       string                      `json:"target_id"`
	CellID         string                      `json:"cell_id"`
	Disposition    string                      `json:"disposition"`
	Artifacts      []WorkflowArtifactReference `json:"artifacts,omitempty"`
}

type Runner struct {
	ID           string   `json:"id"`
	Kind         string   `json:"kind"`
	HostOS       string   `json:"host_os"`
	TargetOS     string   `json:"target_os"`
	Isolation    string   `json:"isolation"`
	Capabilities []string `json:"capabilities,omitempty"`
}

type Provenance struct {
	ArtifactDigest string    `json:"artifact_digest"`
	GitCommit      string    `json:"git_commit"`
	StartedAt      time.Time `json:"started_at"`
	CompletedAt    time.Time `json:"completed_at"`
}

type TimelineSummary struct {
	Version          string             `json:"version"`
	JourneyRef       string             `json:"journey_ref"`
	Capability       string             `json:"capability,omitempty"`
	ProviderTier     string             `json:"provider_tier,omitempty"`
	SafeRouteClass   string             `json:"safe_route_class,omitempty"`
	FallbackDecision string             `json:"fallback_decision,omitempty"`
	ChapterIDs       []string           `json:"chapter_ids"`
	EventCount       int                `json:"event_count"`
	Ordered          bool               `json:"ordered"`
	RedactionStatus  string             `json:"redaction_status"`
	WorkflowRequired bool               `json:"workflow_required,omitempty"`
	Workflow         *WorkflowReference `json:"workflow,omitempty"`
}

// PerformanceEvidence is independent from capability gates. Missing data is
// explicit and never becomes a zero-valued timing or resource measurement.
type PerformanceEvidence struct {
	Status          string                         `json:"status"`
	Reason          string                         `json:"reason,omitempty"`
	ProtocolSummary *procmetrics.Summary           `json:"protocol_summary,omitempty"`
	DemoSummary     *procmetrics.Summary           `json:"demo_summary,omitempty"`
	DemoProcessTree *procmetrics.ProcessTreeReport `json:"demo_process_tree,omitempty"`
	ProtocolPhases  []PhaseDuration                `json:"protocol_phases,omitempty"`
	DemoPhases      []PhaseDuration                `json:"demo_phases,omitempty"`
	TraceRefs       []string                       `json:"trace_refs,omitempty"`
	ProfileRefs     []string                       `json:"profile_refs,omitempty"`
}

// Manifest is the producer-owned, reviewable evidence record. Paths are
// intentionally retained for local operator evidence, while remote consumers
// use ImmutableRef and checksum instead of reading local files.
type Manifest struct {
	SchemaVersion int                 `json:"schema_version"`
	RunID         string              `json:"run_id"`
	Profile       Profile             `json:"profile"`
	State         RunState            `json:"state"`
	Target        Target              `json:"target"`
	CellID        string              `json:"cell_id,omitempty"`
	Runner        Runner              `json:"runner"`
	Provenance    Provenance          `json:"provenance"`
	Timeline      TimelineSummary     `json:"timeline"`
	Gates         []GateResult        `json:"gates"`
	Artifacts     []Artifact          `json:"artifacts"`
	Performance   PerformanceEvidence `json:"performance"`
}

func validateWorkflowReference(ref WorkflowReference, runID, artifactDigest, targetID, cellID string) error {
	for name, value := range map[string]string{
		"provider": ref.Provider, "asset_id": ref.AssetID, "execution_id": ref.ExecutionID,
		"run_id": ref.RunID, "artifact_digest": ref.ArtifactDigest, "target_id": ref.TargetID,
		"cell_id": ref.CellID, "disposition": ref.Disposition,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("workflow reference %s is required", name)
		}
	}
	if ref.RunID != runID || ref.ArtifactDigest != artifactDigest || ref.TargetID != targetID || ref.CellID != cellID {
		return fmt.Errorf("workflow reference is not bound to the desktop validation identity")
	}
	if strings.EqualFold(ref.Disposition, "pass") || strings.EqualFold(ref.Disposition, "passed") {
		if len(ref.Artifacts) == 0 {
			return fmt.Errorf("passing workflow reference requires artifacts")
		}
		for index, artifact := range ref.Artifacts {
			if strings.TrimSpace(artifact.ID) == "" || strings.TrimSpace(artifact.Kind) == "" || strings.TrimSpace(artifact.URI) == "" || strings.TrimSpace(artifact.Checksum) == "" || !artifact.Redacted {
				return fmt.Errorf("workflow artifact %d is missing identity or checksum", index)
			}
		}
	}
	return nil
}

var transitionTable = map[RunState][]RunState{
	StateCreated:            {StateProtocolReady, StateFailed, StateUnavailable},
	StateProtocolReady:      {StateVisualLaunched, StateFailed, StateDegraded, StateUnavailable},
	StateVisualLaunched:     {StateJourneyPassed, StateFailed, StateDegraded, StateUnavailable},
	StateJourneyPassed:      {StateCaptureIntegrity, StateFailed, StateDegraded},
	StateCaptureIntegrity:   {StateArtifactsPersisted, StateFailed, StateDegraded},
	StateArtifactsPersisted: {StateGovernanceReported, StatePassed, StateFailed, StateDegraded},
	StateGovernanceReported: {StatePassed, StateFailed, StateDegraded},
	StatePassed:             {},
	StateFailed:             {},
	StateDegraded:           {},
	StateUnavailable:        {},
}

func (s RunState) CanTransitionTo(next RunState) bool {
	for _, candidate := range transitionTable[s] {
		if candidate == next {
			return true
		}
	}
	return false
}

func (s RunState) IsTerminal() bool {
	_, ok := transitionTable[s]
	return ok && len(transitionTable[s]) == 0
}

func RequiredGates(profile Profile) []GateName {
	if profile == ProfileProtocol {
		return []GateName{GateProtocol}
	}
	result := []GateName{GateProtocol, GateVisual, GateJourney, GateCapture, GatePersistence}
	if profile == ProfileReleaseVisual {
		result = append(result, GateGovernance)
	}
	return result
}

func (m Manifest) MarshalJSON() ([]byte, error) {
	type alias Manifest
	return json.Marshal(alias(m))
}

// Validate enforces the fail-closed rules shared by local review and
// deployment reporting. In particular, a visual or release manifest cannot
// become a pass with compile/protocol evidence alone.
func (m Manifest) Validate() error {
	if m.SchemaVersion != ManifestSchemaVersion {
		return fmt.Errorf("unsupported manifest schema version %d", m.SchemaVersion)
	}
	if m.Profile != ProfileProtocol && m.Profile != ProfileVisual && m.Profile != ProfileReleaseVisual {
		return fmt.Errorf("unsupported evidence profile %q", m.Profile)
	}
	if strings.TrimSpace(m.RunID) == "" || strings.TrimSpace(string(m.Profile)) == "" {
		return fmt.Errorf("run_id and profile are required")
	}
	if !m.State.IsTerminal() {
		return fmt.Errorf("manifest state %q is not terminal", m.State)
	}
	if m.Target.Ramp == "" || m.Target.Platform == "" || m.Target.OS == "" || m.Runner.ID == "" || m.Runner.TargetOS == "" {
		return fmt.Errorf("target and runner identity are required")
	}
	if m.Provenance.ArtifactDigest == "" || m.Provenance.StartedAt.IsZero() || m.Provenance.CompletedAt.IsZero() {
		return fmt.Errorf("artifact digest and provenance timestamps are required")
	}
	if m.Provenance.CompletedAt.Before(m.Provenance.StartedAt) {
		return fmt.Errorf("provenance completion precedes start")
	}
	if m.Profile != ProfileProtocol {
		if strings.TrimSpace(m.Timeline.Version) == "" || strings.TrimSpace(m.Timeline.JourneyRef) == "" || len(m.Timeline.ChapterIDs) == 0 {
			return fmt.Errorf("visual profiles require a versioned timeline with chapters")
		}
		if !m.Timeline.Ordered {
			return fmt.Errorf("timeline ordering is not verified")
		}
		if m.Timeline.RedactionStatus != "verified" {
			return fmt.Errorf("timeline redaction is not verified")
		}
		if m.Timeline.WorkflowRequired && m.Timeline.Workflow == nil {
			return fmt.Errorf("required workflow reference is missing")
		}
		if m.Timeline.Workflow != nil {
			if err := validateWorkflowReference(*m.Timeline.Workflow, m.RunID, m.Provenance.ArtifactDigest, m.Target.ID, m.CellID); err != nil {
				return err
			}
			if m.Timeline.WorkflowRequired && !strings.EqualFold(m.Timeline.Workflow.Disposition, "pass") && !strings.EqualFold(m.Timeline.Workflow.Disposition, "passed") {
				return fmt.Errorf("required workflow reference disposition is %q", m.Timeline.Workflow.Disposition)
			}
		}
	}

	gates := make(map[GateName]GateResult, len(m.Gates))
	for _, gate := range m.Gates {
		if gate.Name == "" || gate.Disposition == "" {
			return fmt.Errorf("gate name and disposition are required")
		}
		if _, exists := gates[gate.Name]; exists {
			return fmt.Errorf("duplicate gate %q", gate.Name)
		}
		gates[gate.Name] = gate
	}
	for _, required := range RequiredGates(m.Profile) {
		gate, ok := gates[required]
		if !ok {
			return fmt.Errorf("required gate %q is missing", required)
		}
		if !gate.Required {
			return fmt.Errorf("required gate %q is not passed", required)
		}
	}
	if m.State == StatePassed {
		for _, required := range RequiredGates(m.Profile) {
			if gates[required].Disposition != GatePassed {
				return fmt.Errorf("passed manifest has non-passing gate %q", required)
			}
		}
	}

	if m.Profile != ProfileProtocol {
		if len(m.Artifacts) == 0 {
			return fmt.Errorf("visual profiles require artifacts")
		}
		for _, artifact := range m.Artifacts {
			if err := validateArtifact(artifact); err != nil {
				return err
			}
		}
	}
	if m.Profile == ProfileReleaseVisual && m.State == StatePassed && gates[GateGovernance].Disposition != GatePassed {
		return fmt.Errorf("release visual pass requires governance reporting")
	}
	return nil
}

func validateArtifact(artifact Artifact) error {
	if strings.TrimSpace(artifact.ImmutableRef) == "" || strings.TrimSpace(artifact.Kind) == "" || strings.TrimSpace(artifact.Checksum) == "" {
		return fmt.Errorf("artifact immutable_ref, kind, and checksum are required")
	}
	if !strings.HasPrefix(artifact.Checksum, "sha256:") {
		return fmt.Errorf("artifact %q checksum must be sha256", artifact.ImmutableRef)
	}
	if artifact.SizeBytes <= 0 || artifact.CreatedAt.IsZero() {
		return fmt.Errorf("artifact %q must have positive size and creation time", artifact.ImmutableRef)
	}
	if artifact.LocalPath != "" && !filepath.IsAbs(artifact.LocalPath) {
		return fmt.Errorf("artifact %q local_path must be absolute", artifact.ImmutableRef)
	}
	if artifact.Kind == "recording" {
		if artifact.Width <= 0 || artifact.Height <= 0 || artifact.DurationMs <= 0 || artifact.Container == "" || artifact.Codec == "" {
			return fmt.Errorf("recording %q is missing media metadata", artifact.ImmutableRef)
		}
		if !artifact.UsefulFrames {
			return fmt.Errorf("recording %q has no useful frames", artifact.ImmutableRef)
		}
	}
	return nil
}
