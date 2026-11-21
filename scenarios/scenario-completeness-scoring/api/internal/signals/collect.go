package signals

import "fmt"

// Collector contributes one section of a Snapshot. Collect must be
// filesystem-only and may return an error for genuinely unexpected input;
// "artifact absent" is expressed by leaving the section's Collected flag
// false, not by erroring.
type Collector interface {
	Name() string
	Collect(snap *Snapshot) error
}

// Service runs the collectors in a fixed order (service, requirements,
// phases, ui), each behind its own circuit breaker. A failing collector
// contributes a Degradation; it never fails the whole snapshot.
type Service struct {
	collectors []Collector
	breakers   *breakerSet
}

// NewService returns a Service with the standard collector set.
func NewService() *Service {
	return newService(
		serviceCollector{},
		requirementsCollector{syncSource: testGenieRequirementsSource{}},
		phasesCollector{},
		uiCollector{},
	)
}

func newService(collectors ...Collector) *Service {
	return &Service{collectors: collectors, breakers: newBreakerSet(nil)}
}

// Collect assembles the signal snapshot for one scenario rooted at root.
// It always returns a usable Snapshot; collector failures appear as
// Degradations.
func (s *Service) Collect(scenario, root string) Snapshot {
	snap := Snapshot{Scenario: scenario, Root: root, Category: defaultCategory}
	for _, c := range s.collectors {
		s.run(c, &snap)
	}
	return snap
}

func (s *Service) run(c Collector, snap *Snapshot) {
	br := s.breakers.get(c.Name())
	if !br.allow() {
		snap.Degradations = append(snap.Degradations, Degradation{
			Collector: c.Name(),
			State:     "open",
			Reason:    "circuit breaker open after repeated failures",
		})
		return
	}
	if err := runRecovered(c, snap); err != nil {
		br.recordFailure()
		snap.Degradations = append(snap.Degradations, Degradation{
			Collector: c.Name(),
			State:     "failed",
			Reason:    err.Error(),
		})
		return
	}
	br.recordSuccess()
}

// runRecovered converts collector panics into failures: malformed input
// must never crash the score path (OT-P0-006).
func runRecovered(c Collector, snap *Snapshot) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("collector panicked: %v", r)
		}
	}()
	return c.Collect(snap)
}
