package strategy

import (
	"context"
	"fmt"
)

type ConformanceReport struct {
	StrategyID          string   `json:"strategy_id"`
	Status              string   `json:"status"`
	Passed              []string `json:"passed"`
	Failed              []string `json:"failed"`
	Tiers               []string `json:"tiers"`
	ExecutableStepKinds []string `json:"executable_step_kinds"`
	NextActions         []string `json:"next_actions,omitempty"`
	Promotable          bool     `json:"promotable"`
	EvidenceClass       string   `json:"evidence_class"`
	MinimumUsefulFPS    float64  `json:"minimum_useful_fps"`
}

// Verify is the fixed strategy grading suite. It is transport-agnostic: a new
// adapter only implements the floor and declares what its probes prove.
func Verify(ctx context.Context, s Strategy) ConformanceReport {
	d, err := s.Describe(ctx)
	r := ConformanceReport{StrategyID: s.ID(), Passed: []string{}, Failed: []string{}, NextActions: []string{}}
	if err != nil {
		r.Status = StatusUnavailable
		r.Failed = append(r.Failed, "describe: "+err.Error())
		return r
	}
	r.Status, r.Tiers, r.ExecutableStepKinds, r.NextActions, r.Promotable, r.EvidenceClass, r.MinimumUsefulFPS = d.Status, Tiers(d), StepKinds(d), append([]string{}, d.NextActions...), d.Promotable, d.EvidenceClass, d.MinimumUsefulFPS
	if d.Status != StatusAvailable {
		r.Passed = append(r.Passed, "transcript-replay", "unavailable-disposition")
		return r
	}
	if d.Capabilities[CapScreenshot].Status == StatusAvailable {
		observer, ok := s.(Observer)
		if !ok {
			r.Failed = append(r.Failed, "observe: declared screenshot capability without Observer")
		} else if _, err := observer.Observe(ctx); err != nil {
			r.Failed = append(r.Failed, "observe: "+err.Error())
		} else {
			r.Passed = append(r.Passed, "observe")
		}
	}
	if d.Capabilities[CapInput].Status == StatusAvailable {
		actuator, ok := s.(InputActuator)
		if !ok {
			r.Failed = append(r.Failed, "actuate: declared input capability without InputActuator")
		} else if err := actuator.Actuate(ctx, Actuation{Pointer: &PointerEvent{Kind: "move", X: 0, Y: 0}}); err != nil {
			r.Failed = append(r.Failed, "actuate: "+err.Error())
		} else {
			r.Passed = append(r.Passed, "actuate")
		}
	} else {
		if _, ok := s.(Observer); !ok {
			r.Passed = append(r.Passed, "screenless-floor")
		}
		if _, ok := s.(InputActuator); !ok {
			r.Passed = append(r.Passed, "non-actuating-floor")
		}
	}
	if len(r.Failed) > 0 {
		r.Status = StatusDegraded
	}
	return r
}

func ValidateReport(r ConformanceReport) error {
	if r.StrategyID == "" {
		return fmt.Errorf("strategy_id is required")
	}
	if r.Status == "" {
		return fmt.Errorf("status is required")
	}
	if r.Status == StatusAvailable && len(r.Failed) > 0 {
		return fmt.Errorf("available strategy has failed checks")
	}
	return nil
}
