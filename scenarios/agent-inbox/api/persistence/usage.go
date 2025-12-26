// Package persistence provides database operations for the Agent Inbox scenario.
// This file contains usage record operations for tracking token usage and costs.
package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"agent-inbox/domain"
)

// SaveUsageRecord persists a usage record to the database.
func (r *Repository) SaveUsageRecord(ctx context.Context, record *domain.UsageRecord) error {
	var messageID interface{} = nil
	if record.MessageID != "" {
		messageID = record.MessageID
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO usage_records (chat_id, message_id, model, prompt_tokens, completion_tokens, total_tokens, prompt_cost, completion_cost, total_cost)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, record.ChatID, messageID, record.Model, record.PromptTokens, record.CompletionTokens, record.TotalTokens,
		record.PromptCost, record.CompletionCost, record.TotalCost)
	if err != nil {
		return fmt.Errorf("failed to save usage record: %w", err)
	}
	return nil
}

// GetUsageStats returns aggregated usage statistics across all chats.
// If startDate and endDate are provided, filters to that date range.
func (r *Repository) GetUsageStats(ctx context.Context, startDate, endDate *time.Time) (*domain.UsageStats, error) {
	stats := &domain.UsageStats{
		ByModel: make(map[string]*domain.ModelUsage),
		ByDay:   make(map[string]*domain.DailyUsage),
	}

	// Build query with optional date filters
	query := `
		SELECT
			COALESCE(SUM(prompt_tokens), 0) as total_prompt,
			COALESCE(SUM(completion_tokens), 0) as total_completion,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(total_cost), 0) as total_cost
		FROM usage_records
		WHERE 1=1
	`
	args := []interface{}{}
	argNum := 1

	if startDate != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argNum)
		args = append(args, *startDate)
		argNum++
	}
	if endDate != nil {
		query += fmt.Sprintf(" AND created_at < $%d", argNum)
		args = append(args, *endDate)
		argNum++
	}

	// Get totals
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&stats.TotalPromptTokens,
		&stats.TotalCompletionTokens,
		&stats.TotalTokens,
		&stats.TotalCost,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get usage totals: %w", err)
	}

	// Get by-model breakdown
	modelQuery := `
		SELECT
			model,
			COALESCE(SUM(prompt_tokens), 0) as prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) as completion_tokens,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(total_cost), 0) as total_cost,
			COUNT(*) as request_count
		FROM usage_records
		WHERE 1=1
	`
	args = []interface{}{}
	argNum = 1

	if startDate != nil {
		modelQuery += fmt.Sprintf(" AND created_at >= $%d", argNum)
		args = append(args, *startDate)
		argNum++
	}
	if endDate != nil {
		modelQuery += fmt.Sprintf(" AND created_at < $%d", argNum)
		args = append(args, *endDate)
		argNum++
	}
	modelQuery += " GROUP BY model ORDER BY total_cost DESC"

	rows, err := r.db.QueryContext(ctx, modelQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get model usage: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var mu domain.ModelUsage
		if err := rows.Scan(&mu.Model, &mu.PromptTokens, &mu.CompletionTokens, &mu.TotalTokens, &mu.TotalCost, &mu.RequestCount); err != nil {
			continue
		}
		stats.ByModel[mu.Model] = &mu
	}

	// Get by-day breakdown
	dayQuery := `
		SELECT
			DATE(created_at) as date,
			COALESCE(SUM(prompt_tokens), 0) as prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) as completion_tokens,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(total_cost), 0) as total_cost,
			COUNT(*) as request_count
		FROM usage_records
		WHERE 1=1
	`
	args = []interface{}{}
	argNum = 1

	if startDate != nil {
		dayQuery += fmt.Sprintf(" AND created_at >= $%d", argNum)
		args = append(args, *startDate)
		argNum++
	}
	if endDate != nil {
		dayQuery += fmt.Sprintf(" AND created_at < $%d", argNum)
		args = append(args, *endDate)
		argNum++
	}
	dayQuery += " GROUP BY DATE(created_at) ORDER BY date DESC LIMIT 30"

	rows, err = r.db.QueryContext(ctx, dayQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily usage: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var du domain.DailyUsage
		var date time.Time
		if err := rows.Scan(&date, &du.PromptTokens, &du.CompletionTokens, &du.TotalTokens, &du.TotalCost, &du.RequestCount); err != nil {
			continue
		}
		du.Date = date.Format("2006-01-02")
		stats.ByDay[du.Date] = &du
	}

	return stats, nil
}

// GetChatUsageStats returns usage statistics for a specific chat.
func (r *Repository) GetChatUsageStats(ctx context.Context, chatID string) (*domain.UsageStats, error) {
	stats := &domain.UsageStats{
		ByModel: make(map[string]*domain.ModelUsage),
	}

	// Get totals for this chat
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(prompt_tokens), 0) as total_prompt,
			COALESCE(SUM(completion_tokens), 0) as total_completion,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(total_cost), 0) as total_cost
		FROM usage_records
		WHERE chat_id = $1
	`, chatID).Scan(
		&stats.TotalPromptTokens,
		&stats.TotalCompletionTokens,
		&stats.TotalTokens,
		&stats.TotalCost,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get chat usage totals: %w", err)
	}

	// Get by-model breakdown for this chat
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			model,
			COALESCE(SUM(prompt_tokens), 0) as prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) as completion_tokens,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(total_cost), 0) as total_cost,
			COUNT(*) as request_count
		FROM usage_records
		WHERE chat_id = $1
		GROUP BY model
		ORDER BY total_cost DESC
	`, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat model usage: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var mu domain.ModelUsage
		if err := rows.Scan(&mu.Model, &mu.PromptTokens, &mu.CompletionTokens, &mu.TotalTokens, &mu.TotalCost, &mu.RequestCount); err != nil {
			continue
		}
		stats.ByModel[mu.Model] = &mu
	}

	return stats, nil
}

// GetUsageRecords returns usage records with pagination.
func (r *Repository) GetUsageRecords(ctx context.Context, chatID string, limit, offset int) ([]domain.UsageRecord, error) {
	query := `
		SELECT id, chat_id, message_id, model, prompt_tokens, completion_tokens, total_tokens,
			prompt_cost, completion_cost, total_cost, created_at
		FROM usage_records
	`
	args := []interface{}{}
	argNum := 1

	if chatID != "" {
		query += fmt.Sprintf(" WHERE chat_id = $%d", argNum)
		args = append(args, chatID)
		argNum++
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, limit)
		argNum++
	}
	if offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argNum)
		args = append(args, offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage records: %w", err)
	}
	defer rows.Close()

	records := make([]domain.UsageRecord, 0) // Always return [] instead of null in JSON
	for rows.Next() {
		var rec domain.UsageRecord
		var messageID sql.NullString
		if err := rows.Scan(&rec.ID, &rec.ChatID, &messageID, &rec.Model, &rec.PromptTokens, &rec.CompletionTokens,
			&rec.TotalTokens, &rec.PromptCost, &rec.CompletionCost, &rec.TotalCost, &rec.CreatedAt); err != nil {
			continue
		}
		if messageID.Valid {
			rec.MessageID = messageID.String
		}
		records = append(records, rec)
	}

	return records, nil
}
