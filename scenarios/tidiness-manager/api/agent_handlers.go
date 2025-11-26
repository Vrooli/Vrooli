package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// AgentIssue represents an issue in agent-friendly format
type AgentIssue struct {
	ID               int    `json:"id"`
	Scenario         string `json:"scenario"`
	FilePath         string `json:"file_path"`
	Category         string `json:"category"`
	Severity         string `json:"severity"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	LineNumber       *int   `json:"line_number,omitempty"`
	ColumnNumber     *int   `json:"column_number,omitempty"`
	AgentNotes       string `json:"agent_notes,omitempty"`
	RemediationSteps string `json:"remediation_steps,omitempty"`
	Status           string `json:"status"`
	CreatedAt        string `json:"created_at"`
}

// AgentIssuesRequest defines query parameters for agent issue requests
type AgentIssuesRequest struct {
	Scenario string
	File     string
	Folder   string
	Category string
	Limit    int
	Force    bool
}

// handleAgentGetIssues returns top N issues for a scenario (TM-API-001, TM-API-002, TM-API-003, TM-API-004, TM-API-006)
func (s *Server) handleAgentGetIssues(w http.ResponseWriter, r *http.Request) {
	req := parseAgentIssuesRequest(r)

	if req.Scenario == "" {
		respondError(w, http.StatusBadRequest, "scenario parameter is required")
		return
	}

	// TM-DA-006: Agent read calls return existing data without triggering new scans by default
	if req.Force {
		// TM-DA-007, TM-DA-008: Enqueue force-scan in controlled queue
		s.log("force scan requested", map[string]interface{}{
			"scenario": req.Scenario,
			"file":     req.File,
			"folder":   req.Folder,
		})
		// NOTE: Force scan queueing implementation deferred to P1
		respondError(w, http.StatusNotImplemented, "force scans not yet implemented (P1)")
		return
	}

	// Build query based on filters (TM-API-002, TM-API-003)
	query := buildIssuesQuery(req)
	args := buildIssuesArgs(req)

	rows, err := s.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		s.log("query failed", map[string]interface{}{
			"error": err.Error(),
			"query": query,
		})
		respondError(w, http.StatusInternalServerError, "failed to query issues")
		return
	}
	defer rows.Close()

	issues := []AgentIssue{}
	for rows.Next() {
		var issue AgentIssue
		var lineNum, colNum sql.NullInt64
		var agentNotes, remediation sql.NullString

		err := rows.Scan(
			&issue.ID,
			&issue.Scenario,
			&issue.FilePath,
			&issue.Category,
			&issue.Severity,
			&issue.Title,
			&issue.Description,
			&lineNum,
			&colNum,
			&agentNotes,
			&remediation,
			&issue.Status,
			&issue.CreatedAt,
		)
		if err != nil {
			s.log("scan failed", map[string]interface{}{"error": err.Error()})
			continue
		}

		if lineNum.Valid {
			v := int(lineNum.Int64)
			issue.LineNumber = &v
		}
		if colNum.Valid {
			v := int(colNum.Int64)
			issue.ColumnNumber = &v
		}
		if agentNotes.Valid {
			issue.AgentNotes = agentNotes.String
		}
		if remediation.Valid {
			issue.RemediationSteps = remediation.String
		}

		issues = append(issues, issue)
	}

	// Return plain array for simpler agent consumption (TM-API-001)
	respondJSON(w, http.StatusOK, issues)
}

// handleAgentStoreIssue stores a new issue (TM-API-006, TM-DA-001, TM-DA-002, TM-DA-003)
func (s *Server) handleAgentStoreIssue(w http.ResponseWriter, r *http.Request) {
	var issue struct {
		Scenario         string `json:"scenario"`
		FilePath         string `json:"file_path"`
		Category         string `json:"category"`
		Severity         string `json:"severity"`
		Title            string `json:"title"`
		Description      string `json:"description"`
		LineNumber       *int   `json:"line_number"`
		ColumnNumber     *int   `json:"column_number"`
		AgentNotes       string `json:"agent_notes"`
		RemediationSteps string `json:"remediation_steps"`
		CampaignID       *int   `json:"campaign_id"`
		SessionID        string `json:"session_id"`
		ResourceUsed     string `json:"resource_used"`
	}

	if err := json.NewDecoder(r.Body).Decode(&issue); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if issue.Scenario == "" || issue.FilePath == "" || issue.Category == "" || issue.Severity == "" {
		respondError(w, http.StatusBadRequest, "scenario, file_path, category, and severity are required")
		return
	}

	query := `
		INSERT INTO issues (
			scenario, file_path, category, severity, title, description,
			line_number, column_number, agent_notes, remediation_steps,
			campaign_id, session_id, resource_used
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at
	`

	var id int
	var createdAt string
	err := s.db.QueryRowContext(
		r.Context(),
		query,
		issue.Scenario, issue.FilePath, issue.Category, issue.Severity,
		issue.Title, issue.Description, issue.LineNumber, issue.ColumnNumber,
		issue.AgentNotes, issue.RemediationSteps, issue.CampaignID,
		issue.SessionID, issue.ResourceUsed,
	).Scan(&id, &createdAt)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			respondError(w, http.StatusConflict, "duplicate issue")
			return
		}
		s.log("insert failed", map[string]interface{}{"error": err.Error()})
		respondError(w, http.StatusInternalServerError, "failed to store issue")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":         id,
		"created_at": createdAt,
	})
}

// handleAgentUpdateIssue updates issue status (TM-IM-001, TM-IM-002, TM-IM-003)
func (s *Server) handleAgentUpdateIssue(w http.ResponseWriter, r *http.Request) {
	// Extract issue ID from URL
	vars := mux.Vars(r)
	idStr := vars["id"]
	if idStr == "" {
		respondError(w, http.StatusBadRequest, "issue ID is required")
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid issue ID")
		return
	}

	var update struct {
		Status          string `json:"status"`
		ResolutionNotes string `json:"resolution_notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate status
	validStatuses := map[string]bool{"open": true, "resolved": true, "ignored": true}
	if !validStatuses[update.Status] {
		respondError(w, http.StatusBadRequest, "status must be one of: open, resolved, ignored")
		return
	}

	// Update the issue
	query := `
		UPDATE issues
		SET status = $1, resolution_notes = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
		RETURNING id, status, updated_at
	`

	var returnedID int
	var returnedStatus, updatedAt string
	err = s.db.QueryRowContext(r.Context(), query, update.Status, update.ResolutionNotes, id).
		Scan(&returnedID, &returnedStatus, &updatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, "issue not found")
			return
		}
		s.log("update failed", map[string]interface{}{"error": err.Error()})
		respondError(w, http.StatusInternalServerError, "failed to update issue")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":         returnedID,
		"status":     returnedStatus,
		"updated_at": updatedAt,
	})
}

// handleAgentGetScenarios returns list of scenarios with issue counts
func (s *Server) handleAgentGetScenarios(w http.ResponseWriter, r *http.Request) {
	// Get all scenarios from filesystem using vrooli CLI
	allScenarios, err := s.getAllScenarios(r.Context())
	if err != nil {
		s.log("failed to get scenarios from CLI", map[string]interface{}{"error": err.Error()})
		respondError(w, http.StatusInternalServerError, "failed to query scenarios")
		return
	}

	// Get issue counts from database for scenarios that have issues
	query := `
		SELECT
			scenario,
			COUNT(*) as total_issues,
			COUNT(CASE WHEN category = 'lint' THEN 1 END) as lint_issues,
			COUNT(CASE WHEN category = 'type' THEN 1 END) as type_issues,
			COUNT(CASE WHEN category = 'length' THEN 1 END) as long_files
		FROM issues
		WHERE status = 'open'
		GROUP BY scenario
	`

	rows, err := s.db.QueryContext(r.Context(), query)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query issues")
		return
	}
	defer rows.Close()

	// Build map of scenario -> issue counts
	issueCounts := make(map[string]map[string]interface{})
	for rows.Next() {
		var scenario string
		var total, lint, typeIssues, longFiles int
		if err := rows.Scan(&scenario, &total, &lint, &typeIssues, &longFiles); err != nil {
			continue
		}
		issueCounts[scenario] = map[string]interface{}{
			"total":      total,
			"lint":       lint,
			"type":       typeIssues,
			"long_files": longFiles,
		}
	}

	// Combine all scenarios with their issue counts (0 if no issues)
	scenarios := []map[string]interface{}{}
	for _, scenarioName := range allScenarios {
		counts, hasIssues := issueCounts[scenarioName]
		if !hasIssues {
			counts = map[string]interface{}{
				"total":      0,
				"lint":       0,
				"type":       0,
				"long_files": 0,
			}
		}

		scenario := map[string]interface{}{
			"scenario":   scenarioName,
			"total":      counts["total"],
			"lint":       counts["lint"],
			"type":       counts["type"],
			"long_files": counts["long_files"],
		}
		scenarios = append(scenarios, scenario)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"scenarios": scenarios,
		"count":     len(scenarios),
	})
}

func parseAgentIssuesRequest(r *http.Request) AgentIssuesRequest {
	q := r.URL.Query()
	limit := 10
	if l := q.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}

	return AgentIssuesRequest{
		Scenario: q.Get("scenario"),
		File:     q.Get("file"),
		Folder:   q.Get("folder"),
		Category: q.Get("category"),
		Limit:    limit,
		Force:    q.Get("force") == "true",
	}
}

func buildIssuesQuery(req AgentIssuesRequest) string {
	base := `
		SELECT
			id, scenario, file_path, category, severity, title, description,
			line_number, column_number, agent_notes, remediation_steps,
			status, created_at
		FROM issues
		WHERE scenario = $1 AND status = 'open'
	`

	conditions := []string{}
	if req.File != "" {
		conditions = append(conditions, "AND file_path = $2")
	} else if req.Folder != "" {
		conditions = append(conditions, "AND file_path LIKE $2")
	}

	if req.Category != "" {
		if req.File != "" || req.Folder != "" {
			conditions = append(conditions, "AND category = $3")
		} else {
			conditions = append(conditions, "AND category = $2")
		}
	}

	// TM-API-004: Rank by severity, then criticality
	ordering := `
		ORDER BY
			CASE severity
				WHEN 'critical' THEN 1
				WHEN 'high' THEN 2
				WHEN 'medium' THEN 3
				WHEN 'low' THEN 4
				WHEN 'info' THEN 5
				ELSE 6
			END,
			created_at DESC
	`

	limit := " LIMIT $"
	paramCount := 1
	if req.File != "" || req.Folder != "" {
		paramCount++
	}
	if req.Category != "" {
		paramCount++
	}
	paramCount++
	limit += strconv.Itoa(paramCount)

	return base + strings.Join(conditions, " ") + ordering + limit
}

func buildIssuesArgs(req AgentIssuesRequest) []interface{} {
	args := []interface{}{req.Scenario}

	if req.File != "" {
		args = append(args, req.File)
	} else if req.Folder != "" {
		args = append(args, req.Folder+"%")
	}

	if req.Category != "" {
		args = append(args, req.Category)
	}

	args = append(args, req.Limit)
	return args
}

// handleAgentGetScenarioDetail returns detailed stats and file list for a specific scenario (OT-P0-010, TM-UI-003, TM-UI-004)
func (s *Server) handleAgentGetScenarioDetail(w http.ResponseWriter, r *http.Request) {
	vars := map[string]string{}
	// Extract {name} path parameter from request URL
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/agent/scenarios/"), "/")
	if len(parts) > 0 {
		vars["name"] = parts[0]
	}

	scenarioName := vars["name"]
	if scenarioName == "" {
		respondError(w, http.StatusBadRequest, "scenario name is required")
		return
	}

	// Get aggregate stats for this scenario
	var lightIssues, aiIssues, longFiles int
	query := `
		SELECT
			COUNT(CASE WHEN category IN ('lint', 'type') THEN 1 END) as light_issues,
			COUNT(CASE WHEN category NOT IN ('lint', 'type', 'length') THEN 1 END) as ai_issues,
			COUNT(CASE WHEN category = 'length' THEN 1 END) as long_files
		FROM issues
		WHERE scenario = $1 AND status = 'open'
	`
	err := s.db.QueryRowContext(r.Context(), query, scenarioName).Scan(&lightIssues, &aiIssues, &longFiles)
	if err != nil && err != sql.ErrNoRows {
		s.log("query failed", map[string]interface{}{"error": err.Error()})
		respondError(w, http.StatusInternalServerError, "failed to query stats")
		return
	}

	// Get file-level metrics from database (TM-UI-003, TM-UI-004)
	filesQuery := `
		SELECT
			fm.file_path,
			fm.line_count,
			COUNT(i.id) as issue_count
		FROM file_metrics fm
		LEFT JOIN issues i ON i.scenario = fm.scenario AND i.file_path = fm.file_path AND i.status = 'open'
		WHERE fm.scenario = $1
		GROUP BY fm.file_path, fm.line_count
		ORDER BY issue_count DESC, fm.line_count DESC
		LIMIT 100
	`

	rows, err := s.db.QueryContext(r.Context(), filesQuery, scenarioName)
	if err != nil {
		s.log("query failed", map[string]interface{}{"error": err.Error()})
		respondError(w, http.StatusInternalServerError, "failed to query files")
		return
	}
	defer rows.Close()

	files := []map[string]interface{}{}
	for rows.Next() {
		var filePath sql.NullString
		var lineCount, issueCount sql.NullInt64
		if err := rows.Scan(&filePath, &lineCount, &issueCount); err != nil {
			continue
		}

		file := map[string]interface{}{
			"path":        "",
			"lines":       0,
			"totalIssues": 0,
			"visitCount":  0,
		}

		if filePath.Valid {
			file["path"] = filePath.String
		}
		if lineCount.Valid {
			file["lines"] = int(lineCount.Int64)
		}
		if issueCount.Valid {
			file["totalIssues"] = int(issueCount.Int64)
		}

		files = append(files, file)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"scenario":    scenarioName,
		"lightIssues": lightIssues,
		"aiIssues":    aiIssues,
		"longFiles":   longFiles,
		"files":       files,
	})
}

// getAllScenarios fetches all scenarios from the vrooli CLI with caching
func (s *Server) getAllScenarios(parentCtx context.Context) ([]string, error) {
	// Check if cache is still valid
	if time.Since(s.scenarioCacheTime) < s.scenarioCacheTTL && len(s.scenarioCache) > 0 {
		return s.scenarioCache, nil
	}

	ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "status", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var resp struct {
		Scenarios []map[string]interface{} `json:"scenarios"`
	}
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, err
	}

	scenarios := []string{}
	for _, scenario := range resp.Scenarios {
		if name, ok := scenario["name"].(string); ok {
			scenarios = append(scenarios, name)
		}
	}

	// Update cache
	s.scenarioCache = scenarios
	s.scenarioCacheTime = time.Now()

	return scenarios, nil
}
