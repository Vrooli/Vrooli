// Package persistence provides database operations for the Agent Inbox scenario.
//
// This file provides user settings persistence operations.
// Settings are stored as key-value pairs with JSONB values for flexibility.
package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// YOLO mode setting key
const YoloModeKey = "yolo_mode"

// GetUserSetting retrieves a user setting by key.
// Returns empty string if not found.
func (r *Repository) GetUserSetting(ctx context.Context, key string) (string, error) {
	query := `SELECT value FROM user_settings WHERE key = $1`

	var valueJSON []byte
	err := r.db.QueryRowContext(ctx, query, key).Scan(&valueJSON)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get user setting %s: %w", key, err)
	}

	// Parse JSONB value - it could be a string, bool, number, etc.
	var value interface{}
	if err := json.Unmarshal(valueJSON, &value); err != nil {
		return "", fmt.Errorf("failed to parse setting value: %w", err)
	}

	// Convert to string representation
	switch v := value.(type) {
	case string:
		return v, nil
	case bool:
		if v {
			return "true", nil
		}
		return "false", nil
	case float64:
		return fmt.Sprintf("%v", v), nil
	default:
		return string(valueJSON), nil
	}
}

// SetUserSetting creates or updates a user setting.
func (r *Repository) SetUserSetting(ctx context.Context, key, value string) error {
	now := time.Now()

	// Wrap string value in quotes for JSONB
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal setting value: %w", err)
	}

	query := `
		INSERT INTO user_settings (key, value, created_at, updated_at)
		VALUES ($1, $2, $3, $3)
		ON CONFLICT (key)
		DO UPDATE SET value = $2, updated_at = $3`

	_, err = r.db.ExecContext(ctx, query, key, valueJSON, now)
	if err != nil {
		return fmt.Errorf("failed to set user setting %s: %w", key, err)
	}

	return nil
}

// GetYoloMode returns whether YOLO mode is enabled.
// YOLO mode bypasses all tool approval requirements.
// Returns false if not set.
func (r *Repository) GetYoloMode(ctx context.Context) (bool, error) {
	value, err := r.GetUserSetting(ctx, YoloModeKey)
	if err != nil {
		return false, err
	}

	return value == "true", nil
}

// SetYoloMode enables or disables YOLO mode.
func (r *Repository) SetYoloMode(ctx context.Context, enabled bool) error {
	value := "false"
	if enabled {
		value = "true"
	}
	return r.SetUserSetting(ctx, YoloModeKey, value)
}

// DeleteUserSetting removes a user setting.
func (r *Repository) DeleteUserSetting(ctx context.Context, key string) error {
	query := `DELETE FROM user_settings WHERE key = $1`

	_, err := r.db.ExecContext(ctx, query, key)
	if err != nil {
		return fmt.Errorf("failed to delete user setting %s: %w", key, err)
	}

	return nil
}

// ListUserSettings returns all user settings.
func (r *Repository) ListUserSettings(ctx context.Context) (map[string]string, error) {
	query := `SELECT key, value FROM user_settings ORDER BY key`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list user settings: %w", err)
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key string
		var valueJSON []byte
		if err := rows.Scan(&key, &valueJSON); err != nil {
			return nil, fmt.Errorf("failed to scan user setting: %w", err)
		}

		var value interface{}
		if err := json.Unmarshal(valueJSON, &value); err != nil {
			settings[key] = string(valueJSON)
			continue
		}

		switch v := value.(type) {
		case string:
			settings[key] = v
		case bool:
			if v {
				settings[key] = "true"
			} else {
				settings[key] = "false"
			}
		default:
			settings[key] = string(valueJSON)
		}
	}

	return settings, rows.Err()
}
