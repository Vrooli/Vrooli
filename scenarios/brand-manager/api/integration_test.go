package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

// TestListBrandsWithMock tests ListBrands handler with mocked database
func TestListBrandsWithMock(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	service := NewBrandManagerService(db, "", "", "", "")

	t.Run("Success", func(t *testing.T) {
		brandID := uuid.New()
		rows := sqlmock.NewRows([]string{
			"id", "name", "short_name", "slogan", "ad_copy", "description",
			"brand_colors", "logo_url", "favicon_url", "assets", "metadata",
			"created_at", "updated_at",
		}).AddRow(
			brandID, "Test Brand", "TB", "Test Slogan", "Test Ad", "Test Description",
			`{"primary": "#FF0000"}`, "http://logo.png", "http://favicon.ico",
			`[]`, `{}`, time.Now(), time.Now(),
		)

		mock.ExpectQuery("SELECT .+ FROM brands").
			WithArgs(20, 0).
			WillReturnRows(rows)

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/brands",
		}

		w, err := makeHTTPRequest(req, service.ListBrands)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("WithPagination", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "name", "short_name", "slogan", "ad_copy", "description",
			"brand_colors", "logo_url", "favicon_url", "assets", "metadata",
			"created_at", "updated_at",
		})

		mock.ExpectQuery("SELECT .+ FROM brands").
			WithArgs(10, 5).
			WillReturnRows(rows)

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/brands?limit=10&offset=5",
		}

		w, err := makeHTTPRequest(req, service.ListBrands)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("DatabaseError", func(t *testing.T) {
		mock.ExpectQuery("SELECT .+ FROM brands").
			WillReturnError(sql.ErrConnDone)

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/brands",
		}

		w, err := makeHTTPRequest(req, service.ListBrands)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}

// TestGetBrandByIDWithMock tests GetBrandByID handler with mocked database
func TestGetBrandByIDWithMock(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	service := NewBrandManagerService(db, "", "", "", "")

	t.Run("Success", func(t *testing.T) {
		brandID := uuid.New()
		rows := sqlmock.NewRows([]string{
			"id", "name", "short_name", "slogan", "ad_copy", "description",
			"brand_colors", "logo_url", "favicon_url", "assets", "metadata",
			"created_at", "updated_at",
		}).AddRow(
			brandID, "Test Brand", "TB", "Test Slogan", "Test Ad", "Test Description",
			`{"primary": "#FF0000"}`, "http://logo.png", "http://favicon.ico",
			`[]`, `{}`, time.Now(), time.Now(),
		)

		mock.ExpectQuery("SELECT .+ FROM brands WHERE id").
			WithArgs(brandID.String()).
			WillReturnRows(rows)

		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/brands/" + brandID.String(),
			URLVars: map[string]string{"id": brandID.String()},
		}

		w, err := makeHTTPRequest(req, service.GetBrandByID)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var brand Brand
		if err := json.Unmarshal(w.Body.Bytes(), &brand); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if brand.ID != brandID {
			t.Errorf("Expected brand ID %s, got %s", brandID, brand.ID)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		brandID := uuid.New()

		mock.ExpectQuery("SELECT .+ FROM brands WHERE id").
			WithArgs(brandID.String()).
			WillReturnError(sql.ErrNoRows)

		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/brands/" + brandID.String(),
			URLVars: map[string]string{"id": brandID.String()},
		}

		w, err := makeHTTPRequest(req, service.GetBrandByID)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("DatabaseError", func(t *testing.T) {
		brandID := uuid.New()

		mock.ExpectQuery("SELECT .+ FROM brands WHERE id").
			WithArgs(brandID.String()).
			WillReturnError(sql.ErrConnDone)

		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/brands/" + brandID.String(),
			URLVars: map[string]string{"id": brandID.String()},
		}

		w, err := makeHTTPRequest(req, service.GetBrandByID)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}

// TestGetBrandStatusWithMock tests GetBrandStatus handler with mocked database
func TestGetBrandStatusWithMock(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	service := NewBrandManagerService(db, "", "", "", "")

	t.Run("BrandCompleted", func(t *testing.T) {
		brandID := uuid.New()
		rows := sqlmock.NewRows([]string{
			"id", "name", "short_name", "slogan", "ad_copy", "description",
			"brand_colors", "logo_url", "favicon_url", "assets", "metadata",
			"created_at", "updated_at",
		}).AddRow(
			brandID, "Test Brand", "TB", "Test Slogan", "Test Ad", "Test Description",
			`{"primary": "#FF0000"}`, "http://logo.png", "http://favicon.ico",
			`[]`, `{}`, time.Now(), time.Now(),
		)

		mock.ExpectQuery("SELECT .+ FROM brands WHERE name").
			WithArgs("Test Brand").
			WillReturnRows(rows)

		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/brands/status/Test Brand",
			URLVars: map[string]string{"name": "Test Brand"},
		}

		w, err := makeHTTPRequest(req, service.GetBrandStatus)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if status, ok := response["status"].(string); !ok || status != "completed" {
			t.Errorf("Expected status 'completed', got %v", response["status"])
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("BrandInProgress", func(t *testing.T) {
		mock.ExpectQuery("SELECT .+ FROM brands WHERE name").
			WithArgs("InProgressBrand").
			WillReturnError(sql.ErrNoRows)

		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/brands/status/InProgressBrand",
			URLVars: map[string]string{"name": "InProgressBrand"},
		}

		w, err := makeHTTPRequest(req, service.GetBrandStatus)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if status, ok := response["status"].(string); !ok || status != "in_progress" {
			t.Errorf("Expected status 'in_progress', got %v", response["status"])
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("DatabaseError", func(t *testing.T) {
		mock.ExpectQuery("SELECT .+ FROM brands WHERE name").
			WithArgs("ErrorBrand").
			WillReturnError(sql.ErrConnDone)

		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/brands/status/ErrorBrand",
			URLVars: map[string]string{"name": "ErrorBrand"},
		}

		w, err := makeHTTPRequest(req, service.GetBrandStatus)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}

// TestListIntegrationsWithMock tests ListIntegrations handler with mocked database
func TestListIntegrationsWithMock(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	service := NewBrandManagerService(db, "", "", "", "")

	t.Run("Success", func(t *testing.T) {
		integrationID := uuid.New()
		brandID := uuid.New()
		rows := sqlmock.NewRows([]string{
			"id", "brand_id", "target_app_path", "integration_type", "claude_session_id",
			"status", "request_payload", "response_payload", "created_at", "completed_at",
		}).AddRow(
			integrationID, brandID, "/test/app", "full", "session-123",
			"pending", `{}`, `{}`, time.Now(), nil,
		)

		mock.ExpectQuery("SELECT .+ FROM integration_requests").
			WithArgs(20, 0).
			WillReturnRows(rows)

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/integrations",
		}

		w, err := makeHTTPRequest(req, service.ListIntegrations)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("DatabaseError", func(t *testing.T) {
		mock.ExpectQuery("SELECT .+ FROM integration_requests").
			WillReturnError(sql.ErrConnDone)

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/integrations",
		}

		w, err := makeHTTPRequest(req, service.ListIntegrations)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}

// TestGetServiceURLsWithMock tests GetServiceURLs handler
func TestGetServiceURLsWithMock(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	service := NewBrandManagerService(
		db,
		"http://n8n:5678",
		"http://comfyui:8188",
		"minio:9000",
		"http://vault:8200",
	)

	req := HTTPTestRequest{
		Method: "GET",
		Path:   "/api/services",
	}

	w, err := makeHTTPRequest(req, service.GetServiceURLs)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	services, ok := response["services"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'services' in response")
	}

	if services["n8n"] != "http://n8n:5678" {
		t.Errorf("Expected n8n URL, got %v", services["n8n"])
	}

	dashboards, ok := response["dashboards"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'dashboards' in response")
	}

	if dashboards["brand_manager"] == nil {
		t.Error("Expected brand_manager dashboard URL")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}
