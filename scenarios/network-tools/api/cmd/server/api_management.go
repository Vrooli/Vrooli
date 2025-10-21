package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// APIDefinition represents an API specification
type APIDefinition struct {
	ID                    string                 `json:"id"`
	Name                  string                 `json:"name"`
	BaseURL               string                 `json:"base_url"`
	Version               string                 `json:"version,omitempty"`
	Specification         string                 `json:"specification,omitempty"`
	SpecDocument          map[string]interface{} `json:"spec_document,omitempty"`
	AuthenticationMethods []string               `json:"authentication_methods,omitempty"`
	RateLimits            map[string]interface{} `json:"rate_limits,omitempty"`
	EndpointsCount        int                    `json:"endpoints_count,omitempty"`
	LastValidated         *time.Time             `json:"last_validated,omitempty"`
	ValidationStatus      string                 `json:"validation_status,omitempty"`
	DocumentationURL      string                 `json:"documentation_url,omitempty"`
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
}

// APITestSuite represents a collection of API tests
type APITestSuite struct {
	ID              string                 `json:"id"`
	APIDefinitionID string                 `json:"api_definition_id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description,omitempty"`
	Tests           []APITestCase          `json:"tests"`
	Config          map[string]interface{} `json:"config,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
}

// APITestCase represents a single API test case
type APITestCase struct {
	Name           string                 `json:"name"`
	Endpoint       string                 `json:"endpoint"`
	Method         string                 `json:"method"`
	Input          map[string]interface{} `json:"input,omitempty"`
	Headers        map[string]string      `json:"headers,omitempty"`
	ExpectedStatus int                    `json:"expected_status"`
	ExpectedSchema map[string]interface{} `json:"expected_schema,omitempty"`
	Assertions     []string               `json:"assertions,omitempty"`
}

// handleListAPIDefinitions lists all registered API definitions
func (s *Server) handleListAPIDefinitions(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		sendError(w, "Database not configured", http.StatusInternalServerError)
		return
	}

	rows, err := s.db.Query(`
		SELECT id, name, base_url, version, specification, endpoints_count,
		       last_validated, validation_status, documentation_url,
		       created_at, updated_at
		FROM api_definitions
		ORDER BY created_at DESC`)

	if err != nil {
		sendError(w, fmt.Sprintf("Failed to query API definitions: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	definitions := make([]APIDefinition, 0)
	for rows.Next() {
		var def APIDefinition
		var version, specification, documentationURL sql.NullString
		var endpointsCount sql.NullInt64
		var lastValidated sql.NullTime
		var validationStatus sql.NullString

		err := rows.Scan(&def.ID, &def.Name, &def.BaseURL, &version, &specification,
			&endpointsCount, &lastValidated, &validationStatus, &documentationURL,
			&def.CreatedAt, &def.UpdatedAt)
		if err != nil {
			continue
		}

		if version.Valid {
			def.Version = version.String
		}
		if specification.Valid {
			def.Specification = specification.String
		}
		if endpointsCount.Valid {
			def.EndpointsCount = int(endpointsCount.Int64)
		}
		if lastValidated.Valid {
			def.LastValidated = &lastValidated.Time
		}
		if validationStatus.Valid {
			def.ValidationStatus = validationStatus.String
		}
		if documentationURL.Valid {
			def.DocumentationURL = documentationURL.String
		}

		definitions = append(definitions, def)
	}

	sendSuccess(w, definitions)
}

// handleGetAPIDefinition retrieves a single API definition by ID
func (s *Server) handleGetAPIDefinition(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		sendError(w, "Database not configured", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	var def APIDefinition
	var version, specification, documentationURL sql.NullString
	var endpointsCount sql.NullInt64
	var lastValidated sql.NullTime
	var validationStatus sql.NullString
	var specDocumentJSON sql.NullString
	var authMethodsJSON, rateLimitsJSON sql.NullString

	err := s.db.QueryRow(`
		SELECT id, name, base_url, version, specification, spec_document,
		       authentication_methods, rate_limits, endpoints_count,
		       last_validated, validation_status, documentation_url,
		       created_at, updated_at
		FROM api_definitions
		WHERE id = $1`, id).Scan(
		&def.ID, &def.Name, &def.BaseURL, &version, &specification, &specDocumentJSON,
		&authMethodsJSON, &rateLimitsJSON, &endpointsCount,
		&lastValidated, &validationStatus, &documentationURL,
		&def.CreatedAt, &def.UpdatedAt)

	if err == sql.ErrNoRows {
		sendError(w, "API definition not found", http.StatusNotFound)
		return
	}
	if err != nil {
		sendError(w, fmt.Sprintf("Failed to query API definition: %v", err), http.StatusInternalServerError)
		return
	}

	// Populate optional fields
	if version.Valid {
		def.Version = version.String
	}
	if specification.Valid {
		def.Specification = specification.String
	}
	if endpointsCount.Valid {
		def.EndpointsCount = int(endpointsCount.Int64)
	}
	if lastValidated.Valid {
		def.LastValidated = &lastValidated.Time
	}
	if validationStatus.Valid {
		def.ValidationStatus = validationStatus.String
	}
	if documentationURL.Valid {
		def.DocumentationURL = documentationURL.String
	}
	if specDocumentJSON.Valid {
		json.Unmarshal([]byte(specDocumentJSON.String), &def.SpecDocument)
	}
	if authMethodsJSON.Valid {
		json.Unmarshal([]byte(authMethodsJSON.String), &def.AuthenticationMethods)
	}
	if rateLimitsJSON.Valid {
		json.Unmarshal([]byte(rateLimitsJSON.String), &def.RateLimits)
	}

	sendSuccess(w, def)
}

// handleCreateAPIDefinition creates a new API definition
func (s *Server) handleCreateAPIDefinition(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		sendError(w, "Database not configured", http.StatusInternalServerError)
		return
	}

	var req struct {
		Name                  string                 `json:"name"`
		BaseURL               string                 `json:"base_url"`
		Version               string                 `json:"version,omitempty"`
		Specification         string                 `json:"specification,omitempty"`
		SpecDocument          map[string]interface{} `json:"spec_document,omitempty"`
		AuthenticationMethods []string               `json:"authentication_methods,omitempty"`
		RateLimits            map[string]interface{} `json:"rate_limits,omitempty"`
		DocumentationURL      string                 `json:"documentation_url,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" || req.BaseURL == "" {
		sendError(w, "Name and base_url are required", http.StatusBadRequest)
		return
	}

	// Generate ID
	id := uuid.New().String()

	// Marshal JSON fields
	specDocJSON, _ := json.Marshal(req.SpecDocument)
	authMethodsJSON, _ := json.Marshal(req.AuthenticationMethods)
	rateLimitsJSON, _ := json.Marshal(req.RateLimits)

	// Count endpoints from spec if available
	endpointsCount := 0
	if req.SpecDocument != nil {
		if paths, ok := req.SpecDocument["paths"].(map[string]interface{}); ok {
			endpointsCount = len(paths)
		}
	}

	_, err := s.db.Exec(`
		INSERT INTO api_definitions (
			id, name, base_url, version, specification, spec_document,
			authentication_methods, rate_limits, endpoints_count, documentation_url,
			validation_status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'unknown')`,
		id, req.Name, req.BaseURL, req.Version, req.Specification,
		string(specDocJSON), string(authMethodsJSON), string(rateLimitsJSON),
		endpointsCount, req.DocumentationURL)

	if err != nil {
		sendError(w, fmt.Sprintf("Failed to create API definition: %v", err), http.StatusInternalServerError)
		return
	}

	sendSuccess(w, map[string]interface{}{
		"id":      id,
		"message": "API definition created successfully",
	})
}

// handleUpdateAPIDefinition updates an existing API definition
func (s *Server) handleUpdateAPIDefinition(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		sendError(w, "Database not configured", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		Name                  string                 `json:"name,omitempty"`
		BaseURL               string                 `json:"base_url,omitempty"`
		Version               string                 `json:"version,omitempty"`
		Specification         string                 `json:"specification,omitempty"`
		SpecDocument          map[string]interface{} `json:"spec_document,omitempty"`
		AuthenticationMethods []string               `json:"authentication_methods,omitempty"`
		RateLimits            map[string]interface{} `json:"rate_limits,omitempty"`
		DocumentationURL      string                 `json:"documentation_url,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Build dynamic update query
	updates := make([]string, 0)
	args := make([]interface{}, 0)
	argCount := 1

	if req.Name != "" {
		updates = append(updates, fmt.Sprintf("name = $%d", argCount))
		args = append(args, req.Name)
		argCount++
	}
	if req.BaseURL != "" {
		updates = append(updates, fmt.Sprintf("base_url = $%d", argCount))
		args = append(args, req.BaseURL)
		argCount++
	}
	if req.Version != "" {
		updates = append(updates, fmt.Sprintf("version = $%d", argCount))
		args = append(args, req.Version)
		argCount++
	}
	if req.Specification != "" {
		updates = append(updates, fmt.Sprintf("specification = $%d", argCount))
		args = append(args, req.Specification)
		argCount++
	}
	if req.SpecDocument != nil {
		specDocJSON, _ := json.Marshal(req.SpecDocument)
		updates = append(updates, fmt.Sprintf("spec_document = $%d", argCount))
		args = append(args, string(specDocJSON))
		argCount++

		// Update endpoints count from spec
		if paths, ok := req.SpecDocument["paths"].(map[string]interface{}); ok {
			updates = append(updates, fmt.Sprintf("endpoints_count = $%d", argCount))
			args = append(args, len(paths))
			argCount++
		}
	}
	if req.AuthenticationMethods != nil {
		authMethodsJSON, _ := json.Marshal(req.AuthenticationMethods)
		updates = append(updates, fmt.Sprintf("authentication_methods = $%d", argCount))
		args = append(args, string(authMethodsJSON))
		argCount++
	}
	if req.RateLimits != nil {
		rateLimitsJSON, _ := json.Marshal(req.RateLimits)
		updates = append(updates, fmt.Sprintf("rate_limits = $%d", argCount))
		args = append(args, string(rateLimitsJSON))
		argCount++
	}
	if req.DocumentationURL != "" {
		updates = append(updates, fmt.Sprintf("documentation_url = $%d", argCount))
		args = append(args, req.DocumentationURL)
		argCount++
	}

	if len(updates) == 0 {
		sendError(w, "No fields to update", http.StatusBadRequest)
		return
	}

	// Add ID as last argument
	args = append(args, id)
	query := fmt.Sprintf("UPDATE api_definitions SET %s WHERE id = $%d",
		joinStrings(updates, ", "), argCount)

	result, err := s.db.Exec(query, args...)
	if err != nil {
		sendError(w, fmt.Sprintf("Failed to update API definition: %v", err), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		sendError(w, "API definition not found", http.StatusNotFound)
		return
	}

	sendSuccess(w, map[string]interface{}{
		"message": "API definition updated successfully",
	})
}

// handleDeleteAPIDefinition deletes an API definition
func (s *Server) handleDeleteAPIDefinition(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		sendError(w, "Database not configured", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	result, err := s.db.Exec("DELETE FROM api_definitions WHERE id = $1", id)
	if err != nil {
		sendError(w, fmt.Sprintf("Failed to delete API definition: %v", err), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		sendError(w, "API definition not found", http.StatusNotFound)
		return
	}

	sendSuccess(w, map[string]interface{}{
		"message": "API definition deleted successfully",
	})
}

// handleDiscoverAPIEndpoints discovers endpoints from an OpenAPI specification
func (s *Server) handleDiscoverAPIEndpoints(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SpecURL string                 `json:"spec_url,omitempty"`
		SpecDoc map[string]interface{} `json:"spec_doc,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var specDoc map[string]interface{}

	// Fetch spec from URL if provided
	if req.SpecURL != "" {
		resp, err := http.Get(req.SpecURL)
		if err != nil {
			sendError(w, fmt.Sprintf("Failed to fetch OpenAPI spec: %v", err), http.StatusBadRequest)
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			sendError(w, fmt.Sprintf("Failed to read spec response: %v", err), http.StatusInternalServerError)
			return
		}

		if err := json.Unmarshal(body, &specDoc); err != nil {
			sendError(w, fmt.Sprintf("Failed to parse OpenAPI spec: %v", err), http.StatusBadRequest)
			return
		}
	} else if req.SpecDoc != nil {
		specDoc = req.SpecDoc
	} else {
		sendError(w, "Either spec_url or spec_doc is required", http.StatusBadRequest)
		return
	}

	// Extract endpoints from OpenAPI spec
	endpoints := make([]map[string]interface{}, 0)

	if paths, ok := specDoc["paths"].(map[string]interface{}); ok {
		for path, pathItemInterface := range paths {
			if pathItem, ok := pathItemInterface.(map[string]interface{}); ok {
				for method, operationInterface := range pathItem {
					// Only process HTTP methods
					httpMethods := map[string]bool{
						"get": true, "post": true, "put": true, "delete": true,
						"patch": true, "options": true, "head": true,
					}
					if !httpMethods[method] {
						continue
					}

					endpoint := map[string]interface{}{
						"path":   path,
						"method": method,
					}

					if operation, ok := operationInterface.(map[string]interface{}); ok {
						if summary, ok := operation["summary"].(string); ok {
							endpoint["summary"] = summary
						}
						if description, ok := operation["description"].(string); ok {
							endpoint["description"] = description
						}
						if tags, ok := operation["tags"].([]interface{}); ok {
							endpoint["tags"] = tags
						}
					}

					endpoints = append(endpoints, endpoint)
				}
			}
		}
	}

	// Extract API info
	info := map[string]interface{}{
		"endpoints_count": len(endpoints),
	}

	if infoObj, ok := specDoc["info"].(map[string]interface{}); ok {
		if title, ok := infoObj["title"].(string); ok {
			info["title"] = title
		}
		if version, ok := infoObj["version"].(string); ok {
			info["version"] = version
		}
		if description, ok := infoObj["description"].(string); ok {
			info["description"] = description
		}
	}

	if servers, ok := specDoc["servers"].([]interface{}); ok && len(servers) > 0 {
		if server, ok := servers[0].(map[string]interface{}); ok {
			if url, ok := server["url"].(string); ok {
				info["base_url"] = url
			}
		}
	}

	sendSuccess(w, map[string]interface{}{
		"info":      info,
		"endpoints": endpoints,
	})
}

// Helper function to join strings
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
