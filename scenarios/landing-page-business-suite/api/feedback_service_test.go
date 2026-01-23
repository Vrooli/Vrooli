package main

import (
	"database/sql"
	"testing"
)

func TestNewFeedbackService(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewFeedbackService(db)
	if service == nil {
		t.Fatal("NewFeedbackService returned nil")
	}
	if service.db != db {
		t.Error("Expected service to hold reference to provided db")
	}
}

func TestFeedbackService_Create_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupFeedbackTable(t, db)

	service := NewFeedbackService(db)

	input := &CreateFeedbackInput{
		Type:    "bug",
		Email:   "test@example.com",
		Subject: "Test Subject",
		Message: "Test Message",
	}

	feedback, err := service.Create(input)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if feedback == nil {
		t.Fatal("Create returned nil")
	}
	if feedback.ID == 0 {
		t.Error("Expected non-zero ID")
	}
	if feedback.Type != "bug" {
		t.Errorf("Expected type 'bug', got '%s'", feedback.Type)
	}
	if feedback.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", feedback.Email)
	}
	if feedback.Subject != "Test Subject" {
		t.Errorf("Expected subject 'Test Subject', got '%s'", feedback.Subject)
	}
	if feedback.Message != "Test Message" {
		t.Errorf("Expected message 'Test Message', got '%s'", feedback.Message)
	}
}

func TestFeedbackService_Create_StatusPending(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupFeedbackTable(t, db)

	service := NewFeedbackService(db)

	input := &CreateFeedbackInput{
		Type:    "feature",
		Email:   "pending@example.com",
		Subject: "Feature Request",
		Message: "Please add this feature",
	}

	feedback, err := service.Create(input)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if feedback.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", feedback.Status)
	}
}

func TestFeedbackService_Create_WithOrderID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupFeedbackTable(t, db)

	service := NewFeedbackService(db)
	orderID := "order_123"

	input := &CreateFeedbackInput{
		Type:    "refund",
		Email:   "order@example.com",
		Subject: "Refund Request",
		Message: "Please refund my order",
		OrderID: &orderID,
	}

	feedback, err := service.Create(input)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if feedback.OrderID == nil {
		t.Fatal("Expected OrderID to be set")
	}
	if *feedback.OrderID != "order_123" {
		t.Errorf("Expected OrderID 'order_123', got '%s'", *feedback.OrderID)
	}
}

func TestFeedbackService_List_All(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupFeedbackTable(t, db)

	service := NewFeedbackService(db)

	// Create multiple feedback entries
	for i := 0; i < 3; i++ {
		_, err := service.Create(&CreateFeedbackInput{
			Type:    "general",
			Email:   "list@example.com",
			Subject: "Test",
			Message: "Message",
		})
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// List all (no filter)
	requests, err := service.List("")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(requests) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(requests))
	}
}

func TestFeedbackService_List_ByStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupFeedbackTable(t, db)

	service := NewFeedbackService(db)

	// Create entries
	feedback1, err := service.Create(&CreateFeedbackInput{
		Type: "bug", Email: "1@example.com", Subject: "S1", Message: "M1",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = service.Create(&CreateFeedbackInput{
		Type: "bug", Email: "2@example.com", Subject: "S2", Message: "M2",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update one to resolved
	_, err = service.UpdateStatus(feedback1.ID, "resolved")
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	// List only pending
	pendingRequests, err := service.List("pending")
	if err != nil {
		t.Fatalf("List pending failed: %v", err)
	}
	if len(pendingRequests) != 1 {
		t.Errorf("Expected 1 pending entry, got %d", len(pendingRequests))
	}

	// List only resolved
	resolvedRequests, err := service.List("resolved")
	if err != nil {
		t.Fatalf("List resolved failed: %v", err)
	}
	if len(resolvedRequests) != 1 {
		t.Errorf("Expected 1 resolved entry, got %d", len(resolvedRequests))
	}
}

func TestFeedbackService_List_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupFeedbackTable(t, db)

	service := NewFeedbackService(db)

	requests, err := service.List("")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	// Note: nil slice and empty slice are semantically equivalent in Go
	if len(requests) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(requests))
	}
}

func TestFeedbackService_GetByID_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupFeedbackTable(t, db)

	service := NewFeedbackService(db)

	created, err := service.Create(&CreateFeedbackInput{
		Type:    "general",
		Email:   "getbyid@example.com",
		Subject: "Help needed",
		Message: "I need help",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	fetched, err := service.GetByID(created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if fetched == nil {
		t.Fatal("GetByID returned nil")
	}
	if fetched.ID != created.ID {
		t.Errorf("Expected ID %d, got %d", created.ID, fetched.ID)
	}
	if fetched.Email != "getbyid@example.com" {
		t.Errorf("Expected email 'getbyid@example.com', got '%s'", fetched.Email)
	}
}

func TestFeedbackService_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupFeedbackTable(t, db)

	service := NewFeedbackService(db)

	_, err := service.GetByID(99999)
	if err == nil {
		t.Fatal("Expected error for non-existent ID")
	}
	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows, got %v", err)
	}
}

func TestFeedbackService_UpdateStatus_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupFeedbackTable(t, db)

	service := NewFeedbackService(db)

	created, err := service.Create(&CreateFeedbackInput{
		Type: "bug", Email: "update@example.com", Subject: "S", Message: "M",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := service.UpdateStatus(created.ID, "resolved")
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}
	if updated.Status != "resolved" {
		t.Errorf("Expected status 'resolved', got '%s'", updated.Status)
	}
	if !updated.UpdatedAt.After(created.CreatedAt) || updated.UpdatedAt.Equal(created.CreatedAt) {
		// UpdatedAt should be >= CreatedAt
	}
}

func TestFeedbackService_UpdateStatus_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupFeedbackTable(t, db)

	service := NewFeedbackService(db)

	_, err := service.UpdateStatus(99999, "resolved")
	if err == nil {
		t.Fatal("Expected error for non-existent ID")
	}
	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows, got %v", err)
	}
}

func TestFeedbackService_Delete_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupFeedbackTable(t, db)

	service := NewFeedbackService(db)

	created, err := service.Create(&CreateFeedbackInput{
		Type: "bug", Email: "delete@example.com", Subject: "S", Message: "M",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = service.Delete(created.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	_, err = service.GetByID(created.ID)
	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestFeedbackService_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupFeedbackTable(t, db)

	service := NewFeedbackService(db)

	err := service.Delete(99999)
	if err == nil {
		t.Fatal("Expected error for non-existent ID")
	}
	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows, got %v", err)
	}
}

func TestFeedbackService_DeleteBulk_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupFeedbackTable(t, db)

	service := NewFeedbackService(db)

	// Create multiple entries
	var ids []int
	for i := 0; i < 5; i++ {
		created, err := service.Create(&CreateFeedbackInput{
			Type: "bug", Email: "bulk@example.com", Subject: "S", Message: "M",
		})
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		ids = append(ids, created.ID)
	}

	// Delete first 3
	affected, err := service.DeleteBulk(ids[:3])
	if err != nil {
		t.Fatalf("DeleteBulk failed: %v", err)
	}
	if affected != 3 {
		t.Errorf("Expected 3 affected rows, got %d", affected)
	}

	// Verify remaining count
	remaining, err := service.List("")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(remaining) != 2 {
		t.Errorf("Expected 2 remaining entries, got %d", len(remaining))
	}
}

func TestFeedbackService_DeleteBulk_EmptySlice(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupFeedbackTable(t, db)

	service := NewFeedbackService(db)

	affected, err := service.DeleteBulk([]int{})
	if err != nil {
		t.Fatalf("DeleteBulk with empty slice failed: %v", err)
	}
	if affected != 0 {
		t.Errorf("Expected 0 affected rows, got %d", affected)
	}
}

func TestFeedbackService_DeleteBulk_PartialMatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupFeedbackTable(t, db)

	service := NewFeedbackService(db)

	// Create one entry
	created, err := service.Create(&CreateFeedbackInput{
		Type: "bug", Email: "partial@example.com", Subject: "S", Message: "M",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Try to delete the real ID and a non-existent one
	affected, err := service.DeleteBulk([]int{created.ID, 99999})
	if err != nil {
		t.Fatalf("DeleteBulk failed: %v", err)
	}
	if affected != 1 {
		t.Errorf("Expected 1 affected row, got %d", affected)
	}
}

func TestFeedbackService_List_OrderByCreatedDesc(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupFeedbackTable(t, db)

	service := NewFeedbackService(db)

	// Create entries with different subjects to identify them
	subjects := []string{"First", "Second", "Third"}
	for _, subject := range subjects {
		_, err := service.Create(&CreateFeedbackInput{
			Type: "general", Email: "order@example.com", Subject: subject, Message: "M",
		})
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	requests, err := service.List("")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Should be in reverse order (most recent first)
	if len(requests) != 3 {
		t.Fatalf("Expected 3 entries, got %d", len(requests))
	}
	if requests[0].Subject != "Third" {
		t.Errorf("Expected first entry subject 'Third', got '%s'", requests[0].Subject)
	}
	if requests[2].Subject != "First" {
		t.Errorf("Expected last entry subject 'First', got '%s'", requests[2].Subject)
	}
}

// cleanupFeedbackTable removes all entries from feedback_requests table
func cleanupFeedbackTable(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec("DELETE FROM feedback_requests"); err != nil {
		t.Fatalf("Failed to cleanup feedback_requests table: %v", err)
	}
}
