package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"scenario-authenticator/db"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID           string                 `json:"id"`
	UserID       *string                `json:"user_id,omitempty"`
	Action       string                 `json:"action"`
	ResourceType *string                `json:"resource_type,omitempty"`
	ResourceID   *string                `json:"resource_id,omitempty"`
	IPAddress    *string                `json:"ip_address,omitempty"`
	UserAgent    *string                `json:"user_agent,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Success      bool                   `json:"success"`
	ErrorMessage *string                `json:"error_message,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// GetAuditLogsHandler retrieves audit logs with optional filters
// GET /api/v1/audit-logs?limit=50&offset=0&user_id=xxx&action=xxx
func GetAuditLogsHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	userID := r.URL.Query().Get("user_id")
	action := r.URL.Query().Get("action")

	// Default pagination
	limit := 50
	offset := 0

	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
			limit = parsedLimit
		}
	}

	if offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Build query
	query := `
		SELECT id, user_id, action, resource_type, resource_id,
		       ip_address, user_agent, metadata, success, error_message, created_at
		FROM audit_logs
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 1

	if userID != "" {
		query += ` AND user_id = $` + strconv.Itoa(argCount)
		args = append(args, userID)
		argCount++
	}

	if action != "" {
		query += ` AND action = $` + strconv.Itoa(argCount)
		args = append(args, action)
		argCount++
	}

	query += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(argCount) + ` OFFSET $` + strconv.Itoa(argCount+1)
	args = append(args, limit, offset)

	// Execute query
	rows, err := db.DB.Query(query, args...)
	if err != nil {
		log.Printf("Failed to query audit logs: %v", err)
		http.Error(w, `{"error":"Failed to retrieve audit logs"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Parse results
	logs := []AuditLog{}
	for rows.Next() {
		var auditLog AuditLog
		var metadataJSON []byte

		err := rows.Scan(
			&auditLog.ID,
			&auditLog.UserID,
			&auditLog.Action,
			&auditLog.ResourceType,
			&auditLog.ResourceID,
			&auditLog.IPAddress,
			&auditLog.UserAgent,
			&metadataJSON,
			&auditLog.Success,
			&auditLog.ErrorMessage,
			&auditLog.CreatedAt,
		)
		if err != nil {
			log.Printf("Failed to scan audit log row: %v", err)
			continue
		}

		// Parse metadata JSON
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &auditLog.Metadata); err != nil {
				log.Printf("Failed to parse metadata JSON: %v", err)
				auditLog.Metadata = make(map[string]interface{})
			}
		}

		logs = append(logs, auditLog)
	}

	// Get total count for pagination info
	countQuery := `SELECT COUNT(*) FROM audit_logs WHERE 1=1`
	countArgs := []interface{}{}
	argIdx := 1

	if userID != "" {
		countQuery += ` AND user_id = $` + strconv.Itoa(argIdx)
		countArgs = append(countArgs, userID)
		argIdx++
	}

	if action != "" {
		countQuery += ` AND action = $` + strconv.Itoa(argIdx)
		countArgs = append(countArgs, action)
	}

	var totalCount int
	err = db.DB.QueryRow(countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		log.Printf("Failed to count audit logs: %v", err)
		totalCount = len(logs) // Fallback to returned count
	}

	// Build response
	response := map[string]interface{}{
		"logs":   logs,
		"total":  totalCount,
		"limit":  limit,
		"offset": offset,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAuditLogHandler retrieves a single audit log by ID
// GET /api/v1/audit-logs/{id}
func GetAuditLogHandler(w http.ResponseWriter, r *http.Request) {
	auditLogID := chi.URLParam(r, "id")
	if auditLogID == "" {
		http.Error(w, `{"error":"Audit log ID is required"}`, http.StatusBadRequest)
		return
	}

	var auditLog AuditLog
	var metadataJSON []byte

	err := db.DB.QueryRow(`
		SELECT id, user_id, action, resource_type, resource_id,
		       ip_address, user_agent, metadata, success, error_message, created_at
		FROM audit_logs
		WHERE id = $1
	`, auditLogID).Scan(
		&auditLog.ID,
		&auditLog.UserID,
		&auditLog.Action,
		&auditLog.ResourceType,
		&auditLog.ResourceID,
		&auditLog.IPAddress,
		&auditLog.UserAgent,
		&metadataJSON,
		&auditLog.Success,
		&auditLog.ErrorMessage,
		&auditLog.CreatedAt,
	)

	if err != nil {
		log.Printf("Failed to retrieve audit log: %v", err)
		http.Error(w, `{"error":"Audit log not found"}`, http.StatusNotFound)
		return
	}

	// Parse metadata JSON
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &auditLog.Metadata); err != nil {
			log.Printf("Failed to parse metadata JSON: %v", err)
			auditLog.Metadata = make(map[string]interface{})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(auditLog)
}
