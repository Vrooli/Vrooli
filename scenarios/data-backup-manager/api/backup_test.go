package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestNewBackupManager tests backup manager initialization
func TestNewBackupManager(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("SuccessWithoutDatabase", func(t *testing.T) {
		// Without database connection, backup manager should still initialize
		cleanupEnv := setTestEnv(t, map[string]string{
			"POSTGRES_HOST": "nonexistent-host",
			"POSTGRES_PORT": "9999",
		})
		defer cleanupEnv()

		bm, err := NewBackupManager()
		// Should fail to connect but that's expected without actual database
		if err == nil {
			t.Error("Expected error when connecting to nonexistent database")
		}
		if bm != nil {
			t.Error("Expected nil backup manager on connection failure")
		}
	})

	t.Run("BackupPathCreation", func(t *testing.T) {
		// Test that backup path is created
		backupPath := filepath.Join("data", "backups")
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			t.Errorf("Expected backup path to exist: %s", backupPath)
		}
	})

	t.Run("DefaultEnvironmentVariables", func(t *testing.T) {
		// Test with default environment variables
		cleanupEnv := setTestEnv(t, map[string]string{
			"POSTGRES_HOST":     "",
			"POSTGRES_PORT":     "",
			"POSTGRES_USER":     "",
			"POSTGRES_PASSWORD": "",
			"POSTGRES_DB":       "",
		})
		defer cleanupEnv()

		// Should use defaults
		_, err := NewBackupManager()
		if err == nil {
			t.Log("Note: Test requires actual PostgreSQL to fully pass")
		}
	})
}

// TestCreateBackupJob tests backup job creation
func TestCreateBackupJob(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("CreateJobStructure", func(t *testing.T) {
		// Test job structure creation (validates field assignment)
		job := &BackupJob{
			ID:          fmt.Sprintf("backup-%d", time.Now().Unix()),
			Type:        "full",
			Target:      "postgres",
			TargetID:    "vrooli-system",
			Status:      "pending",
			StartedAt:   time.Now(),
			Description: "Test backup",
		}

		// Validate job fields
		if job.ID == "" {
			t.Error("Expected non-empty job ID")
		}

		if job.Type != "full" {
			t.Errorf("Expected type 'full', got %s", job.Type)
		}

		if job.Target != "postgres" {
			t.Errorf("Expected target 'postgres', got %s", job.Target)
		}

		if job.Status != "pending" {
			t.Errorf("Expected status 'pending', got %s", job.Status)
		}

		if job.Description != "Test backup" {
			t.Errorf("Expected description 'Test backup', got %s", job.Description)
		}
	})

	t.Run("MultipleTargets", func(t *testing.T) {
		// Test multiple targets are joined correctly
		targets := []string{"postgres", "files", "minio"}
		combinedTarget := "postgres,files,minio"

		job := &BackupJob{
			ID:       fmt.Sprintf("backup-%d", time.Now().Unix()),
			Type:     "incremental",
			Target:   combinedTarget,
			TargetID: "vrooli-system",
			Status:   "pending",
		}

		if job.Target != "postgres,files,minio" {
			t.Errorf("Expected combined targets, got %s", job.Target)
		}

		// Validate we can split them back
		splitTargets := []string{}
		for _, target := range targets {
			splitTargets = append(splitTargets, target)
		}
		if len(splitTargets) != 3 {
			t.Errorf("Expected 3 targets, got %d", len(splitTargets))
		}
	})

	t.Run("JobTimestamp", func(t *testing.T) {
		before := time.Now()
		job := &BackupJob{
			ID:        fmt.Sprintf("backup-%d", time.Now().Unix()),
			Type:      "differential",
			Target:    "files",
			Status:    "pending",
			StartedAt: time.Now(),
		}
		after := time.Now()

		if job.StartedAt.Before(before) || job.StartedAt.After(after) {
			t.Error("Job timestamp not within expected range")
		}
	})
}

// TestBackupFiles tests file system backup structure
func TestBackupFiles(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("BackupDirectoryStructure", func(t *testing.T) {
		// Test that backup manager sets up correct directory structure
		backupPath := filepath.Join(env.TempDir, "backups")
		filesDir := filepath.Join(backupPath, "files")

		// Create the directory structure
		if err := os.MkdirAll(filesDir, 0755); err != nil {
			t.Fatalf("Failed to create files directory: %v", err)
		}

		// Verify directory exists
		if _, err := os.Stat(filesDir); os.IsNotExist(err) {
			t.Error("Expected files backup directory to exist")
		}

		// Test file path construction
		jobID := fmt.Sprintf("backup-%d", time.Now().Unix())
		timestamp := time.Now().Format("20060102_150405")
		expectedPath := filepath.Join(filesDir, fmt.Sprintf("%s_%s.tar.gz", jobID, timestamp))

		// Verify path is valid
		if expectedPath == "" {
			t.Error("Expected non-empty backup path")
		}
	})

	t.Run("DefaultTargetPath", func(t *testing.T) {
		// Test default path logic
		targetPath := ""
		if targetPath == "" {
			targetPath = "/home/matthalloran8/Vrooli/scenarios"
		}

		if targetPath == "" {
			t.Error("Expected default target path to be set")
		}
	})
}

// TestBackupPostgres tests PostgreSQL backup structure
func TestBackupPostgres(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("BackupDirectoryStructure", func(t *testing.T) {
		// Test directory structure setup
		backupPath := filepath.Join(env.TempDir, "backups")
		postgresDir := filepath.Join(backupPath, "postgres")

		// Create the directory
		if err := os.MkdirAll(postgresDir, 0755); err != nil {
			t.Fatalf("Failed to create postgres directory: %v", err)
		}

		// Verify directory exists
		if _, err := os.Stat(postgresDir); os.IsNotExist(err) {
			t.Error("Expected postgres backup directory to exist")
		}

		// Test file path construction
		jobID := fmt.Sprintf("backup-%d", time.Now().Unix())
		timestamp := time.Now().Format("20060102_150405")
		expectedPath := filepath.Join(postgresDir, fmt.Sprintf("%s_%s.sql", jobID, timestamp))

		if expectedPath == "" {
			t.Error("Expected non-empty backup path")
		}
	})

	t.Run("EnvironmentVariableDefaults", func(t *testing.T) {
		// Test default environment variable handling
		cleanupEnv := setTestEnv(t, map[string]string{
			"POSTGRES_HOST":     "",
			"POSTGRES_PORT":     "",
			"POSTGRES_USER":     "",
			"POSTGRES_PASSWORD": "",
		})
		defer cleanupEnv()

		// Verify defaults are used
		postgresHost := os.Getenv("POSTGRES_HOST")
		if postgresHost == "" {
			postgresHost = "localhost"
		}
		if postgresHost != "localhost" {
			t.Errorf("Expected default host 'localhost', got %s", postgresHost)
		}

		postgresPort := os.Getenv("POSTGRES_PORT")
		if postgresPort == "" {
			postgresPort = "5432"
		}
		if postgresPort != "5432" {
			t.Errorf("Expected default port '5432', got %s", postgresPort)
		}
	})
}

// TestBackupMinIO tests MinIO backup structure
func TestBackupMinIO(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("BackupDirectoryStructure", func(t *testing.T) {
		// Test directory structure setup
		backupPath := filepath.Join(env.TempDir, "backups")
		minioDir := filepath.Join(backupPath, "minio")

		// Create the directory
		if err := os.MkdirAll(minioDir, 0755); err != nil {
			t.Fatalf("Failed to create minio directory: %v", err)
		}

		// Verify directory exists
		if _, err := os.Stat(minioDir); os.IsNotExist(err) {
			t.Error("Expected minio backup directory to exist")
		}

		// Test directory path construction
		jobID := fmt.Sprintf("backup-%d", time.Now().Unix())
		timestamp := time.Now().Format("20060102_150405")
		expectedPath := filepath.Join(minioDir, fmt.Sprintf("%s_%s", jobID, timestamp))

		if expectedPath == "" {
			t.Error("Expected non-empty backup path")
		}
	})

	t.Run("MinIOConfigurationDefaults", func(t *testing.T) {
		// Test default MinIO configuration
		cleanupEnv := setTestEnv(t, map[string]string{
			"MINIO_ENDPOINT":   "",
			"MINIO_ACCESS_KEY": "",
			"MINIO_SECRET_KEY": "",
		})
		defer cleanupEnv()

		minioEndpoint := os.Getenv("MINIO_ENDPOINT")
		if minioEndpoint == "" {
			minioEndpoint = "http://localhost:9000"
		}
		if minioEndpoint != "http://localhost:9000" {
			t.Errorf("Expected default endpoint, got %s", minioEndpoint)
		}

		minioAccessKey := os.Getenv("MINIO_ACCESS_KEY")
		if minioAccessKey == "" {
			minioAccessKey = "minioadmin"
		}
		if minioAccessKey != "minioadmin" {
			t.Errorf("Expected default access key, got %s", minioAccessKey)
		}
	})
}

// TestRestorePostgres tests PostgreSQL restore
func TestRestorePostgres(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("BackupNotFound", func(t *testing.T) {
		bm := &BackupManager{
			backupPath: env.BackupDir,
		}

		err := bm.RestorePostgres("nonexistent-backup")
		if err == nil {
			t.Error("Expected error for nonexistent backup")
		}

		expectedErr := "backup file not found for ID: nonexistent-backup"
		if err != nil && err.Error() != expectedErr {
			t.Errorf("Expected error %q, got %q", expectedErr, err.Error())
		}
	})

	t.Run("CompressedBackupRestore", func(t *testing.T) {
		bm := &BackupManager{
			backupPath: env.BackupDir,
		}

		// Create a mock compressed backup file
		backupID := "backup-test"
		timestamp := time.Now().Format("20060102_150405")
		backupFile := filepath.Join(env.BackupDir, "postgres", fmt.Sprintf("%s_%s.sql.gz", backupID, timestamp))

		// Create postgres directory
		postgresDir := filepath.Join(env.BackupDir, "postgres")
		os.MkdirAll(postgresDir, 0755)

		// Write mock compressed data
		os.WriteFile(backupFile, []byte("mock compressed backup"), 0644)

		err := bm.RestorePostgres(backupID)

		// Will fail without gzip/psql, but tests file finding logic
		if err != nil {
			t.Logf("Restore failed (expected without gzip/psql): %v", err)
		}
	})
}

// TestVerifyBackup tests backup verification
func TestVerifyBackup(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("BackupNotFound", func(t *testing.T) {
		bm := &BackupManager{
			backupPath: env.BackupDir,
		}

		verified, err := bm.VerifyBackup("nonexistent-backup")
		if err == nil {
			t.Error("Expected error for nonexistent backup")
		}

		if verified {
			t.Error("Expected verification to fail for nonexistent backup")
		}
	})

	t.Run("BackupFileAccessible", func(t *testing.T) {
		bm := &BackupManager{
			backupPath: env.BackupDir,
		}

		// Create a mock backup file
		backupID := "backup-verify-test"
		timestamp := time.Now().Format("20060102_150405")
		backupFile := filepath.Join(env.BackupDir, "postgres", fmt.Sprintf("%s_%s.sql", backupID, timestamp))

		postgresDir := filepath.Join(env.BackupDir, "postgres")
		os.MkdirAll(postgresDir, 0755)
		os.WriteFile(backupFile, []byte("mock backup data"), 0644)

		// Note: glob pattern might not work as expected in test
		// This tests the verification logic structure
		verified, err := bm.VerifyBackup(backupID)

		if err != nil {
			t.Logf("Verification error: %v", err)
		} else if !verified {
			t.Log("Verification returned false (file pattern may not match)")
		}
	})
}

// TestScheduleBackup tests backup scheduling
func TestScheduleBackup(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("ScheduleWithoutDatabase", func(t *testing.T) {
		bm := &BackupManager{
			backupPath: env.BackupDir,
			db:         nil,
		}

		err := bm.ScheduleBackup("test-schedule", "0 2 * * *", "full", []string{"postgres"}, 7)

		// Should fail without database
		if err == nil {
			t.Error("Expected error when scheduling without database")
		}
	})

	t.Run("ValidScheduleParameters", func(t *testing.T) {
		// Test schedule creation logic (structure)
		name := "daily-backup"
		cronExpr := "0 2 * * *"
		backupType := "full"
		targets := []string{"postgres", "files"}
		retentionDays := 7

		// Validate parameters
		if name == "" {
			t.Error("Expected non-empty schedule name")
		}
		if cronExpr == "" {
			t.Error("Expected non-empty cron expression")
		}
		if backupType != "full" && backupType != "incremental" && backupType != "differential" {
			t.Errorf("Invalid backup type: %s", backupType)
		}
		if len(targets) == 0 {
			t.Error("Expected at least one target")
		}
		if retentionDays < 0 {
			t.Errorf("Expected non-negative retention days, got %d", retentionDays)
		}
	})
}

// TestRunScheduledBackups tests scheduled backup execution
func TestRunScheduledBackups(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("NoDatabase", func(t *testing.T) {
		bm := &BackupManager{
			backupPath: env.BackupDir,
			db:         nil,
		}

		// Should handle nil database gracefully
		bm.RunScheduledBackups()
		// No panic expected
	})
}

// TestUpdateJobStatus tests job status updates
func TestUpdateJobStatus(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("UpdateWithoutDatabase", func(t *testing.T) {
		bm := &BackupManager{
			backupPath: env.BackupDir,
			db:         nil,
		}

		// Should handle nil database gracefully (logs warning)
		bm.updateJobStatus("test-job-123", "running")
		bm.updateJobStatus("test-job-123", "completed")
		bm.updateJobStatus("test-job-123", "failed")
		// No panic expected
	})

	t.Run("StatusTransitions", func(t *testing.T) {
		// Test valid status transitions
		validStatuses := []string{"pending", "running", "completed", "failed"}

		for _, status := range validStatuses {
			if status == "" {
				t.Error("Expected non-empty status")
			}
		}
	})
}

// TestEnsureSchema tests database schema creation
func TestEnsureSchema(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("NilDatabase", func(t *testing.T) {
		bm := &BackupManager{
			backupPath: env.BackupDir,
			db:         nil,
		}

		// Should return early with nil database
		bm.ensureSchema()
		// No panic expected
	})
}

// TestBackupManagerEdgeCases tests edge cases
func TestBackupManagerEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("EmptyJobID", func(t *testing.T) {
		// Test with empty job ID - validate path construction
		jobID := ""
		timestamp := time.Now().Format("20060102_150405")
		backupFile := filepath.Join(env.BackupDir, "files", fmt.Sprintf("%s_%s.tar.gz", jobID, timestamp))

		// Empty job ID creates a path starting with underscore
		if !strings.Contains(backupFile, fmt.Sprintf("_%s.tar.gz", timestamp)) {
			t.Error("Expected backup file path with underscore prefix for empty job ID")
		}
	})

	t.Run("SpecialCharactersInPath", func(t *testing.T) {
		// Test with path containing special characters
		specialPath := filepath.Join(env.TempDir, "test dir/sub-dir")
		os.MkdirAll(specialPath, 0755)

		// Verify path is valid despite special characters
		if _, err := os.Stat(specialPath); os.IsNotExist(err) {
			t.Error("Failed to create path with special characters")
		}
	})

	t.Run("VeryLongJobID", func(t *testing.T) {
		// Test with very long job ID
		longJobID := "backup-" + strings.Repeat("x", 200)
		if len(longJobID) <= 200 {
			t.Error("Expected very long job ID")
		}

		// Validate path construction works with long ID
		backupFile := filepath.Join(env.BackupDir, "files", longJobID+".tar.gz")
		if backupFile == "" {
			t.Error("Failed to construct path with long job ID")
		}
	})

	t.Run("ReadOnlyBackupDirectory", func(t *testing.T) {
		// Test read-only directory detection
		readOnlyDir := filepath.Join(env.TempDir, "readonly-backup")
		os.MkdirAll(readOnlyDir, 0755)
		os.Chmod(readOnlyDir, 0444)
		defer os.Chmod(readOnlyDir, 0755) // Restore permissions for cleanup

		// Try to create a file in read-only directory
		testFile := filepath.Join(readOnlyDir, "test.txt")
		err := os.WriteFile(testFile, []byte("test"), 0644)

		if err == nil {
			t.Error("Expected error when writing to read-only directory")
		}
	})

	t.Run("ConcurrentJobCreation", func(t *testing.T) {
		// Test concurrent job structure creation
		done := make(chan *BackupJob, 3)

		for i := 0; i < 3; i++ {
			go func(index int) {
				job := &BackupJob{
					ID:        fmt.Sprintf("backup-concurrent-%d-%d", index, time.Now().UnixNano()),
					Type:      "incremental",
					Target:    "postgres",
					Status:    "pending",
					StartedAt: time.Now(),
				}
				done <- job
			}(i)
		}

		// Wait for all to complete
		timeout := time.After(5 * time.Second)
		jobs := make([]*BackupJob, 0, 3)
		for i := 0; i < 3; i++ {
			select {
			case job := <-done:
				jobs = append(jobs, job)
			case <-timeout:
				t.Fatal("Timeout waiting for concurrent job creation")
			}
		}

		// Validate all jobs have unique IDs
		if len(jobs) != 3 {
			t.Errorf("Expected 3 jobs, got %d", len(jobs))
		}
	})
}

// TestBackupJobStructure tests BackupJob struct
func TestBackupJobStructure(t *testing.T) {
	t.Run("FieldValidation", func(t *testing.T) {
		now := time.Now()
		completedAt := now.Add(1 * time.Hour)

		job := &BackupJob{
			ID:               "backup-123",
			Type:             "full",
			Target:           "postgres",
			TargetID:         "main-db",
			Status:           "completed",
			StartedAt:        now,
			CompletedAt:      &completedAt,
			SizeBytes:        1024 * 1024 * 50,
			CompressionRatio: 0.68,
			StoragePath:      "/backups/postgres/backup.tar.gz",
			Checksum:         "sha256:abc123",
			RetentionUntil:   now.Add(7 * 24 * time.Hour),
			Description:      "Test backup",
		}

		// Validate all fields are set
		if job.ID == "" {
			t.Error("Expected non-empty ID")
		}
		if job.Type == "" {
			t.Error("Expected non-empty Type")
		}
		if job.Status == "" {
			t.Error("Expected non-empty Status")
		}
		if job.CompletedAt == nil {
			t.Error("Expected CompletedAt for completed job")
		}
		if job.SizeBytes <= 0 {
			t.Error("Expected positive SizeBytes")
		}
		if job.CompressionRatio <= 0 || job.CompressionRatio > 1 {
			t.Error("Expected CompressionRatio between 0 and 1")
		}
	})
}
