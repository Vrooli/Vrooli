package database

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Common errors
var (
	ErrNotFound = errors.New("not found")
)

// JSONMap represents a JSON object stored in the database
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

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, j)
}

// NullableString represents a nullable string in the database
type NullableString struct {
	sql.NullString
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

// Project represents a project containing related workflows
type Project struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description,omitempty" db:"description"`
	FolderPath  string    `json:"folder_path" db:"folder_path"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// WorkflowFolder represents a workflow organization folder
type WorkflowFolder struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	Path        string     `json:"path" db:"path"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty" db:"parent_id"`
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// Workflow represents a browser automation workflow
type Workflow struct {
	ID                    uuid.UUID      `json:"id" db:"id"`
	ProjectID             *uuid.UUID     `json:"project_id,omitempty" db:"project_id"`
	Name                  string         `json:"name" db:"name"`
	FolderPath            string         `json:"folder_path" db:"folder_path"`
	FlowDefinition        JSONMap        `json:"flow_definition" db:"flow_definition"`
	Description           string         `json:"description,omitempty" db:"description"`
	Tags                  pq.StringArray `json:"tags" db:"tags"`
	Version               int            `json:"version" db:"version"`
	IsTemplate            bool           `json:"is_template" db:"is_template"`
	CreatedBy             string         `json:"created_by,omitempty" db:"created_by"`
	LastChangeSource      string         `json:"last_change_source,omitempty" db:"last_change_source"`
	LastChangeDescription string         `json:"last_change_description,omitempty" db:"last_change_description"`
	CreatedAt             time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at" db:"updated_at"`
}

// WorkflowVersion represents a version of a workflow
type WorkflowVersion struct {
	ID                uuid.UUID `json:"id" db:"id"`
	WorkflowID        uuid.UUID `json:"workflow_id" db:"workflow_id"`
	Version           int       `json:"version" db:"version"`
	FlowDefinition    JSONMap   `json:"flow_definition" db:"flow_definition"`
	ChangeDescription string    `json:"change_description,omitempty" db:"change_description"`
	CreatedBy         string    `json:"created_by,omitempty" db:"created_by"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

// Execution represents a workflow execution
type Execution struct {
	ID              uuid.UUID      `json:"id" db:"id"`
	WorkflowID      uuid.UUID      `json:"workflow_id" db:"workflow_id"`
	WorkflowVersion int            `json:"workflow_version,omitempty" db:"workflow_version"`
	Status          string         `json:"status" db:"status"`
	TriggerType     string         `json:"trigger_type" db:"trigger_type"`
	TriggerMetadata JSONMap        `json:"trigger_metadata,omitempty" db:"trigger_metadata"`
	Parameters      JSONMap        `json:"parameters,omitempty" db:"parameters"`
	StartedAt       time.Time      `json:"started_at" db:"started_at"`
	CompletedAt     *time.Time     `json:"completed_at,omitempty" db:"completed_at"`
	Error           NullableString `json:"error,omitempty" db:"error"`
	Result          JSONMap        `json:"result,omitempty" db:"result"`
	Progress        int            `json:"progress" db:"progress"`
	CurrentStep     string         `json:"current_step,omitempty" db:"current_step"`
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at" db:"updated_at"`
}

// ExecutionLog represents a log entry for a workflow execution
type ExecutionLog struct {
	ID          uuid.UUID `json:"id" db:"id"`
	ExecutionID uuid.UUID `json:"execution_id" db:"execution_id"`
	Timestamp   time.Time `json:"timestamp" db:"timestamp"`
	Level       string    `json:"level" db:"level"`
	StepName    string    `json:"step_name,omitempty" db:"step_name"`
	Message     string    `json:"message" db:"message"`
	Metadata    JSONMap   `json:"metadata,omitempty" db:"metadata"`
}

// Screenshot represents a captured screenshot during execution
type Screenshot struct {
	ID           uuid.UUID `json:"id" db:"id"`
	ExecutionID  uuid.UUID `json:"execution_id" db:"execution_id"`
	StepName     string    `json:"step_name" db:"step_name"`
	Timestamp    time.Time `json:"timestamp" db:"timestamp"`
	StorageURL   string    `json:"storage_url" db:"storage_url"`
	ThumbnailURL string    `json:"thumbnail_url,omitempty" db:"thumbnail_url"`
	Width        int       `json:"width,omitempty" db:"width"`
	Height       int       `json:"height,omitempty" db:"height"`
	SizeBytes    int64     `json:"size_bytes,omitempty" db:"size_bytes"`
	Metadata     JSONMap   `json:"metadata,omitempty" db:"metadata"`
}

// ExtractedData represents data extracted during workflow execution
type ExtractedData struct {
	ID          uuid.UUID `json:"id" db:"id"`
	ExecutionID uuid.UUID `json:"execution_id" db:"execution_id"`
	StepName    string    `json:"step_name" db:"step_name"`
	Timestamp   time.Time `json:"timestamp" db:"timestamp"`
	DataKey     string    `json:"data_key" db:"data_key"`
	DataValue   JSONMap   `json:"data_value" db:"data_value"`
	DataType    string    `json:"data_type,omitempty" db:"data_type"`
	Metadata    JSONMap   `json:"metadata,omitempty" db:"metadata"`
}

// ExecutionStep represents the normalized record for each executed workflow step.
type ExecutionStep struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	ExecutionID uuid.UUID  `json:"execution_id" db:"execution_id"`
	StepIndex   int        `json:"step_index" db:"step_index"`
	NodeID      string     `json:"node_id" db:"node_id"`
	StepType    string     `json:"step_type" db:"step_type"`
	Status      string     `json:"status" db:"status"`
	StartedAt   time.Time  `json:"started_at" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	DurationMs  int        `json:"duration_ms,omitempty" db:"duration_ms"`
	Error       string     `json:"error,omitempty" db:"error"`
	Input       JSONMap    `json:"input,omitempty" db:"input"`
	Output      JSONMap    `json:"output,omitempty" db:"output"`
	Metadata    JSONMap    `json:"metadata,omitempty" db:"metadata"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// ExecutionArtifact stores artifacts (screenshots, telemetry blobs, etc.) linked to a step.
type ExecutionArtifact struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	ExecutionID  uuid.UUID  `json:"execution_id" db:"execution_id"`
	StepID       *uuid.UUID `json:"step_id,omitempty" db:"step_id"`
	StepIndex    *int       `json:"step_index,omitempty" db:"step_index"`
	ArtifactType string     `json:"artifact_type" db:"artifact_type"`
	Label        string     `json:"label,omitempty" db:"label"`
	StorageURL   string     `json:"storage_url,omitempty" db:"storage_url"`
	ThumbnailURL string     `json:"thumbnail_url,omitempty" db:"thumbnail_url"`
	ContentType  string     `json:"content_type,omitempty" db:"content_type"`
	SizeBytes    *int64     `json:"size_bytes,omitempty" db:"size_bytes"`
	Payload      JSONMap    `json:"payload,omitempty" db:"payload"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// WorkflowSchedule represents a scheduled workflow execution
type WorkflowSchedule struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	WorkflowID     uuid.UUID  `json:"workflow_id" db:"workflow_id"`
	Name           string     `json:"name" db:"name"`
	Description    string     `json:"description,omitempty" db:"description"`
	CronExpression string     `json:"cron_expression" db:"cron_expression"`
	Timezone       string     `json:"timezone" db:"timezone"`
	IsActive       bool       `json:"is_active" db:"is_active"`
	Parameters     JSONMap    `json:"parameters,omitempty" db:"parameters"`
	NextRunAt      *time.Time `json:"next_run_at,omitempty" db:"next_run_at"`
	LastRunAt      *time.Time `json:"last_run_at,omitempty" db:"last_run_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// WorkflowTemplate represents a reusable workflow template
type WorkflowTemplate struct {
	ID                uuid.UUID      `json:"id" db:"id"`
	Name              string         `json:"name" db:"name"`
	Category          string         `json:"category" db:"category"`
	Description       string         `json:"description,omitempty" db:"description"`
	FlowDefinition    JSONMap        `json:"flow_definition" db:"flow_definition"`
	Icon              string         `json:"icon,omitempty" db:"icon"`
	ExampleParameters JSONMap        `json:"example_parameters,omitempty" db:"example_parameters"`
	Tags              pq.StringArray `json:"tags" db:"tags"`
	UsageCount        int            `json:"usage_count" db:"usage_count"`
	CreatedAt         time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at" db:"updated_at"`
}

// AIGeneration represents an AI-generated workflow
type AIGeneration struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	WorkflowID       *uuid.UUID `json:"workflow_id,omitempty" db:"workflow_id"`
	Prompt           string     `json:"prompt" db:"prompt"`
	GeneratedFlow    JSONMap    `json:"generated_flow" db:"generated_flow"`
	Model            string     `json:"model,omitempty" db:"model"`
	GenerationTimeMs int        `json:"generation_time_ms,omitempty" db:"generation_time_ms"`
	Success          bool       `json:"success" db:"success"`
	Error            string     `json:"error,omitempty" db:"error"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
}
