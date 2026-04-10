// DOC: docs/internal/STORAGE_AUDIT.md
// DOC: PRD.md#OT-P0-006
//
// Package repository provides SQLite implementation of StorageRepository.
// [REQ:LD-UI-STORAGE] Storage management operations for settings page.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"lifestyle-dashboard/domain"
)

// SQLiteStorageRepository implements StorageRepository using SQLite.
type SQLiteStorageRepository struct {
	db *sql.DB
}

// NewSQLiteStorageRepository creates a new SQLite storage repository.
func NewSQLiteStorageRepository(db *sql.DB) *SQLiteStorageRepository {
	return &SQLiteStorageRepository{db: db}
}

// GetStorageInfo returns database size and event counts by domain.
// [REQ:LD-UI-STORAGE] Storage overview data for settings page.
func (r *SQLiteStorageRepository) GetStorageInfo(ctx context.Context) (*domain.StorageInfo, error) {
	info := &domain.StorageInfo{
		EventsByDomain: make([]domain.DomainStorageInfo, 0),
	}

	// Get database file size
	dbPath := os.Getenv("SQLITE_PATH")
	if dbPath != "" {
		// Strip query params if present
		if idx := strings.Index(dbPath, "?"); idx > 0 {
			dbPath = dbPath[:idx]
		}
		if fi, err := os.Stat(dbPath); err == nil {
			info.DatabaseSizeBytes = fi.Size()
		}
	}

	// Get total events
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM events`).Scan(&info.TotalEvents)
	if err != nil {
		return nil, fmt.Errorf("count events: %w", err)
	}

	// Get total domains
	err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM domains`).Scan(&info.TotalDomains)
	if err != nil {
		return nil, fmt.Errorf("count domains: %w", err)
	}

	// Get oldest and newest events
	var oldest, newest sql.NullString
	err = r.db.QueryRowContext(ctx, `SELECT MIN(timestamp), MAX(timestamp) FROM events`).Scan(&oldest, &newest)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("event timestamps: %w", err)
	}
	if oldest.Valid {
		info.OldestEvent = oldest.String
	}
	if newest.Valid {
		info.NewestEvent = newest.String
	}

	// Get event counts by domain with display names
	rows, err := r.db.QueryContext(ctx, `
		SELECT e.domain, COALESCE(d.display_name, e.domain), COUNT(*) as count
		FROM events e
		LEFT JOIN domains d ON e.domain = d.name
		GROUP BY e.domain
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("events by domain: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ds domain.DomainStorageInfo
		if err := rows.Scan(&ds.Domain, &ds.DisplayName, &ds.EventCount); err != nil {
			return nil, fmt.Errorf("scan domain storage: %w", err)
		}
		info.EventsByDomain = append(info.EventsByDomain, ds)
	}

	return info, rows.Err()
}

// CleanupEvents deletes events matching the cleanup request.
// [REQ:LD-UI-STORAGE] Data cleanup functionality.
func (r *SQLiteStorageRepository) CleanupEvents(ctx context.Context, req domain.CleanupRequest) (*domain.CleanupResponse, error) {
	var query string
	var args []interface{}

	// Build the DELETE query based on request
	if len(req.Domains) == 0 && req.Before == "" {
		// Delete all events
		query = `DELETE FROM events`
	} else {
		// Build WHERE clause
		conditions := make([]string, 0, 2)

		if len(req.Domains) > 0 {
			placeholders := make([]string, len(req.Domains))
			for i, d := range req.Domains {
				placeholders[i] = "?"
				args = append(args, d)
			}
			conditions = append(conditions, fmt.Sprintf("domain IN (%s)", strings.Join(placeholders, ",")))
		}

		if req.Before != "" {
			conditions = append(conditions, "timestamp < ?")
			args = append(args, req.Before)
		}

		query = "DELETE FROM events WHERE " + strings.Join(conditions, " AND ")
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("delete events: %w", err)
	}

	deletedCount, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("get affected rows: %w", err)
	}

	// Build response
	resp := &domain.CleanupResponse{
		DeletedEvents:  int(deletedCount),
		DomainsCleared: req.Domains,
	}

	if len(req.Domains) == 0 {
		resp.Message = fmt.Sprintf("Cleared all events (%d deleted)", deletedCount)
		resp.DomainsCleared = []string{"all"}
	} else if req.Before != "" {
		resp.Message = fmt.Sprintf("Cleared %d events from %s before %s",
			deletedCount, strings.Join(req.Domains, ", "), req.Before)
	} else {
		resp.Message = fmt.Sprintf("Cleared %d events from %s",
			deletedCount, strings.Join(req.Domains, ", "))
	}

	return resp, nil
}
