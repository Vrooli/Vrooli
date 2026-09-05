package readiness

import (
	"fmt"
	"time"

	"deployment-manager/crossosgate"
)

// SignalFromCrossOS adapts the bridge-owned cross-OS verdict into the
// deployment readiness checklist. The bridge gate id is retained as the
// evidence/run reference; deployment-manager still owns the aggregate.
func SignalFromCrossOS(verdict crossosgate.Verdict, observedAt time.Time) Signal {
	status := SignalFailed
	if verdict.ProductionReady {
		status = SignalPassed
	}
	detail := fmt.Sprintf("cross-OS gate %s: %s", verdict.GateID, verdict.Verdict)
	if verdict.TimedOut {
		detail += " (timed out)"
	}
	return Signal{ItemID: "ramp-evidence-complete", Status: status, Source: "vrooli-bridge/cross-os", RunID: verdict.GateID, ObservedAt: observedAt.UTC(), Detail: detail}
}
