package database

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ============================================================================
// InvestigationRepository Implementation
// ============================================================================

type investigationRepository struct {
	db  *DB
	log *logrus.Logger
}

var _ repository.InvestigationRepository = (*investigationRepository)(nil)

// investigationRow is the database row representation for investigations.
type investigationRow struct {
	ID                    uuid.UUID                   `db:"id"`
	RunIDs                UUIDSlice                   `db:"run_ids"`
	Status                string                      `db:"status"`
	AnalysisType          AnalysisTypeJSON            `db:"analysis_type"`
	ReportSections        ReportSectionsJSON          `db:"report_sections"`
	CustomContext         sql.NullString              `db:"custom_context"`
	Progress              int                         `db:"progress"`
	AgentRunID            NullableUUID                `db:"agent_run_id"`
	Findings              NullableInvestigationReport `db:"findings"`
	Metrics               NullableMetricsData         `db:"metrics"`
	ErrorMessage          sql.NullString              `db:"error_message"`
	SourceInvestigationID NullableUUID                `db:"source_investigation_id"`
	CreatedAt             SQLiteTime                  `db:"created_at"`
	StartedAt             NullableTime                `db:"started_at"`
	CompletedAt           NullableTime                `db:"completed_at"`
}

func (r *investigationRow) toDomain() *domain.Investigation {
	return &domain.Investigation{
		ID:                    r.ID,
		RunIDs:                r.RunIDs,
		Status:                domain.InvestigationStatus(r.Status),
		AnalysisType:          r.AnalysisType.V,
		ReportSections:        r.ReportSections.V,
		CustomContext:         r.CustomContext.String,
		Progress:              r.Progress,
		AgentRunID:            r.AgentRunID.ToPtr(),
		Findings:              r.Findings.V,
		Metrics:               r.Metrics.V,
		ErrorMessage:          r.ErrorMessage.String,
		SourceInvestigationID: r.SourceInvestigationID.ToPtr(),
		CreatedAt:             r.CreatedAt.Time(),
		StartedAt:             r.StartedAt.ToPtr(),
		CompletedAt:           r.CompletedAt.ToPtr(),
	}
}

func investigationFromDomain(i *domain.Investigation) *investigationRow {
	customContext := sql.NullString{}
	if i.CustomContext != "" {
		customContext = sql.NullString{String: i.CustomContext, Valid: true}
	}
	errorMessage := sql.NullString{}
	if i.ErrorMessage != "" {
		errorMessage = sql.NullString{String: i.ErrorMessage, Valid: true}
	}
	return &investigationRow{
		ID:                    i.ID,
		RunIDs:                i.RunIDs,
		Status:                string(i.Status),
		AnalysisType:          AnalysisTypeJSON{V: i.AnalysisType},
		ReportSections:        ReportSectionsJSON{V: i.ReportSections},
		CustomContext:         customContext,
		Progress:              i.Progress,
		AgentRunID:            NewNullableUUID(i.AgentRunID),
		Findings:              NullableInvestigationReport{V: i.Findings},
		Metrics:               NullableMetricsData{V: i.Metrics},
		ErrorMessage:          errorMessage,
		SourceInvestigationID: NewNullableUUID(i.SourceInvestigationID),
		CreatedAt:             SQLiteTime(i.CreatedAt),
		StartedAt:             NewNullableTime(i.StartedAt),
		CompletedAt:           NewNullableTime(i.CompletedAt),
	}
}

// Create stores a new investigation.
func (r *investigationRepository) Create(ctx context.Context, investigation *domain.Investigation) error {
	row := investigationFromDomain(investigation)

	query := `
		INSERT INTO investigations (
			id, run_ids, status, analysis_type, report_sections, custom_context,
			progress, agent_run_id, findings, metrics, error_message,
			source_investigation_id, created_at, started_at, completed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		row.ID.String(),
		row.RunIDs,
		row.Status,
		row.AnalysisType,
		row.ReportSections,
		row.CustomContext,
		row.Progress,
		row.AgentRunID,
		row.Findings,
		row.Metrics,
		row.ErrorMessage,
		row.SourceInvestigationID,
		row.CreatedAt,
		row.StartedAt,
		row.CompletedAt,
	)
	if err != nil {
		return &domain.DatabaseError{
			Operation:  "create",
			EntityType: "Investigation",
			EntityID:   investigation.ID.String(),
			Cause:      err,
		}
	}
	return nil
}

// Get retrieves an investigation by ID.
func (r *investigationRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Investigation, error) {
	query := `
		SELECT id, run_ids, status, analysis_type, report_sections, custom_context,
			progress, agent_run_id, findings, metrics, error_message,
			source_investigation_id, created_at, started_at, completed_at
		FROM investigations
		WHERE id = ?`

	var row investigationRow
	err := r.db.QueryRowContext(ctx, query, id.String()).Scan(
		&row.ID,
		&row.RunIDs,
		&row.Status,
		&row.AnalysisType,
		&row.ReportSections,
		&row.CustomContext,
		&row.Progress,
		&row.AgentRunID,
		&row.Findings,
		&row.Metrics,
		&row.ErrorMessage,
		&row.SourceInvestigationID,
		&row.CreatedAt,
		&row.StartedAt,
		&row.CompletedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.NewNotFoundError("Investigation", id)
	}
	if err != nil {
		return nil, &domain.DatabaseError{
			Operation:  "get",
			EntityType: "Investigation",
			EntityID:   id.String(),
			Cause:      err,
		}
	}
	return row.toDomain(), nil
}

// List retrieves investigations with optional filtering.
func (r *investigationRepository) List(ctx context.Context, filter repository.InvestigationListFilter) ([]*domain.Investigation, error) {
	query := `
		SELECT id, run_ids, status, analysis_type, report_sections, custom_context,
			progress, agent_run_id, findings, metrics, error_message,
			source_investigation_id, created_at, started_at, completed_at
		FROM investigations
		WHERE 1=1`

	var args []interface{}

	if filter.Status != nil {
		query += " AND status = ?"
		args = append(args, string(*filter.Status))
	}
	if filter.SourceInvestigationID != nil {
		query += " AND source_investigation_id = ?"
		args = append(args, filter.SourceInvestigationID.String())
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, &domain.DatabaseError{
			Operation:  "list",
			EntityType: "Investigation",
			Cause:      err,
		}
	}
	defer rows.Close()

	var investigations []*domain.Investigation
	for rows.Next() {
		var row investigationRow
		if err := rows.Scan(
			&row.ID,
			&row.RunIDs,
			&row.Status,
			&row.AnalysisType,
			&row.ReportSections,
			&row.CustomContext,
			&row.Progress,
			&row.AgentRunID,
			&row.Findings,
			&row.Metrics,
			&row.ErrorMessage,
			&row.SourceInvestigationID,
			&row.CreatedAt,
			&row.StartedAt,
			&row.CompletedAt,
		); err != nil {
			return nil, &domain.DatabaseError{
				Operation:  "list_scan",
				EntityType: "Investigation",
				Cause:      err,
			}
		}
		investigations = append(investigations, row.toDomain())
	}

	if err := rows.Err(); err != nil {
		return nil, &domain.DatabaseError{
			Operation:  "list_iterate",
			EntityType: "Investigation",
			Cause:      err,
		}
	}

	return investigations, nil
}

// GetActive retrieves any active (pending/running) investigation.
func (r *investigationRepository) GetActive(ctx context.Context) (*domain.Investigation, error) {
	query := `
		SELECT id, run_ids, status, analysis_type, report_sections, custom_context,
			progress, agent_run_id, findings, metrics, error_message,
			source_investigation_id, created_at, started_at, completed_at
		FROM investigations
		WHERE status IN ('pending', 'running')
		ORDER BY created_at DESC
		LIMIT 1`

	var row investigationRow
	err := r.db.QueryRowContext(ctx, query).Scan(
		&row.ID,
		&row.RunIDs,
		&row.Status,
		&row.AnalysisType,
		&row.ReportSections,
		&row.CustomContext,
		&row.Progress,
		&row.AgentRunID,
		&row.Findings,
		&row.Metrics,
		&row.ErrorMessage,
		&row.SourceInvestigationID,
		&row.CreatedAt,
		&row.StartedAt,
		&row.CompletedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil // No active investigation
	}
	if err != nil {
		return nil, &domain.DatabaseError{
			Operation:  "get_active",
			EntityType: "Investigation",
			Cause:      err,
		}
	}
	return row.toDomain(), nil
}

// Update modifies an existing investigation.
func (r *investigationRepository) Update(ctx context.Context, investigation *domain.Investigation) error {
	row := investigationFromDomain(investigation)

	query := `
		UPDATE investigations SET
			run_ids = ?, status = ?, analysis_type = ?, report_sections = ?,
			custom_context = ?, progress = ?, agent_run_id = ?, findings = ?,
			metrics = ?, error_message = ?, source_investigation_id = ?,
			started_at = ?, completed_at = ?
		WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query,
		row.RunIDs,
		row.Status,
		row.AnalysisType,
		row.ReportSections,
		row.CustomContext,
		row.Progress,
		row.AgentRunID,
		row.Findings,
		row.Metrics,
		row.ErrorMessage,
		row.SourceInvestigationID,
		row.StartedAt,
		row.CompletedAt,
		row.ID.String(),
	)
	if err != nil {
		return &domain.DatabaseError{
			Operation:  "update",
			EntityType: "Investigation",
			EntityID:   investigation.ID.String(),
			Cause:      err,
		}
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return domain.NewNotFoundError("Investigation", investigation.ID)
	}
	return nil
}

// UpdateStatus updates just the status and related fields.
func (r *investigationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.InvestigationStatus, errorMsg string) error {
	var query string
	var args []interface{}

	now := time.Now()

	if status == domain.InvestigationStatusRunning {
		query = `UPDATE investigations SET status = ?, started_at = ? WHERE id = ?`
		args = []interface{}{string(status), NewNullableTime(&now), id.String()}
	} else if status == domain.InvestigationStatusCompleted || status == domain.InvestigationStatusFailed || status == domain.InvestigationStatusCancelled {
		query = `UPDATE investigations SET status = ?, error_message = ?, completed_at = ? WHERE id = ?`
		errorNullStr := sql.NullString{}
		if errorMsg != "" {
			errorNullStr = sql.NullString{String: errorMsg, Valid: true}
		}
		args = []interface{}{string(status), errorNullStr, NewNullableTime(&now), id.String()}
	} else {
		query = `UPDATE investigations SET status = ? WHERE id = ?`
		args = []interface{}{string(status), id.String()}
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return &domain.DatabaseError{
			Operation:  "update_status",
			EntityType: "Investigation",
			EntityID:   id.String(),
			Cause:      err,
		}
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return domain.NewNotFoundError("Investigation", id)
	}
	return nil
}

// UpdateProgress updates the progress percentage.
func (r *investigationRepository) UpdateProgress(ctx context.Context, id uuid.UUID, progress int) error {
	query := `UPDATE investigations SET progress = ? WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, progress, id.String())
	if err != nil {
		return &domain.DatabaseError{
			Operation:  "update_progress",
			EntityType: "Investigation",
			EntityID:   id.String(),
			Cause:      err,
		}
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return domain.NewNotFoundError("Investigation", id)
	}
	return nil
}

// UpdateFindings stores the investigation results.
func (r *investigationRepository) UpdateFindings(ctx context.Context, id uuid.UUID, findings *domain.InvestigationReport, metrics *domain.MetricsData) error {
	now := time.Now()
	query := `UPDATE investigations SET findings = ?, metrics = ?, status = ?, progress = 100, completed_at = ? WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query,
		NullableInvestigationReport{V: findings},
		NullableMetricsData{V: metrics},
		string(domain.InvestigationStatusCompleted),
		NewNullableTime(&now),
		id.String(),
	)
	if err != nil {
		return &domain.DatabaseError{
			Operation:  "update_findings",
			EntityType: "Investigation",
			EntityID:   id.String(),
			Cause:      err,
		}
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return domain.NewNotFoundError("Investigation", id)
	}
	return nil
}

// Delete removes an investigation by ID.
func (r *investigationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM investigations WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id.String())
	if err != nil {
		return &domain.DatabaseError{
			Operation:  "delete",
			EntityType: "Investigation",
			EntityID:   id.String(),
			Cause:      err,
		}
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return domain.NewNotFoundError("Investigation", id)
	}
	return nil
}
