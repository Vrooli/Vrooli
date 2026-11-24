package catalogcoverage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"react-component-library/internal/gates"
	"react-component-library/internal/sqliterows"
)

// Schema owns the durable browser-backed gate evidence table. Evidence is
// intentionally separate from the file-based catalog join: an absent asset
// must remain representable even when there is no database row for it.
func Schema() string {
	return `
CREATE TABLE IF NOT EXISTS catalog_gate_evidence (
  id TEXT PRIMARY KEY,
  asset_id TEXT NOT NULL,
  target TEXT NOT NULL,
  gate TEXT NOT NULL,
  version TEXT NOT NULL DEFAULT '',
  result TEXT NOT NULL,
  measurement_json TEXT NOT NULL DEFAULT '',
	source_revision TEXT NOT NULL,
	rule_set_digest TEXT NOT NULL DEFAULT '',
	recorded_at TEXT NOT NULL,
  UNIQUE (asset_id, target, gate, version, recorded_at)
);
CREATE INDEX IF NOT EXISTS idx_catalog_gate_evidence_revision
  ON catalog_gate_evidence(source_revision);
`
}

// EvidenceStore is the persisted producer for browser-backed gate outcomes.
type EvidenceStore struct {
	db interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	}
	now func() time.Time
	mu  sync.Mutex
	err error
}

// ExperienceCapture is the small, durable portion of an Experience Manager
// reconciliation record needed by catalog coverage. It deliberately keeps
// the claim type and capture metadata: a passing accessibility claim must not
// be mistaken for a visual capture, and one viewport must not be mistaken for
// responsive coverage.
type ExperienceCapture struct {
	AssetID        string
	Target         string
	Version        string
	ClaimID        string
	ClaimType      string
	Verdict        string
	StateID        string
	ExampleName    string
	Viewport       string
	CaptureRef     string
	CheckedAt      string
	SourceRevision string
}

// ExperienceCaptureFetcher is implemented by the Experience Manager adapter.
// Keeping the network boundary as a callback lets catalog coverage remain
// deterministic and unit-testable when the manager is unavailable.
type ExperienceCaptureFetcher func(context.Context, string, string) ([]ExperienceCapture, error)

func NewEvidenceStore(db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
},
) *EvidenceStore {
	return &EvidenceStore{db: db, now: time.Now}
}

// Database returns the already-open scenario database for gates that need to
// read a domain-owned table. Keeping this seam on EvidenceStore prevents a
// gate from reconstructing a path and accidentally bypassing routed storage.
func (s *EvidenceStore) Database() interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
} {
	if s == nil {
		return nil
	}
	return s.db
}

func (s *EvidenceStore) ensureMeasurementColumn(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("catalog evidence store is not configured")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	columnNames, err := sqliterows.Collect(ctx, s.db, `PRAGMA table_info(catalog_gate_evidence)`, nil, true, func(row sqliterows.Scanner) (string, error) {
		var cid, notNull, pk int
		var name, typ string
		var defaultValue any
		if err := row.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return "", err
		}
		return name, nil
	}, sqliterows.ErrorContext{})
	if err != nil {
		s.rememberSchemaError(err)
		return err
	}
	columns := make(map[string]bool, len(columnNames))
	for _, name := range columnNames {
		columns[name] = true
	}
	if !columns["measurement_json"] {
		if _, err := s.db.ExecContext(ctx, `ALTER TABLE catalog_gate_evidence ADD COLUMN measurement_json TEXT NOT NULL DEFAULT ''`); err != nil {
			s.rememberSchemaError(err)
			return err
		}
	}
	if !columns["rule_set_digest"] {
		if _, err := s.db.ExecContext(ctx, `ALTER TABLE catalog_gate_evidence ADD COLUMN rule_set_digest TEXT NOT NULL DEFAULT ''`); err != nil {
			s.rememberSchemaError(err)
			return err
		}
	}
	return nil
}

// rememberSchemaError caches only stable schema failures. Request-scoped
// cancellation and transient database errors must not poison the store for
// every later request after a client disconnects from a gate matrix.
func (s *EvidenceStore) rememberSchemaError(err error) {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return
	}
	s.err = err
}

// EvidenceFromResult turns a deterministic runner result into independent
// asset observations. A clean inspection must be recorded as pass for every
// inspected asset; otherwise one failing file silently becomes a missing row
// for every clean asset and the score regresses to the old corpus cliff.
func EvidenceFromResult(ctx context.Context, root string, gate GateDefinition, result gates.Result, inspectedAssets []string) ([]GateEvidence, error) {
	_ = ctx
	if gate.Attribution == "corpus" {
		resultValue := "pass"
		if result.Status == "unmeasured" {
			resultValue = "unmeasured"
		} else if len(result.Findings) > 0 || len(result.RunnerError) > 0 {
			resultValue = "fail"
		}
		return []GateEvidence{{
			AssetID: "__corpus__", Target: "corpus", Gate: gate.ID,
			Result: resultValue, SourceRevision: "corpus",
		}}, nil
	}
	assets, err := LoadCatalog(filepath.Join(resolveScenarioRoot(root), "catalog"))
	if err != nil {
		return nil, err
	}
	impls, err := LoadImplementations(filepath.Join(resolveScenarioRoot(root), "library"))
	if err != nil {
		return nil, err
	}
	assetsByID := make(map[string]Asset, len(assets))
	for _, asset := range assets {
		assetsByID[asset.ID] = asset
	}
	ids := make(map[string]bool, len(inspectedAssets)+len(result.InspectedAssets))
	addInspected := func(values []string) {
		for _, id := range values {
			if id == "" {
				continue
			}
			if _, exists := assetsByID[id]; exists {
				ids[id] = true
			}
		}
	}
	addInspected(inspectedAssets)
	addInspected(result.InspectedAssets)
	// Older runners only returned a count. Their declared appliesTo is still a
	// trustworthy inspection boundary, so derive the built asset set here.
	if len(ids) == 0 {
		built := make(map[string]bool, len(impls))
		for _, impl := range impls {
			if impl.CatalogID != "" {
				built[impl.CatalogID] = true
			}
		}
		for id, asset := range assetsByID {
			if built[id] && contains(gate.AppliesTo, asset.Kind) {
				ids[id] = true
			}
		}
	}
	fail := map[string]bool{}
	for _, finding := range result.Findings {
		if finding.AssetID != "" {
			fail[finding.AssetID] = true
		}
	}
	versions := make(map[string]string, len(impls))
	for _, impl := range impls {
		if impl.CatalogID != "" {
			versions[impl.CatalogID] = impl.Latest
		}
	}
	revisions, revErr := BuildRevisionIndex(root)
	if revErr != nil {
		return nil, revErr
	}
	out := make([]GateEvidence, 0, len(ids))
	for id := range ids {
		if !ids[id] || versions[id] == "" {
			continue
		}
		revision, known := revisions[id]
		if !known {
			if isNonCatalogObservation(id) {
				continue
			}
			return nil, fmt.Errorf("catalog asset %q not found", id)
		}
		resultValue := "pass"
		if result.Status == "unmeasured" {
			resultValue = "unmeasured"
		} else if fail[id] {
			resultValue = "fail"
		}
		measurement := ""
		if gate.ID == "composition" {
			measurement = gates.CompositionScoreMetadataJSON(result, id)
		}
		asset := assetsByID[id]
		target := "react-vite"
		if len(asset.Targets) > 0 && asset.Targets[0] != "" {
			target = asset.Targets[0]
		}
		out = append(out, GateEvidence{AssetID: id, Target: target, Gate: gate.ID, Version: versions[id], Result: resultValue, MeasurementJSON: measurement, SourceRevision: revision})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AssetID < out[j].AssetID })
	return out, nil
}

func isNonCatalogObservation(assetID string) bool {
	return strings.HasPrefix(assetID, "__corpus__") ||
		strings.HasPrefix(assetID, "workbench.") ||
		strings.HasPrefix(assetID, "supplemental.")
}

func (s *EvidenceStore) Save(ctx context.Context, evidence []GateEvidence) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("catalog evidence store is not configured")
	}
	if len(evidence) == 0 {
		return nil
	}
	if err := s.ensureMeasurementColumn(ctx); err != nil {
		return err
	}
	now := time.Now
	if s.now != nil {
		now = s.now
	}
	stamp := now().UTC().Format(time.RFC3339Nano)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, item := range evidence {
		if strings.TrimSpace(item.Version) == "" {
			item.Version = "legacy"
		}
		if strings.TrimSpace(item.RecordedAt) == "" {
			item.RecordedAt = stamp
		}
		id := evidenceID(item)
		_, err := tx.ExecContext(ctx, `
INSERT OR IGNORE INTO catalog_gate_evidence(id, asset_id, target, gate, version, result, measurement_json, source_revision, rule_set_digest, recorded_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, item.AssetID, item.Target, item.Gate, item.Version, item.Result, item.MeasurementJSON, item.SourceRevision, item.RuleSetDigest, item.RecordedAt)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
WITH ranked AS (
  SELECT id, ROW_NUMBER() OVER (PARTITION BY asset_id, target, gate, version ORDER BY recorded_at DESC, id DESC) AS rank
  FROM catalog_gate_evidence WHERE asset_id = ? AND target = ? AND gate = ? AND version = ?
)
DELETE FROM catalog_gate_evidence WHERE id IN (SELECT id FROM ranked WHERE rank > 3)`, item.AssetID, item.Target, item.Gate, item.Version); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *EvidenceStore) List(ctx context.Context) ([]GateEvidence, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	if err := s.ensureMeasurementColumn(ctx); err != nil {
		return nil, err
	}
	return sqliterows.Collect(ctx, s.db, `SELECT asset_id, target, gate, version, result, measurement_json, source_revision, rule_set_digest, recorded_at FROM catalog_gate_evidence ORDER BY asset_id, target, gate, recorded_at DESC`, nil, false, func(row sqliterows.Scanner) (GateEvidence, error) {
		var item GateEvidence
		if err := row.Scan(&item.AssetID, &item.Target, &item.Gate, &item.Version, &item.Result, &item.MeasurementJSON, &item.SourceRevision, &item.RuleSetDigest, &item.RecordedAt); err != nil {
			return GateEvidence{}, err
		}
		return item, nil
	})
}

// ScoreHistory is the durable trend projection. Each day carries forward the
// most recent per-asset gate observation, so an asset does not disappear from
// the chart merely because no gate ran that day.
type ScoreHistory struct {
	RecordedAt              string
	Score                   float64
	AssetsAt100             int
	AssetsBelow50           int
	WeightVectorRegenerated bool
	ScoringModelVersion     int
	SourceRevision          string
	InstrumentMoved         int
	KindMismatchCount       int
	Events                  []ScoreHistoryEvent
}

type ScoreHistoryEvent struct {
	Type           string
	AssetID        string
	SourceRevision string
	DeclaredKind   string
	DerivedKind    string
}

func (s *EvidenceStore) ScoreHistory(ctx context.Context, root, since string) ([]ScoreHistory, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	items, err := s.List(ctx)
	if err != nil || len(items) == 0 {
		return nil, err
	}
	assets, err := LoadCatalog(filepath.Join(resolveScenarioRoot(root), "catalog"))
	if err != nil {
		return nil, err
	}
	impls, err := LoadImplementations(filepath.Join(resolveScenarioRoot(root), "library"))
	if err != nil {
		return nil, err
	}
	definitions, err := LoadGateDefinitions(filepath.Join(resolveScenarioRoot(root), "catalog", "config.json"))
	if err != nil {
		return nil, err
	}
	kindMismatches, _ := ReconcileKinds(root, assets, impls)
	start := time.Time{}
	if strings.TrimSpace(since) != "" {
		start, err = time.Parse("2006-01-02", strings.TrimSpace(since))
		if err != nil {
			return nil, fmt.Errorf("parse score history since: %w", err)
		}
	}
	latest := items[0].RecordedAt
	for _, item := range items[1:] {
		if item.RecordedAt > latest {
			latest = item.RecordedAt
		}
	}
	end, err := time.Parse(time.RFC3339Nano, latest)
	if err != nil {
		end = time.Now().UTC()
	}
	if end.Before(time.Now().UTC().Truncate(24 * time.Hour)) {
		end = time.Now().UTC()
	}
	first := time.Time{}
	for _, item := range items {
		stamp, parseErr := time.Parse(time.RFC3339Nano, item.RecordedAt)
		if parseErr != nil {
			continue
		}
		day := stamp.UTC().Truncate(24 * time.Hour)
		if first.IsZero() || day.Before(first) {
			first = day
		}
	}
	if !start.IsZero() && (first.IsZero() || start.After(first)) {
		first = start
	}
	if first.IsZero() {
		first = end.UTC().Truncate(24 * time.Hour)
	}
	if !start.IsZero() && end.Before(start) {
		return nil, nil
	}

	var out []ScoreHistory
	var previousScore float64
	var previousRevision string
	previousExists := false
	for day := first; !day.After(end.UTC().Truncate(24 * time.Hour)); day = day.Add(24 * time.Hour) {
		cutoff := day.Add(24 * time.Hour)
		asOf := make([]GateEvidence, 0, len(items))
		for _, item := range items {
			stamp, parseErr := time.Parse(time.RFC3339Nano, item.RecordedAt)
			if parseErr == nil && stamp.Before(cutoff) {
				asOf = append(asOf, item)
			}
		}
		report := ComputeWithEvidence(assets, impls, asOf, definitions)
		at100, below50 := 0, 0
		for _, row := range report.Rows {
			if row.Bucket != BucketPlannedBuilt {
				continue
			}
			if row.AssetScore >= 1 {
				at100++
			}
			if row.AssetScore < 0.5 {
				below50++
			}
		}
		sourceRevision := evidenceSourceRevision(asOf)
		instrumentMoved := 0
		var events []ScoreHistoryEvent
		if day.Equal(first) {
			for _, mismatch := range kindMismatches {
				events = append(events, ScoreHistoryEvent{Type: "asset-kind-mismatch", AssetID: mismatch.AssetID, DeclaredKind: mismatch.DeclaredKind, DerivedKind: mismatch.DerivedKind})
			}
		}
		if previousExists && report.Score != previousScore && sourceRevision == previousRevision {
			instrumentMoved = 1
			events = append(events, ScoreHistoryEvent{Type: "instrument-moved", SourceRevision: sourceRevision})
		}
		out = append(out, ScoreHistory{
			RecordedAt:              day.Format("2006-01-02"),
			Score:                   report.Score,
			AssetsAt100:             at100,
			AssetsBelow50:           below50,
			WeightVectorRegenerated: day.Equal(first) && fileExists(filepath.Join(resolveScenarioRoot(root), "catalog", "weights.json")),
			ScoringModelVersion:     2,
			SourceRevision:          sourceRevision,
			InstrumentMoved:         instrumentMoved,
			KindMismatchCount:       len(kindMismatches),
			Events:                  events,
		})
		previousScore, previousRevision, previousExists = report.Score, sourceRevision, true
	}
	return out, nil
}

func evidenceSourceRevision(items []GateEvidence) string {
	values := map[string]bool{}
	for _, item := range items {
		if item.AssetID == "__corpus__" || item.SourceRevision == "" {
			continue
		}
		values[item.AssetID+"\x00"+item.SourceRevision] = true
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	sum := sha256.Sum256([]byte(strings.Join(keys, "\x00")))
	return hex.EncodeToString(sum[:])
}

func evidenceID(item GateEvidence) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{item.AssetID, item.Target, item.Gate, item.Version, item.RecordedAt, item.SourceRevision, item.Result}, "\x00")))
	return hex.EncodeToString(sum[:])
}
