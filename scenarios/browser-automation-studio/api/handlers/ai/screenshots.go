package ai

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/constants"
)

const (
	previewMinViewportDimension  = 200
	previewMaxViewportDimension  = 10000
	previewDefaultViewportWidth  = 1920
	previewDefaultViewportHeight = 1080

	browserlessFunctionPath           = "/chrome/function"
	defaultPreviewWaitMilliseconds    = 1200
	defaultPreviewTimeoutMilliseconds = 20000
	maxPreviewTimeoutMilliseconds     = 45000
	maxPreviewWaitMilliseconds        = 8000
	maxPreviewConsoleLogs             = 200
	defaultPreviewWaitUntil           = "networkidle2"
)

var previewScreenshotScript = fmt.Sprintf(`export default async ({ page, context }) => {
  const logs = [];
  const cap = %d;
  const now = () => new Date().toISOString();

  const normalizeLevel = (value) => {
    if (typeof value !== "string") {
      return "log";
    }
    const normalized = value.toLowerCase();
    if (normalized === "warning") {
      return "warn";
    }
    if (normalized === "error" || normalized === "warn" || normalized === "info" || normalized === "debug" || normalized === "trace") {
      return normalized;
    }
    return "log";
  };

  const toMessage = (value) => {
    if (typeof value === "string") {
      return value;
    }
    if (value === null || typeof value === "undefined") {
      return "";
    }
    try {
      return JSON.stringify(value);
    } catch (stringifyError) {
      return String(value);
    }
  };

  const pushLog = (level, message) => {
    if (logs.length >= cap) {
      return;
    }
    logs.push({
      level: normalizeLevel(level),
      message: toMessage(message),
      timestamp: now(),
    });
  };

  const contextValue = context || {};
  const targetUrl = typeof contextValue.url === "string" ? contextValue.url : "";
  const rawWaitUntil = typeof contextValue.waitUntil === "string" ? contextValue.waitUntil.trim() : "";
  const waitUntil = rawWaitUntil ? rawWaitUntil : "%s";

  if (!targetUrl) {
    return {
      success: false,
      error: "Missing URL",
      consoleLogs: logs,
      finalUrl: "",
    };
  }

  try {
    const viewport = contextValue.viewport || {};
    const rawWidth = Number(viewport.width);
    const rawHeight = Number(viewport.height);
    if (Number.isFinite(rawWidth) && Number.isFinite(rawHeight)) {
      const width = Math.max(%d, Math.min(%d, Math.floor(rawWidth)));
      const height = Math.max(%d, Math.min(%d, Math.floor(rawHeight)));
      await page.setViewport({ width, height });
    }

    try {
      await page.setCacheEnabled(true);
    } catch (_) {
      // ignore cache enable errors
    }

    const rawWait = Number(contextValue.waitMs);
    const waitMs = Number.isFinite(rawWait) ? Math.max(0, Math.min(%d, Math.floor(rawWait))) : %d;

    const rawTimeout = Number(contextValue.timeout);
    const timeout = Number.isFinite(rawTimeout) ? Math.max(5000, Math.min(%d, Math.floor(rawTimeout))) : %d;

    if (typeof page.setDefaultNavigationTimeout === "function") {
      page.setDefaultNavigationTimeout(timeout);
    }
    if (typeof page.setDefaultTimeout === "function") {
      page.setDefaultTimeout(timeout);
    }

    page.on("console", (message) => {
      try {
        const level = typeof message.type === "function" ? message.type() : "log";
        const text = typeof message.text === "function" ? message.text() : "";
        pushLog(level, text);
      } catch (consoleError) {
        const message = consoleError && consoleError.message ? consoleError.message : String(consoleError);
        pushLog("warn", "Failed to process console message: " + message);
      }
    });

    page.on("pageerror", (pageError) => {
      const message = pageError && pageError.message ? pageError.message : String(pageError);
      pushLog("error", message);
    });

    page.on("requestfailed", (request) => {
      const failure = typeof request.failure === "function" ? request.failure() : null;
      const parts = ["Request failed"];
      const url = typeof request.url === "function" ? request.url() : "";
      if (url) {
        parts.push(url);
      }
      if (failure && failure.errorText) {
        parts.push("(" + failure.errorText + ")");
      }
      pushLog("error", parts.join(" "));
    });

    const response = await page.goto(targetUrl, { waitUntil, timeout });
    if (response && typeof response.ok === "function" && !response.ok()) {
      const status = typeof response.status === "function" ? response.status() : "unknown";
      pushLog("warn", "Navigation returned status " + status + " for " + targetUrl);
    }

    try {
      await page.waitForFunction(() => document.readyState === "complete", { timeout: 5000 });
    } catch (_) {
      // ignore wait timeout
    }

    if (waitMs > 0 && typeof page.waitForTimeout === "function") {
      await page.waitForTimeout(waitMs);
    }

    const screenshot = await page.screenshot({ type: "png", encoding: "base64" });

    const finalUrl = typeof page.url === "function" ? page.url() : targetUrl;

    return {
      success: true,
      screenshot,
      consoleLogs: logs,
      finalUrl,
    };
  } catch (error) {
    const message = error && error.message ? error.message : "Failed to take screenshot";
    let finalUrl = targetUrl;
    try {
      if (typeof page.url === "function") {
        finalUrl = page.url();
      }
    } catch (urlError) {
      if (urlError && urlError.message) {
        pushLog("warn", "Unable to read final URL: " + urlError.message);
      }
    }
    pushLog("error", message);
    return {
      success: false,
      error: message,
      consoleLogs: logs,
      finalUrl,
    };
  }
};`,
	maxPreviewConsoleLogs,
	defaultPreviewWaitUntil,
	previewMinViewportDimension,
	previewMaxViewportDimension,
	previewMinViewportDimension,
	previewMaxViewportDimension,
	maxPreviewWaitMilliseconds,
	defaultPreviewWaitMilliseconds,
	maxPreviewTimeoutMilliseconds,
	defaultPreviewTimeoutMilliseconds,
)

type previewConsoleLog struct {
	Level     string `json:"level"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

type previewResponse struct {
	Success     bool                `json:"success"`
	Screenshot  string              `json:"screenshot"`
	ConsoleLogs []previewConsoleLog `json:"consoleLogs"`
	Error       string              `json:"error"`
	FinalURL    string              `json:"finalUrl"`
}

type previewRequest struct {
	URL      string `json:"url"`
	Viewport *struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"viewport,omitempty"`
}

type ScreenshotHandler struct {
	log            *logrus.Logger
	httpClient     *http.Client
	browserlessURL string
}

func derivePreviewSessionID(rawURL string, width, height int) string {
	trimmed := strings.TrimSpace(strings.ToLower(rawURL))
	if trimmed == "" {
		return ""
	}
	data := fmt.Sprintf("%s|%d|%d", trimmed, width, height)
	sum := sha1.Sum([]byte(data))
	return fmt.Sprintf("preview-%x", sum)
}

func NewScreenshotHandler(log *logrus.Logger) *ScreenshotHandler {
	client := &http.Client{Timeout: constants.AIRequestTimeout}
	return &ScreenshotHandler{
		log:            log,
		httpClient:     client,
		browserlessURL: resolveBrowserlessURL(),
	}
}

func resolveBrowserlessURL() string {
	candidates := []string{
		strings.TrimSpace(os.Getenv("BROWSERLESS_URL")),
		strings.TrimSpace(os.Getenv("BROWSERLESS_BASE_URL")),
	}
	for _, candidate := range candidates {
		if candidate != "" {
			return strings.TrimRight(candidate, "/")
		}
	}

	port := strings.TrimSpace(os.Getenv("BROWSERLESS_PORT"))
	if port == "" {
		port = "4110"
	}
	host := strings.TrimSpace(os.Getenv("BROWSERLESS_HOST"))
	if host == "" {
		host = "127.0.0.1"
	}
	scheme := strings.TrimSpace(os.Getenv("BROWSERLESS_SCHEME"))
	if scheme == "" {
		scheme = "http"
	}

	return fmt.Sprintf("%s://%s:%s", scheme, host, port)
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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

	if h.browserlessURL == "" {
		h.browserlessURL = resolveBrowserlessURL()
	}
	endpoint := strings.TrimRight(h.browserlessURL, "/") + browserlessFunctionPath

	sessionID := derivePreviewSessionID(req.URL, viewportWidth, viewportHeight)

	contextPayload := map[string]any{
		"url": req.URL,
		"viewport": map[string]int{
			"width":  viewportWidth,
			"height": viewportHeight,
		},
		"waitMs":    defaultPreviewWaitMilliseconds,
		"timeout":   defaultPreviewTimeoutMilliseconds,
		"waitUntil": defaultPreviewWaitUntil,
	}

	if sessionID != "" {
		contextPayload["sessionId"] = sessionID
		contextPayload["keepAlive"] = true
	}

	payload := map[string]any{
		"code":    previewScreenshotScript,
		"context": contextPayload,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		h.log.WithError(err).Error("Failed to encode browserless payload")
		RespondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "encode_payload"}))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.AIRequestTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		h.log.WithError(err).Error("Failed to create browserless request")
		RespondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "create_request"}))
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := h.httpClient.Do(httpReq)
	if err != nil {
		h.log.WithError(err).Error("Browserless request failed")
		RespondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "browserless_request"}))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		h.log.WithFields(logrus.Fields{
			"status": resp.StatusCode,
			"body":   string(bodyBytes),
		}).Error("Browserless returned error status")
		RespondError(w, ErrInternalServer.WithMessage("Failed to take preview screenshot").WithDetails(map[string]any{
			"operation": "browserless_response",
			"status":    resp.StatusCode,
		}))
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		h.log.WithError(err).Error("Failed to read browserless response")
		RespondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "read_response"}))
		return
	}

	var preview previewResponse
	if err := json.Unmarshal(bodyBytes, &preview); err != nil {
		h.log.WithError(err).WithField("body", string(bodyBytes)).Error("Failed to parse browserless response")
		RespondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "decode_response"}))
		return
	}

	if !preview.Success || strings.TrimSpace(preview.Screenshot) == "" {
		detail := map[string]any{
			"operation": "browserless_preview",
		}
		if strings.TrimSpace(preview.Error) != "" {
			detail["error"] = preview.Error
		}
		if len(preview.ConsoleLogs) > 0 {
			detail["consoleLogCount"] = len(preview.ConsoleLogs)
		}
		RespondError(w, ErrInternalServer.WithMessage("Failed to capture preview screenshot").WithDetails(detail))
		return
	}

	rawScreenshot, err := base64.StdEncoding.DecodeString(preview.Screenshot)
	if err != nil {
		h.log.WithError(err).Error("Invalid screenshot base64")
		RespondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "decode_screenshot"}))
		return
	}

	if len(rawScreenshot) < 8 || !bytes.Equal(rawScreenshot[:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
		h.log.Error("Browserless response is not a valid PNG")
		RespondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "invalid_png_format"}))
		return
	}

	dataURI := fmt.Sprintf("data:image/png;base64,%s", preview.Screenshot)

	response := map[string]any{
		"success":        true,
		"screenshot":     dataURI,
		"consoleLogs":    preview.ConsoleLogs,
		"url":            preview.FinalURL,
		"timestamp":      time.Now().Unix(),
		"duration_ms":    time.Since(start).Milliseconds(),
		"viewportWidth":  viewportWidth,
		"viewportHeight": viewportHeight,
	}

	h.log.WithFields(logrus.Fields{
		"url":             req.URL,
		"viewport_width":  viewportWidth,
		"viewport_height": viewportHeight,
		"duration_ms":     response["duration_ms"],
		"console_logs":    len(preview.ConsoleLogs),
	}).Info("Captured preview screenshot")

	RespondSuccess(w, http.StatusOK, response)
}
