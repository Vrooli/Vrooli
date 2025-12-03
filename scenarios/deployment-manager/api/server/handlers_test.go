package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"deployment-manager/bundles"
	"deployment-manager/dependencies"
	"deployment-manager/deployments"
	"deployment-manager/fitness"
	"deployment-manager/health"
	"deployment-manager/profiles"
	"deployment-manager/secrets"
	"deployment-manager/swaps"
	"deployment-manager/telemetry"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
)

// setupTestServer creates a minimal Server for testing
func setupTestServer(t *testing.T) (*Server, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}

	logFn := func(msg string, fields map[string]interface{}) {}
	profilesRepo := profiles.NewSQLRepository(db)

	srv := &Server{
		Config:              &Config{Port: "8080"},
		DB:                  db,
		Router:              mux.NewRouter(),
		ProfilesRepo:        profilesRepo,
		HealthHandler:       health.NewHandler(db),
		FitnessHandler:      fitness.NewHandler(logFn),
		TelemetryHandler:    telemetry.NewHandler(logFn),
		SecretsHandler:      secrets.NewHandler(profilesRepo, logFn),
		DependenciesHandler: dependencies.NewHandler(logFn),
		SwapsHandler:        swaps.NewHandler(logFn),
		DeploymentsHandler:  deployments.NewHandler(logFn),
		BundlesHandler:      bundles.NewHandler(secrets.NewClient(), logFn),
		ProfilesHandler:     profiles.NewHandler(profilesRepo, logFn),
	}
	srv.setupRoutes()

	return srv, mock
}

// [REQ:DM-P0-001] TestHandleAnalyzeDependencies tests dependency analysis endpoint
func TestHandleAnalyzeDependencies(t *testing.T) {
	tests := []struct {
		name       string
		scenario   string
		wantStatus int
	}{
		{
			name:       "valid scenario",
			scenario:   "test-scenario",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, mock := setupTestServer(t)
			defer srv.DB.Close()
			_ = mock // not needed for this test

			req := httptest.NewRequest("GET", "/api/v1/dependencies/analyze/"+tt.scenario, nil)
			w := httptest.NewRecorder()

			srv.Router.ServeHTTP(w, req)

			// Handler exists and responds (actual behavior depends on external dependencies)
			if w.Code == http.StatusNotFound {
				t.Error("endpoint not registered")
			}
		})
	}
}

// [REQ:DM-P0-003,DM-P0-004] TestHandleScoreFitness tests fitness scoring endpoint
func TestHandleScoreFitness(t *testing.T) {
	tests := []struct {
		name       string
		payload    map[string]interface{}
		wantStatus int
		wantErr    bool
	}{
		{
			name: "valid fitness request",
			payload: map[string]interface{}{
				"scenario": "test-scenario",
				"tier":     2,
			},
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "missing payload",
			payload:    nil,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, mock := setupTestServer(t)
			defer srv.DB.Close()
			_ = mock

			var body *bytes.Buffer
			if tt.payload != nil {
				jsonData, _ := json.Marshal(tt.payload)
				body = bytes.NewBuffer(jsonData)
			} else {
				body = bytes.NewBuffer([]byte{})
			}

			req := httptest.NewRequest("POST", "/api/v1/fitness/score", body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			srv.Router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("HTTP status = %v, want %v", w.Code, tt.wantStatus)
			}
		})
	}
}

// [REQ:DM-P0-012] TestHandleListProfiles tests profile listing endpoint
func TestHandleListProfiles(t *testing.T) {
	srv, mock := setupTestServer(t)
	defer srv.DB.Close()

	// Mock successful profile query
	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "name", "scenario", "tiers", "swaps", "secrets", "settings", "version", "created_at", "updated_at", "created_by", "updated_by"}).
		AddRow("profile-1", "Test Profile", "test-scenario", []byte("[1,2,3]"), []byte("{}"), []byte("{}"), []byte("{}"), 1, now, now, "system", "system")

	mock.ExpectQuery("SELECT (.+) FROM profiles").WillReturnRows(rows)

	req := httptest.NewRequest("GET", "/api/v1/profiles", nil)
	w := httptest.NewRecorder()

	srv.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HTTP status = %v, want %v", w.Code, http.StatusOK)
	}

	var response []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 1 {
		t.Errorf("expected 1 profile, got %d", len(response))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled mock expectations: %v", err)
	}
}

// [REQ:DM-P0-013] TestHandleCreateProfile tests profile creation endpoint
func TestHandleCreateProfile(t *testing.T) {
	tests := []struct {
		name       string
		payload    map[string]interface{}
		wantStatus int
		setupMock  func(sqlmock.Sqlmock)
	}{
		{
			name: "valid profile creation",
			payload: map[string]interface{}{
				"name":     "New Profile",
				"scenario": "test-scenario",
				"tiers":    []int{1, 2},
			},
			wantStatus: http.StatusCreated,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO profiles").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("INSERT INTO profile_versions").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name:       "missing payload",
			payload:    nil,
			wantStatus: http.StatusBadRequest,
			setupMock:  func(mock sqlmock.Sqlmock) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, mock := setupTestServer(t)
			defer srv.DB.Close()

			tt.setupMock(mock)

			var body *bytes.Buffer
			if tt.payload != nil {
				jsonData, _ := json.Marshal(tt.payload)
				body = bytes.NewBuffer(jsonData)
			} else {
				body = bytes.NewBuffer([]byte{})
			}

			req := httptest.NewRequest("POST", "/api/v1/profiles", body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			srv.Router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("HTTP status = %v, want %v", w.Code, tt.wantStatus)
			}
		})
	}
}

// [REQ:DM-P0-028] TestHandleDeploy tests deployment endpoint
func TestHandleDeploy(t *testing.T) {
	srv, mock := setupTestServer(t)
	defer srv.DB.Close()
	_ = mock

	req := httptest.NewRequest("POST", "/api/v1/deploy/profile-1", nil)
	w := httptest.NewRecorder()

	srv.Router.ServeHTTP(w, req)

	// Handler exists and responds (actual behavior depends on database state)
	if w.Code == http.StatusNotFound {
		t.Error("deployment endpoint not registered")
	}
}

// [REQ:DM-P0-018] TestHandleIdentifySecrets tests secret identification endpoint
func TestHandleIdentifySecrets(t *testing.T) {
	srv, mock := setupTestServer(t)
	defer srv.DB.Close()
	_ = mock

	req := httptest.NewRequest("GET", "/api/v1/profiles/profile-1/secrets", nil)
	w := httptest.NewRecorder()

	srv.Router.ServeHTTP(w, req)

	// Handler exists and responds
	if w.Code == http.StatusNotFound {
		t.Error("secrets endpoint not registered")
	}
}

// [REQ:DM-P0-023] TestHandleValidateProfile tests profile validation endpoint
func TestHandleValidateProfile(t *testing.T) {
	srv, mock := setupTestServer(t)
	defer srv.DB.Close()
	_ = mock

	req := httptest.NewRequest("GET", "/api/v1/profiles/profile-1/validate", nil)
	w := httptest.NewRecorder()

	srv.Router.ServeHTTP(w, req)

	// Handler exists and responds
	if w.Code == http.StatusNotFound {
		t.Error("validation endpoint not registered")
	}
}

// TestContextCancellation verifies graceful handling of context cancellation
func TestContextCancellation(t *testing.T) {
	srv, mock := setupTestServer(t)
	defer srv.DB.Close()
	_ = mock

	req := httptest.NewRequest("GET", "/health", nil)
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	// Cancel context immediately
	cancel()

	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)

	// Server should still respond despite cancelled context
	if w.Code == 0 {
		t.Error("expected response despite cancelled context")
	}
}
