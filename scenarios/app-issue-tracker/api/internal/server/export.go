package server

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	issuespkg "app-issue-tracker-api/internal/issues"
	"app-issue-tracker-api/internal/logging"
	"app-issue-tracker-api/internal/server/handlers"
)

// exportIssuesHandler exports issues in various formats (JSON, CSV, Markdown)
func (s *Server) exportIssuesHandler(w http.ResponseWriter, r *http.Request) {
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "json"
	}

	status := r.URL.Query().Get("status")
	priority := r.URL.Query().Get("priority")
	issueType := r.URL.Query().Get("type")
	appID := r.URL.Query().Get("app_id")

	// Get all matching issues
	issues, err := s.getAllIssues(status, priority, issueType, appID, 1000)
	if err != nil {
		logging.LogErrorErr("Failed to load issues for export", err,
			"status", status,
			"priority", priority,
			"type", issueType,
			"app_id", appID,
		)
		handlers.WriteError(w, http.StatusInternalServerError, "Failed to load issues")
		return
	}

	switch format {
	case "json":
		s.exportJSON(w, issues)
	case "csv":
		s.exportCSV(w, issues)
	case "markdown", "md":
		s.exportMarkdown(w, issues)
	default:
		handlers.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Unsupported format: %s. Use json, csv, or markdown", format))
	}
}

// exportJSON exports issues as JSON
func (s *Server) exportJSON(w http.ResponseWriter, issues []Issue) {
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"issues-%s.json\"", time.Now().Format("2006-01-02")))

	data := map[string]interface{}{
		"exported_at": time.Now().Format(time.RFC3339),
		"count":       len(issues),
		"issues":      issues,
	}

	if err := handlers.WriteJSON(w, http.StatusOK, data); err != nil {
		logging.LogErrorErr("Failed to write JSON export", err)
	}
}

// exportCSV exports issues as CSV
func (s *Server) exportCSV(w http.ResponseWriter, issues []Issue) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"issues-%s.csv\"", time.Now().Format("2006-01-02")))

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	header := []string{"ID", "Title", "Description", "Type", "Priority", "Status", "App ID", "Tags", "Created At", "Updated At"}
	if err := csvWriter.Write(header); err != nil {
		logging.LogErrorErr("Failed to write CSV header", err)
		return
	}

	for _, issue := range issues {
		record := []string{
			issue.ID,
			issue.Title,
			issue.Description,
			issue.Type,
			issue.Priority,
			issue.Status,
			issue.AppID,
			strings.Join(issue.Metadata.Tags, "; "),
			issue.Metadata.CreatedAt,
			issue.Metadata.UpdatedAt,
		}
		if err := csvWriter.Write(record); err != nil {
			logging.LogErrorErr("Failed to write CSV record", err, "issue_id", issue.ID)
			return
		}
	}

	if err := csvWriter.Error(); err != nil {
		logging.LogErrorErr("CSV writer encountered an error", err)
	}
}

// exportMarkdown exports issues as Markdown
func (s *Server) exportMarkdown(w http.ResponseWriter, issues []Issue) {
	w.Header().Set("Content-Type", "text/markdown")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"issues-%s.md\"", time.Now().Format("2006-01-02")))

	// Write header
	fmt.Fprintf(w, "# Issues Export\n\n")
	fmt.Fprintf(w, "**Exported:** %s  \n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "**Total Issues:** %d\n\n", len(issues))

	// Group by status
	byStatus := make(map[string][]Issue)
	for _, issue := range issues {
		byStatus[issue.Status] = append(byStatus[issue.Status], issue)
	}

	// Write each status section
	caser := cases.Title(language.English)
	statuses := issuespkg.ValidStatuses()
	for _, status := range statuses {
		statusIssues := byStatus[status]
		if len(statusIssues) == 0 {
			continue
		}

		fmt.Fprintf(w, "## %s (%d)\n\n", caser.String(status), len(statusIssues))

		for _, issue := range statusIssues {
			priorityEmoji := map[string]string{
				"critical": "🔴",
				"high":     "🟠",
				"medium":   "🟡",
				"low":      "🟢",
			}[issue.Priority]

			typeEmoji := map[string]string{
				"bug":     "🐛",
				"feature": "✨",
				"task":    "📋",
			}[issue.Type]

			fmt.Fprintf(w, "### %s %s %s\n\n", priorityEmoji, typeEmoji, issue.Title)
			fmt.Fprintf(w, "**ID:** `%s`  \n", issue.ID)
			fmt.Fprintf(w, "**Priority:** %s  \n", issue.Priority)
			fmt.Fprintf(w, "**Type:** %s  \n", issue.Type)
			fmt.Fprintf(w, "**App:** %s  \n", issue.AppID)
			fmt.Fprintf(w, "**Created:** %s  \n", issue.Metadata.CreatedAt)
			fmt.Fprintf(w, "\n%s\n\n", issue.Description)

			if len(issue.Metadata.Tags) > 0 {
				fmt.Fprintf(w, "**Tags:** %s\n\n", strings.Join(issue.Metadata.Tags, ", "))
			}

			fmt.Fprintf(w, "---\n\n")
		}
	}
}
