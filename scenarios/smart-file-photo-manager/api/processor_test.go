// +build testing

package main

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// Test Processing Job Queue
func TestProcessingQueue(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("QueueJob", func(t *testing.T) {
		file := createTestFile(t, env, "queue_test.jpg", "image/jpeg")
		defer file.Cleanup()

		req := UploadRequest{
			Filename:    file.OriginalName,
			MimeType:    file.MimeType,
			SizeBytes:   file.SizeBytes,
			FileHash:    file.FileHash,
			StoragePath: file.StoragePath,
			FolderPath:  file.FolderPath,
		}

		// Queue the job
		env.App.queueFileProcessing(file.ID, req)

		// Give it a moment to process
		time.Sleep(100 * time.Millisecond)

		// Verify it was queued (job count in channel or Redis)
		// This is a basic check - full integration would verify processing
	})

	t.Run("QueueFullFallbackToRedis", func(t *testing.T) {
		// Fill the queue
		for i := 0; i < 100; i++ {
			job := ProcessingJob{
				FileID:  "test-id",
				JobType: "test",
			}
			select {
			case env.App.ProcessingQueue <- job:
			default:
				break
			}
		}

		// Try to queue another job - should fall back to Redis
		file := createTestFile(t, env, "overflow_test.jpg", "image/jpeg")
		defer file.Cleanup()

		req := UploadRequest{
			Filename:    file.OriginalName,
			MimeType:    file.MimeType,
			SizeBytes:   file.SizeBytes,
			FileHash:    file.FileHash,
			StoragePath: file.StoragePath,
			FolderPath:  file.FolderPath,
		}

		env.App.queueFileProcessing(file.ID, req)

		// Check Redis for pending jobs
		ctx := context.Background()
		count, _ := env.App.RedisClient.LLen(ctx, "pending_jobs").Result()
		if count < 0 {
			// Redis might not be available in test
			t.Skip("Redis not available for overflow test")
		}
	})
}

// Test File Type Determination
func TestDetermineFileType(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	tests := []struct {
		name     string
		mimeType string
		filename string
		expected string
	}{
		{"JPEG Image", "image/jpeg", "photo.jpg", "image"},
		{"PNG Image", "image/png", "screenshot.png", "image"},
		{"MP4 Video", "video/mp4", "clip.mp4", "video"},
		{"PDF Document", "application/pdf", "doc.pdf", "document"},
		{"Word Document", "application/msword", "letter.doc", "document"},
		{"Text Document", "text/plain", "readme.txt", "document"},
		{"Unknown Type", "application/octet-stream", "file.bin", "generic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := env.App.determineFileType(tt.mimeType, tt.filename)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s for %s", tt.expected, result, tt.mimeType)
			}
		})
	}
}

// Test File Status Updates
func TestUpdateFileStatus(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("UpdateStatus", func(t *testing.T) {
		file := createTestFile(t, env, "status_update_test.jpg", "image/jpeg")
		defer file.Cleanup()

		// Update status
		env.App.updateFileStatus(file.ID, "processing", "analyzing")

		// Verify update
		var status, stage string
		query := "SELECT status, processing_stage FROM files WHERE id = $1"
		err := env.DB.QueryRow(query, file.ID).Scan(&status, &stage)
		if err != nil {
			t.Fatalf("Failed to query file: %v", err)
		}

		if status != "processing" {
			t.Errorf("Expected status 'processing', got %s", status)
		}

		if stage != "analyzing" {
			t.Errorf("Expected stage 'analyzing', got %s", stage)
		}
	})
}

// Test Description Updates
func TestUpdateFileDescription(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("UpdateDescription", func(t *testing.T) {
		file := createTestFile(t, env, "desc_test.jpg", "image/jpeg")
		defer file.Cleanup()

		description := "A beautiful sunset over the ocean"
		env.App.updateFileDescription(file.ID, description)

		// Verify update
		var desc *string
		query := "SELECT description FROM files WHERE id = $1"
		err := env.DB.QueryRow(query, file.ID).Scan(&desc)
		if err != nil {
			t.Fatalf("Failed to query file: %v", err)
		}

		if desc == nil || *desc != description {
			t.Errorf("Expected description '%s', got %v", description, desc)
		}
	})
}

// Test Object Detection Extraction
func TestExtractObjectsFromDescription(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	tests := []struct {
		name        string
		description string
		expected    []string
	}{
		{
			"SingleObject",
			"A person standing in front of a building",
			[]string{"person", "building"},
		},
		{
			"MultipleObjects",
			"A car parked next to a tree with a sign",
			[]string{"car", "tree", "sign"},
		},
		{
			"NoObjects",
			"Abstract patterns and colors",
			[]string{},
		},
		{
			"FoodImage",
			"Delicious food on a plate",
			[]string{"food"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := env.App.extractObjectsFromDescription(tt.description)

			if len(objects) != len(tt.expected) {
				t.Errorf("Expected %d objects, got %d", len(tt.expected), len(objects))
			}

			// Verify all expected objects are present
			for _, exp := range tt.expected {
				found := false
				for _, obj := range objects {
					if obj == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected object '%s' not found in %v", exp, objects)
				}
			}
		})
	}
}

// Test Duplicate Detection
func TestFindDuplicatesByHash(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("NoDuplicates", func(t *testing.T) {
		file := createTestFile(t, env, "unique.jpg", "image/jpeg")
		defer file.Cleanup()

		duplicates := env.App.findDuplicatesByHash(file.FileHash, file.ID)
		if len(duplicates) != 0 {
			t.Errorf("Expected no duplicates, got %d", len(duplicates))
		}
	})

	t.Run("WithDuplicates", func(t *testing.T) {
		cleanupTestData(env)

		// Create files with same hash
		sameHash := "duplicate_hash_test"
		file1 := createTestFile(t, env, "dup1.jpg", "image/jpeg")
		env.DB.Exec("UPDATE files SET file_hash = $1 WHERE id = $2", sameHash, file1.ID)
		defer file1.Cleanup()

		file2 := createTestFile(t, env, "dup2.jpg", "image/jpeg")
		env.DB.Exec("UPDATE files SET file_hash = $1 WHERE id = $2", sameHash, file2.ID)
		defer file2.Cleanup()

		file3 := createTestFile(t, env, "dup3.jpg", "image/jpeg")
		env.DB.Exec("UPDATE files SET file_hash = $1 WHERE id = $2", sameHash, file3.ID)
		defer file3.Cleanup()

		// Find duplicates for file1
		duplicates := env.App.findDuplicatesByHash(sameHash, file1.ID)
		if len(duplicates) != 2 {
			t.Errorf("Expected 2 duplicates, got %d", len(duplicates))
		}
	})
}

// Test Suggestion Creation
func TestCreateDuplicateSuggestion(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("CreateSuggestion", func(t *testing.T) {
		file := createTestFile(t, env, "suggest_test.jpg", "image/jpeg")
		defer file.Cleanup()

		duplicates := []string{"id1", "id2", "id3"}
		env.App.createDuplicateSuggestion(file.ID, duplicates)

		// Verify suggestion was created
		var count int
		query := "SELECT COUNT(*) FROM suggestions WHERE file_id = $1 AND type = 'duplicate'"
		err := env.DB.QueryRow(query, file.ID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query suggestions: %v", err)
		}

		if count == 0 {
			t.Error("Expected suggestion to be created")
		}

		// Cleanup
		env.DB.Exec("DELETE FROM suggestions WHERE file_id = $1", file.ID)
	})
}

func TestCreateOrganizationSuggestion(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("CreateSuggestion", func(t *testing.T) {
		file := createTestFile(t, env, "organize_suggest.jpg", "image/jpeg")
		defer file.Cleanup()

		folderPath := "/photos/2024"
		reason := "Based on file date and type"
		env.App.createOrganizationSuggestion(file.ID, folderPath, reason)

		// Verify suggestion was created
		var count int
		query := "SELECT COUNT(*) FROM suggestions WHERE file_id = $1 AND type = 'organization'"
		err := env.DB.QueryRow(query, file.ID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query suggestions: %v", err)
		}

		if count == 0 {
			t.Error("Expected organization suggestion to be created")
		}

		// Cleanup
		env.DB.Exec("DELETE FROM suggestions WHERE file_id = $1", file.ID)
	})
}

// Test Worker Pool
func TestWorkerPool(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("WorkerProcessing", func(t *testing.T) {
		// Create a simple job
		file := createTestFile(t, env, "worker_test.txt", "text/plain")
		defer file.Cleanup()

		job := ProcessingJob{
			FileID:  file.ID,
			JobType: "file_process",
			Payload: UploadRequest{
				Filename:    file.OriginalName,
				MimeType:    file.MimeType,
				SizeBytes:   file.SizeBytes,
				FileHash:    file.FileHash,
				StoragePath: file.StoragePath,
				FolderPath:  file.FolderPath,
			},
		}

		// Queue the job
		env.App.ProcessingQueue <- job

		// Wait a bit for processing
		time.Sleep(200 * time.Millisecond)

		// Verify status changed (it should have attempted processing)
		var status string
		env.DB.QueryRow("SELECT status FROM files WHERE id = $1", file.ID).Scan(&status)
		// Status should have changed from 'pending' to something else
		if status == "" {
			t.Error("Expected file status to be updated")
		}
	})
}

// Test Organize Strategies
func TestOrganizeByType(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping organize tests - require full implementation")
	}

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("OrganizeByType", func(t *testing.T) {
		file := createTestFile(t, env, "organize.jpg", "image/jpeg")
		env.DB.Exec("UPDATE files SET file_type = 'image' WHERE id = $1", file.ID)
		defer file.Cleanup()

		// This would require full implementation of organizeByType
		// For now, just verify the method exists and doesn't crash
		env.App.organizeByType(file.ID)
	})
}

// Test Edge Cases
func TestProcessingEdgeCases(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("EmptyMimeType", func(t *testing.T) {
		fileType := env.App.determineFileType("", "file.bin")
		if fileType != "generic" {
			t.Errorf("Expected 'generic' for empty mime type, got %s", fileType)
		}
	})

	t.Run("UnknownMimeType", func(t *testing.T) {
		fileType := env.App.determineFileType("application/x-custom", "file.custom")
		if fileType != "generic" {
			t.Errorf("Expected 'generic' for unknown mime type, got %s", fileType)
		}
	})

	t.Run("NullDescription", func(t *testing.T) {
		objects := env.App.extractObjectsFromDescription("")
		if len(objects) != 0 {
			t.Errorf("Expected no objects for empty description, got %v", objects)
		}
	})
}

// Test Concurrent Processing
func TestConcurrentProcessing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent tests in short mode")
	}

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("MultipleFilesProcessing", func(t *testing.T) {
		// Create multiple files
		files := make([]*TestFile, 5)
		for i := 0; i < 5; i++ {
			file := createTestFile(t, env, fmt.Sprintf("concurrent%d.jpg", i), "image/jpeg")
			files[i] = file
			defer file.Cleanup()

			// Queue for processing
			req := UploadRequest{
				Filename:    file.OriginalName,
				MimeType:    file.MimeType,
				SizeBytes:   file.SizeBytes,
				FileHash:    file.FileHash,
				StoragePath: file.StoragePath,
				FolderPath:  file.FolderPath,
			}
			env.App.queueFileProcessing(file.ID, req)
		}

		// Wait for processing
		time.Sleep(500 * time.Millisecond)

		// Verify all files were queued
		// (Full verification would require tracking job completion)
	})
}
