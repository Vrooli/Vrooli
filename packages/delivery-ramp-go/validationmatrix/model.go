package validationmatrix

import (
	"context"
	"fmt"
	"strings"
	"time"

	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

// RunState is the durable lifecycle of a matrix run. A disconnected client
// does not affect this state; the service owns the work after Start returns.
type RunState string

// DefaultCommand preserves the existing validation entry point when a caller
// does not provide an addressed command. Remote callers may supply any
// manifest-governed command and its arguments through MatrixSelection.
const DefaultCommand = "scenario test"

const (
	RunQueued    RunState = "queued"
	RunRunning   RunState = "running"
	RunCompleted RunState = "completed"
	RunFailed    RunState = "failed"
	RunCancelled RunState = "cancelled"
)

func (s RunState) Terminal() bool {
	return s == RunCompleted || s == RunFailed || s == RunCancelled
}

// CellState is separate from the provider disposition so queued/running and
// terminal provider outcomes remain inspectable independently.
type CellState string

const (
	CellQueued    CellState = "queued"
	CellRunning   CellState = "running"
	CellRetrying  CellState = "retrying"
	CellCompleted CellState = "completed"
	CellFailed    CellState = "failed"
	CellCancelled CellState = "cancelled"
)

type TargetKind string

const (
	TargetLocal  TargetKind = "local"
	TargetBridge TargetKind = "bridge"
)

// JourneySelection and TargetSelection are provider-neutral descriptions of
// catalog records. The source providers own their definitions and adapters
// only receive the immutable snapshot below.
type JourneySelection struct {
	JourneyID            string                                `json:"journey_id"`
	DisplayName          string                                `json:"display_name"`
	SourcePath           string                                `json:"source_path,omitempty"`
	ExecutionMode        string                                `json:"execution_mode,omitempty"`
	Required             bool                                  `json:"required"`
	RequiredCapabilities []domainv1.ValidationTargetCapability `json:"required_capabilities,omitempty"`
	Category             string                                `json:"category,omitempty"`
	Requirements         []string                              `json:"requirements,omitempty"`
	EstimatedDurationSec int                                   `json:"estimated_duration_seconds,omitempty"`
	Safety               JourneySafety                         `json:"safety,omitempty"`
}

type JourneySafety struct {
	Mutating             bool `json:"mutating"`
	RequiresIsolation    bool `json:"requires_isolation"`
	RequiresConfirmation bool `json:"requires_confirmation"`
}

type TargetSelection struct {
	Descriptor *domainv1.ValidationTargetDescriptor `json:"descriptor"`
	Kind       TargetKind                           `json:"kind"`
}

// CatalogResolver is the provider-neutral discovery seam. Implementations
// may be backed by a workflow catalog, a ramp target inventory, or a bridge
// target
// inventory; the matrix service only consumes the normalized records.
type CatalogResolver interface {
	Resolve(context.Context, string) (CatalogSnapshot, error)
}

type CatalogSnapshot struct {
	Journeys []JourneySelection `json:"journeys"`
	Targets  []TargetSelection  `json:"targets"`
}

type MatrixSelection struct {
	ScenarioName        string                                  `json:"scenario_name"`
	Command             string                                  `json:"command,omitempty"`
	CommandArgs         []string                                `json:"command_args,omitempty"`
	ArtifactDigest      string                                  `json:"artifact_digest"`
	ArtifactPath        string                                  `json:"artifact_path,omitempty"`
	DeploymentMode      string                                  `json:"deployment_mode,omitempty"`
	ReleaseProfile      string                                  `json:"release_profile,omitempty"`
	IdempotencyKey      string                                  `json:"idempotency_key,omitempty"`
	Journeys            []JourneySelection                      `json:"journeys"`
	Targets             []TargetSelection                       `json:"targets"`
	EnvironmentProfiles []domainv1.ValidationEnvironmentProfile `json:"environment_profiles"`
	MaxConcurrency      int                                     `json:"max_concurrency"`
	Metadata            map[string]string                       `json:"metadata,omitempty"`
}

func (s MatrixSelection) WithCatalog(ctx context.Context, catalog CatalogResolver) (MatrixSelection, error) {
	if catalog == nil {
		return MatrixSelection{}, fmt.Errorf("validation catalog resolver is unavailable")
	}
	snapshot, err := catalog.Resolve(ctx, s.ScenarioName)
	if err != nil {
		return MatrixSelection{}, fmt.Errorf("resolve validation catalog: %w", err)
	}
	resolved := cloneSelection(s)
	if len(resolved.Journeys) == 0 && snapshot.Journeys != nil {
		resolved.Journeys = snapshot.Journeys
	}
	if len(resolved.Targets) == 0 && snapshot.Targets != nil {
		resolved.Targets = snapshot.Targets
	}
	return resolved, nil
}

// CellRecord wraps the proto cell with durable execution metadata. The proto
// remains the cross-scenario evidence contract; this metadata is owned by the
// ramp orchestration service.
type CellRecord struct {
	Cell       *domainv1.ValidationCell `json:"cell"`
	RowID      string                   `json:"row_id"`
	ColumnID   string                   `json:"column_id"`
	ProfileID  string                   `json:"profile_id"`
	TargetKind TargetKind               `json:"target_kind"`
	State      CellState                `json:"state"`
	Attempts   int                      `json:"attempts"`
	StartedAt  time.Time                `json:"started_at,omitempty"`
	UpdatedAt  time.Time                `json:"updated_at"`
	TerminalAt time.Time                `json:"terminal_at,omitempty"`
	Report     map[string]string        `json:"report,omitempty"`
}

type MatrixRow struct {
	ID        string `json:"id"`
	JourneyID string `json:"journey_id"`
}

type MatrixColumn struct {
	ID       string `json:"id"`
	TargetID string `json:"target_id"`
}

type MatrixProfile struct {
	ID      string                                `json:"id"`
	Profile domainv1.ValidationEnvironmentProfile `json:"profile"`
}

type MatrixRun struct {
	RunID          string                     `json:"run_id"`
	IdempotencyKey string                     `json:"idempotency_key,omitempty"`
	Matrix         *domainv1.ValidationMatrix `json:"matrix"`
	Selection      MatrixSelection            `json:"selection"`
	Rows           []MatrixRow                `json:"rows"`
	Columns        []MatrixColumn             `json:"columns"`
	Profiles       []MatrixProfile            `json:"profiles"`
	State          RunState                   `json:"state"`
	ParentRunID    string                     `json:"parent_run_id,omitempty"`
	CreatedAt      time.Time                  `json:"created_at"`
	UpdatedAt      time.Time                  `json:"updated_at"`
	CompletedAt    time.Time                  `json:"completed_at,omitempty"`
	Cells          []*CellRecord              `json:"cells"`
	Gate           *domainv1.ReleaseGate      `json:"gate,omitempty"`
	ReleaseReport  ReleaseReportStatus        `json:"release_report"`
}

type ReleaseReportStatus struct {
	ReportedAt time.Time `json:"reported_at,omitempty"`
	Error      string    `json:"error,omitempty"`
}

type CellRequest struct {
	RunID          string
	MatrixID       string
	Command        string
	Args           []string
	ArtifactDigest string
	ArtifactPath   string
	Cell           *domainv1.ValidationCell
	Journey        JourneySelection
	Target         *domainv1.ValidationTargetDescriptor
	Metadata       map[string]string
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	copyValues := make(map[string]string, len(values))
	for key, value := range values {
		copyValues[key] = value
	}
	return copyValues
}

type CellResult struct {
	Disposition domainv1.ValidationDisposition
	Reason      string
	Evidence    []*domainv1.LayeredEvidence
	Retryable   bool
	Identity    ExecutionIdentity
	Report      map[string]string
}

type ExecutionIdentity struct {
	NodeID         string `json:"node_id,omitempty"`
	JobID          string `json:"job_id,omitempty"`
	RunID          string `json:"run_id,omitempty"`
	ArtifactDigest string `json:"artifact_digest,omitempty"`
}

// CellTransport is the single execution seam for local and bridge cells. The
// transport owns reachability; the ramp-owned driver remains responsible for
// producing target evidence. A bridge dispatch therefore cannot become a
// desktop pass without evidence from the target.
type CellTransport interface {
	Execute(context.Context, CellRequest) CellResult
}

type Executors struct {
	Local  CellTransport
	Bridge CellTransport
}

func (e Executors) Execute(ctx context.Context, kind TargetKind, request CellRequest) CellResult {
	var transport CellTransport
	switch kind {
	case TargetLocal:
		transport = e.Local
	case TargetBridge:
		transport = e.Bridge
	default:
		return unavailableResult("cell target transport is unspecified")
	}
	if transport == nil {
		return unavailableResult(fmt.Sprintf("%s cell transport is unavailable", kind))
	}
	return transport.Execute(ctx, request)
}

// ReleaseReporter is the deployment-manager integration seam. The reporter
// must receive the exact immutable run, matrix, and artifact identity used for
// the gate; the matrix service never fabricates a deployment approval.
type ReleaseReporter interface {
	ReportValidationGate(context.Context, ReleaseVerdict) error
}

type ReleaseVerdict struct {
	RunID          string
	MatrixID       string
	ScenarioName   string
	ArtifactDigest string
	Gate           *domainv1.ReleaseGate
	Evidence       []*domainv1.LayeredEvidence
}

type MatrixComparison struct {
	CurrentRunID          string           `json:"current_run_id"`
	PriorRunID            string           `json:"prior_run_id"`
	ScenarioName          string           `json:"scenario_name"`
	CurrentArtifactDigest string           `json:"current_artifact_digest"`
	PriorArtifactDigest   string           `json:"prior_artifact_digest"`
	Changed               bool             `json:"changed"`
	Cells                 []CellComparison `json:"cells"`
}

type CellComparison struct {
	Key                  string                         `json:"key"`
	CurrentCellID        string                         `json:"current_cell_id,omitempty"`
	PriorCellID          string                         `json:"prior_cell_id,omitempty"`
	CurrentDisposition   domainv1.ValidationDisposition `json:"current_disposition"`
	PriorDisposition     domainv1.ValidationDisposition `json:"prior_disposition"`
	CurrentEvidenceCount int                            `json:"current_evidence_count"`
	PriorEvidenceCount   int                            `json:"prior_evidence_count"`
	Changed              bool                           `json:"changed"`
}

type RerunSelector struct {
	Kind      string
	JourneyID string
	TargetID  string
	CellID    string
}

const (
	RerunAll     = "all"
	RerunJourney = "journey"
	RerunRow     = "row"
	RerunTarget  = "target"
	RerunColumn  = "column"
	RerunFailed  = "failed"
	RerunCell    = "cell"
)

func (s RerunSelector) matches(record *CellRecord) bool {
	if record == nil || record.Cell == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(s.Kind)) {
	case RerunAll:
		return true
	case RerunJourney:
		return record.Cell.GetJourneyId() == s.JourneyID
	case RerunRow:
		return record.Cell.GetJourneyId() == s.JourneyID
	case RerunTarget:
		return record.Cell.GetTargetId() == s.TargetID
	case RerunColumn:
		return record.Cell.GetTargetId() == s.TargetID
	case RerunCell:
		return record.Cell.GetCellId() == s.CellID
	case RerunFailed:
		return record.State == CellFailed || record.State == CellCancelled || record.Cell.GetDisposition() != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS
	default:
		return false
	}
}

func (s MatrixSelection) validate() error {
	if strings.TrimSpace(s.ScenarioName) == "" || strings.TrimSpace(s.ArtifactDigest) == "" {
		return fmt.Errorf("scenario name and artifact digest are required")
	}
	if len(s.Journeys) == 0 || len(s.Targets) == 0 || len(s.EnvironmentProfiles) == 0 {
		return fmt.Errorf("at least one journey, target, and environment profile are required")
	}
	if s.MaxConcurrency < 0 {
		return fmt.Errorf("max concurrency cannot be negative")
	}
	seenJourneys := make(map[string]struct{}, len(s.Journeys))
	for _, journey := range s.Journeys {
		if strings.TrimSpace(journey.JourneyID) == "" {
			return fmt.Errorf("journey ID is required")
		}
		if _, ok := seenJourneys[journey.JourneyID]; ok {
			return fmt.Errorf("duplicate journey %q", journey.JourneyID)
		}
		seenJourneys[journey.JourneyID] = struct{}{}
	}
	seenTargets := make(map[string]struct{}, len(s.Targets))
	for _, target := range s.Targets {
		if target.Descriptor == nil || strings.TrimSpace(target.Descriptor.GetTargetId()) == "" {
			return fmt.Errorf("target descriptor and ID are required")
		}
		if _, ok := seenTargets[target.Descriptor.GetTargetId()]; ok {
			return fmt.Errorf("duplicate target %q", target.Descriptor.GetTargetId())
		}
		seenTargets[target.Descriptor.GetTargetId()] = struct{}{}
	}
	return nil
}
