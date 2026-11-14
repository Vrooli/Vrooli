package runmodel

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/eventbus"
)

// TerminalReceiptScope identifies the commissioned installation, not a run's
// mutable process owner. Supplying it does not authorize publication or Work
// binding. The normal repository constructor leaves capture disabled.
type TerminalReceiptScope struct {
	SourceInstance string `json:"source_instance"`
	SourceEpoch    string `json:"source_epoch"`
}

var terminalScopeToken = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$`)

func (s TerminalReceiptScope) Validate() error {
	if !terminalScopeToken.MatchString(s.SourceInstance) || !terminalScopeToken.MatchString(s.SourceEpoch) {
		return fmt.Errorf("terminal receipt requires explicit bounded source instance and epoch")
	}
	return nil
}

// TerminalObservation retains only a status transition observed by the native
// run owner. It proves neither process liveness, accepted work, source-wide
// completeness nor independent Work lineage. Unknown stays unknown.
type TerminalObservation struct {
	Schema string `json:"schema"`
	TerminalReceiptScope
	RunID               string     `json:"run_id"`
	TaskID              string     `json:"task_id,omitempty"`
	LifecycleVersion    int64      `json:"lifecycle_version"`
	OwnerEpoch          int64      `json:"owner_epoch"`
	ExecutionMode       string     `json:"execution_mode"`
	Status              string     `json:"status"`
	InvocationStartedAt *time.Time `json:"invocation_started_at,omitempty"`
	EndedAt             *time.Time `json:"ended_at,omitempty"`
	CapturedAt          time.Time  `json:"captured_at"`
	Coverage            string     `json:"capture_coverage"`
	WorkLineage         string     `json:"work_lineage"`
}

// TerminalReceipt is an immutable observation and a prospective event intent.
// EventIntentID is NOT an Events persistence acknowledgment or a backlink to an
// old best-effort notification. PublicationState is the capture-time state, not
// a mutable delivery flag. The separately composed TerminalOutbox uses these
// exact bytes and retains the Events acknowledgment in a separate owner ledger.
type TerminalReceipt struct {
	ObservationJSON   string `json:"observation_json"`
	ObservationSHA256 string `json:"observation_sha256"`
	EventIntentJSON   string `json:"event_intent_json"`
	EventIntentID     string `json:"event_intent_id"`
	PublicationState  string `json:"publication_state"`
}

const TerminalEventType = "agent_manager.run.terminal.v1"

func SHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// NewTerminalReceipt is called only with the row re-read inside the successful
// terminal-transition transaction. The caller must establish that this is a
// prospective nonterminal transition, not an imported/historical row.
func NewTerminalReceipt(scope TerminalReceiptScope, run *domain.Run) (TerminalReceipt, error) {
	if err := scope.Validate(); err != nil {
		return TerminalReceipt{}, err
	}
	if run == nil || run.ID == uuid.Nil || !run.Status.IsTerminal() || run.LifecycleVersion < 1 || run.UpdatedAt.IsZero() {
		return TerminalReceipt{}, fmt.Errorf("terminal receipt requires a committed terminal transition")
	}
	observation := TerminalObservation{
		Schema: "agent-manager.terminal-observation.v1", TerminalReceiptScope: scope,
		RunID: run.ID.String(), LifecycleVersion: run.LifecycleVersion,
		OwnerEpoch: run.OwnerEpoch, ExecutionMode: string(run.ExecutionMode.Normalized()),
		Status: string(run.Status), InvocationStartedAt: utcTime(run.InteractiveInvocationStartedAt),
		EndedAt: utcTime(run.EndedAt), CapturedAt: run.UpdatedAt.UTC(),
		Coverage: "terminal_state_only", WorkLineage: "unverified",
	}
	if run.TaskID != uuid.Nil {
		observation.TaskID = run.TaskID.String()
	}
	encoded, err := json.Marshal(observation)
	if err != nil {
		return TerminalReceipt{}, err
	}
	digest := SHA256(encoded)
	event := eventbus.DomainEvent{
		Source: "agent-manager", EventType: TerminalEventType, Occurred: observation.CapturedAt,
		Payload: map[string]any{
			"source_instance": scope.SourceInstance, "source_epoch": scope.SourceEpoch,
			"run_id": observation.RunID, "lifecycle_version": fmt.Sprint(observation.LifecycleVersion),
			"terminal_observation_sha256": digest, "status": observation.Status,
		},
	}
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return TerminalReceipt{}, err
	}
	return TerminalReceipt{string(encoded), digest, string(eventJSON), SHA256(eventJSON), "not_dispatched"}, nil
}

func utcTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	result := value.UTC()
	return &result
}
