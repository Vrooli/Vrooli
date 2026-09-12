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
	if fixture, ok := s.(ConformanceTarget); ok {
		s = fixture.ConformanceTarget()
	}
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
		} else {
			// A screen-bearing adapter is probed with a neutral pointer event;
			// a frameless adapter may legitimately expose only key input (for
			// example Android TV Remote), so its conformance probe uses ENTER.
			probe := Actuation{Pointer: &PointerEvent{Kind: "move", X: 0, Y: 0}}
			if d.Capabilities[CapScreenshot].Status != StatusAvailable {
				probe = Actuation{Key: &KeyEvent{Kind: "press", Key: "ENTER"}}
			}
			if err := actuator.Actuate(ctx, probe); err != nil {
				r.Failed = append(r.Failed, "actuate: "+err.Error())
			} else {
				r.Passed = append(r.Passed, "actuate")
			}
		}
	} else {
		if _, ok := s.(Observer); !ok {
			r.Passed = append(r.Passed, "screenless-floor")
		}
		if _, ok := s.(InputActuator); !ok {
			r.Passed = append(r.Passed, "non-actuating-floor")
		}
	}
	if d.Capabilities[CapProperty].Status == StatusAvailable {
		if _, ok := s.(PropertyActuator); !ok {
			r.Failed = append(r.Failed, "property: declared property capability without PropertyActuator")
		} else if len(d.Properties) == 0 {
			r.Failed = append(r.Failed, "property: available property capability without descriptors")
		} else {
			invalid := false
			for _, descriptor := range d.Properties {
				if descriptor.Name == "" || descriptor.ValueType == "" {
					invalid = true
					break
				}
			}
			if invalid {
				r.Failed = append(r.Failed, "property: descriptor name and value type are required")
			} else {
				r.Passed = append(r.Passed, "property")
			}
		}
	}
	if len(d.Properties) > 0 {
		if _, ok := s.(StateReader); !ok {
			r.Failed = append(r.Failed, "state-reader: property-bearing strategy without StateReader")
		} else {
			r.Passed = append(r.Passed, "state-reader")
		}
	}
	if d.Capabilities[CapSensor].Status == StatusAvailable {
		if _, ok := s.(SensorReader); !ok {
			r.Failed = append(r.Failed, "sensor: declared sensor capability without SensorReader")
		} else {
			r.Passed = append(r.Passed, "sensor")
		}
	}
	if d.Capabilities[CapMedia].Status == StatusAvailable {
		if _, ok := s.(MediaController); !ok {
			r.Failed = append(r.Failed, "media: declared media capability without MediaController")
		} else {
			r.Passed = append(r.Passed, "media")
		}
	}
	if d.Capabilities[CapPairing].Status == StatusAvailable {
		if _, ok := s.(Pairer); !ok {
			r.Failed = append(r.Failed, "pairing: declared pairing capability without Pairer")
		} else {
			r.Passed = append(r.Passed, "pairing")
		}
	}
	observationMode := d.StateObservation.Mode
	if observationMode == "" {
		observationMode = d.ObservationMode
	}
	if observationMode == "push" {
		if _, ok := s.(StateObserver); !ok {
			r.Failed = append(r.Failed, "state-observation: push mode without StateObserver")
		} else {
			r.Passed = append(r.Passed, "state-observation")
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
