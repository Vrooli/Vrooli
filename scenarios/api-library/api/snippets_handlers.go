package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
)


// CreateSnippetRequest represents the request to create a new snippet
type CreateSnippetRequest struct {
	APIID                string                 `json:"api_id"`
	Title                string                 `json:"title"`
	Description          string                 `json:"description"`
	Language             string                 `json:"language"`
	Framework            string                 `json:"framework,omitempty"`
	Code                 string                 `json:"code"`
	Dependencies         map[string]interface{} `json:"dependencies,omitempty"`
	EnvironmentVariables []string               `json:"environment_variables,omitempty"`
	Prerequisites        string                 `json:"prerequisites,omitempty"`
	SnippetType          string                 `json:"snippet_type"`
	Tags                 []string               `json:"tags,omitempty"`
	Tested               bool                   `json:"tested"`
	Official             bool                   `json:"official"`
	EndpointID           string                 `json:"endpoint_id,omitempty"`
	SourceURL            string                 `json:"source_url,omitempty"`
	Version              string                 `json:"version,omitempty"`
}

// getSnippetsHandler returns all snippets for a specific API
func getSnippetsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]
	
	query := `
		SELECT 
			s.id, s.api_id, s.title, s.description, s.language, s.framework,
			s.code, s.dependencies, s.environment_variables, s.prerequisites,
			s.snippet_type, s.tags, s.tested, s.official, s.community_verified,
			s.usage_count, s.helpful_count, s.not_helpful_count, s.endpoint_id,
			s.created_at, s.updated_at, s.created_by, s.source_url, s.version,
			s.last_verified, a.name, a.provider
		FROM integration_snippets s
		JOIN apis a ON s.api_id = a.id
		WHERE s.api_id = $1
		ORDER BY s.usage_count DESC, s.helpful_count DESC`
	
	rows, err := db.Query(query, apiID)
	if err != nil {
		http.Error(w, "Error fetching snippets", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var snippets []IntegrationSnippet
	for rows.Next() {
		var snippet IntegrationSnippet
		var deps sql.NullString
		var envVars pq.StringArray
		var tags pq.StringArray
		var endpointID sql.NullString
		var lastVerified sql.NullTime
		var framework sql.NullString
		var prerequisites sql.NullString
		var sourceURL sql.NullString
		var version sql.NullString
		
		err := rows.Scan(
			&snippet.ID, &snippet.APIID, &snippet.Title, &snippet.Description,
			&snippet.Language, &framework, &snippet.Code, &deps,
			&envVars, &prerequisites, &snippet.SnippetType, &tags,
			&snippet.Tested, &snippet.Official, &snippet.CommunityVerified,
			&snippet.UsageCount, &snippet.HelpfulCount, &snippet.NotHelpfulCount,
			&endpointID, &snippet.CreatedAt, &snippet.UpdatedAt,
			&snippet.CreatedBy, &sourceURL, &version, &lastVerified,
			&snippet.APIName, &snippet.APIProvider,
		)
		if err != nil {
			log.Printf("Error scanning snippet: %v", err)
			continue
		}
		
		snippet.EnvironmentVariables = []string(envVars)
		snippet.Tags = []string(tags)
		
		if framework.Valid {
			snippet.Framework = framework.String
		}
		if prerequisites.Valid {
			snippet.Prerequisites = prerequisites.String
		}
		if sourceURL.Valid {
			snippet.SourceURL = sourceURL.String
		}
		if version.Valid {
			snippet.Version = version.String
		}
		if endpointID.Valid {
			snippet.EndpointID = &endpointID.String
		}
		if lastVerified.Valid {
			snippet.LastVerified = &lastVerified.Time
		}
		if deps.Valid {
			json.Unmarshal([]byte(deps.String), &snippet.Dependencies)
		}
		
		snippets = append(snippets, snippet)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"snippets": snippets,
		"count":    len(snippets),
	})
}

// createSnippetHandler creates a new integration snippet
func createSnippetHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]
	
	var req CreateSnippetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Override API ID from URL
	req.APIID = apiID
	
	// Validate required fields
	if req.Title == "" || req.Code == "" || req.Language == "" || req.SnippetType == "" {
		http.Error(w, "Missing required fields: title, code, language, snippet_type", http.StatusBadRequest)
		return
	}
	
	// Generate new ID
	snippetID := uuid.New().String()
	
	// Prepare dependencies as JSON
	var depsJSON []byte
	if req.Dependencies != nil {
		depsJSON, _ = json.Marshal(req.Dependencies)
	}
	
	// Prepare optional fields
	var endpointID *string
	if req.EndpointID != "" {
		endpointID = &req.EndpointID
	}
	
	query := `
		INSERT INTO integration_snippets (
			id, api_id, title, description, language, framework,
			code, dependencies, environment_variables, prerequisites,
			snippet_type, tags, tested, official, endpoint_id,
			source_url, version, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING created_at, updated_at`
	
	var createdAt, updatedAt time.Time
	err := db.QueryRow(
		query,
		snippetID, req.APIID, req.Title, req.Description, req.Language, req.Framework,
		req.Code, depsJSON, pq.Array(req.EnvironmentVariables), req.Prerequisites,
		req.SnippetType, pq.Array(req.Tags), req.Tested, req.Official, endpointID,
		req.SourceURL, req.Version, "system",
	).Scan(&createdAt, &updatedAt)
	
	if err != nil {
		log.Printf("Error creating snippet: %v", err)
		http.Error(w, "Error creating snippet", http.StatusInternalServerError)
		return
	}
	
	// Return the created snippet
	snippet := IntegrationSnippet{
		ID:                   snippetID,
		APIID:                req.APIID,
		Title:                req.Title,
		Description:          req.Description,
		Language:             req.Language,
		Framework:            req.Framework,
		Code:                 req.Code,
		Dependencies:         req.Dependencies,
		EnvironmentVariables: req.EnvironmentVariables,
		Prerequisites:        req.Prerequisites,
		SnippetType:          req.SnippetType,
		Tags:                 req.Tags,
		Tested:               req.Tested,
		Official:             req.Official,
		EndpointID:           endpointID,
		SourceURL:            req.SourceURL,
		Version:              req.Version,
		CreatedAt:            createdAt,
		UpdatedAt:            updatedAt,
		CreatedBy:            "system",
		UsageCount:           0,
		HelpfulCount:         0,
		NotHelpfulCount:      0,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(snippet)
}

// getPopularSnippetsHandler returns the most popular snippets across all APIs
func getPopularSnippetsHandler(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	
	languageFilter := r.URL.Query().Get("language")
	typeFilter := r.URL.Query().Get("type")
	
	query := `
		SELECT 
			s.id, s.title, s.description, s.language, s.snippet_type,
			a.name, a.provider, s.helpful_count, s.usage_count,
			s.official, s.tested
		FROM integration_snippets s
		JOIN apis a ON s.api_id = a.id
		WHERE s.helpful_count > s.not_helpful_count`
	
	var args []interface{}
	argCount := 0
	
	if languageFilter != "" {
		argCount++
		query += fmt.Sprintf(" AND s.language = $%d", argCount)
		args = append(args, languageFilter)
	}
	
	if typeFilter != "" {
		argCount++
		query += fmt.Sprintf(" AND s.snippet_type = $%d", argCount)
		args = append(args, typeFilter)
	}
	
	argCount++
	query += fmt.Sprintf(" ORDER BY s.usage_count DESC, s.helpful_count DESC LIMIT $%d", argCount)
	args = append(args, limit)
	
	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Error fetching popular snippets", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	type PopularSnippet struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Language    string `json:"language"`
		SnippetType string `json:"snippet_type"`
		APIName     string `json:"api_name"`
		APIProvider string `json:"api_provider"`
		HelpfulCount int   `json:"helpful_count"`
		UsageCount  int    `json:"usage_count"`
		Official    bool   `json:"official"`
		Tested      bool   `json:"tested"`
	}
	
	var snippets []PopularSnippet
	for rows.Next() {
		var snippet PopularSnippet
		err := rows.Scan(
			&snippet.ID, &snippet.Title, &snippet.Description,
			&snippet.Language, &snippet.SnippetType,
			&snippet.APIName, &snippet.APIProvider,
			&snippet.HelpfulCount, &snippet.UsageCount,
			&snippet.Official, &snippet.Tested,
		)
		if err != nil {
			log.Printf("Error scanning popular snippet: %v", err)
			continue
		}
		snippets = append(snippets, snippet)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"snippets": snippets,
		"count":    len(snippets),
	})
}

// voteSnippetHandler handles voting on snippet helpfulness
func voteSnippetHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	snippetID := vars["snippet_id"]
	
	var req struct {
		Helpful bool `json:"helpful"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	var query string
	if req.Helpful {
		query = `UPDATE integration_snippets SET helpful_count = helpful_count + 1 WHERE id = $1`
	} else {
		query = `UPDATE integration_snippets SET not_helpful_count = not_helpful_count + 1 WHERE id = $1`
	}
	
	result, err := db.Exec(query, snippetID)
	if err != nil {
		http.Error(w, "Error updating vote", http.StatusInternalServerError)
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Snippet not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"message": "Vote recorded",
	})
}

// getSnippetByIDHandler retrieves a single snippet by ID
func getSnippetByIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	snippetID := vars["snippet_id"]
	
	// Increment usage count
	go func() {
		db.Exec("UPDATE integration_snippets SET usage_count = usage_count + 1 WHERE id = $1", snippetID)
	}()
	
	query := `
		SELECT 
			s.id, s.api_id, s.title, s.description, s.language, s.framework,
			s.code, s.dependencies, s.environment_variables, s.prerequisites,
			s.snippet_type, s.tags, s.tested, s.official, s.community_verified,
			s.usage_count, s.helpful_count, s.not_helpful_count, s.endpoint_id,
			s.created_at, s.updated_at, s.created_by, s.source_url, s.version,
			s.last_verified, a.name, a.provider
		FROM integration_snippets s
		JOIN apis a ON s.api_id = a.id
		WHERE s.id = $1`
	
	var snippet IntegrationSnippet
	var deps sql.NullString
	var envVars pq.StringArray
	var tags pq.StringArray
	var endpointID sql.NullString
	var lastVerified sql.NullTime
	var framework sql.NullString
	var prerequisites sql.NullString
	var sourceURL sql.NullString
	var version sql.NullString
	
	err := db.QueryRow(query, snippetID).Scan(
		&snippet.ID, &snippet.APIID, &snippet.Title, &snippet.Description,
		&snippet.Language, &framework, &snippet.Code, &deps,
		&envVars, &prerequisites, &snippet.SnippetType, &tags,
		&snippet.Tested, &snippet.Official, &snippet.CommunityVerified,
		&snippet.UsageCount, &snippet.HelpfulCount, &snippet.NotHelpfulCount,
		&endpointID, &snippet.CreatedAt, &snippet.UpdatedAt,
		&snippet.CreatedBy, &sourceURL, &version, &lastVerified,
		&snippet.APIName, &snippet.APIProvider,
	)
	
	if err == sql.ErrNoRows {
		http.Error(w, "Snippet not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error fetching snippet: %v", err)
		http.Error(w, "Error fetching snippet", http.StatusInternalServerError)
		return
	}
	
	snippet.EnvironmentVariables = []string(envVars)
	snippet.Tags = []string(tags)
	
	if framework.Valid {
		snippet.Framework = framework.String
	}
	if prerequisites.Valid {
		snippet.Prerequisites = prerequisites.String
	}
	if sourceURL.Valid {
		snippet.SourceURL = sourceURL.String
	}
	if version.Valid {
		snippet.Version = version.String
	}
	if endpointID.Valid {
		snippet.EndpointID = &endpointID.String
	}
	if lastVerified.Valid {
		snippet.LastVerified = &lastVerified.Time
	}
	if deps.Valid {
		json.Unmarshal([]byte(deps.String), &snippet.Dependencies)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snippet)
}