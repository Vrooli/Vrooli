package versionledger

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Repository struct {
	db         *sql.DB
	sourceRoot string
}

func NewRepository(db *sql.DB, sourceRoot string) *Repository {
	return &Repository{db: db, sourceRoot: sourceRoot}
}

func (r *Repository) List(ctx context.Context, libraryID string) ([]VersionLedger, error) {
	return r.ListWindow(ctx, libraryID, "")
}

func (r *Repository) ListWindow(ctx context.Context, libraryID, window string) ([]VersionLedger, error) {
	query := `SELECT library_id, version, created_at, released_at, retired_at, lifecycle_state, gate_pass_count, gate_fail_count, test_runs, test_pass_rate, adoption_current, adoption_peak, file_count, lines_of_code, dependency_count FROM version_ledger`
	args := []any{}
	filters := []string{}
	if libraryID != "" {
		filters = append(filters, "library_id = ?")
		args = append(args, libraryID)
	}
	if since := windowSince(window); !since.IsZero() {
		filters = append(filters, "created_at >= ?")
		args = append(args, since.Format(time.RFC3339Nano))
	}
	if len(filters) > 0 {
		query += " WHERE " + strings.Join(filters, " AND ")
	}
	query += " ORDER BY library_id, version"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []VersionLedger
	for rows.Next() {
		var v VersionLedger
		var created, released, retired string
		if err := rows.Scan(&v.LibraryID, &v.Version, &created, &released, &retired, &v.LifecycleState, &v.GatePassCount, &v.GateFailCount, &v.TestRuns, &v.TestPassRate, &v.AdoptionCurrent, &v.AdoptionPeak, &v.FileCount, &v.LinesOfCode, &v.DependencyCount); err != nil {
			return nil, err
		}
		v.CreatedAt = parseTime(created)
		v.ReleasedAt = parseTime(released)
		v.RetiredAt = parseTime(retired)
		out = append(out, v)
	}
	return out, rows.Err()
}

func windowSince(window string) time.Time {
	switch strings.ToLower(strings.TrimSpace(window)) {
	case "last_7d", "this_week":
		return time.Now().UTC().Add(-7 * 24 * time.Hour)
	case "last_30d", "this_month":
		return time.Now().UTC().Add(-30 * 24 * time.Hour)
	case "last_month":
		return time.Now().UTC().Add(-60 * 24 * time.Hour)
	case "this_quarter":
		return time.Now().UTC().Add(-90 * 24 * time.Hour)
	}
	return time.Time{}
}

func (r *Repository) Rebuild(ctx context.Context) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	catalogByLibrary := r.catalogIDs()
	rows, err := tx.QueryContext(ctx, `SELECT id, component_id, library_id, version, status, created_at, released_at, content FROM component_versions`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var versionID, componentID, libraryID, version, status, created, released, content string
		if err := rows.Scan(&versionID, &componentID, &libraryID, &version, &status, &created, &released, &content); err != nil {
			rows.Close()
			return err
		}
		catalogID := catalogByLibrary[libraryID]
		var pass, fail int
		if catalogID != "" {
			_ = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM catalog_gate_evidence WHERE asset_id = ? AND version = ? AND lower(result) = 'pass'`, catalogID, version).Scan(&pass)
			_ = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM catalog_gate_evidence WHERE asset_id = ? AND version = ? AND lower(result) IN ('fail','failed')`, catalogID, version).Scan(&fail)
		}
		var runs, passed, adoption, dependencies, files, lines int
		_ = tx.QueryRowContext(ctx, `SELECT runs_total, runs_passed FROM component_version_test_rollup WHERE library_id = ? AND version = ?`, libraryID, version).Scan(&runs, &passed)
		_ = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM adoption_records WHERE library_id = ? AND adopted_version = ?`, libraryID, version).Scan(&adoption)
		_ = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_asset_dependencies WHERE component_id = ? AND version = ?`, componentID, version).Scan(&dependencies)
		_ = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_version_files WHERE version_id = ?`, versionID).Scan(&files)
		if files == 0 {
			files = 1
		}
		lines = lineCount(content)
		var retired string
		_ = tx.QueryRowContext(ctx, `SELECT retired_at FROM version_ledger WHERE library_id = ? AND version = ?`, libraryID, version).Scan(&retired)
		state := status
		if retired != "" {
			state = "retired"
		}
		rate := 0.0
		if runs > 0 {
			rate = float64(passed) / float64(runs)
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO version_ledger (library_id, version, created_at, released_at, retired_at, lifecycle_state, gate_pass_count, gate_fail_count, test_runs, test_pass_rate, adoption_current, adoption_peak, file_count, lines_of_code, dependency_count) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(library_id, version) DO UPDATE SET created_at=excluded.created_at, released_at=excluded.released_at, lifecycle_state=CASE WHEN version_ledger.retired_at<>'' THEN 'retired' ELSE excluded.lifecycle_state END, gate_pass_count=excluded.gate_pass_count, gate_fail_count=excluded.gate_fail_count, test_runs=excluded.test_runs, test_pass_rate=excluded.test_pass_rate, adoption_current=excluded.adoption_current, adoption_peak=CASE WHEN version_ledger.adoption_peak > excluded.adoption_peak THEN version_ledger.adoption_peak ELSE excluded.adoption_peak END, file_count=excluded.file_count, lines_of_code=excluded.lines_of_code, dependency_count=excluded.dependency_count`, libraryID, version, created, released, retired, state, pass, fail, runs, rate, adoption, adoption, files, lines, dependencies)
		if err != nil {
			rows.Close()
			return err
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	return tx.Commit()
}

type Candidate struct{ ComponentID, LibraryID, Version, Status string }

func (r *Repository) PlanCleanup(ctx context.Context, scope CleanupScope) ([]CleanupItem, string, error) {
	if scope.ComponentID != "" && scope.LibraryID != "" {
		return nil, "", fmt.Errorf("cleanup scope cannot contain both component_id and library_id")
	}
	query := `SELECT c.id, c.library_id, v.version, v.status, c.latest_version, c.draft_version, v.created_at, v.released_at
		FROM components c JOIN component_versions v ON v.component_id = c.id`
	args := []any{}
	filters := []string{"lower(v.status) <> 'retired'"}
	if scope.ComponentID != "" {
		filters = append(filters, "(c.id = ? OR c.library_id = ?)")
		args = append(args, scope.ComponentID, scope.ComponentID)
	}
	if scope.LibraryID != "" {
		filters = append(filters, "c.library_id = ?")
		args = append(args, scope.LibraryID)
	}
	if len(filters) > 0 {
		query += " WHERE " + strings.Join(filters, " AND ")
	}
	query += " ORDER BY c.library_id, v.version"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	type cleanupRaw struct {
		candidate                        Candidate
		latest, draft, created, released string
	}
	var rawRows []cleanupRaw
	for rows.Next() {
		var raw cleanupRaw
		if err := rows.Scan(&raw.candidate.ComponentID, &raw.candidate.LibraryID, &raw.candidate.Version, &raw.candidate.Status, &raw.latest, &raw.draft, &raw.created, &raw.released); err != nil {
			rows.Close()
			return nil, "", err
		}
		rawRows = append(rawRows, raw)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, "", err
	}
	rows.Close()
	sourceRefs, err := r.sourceReferences(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("build source reference graph: %w", err)
	}
	var items []CleanupItem
	for _, raw := range rawRows {
		var item CleanupItem
		item.Candidate = raw.candidate
		item.AgeDays = versionAgeDays(raw.created, raw.released)
		if item.Candidate.Version == raw.latest {
			item.Reason = "latest version"
		} else if item.Candidate.Version == raw.draft || strings.Contains(strings.ToLower(item.Candidate.Status), "draft") {
			item.Reason = "active draft"
		} else if strings.EqualFold(item.Candidate.Status, "retired") {
			item.Reason = "already retired"
		} else {
			var directAdoptions int
			_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM adoption_records WHERE (component_id = ? OR library_id = ?) AND adopted_version = ?`, item.Candidate.ComponentID, item.Candidate.LibraryID, item.Candidate.Version).Scan(&directAdoptions)
			var fileAdoptions int
			_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM adoption_files WHERE source_library_id = ? AND source_version = ?`, item.Candidate.LibraryID, item.Candidate.Version).Scan(&fileAdoptions)
			item.AdoptionCount = directAdoptions
			if fileAdoptions > item.AdoptionCount {
				item.AdoptionCount = fileAdoptions
			}
			_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_asset_dependencies WHERE library_id = ? AND version = ?`, item.Candidate.LibraryID, item.Candidate.Version).Scan(&item.DependencyCount)
			item.References = sourceRefs[sourceReferenceKey(item.Candidate.LibraryID, item.Candidate.Version)]
			switch {
			case item.AdoptionCount > 0:
				item.Reason = "referenced by adoption"
			case item.DependencyCount > 0:
				item.Reason = "pinned dependency"
			case len(item.References) > 0:
				item.Reason = "referenced by source import"
			case scope.OlderThanDays > 0 && item.AgeDays < scope.OlderThanDays:
				item.Reason = fmt.Sprintf("younger than %d days", scope.OlderThanDays)
			default:
				item.Eligible = true
				item.Reason = "safe to retire"
			}
		}
		items = append(items, item)
	}
	return items, cleanupPlanHash(items, scope), nil
}

func versionAgeDays(created, released string) int {
	stamp := parseTime(released)
	if stamp.IsZero() {
		stamp = parseTime(created)
	}
	if stamp.IsZero() {
		return 0
	}
	age := int(time.Since(stamp).Hours() / 24)
	if age < 0 {
		return 0
	}
	return age
}

func cleanupPlanHash(items []CleanupItem, scope CleanupScope) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s|%s|%d|", scope.ComponentID, scope.LibraryID, scope.OlderThanDays)
	for _, item := range items {
		fmt.Fprintf(&b, "%s:%s:%s:%t:%s|", item.Candidate.LibraryID, item.Candidate.Version, item.Candidate.Status, item.Eligible, item.Reason)
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(b.String())))
}

func (r *Repository) CleanupVersions(ctx context.Context, scope CleanupScope, planHash string, confirm bool) ([]CleanupItem, string, int, error) {
	items, currentHash, err := r.PlanCleanup(ctx, scope)
	if err != nil {
		return nil, "", 0, err
	}
	if !confirm {
		return items, currentHash, 0, nil
	}
	if planHash == "" || planHash != currentHash {
		return items, currentHash, 0, fmt.Errorf("cleanup plan changed or --plan-hash is missing; review the current dry-run plan")
	}
	retired := 0
	for _, item := range items {
		if !item.Eligible {
			continue
		}
		if _, err := r.Transition(ctx, item.Candidate.ComponentID, item.Candidate.Version, "retired", true); err != nil {
			return items, currentHash, retired, err
		}
		retired++
	}
	return items, currentHash, retired, nil
}

func (r *Repository) CleanupDraft(ctx context.Context, componentID string, olderThanDays int, confirm bool) (CleanupItem, error) {
	var item CleanupItem
	var sourcePath, created string
	if err := r.db.QueryRowContext(ctx, `SELECT c.id, c.library_id, v.version, v.status, v.source_path, v.created_at FROM components c JOIN component_versions v ON v.component_id = c.id WHERE (c.id = ? OR c.library_id = ?) AND v.version = c.draft_version`, componentID, componentID).Scan(&item.Candidate.ComponentID, &item.Candidate.LibraryID, &item.Candidate.Version, &item.Candidate.Status, &sourcePath, &created); err != nil {
		if err == sql.ErrNoRows {
			return item, fmt.Errorf("no active draft found for %s", componentID)
		}
		return item, err
	}
	item.AgeDays = versionAgeDays(created, "")
	if !strings.Contains(strings.ToLower(item.Candidate.Status), "draft") {
		item.Reason = "version is not a draft"
	} else if olderThanDays > 0 && item.AgeDays < olderThanDays {
		item.Reason = fmt.Sprintf("younger than %d days", olderThanDays)
	} else {
		item.Eligible = true
		item.Reason = "safe to discard draft"
	}
	if !item.Eligible || !confirm {
		return item, nil
	}
	path := filepath.Clean(filepath.Join(r.sourceRoot, sourcePath))
	root := filepath.Clean(r.sourceRoot) + string(os.PathSeparator)
	if !strings.HasPrefix(path, root) {
		return item, fmt.Errorf("refusing to discard path outside library root")
	}
	if err := os.RemoveAll(filepath.Dir(path)); err != nil {
		return item, err
	}
	manifestPath := ""
	if err := r.db.QueryRowContext(ctx, `SELECT manifest_path FROM components WHERE id = ?`, item.Candidate.ComponentID).Scan(&manifestPath); err != nil {
		return item, err
	}
	if err := r.clearDraftManifest(manifestPath); err != nil {
		return item, err
	}
	_, err := r.db.ExecContext(ctx, `UPDATE components SET draft_version = '' WHERE id = ?`, item.Candidate.ComponentID)
	if err != nil {
		return item, err
	}
	_, err = r.db.ExecContext(ctx, `DELETE FROM component_versions WHERE component_id = ? AND version = ?`, item.Candidate.ComponentID, item.Candidate.Version)
	return item, err
}

func (r *Repository) RetireCandidates(ctx context.Context, componentID string) ([]Candidate, error) {
	refs, err := r.sourceReferences(ctx)
	if err != nil {
		return nil, fmt.Errorf("build source reference graph: %w", err)
	}
	query := `SELECT c.id, c.library_id, v.version, v.status FROM components c JOIN component_versions v ON v.component_id = c.id WHERE v.version <> c.latest_version AND v.version <> c.draft_version AND lower(v.status) NOT LIKE 'draft%' AND lower(v.status) <> 'retired' AND NOT EXISTS (SELECT 1 FROM adoption_records a WHERE a.component_id = c.id AND a.adopted_version = v.version) AND NOT EXISTS (SELECT 1 FROM adoption_files f WHERE f.source_library_id = c.library_id AND f.source_version = v.version) AND NOT EXISTS (SELECT 1 FROM component_asset_dependencies d WHERE d.library_id = c.library_id AND d.version = v.version)`
	args := []any{}
	if componentID != "" {
		query += " AND (c.id = ? OR c.library_id = ?)"
		args = append(args, componentID)
		args = append(args, componentID)
	}
	query += " ORDER BY c.library_id, v.version"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Candidate
	for rows.Next() {
		var c Candidate
		if err := rows.Scan(&c.ComponentID, &c.LibraryID, &c.Version, &c.Status); err != nil {
			return nil, err
		}
		if len(refs[sourceReferenceKey(c.LibraryID, c.Version)]) > 0 {
			continue
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) Transition(ctx context.Context, componentID, version, state string, confirm bool) (Candidate, error) {
	var c Candidate
	var latest, draft, sourcePath, manifestPath string
	if err := r.db.QueryRowContext(ctx, `SELECT c.id, c.library_id, v.version, v.status, v.source_path, c.latest_version, c.draft_version, c.manifest_path FROM components c JOIN component_versions v ON v.component_id=c.id WHERE (c.id=? OR c.library_id=?) AND v.version=?`, componentID, componentID, version).Scan(&c.ComponentID, &c.LibraryID, &c.Version, &c.Status, &sourcePath, &latest, &draft, &manifestPath); err != nil {
		return c, err
	}
	if version == latest || version == draft {
		return c, fmt.Errorf("version %s is latest or draft and cannot be retired", version)
	}
	if state == "retired" {
		if !confirm {
			return c, fmt.Errorf("retiring a version requires --confirm")
		}
		candidates, err := r.RetireCandidates(ctx, componentID)
		if err != nil {
			return c, err
		}
		allowed := false
		for _, candidate := range candidates {
			if candidate.Version == version {
				allowed = true
				break
			}
		}
		if !allowed {
			return c, fmt.Errorf("version %s is still referenced and is not safe to retire", version)
		}
		// Record the retirement in the manifest before removing its source
		// directory. The manifest is the durable catalog declaration and must
		// retain the version even after its implementation is retired.
		if err := r.updateManifest(manifestPath, version); err != nil {
			return c, err
		}
		path := filepath.Clean(filepath.Join(r.sourceRoot, sourcePath))
		root := filepath.Clean(r.sourceRoot) + string(os.PathSeparator)
		if !strings.HasPrefix(path, root) {
			return c, fmt.Errorf("refusing to retire path outside library root")
		}
		if err := os.RemoveAll(filepath.Dir(path)); err != nil {
			return c, err
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		if _, err := r.db.ExecContext(ctx, `UPDATE version_ledger SET retired_at=?, lifecycle_state='retired' WHERE library_id=? AND version=?`, now, c.LibraryID, version); err != nil {
			return c, err
		}
		if _, err := r.db.ExecContext(ctx, `UPDATE component_versions SET status='retired' WHERE component_id=? AND version=?`, c.ComponentID, version); err != nil {
			return c, err
		}
		return c, nil
	}
	if state == "deprecated" || state == "archived" {
		if err := r.updateManifest(manifestPath, version); err != nil {
			return c, err
		}
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE component_versions SET status=? WHERE component_id=? AND version=?`, state, componentID, version); err != nil {
		return c, err
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE version_ledger SET lifecycle_state=? WHERE library_id=? AND version=?`, state, c.LibraryID, version); err != nil {
		return c, err
	}
	c.Status = state
	return c, nil
}

func (r *Repository) updateManifest(manifestPath, version string) error {
	path := filepath.Join(r.sourceRoot, manifestPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}
	values := []string{}
	if raw, ok := doc["deprecatedVersions"].([]any); ok {
		for _, value := range raw {
			if v, ok := value.(string); ok {
				values = append(values, v)
			}
		}
	}
	for _, value := range values {
		if value == version {
			return nil
		}
	}
	values = append(values, version)
	doc["deprecatedVersions"] = values
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(path, out, 0o644)
}

func (r *Repository) clearDraftManifest(manifestPath string) error {
	path := filepath.Join(r.sourceRoot, manifestPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}
	doc["draft"] = ""
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(path, out, 0o644)
}

func (r *Repository) catalogIDs() map[string]string {
	out := map[string]string{}
	paths, _ := filepath.Glob(filepath.Join(r.sourceRoot, "*", "*", "component.json"))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var doc struct {
			LibraryID string `json:"libraryId"`
			CatalogID string `json:"catalogId"`
		}
		if json.Unmarshal(data, &doc) == nil && doc.LibraryID != "" {
			out[doc.LibraryID] = doc.CatalogID
		}
	}
	return out
}

func parseTime(raw string) time.Time {
	if raw == "" {
		return time.Time{}
	}
	value, _ := time.Parse(time.RFC3339Nano, raw)
	return value
}

func lineCount(raw string) int {
	if strings.TrimSpace(raw) == "" {
		return 0
	}
	return strings.Count(raw, "\n") + 1
}
func (v VersionLedger) String() string { return fmt.Sprintf("%s@%s", v.LibraryID, v.Version) }
