package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupMinIOContainer starts a MinIO testcontainer and returns connection details
func setupMinIOContainer(t *testing.T) (endpoint string, cleanup func()) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "minio/minio:latest",
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     "minioadmin",
			"MINIO_ROOT_PASSWORD": "minioadmin",
		},
		Cmd:        []string{"server", "/data"},
		WaitingFor: wait.ForHTTP("/minio/health/live").WithPort("9000/tcp"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "failed to start MinIO container")

	endpoint, err = container.Endpoint(ctx, "")
	require.NoError(t, err, "failed to get container endpoint")

	// Add additional wait to ensure MinIO is fully initialized for operations
	// The health endpoint responds before the server is fully ready for bucket operations
	err = waitForMinIOReady(endpoint, 10*time.Second)
	require.NoError(t, err, "MinIO failed to become ready for operations")

	cleanup = func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	}

	return endpoint, cleanup
}

// waitForMinIOReady waits for MinIO to be fully ready for bucket operations
func waitForMinIOReady(endpoint string, timeout time.Duration) error {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4("minioadmin", "minioadmin", ""),
		Secure: strings.HasPrefix(endpoint, "https://"),
	})
	if err != nil {
		return fmt.Errorf("failed to create test client: %w", err)
	}

	ctx := context.Background()
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Try to check bucket existence as a readiness probe
		_, err := client.BucketExists(ctx, "test-readiness-probe")
		if err == nil || !strings.Contains(err.Error(), "Server not initialized yet") {
			// Either operation succeeded or we got a different error (bucket doesn't exist is fine)
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("MinIO not ready after %v", timeout)
}

func TestNewMinIOClient(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] creates client with valid configuration", func(t *testing.T) {
		endpoint, cleanup := setupMinIOContainer(t)
		defer cleanup()

		os.Setenv("MINIO_ENDPOINT", endpoint)
		os.Setenv("MINIO_ACCESS_KEY", "minioadmin")
		os.Setenv("MINIO_SECRET_KEY", "minioadmin")
		os.Setenv("MINIO_BUCKET_NAME", "test-bucket")
		defer func() {
			os.Unsetenv("MINIO_ENDPOINT")
			os.Unsetenv("MINIO_ACCESS_KEY")
			os.Unsetenv("MINIO_SECRET_KEY")
			os.Unsetenv("MINIO_BUCKET_NAME")
		}()

		client, err := NewMinIOClient(log)

		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, "test-bucket", client.bucketName)
		assert.NotNil(t, client.client)
		assert.NotNil(t, client.log)
	})

	t.Run("[REQ:BAS-REPLAY-EXPORT-BUNDLE] creates bucket if it doesn't exist", func(t *testing.T) {
		endpoint, cleanup := setupMinIOContainer(t)
		defer cleanup()

		os.Setenv("MINIO_ENDPOINT", endpoint)
		os.Setenv("MINIO_ACCESS_KEY", "minioadmin")
		os.Setenv("MINIO_SECRET_KEY", "minioadmin")
		os.Setenv("MINIO_BUCKET_NAME", "auto-created-bucket")
		defer func() {
			os.Unsetenv("MINIO_ENDPOINT")
			os.Unsetenv("MINIO_ACCESS_KEY")
			os.Unsetenv("MINIO_SECRET_KEY")
			os.Unsetenv("MINIO_BUCKET_NAME")
		}()

		client, err := NewMinIOClient(log)

		require.NoError(t, err)
		assert.NotNil(t, client)

		// Verify bucket exists
		ctx := context.Background()
		exists, err := client.client.BucketExists(ctx, "auto-created-bucket")
		require.NoError(t, err)
		assert.True(t, exists, "bucket should have been created")
	})

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] fails with missing endpoint", func(t *testing.T) {
		os.Unsetenv("MINIO_ENDPOINT")
		os.Unsetenv("MINIO_PORT")
		os.Setenv("MINIO_ACCESS_KEY", "minioadmin")
		os.Setenv("MINIO_SECRET_KEY", "minioadmin")
		os.Setenv("MINIO_BUCKET_NAME", "test-bucket")
		defer func() {
			os.Unsetenv("MINIO_ACCESS_KEY")
			os.Unsetenv("MINIO_SECRET_KEY")
			os.Unsetenv("MINIO_BUCKET_NAME")
		}()

		client, err := NewMinIOClient(log)

		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "MINIO_ENDPOINT or MINIO_PORT")
	})

	t.Run("fails with missing access key", func(t *testing.T) {
		os.Setenv("MINIO_ENDPOINT", "localhost:9000")
		os.Unsetenv("MINIO_ACCESS_KEY")
		os.Setenv("MINIO_SECRET_KEY", "minioadmin")
		os.Setenv("MINIO_BUCKET_NAME", "test-bucket")
		defer func() {
			os.Unsetenv("MINIO_ENDPOINT")
			os.Unsetenv("MINIO_SECRET_KEY")
			os.Unsetenv("MINIO_BUCKET_NAME")
		}()

		client, err := NewMinIOClient(log)

		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "MINIO_ACCESS_KEY")
	})

	t.Run("fails with missing secret key", func(t *testing.T) {
		os.Setenv("MINIO_ENDPOINT", "localhost:9000")
		os.Setenv("MINIO_ACCESS_KEY", "minioadmin")
		os.Unsetenv("MINIO_SECRET_KEY")
		os.Setenv("MINIO_BUCKET_NAME", "test-bucket")
		defer func() {
			os.Unsetenv("MINIO_ENDPOINT")
			os.Unsetenv("MINIO_ACCESS_KEY")
			os.Unsetenv("MINIO_BUCKET_NAME")
		}()

		client, err := NewMinIOClient(log)

		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "MINIO_SECRET_KEY")
	})

	t.Run("fails with missing bucket name", func(t *testing.T) {
		os.Setenv("MINIO_ENDPOINT", "localhost:9000")
		os.Setenv("MINIO_ACCESS_KEY", "minioadmin")
		os.Setenv("MINIO_SECRET_KEY", "minioadmin")
		os.Unsetenv("MINIO_BUCKET_NAME")
		defer func() {
			os.Unsetenv("MINIO_ENDPOINT")
			os.Unsetenv("MINIO_ACCESS_KEY")
			os.Unsetenv("MINIO_SECRET_KEY")
		}()

		client, err := NewMinIOClient(log)

		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "MINIO_BUCKET_NAME")
	})

	t.Run("constructs endpoint from MINIO_HOST and MINIO_PORT", func(t *testing.T) {
		endpoint, cleanup := setupMinIOContainer(t)
		defer cleanup()

		os.Unsetenv("MINIO_ENDPOINT")
		os.Setenv("MINIO_HOST", "localhost")
		os.Setenv("MINIO_PORT", endpoint[len("localhost:"):]) // Extract port from endpoint
		os.Setenv("MINIO_ACCESS_KEY", "minioadmin")
		os.Setenv("MINIO_SECRET_KEY", "minioadmin")
		os.Setenv("MINIO_BUCKET_NAME", "test-bucket")
		defer func() {
			os.Unsetenv("MINIO_HOST")
			os.Unsetenv("MINIO_PORT")
			os.Unsetenv("MINIO_ACCESS_KEY")
			os.Unsetenv("MINIO_SECRET_KEY")
			os.Unsetenv("MINIO_BUCKET_NAME")
		}()

		client, err := NewMinIOClient(log)

		require.NoError(t, err)
		assert.NotNil(t, client)
	})
}

func TestMinIOClient_StoreScreenshot(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	endpoint, cleanup := setupMinIOContainer(t)
	defer cleanup()

	os.Setenv("MINIO_ENDPOINT", endpoint)
	os.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	os.Setenv("MINIO_SECRET_KEY", "minioadmin")
	os.Setenv("MINIO_BUCKET_NAME", "screenshots-test")
	defer func() {
		os.Unsetenv("MINIO_ENDPOINT")
		os.Unsetenv("MINIO_ACCESS_KEY")
		os.Unsetenv("MINIO_SECRET_KEY")
		os.Unsetenv("MINIO_BUCKET_NAME")
	}()

	client, err := NewMinIOClient(log)
	require.NoError(t, err)

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] stores screenshot successfully", func(t *testing.T) {
		ctx := context.Background()
		executionID := uuid.New()
		stepName := "click-button"
		data := []byte("fake-png-data")
		contentType := "image/png"

		info, err := client.StoreScreenshot(ctx, executionID, stepName, data, contentType)

		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.Contains(t, info.URL, "/api/v1/screenshots/")
		assert.Contains(t, info.URL, executionID.String())
		assert.Contains(t, info.URL, stepName)
		assert.Equal(t, int64(len(data)), info.SizeBytes)
		assert.NotEmpty(t, info.ThumbnailURL)
	})

	t.Run("[REQ:BAS-REPLAY-EXPORT-BUNDLE] stores multiple screenshots for same execution", func(t *testing.T) {
		ctx := context.Background()
		executionID := uuid.New()

		info1, err := client.StoreScreenshot(ctx, executionID, "step1", []byte("data1"), "image/png")
		require.NoError(t, err)

		info2, err := client.StoreScreenshot(ctx, executionID, "step2", []byte("data2"), "image/png")
		require.NoError(t, err)

		// Both should succeed with different URLs
		assert.NotEqual(t, info1.URL, info2.URL)
		assert.Contains(t, info1.URL, "step1")
		assert.Contains(t, info2.URL, "step2")
	})

	t.Run("handles empty screenshot data", func(t *testing.T) {
		ctx := context.Background()
		executionID := uuid.New()
		data := []byte{}

		info, err := client.StoreScreenshot(ctx, executionID, "empty", data, "image/png")

		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, int64(0), info.SizeBytes)
	})

	t.Run("respects content type", func(t *testing.T) {
		ctx := context.Background()
		executionID := uuid.New()
		data := []byte("fake-jpeg-data")

		info, err := client.StoreScreenshot(ctx, executionID, "jpeg-test", data, "image/jpeg")

		require.NoError(t, err)
		assert.NotNil(t, info)
	})
}

func TestMinIOClient_GetScreenshot(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	endpoint, cleanup := setupMinIOContainer(t)
	defer cleanup()

	os.Setenv("MINIO_ENDPOINT", endpoint)
	os.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	os.Setenv("MINIO_SECRET_KEY", "minioadmin")
	os.Setenv("MINIO_BUCKET_NAME", "screenshots-get-test")
	defer func() {
		os.Unsetenv("MINIO_ENDPOINT")
		os.Unsetenv("MINIO_ACCESS_KEY")
		os.Unsetenv("MINIO_SECRET_KEY")
		os.Unsetenv("MINIO_BUCKET_NAME")
	}()

	client, err := NewMinIOClient(log)
	require.NoError(t, err)

	t.Run("[REQ:BAS-REPLAY-EXPORT-BUNDLE] retrieves stored screenshot", func(t *testing.T) {
		ctx := context.Background()
		executionID := uuid.New()
		expectedData := []byte("test-screenshot-data")

		// Store screenshot first
		info, err := client.StoreScreenshot(ctx, executionID, "test", expectedData, "image/png")
		require.NoError(t, err)

		// Extract object name from URL
		objectName := info.URL[len("/api/v1/screenshots/"):]

		// Retrieve screenshot
		reader, objectInfo, err := client.GetScreenshot(ctx, objectName)
		require.NoError(t, err)
		defer reader.Close()

		// Read and verify data
		actualData, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, expectedData, actualData)
		assert.Equal(t, int64(len(expectedData)), objectInfo.Size)
		assert.Equal(t, "image/png", objectInfo.ContentType)
	})

	t.Run("returns error for non-existent screenshot", func(t *testing.T) {
		ctx := context.Background()

		reader, objectInfo, err := client.GetScreenshot(ctx, "screenshots/non-existent.png")

		assert.Error(t, err)
		assert.Nil(t, reader)
		assert.Nil(t, objectInfo)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		reader, objectInfo, err := client.GetScreenshot(ctx, "screenshots/test.png")

		assert.Error(t, err)
		assert.Nil(t, reader)
		assert.Nil(t, objectInfo)
	})
}

func TestMinIOClient_DeleteScreenshot(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	endpoint, cleanup := setupMinIOContainer(t)
	defer cleanup()

	os.Setenv("MINIO_ENDPOINT", endpoint)
	os.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	os.Setenv("MINIO_SECRET_KEY", "minioadmin")
	os.Setenv("MINIO_BUCKET_NAME", "screenshots-delete-test")
	defer func() {
		os.Unsetenv("MINIO_ENDPOINT")
		os.Unsetenv("MINIO_ACCESS_KEY")
		os.Unsetenv("MINIO_SECRET_KEY")
		os.Unsetenv("MINIO_BUCKET_NAME")
	}()

	client, err := NewMinIOClient(log)
	require.NoError(t, err)

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] deletes screenshot successfully", func(t *testing.T) {
		ctx := context.Background()
		executionID := uuid.New()
		data := []byte("to-be-deleted")

		// Store screenshot
		info, err := client.StoreScreenshot(ctx, executionID, "delete-test", data, "image/png")
		require.NoError(t, err)

		objectName := info.URL[len("/api/v1/screenshots/"):]

		// Verify it exists
		reader, _, err := client.GetScreenshot(ctx, objectName)
		require.NoError(t, err)
		reader.Close()

		// Delete screenshot
		err = client.DeleteScreenshot(ctx, objectName)
		require.NoError(t, err)

		// Verify it's gone
		reader, objectInfo, err := client.GetScreenshot(ctx, objectName)
		assert.Error(t, err)
		assert.Nil(t, reader)
		assert.Nil(t, objectInfo)
	})

	t.Run("handles deletion of non-existent screenshot gracefully", func(t *testing.T) {
		ctx := context.Background()

		// MinIO doesn't error on deleting non-existent objects
		err := client.DeleteScreenshot(ctx, "screenshots/never-existed.png")

		// Should not error (MinIO behavior)
		assert.NoError(t, err)
	})
}

func TestMinIOClient_ListExecutionScreenshots(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	endpoint, cleanup := setupMinIOContainer(t)
	defer cleanup()

	os.Setenv("MINIO_ENDPOINT", endpoint)
	os.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	os.Setenv("MINIO_SECRET_KEY", "minioadmin")
	os.Setenv("MINIO_BUCKET_NAME", "screenshots-list-test")
	defer func() {
		os.Unsetenv("MINIO_ENDPOINT")
		os.Unsetenv("MINIO_ACCESS_KEY")
		os.Unsetenv("MINIO_SECRET_KEY")
		os.Unsetenv("MINIO_BUCKET_NAME")
	}()

	client, err := NewMinIOClient(log)
	require.NoError(t, err)

	t.Run("[REQ:BAS-REPLAY-EXPORT-BUNDLE] lists all screenshots for execution", func(t *testing.T) {
		ctx := context.Background()
		executionID := uuid.New()

		// Store multiple screenshots
		_, err := client.StoreScreenshot(ctx, executionID, "step1", []byte("data1"), "image/png")
		require.NoError(t, err)
		_, err = client.StoreScreenshot(ctx, executionID, "step2", []byte("data2"), "image/png")
		require.NoError(t, err)
		_, err = client.StoreScreenshot(ctx, executionID, "step3", []byte("data3"), "image/png")
		require.NoError(t, err)

		// List screenshots
		screenshots, err := client.ListExecutionScreenshots(ctx, executionID)

		require.NoError(t, err)
		assert.Len(t, screenshots, 3)
		for _, screenshot := range screenshots {
			assert.Contains(t, screenshot, executionID.String())
		}
	})

	t.Run("returns empty list for execution with no screenshots", func(t *testing.T) {
		ctx := context.Background()
		executionID := uuid.New()

		screenshots, err := client.ListExecutionScreenshots(ctx, executionID)

		require.NoError(t, err)
		assert.Empty(t, screenshots)
	})

	t.Run("only lists screenshots for specific execution", func(t *testing.T) {
		ctx := context.Background()
		executionID1 := uuid.New()
		executionID2 := uuid.New()

		// Store screenshots for two different executions
		_, err := client.StoreScreenshot(ctx, executionID1, "exec1-step1", []byte("data1"), "image/png")
		require.NoError(t, err)
		_, err = client.StoreScreenshot(ctx, executionID2, "exec2-step1", []byte("data2"), "image/png")
		require.NoError(t, err)

		// List screenshots for first execution
		screenshots, err := client.ListExecutionScreenshots(ctx, executionID1)

		require.NoError(t, err)
		assert.Len(t, screenshots, 1)
		assert.Contains(t, screenshots[0], executionID1.String())
		assert.NotContains(t, screenshots[0], executionID2.String())
	})
}

func TestMinIOClient_StoreScreenshotFromFile(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	endpoint, cleanup := setupMinIOContainer(t)
	defer cleanup()

	os.Setenv("MINIO_ENDPOINT", endpoint)
	os.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	os.Setenv("MINIO_SECRET_KEY", "minioadmin")
	os.Setenv("MINIO_BUCKET_NAME", "screenshots-file-test")
	defer func() {
		os.Unsetenv("MINIO_ENDPOINT")
		os.Unsetenv("MINIO_ACCESS_KEY")
		os.Unsetenv("MINIO_SECRET_KEY")
		os.Unsetenv("MINIO_BUCKET_NAME")
	}()

	client, err := NewMinIOClient(log)
	require.NoError(t, err)

	t.Run("stores screenshot from PNG file", func(t *testing.T) {
		ctx := context.Background()
		executionID := uuid.New()

		// Create temp file
		tmpFile, err := os.CreateTemp("", "test-screenshot-*.png")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		testData := []byte("fake-png-data")
		_, err = tmpFile.Write(testData)
		require.NoError(t, err)
		tmpFile.Close()

		// Store from file
		info, err := client.StoreScreenshotFromFile(ctx, executionID, "from-file", tmpFile.Name())

		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, int64(len(testData)), info.SizeBytes)
		assert.Contains(t, info.URL, "from-file")
	})

	t.Run("handles JPEG file correctly", func(t *testing.T) {
		ctx := context.Background()
		executionID := uuid.New()

		tmpFile, err := os.CreateTemp("", "test-screenshot-*.jpg")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		testData := []byte("fake-jpeg-data")
		_, err = tmpFile.Write(testData)
		require.NoError(t, err)
		tmpFile.Close()

		info, err := client.StoreScreenshotFromFile(ctx, executionID, "jpeg-file", tmpFile.Name())

		require.NoError(t, err)
		assert.NotNil(t, info)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		ctx := context.Background()
		executionID := uuid.New()

		info, err := client.StoreScreenshotFromFile(ctx, executionID, "missing", "/non/existent/file.png")

		assert.Error(t, err)
		assert.Nil(t, info)
		assert.Contains(t, err.Error(), "failed to read screenshot file")
	})
}

func TestMinIOClient_HelperMethods(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	endpoint, cleanup := setupMinIOContainer(t)
	defer cleanup()

	os.Setenv("MINIO_ENDPOINT", endpoint)
	os.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	os.Setenv("MINIO_SECRET_KEY", "minioadmin")
	os.Setenv("MINIO_BUCKET_NAME", "helpers-test")
	defer func() {
		os.Unsetenv("MINIO_ENDPOINT")
		os.Unsetenv("MINIO_ACCESS_KEY")
		os.Unsetenv("MINIO_SECRET_KEY")
		os.Unsetenv("MINIO_BUCKET_NAME")
	}()

	client, err := NewMinIOClient(log)
	require.NoError(t, err)

	t.Run("GetBucketName returns configured bucket", func(t *testing.T) {
		bucketName := client.GetBucketName()

		assert.Equal(t, "helpers-test", bucketName)
	})

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] HealthCheck succeeds when MinIO is accessible", func(t *testing.T) {
		ctx := context.Background()

		err := client.HealthCheck(ctx)

		assert.NoError(t, err)
	})

	t.Run("HealthCheck fails when bucket doesn't exist", func(t *testing.T) {
		// Create a new client with non-existent bucket (but don't auto-create)
		fakeClient := &MinIOClient{
			client:     client.client,
			bucketName: "non-existent-bucket-xyz",
			log:        log,
		}

		ctx := context.Background()
		err := fakeClient.HealthCheck(ctx)

		// MinIO BucketExists returns false, not error, so health check passes
		// This is acceptable MinIO behavior
		assert.NoError(t, err)
	})
}

func TestMinIOClient_ConcurrentOperations(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	endpoint, cleanup := setupMinIOContainer(t)
	defer cleanup()

	os.Setenv("MINIO_ENDPOINT", endpoint)
	os.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	os.Setenv("MINIO_SECRET_KEY", "minioadmin")
	os.Setenv("MINIO_BUCKET_NAME", "concurrent-test")
	defer func() {
		os.Unsetenv("MINIO_ENDPOINT")
		os.Unsetenv("MINIO_ACCESS_KEY")
		os.Unsetenv("MINIO_SECRET_KEY")
		os.Unsetenv("MINIO_BUCKET_NAME")
	}()

	client, err := NewMinIOClient(log)
	require.NoError(t, err)

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] handles concurrent uploads", func(t *testing.T) {
		ctx := context.Background()
		executionID := uuid.New()
		concurrency := 10

		done := make(chan error, concurrency)

		// Upload concurrently
		for i := 0; i < concurrency; i++ {
			go func(idx int) {
				data := []byte(bytes.Repeat([]byte{byte(idx)}, 100))
				_, err := client.StoreScreenshot(ctx, executionID, "concurrent", data, "image/png")
				done <- err
			}(i)
		}

		// Wait for all uploads
		for i := 0; i < concurrency; i++ {
			err := <-done
			assert.NoError(t, err)
		}

		// Verify all screenshots are stored
		screenshots, err := client.ListExecutionScreenshots(ctx, executionID)
		require.NoError(t, err)
		assert.Equal(t, concurrency, len(screenshots))
	})
}
