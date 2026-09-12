package condition

import (
	"context"
	"fmt"
	"time"
)

type Observation struct {
	ID                string
	CellRef           string
	Value             float64
	Unit              string
	Source            string
	ObservedAt        time.Time
	Trust             TrustVerdict
	Band              Band
	UnavailableReason string
	// BandVerdict is recomputed on every read and never persisted, so
	// tightening a target re-grades its own history.
	BandVerdict BandVerdict
	// OutOfScope marks a reading whose target exists but is outside the
	// derived should-be-supervised set. It stays in every aggregate; only the
	// supervision projection cares about it.
	OutOfScope bool
}

type SourceAvailability struct {
	Source    string
	Available bool
	Reason    string
	CheckedAt time.Time
}

type Source interface {
	Read(ctx context.Context, projection string) ([]Observation, SourceAvailability, error)
}

type ReadingRepository interface {
	Save(context.Context, []Observation) error
	History(context.Context, string, int) ([]Observation, error)
}

type Service struct {
	Source               Source
	Repository           ReadingRepository
	Now                  func() time.Time
	RetentionFloor       time.Duration
	RetentionFloorReason string
	BandResolver         func(Observation) Band
}

type Snapshot struct {
	Readings []Observation
	Sources  []SourceAvailability
	Trust    TrustTriple
	At       time.Time
}

type TrustTriple struct {
	Distribution       map[TrustVerdict]int
	CheckedDenominator int
	Total              int
	CheckedAt          time.Time
}

func (s *Service) Read(ctx context.Context, projection string) Snapshot {
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	if s.Source == nil {
		return Snapshot{Sources: []SourceAvailability{{Source: projection, Available: false, Reason: "condition source is not configured", CheckedAt: now}}, At: now}
	}
	readings, availability, err := s.Source.Read(ctx, projection)
	if err != nil {
		availability.Available = false
		availability.Reason = err.Error()
	}
	if availability.CheckedAt.IsZero() {
		availability.CheckedAt = now
	}
	triple := TrustTriple{Distribution: map[TrustVerdict]int{}, CheckedAt: now, Total: len(readings)}
	for i := range readings {
		if readings[i].Trust == "" {
			readings[i].Trust = TrustUntrusted
		}
		if s.BandResolver != nil {
			readings[i].Band = s.BandResolver(readings[i])
		}
		readings[i].BandVerdict = EvaluateBand(readings[i].Value, readings[i].Trust, readings[i].Band)
		triple.Distribution[readings[i].Trust]++
		if readings[i].Trust != TrustUntrusted && readings[i].Trust != TrustUnavailable {
			triple.CheckedDenominator++
		}
	}
	if s.Repository != nil && len(readings) > 0 {
		if err := s.Repository.Save(ctx, readings); err != nil {
			availability.Available = false
			availability.Reason = fmt.Sprintf("condition readings were received but could not be persisted: %v", err)
		}
	}
	return Snapshot{Readings: readings, Sources: []SourceAvailability{availability}, Trust: triple, At: now}
}

type HistorySnapshot struct {
	Readings           []Observation
	Measurable         bool
	UnmeasurableReason string
}

func (s *Service) History(ctx context.Context, cellRef string, limit int) HistorySnapshot {
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	if s.Repository == nil {
		return HistorySnapshot{UnmeasurableReason: "condition reading history is not persisted"}
	}
	if s.RetentionFloorReason != "" {
		return HistorySnapshot{UnmeasurableReason: s.RetentionFloorReason}
	}
	readings, err := s.Repository.History(ctx, cellRef, limit)
	if err != nil {
		return HistorySnapshot{UnmeasurableReason: err.Error()}
	}
	if len(readings) == 0 {
		return HistorySnapshot{UnmeasurableReason: "no retained readings exist for this cell"}
	}
	if s.RetentionFloor > 0 {
		oldest := readings[len(readings)-1].ObservedAt
		if now.Sub(oldest) < s.RetentionFloor {
			return HistorySnapshot{Readings: readings, UnmeasurableReason: fmt.Sprintf("retention is below the declared floor of %s", s.RetentionFloor)}
		}
	}
	if s.BandResolver != nil {
		for i := range readings {
			readings[i].Band = s.BandResolver(readings[i])
			readings[i].BandVerdict = EvaluateBand(readings[i].Value, readings[i].Trust, readings[i].Band)
		}
	}
	return HistorySnapshot{Readings: readings, Measurable: true}
}
