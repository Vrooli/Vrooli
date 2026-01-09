package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"app-monitor-api/repository"
)

func TestSubmitIssueToTrackerReturnsIssueID(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/issues" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode payload: %v", err)
		}
		if payload["title"] == "" {
			t.Fatalf("expected title field to be populated")
		}

		resp := map[string]interface{}{
			"success": true,
			"message": "created",
			"data": map[string]interface{}{
				"issue_id": "issue-abc",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	parts := strings.Split(server.URL, ":")
	if len(parts) < 3 {
		t.Fatalf("unexpected server URL: %s", server.URL)
	}
	portStr := parts[len(parts)-1]
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("failed to parse port: %v", err)
	}

	service := NewAppService(nil)
	result, err := service.submitIssueToTracker(context.Background(), port, map[string]interface{}{"title": "test"})
	if err != nil {
		t.Fatalf("submitIssueToTracker returned error: %v", err)
	}
	if result == nil || result.IssueID != "issue-abc" {
		t.Fatalf("expected issue-abc, got %#v", result)
	}
}

func TestSubmitIssueToTrackerParsesNestedIssueID(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"success": true,
			"message": "created",
			"data": map[string]interface{}{
				"issue": map[string]interface{}{
					"id": "issue-nested",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	parts := strings.Split(server.URL, ":")
	if len(parts) < 3 {
		t.Fatalf("unexpected server URL: %s", server.URL)
	}
	portStr := parts[len(parts)-1]
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("failed to parse port: %v", err)
	}

	service := NewAppService(nil)
	result, err := service.submitIssueToTracker(context.Background(), port, map[string]interface{}{"title": "test"})
	if err != nil {
		t.Fatalf("submitIssueToTracker returned error: %v", err)
	}
	if result.IssueID != "issue-nested" {
		t.Fatalf("expected issue-nested, got %q", result.IssueID)
	}
}

// Mock repository for testing
type mockAppRepository struct {
	apps []repository.App
}

func (m *mockAppRepository) GetApps(ctx context.Context) ([]repository.App, error) {
	return m.apps, nil
}

func (m *mockAppRepository) GetApp(ctx context.Context, id string) (*repository.App, error) {
	for _, app := range m.apps {
		if app.ID == id {
			return &app, nil
		}
	}
	return nil, ErrAppNotFound
}

func (m *mockAppRepository) CreateApp(ctx context.Context, app *repository.App) error {
	m.apps = append(m.apps, *app)
	return nil
}

func (m *mockAppRepository) UpdateApp(ctx context.Context, app *repository.App) error {
	return nil
}

func (m *mockAppRepository) UpdateAppStatus(ctx context.Context, id string, status string) error {
	return nil
}

func (m *mockAppRepository) DeleteApp(ctx context.Context, id string) error {
	return nil
}

func (m *mockAppRepository) CreateAppStatus(ctx context.Context, status *repository.AppStatus) error {
	return nil
}

func (m *mockAppRepository) GetAppStatus(ctx context.Context, appID string) (*repository.AppStatus, error) {
	return nil, nil
}

func (m *mockAppRepository) GetAppStatusHistory(ctx context.Context, appID string, hours int) ([]repository.AppStatus, error) {
	return []repository.AppStatus{}, nil
}

func (m *mockAppRepository) CreateAppLog(ctx context.Context, log *repository.AppLog) error {
	return nil
}

func (m *mockAppRepository) GetAppLogs(ctx context.Context, appID string, limit, offset int) ([]repository.AppLog, error) {
	return []repository.AppLog{}, nil
}

func (m *mockAppRepository) GetAppLogsByLevel(ctx context.Context, appID string, level string, limit int) ([]repository.AppLog, error) {
	return []repository.AppLog{}, nil
}

func (m *mockAppRepository) RecordAppView(ctx context.Context, scenarioName string) (*repository.AppViewStats, error) {
	return nil, nil
}

func (m *mockAppRepository) GetAppViewStats(ctx context.Context) (map[string]repository.AppViewStats, error) {
	return map[string]repository.AppViewStats{}, nil
}

func TestNewAppService(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		repo := &mockAppRepository{}
		service := NewAppService(repo)

		if service == nil {
			t.Fatal("Expected non-nil app service")
		}

		if service.repo == nil {
			t.Error("Expected repository to be set")
		}

		if service.cache == nil {
			t.Error("Expected cache to be initialized")
		}

		if service.viewStats == nil {
			t.Error("Expected viewStats to be initialized")
		}
	})

	t.Run("NilRepository", func(t *testing.T) {
		service := NewAppService(nil)

		if service == nil {
			t.Fatal("Expected non-nil app service")
		}

		if service.repo != nil {
			t.Error("Expected repository to be nil")
		}
	})
}

func TestAppServiceErrors(t *testing.T) {
	t.Run("ErrorTypes", func(t *testing.T) {
		if ErrAppIdentifierRequired == nil {
			t.Error("Expected ErrAppIdentifierRequired to be defined")
		}

		if ErrAppNotFound == nil {
			t.Error("Expected ErrAppNotFound to be defined")
		}

		if ErrScenarioAuditorUnavailable == nil {
			t.Error("Expected ErrScenarioAuditorUnavailable to be defined")
		}

		if ErrScenarioBridgeScenarioMissing == nil {
			t.Error("Expected ErrScenarioBridgeScenarioMissing to be defined")
		}
	})
}

func TestBridgeRuleViolation(t *testing.T) {
	t.Run("ValidateStructure", func(t *testing.T) {
		violation := BridgeRuleViolation{
			RuleID:         "rule-one",
			Type:           "error",
			Title:          "Test Error",
			Description:    "Test description",
			FilePath:       "/path/to/file",
			Line:           10,
			Recommendation: "Fix it",
			Severity:       "high",
			Standard:       "test-standard",
		}

		if violation.Type != "error" {
			t.Errorf("Expected type 'error', got %s", violation.Type)
		}

		if violation.RuleID != "rule-one" {
			t.Errorf("Expected rule_id 'rule-one', got %s", violation.RuleID)
		}

		if violation.Line != 10 {
			t.Errorf("Expected line 10, got %d", violation.Line)
		}
	})
}

func TestBridgeRuleReport(t *testing.T) {
	t.Run("ValidateStructure", func(t *testing.T) {
		report := BridgeRuleReport{
			RuleID:       "test-rule",
			Name:         "Test Rule",
			Scenario:     "test-scenario",
			FilesScanned: 5,
			DurationMs:   100,
			Warnings:     []string{"warn"},
			Violations:   []BridgeRuleViolation{},
		}

		if report.RuleID != "test-rule" {
			t.Errorf("Expected rule_id 'test-rule', got %s", report.RuleID)
		}

		if report.FilesScanned != 5 {
			t.Errorf("Expected 5 files scanned, got %d", report.FilesScanned)
		}

		if report.Name != "Test Rule" {
			t.Errorf("Expected rule name 'Test Rule', got %s", report.Name)
		}

		if len(report.Warnings) != 1 {
			t.Errorf("Expected 1 warning, got %d", len(report.Warnings))
		}
	})
}

func TestBridgeDiagnosticsReport(t *testing.T) {
	t.Run("ValidateStructure", func(t *testing.T) {
		report := BridgeDiagnosticsReport{
			Scenario:     "demo",
			CheckedAt:    time.Now(),
			FilesScanned: 10,
			DurationMs:   250,
			Warnings:     []string{"rule1"},
			Violations: []BridgeRuleViolation{
				{RuleID: "demo", Type: "test", Title: "Test", Description: "desc"},
			},
			Results: []BridgeRuleReport{{RuleID: "demo-rule"}},
		}

		if report.FilesScanned != 10 {
			t.Errorf("Expected 10 files scanned, got %d", report.FilesScanned)
		}

		if len(report.Results) != 1 {
			t.Errorf("Expected 1 rule result, got %d", len(report.Results))
		}

		if report.Results[0].RuleID != "demo-rule" {
			t.Errorf("Expected rule id 'demo-rule', got %s", report.Results[0].RuleID)
		}
	})
}

func TestAppLogsResult(t *testing.T) {
	t.Run("ValidateStructure", func(t *testing.T) {
		logs := AppLogsResult{
			Lifecycle: []string{"line1", "line2"},
			Background: []BackgroundLog{
				{
					Step:    "test-step",
					Phase:   "test-phase",
					Label:   "test-label",
					Command: "test-command",
					Lines:   []string{"log1", "log2"},
				},
			},
		}

		if len(logs.Lifecycle) != 2 {
			t.Errorf("Expected 2 lifecycle lines, got %d", len(logs.Lifecycle))
		}

		if len(logs.Background) != 1 {
			t.Errorf("Expected 1 background log, got %d", len(logs.Background))
		}

		if logs.Background[0].Step != "test-step" {
			t.Errorf("Expected step 'test-step', got %s", logs.Background[0].Step)
		}
	})
}

func TestBackgroundLog(t *testing.T) {
	t.Run("ValidateStructure", func(t *testing.T) {
		log := BackgroundLog{
			Step:    "build",
			Phase:   "compile",
			Label:   "Build Project",
			Command: "go build",
			Lines:   []string{"Building...", "Done"},
		}

		if log.Step != "build" {
			t.Errorf("Expected step 'build', got %s", log.Step)
		}

		if len(log.Lines) != 2 {
			t.Errorf("Expected 2 lines, got %d", len(log.Lines))
		}
	})
}

func TestOrchestratorResponse(t *testing.T) {
	t.Run("ValidateStructure", func(t *testing.T) {
		resp := OrchestratorResponse{
			Success: true,
			Summary: struct {
				TotalScenarios int `json:"total_scenarios"`
				Running        int `json:"running"`
				Stopped        int `json:"stopped"`
			}{
				TotalScenarios: 10,
				Running:        5,
				Stopped:        5,
			},
			Scenarios: []OrchestratorApp{},
		}

		if !resp.Success {
			t.Error("Expected success to be true")
		}

		if resp.Summary.TotalScenarios != 10 {
			t.Errorf("Expected 10 total scenarios, got %d", resp.Summary.TotalScenarios)
		}
	})
}

func TestOrchestratorApp(t *testing.T) {
	t.Run("ValidateStructure", func(t *testing.T) {
		healthStatus := "healthy"
		app := OrchestratorApp{
			Name:         "test-app",
			DisplayName:  "Test App",
			Description:  "Test description",
			Status:       "running",
			HealthStatus: &healthStatus,
			Ports:        map[string]int{"api": 8080},
			Processes:    2,
			Runtime:      "go",
			StartedAt:    "2025-01-01T00:00:00Z",
		}

		if app.Name != "test-app" {
			t.Errorf("Expected name 'test-app', got %s", app.Name)
		}

		if app.Processes != 2 {
			t.Errorf("Expected 2 processes, got %d", app.Processes)
		}

		if *app.HealthStatus != "healthy" {
			t.Errorf("Expected health status 'healthy', got %s", *app.HealthStatus)
		}
	})
}

func TestAppServiceInvalidateCache(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		repo := &mockAppRepository{}
		service := NewAppService(repo)

		// Set some cache data
		service.cache.data = []repository.App{
			{ID: "test", Name: "Test"},
		}

		service.invalidateCache()

		// Cache timestamp should be zero
		if !service.cache.timestamp.IsZero() {
			t.Error("Expected cache timestamp to be zero after invalidation")
		}

		if !service.cache.isPartial {
			t.Error("Expected cache to be marked as partial")
		}
	})

	t.Run("NilService", func(t *testing.T) {
		var service *AppService

		// Should not panic
		service.invalidateCache()
	})

	t.Run("NilCache", func(t *testing.T) {
		service := &AppService{
			cache: nil,
		}

		// Should not panic
		service.invalidateCache()
	})
}
