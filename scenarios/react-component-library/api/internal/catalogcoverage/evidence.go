package catalogcoverage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
		s.err = err
		return err
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var cid, notNull, pk int
		var name, typ string
		var defaultValue any
		if scanErr := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); scanErr != nil {
			_ = rows.Close()
			s.err = scanErr
			return scanErr
		}
		if name == "measurement_json" {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		s.err = err
		return err
	}
	if err := rows.Close(); err != nil {
		s.err = err
		return err
	}
	if !found {
		if _, err := s.db.ExecContext(ctx, `ALTER TABLE catalog_gate_evidence ADD COLUMN measurement_json TEXT NOT NULL DEFAULT ''`); err != nil {
			s.err = err
			return err
		}
	}
	return nil
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
	out := make([]GateEvidence, 0, len(ids))
	for id := range ids {
		if !ids[id] || versions[id] == "" {
			continue
		}
		revision, revErr := CurrentRevision(root, id)
		if revErr != nil {
			return nil, revErr
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
INSERT OR IGNORE INTO catalog_gate_evidence(id, asset_id, target, gate, version, result, measurement_json, source_revision, recorded_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, item.AssetID, item.Target, item.Gate, item.Version, item.Result, item.MeasurementJSON, item.SourceRevision, item.RecordedAt)
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
	rows, err := s.db.QueryContext(ctx, `SELECT asset_id, target, gate, version, result, measurement_json, source_revision, recorded_at FROM catalog_gate_evidence ORDER BY asset_id, target, gate, recorded_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GateEvidence
	for rows.Next() {
		var item GateEvidence
		if err := rows.Scan(&item.AssetID, &item.Target, &item.Gate, &item.Version, &item.Result, &item.MeasurementJSON, &item.SourceRevision, &item.RecordedAt); err != nil {
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

// CurrentRevision is a content hash for the catalog declaration and its
// linked implementation. Persisted evidence with another revision is stale.
func CurrentRevision(root, assetID string) (string, error) {
	scenarioRoot := resolveScenarioRoot(root)
	paths, err := filepath.Glob(filepath.Join(scenarioRoot, "catalog", "assets", "*", "*.json"))
	if err != nil {
		return "", err
	}
	sort.Strings(paths)
	h := sha256.New()
	matched := false
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		var doc struct {
			Asset struct {
				ID string `json:"id"`
			} `json:"asset"`
		}
		if err := json.Unmarshal(data, &doc); err != nil || doc.Asset.ID != assetID {
			continue
		}
		matched = true
		h.Write(data)
		h.Write([]byte{0})
		break
	}
	if !matched {
		return "", fmt.Errorf("catalog asset %q not found", assetID)
	}
	manifestPaths, _ := filepath.Glob(filepath.Join(scenarioRoot, "library", "*", "component.json"))
	sort.Strings(manifestPaths)
	for _, path := range manifestPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		var manifest struct {
			CatalogID string `json:"catalogId"`
			Latest    string `json:"latest"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil || manifest.CatalogID != assetID {
			continue
		}
		h.Write(data)
		h.Write([]byte{0})
		versionPaths := versionSourcePaths(filepath.Join(filepath.Dir(path), "versions", manifest.Latest))
		for _, versionPath := range versionPaths {
			versionData, err := os.ReadFile(versionPath)
			if err != nil {
				return "", err
			}
			h.Write(versionData)
			h.Write([]byte{0})
		}
		break
	}
	return hex.EncodeToString(h.Sum(nil)), nil
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
	sort.Strings(paths)
	return paths
}

// MergedEvidence combines fresh persisted browser outcomes with cheap,
// recomputed catalog gates. Persisted rows are filtered by current revision.
func MergedEvidence(ctx context.Context, root string, store *EvidenceStore) ([]GateEvidence, error) {
	computed, err := RecomputeEvidence(root)
	if err != nil {
		return nil, err
	}
	if store == nil {
		return computed, nil
	}
	persisted, err := store.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range persisted {
		if item.AssetID == "__corpus__" && item.SourceRevision == "corpus" {
			computed = append(computed, item)
			continue
		}
		revision, err := CurrentRevision(root, item.AssetID)
		if err != nil || revision != item.SourceRevision {
			continue
		}
		computed = append(computed, item)
	}
	return computed, nil
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
	// Experience Manager retains an audit history. Reduce it to the newest
	// observation for each declared claim/state/example/viewport before a gate
	// is evaluated; otherwise one old skipped capture can poison a current pass.
	latest := map[string]ExperienceCapture{}
	for _, capture := range captures {
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
			if definition.ExperienceRequiresCapture && strings.TrimSpace(capture.CaptureRef) == "" && result == "pass" {
				result = "skipped"
			}
		}
		if definition.ExperienceMinimumViewports > 0 && len(viewports) < definition.ExperienceMinimumViewports && result == "pass" {
			result = "skipped"
		}
		revision, err := CurrentRevision(root, assetID)
		if err != nil {
			return nil, err
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
		revision, err := CurrentRevision(root, assetID)
		if err != nil {
			return nil, err
		}
		for _, definition := range definitions {
			if definition.ID != "visual" && definition.ID != "responsive" {
				continue
			}
			if definition.ID == "responsive" && len(entry.viewports) < definition.ExperienceMinimumViewports {
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
	if runners["types"], err = gates.ValidateTypes(root); err != nil {
		return nil, err
	}
	if runners["api"], err = gates.ValidateAPI(root); err != nil {
		return nil, err
	}
	if runners["tokens"], err = gates.ValidateTokens(root); err != nil {
		return nil, err
	}
	if runners["conformance"], err = gates.ValidateConformance(root); err != nil {
		return nil, err
	}
	if _, statErr := os.Stat(filepath.Join(root, "scenarios", "react-component-library", "ui", "src", "design-tokens.css")); statErr == nil {
		if runners["token-vocabulary"], err = gates.ValidateTokenVocabulary(root); err != nil {
			return nil, err
		}
		if runners["token-ramp-complete"], err = gates.ValidateTokenRampComplete(root); err != nil {
			return nil, err
		}
	}
	if _, statErr := os.Stat(filepath.Join(root, "scenarios", "react-component-library", "data", "react-component-library.db")); statErr == nil {
		if runners["released-version-immutable"], err = gates.ValidateReleasedVersionImmutable(root); err != nil {
			return nil, err
		}
	}
	if runners["lifecycle"], err = gates.ValidateLifecycle(root); err != nil {
		return nil, err
	}
	if runners["examples"], err = gates.ValidateExamples(root); err != nil {
		return nil, err
	}
	if runners["fixture-adversarial"], err = gates.ValidateFixtures(root); err != nil {
		return nil, err
	}
	if runners["rtl"], err = gates.ValidateRTL(root); err != nil {
		return nil, err
	}
	if runners["reduced-motion"], err = gates.ValidateReducedMotion(root); err != nil {
		return nil, err
	}
	if runners["stress"], err = gates.ValidateStress(root); err != nil {
		return nil, err
	}
	if runners["integration"], err = gates.ValidateIntegration(root); err != nil {
		return nil, err
	}
	if runners["surface-discipline"], err = gates.ValidateSurfaceDiscipline(root); err != nil {
		return nil, err
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
		revision, err := CurrentRevision(root, asset.ID)
		if err != nil {
			return nil, err
		}
		for _, definition := range definitions {
			if definition.Attribution == "corpus" || !containsKind(definition.AppliesTo, asset.Kind) {
				continue
			}
			gateName := definition.ID
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
