package captures

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/api-core/storage"
)

func newTestService(t *testing.T) (*Service, string) {
	t.Helper()
	tmpDir := t.TempDir()
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileDesktop,
		// Override home dir to use temp
		UserHomeDir:   func() (string, error) { return tmpDir, nil },
		UserConfigDir: func() (string, error) { return filepath.Join(tmpDir, ".config"), nil },
		UserCacheDir:  func() (string, error) { return filepath.Join(tmpDir, ".cache"), nil },
	})
	require.NoError(t, err)

	opts := storage.Options{ScenarioID: "scenario-to-desktop-captures"}
	metaPath, err := resolver.Path(opts, storage.ClassData, "captures_meta.json")
	require.NoError(t, err)

	store, err := NewFileStore(metaPath)
	require.NoError(t, err)

	svc := NewService(resolver, opts, store)
	return svc, tmpDir
}

func newTestHandler(t *testing.T) (*Handler, *Service) {
	t.Helper()
	svc, _ := newTestService(t)
	return NewHandler(svc), svc
}

func newTestRouter(h *Handler) *mux.Router {
	r := mux.NewRouter()
	h.RegisterRoutes(r)
	return r
}

func seedCapture(t *testing.T, svc *Service, scenario, sessionID string) *Capture {
	t.Helper()
	// Create a temp source file
	tmpFile := filepath.Join(t.TempDir(), "test-screenshot.png")
	require.NoError(t, os.WriteFile(tmpFile, []byte("PNG fake data"), 0o644))

	cap, err := svc.SaveCapture(scenario, CaptureScreenshot, sessionID, tmpFile, 1280, 720, 0)
	require.NoError(t, err)
	return cap
}

func TestListCaptures_Empty(t *testing.T) {
	h, _ := newTestHandler(t)
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/captures/my-app", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var caps []Capture
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&caps))
	assert.Empty(t, caps)
}

func TestListCaptures_WithData(t *testing.T) {
	h, svc := newTestHandler(t)
	r := newTestRouter(h)

	seedCapture(t, svc, "my-app", "session-1")
	seedCapture(t, svc, "my-app", "session-1")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/captures/my-app", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var caps []Capture
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&caps))
	assert.Len(t, caps, 2)
}

func TestSummary_ReturnsCountAndSize(t *testing.T) {
	h, svc := newTestHandler(t)
	r := newTestRouter(h)

	seedCapture(t, svc, "my-app", "session-1")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/captures/my-app/summary", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var summary CapturesSummary
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&summary))
	assert.Equal(t, 1, summary.Count)
	assert.Greater(t, summary.TotalBytes, int64(0))
}

func TestDeleteCapture_NotFound(t *testing.T) {
	h, _ := newTestHandler(t)
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/captures/my-app/nonexistent", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestDeleteCapture_Success(t *testing.T) {
	h, svc := newTestHandler(t)
	r := newTestRouter(h)

	cap := seedCapture(t, svc, "my-app", "session-1")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/captures/my-app/"+cap.ID, nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify deleted
	caps, err := svc.Store().List("my-app")
	require.NoError(t, err)
	assert.Empty(t, caps)
}

func TestDeleteAll_Success(t *testing.T) {
	h, svc := newTestHandler(t)
	r := newTestRouter(h)

	seedCapture(t, svc, "my-app", "session-1")
	seedCapture(t, svc, "my-app", "session-1")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/captures/my-app", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	caps, err := svc.Store().List("my-app")
	require.NoError(t, err)
	assert.Empty(t, caps)
}

func TestServeFile_NotFound(t *testing.T) {
	h, _ := newTestHandler(t)
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/captures/my-app/nonexistent/file", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestServeFile_Success(t *testing.T) {
	h, svc := newTestHandler(t)
	r := newTestRouter(h)

	cap := seedCapture(t, svc, "my-app", "session-1")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/captures/my-app/"+cap.ID+"/file", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "PNG fake data")
}

func TestDownload_NoIds(t *testing.T) {
	h, _ := newTestHandler(t)
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/captures/my-app/download", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDownload_SingleFile(t *testing.T) {
	h, svc := newTestHandler(t)
	r := newTestRouter(h)

	cap := seedCapture(t, svc, "my-app", "session-1")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/captures/my-app/download?ids="+cap.ID, nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestDownload_MultipleFiles(t *testing.T) {
	h, svc := newTestHandler(t)
	r := newTestRouter(h)

	// Need unique timestamps to avoid filename collision
	cap1 := seedCapture(t, svc, "my-app", "session-1")
	time.Sleep(2 * time.Millisecond) // ensure different timestamp
	cap2 := seedCapture(t, svc, "my-app", "session-1")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/captures/my-app/download?ids="+cap1.ID+","+cap2.ID, nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/zip", rr.Header().Get("Content-Type"))
}
