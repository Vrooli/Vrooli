package dependencies

import (
	"context"
	"database/sql"
	"encoding/json"
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
	if err := s.deleteVulnerabilities(ctx, scenario); err != nil {
		return err
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
		for _, vuln := range r.Vulnerabilities {
			vuln.Scenarios = []string{r.Scenario}
			vuln.SourceFiles = []string{r.SourceFile}
			if vuln.FirstSeen == "" {
				vuln.FirstSeen = now
			}
			vuln.LastSeen = now
			if err := s.upsertVulnerability(ctx, vuln, r.Scenario, r.SourceFile); err != nil {
				return err
			}
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

func (s *Store) deleteVulnerabilities(ctx context.Context, scenario string) error {
	q := `DELETE FROM vulnerability_records`
	var args []any
	if scenario != "" {
		q += ` WHERE scenario = ?`
		args = append(args, scenario)
	}
	if _, err := s.db.ExecContext(ctx, q, args...); err != nil {
		return fmt.Errorf("delete vulnerability records: %w", err)
	}
	return nil
}

func (s *Store) upsertVulnerability(ctx context.Context, vuln VulnerabilityRecord, scenario, sourceFile string) error {
	if strings.TrimSpace(vuln.VulnerabilityID) == "" {
		return nil
	}
	affected, err := json.Marshal(vuln.AffectedRanges)
	if err != nil {
		return fmt.Errorf("marshal affected ranges for %s: %w", vuln.VulnerabilityID, err)
	}
	fixed, err := json.Marshal(vuln.FixedRanges)
	if err != nil {
		return fmt.Errorf("marshal fixed ranges for %s: %w", vuln.VulnerabilityID, err)
	}
	key := strings.Join([]string{
		vuln.VulnerabilityID,
		string(vuln.Ecosystem),
		vuln.Name,
		vuln.Version,
		scenario,
		sourceFile,
		string(vuln.Source),
	}, "|")
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO vulnerability_records (
			vuln_key, vulnerability_id, aliases, ecosystem, name, version,
			affected_ranges, fixed_ranges, severity, normalized_severity,
			advisory_url, summary, details, source, reachability, confidence,
			production, dev_only, first_seen, last_seen, scenario, source_file, remediation
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(vuln_key) DO UPDATE SET
			aliases = excluded.aliases,
			affected_ranges = excluded.affected_ranges,
			fixed_ranges = excluded.fixed_ranges,
			severity = excluded.severity,
			normalized_severity = excluded.normalized_severity,
			advisory_url = excluded.advisory_url,
			summary = excluded.summary,
			details = excluded.details,
			reachability = excluded.reachability,
			confidence = excluded.confidence,
			production = excluded.production,
			dev_only = excluded.dev_only,
			last_seen = excluded.last_seen,
			remediation = excluded.remediation`,
		key,
		vuln.VulnerabilityID,
		strings.Join(uniqueSorted(vuln.Aliases), ","),
		string(vuln.Ecosystem),
		vuln.Name,
		vuln.Version,
		string(affected),
		string(fixed),
		vuln.Severity,
		vuln.NormalizedSeverity,
		vuln.AdvisoryURL,
		vuln.Summary,
		vuln.Details,
		string(vuln.Source),
		string(vuln.Reachability),
		string(vuln.Confidence),
		boolInt(vuln.Production),
		boolInt(vuln.DevOnly),
		vuln.FirstSeen,
		vuln.LastSeen,
		scenario,
		sourceFile,
		vuln.Remediation,
	)
	if err != nil {
		return fmt.Errorf("upsert vulnerability %s: %w", key, err)
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

// ListVulnerabilities returns structured vulnerability evidence grouped by
// vulnerability/package/version/source so callers can see fleet impact.
func (s *Store) ListVulnerabilities(ctx context.Context, req VulnerabilityQuery) (VulnerabilityList, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	if limit > MaxSearchLimit {
		limit = MaxSearchLimit
	}

	var where []string
	var args []any
	if req.Ecosystem != EcosystemUnspecified {
		where = append(where, "ecosystem = ?")
		args = append(args, string(req.Ecosystem))
	}
	if strings.TrimSpace(req.PackageName) != "" {
		where = append(where, "name = ?")
		args = append(args, strings.TrimSpace(req.PackageName))
	}
	if strings.TrimSpace(req.Scenario) != "" {
		where = append(where, "scenario = ?")
		args = append(args, strings.TrimSpace(req.Scenario))
	}
	if strings.TrimSpace(req.VulnerabilityID) != "" {
		where = append(where, "vulnerability_id = ?")
		args = append(args, strings.TrimSpace(req.VulnerabilityID))
	}
	if req.MinimumConfidence != EvidenceConfidenceUnspecified {
		where = append(where, confidencePredicate(req.MinimumConfidence))
	}
	q := `SELECT vulnerability_id, aliases, ecosystem, name, version, affected_ranges, fixed_ranges,
		severity, normalized_severity, advisory_url, summary, details, source, reachability,
		confidence, production, dev_only, first_seen, last_seen, scenario, source_file, remediation
		FROM vulnerability_records`
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += ` ORDER BY normalized_severity DESC, vulnerability_id, ecosystem, name, version, scenario`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return VulnerabilityList{}, err
	}
	defer rows.Close()

	grouped := map[string]*VulnerabilityRecord{}
	var order []string
	for rows.Next() {
		rec, scenario, sourceFile, err := scanVulnerabilityRow(rows)
		if err != nil {
			return VulnerabilityList{}, err
		}
		key := vulnerabilityGroupKey(rec)
		existing, ok := grouped[key]
		if !ok {
			rec.Scenarios = nil
			rec.SourceFiles = nil
			existing = &rec
			grouped[key] = existing
			order = append(order, key)
		}
		existing.Scenarios = appendUnique(existing.Scenarios, scenario)
		existing.SourceFiles = appendUnique(existing.SourceFiles, sourceFile)
		if existing.FirstSeen == "" || (rec.FirstSeen != "" && rec.FirstSeen < existing.FirstSeen) {
			existing.FirstSeen = rec.FirstSeen
		}
		if rec.LastSeen > existing.LastSeen {
			existing.LastSeen = rec.LastSeen
		}
	}
	if err := rows.Err(); err != nil {
		return VulnerabilityList{}, err
	}
	sort.Strings(order)
	total := len(order)
	if total == 0 {
		return s.listVulnerabilitiesFromDependencyRows(ctx, req, limit)
	}
	if len(order) > limit {
		order = order[:limit]
	}
	out := make([]VulnerabilityRecord, 0, len(order))
	for _, key := range order {
		rec := *grouped[key]
		sort.Strings(rec.Scenarios)
		sort.Strings(rec.SourceFiles)
		out = append(out, rec)
	}
	return VulnerabilityList{Vulnerabilities: out, Total: total}, nil
}

// listVulnerabilitiesFromDependencyRows preserves the structured evidence API
// for corpuses reconciled before vulnerability_records existed. It intentionally
// reports degraded confidence because affected/fixed range detail is absent.
func (s *Store) listVulnerabilitiesFromDependencyRows(ctx context.Context, req VulnerabilityQuery, limit int) (VulnerabilityList, error) {
	if req.MinimumConfidence == EvidenceConfidenceAdvisory || req.MinimumConfidence == EvidenceConfidenceGating {
		return VulnerabilityList{}, nil
	}

	var where []string
	var args []any
	where = append(where, "vuln_ids != ''")
	if req.Ecosystem != EcosystemUnspecified {
		where = append(where, "ecosystem = ?")
		args = append(args, string(req.Ecosystem))
	}
	if strings.TrimSpace(req.PackageName) != "" {
		where = append(where, "name = ?")
		args = append(args, strings.TrimSpace(req.PackageName))
	}
	if strings.TrimSpace(req.Scenario) != "" {
		where = append(where, "scenario = ?")
		args = append(args, strings.TrimSpace(req.Scenario))
	}

	q := `SELECT scenario, ecosystem, name, version, source_file, vuln_ids, max_severity, last_seen
		FROM dependency_records WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY ecosystem, name, version, scenario`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return VulnerabilityList{}, err
	}
	defer rows.Close()

	filterID := strings.TrimSpace(req.VulnerabilityID)
	grouped := map[string]*VulnerabilityRecord{}
	var order []string
	for rows.Next() {
		var scenario, eco, name, version, sourceFile, vulnIDs, sev, lastSeen string
		if err := rows.Scan(&scenario, &eco, &name, &version, &sourceFile, &vulnIDs, &sev, &lastSeen); err != nil {
			return VulnerabilityList{}, err
		}
		for _, vulnID := range splitCSV(vulnIDs) {
			if filterID != "" && vulnID != filterID {
				continue
			}
			rec := VulnerabilityRecord{
				VulnerabilityID:    vulnID,
				Ecosystem:          Ecosystem(eco),
				Name:               name,
				Version:            version,
				Severity:           sev,
				NormalizedSeverity: sev,
				Reachability:       ReachabilityLockfileAffected,
				Confidence:         EvidenceConfidenceDegraded,
				FirstSeen:          lastSeen,
				LastSeen:           lastSeen,
				Remediation:        "Re-run dependency reconcile to populate affected and fixed version evidence.",
			}
			key := vulnerabilityGroupKey(rec)
			existing, ok := grouped[key]
			if !ok {
				existing = &rec
				grouped[key] = existing
				order = append(order, key)
			}
			existing.Scenarios = appendUnique(existing.Scenarios, scenario)
			existing.SourceFiles = appendUnique(existing.SourceFiles, sourceFile)
			existing.NormalizedSeverity = worseSeverity(existing.NormalizedSeverity, sev)
			existing.Severity = existing.NormalizedSeverity
			if existing.FirstSeen == "" || (lastSeen != "" && lastSeen < existing.FirstSeen) {
				existing.FirstSeen = lastSeen
			}
			if lastSeen > existing.LastSeen {
				existing.LastSeen = lastSeen
			}
		}
	}
	if err := rows.Err(); err != nil {
		return VulnerabilityList{}, err
	}

	sort.Strings(order)
	total := len(order)
	if len(order) > limit {
		order = order[:limit]
	}
	out := make([]VulnerabilityRecord, 0, len(order))
	for _, key := range order {
		rec := *grouped[key]
		sort.Strings(rec.Scenarios)
		sort.Strings(rec.SourceFiles)
		out = append(out, rec)
	}
	return VulnerabilityList{Vulnerabilities: out, Total: total}, nil
}

func scanVulnerabilityRow(rows *sql.Rows) (VulnerabilityRecord, string, string, error) {
	var rec VulnerabilityRecord
	var aliases, eco, affectedRaw, fixedRaw, source, reachability, confidence, scenario, sourceFile string
	var production, devOnly int
	if err := rows.Scan(
		&rec.VulnerabilityID,
		&aliases,
		&eco,
		&rec.Name,
		&rec.Version,
		&affectedRaw,
		&fixedRaw,
		&rec.Severity,
		&rec.NormalizedSeverity,
		&rec.AdvisoryURL,
		&rec.Summary,
		&rec.Details,
		&source,
		&reachability,
		&confidence,
		&production,
		&devOnly,
		&rec.FirstSeen,
		&rec.LastSeen,
		&scenario,
		&sourceFile,
		&rec.Remediation,
	); err != nil {
		return VulnerabilityRecord{}, "", "", err
	}
	rec.Ecosystem = Ecosystem(eco)
	rec.Aliases = splitCSV(aliases)
	rec.Source = VulnerabilitySource(source)
	rec.Reachability = Reachability(reachability)
	rec.Confidence = EvidenceConfidence(confidence)
	rec.Production = production != 0
	rec.DevOnly = devOnly != 0
	_ = json.Unmarshal([]byte(affectedRaw), &rec.AffectedRanges)
	_ = json.Unmarshal([]byte(fixedRaw), &rec.FixedRanges)
	return rec, scenario, sourceFile, nil
}

func vulnerabilityGroupKey(rec VulnerabilityRecord) string {
	return strings.Join([]string{
		rec.VulnerabilityID,
		string(rec.Ecosystem),
		rec.Name,
		rec.Version,
		string(rec.Source),
	}, "|")
}

func confidencePredicate(min EvidenceConfidence) string {
	switch min {
	case EvidenceConfidenceGating:
		return "confidence = 'gating'"
	case EvidenceConfidenceAdvisory:
		return "confidence IN ('advisory', 'gating')"
	case EvidenceConfidenceDegraded:
		return "confidence IN ('degraded', 'advisory', 'gating')"
	default:
		return "1 = 1"
	}
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return uniqueSorted(strings.Split(raw, ","))
}

func uniqueSorted(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func appendUnique(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
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
