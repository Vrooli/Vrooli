package methods

import (
	"fmt"
	"sync"
	"web-search/internal/evaluation"
)

const (
	StateInsufficient = "insufficient-evidence"
	StateProposal     = "proposal"
	StateRejected     = "rejected"
	StatePromoted     = "promoted"
)

type Friction struct {
	CycleID, FailureClass string
	SampleCount           int
	Truncated             bool
}
type Cycle struct {
	ID, FailureClass, State, Reason, CandidateHash, Grant string
	Report                                                evaluation.Report
}
type Engine struct {
	mu     sync.Mutex
	cycles map[string]Cycle
}

func NewEngine() *Engine { return &Engine{cycles: map[string]Cycle{}} }

func Route(f Friction) string {
	if f.FailureClass == "fetch" || f.FailureClass == "source_unavailable" {
		return "collection"
	}
	if f.FailureClass == "extraction" {
		return "extraction"
	}
	if f.FailureClass == "verification" {
		return "verification"
	}
	return "method"
}

func (e *Engine) Evaluate(f Friction, candidateHash string, baseline, candidate []evaluation.Observation, grant string) (Cycle, error) {
	if f.CycleID == "" || candidateHash == "" {
		return Cycle{}, fmt.Errorf("cycle identity and candidate hash are required")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if existing, ok := e.cycles[f.CycleID]; ok {
		return existing, nil
	}
	cycle := Cycle{ID: f.CycleID, FailureClass: Route(f), CandidateHash: candidateHash, Grant: grant}
	if f.SampleCount <= 0 || f.Truncated {
		cycle.State, cycle.Reason = StateInsufficient, "insufficient-evidence"
		e.cycles[f.CycleID] = cycle
		return cycle, nil
	}
	report, err := evaluation.Compare(baseline, candidate)
	if err != nil {
		return Cycle{}, err
	}
	cycle.Report = report
	if !report.Accepted {
		cycle.State, cycle.Reason = StateRejected, report.Reason
	} else if grant == "" {
		cycle.State, cycle.Reason = StateProposal, "promotion-grant-required"
	} else {
		cycle.State, cycle.Reason = StatePromoted, "quality-preserved-effort-reduced"
	}
	e.cycles[f.CycleID] = cycle
	return cycle, nil
}
