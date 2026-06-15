package facts

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	cacheAnalyzerVersion = "code-facts.phase9"
	cacheSchemaVersion   = "code-facts-cache-v1"
	cacheScopeGraph      = "graph"
	cacheScopeReport     = "report"
)

type CacheRepository interface {
	GetReport(ctx context.Context, key string) (*factsv1.CodeFactsReport, *cacheEntry, bool, error)
	PutReport(ctx context.Context, entry cacheEntry, report *factsv1.CodeFactsReport) error
	GetGraph(ctx context.Context, key string) (*GraphResult, *cacheEntry, bool, error)
	PutGraph(ctx context.Context, entry cacheEntry, result *GraphResult) error
	Status(ctx context.Context, targetRoot, key string) ([]cacheEntry, error)
	Clear(ctx context.Context, targetRoot string, dryRun bool) (matched int64, cleared int64, err error)
}

type cacheEntry struct {
	Key            string
	Scope          string
	TargetRoot     string
	Analyzer       string
	Provider       string
	ProviderVer    string
	SchemaVersion  string
	GraphHash      string
	SourceHash     string
	ConfigHash     string
	FamilyKey      string
	PayloadJSON    string
	WarningsJSON   string
	ExtractionMs   int64
	CreatedAtUnix  int64
	LastUsedAtUnix int64
	HitCount       int64
}

func (e cacheEntry) metadata(state, reason string) *factsv1.CacheMetadata {
	now := time.Now().Unix()
	age := int64(0)
	if e.CreatedAtUnix > 0 && now >= e.CreatedAtUnix {
		age = now - e.CreatedAtUnix
	}
	return &factsv1.CacheMetadata{
		CacheKey:        e.Key,
		Hit:             state == "hit",
		AnalyzerVersion: e.Analyzer,
		GraphHash:       e.GraphHash,
		AgeSeconds:      age,
		State:           state,
		Reason:          reason,
		SourceHash:      e.SourceHash,
		ConfigHash:      e.ConfigHash,
		ProviderVersion: e.ProviderVer,
		SchemaVersion:   e.SchemaVersion,
		CreatedAtUnix:   e.CreatedAtUnix,
		LastUsedAtUnix:  e.LastUsedAtUnix,
		HitCount:        e.HitCount,
		Scope:           e.Scope,
	}
}

type memoryCacheRepository struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
}

func NewMemoryCacheRepository() CacheRepository {
	return &memoryCacheRepository{entries: map[string]cacheEntry{}}
}

func (r *memoryCacheRepository) GetReport(_ context.Context, key string) (*factsv1.CodeFactsReport, *cacheEntry, bool, error) {
	r.mu.Lock()
	entry, ok := r.entries[key]
	if !ok || entry.Scope != cacheScopeReport {
		r.mu.Unlock()
		return nil, nil, false, nil
	}
	entry.HitCount++
	entry.LastUsedAtUnix = time.Now().Unix()
	r.entries[key] = entry
	r.mu.Unlock()
	var report factsv1.CodeFactsReport
	if err := protojson.Unmarshal([]byte(entry.PayloadJSON), &report); err != nil {
		return nil, nil, false, err
	}
	return &report, &entry, true, nil
}

func (r *memoryCacheRepository) PutReport(_ context.Context, entry cacheEntry, report *factsv1.CodeFactsReport) error {
	payload, err := protojson.Marshal(report)
	if err != nil {
		return err
	}
	entry.Scope = cacheScopeReport
	entry.PayloadJSON = string(payload)
	entry.CreatedAtUnix = nowIfZero(entry.CreatedAtUnix)
	entry.LastUsedAtUnix = entry.CreatedAtUnix
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[entry.Key] = entry
	return nil
}

func (r *memoryCacheRepository) GetGraph(_ context.Context, key string) (*GraphResult, *cacheEntry, bool, error) {
	r.mu.Lock()
	entry, ok := r.entries[key]
	if !ok || entry.Scope != cacheScopeGraph {
		r.mu.Unlock()
		return nil, nil, false, nil
	}
	entry.HitCount++
	entry.LastUsedAtUnix = time.Now().Unix()
	r.entries[key] = entry
	r.mu.Unlock()
	result, err := graphResultFromEntry(entry)
	if err != nil {
		return nil, nil, false, err
	}
	return result, &entry, true, nil
}

func (r *memoryCacheRepository) PutGraph(_ context.Context, entry cacheEntry, result *GraphResult) error {
	if result == nil || result.Graph == nil {
		return nil
	}
	payload, warnings, err := marshalGraphResult(result)
	if err != nil {
		return err
	}
	entry.Scope = cacheScopeGraph
	entry.PayloadJSON = payload
	entry.WarningsJSON = warnings
	entry.ExtractionMs = result.ExtractionMs
	entry.CreatedAtUnix = nowIfZero(entry.CreatedAtUnix)
	entry.LastUsedAtUnix = entry.CreatedAtUnix
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[entry.Key] = entry
	return nil
}

func (r *memoryCacheRepository) Status(_ context.Context, targetRoot, key string) ([]cacheEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []cacheEntry
	for _, entry := range r.entries {
		if targetRoot != "" && entry.TargetRoot != targetRoot {
			continue
		}
		if key != "" && entry.Key != key {
			continue
		}
		out = append(out, entry)
	}
	sortCacheEntries(out)
	return out, nil
}

func (r *memoryCacheRepository) Clear(_ context.Context, targetRoot string, dryRun bool) (int64, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var matched int64
	for key, entry := range r.entries {
		if targetRoot != "" && entry.TargetRoot != targetRoot {
			continue
		}
		matched++
		if !dryRun {
			delete(r.entries, key)
		}
	}
	if dryRun {
		return matched, 0, nil
	}
	return matched, matched, nil
}

type SQLiteCacheRepository struct {
	db *sql.DB
}

func NewSQLiteCacheRepository(db *sql.DB) *SQLiteCacheRepository {
	return &SQLiteCacheRepository{db: db}
}

func CacheSchema() string {
	return `
CREATE TABLE IF NOT EXISTS code_facts_cache_entries (
  cache_key TEXT PRIMARY KEY,
  scope TEXT NOT NULL,
  target_root TEXT NOT NULL,
  analyzer TEXT NOT NULL,
  provider TEXT NOT NULL,
  provider_version TEXT NOT NULL,
  schema_version TEXT NOT NULL,
  graph_hash TEXT NOT NULL,
  source_hash TEXT NOT NULL,
  config_hash TEXT NOT NULL,
  family_key TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  warnings_json TEXT NOT NULL DEFAULT '[]',
  extraction_ms INTEGER NOT NULL DEFAULT 0,
  created_at_unix INTEGER NOT NULL,
  last_used_at_unix INTEGER NOT NULL,
  hit_count INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_code_facts_cache_target ON code_facts_cache_entries(target_root);
CREATE INDEX IF NOT EXISTS idx_code_facts_cache_scope ON code_facts_cache_entries(scope);
`
}

func (r *SQLiteCacheRepository) GetReport(ctx context.Context, key string) (*factsv1.CodeFactsReport, *cacheEntry, bool, error) {
	entry, ok, err := r.get(ctx, key, cacheScopeReport)
	if err != nil || !ok {
		return nil, nil, ok, err
	}
	var report factsv1.CodeFactsReport
	if err := protojson.Unmarshal([]byte(entry.PayloadJSON), &report); err != nil {
		return nil, nil, false, err
	}
	return &report, &entry, true, nil
}

func (r *SQLiteCacheRepository) PutReport(ctx context.Context, entry cacheEntry, report *factsv1.CodeFactsReport) error {
	payload, err := protojson.Marshal(report)
	if err != nil {
		return err
	}
	entry.Scope = cacheScopeReport
	entry.PayloadJSON = string(payload)
	return r.put(ctx, entry)
}

func (r *SQLiteCacheRepository) GetGraph(ctx context.Context, key string) (*GraphResult, *cacheEntry, bool, error) {
	entry, ok, err := r.get(ctx, key, cacheScopeGraph)
	if err != nil || !ok {
		return nil, nil, ok, err
	}
	result, err := graphResultFromEntry(entry)
	if err != nil {
		return nil, nil, false, err
	}
	return result, &entry, true, nil
}

func (r *SQLiteCacheRepository) PutGraph(ctx context.Context, entry cacheEntry, result *GraphResult) error {
	if result == nil || result.Graph == nil {
		return nil
	}
	payload, warnings, err := marshalGraphResult(result)
	if err != nil {
		return err
	}
	entry.Scope = cacheScopeGraph
	entry.PayloadJSON = payload
	entry.WarningsJSON = warnings
	entry.ExtractionMs = result.ExtractionMs
	return r.put(ctx, entry)
}

func (r *SQLiteCacheRepository) Status(ctx context.Context, targetRoot, key string) ([]cacheEntry, error) {
	query := `SELECT cache_key, scope, target_root, analyzer, provider, provider_version, schema_version, graph_hash, source_hash, config_hash, family_key, payload_json, warnings_json, extraction_ms, created_at_unix, last_used_at_unix, hit_count FROM code_facts_cache_entries WHERE 1=1`
	args := []any{}
	if targetRoot != "" {
		query += ` AND target_root = ?`
		args = append(args, targetRoot)
	}
	if key != "" {
		query += ` AND cache_key = ?`
		args = append(args, key)
	}
	query += ` ORDER BY scope, cache_key`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []cacheEntry
	for rows.Next() {
		entry, err := scanCacheEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	return out, rows.Err()
}

func (r *SQLiteCacheRepository) Clear(ctx context.Context, targetRoot string, dryRun bool) (int64, int64, error) {
	var matched int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM code_facts_cache_entries WHERE target_root = ?`, targetRoot).Scan(&matched); err != nil {
		return 0, 0, err
	}
	if dryRun || matched == 0 {
		return matched, 0, nil
	}
	res, err := r.db.ExecContext(ctx, `DELETE FROM code_facts_cache_entries WHERE target_root = ?`, targetRoot)
	if err != nil {
		return matched, 0, err
	}
	cleared, err := res.RowsAffected()
	if err != nil {
		return matched, 0, err
	}
	return matched, cleared, nil
}

func (r *SQLiteCacheRepository) get(ctx context.Context, key, scope string) (cacheEntry, bool, error) {
	row := r.db.QueryRowContext(ctx, `SELECT cache_key, scope, target_root, analyzer, provider, provider_version, schema_version, graph_hash, source_hash, config_hash, family_key, payload_json, warnings_json, extraction_ms, created_at_unix, last_used_at_unix, hit_count FROM code_facts_cache_entries WHERE cache_key = ? AND scope = ?`, key, scope)
	entry, err := scanCacheEntry(row)
	if err == sql.ErrNoRows {
		return cacheEntry{}, false, nil
	}
	if err != nil {
		return cacheEntry{}, false, err
	}
	now := time.Now().Unix()
	if _, err := r.db.ExecContext(ctx, `UPDATE code_facts_cache_entries SET hit_count = hit_count + 1, last_used_at_unix = ? WHERE cache_key = ?`, now, key); err != nil {
		return cacheEntry{}, false, err
	}
	entry.HitCount++
	entry.LastUsedAtUnix = now
	return entry, true, nil
}

func (r *SQLiteCacheRepository) put(ctx context.Context, entry cacheEntry) error {
	entry.CreatedAtUnix = nowIfZero(entry.CreatedAtUnix)
	entry.LastUsedAtUnix = entry.CreatedAtUnix
	_, err := r.db.ExecContext(ctx, `
INSERT INTO code_facts_cache_entries (
  cache_key, scope, target_root, analyzer, provider, provider_version, schema_version,
  graph_hash, source_hash, config_hash, family_key, payload_json, warnings_json,
  extraction_ms, created_at_unix, last_used_at_unix, hit_count
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(cache_key) DO UPDATE SET
  scope = excluded.scope,
  target_root = excluded.target_root,
  analyzer = excluded.analyzer,
  provider = excluded.provider,
  provider_version = excluded.provider_version,
  schema_version = excluded.schema_version,
  graph_hash = excluded.graph_hash,
  source_hash = excluded.source_hash,
  config_hash = excluded.config_hash,
  family_key = excluded.family_key,
  payload_json = excluded.payload_json,
  warnings_json = excluded.warnings_json,
  extraction_ms = excluded.extraction_ms,
  created_at_unix = excluded.created_at_unix,
  last_used_at_unix = excluded.last_used_at_unix,
  hit_count = 0`,
		entry.Key, entry.Scope, entry.TargetRoot, entry.Analyzer, entry.Provider, entry.ProviderVer, entry.SchemaVersion,
		entry.GraphHash, entry.SourceHash, entry.ConfigHash, entry.FamilyKey, entry.PayloadJSON, entry.WarningsJSON,
		entry.ExtractionMs, entry.CreatedAtUnix, entry.LastUsedAtUnix, entry.HitCount,
	)
	return err
}

type cacheScanner interface {
	Scan(dest ...any) error
}

func scanCacheEntry(scanner cacheScanner) (cacheEntry, error) {
	var entry cacheEntry
	err := scanner.Scan(
		&entry.Key, &entry.Scope, &entry.TargetRoot, &entry.Analyzer, &entry.Provider,
		&entry.ProviderVer, &entry.SchemaVersion, &entry.GraphHash, &entry.SourceHash,
		&entry.ConfigHash, &entry.FamilyKey, &entry.PayloadJSON, &entry.WarningsJSON,
		&entry.ExtractionMs, &entry.CreatedAtUnix, &entry.LastUsedAtUnix, &entry.HitCount,
	)
	return entry, err
}

func marshalGraphResult(result *GraphResult) (string, string, error) {
	graph, err := protojson.Marshal(result.Graph)
	if err != nil {
		return "", "", err
	}
	warnings := make([]json.RawMessage, 0, len(result.Warnings))
	for _, warning := range result.Warnings {
		payload, err := protojson.Marshal(warning)
		if err != nil {
			return "", "", err
		}
		warnings = append(warnings, json.RawMessage(payload))
	}
	warningsJSON, err := json.Marshal(warnings)
	if err != nil {
		return "", "", err
	}
	return string(graph), string(warningsJSON), nil
}

func graphResultFromEntry(entry cacheEntry) (*GraphResult, error) {
	var graph commonv1.CodeGraph
	if err := protojson.Unmarshal([]byte(entry.PayloadJSON), &graph); err != nil {
		return nil, err
	}
	var rawWarnings []json.RawMessage
	if strings.TrimSpace(entry.WarningsJSON) != "" {
		if err := json.Unmarshal([]byte(entry.WarningsJSON), &rawWarnings); err != nil {
			return nil, err
		}
	}
	warnings := make([]*commonv1.CodeGraphWarning, 0, len(rawWarnings))
	for _, raw := range rawWarnings {
		var warning commonv1.CodeGraphWarning
		if err := protojson.Unmarshal(raw, &warning); err != nil {
			return nil, err
		}
		warnings = append(warnings, &warning)
	}
	return &GraphResult{Graph: &graph, Warnings: warnings, GraphHash: entry.GraphHash, ExtractionMs: entry.ExtractionMs}, nil
}

func sourceFingerprint(target *factsv1.TargetContext, units []*factsv1.ParseUnit) (sourceHash string, configHash string) {
	configs := map[string]bool{}
	roots := map[string]bool{}
	for _, unit := range units {
		if unit.GetConfigPath() != "" {
			configs[unit.GetConfigPath()] = true
		}
		if unit.GetRootPath() != "" && unit.GetStatus() == factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN {
			roots[unit.GetRootPath()] = true
		}
	}
	for root := range roots {
		for _, name := range []string{"go.mod", "go.sum", "tsconfig.json", "package.json", "pnpm-lock.yaml", "package-lock.json", "yarn.lock"} {
			path := filepath.Join(root, name)
			if fileExists(path) {
				configs[path] = true
			}
		}
	}
	return fileStatSignature(target.GetRootPath(), sourceFiles(roots)), fileStatSignature(target.GetRootPath(), keys(configs))
}

func sourceFiles(roots map[string]bool) []string {
	seen := map[string]bool{}
	for root := range roots {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if path != root && shouldPruneDir(d.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			if isSourceFile(path) {
				seen[path] = true
			}
			return nil
		})
	}
	return keys(seen)
}

func isSourceFile(path string) bool {
	switch filepath.Ext(path) {
	case ".go", ".ts", ".tsx", ".js", ".jsx":
		return true
	default:
		return false
	}
}

func fileStatSignature(root string, paths []string) string {
	sort.Strings(paths)
	h := sha256.New()
	for _, path := range paths {
		rel, err := filepath.Rel(root, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			rel = path
		}
		info, err := os.Stat(path)
		if err != nil {
			_, _ = fmt.Fprintf(h, "missing:%s\n", filepath.ToSlash(rel))
			continue
		}
		_, _ = fmt.Fprintf(h, "%s:%d:%d\n", filepath.ToSlash(rel), info.Size(), info.ModTime().UnixNano())
	}
	return hex.EncodeToString(h.Sum(nil))
}

func cacheKey(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		_, _ = h.Write([]byte(part))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func familyKey(families []factsv1.FactFamily) string {
	names := make([]string, 0, len(families))
	for _, family := range families {
		names = append(names, family.String())
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}

func providerVersions(units []*factsv1.ParseUnit) string {
	versions := map[string]bool{cacheAnalyzerVersion: true}
	for _, unit := range units {
		switch unit.GetLanguage() {
		case "go":
			versions[goProviderScenario+":phase8"] = true
		case "typescript":
			versions[tsProviderScenario+":phase8"] = true
		}
	}
	return strings.Join(keys(versions), ",")
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func sortCacheEntries(entries []cacheEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Scope != entries[j].Scope {
			return entries[i].Scope < entries[j].Scope
		}
		return entries[i].Key < entries[j].Key
	})
}

func nowIfZero(ts int64) int64 {
	if ts != 0 {
		return ts
	}
	return time.Now().Unix()
}
