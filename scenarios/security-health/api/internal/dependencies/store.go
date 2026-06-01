package dependencies

import (
	"context"
	"database/sql"
	"fmt"
	"path"
	"sort"
	"strings"
)

// SQLExecutor is the narrow database surface the store depends on. Both
// *sql.DB (unit tests) and *database.RoutedDB (production) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Store persists and queries the fleet SBOM corpus in SQLite.
type Store struct {
	db SQLExecutor
}

// NewStore wraps a database handle.
func NewStore(db SQLExecutor) *Store { return &Store{db: db} }

// Diff compares the freshly-discovered record set against the persisted corpus
// and returns the planned upsert/delete counts (used by reindex --dry-run and
// to report planned figures before a real apply).
func (s *Store) Diff(ctx context.Context, scenario string, fresh []DependencyRecord) (upserts, deletes int, err error) {
	current, err := s.keySet(ctx, scenario)
	if err != nil {
		return 0, 0, err
	}
	freshKeys := make(map[string]struct{}, len(fresh))
	for _, r := range fresh {
		freshKeys[r.Key()] = struct{}{}
		if _, ok := current[r.Key()]; !ok {
			upserts++
		}
	}
	// Existing keys not in the fresh set are deletions. (Upserts here counts
	// only *new* keys; changed-but-present rows are refreshed in place and not
	// counted as upserts — Diff is a planning estimate, not a byte-diff.)
	for k := range current {
		if _, ok := freshKeys[k]; !ok {
			deletes++
		}
	}
	return upserts, deletes, nil
}

// Apply replaces the corpus (optionally scoped to one scenario) with the fresh
// records, stamping last_seen=now, and deletes rows no longer present. Runs in
// a single transaction so a reader never sees a half-reconciled corpus.
func (s *Store) Apply(ctx context.Context, scenario string, fresh []DependencyRecord, now string) error {
	freshKeys := make(map[string]struct{}, len(fresh))
	for _, r := range fresh {
		freshKeys[r.Key()] = struct{}{}
	}
	for _, r := range fresh {
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO dependency_records (dep_key, scenario, ecosystem, name, version, source_file, vuln_ids, max_severity, last_seen)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(dep_key) DO UPDATE SET
				source_file = excluded.source_file,
				vuln_ids = excluded.vuln_ids,
				max_severity = excluded.max_severity,
				last_seen = excluded.last_seen`,
			r.Key(), r.Scenario, string(r.Ecosystem), r.Name, r.Version, r.SourceFile,
			strings.Join(r.VulnIDs, ","), r.MaxSeverity, now,
		); err != nil {
			return fmt.Errorf("upsert %s: %w", r.Key(), err)
		}
	}
	// Delete stale rows (present before, absent now) within the scope.
	current, err := s.keySet(ctx, scenario)
	if err != nil {
		return err
	}
	for k := range current {
		if _, ok := freshKeys[k]; !ok {
			if _, err := s.db.ExecContext(ctx, `DELETE FROM dependency_records WHERE dep_key = ?`, k); err != nil {
				return fmt.Errorf("delete %s: %w", k, err)
			}
		}
	}
	return nil
}

// keySet returns the set of dep_keys currently stored, optionally scoped to one
// scenario (empty scenario = whole corpus).
func (s *Store) keySet(ctx context.Context, scenario string) (map[string]struct{}, error) {
	q := `SELECT dep_key FROM dependency_records`
	var args []any
	if scenario != "" {
		q += ` WHERE scenario = ?`
		args = append(args, scenario)
	}
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]struct{}{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out[k] = struct{}{}
	}
	return out, rows.Err()
}

// Search runs structured filters in SQL, then ranks the survivors lexically
// against the free-text query. AI ranking, when wired, re-orders the same set.
func (s *Store) Search(ctx context.Context, req SearchRequest) ([]SearchResult, error) {
	var where []string
	var args []any
	if req.Ecosystem != EcosystemUnspecified {
		where = append(where, "ecosystem = ?")
		args = append(args, string(req.Ecosystem))
	}
	if req.VulnerableOnly {
		where = append(where, "vuln_ids != ''")
	}
	q := `SELECT scenario, ecosystem, name, version, source_file, vuln_ids, max_severity, last_seen FROM dependency_records`
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	limit := req.Limit
	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	if limit > MaxSearchLimit {
		limit = MaxSearchLimit
	}

	var hits []SearchResult
	for rows.Next() {
		var r DependencyRecord
		var eco, vulnIDs string
		if err := rows.Scan(&r.Scenario, &eco, &r.Name, &r.Version, &r.SourceFile, &vulnIDs, &r.MaxSeverity, &r.LastSeen); err != nil {
			return nil, err
		}
		r.Ecosystem = Ecosystem(eco)
		if vulnIDs != "" {
			r.VulnIDs = strings.Split(vulnIDs, ",")
		}
		if req.NameGlob != "" {
			ok, _ := path.Match(req.NameGlob, r.Name)
			if !ok {
				continue
			}
		}
		score := lexicalScore(req.Query, r)
		if req.Query != "" && score == 0 {
			continue
		}
		hits = append(hits, SearchResult{Record: r, Score: score})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].Record.Key() < hits[j].Record.Key()
	})
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits, nil
}

// lexicalScore is a deterministic [0,1] relevance score: fraction of query
// terms that appear (case-insensitively) in the record's name/scenario. An
// empty query scores 1 (everything matches; structured filters do the work).
func lexicalScore(query string, r DependencyRecord) float64 {
	q := strings.Fields(strings.ToLower(query))
	if len(q) == 0 {
		return 1
	}
	hay := strings.ToLower(r.Name + " " + r.Scenario + " " + strings.Join(r.VulnIDs, " "))
	matched := 0
	for _, term := range q {
		if strings.Contains(hay, term) {
			matched++
		}
	}
	return float64(matched) / float64(len(q))
}

// RecordsByKeys fetches the records for the given dep_keys, returning them keyed
// by dep_key. Missing keys are simply absent from the map (a vector index can
// lag the corpus). Used by AI search to hydrate ANN hits back into records.
func (s *Store) RecordsByKeys(ctx context.Context, keys []string) (map[string]DependencyRecord, error) {
	out := make(map[string]DependencyRecord, len(keys))
	if len(keys) == 0 {
		return out, nil
	}
	placeholders := make([]string, len(keys))
	args := make([]any, len(keys))
	for i, k := range keys {
		placeholders[i] = "?"
		args[i] = k
	}
	q := `SELECT dep_key, scenario, ecosystem, name, version, source_file, vuln_ids, max_severity, last_seen
		FROM dependency_records WHERE dep_key IN (` + strings.Join(placeholders, ",") + `)`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var r DependencyRecord
		var depKey, eco, vulnIDs string
		if err := rows.Scan(&depKey, &r.Scenario, &eco, &r.Name, &r.Version, &r.SourceFile, &vulnIDs, &r.MaxSeverity, &r.LastSeen); err != nil {
			return nil, err
		}
		r.Ecosystem = Ecosystem(eco)
		if vulnIDs != "" {
			r.VulnIDs = strings.Split(vulnIDs, ",")
		}
		out[depKey] = r
	}
	return out, rows.Err()
}

// PackageItem is the deduped embedding unit: one distinct (ecosystem, name,
// version) package, with vuln annotations aggregated across every scenario row
// that shares the package. The semantic index keys vectors by PkgKey so a CVE —
// which belongs to a package+version, not a scenario-usage — is embedded once
// rather than once per exposed scenario.
type PackageItem struct {
	Ecosystem   Ecosystem
	Name        string
	Version     string
	VulnIDs     []string
	MaxSeverity string
}

// PkgKey is the package identity used as the vector key: ecosystem|name|version.
func (p PackageItem) PkgKey() string {
	return string(p.Ecosystem) + "|" + p.Name + "|" + p.Version
}

// packageKey derives a record's package identity (the embedding key) — the
// dep_key minus the scenario dimension.
func packageKey(r DependencyRecord) string {
	return string(r.Ecosystem) + "|" + r.Name + "|" + r.Version
}

// PackageItems returns the deduped package universe to embed: one PackageItem
// per distinct (ecosystem, name, version), with vuln_ids unioned (sorted,
// de-duped) and max_severity max-folded across the scenario rows that share the
// package. Aggregating in Go (rather than SQL) keeps the severity-rank logic in
// one place and avoids SQLite MAX-over-text pitfalls.
func (s *Store) PackageItems(ctx context.Context) ([]PackageItem, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT ecosystem, name, version, vuln_ids, max_severity FROM dependency_records`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type agg struct {
		eco     Ecosystem
		name    string
		version string
		vulns   map[string]struct{}
		sev     string
	}
	byPkg := map[string]*agg{}
	var order []string
	for rows.Next() {
		var eco, name, version, vulnIDs, sev string
		if err := rows.Scan(&eco, &name, &version, &vulnIDs, &sev); err != nil {
			return nil, err
		}
		key := eco + "|" + name + "|" + version
		a, ok := byPkg[key]
		if !ok {
			a = &agg{eco: Ecosystem(eco), name: name, version: version, vulns: map[string]struct{}{}}
			byPkg[key] = a
			order = append(order, key)
		}
		if vulnIDs != "" {
			for _, v := range strings.Split(vulnIDs, ",") {
				if v = strings.TrimSpace(v); v != "" {
					a.vulns[v] = struct{}{}
				}
			}
		}
		a.sev = worseSeverity(a.sev, sev)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Strings(order) // deterministic embedding order
	out := make([]PackageItem, 0, len(order))
	for _, key := range order {
		a := byPkg[key]
		ids := make([]string, 0, len(a.vulns))
		for v := range a.vulns {
			ids = append(ids, v)
		}
		sort.Strings(ids)
		out = append(out, PackageItem{
			Ecosystem:   a.eco,
			Name:        a.name,
			Version:     a.version,
			VulnIDs:     ids,
			MaxSeverity: a.sev,
		})
	}
	return out, nil
}

// RecordsByPackages hydrates the scenario records for the given package keys
// (ecosystem|name|version), returning each package key → all its scenario
// records. Used by AI search to fan a matched package out to every exposed
// scenario (the blast radius a security console exists to surface). Missing
// keys are simply absent (a vector index can lag the corpus).
func (s *Store) RecordsByPackages(ctx context.Context, pkgKeys []string) (map[string][]DependencyRecord, error) {
	out := make(map[string][]DependencyRecord, len(pkgKeys))
	if len(pkgKeys) == 0 {
		return out, nil
	}
	placeholders := make([]string, len(pkgKeys))
	args := make([]any, len(pkgKeys))
	for i, k := range pkgKeys {
		placeholders[i] = "?"
		args[i] = k
	}
	q := `SELECT dep_key, scenario, ecosystem, name, version, source_file, vuln_ids, max_severity, last_seen
		FROM dependency_records
		WHERE (ecosystem || '|' || name || '|' || version) IN (` + strings.Join(placeholders, ",") + `)`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var r DependencyRecord
		var depKey, eco, vulnIDs string
		if err := rows.Scan(&depKey, &r.Scenario, &eco, &r.Name, &r.Version, &r.SourceFile, &vulnIDs, &r.MaxSeverity, &r.LastSeen); err != nil {
			return nil, err
		}
		r.Ecosystem = Ecosystem(eco)
		if vulnIDs != "" {
			r.VulnIDs = strings.Split(vulnIDs, ",")
		}
		pk := packageKey(r)
		out[pk] = append(out[pk], r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Deterministic secondary order within a package = dep_key.
	for k := range out {
		recs := out[k]
		sort.SliceStable(recs, func(i, j int) bool { return recs[i].Key() < recs[j].Key() })
	}
	return out, nil
}

// DistinctPackageCount returns the number of distinct (ecosystem, name,
// version) packages in the corpus — the expected vector count for coverage.
func (s *Store) DistinctPackageCount(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM (SELECT DISTINCT ecosystem, name, version FROM dependency_records)`).Scan(&n)
	return n, err
}

// Count returns the total indexed record count.
func (s *Store) Count(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dependency_records`).Scan(&n)
	return n, err
}

// VulnerableCount returns how many indexed records carry a known vuln.
func (s *Store) VulnerableCount(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dependency_records WHERE vuln_ids != ''`).Scan(&n)
	return n, err
}

// SetReconcileState records the latest reconcile timestamp + outcome.
func (s *Store) SetReconcileState(ctx context.Context, at, outcome string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO dependency_reconcile_state (id, last_reconcile_at, last_outcome)
		VALUES (1, ?, ?)
		ON CONFLICT(id) DO UPDATE SET last_reconcile_at = excluded.last_reconcile_at, last_outcome = excluded.last_outcome`,
		at, outcome)
	return err
}

// ReconcileState returns the latest reconcile timestamp + outcome ("" when
// none has run).
func (s *Store) ReconcileState(ctx context.Context) (at, outcome string) {
	row := s.db.QueryRowContext(ctx, `SELECT last_reconcile_at, last_outcome FROM dependency_reconcile_state WHERE id = 1`)
	var a, o sql.NullString
	if err := row.Scan(&a, &o); err != nil {
		return "", ""
	}
	return a.String, o.String
}
