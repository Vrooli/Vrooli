package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMinIOStorage(t *testing.T) {
	t.Run("InvalidEndpoint", func(t *testing.T) {
		_, err := NewMinIOStorage("invalid:endpoint", "key", "secret", "bucket", false)
		assert.Error(t, err)
	})

	t.Run("EmptyEndpoint", func(t *testing.T) {
		_, err := NewMinIOStorage("", "key", "secret", "bucket", false)
		assert.Error(t, err)
	})
}

func TestMinIOStorage_UploadFile(t *testing.T) {
	// Skip if MinIO not available
	if os.Getenv("MINIO_ENDPOINT") == "" {
		t.Skip("MinIO not configured, skipping integration test")
	}

	storage, err := NewMinIOStorage(
		os.Getenv("MINIO_ENDPOINT"),
		os.Getenv("MINIO_ACCESS_KEY"),
		os.Getenv("MINIO_SECRET_KEY"),
		"test-bucket",
		false,
	)
	if err != nil {
		t.Skipf("Could not connect to MinIO: %v", err)
	}

	t.Run("UploadNonExistentFile", func(t *testing.T) {
		ctx := context.Background()
		_, err := storage.UploadFile(ctx, "/nonexistent/file.wav", "test.wav")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open file")
	})

	t.Run("UploadValidFile", func(t *testing.T) {
		// Create temp file
		tempFile := filepath.Join(os.TempDir(), "test-upload.wav")
		err := os.WriteFile(tempFile, []byte("test audio data"), 0644)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tempFile)

		ctx := context.Background()
		url, err := storage.UploadFile(ctx, tempFile, "test-upload.wav")

		if err == nil {
			assert.NotEmpty(t, url)
			assert.Contains(t, url, "test-upload.wav")

			// Cleanup
			storage.DeleteFile(ctx, "test-upload.wav")
		}
	})
}

func TestMinIOStorage_DownloadFile(t *testing.T) {
	if os.Getenv("MINIO_ENDPOINT") == "" {
		t.Skip("MinIO not configured, skipping integration test")
	}

	t.Run("DownloadNonExistentFile", func(t *testing.T) {
		storage := &MinIOStorage{
			client:     nil, // Will cause error
			bucketName: "test",
		}

		ctx := context.Background()
		err := storage.DownloadFile(ctx, "nonexistent.wav", "/tmp/output.wav")
		assert.Error(t, err)
	})
}

func TestMinIOStorage_DeleteFile(t *testing.T) {
	if os.Getenv("MINIO_ENDPOINT") == "" {
		t.Skip("MinIO not configured, skipping integration test")
	}

	t.Run("DeleteNonExistentFile", func(t *testing.T) {
		storage := &MinIOStorage{
			client:     nil,
			bucketName: "test",
		}

		ctx := context.Background()
		err := storage.DeleteFile(ctx, "nonexistent.wav")
		assert.Error(t, err)
	})
}

func TestMinIOStorage_GeneratePresignedURL(t *testing.T) {
	if os.Getenv("MINIO_ENDPOINT") == "" {
		t.Skip("MinIO not configured, skipping integration test")
	}

	t.Run("GenerateURLWithInvalidClient", func(t *testing.T) {
		storage := &MinIOStorage{
			client:     nil,
			bucketName: "test",
		}

		ctx := context.Background()
		_, err := storage.GeneratePresignedURL(ctx, "test.wav", 1*time.Hour)
		assert.Error(t, err)
	})
}

func TestMinIOStorage_ListFiles(t *testing.T) {
	if os.Getenv("MINIO_ENDPOINT") == "" {
		t.Skip("MinIO not configured, skipping integration test")
	}

	t.Run("ListWithInvalidClient", func(t *testing.T) {
		storage := &MinIOStorage{
			client:     nil,
			bucketName: "test",
		}

		ctx := context.Background()
		_, err := storage.ListFiles(ctx, "prefix/")
		assert.Error(t, err)
	})
}

func TestGetContentType(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"test.mp3", "audio/mpeg"},
		{"test.wav", "audio/wav"},
		{"test.flac", "audio/flac"},
		{"test.ogg", "audio/ogg"},
		{"test.aac", "audio/aac"},
		{"test.m4a", "audio/mp4"},
		{"test.unknown", "application/octet-stream"},
		{"noextension", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := getContentType(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}
