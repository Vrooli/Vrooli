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
	"time"

	"react-component-library/internal/gates"
)

// Schema owns the durable browser-backed gate evidence table. Evidence is
// intentionally separate from the file-based catalog join: an absent asset
// must remain representable even when there is no database row for it.
func Schema() string {
	return `
CREATE TABLE IF NOT EXISTS catalog_gate_evidence (
  asset_id TEXT NOT NULL,
  target TEXT NOT NULL,
  gate TEXT NOT NULL,
  result TEXT NOT NULL,
  source_revision TEXT NOT NULL,
  recorded_at TEXT NOT NULL,
  PRIMARY KEY (asset_id, target, gate)
);
CREATE INDEX IF NOT EXISTS idx_catalog_gate_evidence_revision
  ON catalog_gate_evidence(source_revision);
`
}

// EvidenceStore is the persisted producer for browser-backed gate outcomes.
type EvidenceStore struct {
	db  *sql.DB
	now func() time.Time
}

func NewEvidenceStore(db *sql.DB) *EvidenceStore {
	return &EvidenceStore{db: db, now: time.Now}
}

func (s *EvidenceStore) Save(ctx context.Context, evidence []GateEvidence) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("catalog evidence store is not configured")
	}
	if len(evidence) == 0 {
		return nil
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
		if strings.TrimSpace(item.RecordedAt) == "" {
			item.RecordedAt = stamp
		}
		_, err := tx.ExecContext(ctx, `
INSERT INTO catalog_gate_evidence(asset_id, target, gate, result, source_revision, recorded_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(asset_id, target, gate) DO UPDATE SET
  result=excluded.result,
  source_revision=excluded.source_revision,
  recorded_at=excluded.recorded_at`, item.AssetID, item.Target, item.Gate, item.Result, item.SourceRevision, item.RecordedAt)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *EvidenceStore) List(ctx context.Context) ([]GateEvidence, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT asset_id, target, gate, result, source_revision, recorded_at FROM catalog_gate_evidence ORDER BY asset_id, target, gate`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GateEvidence
	for rows.Next() {
		var item GateEvidence
		if err := rows.Scan(&item.AssetID, &item.Target, &item.Gate, &item.Result, &item.SourceRevision, &item.RecordedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
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
		revision, err := CurrentRevision(root, item.AssetID)
		if err != nil || revision != item.SourceRevision {
			continue
		}
		computed = append(computed, item)
	}
	return computed, nil
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
	if runners["lifecycle"], err = gates.ValidateLifecycle(root); err != nil {
		return nil, err
	}
	if runners["examples"], err = gates.ValidateExamples(root); err != nil {
		return nil, err
	}
	if runners["fixture-adversarial"], err = gates.ValidateFixtures(root); err != nil {
		return nil, err
	}
	var out []GateEvidence
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
			gateName := definition.ID
			if !containsKind(definition.AppliesTo, asset.Kind) {
				continue
			}
			runner, ok := runners[gateName]
			if !ok {
				continue
			}
			result := "pass"
			if hasFinding(runner.Findings, asset.ID, implByAsset[asset.ID].Name, gateName) {
				result = "fail"
			} else if runner.Inspected == 0 {
				result = "skipped"
			}
			out = append(out, GateEvidence{AssetID: asset.ID, Target: target, Gate: gateName, Result: result, SourceRevision: revision})
		}
	}
	return out, nil
}

func hasFinding(findings []gates.Finding, assetID, implementation, gate string) bool {
	for _, finding := range findings {
		if finding.AssetID == "catalog.runner" || finding.AssetID == assetID || finding.AssetID == implementation {
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
