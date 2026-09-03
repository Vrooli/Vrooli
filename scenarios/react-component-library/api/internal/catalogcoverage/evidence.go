package catalogcoverage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"react-component-library/internal/gates"
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
	db  *sql.DB
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

func NewEvidenceStore(db *sql.DB) *EvidenceStore {
	return &EvidenceStore{db: db, now: time.Now}
}

// Database returns the already-open scenario database for gates that need to
// read a domain-owned table. Keeping this seam on EvidenceStore prevents a
// gate from reconstructing a path and accidentally bypassing routed storage.
func (s *EvidenceStore) Database() *sql.DB {
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
	rows, err := s.db.QueryContext(ctx, `PRAGMA table_info(catalog_gate_evidence)`)
	if err != nil {
		s.rememberSchemaError(err)
		return err
	}
	defer rows.Close()
	columns := map[string]bool{}
	for rows.Next() {
		var cid, notNull, pk int
		var name, typ string
		var defaultValue any
		if scanErr := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); scanErr != nil {
			_ = rows.Close()
			s.rememberSchemaError(scanErr)
			return scanErr
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		s.rememberSchemaError(err)
		return err
	}
	if err := rows.Close(); err != nil {
		s.rememberSchemaError(err)
		return err
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
	ids := map[string]bool{}
	for _, id := range inspectedAssets {
		if id != "" {
			for _, asset := range assets {
				if asset.ID == id {
					ids[id] = true
					break
				}
			}
		}
	}
	for _, id := range result.InspectedAssets {
		if id != "" {
			for _, asset := range assets {
				if asset.ID == id {
					ids[id] = true
					break
				}
			}
		}
	}
	// Older runners only returned a count. Their declared appliesTo is still a
	// trustworthy inspection boundary, so derive the built asset set here.
	if len(ids) == 0 {
		built := map[string]bool{}
		for _, impl := range impls {
			if impl.CatalogID != "" {
				built[impl.CatalogID] = true
			}
		}
		for _, asset := range assets {
			if built[asset.ID] && contains(gate.AppliesTo, asset.Kind) {
				ids[asset.ID] = true
			}
		}
	}
	fail := map[string]bool{}
	for _, finding := range result.Findings {
		if finding.AssetID != "" {
			fail[finding.AssetID] = true
		}
	}
	versions := map[string]string{}
	targets := map[string]string{}
	for _, asset := range assets {
		target := "react-vite"
		if len(asset.Targets) > 0 && asset.Targets[0] != "" {
			target = asset.Targets[0]
		}
		targets[asset.ID] = target
	}
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
		out = append(out, GateEvidence{AssetID: id, Target: targets[id], Gate: gate.ID, Version: versions[id], Result: resultValue, MeasurementJSON: measurement, SourceRevision: revision})
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
	rows, err := s.db.QueryContext(ctx, `SELECT asset_id, target, gate, version, result, measurement_json, source_revision, rule_set_digest, recorded_at FROM catalog_gate_evidence ORDER BY asset_id, target, gate, recorded_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GateEvidence
	for rows.Next() {
		var item GateEvidence
		if err := rows.Scan(&item.AssetID, &item.Target, &item.Gate, &item.Version, &item.Result, &item.MeasurementJSON, &item.SourceRevision, &item.RuleSetDigest, &item.RecordedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func evidenceID(item GateEvidence) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{item.AssetID, item.Target, item.Gate, item.Version, item.RecordedAt, item.SourceRevision, item.Result}, "\x00")))
	return hex.EncodeToString(sum[:])
}

// BuildRevisionIndex hashes every catalog asset's own inputs — its catalog
// declaration, its manifest, and the source and lock of its latest released
// version — and then folds in the revisions of the assets it depends on.
//
// Both halves are load-bearing, and neither used to work. The manifest glob
// was one directory short of where manifests actually live, so the loop that
// hashes the manifest and the version source never executed and an asset's
// revision was a digest of its catalog declaration alone: editing a
// component's source left every cached per-asset gate verdict looking current.
// Folding dependencies in is what makes a foundation change invalidate the
// assets built on it, which the generated per-version locks describe exactly.
//
// The fold is memoized and cycle-guarded. The dependency-rank gate already
// proves the graph is acyclic, so the guard is a safety net rather than an
// expected path — an asset caught in one falls back to its own digest, which
// is stale-safe in the only direction that matters: it can force an extra
// recomputation, never suppress a needed one.
func BuildRevisionIndex(root string) (map[string]string, error) {
	scenarioRoot := resolveScenarioRoot(root)

	own := map[string]hash.Hash{}
	declarationPaths, err := filepath.Glob(filepath.Join(scenarioRoot, "catalog", "assets", "*", "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(declarationPaths)
	for _, path := range declarationPaths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, readErr
		}
		var doc struct {
			Asset struct {
				ID string `json:"id"`
			} `json:"asset"`
		}
		if json.Unmarshal(data, &doc) != nil || doc.Asset.ID == "" {
			continue
		}
		h := sha256.New()
		h.Write(data)
		h.Write([]byte{0})
		own[doc.Asset.ID] = h
	}

	dependencies := map[string][]string{}
	catalogByLibrary := map[string]string{}
	type manifestRecord struct {
		path      string
		data      []byte
		libraryID string
		requires  []string
	}
	var manifests []manifestRecord
	paths, globErr := filepath.Glob(filepath.Join(scenarioRoot, "library", "*", "*", "component.json"))
	if globErr != nil {
		return nil, globErr
	}
	sort.Strings(paths)
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, readErr
		}
		var manifest struct {
			CatalogID string `json:"catalogId"`
			LibraryID string `json:"libraryId"`
			Latest    string `json:"latest"`
		}
		if json.Unmarshal(data, &manifest) != nil {
			continue
		}
		record := manifestRecord{path: path, data: data, libraryID: manifest.LibraryID}
		// Edges come from the version's generated lock, never from the
		// manifest's dependencies array. That array is per-component while
		// immutability is per-version, and it is empty on all 234 manifests
		// in the live tree — folding on it would silently produce a graph
		// with no edges at all. The lock is generated from the version's
		// real imports, so it is the only per-version dependency record
		// that exists.
		if lock, lockErr := os.ReadFile(filepath.Join(filepath.Dir(path), "versions", manifest.Latest, "dependencies.json")); lockErr == nil {
			var doc struct {
				Dependencies []struct {
					LibraryID string `json:"libraryId"`
				} `json:"dependencies"`
			}
			if json.Unmarshal(lock, &doc) == nil {
				for _, dependency := range doc.Dependencies {
					record.requires = append(record.requires, dependency.LibraryID)
				}
			}
		}
		if manifest.CatalogID != "" && manifest.LibraryID != "" {
			catalogByLibrary[manifest.LibraryID] = manifest.CatalogID
		}
		manifests = append(manifests, record)
		if manifest.CatalogID == "" {
			continue
		}
		h, ok := own[manifest.CatalogID]
		if !ok {
			continue
		}
		h.Write(data)
		h.Write([]byte{0})
		// Fold every live version, not only latest. A historical but live
		// major line can be imported by an adopter, so changing it must
		// invalidate evidence just as changing latest does.
		versionDirs, versionGlobErr := filepath.Glob(filepath.Join(filepath.Dir(path), "versions", "*"))
		if versionGlobErr != nil {
			return nil, versionGlobErr
		}
		sort.Strings(versionDirs)
		for _, versionDir := range versionDirs {
			if info, statErr := os.Stat(versionDir); statErr != nil || !info.IsDir() {
				continue
			}
			h.Write([]byte(filepath.Base(versionDir)))
			h.Write([]byte{0})
			for _, versionPath := range versionSourcePaths(versionDir) {
				versionData, versionErr := revisionFileData(versionPath)
				if versionErr != nil {
					return nil, versionErr
				}
				h.Write(versionData)
				h.Write([]byte{0})
			}
		}
	}
	// The manifest's own requires list is the declared edge. It is resolved to
	// catalog ids in a second pass because an asset may be declared after the
	// assets it depends on have already been read.
	for _, record := range manifests {
		catalogID := catalogByLibrary[record.libraryID]
		if catalogID == "" {
			continue
		}
		for _, libraryID := range record.requires {
			if dependency := catalogByLibrary[libraryID]; dependency != "" && dependency != catalogID {
				dependencies[catalogID] = append(dependencies[catalogID], dependency)
			}
		}
	}

	digests := make(map[string]string, len(own))
	for id, h := range own {
		digests[id] = hex.EncodeToString(h.Sum(nil))
	}

	index := make(map[string]string, len(digests))
	visiting := map[string]bool{}
	var resolve func(string) string
	resolve = func(id string) string {
		if revision, ok := index[id]; ok {
			return revision
		}
		if visiting[id] {
			return digests[id]
		}
		visiting[id] = true
		requires := append([]string(nil), dependencies[id]...)
		sort.Strings(requires)
		h := sha256.New()
		h.Write([]byte(digests[id]))
		h.Write([]byte{0})
		seen := map[string]bool{}
		for _, dependency := range requires {
			if seen[dependency] || digests[dependency] == "" {
				continue
			}
			seen[dependency] = true
			h.Write([]byte(resolve(dependency)))
			h.Write([]byte{0})
		}
		delete(visiting, id)
		revision := hex.EncodeToString(h.Sum(nil))
		index[id] = revision
		return revision
	}
	for id := range digests {
		resolve(id)
	}
	return index, nil
}

// CurrentRevision is a content hash for the catalog declaration, its linked
// implementation, and everything that implementation depends on. Persisted
// evidence with another revision is stale.
func CurrentRevision(root, assetID string) (string, error) {
	index, err := BuildRevisionIndex(root)
	if err != nil {
		return "", err
	}
	revision, ok := index[assetID]
	if !ok {
		return "", fmt.Errorf("catalog asset %q not found", assetID)
	}
	return revision, nil
}

// CurrentRevisionForVersion returns the revision of one materialized version
// and the dependency revisions selected by that version's lock. It is the
// identity used by version-pinned component-test reports; the aggregate index
// above remains the cheap latest-version projection for catalog coverage.
func CurrentRevisionForVersion(root, libraryID, version string) (string, error) {
	scenarioRoot := resolveScenarioRoot(root)
	resolver, err := NewVersionRevisionResolver(scenarioRoot)
	if err != nil {
		return "", err
	}
	return resolver.Resolve(libraryID, version)
}

type versionManifest struct {
	path      string
	data      []byte
	catalogID string
}

// VersionRevisionResolver indexes manifests once and resolves materialized
// version revisions through that index. Callers that resolve many versions
// (for example the freshness gate) must share one resolver; constructing one
// per version needlessly rereads the entire manifest corpus.
type VersionRevisionResolver struct {
	root      string
	manifests map[string]versionManifest
	cache     map[string]string
}

func NewVersionRevisionResolver(root string) (*VersionRevisionResolver, error) {
	root = resolveScenarioRoot(root)
	manifestPaths, err := filepath.Glob(filepath.Join(root, "library", "*", "*", "component.json"))
	if err != nil {
		return nil, err
	}
	manifests := make(map[string]versionManifest, len(manifestPaths))
	for _, path := range manifestPaths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, readErr
		}
		var manifest struct {
			LibraryID string `json:"libraryId"`
			CatalogID string `json:"catalogId"`
		}
		if json.Unmarshal(data, &manifest) != nil || manifest.LibraryID == "" {
			continue
		}
		manifests[manifest.LibraryID] = versionManifest{path: path, data: data, catalogID: manifest.CatalogID}
	}
	return &VersionRevisionResolver{root: root, manifests: manifests, cache: map[string]string{}}, nil
}

func (r *VersionRevisionResolver) Resolve(libraryID, version string) (string, error) {
	key := libraryID + "@" + version
	if revision, ok := r.cache[key]; ok {
		return revision, nil
	}
	revision, err := r.resolve(libraryID, version, map[string]bool{})
	if err != nil {
		return "", err
	}
	r.cache[key] = revision
	return revision, nil
}

func (r *VersionRevisionResolver) resolve(libraryID, version string, visiting map[string]bool) (string, error) {
	key := libraryID + "@" + version
	if visiting[key] {
		return "", fmt.Errorf("dependency revision cycle at %s", key)
	}
	visiting[key] = true
	defer delete(visiting, key)

	manifest, ok := r.manifests[libraryID]
	if !ok {
		return "", fmt.Errorf("library asset %q not found", libraryID)
	}
	versionDir := filepath.Join(filepath.Dir(manifest.path), "versions", version)
	if info, statErr := os.Stat(versionDir); statErr != nil || !info.IsDir() {
		return "", fmt.Errorf("version %s@%s is not materialized", libraryID, version)
	}
	h := sha256.New()
	if manifest.catalogID != "" {
		parts := strings.SplitN(manifest.catalogID, ".", 2)
		if len(parts) == 2 {
			declaration := filepath.Join(r.root, "catalog", "assets", parts[0], parts[1]+".json")
			if data, readErr := os.ReadFile(declaration); readErr == nil {
				h.Write(data)
				h.Write([]byte{0})
			}
		}
	}
	h.Write(manifest.data)
	h.Write([]byte{0})
	for _, sourcePath := range versionSourcePaths(versionDir) {
		data, readErr := revisionFileData(sourcePath)
		if readErr != nil {
			return "", readErr
		}
		h.Write([]byte(filepath.Base(sourcePath)))
		h.Write([]byte{0})
		h.Write(data)
		h.Write([]byte{0})
	}
	lockData, lockErr := os.ReadFile(filepath.Join(versionDir, "dependencies.json"))
	if lockErr == nil {
		var lock struct {
			Dependencies []struct {
				LibraryID string `json:"libraryId"`
				Version   string `json:"version"`
				Major     int    `json:"major"`
				Observed  string `json:"observed"`
			} `json:"dependencies"`
		}
		if json.Unmarshal(lockData, &lock) == nil {
			for _, dependency := range lock.Dependencies {
				observed := dependency.Observed
				if observed == "" {
					observed = dependency.Version
				}
				if dependency.Major > 0 {
					observed = resolveMajorVersion(r.root, dependency.LibraryID, dependency.Major, observed)
				}
				child, childErr := r.resolve(dependency.LibraryID, observed, visiting)
				if childErr != nil {
					return "", childErr
				}
				h.Write([]byte(dependency.LibraryID))
				h.Write([]byte{0})
				h.Write([]byte(child))
				h.Write([]byte{0})
			}
		}
	}
	revision := hex.EncodeToString(h.Sum(nil))
	r.cache[key] = revision
	return revision, nil
}

func resolveMajorVersion(root, libraryID string, major int, fallback string) string {
	name := strings.TrimPrefix(libraryID, "react-component-library:")
	paths, _ := filepath.Glob(filepath.Join(root, "library", "*", name, "versions", fmt.Sprintf("%d.*", major)))
	versions := make([]string, 0, len(paths))
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			versions = append(versions, filepath.Base(path))
		}
	}
	sort.Strings(versions)
	if len(versions) == 0 {
		return fallback
	}
	return versions[len(versions)-1]
}

// resolveScenarioRoot accepts the repository root used by API handlers, the
// scenario root used by local tools, and the library root used by the
// component service. Keeping this normalization at the revision boundary
// prevents evidence from silently disappearing when those callers use their
// own legitimate source-root abstraction.
func resolveScenarioRoot(root string) string {
	clean := filepath.Clean(root)
	if filepath.Base(clean) == "library" {
		return filepath.Dir(clean)
	}
	if _, err := os.Stat(filepath.Join(clean, "catalog")); err == nil {
		return clean
	}
	candidate := filepath.Join(clean, "scenarios", "react-component-library")
	if _, err := os.Stat(filepath.Join(candidate, "catalog")); err == nil {
		return candidate
	}
	return clean
}

func versionSourcePaths(versionDir string) []string {
	var paths []string
	for _, extension := range []string{"*.tsx", "*.ts"} {
		matches, _ := filepath.Glob(filepath.Join(versionDir, extension))
		paths = append(paths, matches...)
	}
	if lockPath := filepath.Join(versionDir, "dependencies.json"); fileExists(lockPath) {
		paths = append(paths, lockPath)
	}
	sort.Strings(paths)
	return paths
}

// revisionFileData removes generator bookkeeping that is intentionally
// refreshed on every catalog build. The lock's dependency choices remain in
// the digest; only resolvedAt is non-semantic and must not invalidate test
// evidence or cache entries by itself.
func revisionFileData(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil || filepath.Base(path) != "dependencies.json" {
		return data, err
	}
	var lock map[string]any
	if json.Unmarshal(data, &lock) != nil {
		return data, nil
	}
	delete(lock, "resolvedAt")
	canonical, marshalErr := json.Marshal(lock)
	if marshalErr != nil {
		return data, nil
	}
	return canonical, nil
}

// MergedEvidence combines fresh persisted browser outcomes with cheap,
// recomputed catalog gates. Persisted rows are filtered by current revision.
func MergedEvidence(ctx context.Context, root string, store *EvidenceStore) ([]GateEvidence, error) {
	var runtimeDB *sql.DB
	var persisted []GateEvidence
	// One revision index serves the whole pass. Every per-asset freshness
	// decision below is a map lookup against it rather than its own corpus
	// walk, which is what makes per-asset attribution affordable at all.
	revisions, err := BuildRevisionIndex(root)
	if err != nil {
		return nil, err
	}
	var stale map[string]map[string]bool
	if store != nil {
		runtimeDB = store.Database()
		persisted, err = store.List(ctx)
		if err != nil {
			return nil, err
		}
		definitions, definitionErr := LoadGateDefinitions(filepath.Join(resolveScenarioRoot(root), "catalog", "config.json"))
		if definitionErr != nil {
			return nil, definitionErr
		}
		stale = staleAssetsByGate(root, persisted, definitions, revisions)
	}
	computed, err := recomputeEvidenceWithSkip(root, runtimeDB, stale, revisions)
	if err != nil {
		return nil, err
	}
	if store == nil {
		return computed, nil
	}
	if len(computed) > 0 {
		var attributableRows []GateEvidence
		for _, item := range computed {
			if item.AssetID != "__corpus__" {
				attributableRows = append(attributableRows, item)
			}
		}
		if err := store.Save(ctx, attributableRows); err != nil {
			return nil, err
		}
		persisted, err = store.List(ctx)
		if err != nil {
			return nil, err
		}
	}
	for _, item := range persisted {
		if item.AssetID == "__corpus__" && item.SourceRevision == "corpus" {
			computed = append(computed, item)
			continue
		}
		// Older evidence producers could persist pseudo-assets such as
		// workbench.conformance. They are corpus observations, not catalog
		// assets, and must never enter the attributable revision join.
		if _, known := revisions[item.AssetID]; !known {
			continue
		}
		if revisions[item.AssetID] != item.SourceRevision {
			continue
		}
		computed = append(computed, item)
	}
	return computed, nil
}

// staleAssetsByGate returns, for every attributable gate, the assets whose
// persisted verdict can no longer be trusted: no evidence recorded, or
// evidence recorded against a different revision. An entry with an empty set
// means every applicable asset is current and the runner can be skipped
// outright.
//
// This used to be an all-or-nothing question — a gate counted as fresh only if
// every applicable asset was current, so one edited component re-ran every
// runner across the whole corpus. Corpus gates are absent from the result
// because their verdict is not attributable to an individual asset and must
// always be recomputed.
func staleAssetsByGate(root string, persisted []GateEvidence, definitions []GateDefinition, revisions map[string]string) map[string]map[string]bool {
	assets, err := LoadCatalog(filepath.Join(resolveScenarioRoot(root), "catalog"))
	if err != nil {
		return nil
	}
	// Track exactly the assets the recompute loop will visit, which is the
	// built ones: that loop skips any catalog asset with no implementation, so
	// tracking more here would mark a never-emitted row permanently stale and
	// keep every runner warm-proof. Tracking fewer would drop a row on warm
	// passes and silently move the reported rungs. The two sets have to be the
	// same set, and TestWarmPassProducesIdenticalEvidenceToColdPass is what
	// holds them together.
	impls, err := LoadImplementations(filepath.Join(resolveScenarioRoot(root), "library"))
	if err != nil {
		return nil
	}
	built := make(map[string]bool, len(impls))
	for _, impl := range impls {
		if impl.CatalogID != "" {
			built[impl.CatalogID] = true
		}
	}
	bindingsByAsset := make(map[string]map[string]RuleBinding, len(assets))
	for _, asset := range assets {
		if !built[asset.ID] {
			continue
		}
		bindings, resolveErr := ResolveRuleSet(root, asset.ID)
		if resolveErr != nil {
			// Small isolated fixtures used by the evidence tests often provide
			// catalog assets without the production config.json. Preserve the
			// legacy definition-driven applicability in that case; production
			// catalogs still use the resolver above as the source of truth.
			for _, definition := range definitions {
				if definition.Attribution == "corpus" || !containsKind(definition.AppliesTo, asset.Kind) {
					continue
				}
				bindings = append(bindings, RuleBinding{GateID: definition.ID})
			}
		}
		bindingsByAsset[asset.ID] = make(map[string]RuleBinding, len(bindings))
		for _, binding := range bindings {
			bindingsByAsset[asset.ID][binding.GateID] = binding
		}
	}
	byGateAsset := map[string]map[string]GateEvidence{}
	for _, item := range persisted {
		if item.AssetID == "__corpus__" {
			continue
		}
		if byGateAsset[item.Gate] == nil {
			byGateAsset[item.Gate] = map[string]GateEvidence{}
		}
		if current, ok := byGateAsset[item.Gate][item.AssetID]; !ok || item.RecordedAt > current.RecordedAt {
			byGateAsset[item.Gate][item.AssetID] = item
		}
	}
	out := map[string]map[string]bool{}
	for _, definition := range definitions {
		if definition.Attribution == "corpus" {
			continue
		}
		gateStale := map[string]bool{}
		for assetID := range bindingsByAsset {
			if _, applicable := bindingsByAsset[assetID][definition.ID]; !applicable {
				continue
			}
			revision, known := revisions[assetID]
			if !known {
				// An asset with no computable revision can never be shown to be
				// current, so it is always recomputed rather than assumed good.
				gateStale[assetID] = true
				continue
			}
			item, ok := byGateAsset[definition.ID][assetID]
			digest, digestErr := RuleSetDigest(root, assetID)
			digestStale := digestErr == nil && (item.RuleSetDigest == "" || digest != item.RuleSetDigest)
			if !ok || revision != item.SourceRevision || digestStale {
				gateStale[assetID] = true
			}
		}
		out[definition.ID] = gateStale
	}
	return out
}

// MergeExperienceEvidence adds only evidence that is both declared by a
// catalog gate and observed by Experience Manager. A manager outage is
// intentionally non-fatal: it leaves the gate absent (and therefore below
// target) instead of converting an unavailable capture into a pass.
func MergeExperienceEvidence(ctx context.Context, root string, store *EvidenceStore, fetch ExperienceCaptureFetcher) ([]GateEvidence, error) {
	evidence, err := MergedEvidence(ctx, root, store)
	if err != nil || fetch == nil {
		return evidence, err
	}
	assets, err := LoadCatalog(filepath.Join(resolveScenarioRoot(root), "catalog"))
	if err != nil {
		return nil, err
	}
	impls, err := LoadImplementations(filepath.Join(resolveScenarioRoot(root), "library"))
	if err != nil {
		return nil, err
	}
	gates, err := LoadGateDefinitions(filepath.Join(resolveScenarioRoot(root), "catalog", "config.json"))
	if err != nil {
		return nil, err
	}
	targets := map[string]string{}
	for _, asset := range assets {
		target := "react-vite"
		if len(asset.Targets) > 0 && strings.TrimSpace(asset.Targets[0]) != "" {
			target = asset.Targets[0]
		}
		targets[asset.ID] = target
	}
	// Experience Manager evidence is an optional enrichment, but the catalog
	// contains hundreds of exact-version implementations. Fetching them in a
	// single serial loop can consume the validation RPC deadline before the
	// component gate even starts. Keep the fan-out bounded so the manager and
	// its database are protected while the validation path remains responsive.
	type fetchJob struct {
		libraryID string
		assetID   string
		target    string
		version   string
	}
	type fetchResult struct{ captures []ExperienceCapture }
	jobs := make(chan fetchJob)
	results := make(chan fetchResult)
	const workers = 8
	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for job := range jobs {
				if ctx.Err() != nil {
					return
				}
				fetched, fetchErr := fetch(ctx, job.libraryID, job.version)
				if fetchErr != nil {
					continue
				}
				for index := range fetched {
					fetched[index].AssetID = job.assetID
					fetched[index].Target = job.target
					fetched[index].Version = job.version
				}
				results <- fetchResult{captures: fetched}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, impl := range impls {
			if impl.CatalogID == "" || impl.Latest == "" || targets[impl.CatalogID] == "" || ctx.Err() != nil {
				continue
			}
			job := fetchJob{libraryID: "react-component-library:" + impl.Name, assetID: impl.CatalogID, target: targets[impl.CatalogID], version: impl.Latest}
			select {
			case jobs <- job:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		wait.Wait()
		close(results)
	}()
	var captures []ExperienceCapture
	for result := range results {
		captures = append(captures, result.captures...)
	}
	fresh, err := deriveExperienceEvidence(root, captures, gates)
	if err != nil {
		return nil, err
	}
	if len(fresh) == 0 {
		return evidence, nil
	}
	if store != nil {
		if err := store.Save(ctx, fresh); err != nil {
			return nil, err
		}
	}
	return append(evidence, fresh...), nil
}

func deriveExperienceEvidence(root string, captures []ExperienceCapture, definitions []GateDefinition) ([]GateEvidence, error) {
	revisions, revisionErr := BuildRevisionIndex(root)
	if revisionErr != nil {
		return nil, revisionErr
	}
	// Experience Manager retains an audit history. Reduce it to the newest
	// observation for each declared claim/state/example/viewport before a gate
	// is evaluated; otherwise one old skipped capture can poison a current pass.
	latest := map[string]ExperienceCapture{}
	for _, capture := range captures {
		// Experience Manager may retain observations for workbench or
		// supplemental surfaces that are not catalog assets. They cannot be
		// scored by this catalog and must not make the evidence join fail.
		if _, known := revisions[capture.AssetID]; !known {
			continue
		}
		claim := capture.ClaimID
		if claim == "" {
			claim = capture.ClaimType
		}
		key := strings.Join([]string{capture.AssetID, capture.Target, claim, capture.StateID, capture.ExampleName, capture.Viewport}, "\x00")
		current, exists := latest[key]
		if !exists || capture.CheckedAt > current.CheckedAt {
			latest[key] = capture
		}
	}
	type aggregate struct {
		captures []ExperienceCapture
	}
	groups := map[string]*aggregate{}
	for _, capture := range latest {
		for _, definition := range definitions {
			if !contains(definition.ExperienceClaimTypes, capture.ClaimType) {
				continue
			}
			key := capture.AssetID + "\x00" + capture.Target + "\x00" + definition.ID
			if groups[key] == nil {
				groups[key] = &aggregate{}
			}
			groups[key].captures = append(groups[key].captures, capture)
		}
	}
	var out []GateEvidence
	for key, group := range groups {
		parts := strings.Split(key, "\x00")
		if len(parts) != 3 || len(group.captures) == 0 {
			continue
		}
		assetID, target, gateID := parts[0], parts[1], parts[2]
		var definition GateDefinition
		for _, candidate := range definitions {
			if candidate.ID == gateID {
				definition = candidate
				break
			}
		}
		result := "pass"
		viewports := map[string]struct{}{}
		for _, capture := range group.captures {
			if capture.Viewport != "" {
				viewports[capture.Viewport] = struct{}{}
			}
			switch strings.ToLower(strings.TrimSpace(capture.Verdict)) {
			case "failed", "fail", "error":
				result = "fail"
			case "blocked", "skipped", "not-run":
				if result == "pass" {
					result = "skipped"
				}
			}
			if definition.requiresCapture() && strings.TrimSpace(capture.CaptureRef) == "" && result == "pass" {
				result = "skipped"
			}
		}
		if definition.minimumViewports() > 0 && len(viewports) < definition.minimumViewports() && result == "pass" {
			result = "skipped"
		}
		revision, known := revisions[assetID]
		if !known {
			return nil, fmt.Errorf("catalog asset %q not found", assetID)
		}
		out = append(out, GateEvidence{AssetID: assetID, Target: target, Version: group.captures[0].Version, Gate: gateID, Result: result, SourceRevision: revision})
	}
	// A completed, passing capture matrix is also direct evidence for the
	// capture-backed visual and responsive gates. These gates used to depend on
	// a claim type being present in a particular contract, which meant a
	// perfectly captured component with only element-presence claims remained
	// permanently below verified. Keep the proof conservative: every latest
	// observation must pass, every observation must have a screenshot reference,
	// and responsive evidence must span the configured viewport count.
	type matrix struct {
		allPassed bool
		refs      int
		viewports map[string]struct{}
	}
	matrixByAsset := map[string]*matrix{}
	for _, capture := range latest {
		if _, known := revisions[capture.AssetID]; !known {
			continue
		}
		key := capture.AssetID + "\x00" + capture.Target
		entry := matrixByAsset[key]
		if entry == nil {
			entry = &matrix{allPassed: true, viewports: map[string]struct{}{}}
			matrixByAsset[key] = entry
		}
		if !strings.EqualFold(strings.TrimSpace(capture.Verdict), "passed") {
			entry.allPassed = false
		}
		if strings.TrimSpace(capture.CaptureRef) == "" {
			entry.allPassed = false
		} else {
			entry.refs++
		}
		if strings.TrimSpace(capture.Viewport) != "" {
			entry.viewports[capture.Viewport] = struct{}{}
		}
	}
	for key, entry := range matrixByAsset {
		if !entry.allPassed || entry.refs == 0 {
			continue
		}
		parts := strings.Split(key, "\x00")
		if len(parts) != 2 {
			continue
		}
		assetID, target := parts[0], parts[1]
		revision, known := revisions[assetID]
		if !known {
			return nil, fmt.Errorf("catalog asset %q not found", assetID)
		}
		for _, definition := range definitions {
			if definition.ID != "visual" && definition.ID != "responsive" {
				continue
			}
			if definition.ID == "responsive" && len(entry.viewports) < definition.minimumViewports() {
				continue
			}
			out = append(out, GateEvidence{AssetID: assetID, Target: target, Version: latestCaptureVersion(latest, assetID, target), Gate: definition.ID, Result: "pass", SourceRevision: revision})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].AssetID != out[j].AssetID {
			return out[i].AssetID < out[j].AssetID
		}
		return out[i].Gate < out[j].Gate
	})
	return out, nil
}

func latestCaptureVersion(captures map[string]ExperienceCapture, assetID, target string) string {
	for _, capture := range captures {
		if capture.AssetID == assetID && capture.Target == target && capture.Version != "" {
			return capture.Version
		}
	}
	return "legacy"
}

func RecomputeEvidence(root string) ([]GateEvidence, error) {
	return recomputeEvidence(root, nil)
}

func recomputeEvidence(root string, runtimeDB *sql.DB) ([]GateEvidence, error) {
	revisions, err := BuildRevisionIndex(root)
	if err != nil {
		return nil, err
	}
	return recomputeEvidenceWithSkip(root, runtimeDB, nil, revisions)
}

// recomputeEvidenceWithSkip runs the gate corpus. When stale is nil every
// runner executes, which is the cold path and the path RecomputeEvidence takes.
// When stale is present a runner is skipped only if no applicable asset needs
// it; the runners themselves are whole-corpus by construction, so the saving is
// the runner's entire cost or nothing.
func recomputeEvidenceWithSkip(root string, runtimeDB *sql.DB, stale map[string]map[string]bool, revisions map[string]string) ([]GateEvidence, error) {
	// needed reports whether a runner must execute this pass. A gate the
	// freshness pass knows nothing about — every corpus gate, and any gate a
	// stale map does not mention — always runs.
	needed := func(gate string) bool {
		if stale == nil {
			return true
		}
		assets, known := stale[gate]
		if !known {
			return true
		}
		return len(assets) > 0
	}
	assets, err := LoadCatalog(filepath.Join(root, "scenarios", "react-component-library", "catalog"))
	if err != nil {
		return nil, err
	}
	impls, err := LoadImplementations(filepath.Join(root, "scenarios", "react-component-library", "library"))
	if err != nil {
		return nil, err
	}
	implByAsset := map[string]Implementation{}
	for _, impl := range impls {
		if impl.CatalogID != "" {
			implByAsset[impl.CatalogID] = impl
		}
	}
	definitions, err := LoadGateDefinitions(filepath.Join(root, "scenarios", "react-component-library", "catalog", "config.json"))
	if err != nil {
		return nil, err
	}
	runners := map[string]gates.Result{}
	bindingsByAsset := make(map[string]map[string]RuleBinding, len(assets))
	for _, asset := range assets {
		if _, ok := implByAsset[asset.ID]; !ok {
			continue
		}
		bindings, resolveErr := ResolveRuleSet(root, asset.ID)
		if resolveErr != nil {
			return nil, resolveErr
		}
		bindingsByAsset[asset.ID] = make(map[string]RuleBinding, len(bindings))
		for _, binding := range bindings {
			bindingsByAsset[asset.ID][binding.GateID] = binding
		}
	}
	for _, definition := range definitions {
		if !needed(definition.ID) && definition.Attribution != "corpus" {
			continue
		}
		var selected []string
		if definition.Attribution != "corpus" && stale != nil {
			for assetID := range stale[definition.ID] {
				selected = append(selected, assetID)
			}
			sort.Strings(selected)
		}
		result, available, runErr := gates.Run(definition.ID, gates.Scope{Root: root, Assets: selected, DB: runtimeDB})
		if runErr != nil {
			return nil, runErr
		}
		if available {
			result = gates.NormalizeResult(root, result)
			if err := AnnotateFindings(root, definition.ID, &result); err != nil {
				return nil, err
			}
			runners[definition.ID] = result
		}
	}
	for name, runner := range runners {
		runners[name] = gates.NormalizeResult(root, runner)
	}
	quarantined := map[string]bool{}
	for _, definition := range definitions {
		if !definition.Blocking {
			continue
		}
		runner := gates.GateRunnerFor(definition.ID)
		calibration, calibrationErr := gates.Calibrate(root, definition.ID, runner)
		if calibrationErr != nil {
			return nil, calibrationErr
		}
		if calibration.NonDiscriminating {
			quarantined[definition.ID] = true
		}
	}
	var out []GateEvidence
	// Gates without a declared runner are explicitly unmeasured. They are
	// still represented for every applicable built asset (and for corpus gates)
	// so the score cannot improve when a runner disappears.
	for _, asset := range assets {
		if _, ok := implByAsset[asset.ID]; !ok {
			continue
		}
		target := "react-vite"
		if len(asset.Targets) > 0 {
			target = asset.Targets[0]
		}
		revision, known := revisions[asset.ID]
		if !known {
			return nil, fmt.Errorf("catalog asset %q has no computable revision", asset.ID)
		}
		for _, definition := range definitions {
			binding, applicable := bindingsByAsset[asset.ID][definition.ID]
			if !applicable || binding.Source == RuleSourceCorpus {
				continue
			}
			gateName := definition.ID
			// An asset the freshness pass found current keeps the verdict
			// already in the evidence store. Emitting nothing here is what
			// makes that verdict survive; emitting "unmeasured" would quietly
			// lower the asset's rung on every warm pass.
			if stale != nil {
				if assets, known := stale[gateName]; known && !assets[asset.ID] {
					continue
				}
			}
			runner, ok := runners[gateName]
			if !ok || quarantined[gateName] {
				out = append(out, GateEvidence{AssetID: asset.ID, Target: target, Version: implByAsset[asset.ID].Latest, Gate: gateName, Result: "unmeasured", SourceRevision: revision})
				continue
			}
			result := "pass"
			if contains(runner.UnmeasuredAssets, asset.ID) || (runner.Status == "unmeasured" && len(runner.UnmeasuredAssets) == 0) {
				result = "unmeasured"
			} else if hasFinding(runner.Findings, asset.ID, implByAsset[asset.ID].Name, gateName) || len(runner.RunnerError) > 0 {
				result = "fail"
			} else if runner.Inspected == 0 {
				result = "skipped"
			}
			out = append(out, GateEvidence{AssetID: asset.ID, Target: target, Version: implByAsset[asset.ID].Latest, Gate: gateName, Result: result, SourceRevision: revision})
		}
	}
	for _, definition := range definitions {
		if definition.Attribution != "corpus" || (len(definition.Runner) > 0 && !quarantined[definition.ID]) {
			continue
		}
		out = append(out, GateEvidence{AssetID: "__corpus__", Target: "corpus", Version: "", Gate: definition.ID, Result: "unmeasured", SourceRevision: "corpus"})
	}
	for i := range out {
		if out[i].AssetID == "__corpus__" {
			continue
		}
		digest, err := RuleSetDigest(root, out[i].AssetID)
		if err != nil {
			return nil, err
		}
		out[i].RuleSetDigest = digest
	}
	return out, nil
}

func hasFinding(findings []gates.Finding, assetID, implementation, gate string) bool {
	for _, finding := range findings {
		if finding.AssetID == assetID || finding.AssetID == implementation {
			return true
		}
	}
	return false
}

func containsKind(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
