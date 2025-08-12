package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"metareasoning-api/internal/pkg/errors"
)

// ErrorHandler defines the interface for handling HTTP errors
type ErrorHandler interface {
	HandleError(w http.ResponseWriter, r *http.Request, err error)
	WrapErrorHandler(handler func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc
}

// ErrorHandlerFunc represents a handler function that can return an error
type ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request) error

// Response represents a standard API response structure
type Response struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorInfo  `json:"error,omitempty"`
	Metadata  *Metadata   `json:"metadata,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// ErrorInfo represents error information in responses
type ErrorInfo struct {
	Type    string                 `json:"type"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
	Code    string                 `json:"code,omitempty"`
}

// Metadata represents response metadata
type Metadata struct {
	RequestID     string            `json:"request_id,omitempty"`
	Version       string            `json:"version,omitempty"`
	ExecutionTime string            `json:"execution_time,omitempty"`
	Links         map[string]string `json:"links,omitempty"`
}

// WriteJSON writes a JSON response to the response writer
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		return errors.NewInternalError("failed to encode JSON response", err)
	}
	
	return nil
}

// WriteSuccessResponse writes a standardized success response
func WriteSuccessResponse(w http.ResponseWriter, statusCode int, data interface{}, metadata *Metadata) error {
	response := Response{
		Success:   true,
		Data:      data,
		Metadata:  metadata,
		Timestamp: time.Now(),
	}
	
	return WriteJSON(w, statusCode, response)
}

// WriteErrorResponse writes a standardized error response
func WriteErrorResponse(w http.ResponseWriter, err error, requestID string) error {
	var statusCode int
	var errorInfo *ErrorInfo
	
	if appErr, ok := errors.AsAppError(err); ok {
		statusCode = appErr.StatusCode
		errorInfo = &ErrorInfo{
			Type:    string(appErr.Type),
			Message: appErr.Message,
			Details: appErr.Details,
		}
	} else {
		statusCode = http.StatusInternalServerError
		errorInfo = &ErrorInfo{
			Type:    string(errors.InternalError),
			Message: "An internal error occurred",
		}
	}
	
	response := Response{
		Success:   false,
		Error:     errorInfo,
		Metadata:  &Metadata{RequestID: requestID},
		Timestamp: time.Now(),
	}
	
	return WriteJSON(w, statusCode, response)
}

// PaginationParams represents pagination parameters from request
type PaginationParams struct {
	Page     int    `json:"page" query:"page" default:"1" validate:"min=1"`
	PageSize int    `json:"page_size" query:"page_size" default:"20" validate:"min=1,max=100"`
	SortBy   string `json:"sort_by" query:"sort_by"`
	SortDir  string `json:"sort_dir" query:"sort_dir" validate:"omitempty,oneof=asc desc ASC DESC"`
}

// NormalizeSortDirection normalizes sort direction to uppercase
func (p *PaginationParams) NormalizeSortDirection() {
	switch p.SortDir {
	case "asc", "ASC", "":
		p.SortDir = "ASC"
	case "desc", "DESC":
		p.SortDir = "DESC"
	default:
		p.SortDir = "ASC" // default
	}
}

// FilterParams represents common filter parameters
type FilterParams struct {
	ActiveOnly bool   `json:"active_only" query:"active" default:"true"`
	Platform   string `json:"platform" query:"platform"`
	Type       string `json:"type" query:"type"`
	CreatedBy  string `json:"created_by" query:"created_by"`
	Tags       string `json:"tags" query:"tags"` // comma-separated
}

// ParseTags parses comma-separated tags string into slice
func (f *FilterParams) ParseTags() []string {
	if f.Tags == "" {
		return nil
	}
	
	tags := make([]string, 0)
	for _, tag := range splitAndTrim(f.Tags, ",") {
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}

// RequestContext represents common request context information
type RequestContext struct {
	RequestID   string
	UserID      string
	UserAgent   string
	RemoteAddr  string
	StartTime   time.Time
	TraceID     string
}

// NewRequestContext creates a new request context from HTTP request
func NewRequestContext(r *http.Request) *RequestContext {
	return &RequestContext{
		RequestID:  getOrGenerateRequestID(r),
		UserAgent:  r.UserAgent(),
		RemoteAddr: r.RemoteAddr,
		StartTime:  time.Now(),
		TraceID:    r.Header.Get("X-Trace-ID"),
	}
}

// Duration returns the duration since request start
func (rc *RequestContext) Duration() time.Duration {
	return time.Since(rc.StartTime)
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status     string            `json:"status"`
	Version    string            `json:"version"`
	Timestamp  time.Time         `json:"timestamp"`
	Services   map[string]bool   `json:"services,omitempty"`
	Database   bool              `json:"database"`
	Metrics    map[string]interface{} `json:"metrics,omitempty"`
	Uptime     string            `json:"uptime,omitempty"`
}

// ValidationErrorResponse represents validation error details
type ValidationErrorResponse struct {
	Field   string      `json:"field"`
	Value   interface{} `json:"value"`
	Message string      `json:"message"`
	Code    string      `json:"code"`
}

// Helper functions

// getOrGenerateRequestID extracts request ID from headers or generates one
func getOrGenerateRequestID(r *http.Request) string {
	if id := r.Header.Get("X-Request-ID"); id != "" {
		return id
	}
	if id := r.Header.Get("Request-ID"); id != "" {
		return id
	}
	// Generate a simple ID (in real implementation, use UUID or similar)
	return "req-" + time.Now().Format("20060102150405")
}

// splitAndTrim splits string by delimiter and trims whitespace
func splitAndTrim(s, delimiter string) []string {
	if s == "" {
		return nil
	}
	
	parts := make([]string, 0)
	start := 0
	
	for i := 0; i < len(s); i++ {
		if i == len(s)-1 || s[i:i+len(delimiter)] == delimiter {
			end := i
			if i == len(s)-1 {
				end = len(s)
			}
			
			part := trimWhitespace(s[start:end])
			if part != "" {
				parts = append(parts, part)
			}
			
			start = i + len(delimiter)
			i += len(delimiter) - 1
		}
	}
	
	return parts
}

// trimWhitespace trims whitespace from string
func trimWhitespace(s string) string {
	start := 0
	end := len(s)
	
	// Trim leading whitespace
	for start < len(s) && isWhitespace(s[start]) {
		start++
	}
	
	// Trim trailing whitespace
	for end > start && isWhitespace(s[end-1]) {
		end--
	}
	
	return s[start:end]
}

// isWhitespace checks if character is whitespace
func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// CreateLinks creates HATEOAS links for resources
func CreateLinks(baseURL string, resourceType string, resourceID string) map[string]string {
	links := make(map[string]string)
	
	if resourceID != "" {
		links["self"] = baseURL + "/" + resourceType + "/" + resourceID
		links["edit"] = baseURL + "/" + resourceType + "/" + resourceID
		links["delete"] = baseURL + "/" + resourceType + "/" + resourceID
		
		// Resource-specific links
		if resourceType == "workflows" {
			links["execute"] = baseURL + "/" + resourceType + "/" + resourceID + "/execute"
			links["metrics"] = baseURL + "/" + resourceType + "/" + resourceID + "/metrics"
			links["history"] = baseURL + "/" + resourceType + "/" + resourceID + "/history"
			links["clone"] = baseURL + "/" + resourceType + "/" + resourceID + "/clone"
		}
	}
	
	links["collection"] = baseURL + "/" + resourceType
	return links
}

// Constants for response codes
const (
	CodeValidationFailed = "VALIDATION_FAILED"
	CodeResourceNotFound = "RESOURCE_NOT_FOUND"
	CodeDuplicateResource = "DUPLICATE_RESOURCE"
	CodeInternalError = "INTERNAL_ERROR"
	CodeExternalServiceError = "EXTERNAL_SERVICE_ERROR"
)