package manifest

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"architecture-cartographer/internal/clock"
)

// SQLExecutor is the narrow database surface the manifest repository
// needs. Kept domain-local so swapping engines stays a one-file
// change.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db    SQLExecutor
	clock clock.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db SQLExecutor, clk clock.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

const manifestTimeFormat = time.RFC3339Nano

const (
	upsertManifestSQL = `
INSERT INTO manifests (scenario, version, content_hash, payload, parsed_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(scenario) DO UPDATE SET
  version = excluded.version,
  content_hash = excluded.content_hash,
  payload = excluded.payload,
  parsed_at = excluded.parsed_at`

	selectManifestSQL = `
SELECT scenario, version, content_hash, payload, parsed_at
FROM manifests WHERE scenario = ?`
)

// manifestPayload is the canonical JSON shape persisted to the
// payload BLOB. Wire-format proto would also work but the proto types
// already cover that and JSON keeps the migration story simple.
type manifestPayload struct {
	Domains         []DomainSpec              `json:"domains"`
	SharedSubstrate []string                  `json:"shared_substrate"`
	SignalWeights   map[string]float64        `json:"signal_weights"`
	Thresholds      []Threshold               `json:"thresholds"`
	Transitional    []TransitionalDeclaration `json:"transitional"`
}

func (r *sqliteRepository) SaveManifest(ctx context.Context, m ManifestDefinition) (ManifestDefinition, error) {
	if m.ParsedAt.IsZero() {
		m.ParsedAt = r.clock.Now().UTC()
	}
	if m.Version == ManifestVersionUnspecified {
		m.Version = ManifestVersionV1
	}
	payload, err := json.Marshal(manifestPayload{
		Domains:         m.Domains,
		SharedSubstrate: m.SharedSubstrate,
		SignalWeights:   m.SignalWeights.Weights,
		Thresholds:      m.Thresholds,
		Transitional:    m.Transitional,
	})
	if err != nil {
		return ManifestDefinition{}, fmt.Errorf("encode manifest %q: %w", m.Scenario, err)
	}
	if _, err := r.db.ExecContext(ctx, upsertManifestSQL,
		m.Scenario, string(m.Version), m.ContentHash, payload,
		m.ParsedAt.Format(manifestTimeFormat),
	); err != nil {
		return ManifestDefinition{}, fmt.Errorf("upsert manifest %q: %w", m.Scenario, err)
	}
	return m, nil
}

func (r *sqliteRepository) GetManifest(ctx context.Context, scenario string) (ManifestDefinition, error) {
	row := r.db.QueryRowContext(ctx, selectManifestSQL, scenario)
	m, err := scanManifest(row)
	if errors.Is(err, sql.ErrNoRows) {
		return ManifestDefinition{}, ErrManifestNotFound{Scenario: scenario}
	}
	if err != nil {
		return ManifestDefinition{}, fmt.Errorf("get manifest %q: %w", scenario, err)
	}
	return m, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanManifest(s rowScanner) (ManifestDefinition, error) {
	var (
		m            ManifestDefinition
		versionRaw   string
		payload      []byte
		parsedAtText string
	)
	if err := s.Scan(&m.Scenario, &versionRaw, &m.ContentHash, &payload, &parsedAtText); err != nil {
		return ManifestDefinition{}, err
	}
	m.Version = ManifestVersion(versionRaw)
	t, err := time.Parse(manifestTimeFormat, parsedAtText)
	if err != nil {
		return ManifestDefinition{}, fmt.Errorf("parse parsed_at: %w", err)
	}
	m.ParsedAt = t
	var p manifestPayload
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &p); err != nil {
			return ManifestDefinition{}, fmt.Errorf("decode payload: %w", err)
		}
	}
	m.Domains = p.Domains
	m.SharedSubstrate = p.SharedSubstrate
	m.SignalWeights = SignalWeights{Weights: p.SignalWeights}
	m.Thresholds = p.Thresholds
	m.Transitional = p.Transitional
	return m, nil
}
