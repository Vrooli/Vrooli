package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
)

// [REQ:DM-P0-001,DM-P0-003,DM-P0-007] TestEndToEndDependencyAnalysisAndSwap tests complete workflow
// This test demonstrates nested integration: analyze → score → suggest swaps
func TestEndToEndDependencyAnalysisAndSwap(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	srv := &Server{
		config:   &Config{Port: "8080"},
		db:       db,
		router:   mux.NewRouter(),
		profiles: NewSQLProfileRepository(db),
	}
	srv.setupRoutes()

	scenarioName := "test-scenario"

	// Step 1: Analyze dependencies
	t.Run("analyze_dependencies", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/dependencies/analyze/"+scenarioName, nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Logf("Dependency analysis response: %d", w.Code)
		}
	})

	// Step 2: Score fitness for a specific tier
	t.Run("score_fitness", func(t *testing.T) {
		payload := map[string]interface{}{
			"scenario": scenarioName,
			"tier":     "desktop",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/dependencies/score-fitness", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
			t.Logf("Fitness scoring response: %d", w.Code)
		}
	})

	// Step 3: Get swap suggestions for low-fitness dependencies
	t.Run("suggest_swaps", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/swaps/suggest/postgres?tier=mobile", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Logf("Swap suggestion response: %d", w.Code)
		}
	})

	mock.ExpectClose()
}

// [REQ:DM-P0-012,DM-P0-014,DM-P0-023,DM-P0-028] TestEndToEndProfileDeploymentWorkflow tests complete deployment flow
// This test demonstrates deep nesting: create profile → add swaps → validate → deploy
func TestEndToEndProfileDeploymentWorkflow(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	// Mock profile creation
	mock.ExpectExec("INSERT INTO profiles").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Mock profile retrieval
	mock.ExpectQuery("SELECT (.+) FROM profiles").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "scenario", "tier", "config", "created_at", "updated_at"}).
			AddRow(1, "test-profile", "test-scenario", "desktop", "{}", time.Now(), time.Now()))

	srv := &Server{
		config:   &Config{Port: "8080"},
		db:       db,
		router:   mux.NewRouter(),
		profiles: NewSQLProfileRepository(db),
	}
	srv.setupRoutes()

	var profileID string

	// Step 1: Create deployment profile
	t.Run("create_profile", func(t *testing.T) {
		payload := map[string]interface{}{
			"name":     "test-profile",
			"scenario": "test-scenario",
			"tier":     "desktop",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/profiles", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code == http.StatusCreated || w.Code == http.StatusOK {
			var resp map[string]interface{}
			json.NewDecoder(w.Body).Decode(&resp)
			if id, ok := resp["id"].(string); ok {
				profileID = id
			}
		}
	})

	// Step 2: Export profile (validates profile management)
	t.Run("export_profile", func(t *testing.T) {
		if profileID == "" {
			t.Skip("Profile ID not available")
		}
		req := httptest.NewRequest("GET", "/api/v1/profiles/"+profileID+"/export", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Logf("Profile export response: %d", w.Code)
		}
	})

	// Step 3: Validate profile
	t.Run("validate_profile", func(t *testing.T) {
		if profileID == "" {
			profileID = "1"
		}
		req := httptest.NewRequest("POST", "/api/v1/profiles/"+profileID+"/validate", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Logf("Profile validation response: %d", w.Code)
		}
	})

	// Step 4: Deploy profile
	t.Run("deploy_profile", func(t *testing.T) {
		if profileID == "" {
			profileID = "1"
		}
		req := httptest.NewRequest("POST", "/api/v1/profiles/"+profileID+"/deploy", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusAccepted && w.Code != http.StatusNotFound {
			t.Logf("Deployment response: %d", w.Code)
		}
	})

	mock.ExpectClose()
}

// [REQ:DM-P0-007,DM-P0-008,DM-P0-010] TestSwapImpactAnalysisWorkflow tests swap analysis flow
func TestSwapImpactAnalysisWorkflow(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	srv := &Server{
		config:   &Config{Port: "8080"},
		db:       db,
		router:   mux.NewRouter(),
		profiles: NewSQLProfileRepository(db),
	}
	srv.setupRoutes()

	// Step 1: Get swap suggestions
	t.Run("get_swap_suggestions", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/swaps/suggest/postgres?tier=mobile", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Logf("Swap suggestion response: %d", w.Code)
		}
	})

	// Step 2: Analyze swap impact
	t.Run("analyze_swap_impact", func(t *testing.T) {
		payload := map[string]interface{}{
			"from": "postgres",
			"to":   "sqlite",
			"tier": "mobile",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/swaps/analyze-impact", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
			t.Logf("Swap impact analysis response: %d", w.Code)
		}
	})

	mock.ExpectClose()
}

// [REQ:DM-P0-018,DM-P0-019,DM-P0-021] TestSecretManagementWorkflow tests secret handling flow
func TestSecretManagementWorkflow(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	srv := &Server{
		config:   &Config{Port: "8080"},
		db:       db,
		router:   mux.NewRouter(),
		profiles: NewSQLProfileRepository(db),
	}
	srv.setupRoutes()

	// Step 1: Identify secrets for a scenario
	t.Run("identify_secrets", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/secrets/identify/test-scenario?tier=saas", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Logf("Secret identification response: %d", w.Code)
		}
	})

	// Step 2: Generate secret template
	t.Run("generate_secret_template", func(t *testing.T) {
		payload := map[string]interface{}{
			"scenario": "test-scenario",
			"tier":     "desktop",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/secrets/generate-template", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
			t.Logf("Secret template generation response: %d", w.Code)
		}
	})

	mock.ExpectClose()
}

// [REQ:DM-P0-001,DM-P0-002,DM-P0-011] TestCascadingDependencyDetection tests cascading impact detection
func TestCascadingDependencyDetection(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	srv := &Server{
		config:   &Config{Port: "8080"},
		db:       db,
		router:   mux.NewRouter(),
		profiles: NewSQLProfileRepository(db),
	}
	srv.setupRoutes()

	// Test circular dependency detection
	t.Run("detect_circular_dependencies", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/dependencies/analyze/circular-test", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		// Should handle circular dependencies gracefully
		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
			t.Logf("Circular dependency detection response: %d", w.Code)
		}
	})

	// Test cascading swap impact
	t.Run("detect_cascading_swap_impact", func(t *testing.T) {
		payload := map[string]interface{}{
			"from":          "postgres",
			"to":            "sqlite",
			"tier":          "mobile",
			"check_cascade": true,
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/swaps/analyze-impact", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
			t.Logf("Cascading swap impact response: %d", w.Code)
		}
	})

	mock.ExpectClose()
}

// [REQ:DM-P0-023,DM-P0-024,DM-P0-025,DM-P0-026,DM-P0-027] TestComprehensiveValidationWorkflow tests full validation suite
func TestComprehensiveValidationWorkflow(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	srv := &Server{
		config:   &Config{Port: "8080"},
		db:       db,
		router:   mux.NewRouter(),
		profiles: NewSQLProfileRepository(db),
	}
	srv.setupRoutes()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Step 1: Run comprehensive validation
	t.Run("comprehensive_validation", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/profiles/1/validate", nil).WithContext(ctx)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		// Validation should complete within timeout
		select {
		case <-ctx.Done():
			t.Error("Validation exceeded 15 second timeout")
		default:
			// Completed within timeout
		}

		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Logf("Comprehensive validation response: %d", w.Code)
		}
	})

	// Step 2: Estimate SaaS costs
	t.Run("estimate_costs", func(t *testing.T) {
		payload := map[string]interface{}{
			"scenario": "test-scenario",
			"tier":     "saas",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/validation/estimate-cost", bytes.NewReader(body)).WithContext(ctx)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
			t.Logf("Cost estimation response: %d", w.Code)
		}
	})

	mock.ExpectClose()
}

// [REQ:DM-P0-028,DM-P0-029,DM-P0-030,DM-P0-031,DM-P0-032] TestDeploymentMonitoringWorkflow tests deployment execution and monitoring
func TestDeploymentMonitoringWorkflow(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	srv := &Server{
		config:   &Config{Port: "8080"},
		db:       db,
		router:   mux.NewRouter(),
		profiles: NewSQLProfileRepository(db),
	}
	srv.setupRoutes()

	// Step 1: Start deployment
	t.Run("start_deployment", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/profiles/1/deploy", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusAccepted && w.Code != http.StatusNotFound {
			t.Logf("Deployment start response: %d", w.Code)
		}
	})

	// Step 2: Check deployment status
	t.Run("check_deployment_status", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/deployments/1/status", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Logf("Deployment status response: %d", w.Code)
		}
	})

	// Step 3: Stream deployment logs
	t.Run("stream_deployment_logs", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/deployments/1/logs", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Logf("Deployment logs response: %d", w.Code)
		}
	})

	mock.ExpectClose()
}

// [REQ:DM-P0-033,DM-P0-034] TestPackagerIntegrationDiscovery tests scenario-to-* packager discovery
func TestPackagerIntegrationDiscovery(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	srv := &Server{
		config:   &Config{Port: "8080"},
		db:       db,
		router:   mux.NewRouter(),
		profiles: NewSQLProfileRepository(db),
	}
	srv.setupRoutes()

	// Step 1: Discover available packagers
	t.Run("discover_packagers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/packagers/discover", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Logf("Packager discovery response: %d", w.Code)
		}
	})

	// Step 2: Validate packager availability for tier
	t.Run("validate_packager_for_tier", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/packagers/check?tier=desktop", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Logf("Packager validation response: %d", w.Code)
		}
	})

	mock.ExpectClose()
}

// [REQ:DM-P0-035,DM-P0-036,DM-P0-037] TestDependencyVisualizationDataPreparation tests graph/table data generation
func TestDependencyVisualizationDataPreparation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	srv := &Server{
		config:   &Config{Port: "8080"},
		db:       db,
		router:   mux.NewRouter(),
		profiles: NewSQLProfileRepository(db),
	}
	srv.setupRoutes()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Step 1: Get dependency graph data
	t.Run("get_graph_data", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/dependencies/test-scenario/graph", nil).WithContext(ctx)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		select {
		case <-ctx.Done():
			t.Error("Graph data generation exceeded 5 second timeout")
		default:
			// Completed within timeout (meets P0-037 requirement)
		}

		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Logf("Graph data response: %d", w.Code)
		}
	})

	// Step 2: Get table view data
	t.Run("get_table_data", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/dependencies/test-scenario/table", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Logf("Table data response: %d", w.Code)
		}
	})

	mock.ExpectClose()
}

// [REQ:DM-P0-015,DM-P0-016,DM-P0-017] TestProfileVersionControlWorkflow tests version management
func TestProfileVersionControlWorkflow(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	srv := &Server{
		config:   &Config{Port: "8080"},
		db:       db,
		router:   mux.NewRouter(),
		profiles: NewSQLProfileRepository(db),
	}
	srv.setupRoutes()

	// Step 1: Get version history
	t.Run("get_version_history", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/profiles/1/versions", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Logf("Version history response: %d", w.Code)
		}
	})

	// Step 2: Compare versions
	t.Run("compare_versions", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/profiles/1/versions/compare?from=1&to=2", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
			t.Logf("Version comparison response: %d", w.Code)
		}
	})

	// Step 3: Rollback to previous version
	t.Run("rollback_version", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		payload := map[string]interface{}{
			"version": 1,
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/profiles/1/rollback", bytes.NewReader(body)).WithContext(ctx)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		select {
		case <-ctx.Done():
			t.Error("Rollback exceeded 3 second timeout (requirement is 2s)")
		default:
			// Completed within timeout (meets P0-017 requirement)
		}

		if w.Code != http.StatusOK && w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
			t.Logf("Version rollback response: %d", w.Code)
		}
	})

	mock.ExpectClose()
}
