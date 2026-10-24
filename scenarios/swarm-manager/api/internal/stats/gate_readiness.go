package stats

// GateReadiness is intentionally a small, closed vocabulary. In particular,
// insufficient evidence is not represented as a zero acceptance rate.
type GateReadiness string

const (
	GateReadinessBelowThreshold     GateReadiness = "below-threshold"
	GateReadinessInsufficientSample GateReadiness = "insufficient-sample"
	GateReadinessReady              GateReadiness = "ready"
)

// EvaluateGateReadiness applies the minimum-sample guard before evaluating the
// acceptance threshold. Callers must pass acceptance computed from attributed
// decision events only.
func EvaluateGateReadiness(acceptance float64, sample, minSample int, threshold float64) GateReadiness {
	if sample < minSample {
		return GateReadinessInsufficientSample
	}
	if acceptance >= threshold {
		return GateReadinessReady
	}
	return GateReadinessBelowThreshold
}
