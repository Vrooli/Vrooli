package main

import (
	"bytes"
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
)

// mockFeedbackService provides a configurable mock for FeedbackServicer
type mockFeedbackService struct {
	feedbacks map[int]*FeedbackRequest
	nextID    int

	// Error injection
	createErr       error
	listErr         error
	getByIDErr      error
	updateStatusErr error
	deleteErr       error
	deleteBulkErr   error
}

func newMockFeedbackService() *mockFeedbackService {
	return &mockFeedbackService{
		feedbacks: make(map[int]*FeedbackRequest),
		nextID:    1,
	}
}

func (m *mockFeedbackService) Create(_ context.Context, input *CreateFeedbackInput) (*FeedbackRequest, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	f := &FeedbackRequest{
		ID:        m.nextID,
		Type:      input.Type,
		Email:     input.Email,
		Subject:   input.Subject,
		Message:   input.Message,
		OrderID:   input.OrderID,
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.feedbacks[f.ID] = f
	m.nextID++
	return f, nil
}

func (m *mockFeedbackService) List(_ context.Context, status string) ([]FeedbackRequest, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []FeedbackRequest
	for _, f := range m.feedbacks {
		if status == "" || f.Status == status {
			result = append(result, *f)
		}
	}
	return result, nil
}

func (m *mockFeedbackService) GetByID(_ context.Context, id int) (*FeedbackRequest, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	f, ok := m.feedbacks[id]
	if !ok {
		return nil, errors.New("feedback not found")
	}
	return f, nil
}

func (m *mockFeedbackService) UpdateStatus(_ context.Context, id int, status string) (*FeedbackRequest, error) {
	if m.updateStatusErr != nil {
		return nil, m.updateStatusErr
	}
	f, ok := m.feedbacks[id]
	if !ok {
		return nil, errors.New("feedback not found")
	}
	f.Status = status
	f.UpdatedAt = time.Now()
	return f, nil
}

func (m *mockFeedbackService) Delete(_ context.Context, id int) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if _, ok := m.feedbacks[id]; !ok {
		return errors.New("feedback not found")
	}
	delete(m.feedbacks, id)
	return nil
}

func (m *mockFeedbackService) DeleteBulk(_ context.Context, ids []int) (int64, error) {
	if m.deleteBulkErr != nil {
		return 0, m.deleteBulkErr
	}
	var deleted int64
	for _, id := range ids {
		if _, ok := m.feedbacks[id]; ok {
			delete(m.feedbacks, id)
			deleted++
		}
	}
	return deleted, nil
}

// Compile-time check that mockFeedbackService implements FeedbackServicer
var _ FeedbackServicer = (*mockFeedbackService)(nil)

// --- handleFeedbackCreateWithConfigStore Tests ---

func TestHandleFeedbackCreate_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewFeedbackService(db)
	cs := setupTestConfigStore(t)
	emailSvc := NewEmailService()

	handler := handleFeedbackCreateWithConfigStore(svc, cs, emailSvc)

	body := `{"type": "general", "email": "test@example.com", "subject": "Test Subject", "message": "Test message"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["success"] != true {
		t.Errorf("Expected success true, got %v", resp["success"])
	}
}

func TestHandleFeedbackCreate_InvalidEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewFeedbackService(db)
	cs := setupTestConfigStore(t)
	emailSvc := NewEmailService()

	handler := handleFeedbackCreateWithConfigStore(svc, cs, emailSvc)

	// Missing email
	body := `{"type": "general", "subject": "Test", "message": "Test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for missing email, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestHandleFeedbackCreate_InvalidSubject(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewFeedbackService(db)
	cs := setupTestConfigStore(t)
	emailSvc := NewEmailService()

	handler := handleFeedbackCreateWithConfigStore(svc, cs, emailSvc)

	// Missing subject
	body := `{"type": "general", "email": "test@example.com", "message": "Test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for missing subject, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestHandleFeedbackCreate_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewFeedbackService(db)
	cs := setupTestConfigStore(t)
	emailSvc := NewEmailService()

	handler := handleFeedbackCreateWithConfigStore(svc, cs, emailSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleFeedbackCreate_DefaultsToGeneralType(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewFeedbackService(db)
	cs := setupTestConfigStore(t)
	emailSvc := NewEmailService()

	handler := handleFeedbackCreateWithConfigStore(svc, cs, emailSvc)

	// Invalid type should default to "general"
	body := `{"type": "invalid_type", "email": "test@example.com", "subject": "Test", "message": "Test msg"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}
}

// --- handleFeedbackList Tests ---

func TestHandleFeedbackList_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewFeedbackService(db)

	// Create some test feedback
	_, _ = svc.Create(context.Background(), &CreateFeedbackInput{
		Type:    "general",
		Email:   "test1@example.com",
		Subject: "Test 1",
		Message: "Message 1",
	})
	_, _ = svc.Create(context.Background(), &CreateFeedbackInput{
		Type:    "bug",
		Email:   "test2@example.com",
		Subject: "Test 2",
		Message: "Message 2",
	})

	handler := handleFeedbackList(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/feedback", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var feedbacks []FeedbackRequest
	if err := json.NewDecoder(w.Body).Decode(&feedbacks); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(feedbacks) < 2 {
		t.Errorf("Expected at least 2 feedbacks, got %d", len(feedbacks))
	}
}

func TestHandleFeedbackList_FilterByStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewFeedbackService(db)

	handler := handleFeedbackList(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/feedback?status=pending", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestHandleFeedbackList_EmptyList(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewFeedbackService(db)
	handler := handleFeedbackList(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/feedback", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var feedbacks []FeedbackRequest
	if err := json.NewDecoder(w.Body).Decode(&feedbacks); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should return empty array, not null
	if feedbacks == nil {
		t.Error("Expected empty array, got nil")
	}
}

// --- handleFeedbackGet Tests ---

func TestHandleFeedbackGet_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewFeedbackService(db)

	// Create test feedback
	feedback, _ := svc.Create(context.Background(), &CreateFeedbackInput{
		Type:    "general",
		Email:   "test@example.com",
		Subject: "Test",
		Message: "Message",
	})

	handler := handleFeedbackGet(svc)

	// Use mux router to extract path variables
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/feedback/{id}", handler).Methods("GET")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/feedback/"+strconv.Itoa(feedback.ID), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestHandleFeedbackGet_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewFeedbackService(db)
	handler := handleFeedbackGet(svc)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/feedback/{id}", handler).Methods("GET")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/feedback/99999", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestHandleFeedbackGet_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewFeedbackService(db)
	handler := handleFeedbackGet(svc)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/feedback/{id}", handler).Methods("GET")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/feedback/invalid", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid ID, got %d", http.StatusBadRequest, w.Code)
	}
}

// --- handleFeedbackUpdateStatus Tests ---

func TestHandleFeedbackUpdateStatus_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewFeedbackService(db)

	// Create test feedback
	feedback, _ := svc.Create(context.Background(), &CreateFeedbackInput{
		Type:    "general",
		Email:   "test@example.com",
		Subject: "Test",
		Message: "Message",
	})

	handler := handleFeedbackUpdateStatus(svc)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/feedback/{id}/status", handler).Methods("PATCH")

	body := `{"status": "resolved"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/feedback/"+strconv.Itoa(feedback.ID)+"/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestHandleFeedbackUpdateStatus_InvalidStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewFeedbackService(db)

	// Create test feedback
	feedback, _ := svc.Create(context.Background(), &CreateFeedbackInput{
		Type:    "general",
		Email:   "test@example.com",
		Subject: "Test",
		Message: "Message",
	})

	handler := handleFeedbackUpdateStatus(svc)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/feedback/{id}/status", handler).Methods("PATCH")

	body := `{"status": "invalid_status"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/feedback/"+strconv.Itoa(feedback.ID)+"/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid status, got %d", http.StatusBadRequest, w.Code)
	}
}

// --- handleFeedbackDelete Tests ---

func TestHandleFeedbackDelete_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewFeedbackService(db)

	// Create test feedback
	feedback, _ := svc.Create(context.Background(), &CreateFeedbackInput{
		Type:    "general",
		Email:   "test@example.com",
		Subject: "Test",
		Message: "Message",
	})

	handler := handleFeedbackDelete(svc)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/feedback/{id}", handler).Methods("DELETE")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/feedback/"+strconv.Itoa(feedback.ID), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

// --- handleFeedbackDeleteBulk Tests ---

func TestHandleFeedbackDeleteBulk_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewFeedbackService(db)

	// Create test feedbacks
	f1, _ := svc.Create(context.Background(), &CreateFeedbackInput{Type: "general", Email: "a@x.com", Subject: "A", Message: "A"})
	f2, _ := svc.Create(context.Background(), &CreateFeedbackInput{Type: "general", Email: "b@x.com", Subject: "B", Message: "B"})

	handler := handleFeedbackDeleteBulk(svc)

	body, _ := json.Marshal(map[string][]int{"ids": {f1.ID, f2.ID}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/feedback/bulk-delete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	deleted, ok := resp["deleted"].(float64)
	if !ok || deleted != 2 {
		t.Errorf("Expected 2 deleted, got %v", resp["deleted"])
	}
}

func TestHandleFeedbackDeleteBulk_EmptyList(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewFeedbackService(db)
	handler := handleFeedbackDeleteBulk(svc)

	body := `{"ids": []}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/feedback/bulk-delete", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for empty IDs, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleFeedbackDeleteBulk_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewFeedbackService(db)
	handler := handleFeedbackDeleteBulk(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/feedback/bulk-delete", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, w.Code)
	}
}

// --- Error Injection Tests Using Mock ---

func TestHandleFeedbackList_ServiceError(t *testing.T) {
	mock := newMockFeedbackService()
	mock.listErr = errors.New("database unavailable")

	handler := handleFeedbackList(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/feedback", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d for service error, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestHandleFeedbackList_VerifyResponseBody(t *testing.T) {
	mock := newMockFeedbackService()
	// Add test feedback
	if _, err := mock.Create(context.Background(), &CreateFeedbackInput{
		Type:    "bug",
		Email:   "test@example.com",
		Subject: "Bug Report",
		Message: "Found a bug",
	}); err != nil {
		t.Fatalf("Failed to create feedback: %v", err)
	}

	handler := handleFeedbackList(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/feedback", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var feedbacks []FeedbackRequest
	if err := json.NewDecoder(w.Body).Decode(&feedbacks); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(feedbacks) != 1 {
		t.Errorf("Expected 1 feedback, got %d", len(feedbacks))
	}

	if feedbacks[0].Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %s", feedbacks[0].Email)
	}
}

func TestHandleFeedbackGet_ServiceError(t *testing.T) {
	mock := newMockFeedbackService()
	mock.getByIDErr = errors.New("record not found")

	handler := handleFeedbackGet(mock)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/feedback/{id}", handler).Methods("GET")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/feedback/1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d for service error, got %d", http.StatusNotFound, w.Code)
	}
}

func TestHandleFeedbackUpdateStatus_ServiceError(t *testing.T) {
	mock := newMockFeedbackService()
	// Create a feedback entry first
	if _, err := mock.Create(context.Background(), &CreateFeedbackInput{
		Type:    "general",
		Email:   "test@example.com",
		Subject: "Test",
		Message: "Test",
	}); err != nil {
		t.Fatalf("Failed to create feedback: %v", err)
	}
	// Now inject the error
	mock.updateStatusErr = errors.New("update failed")

	handler := handleFeedbackUpdateStatus(mock)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/feedback/{id}/status", handler).Methods("PATCH")

	body := `{"status": "resolved"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/feedback/1/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d for service error, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestHandleFeedbackUpdateStatus_InvalidJSON(t *testing.T) {
	mock := newMockFeedbackService()
	handler := handleFeedbackUpdateStatus(mock)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/feedback/{id}/status", handler).Methods("PATCH")

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/feedback/1/status", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleFeedbackUpdateStatus_AllValidStatuses(t *testing.T) {
	validStatuses := []string{"pending", "in_progress", "resolved", "rejected"}

	for _, status := range validStatuses {
		t.Run(status, func(t *testing.T) {
			mock := newMockFeedbackService()
			// Create a feedback entry
			if _, err := mock.Create(context.Background(), &CreateFeedbackInput{
				Type:    "general",
				Email:   "test@example.com",
				Subject: "Test",
				Message: "Test",
			}); err != nil {
				t.Fatalf("Failed to create feedback: %v", err)
			}

			handler := handleFeedbackUpdateStatus(mock)

			router := mux.NewRouter()
			router.HandleFunc("/api/v1/admin/feedback/{id}/status", handler).Methods("PATCH")

			body := `{"status": "` + status + `"}`
			req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/feedback/1/status", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Status '%s': Expected status %d, got %d: %s", status, http.StatusOK, w.Code, w.Body.String())
			}

			var resp FeedbackRequest
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if resp.Status != status {
				t.Errorf("Expected status '%s', got '%s'", status, resp.Status)
			}
		})
	}
}

func TestHandleFeedbackDelete_ServiceError(t *testing.T) {
	mock := newMockFeedbackService()
	// Create a feedback entry
	if _, err := mock.Create(context.Background(), &CreateFeedbackInput{
		Type:    "general",
		Email:   "test@example.com",
		Subject: "Test",
		Message: "Test",
	}); err != nil {
		t.Fatalf("Failed to create feedback: %v", err)
	}
	mock.deleteErr = errors.New("deletion blocked")

	handler := handleFeedbackDelete(mock)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/feedback/{id}", handler).Methods("DELETE")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/feedback/1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d for service error, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestHandleFeedbackDeleteBulk_ServiceError(t *testing.T) {
	mock := newMockFeedbackService()
	mock.deleteBulkErr = errors.New("bulk delete failed")

	handler := handleFeedbackDeleteBulk(mock)

	body := `{"ids": [1, 2, 3]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/feedback/bulk-delete", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d for service error, got %d", http.StatusInternalServerError, w.Code)
	}
}
