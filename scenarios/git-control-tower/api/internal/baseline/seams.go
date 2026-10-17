package baseline

import (
	"context"
	"fmt"
	"time"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

// Target identifies what a baseline operates on for a single request.
type Target struct {
	RepoID   int64
	RepoDir  string
	Scenario string
}

// Staleness reports working-tree drift from the baseline commit.
type Staleness struct {
	CommitsSince int
	FilesChanged int
	LikelyStale  bool
}

// ComputeStaleness reports how far repoDir has drifted from baselineSha.
func ComputeStaleness(ctx context.Context, probe StalenessProbe, repoDir, baselineSha string) (Staleness, error) {
	if probe == nil || baselineSha == "" {
		return Staleness{}, nil
	}
	commits, files, err := probe.Since(ctx, repoDir, baselineSha)
	if err != nil {
		return Staleness{}, err
	}
	return Staleness{CommitsSince: commits, FilesChanged: files, LikelyStale: commits > 10 || files > 20}, nil
}

// PhaseStatus is a terminal run phase status retained for operator summaries.
type PhaseStatus struct {
	Name   string
	Status string
}

// ExecResult is the canonical terminal identity returned by Test Genie.
type ExecResult struct {
	RunID                             string
	Success                           bool
	CompletedAt                       time.Time
	TreeDigest                        string
	PhaseSetDigest                    string
	CaptureProfile                    string
	DescriptorSnapshotDigest          string
	DescriptorSnapshotSchemaVersion   int
	GitSha                            string
	GitDirty                          bool
	ExecutionConfigurationFingerprint string
	GateQuality                       bool
	EvidenceTier                      string
	SourceScope                       string
	SourceStable                      bool
	Phases                            []PhaseStatus
}

// CompareResult carries Test Genie's PhaseDiff messages unchanged. Keeping the
// owning proto at this seam prevents GCT from losing future descriptor fields,
// reason codes, or unknown phases while forwarding comparison evidence.
type CompareResult struct {
	Comparison *runspb.CompareRunsResponse
	Verdict    string
	Phases     []*runspb.PhaseDiff
}

type (
	PhaseDiff   = runspb.PhaseDiff
	ArtifactRef = runspb.ArtifactRef
)

// ArtifactCatalog is Test Genie's path-free evidence catalog for one run.
type ArtifactCatalog struct {
	RunID            string
	SchemaVersion    int
	Digest           string
	Artifacts        []*runspb.ArtifactRef
	LegacyDiscovered bool
	DegradedReasons  []string
}

type RunHandle struct {
	RunID                 string
	EstimatedTotalSeconds int
	EtaKnown              bool
	Coalesced             bool
}

type RunStatusInfo struct {
	Status   string
	Terminal bool
	// Missing means Test Genie authoritatively has no durable record for this
	// run id. It is distinct from a transient status-read error: a parent can
	// terminalize this impossible handoff as infrastructure failure.
	Missing                     bool
	Success                     bool
	RecommendedNextCheckSeconds int
	Standing                    *commonv1.OperationStanding
}

type ReusableRun struct {
	RunID          string
	CompletedAt    time.Time
	CaptureProfile string
}

type RunBusyError struct {
	Scenario string
	RunID    string
	Preset   string
}

func (e *RunBusyError) Error() string {
	preset := e.Preset
	if preset == "" {
		preset = "(default)"
	}
	return fmt.Sprintf("scenario %s already has an in-progress run %s (preset %s); wait for it or abort it before starting a different run", e.Scenario, e.RunID, preset)
}

type Executor interface {
	StartRun(ctx context.Context, scenario string) (RunHandle, error)
	AwaitResult(ctx context.Context, scenario, runID string) (ExecResult, error)
	RunStatus(ctx context.Context, scenario, runID string) (RunStatusInfo, error)
	FindReusableRun(ctx context.Context, scenario string) (ReusableRun, bool, error)
}

// ReservationExecutor is the optional collection-aware start seam. Ordinary
// baseline captures remain compatible with Executor; collection captures use
// this method when the concrete Test Genie client can declare one logical
// reservation for all members.
type ReservationExecutor interface {
	StartRunWithReservation(ctx context.Context, scenario, reservationID string, memberCount int) (RunHandle, error)
}

type RunsClient interface {
	PinRun(ctx context.Context, scenario, runID, pinnedBy, reason string) error
	UnpinRun(ctx context.Context, scenario, runID, pinnedBy string) error
	CompareRuns(ctx context.Context, scenario, runIDA, runIDB, phase string) (CompareResult, error)
	ListRunArtifacts(ctx context.Context, scenario, runID string) (ArtifactCatalog, error)
	CompareRunVisuals(ctx context.Context, scenario, baseRunID, curRunID string) ([]VisualDelta, error)
}

// MissingEvidenceCapturer is an optional seam for providers that can collect
// only the evidence a baseline diff is missing. It must not re-run the
// comprehensive suite; implementations should return the run id containing
// the newly captured evidence.
type MissingEvidenceCapturer interface {
	CaptureMissingEvidence(ctx context.Context, scenario, sourceRunID string, kinds []string) (string, error)
}

type VisualDelta struct {
	Page            string
	Label           string
	Status          string
	ChangedFraction float64
}

type StalenessProbe interface {
	Since(ctx context.Context, repoDir, sha string) (commits int, files int, err error)
}

type Reachability interface {
	Probe(ctx context.Context) error
}
