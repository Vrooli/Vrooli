package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"lifestyle-dashboard/domain"
	"lifestyle-dashboard/repository"
)

// setupTestDB creates an in-memory SQLite database with schema for testing.
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := domain.InitSchema(db); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}
	return db
}

// setupTestHandler creates a handler with SQLite repositories for testing.
func setupTestHandler(t *testing.T) (*Handler, *sql.DB) {
	db := setupTestDB(t)
	eventRepo := repository.NewSQLiteEventRepository(db)
	domainRepo := repository.NewSQLiteDomainRepository(db)
	statsRepo := repository.NewSQLiteStatsRepository(db)
	storageRepo := repository.NewSQLiteStorageRepository(db)
	briefsRepo := repository.NewSQLiteBriefRepository(db)
	scoreConfigRepo := repository.NewSQLiteScoreConfigRepository(db)
	h := New(eventRepo, domainRepo, statsRepo, storageRepo, briefsRepo, scoreConfigRepo)
	return h, db
}

// TestNew_CreatesHandler verifies that New creates a valid Handler instance.
func TestNew_CreatesHandler(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	if h == nil {
		t.Fatal("Expected handler, got nil")
	}
	if h.Events == nil {
		t.Error("Handler Events repository should not be nil")
	}
	if h.Domains == nil {
		t.Error("Handler Domains repository should not be nil")
	}
	if h.Stats == nil {
		t.Error("Handler Stats repository should not be nil")
	}
	if h.Storage == nil {
		t.Error("Handler Storage repository should not be nil")
	}
	if h.Briefs == nil {
		t.Error("Handler Briefs repository should not be nil")
	}
	if h.ScoreConfig == nil {
		t.Error("Handler ScoreConfig repository should not be nil")
	}
}

// TestWriteJSON_SetsContentType verifies JSON response headers.
func TestWriteJSON_SetsContentType(t *testing.T) {
	rr := httptest.NewRecorder()
	data := map[string]string{"message": "test"}

	WriteJSON(rr, http.StatusOK, data)

	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", ct)
	}
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

// TestWriteError_ReturnsErrorStructure verifies error response format.
func TestWriteError_ReturnsErrorStructure(t *testing.T) {
	rr := httptest.NewRecorder()

	WriteError(rr, http.StatusBadRequest, "test error message")

	var errResp domain.ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}
	if !errResp.Error {
		t.Error("Expected error field to be true")
	}
	if errResp.Message != "test error message" {
		t.Errorf("Expected message 'test error message', got '%s'", errResp.Message)
	}
}

// Helper: create a test event directly via handler
func (h *Handler) createTestEvent(t *testing.T, domainName, eventType string) {
	body := `{"domain": "` + domainName + `", "event_type": "` + eventType + `", "payload": {}}`
	req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.CreateEvent(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("Failed to create test event: %s", rr.Body.String())
	}
}

// Helper: register a test domain directly via handler
func (h *Handler) registerTestDomain(t *testing.T, name, displayName string) {
	body := `{"name": "` + name + `", "display_name": "` + displayName + `"}`
	req := httptest.NewRequest("POST", "/api/v1/domains", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.RegisterDomain(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("Failed to register test domain: %s", rr.Body.String())
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
