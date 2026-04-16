package support

import (
	"encoding/json"
	"strings"
)

// Target mirrors issuespkg.Target from the API.
type Target struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// IssueMetadata projects the issue.metadata subset the CLI cares about.
type IssueMetadata struct {
	CreatedAt string   `json:"created_at,omitempty"`
	UpdatedAt string   `json:"updated_at,omitempty"`
	Tags      []string `json:"tags,omitempty"`
}

// Issue is a compact projection of issuespkg.Issue covering the fields rendered
// by list/show/search commands. Unknown fields are ignored.
type Issue struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description,omitempty"`
	Type        string          `json:"type,omitempty"`
	Priority    string          `json:"priority,omitempty"`
	Status      string          `json:"status,omitempty"`
	Targets     []Target        `json:"targets,omitempty"`
	Metadata    IssueMetadata   `json:"metadata,omitempty"`
	Investig    json.RawMessage `json:"investigation,omitempty"`
	Fix         json.RawMessage `json:"fix,omitempty"`
}

// IssueListData wraps GET /issues responses.
type IssueListData struct {
	Issues []Issue `json:"issues"`
	Count  int     `json:"count"`
}

// IssueDetailData wraps GET /issues/{id} responses.
type IssueDetailData struct {
	Issue *Issue `json:"issue"`
}

// IssueCreateData wraps POST /issues responses.
type IssueCreateData struct {
	Issue       *Issue `json:"issue"`
	IssueID     string `json:"issue_id"`
	StoragePath string `json:"storage_path,omitempty"`
}

// IssueSearchData wraps GET /issues/search responses.
type IssueSearchData struct {
	Results []Issue `json:"results"`
	Count   int     `json:"count"`
	Query   string  `json:"query"`
	Method  string  `json:"method,omitempty"`
}

// Agent mirrors issuespkg.Agent.
type Agent struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	DisplayName    string   `json:"display_name"`
	Description    string   `json:"description,omitempty"`
	Capabilities   []string `json:"capabilities,omitempty"`
	IsActive       bool     `json:"is_active"`
	SuccessRate    float64  `json:"success_rate"`
	TotalRuns      int      `json:"total_runs"`
	SuccessfulRuns int      `json:"successful_runs"`
}

// AgentsData wraps GET /agents responses.
type AgentsData struct {
	Agents []Agent `json:"agents"`
	Count  int     `json:"count"`
	Runner string  `json:"runner,omitempty"`
}

// App mirrors issuespkg.App.
type App struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type,omitempty"`
	Status      string `json:"status,omitempty"`
	TotalIssues int    `json:"total_issues"`
	OpenIssues  int    `json:"open_issues"`
}

// AppsData wraps GET /apps responses.
type AppsData struct {
	Apps  []App `json:"apps"`
	Count int   `json:"count"`
}

// FormatTargets renders an issue's target list as a compact string.
func FormatTargets(targets []Target) string {
	if len(targets) == 0 {
		return ""
	}
	parts := make([]string, 0, len(targets))
	for _, t := range targets {
		id := strings.TrimSpace(t.ID)
		typ := strings.TrimSpace(t.Type)
		if id == "" && typ == "" {
			continue
		}
		if typ == "" {
			parts = append(parts, id)
			continue
		}
		parts = append(parts, typ+":"+id)
	}
	return strings.Join(parts, ", ")
}
