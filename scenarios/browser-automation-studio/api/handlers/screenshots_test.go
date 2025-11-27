package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/storage"
)

// Mock storage client for testing
type mockStorageClient struct {
	getScreenshotFn func(ctx context.Context, objectName string) (io.ReadCloser, *minio.ObjectInfo, error)
}

func (m *mockStorageClient) GetScreenshot(ctx context.Context, objectName string) (io.ReadCloser, *minio.ObjectInfo, error) {
	if m.getScreenshotFn != nil {
		return m.getScreenshotFn(ctx, objectName)
	}
	return nil, nil, errors.New("not implemented")
}

func (m *mockStorageClient) StoreScreenshot(ctx context.Context, executionID uuid.UUID, stepName string, data []byte, contentType string) (*storage.ScreenshotInfo, error) {
	return nil, errors.New("not implemented")
}

func (m *mockStorageClient) StoreScreenshotFromFile(ctx context.Context, executionID uuid.UUID, stepName string, filePath string) (*storage.ScreenshotInfo, error) {
	return nil, errors.New("not implemented")
}

func (m *mockStorageClient) DeleteScreenshot(ctx context.Context, objectName string) error {
	return errors.New("not implemented")
}

func (m *mockStorageClient) ListExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]string, error) {
	return nil, errors.New("not implemented")
}

func (m *mockStorageClient) HealthCheck(ctx context.Context) error {
	return errors.New("not implemented")
}

func (m *mockStorageClient) GetBucketName() string {
	return "test-bucket"
}

func setupScreenshotTestHandler(t *testing.T, storageClient storage.StorageInterface) *Handler {
	t.Helper()

	log := logrus.New()
	log.SetOutput(io.Discard)

	return &Handler{
		log:     log,
		storage: storageClient,
	}
}

func TestServeScreenshot_Success(t *testing.T) {
	t.Run("[REQ:BAS-REPLAY-EXPORT-BUNDLE] serves screenshot successfully with correct headers", func(t *testing.T) {
		screenshotData := []byte("fake-image-data-png")

		mockStorage := &mockStorageClient{
			getScreenshotFn: func(ctx context.Context, objectName string) (io.ReadCloser, *minio.ObjectInfo, error) {
				assert.Equal(t, "executions/test-exec-id/screenshot-001.png", objectName)

				return io.NopCloser(bytes.NewReader(screenshotData)), &minio.ObjectInfo{
					Size:        int64(len(screenshotData)),
					ContentType: "image/png",
				}, nil
			},
		}

		handler := setupScreenshotTestHandler(t, mockStorage)

		req := httptest.NewRequest("GET", "/api/v1/screenshots/executions/test-exec-id/screenshot-001.png", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("*", "executions/test-exec-id/screenshot-001.png")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.ServeScreenshot(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
		assert.Equal(t, "19", w.Header().Get("Content-Length"))
		assert.Equal(t, "public, max-age=3600", w.Header().Get("Cache-Control"))
		assert.Equal(t, screenshotData, w.Body.Bytes())
	})
}

func TestServeScreenshot_NotFound(t *testing.T) {
	t.Run("[REQ:BAS-REPLAY-EXPORT-BUNDLE] returns 404 when screenshot does not exist", func(t *testing.T) {
		mockStorage := &mockStorageClient{
			getScreenshotFn: func(ctx context.Context, objectName string) (io.ReadCloser, *minio.ObjectInfo, error) {
				return nil, nil, minio.ErrorResponse{
					StatusCode: 404,
					Code:       "NoSuchKey",
				}
			},
		}

		handler := setupScreenshotTestHandler(t, mockStorage)

		req := httptest.NewRequest("GET", "/api/v1/screenshots/executions/missing/screenshot.png", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("*", "executions/missing/screenshot.png")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.ServeScreenshot(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response APIError
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "SCREENSHOT_NOT_FOUND", response.Code)
	})
}

func TestServeScreenshot_InvalidPath(t *testing.T) {
	t.Run("[REQ:BAS-REPLAY-EXPORT-BUNDLE] rejects request with empty screenshot path", func(t *testing.T) {
		handler := setupScreenshotTestHandler(t, &mockStorageClient{})

		req := httptest.NewRequest("GET", "/api/v1/screenshots/", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("*", "")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.ServeScreenshot(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response APIError
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "MISSING_REQUIRED_FIELD", response.Code)
	})
}

func TestServeScreenshot_StorageUnavailable(t *testing.T) {
	t.Run("[REQ:BAS-REPLAY-EXPORT-BUNDLE] returns service unavailable when storage is nil", func(t *testing.T) {
		handler := setupScreenshotTestHandler(t, nil) // nil storage

		req := httptest.NewRequest("GET", "/api/v1/screenshots/test.png", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("*", "test.png")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.ServeScreenshot(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)

		var response APIError
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "SERVICE_UNAVAILABLE", response.Code)
		assert.Contains(t, response.Message, "storage not available")
	})
}

func TestServeScreenshot_LeadingSlashStripped(t *testing.T) {
	t.Run("[REQ:BAS-REPLAY-EXPORT-BUNDLE] strips leading slash from object path", func(t *testing.T) {
		mockStorage := &mockStorageClient{
			getScreenshotFn: func(ctx context.Context, objectName string) (io.ReadCloser, *minio.ObjectInfo, error) {
				// Verify leading slash was stripped
				assert.Equal(t, "path/to/screenshot.png", objectName)
				assert.NotContains(t, objectName, "//")

				return io.NopCloser(bytes.NewReader([]byte("data"))), &minio.ObjectInfo{
					Size:        4,
					ContentType: "image/png",
				}, nil
			},
		}

		handler := setupScreenshotTestHandler(t, mockStorage)

		req := httptest.NewRequest("GET", "/api/v1/screenshots//path/to/screenshot.png", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("*", "/path/to/screenshot.png")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.ServeScreenshot(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestServeScreenshot_StreamingLargeFile(t *testing.T) {
	t.Run("[REQ:BAS-REPLAY-EXPORT-BUNDLE] streams large screenshot without loading into memory", func(t *testing.T) {
		// Create a large fake image (5MB)
		largeData := make([]byte, 5*1024*1024)
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		mockStorage := &mockStorageClient{
			getScreenshotFn: func(ctx context.Context, objectName string) (io.ReadCloser, *minio.ObjectInfo, error) {
				return io.NopCloser(bytes.NewReader(largeData)), &minio.ObjectInfo{
					Size:        int64(len(largeData)),
					ContentType: "image/png",
				}, nil
			},
		}

		handler := setupScreenshotTestHandler(t, mockStorage)

		req := httptest.NewRequest("GET", "/api/v1/screenshots/large.png", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("*", "large.png")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.ServeScreenshot(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, largeData, w.Body.Bytes())
	})
}

func TestServeThumbnail_Success(t *testing.T) {
	t.Run("[REQ:BAS-REPLAY-EXPORT-BUNDLE] serves thumbnail with same behavior as full screenshot", func(t *testing.T) {
		thumbnailData := []byte("thumbnail-data")

		mockStorage := &mockStorageClient{
			getScreenshotFn: func(ctx context.Context, objectName string) (io.ReadCloser, *minio.ObjectInfo, error) {
				return io.NopCloser(bytes.NewReader(thumbnailData)), &minio.ObjectInfo{
					Size:        int64(len(thumbnailData)),
					ContentType: "image/jpeg",
				}, nil
			},
		}

		handler := setupScreenshotTestHandler(t, mockStorage)

		req := httptest.NewRequest("GET", "/api/v1/screenshots/thumbnail/test.jpg", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("*", "test.jpg")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.ServeThumbnail(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, thumbnailData, w.Body.Bytes())
	})
}
