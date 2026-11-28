package budgets

import (
	"fmt"
	"time"
)

// DeadlineProposal is an evidence-backed candidate for review. It has no
// connection to the runtime budget ladder and cannot change a timeout.
type DeadlineProposal struct {
	Evidence   StratifiedLatencySummary
	SlowestP95 time.Duration
	Headroom   time.Duration
	Candidate  time.Duration
}

// ProposeDeadline qualifies the complete stratified sample, then adds explicit
// caller-selected headroom to the slowest p95 stratum. Requiring headroom as an
// argument avoids embedding an unmeasured multiplier or silently treating a
// descriptive p95 as a runtime deadline. The result is advisory only.
func ProposeDeadline(samples []LatencyObservation, evidencePolicy LatencyEvidencePolicy, headroom time.Duration) (DeadlineProposal, error) {
	if headroom < 0 {
		return DeadlineProposal{}, fmt.Errorf("deadline headroom must not be negative")
	}
	evidence, err := QualifyLatencyEvidence(samples, evidencePolicy)
	if err != nil {
		return DeadlineProposal{}, err
	}
	var slowest time.Duration
	for _, p95 := range evidence.P95 {
		if p95 > slowest {
			slowest = p95
		}
	}
	if slowest <= 0 {
		return DeadlineProposal{}, fmt.Errorf("qualified latency evidence produced no positive p95")
	}
	if headroom > time.Duration(1<<63-1)-slowest {
		return DeadlineProposal{}, fmt.Errorf("deadline proposal overflows time.Duration")
	}
	return DeadlineProposal{Evidence: evidence, SlowestP95: slowest, Headroom: headroom, Candidate: slowest + headroom}, nil
}
