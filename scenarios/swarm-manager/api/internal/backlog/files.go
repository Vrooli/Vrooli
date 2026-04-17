// File operations for backlog items: listing, reading, uploading, and
// manipulating files within a backlog item's directory tree.
package backlog

import (
	"net/http"
	"os"
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

	fileserve.Upload(w, r, itemDir, "", "[backlog]")
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
