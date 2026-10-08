package versionledger

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"react-component-library/internal/librarywalk"
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
	query := `SELECT l.library_id, l.version, l.created_at, l.released_at, l.retired_at, l.lifecycle_state, l.gate_pass_count, l.gate_fail_count, l.test_runs, l.test_pass_rate, l.adoption_current, l.adoption_peak, l.file_count, l.lines_of_code, l.dependency_count, COALESCE(v.presence, 'materialized') FROM version_ledger l LEFT JOIN component_versions v ON v.library_id = l.library_id AND v.version = l.version`
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
		if err := rows.Scan(&v.LibraryID, &v.Version, &created, &released, &retired, &v.LifecycleState, &v.GatePassCount, &v.GateFailCount, &v.TestRuns, &v.TestPassRate, &v.AdoptionCurrent, &v.AdoptionPeak, &v.FileCount, &v.LinesOfCode, &v.DependencyCount, &v.Presence); err != nil {
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
	if err := rows.Close(); err != nil {
		return nil, "", err
	}
	rows.Close()
	graph, err := r.BuildReachability(ctx)
	if err != nil {
		return nil, "", err
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
		} else if strings.EqualFold(item.Candidate.Status, "retired") || r.isEvicted(ctx, item.Candidate.ComponentID, item.Candidate.Version) {
			item.Reason = "already retired"
		} else if hasUnreadableVersion(graph.Unreadable, item.Candidate.LibraryID, item.Candidate.Version) {
			// An evicted version without mirror bytes is a repair defect, not
			// cleanup work. Keep it visible to doctor/purge and never transition
			// it through the destructive retirement path.
			item.Reason = "unreadable mirror"
		} else {
			var directAdoptions int
			_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM adoption_records WHERE (component_id = ? OR library_id = ?) AND adopted_version = ? AND lower(COALESCE(mode, 'copied')) <> 'ejected'`, item.Candidate.ComponentID, item.Candidate.LibraryID, item.Candidate.Version).Scan(&directAdoptions)
			var fileAdoptions int
			_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM adoption_files f JOIN adoption_records a ON a.id = f.adoption_id WHERE f.source_library_id = ? AND f.source_version = ? AND lower(COALESCE(a.mode, 'copied')) <> 'ejected'`, item.Candidate.LibraryID, item.Candidate.Version).Scan(&fileAdoptions)
			item.AdoptionCount = directAdoptions
			if fileAdoptions > item.AdoptionCount {
				item.AdoptionCount = fileAdoptions
			}
			_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_asset_dependencies WHERE library_id = ? AND version = ?`, item.Candidate.LibraryID, item.Candidate.Version).Scan(&item.DependencyCount)
			item.References = graph.References[sourceReferenceKey(item.Candidate.LibraryID, item.Candidate.Version)]
			if graph.IsReachable(item.Candidate.LibraryID, item.Candidate.Version) {
				item.Reason = "referenced by source import"
				items = append(items, item)
				continue
			}
			if r.hasUnindexedReference(ctx, item.References) {
				// A lock or source reference whose owner is absent from the
				// index is an unobserved consumer. Retention cannot prove that
				// owner will be retired with this plan, so keep the target.
				item.Reason = "referenced by source import"
				items = append(items, item)
				continue
			}
			switch {
			case item.AdoptionCount > 0:
				item.Reason = "referenced by adoption"
			case scope.OlderThanDays > 0 && item.AgeDays < scope.OlderThanDays:
				item.Reason = fmt.Sprintf("younger than %d days", scope.OlderThanDays)
			default:
				// References from an unreachable historical owner do not keep
				// this candidate live: that owner is itself outside the root
				// closure and can be retired in the same governed cleanup plan.
				// References from live owners were handled by graph.IsReachable
				// above, and external references are roots in BuildReachability.
				item.Eligible = true
				item.Reason = "safe to retire"
			}
		}
		items = append(items, item)
	}
	return items, cleanupPlanHash(items, scope), nil
}

func (r *Repository) isEvicted(ctx context.Context, componentID, version string) bool {
	var presence string
	err := r.db.QueryRowContext(ctx, `SELECT presence FROM component_versions WHERE component_id=? AND version=?`, componentID, version).Scan(&presence)
	return err == nil && strings.EqualFold(presence, "evicted")
}

func (r *Repository) hasUnindexedReference(ctx context.Context, references []VersionReference) bool {
	for _, ref := range references {
		if strings.TrimSpace(ref.OwnerLibraryID) == "" || strings.TrimSpace(ref.OwnerVersion) == "" {
			continue
		}
		var exists int
		err := r.db.QueryRowContext(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM components c
				JOIN component_versions v ON v.component_id = c.id
				WHERE c.library_id = ? AND v.version = ?
			)`, ref.OwnerLibraryID, ref.OwnerVersion).Scan(&exists)
		if err == nil && exists == 0 {
			return true
		}
	}
	return false
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
	// Every protection above is a search for evidence that a version is still
	// needed, so a missing index row reads exactly like an absent reference.
	// That asymmetry only ever errs toward deleting, which is why an index that
	// disagrees with the filesystem refuses the destructive pass rather than
	// warning through it.
	drift, err := r.IndexDrift(ctx)
	if err != nil {
		return items, currentHash, 0, fmt.Errorf("compare index against library tree: %w", err)
	}
	if !drift.Empty() {
		return items, currentHash, 0, fmt.Errorf("refusing to retire versions: %s; run the catalog reindex and re-plan before cleanup", drift.Error())
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
	graph, err := r.BuildReachability(ctx)
	if err != nil {
		return nil, err
	}
	query := `SELECT c.id, c.library_id, v.version, v.status FROM components c JOIN component_versions v ON v.component_id = c.id WHERE v.version <> c.latest_version AND v.version <> c.draft_version AND lower(v.status) NOT LIKE 'draft%' AND lower(v.status) <> 'retired' AND NOT EXISTS (SELECT 1 FROM adoption_records a WHERE a.component_id = c.id AND a.adopted_version = v.version AND lower(COALESCE(a.mode, 'copied')) <> 'ejected') AND NOT EXISTS (SELECT 1 FROM adoption_files f JOIN adoption_records a ON a.id = f.adoption_id WHERE f.source_library_id = c.library_id AND f.source_version = v.version AND lower(COALESCE(a.mode, 'copied')) <> 'ejected')`
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
		if graph.IsReachable(c.LibraryID, c.Version) || hasUnreadableVersion(graph.Unreadable, c.LibraryID, c.Version) {
			continue
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) Transition(ctx context.Context, componentID, version, state string, confirm bool, planHashes ...string) (Candidate, error) {
	var c Candidate
	var latest, draft, sourcePath, manifestPath string
	if err := r.db.QueryRowContext(ctx, `SELECT c.id, c.library_id, v.version, v.status, v.source_path, c.latest_version, c.draft_version, c.manifest_path FROM components c JOIN component_versions v ON v.component_id=c.id WHERE (c.id=? OR c.library_id=?) AND v.version=?`, componentID, componentID, version).Scan(&c.ComponentID, &c.LibraryID, &c.Version, &c.Status, &sourcePath, &latest, &draft, &manifestPath); err != nil {
		return c, err
	}
	presence := "materialized"
	_ = r.db.QueryRowContext(ctx, `SELECT COALESCE(presence, 'materialized') FROM component_versions WHERE component_id=? AND version=?`, c.ComponentID, version).Scan(&presence)
	assetRetirement := state == "retired" && version == latest
	if state == "retired" && strings.EqualFold(c.Status, "retired") {
		if !confirm {
			return c, fmt.Errorf("retiring a version requires --confirm")
		}
		return c, r.reclaimRetiredMaterialization(ctx, c, sourcePath, manifestPath)
	}
	if version == draft || version == latest && !assetRetirement {
		return c, fmt.Errorf("version %s is latest or draft and cannot be retired", version)
	}
	if state == "retired" {
		if !confirm {
			return c, fmt.Errorf("retiring a version requires --confirm")
		}
		if assetRetirement {
			if err := r.validateAssetRetirement(ctx, c, draft, manifestPath); err != nil {
				return c, err
			}
		} else {
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
		}
		var mirrorRows int
		mirrorErr := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_version_files WHERE version_id=(SELECT id FROM component_versions WHERE component_id=? AND version=?)`, c.ComponentID, version).Scan(&mirrorRows)
		if mirrorErr == nil && mirrorRows == 0 {
			return c, fmt.Errorf("refusing to retire %s@%s: file mirror is empty", c.LibraryID, version)
		}
		if mirrorErr != nil && !strings.Contains(strings.ToLower(mirrorErr.Error()), "no such table") {
			return c, fmt.Errorf("check retirement mirror: %w", mirrorErr)
		}
		// Record the retirement in the manifest before removing its source
		// directory. The manifest is the durable catalog declaration and must
		// retain the version even after its implementation is retired.
		if !assetRetirement {
			if err := r.updateManifest(manifestPath, version); err != nil {
				return c, err
			}
		}
		path := filepath.Clean(filepath.Join(r.sourceRoot, sourcePath))
		root := filepath.Clean(r.sourceRoot) + string(os.PathSeparator)
		if !strings.HasPrefix(path, root) {
			return c, fmt.Errorf("refusing to retire path outside library root")
		}
		versionDir := filepath.Dir(path)
		if _, err := os.Stat(versionDir); err == nil {
			backup := r.retiredSourcePath(c.LibraryID, version)
			if err := os.MkdirAll(filepath.Dir(backup), 0o755); err != nil {
				return c, err
			}
			_ = os.RemoveAll(backup)
			if err := os.Rename(versionDir, backup); err != nil {
				return c, err
			}
		} else if !errors.Is(err, os.ErrNotExist) || !strings.EqualFold(presence, "evicted") && !strings.EqualFold(c.Status, "deprecated") && !strings.EqualFold(c.Status, "archived") {
			return c, err
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		if _, err := r.db.ExecContext(ctx, `UPDATE version_ledger SET retired_at=?, lifecycle_state='retired' WHERE library_id=? AND version=?`, now, c.LibraryID, version); err != nil {
			return c, err
		}
		if _, err := r.db.ExecContext(ctx, `UPDATE component_versions SET status='retired' WHERE component_id=? AND version=?`, c.ComponentID, version); err != nil {
			return c, err
		}
		if !assetRetirement {
			if err := r.updateManifestArray(manifestPath, "evictedVersions", version); err != nil {
				return c, err
			}
			if _, err := r.db.ExecContext(ctx, `UPDATE component_versions SET presence='evicted' WHERE component_id=? AND version=?`, c.ComponentID, version); err != nil {
				return c, err
			}
		}
		if assetRetirement {
			manifest := filepath.Clean(filepath.Join(r.sourceRoot, manifestPath))
			if !strings.HasPrefix(manifest, root) {
				return c, fmt.Errorf("refusing to retire manifest outside library root")
			}
			if err := os.Remove(manifest); err != nil {
				return c, fmt.Errorf("remove retired asset manifest: %w", err)
			}
			_ = os.Remove(filepath.Dir(versionDir))
			_ = os.Remove(filepath.Dir(manifest))
		}
		if err := r.removeReleaseLedgerEntries(sourcePath); err != nil {
			return c, fmt.Errorf("remove retired version ledger entries: %w", err)
		}
		return c, nil
	}
	if state == "deprecated" || state == "archived" {
		if state == "archived" && strings.EqualFold(c.Status, "retired") {
			if err := r.restoreRetiredSource(sourcePath, c.LibraryID, version); err != nil {
				return c, err
			}
		} else if state == "archived" {
			planHash := ""
			if len(planHashes) > 0 {
				planHash = planHashes[0]
			}
			if !confirm {
				return c, fmt.Errorf("archiving a version requires --confirm")
			}
			if err := r.evictVersion(ctx, c, version, manifestPath, planHash, componentID); err != nil {
				return c, err
			}
			return c, nil
		}
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

// PurgeUnreadableVersion permanently removes a named evicted version whose
// mirror is empty. This is a repair operation, not a retention outcome: the
// caller must name the version explicitly and the reachability graph must
// contain no incoming edge before the transaction is allowed to proceed.
func (r *Repository) PurgeUnreadableVersion(ctx context.Context, libraryID, version string, confirm bool) error {
	if !confirm {
		return fmt.Errorf("purging an unreadable version requires explicit confirmation")
	}
	graph, err := r.BuildReachability(ctx)
	if err != nil {
		return err
	}
	key := sourceReferenceKey(libraryID, version)
	if refs := graph.References[key]; len(refs) > 0 {
		return fmt.Errorf("refusing to purge %s@%s: %d incoming references remain", libraryID, version, len(refs))
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var versionID, presence, sourcePath string
	err = tx.QueryRowContext(ctx, `SELECT id, COALESCE(presence, 'materialized'), source_path FROM component_versions WHERE library_id=? AND version=?`, libraryID, version).Scan(&versionID, &presence, &sourcePath)
	if err != nil {
		return err
	}
	if !strings.EqualFold(presence, "evicted") {
		return fmt.Errorf("refusing to purge %s@%s: presence is %s, not evicted", libraryID, version, presence)
	}
	var mirrorRows int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_version_files WHERE version_id=?`, versionID).Scan(&mirrorRows); err != nil {
		return err
	}
	if mirrorRows != 0 {
		return fmt.Errorf("refusing to purge %s@%s: mirror has %d rows", libraryID, version, mirrorRows)
	}
	for _, statement := range []string{
		`DELETE FROM component_version_kit_compatibility WHERE version_id=?`,
		`DELETE FROM component_version_required_token_patterns WHERE version_id=?`,
		`DELETE FROM component_version_required_tokens WHERE version_id=?`,
		`DELETE FROM component_version_parity_reports WHERE version_id=?`,
		`DELETE FROM component_stories WHERE library_id=? AND version=?`,
		`DELETE FROM component_version_test_rollup WHERE library_id=? AND version=?`,
		`DELETE FROM component_version_files WHERE version_id=?`,
		`DELETE FROM version_ledger WHERE library_id=? AND version=?`,
		`DELETE FROM component_versions WHERE id=?`,
	} {
		args := []any{versionID}
		if strings.Contains(statement, "library_id=?") {
			args = []any{libraryID, version}
		}
		if _, err := tx.ExecContext(ctx, statement, args...); err != nil {
			return fmt.Errorf("purge %s@%s: %w", libraryID, version, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if err := r.removeReleaseLedgerEntries(sourcePath); err != nil {
		return fmt.Errorf("remove purged version ledger entries: %w", err)
	}
	return nil
}

// removeReleaseLedgerEntries keeps the append-only hash ledger aligned with
// the governed removal of a version. Derived locks are deliberately included
// in this cleanup even though they are not immutable; authored entries for a
// removed version must not become orphaned integrity failures.
func (r *Repository) removeReleaseLedgerEntries(sourcePath string) error {
	path := filepath.Join(r.sourceRoot, "released-version-hashes.json")
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var ledger struct {
		SchemaVersion int `json:"schemaVersion"`
		Entries       []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(raw, &ledger); err != nil {
		return err
	}
	prefix := filepath.ToSlash(filepath.Dir(sourcePath)) + "/"
	filtered := ledger.Entries[:0]
	for _, entry := range ledger.Entries {
		if !strings.HasPrefix(filepath.ToSlash(entry.Path), prefix) {
			filtered = append(filtered, entry)
		}
	}
	ledger.Entries = filtered
	out, err := json.MarshalIndent(ledger, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(path, out, 0o600)
}

func (r *Repository) reclaimRetiredMaterialization(ctx context.Context, c Candidate, sourcePath, manifestPath string) error {
	versionDir := filepath.Clean(filepath.Join(r.sourceRoot, filepath.Dir(sourcePath)))
	root := filepath.Clean(r.sourceRoot) + string(os.PathSeparator)
	if !strings.HasPrefix(versionDir, root) {
		return fmt.Errorf("refusing to retire path outside library root")
	}
	if _, err := os.Stat(versionDir); err == nil {
		if err := r.verifyVersionDirectory(ctx, c, versionDir); err != nil {
			return err
		}
		backup := r.retiredSourcePath(c.LibraryID, c.Version)
		if _, backupErr := os.Stat(backup); errors.Is(backupErr, os.ErrNotExist) {
			if err := os.MkdirAll(filepath.Dir(backup), 0o755); err != nil {
				return err
			}
			if err := os.Rename(versionDir, backup); err != nil {
				return err
			}
		} else if backupErr != nil {
			return backupErr
		} else if err := os.RemoveAll(versionDir); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := r.updateManifestArray(manifestPath, "evictedVersions", c.Version); err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE component_versions SET status='retired', presence='evicted' WHERE component_id=? AND version=?`, c.ComponentID, c.Version); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `UPDATE version_ledger SET lifecycle_state='retired' WHERE library_id=? AND version=?`, c.LibraryID, c.Version)
	if err != nil {
		return err
	}
	return r.removeReleaseLedgerEntries(sourcePath)
}

func (r *Repository) verifyVersionDirectory(ctx context.Context, c Candidate, versionDir string) error {
	rows, err := r.db.QueryContext(ctx, `SELECT path, content_sha256 FROM component_version_files WHERE version_id=(SELECT id FROM component_versions WHERE component_id=? AND version=?) ORDER BY path`, c.ComponentID, c.Version)
	if err != nil {
		return fmt.Errorf("read retirement mirror: %w", err)
	}
	defer rows.Close()
	expected := map[string]string{}
	for rows.Next() {
		var path, digest string
		if err := rows.Scan(&path, &digest); err != nil {
			return err
		}
		expected[filepath.Clean(filepath.FromSlash(path))] = digest
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(expected) == 0 {
		return fmt.Errorf("cannot retire %s@%s: file mirror is empty", c.LibraryID, c.Version)
	}
	seen := map[string]struct{}{}
	if err := librarywalk.WalkContext(context.Background(), versionDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(versionDir, path)
		if err != nil {
			return err
		}
		digest, ok := expected[filepath.Clean(rel)]
		if !ok {
			return ErrEvictionMirrorMismatch{LibraryID: c.LibraryID, Version: c.Version, Path: filepath.ToSlash(rel), Expected: "absent", Actual: "unexpected"}
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		actual := fmt.Sprintf("%x", sha256.Sum256(body))
		if actual != digest {
			return ErrEvictionMirrorMismatch{LibraryID: c.LibraryID, Version: c.Version, Path: filepath.ToSlash(rel), Expected: digest, Actual: actual}
		}
		seen[filepath.Clean(rel)] = struct{}{}
		return nil
	}); err != nil {
		return err
	}
	for path, digest := range expected {
		if _, ok := seen[path]; !ok {
			return ErrEvictionMirrorMismatch{LibraryID: c.LibraryID, Version: c.Version, Path: filepath.ToSlash(path), Expected: digest, Actual: "missing"}
		}
	}
	return nil
}

func (r *Repository) validateAssetRetirement(ctx context.Context, c Candidate, draft, manifestPath string) error {
	if strings.TrimSpace(draft) != "" {
		return fmt.Errorf("asset %s has active draft %s", c.LibraryID, draft)
	}
	manifest := filepath.Clean(filepath.Join(r.sourceRoot, manifestPath))
	root := filepath.Clean(r.sourceRoot) + string(os.PathSeparator)
	if !strings.HasPrefix(manifest, root) {
		return fmt.Errorf("refusing to inspect manifest outside library root")
	}
	data, err := os.ReadFile(manifest)
	if err != nil {
		return err
	}
	var doc struct {
		ReplacedBy []string `json:"replacedBy"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}
	if len(doc.ReplacedBy) == 0 {
		return fmt.Errorf("asset %s cannot retire its latest version without replacedBy metadata", c.LibraryID)
	}
	var remaining int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_versions WHERE component_id=? AND version<>? AND lower(status)<>'retired'`, c.ComponentID, c.Version).Scan(&remaining); err != nil {
		return err
	}
	if remaining != 0 {
		return fmt.Errorf("asset %s still has %d non-retired historical version(s)", c.LibraryID, remaining)
	}
	queries := []string{
		`SELECT COUNT(*) FROM adoption_records WHERE component_id=? AND adopted_version=?`,
		`SELECT COUNT(*) FROM adoption_files WHERE source_library_id=? AND source_version=?`,
		`SELECT COUNT(*) FROM component_asset_dependencies WHERE library_id=? AND version=?`,
	}
	for index, query := range queries {
		identity := c.ComponentID
		if index > 0 {
			identity = c.LibraryID
		}
		var count int
		if err := r.db.QueryRowContext(ctx, query, identity, c.Version).Scan(&count); err != nil {
			return err
		}
		if count != 0 {
			return fmt.Errorf("version %s is still referenced and is not safe to retire", c.Version)
		}
	}
	graph, err := r.BuildReachability(ctx)
	if err != nil {
		return fmt.Errorf("build source reference graph: %w", err)
	}
	if len(graph.References[sourceReferenceKey(c.LibraryID, c.Version)]) != 0 || hasUnreadableVersion(graph.Unreadable, c.LibraryID, c.Version) {
		return fmt.Errorf("version %s is still referenced and is not safe to retire", c.Version)
	}
	return nil
}

func (r *Repository) evictVersion(ctx context.Context, c Candidate, version, manifestPath, suppliedPlanHash, scopeComponentID string) error {
	// Preserve the caller's scope spelling (UUID or stable library id) while
	// recomputing the confirmation hash. PlanCleanup includes the scope in its
	// hash, so silently replacing a library id with the resolved UUID would
	// reject a valid operator plan.
	items, currentPlanHash, err := r.PlanCleanup(ctx, CleanupScope{ComponentID: scopeComponentID})
	if err != nil {
		return err
	}
	// The RPC supplies its confirmation through the legacy Transition API. A
	// direct archive call is intentionally rejected unless the caller uses the
	// plan-aware helper below; this keeps old callers from accidentally gaining
	// destructive behavior.
	if suppliedPlanHash == "" || suppliedPlanHash != currentPlanHash {
		return fmt.Errorf("archive plan changed or --plan-hash is missing; regenerate with versions reap")
	}
	allowed := false
	for _, item := range items {
		if item.Candidate.Version == version && item.Eligible {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("version %s is still referenced and is not safe to archive", version)
	}
	var sourcePath string
	if err := r.db.QueryRowContext(ctx, `SELECT source_path FROM component_versions WHERE component_id=? AND version=?`, c.ComponentID, version).Scan(&sourcePath); err != nil {
		return err
	}
	versionDir := filepath.Clean(filepath.Join(r.sourceRoot, filepath.Dir(sourcePath)))
	root := filepath.Clean(r.sourceRoot) + string(os.PathSeparator)
	if !strings.HasPrefix(versionDir, root) {
		return fmt.Errorf("refusing to archive path outside library root")
	}
	rows, err := r.db.QueryContext(ctx, `SELECT path, content_sha256 FROM component_version_files WHERE version_id=(SELECT id FROM component_versions WHERE component_id=? AND version=?) ORDER BY path`, c.ComponentID, version)
	if err != nil {
		return fmt.Errorf("read archive mirror: %w", err)
	}
	defer rows.Close()
	checked := 0
	for rows.Next() {
		var path, expected string
		if err := rows.Scan(&path, &expected); err != nil {
			return err
		}
		actualBytes, readErr := os.ReadFile(filepath.Join(versionDir, filepath.FromSlash(path)))
		if readErr != nil {
			return ErrEvictionMirrorMismatch{LibraryID: c.LibraryID, Version: version, Path: path, Expected: expected, Actual: "missing"}
		}
		actual := fmt.Sprintf("%x", sha256.Sum256(actualBytes))
		if actual != expected {
			return ErrEvictionMirrorMismatch{LibraryID: c.LibraryID, Version: version, Path: path, Expected: expected, Actual: actual}
		}
		checked++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if checked == 0 {
		return fmt.Errorf("cannot archive %s@%s: file mirror is empty", c.LibraryID, version)
	}
	if err := os.RemoveAll(versionDir); err != nil {
		return fmt.Errorf("remove archived version directory: %w", err)
	}
	if err := r.updateManifestArray(manifestPath, "evictedVersions", version); err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE component_versions SET presence='evicted' WHERE component_id=? AND version=?`, c.ComponentID, version); err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `UPDATE version_ledger SET lifecycle_state='archived' WHERE library_id=? AND version=?`, c.LibraryID, version)
	if err != nil {
		return err
	}
	return r.removeReleaseLedgerEntries(sourcePath)
}

func (r *Repository) retiredSourcePath(libraryID, version string) string {
	key := sha256.Sum256([]byte(libraryID + "@" + version))
	return filepath.Join(r.sourceRoot, ".retired", hex.EncodeToString(key[:]))
}

func (r *Repository) restoreRetiredSource(sourcePath, libraryID, version string) error {
	backup := r.retiredSourcePath(libraryID, version)
	if _, err := os.Stat(backup); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	destination := filepath.Clean(filepath.Join(r.sourceRoot, sourcePath))
	root := filepath.Clean(r.sourceRoot) + string(os.PathSeparator)
	if !strings.HasPrefix(destination, root) {
		return fmt.Errorf("refusing to restore path outside library root")
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	_ = os.RemoveAll(filepath.Dir(destination))
	return os.Rename(backup, filepath.Dir(destination))
}

func (r *Repository) updateManifest(manifestPath, version string) error {
	return r.updateManifestArray(manifestPath, "deprecatedVersions", version)
}

func (r *Repository) updateManifestArray(manifestPath, field, version string) error {
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
	if raw, ok := doc[field].([]any); ok {
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
	doc[field] = values
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
