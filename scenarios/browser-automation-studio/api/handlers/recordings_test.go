package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/ai"
	"github.com/vrooli/browser-automation-studio/services/export"
	"github.com/vrooli/browser-automation-studio/services/logutil"
	"github.com/vrooli/browser-automation-studio/services/recording"
	"github.com/vrooli/browser-automation-studio/services/replay"
	"github.com/vrooli/browser-automation-studio/services/workflow"
)

// Mock recording service for testing
type mockRecordingService struct {
	importArchiveFn func(ctx context.Context, archivePath string, opts services.RecordingImportOptions) (*services.RecordingImportResult, error)
}

func (m *mockRecordingService) ImportArchive(ctx context.Context, archivePath string, opts services.RecordingImportOptions) (*services.RecordingImportResult, error) {
	if m.importArchiveFn != nil {
		return m.importArchiveFn(ctx, archivePath, opts)
	}
	return nil, errors.New("not implemented")
}

func setupRecordingTestHandler(t *testing.T, recordingService recording.RecordingServiceInterface) *Handler {
	t.Helper()

	log := logrus.New()
	log.SetOutput(io.Discard)

	return &Handler{
		log:              log,
		recordingService: recordingService,
	}
}

func TestImportRecording_Success(t *testing.T) {
	t.Run("[REQ:BAS-REPLAY-TIMELINE-PERSISTENCE] imports valid recording archive successfully", func(t *testing.T) {
		executionID := uuid.New()
		workflowID := uuid.New()
		projectID := uuid.New()

		mockService := &mockRecordingService{
			importArchiveFn: func(ctx context.Context, archivePath string, opts services.RecordingImportOptions) (*services.RecordingImportResult, error) {
				return &services.RecordingImportResult{
					Execution: &database.Execution{
						ID:          executionID,
						WorkflowID:  workflowID,
						TriggerType: "extension",
					},
					Workflow: &database.Workflow{
						ID:        workflowID,
						Name:      "Imported Workflow",
						ProjectID: &projectID,
					},
					Project: &database.Project{
						ID:   projectID,
						Name: "Imported Project",
					},
					FrameCount: 10,
					AssetCount: 15,
					DurationMs: 5000,
				}, nil
			},
		}

		handler := setupRecordingTestHandler(t, mockService)

		// Create a multipart form with a test file
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "recording.zip")
		require.NoError(t, err)

		// Write some dummy ZIP content
		_, err = part.Write([]byte("PK\x03\x04dummy-zip-content"))
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/recordings/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		handler.ImportRecording(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]any
		err = json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, executionID.String(), response["execution_id"])
		assert.Equal(t, workflowID.String(), response["workflow_id"])
		assert.Equal(t, "Imported Workflow", response["workflow_name"])
		assert.Equal(t, projectID.String(), response["project_id"])
		assert.Equal(t, "Imported Project", response["project_name"])
		assert.Equal(t, float64(10), response["frame_count"])
		assert.Equal(t, float64(15), response["asset_count"])
		assert.Equal(t, float64(5000), response["duration_ms"])
		assert.Equal(t, "extension", response["trigger_type"])
	})
}

func TestImportRecording_ServiceUnavailable(t *testing.T) {
	t.Run("[REQ:BAS-REPLAY-TIMELINE-PERSISTENCE] returns error when recording service unavailable", func(t *testing.T) {
		handler := setupRecordingTestHandler(t, nil) // nil service

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "recording.zip")
		part.Write([]byte("PK\x03\x04dummy"))
		writer.Close()

		req := httptest.NewRequest("POST", "/api/v1/recordings/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		handler.ImportRecording(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)

		var response APIError
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "SERVICE_UNAVAILABLE", response.Code)
	})
}

func TestImportRecording_ArchiveTooLarge(t *testing.T) {
	t.Run("[REQ:BAS-REPLAY-TIMELINE-PERSISTENCE] rejects archives exceeding size limit", func(t *testing.T) {
		mockService := &mockRecordingService{
			importArchiveFn: func(ctx context.Context, archivePath string, opts services.RecordingImportOptions) (*services.RecordingImportResult, error) {
				return nil, services.ErrRecordingArchiveTooLarge
			},
		}

		handler := setupRecordingTestHandler(t, mockService)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "huge-recording.zip")
		// Write large dummy content
		largeData := make([]byte, 1024*1024) // 1MB
		part.Write(largeData)
		writer.Close()

		req := httptest.NewRequest("POST", "/api/v1/recordings/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		handler.ImportRecording(w, req)

		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)

		var response APIError
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "REQUEST_TOO_LARGE", response.Code)
		assert.Contains(t, response.Message, "exceeds maximum")
	})
}

func TestImportRecording_MissingManifest(t *testing.T) {
	t.Run("[REQ:BAS-REPLAY-TIMELINE-PERSISTENCE] rejects archives with missing manifest", func(t *testing.T) {
		mockService := &mockRecordingService{
			importArchiveFn: func(ctx context.Context, archivePath string, opts services.RecordingImportOptions) (*services.RecordingImportResult, error) {
				return nil, services.ErrRecordingManifestMissingFrames
			},
		}

		handler := setupRecordingTestHandler(t, mockService)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "invalid-recording.zip")
		part.Write([]byte("PK\x03\x04invalid-content"))
		writer.Close()

		req := httptest.NewRequest("POST", "/api/v1/recordings/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		handler.ImportRecording(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response APIError
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_REQUEST", response.Code)
	})
}

func TestImportRecording_TooManyFrames(t *testing.T) {
	t.Run("[REQ:BAS-REPLAY-TIMELINE-PERSISTENCE] rejects archives with too many frames", func(t *testing.T) {
		mockService := &mockRecordingService{
			importArchiveFn: func(ctx context.Context, archivePath string, opts services.RecordingImportOptions) (*services.RecordingImportResult, error) {
				return nil, services.ErrRecordingTooManyFrames
			},
		}

		handler := setupRecordingTestHandler(t, mockService)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "recording-many-frames.zip")
		part.Write([]byte("PK\x03\x04many-frames"))
		writer.Close()

		req := httptest.NewRequest("POST", "/api/v1/recordings/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		handler.ImportRecording(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response APIError
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_REQUEST", response.Code)
	})
}

func TestImportRecording_Timeout(t *testing.T) {
	t.Run("[REQ:BAS-REPLAY-TIMELINE-PERSISTENCE] handles import timeout gracefully", func(t *testing.T) {
		mockService := &mockRecordingService{
			importArchiveFn: func(ctx context.Context, archivePath string, opts services.RecordingImportOptions) (*services.RecordingImportResult, error) {
				// Simulate timeout
				<-ctx.Done()
				return nil, ctx.Err()
			},
		}

		handler := setupRecordingTestHandler(t, mockService)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "slow-recording.zip")
		part.Write([]byte("PK\x03\x04slow"))
		writer.Close()

		// Create request with short timeout
		req := httptest.NewRequest("POST", "/api/v1/recordings/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.ImportRecording(w, req)

		assert.Equal(t, http.StatusRequestTimeout, w.Code)

		var response APIError
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "REQUEST_TIMEOUT", response.Code)
	})
}

func TestImportRecording_MissingFile(t *testing.T) {
	t.Run("[REQ:BAS-REPLAY-TIMELINE-PERSISTENCE] rejects multipart request without file field", func(t *testing.T) {
		handler := setupRecordingTestHandler(t, &mockRecordingService{})

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		// Create a different form field, not "file"
		writer.WriteField("wrong_field", "some value")
		writer.Close()

		req := httptest.NewRequest("POST", "/api/v1/recordings/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		handler.ImportRecording(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response APIError
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_REQUEST", response.Code)
	})
}
