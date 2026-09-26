package categories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"signal-inbox/internal/signals"

	"github.com/google/uuid"
)

const timestampFormat = time.RFC3339Nano

type sqliteRepository struct{ db signals.SQLExecutor }

func NewSQLiteRepository(db signals.SQLExecutor) Repository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) EnsureUncategorized(ctx context.Context, now time.Time) (Category, error) {
	category, err := r.categoryByName(ctx, UncategorizedName)
	if err == nil {
		return category, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Category{}, err
	}
	return r.Create(ctx, Category{ID: uuid.NewString(), Name: UncategorizedName, Reserved: true, CreatedAt: now.UTC()})
}

func (r *sqliteRepository) Create(ctx context.Context, category Category) (Category, error) {
	category.Name = strings.TrimSpace(category.Name)
	category.Description = strings.TrimSpace(category.Description)
	if category.Name == "" {
		return Category{}, ErrInvalidCategory{Reason: "name is required"}
	}
	if category.ID == "" {
		category.ID = uuid.NewString()
	}
	if category.CreatedAt.IsZero() {
		category.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO category (id, name, description, reserved, created_at) VALUES (?, ?, ?, ?, ?)`, category.ID, category.Name, category.Description, category.Reserved, category.CreatedAt.UTC().Format(timestampFormat))
	if err != nil {
		return Category{}, fmt.Errorf("create category: %w", err)
	}
	return category, nil
}

func (r *sqliteRepository) List(ctx context.Context, includeRetired bool) ([]Category, error) {
	query := categorySelect
	if !includeRetired {
		query += ` WHERE retired_at = ''`
	}
	query += ` ORDER BY name ASC, id ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()
	var categories []Category
	for rows.Next() {
		category, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, rows.Err()
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (Category, error) {
	category, err := scanCategory(r.db.QueryRowContext(ctx, categorySelect+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Category{}, ErrCategoryNotFound{ID: id}
	}
	if err != nil {
		return Category{}, fmt.Errorf("get category: %w", err)
	}
	return category, nil
}

func (r *sqliteRepository) Rename(ctx context.Context, id, name, description string) (Category, error) {
	category, err := r.Get(ctx, id)
	if err != nil {
		return Category{}, err
	}
	if category.Reserved {
		return Category{}, ErrReservedCategory{ID: id}
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return Category{}, ErrInvalidCategory{Reason: "name is required"}
	}
	_, err = r.db.ExecContext(ctx, `UPDATE category SET name = ?, description = ? WHERE id = ?`, name, strings.TrimSpace(description), id)
	if err != nil {
		return Category{}, fmt.Errorf("rename category: %w", err)
	}
	return r.Get(ctx, id)
}

func (r *sqliteRepository) Retire(ctx context.Context, id string, now time.Time) (Category, error) {
	category, err := r.Get(ctx, id)
	if err != nil {
		return Category{}, err
	}
	if category.Reserved {
		return Category{}, ErrReservedCategory{ID: id}
	}
	if !category.Active() {
		return category, nil
	}
	_, err = r.db.ExecContext(ctx, `UPDATE category SET retired_at = ? WHERE id = ?`, now.UTC().Format(timestampFormat), id)
	if err != nil {
		return Category{}, fmt.Errorf("retire category: %w", err)
	}
	return r.Get(ctx, id)
}

func (r *sqliteRepository) AppendClassification(ctx context.Context, classification Classification) (Classification, error) {
	if classification.SignalID == "" || classification.ProposedCategoryID == "" {
		return Classification{}, ErrInvalidCategory{Reason: "signal and proposed category are required"}
	}
	if classification.ProposedConfidence < 0 || classification.ProposedConfidence > 1 {
		return Classification{}, ErrInvalidCategory{Reason: "proposal confidence must be in [0,1]"}
	}
	if classification.ID == "" {
		classification.ID = uuid.NewString()
	}
	if classification.CreatedAt.IsZero() {
		classification.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO classification (id, signal_id, proposed_category_id, proposed_confidence, model, confirmed_category_id, state, reason, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		classification.ID, classification.SignalID, classification.ProposedCategoryID, classification.ProposedConfidence, classification.Model, classification.ConfirmedCategoryID, classification.State, classification.Reason, classification.CreatedAt.UTC().Format(timestampFormat))
	if err != nil {
		return Classification{}, fmt.Errorf("append classification: %w", err)
	}
	return classification, nil
}

func (r *sqliteRepository) LatestClassification(ctx context.Context, signalID string) (Classification, bool, error) {
	classification, err := scanClassification(r.db.QueryRowContext(ctx, classificationSelect+` WHERE signal_id = ? ORDER BY created_at DESC, rowid DESC LIMIT 1`, signalID))
	if errors.Is(err, sql.ErrNoRows) {
		return Classification{}, false, nil
	}
	if err != nil {
		return Classification{}, false, err
	}
	return classification, true, nil
}

func (r *sqliteRepository) LatestConfirmedByCategory(ctx context.Context, categoryID string) ([]Classification, error) {
	rows, err := r.db.QueryContext(ctx, classificationSelect+`
WHERE confirmed_category_id = ? AND state IN ('confirmed', 'overridden')
  AND NOT EXISTS (
    SELECT 1 FROM classification later
    WHERE later.signal_id = classification.signal_id
      AND (later.created_at > classification.created_at OR (later.created_at = classification.created_at AND later.rowid > classification.rowid))
  )`, categoryID)
	if err != nil {
		return nil, fmt.Errorf("list category classifications: %w", err)
	}
	defer rows.Close()
	var classifications []Classification
	for rows.Next() {
		classification, err := scanClassification(rows)
		if err != nil {
			return nil, err
		}
		classifications = append(classifications, classification)
	}
	return classifications, rows.Err()
}

func (r *sqliteRepository) EnqueueReclassification(ctx context.Context, signalID, reason string, now time.Time) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO classification_queue (signal_id, reason, enqueued_at) VALUES (?, ?, ?) ON CONFLICT(signal_id) DO UPDATE SET reason = excluded.reason, enqueued_at = excluded.enqueued_at`, signalID, reason, now.UTC().Format(timestampFormat))
	if err != nil {
		return fmt.Errorf("enqueue reclassification: %w", err)
	}
	return nil
}

const (
	categorySelect       = `SELECT id, name, description, reserved, created_at, retired_at FROM category`
	classificationSelect = `SELECT id, signal_id, proposed_category_id, proposed_confidence, model, confirmed_category_id, state, reason, created_at FROM classification`
)

type scanner interface{ Scan(...any) error }

func scanCategory(row scanner) (Category, error) {
	var category Category
	var createdAt, retiredAt string
	if err := row.Scan(&category.ID, &category.Name, &category.Description, &category.Reserved, &createdAt, &retiredAt); err != nil {
		return Category{}, err
	}
	parsed, err := time.Parse(timestampFormat, createdAt)
	if err != nil {
		return Category{}, fmt.Errorf("parse category created_at: %w", err)
	}
	category.CreatedAt = parsed
	if retiredAt != "" {
		parsed, err := time.Parse(timestampFormat, retiredAt)
		if err != nil {
			return Category{}, fmt.Errorf("parse category retired_at: %w", err)
		}
		category.RetiredAt = &parsed
	}
	return category, nil
}

func (r *sqliteRepository) categoryByName(ctx context.Context, name string) (Category, error) {
	return scanCategory(r.db.QueryRowContext(ctx, categorySelect+` WHERE name = ?`, name))
}

func scanClassification(row scanner) (Classification, error) {
	var classification Classification
	var createdAt string
	if err := row.Scan(&classification.ID, &classification.SignalID, &classification.ProposedCategoryID, &classification.ProposedConfidence, &classification.Model, &classification.ConfirmedCategoryID, &classification.State, &classification.Reason, &createdAt); err != nil {
		return Classification{}, err
	}
	parsed, err := time.Parse(timestampFormat, createdAt)
	if err != nil {
		return Classification{}, fmt.Errorf("parse classification created_at: %w", err)
	}
	classification.CreatedAt = parsed
	return classification, nil
}
