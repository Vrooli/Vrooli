package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	domainmetrics "landing-page-business-suite-api/internal/metrics"
)

// mockWaitlistService provides a configurable mock for WaitlistServicer
type mockWaitlistService struct {
	emails map[int64]*domainmetrics.WaitlistEmail
	nextID int64

	// Error injection
	createErr error
	listErr   error
	deleteErr error
	countErr  error
}

func newMockWaitlistService() *mockWaitlistService {
	return &mockWaitlistService{
		emails: make(map[int64]*domainmetrics.WaitlistEmail),
		nextID: 1,
	}
}

func (m *mockWaitlistService) Create(ctx context.Context, email, source string) (*domainmetrics.WaitlistEmail, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	// Check for existing email (upsert behavior)
	for _, e := range m.emails {
		if e.Email == email {
			e.Source = source
			return e, nil
		}
	}
	entry := &domainmetrics.WaitlistEmail{
		ID:        m.nextID,
		Email:     email,
		Source:    source,
		CreatedAt: time.Now(),
	}
	m.emails[entry.ID] = entry
	m.nextID++
	return entry, nil
}

func (m *mockWaitlistService) List(ctx context.Context) ([]domainmetrics.WaitlistEmail, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []domainmetrics.WaitlistEmail
	for _, e := range m.emails {
		result = append(result, *e)
	}
	return result, nil
}

func (m *mockWaitlistService) Delete(ctx context.Context, id int64) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if _, ok := m.emails[id]; !ok {
		return errors.New("email not found")
	}
	delete(m.emails, id)
	return nil
}

func (m *mockWaitlistService) Count(ctx context.Context) (int64, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	return int64(len(m.emails)), nil
}

// Compile-time check that mockWaitlistService implements WaitlistServicer
var _ domainmetrics.WaitlistServicer = (*mockWaitlistService)(nil)

// --- handleWaitlistCreate Tests ---

func TestHandleWaitlistCreate_Success(t *testing.T) {
	db := setupTestDB(t)

	svc := NewWaitlistService(db)
	handler := metricsHTTPDependencies.CreateWaitlist(svc)

	body := `{"email": "test@example.com", "source": "landing_page"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/waitlist", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestHandleWaitlistCreate_InvalidEmail(t *testing.T) {
	db := setupTestDB(t)

	svc := NewWaitlistService(db)
	handler := metricsHTTPDependencies.CreateWaitlist(svc)

	body := `{"email": "not-an-email", "source": "landing_page"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/waitlist", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid email, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestHandleWaitlistCreate_MissingEmail(t *testing.T) {
	db := setupTestDB(t)

	svc := NewWaitlistService(db)
	handler := metricsHTTPDependencies.CreateWaitlist(svc)

	body := `{"source": "landing_page"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/waitlist", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for missing email, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestHandleWaitlistCreate_DuplicateEmail(t *testing.T) {
	db := setupTestDB(t)

	svc := NewWaitlistService(db)
	handler := metricsHTTPDependencies.CreateWaitlist(svc)

	// First request
	body := `{"email": "duplicate@example.com", "source": "source1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/waitlist", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("First request expected %d, got %d", http.StatusOK, w.Code)
	}

	// Second request with same email (should upsert)
	body = `{"email": "duplicate@example.com", "source": "source2"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/waitlist", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Duplicate email should upsert, got status %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleWaitlistCreate_DefaultSource(t *testing.T) {
	db := setupTestDB(t)

	svc := NewWaitlistService(db)
	handler := metricsHTTPDependencies.CreateWaitlist(svc)

	// No source provided - should default to "coming_soon"
	body := `{"email": "nosource@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/waitlist", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

// --- handleWaitlistList Tests ---

func TestHandleWaitlistList_Success(t *testing.T) {
	db := setupTestDB(t)

	svc := NewWaitlistService(db)

	// Add some test entries
	ctx := context.Background()
	_, _ = svc.Create(ctx, "list1@example.com", "test")
	_, _ = svc.Create(ctx, "list2@example.com", "test")

	handler := metricsHTTPDependencies.ListWaitlist(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/waitlist", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// writeJSONSuccessData encodes data directly (array)
	var emails []domainmetrics.WaitlistEmail
	if err := json.NewDecoder(w.Body).Decode(&emails); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(emails) < 2 {
		t.Errorf("Expected at least 2 entries, got %d", len(emails))
	}
}

func TestHandleWaitlistList_ReturnsArray(t *testing.T) {
	db := setupTestDB(t)

	svc := NewWaitlistService(db)
	handler := metricsHTTPDependencies.ListWaitlist(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/waitlist", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify response is a valid JSON array (not null)
	var emails []domainmetrics.WaitlistEmail
	if err := json.NewDecoder(w.Body).Decode(&emails); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Response should be an array (may have items from previous tests but shouldn't be nil)
	if emails == nil {
		t.Error("Expected array, got nil")
	}
}

// --- handleWaitlistDelete Tests ---

func TestHandleWaitlistDelete_Success(t *testing.T) {
	db := setupTestDB(t)

	svc := NewWaitlistService(db)

	// Create test entry
	ctx := context.Background()
	entry, _ := svc.Create(ctx, "delete@example.com", "test")

	handler := metricsHTTPDependencies.DeleteWaitlist(svc)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/waitlist/{id}", handler).Methods("DELETE")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/waitlist/"+strconv.FormatInt(entry.ID, 10), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestHandleWaitlistDelete_NotFound(t *testing.T) {
	db := setupTestDB(t)

	svc := NewWaitlistService(db)
	handler := metricsHTTPDependencies.DeleteWaitlist(svc)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/waitlist/{id}", handler).Methods("DELETE")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/waitlist/99999", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d for not found, got %d", http.StatusInternalServerError, w.Code)
	}
}

// --- handleWaitlistExport Tests ---

func TestHandleWaitlistExport_Success(t *testing.T) {
	db := setupTestDB(t)

	svc := NewWaitlistService(db)

	// Add some test entries
	ctx := context.Background()
	_, _ = svc.Create(ctx, "export1@example.com", "test")
	_, _ = svc.Create(ctx, "export2@example.com", "test")

	handler := metricsHTTPDependencies.ExportWaitlist(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/waitlist/export", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check Content-Type
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/csv" {
		t.Errorf("Expected Content-Type 'text/csv', got '%s'", contentType)
	}

	// Check Content-Disposition
	disposition := w.Header().Get("Content-Disposition")
	if !strings.Contains(disposition, "attachment") {
		t.Errorf("Expected attachment disposition, got '%s'", disposition)
	}

	// Check CSV content
	body := w.Body.String()
	if !strings.Contains(body, "ID") || !strings.Contains(body, "Email") {
		t.Errorf("Expected CSV headers, got: %s", body)
	}
	if !strings.Contains(body, "export1@example.com") {
		t.Errorf("Expected email in CSV, got: %s", body)
	}
}

func TestHandleWaitlistExport_Empty(t *testing.T) {
	db := setupTestDB(t)

	svc := NewWaitlistService(db)
	handler := metricsHTTPDependencies.ExportWaitlist(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/waitlist/export", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d for empty export, got %d", http.StatusOK, w.Code)
	}

	// Should still have headers
	body := w.Body.String()
	if !strings.Contains(body, "ID") || !strings.Contains(body, "Email") {
		t.Errorf("Expected CSV headers even for empty export, got: %s", body)
	}
}

// --- Mock Service Tests ---

func TestMockWaitlistService_Create(t *testing.T) {
	mock := newMockWaitlistService()

	entry, err := mock.Create(context.Background(), "mock@test.com", "test")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if entry.Email != "mock@test.com" {
		t.Errorf("Expected email 'mock@test.com', got %s", entry.Email)
	}
	if entry.ID != 1 {
		t.Errorf("Expected ID 1, got %d", entry.ID)
	}
}

func TestMockWaitlistService_Upsert(t *testing.T) {
	mock := newMockWaitlistService()
	ctx := context.Background()

	// First create
	_, _ = mock.Create(ctx, "upsert@test.com", "source1")

	// Second create with same email (upsert)
	entry, err := mock.Create(ctx, "upsert@test.com", "source2")
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	if entry.Source != "source2" {
		t.Errorf("Expected source 'source2' after upsert, got %s", entry.Source)
	}

	// Should still only have 1 entry
	list, _ := mock.List(ctx)
	if len(list) != 1 {
		t.Errorf("Expected 1 entry after upsert, got %d", len(list))
	}
}

func TestMockWaitlistService_ErrorInjection(t *testing.T) {
	mock := newMockWaitlistService()
	mock.createErr = errors.New("forced error")

	_, err := mock.Create(context.Background(), "test@test.com", "test")
	if err == nil {
		t.Error("Expected error from mock")
	}
}

// --- Error Injection Tests Using Mock ---

func TestHandleWaitlistCreate_ServiceError(t *testing.T) {
	mock := newMockWaitlistService()
	mock.createErr = errors.New("database connection failed")

	handler := metricsHTTPDependencies.CreateWaitlist(mock)

	body := `{"email": "test@example.com", "source": "landing_page"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/waitlist", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d for service error, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestHandleWaitlistCreate_InvalidJSON(t *testing.T) {
	mock := newMockWaitlistService()
	handler := metricsHTTPDependencies.CreateWaitlist(mock)

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/waitlist", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleWaitlistList_ServiceError(t *testing.T) {
	mock := newMockWaitlistService()
	mock.listErr = errors.New("query failed")

	handler := metricsHTTPDependencies.ListWaitlist(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/waitlist", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d for service error, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestHandleWaitlistList_NilToEmptyArray(t *testing.T) {
	mock := newMockWaitlistService()
	// Empty mock returns empty array (not nil)

	handler := metricsHTTPDependencies.ListWaitlist(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/waitlist", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify we get an array, not null
	body := w.Body.String()
	if body == "null" || body == "null\n" {
		t.Error("Expected empty array, got null")
	}

	var emails []domainmetrics.WaitlistEmail
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&emails); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if emails == nil {
		t.Error("Expected non-nil empty array")
	}
}

func TestHandleWaitlistDelete_InvalidID(t *testing.T) {
	mock := newMockWaitlistService()
	handler := metricsHTTPDependencies.DeleteWaitlist(mock)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/waitlist/{id}", handler).Methods("DELETE")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/waitlist/not-a-number", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid ID, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleWaitlistDelete_ServiceError(t *testing.T) {
	mock := newMockWaitlistService()
	mock.deleteErr = errors.New("cannot delete")

	// Add an email so it exists
	if _, err := mock.Create(context.Background(), "test@example.com", "test"); err != nil {
		t.Fatalf("Failed to create waitlist entry: %v", err)
	}

	handler := metricsHTTPDependencies.DeleteWaitlist(mock)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/waitlist/{id}", handler).Methods("DELETE")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/waitlist/1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d for service error, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestHandleWaitlistExport_ServiceError(t *testing.T) {
	mock := newMockWaitlistService()
	mock.listErr = errors.New("export query failed")

	handler := metricsHTTPDependencies.ExportWaitlist(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/waitlist/export", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d for service error, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestHandleWaitlistExport_VerifyCSVFormat(t *testing.T) {
	mock := newMockWaitlistService()
	// Add test data
	if _, err := mock.Create(context.Background(), "user1@example.com", "landing"); err != nil {
		t.Fatalf("Failed to create waitlist entry: %v", err)
	}
	if _, err := mock.Create(context.Background(), "user2@example.com", "beta"); err != nil {
		t.Fatalf("Failed to create waitlist entry: %v", err)
	}

	handler := metricsHTTPDependencies.ExportWaitlist(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/waitlist/export", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify CSV structure
	body := w.Body.String()
	lines := strings.Split(strings.TrimSpace(body), "\n")

	if len(lines) < 1 {
		t.Fatal("Expected at least header row")
	}

	// Check header
	header := lines[0]
	expectedHeaders := []string{"ID", "Email", "Source", "Created At"}
	for _, h := range expectedHeaders {
		if !strings.Contains(header, h) {
			t.Errorf("Expected header to contain '%s', got: %s", h, header)
		}
	}

	// Check we have data rows (header + 2 data rows)
	if len(lines) < 3 {
		t.Errorf("Expected 3 lines (header + 2 data), got %d", len(lines))
	}
}
