package support

import (
	"encoding/json"
	"time"
)

// Schema mirrors the shape returned by GET /api/v1/schemas and /api/v1/schemas/:id.
type Schema struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description,omitempty"`
	SchemaDefinition map[string]interface{} `json:"schema_definition,omitempty"`
	ExampleData      map[string]interface{} `json:"example_data,omitempty"`
	Version          int                    `json:"version,omitempty"`
	IsActive         bool                   `json:"is_active,omitempty"`
	CreatedAt        *time.Time             `json:"created_at,omitempty"`
	UpdatedAt        *time.Time             `json:"updated_at,omitempty"`
	CreatedBy        string                 `json:"created_by,omitempty"`
	UsageCount       int                    `json:"usage_count,omitempty"`
	AvgConfidence    float64                `json:"avg_confidence,omitempty"`
}

// SchemaListResponse wraps the GET /api/v1/schemas response envelope.
type SchemaListResponse struct {
	Schemas []Schema `json:"schemas"`
	Count   int      `json:"count"`
}

// SchemaMutationResponse wraps the POST/PUT/DELETE /api/v1/schemas response.
type SchemaMutationResponse struct {
	ID         string     `json:"id"`
	Name       string     `json:"name,omitempty"`
	Status     string     `json:"status,omitempty"`
	CreatedAt  *time.Time `json:"created_at,omitempty"`
	TemplateID string     `json:"template_id,omitempty"`
}

// SchemaTemplate mirrors entries returned by /api/v1/schema-templates.
type SchemaTemplate struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Category         string                 `json:"category,omitempty"`
	Description      string                 `json:"description,omitempty"`
	SchemaDefinition map[string]interface{} `json:"schema_definition,omitempty"`
	ExampleData      map[string]interface{} `json:"example_data,omitempty"`
	UsageCount       int                    `json:"usage_count,omitempty"`
	IsPublic         bool                   `json:"is_public,omitempty"`
	CreatedAt        *time.Time             `json:"created_at,omitempty"`
	UpdatedAt        *time.Time             `json:"updated_at,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
}

// SchemaTemplateListResponse wraps /api/v1/schema-templates.
type SchemaTemplateListResponse struct {
	Templates []SchemaTemplate `json:"templates"`
	Count     int              `json:"count"`
}

// ProcessingResponse wraps POST /api/v1/process (single-item mode).
type ProcessingResponse struct {
	ProcessingID    string                 `json:"processing_id"`
	Status          string                 `json:"status"`
	StructuredData  map[string]interface{} `json:"structured_data,omitempty"`
	ConfidenceScore *float64               `json:"confidence_score,omitempty"`
	Errors          []string               `json:"errors,omitempty"`

	// Batch fields are also decoded into the same struct when the server
	// returns a BatchProcessingResponse; callers can inspect BatchID to
	// distinguish batch responses.
	BatchID          string             `json:"batch_id,omitempty"`
	TotalItems       int                `json:"total_items,omitempty"`
	Completed        int                `json:"completed,omitempty"`
	Failed           int                `json:"failed,omitempty"`
	Results          []ProcessingResult `json:"results,omitempty"`
	AvgConfidence    *float64           `json:"avg_confidence,omitempty"`
	ProcessingTimeMs int                `json:"processing_time_ms,omitempty"`
}

// ProcessingResult is a single entry inside a batch processing response.
type ProcessingResult struct {
	ProcessingID    string                 `json:"processing_id"`
	Status          string                 `json:"status"`
	StructuredData  map[string]interface{} `json:"structured_data,omitempty"`
	ConfidenceScore *float64               `json:"confidence_score,omitempty"`
	Error           string                 `json:"error,omitempty"`
}

// ProcessingResultDetail is the shape returned by GET /api/v1/process/:id.
type ProcessingResultDetail struct {
	ID               string                 `json:"id"`
	SchemaID         string                 `json:"schema_id"`
	SourceFileName   string                 `json:"source_file_name,omitempty"`
	SourceFilePath   string                 `json:"source_file_path,omitempty"`
	SourceFileType   string                 `json:"source_file_type,omitempty"`
	SourceFileSize   int64                  `json:"source_file_size,omitempty"`
	RawContent       string                 `json:"raw_content,omitempty"`
	StructuredData   map[string]interface{} `json:"structured_data,omitempty"`
	ConfidenceScore  *float64               `json:"confidence_score,omitempty"`
	ProcessingStatus string                 `json:"processing_status"`
	ErrorMessage     string                 `json:"error_message,omitempty"`
	ProcessingTimeMs *int                   `json:"processing_time_ms,omitempty"`
	CreatedAt        *time.Time             `json:"created_at,omitempty"`
	ProcessedAt      *time.Time             `json:"processed_at,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// ProcessedDataItem is one row in the /api/v1/data/:schema_id listing.
type ProcessedDataItem struct {
	ID               string                 `json:"id"`
	SourceFileName   string                 `json:"source_file_name,omitempty"`
	StructuredData   map[string]interface{} `json:"structured_data,omitempty"`
	ConfidenceScore  *float64               `json:"confidence_score,omitempty"`
	ProcessingStatus string                 `json:"processing_status,omitempty"`
	CreatedAt        *time.Time             `json:"created_at,omitempty"`
	ProcessedAt      *time.Time             `json:"processed_at,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// ProcessedDataResponse wraps the JSON form of /api/v1/data/:schema_id.
type ProcessedDataResponse struct {
	Schema     SchemaSummary       `json:"schema"`
	Data       []ProcessedDataItem `json:"data"`
	Pagination Pagination          `json:"pagination"`
}

// SchemaSummary is the embedded schema descriptor returned alongside processed data.
type SchemaSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Pagination mirrors the pagination block on listing endpoints.
type Pagination struct {
	Limit      int  `json:"limit"`
	Offset     int  `json:"offset"`
	TotalCount int  `json:"total_count"`
	HasMore    bool `json:"has_more"`
}

// Job is one entry in /api/v1/jobs.
type Job struct {
	ID             string                 `json:"id"`
	SchemaID       string                 `json:"schema_id,omitempty"`
	InputType      string                 `json:"input_type,omitempty"`
	InputData      string                 `json:"input_data,omitempty"`
	BatchMode      bool                   `json:"batch_mode,omitempty"`
	TotalItems     int                    `json:"total_items,omitempty"`
	ProcessedItems int                    `json:"processed_items,omitempty"`
	FailedItems    int                    `json:"failed_items,omitempty"`
	Status         string                 `json:"status,omitempty"`
	Priority       json.RawMessage        `json:"priority,omitempty"`
	CreatedAt      *time.Time             `json:"created_at,omitempty"`
	StartedAt      *time.Time             `json:"started_at,omitempty"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	UpdatedAt      *time.Time             `json:"updated_at,omitempty"`
	ErrorDetails   map[string]interface{} `json:"error_details,omitempty"`
	ResultSummary  map[string]interface{} `json:"result_summary,omitempty"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
}

// JobListResponse wraps /api/v1/jobs.
type JobListResponse struct {
	Jobs  []Job `json:"jobs"`
	Count int   `json:"count"`
}
