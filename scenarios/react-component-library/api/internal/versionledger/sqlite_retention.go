package versionledger

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"react-component-library/internal/sqliterows"
	"strings"
	"time"
)

type Candidate struct{ ComponentID, LibraryID, Version, Status string }

func (r *Repository) PlanCleanup(ctx context.Context, scope CleanupScope) ([]CleanupItem, string, error) {
	if scope.ComponentID != "" && scope.LibraryID != "" {
		return nil, "", fmt.Errorf("cleanup scope cannot contain both component_id and library_id")
	}
	type cleanupRaw struct {
		candidate                        Candidate
		latest, draft, created, released string
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
	rawRows, err := sqliterows.Collect(ctx, r.db, query, args, true, func(row sqliterows.Scanner) (cleanupRaw, error) {
		var raw cleanupRaw
		if err := row.Scan(&raw.candidate.ComponentID, &raw.candidate.LibraryID, &raw.candidate.Version, &raw.candidate.Status, &raw.latest, &raw.draft, &raw.created, &raw.released); err != nil {
			return cleanupRaw{}, err
		}
		return raw, nil
	})
	if err != nil {
		return nil, "", err
	}
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
			switch {
			case item.AdoptionCount > 0:
				item.Reason = "referenced by adoption"
			case scope.OlderThanDays > 0 && item.AgeDays < scope.OlderThanDays:
				item.Reason = fmt.Sprintf("younger than %d days", scope.OlderThanDays)
			default:
				// Every reference that can still break something is a root in
				// BuildReachability. What remains are pins held only by cold,
				// unreachable versions, which leave with them.
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
	root, err := r.openSourceRoot()
	if err != nil {
		return item, err
	}
	defer root.Close()
	path, err := sourceRelativePath(sourcePath)
	if err != nil {
		return item, fmt.Errorf("refusing to discard path outside library root")
	}
	if err := root.RemoveAll(filepath.Dir(path)); err != nil {
		return item, err
	}
	manifestPath := ""
	if err := r.db.QueryRowContext(ctx, `SELECT manifest_path FROM components WHERE id = ?`, item.Candidate.ComponentID).Scan(&manifestPath); err != nil {
		return item, err
	}
	if err := r.clearDraftManifest(manifestPath); err != nil {
		return item, err
	}
	_, err = r.db.ExecContext(ctx, `UPDATE components SET draft_version = '' WHERE id = ?`, item.Candidate.ComponentID)
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
	rows, err := sqliterows.Collect(ctx, r.db, query, args, false, func(row sqliterows.Scanner) (Candidate, error) {
		var candidate Candidate
		if err := row.Scan(&candidate.ComponentID, &candidate.LibraryID, &candidate.Version, &candidate.Status); err != nil {
			return Candidate{}, err
		}
		return candidate, nil
	})
	if err != nil {
		return nil, err
	}
	var out []Candidate
	for _, candidate := range rows {
		if graph.IsReachable(candidate.LibraryID, candidate.Version) || hasUnreadableVersion(graph.Unreadable, candidate.LibraryID, candidate.Version) {
			continue
		}
		out = append(out, candidate)
	}
	return out, nil
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
		root, err := r.openSourceRoot()
		if err != nil {
			return c, err
		}
		defer root.Close()
		path, err := sourceRelativePath(sourcePath)
		if err != nil {
			return c, fmt.Errorf("refusing to retire path outside library root")
		}
		versionDir := filepath.Dir(path)
		if _, err := root.Stat(versionDir); err == nil {
			backup, err := r.retiredSourceRelativePath(c.LibraryID, version)
			if err != nil {
				return c, err
			}
			if err := root.MkdirAll(filepath.Dir(backup), 0o755); err != nil {
				return c, err
			}
			_ = root.RemoveAll(backup)
			if err := root.Rename(versionDir, backup); err != nil {
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
			manifest, err := sourceRelativePath(manifestPath)
			if err != nil {
				return c, fmt.Errorf("refusing to retire manifest outside library root")
			}
			if err := root.Remove(manifest); err != nil {
				return c, fmt.Errorf("remove retired asset manifest: %w", err)
			}
			_ = root.Remove(filepath.Dir(versionDir))
			_ = root.Remove(filepath.Dir(manifest))
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
