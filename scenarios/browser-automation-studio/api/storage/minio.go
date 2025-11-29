package storage

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
)

// MinIOClient wraps MinIO operations for screenshot storage
type MinIOClient struct {
	client     *minio.Client
	bucketName string
	log        *logrus.Logger
}

// ScreenshotInfo represents stored screenshot information
type ScreenshotInfo struct {
	URL          string
	ThumbnailURL string
	SizeBytes    int64
	Width        int
	Height       int
}

// NewMinIOClient creates a new MinIO client for screenshot storage
func NewMinIOClient(log *logrus.Logger) (*MinIOClient, error) {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	if endpoint == "" {
		// Try to get from resource env vars
		if port := os.Getenv("MINIO_PORT"); port != "" {
			host := os.Getenv("MINIO_HOST")
			if host == "" {
				host = "localhost"
			}
			endpoint = fmt.Sprintf("%s:%s", host, port)
		} else {
			return nil, fmt.Errorf("MINIO_ENDPOINT or MINIO_PORT environment variable is required")
		}
	}

	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	if accessKey == "" {
		return nil, fmt.Errorf("MINIO_ACCESS_KEY environment variable not set")
	}

	secretKey := os.Getenv("MINIO_SECRET_KEY")
	if secretKey == "" {
		return nil, fmt.Errorf("MINIO_SECRET_KEY environment variable not set")
	}

	bucketName := os.Getenv("MINIO_BUCKET_NAME")
	if bucketName == "" {
		return nil, fmt.Errorf("MINIO_BUCKET_NAME environment variable not set")
	}

	// Initialize MinIO client
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: strings.HasPrefix(endpoint, "https://"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	minioClient := &MinIOClient{
		client:     client,
		bucketName: bucketName,
		log:        log,
	}

	// Ensure bucket exists
	if err := minioClient.ensureBucket(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	return minioClient, nil
}

// ensureBucket ensures the screenshot bucket exists
func (m *MinIOClient) ensureBucket(ctx context.Context) error {
	exists, err := m.client.BucketExists(ctx, m.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check if bucket exists: %w", err)
	}

	if !exists {
		err = m.client.MakeBucket(ctx, m.bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		m.log.WithField("bucket", m.bucketName).Info("Created MinIO bucket for screenshots")
	}

	return nil
}

// StoreScreenshot stores a screenshot file in MinIO
func (m *MinIOClient) StoreScreenshot(ctx context.Context, executionID uuid.UUID, stepName string, data []byte, contentType string) (*ScreenshotInfo, error) {
	// Generate object name
	objectName := fmt.Sprintf("screenshots/%s/%s-%s.png", executionID, stepName, uuid.New())

	// Derive image dimensions before streaming to storage so replay UI can size thumbnails accurately.
	width, height := decodeDimensions(data)

	// Upload to MinIO
	reader := bytes.NewReader(data)
	_, err := m.client.PutObject(ctx, m.bucketName, objectName, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store screenshot: %w", err)
	}

	m.log.WithFields(logrus.Fields{
		"execution_id": executionID,
		"step_name":    stepName,
		"object_name":  objectName,
		"size_bytes":   len(data),
	}).Info("Screenshot stored in MinIO")

	// Generate URLs
	screenshotURL := fmt.Sprintf("/api/v1/screenshots/%s", objectName)
	thumbnailURL := fmt.Sprintf("/api/v1/screenshots/thumbnail/%s", objectName)

	return &ScreenshotInfo{
		URL:          screenshotURL,
		ThumbnailURL: thumbnailURL,
		SizeBytes:    int64(len(data)),
		Width:        width,  // Should be parsed from actual image
		Height:       height, // Should be parsed from actual image
	}, nil
}

func decodeDimensions(payload []byte) (int, int) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(payload))
	if err == nil && cfg.Width > 0 && cfg.Height > 0 {
		return cfg.Width, cfg.Height
	}
	width := 0
	height := 0
	if widthStr := os.Getenv("SCREENSHOT_DEFAULT_WIDTH"); widthStr != "" {
		if w, convErr := strconv.Atoi(widthStr); convErr == nil {
			width = w
		}
	}
	if heightStr := os.Getenv("SCREENSHOT_DEFAULT_HEIGHT"); heightStr != "" {
		if h, convErr := strconv.Atoi(heightStr); convErr == nil {
			height = h
		}
	}
	return width, height
}

// GetScreenshot retrieves a screenshot from MinIO
func (m *MinIOClient) GetScreenshot(ctx context.Context, objectName string) (io.ReadCloser, *minio.ObjectInfo, error) {
	object, err := m.client.GetObject(ctx, m.bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get screenshot: %w", err)
	}

	info, err := object.Stat()
	if err != nil {
		object.Close()
		return nil, nil, fmt.Errorf("failed to get screenshot info: %w", err)
	}

	return object, &info, nil
}

// DeleteScreenshot deletes a screenshot from MinIO
func (m *MinIOClient) DeleteScreenshot(ctx context.Context, objectName string) error {
	err := m.client.RemoveObject(ctx, m.bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete screenshot: %w", err)
	}

	m.log.WithField("object_name", objectName).Info("Screenshot deleted from MinIO")
	return nil
}

// ListExecutionScreenshots lists all screenshots for an execution
func (m *MinIOClient) ListExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]string, error) {
	prefix := fmt.Sprintf("screenshots/%s/", executionID)

	objectCh := m.client.ListObjects(ctx, m.bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	var objects []string
	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list screenshots: %w", object.Err)
		}
		objects = append(objects, object.Key)
	}

	return objects, nil
}

// StoreScreenshotFromFile stores a screenshot from a file path
func (m *MinIOClient) StoreScreenshotFromFile(ctx context.Context, executionID uuid.UUID, stepName string, filePath string) (*ScreenshotInfo, error) {
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read screenshot file: %w", err)
	}

	// Determine content type based on file extension
	contentType := "image/png"
	if ext := filepath.Ext(filePath); ext != "" {
		switch strings.ToLower(ext) {
		case ".jpg", ".jpeg":
			contentType = "image/jpeg"
		case ".png":
			contentType = "image/png"
		case ".gif":
			contentType = "image/gif"
		}
	}

	return m.StoreScreenshot(ctx, executionID, stepName, data, contentType)
}

// GetBucketName returns the configured bucket name
func (m *MinIOClient) GetBucketName() string {
	return m.bucketName
}

// HealthCheck checks if MinIO is accessible
func (m *MinIOClient) HealthCheck(ctx context.Context) error {
	_, err := m.client.BucketExists(ctx, m.bucketName)
	if err != nil {
		return fmt.Errorf("MinIO health check failed: %w", err)
	}
	return nil
}

// Compile-time interface enforcement
var _ StorageInterface = (*MinIOClient)(nil)
