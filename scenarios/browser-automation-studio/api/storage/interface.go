package storage

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

// StorageInterface defines the interface for screenshot storage operations
// This interface allows for easier testing by enabling mock implementations
type StorageInterface interface {
	// GetScreenshot retrieves a screenshot from storage
	GetScreenshot(ctx context.Context, objectName string) (io.ReadCloser, *minio.ObjectInfo, error)

	// StoreScreenshot stores a screenshot in storage
	StoreScreenshot(ctx context.Context, executionID uuid.UUID, stepName string, data []byte, contentType string) (*ScreenshotInfo, error)

	// StoreScreenshotFromFile stores a screenshot from a file path
	StoreScreenshotFromFile(ctx context.Context, executionID uuid.UUID, stepName string, filePath string) (*ScreenshotInfo, error)

	// DeleteScreenshot deletes a screenshot from storage
	DeleteScreenshot(ctx context.Context, objectName string) error

	// ListExecutionScreenshots lists all screenshots for an execution
	ListExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]string, error)

	// HealthCheck checks if storage is accessible
	HealthCheck(ctx context.Context) error

	// GetBucketName returns the configured bucket name
	GetBucketName() string
}
