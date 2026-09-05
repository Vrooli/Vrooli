package deliveryramp

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	// JourneySchemaVersion is the numeric version persisted in journey sidecars.
	JourneySchemaVersion = 2
	// JourneyEvidenceVersion is the stable evidence contract name.
	JourneyEvidenceVersion = "journey-evidence.v2"
)

type Disposition string

const (
	DispositionPass        Disposition = "pass"
	DispositionFailed      Disposition = "failed"
	DispositionDegraded    Disposition = "degraded"
	DispositionUnavailable Disposition = "unavailable"
	DispositionUnsupported Disposition = "unsupported"
	DispositionNotRun      Disposition = "not_run"
)

type StepDisposition string

const (
	StepPassed      StepDisposition = "passed"
	StepFailed      StepDisposition = "failed"
	StepDegraded    StepDisposition = "degraded"
	StepUnavailable StepDisposition = "unavailable"
	StepNotRun      StepDisposition = "not_run"
)

// DispositionResult is the fail-closed result shared by probes, cells, and
// journeys. MissingCapability and NextAction are required for unavailable
// results so an operator can repair the exact readiness gap.
type DispositionResult struct {
	Disposition       Disposition `json:"disposition"`
	Reason            string      `json:"reason,omitempty"`
	MissingCapability string      `json:"missing_capability,omitempty"`
	NextAction        string      `json:"next_action,omitempty"`
}

func (r DispositionResult) Validate() error {
	if !r.Disposition.Valid() {
		return fmt.Errorf("invalid disposition %q", r.Disposition)
	}
	if r.Disposition == DispositionUnavailable {
		if strings.TrimSpace(r.MissingCapability) == "" {
			return fmt.Errorf("unavailable result requires missing_capability")
		}
		if strings.TrimSpace(r.NextAction) == "" {
			return fmt.Errorf("unavailable result requires next_action")
		}
	}
	return nil
}

func (d Disposition) Valid() bool {
	switch d {
	case DispositionPass, DispositionFailed, DispositionDegraded, DispositionUnavailable, DispositionUnsupported, DispositionNotRun:
		return true
	default:
		return false
	}
}

// ValidatePromotion rejects the only unsafe promotion: treating a target the
// ramp could not execute or does not support as a passing target.
func ValidatePromotion(current, desired Disposition) error {
	if !current.Valid() || !desired.Valid() {
		return fmt.Errorf("invalid disposition transition %q -> %q", current, desired)
	}
	if desired == DispositionPass && (current == DispositionUnavailable || current == DispositionUnsupported) {
		return fmt.Errorf("cannot promote %q to pass", current)
	}
	return nil
}

// ReadinessPolicy describes a bounded condition wait. Durations are encoded
// as milliseconds to preserve the journey-evidence.v2 JSON contract.
type ReadinessPolicy struct {
	ID             string        `json:"id"`
	Reason         string        `json:"reason"`
	Timeout        time.Duration `json:"timeout_ms"`
	PollInterval   time.Duration `json:"poll_interval_ms"`
	StabilityCount int           `json:"stability_count"`
	Cancellation   string        `json:"cancellation"`
}

func (p ReadinessPolicy) MarshalJSON() ([]byte, error) {
	type wire struct {
		ID             string `json:"id"`
		Reason         string `json:"reason"`
		TimeoutMs      int64  `json:"timeout_ms"`
		PollIntervalMs int64  `json:"poll_interval_ms"`
		StabilityCount int    `json:"stability_count"`
		Cancellation   string `json:"cancellation"`
	}
	return json.Marshal(wire{p.ID, p.Reason, p.Timeout.Milliseconds(), p.PollInterval.Milliseconds(), p.StabilityCount, p.Cancellation})
}

func (p *ReadinessPolicy) UnmarshalJSON(data []byte) error {
	type wire struct {
		ID             string `json:"id"`
		Reason         string `json:"reason"`
		TimeoutMs      int64  `json:"timeout_ms"`
		PollIntervalMs int64  `json:"poll_interval_ms"`
		StabilityCount int    `json:"stability_count"`
		Cancellation   string `json:"cancellation"`
	}
	var value wire
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	p.ID, p.Reason = value.ID, value.Reason
	p.Timeout, p.PollInterval = time.Duration(value.TimeoutMs)*time.Millisecond, time.Duration(value.PollIntervalMs)*time.Millisecond
	p.StabilityCount, p.Cancellation = value.StabilityCount, value.Cancellation
	return nil
}

// SettlePolicy records an intentional visual settle period after an action.
type SettlePolicy struct {
	ID           string        `json:"id"`
	Reason       string        `json:"reason"`
	Minimum      time.Duration `json:"minimum_ms"`
	Maximum      time.Duration `json:"maximum_ms"`
	PollInterval time.Duration `json:"poll_interval_ms"`
	Cancellation string        `json:"cancellation"`
}

func (p SettlePolicy) MarshalJSON() ([]byte, error) {
	type wire struct {
		ID           string `json:"id"`
		Reason       string `json:"reason"`
		MinimumMs    int64  `json:"minimum_ms"`
		MaximumMs    int64  `json:"maximum_ms"`
		PollInterval int64  `json:"poll_interval_ms"`
		Cancellation string `json:"cancellation"`
	}
	return json.Marshal(wire{p.ID, p.Reason, p.Minimum.Milliseconds(), p.Maximum.Milliseconds(), p.PollInterval.Milliseconds(), p.Cancellation})
}

func (p *SettlePolicy) UnmarshalJSON(data []byte) error {
	type wire struct {
		ID           string `json:"id"`
		Reason       string `json:"reason"`
		MinimumMs    int64  `json:"minimum_ms"`
		MaximumMs    int64  `json:"maximum_ms"`
		PollInterval int64  `json:"poll_interval_ms"`
		Cancellation string `json:"cancellation"`
	}
	var value wire
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	p.ID, p.Reason = value.ID, value.Reason
	p.Minimum, p.Maximum, p.PollInterval = time.Duration(value.MinimumMs)*time.Millisecond, time.Duration(value.MaximumMs)*time.Millisecond, time.Duration(value.PollInterval)*time.Millisecond
	p.Cancellation = value.Cancellation
	return nil
}

type AssertionSpec struct {
	ID          string `json:"id"`
	Expected    string `json:"expected"`
	Description string `json:"description,omitempty"`
}

type JourneyStepSpec struct {
	ID        string            `json:"id"`
	Purpose   string            `json:"purpose"`
	Action    string            `json:"action"`
	Arguments map[string]string `json:"arguments,omitempty"`
	Capture   bool              `json:"capture"`
	Readiness ReadinessPolicy   `json:"readiness"`
	Settle    SettlePolicy      `json:"settle"`
	Assertion *AssertionSpec    `json:"assertion,omitempty"`
}

type JourneyPlan struct {
	SchemaVersion string            `json:"schema_version"`
	ID            string            `json:"id"`
	Capability    string            `json:"capability"`
	Purpose       string            `json:"purpose"`
	Profile       string            `json:"profile"`
	Steps         []JourneyStepSpec `json:"steps"`
}

type JourneyEvent struct {
	Type             string    `json:"type"`
	StepID           string    `json:"step_id,omitempty"`
	PolicyID         string    `json:"policy_id,omitempty"`
	Observed         string    `json:"observed,omitempty"`
	StartedAt        time.Time `json:"started_at"`
	CompletedAt      time.Time `json:"completed_at"`
	MonotonicStartMs int64     `json:"monotonic_start_ms"`
	MonotonicEndMs   int64     `json:"monotonic_end_ms"`
	Reason           string    `json:"reason,omitempty"`
	Source           string    `json:"source,omitempty"`
	DeviceTimestamp  float64   `json:"device_timestamp,omitempty"`
	Raw              string    `json:"raw,omitempty"`
}

// ClockOffsetSample records a producer-owned host/device wall-clock
// calibration. OffsetMs is host UTC minus the device clock; uncertainty is the
// host-side request round-trip bound used to avoid presenting a transport
// measurement as exact synchronization.
type ClockOffsetSample struct {
	CapturedAt    time.Time         `json:"captured_at"`
	HostTime      time.Time         `json:"host_time"`
	DeviceTime    time.Time         `json:"device_time"`
	OffsetMs      int64             `json:"offset_ms"`
	UncertaintyMs int64             `json:"uncertainty_ms"`
	Evidence      EvidenceReference `json:"evidence,omitempty"`
}

type EvidenceReference struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	URI       string `json:"uri,omitempty"`
	MediaType string `json:"media_type,omitempty"`
	Checksum  string `json:"checksum,omitempty"`
	Redacted  bool   `json:"redacted,omitempty"`
}

type Geometry struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type JourneyStep struct {
	ID                 string              `json:"id,omitempty"`
	ChapterID          string              `json:"chapter_id,omitempty"`
	Name               string              `json:"name"`
	Purpose            string              `json:"purpose,omitempty"`
	Action             string              `json:"action"`
	Disposition        StepDisposition     `json:"disposition"`
	BeforeCaptureID    string              `json:"before_capture_id,omitempty"`
	AfterCaptureID     string              `json:"after_capture_id,omitempty"`
	Evidence           []EvidenceReference `json:"evidence,omitempty"`
	Geometry           *Geometry           `json:"geometry,omitempty"`
	AssertionID        string              `json:"assertion_id,omitempty"`
	AssertionStatus    string              `json:"assertion_status,omitempty"`
	ExpectedState      string              `json:"expected_state,omitempty"`
	ObservedState      string              `json:"observed_state,omitempty"`
	ProcessBefore      string              `json:"process_before,omitempty"`
	ProcessAfter       string              `json:"process_after,omitempty"`
	Route              string              `json:"route,omitempty"`
	Error              string              `json:"error,omitempty"`
	DegradedReason     string              `json:"degraded_reason,omitempty"`
	Readiness          ReadinessPolicy     `json:"readiness,omitempty"`
	Settle             SettlePolicy        `json:"settle,omitempty"`
	StartedAt          time.Time           `json:"started_at"`
	CompletedAt        time.Time           `json:"completed_at"`
	MonotonicStartMs   int64               `json:"monotonic_start_ms"`
	MonotonicEndMs     int64               `json:"monotonic_end_ms"`
	VideoStartOffsetMs *int64              `json:"video_start_offset_ms,omitempty"`
	VideoEndOffsetMs   *int64              `json:"video_end_offset_ms,omitempty"`
	VideoDisposition   StepDisposition     `json:"video_disposition,omitempty"`
	VideoError         string              `json:"video_error,omitempty"`
}

type WorkflowArtifactReference struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	URI       string `json:"uri"`
	MediaType string `json:"media_type,omitempty"`
	Checksum  string `json:"checksum"`
	Redacted  bool   `json:"redacted"`
}

type WorkflowExecutionReference struct {
	Provider       string              `json:"provider"`
	AssetID        string              `json:"asset_id"`
	ExecutionID    string              `json:"execution_id"`
	RunID          string              `json:"run_id"`
	ArtifactDigest string              `json:"artifact_digest"`
	TargetID       string              `json:"target_id"`
	CellID         string              `json:"cell_id"`
	Disposition    string              `json:"disposition"`
	Artifacts      []EvidenceReference `json:"artifacts,omitempty"`
}

// ValidateLink verifies that provider-owned evidence is bound to the same
// durable validation identity as the desktop evidence. A passing workflow
// must expose at least one checksummed, explicitly redacted artifact.
func (r WorkflowExecutionReference) ValidateLink(runID, artifactDigest, targetID, cellID string) error {
	for name, value := range map[string]string{
		"provider": r.Provider, "asset_id": r.AssetID, "execution_id": r.ExecutionID,
		"run_id": r.RunID, "artifact_digest": r.ArtifactDigest, "target_id": r.TargetID,
		"cell_id": r.CellID, "disposition": r.Disposition,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("workflow reference %s is required", name)
		}
	}
	if r.RunID != runID || r.ArtifactDigest != artifactDigest || r.TargetID != targetID || r.CellID != cellID {
		return fmt.Errorf("workflow reference is not bound to the desktop validation identity")
	}
	if strings.EqualFold(r.Disposition, string(DispositionPass)) {
		if len(r.Artifacts) == 0 {
			return fmt.Errorf("passing workflow reference requires artifacts")
		}
		for index, artifact := range r.Artifacts {
			if strings.TrimSpace(artifact.ID) == "" || strings.TrimSpace(artifact.Kind) == "" || strings.TrimSpace(artifact.URI) == "" || strings.TrimSpace(artifact.Checksum) == "" || !artifact.Redacted {
				return fmt.Errorf("workflow artifact %d is missing identity or checksum", index)
			}
		}
	}
	return nil
}

type ProviderObservation struct {
	DeploymentMode   string `json:"deployment_mode"`
	ProviderTier     string `json:"provider_tier"`
	ServiceIdentity  string `json:"service_identity"`
	ArtifactDigest   string `json:"artifact_digest,omitempty"`
	Readiness        string `json:"readiness"`
	FallbackDecision string `json:"fallback_decision,omitempty"`
	SafeRouteClass   string `json:"safe_route_class"`
	LeaseExpiresAt   string `json:"lease_expires_at,omitempty"`
}

type JourneyResult struct {
	SchemaVersion                int                         `json:"schema_version"`
	EvidenceVersion              string                      `json:"evidence_version"`
	SmokeTestID                  string                      `json:"smoke_test_id"`
	ScenarioName                 string                      `json:"scenario_name"`
	Capability                   string                      `json:"capability"`
	PlanID                       string                      `json:"plan_id"`
	Profile                      string                      `json:"profile"`
	Platform                     string                      `json:"platform"`
	Display                      string                      `json:"display"`
	WindowManager                string                      `json:"window_manager,omitempty"`
	Titlebar                     bool                        `json:"titlebar"`
	RecordingStartedBeforeLaunch bool                        `json:"recording_started_before_launch"`
	TargetID                     string                      `json:"target_id,omitempty"`
	CellID                       string                      `json:"cell_id,omitempty"`
	WorkflowRequired             bool                        `json:"workflow_required,omitempty"`
	WorkflowReference            *WorkflowExecutionReference `json:"workflow_reference,omitempty"`
	ProviderObservation          *ProviderObservation        `json:"provider_observation,omitempty"`
	Disposition                  Disposition                 `json:"disposition"`
	DegradedReason               string                      `json:"degraded_reason,omitempty"`
	Events                       []JourneyEvent              `json:"events,omitempty"`
	ClockOffsetStart             *ClockOffsetSample          `json:"clock_offset_start,omitempty"`
	ClockOffsetEnd               *ClockOffsetSample          `json:"clock_offset_end,omitempty"`
	ReviewRecording              *EvidenceReference          `json:"review_recording,omitempty"`
	ReviewRecordingPath          string                      `json:"review_recording_path,omitempty"`
	Steps                        []JourneyStep               `json:"steps"`
	CreatedAt                    time.Time                   `json:"created_at"`
	CompletedAt                  time.Time                   `json:"completed_at,omitempty"`
}
