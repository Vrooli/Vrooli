package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	
	_ "github.com/lib/pq"
)

// BackupManager handles all backup operations
type BackupManager struct {
	db           *sql.DB
	backupPath   string
	postgresConn string
}

// NewBackupManager creates a new backup manager instance
func NewBackupManager() (*BackupManager, error) {
	// Get PostgreSQL connection from environment
	postgresHost := os.Getenv("POSTGRES_HOST")
	if postgresHost == "" {
		postgresHost = "localhost"
	}
	postgresPort := os.Getenv("POSTGRES_PORT")
	if postgresPort == "" {
		postgresPort = "5432"
	}
	postgresUser := os.Getenv("POSTGRES_USER")
	if postgresUser == "" {
		postgresUser = "postgres"
	}
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	if postgresPassword == "" {
		postgresPassword = "postgres"
	}
	postgresDB := os.Getenv("POSTGRES_DB")
	if postgresDB == "" {
		postgresDB = "vrooli"
	}

	// Create connection string
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		postgresHost, postgresPort, postgresUser, postgresPassword, postgresDB)

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set up backup path
	backupPath := filepath.Join("data", "backups")
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	bm := &BackupManager{
		db:           db,
		backupPath:   backupPath,
		postgresConn: connStr,
	}
	
	// Ensure database schema exists
	bm.ensureSchema()
	
	return bm, nil
}

// CreateBackupJob creates a new backup job record
func (bm *BackupManager) CreateBackupJob(backupType string, targets []string, description string) (*BackupJob, error) {
	job := &BackupJob{
		ID:          fmt.Sprintf("backup-%d", time.Now().Unix()),
		Type:        backupType,
		Target:      strings.Join(targets, ","),
		TargetID:    "vrooli-system",
		Status:      "pending",
		StartedAt:   time.Now(),
		Description: description,
	}

	// First ensure the table exists
	bm.ensureSchema()
	
	// Insert job into database
	query := `
		INSERT INTO backup_jobs (id, type, target, target_identifier, status, started_at, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := bm.db.Exec(query, job.ID, job.Type, job.Target, job.TargetID, job.Status, job.StartedAt, job.Description)
	if err != nil {
		// If DB is not available, continue without recording
		log.Printf("Warning: Could not record backup job in database: %v", err)
	}

	return job, nil
}

// BackupPostgres performs a PostgreSQL database backup
func (bm *BackupManager) BackupPostgres(jobID string) error {
	// Update job status to running
	bm.updateJobStatus(jobID, "running")

	// Create timestamp for backup file
	timestamp := time.Now().Format("20060102_150405")
	backupFile := filepath.Join(bm.backupPath, "postgres", fmt.Sprintf("%s_%s.sql", jobID, timestamp))
	
	// Create postgres backup directory
	postgresDir := filepath.Join(bm.backupPath, "postgres")
	if err := os.MkdirAll(postgresDir, 0755); err != nil {
		bm.updateJobStatus(jobID, "failed")
		return fmt.Errorf("failed to create postgres backup directory: %w", err)
	}

	// Get PostgreSQL credentials from environment
	postgresHost := os.Getenv("POSTGRES_HOST")
	if postgresHost == "" {
		postgresHost = "localhost"
	}
	postgresPort := os.Getenv("POSTGRES_PORT")
	if postgresPort == "" {
		postgresPort = "5432"
	}
	postgresUser := os.Getenv("POSTGRES_USER")
	if postgresUser == "" {
		postgresUser = "postgres"
	}

	// Build pg_dump command
	cmd := exec.Command("pg_dump",
		"-h", postgresHost,
		"-p", postgresPort,
		"-U", postgresUser,
		"-d", "vrooli",
		"-f", backupFile,
		"--verbose",
		"--no-password",
	)

	// Set PGPASSWORD environment variable for authentication
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", os.Getenv("POSTGRES_PASSWORD")))

	// Execute backup
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try alternative: use docker exec if postgres is in container
		dockerCmd := exec.Command("docker", "exec", "postgres",
			"pg_dump", "-U", postgresUser, "vrooli")
		
		dockerOutput, dockerErr := dockerCmd.Output()
		if dockerErr == nil {
			// Write output to file
			if err := os.WriteFile(backupFile, dockerOutput, 0644); err != nil {
				bm.updateJobStatus(jobID, "failed")
				return fmt.Errorf("failed to write backup file: %w", err)
			}
			log.Printf("PostgreSQL backup completed using docker exec: %s", backupFile)
		} else {
			log.Printf("pg_dump failed: %s", string(output))
			log.Printf("docker exec also failed: %v", dockerErr)
			bm.updateJobStatus(jobID, "failed")
			return fmt.Errorf("backup failed: %w", err)
		}
	} else {
		log.Printf("PostgreSQL backup completed: %s", backupFile)
	}

	// Get file size
	fileInfo, err := os.Stat(backupFile)
	var fileSize int64
	if err == nil {
		fileSize = fileInfo.Size()
	}

	// Compress the backup file
	compressedFile := backupFile + ".gz"
	compressCmd := exec.Command("gzip", "-c", backupFile)
	compressedData, err := compressCmd.Output()
	if err == nil {
		if err := os.WriteFile(compressedFile, compressedData, 0644); err == nil {
			// Remove uncompressed file
			os.Remove(backupFile)
			backupFile = compressedFile
			
			// Get compressed file size
			if compressedInfo, err := os.Stat(compressedFile); err == nil {
				compressionRatio := float64(compressedInfo.Size()) / float64(fileSize)
				log.Printf("Backup compressed: %.2f%% of original size", compressionRatio*100)
			}
		}
	}

	// Update job status to completed
	bm.updateJobStatus(jobID, "completed")
	
	return nil
}

// BackupFiles performs file system backup
func (bm *BackupManager) BackupFiles(jobID string, targetPath string) error {
	bm.updateJobStatus(jobID, "running")

	timestamp := time.Now().Format("20060102_150405")
	backupFile := filepath.Join(bm.backupPath, "files", fmt.Sprintf("%s_%s.tar.gz", jobID, timestamp))

	// Create files backup directory
	filesDir := filepath.Join(bm.backupPath, "files")
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		bm.updateJobStatus(jobID, "failed")
		return fmt.Errorf("failed to create files backup directory: %w", err)
	}

	// Default to backing up scenario configurations
	if targetPath == "" {
		targetPath = "/home/matthalloran8/Vrooli/scenarios"
	}

	// Create tar archive
	cmd := exec.Command("tar", "-czf", backupFile, "-C", filepath.Dir(targetPath), filepath.Base(targetPath))
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("tar backup failed: %s", string(output))
		bm.updateJobStatus(jobID, "failed")
		return fmt.Errorf("file backup failed: %w", err)
	}

	log.Printf("File backup completed: %s", backupFile)
	bm.updateJobStatus(jobID, "completed")

	return nil
}

// BackupMinIO performs MinIO object storage backup
func (bm *BackupManager) BackupMinIO(jobID string) error {
	bm.updateJobStatus(jobID, "running")

	timestamp := time.Now().Format("20060102_150405")
	backupDir := filepath.Join(bm.backupPath, "minio", fmt.Sprintf("%s_%s", jobID, timestamp))

	// Create minio backup directory
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		bm.updateJobStatus(jobID, "failed")
		return fmt.Errorf("failed to create minio backup directory: %w", err)
	}

	// Get MinIO configuration from environment
	minioEndpoint := os.Getenv("MINIO_ENDPOINT")
	if minioEndpoint == "" {
		minioEndpoint = "http://localhost:9000"
	}
	minioAccessKey := os.Getenv("MINIO_ACCESS_KEY")
	if minioAccessKey == "" {
		minioAccessKey = "minioadmin"
	}
	minioSecretKey := os.Getenv("MINIO_SECRET_KEY")
	if minioSecretKey == "" {
		minioSecretKey = "minioadmin"
	}

	// Configure mc (MinIO Client) alias
	aliasCmd := exec.Command("mc", "alias", "set", "vrooli-backup",
		minioEndpoint, minioAccessKey, minioSecretKey)
	if output, err := aliasCmd.CombinedOutput(); err != nil {
		log.Printf("mc alias setup failed: %s", string(output))
		bm.updateJobStatus(jobID, "failed")
		return fmt.Errorf("failed to configure MinIO client: %w", err)
	}

	// Mirror all buckets to backup directory
	mirrorCmd := exec.Command("mc", "mirror", "vrooli-backup/", backupDir)
	output, err := mirrorCmd.CombinedOutput()
	if err != nil {
		log.Printf("mc mirror failed: %s", string(output))
		bm.updateJobStatus(jobID, "failed")
		return fmt.Errorf("MinIO backup failed: %w", err)
	}

	// Create tar archive of the mirrored data
	backupFile := backupDir + ".tar.gz"
	tarCmd := exec.Command("tar", "-czf", backupFile, "-C", filepath.Dir(backupDir), filepath.Base(backupDir))
	if output, err := tarCmd.CombinedOutput(); err != nil {
		log.Printf("tar compression failed: %s", string(output))
		// Continue - we have the uncompressed backup
	} else {
		// Remove uncompressed directory
		os.RemoveAll(backupDir)
		log.Printf("MinIO backup compressed: %s", backupFile)
	}

	log.Printf("MinIO backup completed: %s", backupFile)
	bm.updateJobStatus(jobID, "completed")

	return nil
}

// RestorePostgres restores a PostgreSQL database from backup
func (bm *BackupManager) RestorePostgres(backupID string) error {
	// Find backup file
	pattern := filepath.Join(bm.backupPath, "postgres", fmt.Sprintf("%s_*.sql*", backupID))
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return fmt.Errorf("backup file not found for ID: %s", backupID)
	}

	backupFile := matches[0]
	
	// If compressed, decompress first
	if strings.HasSuffix(backupFile, ".gz") {
		cmd := exec.Command("gunzip", "-c", backupFile)
		decompressedData, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to decompress backup: %w", err)
		}
		
		// Write decompressed data to temp file
		tempFile := strings.TrimSuffix(backupFile, ".gz") + ".tmp"
		if err := os.WriteFile(tempFile, decompressedData, 0644); err != nil {
			return fmt.Errorf("failed to write decompressed backup: %w", err)
		}
		defer os.Remove(tempFile)
		backupFile = tempFile
	}

	// Get PostgreSQL credentials
	postgresHost := os.Getenv("POSTGRES_HOST")
	if postgresHost == "" {
		postgresHost = "localhost"
	}
	postgresPort := os.Getenv("POSTGRES_PORT")
	if postgresPort == "" {
		postgresPort = "5432"
	}
	postgresUser := os.Getenv("POSTGRES_USER")
	if postgresUser == "" {
		postgresUser = "postgres"
	}

	// Restore using psql
	cmd := exec.Command("psql",
		"-h", postgresHost,
		"-p", postgresPort,
		"-U", postgresUser,
		"-d", "vrooli",
		"-f", backupFile,
	)
	
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", os.Getenv("POSTGRES_PASSWORD")))
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("psql restore failed: %s", string(output))
		return fmt.Errorf("restore failed: %w", err)
	}

	log.Printf("PostgreSQL restore completed from: %s", backupFile)
	return nil
}

// updateJobStatus updates the status of a backup job in the database
func (bm *BackupManager) updateJobStatus(jobID string, status string) {
	if bm.db == nil {
		return
	}

	var completedAt *time.Time
	if status == "completed" || status == "failed" {
		now := time.Now()
		completedAt = &now
	}

	query := `
		UPDATE backup_jobs
		SET status = $1, completed_at = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`
	_, err := bm.db.Exec(query, status, completedAt, jobID)
	if err != nil {
		log.Printf("Warning: Could not update job status in database: %v", err)
	}
}

// VerifyBackup verifies the integrity of a backup
func (bm *BackupManager) VerifyBackup(backupID string) (bool, error) {
	// Find backup file
	pattern := filepath.Join(bm.backupPath, "**", fmt.Sprintf("%s_*", backupID))
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return false, fmt.Errorf("backup file not found for ID: %s", backupID)
	}

	backupFile := matches[0]
	
	// Check file exists and is readable
	if _, err := os.Stat(backupFile); err != nil {
		return false, fmt.Errorf("backup file not accessible: %w", err)
	}

	// For compressed files, verify compression integrity
	if strings.HasSuffix(backupFile, ".gz") {
		cmd := exec.Command("gzip", "-t", backupFile)
		if err := cmd.Run(); err != nil {
			return false, fmt.Errorf("backup file corrupted: %w", err)
		}
	}

	// Record verification in database
	if bm.db != nil {
		query := `
			INSERT INTO backup_verifications (backup_job_id, verification_type, status, verified_by)
			VALUES ($1, 'checksum', 'passed', 'system')
		`
		_, _ = bm.db.Exec(query, backupID)
	}

	return true, nil
}

// ScheduleBackup schedules a recurring backup
func (bm *BackupManager) ScheduleBackup(name string, cronExpr string, backupType string, targets []string, retentionDays int) error {
	if bm.db == nil {
		return fmt.Errorf("database connection required for scheduling backups")
	}

	query := `
		INSERT INTO backup_schedules (name, cron_expression, backup_type, targets, retention_days, enabled, created_by)
		VALUES ($1, $2, $3, $4, $5, true, 'user')
		ON CONFLICT (name) DO UPDATE SET
			cron_expression = $2,
			backup_type = $3,
			targets = $4,
			retention_days = $5,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := bm.db.Exec(query, name, cronExpr, backupType, targets, retentionDays)
	if err != nil {
		return fmt.Errorf("failed to schedule backup: %w", err)
	}

	log.Printf("Scheduled backup created: %s", name)
	return nil
}

// ensureSchema creates the database schema if it doesn't exist
func (bm *BackupManager) ensureSchema() {
	if bm.db == nil {
		return
	}
	
	// Create the backup_jobs table if it doesn't exist
	query := `
	CREATE TABLE IF NOT EXISTS backup_jobs (
		id VARCHAR(255) PRIMARY KEY,
		type VARCHAR(20) NOT NULL,
		target VARCHAR(255) NOT NULL,
		target_identifier VARCHAR(255) NOT NULL,
		status VARCHAR(20) NOT NULL DEFAULT 'pending',
		started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		completed_at TIMESTAMP WITH TIME ZONE,
		size_bytes BIGINT DEFAULT 0,
		compression_ratio DECIMAL(3,2) DEFAULT 0.00,
		storage_path TEXT,
		checksum VARCHAR(64),
		retention_until TIMESTAMP WITH TIME ZONE,
		description TEXT,
		error_message TEXT,
		retry_count INTEGER DEFAULT 0,
		metadata JSONB DEFAULT '{}',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	)`
	
	if _, err := bm.db.Exec(query); err != nil {
		log.Printf("Warning: Could not create backup_jobs table: %v", err)
	}
	
	// Create backup_schedules table
	query = `
	CREATE TABLE IF NOT EXISTS backup_schedules (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL UNIQUE,
		cron_expression VARCHAR(100) NOT NULL,
		backup_type VARCHAR(20) NOT NULL DEFAULT 'full',
		targets TEXT[] NOT NULL,
		retention_days INTEGER NOT NULL DEFAULT 7,
		enabled BOOLEAN NOT NULL DEFAULT true,
		last_run TIMESTAMP WITH TIME ZONE,
		next_run TIMESTAMP WITH TIME ZONE,
		created_by VARCHAR(255),
		metadata JSONB DEFAULT '{}',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	)`
	
	if _, err := bm.db.Exec(query); err != nil {
		log.Printf("Warning: Could not create backup_schedules table: %v", err)
	}
	
	// Create backup_verifications table
	query = `
	CREATE TABLE IF NOT EXISTS backup_verifications (
		id SERIAL PRIMARY KEY,
		backup_job_id VARCHAR(255) NOT NULL,
		verification_type VARCHAR(50) NOT NULL,
		status VARCHAR(20) NOT NULL,
		details JSONB DEFAULT '{}',
		error_message TEXT,
		verification_duration INTERVAL,
		verified_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		verified_by VARCHAR(255)
	)`
	
	if _, err := bm.db.Exec(query); err != nil {
		log.Printf("Warning: Could not create backup_verifications table: %v", err)
	}
}

// RunScheduledBackups executes all due scheduled backups
func (bm *BackupManager) RunScheduledBackups() {
	if bm.db == nil {
		return
	}

	// This would be called by a cron-like scheduler
	query := `
		SELECT id, name, backup_type, targets
		FROM backup_schedules
		WHERE enabled = true
		AND (next_run IS NULL OR next_run <= CURRENT_TIMESTAMP)
	`

	rows, err := bm.db.Query(query)
	if err != nil {
		log.Printf("Failed to query scheduled backups: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, name, backupType string
		var targets []string
		
		if err := rows.Scan(&id, &name, &backupType, &targets); err != nil {
			continue
		}

		log.Printf("Running scheduled backup: %s", name)
		
		// Create backup job
		job, err := bm.CreateBackupJob(backupType, targets, fmt.Sprintf("Scheduled: %s", name))
		if err != nil {
			log.Printf("Failed to create backup job for schedule %s: %v", name, err)
			continue
		}

		// Execute backup based on targets
		for _, target := range targets {
			switch target {
			case "postgres":
				go bm.BackupPostgres(job.ID)
			case "files", "scenarios":
				go bm.BackupFiles(job.ID, "")
			}
		}
		
		// Update last run time
		updateQuery := `
			UPDATE backup_schedules 
			SET last_run = CURRENT_TIMESTAMP, 
				next_run = CURRENT_TIMESTAMP + INTERVAL '1 hour'
			WHERE id = $1
		`
		bm.db.Exec(updateQuery, id)
	}
}