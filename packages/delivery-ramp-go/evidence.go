package deliveryramp

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

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
	Version          string                     `json:"version"`
	JourneyRef       string                     `json:"journey_ref"`
	Capability       string                     `json:"capability,omitempty"`
	ProviderTier     string                     `json:"provider_tier,omitempty"`
	SafeRouteClass   string                     `json:"safe_route_class,omitempty"`
	FallbackDecision string                     `json:"fallback_decision,omitempty"`
	ChapterIDs       []string                   `json:"chapter_ids"`
	EventCount       int                        `json:"event_count"`
	Ordered          bool                       `json:"ordered"`
	RedactionStatus  string                     `json:"redaction_status"`
	WorkflowRequired bool                       `json:"workflow_required,omitempty"`
	Workflow         *WorkflowManifestReference `json:"workflow,omitempty"`
}

type WorkflowManifestArtifact struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	URI       string `json:"uri"`
	MediaType string `json:"media_type,omitempty"`
	Checksum  string `json:"checksum"`
	Redacted  bool   `json:"redacted"`
}

type WorkflowManifestReference struct {
	Provider       string                     `json:"provider"`
	AssetID        string                     `json:"asset_id"`
	ExecutionID    string                     `json:"execution_id"`
	RunID          string                     `json:"run_id"`
	ArtifactDigest string                     `json:"artifact_digest"`
	TargetID       string                     `json:"target_id"`
	CellID         string                     `json:"cell_id"`
	Disposition    string                     `json:"disposition"`
	Artifacts      []WorkflowManifestArtifact `json:"artifacts,omitempty"`
}

// PerformanceEvidence is deliberately open to producer-specific summaries.
// The spine owns the presence/status contract; a ramp owns the shape of its
// platform metrics without importing a process or graphics package here.
type PerformanceEvidence struct {
	Status          string          `json:"status"`
	Reason          string          `json:"reason,omitempty"`
	ProtocolSummary any             `json:"protocol_summary,omitempty"`
	DemoSummary     any             `json:"demo_summary,omitempty"`
	DemoProcessTree any             `json:"demo_process_tree,omitempty"`
	ProtocolPhases  []PhaseDuration `json:"protocol_phases,omitempty"`
	DemoPhases      []PhaseDuration `json:"demo_phases,omitempty"`
	TraceRefs       []string        `json:"trace_refs,omitempty"`
	ProfileRefs     []string        `json:"profile_refs,omitempty"`
}

type PhaseDuration struct {
	Name       string `json:"name"`
	Available  bool   `json:"available"`
	DurationMs int64  `json:"duration_ms,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

type Manifest struct {
	SchemaVersion int                 `json:"schema_version"`
	RunID         string              `json:"run_id"`
	Profile       Profile             `json:"profile"`
	State         RunState            `json:"state"`
	Target        EvidenceTarget      `json:"target"`
	CellID        string              `json:"cell_id,omitempty"`
	Runner        Runner              `json:"runner"`
	Provenance    Provenance          `json:"provenance"`
	Timeline      TimelineSummary     `json:"timeline"`
	Gates         []GateResult        `json:"gates"`
	Artifacts     []Artifact          `json:"artifacts"`
	Performance   PerformanceEvidence `json:"performance"`
}

// EvidenceTarget is intentionally narrower than the inventory Target. The
// manifest contract must not leak probe endpoints, credentials, or host-only
// discovery fields.
type EvidenceTarget struct {
	ID         string `json:"id,omitempty"`
	Ramp       string `json:"ramp"`
	Platform   string `json:"platform"`
	OS         string `json:"os"`
	DeviceKind string `json:"device_kind"`
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
	transitions, ok := transitionTable[s]
	return ok && len(transitions) == 0
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

func (m Manifest) Validate() error {
	if m.SchemaVersion != ManifestSchemaVersion {
		return fmt.Errorf("unsupported manifest schema version %d", m.SchemaVersion)
	}
	if m.Profile != ProfileProtocol && m.Profile != ProfileVisual && m.Profile != ProfileReleaseVisual {
		return fmt.Errorf("unsupported evidence profile %q", m.Profile)
	}
	if strings.TrimSpace(m.RunID) == "" {
		return fmt.Errorf("run_id is required")
	}
	if !m.State.IsTerminal() {
		return fmt.Errorf("manifest state %q is not terminal", m.State)
	}
	if m.Target.Ramp == "" || m.Target.Platform == "" || m.Target.OS == "" || m.Runner.ID == "" || m.Runner.TargetOS == "" {
		return fmt.Errorf("target and runner identity are required")
	}
	if m.Provenance.ArtifactDigest == "" || m.Provenance.StartedAt.IsZero() || m.Provenance.CompletedAt.IsZero() || m.Provenance.CompletedAt.Before(m.Provenance.StartedAt) {
		return fmt.Errorf("valid artifact digest and provenance timestamps are required")
	}
	if m.Profile != ProfileProtocol {
		if m.Timeline.Version == "" || m.Timeline.JourneyRef == "" || len(m.Timeline.ChapterIDs) == 0 || !m.Timeline.Ordered || m.Timeline.RedactionStatus != "verified" {
			return fmt.Errorf("visual profiles require an ordered, verified timeline with chapters")
		}
		if m.Timeline.WorkflowRequired && m.Timeline.Workflow == nil {
			return fmt.Errorf("required workflow reference is missing")
		}
		if m.Timeline.Workflow != nil {
			if err := validateWorkflowManifestReference(*m.Timeline.Workflow, m.RunID, m.Provenance.ArtifactDigest, m.Target.ID, m.CellID); err != nil {
				return err
			}
			if m.Timeline.WorkflowRequired && m.Timeline.Workflow.Disposition != string(DispositionPass) && m.Timeline.Workflow.Disposition != "passed" {
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
		if !ok || !gate.Required {
			return fmt.Errorf("required gate %q is missing or not required", required)
		}
		if m.State == StatePassed && gate.Disposition != GatePassed {
			return fmt.Errorf("passed manifest has non-passing gate %q", required)
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
	return nil
}

func validateWorkflowManifestReference(ref WorkflowManifestReference, runID, artifactDigest, targetID, cellID string) error {
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
		return fmt.Errorf("workflow reference is not bound to the validation identity")
	}
	if ref.Disposition == string(DispositionPass) || ref.Disposition == "passed" {
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

func validateArtifact(artifact Artifact) error {
	if strings.TrimSpace(artifact.ImmutableRef) == "" || strings.TrimSpace(artifact.Kind) == "" || !strings.HasPrefix(artifact.Checksum, "sha256:") {
		return fmt.Errorf("artifact %q requires an immutable ref, kind, and sha256 checksum", artifact.ImmutableRef)
	}
	if artifact.SizeBytes <= 0 || artifact.CreatedAt.IsZero() {
		return fmt.Errorf("artifact %q must have positive size and creation time", artifact.ImmutableRef)
	}
	if artifact.LocalPath != "" && !filepath.IsAbs(artifact.LocalPath) {
		return fmt.Errorf("artifact %q local_path must be absolute", artifact.ImmutableRef)
	}
	if artifact.Kind == "recording" {
		if !artifact.UsefulFrames {
			return fmt.Errorf("recording %q has no useful frames", artifact.ImmutableRef)
		}
		if artifact.Width <= 0 || artifact.Height <= 0 || artifact.DurationMs <= 0 || artifact.Container == "" || artifact.Codec == "" {
			return fmt.Errorf("recording %q is missing useful media metadata", artifact.ImmutableRef)
		}
	}
	return nil
}
