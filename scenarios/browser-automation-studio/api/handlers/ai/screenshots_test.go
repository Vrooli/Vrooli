package ai

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
)

func TestClampPreviewViewport(t *testing.T) {
	t.Run("returns 0 when value <= 0", func(t *testing.T) {
		assert.Equal(t, 0, clampPreviewViewport(0))
		assert.Equal(t, 0, clampPreviewViewport(-10))
	})

	t.Run("raises to minimum when below bounds", func(t *testing.T) {
		assert.Equal(t, previewMinViewportDimension, clampPreviewViewport(previewMinViewportDimension-50))
	})

	t.Run("caps at maximum", func(t *testing.T) {
		assert.Equal(t, previewMaxViewportDimension, clampPreviewViewport(previewMaxViewportDimension+500))
	})

	t.Run("passes through valid values", func(t *testing.T) {
		assert.Equal(t, 800, clampPreviewViewport(800))
	})
}

func TestScreenshotHandler_TakePreviewScreenshot_Success(t *testing.T) {
	mockRunner := NewMockAutomationRunner()
	mockRunner.Outcomes = []autocontracts.StepOutcome{
		{
			Success:  true,
			NodeID:   "preview.navigate",
			StepType: "navigate",
		},
		{
			Success:  true,
			NodeID:   "preview.screenshot",
			StepType: "screenshot",
			Screenshot: &autocontracts.Screenshot{
				Data: []byte{0x89, 0x50, 0x4E},
			},
			FinalURL: "https://example.com/",
			ConsoleLogs: []autocontracts.ConsoleLogEntry{
				{Type: "log", Text: "ready"},
			},
		},
	}
	mockRunner.Events = []autocontracts.EventEnvelope{
		{Kind: autocontracts.EventKindExecutionCompleted},
	}

	handler := newScreenshotHandlerForTest(mockRunner)

	reqBody := previewRequest{
		URL: "https://example.com",
		Viewport: &struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		}{Width: 100, Height: 20000},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/preview-screenshot", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler.TakePreviewScreenshot(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&response))
	assert.Equal(t, true, response["success"])
	assert.Contains(t, response["screenshot"].(string), "data:image/png;base64,")
	events, ok := response["events"].([]any)
	require.True(t, ok)
	assert.Len(t, events, len(mockRunner.Events))

	require.Len(t, mockRunner.RunCalls, 1)
	call := mockRunner.RunCalls[0]
	assert.Equal(t, previewMinViewportDimension, call.ViewportWidth)  // width clamped up from 100
	assert.Equal(t, previewMaxViewportDimension, call.ViewportHeight) // height clamped down from 20000
	require.Len(t, call.Instructions, 2)
	assert.Equal(t, "preview.navigate", call.Instructions[0].NodeID)
	assert.Equal(t, "preview.screenshot", call.Instructions[1].NodeID)
}

func TestScreenshotHandler_TakePreviewScreenshot_Errors(t *testing.T) {
	t.Run("rejects invalid JSON", func(t *testing.T) {
		handler := newScreenshotHandlerForTest(NewMockAutomationRunner())
		req := httptest.NewRequest("POST", "/preview", bytes.NewBufferString("not json"))
		w := httptest.NewRecorder()

		handler.TakePreviewScreenshot(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("rejects missing URL", func(t *testing.T) {
		handler := newScreenshotHandlerForTest(NewMockAutomationRunner())
		body, _ := json.Marshal(previewRequest{})
		req := httptest.NewRequest("POST", "/preview", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handler.TakePreviewScreenshot(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("errors when runner missing", func(t *testing.T) {
		log := logrus.New()
		log.SetOutput(io.Discard)
		handler := &ScreenshotHandler{log: log}

		body, _ := json.Marshal(previewRequest{URL: "https://example.com"})
		req := httptest.NewRequest("POST", "/preview", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handler.TakePreviewScreenshot(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("propagates automation runner error", func(t *testing.T) {
		mockRunner := &MockAutomationRunner{Err: assert.AnError}
		handler := newScreenshotHandlerForTest(mockRunner)

		body, _ := json.Marshal(previewRequest{URL: "https://example.com"})
		req := httptest.NewRequest("POST", "/preview", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handler.TakePreviewScreenshot(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("reports navigation failure message", func(t *testing.T) {
		mockRunner := NewMockAutomationRunner()
		mockRunner.Outcomes = []autocontracts.StepOutcome{
			{
				Success: false,
				NodeID:  "preview.navigate",
				Failure: &autocontracts.StepFailure{Message: "dns failed"},
			},
			{Success: true, NodeID: "preview.screenshot"},
		}
		handler := newScreenshotHandlerForTest(mockRunner)

		body, _ := json.Marshal(previewRequest{URL: "https://example.com"})
		req := httptest.NewRequest("POST", "/preview", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handler.TakePreviewScreenshot(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("reports screenshot data issues", func(t *testing.T) {
		mockRunner := NewMockAutomationRunner()
		mockRunner.Outcomes = []autocontracts.StepOutcome{
			{Success: true, NodeID: "preview.navigate"},
			{Success: true, NodeID: "preview.screenshot"},
		}
		handler := newScreenshotHandlerForTest(mockRunner)

		body, _ := json.Marshal(previewRequest{URL: "https://example.com"})
		req := httptest.NewRequest("POST", "/preview", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handler.TakePreviewScreenshot(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func newScreenshotHandlerForTest(runner AutomationRunner) *ScreenshotHandler {
	log := logrus.New()
	log.SetOutput(io.Discard)
	return NewScreenshotHandler(log, WithScreenshotRunner(runner))
}
