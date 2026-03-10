package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"lifestyle-dashboard/domain"
)

// SQLiteDomainRepository implements DomainRepository for SQLite.
type SQLiteDomainRepository struct {
	db *sql.DB
}

// NewSQLiteDomainRepository creates a new SQLite domain repository.
func NewSQLiteDomainRepository(db *sql.DB) *SQLiteDomainRepository {
	return &SQLiteDomainRepository{db: db}
}

// Upsert creates or updates a domain registration.
// [REQ:LD-DOMAIN-REGISTER] Stores domain with capabilities JSON.
func (r *SQLiteDomainRepository) Upsert(ctx context.Context, d *domain.Domain) error {
	now := time.Now().UTC().Format(time.RFC3339)

	capabilities := "[]"
	if len(d.Capabilities) > 0 {
		capsJSON, _ := json.Marshal(d.Capabilities)
		capabilities = string(capsJSON)
	}

	// Set timestamps if not provided
	if d.RegisteredAt == "" {
		d.RegisteredAt = now
	}
	d.UpdatedAt = now

	// Set default status
	if d.Status == "" {
		d.Status = "active"
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO domains (name, display_name, description, capabilities, status, health_url, registered_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			display_name = excluded.display_name,
			description = excluded.description,
			capabilities = excluded.capabilities,
			health_url = excluded.health_url,
			updated_at = excluded.updated_at
	`, d.Name, d.DisplayName, d.Description, capabilities, d.Status, d.HealthURL, d.RegisteredAt, d.UpdatedAt)

	return err
}

// GetByName retrieves a single domain by name.
// [REQ:LD-DOMAIN-DISCOVER] Fetches domain with capabilities parsing.
func (r *SQLiteDomainRepository) GetByName(ctx context.Context, name string) (*domain.Domain, error) {
	var d domain.Domain
	var capabilities string
	var lastHealthAt sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT name, display_name, description, capabilities, status, health_url, last_health_at, registered_at, updated_at
		FROM domains WHERE name = ?
	`, name).Scan(&d.Name, &d.DisplayName, &d.Description, &capabilities, &d.Status, &d.HealthURL, &lastHealthAt, &d.RegisteredAt, &d.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound{Entity: "domain", ID: name}
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(capabilities), &d.Capabilities)
	if lastHealthAt.Valid {
		d.LastHealthAt = &lastHealthAt.String
	}

	return &d, nil
}

// List retrieves all registered domains.
// [REQ:LD-DOMAIN-DISCOVER] Lists all domains with parsed capabilities.
func (r *SQLiteDomainRepository) List(ctx context.Context) ([]domain.Domain, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT name, display_name, description, capabilities, status, health_url, last_health_at, registered_at, updated_at
		FROM domains ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	domains := []domain.Domain{}
	for rows.Next() {
		var d domain.Domain
		var capabilities string
		var lastHealthAt sql.NullString
		if err := rows.Scan(&d.Name, &d.DisplayName, &d.Description, &capabilities, &d.Status, &d.HealthURL, &lastHealthAt, &d.RegisteredAt, &d.UpdatedAt); err != nil {
			continue
		}
		json.Unmarshal([]byte(capabilities), &d.Capabilities)
		if lastHealthAt.Valid {
			d.LastHealthAt = &lastHealthAt.String
		}
		domains = append(domains, d)
	}

	return domains, rows.Err()
}

// UpdateStatus updates a domain's status and last health check time.
// [REQ:LD-DOMAIN-HEALTH] Records health check results.
func (r *SQLiteDomainRepository) UpdateStatus(ctx context.Context, name, status, lastHealthAt string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		"UPDATE domains SET status = ?, last_health_at = ?, updated_at = ? WHERE name = ?",
		status, lastHealthAt, now, name)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound{Entity: "domain", ID: name}
	}
	return nil
}

// Update applies partial updates to a domain.
// [REQ:LD-DOMAIN-REGISTER] Supports partial domain updates.
func (r *SQLiteDomainRepository) Update(ctx context.Context, name string, updates map[string]interface{}) error {
	now := time.Now().UTC().Format(time.RFC3339)

	setClause := "updated_at = ?"
	args := []interface{}{now}

	if status, ok := updates["status"].(string); ok {
		setClause += ", status = ?"
		args = append(args, status)
	}
	if displayName, ok := updates["display_name"].(string); ok {
		setClause += ", display_name = ?"
		args = append(args, displayName)
	}
	if description, ok := updates["description"].(string); ok {
		setClause += ", description = ?"
		args = append(args, description)
	}

	args = append(args, name)
	result, err := r.db.ExecContext(ctx, fmt.Sprintf("UPDATE domains SET %s WHERE name = ?", setClause), args...)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound{Entity: "domain", ID: name}
	}
	return nil
}
