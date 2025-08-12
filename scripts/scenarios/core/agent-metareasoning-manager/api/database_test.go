package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Test database setup and teardown
func setupTestDB(t *testing.T) *sql.DB {
	// Use test database URL from environment or default
	testDBURL := os.Getenv("TEST_DATABASE_URL")
	if testDBURL == "" {
		testDBURL = "postgres://postgres:postgres@localhost:5432/metareasoning_test?sslmode=disable"
	}

	testDB, err := sql.Open("postgres", testDBURL)
	if err != nil {
		t.Skipf("Failed to connect to test database: %v", err)
	}

	// Create test schema
	createTestSchema(t, testDB)
	
	// Set global db variable for tests
	db = testDB
	return testDB
}

func createTestSchema(t *testing.T, testDB *sql.DB) {
	schema := `
	DROP TABLE IF EXISTS execution_history CASCADE;
	DROP TABLE IF EXISTS workflows CASCADE;
	
	CREATE TABLE IF NOT EXISTS workflows (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(100) NOT NULL,
		description TEXT,
		type VARCHAR(50) NOT NULL,
		platform VARCHAR(20) NOT NULL CHECK (platform IN ('n8n', 'windmill', 'both')),
		config JSONB NOT NULL,
		webhook_path VARCHAR(255),
		job_path VARCHAR(255),
		schema JSONB,
		estimated_duration_ms INTEGER DEFAULT 10000,
		version INTEGER DEFAULT 1,
		parent_id UUID REFERENCES workflows(id) ON DELETE SET NULL,
		is_active BOOLEAN DEFAULT true,
		is_builtin BOOLEAN DEFAULT false,
		tags TEXT[],
		embedding_id VARCHAR(100),
		usage_count INTEGER DEFAULT 0,
		success_count INTEGER DEFAULT 0,
		failure_count INTEGER DEFAULT 0,
		avg_execution_time_ms INTEGER,
		created_by VARCHAR(100) DEFAULT 'system',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(name, version)
	);

	CREATE TABLE IF NOT EXISTS execution_history (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		workflow_id UUID REFERENCES workflows(id) ON DELETE CASCADE,
		workflow_type VARCHAR(50) NOT NULL,
		resource_type VARCHAR(50) NOT NULL,
		resource_id VARCHAR(100) NOT NULL,
		input_data JSONB NOT NULL,
		output_data JSONB,
		execution_time_ms INTEGER,
		status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'running', 'success', 'failed', 'timeout')),
		model_used VARCHAR(50),
		error_message TEXT,
		metadata JSONB,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_workflows_active ON workflows(is_active);
	CREATE INDEX IF NOT EXISTS idx_workflows_platform ON workflows(platform);
	CREATE INDEX IF NOT EXISTS idx_workflows_type ON workflows(type);
	CREATE INDEX IF NOT EXISTS idx_execution_history_workflow ON execution_history(workflow_id);
	CREATE INDEX IF NOT EXISTS idx_execution_history_created ON execution_history(created_at);
	`

	_, err := testDB.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create test schema: %v", err)
	}
}

func cleanupTestDB(t *testing.T, testDB *sql.DB) {
	testDB.Exec("DROP TABLE IF EXISTS execution_history CASCADE")
	testDB.Exec("DROP TABLE IF EXISTS workflows CASCADE")
	testDB.Close()
}

func TestInitDatabase(t *testing.T) {
	testDBURL := os.Getenv("TEST_DATABASE_URL")
	if testDBURL == "" {
		testDBURL = "postgres://postgres:postgres@localhost:5432/metareasoning_test?sslmode=disable"
	}

	err := InitDatabase(testDBURL)
	if err != nil {
		t.Skipf("Failed to initialize database: %v", err)
	}
	defer CloseDatabase()

	// Test that connection works
	err = db.Ping()
	if err != nil {
		t.Errorf("Failed to ping database: %v", err)
	}
}

func TestListWorkflows(t *testing.T) {
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Insert test workflows
	testWorkflows := []struct {
		name     string
		platform string
		wfType   string
		active   bool
	}{
		{"Test Workflow 1", "n8n", "automation", true},
		{"Test Workflow 2", "windmill", "analysis", true},
		{"Test Workflow 3", "n8n", "automation", false},
		{"Test Workflow 4", "both", "monitoring", true},
	}

	for _, tw := range testWorkflows {
		_, err := testDB.Exec(`
			INSERT INTO workflows (name, description, type, platform, config, is_active, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, tw.name, "Test description", tw.wfType, tw.platform, `{}`, tw.active, "test")
		if err != nil {
			t.Fatalf("Failed to insert test workflow: %v", err)
		}
	}

	tests := []struct {
		name       string
		platform   string
		typeFilter string
		activeOnly bool
		page       int
		pageSize   int
		wantCount  int
		wantTotal  int
	}{
		{
			name:       "list all active",
			platform:   "",
			typeFilter: "",
			activeOnly: true,
			page:       1,
			pageSize:   10,
			wantCount:  3,
			wantTotal:  3,
		},
		{
			name:       "filter by platform",
			platform:   "n8n",
			typeFilter: "",
			activeOnly: true,
			page:       1,
			pageSize:   10,
			wantCount:  1,
			wantTotal:  1,
		},
		{
			name:       "filter by type",
			platform:   "",
			typeFilter: "automation",
			activeOnly: true,
			page:       1,
			pageSize:   10,
			wantCount:  1,
			wantTotal:  1,
		},
		{
			name:       "include inactive",
			platform:   "",
			typeFilter: "",
			activeOnly: false,
			page:       1,
			pageSize:   10,
			wantCount:  4,
			wantTotal:  4,
		},
		{
			name:       "pagination",
			platform:   "",
			typeFilter: "",
			activeOnly: false,
			page:       1,
			pageSize:   2,
			wantCount:  2,
			wantTotal:  4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflows, total, err := listWorkflows(tt.platform, tt.typeFilter, tt.activeOnly, tt.page, tt.pageSize)
			if err != nil {
				t.Fatalf("listWorkflows failed: %v", err)
			}

			if len(workflows) != tt.wantCount {
				t.Errorf("got %d workflows, want %d", len(workflows), tt.wantCount)
			}
			if total != tt.wantTotal {
				t.Errorf("got total %d, want %d", total, tt.wantTotal)
			}
		})
	}
}

func TestGetWorkflow(t *testing.T) {
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Insert a test workflow
	var workflowID uuid.UUID
	err := testDB.QueryRow(`
		INSERT INTO workflows (name, description, type, platform, config, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, "Test Get Workflow", "Test description", "test", "n8n", `{"test": true}`, "test").Scan(&workflowID)
	if err != nil {
		t.Fatalf("Failed to insert test workflow: %v", err)
	}

	// Test getting existing workflow
	workflow, err := getWorkflow(workflowID)
	if err != nil {
		t.Fatalf("getWorkflow failed: %v", err)
	}
	if workflow == nil {
		t.Fatal("Expected workflow, got nil")
	}
	if workflow.Name != "Test Get Workflow" {
		t.Errorf("got name %s, want 'Test Get Workflow'", workflow.Name)
	}

	// Test getting non-existent workflow
	nonExistentID := uuid.New()
	workflow, err = getWorkflow(nonExistentID)
	if err != nil {
		t.Fatalf("getWorkflow failed: %v", err)
	}
	if workflow != nil {
		t.Error("Expected nil workflow for non-existent ID")
	}
}

func TestCreateWorkflow(t *testing.T) {
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	wc := WorkflowCreate{
		Name:        "Test Create Workflow",
		Description: "Created in test",
		Type:        "automation",
		Platform:    "windmill",
		Config:      json.RawMessage(`{"nodes": [], "connections": []}`),
		Tags:        []string{"test", "created"},
	}

	workflow, err := createWorkflow(wc, "testuser")
	if err != nil {
		t.Fatalf("createWorkflow failed: %v", err)
	}

	if workflow.Name != wc.Name {
		t.Errorf("got name %s, want %s", workflow.Name, wc.Name)
	}
	if workflow.CreatedBy != "testuser" {
		t.Errorf("got created_by %s, want 'testuser'", workflow.CreatedBy)
	}
	if workflow.Version != 1 {
		t.Errorf("got version %d, want 1", workflow.Version)
	}
	if !workflow.IsActive {
		t.Error("Expected workflow to be active")
	}
}

func TestUpdateWorkflow(t *testing.T) {
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Create initial workflow
	wc := WorkflowCreate{
		Name:        "Test Update Workflow",
		Description: "Initial version",
		Type:        "analysis",
		Platform:    "n8n",
		Config:      json.RawMessage(`{"version": 1}`),
	}

	original, err := createWorkflow(wc, "testuser")
	if err != nil {
		t.Fatalf("Failed to create initial workflow: %v", err)
	}

	// Update the workflow
	update := WorkflowCreate{
		Name:        "Test Update Workflow",
		Description: "Updated version",
		Type:        "analysis",
		Platform:    "windmill",
		Config:      json.RawMessage(`{"version": 2}`),
	}

	updated, err := updateWorkflow(original.ID, update, "updater")
	if err != nil {
		t.Fatalf("updateWorkflow failed: %v", err)
	}

	if updated.Description != update.Description {
		t.Errorf("got description %s, want %s", updated.Description, update.Description)
	}
	if updated.Platform != update.Platform {
		t.Errorf("got platform %s, want %s", updated.Platform, update.Platform)
	}
	if updated.Version != 2 {
		t.Errorf("got version %d, want 2", updated.Version)
	}
	if updated.ParentID == nil || *updated.ParentID != original.ID {
		t.Error("Expected parent_id to reference original workflow")
	}

	// Verify original is now inactive
	orig, _ := getWorkflow(original.ID)
	if orig.IsActive {
		t.Error("Expected original workflow to be inactive after update")
	}
}

func TestDeleteWorkflow(t *testing.T) {
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Create a workflow
	wc := WorkflowCreate{
		Name:     "Test Delete Workflow",
		Type:     "test",
		Platform: "n8n",
		Config:   json.RawMessage(`{}`),
	}

	workflow, err := createWorkflow(wc, "testuser")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	// Delete it
	err = deleteWorkflow(workflow.ID)
	if err != nil {
		t.Fatalf("deleteWorkflow failed: %v", err)
	}

	// Verify it's inactive
	deleted, _ := getWorkflow(workflow.ID)
	if deleted.IsActive {
		t.Error("Expected workflow to be inactive after deletion")
	}
}

func TestSearchWorkflows(t *testing.T) {
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Insert searchable workflows
	testWorkflows := []struct {
		name        string
		description string
		tags        []string
	}{
		{"Data Analysis Pipeline", "Analyzes customer data", []string{"analytics", "customer"}},
		{"Email Automation", "Sends automated emails", []string{"email", "automation"}},
		{"Report Generator", "Generates daily reports", []string{"reporting", "daily"}},
	}

	for _, tw := range testWorkflows {
		tagsArray := fmt.Sprintf("{%s}", joinStrings(tw.tags, ","))
		_, err := testDB.Exec(`
			INSERT INTO workflows (name, description, type, platform, config, tags, created_by)
			VALUES ($1, $2, $3, $4, $5, $6::text[], $7)
		`, tw.name, tw.description, "test", "n8n", `{}`, tagsArray, "test")
		if err != nil {
			t.Fatalf("Failed to insert test workflow: %v", err)
		}
	}

	tests := []struct {
		query     string
		wantCount int
	}{
		{"analysis", 1},
		{"email", 1},
		{"report", 1},
		{"automation", 1},
		{"data", 1},
		{"nonexistent", 0},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			results, err := searchWorkflows(tt.query)
			if err != nil {
				t.Fatalf("searchWorkflows failed: %v", err)
			}
			if len(results) != tt.wantCount {
				t.Errorf("got %d results for query '%s', want %d", len(results), tt.query, tt.wantCount)
			}
		})
	}
}

func TestCloneWorkflow(t *testing.T) {
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Create original workflow
	original := WorkflowCreate{
		Name:        "Original Workflow",
		Description: "To be cloned",
		Type:        "automation",
		Platform:    "windmill",
		Config:      json.RawMessage(`{"original": true}`),
		Tags:        []string{"original", "test"},
	}

	orig, err := createWorkflow(original, "testuser")
	if err != nil {
		t.Fatalf("Failed to create original workflow: %v", err)
	}

	// Clone it
	clone, err := cloneWorkflow(orig.ID, "Cloned Workflow")
	if err != nil {
		t.Fatalf("cloneWorkflow failed: %v", err)
	}

	if clone.Name != "Cloned Workflow" {
		t.Errorf("got name %s, want 'Cloned Workflow'", clone.Name)
	}
	if clone.Type != orig.Type {
		t.Errorf("got type %s, want %s", clone.Type, orig.Type)
	}
	if clone.Platform != orig.Platform {
		t.Errorf("got platform %s, want %s", clone.Platform, orig.Platform)
	}
	if clone.ID == orig.ID {
		t.Error("Clone should have different ID")
	}
	if clone.ParentID == nil || *clone.ParentID != orig.ID {
		t.Error("Clone should reference original as parent")
	}
}

func TestRecordExecution(t *testing.T) {
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Create a workflow
	wc := WorkflowCreate{
		Name:     "Test Execution Workflow",
		Type:     "test",
		Platform: "n8n",
		Config:   json.RawMessage(`{}`),
	}

	workflow, err := createWorkflow(wc, "testuser")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	// Record successful execution
	req := ExecutionRequest{
		Input:    map[string]interface{}{"test": "input"},
		Model:    "test-model",
		Metadata: map[string]interface{}{"test": true},
	}
	resp := &ExecutionResponse{
		ID:          uuid.New(),
		WorkflowID:  workflow.ID,
		Status:      "success",
		Data:        map[string]interface{}{"test": "output"},
		ExecutionMS: 1500,
		Timestamp:   time.Now(),
	}
	
	err = recordExecution(workflow.ID, req, resp)
	if err != nil {
		t.Fatalf("recordExecution failed: %v", err)
	}

	// Verify metrics updated
	updated, _ := getWorkflow(workflow.ID)
	if updated.UsageCount != 1 {
		t.Errorf("got usage_count %d, want 1", updated.UsageCount)
	}
	if updated.SuccessCount != 1 {
		t.Errorf("got success_count %d, want 1", updated.SuccessCount)
	}

	// Record failed execution
	req2 := ExecutionRequest{
		Input:    map[string]interface{}{"test": "input"},
		Model:    "test-model",
		Metadata: map[string]interface{}{"test": true},
	}
	resp2 := &ExecutionResponse{
		ID:          uuid.New(),
		WorkflowID:  workflow.ID,
		Status:      "failed",
		Error:       "Test error",
		ExecutionMS: 500,
		Timestamp:   time.Now(),
	}
	
	err = recordExecution(workflow.ID, req2, resp2)
	if err != nil {
		t.Fatalf("recordExecution (failure) failed: %v", err)
	}

	// Verify metrics updated
	updated, _ = getWorkflow(workflow.ID)
	if updated.UsageCount != 2 {
		t.Errorf("got usage_count %d, want 2", updated.UsageCount)
	}
	if updated.FailureCount != 1 {
		t.Errorf("got failure_count %d, want 1", updated.FailureCount)
	}
}

func TestGetExecutionHistory(t *testing.T) {
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Create a workflow
	wc := WorkflowCreate{
		Name:     "Test History Workflow",
		Type:     "test",
		Platform: "n8n",
		Config:   json.RawMessage(`{}`),
	}

	workflow, err := createWorkflow(wc, "testuser")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	// Record multiple executions
	for i := 0; i < 5; i++ {
		status := "success"
		if i%2 == 1 {
			status = "failed"
		}
		req := ExecutionRequest{
			Input:    map[string]interface{}{"iteration": i},
			Model:    "test-model",
		}
		resp := &ExecutionResponse{
			ID:          uuid.New(),
			WorkflowID:  workflow.ID,
			Status:      status,
			Data:        map[string]interface{}{"result": i * 2},
			ExecutionMS: int(1000 + i*100),
			Timestamp:   time.Now(),
		}
		err := recordExecution(workflow.ID, req, resp)
		if err != nil {
			t.Fatalf("Failed to record execution %d: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Get history
	history, err := getExecutionHistory(workflow.ID, 10)
	if err != nil {
		t.Fatalf("getExecutionHistory failed: %v", err)
	}

	if len(history) != 5 {
		t.Errorf("got %d history entries, want 5", len(history))
	}

	// Test limit
	limited, err := getExecutionHistory(workflow.ID, 3)
	if err != nil {
		t.Fatalf("getExecutionHistory with limit failed: %v", err)
	}
	if len(limited) != 3 {
		t.Errorf("got %d limited history entries, want 3", len(limited))
	}
}

func TestGetWorkflowMetrics(t *testing.T) {
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Create a workflow
	wc := WorkflowCreate{
		Name:     "Test Metrics Workflow",
		Type:     "test",
		Platform: "n8n",
		Config:   json.RawMessage(`{}`),
	}

	workflow, err := createWorkflow(wc, "testuser")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	// Record executions with varying durations
	durations := []int64{500, 1000, 1500, 2000, 750}
	for i, duration := range durations {
		status := "success"
		if i == 2 {
			status = "failed" // One failure
		}
		req := ExecutionRequest{
			Input:    map[string]interface{}{"test": "input"},
			Model:    "test-model",
		}
		resp := &ExecutionResponse{
			ID:          uuid.New(),
			WorkflowID:  workflow.ID,
			Status:      status,
			ExecutionMS: int(duration),
			Timestamp:   time.Now(),
		}
		err := recordExecution(workflow.ID, req, resp)
		if err != nil {
			t.Fatalf("Failed to record execution: %v", err)
		}
	}

	// Get metrics
	metrics, err := getWorkflowMetrics(workflow.ID)
	if err != nil {
		t.Fatalf("getWorkflowMetrics failed: %v", err)
	}

	if metrics.TotalExecutions != 5 {
		t.Errorf("got total_executions %d, want 5", metrics.TotalExecutions)
	}
	if metrics.SuccessCount != 4 {
		t.Errorf("got success_count %d, want 4", metrics.SuccessCount)
	}
	if metrics.FailureCount != 1 {
		t.Errorf("got failure_count %d, want 1", metrics.FailureCount)
	}
	if metrics.SuccessRate != 80.0 {
		t.Errorf("got success_rate %f, want 80.0", metrics.SuccessRate)
	}
	if metrics.MinExecutionTime != 500 {
		t.Errorf("got min_duration %d, want 500", metrics.MinExecutionTime)
	}
	if metrics.MaxExecutionTime != 2000 {
		t.Errorf("got max_duration %d, want 2000", metrics.MaxExecutionTime)
	}
}

func TestGetSystemStats(t *testing.T) {
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Create test workflows with executions
	platforms := []string{"n8n", "windmill", "n8n"}
	for i, platform := range platforms {
		wc := WorkflowCreate{
			Name:     fmt.Sprintf("Stats Workflow %d", i),
			Type:     "test",
			Platform: platform,
			Config:   json.RawMessage(`{}`),
		}

		workflow, err := createWorkflow(wc, "testuser")
		if err != nil {
			t.Fatalf("Failed to create workflow: %v", err)
		}

		// Add executions
		for j := 0; j < i+1; j++ {
			req := ExecutionRequest{
				Input: map[string]interface{}{"batch": i},
				Model: "test-model",
			}
			resp := &ExecutionResponse{
				ID:          uuid.New(),
				WorkflowID:  workflow.ID,
				Status:      "success",
				ExecutionMS: 1000,
				Timestamp:   time.Now(),
			}
			err := recordExecution(workflow.ID, req, resp)
			if err != nil {
				t.Fatalf("Failed to record execution: %v", err)
			}
		}
	}

	// Get system stats
	stats, err := getSystemStats()
	if err != nil {
		t.Fatalf("getSystemStats failed: %v", err)
	}

	if stats.TotalWorkflows != 3 {
		t.Errorf("got total_workflows %d, want 3", stats.TotalWorkflows)
	}
	if stats.ActiveWorkflows != 3 {
		t.Errorf("got active_workflows %d, want 3", stats.ActiveWorkflows)
	}
	if stats.TotalExecutions != 6 { // 1 + 2 + 3
		t.Errorf("got total_executions %d, want 6", stats.TotalExecutions)
	}
	// Note: PlatformStats field doesn't exist in StatsResponse,
	// but we can verify other stats
	if stats.TotalWorkflows == 0 {
		t.Error("Expected some workflows to be counted")
	}
}

// Helper function
func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}