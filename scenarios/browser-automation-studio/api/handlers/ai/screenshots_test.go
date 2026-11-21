package ai

import (
	"context"
	"errors"
	"io"
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

func TestScreenshotHandler_RunPreviewScreenshot_Success(t *testing.T) {
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

	res, err := handler.RunPreviewScreenshot(context.Background(), PreviewScreenshotArgs{
		URL:            "https://example.com",
		ViewportWidth:  100,
		ViewportHeight: 20000,
	})

	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, []byte{0x89, 0x50, 0x4E}, res.ScreenshotPNG)
	assert.Equal(t, "image/png", res.ContentType)
	assert.Equal(t, "https://example.com/", res.URL)
	assert.Len(t, res.ConsoleLogs, 1)
	assert.Len(t, res.Events, 1)

	require.Len(t, mockRunner.RunCalls, 1)
	call := mockRunner.RunCalls[0]
	assert.Equal(t, previewMinViewportDimension, call.ViewportWidth)  // width clamped up from 100
	assert.Equal(t, previewMaxViewportDimension, call.ViewportHeight) // height clamped down from 20000
}

func TestScreenshotHandler_RunPreviewScreenshot_LooksUpOutcomesByNodeID(t *testing.T) {
	mockRunner := NewMockAutomationRunner()
	mockRunner.Outcomes = []autocontracts.StepOutcome{
		{Success: true, NodeID: "unrelated"},
		{Success: true, NodeID: "preview.screenshot", Screenshot: &autocontracts.Screenshot{Data: []byte{1}}, FinalURL: "https://example.com/"},
		{Success: true, NodeID: "preview.navigate"},
	}
	result, err := newScreenshotHandlerForTest(mockRunner).RunPreviewScreenshot(context.Background(), PreviewScreenshotArgs{URL: "https://example.com"})
	require.NoError(t, err)
	require.Equal(t, []byte{1}, result.ScreenshotPNG)
}

func TestScreenshotHandler_RunPreviewScreenshot_Errors(t *testing.T) {
	t.Run("rejects empty URL", func(t *testing.T) {
		handler := newScreenshotHandlerForTest(NewMockAutomationRunner())
		_, err := handler.RunPreviewScreenshot(context.Background(), PreviewScreenshotArgs{})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrMissingURL)
	})

	t.Run("errors when runner missing", func(t *testing.T) {
		log := logrus.New()
		log.SetOutput(io.Discard)
		handler := &ScreenshotHandler{log: log}
		_, err := handler.RunPreviewScreenshot(context.Background(), PreviewScreenshotArgs{URL: "https://example.com"})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAutomationRunnerNotReady)
	})

	t.Run("propagates automation runner error", func(t *testing.T) {
		mockRunner := &MockAutomationRunner{Err: errors.New("driver gone")}
		handler := newScreenshotHandlerForTest(mockRunner)
		_, err := handler.RunPreviewScreenshot(context.Background(), PreviewScreenshotArgs{URL: "https://example.com"})
		require.Error(t, err)
	})

	t.Run("reports navigation failure", func(t *testing.T) {
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
		_, err := handler.RunPreviewScreenshot(context.Background(), PreviewScreenshotArgs{URL: "https://example.com"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dns failed")
	})

	t.Run("reports screenshot data issues", func(t *testing.T) {
		mockRunner := NewMockAutomationRunner()
		mockRunner.Outcomes = []autocontracts.StepOutcome{
			{Success: true, NodeID: "preview.navigate"},
			{Success: true, NodeID: "preview.screenshot"},
		}
		handler := newScreenshotHandlerForTest(mockRunner)
		_, err := handler.RunPreviewScreenshot(context.Background(), PreviewScreenshotArgs{URL: "https://example.com"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no image data")
	})
}

func newScreenshotHandlerForTest(runner AutomationRunner) *ScreenshotHandler {
	log := logrus.New()
	log.SetOutput(io.Discard)
	return NewScreenshotHandler(log, WithScreenshotRunner(runner))
}
