package captures

import (
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/idgen"
)

// allowedImageTypes lists Content-Types accepted for capture attachments.
var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// Create creates a new capture from a multipart form (text + optional image files).
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		apierr.MapError(w, "[captures] create", apierr.BadRequest("invalid multipart form"))
		return
	}

	text := strings.TrimSpace(r.FormValue("text"))
	if text == "" {
		apierr.MapError(w, "[captures] create", apierr.BadRequest("text is required"))
		return
	}

	id := fmt.Sprintf("cap-%s", idgen.Generate())
	now := time.Now().UTC().Format(time.RFC3339)

	cap := capture{
		ID:          id,
		Text:        text,
		Attachments: []string{},
		Created:     now,
		Status:      "classifying",
	}

	dir := h.captureDir(id)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		apierr.MapError(w, "[captures] create", apierr.Internal("failed to create capture directory"))
		return
	}

	// Save attached image files.
	files := r.MultipartForm.File["files"]
	for _, fh := range files {
		mediaType, _, _ := mime.ParseMediaType(fh.Header.Get("Content-Type"))
		if !allowedImageTypes[mediaType] {
			// Clean up the capture directory on rejection.
			cleanupCaptureDir(dir)
			apierr.MapError(w, "[captures] create", apierr.BadRequest("unsupported file type: %s", mediaType))
			return
		}

		attDir := filepath.Join(dir, "attachments")
		if err := os.MkdirAll(attDir, 0o750); err != nil {
			cleanupCaptureDir(dir)
			apierr.MapError(w, "[captures] create", apierr.Internal("failed to create attachments directory"))
			return
		}

		safeName := sanitizeFilename(fh.Filename)
		destPath := filepath.Join(attDir, safeName)

		src, err := fh.Open()
		if err != nil {
			cleanupCaptureDir(dir)
			apierr.MapError(w, "[captures] create", apierr.Internal("failed to read uploaded file"))
			return
		}

		dst, err := os.Create(destPath)
		if err != nil {
			closeBestEffort(src, "captures: close uploaded source")
			cleanupCaptureDir(dir)
			apierr.MapError(w, "[captures] create", apierr.Internal("failed to save uploaded file"))
			return
		}

		_, copyErr := io.Copy(dst, src)
		closeBestEffort(src, "captures: close uploaded source")
		closeBestEffort(dst, "captures: close attachment file")
		if copyErr != nil {
			cleanupCaptureDir(dir)
			apierr.MapError(w, "[captures] create", apierr.Internal("failed to write uploaded file"))
			return
		}

		cap.Attachments = append(cap.Attachments, filepath.Join("attachments", safeName))
	}

	if err := h.writeCapture(&cap); err != nil {
		apierr.MapError(w, "[captures] create", apierr.Internal("failed to write capture"))
		return
	}

	// Auto-trigger the declared classification workflow. Its typed output is
	// applied separately, so the workflow never receives filesystem authority.
	resp := map[string]any{"capture": cap}
	if h.transitionRunner == nil {
		apierr.MapError(w, "[captures] create", apierr.Unavailable("agent-manager is not available — try again once it's running"))
		return
	}
	start, err := h.transitionRunner.Start(r.Context(), "capture.classify", cap.ID)
	if err != nil {
		// Classification failed to start, but capture was created. Mark as failed
		// with a categorized reason so the UI can show actionable guidance.
		cap.Status = "failed"
		cap.FailureReason = classifyFailureReason(err)
		slog.Error("classification spawn failed",
			"capture_id", cap.ID,
			"failure_reason", cap.FailureReason,
			"error", err,
		)
		if writeErr := h.writeCapture(&cap); writeErr != nil {
			slog.Warn("captures: persist failed-status capture failed", "err", writeErr, "capture_id", cap.ID)
		}
		resp["capture"] = cap
	} else {
		resp["capture"] = cap
		resp["workflow_execution_id"] = start.ExecutionID
		resp["workflow_definition_digest"] = start.DefinitionDigest
	}

	h.invalidateTopologyGraph()
	if writeErr := httputil.JSONWithStatus(w, http.StatusCreated, resp); writeErr != nil {
		slog.Debug("captures: write create response failed", "err", writeErr)
	}
}

// sanitizeFilename removes path separators and dangerous characters from a filename.
func sanitizeFilename(name string) string {
	// Use only the base name (strip any directory components).
	name = filepath.Base(name)
	// Replace characters that could cause issues.
	replacer := strings.NewReplacer(
		"..", "_",
		"/", "_",
		"\\", "_",
		"\x00", "_",
	)
	name = replacer.Replace(name)
	if name == "" || name == "." {
		name = "unnamed"
	}
	return name
}

func cleanupCaptureDir(dir string) {
	if err := os.RemoveAll(dir); err != nil {
		slog.Debug("captures: cleanup capture dir failed", "err", err, "dir", dir)
	}
}

func closeBestEffort(c io.Closer, msg string) {
	if err := c.Close(); err != nil {
		slog.Debug(msg, "err", err)
	}
}
