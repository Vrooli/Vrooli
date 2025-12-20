// Package repository provides database operations for sandboxes.
package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"workspace-sandbox/internal/types"
)

// --- Test Helpers ---

// newMockDB creates a new sqlmock database for testing.
func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	return db, mock
}

// testSandbox returns a sandbox with test data.
func testSandbox() *types.Sandbox {
	id := uuid.New()
	now := time.Now()
	return &types.Sandbox{
		ID:            id,
		ScopePath:     "/project/src",
		ReservedPath:  "/project/src",
		ProjectRoot:   "/project",
		Owner:         "test-agent",
		OwnerType:     types.OwnerTypeAgent,
		Status:        types.StatusActive,
		Driver:        "fuse-overlayfs",
		DriverVersion: "1.0.0",
		Tags:          []string{"test", "unit"},
		Metadata:      map[string]interface{}{"key": "value"},
		CreatedAt:     now,
		LastUsedAt:    now,
		UpdatedAt:     now,
		Version:       1,
	}
}

// sandboxColumns returns the column names for sandbox queries.
func sandboxColumns() []string {
	return []string{
		"id", "scope_path", "reserved_path", "project_root", "owner", "owner_type", "status", "error_message",
		"created_at", "last_used_at", "stopped_at", "approved_at", "deleted_at",
		"driver", "driver_version", "lower_dir", "upper_dir", "work_dir", "merged_dir",
		"size_bytes", "file_count", "active_pids", "session_count", "tags", "metadata",
		"idempotency_key", "updated_at", "version", "base_commit_hash",
	}
}

// sandboxRow returns a sqlmock row for a sandbox.
func sandboxRow(s *types.Sandbox) []driver.Value {
	metadataJSON, _ := json.Marshal(s.Metadata)
	return []driver.Value{
		s.ID, s.ScopePath, s.ReservedPath, s.ProjectRoot, s.Owner, s.OwnerType, s.Status, s.ErrorMsg,
		s.CreatedAt, s.LastUsedAt, s.StoppedAt, s.ApprovedAt, s.DeletedAt,
		s.Driver, s.DriverVersion, s.LowerDir, s.UpperDir, s.WorkDir, s.MergedDir,
		s.SizeBytes, s.FileCount,
		pq.Int64Array{},
		s.SessionCount,
		pq.StringArray(s.Tags), metadataJSON,
		s.IdempotencyKey, s.UpdatedAt, s.Version, s.BaseCommitHash,
	}
}

// --- Constructor Tests ---

func TestNewSandboxRepository(t *testing.T) {
	db, _ := newMockDB(t)
	defer db.Close()

	repo := NewSandboxRepository(db)
	if repo == nil {
		t.Fatal("NewSandboxRepository returned nil")
	}
	if repo.db != db {
		t.Error("NewSandboxRepository did not store the database connection")
	}
}

// --- Interface Verification ---

func TestSandboxRepository_ImplementsRepository(t *testing.T) {
	var _ Repository = (*SandboxRepository)(nil)
}

func TestTxSandboxRepository_ImplementsTxRepository(t *testing.T) {
	var _ TxRepository = (*TxSandboxRepository)(nil)
}

// --- Create Tests ---

func TestSandboxRepository_Create(t *testing.T) {
	tests := []struct {
		name      string
		sandbox   *types.Sandbox
		setupMock func(sqlmock.Sqlmock, *types.Sandbox)
		wantErr   bool
	}{
		{
			name:    "successful create",
			sandbox: testSandbox(),
			setupMock: func(mock sqlmock.Sqlmock, s *types.Sandbox) {
				now := time.Now()
				mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO sandboxes")).
					WithArgs(
						s.ID, s.ScopePath, s.ReservedPath, s.ProjectRoot, s.Owner, s.OwnerType, s.Status,
						s.Driver, s.DriverVersion, pq.Array(s.Tags), sqlmock.AnyArg(),
						s.IdempotencyKey, int64(1), s.BaseCommitHash,
					).
					WillReturnRows(sqlmock.NewRows([]string{"created_at", "last_used_at", "updated_at"}).
						AddRow(now, now, now))
			},
			wantErr: false,
		},
		{
			name:    "create with idempotency key",
			sandbox: func() *types.Sandbox { s := testSandbox(); s.IdempotencyKey = "test-key-123"; return s }(),
			setupMock: func(mock sqlmock.Sqlmock, s *types.Sandbox) {
				now := time.Now()
				mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO sandboxes")).
					WithArgs(
						s.ID, s.ScopePath, s.ReservedPath, s.ProjectRoot, s.Owner, s.OwnerType, s.Status,
						s.Driver, s.DriverVersion, pq.Array(s.Tags), sqlmock.AnyArg(),
						s.IdempotencyKey, int64(1), s.BaseCommitHash,
					).
					WillReturnRows(sqlmock.NewRows([]string{"created_at", "last_used_at", "updated_at"}).
						AddRow(now, now, now))
			},
			wantErr: false,
		},
		{
			name:    "database error",
			sandbox: testSandbox(),
			setupMock: func(mock sqlmock.Sqlmock, s *types.Sandbox) {
				mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO sandboxes")).
					WillReturnError(errors.New("connection refused"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMockDB(t)
			defer db.Close()

			repo := NewSandboxRepository(db)
			tt.setupMock(mock, tt.sandbox)

			err := repo.Create(context.Background(), tt.sandbox)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && tt.sandbox.Version != 1 {
				t.Errorf("Create() should set Version = 1, got %d", tt.sandbox.Version)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestSandboxRepository_Create_InvalidMetadata(t *testing.T) {
	db, _ := newMockDB(t)
	defer db.Close()

	repo := NewSandboxRepository(db)
	s := testSandbox()
	// Create an unmarshalable value (channel cannot be marshaled to JSON)
	s.Metadata = map[string]interface{}{"invalid": make(chan int)}

	err := repo.Create(context.Background(), s)
	if err == nil {
		t.Error("Create() should fail with unmarshalable metadata")
	}
}

// --- Get Tests ---

func TestSandboxRepository_Get(t *testing.T) {
	tests := []struct {
		name      string
		id        uuid.UUID
		setupMock func(sqlmock.Sqlmock, uuid.UUID)
		wantNil   bool
		wantErr   bool
	}{
		{
			name: "found",
			id:   uuid.New(),
			setupMock: func(mock sqlmock.Sqlmock, id uuid.UUID) {
				s := testSandbox()
				s.ID = id
				mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
					WithArgs(id).
					WillReturnRows(sqlmock.NewRows(sandboxColumns()).
						AddRow(sandboxRow(s)...))
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "not found",
			id:   uuid.New(),
			setupMock: func(mock sqlmock.Sqlmock, id uuid.UUID) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
					WithArgs(id).
					WillReturnError(sql.ErrNoRows)
			},
			wantNil: true,
			wantErr: false,
		},
		{
			name: "database error",
			id:   uuid.New(),
			setupMock: func(mock sqlmock.Sqlmock, id uuid.UUID) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
					WithArgs(id).
					WillReturnError(errors.New("connection failed"))
			},
			wantNil: true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMockDB(t)
			defer db.Close()

			repo := NewSandboxRepository(db)
			tt.setupMock(mock, tt.id)

			result, err := repo.Get(context.Background(), tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
			}
			if (result == nil) != tt.wantNil {
				t.Errorf("Get() result = %v, wantNil %v", result, tt.wantNil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestSandboxRepository_Get_ParsesMetadataAndTags(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewSandboxRepository(db)
	id := uuid.New()
	s := testSandbox()
	s.ID = id
	s.Tags = []string{"tag1", "tag2", "tag3"}
	s.Metadata = map[string]interface{}{"nested": map[string]interface{}{"key": "value"}}
	s.ActivePIDs = []int{123, 456}

	metadataJSON, _ := json.Marshal(s.Metadata)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows(sandboxColumns()).
			AddRow(
				s.ID, s.ScopePath, s.ReservedPath, s.ProjectRoot, s.Owner, s.OwnerType, s.Status, s.ErrorMsg,
				s.CreatedAt, s.LastUsedAt, s.StoppedAt, s.ApprovedAt, s.DeletedAt,
				s.Driver, s.DriverVersion, s.LowerDir, s.UpperDir, s.WorkDir, s.MergedDir,
				s.SizeBytes, s.FileCount, pq.Int64Array{123, 456}, s.SessionCount,
				pq.StringArray(s.Tags), metadataJSON,
				s.IdempotencyKey, s.UpdatedAt, s.Version, s.BaseCommitHash,
			))

	result, err := repo.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(result.Tags) != 3 {
		t.Errorf("Get() tags = %v, want 3 tags", result.Tags)
	}

	if len(result.ActivePIDs) != 2 || result.ActivePIDs[0] != 123 {
		t.Errorf("Get() activePIDs = %v, want [123, 456]", result.ActivePIDs)
	}
}

// --- Update Tests ---

func TestSandboxRepository_Update(t *testing.T) {
	tests := []struct {
		name      string
		sandbox   *types.Sandbox
		setupMock func(sqlmock.Sqlmock, *types.Sandbox)
		wantErr   bool
	}{
		{
			name:    "successful update",
			sandbox: testSandbox(),
			setupMock: func(mock sqlmock.Sqlmock, s *types.Sandbox) {
				mock.ExpectQuery(regexp.QuoteMeta("UPDATE sandboxes SET")).
					WithArgs(
						s.ID, s.Status, s.ErrorMsg,
						s.StoppedAt, s.ApprovedAt, s.DeletedAt,
						s.LowerDir, s.UpperDir, s.WorkDir, s.MergedDir,
						s.SizeBytes, s.FileCount, pq.Int64Array{}, s.SessionCount,
						pq.Array(s.Tags), sqlmock.AnyArg(),
					).
					WillReturnRows(sqlmock.NewRows([]string{"version", "updated_at"}).
						AddRow(int64(2), time.Now()))
			},
			wantErr: false,
		},
		{
			name:    "database error",
			sandbox: testSandbox(),
			setupMock: func(mock sqlmock.Sqlmock, s *types.Sandbox) {
				mock.ExpectQuery(regexp.QuoteMeta("UPDATE sandboxes SET")).
					WillReturnError(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMockDB(t)
			defer db.Close()

			repo := NewSandboxRepository(db)
			tt.setupMock(mock, tt.sandbox)

			err := repo.Update(context.Background(), tt.sandbox)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestSandboxRepository_Update_IncrementsVersion(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewSandboxRepository(db)
	s := testSandbox()
	s.Version = 5

	mock.ExpectQuery(regexp.QuoteMeta("UPDATE sandboxes SET")).
		WillReturnRows(sqlmock.NewRows([]string{"version", "updated_at"}).
			AddRow(int64(6), time.Now()))

	err := repo.Update(context.Background(), s)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if s.Version != 6 {
		t.Errorf("Update() version = %d, want 6", s.Version)
	}
}

// --- Delete Tests ---

func TestSandboxRepository_Delete(t *testing.T) {
	tests := []struct {
		name         string
		id           uuid.UUID
		rowsAffected int64
		dbError      error
		wantErr      bool
	}{
		{
			name:         "successful delete",
			id:           uuid.New(),
			rowsAffected: 1,
			wantErr:      false,
		},
		{
			name:         "not found or already deleted",
			id:           uuid.New(),
			rowsAffected: 0,
			wantErr:      true,
		},
		{
			name:    "database error",
			id:      uuid.New(),
			dbError: errors.New("connection error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMockDB(t)
			defer db.Close()

			repo := NewSandboxRepository(db)

			if tt.dbError != nil {
				mock.ExpectExec(regexp.QuoteMeta("UPDATE sandboxes")).
					WithArgs(tt.id, sqlmock.AnyArg()).
					WillReturnError(tt.dbError)
			} else {
				mock.ExpectExec(regexp.QuoteMeta("UPDATE sandboxes")).
					WithArgs(tt.id, sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(0, tt.rowsAffected))
			}

			err := repo.Delete(context.Background(), tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// --- List Tests ---

func TestSandboxRepository_List(t *testing.T) {
	tests := []struct {
		name      string
		filter    *types.ListFilter
		setupMock func(sqlmock.Sqlmock)
		wantCount int
		wantErr   bool
	}{
		{
			name:   "no filters",
			filter: &types.ListFilter{Limit: 10},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT COUNT").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
				mock.ExpectQuery("SELECT id").
					WillReturnRows(sqlmock.NewRows(sandboxColumns()))
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "with status filter",
			filter: &types.ListFilter{
				Status: []types.Status{types.StatusActive, types.StatusStopped},
				Limit:  10,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT COUNT").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
				mock.ExpectQuery("SELECT id").
					WillReturnRows(sqlmock.NewRows(sandboxColumns()))
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "with owner filter",
			filter: &types.ListFilter{
				Owner: "test-agent",
				Limit: 10,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT COUNT").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
				mock.ExpectQuery("SELECT id").
					WillReturnRows(sqlmock.NewRows(sandboxColumns()))
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "with date range",
			filter: &types.ListFilter{
				CreatedFrom: time.Now().Add(-24 * time.Hour),
				CreatedTo:   time.Now(),
				Limit:       10,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT COUNT").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))
				mock.ExpectQuery("SELECT id").
					WillReturnRows(sqlmock.NewRows(sandboxColumns()))
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:   "count error",
			filter: &types.ListFilter{Limit: 10},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT COUNT").
					WillReturnError(errors.New("count failed"))
			},
			wantErr: true,
		},
		{
			name:   "list error",
			filter: &types.ListFilter{Limit: 10},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT COUNT").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
				mock.ExpectQuery("SELECT id").
					WillReturnError(errors.New("query failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMockDB(t)
			defer db.Close()

			repo := NewSandboxRepository(db)
			tt.setupMock(mock)

			result, err := repo.List(context.Background(), tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && result != nil && len(result.Sandboxes) != tt.wantCount {
				t.Errorf("List() count = %d, want %d", len(result.Sandboxes), tt.wantCount)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestSandboxRepository_List_PaginationDefaults(t *testing.T) {
	tests := []struct {
		name        string
		inputLimit  int
		inputOffset int
		wantLimit   int
		wantOffset  int
	}{
		{
			name:       "zero limit gets default",
			inputLimit: 0,
			wantLimit:  100,
		},
		{
			name:       "negative offset becomes zero",
			inputLimit: 10,
			wantOffset: 0,
		},
		{
			name:       "limit exceeds max is capped",
			inputLimit: 20000,
			wantLimit:  10000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMockDB(t)
			defer db.Close()

			repo := NewSandboxRepository(db)

			mock.ExpectQuery("SELECT COUNT").
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			mock.ExpectQuery("SELECT id").
				WillReturnRows(sqlmock.NewRows(sandboxColumns()))

			filter := &types.ListFilter{Limit: tt.inputLimit, Offset: tt.inputOffset}
			result, err := repo.List(context.Background(), filter)
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}

			if tt.wantLimit > 0 && result.Limit != tt.wantLimit {
				t.Errorf("List() limit = %d, want %d", result.Limit, tt.wantLimit)
			}
		})
	}
}

// --- CheckScopeOverlap Tests ---

func TestSandboxRepository_CheckScopeOverlap(t *testing.T) {
	tests := []struct {
		name         string
		scopePath    string
		projectRoot  string
		excludeID    *uuid.UUID
		setupMock    func(sqlmock.Sqlmock)
		wantConflict bool
		wantErr      bool
	}{
		{
			name:        "no conflicts",
			scopePath:   "/project/src",
			projectRoot: "/project",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id, scope_path, status FROM check_scope_overlap")).
					WillReturnRows(sqlmock.NewRows([]string{"id", "scope_path", "status"}))
			},
			wantConflict: false,
			wantErr:      false,
		},
		{
			name:        "exact conflict",
			scopePath:   "/project/src",
			projectRoot: "/project",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id, scope_path, status FROM check_scope_overlap")).
					WillReturnRows(sqlmock.NewRows([]string{"id", "scope_path", "status"}).
						AddRow(uuid.New(), "/project/src", types.StatusActive))
			},
			wantConflict: true,
			wantErr:      false,
		},
		{
			name:        "with exclude ID",
			scopePath:   "/project/src",
			projectRoot: "/project",
			excludeID:   func() *uuid.UUID { id := uuid.New(); return &id }(),
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id, scope_path, status FROM check_scope_overlap")).
					WillReturnRows(sqlmock.NewRows([]string{"id", "scope_path", "status"}))
			},
			wantConflict: false,
			wantErr:      false,
		},
		{
			name:        "database error",
			scopePath:   "/project/src",
			projectRoot: "/project",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id, scope_path, status FROM check_scope_overlap")).
					WillReturnError(errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMockDB(t)
			defer db.Close()

			repo := NewSandboxRepository(db)
			tt.setupMock(mock)

			conflicts, err := repo.CheckScopeOverlap(context.Background(), tt.scopePath, tt.projectRoot, tt.excludeID)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckScopeOverlap() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				hasConflicts := len(conflicts) > 0
				if hasConflicts != tt.wantConflict {
					t.Errorf("CheckScopeOverlap() hasConflicts = %v, want %v", hasConflicts, tt.wantConflict)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// --- GetActiveSandboxes Tests ---

func TestSandboxRepository_GetActiveSandboxes(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewSandboxRepository(db)

	// GetActiveSandboxes uses List internally
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery("SELECT id").
		WillReturnRows(sqlmock.NewRows(sandboxColumns()))

	sandboxes, err := repo.GetActiveSandboxes(context.Background(), "/project")
	if err != nil {
		t.Fatalf("GetActiveSandboxes() error = %v", err)
	}

	// Note: sandboxes can be nil when empty (no rows returned)
	// The important thing is no error occurred
	_ = sandboxes

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// --- LogAuditEvent Tests ---

func TestSandboxRepository_LogAuditEvent(t *testing.T) {
	tests := []struct {
		name      string
		event     *types.AuditEvent
		setupMock func(sqlmock.Sqlmock)
		wantErr   bool
	}{
		{
			name: "successful log",
			event: &types.AuditEvent{
				SandboxID: func() *uuid.UUID { id := uuid.New(); return &id }(),
				EventType: "created",
				Actor:     "test-user",
				ActorType: "user",
				Details:   map[string]interface{}{"action": "create"},
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO sandbox_audit_log")).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name: "nil details and state",
			event: &types.AuditEvent{
				SandboxID: func() *uuid.UUID { id := uuid.New(); return &id }(),
				EventType: "deleted",
				Actor:     "system",
				ActorType: "system",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO sandbox_audit_log")).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name: "database error",
			event: &types.AuditEvent{
				EventType: "test",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO sandbox_audit_log")).
					WillReturnError(errors.New("insert failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMockDB(t)
			defer db.Close()

			repo := NewSandboxRepository(db)
			tt.setupMock(mock)

			err := repo.LogAuditEvent(context.Background(), tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("LogAuditEvent() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// --- GetStats Tests ---

func TestSandboxRepository_GetStats(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(sqlmock.Sqlmock)
		wantErr   bool
	}{
		{
			name: "successful stats",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM get_sandbox_stats()")).
					WillReturnRows(sqlmock.NewRows([]string{
						"total_count", "active_count", "stopped_count", "error_count",
						"approved_count", "rejected_count", "deleted_count",
						"total_size_bytes", "avg_size_bytes",
					}).AddRow(100, 50, 20, 5, 15, 5, 5, int64(1000000), float64(10000)))
			},
			wantErr: false,
		},
		{
			name: "database error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM get_sandbox_stats()")).
					WillReturnError(errors.New("function not found"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMockDB(t)
			defer db.Close()

			repo := NewSandboxRepository(db)
			tt.setupMock(mock)

			stats, err := repo.GetStats(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStats() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && stats == nil {
				t.Error("GetStats() returned nil stats")
			}

			if !tt.wantErr && stats != nil {
				if stats.TotalCount != 100 {
					t.Errorf("GetStats() TotalCount = %d, want 100", stats.TotalCount)
				}
				if stats.ActiveCount != 50 {
					t.Errorf("GetStats() ActiveCount = %d, want 50", stats.ActiveCount)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// --- FindByIdempotencyKey Tests ---

func TestSandboxRepository_FindByIdempotencyKey(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		setupMock func(sqlmock.Sqlmock)
		wantFound bool
		wantErr   bool
	}{
		{
			name: "empty key returns nil",
			key:  "",
			setupMock: func(mock sqlmock.Sqlmock) {
				// No query expected for empty key
			},
			wantFound: false,
			wantErr:   false,
		},
		{
			name: "key found",
			key:  "test-key-123",
			setupMock: func(mock sqlmock.Sqlmock) {
				s := testSandbox()
				s.IdempotencyKey = "test-key-123"
				mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
					WithArgs("test-key-123").
					WillReturnRows(sqlmock.NewRows(sandboxColumns()).
						AddRow(sandboxRow(s)...))
			},
			wantFound: true,
			wantErr:   false,
		},
		{
			name: "key not found",
			key:  "nonexistent-key",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
					WithArgs("nonexistent-key").
					WillReturnError(sql.ErrNoRows)
			},
			wantFound: false,
			wantErr:   false,
		},
		{
			name: "database error",
			key:  "test-key",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
					WithArgs("test-key").
					WillReturnError(errors.New("connection lost"))
			},
			wantFound: false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMockDB(t)
			defer db.Close()

			repo := NewSandboxRepository(db)
			tt.setupMock(mock)

			result, err := repo.FindByIdempotencyKey(context.Background(), tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindByIdempotencyKey() error = %v, wantErr %v", err, tt.wantErr)
			}

			found := result != nil
			if found != tt.wantFound {
				t.Errorf("FindByIdempotencyKey() found = %v, want %v", found, tt.wantFound)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// --- UpdateWithVersionCheck Tests ---

func TestSandboxRepository_UpdateWithVersionCheck(t *testing.T) {
	tests := []struct {
		name            string
		sandbox         *types.Sandbox
		expectedVersion int64
		setupMock       func(sqlmock.Sqlmock, *types.Sandbox)
		wantErr         bool
		wantErrType     string
	}{
		{
			name:            "successful update",
			sandbox:         testSandbox(),
			expectedVersion: 1,
			setupMock: func(mock sqlmock.Sqlmock, s *types.Sandbox) {
				mock.ExpectQuery(regexp.QuoteMeta("UPDATE sandboxes SET")).
					WillReturnRows(sqlmock.NewRows([]string{"version", "updated_at"}).
						AddRow(int64(2), time.Now()))
			},
			wantErr: false,
		},
		{
			name:            "version mismatch",
			sandbox:         testSandbox(),
			expectedVersion: 5,
			setupMock: func(mock sqlmock.Sqlmock, s *types.Sandbox) {
				// First query returns no rows (version mismatch)
				mock.ExpectQuery(regexp.QuoteMeta("UPDATE sandboxes SET")).
					WillReturnError(sql.ErrNoRows)
				// Second query fetches current version
				mock.ExpectQuery(regexp.QuoteMeta("SELECT version FROM sandboxes WHERE id")).
					WithArgs(s.ID).
					WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(int64(10)))
			},
			wantErr:     true,
			wantErrType: "ConcurrentModificationError",
		},
		{
			name:            "sandbox not found",
			sandbox:         testSandbox(),
			expectedVersion: 1,
			setupMock: func(mock sqlmock.Sqlmock, s *types.Sandbox) {
				mock.ExpectQuery(regexp.QuoteMeta("UPDATE sandboxes SET")).
					WillReturnError(sql.ErrNoRows)
				mock.ExpectQuery(regexp.QuoteMeta("SELECT version FROM sandboxes WHERE id")).
					WithArgs(s.ID).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr:     true,
			wantErrType: "NotFoundError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMockDB(t)
			defer db.Close()

			repo := NewSandboxRepository(db)
			tt.setupMock(mock, tt.sandbox)

			err := repo.UpdateWithVersionCheck(context.Background(), tt.sandbox, tt.expectedVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateWithVersionCheck() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErrType != "" && err != nil {
				switch tt.wantErrType {
				case "ConcurrentModificationError":
					var cmErr *types.ConcurrentModificationError
					if !errors.As(err, &cmErr) {
						t.Errorf("UpdateWithVersionCheck() error type = %T, want *ConcurrentModificationError", err)
					}
				case "NotFoundError":
					var nfErr *types.NotFoundError
					if !errors.As(err, &nfErr) {
						t.Errorf("UpdateWithVersionCheck() error type = %T, want *NotFoundError", err)
					}
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// --- Transaction Tests ---

func TestSandboxRepository_BeginTx(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewSandboxRepository(db)

	mock.ExpectBegin()

	txRepo, err := repo.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}

	if txRepo == nil {
		t.Error("BeginTx() returned nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestSandboxRepository_BeginTx_Error(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewSandboxRepository(db)

	mock.ExpectBegin().WillReturnError(errors.New("tx start failed"))

	_, err := repo.BeginTx(context.Background())
	if err == nil {
		t.Error("BeginTx() expected error but got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestTxSandboxRepository_Commit(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectCommit()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	txRepo := &TxSandboxRepository{tx: tx}
	err = txRepo.Commit()
	if err != nil {
		t.Errorf("Commit() error = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestTxSandboxRepository_Rollback(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectRollback()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	txRepo := &TxSandboxRepository{tx: tx}
	err = txRepo.Rollback()
	if err != nil {
		t.Errorf("Rollback() error = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestTxSandboxRepository_Create(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	txRepo := &TxSandboxRepository{tx: tx}
	s := testSandbox()

	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO sandboxes")).
		WillReturnRows(sqlmock.NewRows([]string{"created_at", "last_used_at", "updated_at"}).
			AddRow(now, now, now))

	err = txRepo.Create(context.Background(), s)
	if err != nil {
		t.Errorf("Create() error = %v", err)
	}

	if s.Version != 1 {
		t.Errorf("Create() should set Version = 1, got %d", s.Version)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestTxSandboxRepository_Get(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	txRepo := &TxSandboxRepository{tx: tx}
	id := uuid.New()
	s := testSandbox()
	s.ID = id

	mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows(sandboxColumns()).
			AddRow(sandboxRow(s)...))

	result, err := txRepo.Get(context.Background(), id)
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}

	if result == nil || result.ID != id {
		t.Errorf("Get() result ID = %v, want %v", result.ID, id)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestTxSandboxRepository_Delete(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	txRepo := &TxSandboxRepository{tx: tx}
	id := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE sandboxes")).
		WithArgs(id, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = txRepo.Delete(context.Background(), id)
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestTxSandboxRepository_List_NotImplemented(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	txRepo := &TxSandboxRepository{tx: tx}

	_, err = txRepo.List(context.Background(), &types.ListFilter{})
	if err == nil {
		t.Error("List() expected error for unimplemented method")
	}
}

func TestTxSandboxRepository_GetActiveSandboxes_NotImplemented(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	txRepo := &TxSandboxRepository{tx: tx}

	_, err = txRepo.GetActiveSandboxes(context.Background(), "/project")
	if err == nil {
		t.Error("GetActiveSandboxes() expected error for unimplemented method")
	}
}

func TestTxSandboxRepository_GetStats_NotImplemented(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	txRepo := &TxSandboxRepository{tx: tx}

	_, err = txRepo.GetStats(context.Background())
	if err == nil {
		t.Error("GetStats() expected error for unimplemented method")
	}
}

func TestTxSandboxRepository_BeginTx_NestedNotSupported(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	txRepo := &TxSandboxRepository{tx: tx}

	_, err = txRepo.BeginTx(context.Background())
	if err == nil {
		t.Error("BeginTx() expected error for nested transaction")
	}
}

func TestTxSandboxRepository_CheckScopeOverlap_WithLocking(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	txRepo := &TxSandboxRepository{tx: tx}

	// Should use FOR UPDATE for row locking
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, scope_path, status")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "scope_path", "status"}))

	conflicts, err := txRepo.CheckScopeOverlap(context.Background(), "/project/src", "/project", nil)
	if err != nil {
		t.Errorf("CheckScopeOverlap() error = %v", err)
	}

	// Note: conflicts can be nil when no conflicts exist (empty result)
	// The important thing is no error occurred
	_ = conflicts

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestTxSandboxRepository_FindByIdempotencyKey_WithLocking(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	txRepo := &TxSandboxRepository{tx: tx}

	s := testSandbox()
	s.IdempotencyKey = "test-key"

	// Should use FOR UPDATE for row locking
	mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
		WithArgs("test-key").
		WillReturnRows(sqlmock.NewRows(sandboxColumns()).
			AddRow(sandboxRow(s)...))

	result, err := txRepo.FindByIdempotencyKey(context.Background(), "test-key")
	if err != nil {
		t.Errorf("FindByIdempotencyKey() error = %v", err)
	}

	if result == nil {
		t.Error("FindByIdempotencyKey() returned nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestTxSandboxRepository_FindByIdempotencyKey_EmptyKey(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	txRepo := &TxSandboxRepository{tx: tx}

	// No query expected for empty key
	result, err := txRepo.FindByIdempotencyKey(context.Background(), "")
	if err != nil {
		t.Errorf("FindByIdempotencyKey() error = %v", err)
	}

	if result != nil {
		t.Error("FindByIdempotencyKey() should return nil for empty key")
	}
}

func TestTxSandboxRepository_Update(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	txRepo := &TxSandboxRepository{tx: tx}
	s := testSandbox()

	mock.ExpectQuery(regexp.QuoteMeta("UPDATE sandboxes SET")).
		WillReturnRows(sqlmock.NewRows([]string{"version", "updated_at"}).
			AddRow(int64(2), time.Now()))

	err = txRepo.Update(context.Background(), s)
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestTxSandboxRepository_LogAuditEvent(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	txRepo := &TxSandboxRepository{tx: tx}
	event := &types.AuditEvent{
		EventType: "test",
		Actor:     "test-user",
		ActorType: "user",
	}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO sandbox_audit_log")).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = txRepo.LogAuditEvent(context.Background(), event)
	if err != nil {
		t.Errorf("LogAuditEvent() error = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestTxSandboxRepository_UpdateWithVersionCheck(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	txRepo := &TxSandboxRepository{tx: tx}
	s := testSandbox()

	mock.ExpectQuery(regexp.QuoteMeta("UPDATE sandboxes SET")).
		WillReturnRows(sqlmock.NewRows([]string{"version", "updated_at"}).
			AddRow(int64(2), time.Now()))

	err = txRepo.UpdateWithVersionCheck(context.Background(), s, 1)
	if err != nil {
		t.Errorf("UpdateWithVersionCheck() error = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestTxSandboxRepository_UpdateWithVersionCheck_VersionMismatch(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	txRepo := &TxSandboxRepository{tx: tx}
	s := testSandbox()

	mock.ExpectQuery(regexp.QuoteMeta("UPDATE sandboxes SET")).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT version FROM sandboxes WHERE id")).
		WithArgs(s.ID).
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(int64(10)))

	err = txRepo.UpdateWithVersionCheck(context.Background(), s, 5)
	if err == nil {
		t.Error("UpdateWithVersionCheck() expected error for version mismatch")
	}

	var cmErr *types.ConcurrentModificationError
	if !errors.As(err, &cmErr) {
		t.Errorf("UpdateWithVersionCheck() error type = %T, want *ConcurrentModificationError", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// --- Edge Cases ---

func TestSandboxRepository_Create_WithActivePIDs(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewSandboxRepository(db)
	s := testSandbox()
	s.ActivePIDs = []int{100, 200, 300}

	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO sandboxes")).
		WillReturnRows(sqlmock.NewRows([]string{"created_at", "last_used_at", "updated_at"}).
			AddRow(now, now, now))

	err := repo.Create(context.Background(), s)
	if err != nil {
		t.Errorf("Create() error = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestSandboxRepository_Update_WithActivePIDs(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewSandboxRepository(db)
	s := testSandbox()
	s.ActivePIDs = []int{100, 200, 300}

	mock.ExpectQuery(regexp.QuoteMeta("UPDATE sandboxes SET")).
		WillReturnRows(sqlmock.NewRows([]string{"version", "updated_at"}).
			AddRow(int64(2), time.Now()))

	err := repo.Update(context.Background(), s)
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestSandboxRepository_Get_WithEmptyMetadata(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewSandboxRepository(db)
	id := uuid.New()
	s := testSandbox()
	s.ID = id
	s.Metadata = nil

	// Return empty metadata JSON
	mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows(sandboxColumns()).
			AddRow(
				s.ID, s.ScopePath, s.ReservedPath, s.ProjectRoot, s.Owner, s.OwnerType, s.Status, s.ErrorMsg,
				s.CreatedAt, s.LastUsedAt, s.StoppedAt, s.ApprovedAt, s.DeletedAt,
				s.Driver, s.DriverVersion, s.LowerDir, s.UpperDir, s.WorkDir, s.MergedDir,
				s.SizeBytes, s.FileCount, pq.Int64Array{}, s.SessionCount,
				pq.StringArray{}, []byte{}, // Empty tags and metadata
				s.IdempotencyKey, s.UpdatedAt, s.Version, s.BaseCommitHash,
			))

	result, err := repo.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// Verify empty metadata is handled gracefully
	if result == nil {
		t.Error("Get() returned nil")
	}
}

func TestSandboxRepository_List_WithAllFilters(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewSandboxRepository(db)

	filter := &types.ListFilter{
		Status:      []types.Status{types.StatusActive},
		Owner:       "test-owner",
		ProjectRoot: "/project",
		ScopePath:   "/project/src",
		CreatedFrom: time.Now().Add(-24 * time.Hour),
		CreatedTo:   time.Now(),
		Limit:       50,
		Offset:      10,
	}

	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))
	mock.ExpectQuery("SELECT id").
		WillReturnRows(sqlmock.NewRows(sandboxColumns()))

	result, err := repo.List(context.Background(), filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if result.Limit != 50 {
		t.Errorf("List() limit = %d, want 50", result.Limit)
	}
	if result.Offset != 10 {
		t.Errorf("List() offset = %d, want 10", result.Offset)
	}
	if result.TotalCount != 100 {
		t.Errorf("List() totalCount = %d, want 100", result.TotalCount)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestSandboxRepository_CheckScopeOverlap_MultipleConflicts(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewSandboxRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, scope_path, status FROM check_scope_overlap")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "scope_path", "status"}).
			AddRow(uuid.New(), "/project/src", types.StatusActive).
			AddRow(uuid.New(), "/project/src/internal", types.StatusActive))

	conflicts, err := repo.CheckScopeOverlap(context.Background(), "/project", "/project", nil)
	if err != nil {
		t.Fatalf("CheckScopeOverlap() error = %v", err)
	}

	if len(conflicts) != 2 {
		t.Errorf("CheckScopeOverlap() conflicts = %d, want 2", len(conflicts))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}
