package main

import (
	"encoding/json"
	"time"
)

// Plugin represents a graph plugin with its capabilities
type Plugin struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Category    string                 `json:"category"`
	Description string                 `json:"description"`
	Formats     []string               `json:"formats"`
	Enabled     bool                   `json:"enabled"`
	Priority    int                    `json:"priority"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// GraphPermissions represents access control for a graph
type GraphPermissions struct {
	Public       bool     `json:"public"`                  // Public read access
	AllowedUsers []string `json:"allowed_users,omitempty"` // User IDs with access
	Editors      []string `json:"editors,omitempty"`       // User IDs who can edit
}

// Graph represents a graph document
type Graph struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Data        json.RawMessage        `json:"data"`
	Metadata    map[string]interface{} `json:"metadata"`
	Version     int                    `json:"version"`
	CreatedBy   string                 `json:"created_by"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Tags        []string               `json:"tags"`
	Permissions *GraphPermissions      `json:"permissions,omitempty"`
}

// GraphVersion represents a version of a graph
type GraphVersion struct {
	ID                string          `json:"id"`
	GraphID           string          `json:"graph_id"`
	VersionNumber     int             `json:"version_number"`
	Data              json.RawMessage `json:"data"`
	ChangeDescription string          `json:"change_description"`
	CreatedBy         string          `json:"created_by"`
	CreatedAt         time.Time       `json:"created_at"`
}

// CreateGraphRequest represents a request to create a new graph
type CreateGraphRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Type        string                 `json:"type" binding:"required"`
	Description string                 `json:"description"`
	Data        json.RawMessage        `json:"data"`
	Metadata    map[string]interface{} `json:"metadata"`
	Tags        []string               `json:"tags"`
}

// UpdateGraphRequest represents a request to update a graph
type UpdateGraphRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Data        json.RawMessage        `json:"data"`
	Metadata    map[string]interface{} `json:"metadata"`
	Tags        []string               `json:"tags"`
}

// ConversionRequest represents a graph format conversion request
type ConversionRequest struct {
	TargetFormat string                 `json:"target_format" binding:"required"`
	Options      map[string]interface{} `json:"options,omitempty"`
}

// ValidationResult represents the result of graph validation
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// RenderRequest represents a graph rendering request
type RenderRequest struct {
	Format  string                 `json:"format"` // svg, png, html
	Options map[string]interface{} `json:"options,omitempty"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status        string `json:"status"`
	Version       string `json:"version"`
	PluginsLoaded int    `json:"plugins_loaded"`
	PluginsActive int    `json:"plugins_active"`
	Timestamp     int64  `json:"timestamp"`
	Error         string `json:"error,omitempty"`
	Details       string `json:"details,omitempty"`
}

// ListResponse represents a paginated list response
type ListResponse struct {
	Data   interface{} `json:"data"`
	Total  int         `json:"total"`
	Limit  string      `json:"limit,omitempty"`
	Offset string      `json:"offset,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// GraphTemplate represents a reusable graph template
type GraphTemplate struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Description  string                 `json:"description"`
	TemplateData json.RawMessage        `json:"template_data"`
	Category     string                 `json:"category"`
	Metadata     map[string]interface{} `json:"metadata"`
	UsageCount   int                    `json:"usage_count"`
	CreatedBy    string                 `json:"created_by"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// GraphRelationship represents a relationship between graphs
type GraphRelationship struct {
	ID               string                 `json:"id"`
	SourceGraphID    string                 `json:"source_graph_id"`
	TargetGraphID    string                 `json:"target_graph_id"`
	RelationshipType string                 `json:"relationship_type"` // parent, child, related, derived_from
	Metadata         map[string]interface{} `json:"metadata"`
	CreatedAt        time.Time              `json:"created_at"`
}

// GraphConversion represents a format conversion record
type GraphConversion struct {
	ID              string                 `json:"id"`
	SourceGraphID   string                 `json:"source_graph_id"`
	TargetGraphID   string                 `json:"target_graph_id"`
	SourceFormat    string                 `json:"source_format"`
	TargetFormat    string                 `json:"target_format"`
	ConversionRules map[string]interface{} `json:"conversion_rules"`
	Status          string                 `json:"status"` // pending, processing, completed, failed
	ErrorMessage    string                 `json:"error_message,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
}

// GraphCollaboration represents a collaborative editing session
type GraphCollaboration struct {
	ID        string                 `json:"id"`
	GraphID   string                 `json:"graph_id"`
	UserID    string                 `json:"user_id"`
	SessionID string                 `json:"session_id"`
	Action    string                 `json:"action"` // view, edit, comment
	Changes   map[string]interface{} `json:"changes"`
	Timestamp time.Time              `json:"timestamp"`
}

// GraphAnalytics represents usage and performance metrics
type GraphAnalytics struct {
	ID         string                 `json:"id"`
	GraphID    string                 `json:"graph_id"`
	PluginID   string                 `json:"plugin_id"`
	EventType  string                 `json:"event_type"` // created, viewed, edited, converted, rendered
	UserID     string                 `json:"user_id"`
	Metadata   map[string]interface{} `json:"metadata"`
	DurationMs int                    `json:"duration_ms"`
	Timestamp  time.Time              `json:"timestamp"`
}

// CreateTemplateRequest represents a request to create a graph template
type CreateTemplateRequest struct {
	Name         string                 `json:"name" binding:"required"`
	Type         string                 `json:"type" binding:"required"`
	Description  string                 `json:"description"`
	TemplateData json.RawMessage        `json:"template_data" binding:"required"`
	Category     string                 `json:"category"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// UpdateTemplateRequest represents a request to update a graph template
type UpdateTemplateRequest struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	TemplateData json.RawMessage        `json:"template_data"`
	Category     string                 `json:"category"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// ConversionMatrixResponse represents the conversion capabilities matrix
type ConversionMatrixResponse struct {
	Matrix      map[string][]string                 `json:"matrix"`      // from -> [to]
	Conversions map[string][]map[string]interface{} `json:"conversions"` // detailed conversion info
	TotalPaths  int                                 `json:"total_paths"`
}

// GraphStatsResponse represents graph statistics
// DashboardStatsResponse represents dashboard statistics
type DashboardStatsResponse struct {
	TotalGraphs      int `json:"totalGraphs"`
	ConversionsToday int `json:"conversionsToday"`
	ActiveUsers      int `json:"activeUsers"`
}

type GraphStatsResponse struct {
	TotalGraphs      int                      `json:"total_graphs"`
	GraphsByType     map[string]int           `json:"graphs_by_type"`
	RecentActivity   []map[string]interface{} `json:"recent_activity"`
	PopularTemplates []GraphTemplate          `json:"popular_templates"`
	ConversionStats  map[string]int           `json:"conversion_stats"`
	ActiveUsers      int                      `json:"active_users"`
	StorageUsed      int64                    `json:"storage_used_bytes"`
}

// SearchGraphsRequest represents a graph search request
type SearchGraphsRequest struct {
	Query    string   `json:"query"`
	Types    []string `json:"types"`
	Tags     []string `json:"tags"`
	Category string   `json:"category"`
	Limit    int      `json:"limit"`
	Offset   int      `json:"offset"`
}

// BulkOperationRequest represents a bulk operation on multiple graphs
type BulkOperationRequest struct {
	GraphIDs  []string               `json:"graph_ids" binding:"required"`
	Operation string                 `json:"operation" binding:"required"` // delete, convert, tag, move
	Options   map[string]interface{} `json:"options"`
}

// BulkOperationResponse represents the result of a bulk operation
type BulkOperationResponse struct {
	Success    []string `json:"success"`
	Failed     []string `json:"failed"`
	TotalCount int      `json:"total_count"`
	Errors     []string `json:"errors,omitempty"`
}

// ExportRequest represents a graph export request
type ExportRequest struct {
	Format  string                 `json:"format" binding:"required"` // graphml, gexf, json, csv
	Options map[string]interface{} `json:"options,omitempty"`
}

// ExportResponse represents a graph export response
type ExportResponse struct {
	Format   string `json:"format"`
	Filename string `json:"filename"`
	Content  string `json:"content"` // Base64 encoded for binary formats
	MimeType string `json:"mime_type"`
}
