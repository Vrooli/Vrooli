package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/runmodel"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/eventbus"
)

// TerminalEvidenceSelector names one immutable native receipt and its Events
// acknowledgment. SourceInstance/SourceEpoch identify the native database, NOT
// the independently authorized Events installation.
type TerminalEvidenceSelector struct {
	Schema                    string `json:"schema"`
	OwnerID                   string `json:"owner_id"`
	SourceInstance            string `json:"source_instance"`
	SourceEpoch               string `json:"source_epoch"`
	RunID                     string `json:"run_id"`
	LifecycleVersion          string `json:"lifecycle_version"`
	TerminalObservationSHA256 string `json:"terminal_observation_sha256"`
	TerminalEventID           string `json:"terminal_event_id"`
	CanonicalEventSHA256      string `json:"canonical_event_sha256"`
}

type TerminalEvidenceReference struct {
	URI     string `json:"uri"`
	Version string `json:"version"`
}

// SelectedTerminalEvidence contains no task, transcript, paths, labels or
// caller-supplied Work references. A terminal observation is not accepted work.
type SelectedTerminalEvidence struct {
	Schema string `json:"schema"`
	TerminalEvidenceSelector
	Status     string                    `json:"status"`
	ObservedAt time.Time                 `json:"observed_at"`
	Reference  TerminalEvidenceReference `json:"reference"`
	Lineage    *struct{}                 `json:"lineage"`
}

// TerminalEvidenceAuthority must be supplied by a trusted composition, never
// from request fields. It must check current read authority for this exact
// owner/source/epoch/selection. It is called before storage access and again at
// release; revocation/expiry on the second check suppresses the entire result.
type TerminalEvidenceAuthority func(context.Context, TerminalEvidenceSelector) error

// TerminalEvidenceReader is dormant: no normal constructor, handler, publisher,
// credential or runtime worker installs it. It reads only existing receipts and
// locally retained Events acknowledgments; it cannot qualify historical runs.
type TerminalEvidenceReader struct {
	runs      *runRepository
	authority TerminalEvidenceAuthority
}

func NewTerminalEvidenceReader(repos *Repositories, authority TerminalEvidenceAuthority) (*TerminalEvidenceReader, error) {
	if repos == nil || authority == nil {
		return nil, errors.New("terminal evidence requires trusted read authority")
	}
	runs, ok := repos.Runs.(*runRepository)
	if !ok || runs == nil || runs.db == nil || runs.terminalReceiptScope == nil {
		return nil, errors.New("terminal receipts disabled")
	}
	if err := runs.terminalReceiptScope.Validate(); err != nil {
		return nil, err
	}
	return &TerminalEvidenceReader{runs: runs, authority: authority}, nil
}

var terminalEvidenceDigest = regexp.MustCompile(`^[a-f0-9]{64}$`)

func (r *TerminalEvidenceReader) Select(ctx context.Context, s TerminalEvidenceSelector) (SelectedTerminalEvidence, error) {
	var empty SelectedTerminalEvidence
	if r == nil || r.runs == nil || r.authority == nil || ctx == nil {
		return empty, errors.New("terminal evidence reader unavailable")
	}
	runID, err := uuid.Parse(s.RunID)
	version, versionErr := strconv.ParseInt(s.LifecycleVersion, 10, 64)
	scope := *r.runs.terminalReceiptScope
	if err != nil || runID == uuid.Nil || runID.String() != s.RunID || versionErr != nil || version < 1 || strconv.FormatInt(version, 10) != s.LifecycleVersion || s.Schema != "agent-manager.run-terminal.v1" || s.OwnerID != "agent-manager" || s.SourceInstance != scope.SourceInstance || s.SourceEpoch != scope.SourceEpoch || !terminalEvidenceDigest.MatchString(s.TerminalObservationSHA256) || !terminalEvidenceDigest.MatchString(s.TerminalEventID) || !terminalEvidenceDigest.MatchString(s.CanonicalEventSHA256) {
		return empty, errors.New("invalid exact terminal evidence selection")
	}
	checkAuthority := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := r.authority(ctx, s); err != nil {
			return err
		}
		return ctx.Err()
	}
	if err := checkAuthority(); err != nil {
		return empty, err
	}
	// Check the durable installation binding, not just caller/configuration text.
	var instance, epoch string
	if err := r.runs.db.QueryRowContext(ctx, `SELECT source_instance,source_epoch FROM run_terminal_receipt_source WHERE singleton=1`).Scan(&instance, &epoch); err != nil {
		return empty, err
	}
	if instance != scope.SourceInstance || epoch != scope.SourceEpoch {
		return empty, errors.New("terminal evidence source binding mismatch")
	}
	receipt, err := r.runs.TerminalReceipt(ctx, runID, version)
	if err != nil {
		return empty, err
	}
	ack, err := r.runs.TerminalEventAcknowledgment(ctx, runID, version)
	if err != nil {
		return empty, err
	}
	if receipt.ObservationSHA256 != s.TerminalObservationSHA256 || receipt.EventIntentID != s.TerminalEventID || ack.EventID != s.TerminalEventID || ack.CanonicalSHA256 != s.CanonicalEventSHA256 {
		return empty, errors.New("terminal receipt and selected event mismatch")
	}
	var observation runmodel.TerminalObservation
	if err := json.Unmarshal([]byte(receipt.ObservationJSON), &observation); err != nil {
		return empty, err
	}
	canonical, err := json.Marshal(observation)
	if err != nil || string(canonical) != receipt.ObservationJSON || observation.Schema != "agent-manager.terminal-observation.v1" || observation.CapturedAt.IsZero() || observation.Coverage != "terminal_state_only" || observation.WorkLineage != "unverified" || observation.ExecutionMode == string(domain.ExecutionModeImported) || !domain.RunStatus(observation.Status).IsTerminal() {
		return empty, errors.New("unqualified terminal observation")
	}
	// Cross-check all retained event payload facts against the receipt, rather
	// than trusting two individually valid hashes to prove their relationship.
	expected := eventbus.DomainEvent{Source: "agent-manager", EventType: runmodel.TerminalEventType, Occurred: observation.CapturedAt, Payload: map[string]any{
		"source_instance": s.SourceInstance, "source_epoch": s.SourceEpoch, "run_id": s.RunID, "lifecycle_version": s.LifecycleVersion, "terminal_observation_sha256": s.TerminalObservationSHA256, "status": observation.Status,
	}}
	encoded, err := json.Marshal(expected)
	if err != nil || string(encoded) != receipt.EventIntentJSON {
		return empty, errors.New("terminal event intent does not describe receipt")
	}
	observedAt := time.Now().UTC()
	if observation.CapturedAt.After(observedAt) {
		return empty, errors.New("terminal observation is in the future")
	}
	result := SelectedTerminalEvidence{Schema: "agent-manager.selected-terminal-run-evidence.v1", TerminalEvidenceSelector: s, Status: observation.Status, ObservedAt: observedAt, Reference: TerminalEvidenceReference{URI: fmt.Sprintf("agent-manager://runs/%s/terminal-receipts/%s", s.RunID, s.LifecycleVersion), Version: receipt.ObservationSHA256}}
	if err := checkAuthority(); err != nil {
		return empty, err
	}
	return result, nil
}
