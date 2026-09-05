package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	autocompiler "github.com/vrooli/browser-automation-studio/automation/compiler"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/constants"
)

const (
	previewMinViewportDimension       = 200
	previewMaxViewportDimension       = 10000
	previewDefaultViewportWidth       = 1920
	previewDefaultViewportHeight      = 1080
	defaultPreviewWaitMilliseconds    = 1200
	defaultPreviewTimeoutMilliseconds = 20000
	defaultPreviewWaitUntil           = "networkidle"
)

// PreviewConsoleLog mirrors one browser console entry captured during a
// preview screenshot run. Transport-agnostic; callers (Connect handlers,
// tests) reuse this type rather than redefining it.
type PreviewConsoleLog struct {
	Level     string
	Message   string
	Timestamp time.Time
}

// PreviewScreenshotResult is the Go-typed return shape of
// ScreenshotHandler.RunPreviewScreenshot. The Connect adapter maps this onto
// the proto response.
type PreviewScreenshotResult struct {
	ScreenshotPNG  []byte
	ContentType    string
	ConsoleLogs    []PreviewConsoleLog
	URL            string
	CapturedAt     time.Time
	DurationMS     int64
	ViewportWidth  int
	ViewportHeight int
	Events         []autocontracts.EventEnvelope
}

// PreviewScreenshotArgs is the Go-typed input to RunPreviewScreenshot.
// ViewportWidth/Height of 0 = use defaults.
type PreviewScreenshotArgs struct {
	URL               string
	ViewportWidth     int
	ViewportHeight    int
	DeviceScaleFactor float64
}

type ScreenshotHandler struct {
	log    *logrus.Logger
	runner AutomationRunner
}

// ScreenshotHandlerOption configures the ScreenshotHandler.
type ScreenshotHandlerOption func(*ScreenshotHandler)

// WithScreenshotRunner sets a custom automation runner for screenshots.
func WithScreenshotRunner(runner AutomationRunner) ScreenshotHandlerOption {
	return func(h *ScreenshotHandler) {
		h.runner = runner
	}
}

// NewScreenshotHandler creates a new ScreenshotHandler with optional configuration.
func NewScreenshotHandler(log *logrus.Logger, opts ...ScreenshotHandlerOption) *ScreenshotHandler {
	handler := &ScreenshotHandler{log: log}

	// Apply options first
	for _, opt := range opts {
		opt(handler)
	}

	// Create default runner if not provided
	if handler.runner == nil {
		runner, err := newAutomationRunner(log)
		if err != nil && log != nil {
			log.WithError(err).Warn("Failed to initialize automation runner for screenshots; requests will fail")
		}
		handler.runner = runner
	}

	return handler
}

func clampPreviewViewport(value int) int {
	if value <= 0 {
		return 0
	}
	if value < previewMinViewportDimension {
		return previewMinViewportDimension
	}
	if value > previewMaxViewportDimension {
		return previewMaxViewportDimension
	}
	return value
}

// RunPreviewScreenshot navigates to a URL and captures a full-page PNG.
// It is the transport-agnostic core used by the AIService Connect handler.
// Returns sentinel errors that callers can map onto transport codes:
//   - ErrMissingURL                — request had empty URL
//   - ErrAutomationRunnerNotReady — handler missing a runner
//   - All other errors            — wrap underlying automation failures.
func (h *ScreenshotHandler) RunPreviewScreenshot(ctx context.Context, args PreviewScreenshotArgs) (*PreviewScreenshotResult, error) {
	if strings.TrimSpace(args.URL) == "" {
		return nil, ErrMissingURL
	}
	if h.runner == nil {
		return nil, ErrAutomationRunnerNotReady
	}

	viewportWidth := previewDefaultViewportWidth
	viewportHeight := previewDefaultViewportHeight
	if w := clampPreviewViewport(args.ViewportWidth); w > 0 {
		viewportWidth = w
	}
	if hVal := clampPreviewViewport(args.ViewportHeight); hVal > 0 {
		viewportHeight = hVal
	}

	ctx, cancel := context.WithTimeout(ctx, constants.AIRequestTimeout)
	defer cancel()

	instructions, err := h.buildPreviewScreenshotInstructions(args.URL)
	if err != nil {
		return nil, fmt.Errorf("build preview instructions: %w", err)
	}

	start := time.Now()
	if args.DeviceScaleFactor != 0 && (args.DeviceScaleFactor < 0.5 || args.DeviceScaleFactor > 4.0) {
		return nil, fmt.Errorf("device scale factor must be between 0.5 and 4.0")
	}

	var outcomes []autocontracts.StepOutcome
	var events []autocontracts.EventEnvelope
	if scaledRunner, ok := h.runner.(deviceScaleAutomationRunner); ok {
		outcomes, events, err = scaledRunner.RunWithDeviceScale(ctx, viewportWidth, viewportHeight, args.DeviceScaleFactor, instructions)
	} else {
		outcomes, events, err = h.runner.Run(ctx, viewportWidth, viewportHeight, instructions)
	}
	if err != nil {
		return nil, fmt.Errorf("automation run: %w", err)
	}

	outcomeByNodeID := make(map[string]autocontracts.StepOutcome, len(outcomes))
	for _, outcome := range outcomes {
		outcomeByNodeID[outcome.NodeID] = outcome
	}
	nav, ok := outcomeByNodeID["preview.navigate"]
	if !ok {
		return nil, errors.New("automation run returned no navigation outcome")
	}
	if !nav.Success {
		message := "navigation failed"
		if nav.Failure != nil && strings.TrimSpace(nav.Failure.Message) != "" {
			message = strings.TrimSpace(nav.Failure.Message)
		}
		return nil, fmt.Errorf("navigate: %s", message)
	}

	shot, ok := outcomeByNodeID["preview.screenshot"]
	if !ok {
		return nil, errors.New("automation run returned no screenshot outcome")
	}
	if !shot.Success {
		message := "screenshot failed"
		if shot.Failure != nil && strings.TrimSpace(shot.Failure.Message) != "" {
			message = strings.TrimSpace(shot.Failure.Message)
		}
		return nil, fmt.Errorf("screenshot: %s", message)
	}

	if shot.Screenshot == nil || len(shot.Screenshot.Data) == 0 {
		return nil, errors.New("screenshot: no image data")
	}

	logs := make([]PreviewConsoleLog, 0, len(shot.ConsoleLogs))
	for _, entry := range shot.ConsoleLogs {
		logs = append(logs, PreviewConsoleLog{
			Level:     entry.Type,
			Message:   entry.Text,
			Timestamp: entry.Timestamp,
		})
	}

	durationMS := time.Since(start).Milliseconds()
	if h.log != nil {
		h.log.WithFields(logrus.Fields{
			"url":             args.URL,
			"viewport_width":  viewportWidth,
			"viewport_height": viewportHeight,
			"duration_ms":     durationMS,
			"console_logs":    len(logs),
		}).Info("Captured preview screenshot")
	}

	return &PreviewScreenshotResult{
		ScreenshotPNG:  shot.Screenshot.Data,
		ContentType:    "image/png",
		ConsoleLogs:    logs,
		URL:            shot.FinalURL,
		CapturedAt:     time.Now(),
		DurationMS:     durationMS,
		ViewportWidth:  viewportWidth,
		ViewportHeight: viewportHeight,
		Events:         events,
	}, nil
}

// ErrMissingURL signals that a caller did not provide a non-empty URL.
var ErrMissingURL = errors.New("url is required")

// ErrAutomationRunnerNotReady signals that an AI helper handler was created
// without a working automation runner. Treated as Internal by transports.
var ErrAutomationRunnerNotReady = errors.New("automation runner not configured")

// buildPreviewScreenshotInstructions creates the compiled instructions for taking a preview screenshot.
// Returns an error if any action type fails to build (indicates a programming error).
func (h *ScreenshotHandler) buildPreviewScreenshotInstructions(url string) ([]autocontracts.CompiledInstruction, error) {
	steps := []struct {
		nodeID   string
		stepType string
		params   map[string]any
	}{
		{
			nodeID:   "preview.navigate",
			stepType: "navigate",
			params: map[string]any{
				"url":       url,
				"waitUntil": defaultPreviewWaitUntil,
				"timeoutMs": defaultPreviewTimeoutMilliseconds,
			},
		},
		{
			nodeID:   "preview.screenshot",
			stepType: "screenshot",
			params: map[string]any{
				"fullPage":  true,
				"waitForMs": defaultPreviewWaitMilliseconds,
				"timeoutMs": defaultPreviewTimeoutMilliseconds,
			},
		},
	}

	instructions := make([]autocontracts.CompiledInstruction, 0, len(steps))
	for i, step := range steps {
		action, err := autocompiler.BuildActionDefinition(step.stepType, step.params)
		if err != nil {
			return nil, fmt.Errorf("build action %q (node %s): %w", step.stepType, step.nodeID, err)
		}
		instructions = append(instructions, autocontracts.CompiledInstruction{
			Index:  i,
			NodeID: step.nodeID,
			Action: action,
		})
	}

	return instructions, nil
}
