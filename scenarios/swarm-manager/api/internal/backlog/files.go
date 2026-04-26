// File operations for backlog items: listing, reading, uploading, and
// manipulating files within a backlog item's directory tree.
package backlog

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/fileops"
	"swarm-manager/internal/fileserve"
	"swarm-manager/internal/httputil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

// ListFiles returns the file tree for a backlog item.
func (h *Handler) ListFiles(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "files")
	if !ok {
		return
	}

	itemDir := h.store.ItemDir(kind, name)
	if _, err := os.Stat(itemDir); os.IsNotExist(err) {
		apierr.MapError(w, "", apierr.NotFound("backlog item not found"))
		return
	}

	nodes, err := fileops.BuildFileTree(itemDir, "")
	if err != nil {
		apierr.MapError(w, "[backlog] list files", apierr.Internal("failed to read file tree"))
		return
	}

	resp := &apipb.BacklogFilesResponse{Files: fileserve.FileNodesToProto(nodes)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] list files", apierr.Internal("failed to encode response"))
	}
}

// GetFileContent returns the content of a specific file within a backlog item.
func (h *Handler) GetFileContent(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "get file")
	if !ok {
		return
	}

	itemDir := h.store.ItemDir(kind, name)
	if _, err := os.Stat(itemDir); os.IsNotExist(err) {
		apierr.MapError(w, "", apierr.NotFound("backlog item not found"))
		return
	}

	fileserve.GetContent(w, r, itemDir, "[backlog]")
}

// UploadFile handles file uploads to a backlog item folder.
//
// Acceptance validation runs after every successful upload: the agent writes
// rounds, plans, conclusions, and spec.json edits through this endpoint, so
// it is the canonical seam for catching plan/repo drift. A finalize round
// upload is rejected with `plan_stale` (409) if any acceptance globs reference
// paths that do not exist and are not declared in `creates`.
func (h *Handler) UploadFile(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "upload file")
	if !ok {
		return
	}

	itemDir := h.store.ItemDir(kind, name)
	if _, err := os.Stat(itemDir); os.IsNotExist(err) {
		apierr.MapError(w, "", apierr.NotFound("backlog item not found"))
		return
	}

	// Buffer the upload response so we can swap in a plan_stale 409 if
	// finalize-round validation fails. The file lands on disk during
	// fileserve.Upload regardless; if we block, we delete it before
	// replacing the response.
	rec := httptest.NewRecorder()
	fileserve.Upload(rec, r, itemDir, "", "[backlog]")

	// fileserve.Upload parses the multipart form, so r.FormValue and
	// r.MultipartForm are populated by the time it returns.
	uploadedPath := uploadedFilePath(r)

	uploadOK := rec.Code < 400
	if !uploadOK {
		flushRecorded(w, rec)
		return
	}

	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		flushRecorded(w, rec)
		return
	}

	report, valErr := runAcceptanceValidation(item, itemDir)
	if valErr != nil {
		slog.Warn("acceptance validation could not run after upload", "kind", kind, "name", name, "err", valErr)
		flushRecorded(w, rec)
		return
	}

	if report != nil && !report.Clean() && isFinalizeRoundUpload(itemDir, uploadedPath) {
		// Roll back the finalize round file: a stale plan must not be
		// recorded as finalized.
		if uploadedPath != "" {
			_ = os.Remove(filepath.Join(itemDir, filepath.FromSlash(filepath.Clean(uploadedPath))))
		}
		apierr.MapError(w, "[backlog] upload-file", apierr.PlanStale(
			"finalization blocked: plan references paths that do not exist and are not declared in `creates`",
			map[string]any{
				"missingPaths": report.Problems,
			},
		))
		return
	}

	flushRecorded(w, rec)
}

// uploadedFilePath reconstructs the server-relative destination path of an
// upload from the multipart form (path-field directory + form-file filename).
// Returns "" if either piece is missing.
func uploadedFilePath(r *http.Request) string {
	if r.MultipartForm == nil {
		return ""
	}
	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		return ""
	}
	name := files[0].Filename
	if name == "" {
		return ""
	}
	dir := strings.TrimSpace(r.FormValue("path"))
	if dir == "" || dir == "." {
		return name
	}
	return filepath.ToSlash(filepath.Join(dir, name))
}

// flushRecorded copies a buffered response onto the real ResponseWriter.
func flushRecorded(w http.ResponseWriter, rec *httptest.ResponseRecorder) {
	for k, v := range rec.Header() {
		w.Header()[k] = v
	}
	if rec.Code != 0 {
		w.WriteHeader(rec.Code)
	}
	_, _ = w.Write(rec.Body.Bytes())
}

// isFinalizeRoundUpload reports whether the just-uploaded path is a workshop
// round file whose JSON contains `"mode": "finalize"`. Other uploads (plan
// edits, conclusion edits, spec edits, non-finalize rounds) refresh the
// validation artifact but do not block.
func isFinalizeRoundUpload(itemDir, uploadedPath string) bool {
	if uploadedPath == "" {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(uploadedPath))
	if !strings.HasPrefix(clean, "workshop/round-") || !strings.HasSuffix(clean, ".json") {
		return false
	}
	data, err := os.ReadFile(filepath.Join(itemDir, clean))
	if err != nil {
		return false
	}
	var probe struct {
		Mode string `json:"mode"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return false
	}
	return probe.Mode == "finalize"
}

// OperateFile applies rename, move, copy, or delete to a backlog file path.
func (h *Handler) OperateFile(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "operate file")
	if !ok {
		return
	}

	itemDir := h.store.ItemDir(kind, name)
	if _, err := os.Stat(itemDir); os.IsNotExist(err) {
		apierr.MapError(w, "", apierr.NotFound("backlog item not found"))
		return
	}

	fileserve.Operate(w, r, itemDir, protectedBacklogFileName, "[backlog]")
}
