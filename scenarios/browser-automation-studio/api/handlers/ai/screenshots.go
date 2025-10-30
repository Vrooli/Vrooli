package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/constants"
)

// ScreenshotHandler handles screenshot-related operations
type ScreenshotHandler struct {
	log *logrus.Logger
}

// NewScreenshotHandler creates a new screenshot handler
func NewScreenshotHandler(log *logrus.Logger) *ScreenshotHandler {
	return &ScreenshotHandler{log: log}
}

// TakePreviewScreenshot handles POST /api/v1/ai/preview-screenshot
func (h *ScreenshotHandler) TakePreviewScreenshot(w http.ResponseWriter, r *http.Request) {
	type PreviewRequest struct {
		URL string `json:"url"`
	}

	var req PreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WithError(err).Error("Failed to decode preview request")
		RespondError(w, ErrInvalidRequest)
		return
	}

	if req.URL == "" {
		RespondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "url"}))
		return
	}

	h.log.WithField("url", req.URL).Info("Taking preview screenshot and capturing console logs")

	// Create temporary files for screenshot and console logs
	tmpScreenshotFile, err := os.CreateTemp("", "preview-*.png")
	if err != nil {
		h.log.WithError(err).Error("Failed to create temp screenshot file")
		RespondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "create_temp_file"}))
		return
	}
	defer os.Remove(tmpScreenshotFile.Name())
	defer tmpScreenshotFile.Close()

	tmpConsoleFile, err := os.CreateTemp("", "console-*.json")
	if err != nil {
		h.log.WithError(err).Error("Failed to create temp console file")
		RespondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "create_console_file"}))
		return
	}
	defer os.Remove(tmpConsoleFile.Name())
	defer tmpConsoleFile.Close()

	ctx, cancel := context.WithTimeout(r.Context(), constants.AIRequestTimeout)
	defer cancel()

	// Take screenshot first with explicit viewport to match extract viewport
	screenshotCmd := exec.CommandContext(ctx, "resource-browserless", "screenshot",
		"--url", req.URL,
		"--output", tmpScreenshotFile.Name())

	screenshotOutput, err := screenshotCmd.CombinedOutput()
	if err != nil {
		h.log.WithError(err).WithField("output", string(screenshotOutput)).Error("Failed to take screenshot with resource-browserless")

		// Provide more helpful error messages
		errorMsg := "Failed to take screenshot"
		details := map[string]string{"operation": "take_screenshot"}
		if strings.Contains(string(screenshotOutput), "HTTP 500") {
			errorMsg = "Unable to access the URL - it may not be reachable from the browser automation service"
			details["reason"] = "http_500"
		} else if strings.Contains(string(screenshotOutput), "timeout") {
			errorMsg = "Screenshot request timed out - the page may be taking too long to load"
			details["reason"] = "timeout"
		} else if strings.Contains(string(screenshotOutput), "connection") {
			errorMsg = "Cannot connect to the URL - please check if it's accessible"
			details["reason"] = "connection_error"
		}

		RespondError(w, ErrInternalServer.WithMessage(errorMsg).WithDetails(details))
		return
	}

	// Capture console logs
	consoleCmd := exec.CommandContext(ctx, "resource-browserless", "console",
		req.URL,
		"--output", tmpConsoleFile.Name(),
		"--wait-ms", "3000") // Wait a bit longer for dynamic content

	consoleOutput, consoleErr := consoleCmd.CombinedOutput()
	var consoleLogs any
	if consoleErr != nil {
		h.log.WithError(consoleErr).WithField("output", string(consoleOutput)).Warn("Failed to capture console logs, continuing with screenshot only")
		// Set empty console logs if capture fails
		consoleLogs = []any{}
	} else {
		// Read and parse console logs
		consoleData, err := os.ReadFile(tmpConsoleFile.Name())
		if err != nil {
			h.log.WithError(err).Warn("Failed to read console log file")
			consoleLogs = []any{}
		} else {
			var consoleResult map[string]any
			if err := json.Unmarshal(consoleData, &consoleResult); err != nil {
				h.log.WithError(err).Warn("Failed to parse console log JSON")
				consoleLogs = []any{}
			} else {
				if logs, ok := consoleResult["logs"]; ok {
					consoleLogs = logs
				} else {
					consoleLogs = []any{}
				}
			}
		}
	}

	// Check if the screenshot file was actually created and is valid
	fileInfo, err := os.Stat(tmpScreenshotFile.Name())
	if err != nil {
		h.log.WithError(err).Error("Screenshot file was not created")
		RespondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "stat_screenshot"}))
		return
	}

	// Check if file has content
	if fileInfo.Size() == 0 {
		h.log.Error("Screenshot file is empty")
		RespondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "validate_screenshot", "error": "empty_file"}))
		return
	}

	// Read the screenshot file
	screenshotData, err := os.ReadFile(tmpScreenshotFile.Name())
	if err != nil {
		h.log.WithError(err).Error("Failed to read screenshot file")
		RespondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "read_screenshot"}))
		return
	}

	// Validate that it's a PNG file by checking the magic bytes
	if len(screenshotData) < 8 || !bytes.Equal(screenshotData[:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
		h.log.Error("Generated file is not a valid PNG")
		RespondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "invalid_png_format"}))
		return
	}

	// Encode to base64
	base64Data := base64.StdEncoding.EncodeToString(screenshotData)

	// Return both screenshot and console logs
	response := map[string]any{
		"success":     true,
		"screenshot":  fmt.Sprintf("data:image/png;base64,%s", base64Data),
		"consoleLogs": consoleLogs,
		"url":         req.URL,
		"timestamp":   time.Now().Unix(),
	}

	RespondSuccess(w, http.StatusOK, response)
}
