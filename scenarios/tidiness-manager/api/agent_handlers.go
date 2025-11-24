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

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"issues": issues,
		"count":  len(issues),
		"query": map[string]interface{}{
			"scenario": req.Scenario,
			"file":     req.File,
			"folder":   req.Folder,
			"category": req.Category,
			"limit":    req.Limit,
		},
	})
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
