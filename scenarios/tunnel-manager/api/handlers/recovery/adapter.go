package recovery

import (
	"time"

	"tunnel-manager/internal/recovery"

	recoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/recovery"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// stateToProto converts the internal state snapshot into the wire shape.
// Zero timestamps map to nil (the field stays unset) so the UI can show
// "never" rather than the epoch.
func stateToProto(s recovery.RecoveryState) *recoveryv1.RecoveryState {
	return &recoveryv1.RecoveryState{
		Status:           statusToProto(s.Status),
		ConsecFailures:   int32(s.ConsecFailures),
		BackoffLevel:     int32(s.BackoffLevel),
		FailedRecoveries: int32(s.FailedRecovery),
		CircuitOpen:      s.CircuitOpen,
		LastCheck:        optionalTS(s.LastCheck),
		LastRecovery:     optionalTS(s.LastRecovery),
		NextRetryAfter:   optionalTS(s.NextRetryAfter),
	}
}

func eventToProto(e recovery.RecoveryEvent) *recoveryv1.RecoveryEvent {
	if e.ID == "" && e.Trigger == "" {
		return nil
	}
	return &recoveryv1.RecoveryEvent{
		Id:        e.ID,
		Trigger:   e.Trigger,
		Action:    e.Action,
		Outcome:   outcomeToProto(e.Outcome),
		Details:   e.Details,
		Attempt:   int32(e.Attempt),
		CreatedAt: optionalTS(e.CreatedAt),
	}
}

func statusToProto(s recovery.RecoveryStatus) recoveryv1.RecoveryStatus {
	switch s {
	case recovery.StatusIdle:
		return recoveryv1.RecoveryStatus_RECOVERY_STATUS_IDLE
	case recovery.StatusMonitoring:
		return recoveryv1.RecoveryStatus_RECOVERY_STATUS_MONITORING
	case recovery.StatusRecovering:
		return recoveryv1.RecoveryStatus_RECOVERY_STATUS_RECOVERING
	case recovery.StatusCircuitOpen:
		return recoveryv1.RecoveryStatus_RECOVERY_STATUS_CIRCUIT_OPEN
	default:
		return recoveryv1.RecoveryStatus_RECOVERY_STATUS_UNSPECIFIED
	}
}

func outcomeToProto(o recovery.EventOutcome) recoveryv1.EventOutcome {
	switch o {
	case recovery.OutcomeSuccess:
		return recoveryv1.EventOutcome_EVENT_OUTCOME_SUCCESS
	case recovery.OutcomeFailure:
		return recoveryv1.EventOutcome_EVENT_OUTCOME_FAILURE
	case recovery.OutcomeSkipped:
		return recoveryv1.EventOutcome_EVENT_OUTCOME_SKIPPED
	default:
		return recoveryv1.EventOutcome_EVENT_OUTCOME_UNSPECIFIED
	}
}

// optionalTS maps a zero time.Time to nil (leaving the proto field
// unset) and otherwise to a UTC timestamp.
func optionalTS(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t.UTC())
}
