package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
)

// ============================================================================
// ServeArtifact Tests
// ============================================================================

func TestServeArtifact_Success(t *testing.T) {
	handler, _, _, _, _, storageMock := createTestHandler()

	// Store a test artifact
	objectName := "test-execution/artifacts/test.json"
	artifactData := []byte(`{"test": "data"}`)
	storageMock.artifacts[objectName] = artifactData

	req := httptest.NewRequest(http.MethodGet, "/api/v1/artifacts/"+objectName, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("*", objectName)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.ServeArtifact(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := rr.Body.String()
	if body != string(artifactData) {
		t.Fatalf("expected body %q, got %q", string(artifactData), body)
	}
}

func TestServeArtifact_EmptyPath(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/artifacts/", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("*", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.ServeArtifact(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for empty path, got %d", rr.Code)
	}
}

func TestServeArtifact_NotFound(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	objectName := "nonexistent/artifact.json"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/artifacts/"+objectName, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("*", objectName)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.ServeArtifact(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestServeArtifact_StorageError(t *testing.T) {
	handler, _, _, _, _, storageMock := createTestHandler()

	storageMock.GetArtifactError = errors.New("storage failure")

	objectName := "test/artifact.json"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/artifacts/"+objectName, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("*", objectName)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.ServeArtifact(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for storage error, got %d", rr.Code)
	}
}

func TestServeArtifact_StorageNil(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	handler := &Handler{
		storage: nil,
		log:     log,
	}

	objectName := "test/artifact.json"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/artifacts/"+objectName, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("*", objectName)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.ServeArtifact(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503 when storage is nil, got %d", rr.Code)
	}
}

func TestServeArtifact_LeadingSlashTrimmed(t *testing.T) {
	handler, _, _, _, _, storageMock := createTestHandler()

	// Store artifact without leading slash
	objectName := "test-execution/artifacts/test.json"
	artifactData := []byte(`{"test": "trimmed"}`)
	storageMock.artifacts[objectName] = artifactData

	// Request with leading slash
	req := httptest.NewRequest(http.MethodGet, "/api/v1/artifacts//"+objectName, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("*", "/"+objectName)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.ServeArtifact(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestServeArtifact_ContentTypeHeader(t *testing.T) {
	handler, _, _, _, _, storageMock := createTestHandler()

	// Override GetArtifact to return specific content type
	objectName := "test-execution/video.mp4"
	storageMock.artifacts[objectName] = []byte("fake video data")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/artifacts/"+objectName, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("*", objectName)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.ServeArtifact(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	// Verify Cache-Control header is set
	cacheControl := rr.Header().Get("Cache-Control")
	if !strings.Contains(cacheControl, "max-age") {
		t.Errorf("expected Cache-Control header with max-age, got %q", cacheControl)
	}
}

// ============================================================================
// Mock Storage with ContentType support
// ============================================================================

// Extend MockStorage to properly return content type info
type MockStorageWithInfo struct {
	*MockStorage
	contentTypes map[string]string
}

func NewMockStorageWithInfo() *MockStorageWithInfo {
	return &MockStorageWithInfo{
		MockStorage:  NewMockStorage(),
		contentTypes: make(map[string]string),
	}
}

func (m *MockStorageWithInfo) GetArtifact(ctx context.Context, objectName string) (io.ReadCloser, *minio.ObjectInfo, error) {
	if m.GetArtifactError != nil {
		return nil, nil, m.GetArtifactError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	data, ok := m.artifacts[objectName]
	if !ok {
		return nil, nil, errors.New("not found")
	}

	contentType := m.contentTypes[objectName]
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return io.NopCloser(strings.NewReader(string(data))), &minio.ObjectInfo{
		Key:         objectName,
		Size:        int64(len(data)),
		ContentType: contentType,
	}, nil
}
