package packaged

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

//go:embed schema.sql
var schema string

func Schema() string { return schema }

type Executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type Store struct{ db Executor }

func NewStore(db Executor) *Store { return &Store{db: db} }

func (s *Store) Insert(ctx context.Context, sample Sample) error {
	if s == nil || s.db == nil {
		return errors.New("packaged: nil store")
	}
	if err := sample.Validate(); err != nil {
		return err
	}
	data, err := json.Marshal(sample)
	if err != nil {
		return fmt.Errorf("packaged: encode sample: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO packaged_launch_samples
		(scenario,ramp,strategy,platform,experiment_id,host_fingerprint,artifact_digest,condition,display,profiler_mode,power_state,captured_at,sample_json)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`, sample.Scenario, sample.Ramp, sample.Strategy, sample.Platform,
		sample.ExperimentID, sample.HostFingerprint, sample.ArtifactDigest, sample.Condition, sample.Display,
		sample.ProfilerMode, sample.PowerState, sample.CapturedAt.UTC().Format(time.RFC3339Nano), string(data))
	if err != nil {
		return fmt.Errorf("packaged: insert sample: %w", err)
	}
	return nil
}

func (s *Store) RecordAttempt(ctx context.Context, attempt LaunchAttempt) error {
	if s == nil || s.db == nil {
		return errors.New("packaged: nil store")
	}
	if attempt.Scenario == "" || attempt.Ramp != "desktop" || attempt.Strategy == "" || attempt.Experiment == "" || attempt.Condition != "cold" && attempt.Condition != "warm" || attempt.Attempted.IsZero() {
		return errors.New("packaged: launch attempt requires scenario, desktop ramp, strategy, experiment, condition, and timestamp")
	}
	data, err := json.Marshal(attempt)
	if err != nil {
		return fmt.Errorf("packaged: encode launch attempt: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO packaged_launch_attempts
		(scenario,ramp,strategy,experiment_id,condition,toggle_identity,attempted_at,attempt_json)
		VALUES (?,?,?,?,?,?,?,?)`, attempt.Scenario, attempt.Ramp, attempt.Strategy, attempt.Experiment, attempt.Condition,
		attempt.Toggles.Identity(), attempt.Attempted.UTC().Format(time.RFC3339Nano), string(data))
	if err != nil {
		return fmt.Errorf("packaged: record launch attempt: %w", err)
	}
	return nil
}

func (s *Store) ListAttempts(ctx context.Context, scenario, ramp, strategy, condition, experiment string) ([]LaunchAttempt, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("packaged: nil store")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT attempt_json FROM packaged_launch_attempts
		WHERE (?='' OR scenario=?) AND (?='' OR ramp=?) AND (?='' OR strategy=?) AND (?='' OR condition=?) AND (?='' OR experiment_id=?)
		ORDER BY attempted_at,id`, scenario, scenario, ramp, ramp, strategy, strategy, condition, condition, experiment, experiment)
	if err != nil {
		return nil, fmt.Errorf("packaged: list launch attempts: %w", err)
	}
	defer rows.Close()
	var attempts []LaunchAttempt
	for rows.Next() {
		var raw string
		var attempt LaunchAttempt
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("packaged: scan launch attempt: %w", err)
		}
		if err := json.Unmarshal([]byte(raw), &attempt); err != nil {
			return nil, fmt.Errorf("packaged: decode launch attempt: %w", err)
		}
		attempts = append(attempts, attempt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("packaged: iterate launch attempts: %w", err)
	}
	return attempts, nil
}

func (s *Store) List(ctx context.Context, scenario, ramp, strategy, condition, experiment string) ([]Sample, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("packaged: nil store")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT sample_json FROM packaged_launch_samples
		WHERE (?='' OR scenario=?) AND (?='' OR ramp=?) AND (?='' OR strategy=?) AND (?='' OR condition=?) AND (?='' OR experiment_id=?)
		ORDER BY captured_at,id`, scenario, scenario, ramp, ramp, strategy, strategy, condition, condition, experiment, experiment)
	if err != nil {
		return nil, fmt.Errorf("packaged: list samples: %w", err)
	}
	defer rows.Close()
	var samples []Sample
	for rows.Next() {
		var raw string
		var sample Sample
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("packaged: scan sample: %w", err)
		}
		if err := json.Unmarshal([]byte(raw), &sample); err != nil {
			return nil, fmt.Errorf("packaged: decode sample: %w", err)
		}
		samples = append(samples, sample)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("packaged: iterate samples: %w", err)
	}
	return samples, nil
}
