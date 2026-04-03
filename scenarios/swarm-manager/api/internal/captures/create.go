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
	if err := os.MkdirAll(dir, 0o755); err != nil {
		apierr.MapError(w, "[captures] create", apierr.Internal("failed to create capture directory"))
		return
	}

	// Save attached image files.
	files := r.MultipartForm.File["files"]
	for _, fh := range files {
		mediaType, _, _ := mime.ParseMediaType(fh.Header.Get("Content-Type"))
		if !allowedImageTypes[mediaType] {
			// Clean up the capture directory on rejection.
			_ = os.RemoveAll(dir)
			apierr.MapError(w, "[captures] create", apierr.BadRequest("unsupported file type: %s", mediaType))
			return
		}

		attDir := filepath.Join(dir, "attachments")
		if err := os.MkdirAll(attDir, 0o755); err != nil {
			_ = os.RemoveAll(dir)
			apierr.MapError(w, "[captures] create", apierr.Internal("failed to create attachments directory"))
			return
		}

		safeName := sanitizeFilename(fh.Filename)
		destPath := filepath.Join(attDir, safeName)

		src, err := fh.Open()
		if err != nil {
			_ = os.RemoveAll(dir)
			apierr.MapError(w, "[captures] create", apierr.Internal("failed to read uploaded file"))
			return
		}

		dst, err := os.Create(destPath)
		if err != nil {
			src.Close()
			_ = os.RemoveAll(dir)
			apierr.MapError(w, "[captures] create", apierr.Internal("failed to save uploaded file"))
			return
		}

		_, copyErr := io.Copy(dst, src)
		src.Close()
		dst.Close()
		if copyErr != nil {
			_ = os.RemoveAll(dir)
			apierr.MapError(w, "[captures] create", apierr.Internal("failed to write uploaded file"))
			return
		}

		cap.Attachments = append(cap.Attachments, filepath.Join("attachments", safeName))
	}

	if err := h.writeCapture(&cap); err != nil {
		apierr.MapError(w, "[captures] create", apierr.Internal("failed to write capture"))
		return
	}

	// Auto-trigger classification agent.
	resp := map[string]any{"capture": cap}
	runResult, err := h.spawnClassifyAgent(r, &cap)
	if err != nil {
		// Classification failed to start, but capture was created. Mark as failed.
		slog.Error("classification spawn failed", "error", err)
		cap.Status = "failed"
		_ = h.writeCapture(&cap)
		resp["capture"] = cap
	} else {
		resp["task_id"] = runResult.TaskID
		resp["run_id"] = runResult.RunID
		resp["base_url"] = runResult.BaseURL
	}

	h.invalidateTopologyGraph()
	_ = httputil.JSONWithStatus(w, http.StatusCreated, resp)
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
