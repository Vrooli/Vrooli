package trends

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"
)

// Policy is deliberately explicit: a metric must opt in before a trend is
// calculated. Direction describes movement, not whether movement is desirable.
type Policy struct {
	Enabled             bool    `json:"enabled"`
	WindowSeconds       int     `json:"windowSeconds"`
	Comparison          string  `json:"comparison"`
	Aggregation         string  `json:"aggregation"`
	Direction           string  `json:"direction"`
	MinimumObservations int     `json:"minimumObservations"`
	NeutralPercent      float64 `json:"neutralPercent"`
}

type Observation struct {
	MetricID string
	Source   string
	Value    float64
	Observed time.Time
}

type Result struct {
	State        string   `json:"state"`
	Movement     string   `json:"movement,omitempty"`
	Delta        float64  `json:"delta,omitempty"`
	Percent      *float64 `json:"percent,omitempty"`
	Comparison   string   `json:"comparison,omitempty"`
	Polarity     string   `json:"polarity,omitempty"`
	Observations int      `json:"observations,omitempty"`
}

type Store interface {
	Record(context.Context, Observation) error
	Trend(context.Context, string, string, Policy, time.Time) (Result, error)
	Close() error
}

type memoryStore struct {
	mu     sync.Mutex
	values map[string][]Observation
}

func NewMemoryStore() Store { return &memoryStore{values: map[string][]Observation{}} }
func (m *memoryStore) Record(_ context.Context, o Observation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := o.MetricID + "\x00" + o.Source
	for _, existing := range m.values[key] {
		if existing.Observed.Equal(o.Observed) {
			return nil
		}
	}
	m.values[key] = append(m.values[key], o)
	return nil
}
func (m *memoryStore) Trend(_ context.Context, metric, source string, p Policy, at time.Time) (Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return calculate(m.values[metric+"\x00"+source], p, at), nil
}
func (m *memoryStore) Close() error { return nil }

type SQLiteStore struct{ db *sql.DB }

// NewResilientStore keeps observations recorded during a temporary primary
// store outage in process memory. Each later read/write retries those exact
// rows before returning to the durable store; it never re-fetches a producer.
func NewResilientStore(primary Store) Store {
	return &resilientStore{primary: primary, pending: map[string]Observation{}}
}

type resilientStore struct {
	mu      sync.Mutex
	primary Store
	pending map[string]Observation
}

func (s *resilientStore) Record(ctx context.Context, observation Observation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.flush(ctx); err != nil {
		s.retain(observation)
		return nil
	}
	if err := s.primary.Record(ctx, observation); err != nil {
		s.retain(observation)
	}
	return nil
}

func (s *resilientStore) retain(observation Observation) {
	key := observationKey(observation)
	if _, exists := s.pending[key]; !exists {
		s.pending[key] = observation
	}
}

func (s *resilientStore) Trend(ctx context.Context, metric, source string, policy Policy, at time.Time) (Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.flush(ctx); err != nil {
		count := 0
		for _, observation := range s.pending {
			if observation.MetricID == metric && observation.Source == source {
				count++
			}
		}
		return Result{State: "provider_unavailable", Comparison: "previous_window", Observations: count}, nil
	}
	result, err := s.primary.Trend(ctx, metric, source, policy, at)
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func (s *resilientStore) flush(ctx context.Context) error {
	if s.primary == nil {
		return errors.New("trend primary store is unavailable")
	}
	for key, observation := range s.pending {
		if err := s.primary.Record(ctx, observation); err != nil {
			return err
		}
		delete(s.pending, key)
	}
	return nil
}

func (s *resilientStore) Close() error {
	if s.primary == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	flushErr := s.flush(ctx)
	return errors.Join(flushErr, s.primary.Close())
}

func observationKey(o Observation) string {
	return o.MetricID + "\x00" + o.Source + "\x00" + o.Observed.UTC().Format(time.RFC3339Nano)
}

func NewSQLiteStore(db *sql.DB) Store { return &SQLiteStore{db: db} }
func (s *SQLiteStore) Record(ctx context.Context, o Observation) error {
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO command_center_metric_observations (metric_id, source, value, observed_at, recorded_at) VALUES (?, ?, ?, ?, ?)`, o.MetricID, o.Source, o.Value, o.Observed.UTC().Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
func (s *SQLiteStore) Trend(ctx context.Context, metric, source string, p Policy, at time.Time) (Result, error) {
	window := time.Duration(p.WindowSeconds) * time.Second
	rows, err := s.db.QueryContext(ctx, `SELECT value, observed_at FROM command_center_metric_observations WHERE metric_id = ? AND source = ? AND observed_at >= ? AND observed_at < ? ORDER BY observed_at`, metric, source, at.Add(-2*window).UTC().Format(time.RFC3339Nano), at.Add(time.Nanosecond).UTC().Format(time.RFC3339Nano))
	if err != nil {
		return Result{}, err
	}
	defer rows.Close()
	var observations []Observation
	for rows.Next() {
		var value float64
		var raw string
		if err := rows.Scan(&value, &raw); err != nil {
			return Result{}, err
		}
		observed, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return Result{}, err
		}
		observations = append(observations, Observation{MetricID: metric, Source: source, Value: value, Observed: observed})
	}
	if err := rows.Err(); err != nil {
		return Result{}, err
	}
	return calculate(observations, p, at), nil
}
func (s *SQLiteStore) Close() error { return s.db.Close() }

func calculate(values []Observation, p Policy, at time.Time) Result {
	if !p.Enabled || p.WindowSeconds <= 0 || p.Comparison != "previous_window" {
		return Result{State: "not_applicable"}
	}
	min := p.MinimumObservations
	if min < 1 {
		min = 2
	}
	window := time.Duration(p.WindowSeconds) * time.Second
	var current, previous []Observation
	for _, o := range values {
		if o.Observed.After(at) {
			continue
		}
		age := at.Sub(o.Observed)
		if age >= 0 && age < window {
			current = append(current, o)
		} else if age >= window && age < 2*window {
			previous = append(previous, o)
		}
	}
	if len(current) < min || len(previous) < min {
		return Result{State: "insufficient_data", Comparison: "previous_window", Observations: len(current) + len(previous)}
	}
	now := aggregate(current, p.Aggregation)
	before := aggregate(previous, p.Aggregation)
	delta := now - before
	r := Result{State: "meaningful", Delta: delta, Comparison: "previous_window", Observations: len(current) + len(previous)}
	if before != 0 {
		percent := delta / before * 100
		r.Percent = &percent
		if abs(percent) <= p.NeutralPercent {
			r.State = "neutral"
		}
	}
	if delta > 0 {
		r.Movement = "up"
	} else if delta < 0 {
		r.Movement = "down"
	} else {
		r.Movement = "flat"
	}
	if p.Direction == "higher_is_better" && r.Movement != "flat" {
		if r.Movement == "up" {
			r.Polarity = "favorable"
		} else {
			r.Polarity = "unfavorable"
		}
	}
	if p.Direction == "lower_is_better" && r.Movement != "flat" {
		if r.Movement == "down" {
			r.Polarity = "favorable"
		} else {
			r.Polarity = "unfavorable"
		}
	}
	return r
}
func aggregate(values []Observation, mode string) float64 {
	if mode == "average" {
		var total float64
		for _, v := range values {
			total += v.Value
		}
		return total / float64(len(values))
	}
	if mode == "sum" {
		var total float64
		for _, v := range values {
			total += v.Value
		}
		return total
	}
	return values[len(values)-1].Value
}
func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
