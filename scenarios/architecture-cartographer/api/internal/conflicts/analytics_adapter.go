package conflicts

import (
	"context"
	"encoding/json"

	"architecture-cartographer/internal/analytics"
)

// NewAnalyticsAdapter constructs an AnalyticsRecorder backed by an
// analytics.Service. The adapter lives in the conflicts package so
// the conflicts service does not import analytics directly — it only
// knows about the AnalyticsRecorder interface — keeping the seam
// inspectable from each side.
func NewAnalyticsAdapter(svc analytics.Service) AnalyticsRecorder {
	return &analyticsAdapter{svc: svc}
}

type analyticsAdapter struct {
	svc analytics.Service
}

func (a *analyticsAdapter) Record(ctx context.Context, scenario, kind, conflictID string, payload map[string]any) {
	if a.svc == nil {
		return
	}
	b, _ := json.Marshal(payload)
	_, _ = a.svc.RecordEvent(ctx, analytics.Event{
		Kind:       toEventKind(kind),
		Scenario:   scenario,
		ConflictID: conflictID,
		Payload:    b,
		Actor:      "conflicts.service",
	})
}

func toEventKind(s string) analytics.EventKind {
	switch s {
	case "conflict_detected":
		return analytics.EventKindConflictDetected
	case "conflict_assigned":
		return analytics.EventKindConflictAssigned
	case "conflict_resolved":
		return analytics.EventKindConflictResolved
	case "conflict_reopened":
		return analytics.EventKindConflictReopened
	case "conflict_force_resolved":
		return analytics.EventKindConflictForceResolved
	default:
		return analytics.EventKind(s)
	}
}
