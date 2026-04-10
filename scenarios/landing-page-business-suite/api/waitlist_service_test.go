package main

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func TestNewWaitlistService(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewWaitlistService(db)
	if service == nil {
		t.Fatal("NewWaitlistService returned nil")
	}
	if service.db != db {
		t.Error("Expected service to hold reference to provided db")
	}
}

func TestWaitlistService_Create_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupWaitlistTable(t, db)

	service := NewWaitlistService(db)
	ctx := context.Background()

	entry, err := service.Create(ctx, "test@example.com", "homepage")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if entry == nil {
		t.Fatal("Create returned nil entry")
	}
	if entry.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", entry.Email)
	}
	if entry.Source != "homepage" {
		t.Errorf("Expected source 'homepage', got '%s'", entry.Source)
	}
	if entry.ID == 0 {
		t.Error("Expected non-zero ID")
	}
	if entry.CreatedAt.IsZero() {
		t.Error("Expected non-zero CreatedAt")
	}
}

func TestWaitlistService_Create_DefaultSource(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupWaitlistTable(t, db)

	service := NewWaitlistService(db)
	ctx := context.Background()

	entry, err := service.Create(ctx, "default@example.com", "")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if entry.Source != "coming_soon" {
		t.Errorf("Expected default source 'coming_soon', got '%s'", entry.Source)
	}
}

func TestWaitlistService_Create_UpsertOnDuplicate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupWaitlistTable(t, db)

	service := NewWaitlistService(db)
	ctx := context.Background()

	// Create first entry
	entry1, err := service.Create(ctx, "upsert@example.com", "source1")
	if err != nil {
		t.Fatalf("First Create failed: %v", err)
	}

	// Create duplicate - should update source
	entry2, err := service.Create(ctx, "upsert@example.com", "source2")
	if err != nil {
		t.Fatalf("Second Create failed: %v", err)
	}

	// IDs should match (same row updated)
	if entry1.ID != entry2.ID {
		t.Errorf("Expected same ID on upsert, got %d and %d", entry1.ID, entry2.ID)
	}
	if entry2.Source != "source2" {
		t.Errorf("Expected updated source 'source2', got '%s'", entry2.Source)
	}

	// Verify only one entry exists
	count, err := service.Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 entry after upsert, got %d", count)
	}
}

func TestWaitlistService_List_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupWaitlistTable(t, db)

	service := NewWaitlistService(db)
	ctx := context.Background()

	entries, err := service.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	// Note: nil slice and empty slice are semantically equivalent in Go
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(entries))
	}
}

func TestWaitlistService_List_OrderByCreatedDesc(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupWaitlistTable(t, db)

	service := NewWaitlistService(db)
	ctx := context.Background()

	// Create multiple entries with slight delay to ensure ordering
	_, err := service.Create(ctx, "first@example.com", "test")
	if err != nil {
		t.Fatalf("Create first failed: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	_, err = service.Create(ctx, "second@example.com", "test")
	if err != nil {
		t.Fatalf("Create second failed: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	_, err = service.Create(ctx, "third@example.com", "test")
	if err != nil {
		t.Fatalf("Create third failed: %v", err)
	}

	entries, err := service.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("Expected 3 entries, got %d", len(entries))
	}

	// Should be ordered by created_at DESC (most recent first)
	if entries[0].Email != "third@example.com" {
		t.Errorf("Expected first entry to be 'third@example.com', got '%s'", entries[0].Email)
	}
	if entries[1].Email != "second@example.com" {
		t.Errorf("Expected second entry to be 'second@example.com', got '%s'", entries[1].Email)
	}
	if entries[2].Email != "first@example.com" {
		t.Errorf("Expected third entry to be 'first@example.com', got '%s'", entries[2].Email)
	}
}

func TestWaitlistService_Delete_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupWaitlistTable(t, db)

	service := NewWaitlistService(db)
	ctx := context.Background()

	entry, err := service.Create(ctx, "delete@example.com", "test")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = service.Delete(ctx, entry.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	count, err := service.Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 entries after delete, got %d", count)
	}
}

func TestWaitlistService_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupWaitlistTable(t, db)

	service := NewWaitlistService(db)
	ctx := context.Background()

	err := service.Delete(ctx, 99999)
	if err == nil {
		t.Fatal("Expected error for non-existent ID")
	}
	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows, got %v", err)
	}
}

func TestWaitlistService_Count_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupWaitlistTable(t, db)

	service := NewWaitlistService(db)
	ctx := context.Background()

	count, err := service.Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0, got %d", count)
	}
}

func TestWaitlistService_Count_WithEntries(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupWaitlistTable(t, db)

	service := NewWaitlistService(db)
	ctx := context.Background()

	// Create multiple entries
	for i := 0; i < 5; i++ {
		email := "count" + string(rune('a'+i)) + "@example.com"
		_, err := service.Create(ctx, email, "test")
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	count, err := service.Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 5 {
		t.Errorf("Expected 5, got %d", count)
	}
}

// cleanupWaitlistTable removes all entries from waitlist_emails table
func cleanupWaitlistTable(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec("DELETE FROM waitlist_emails"); err != nil {
		t.Fatalf("Failed to cleanup waitlist_emails table: %v", err)
	}
}
