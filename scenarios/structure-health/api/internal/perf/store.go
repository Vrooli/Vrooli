// Package perf measures scenario startup performance — time-to-healthy plus a
// resource envelope — behind a runner seam, and persists the measurements as a
// per-scenario trend. It is decoupled from validation: it is NEVER invoked by a
// test-genie phase, only by its own RPC / CLI, because it restarts the target
// scenario and is resource-intensive.
package perf

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

//go:embed schema.sql
var schemaSQL string

// Schema returns the declarative DDL for the perf trend store; it is idempotent
// (CREATE TABLE IF NOT EXISTS) so EnsureSchemas can run it on every boot.
func Schema() string { return schemaSQL }

const timeLayout = time.RFC3339Nano

// Executor is the narrow database seam the store needs; *sql.DB satisfies it.
type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// SurfaceTiming is one surface's time-to-reachable during a benchmark.
type SurfaceTiming struct {
	Surface         string `json:"surface"`
	TimeToHealthyMs int64  `json:"time_to_healthy_ms"`
	Healthy         bool   `json:"healthy"`
}

// Measurement is one startup benchmark of one scenario.
type Measurement struct {
	Scenario        string
	CapturedAt      time.Time
	TimeToHealthyMs int64
	Healthy         bool
	SurfaceTimings  []SurfaceTiming
	Metrics         *commonv1.ExecutionMetrics
	Note            string
}

// Store persists and reads startup measurements.
type Store struct {
	db Executor
}

// NewStore binds a store to the database executor seam.
func NewStore(db Executor) *Store { return &Store{db: db} }

// Insert persists one measurement.
func (s *Store) Insert(ctx context.Context, m Measurement) error {
	if s == nil || s.db == nil {
		return errors.New("perf: nil store")
	}
	if m.Scenario == "" {
		return errors.New("perf: scenario is required")
	}
	capturedAt := m.CapturedAt
	if capturedAt.IsZero() {
		capturedAt = time.Now()
	}
	surfacesJSON, err := marshalSurfaceTimings(m.SurfaceTimings)
	if err != nil {
		return err
	}
	metricsJSON, err := marshalMetrics(m.Metrics)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO perf_measurements
			(scenario, captured_at, time_to_healthy_ms, healthy, surface_timings_json, metrics_json, note)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		m.Scenario, capturedAt.UTC().Format(timeLayout), m.TimeToHealthyMs, boolToInt(m.Healthy),
		surfacesJSON, metricsJSON, m.Note,
	)
	if err != nil {
		return fmt.Errorf("perf: insert measurement: %w", err)
	}
	return nil
}

// Series returns a scenario's measurements newest-first, bounded by limit
// (limit <= 0 returns the default page).
func (s *Store) Series(ctx context.Context, scenario string, limit int) ([]Measurement, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("perf: nil store")
	}
	if scenario == "" {
		return nil, errors.New("perf: scenario is required")
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT scenario, captured_at, time_to_healthy_ms, healthy, surface_timings_json, metrics_json, note
		FROM perf_measurements
		WHERE scenario = ?
		ORDER BY captured_at DESC, id DESC
		LIMIT ?`, scenario, limit)
	if err != nil {
		return nil, fmt.Errorf("perf: query series: %w", err)
	}
	defer rows.Close()

	var out []Measurement
	for rows.Next() {
		var (
			m            Measurement
			capturedAt   string
			healthy      int
			surfacesJSON string
			metricsJSON  string
		)
		if scanErr := rows.Scan(&m.Scenario, &capturedAt, &m.TimeToHealthyMs, &healthy, &surfacesJSON, &metricsJSON, &m.Note); scanErr != nil {
			return nil, fmt.Errorf("perf: scan measurement: %w", scanErr)
		}
		if t, perr := time.Parse(timeLayout, capturedAt); perr == nil {
			m.CapturedAt = t
		}
		m.Healthy = healthy != 0
		m.SurfaceTimings = unmarshalSurfaceTimings(surfacesJSON)
		m.Metrics = unmarshalMetrics(metricsJSON)
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("perf: iterate series: %w", err)
	}
	return out, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func marshalSurfaceTimings(timings []SurfaceTiming) (string, error) {
	if len(timings) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal(timings)
	if err != nil {
		return "", fmt.Errorf("perf: marshal surface timings: %w", err)
	}
	return string(b), nil
}

func unmarshalSurfaceTimings(raw string) []SurfaceTiming {
	if raw == "" {
		return nil
	}
	var out []SurfaceTiming
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func marshalMetrics(m *commonv1.ExecutionMetrics) (string, error) {
	if m == nil {
		return "{}", nil
	}
	b, err := protojson.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("perf: marshal metrics: %w", err)
	}
	return string(b), nil
}

func unmarshalMetrics(raw string) *commonv1.ExecutionMetrics {
	if raw == "" || raw == "{}" {
		return nil
	}
	m := &commonv1.ExecutionMetrics{}
	if err := protojson.Unmarshal([]byte(raw), m); err != nil {
		return nil
	}
	return m
}
