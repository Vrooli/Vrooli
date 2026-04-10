package repository

import (
	"context"
	"database/sql"
	"time"

	"lifestyle-dashboard/domain"
)

// SQLiteScoreConfigRepository implements ScoreConfigRepository for SQLite.
// [REQ:LD-SCORE-CALC] Stores and retrieves configurable domain weights.
type SQLiteScoreConfigRepository struct {
	db *sql.DB
}

// NewSQLiteScoreConfigRepository creates a new SQLite score config repository.
func NewSQLiteScoreConfigRepository(db *sql.DB) *SQLiteScoreConfigRepository {
	return &SQLiteScoreConfigRepository{db: db}
}

// GetWeights returns weight configurations for all registered domains.
// Domains without explicit weight configurations get the default weight based on presets.
// [REQ:LD-SCORE-CALC] Retrieves all domain weights.
func (r *SQLiteScoreConfigRepository) GetWeights(ctx context.Context) ([]domain.DomainWeightConfig, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			d.name,
			d.display_name,
			COALESCE(w.weight, 'medium') as weight
		FROM domains d
		LEFT JOIN domain_weights w ON d.name = w.domain
		WHERE d.status = 'active'
		ORDER BY d.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []domain.DomainWeightConfig
	for rows.Next() {
		var cfg domain.DomainWeightConfig
		if err := rows.Scan(&cfg.Domain, &cfg.DisplayName, &cfg.Weight); err != nil {
			continue
		}
		// Apply preset if no explicit weight is set
		if cfg.Weight == "medium" {
			if preset, ok := domain.WeightPresets[cfg.Domain]; ok {
				cfg.Weight = preset
			}
		}
		cfg.Multiplier = domain.WeightMultipliers[cfg.Weight]
		configs = append(configs, cfg)
	}

	return configs, rows.Err()
}

// GetWeight returns the weight configuration for a specific domain.
// Returns the preset weight if no explicit configuration exists.
// [REQ:LD-SCORE-CALC] Retrieves single domain weight.
func (r *SQLiteScoreConfigRepository) GetWeight(ctx context.Context, domainName string) (*domain.DomainWeightConfig, error) {
	var cfg domain.DomainWeightConfig
	err := r.db.QueryRowContext(ctx, `
		SELECT
			d.name,
			d.display_name,
			COALESCE(w.weight, 'medium') as weight
		FROM domains d
		LEFT JOIN domain_weights w ON d.name = w.domain
		WHERE d.name = ?
	`, domainName).Scan(&cfg.Domain, &cfg.DisplayName, &cfg.Weight)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound{Entity: "domain", ID: domainName}
	}
	if err != nil {
		return nil, err
	}

	// Apply preset if no explicit weight is set
	if cfg.Weight == "medium" {
		if preset, ok := domain.WeightPresets[cfg.Domain]; ok {
			cfg.Weight = preset
		}
	}
	cfg.Multiplier = domain.WeightMultipliers[cfg.Weight]

	return &cfg, nil
}

// SetWeight updates the weight for a specific domain.
// Creates a new record if none exists (upsert).
// [REQ:LD-SCORE-CALC] Updates domain weight.
func (r *SQLiteScoreConfigRepository) SetWeight(ctx context.Context, domainName, weight string) error {
	// Validate weight value
	if _, ok := domain.WeightMultipliers[weight]; !ok {
		return ErrNotFound{Entity: "weight", ID: weight}
	}

	// Verify domain exists
	var exists int
	err := r.db.QueryRowContext(ctx, "SELECT 1 FROM domains WHERE name = ?", domainName).Scan(&exists)
	if err == sql.ErrNoRows {
		return ErrNotFound{Entity: "domain", ID: domainName}
	}
	if err != nil {
		return err
	}

	// Upsert weight
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO domain_weights (domain, weight, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(domain) DO UPDATE SET
			weight = excluded.weight,
			updated_at = excluded.updated_at
	`, domainName, weight, now)

	return err
}
