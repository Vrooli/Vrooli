package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// [REQ:DM-P0-001,DM-P0-003,DM-P0-007] TestEndToEndDependencyAnalysisAndSwap tests complete workflow
func TestEndToEndDependencyAnalysisAndSwap(t *testing.T) {
	srv, mock := setupTestServer(t)
	defer srv.DB.Close()
	_ = mock

	scenarioName := "test-scenario"

	// Step 1: Analyze dependencies
	// Note: This endpoint requires an external analyzer service
	t.Run("analyze_dependencies", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/dependencies/analyze/"+scenarioName, nil)
		w := httptest.NewRecorder()
		srv.Router.ServeHTTP(w, req)

		// Without external analyzer, expect 503 (service unavailable)
		// With analyzer, would get 200 with scenario field
		if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 200 or 503, got %d", w.Code)
		}

		// Verify response is valid JSON
		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// If successful, should contain scenario field
		if w.Code == http.StatusOK {
			if scenario, ok := response["scenario"].(string); !ok || scenario == "" {
				t.Error("successful response should contain non-empty 'scenario' field")
			}
		}
	})

	// Step 2: Score fitness for a specific tier
	t.Run("score_fitness", func(t *testing.T) {
		payload := map[string]interface{}{
			"scenario": scenarioName,
			"tiers":    []int{2},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/fitness/score", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.Router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		// Verify fitness response structure
		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Should contain scores map (not fitness_score)
		if _, ok := response["scores"]; !ok {
			t.Error("response should contain 'scores' field")
		}
		// Should contain scenario
		if _, ok := response["scenario"]; !ok {
			t.Error("response should contain 'scenario' field")
		}
	})

	// Step 3: Get swap analysis
	t.Run("analyze_swaps", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/swaps/analyze/postgres/sqlite", nil)
		w := httptest.NewRecorder()
		srv.Router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		// Verify swap analysis response
		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Should contain from and to fields
		if _, ok := response["from"]; !ok {
			t.Error("response should contain 'from' field")
		}
		if _, ok := response["to"]; !ok {
			t.Error("response should contain 'to' field")
		}
		// Should contain fitness_delta
		if _, ok := response["fitness_delta"]; !ok {
			t.Error("response should contain 'fitness_delta' field")
		}
	})
}

// [REQ:DM-P0-012,DM-P0-014,DM-P0-023,DM-P0-028] TestEndToEndProfileDeploymentWorkflow tests complete deployment flow
func TestEndToEndProfileDeploymentWorkflow(t *testing.T) {
	srv, mock := setupTestServer(t)
	defer srv.DB.Close()

	// Mock profile creation
	mock.ExpectExec("INSERT INTO profiles").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO profile_versions").
		WillReturnResult(sqlmock.NewResult(1, 1))

	var profileID string

	// Step 1: Create deployment profile
	t.Run("create_profile", func(t *testing.T) {
		payload := map[string]interface{}{
			"name":     "test-profile",
			"scenario": "test-scenario",
			"tiers":    []int{1, 2},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/profiles", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.Router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if id, ok := resp["id"].(string); ok && id != "" {
			profileID = id
		} else {
			t.Fatal("response should contain a valid 'id' field")
		}

		// Verify version is set
		if version, ok := resp["version"].(float64); !ok || version != 1 {
			t.Errorf("expected version 1, got %v", resp["version"])
		}
	})

	// Step 2: Validate profile - need to mock the get operation
	t.Run("validate_profile", func(t *testing.T) {
		if profileID == "" {
			t.Skip("profile ID not set from previous step")
		}

		// Mock the database call for validation
		now := time.Now()
		rows := sqlmock.NewRows([]string{"scenario", "tier"}).
			AddRow("test-scenario", 2)
		mock.ExpectQuery("SELECT scenario").WillReturnRows(rows)

		req := httptest.NewRequest("GET", "/api/v1/profiles/"+profileID+"/validate", nil)
		w := httptest.NewRecorder()
		srv.Router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Verify validation response structure
		if status, ok := response["status"].(string); !ok || status == "" {
			t.Error("response should contain 'status' field")
		}
		if _, ok := response["checks"].([]interface{}); !ok {
			t.Error("response should contain 'checks' array")
		}

		_ = now // silence unused variable
	})

	// Step 3: Deploy profile
	t.Run("deploy_profile", func(t *testing.T) {
		if profileID == "" {
			t.Skip("profile ID not set from previous step")
		}

		req := httptest.NewRequest("POST", "/api/v1/deploy/"+profileID, nil)
		w := httptest.NewRecorder()
		srv.Router.ServeHTTP(w, req)

		// Deployment endpoint should exist and respond
		if w.Code == http.StatusNotFound {
			t.Error("deployment endpoint should be registered")
		}

		// Verify response is valid JSON
		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
	})
}

// [REQ:DM-P0-007,DM-P0-008,DM-P0-010] TestSwapImpactAnalysisWorkflow tests swap analysis flow
func TestSwapImpactAnalysisWorkflow(t *testing.T) {
	srv, mock := setupTestServer(t)
	defer srv.DB.Close()
	_ = mock

	// Get swap analysis
	t.Run("get_swap_analysis", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/swaps/analyze/postgres/sqlite", nil)
		w := httptest.NewRecorder()
		srv.Router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		// Verify response structure
		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Should have from/to fields
		if _, ok := response["from"]; !ok {
			t.Error("response should contain 'from' field")
		}
		if _, ok := response["to"]; !ok {
			t.Error("response should contain 'to' field")
		}
		// Should have fitness_delta
		if _, ok := response["fitness_delta"]; !ok {
			t.Error("response should contain 'fitness_delta' field")
		}
	})

	// Get cascade impact
	t.Run("get_cascade_impact", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/swaps/cascade/postgres/sqlite", nil)
		w := httptest.NewRecorder()
		srv.Router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		// Verify cascade response structure
		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Should contain cascading_impacts (not affected_services)
		if _, ok := response["cascading_impacts"]; !ok {
			t.Error("response should contain 'cascading_impacts' field")
		}
		// Should contain from and to
		if _, ok := response["from"]; !ok {
			t.Error("response should contain 'from' field")
		}
		if _, ok := response["to"]; !ok {
			t.Error("response should contain 'to' field")
		}
	})
}

// [REQ:DM-P0-018,DM-P0-019,DM-P0-021] TestSecretManagementWorkflow tests secret handling flow
func TestSecretManagementWorkflow(t *testing.T) {
	srv, mock := setupTestServer(t)
	defer srv.DB.Close()

	profileID := "profile-1"

	// Step 1: Identify secrets for a profile
	t.Run("identify_secrets", func(t *testing.T) {
		// Mock the profile lookup
		rows := sqlmock.NewRows([]string{"scenario", "tier"}).
			AddRow("test-scenario", 2)
		mock.ExpectQuery("SELECT scenario").WillReturnRows(rows)

		req := httptest.NewRequest("GET", "/api/v1/profiles/"+profileID+"/secrets", nil)
		w := httptest.NewRecorder()
		srv.Router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		// Verify response is valid JSON with expected structure
		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if _, ok := response["secrets"]; !ok {
			t.Error("response should contain 'secrets' field")
		}
		if _, ok := response["profile_id"]; !ok {
			t.Error("response should contain 'profile_id' field")
		}
	})

	// Step 2: Generate secret template
	t.Run("generate_secret_template", func(t *testing.T) {
		// Mock the profile lookup
		rows := sqlmock.NewRows([]string{"scenario", "tier"}).
			AddRow("test-scenario", 2)
		mock.ExpectQuery("SELECT scenario").WillReturnRows(rows)

		req := httptest.NewRequest("GET", "/api/v1/profiles/"+profileID+"/secrets/template", nil)
		w := httptest.NewRecorder()
		srv.Router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		// Verify response is valid JSON
		var response interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
	})
}

// [REQ:DM-P0-015,DM-P0-016,DM-P0-017] TestProfileVersionControlWorkflow tests version management
func TestProfileVersionControlWorkflow(t *testing.T) {
	srv, mock := setupTestServer(t)
	defer srv.DB.Close()

	profileID := "profile-1"

	// Get version history
	t.Run("get_version_history", func(t *testing.T) {
		now := time.Now()

		// First, mock the ID resolution query
		idRows := sqlmock.NewRows([]string{"id"}).AddRow(profileID)
		mock.ExpectQuery("SELECT id FROM profiles").WillReturnRows(idRows)

		// Then mock the version history query
		versionRows := sqlmock.NewRows([]string{"profile_id", "version", "name", "scenario", "tiers", "swaps", "secrets", "settings", "created_at", "created_by", "change_description"}).
			AddRow(profileID, 2, "Test Profile", "test-scenario", []byte("[1,2,3]"), []byte("{}"), []byte("{}"), []byte("{}"), now, "system", "Added tier 3").
			AddRow(profileID, 1, "Test Profile", "test-scenario", []byte("[1,2]"), []byte("{}"), []byte("{}"), []byte("{}"), now, "system", "Initial version")

		mock.ExpectQuery("SELECT (.+) FROM profile_versions").WillReturnRows(versionRows)

		req := httptest.NewRequest("GET", "/api/v1/profiles/"+profileID+"/versions", nil)
		w := httptest.NewRecorder()
		srv.Router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Should contain profile_id and versions
		if response["profile_id"] != profileID {
			t.Errorf("expected profile_id %s, got %v", profileID, response["profile_id"])
		}

		versions, ok := response["versions"].([]interface{})
		if !ok {
			t.Fatal("response should contain 'versions' array")
		}

		if len(versions) != 2 {
			t.Errorf("expected 2 versions, got %d", len(versions))
		}
	})

	// Verify mock expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled mock expectations: %v", err)
	}
}

// TestInvalidScenarioAnalysis tests error handling for invalid scenarios
func TestInvalidScenarioAnalysis(t *testing.T) {
	srv, mock := setupTestServer(t)
	defer srv.DB.Close()
	_ = mock

	t.Run("empty_scenario_name", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/dependencies/analyze/", nil)
		w := httptest.NewRecorder()
		srv.Router.ServeHTTP(w, req)

		// Empty scenario should return 404 (route not matched) or 400
		if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
			t.Errorf("expected status 404 or 400, got %d", w.Code)
		}
	})

	t.Run("invalid_fitness_payload", func(t *testing.T) {
		// Malformed JSON
		req := httptest.NewRequest("POST", "/api/v1/fitness/score", bytes.NewReader([]byte("{invalid")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.Router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

// TestConcurrentRequests tests handling of concurrent requests
func TestConcurrentRequests(t *testing.T) {
	srv, mock := setupTestServer(t)
	defer srv.DB.Close()
	_ = mock

	// Run multiple requests concurrently
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/api/v1/swaps/analyze/postgres/sqlite", nil)
			w := httptest.NewRecorder()
			srv.Router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("concurrent request failed with status %d", w.Code)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}
}
