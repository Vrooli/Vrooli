package scenarioruntime

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// StopReasonReaperFinalize labels instances finalized by FinalizeStuckInstance.
// It distinguishes reaper-driven stops from normal lifecycle stops (which use
// "scenario stopped") and stale-instance expiries (which use the reconcile
// reason).
const StopReasonReaperFinalize = "reaper-finalize"

// EventTypeInstanceReaped is the runtime_events row recorded for forensics
// whenever a stuck-stopping instance is finalized.
const EventTypeInstanceReaped = "instance_reaped"

// FinalizerStore is the union of repositories needed to finalize a single
// stuck runtime instance. It is intentionally narrow so both the maintenance
// reaper and the ports preemption path can satisfy it from their own store
// surfaces without depending on each other.
type FinalizerStore interface {
	ProcessRefRepository
	EventRepository
	ReleaseActivePortClaimsForInstance(ctx context.Context, instanceID string) ([]PortClaim, error)
	StopLease(ctx context.Context, instanceID string, generation int64, reason string) (Instance, error)
}

// FinalizeStuckInstance releases the instance's active port claims, marks all
// of its running process_refs exited, transitions the instance row to stopped
// with stop_reason=reaper-finalize, and emits an instance_reaped event with
// the provided trigger label embedded in the details payload.
//
// The function is idempotent: ErrNotFound and ErrStaleGeneration from the
// underlying store are swallowed so a concurrent finalizer cannot cause an
// error here. Any other store error is propagated.
//
// `at` is the wall-clock time recorded on process_ref ended_at. Callers can
// pass time.Now().UTC() in production or a fixed clock value in tests.
func FinalizeStuckInstance(ctx context.Context, store FinalizerStore, instance Instance, trigger string, at time.Time) error {
	if _, err := store.ReleaseActivePortClaimsForInstance(ctx, instance.InstanceID); err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	refs, err := store.ListProcessRefs(ctx, instance.InstanceID)
	if err != nil {
		return err
	}
	endedAt := at
	for _, ref := range refs {
		if ref.Status != SupervisorStatusRunning {
			continue
		}
		if _, err := store.UpdateProcessRefStatus(ctx, ref.RefID, "exited", &endedAt); err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
	}
	if _, err := store.StopLease(ctx, instance.InstanceID, instance.Generation, StopReasonReaperFinalize); err != nil && !errors.Is(err, ErrNotFound) && !errors.Is(err, ErrStaleGeneration) {
		return err
	}
	details, _ := json.Marshal(map[string]string{
		"trigger":      trigger,
		"prior_status": instance.Status,
	})
	if _, err := store.RecordEvent(ctx, Event{
		InstanceID:  instance.InstanceID,
		Scenario:    instance.Scenario,
		EventType:   EventTypeInstanceReaped,
		DetailsJSON: string(details),
	}); err != nil {
		return err
	}
	return nil
}
