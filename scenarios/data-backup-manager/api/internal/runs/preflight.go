package runs

import (
	"context"
	"strings"
	"time"

	"data-backup-manager/internal/capacity"
	"data-backup-manager/internal/preflight"

	"github.com/vrooli/api-core/schedule"
)

type preflightTargets struct{ lookup TargetLookup }

func (a preflightTargets) TargetForRun(ctx context.Context, id string) (preflight.Target, error) {
	t, err := a.lookup.TargetForRun(ctx, id)
	if err != nil {
		return preflight.Target{}, err
	}
	return preflight.Target{ID: t.ID, Kind: t.Kind, Locator: t.Locator}, nil
}

type preflightDestinations struct{ lookup DestinationLookup }

func (a preflightDestinations) DestinationForRun(ctx context.Context, id string) (preflight.Destination, error) {
	d, err := a.lookup.DestinationForRun(ctx, id)
	if err != nil {
		return preflight.Destination{}, err
	}
	return preflight.Destination{
		ID:             d.ID,
		Name:           d.Name,
		BackendKind:    d.BackendKind,
		Location:       d.Location,
		DeviceIdentity: d.DeviceIdentity,
	}, nil
}

func (a preflightDestinations) WithResolvedDestinationForRead(ctx context.Context, id string, use func(preflight.Destination) error) error {
	if use == nil {
		return nil
	}
	return a.lookup.WithResolvedDestinationForRead(ctx, id, func(d DestinationForRun) error {
		return use(preflight.Destination{ID: d.ID, Name: d.Name, BackendKind: d.BackendKind, Location: d.Location, DeviceIdentity: d.DeviceIdentity})
	})
}

func (s *service) runPreflight(ctx context.Context, plan PlanForRun) preflight.Result {
	history, hasHistory := s.lastSuccessfulPhysicalBytes(ctx, plan.ID)
	var statuses []TargetDestinationStatus
	if list, err := s.deps.Repo.TargetDestinationStatuses(ctx, plan.TargetIDs); err == nil {
		statuses = list
	}
	// Degraded-write safety depends on a *different device* already holding a
	// verified copy, and that copy may live on a destination this plan does not
	// write to. Resolve the device identity for every destination referenced by
	// history as well as the plan's own destinations, so the rule reflects the
	// real failure-domain topology instead of only the plan's narrow set.
	deviceIDs := make(map[string]bool, len(plan.DestinationIDs))
	for _, id := range plan.DestinationIDs {
		deviceIDs[id] = true
	}
	for _, status := range statuses {
		if status.DestinationID != "" {
			deviceIDs[status.DestinationID] = true
		}
	}
	devices := make(map[string]string, len(deviceIDs))
	for id := range deviceIDs {
		if d, err := s.deps.Destinations.DestinationForRun(ctx, id); err == nil {
			devices[id] = destinationDeviceKey(d)
		}
	}
	var clk schedule.Clock = s.deps.Clock
	result := preflight.Check(ctx, preflight.Input{
		Plan:             buildPreflightPlan(plan, history, hasHistory, statuses, devices),
		Targets:          preflightTargets{lookup: s.deps.Targets},
		Destinations:     preflightDestinations{lookup: s.deps.Destinations},
		Engine:           s.deps.Engine,
		Sources:          s.deps.Sources,
		Clock:            clk,
		CheckSourcePaths: s.deps.PreflightSourcePaths,
		Readiness:        s.deps.Readiness,
	})
	// Last-known-good is historical evidence, not a readiness probe. Attach
	// the oldest known successful target backup to each grouped incident so a
	// shared failure cannot overstate the protection window.
	if len(result.Incidents) > 0 {
		if statuses, err := s.deps.Repo.TargetStatuses(ctx, plan.TargetIDs); err == nil {
			lastGood := make(map[string]time.Time, len(statuses))
			for _, status := range statuses {
				lastGood[status.TargetID] = status.LastSuccessAt
			}
			for i := range result.Incidents {
				for _, targetID := range result.Incidents[i].TargetIDs {
					candidate := lastGood[targetID]
					if candidate.IsZero() {
						continue
					}
					if result.Incidents[i].LastKnownGood.IsZero() || candidate.Before(result.Incidents[i].LastKnownGood) {
						result.Incidents[i].LastKnownGood = candidate
					}
				}
			}
		}
	}
	return result
}

// retentionSnapshotCount is the maximum number of snapshots the configured GFS
// policy can retain for one source. It is a count, not a byte multiplier: the
// capacity model applies dedup and incremental growth rather than treating
// every tier as another full copy.
func retentionSnapshotCount(plan PlanForRun) int {
	return plan.KeepLatest + plan.KeepHourly + plan.KeepDaily + plan.KeepWeekly + plan.KeepMonthly + plan.KeepAnnual
}

// buildPreflightPlan is the one translation from run history to the capacity
// model. Keeping it pure makes the first-copy and degraded-readiness rules
// testable without a live repository.
func buildPreflightPlan(plan PlanForRun, physicalBytes int64, hasPhysicalHistory bool, statuses []TargetDestinationStatus, devices map[string]string) preflight.Plan {
	out := preflight.Plan{
		TargetIDs:            append([]string(nil), plan.TargetIDs...),
		DestinationIDs:       append([]string(nil), plan.DestinationIDs...),
		RetainedSnapshots:    retentionSnapshotCount(plan),
		FirstCopy:            true,
		ReserveBytes:         capacity.DefaultReserveBytes,
		CapacityPolicy:       capacity.PolicyAdvisory,
		AllowWriteProbe:      true,
		DegradedDestinations: map[string]bool{},
	}
	if hasPhysicalHistory && physicalBytes > 0 {
		out.MeasuredIncrementalBytes = physicalBytes
	}

	// A target×destination success is the only proof that the destination has a
	// copy. A target-level success elsewhere does not prove this destination is
	// covered, so the pair is the key.
	success := make(map[string]bool, len(statuses))
	for _, status := range statuses {
		if !status.LastSuccessAt.IsZero() {
			success[status.TargetID+"\x00"+status.DestinationID] = true
		}
	}
	if len(statuses) > 0 {
		out.FirstCopy = false
		for _, targetID := range plan.TargetIDs {
			for _, destinationID := range plan.DestinationIDs {
				if !success[targetID+"\x00"+destinationID] {
					out.FirstCopy = true
				}
			}
		}
	}

	// Degraded writes are allowed only when every selected target has a
	// separate verified copy. That keeps a single-copy destination strict while
	// letting a redundant destination keep working through an unobservable
	// filesystem state.
	for _, destinationID := range plan.DestinationIDs {
		currentDevice := devices[destinationID]
		allHaveOther := len(plan.TargetIDs) > 0 && currentDevice != ""
		for _, targetID := range plan.TargetIDs {
			hasOther := false
			for otherID, otherDevice := range devices {
				if otherID == destinationID {
					continue
				}
				// A second destination on the same device is not a second copy.
				// Require a different proven device (or a distinct remote backend)
				// before allowing a degraded write. The other copy may belong to
				// any plan, not only this one.
				if otherDevice == "" || otherDevice == currentDevice {
					continue
				}
				if success[targetID+"\x00"+otherID] {
					hasOther = true
					break
				}
			}
			if !hasOther {
				allHaveOther = false
				break
			}
		}
		out.DegradedDestinations[destinationID] = allHaveOther
	}
	return out
}

// destinationDeviceKey is the failure-domain identity used by the degraded
// policy. Filesystems need a proven device; a second location on the same
// device is not an independent copy. A persisted UUID/serial is preferred. A
// legacy destination without one falls back to the current mounted-filesystem
// identity, which is enough for a per-run redundancy decision and keeps a real
// second copy usable instead of permanently blocked.
//
// Remote backends are a separate failure domain by construction, though their
// independence is still not claimed as proven in coverage.
func destinationDeviceKey(d DestinationForRun) string {
	if d.BackendKind != "filesystem" {
		if d.BackendKind == "" {
			return ""
		}
		return "backend:" + d.BackendKind + ":" + d.Name
	}
	if d.DeviceIdentity != nil && d.DeviceIdentity.StrongIdentity() {
		if d.DeviceIdentity.UUID != "" {
			return "device:" + d.DeviceIdentity.UUID
		}
		return "device:" + d.DeviceIdentity.Serial
	}
	return runtimeDeviceKey(d.Location)
}

// lastSuccessfulPhysicalBytes is best-effort evidence. A repository without
// the optional history seam, or a read error, leaves the estimate forecast-only
// rather than inventing a measured increment.
func (s *service) lastSuccessfulPhysicalBytes(ctx context.Context, planID string) (int64, bool) {
	reader, ok := s.deps.Repo.(PhysicalHistoryReader)
	if !ok {
		return 0, false
	}
	physical, ok, err := reader.LastSuccessfulPhysicalBytes(ctx, strings.TrimSpace(planID))
	if err != nil {
		return 0, false
	}
	return physical, ok
}
