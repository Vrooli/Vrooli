package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/browser-automation-studio/database"
)

// ============================================================================
// Health Endpoint Tests
// ============================================================================

func TestHealth_AllHealthy(t *testing.T) {
	handler, catalogSvc, _, repo, _, storageMock := createTestHandler()

	// Ensure all services are healthy
	catalogSvc.AutomationHealthy = true
	storageMock.Healthy = true

	// Add a project so ListProjects works
	repo.AddProject(&database.ProjectIndex{
		Name:       "test",
		FolderPath: "/tmp/test",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()

	handler.Health(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response HealthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Status != "healthy" {
		t.Errorf("expected status 'healthy', got %q", response.Status)
	}
	if !response.Readiness {
		t.Error("expected readiness to be true")
	}
	if response.Service != "browser-automation-studio-api" {
		t.Errorf("expected service name 'browser-automation-studio-api', got %q", response.Service)
	}
	if response.Timestamp == "" {
		t.Error("expected timestamp to be set")
	}
	if response.Version == "" {
		t.Error("expected version to be set")
	}
}

func TestHealth_DatabaseUnhealthy(t *testing.T) {
	handler, catalogSvc, _, repo, _, storageMock := createTestHandler()

	// Make database unhealthy by having ListProjects fail
	repo.ListProjectsError = errors.New("database connection failed")
	catalogSvc.AutomationHealthy = true
	storageMock.Healthy = true

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()

	handler.Health(rr, req)

	// Database issues should return 503 Service Unavailable
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", rr.Code, rr.Body.String())
	}

	var response HealthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Status != "unhealthy" {
		t.Errorf("expected status 'unhealthy', got %q", response.Status)
	}
	if response.Readiness {
		t.Error("expected readiness to be false when database is unhealthy")
	}
}

func TestHealth_AutomationUnhealthy(t *testing.T) {
	handler, catalogSvc, _, repo, _, storageMock := createTestHandler()

	catalogSvc.AutomationHealthy = false
	storageMock.Healthy = true
	repo.AddProject(&database.ProjectIndex{
		Name:       "test",
		FolderPath: "/tmp/test",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()

	handler.Health(rr, req)

	// Automation issues should return 200 but with degraded status
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response HealthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Status != "degraded" {
		t.Errorf("expected status 'degraded', got %q", response.Status)
	}
	// Readiness should still be true for degraded state
	if !response.Readiness {
		t.Error("expected readiness to be true for degraded state")
	}
}

func TestHealth_StorageUnhealthy(t *testing.T) {
	handler, catalogSvc, _, repo, _, storageMock := createTestHandler()

	catalogSvc.AutomationHealthy = true
	storageMock.Healthy = false
	storageMock.HealthCheckError = errors.New("storage connection failed")
	repo.AddProject(&database.ProjectIndex{
		Name:       "test",
		FolderPath: "/tmp/test",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()

	handler.Health(rr, req)

	// Storage issues should return 200 but with degraded status
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response HealthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Status != "degraded" {
		t.Errorf("expected status 'degraded', got %q", response.Status)
	}
}

func TestHealth_Dependencies(t *testing.T) {
	handler, catalogSvc, _, repo, _, storageMock := createTestHandler()

	catalogSvc.AutomationHealthy = true
	storageMock.Healthy = true
	repo.AddProject(&database.ProjectIndex{
		Name:       "test",
		FolderPath: "/tmp/test",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()

	handler.Health(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var response HealthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Dependencies == nil {
		t.Fatal("expected dependencies to be set")
	}

	// Check database dependency
	db, ok := response.Dependencies["database"].(map[string]any)
	if !ok {
		t.Fatal("expected database dependency")
	}
	if db["connected"] != true {
		t.Errorf("expected database connected to be true, got %v", db["connected"])
	}

	// Check storage dependency
	storage, ok := response.Dependencies["storage"].(map[string]any)
	if !ok {
		t.Fatal("expected storage dependency")
	}
	if storage["connected"] != true {
		t.Errorf("expected storage connected to be true, got %v", storage["connected"])
	}

	// Check external services
	extSvc, ok := response.Dependencies["external_services"].([]any)
	if !ok {
		t.Fatal("expected external_services dependency")
	}
	if len(extSvc) == 0 {
		t.Fatal("expected at least one external service")
	}
}

func TestHealth_Metrics(t *testing.T) {
	handler, catalogSvc, _, repo, _, storageMock := createTestHandler()

	catalogSvc.AutomationHealthy = true
	storageMock.Healthy = true
	repo.AddProject(&database.ProjectIndex{
		Name:       "test",
		FolderPath: "/tmp/test",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()

	handler.Health(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var response HealthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Metrics == nil {
		t.Fatal("expected metrics to be set")
	}

	// Check expected metrics are present
	expectedMetrics := []string{"goroutines", "uptime_seconds", "heap_alloc_mb"}
	for _, metric := range expectedMetrics {
		if _, ok := response.Metrics[metric]; !ok {
			t.Errorf("expected metric %q to be present", metric)
		}
	}

	// Verify goroutines is a positive number
	goroutines, ok := response.Metrics["goroutines"].(float64)
	if !ok {
		t.Fatal("expected goroutines to be a number")
	}
	if goroutines <= 0 {
		t.Errorf("expected goroutines > 0, got %v", goroutines)
	}

	// Verify uptime is non-negative
	uptime, ok := response.Metrics["uptime_seconds"].(float64)
	if !ok {
		t.Fatal("expected uptime_seconds to be a number")
	}
	if uptime < 0 {
		t.Errorf("expected uptime_seconds >= 0, got %v", uptime)
	}
}

func TestHealth_NilRepo(t *testing.T) {
	handler, catalogSvc, execSvc, _, hub, storageMock := createTestHandler()

	// Create handler with nil repo
	handler = &Handler{
		catalogService:   catalogSvc,
		executionService: execSvc,
		repo:             nil, // Nil repo
		wsHub:            hub,
		storage:          storageMock,
	}

	catalogSvc.AutomationHealthy = true
	storageMock.Healthy = true

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()

	handler.Health(rr, req)

	// Should return 503 when repo is nil
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rr.Code)
	}

	var response HealthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Status != "unhealthy" {
		t.Errorf("expected status 'unhealthy', got %q", response.Status)
	}
}

func TestHealth_NilStorage(t *testing.T) {
	handler, catalogSvc, execSvc, repo, hub, _ := createTestHandler()

	// Create handler with nil storage
	handler = &Handler{
		catalogService:   catalogSvc,
		executionService: execSvc,
		repo:             repo,
		wsHub:            hub,
		storage:          nil, // Nil storage
	}

	catalogSvc.AutomationHealthy = true
	repo.AddProject(&database.ProjectIndex{
		Name:       "test",
		FolderPath: "/tmp/test",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()

	handler.Health(rr, req)

	// Should return 200 but degraded when storage is nil
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var response HealthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Status != "degraded" {
		t.Errorf("expected status 'degraded', got %q", response.Status)
	}
}
