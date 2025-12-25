// Package persistence provides database operations for the Agent Inbox scenario.
// This file contains label operations.
package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"agent-inbox/domain"
)

// Label Operations

// ListLabels returns all labels.
func (r *Repository) ListLabels(ctx context.Context) ([]domain.Label, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, name, color, created_at FROM labels ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("failed to list labels: %w", err)
	}
	defer rows.Close()

	var labels []domain.Label
	for rows.Next() {
		var l domain.Label
		if err := rows.Scan(&l.ID, &l.Name, &l.Color, &l.CreatedAt); err != nil {
			continue
		}
		labels = append(labels, l)
	}

	return labels, nil
}

// CreateLabel creates a new label.
func (r *Repository) CreateLabel(ctx context.Context, name, color string) (*domain.Label, error) {
	var label domain.Label
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO labels (name, color)
		VALUES ($1, $2)
		RETURNING id, name, color, created_at
	`, name, color).Scan(&label.ID, &label.Name, &label.Color, &label.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			return nil, fmt.Errorf("label with this name already exists")
		}
		return nil, fmt.Errorf("failed to create label: %w", err)
	}
	return &label, nil
}

// UpdateLabel updates a label's name and/or color.
func (r *Repository) UpdateLabel(ctx context.Context, labelID string, name, color *string) (*domain.Label, error) {
	updates := []string{}
	args := []interface{}{}
	argNum := 1

	if name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argNum))
		args = append(args, *name)
		argNum++
	}
	if color != nil {
		updates = append(updates, fmt.Sprintf("color = $%d", argNum))
		args = append(args, *color)
		argNum++
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	args = append(args, labelID)
	query := fmt.Sprintf("UPDATE labels SET %s WHERE id = $%d RETURNING id, name, color, created_at",
		strings.Join(updates, ", "), argNum)

	var label domain.Label
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&label.ID, &label.Name, &label.Color, &label.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update label: %w", err)
	}
	return &label, nil
}

// DeleteLabel removes a label by ID.
func (r *Repository) DeleteLabel(ctx context.Context, labelID string) (bool, error) {
	result, err := r.db.ExecContext(ctx, "DELETE FROM labels WHERE id = $1", labelID)
	if err != nil {
		return false, fmt.Errorf("failed to delete label: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0, nil
}

// LabelExists checks if a label with the given ID exists.
func (r *Repository) LabelExists(ctx context.Context, labelID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM labels WHERE id = $1)", labelID).Scan(&exists)
	return exists, err
}

// AssignLabel assigns a label to a chat.
func (r *Repository) AssignLabel(ctx context.Context, chatID, labelID string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO chat_labels (chat_id, label_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, chatID, labelID)
	return err
}

// RemoveLabel removes a label from a chat.
func (r *Repository) RemoveLabel(ctx context.Context, chatID, labelID string) (bool, error) {
	result, err := r.db.ExecContext(ctx, "DELETE FROM chat_labels WHERE chat_id = $1 AND label_id = $2", chatID, labelID)
	if err != nil {
		return false, fmt.Errorf("failed to remove label: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0, nil
}

// getChatLabelIDs retrieves label IDs for a chat.
func (r *Repository) getChatLabelIDs(ctx context.Context, chatID string) []string {
	rows, err := r.db.QueryContext(ctx, "SELECT label_id FROM chat_labels WHERE chat_id = $1", chatID)
	if err != nil {
		return []string{}
	}
	defer rows.Close()

	var labelIDs []string
	for rows.Next() {
		var labelID string
		if rows.Scan(&labelID) == nil {
			labelIDs = append(labelIDs, labelID)
		}
	}
	if labelIDs == nil {
		return []string{}
	}
	return labelIDs
}
