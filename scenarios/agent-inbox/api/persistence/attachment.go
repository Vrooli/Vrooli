// Package persistence provides database operations for the Agent Inbox scenario.
// This file contains attachment operations.
package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"agent-inbox/domain"
)

// Attachment Operations

// CreateAttachment saves a new attachment to the database.
// The attachment may or may not be associated with a message yet.
func (r *Repository) CreateAttachment(ctx context.Context, att *domain.Attachment) error {
	var messageID interface{}
	if att.MessageID != "" {
		messageID = att.MessageID
	} else {
		messageID = nil
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO attachments (id, message_id, file_name, content_type, file_size, storage_path, width, height, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, att.ID, messageID, att.FileName, att.ContentType, att.FileSize, att.StoragePath, att.Width, att.Height, att.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create attachment: %w", err)
	}
	return nil
}

// GetAttachmentByID retrieves an attachment by its ID.
func (r *Repository) GetAttachmentByID(ctx context.Context, id string) (*domain.Attachment, error) {
	var att domain.Attachment
	var messageID sql.NullString
	var width, height sql.NullInt32

	err := r.db.QueryRowContext(ctx, `
		SELECT id, message_id, file_name, content_type, file_size, storage_path, width, height, created_at
		FROM attachments WHERE id = $1
	`, id).Scan(&att.ID, &messageID, &att.FileName, &att.ContentType, &att.FileSize, &att.StoragePath, &width, &height, &att.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get attachment: %w", err)
	}

	if messageID.Valid {
		att.MessageID = messageID.String
	}
	if width.Valid {
		att.Width = int(width.Int32)
	}
	if height.Valid {
		att.Height = int(height.Int32)
	}

	return &att, nil
}

// AttachToMessage associates an attachment with a message.
// This is called after uploading a file and sending the message.
func (r *Repository) AttachToMessage(ctx context.Context, attachmentID, messageID string) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE attachments SET message_id = $1 WHERE id = $2
	`, messageID, attachmentID)
	if err != nil {
		return fmt.Errorf("failed to attach to message: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("attachment not found: %s", attachmentID)
	}

	return nil
}

// LinkAttachmentsToMessage associates multiple attachments with a message.
// Errors linking individual attachments are collected and returned as a combined error.
func (r *Repository) LinkAttachmentsToMessage(ctx context.Context, messageID string, attachmentIDs []string) error {
	var errors []string
	for _, attID := range attachmentIDs {
		if err := r.AttachToMessage(ctx, attID, messageID); err != nil {
			errors = append(errors, err.Error())
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("failed to link %d attachments: %s", len(errors), joinStrings(errors, "; "))
	}
	return nil
}

// GetAttachmentsByMessageID retrieves all attachments for a message.
func (r *Repository) GetAttachmentsByMessageID(ctx context.Context, messageID string) ([]domain.Attachment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, message_id, file_name, content_type, file_size, storage_path, width, height, created_at
		FROM attachments WHERE message_id = $1 ORDER BY created_at
	`, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get attachments: %w", err)
	}
	defer rows.Close()

	attachments := make([]domain.Attachment, 0)
	for rows.Next() {
		var att domain.Attachment
		var msgID sql.NullString
		var width, height sql.NullInt32

		if err := rows.Scan(&att.ID, &msgID, &att.FileName, &att.ContentType, &att.FileSize, &att.StoragePath, &width, &height, &att.CreatedAt); err != nil {
			continue
		}
		if msgID.Valid {
			att.MessageID = msgID.String
		}
		if width.Valid {
			att.Width = int(width.Int32)
		}
		if height.Valid {
			att.Height = int(height.Int32)
		}
		attachments = append(attachments, att)
	}

	return attachments, nil
}

// GetAttachmentsForMessages retrieves attachments for multiple messages in a single query.
// Returns a map from message ID to attachments.
func (r *Repository) GetAttachmentsForMessages(ctx context.Context, messageIDs []string) (map[string][]domain.Attachment, error) {
	if len(messageIDs) == 0 {
		return make(map[string][]domain.Attachment), nil
	}

	// Build placeholder string
	placeholders := make([]string, len(messageIDs))
	args := make([]interface{}, len(messageIDs))
	for i, id := range messageIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, message_id, file_name, content_type, file_size, storage_path, width, height, created_at
		FROM attachments
		WHERE message_id IN (%s)
		ORDER BY message_id, created_at
	`, joinStrings(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get attachments for messages: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]domain.Attachment)
	for rows.Next() {
		var att domain.Attachment
		var msgID sql.NullString
		var width, height sql.NullInt32

		if err := rows.Scan(&att.ID, &msgID, &att.FileName, &att.ContentType, &att.FileSize, &att.StoragePath, &width, &height, &att.CreatedAt); err != nil {
			continue
		}
		if msgID.Valid {
			att.MessageID = msgID.String
			if width.Valid {
				att.Width = int(width.Int32)
			}
			if height.Valid {
				att.Height = int(height.Int32)
			}
			result[att.MessageID] = append(result[att.MessageID], att)
		}
	}

	return result, nil
}

// DeleteAttachment removes an attachment from the database.
// Note: This does NOT delete the file from storage - caller must do that.
func (r *Repository) DeleteAttachment(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM attachments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}
	return nil
}

// GetOrphanedAttachments finds attachments not associated with any message.
// These may be uploads that were never used (user abandoned the message).
// Useful for cleanup jobs.
func (r *Repository) GetOrphanedAttachments(ctx context.Context, olderThan string) ([]domain.Attachment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, message_id, file_name, content_type, file_size, storage_path, width, height, created_at
		FROM attachments
		WHERE message_id IS NULL AND created_at < NOW() - $1::interval
		ORDER BY created_at
	`, olderThan)
	if err != nil {
		return nil, fmt.Errorf("failed to get orphaned attachments: %w", err)
	}
	defer rows.Close()

	attachments := make([]domain.Attachment, 0)
	for rows.Next() {
		var att domain.Attachment
		var msgID sql.NullString
		var width, height sql.NullInt32

		if err := rows.Scan(&att.ID, &msgID, &att.FileName, &att.ContentType, &att.FileSize, &att.StoragePath, &width, &height, &att.CreatedAt); err != nil {
			continue
		}
		if width.Valid {
			att.Width = int(width.Int32)
		}
		if height.Valid {
			att.Height = int(height.Int32)
		}
		attachments = append(attachments, att)
	}

	return attachments, nil
}

// Helper function to join strings
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
