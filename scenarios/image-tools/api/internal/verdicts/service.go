package verdicts

import (
	"context"
	"fmt"

	"image-tools/internal/jobs"
)

// RecentJobs is how many of the latest terminal jobs per (model, operation)
// count as job history. A failed job counts only when it was a fault of the
// model or the runtime: a refusal, a request the model cannot serve as given and
// a restart interruption say nothing about whether the model works. Recent history answers "does it work now"; an old run
// of failures from a since-fixed bug should not mark a model broken forever.
const RecentJobs = 20

type key struct{ model, op string }

// Service derives verdicts from the curated seed and live signals.
type Service struct {
	Seed    *Seed
	Ratings *Ratings
	Policy  Policy
	db      SQLExecutor
}

// NewService builds the verdict service over db (job traces + ratings) and the
// embedded evidence seed.
func NewService(db SQLExecutor) (*Service, error) {
	seed, err := LoadSeed()
	if err != nil {
		return nil, err
	}
	return &Service{Seed: seed, Ratings: NewRatings(db), Policy: DefaultPolicy, db: db}, nil
}

// Board is a point-in-time read of every signal: build one per request, then ask
// it for as many verdicts as the response needs without touching the database
// again.
type Board struct {
	seed    *Seed
	policy  Policy
	signals map[key]Signals
}

// Load reads the live signals once.
func (s *Service) Load(ctx context.Context) (*Board, error) {
	b := &Board{seed: s.Seed, policy: s.Policy, signals: map[key]Signals{}}
	if s.db == nil {
		return b, nil
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT model_id, operation, state FROM (
  SELECT model_id, operation, state,
         ROW_NUMBER() OVER (PARTITION BY model_id, operation ORDER BY recorded_at DESC, job_id DESC) AS rn
  FROM job_trace
  WHERE model_id <> '' AND (state = 'succeeded' OR (state = 'failed' AND failure_kind IN ('', ?)))
) WHERE rn <= ?`, string(jobs.FailureFault), RecentJobs)
	if err != nil {
		return nil, fmt.Errorf("verdicts: read job history: %w", err)
	}
	for rows.Next() {
		var model, op, state string
		if err := rows.Scan(&model, &op, &state); err != nil {
			rows.Close()
			return nil, fmt.Errorf("verdicts: scan job history: %w", err)
		}
		sig := b.signals[key{model, op}]
		if state == "succeeded" {
			sig.Jobs.Succeeded++
		} else {
			sig.Jobs.Failed++
		}
		b.signals[key{model, op}] = sig
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	rated, err := s.db.QueryContext(ctx, `
SELECT model_id, operation, COUNT(*), AVG(score) FROM job_rating WHERE model_id <> '' GROUP BY model_id, operation`)
	if err != nil {
		return nil, fmt.Errorf("verdicts: read ratings: %w", err)
	}
	defer rated.Close()
	for rated.Next() {
		var model, op string
		var stats RatingStats
		if err := rated.Scan(&model, &op, &stats.Count, &stats.Mean); err != nil {
			return nil, fmt.Errorf("verdicts: scan ratings: %w", err)
		}
		sig := b.signals[key{model, op}]
		sig.Ratings = stats
		b.signals[key{model, op}] = sig
	}
	return b, rated.Err()
}

// For is the verdict for one model on one operation.
func (b *Board) For(modelID, operation string) Verdict {
	v := Derive(b.seed.ClaimsFor(modelID, operation), b.signals[key{modelID, operation}], b.policy)
	v.Operation = operation
	return v
}

// ForModel is the model's per-operation verdicts (in the order given) and the
// roll-up across them.
func (b *Board) ForModel(modelID string, operations []string) ([]Verdict, Verdict) {
	per := make([]Verdict, 0, len(operations))
	for _, op := range operations {
		per = append(per, b.For(modelID, op))
	}
	return per, Rollup(per)
}

// UntestedVerdict reports a verdict a caller can use when no board is available (a
// read-only wiring with no database): everything is untested rather than hidden.
func UntestedVerdict(operation string) Verdict {
	return Verdict{Operation: operation, Grade: Untested, Reason: "no qualification, job history or rating yet"}
}
