package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Screenshot, logs, and extracted data repository methods

func (r *repository) CreateScreenshot(ctx context.Context, screenshot *Screenshot) error {
	query := `
		INSERT INTO screenshots (id, execution_id, step_name, storage_url, thumbnail_url,
		                        width, height, size_bytes, metadata)
		VALUES (:id, :execution_id, :step_name, :storage_url, :thumbnail_url,
		        :width, :height, :size_bytes, :metadata)`

	// Generate ID if not set
	if screenshot.ID == uuid.Nil {
		screenshot.ID = uuid.New()
	}

	_, err := r.db.NamedExecContext(ctx, query, screenshot)
	if err != nil {
		r.log.WithError(err).Error("Failed to create screenshot")
		return fmt.Errorf("failed to create screenshot: %w", err)
	}

	return nil
}

func (r *repository) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*Screenshot, error) {
	query := r.db.Rebind(`SELECT * FROM screenshots WHERE execution_id = ? ORDER BY timestamp ASC`)

	var screenshots []*Screenshot
	err := r.db.SelectContext(ctx, &screenshots, query, executionID)
	if err != nil {
		r.log.WithError(err).WithField("execution_id", executionID).Error("Failed to get screenshots")
		return nil, fmt.Errorf("failed to get screenshots: %w", err)
	}

	return screenshots, nil
}

func (r *repository) CreateExecutionLog(ctx context.Context, log *ExecutionLog) error {
	query := `
		INSERT INTO execution_logs (id, execution_id, level, step_name, message, metadata)
		VALUES (:id, :execution_id, :level, :step_name, :message, :metadata)`

	// Generate ID if not set
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}

	_, err := r.db.NamedExecContext(ctx, query, log)
	if err != nil {
		r.log.WithError(err).Error("Failed to create execution log")
		return fmt.Errorf("failed to create execution log: %w", err)
	}

	return nil
}

func (r *repository) GetExecutionLogs(ctx context.Context, executionID uuid.UUID) ([]*ExecutionLog, error) {
	query := r.db.Rebind(`SELECT * FROM execution_logs WHERE execution_id = ? ORDER BY timestamp ASC`)

	var logs []*ExecutionLog
	err := r.db.SelectContext(ctx, &logs, query, executionID)
	if err != nil {
		r.log.WithError(err).WithField("execution_id", executionID).Error("Failed to get execution logs")
		return nil, fmt.Errorf("failed to get execution logs: %w", err)
	}

	return logs, nil
}

func (r *repository) CreateExtractedData(ctx context.Context, data *ExtractedData) error {
	query := `
		INSERT INTO extracted_data (id, execution_id, step_name, data_key, data_value, data_type, metadata)
		VALUES (:id, :execution_id, :step_name, :data_key, :data_value, :data_type, :metadata)`

	// Generate ID if not set
	if data.ID == uuid.Nil {
		data.ID = uuid.New()
	}

	_, err := r.db.NamedExecContext(ctx, query, data)
	if err != nil {
		r.log.WithError(err).Error("Failed to create extracted data")
		return fmt.Errorf("failed to create extracted data: %w", err)
	}

	return nil
}

func (r *repository) GetExecutionExtractedData(ctx context.Context, executionID uuid.UUID) ([]*ExtractedData, error) {
	query := r.db.Rebind(`SELECT * FROM extracted_data WHERE execution_id = ? ORDER BY timestamp ASC`)

	var data []*ExtractedData
	err := r.db.SelectContext(ctx, &data, query, executionID)
	if err != nil {
		r.log.WithError(err).WithField("execution_id", executionID).Error("Failed to get extracted data")
		return nil, fmt.Errorf("failed to get extracted data: %w", err)
	}

	return data, nil
}
