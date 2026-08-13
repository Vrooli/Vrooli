package components

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// BackfillCreatedAt reads the first commit date for every indexed version
// folder and persists it as the version's immutable first-seen time. It is a
// one-shot maintenance operation: rows without a matching git path retain
// their existing value rather than being fabricated from the current clock.
func BackfillCreatedAt(ctx context.Context, db *sql.DB, repoRoot string) (int, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, source_path, created_at FROM component_versions`)
	if err != nil {
		return 0, fmt.Errorf("list versions for created_at backfill: %w", err)
	}
	defer rows.Close()
	type row struct {
		id, sourcePath, createdAt string
	}
	var versions []row
	for rows.Next() {
		var version row
		if err := rows.Scan(&version.id, &version.sourcePath, &version.createdAt); err != nil {
			return 0, fmt.Errorf("scan version for created_at backfill: %w", err)
		}
		versions = append(versions, version)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate versions for created_at backfill: %w", err)
	}

	updated := 0
	for _, version := range versions {
		folder, ok := versionFolder(version.sourcePath)
		if !ok {
			continue
		}
		// component_versions.source_path is relative to the scenario's
		// library/ root; git history is rooted at the repository.
		gitPath := filepath.ToSlash(filepath.Join("scenarios/react-component-library/library", folder))
		date, ok, err := firstGitCommitDate(ctx, repoRoot, gitPath)
		if err != nil {
			return updated, err
		}
		if !ok {
			continue
		}
		if _, err := db.ExecContext(ctx, `UPDATE component_versions SET created_at = ? WHERE id = ?`, date.UTC().Format(timeFormat), version.id); err != nil {
			return updated, fmt.Errorf("update created_at for version %q: %w", version.id, err)
		}
		updated++
	}
	return updated, nil
}

func versionFolder(sourcePath string) (string, bool) {
	parts := strings.Split(filepath.ToSlash(sourcePath), "/")
	for i := 0; i+2 < len(parts); i++ {
		if parts[i+2] == "versions" && i+3 < len(parts) {
			return strings.Join(parts[:i+4], "/"), true
		}
	}
	for i, part := range parts {
		if part == "versions" && i+1 < len(parts) {
			return strings.Join(parts[:i+2], "/"), true
		}
	}
	return "", false
}

func firstGitCommitDate(ctx context.Context, repoRoot, path string) (time.Time, bool, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoRoot, "log", "--follow", "--reverse", "--format=%cI", "--", path)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) == 0 {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, fmt.Errorf("read first git commit for %s: %w", path, err)
	}
	first := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	if first == "" {
		return time.Time{}, false, nil
	}
	date, err := time.Parse(time.RFC3339, first)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("parse first git commit date %q for %s: %w", first, path, err)
	}
	return date, true, nil
}
