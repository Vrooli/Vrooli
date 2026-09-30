package research

import (
	"sync"
	"time"
)

// OutcomeMetric records one admitted research operation using bounded numeric
// facts. It intentionally carries no query or source text, keeping operational
// readings low-cardinality and safe to expose through measures.
type OutcomeMetric struct {
	At                 time.Time
	SupportedClaims    int
	AssessedClaims     int
	SupportedQuestions int
	RequiredQuestions  int
	FetchSuccesses     int
	FetchAttempts      int
	Calls              int
	Bytes              int
	Duration           time.Duration
}

type OutcomeMetrics struct {
	mu        sync.Mutex
	events    []OutcomeMetric
	truncated bool
}

const maxOutcomeMetricEvents = 4096

func NewOutcomeMetrics() *OutcomeMetrics { return &OutcomeMetrics{} }

func (m *OutcomeMetrics) Record(event OutcomeMetric) {
	if m == nil {
		return
	}
	event.At = event.At.UTC()
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.events) >= maxOutcomeMetricEvents {
		copy(m.events, m.events[1:])
		m.events = m.events[:len(m.events)-1]
		m.truncated = true
	}
	m.events = append(m.events, event)
}

// OutcomeMetricTotals is the fixed-window aggregate used by owner measures.
type OutcomeMetricTotals struct {
	Attempts, SupportedClaims, AssessedClaims   int
	SupportedQuestions, RequiredQuestions       int
	FetchSuccesses, FetchAttempts, Calls, Bytes int
	Duration                                    time.Duration
}

func (m *OutcomeMetrics) Snapshot(from, to time.Time) (OutcomeMetricTotals, int, bool) {
	if m == nil {
		return OutcomeMetricTotals{}, 0, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var total OutcomeMetricTotals
	for _, event := range m.events {
		if event.At.Before(from) || !event.At.Before(to) {
			continue
		}
		total.Attempts++
		total.SupportedClaims += event.SupportedClaims
		total.AssessedClaims += event.AssessedClaims
		total.SupportedQuestions += event.SupportedQuestions
		total.RequiredQuestions += event.RequiredQuestions
		total.FetchSuccesses += event.FetchSuccesses
		total.FetchAttempts += event.FetchAttempts
		total.Calls += event.Calls
		total.Bytes += event.Bytes
		total.Duration += event.Duration
	}
	return total, total.Attempts, !m.truncated
}
