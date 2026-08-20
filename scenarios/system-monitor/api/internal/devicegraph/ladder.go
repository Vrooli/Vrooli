package devicegraph

import "time"

// grader stamps every rung state with one observation time so a whole graph
// shares a single timestamp and callers can compare grades across devices.
type grader struct {
	at time.Time
}

func (g grader) measured(rung Rung, mechanism string) RungState {
	return RungState{Rung: rung, State: StateMeasured, Mechanism: mechanism, ObservedAt: g.at}
}

// unmeasurable records that the rung applies and a value should exist, but the
// host refused or could not produce it. The reason is mandatory.
func (g grader) unmeasurable(rung Rung, reason, mechanism string) RungState {
	return RungState{Rung: rung, State: StateUnmeasurable, Reason: reason, Mechanism: mechanism, ObservedAt: g.at}
}

// unavailable records that the mechanism itself is absent from this host.
func (g grader) unavailable(rung Rung, reason, mechanism string) RungState {
	return RungState{Rung: rung, State: StateUnavailable, Reason: reason, Mechanism: mechanism, ObservedAt: g.at}
}

// notApplicable records that the rung is meaningless for this device class.
func (g grader) notApplicable(rung Rung, reason string) RungState {
	return RungState{Rung: rung, State: StateNotApplicable, Reason: reason, ObservedAt: g.at}
}

// remediated attaches the declared commissioning-time fix for a control gap.
func remediated(state RungState, remediation string) RungState {
	state.Remediation = remediation
	return state
}

// evidenceFor derives the evidence rung from the telemetry rung. The monitor
// persists every collected metric payload, so a reading that was taken is
// retained; a reading that could not be taken has nothing to retain and
// inherits the blocking reason rather than reporting a healthy record.
func (g grader) evidenceFor(telemetry RungState) RungState {
	switch telemetry.State {
	case StateMeasured:
		return g.measured(RungEvidence, evidenceMechanism)
	case StateNotApplicable:
		return g.notApplicable(RungEvidence, telemetry.Reason)
	case StateUnavailable:
		return g.unavailable(RungEvidence, "nothing to retain: "+telemetry.Reason, evidenceMechanism)
	default:
		return g.unmeasurable(RungEvidence, "nothing to retain: "+telemetry.Reason, evidenceMechanism)
	}
}

// evidenceMechanism names where a retained device reading ends up. The monitor
// service writes every collector payload through repository SaveMetrics.
const evidenceMechanism = "system-monitor metric repository"

// Remediation strings name declared, commissioning-time host changes. Nothing
// in this package ever escalates privilege at runtime; these are the honest
// descriptions of what `vrooli setup` would have to do to close the gap.
const (
	// RemediationSMARTAccess is the declared fix for a host where the SMART
	// reader exists but the scenario user cannot open the raw device.
	RemediationSMARTAccess = "commission the smartctl host tool (internal/tools/smartctl) and grant the monitor user raw block-device access at setup time; the scenario never escalates at runtime"
	// RemediationSMARTTool is the declared fix for a host with no SMART reader.
	RemediationSMARTTool = "install the smartctl host tool (internal/tools/smartctl) via `vrooli setup`"
	// RemediationECCExposure is the declared fix for a host where the EDAC
	// driver loads but no memory controller registers.
	RemediationECCExposure = "ECC reporting must be enabled in firmware and backed by ECC-capable memory; loading the EDAC driver alone does not register a controller"
)
