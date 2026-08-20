package facts

import (
	"bytes"
	"compress/gzip"
	"container/list"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	cacheAnalyzerVersion = "code-facts.phase9"
	cacheSchemaVersion   = "code-facts-cache-v2"
	cacheScopeGraph      = "graph"
	cacheScopeReport     = "report"
	DefaultCacheMaxBytes = int64(2 * 1024 * 1024 * 1024)
	cacheCodecGzip       = "gzip"
)

var lastCacheSweepAtUnix atomic.Int64

type CacheRepository interface {
	GetReport(ctx context.Context, key string) (*factsv1.CodeFactsReport, *cacheEntry, bool, error)
	PutReport(ctx context.Context, entry cacheEntry, report *factsv1.CodeFactsReport) error
	GetGraph(ctx context.Context, key string) (*GraphResult, *cacheEntry, bool, error)
	PutGraph(ctx context.Context, entry cacheEntry, result *GraphResult) error
	Status(ctx context.Context, targetRoot, key string) ([]cacheEntry, error)
	Stats(ctx context.Context) (CacheStats, error)
	Clear(ctx context.Context, targetRoot string, dryRun bool) (matched int64, cleared int64, err error)
	Sweep(ctx context.Context) (CacheSweepResult, error)
}

type CacheStats struct {
	TotalRows         int64
	TotalPayloadBytes int64
	BudgetBytes       int64
	LastSweepAtUnix   int64
	Scopes            []CacheScopeStats
}

type CacheScopeStats struct {
	Scope        string
	Rows         int64
	PayloadBytes int64
}

type CacheSweepResult struct {
	StaleRows     int64
	EvictedRows   int64
	ReclaimedByte int64
	RemainingByte int64
	SweptAtUnix   int64
}

type cacheEntry struct {
	Key            string
	LogicalKey     string
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
	Identity       string
	Codec          string
	PayloadJSON    string
	PayloadBytes   int64
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
		LogicalKey:      e.LogicalKey,
		PayloadBytes:    e.PayloadBytes,
		Codec:           e.Codec,
	}
}

type memoryCacheRepository struct {
	mu       sync.RWMutex
	entries  map[string]cacheEntry
	maxBytes int64
}

func NewMemoryCacheRepository(maxBytes ...int64) CacheRepository {
	return &memoryCacheRepository{entries: map[string]cacheEntry{}, maxBytes: firstMaxBytes(maxBytes)}
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
	entry.LogicalKey = logicalKeyForEntry(entry)
	entry.PayloadBytes = int64(len(entry.PayloadJSON))
	r.mu.Lock()
	defer r.mu.Unlock()
	r.supersedeLocked(entry)
	r.entries[entry.Key] = entry
	r.enforceBudgetLocked(entry.Key)
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
	entry.LogicalKey = logicalKeyForEntry(entry)
	entry.PayloadBytes = int64(len(entry.PayloadJSON))
	r.mu.Lock()
	defer r.mu.Unlock()
	r.supersedeLocked(entry)
	r.entries[entry.Key] = entry
	r.enforceBudgetLocked(entry.Key)
	return nil
}

func (r *memoryCacheRepository) supersedeLocked(entry cacheEntry) {
	for key, existing := range r.entries {
		if key != entry.Key && existing.LogicalKey != "" && existing.LogicalKey == entry.LogicalKey {
			delete(r.entries, key)
		}
	}
}

func (r *memoryCacheRepository) enforceBudgetLocked(protectedKey string) CacheSweepResult {
	var result CacheSweepResult
	if r.maxBytes <= 0 {
		result.RemainingByte = r.totalPayloadBytesLocked()
		return result
	}
	for {
		total := r.totalPayloadBytesLocked()
		result.RemainingByte = total
		if total <= r.maxBytes {
			return result
		}
		var victimKey string
		var victim cacheEntry
		for key, entry := range r.entries {
			if key == protectedKey {
				continue
			}
			if victimKey == "" || entry.LastUsedAtUnix < victim.LastUsedAtUnix ||
				(entry.LastUsedAtUnix == victim.LastUsedAtUnix && entry.CreatedAtUnix < victim.CreatedAtUnix) ||
				(entry.LastUsedAtUnix == victim.LastUsedAtUnix && entry.CreatedAtUnix == victim.CreatedAtUnix && key < victimKey) {
				victimKey = key
				victim = entry
			}
		}
		if victimKey == "" {
			return result
		}
		delete(r.entries, victimKey)
		result.EvictedRows++
		result.ReclaimedByte += victim.PayloadBytes
	}
}

func (r *memoryCacheRepository) totalPayloadBytesLocked() int64 {
	var total int64
	for _, entry := range r.entries {
		total += entry.PayloadBytes
	}
	return total
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

func (r *memoryCacheRepository) Stats(_ context.Context) (CacheStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	stats := CacheStats{BudgetBytes: r.maxBytes, LastSweepAtUnix: lastCacheSweepAtUnix.Load()}
	byScope := map[string]*CacheScopeStats{}
	for _, entry := range r.entries {
		stats.TotalRows++
		stats.TotalPayloadBytes += entry.PayloadBytes
		scope := entry.Scope
		if scope == "" {
			scope = "unknown"
		}
		scopeStats := byScope[scope]
		if scopeStats == nil {
			scopeStats = &CacheScopeStats{Scope: scope}
			byScope[scope] = scopeStats
		}
		scopeStats.Rows++
		scopeStats.PayloadBytes += entry.PayloadBytes
	}
	stats.Scopes = sortedScopeStats(byScope)
	return stats, nil
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

func (r *memoryCacheRepository) Sweep(_ context.Context) (CacheSweepResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result CacheSweepResult
	for key, entry := range r.entries {
		if entry.SchemaVersion != cacheSchemaVersion {
			delete(r.entries, key)
			result.StaleRows++
		}
	}
	evicted := r.enforceBudgetLocked("")
	result.EvictedRows = evicted.EvictedRows
	result.ReclaimedByte = evicted.ReclaimedByte
	result.RemainingByte = evicted.RemainingByte
	result.SweptAtUnix = time.Now().Unix()
	lastCacheSweepAtUnix.Store(result.SweptAtUnix)
	return result, nil
}

type SQLiteCacheRepository struct {
	db       *sql.DB
	maxBytes int64
}

func NewSQLiteCacheRepository(db *sql.DB, maxBytes ...int64) *SQLiteCacheRepository {
	return &SQLiteCacheRepository{db: db, maxBytes: firstMaxBytes(maxBytes)}
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
  logical_key TEXT NOT NULL DEFAULT '',
  payload_bytes INTEGER NOT NULL DEFAULT 0,
  codec TEXT NOT NULL DEFAULT '',
  payload_json TEXT NOT NULL,
  warnings_json TEXT NOT NULL DEFAULT '[]',
  extraction_ms INTEGER NOT NULL DEFAULT 0,
  created_at_unix INTEGER NOT NULL,
  last_used_at_unix INTEGER NOT NULL,
  hit_count INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_code_facts_cache_target ON code_facts_cache_entries(target_root);
CREATE INDEX IF NOT EXISTS idx_code_facts_cache_scope ON code_facts_cache_entries(scope);
CREATE INDEX IF NOT EXISTS idx_code_facts_cache_logical_key ON code_facts_cache_entries(logical_key);
`
}

func MigrateCacheSchema(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return nil
	}
	exists, err := cacheTableExists(ctx, db)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if err := ensureCacheColumn(ctx, db, "logical_key", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureCacheColumn(ctx, db, "payload_bytes", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := ensureCacheColumn(ctx, db, "codec", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `UPDATE code_facts_cache_entries SET logical_key = cache_key WHERE logical_key = ''`); err != nil {
		return fmt.Errorf("backfill cache logical key: %w", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE code_facts_cache_entries SET payload_bytes = length(payload_json) WHERE payload_bytes = 0 AND schema_version = ?`, cacheSchemaVersion); err != nil {
		return fmt.Errorf("backfill cache payload bytes: %w", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_code_facts_cache_logical_key ON code_facts_cache_entries(logical_key)`); err != nil {
		return fmt.Errorf("create cache logical key index: %w", err)
	}
	return nil
}

func cacheTableExists(ctx context.Context, db *sql.DB) (bool, error) {
	var name string
	err := db.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'code_facts_cache_entries'`).Scan(&name)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect cache table: %w", err)
	}
	return true, nil
}

func ensureCacheColumn(ctx context.Context, db *sql.DB, column string, definition string) error {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(code_facts_cache_entries)`)
	if err != nil {
		return fmt.Errorf("inspect cache columns: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("scan cache column: %w", err)
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("inspect cache columns: %w", err)
	}
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE code_facts_cache_entries ADD COLUMN %s %s`, column, definition)); err != nil {
		return fmt.Errorf("add cache column %s: %w", column, err)
	}
	return nil
}

func (r *SQLiteCacheRepository) GetReport(ctx context.Context, key string) (*factsv1.CodeFactsReport, *cacheEntry, bool, error) {
	entry, ok, err := r.get(ctx, key, cacheScopeReport)
	if err != nil || !ok {
		return nil, nil, ok, err
	}
	var report factsv1.CodeFactsReport
	payload, err := decodeCachePayload(entry)
	if err != nil {
		return nil, nil, false, err
	}
	if err := protojson.Unmarshal(payload, &report); err != nil {
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
	compressed, err := gzipPayload(payload)
	if err != nil {
		return err
	}
	entry.Codec = cacheCodecGzip
	entry.PayloadJSON = string(compressed)
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
	compressed, err := gzipPayload([]byte(payload))
	if err != nil {
		return err
	}
	entry.Codec = cacheCodecGzip
	entry.PayloadJSON = string(compressed)
	entry.WarningsJSON = warnings
	entry.ExtractionMs = result.ExtractionMs
	return r.put(ctx, entry)
}

func (r *SQLiteCacheRepository) Status(ctx context.Context, targetRoot, key string) ([]cacheEntry, error) {
	query, args := cacheStatusQuery(targetRoot, key)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []cacheEntry
	for rows.Next() {
		entry, err := scanCacheEntryMetadata(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	return out, rows.Err()
}

func (r *SQLiteCacheRepository) Stats(ctx context.Context) (CacheStats, error) {
	var stats CacheStats
	stats.BudgetBytes = r.maxBytes
	stats.LastSweepAtUnix = lastCacheSweepAtUnix.Load()
	row := r.db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(payload_bytes), 0) FROM code_facts_cache_entries`)
	if err := row.Scan(&stats.TotalRows, &stats.TotalPayloadBytes); err != nil {
		return stats, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT scope, COUNT(*), COALESCE(SUM(payload_bytes), 0) FROM code_facts_cache_entries GROUP BY scope ORDER BY scope`)
	if err != nil {
		return stats, err
	}
	defer rows.Close()
	for rows.Next() {
		var scope CacheScopeStats
		if err := rows.Scan(&scope.Scope, &scope.Rows, &scope.PayloadBytes); err != nil {
			return stats, err
		}
		stats.Scopes = append(stats.Scopes, scope)
	}
	if err := rows.Err(); err != nil {
		return stats, err
	}
	return stats, nil
}

func (r *SQLiteCacheRepository) Clear(ctx context.Context, targetRoot string, dryRun bool) (int64, int64, error) {
	var matched int64
	countQuery, args := cacheClearCountQuery(targetRoot)
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&matched); err != nil {
		return 0, 0, err
	}
	if dryRun || matched == 0 {
		return matched, 0, nil
	}
	deleteQuery, args := cacheClearDeleteQuery(targetRoot)
	res, err := r.db.ExecContext(ctx, deleteQuery, args...)
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
	row := r.db.QueryRowContext(ctx, cacheEntrySelect()+` WHERE cache_key = ? AND scope = ?`, key, scope)
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
	entry.LogicalKey = logicalKeyForEntry(entry)
	entry.PayloadBytes = int64(len(entry.PayloadJSON))
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if _, err = tx.ExecContext(ctx, `DELETE FROM code_facts_cache_entries WHERE logical_key = ? AND cache_key != ?`, entry.LogicalKey, entry.Key); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO code_facts_cache_entries (
  cache_key, scope, target_root, analyzer, provider, provider_version, schema_version,
  graph_hash, source_hash, config_hash, family_key, logical_key, payload_bytes,
  codec, payload_json, warnings_json, extraction_ms, created_at_unix, last_used_at_unix, hit_count
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
  logical_key = excluded.logical_key,
  payload_bytes = excluded.payload_bytes,
  codec = excluded.codec,
  payload_json = excluded.payload_json,
  warnings_json = excluded.warnings_json,
  extraction_ms = excluded.extraction_ms,
  created_at_unix = excluded.created_at_unix,
  last_used_at_unix = excluded.last_used_at_unix,
  hit_count = 0`,
		entry.Key, entry.Scope, entry.TargetRoot, entry.Analyzer, entry.Provider, entry.ProviderVer, entry.SchemaVersion,
		entry.GraphHash, entry.SourceHash, entry.ConfigHash, entry.FamilyKey, entry.LogicalKey, entry.PayloadBytes,
		entry.Codec, entry.PayloadJSON, entry.WarningsJSON,
		entry.ExtractionMs, entry.CreatedAtUnix, entry.LastUsedAtUnix, entry.HitCount,
	)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	_, err = r.enforceBudget(ctx, entry.Key)
	return err
}

func (r *SQLiteCacheRepository) Sweep(ctx context.Context) (CacheSweepResult, error) {
	var result CacheSweepResult
	for {
		res, err := r.db.ExecContext(ctx, `DELETE FROM code_facts_cache_entries WHERE rowid IN (
			SELECT rowid FROM code_facts_cache_entries WHERE schema_version != ? LIMIT 100
		)`, cacheSchemaVersion)
		if err != nil {
			return result, err
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return result, err
		}
		if rows == 0 {
			break
		}
		result.StaleRows += rows
	}
	evicted, err := r.enforceBudget(ctx, "")
	if err != nil {
		return result, err
	}
	result.EvictedRows = evicted.EvictedRows
	result.ReclaimedByte = evicted.ReclaimedByte
	result.RemainingByte = evicted.RemainingByte
	result.SweptAtUnix = time.Now().Unix()
	lastCacheSweepAtUnix.Store(result.SweptAtUnix)
	if result.StaleRows > 0 || result.EvictedRows > 0 {
		if _, err := r.db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
			return result, fmt.Errorf("checkpoint cache storage: %w", err)
		}
		if _, err := r.db.ExecContext(ctx, `VACUUM`); err != nil {
			return result, fmt.Errorf("reclaim cache storage: %w", err)
		}
	}
	return result, nil
}

func (r *SQLiteCacheRepository) enforceBudget(ctx context.Context, protectedKey string) (CacheSweepResult, error) {
	var result CacheSweepResult
	for {
		total, err := r.totalPayloadBytes(ctx)
		if err != nil {
			return result, err
		}
		result.RemainingByte = total
		if r.maxBytes <= 0 || total <= r.maxBytes {
			return result, nil
		}
		victim, bytes, ok, err := r.oldestEvictableEntry(ctx, protectedKey)
		if err != nil {
			return result, err
		}
		if !ok {
			return result, nil
		}
		res, err := r.db.ExecContext(ctx, `DELETE FROM code_facts_cache_entries WHERE cache_key = ?`, victim)
		if err != nil {
			return result, err
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return result, err
		}
		if rows == 0 {
			return result, nil
		}
		result.EvictedRows += rows
		result.ReclaimedByte += bytes
	}
}

func (r *SQLiteCacheRepository) totalPayloadBytes(ctx context.Context) (int64, error) {
	var total sql.NullInt64
	if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(payload_bytes), 0) FROM code_facts_cache_entries`).Scan(&total); err != nil {
		return 0, err
	}
	if !total.Valid {
		return 0, nil
	}
	return total.Int64, nil
}

func (r *SQLiteCacheRepository) oldestEvictableEntry(ctx context.Context, protectedKey string) (string, int64, bool, error) {
	query, args := cacheOldestEvictableQuery(protectedKey)
	var key string
	var bytes int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&key, &bytes)
	if err == sql.ErrNoRows {
		return "", 0, false, nil
	}
	if err != nil {
		return "", 0, false, err
	}
	return key, bytes, true, nil
}

type cacheScanner interface {
	Scan(dest ...any) error
}

func scanCacheEntry(scanner cacheScanner) (cacheEntry, error) {
	var entry cacheEntry
	err := scanner.Scan(
		&entry.Key, &entry.LogicalKey, &entry.Scope, &entry.TargetRoot, &entry.Analyzer, &entry.Provider,
		&entry.ProviderVer, &entry.SchemaVersion, &entry.GraphHash, &entry.SourceHash,
		&entry.ConfigHash, &entry.FamilyKey, &entry.PayloadBytes, &entry.Codec, &entry.PayloadJSON, &entry.WarningsJSON,
		&entry.ExtractionMs, &entry.CreatedAtUnix, &entry.LastUsedAtUnix, &entry.HitCount,
	)
	return entry, err
}

func cacheEntrySelect() string {
	return `SELECT cache_key, logical_key, scope, target_root, analyzer, provider, provider_version, schema_version, graph_hash, source_hash, config_hash, family_key, payload_bytes, codec, payload_json, warnings_json, extraction_ms, created_at_unix, last_used_at_unix, hit_count FROM code_facts_cache_entries`
}

// cacheEntryMetadataSelect intentionally excludes payload_json and
// warnings_json. Status and inspection endpoints only expose cache metadata;
// reading compressed graph/report bodies here can materialize the entire cache
// in the Go heap without decoding or returning any of those bytes.
func cacheEntryMetadataSelect() string {
	return `SELECT cache_key, logical_key, scope, target_root, analyzer, provider, provider_version, schema_version, graph_hash, source_hash, config_hash, family_key, payload_bytes, codec, extraction_ms, created_at_unix, last_used_at_unix, hit_count FROM code_facts_cache_entries`
}

func scanCacheEntryMetadata(scanner cacheScanner) (cacheEntry, error) {
	var entry cacheEntry
	err := scanner.Scan(
		&entry.Key, &entry.LogicalKey, &entry.Scope, &entry.TargetRoot, &entry.Analyzer, &entry.Provider,
		&entry.ProviderVer, &entry.SchemaVersion, &entry.GraphHash, &entry.SourceHash,
		&entry.ConfigHash, &entry.FamilyKey, &entry.PayloadBytes, &entry.Codec,
		&entry.ExtractionMs, &entry.CreatedAtUnix, &entry.LastUsedAtUnix, &entry.HitCount,
	)
	return entry, err
}

func cacheStatusQuery(targetRoot, key string) (string, []any) {
	const order = ` ORDER BY scope, cache_key`
	switch {
	case targetRoot != "" && key != "":
		return cacheEntryMetadataSelect() + ` WHERE target_root = ? AND cache_key = ?` + order, []any{targetRoot, key}
	case targetRoot != "":
		return cacheEntryMetadataSelect() + ` WHERE target_root = ?` + order, []any{targetRoot}
	case key != "":
		return cacheEntryMetadataSelect() + ` WHERE cache_key = ?` + order, []any{key}
	default:
		return cacheEntryMetadataSelect() + order, nil
	}
}

func cacheClearCountQuery(targetRoot string) (string, []any) {
	if targetRoot == "" {
		return `SELECT COUNT(*) FROM code_facts_cache_entries`, nil
	}
	return `SELECT COUNT(*) FROM code_facts_cache_entries WHERE target_root = ?`, []any{targetRoot}
}

func cacheClearDeleteQuery(targetRoot string) (string, []any) {
	if targetRoot == "" {
		return `DELETE FROM code_facts_cache_entries`, nil
	}
	return `DELETE FROM code_facts_cache_entries WHERE target_root = ?`, []any{targetRoot}
}

func cacheOldestEvictableQuery(protectedKey string) (string, []any) {
	const order = ` ORDER BY last_used_at_unix ASC, created_at_unix ASC, cache_key ASC LIMIT 1`
	if protectedKey == "" {
		return `SELECT cache_key, payload_bytes FROM code_facts_cache_entries` + order, nil
	}
	return `SELECT cache_key, payload_bytes FROM code_facts_cache_entries WHERE cache_key != ?` + order, []any{protectedKey}
}

func logicalKeyForEntry(entry cacheEntry) string {
	if entry.LogicalKey != "" {
		return entry.LogicalKey
	}
	identity := entry.Identity
	if identity == "" {
		identity = entry.Provider + "|" + entry.ProviderVer
	}
	return cacheKey(entry.Scope, entry.TargetRoot, entry.Analyzer, entry.FamilyKey, identity)
}

func firstMaxBytes(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	return values[0]
}

func sortedScopeStats(byScope map[string]*CacheScopeStats) []CacheScopeStats {
	scopes := make([]string, 0, len(byScope))
	for scope := range byScope {
		scopes = append(scopes, scope)
	}
	sort.Strings(scopes)
	out := make([]CacheScopeStats, 0, len(scopes))
	for _, scope := range scopes {
		out = append(out, *byScope[scope])
	}
	return out
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
	payload, err := decodeCachePayload(entry)
	if err != nil {
		return nil, err
	}
	if err := protojson.Unmarshal(payload, &graph); err != nil {
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

func gzipPayload(payload []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	if _, err := writer.Write(payload); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decodeCachePayload(entry cacheEntry) ([]byte, error) {
	switch entry.Codec {
	case "":
		return []byte(entry.PayloadJSON), nil
	case cacheCodecGzip:
		reader, err := gzip.NewReader(bytes.NewReader([]byte(entry.PayloadJSON)))
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		return io.ReadAll(reader)
	default:
		return nil, fmt.Errorf("unsupported cache payload codec %q", entry.Codec)
	}
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
	if target.GetRootPath() != "" {
		for _, name := range []string{filepath.Join("docs", "concepts", "DOMAINS.md")} {
			path := filepath.Join(target.GetRootPath(), name)
			if fileExists(path) {
				configs[path] = true
			}
		}
	}
	return fileStatSignature(target.GetRootPath(), sourceFiles(roots)), fileStatSignature(target.GetRootPath(), keys(configs))
}

func sourceFingerprintForUnit(unit *factsv1.ParseUnit) (sourceHash string, configHash string) {
	if unit == nil {
		return "", ""
	}
	root := filepath.Clean(unit.GetRootPath())
	if root == "." || root == "" {
		return "", ""
	}
	configs := map[string]bool{}
	if unit.GetConfigPath() != "" {
		configs[unit.GetConfigPath()] = true
	}
	return fileStatSignature(root, sourceFiles(map[string]bool{root: true})), fileStatSignature(root, keys(configs))
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
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".sh", ".bats":
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
		contentHash, err := sourceFileContentHash(path, info)
		if err != nil {
			_, _ = fmt.Fprintf(h, "missing:%s\n", filepath.ToSlash(rel))
			continue
		}
		_, _ = fmt.Fprintf(h, "%s:%s\n", filepath.ToSlash(rel), contentHash)
	}
	return hex.EncodeToString(h.Sum(nil))
}

var sourceFileHashMemo = newFileHashMemo(8192)

func sourceFileContentHash(path string, info os.FileInfo) (string, error) {
	key := fileHashMemoKey(path, info)
	if hash, ok := sourceFileHashMemo.get(key); ok {
		return hash, nil
	}
	payload, err := os.ReadFile(path) // #nosec G304 -- paths come from the caller-selected source tree and are hashed, not executed.
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	hash := hex.EncodeToString(sum[:])
	sourceFileHashMemo.put(key, hash)
	return hash, nil
}

func fileHashMemoKey(path string, info os.FileInfo) string {
	return path + "\x00" + fmt.Sprint(info.Size()) + "\x00" + fmt.Sprint(info.ModTime().UnixNano())
}

type fileHashMemo struct {
	mu      sync.Mutex
	cap     int
	entries map[string]*list.Element
	order   *list.List
}

type fileHashMemoEntry struct {
	key  string
	hash string
}

func newFileHashMemo(capacity int) *fileHashMemo {
	return &fileHashMemo{cap: capacity, entries: map[string]*list.Element{}, order: list.New()}
}

func (m *fileHashMemo) get(key string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	el, ok := m.entries[key]
	if !ok {
		return "", false
	}
	m.order.MoveToFront(el)
	return el.Value.(fileHashMemoEntry).hash, true
}

func (m *fileHashMemo) put(key string, hash string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if el, ok := m.entries[key]; ok {
		el.Value = fileHashMemoEntry{key: key, hash: hash}
		m.order.MoveToFront(el)
		return
	}
	el := m.order.PushFront(fileHashMemoEntry{key: key, hash: hash})
	m.entries[key] = el
	for m.cap > 0 && len(m.entries) > m.cap {
		back := m.order.Back()
		if back == nil {
			return
		}
		entry := back.Value.(fileHashMemoEntry)
		delete(m.entries, entry.key)
		m.order.Remove(back)
	}
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
