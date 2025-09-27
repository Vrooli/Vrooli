package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOStorage handles file storage operations with MinIO
type MinIOStorage struct {
	client     *minio.Client
	bucketName string
}

// NewMinIOStorage creates a new MinIO storage client
func NewMinIOStorage(endpoint, accessKey, secretKey, bucketName string, useSSL bool) (*MinIOStorage, error) {
	// Initialize minio client
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	// Check if bucket exists, create if not
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return &MinIOStorage{
		client:     client,
		bucketName: bucketName,
	}, nil
}

// UploadFile uploads a file to MinIO
func (s *MinIOStorage) UploadFile(ctx context.Context, localPath, objectName string) (string, error) {
	// Open the file
	file, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	// Upload the file
	_, err = s.client.PutObject(ctx, s.bucketName, objectName, file, fileInfo.Size(), minio.PutObjectOptions{
		ContentType: getContentType(localPath),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	// Return the URL
	return fmt.Sprintf("/%s/%s", s.bucketName, objectName), nil
}

// DownloadFile downloads a file from MinIO
func (s *MinIOStorage) DownloadFile(ctx context.Context, objectName, localPath string) error {
	// Get object from MinIO
	object, err := s.client.GetObject(ctx, s.bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to get object: %w", err)
	}
	defer object.Close()

	// Create local file
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer file.Close()

	// Copy content
	_, err = io.Copy(file, object)
	if err != nil {
		return fmt.Errorf("failed to copy content: %w", err)
	}

	return nil
}

// DeleteFile deletes a file from MinIO
func (s *MinIOStorage) DeleteFile(ctx context.Context, objectName string) error {
	err := s.client.RemoveObject(ctx, s.bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

// GeneratePresignedURL generates a presigned URL for direct access
func (s *MinIOStorage) GeneratePresignedURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	url, err := s.client.PresignedGetObject(ctx, s.bucketName, objectName, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return url.String(), nil
}

// ListFiles lists files in the bucket with a prefix
func (s *MinIOStorage) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	var files []string

	objectCh := s.client.ListObjects(ctx, s.bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", object.Err)
		}
		files = append(files, object.Key)
	}

	return files, nil
}

// getContentType returns the MIME type based on file extension
func getContentType(filename string) string {
	ext := filepath.Ext(filename)
	switch ext {
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".flac":
		return "audio/flac"
	case ".ogg":
		return "audio/ogg"
	case ".aac":
		return "audio/aac"
	case ".m4a":
		return "audio/mp4"
	default:
		return "application/octet-stream"
	}
}