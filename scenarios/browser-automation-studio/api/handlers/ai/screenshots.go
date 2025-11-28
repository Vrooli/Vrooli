package ai

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/internal/httpjson"
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

type previewConsoleLog struct {
	Level     string `json:"level"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

type previewRequest struct {
	URL      string `json:"url"`
	Viewport *struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"viewport,omitempty"`
}

type ScreenshotHandler struct {
	log    *logrus.Logger
	runner *automationRunner
}

func NewScreenshotHandler(log *logrus.Logger) *ScreenshotHandler {
	runner, err := newAutomationRunner(log)
	if err != nil && log != nil {
		log.WithError(err).Warn("Failed to initialize automation runner for screenshots; requests will fail")
	}
	return &ScreenshotHandler{log: log, runner: runner}
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

func (h *ScreenshotHandler) TakePreviewScreenshot(w http.ResponseWriter, r *http.Request) {
	var req previewRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		h.log.WithError(err).Error("Failed to decode preview request")
		RespondError(w, ErrInvalidRequest)
		return
	}

	if strings.TrimSpace(req.URL) == "" {
		RespondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "url"}))
		return
	}

	viewportWidth := previewDefaultViewportWidth
	viewportHeight := previewDefaultViewportHeight
	if req.Viewport != nil {
		if w := clampPreviewViewport(req.Viewport.Width); w > 0 {
			viewportWidth = w
		}
		if hVal := clampPreviewViewport(req.Viewport.Height); hVal > 0 {
			viewportHeight = hVal
		}
	}

	if h.runner == nil {
		RespondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "automation_runner"}))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.AIRequestTimeout)
	defer cancel()

	instructions := []autocontracts.CompiledInstruction{
		{
			Index:  0,
			NodeID: "preview.navigate",
			Type:   "navigate",
			Params: map[string]any{
				"url":       req.URL,
				"waitUntil": defaultPreviewWaitUntil,
				"timeoutMs": defaultPreviewTimeoutMilliseconds,
			},
		},
		{
			Index:  1,
			NodeID: "preview.screenshot",
			Type:   "screenshot",
			Params: map[string]any{
				"fullPage":  true,
				"waitForMs": defaultPreviewWaitMilliseconds,
				"timeoutMs": defaultPreviewTimeoutMilliseconds,
			},
		},
	}

	start := time.Now()
	outcomes, events, err := h.runner.run(ctx, viewportWidth, viewportHeight, instructions)
	if err != nil {
		h.log.WithError(err).Error("Preview automation failed")
		RespondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "automation_run", "error": err.Error()}))
		return
	}

	if len(outcomes) < 2 {
		RespondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "automation_run", "error": "no screenshot outcome"}))
		return
	}

	nav := outcomes[0]
	if !nav.Success {
		message := "navigation failed"
		if nav.Failure != nil && strings.TrimSpace(nav.Failure.Message) != "" {
			message = strings.TrimSpace(nav.Failure.Message)
		}
		RespondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "navigate", "error": message}))
		return
	}

	shot := outcomes[1]
	if !shot.Success {
		message := "screenshot failed"
		if shot.Failure != nil && strings.TrimSpace(shot.Failure.Message) != "" {
			message = strings.TrimSpace(shot.Failure.Message)
		}
		RespondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "screenshot", "error": message}))
		return
	}

	if shot.Screenshot == nil || len(shot.Screenshot.Data) == 0 {
		RespondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "screenshot", "error": "no image data"}))
		return
	}

	encoded := base64.StdEncoding.EncodeToString(shot.Screenshot.Data)
	logs := make([]previewConsoleLog, 0, len(shot.ConsoleLogs))
	for _, entry := range shot.ConsoleLogs {
		logs = append(logs, previewConsoleLog{
			Level:     entry.Type,
			Message:   entry.Text,
			Timestamp: entry.Timestamp.Format(time.RFC3339Nano),
		})
	}

	response := map[string]any{
		"success":        true,
		"screenshot":     fmt.Sprintf("data:image/png;base64,%s", encoded),
		"consoleLogs":    logs,
		"url":            shot.FinalURL,
		"timestamp":      time.Now().Unix(),
		"duration_ms":    time.Since(start).Milliseconds(),
		"viewportWidth":  viewportWidth,
		"viewportHeight": viewportHeight,
		"events":         events,
	}

	h.log.WithFields(logrus.Fields{
		"url":             req.URL,
		"viewport_width":  viewportWidth,
		"viewport_height": viewportHeight,
		"duration_ms":     response["duration_ms"],
		"console_logs":    len(logs),
	}).Info("Captured preview screenshot")

	RespondSuccess(w, http.StatusOK, response)
}
