package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"lifestyle-dashboard/domain"
	"lifestyle-dashboard/errors"
)

// TestGetStorageInfo_Success verifies storage info retrieval.
// [REQ:LD-UI-STORAGE] Handler returns storage overview data.
func TestGetStorageInfo_Success(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/storage", nil)
	rr := httptest.NewRecorder()

	h.GetStorageInfo(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.StorageInfo
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Empty database should have 0 events and 0 domains
	if resp.TotalEvents != 0 {
		t.Errorf("Expected 0 events, got %d", resp.TotalEvents)
	}
	if resp.TotalDomains != 0 {
		t.Errorf("Expected 0 domains, got %d", resp.TotalDomains)
	}
}

// TestGetStorageInfo_WithData verifies storage info with data.
// [REQ:LD-UI-STORAGE] Handler returns accurate storage data.
func TestGetStorageInfo_WithData(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Create test data
	h.registerTestDomain(t, "storage-domain", "Storage Domain")
	h.createTestEvent(t, "storage-domain", "test.event")
	h.createTestEvent(t, "storage-domain", "test.event")
	h.createTestEvent(t, "storage-domain", "test.event")

	req := httptest.NewRequest("GET", "/api/v1/storage", nil)
	rr := httptest.NewRecorder()

	h.GetStorageInfo(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.StorageInfo
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.TotalEvents != 3 {
		t.Errorf("Expected 3 events, got %d", resp.TotalEvents)
	}
	if resp.TotalDomains != 1 {
		t.Errorf("Expected 1 domain, got %d", resp.TotalDomains)
	}
}

// TestGetStorageInfo_EventsByDomain verifies domain breakdown.
// [REQ:LD-UI-STORAGE] Handler provides per-domain event counts.
func TestGetStorageInfo_EventsByDomain(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Create multiple domains
	h.registerTestDomain(t, "domain-a", "Domain A")
	h.registerTestDomain(t, "domain-b", "Domain B")

	// Add events
	h.createTestEvent(t, "domain-a", "test")
	h.createTestEvent(t, "domain-a", "test")
	h.createTestEvent(t, "domain-b", "test")

	req := httptest.NewRequest("GET", "/api/v1/storage", nil)
	rr := httptest.NewRecorder()

	h.GetStorageInfo(rr, req)

	var resp domain.StorageInfo
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.EventsByDomain) != 2 {
		t.Errorf("Expected 2 domain entries, got %d", len(resp.EventsByDomain))
	}
}

// TestCleanupEvents_ClearAll verifies clearing all events.
// [REQ:LD-UI-STORAGE] Handler clears all events when no filters.
func TestCleanupEvents_ClearAll(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Create test events
	h.createTestEvent(t, "domain-1", "test")
	h.createTestEvent(t, "domain-2", "test")
	h.createTestEvent(t, "domain-3", "test")

	// Clear all - empty body
	req := httptest.NewRequest("DELETE", "/api/v1/storage/events", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CleanupEvents(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.CleanupResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.DeletedEvents != 3 {
		t.Errorf("Expected 3 deleted events, got %d", resp.DeletedEvents)
	}
	if resp.Message == "" {
		t.Error("Expected cleanup message")
	}
}

// TestCleanupEvents_ClearDomain verifies domain-specific cleanup.
// [REQ:LD-UI-STORAGE] Handler clears only specified domains.
func TestCleanupEvents_ClearDomain(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Create test events
	h.createTestEvent(t, "keep-domain", "test")
	h.createTestEvent(t, "clear-domain", "test")
	h.createTestEvent(t, "clear-domain", "test")

	// Clear specific domain
	body := `{"domains": ["clear-domain"]}`
	req := httptest.NewRequest("DELETE", "/api/v1/storage/events", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CleanupEvents(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.CleanupResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.DeletedEvents != 2 {
		t.Errorf("Expected 2 deleted events, got %d", resp.DeletedEvents)
	}

	// Verify keep-domain still has data
	storageReq := httptest.NewRequest("GET", "/api/v1/storage", nil)
	storageRR := httptest.NewRecorder()
	h.GetStorageInfo(storageRR, storageReq)

	var storageInfo domain.StorageInfo
	json.NewDecoder(storageRR.Body).Decode(&storageInfo)

	if storageInfo.TotalEvents != 1 {
		t.Errorf("Expected 1 remaining event, got %d", storageInfo.TotalEvents)
	}
}

// TestCleanupEvents_EmptyBody verifies empty body handling.
// [REQ:LD-UI-STORAGE] Handler accepts empty body for clear all.
func TestCleanupEvents_EmptyBody(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	h.createTestEvent(t, "test-domain", "test")

	// Empty body should work (clear all)
	req := httptest.NewRequest("DELETE", "/api/v1/storage/events", nil)
	rr := httptest.NewRecorder()

	h.CleanupEvents(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
}

// TestCleanupEvents_InvalidJSON verifies JSON validation.
// [REQ:LD-UI-STORAGE] Handler validates JSON body.
func TestCleanupEvents_InvalidJSON(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest("DELETE", "/api/v1/storage/events", bytes.NewBufferString("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CleanupEvents(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

// TestCleanupEvents_MultipleDomains verifies multiple domain cleanup.
// [REQ:LD-UI-STORAGE] Handler clears multiple specified domains.
func TestCleanupEvents_MultipleDomains(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Create test events
	h.createTestEvent(t, "clear-1", "test")
	h.createTestEvent(t, "clear-2", "test")
	h.createTestEvent(t, "keep", "test")

	// Clear multiple domains
	body := `{"domains": ["clear-1", "clear-2"]}`
	req := httptest.NewRequest("DELETE", "/api/v1/storage/events", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CleanupEvents(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.CleanupResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.DeletedEvents != 2 {
		t.Errorf("Expected 2 deleted events, got %d", resp.DeletedEvents)
	}
}

// TestGetStorageInfo_NilRepository verifies error when storage repository is nil.
// [REQ:LD-UI-STORAGE] Handler gracefully handles missing repository.
func TestGetStorageInfo_NilRepository(t *testing.T) {
	h := &Handler{Storage: nil}

	req := httptest.NewRequest("GET", "/api/v1/storage", nil)
	rr := httptest.NewRecorder()

	h.GetStorageInfo(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d: %s", http.StatusInternalServerError, rr.Code, rr.Body.String())
	}
}

// TestCleanupEvents_NilRepository verifies error when storage repository is nil.
// [REQ:LD-UI-STORAGE] Handler gracefully handles missing repository.
func TestCleanupEvents_NilRepository(t *testing.T) {
	h := &Handler{Storage: nil}

	req := httptest.NewRequest("DELETE", "/api/v1/storage/events", nil)
	rr := httptest.NewRecorder()

	h.CleanupEvents(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d: %s", http.StatusInternalServerError, rr.Code, rr.Body.String())
	}
}

// TestGetStorageInfo_DatabaseError verifies error handling when GetStorageInfo returns error.
// [REQ:LD-UI-STORAGE] Handler handles database errors for storage info.
func TestGetStorageInfo_DatabaseError(t *testing.T) {
	mockStorage := &mockStorageRepoWithError{
		getInfoError: fmt.Errorf("database connection lost"),
	}
	h := &Handler{Storage: mockStorage}

	req := httptest.NewRequest("GET", "/api/v1/storage", nil)
	rr := httptest.NewRecorder()

	h.GetStorageInfo(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d: %s", http.StatusInternalServerError, rr.Code, rr.Body.String())
	}

	var errResp errors.APIError
	if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}
	if errResp.Category != errors.CategoryInternal {
		t.Errorf("Expected category 'internal', got '%s'", errResp.Category)
	}
}

// TestCleanupEvents_DatabaseError verifies error handling when CleanupEvents returns error.
// [REQ:LD-UI-STORAGE] Handler handles database errors for cleanup.
func TestCleanupEvents_DatabaseError(t *testing.T) {
	mockStorage := &mockStorageRepoWithError{
		cleanupError: fmt.Errorf("delete operation failed"),
	}
	h := &Handler{Storage: mockStorage}

	req := httptest.NewRequest("DELETE", "/api/v1/storage/events", nil)
	rr := httptest.NewRecorder()

	h.CleanupEvents(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d: %s", http.StatusInternalServerError, rr.Code, rr.Body.String())
	}
}

// mockStorageRepoWithError provides controlled error injection for storage repository.
type mockStorageRepoWithError struct {
	getInfoError error
	cleanupError error
}

func (m *mockStorageRepoWithError) GetStorageInfo(ctx context.Context) (*domain.StorageInfo, error) {
	if m.getInfoError != nil {
		return nil, m.getInfoError
	}
	return &domain.StorageInfo{}, nil
}

func (m *mockStorageRepoWithError) CleanupEvents(ctx context.Context, req domain.CleanupRequest) (*domain.CleanupResponse, error) {
	if m.cleanupError != nil {
		return nil, m.cleanupError
	}
	return &domain.CleanupResponse{}, nil
}
