package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Attachment represents a file attachment for a person
type Attachment struct {
	ID              string                 `json:"id" db:"id"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`
	PersonID        *string                `json:"person_id" db:"person_id"`
	OrganizationID  *string                `json:"organization_id" db:"organization_id"`
	FileName        string                 `json:"file_name" db:"file_name"`
	FileType        string                 `json:"file_type" db:"file_type"`
	MimeType        string                 `json:"mime_type" db:"mime_type"`
	FileSizeBytes   int                    `json:"file_size_bytes" db:"file_size_bytes"`
	StorageBackend  string                 `json:"storage_backend" db:"storage_backend"`
	StoragePath     string                 `json:"storage_path" db:"storage_path"`
	StorageBucket   *string                `json:"storage_bucket" db:"storage_bucket"`
	ThumbnailPath   *string                `json:"thumbnail_path" db:"thumbnail_path"`
	ThumbnailBucket *string                `json:"thumbnail_bucket" db:"thumbnail_bucket"`
	Description     *string                `json:"description" db:"description"`
	Tags            []string               `json:"tags" db:"tags"`
	IsPrimary       bool                   `json:"is_primary" db:"is_primary"`
	IsPublic        bool                   `json:"is_public" db:"is_public"`
	ConsentVerified bool                   `json:"consent_verified" db:"consent_verified"`
	ConsentVerifiedAt *time.Time           `json:"consent_verified_at" db:"consent_verified_at"`
	Metadata        map[string]interface{} `json:"metadata" db:"metadata"`
	DeletedAt       *time.Time             `json:"deleted_at" db:"deleted_at"`
}

// MinIOClient wraps MinIO operations using the resource CLI
type MinIOClient struct {
	bucket string
}

// NewMinIOClient creates a new MinIO client wrapper
func NewMinIOClient() *MinIOClient {
	bucket := os.Getenv("MINIO_BUCKET")
	if bucket == "" {
		bucket = "contact-book"
	}

	// Ensure bucket exists
	cmd := exec.Command("resource-minio", "content", "create-bucket", "--name", bucket)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Check if bucket already exists (not an error)
		if !strings.Contains(string(output), "already exists") {
			log.Printf("Failed to create MinIO bucket: %v - %s", err, output)
		}
	}

	return &MinIOClient{
		bucket: bucket,
	}
}

// UploadFile uploads a file to MinIO using the resource CLI
func (m *MinIOClient) UploadFile(objectKey string, data []byte, contentType string) error {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "upload-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write data to temp file
	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Upload using resource-minio CLI
	cmd := exec.Command("resource-minio", "content", "add",
		"--bucket", m.bucket,
		"--key", objectKey,
		"--file", tmpFile.Name())

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to upload to MinIO: %w - %s", err, output)
	}

	return nil
}

// DownloadFile downloads a file from MinIO using the resource CLI
func (m *MinIOClient) DownloadFile(objectKey string) ([]byte, error) {
	// Create a temporary file for download
	tmpFile, err := os.CreateTemp("", "download-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Download using resource-minio CLI
	cmd := exec.Command("resource-minio", "content", "get",
		"--bucket", m.bucket,
		"--key", objectKey,
		"--output", tmpFile.Name())

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to download from MinIO: %w - %s", err, output)
	}

	// Read the downloaded file
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read downloaded file: %w", err)
	}

	return data, nil
}

// DeleteFile deletes a file from MinIO
func (m *MinIOClient) DeleteFile(objectKey string) error {
	cmd := exec.Command("resource-minio", "content", "remove",
		"--bucket", m.bucket,
		"--key", objectKey)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete from MinIO: %w - %s", err, output)
	}

	return nil
}

var minioClient *MinIOClient

// initMinIO initializes the MinIO client
func initMinIO() {
	minioClient = NewMinIOClient()
	log.Printf("MinIO client initialized with bucket: %s", minioClient.bucket)
}

// uploadAttachment handles file upload for a person
func uploadAttachment(c *gin.Context) {
	personID := c.Param("id")
	if personID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Person ID required"})
		return
	}

	// Parse multipart form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file from request"})
		return
	}
	defer file.Close()

	// Read file data
	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	// Get metadata from form
	fileType := c.DefaultPostForm("file_type", "document")
	description := c.PostForm("description")
	isPrimary := c.DefaultPostForm("is_primary", "false") == "true"
	isPublic := c.DefaultPostForm("is_public", "false") == "true"

	// Generate unique object key
	objectKey := fmt.Sprintf("person/%s/%s/%s", personID, uuid.New().String(), header.Filename)

	// Upload to MinIO if available
	storageBackend := "filesystem"
	storagePath := objectKey
	var storageBucket *string

	if minioClient != nil {
		if err := minioClient.UploadFile(objectKey, data, header.Header.Get("Content-Type")); err != nil {
			log.Printf("Failed to upload to MinIO, falling back to filesystem: %v", err)
		} else {
			storageBackend = "minio"
			storageBucket = &minioClient.bucket
		}
	}

	// If MinIO failed or not available, store reference only
	if storageBackend == "filesystem" {
		// In production, you would save to filesystem here
		// For now, we just store the metadata
		log.Printf("MinIO not available, storing metadata only for file: %s", header.Filename)
	}

	// Parse tags from form
	var tags []string
	if tagsStr := c.PostForm("tags"); tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
	}

	// Create attachment record
	attachment := Attachment{
		ID:             uuid.New().String(),
		PersonID:       &personID,
		FileName:       header.Filename,
		FileType:       fileType,
		MimeType:       header.Header.Get("Content-Type"),
		FileSizeBytes:  int(header.Size),
		StorageBackend: storageBackend,
		StoragePath:    storagePath,
		StorageBucket:  storageBucket,
		Description:    nilIfEmpty(description),
		Tags:           tags,
		IsPrimary:      isPrimary,
		IsPublic:       isPublic,
	}

	// Insert into database
	query := `
		INSERT INTO attachments (
			id, person_id, file_name, file_type, mime_type, file_size_bytes,
			storage_backend, storage_path, storage_bucket, description,
			tags, is_primary, is_public
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
		RETURNING created_at, updated_at`

	err = db.QueryRow(query,
		attachment.ID, attachment.PersonID, attachment.FileName,
		attachment.FileType, attachment.MimeType, attachment.FileSizeBytes,
		attachment.StorageBackend, attachment.StoragePath, attachment.StorageBucket,
		attachment.Description, pq.Array(attachment.Tags), attachment.IsPrimary,
		attachment.IsPublic,
	).Scan(&attachment.CreatedAt, &attachment.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save attachment: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, attachment)
}

// getAttachments returns all attachments for a person
func getAttachments(c *gin.Context) {
	personID := c.Param("id")

	query := `
		SELECT
			id, created_at, updated_at, person_id, organization_id,
			file_name, file_type, mime_type, file_size_bytes,
			storage_backend, storage_path, storage_bucket,
			thumbnail_path, thumbnail_bucket, description,
			tags, is_primary, is_public, consent_verified,
			consent_verified_at, metadata
		FROM attachments
		WHERE person_id = $1 AND deleted_at IS NULL
		ORDER BY is_primary DESC, created_at DESC`

	rows, err := db.Query(query, personID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch attachments"})
		return
	}
	defer rows.Close()

	attachments := []Attachment{}
	for rows.Next() {
		var a Attachment
		var metadata, tagsArray interface{}

		err := rows.Scan(
			&a.ID, &a.CreatedAt, &a.UpdatedAt, &a.PersonID, &a.OrganizationID,
			&a.FileName, &a.FileType, &a.MimeType, &a.FileSizeBytes,
			&a.StorageBackend, &a.StoragePath, &a.StorageBucket,
			&a.ThumbnailPath, &a.ThumbnailBucket, &a.Description,
			&tagsArray, &a.IsPrimary, &a.IsPublic, &a.ConsentVerified,
			&a.ConsentVerifiedAt, &metadata,
		)

		if err != nil {
			log.Printf("Error scanning attachment: %v", err)
			continue
		}

		// Parse JSONB fields
		if metadata != nil {
			if metaBytes, ok := metadata.([]byte); ok {
				json.Unmarshal(metaBytes, &a.Metadata)
			}
		}

		// Parse tags array
		a.Tags = parseStringArray(tagsArray)

		attachments = append(attachments, a)
	}

	c.JSON(http.StatusOK, gin.H{
		"attachments": attachments,
		"count":       len(attachments),
	})
}

// downloadAttachment downloads an attachment file
func downloadAttachment(c *gin.Context) {
	attachmentID := c.Param("attachmentId")

	// Get attachment metadata
	var a Attachment
	query := `
		SELECT
			file_name, mime_type, storage_backend, storage_path, storage_bucket
		FROM attachments
		WHERE id = $1 AND deleted_at IS NULL`

	err := db.QueryRow(query, attachmentID).Scan(
		&a.FileName, &a.MimeType, &a.StorageBackend, &a.StoragePath, &a.StorageBucket,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Attachment not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch attachment"})
		return
	}

	// Download from MinIO if stored there
	if a.StorageBackend == "minio" && minioClient != nil {
		data, err := minioClient.DownloadFile(a.StoragePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to download file: %v", err)})
			return
		}

		// Set headers and send file
		c.Header("Content-Type", a.MimeType)
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", a.FileName))
		c.Data(http.StatusOK, a.MimeType, data)
		return
	}

	// For filesystem or URL storage, return the path/URL
	c.JSON(http.StatusOK, gin.H{
		"file_name":    a.FileName,
		"storage_path": a.StoragePath,
		"message":      "File metadata only - actual file not available",
	})
}

// deleteAttachment deletes an attachment
func deleteAttachment(c *gin.Context) {
	attachmentID := c.Param("attachmentId")

	// Get attachment info first
	var storagePath string
	var storageBackend string
	err := db.QueryRow(`
		SELECT storage_path, storage_backend
		FROM attachments
		WHERE id = $1 AND deleted_at IS NULL`,
		attachmentID).Scan(&storagePath, &storageBackend)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Attachment not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch attachment"})
		return
	}

	// Delete from MinIO if stored there
	if storageBackend == "minio" && minioClient != nil {
		if err := minioClient.DeleteFile(storagePath); err != nil {
			log.Printf("Failed to delete file from MinIO: %v", err)
			// Continue with database deletion even if MinIO fails
		}
	}

	// Soft delete in database
	_, err = db.Exec(`
		UPDATE attachments
		SET deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`,
		attachmentID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete attachment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Attachment deleted successfully"})
}

// Helper function to convert empty string to nil
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Helper function to parse string array from database
func parseStringArray(input interface{}) []string {
	if input == nil {
		return []string{}
	}

	switch v := input.(type) {
	case []string:
		return v
	case pq.StringArray:
		return []string(v)
	default:
		return []string{}
	}
}