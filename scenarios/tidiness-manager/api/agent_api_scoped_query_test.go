package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// [REQ:TM-API-002] File-scoped issue query
func TestAgentGetIssues_FileFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	srv := setupTestServer(t)
	if srv == nil {
		return
	}

	// Insert issues for different files
	insertTestIssue(t, srv.db, "test-scenario", "api/main.go", "length", "high", "Issue 1", "Desc 1")
	insertTestIssue(t, srv.db, "test-scenario", "api/handlers.go", "length", "high", "Issue 2", "Desc 2")

	req, err := http.NewRequest("GET", "/api/v1/agent/issues?scenario=test-scenario&file=api/main.go", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	var issues []AgentIssue
	if err := json.Unmarshal(rr.Body.Bytes(), &issues); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	for _, issue := range issues {
		if issue.FilePath != "api/main.go" {
			t.Errorf("Expected only issues from api/main.go, got %s", issue.FilePath)
		}
	}
}

// [REQ:TM-API-002] Folder-scoped issue query
func TestAgentGetIssues_FolderFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	srv := setupTestServer(t)
	if srv == nil {
		return
	}

	// Insert issues in different folders
	insertTestIssue(t, srv.db, "test-scenario", "api/main.go", "length", "high", "Issue 1", "Desc 1")
	insertTestIssue(t, srv.db, "test-scenario", "ui/App.tsx", "complexity", "medium", "Issue 2", "Desc 2")

	req, err := http.NewRequest("GET", "/api/v1/agent/issues?scenario=test-scenario&folder=api", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	var issues []AgentIssue
	if err := json.Unmarshal(rr.Body.Bytes(), &issues); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	for _, issue := range issues {
		if len(issue.FilePath) < 4 || issue.FilePath[:4] != "api/" {
			t.Errorf("Expected only issues from api/ folder, got %s", issue.FilePath)
		}
	}
}

// [REQ:TM-API-002] Test folder filter edge cases
func TestAgentGetIssues_FolderFilterEdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	srv := setupTestServer(t)
	if srv == nil {
		return
	}

	// Insert test data with nested folders
	insertTestIssue(t, srv.db, "test-scenario", "api/v1/handlers.go", "length", "high", "Issue 1", "Desc")
	insertTestIssue(t, srv.db, "test-scenario", "api/v2/handlers.go", "length", "high", "Issue 2", "Desc")
	insertTestIssue(t, srv.db, "test-scenario", "ui/components/Button.tsx", "complexity", "medium", "Issue 3", "Desc")

	tests := []struct {
		name           string
		folder         string
		expectedPrefix string
		minIssues      int
	}{
		{
			name:           "top-level folder",
			folder:         "api",
			expectedPrefix: "api/",
			minIssues:      2,
		},
		{
			name:           "nested folder",
			folder:         "api/v1",
			expectedPrefix: "api/v1/",
			minIssues:      1,
		},
		{
			name:           "non-existent folder",
			folder:         "nonexistent",
			expectedPrefix: "nonexistent/",
			minIssues:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/agent/issues?scenario=test-scenario&folder="+tt.folder, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			rr := httptest.NewRecorder()
			srv.router.ServeHTTP(rr, req)

			var issues []AgentIssue
			if err := json.Unmarshal(rr.Body.Bytes(), &issues); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if len(issues) < tt.minIssues {
				t.Errorf("Expected at least %d issues, got %d", tt.minIssues, len(issues))
			}

			for _, issue := range issues {
				if len(issue.FilePath) < len(tt.expectedPrefix) ||
					issue.FilePath[:len(tt.expectedPrefix)] != tt.expectedPrefix {
					t.Errorf("Issue path %s doesn't start with %s", issue.FilePath, tt.expectedPrefix)
				}
			}
		})
	}
}

// [REQ:TM-API-002] Test combining multiple filters
func TestAgentGetIssues_CombinedFilters(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	srv := setupTestServer(t)
	if srv == nil {
		return
	}

	// Insert diverse test data
	insertTestIssue(t, srv.db, "test-scenario", "api/main.go", "length", "high", "Issue 1", "Desc 1")
	insertTestIssue(t, srv.db, "test-scenario", "api/main.go", "complexity", "medium", "Issue 2", "Desc 2")
	insertTestIssue(t, srv.db, "test-scenario", "api/handlers.go", "length", "high", "Issue 3", "Desc 3")
	insertTestIssue(t, srv.db, "test-scenario", "ui/App.tsx", "length", "high", "Issue 4", "Desc 4")

	tests := []struct {
		name          string
		queryParams   string
		expectedCount int
		validate      func(t *testing.T, issues []AgentIssue)
	}{
		{
			name:          "file and category",
			queryParams:   "scenario=test-scenario&file=api/main.go&category=length",
			expectedCount: 1,
			validate: func(t *testing.T, issues []AgentIssue) {
				for _, issue := range issues {
					if issue.FilePath != "api/main.go" || issue.Category != "length" {
						t.Errorf("Filter mismatch: got file=%s, category=%s", issue.FilePath, issue.Category)
					}
				}
			},
		},
		{
			name:          "folder and category",
			queryParams:   "scenario=test-scenario&folder=api&category=length",
			expectedCount: 2,
			validate: func(t *testing.T, issues []AgentIssue) {
				for _, issue := range issues {
					if len(issue.FilePath) < 4 || issue.FilePath[:4] != "api/" {
						t.Errorf("Expected api/ folder, got %s", issue.FilePath)
					}
					if issue.Category != "length" {
						t.Errorf("Expected length category, got %s", issue.Category)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/agent/issues?"+tt.queryParams, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			rr := httptest.NewRecorder()
			srv.router.ServeHTTP(rr, req)

			var issues []AgentIssue
			if err := json.Unmarshal(rr.Body.Bytes(), &issues); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if len(issues) != tt.expectedCount {
				t.Errorf("Expected %d issues, got %d", tt.expectedCount, len(issues))
			}

			if tt.validate != nil {
				tt.validate(t, issues)
			}
		})
	}
}

// [REQ:TM-API-003] Category-scoped issue query
func TestAgentGetIssues_CategoryFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	srv := setupTestServer(t)
	if srv == nil {
		return
	}

	// Insert issues with different categories
	insertTestIssue(t, srv.db, "test-scenario", "api/main.go", "length", "high", "Long file", "Too many lines")
	insertTestIssue(t, srv.db, "test-scenario", "api/handlers.go", "dead_code", "medium", "Unused function", "Remove unused")

	req, err := http.NewRequest("GET", "/api/v1/agent/issues?scenario=test-scenario&category=length", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	var issues []AgentIssue
	if err := json.Unmarshal(rr.Body.Bytes(), &issues); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	for _, issue := range issues {
		if issue.Category != "length" {
			t.Errorf("Expected only 'length' category issues, got '%s'", issue.Category)
		}
	}
}

// [REQ:TM-API-003] Test category filter with all valid categories
func TestAgentGetIssues_AllCategories(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	srv := setupTestServer(t)
	if srv == nil {
		return
	}

	categories := []string{"dead_code", "duplication", "length", "complexity", "style"}

	// Insert one issue per category
	for i, cat := range categories {
		insertTestIssue(t, srv.db, "test-scenario", "file.go", cat, "medium", "Issue "+string(rune('0'+i)), "Desc")
	}

	// Test filtering by each category
	for _, cat := range categories {
		t.Run("category_"+cat, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/agent/issues?scenario=test-scenario&category="+cat, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			rr := httptest.NewRecorder()
			srv.router.ServeHTTP(rr, req)

			var issues []AgentIssue
			if err := json.Unmarshal(rr.Body.Bytes(), &issues); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if len(issues) == 0 {
				t.Errorf("Expected at least one %s issue", cat)
			}

			for _, issue := range issues {
				if issue.Category != cat {
					t.Errorf("Expected category %s, got %s", cat, issue.Category)
				}
			}
		})
	}
}

// Test buildIssuesQuery generates correct SQL
func TestBuildIssuesQuery(t *testing.T) {
	tests := []struct {
		name        string
		req         AgentIssuesRequest
		wantContain []string
	}{
		{
			name: "basic query",
			req: AgentIssuesRequest{
				Scenario: "test",
				Limit:    10,
			},
			wantContain: []string{
				"SELECT",
				"FROM issues",
				"WHERE scenario = $1",
				"status = 'open'",
				"ORDER BY",
				"CASE severity",
				"LIMIT $2",
			},
		},
		{
			name: "with file filter",
			req: AgentIssuesRequest{
				Scenario: "test",
				File:     "api/main.go",
				Limit:    10,
			},
			wantContain: []string{
				"file_path = $2",
				"LIMIT $3",
			},
		},
		{
			name: "with folder filter",
			req: AgentIssuesRequest{
				Scenario: "test",
				Folder:   "api/",
				Limit:    10,
			},
			wantContain: []string{
				"file_path LIKE $2",
				"LIMIT $3",
			},
		},
		{
			name: "with category only",
			req: AgentIssuesRequest{
				Scenario: "test",
				Category: "dead_code",
				Limit:    10,
			},
			wantContain: []string{
				"category = $2",
				"LIMIT $3",
			},
		},
		{
			name: "with file and category",
			req: AgentIssuesRequest{
				Scenario: "test",
				File:     "api/main.go",
				Category: "length",
				Limit:    10,
			},
			wantContain: []string{
				"file_path = $2",
				"category = $3",
				"LIMIT $4",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := buildIssuesQuery(tt.req)

			for _, want := range tt.wantContain {
				if !strings.Contains(query, want) {
					t.Errorf("Query does not contain expected substring: %q\nGot query: %s", want, query)
				}
			}
		})
	}
}

// Test buildIssuesArgs generates correct argument list
func TestBuildIssuesArgs(t *testing.T) {
	tests := []struct {
		name     string
		req      AgentIssuesRequest
		wantLen  int
		wantArgs []interface{}
	}{
		{
			name: "basic query",
			req: AgentIssuesRequest{
				Scenario: "test",
				Limit:    10,
			},
			wantLen:  2,
			wantArgs: []interface{}{"test", 10},
		},
		{
			name: "with file filter",
			req: AgentIssuesRequest{
				Scenario: "test",
				File:     "api/main.go",
				Limit:    10,
			},
			wantLen:  3,
			wantArgs: []interface{}{"test", "api/main.go", 10},
		},
		{
			name: "with folder filter",
			req: AgentIssuesRequest{
				Scenario: "test",
				Folder:   "api/",
				Limit:    10,
			},
			wantLen:  3,
			wantArgs: []interface{}{"test", "api/%", 10},
		},
		{
			name: "with category only",
			req: AgentIssuesRequest{
				Scenario: "test",
				Category: "dead_code",
				Limit:    5,
			},
			wantLen:  3,
			wantArgs: []interface{}{"test", "dead_code", 5},
		},
		{
			name: "with file and category",
			req: AgentIssuesRequest{
				Scenario: "test",
				File:     "api/main.go",
				Category: "length",
				Limit:    15,
			},
			wantLen:  4,
			wantArgs: []interface{}{"test", "api/main.go", "length", 15},
		},
		{
			name: "with folder and category",
			req: AgentIssuesRequest{
				Scenario: "test",
				Folder:   "api/",
				Category: "complexity",
				Limit:    20,
			},
			wantLen:  4,
			wantArgs: []interface{}{"test", "api/%", "complexity", 20},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildIssuesArgs(tt.req)

			if len(args) != tt.wantLen {
				t.Errorf("Args length = %v, want %v", len(args), tt.wantLen)
			}

			for i, wantArg := range tt.wantArgs {
				if i >= len(args) {
					t.Errorf("Missing arg at index %d", i)
					continue
				}
				if args[i] != wantArg {
					t.Errorf("Arg[%d] = %v, want %v", i, args[i], wantArg)
				}
			}
		})
	}
}
