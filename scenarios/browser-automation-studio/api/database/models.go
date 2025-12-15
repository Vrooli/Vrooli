// Package database provides database index types and repository operations.
//
// IMPORTANT: This package contains INDEX types, not domain types.
// Domain types are defined as Protocol Buffers in packages/proto.
//
// The database stores minimal index data for queryable operations:
// - projects: id, name, folder_path
// - workflows: id, project_id, name, folder_path, file_path, version
// - executions: id, workflow_id, status, started_at, completed_at, error_message, result_path
// - schedules: id, workflow_id, name, cron, timezone, is_active, next_run_at, last_run_at
// - settings: key, value
//
// Rich domain data lives on disk as protojson and is assembled by higher layers.
package database

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Common errors
var (
	ErrNotFound = errors.New("not found")
)

// ============================================================================
// INDEX TYPES - Stored in database for queryable lookups
// ============================================================================

// ProjectIndex is the database index for a project.
// Use basprojects.Project for domain operations.
type ProjectIndex struct {
	ID         uuid.UUID `json:"id" db:"id"`
	Name       string    `json:"name" db:"name"`
	FolderPath string    `json:"folder_path" db:"folder_path"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// ProjectStats captures computed project metrics (not persisted as domain state).
// These are derived from the index tables and used for UI summaries.
type ProjectStats struct {
	ProjectID      uuid.UUID  `json:"project_id" db:"project_id"`
	WorkflowCount  int        `json:"workflow_count" db:"workflow_count"`
	ExecutionCount int        `json:"execution_count" db:"execution_count"`
	LastExecution  *time.Time `json:"last_execution,omitempty" db:"last_execution"`
}

// WorkflowIndex is the database index for a workflow.
// Use basworkflows.WorkflowDefinitionV2 for workflow definitions.
type WorkflowIndex struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	ProjectID  *uuid.UUID `json:"project_id,omitempty" db:"project_id"`
	Name       string     `json:"name" db:"name"`
	FolderPath string     `json:"folder_path" db:"folder_path"`
	FilePath   string     `json:"file_path,omitempty" db:"file_path"`
	Version    int        `json:"version" db:"version"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
}

// ExecutionIndex is the database index for an execution.
// Use basexecution.Execution for full execution details.
type ExecutionIndex struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	WorkflowID   uuid.UUID  `json:"workflow_id" db:"workflow_id"`
	Status       string     `json:"status" db:"status"`
	StartedAt    time.Time  `json:"started_at" db:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	ErrorMessage string     `json:"error_message,omitempty" db:"error_message"`
	ResultPath   string     `json:"result_path,omitempty" db:"result_path"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// Execution status constants
const (
	ExecutionStatusPending   = "pending"
	ExecutionStatusRunning   = "running"
	ExecutionStatusCompleted = "completed"
	ExecutionStatusFailed    = "failed"
)

// ScheduleIndex is the database index for a workflow schedule.
type ScheduleIndex struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	WorkflowID     uuid.UUID  `json:"workflow_id" db:"workflow_id"`
	Name           string     `json:"name" db:"name"`
	CronExpression string     `json:"cron_expression" db:"cron_expression"`
	Timezone       string     `json:"timezone" db:"timezone"`
	IsActive       bool       `json:"is_active" db:"is_active"`
	ParametersJSON string     `json:"parameters_json,omitempty" db:"parameters_json"`
	NextRunAt      *time.Time `json:"next_run_at,omitempty" db:"next_run_at"`
	LastRunAt      *time.Time `json:"last_run_at,omitempty" db:"last_run_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// GetParameters parses the JSON parameters into a map
func (s *ScheduleIndex) GetParameters() (map[string]any, error) {
	if s.ParametersJSON == "" || s.ParametersJSON == "{}" {
		return make(map[string]any), nil
	}
	var params map[string]any
	if err := json.Unmarshal([]byte(s.ParametersJSON), &params); err != nil {
		return nil, fmt.Errorf("failed to parse schedule parameters: %w", err)
	}
	return params, nil
}

// SetParameters serializes a map to JSON parameters
func (s *ScheduleIndex) SetParameters(params map[string]any) error {
	if params == nil {
		s.ParametersJSON = "{}"
		return nil
	}
	data, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to serialize schedule parameters: %w", err)
	}
	s.ParametersJSON = string(data)
	return nil
}

// Setting is a key-value pair for user preferences.
type Setting struct {
	Key       string    `json:"key" db:"key"`
	Value     string    `json:"value" db:"value"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ============================================================================
// HELPER TYPES
// ============================================================================

// JSONMap represents a JSON object for marshaling/unmarshaling
type JSONMap map[string]any

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONMap) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, j)
	case string:
		return json.Unmarshal([]byte(v), j)
	case json.RawMessage:
		return json.Unmarshal(v, j)
	default:
		return fmt.Errorf("jsonmap: unsupported scan type %T", value)
	}
}

// NullableString represents a nullable string in the database
type NullableString struct {
	String string
	Valid  bool
}

func (ns NullableString) MarshalJSON() ([]byte, error) {
	if ns.Valid {
		return json.Marshal(ns.String)
	}
	return json.Marshal(nil)
}

func (ns *NullableString) UnmarshalJSON(data []byte) error {
	var s *string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s != nil {
		ns.Valid = true
		ns.String = *s
	} else {
		ns.Valid = false
	}
	return nil
}

// StringArray stores string slices in a database-friendly format
type StringArray []string

func (s StringArray) Value() (driver.Value, error) {
	return json.Marshal([]string(s))
}

func (s *StringArray) Scan(src any) error {
	if src == nil {
		*s = nil
		return nil
	}
	var raw string
	switch v := src.(type) {
	case []byte:
		raw = string(v)
	case string:
		raw = v
	default:
		return fmt.Errorf("StringArray: unsupported scan type %T", src)
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return fmt.Errorf("StringArray: scan failed: %w", err)
	}
	*s = out
	return nil
}
