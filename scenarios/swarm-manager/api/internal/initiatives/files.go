// File operations for initiatives: listing, reading, uploading, and
// manipulating files within an initiative's directory tree.
package initiatives

import (
	"net/http"
	"os"
	"strings"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/fileops"
	"swarm-manager/internal/fileserve"
	"swarm-manager/internal/httputil"

	"github.com/gorilla/mux"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

// protectedInitiativeFile is the metadata file that cannot be modified
// through file operations.
const protectedInitiativeFile = "initiative.json"

// extractName reads the {name} path variable and validates it.
// Returns the name and true, or writes a 400 and returns ("", false).
func (h *Handler) extractName(w http.ResponseWriter, r *http.Request, context string) (string, bool) {
	name := mux.Vars(r)["name"]
	if strings.TrimSpace(name) == "" {
		apierr.MapError(w, "[initiatives] "+context, apierr.BadRequest("name is required"))
		return "", false
	}
	return name, true
}

// ListInitiativeFiles returns the file tree for an initiative.
func (h *Handler) ListInitiativeFiles(w http.ResponseWriter, r *http.Request) {
	name, ok := h.extractName(w, r, "list files")
	if !ok {
		return
	}

	initDir := h.service.InitDir(name)
	if _, err := os.Stat(initDir); os.IsNotExist(err) {
		apierr.MapError(w, "", apierr.NotFound("initiative not found"))
		return
	}

	nodes, err := fileops.BuildFileTree(initDir, "")
	if err != nil {
		apierr.MapError(w, "[initiatives] list files", apierr.Internal("failed to read file tree"))
		return
	}

	resp := &apipb.BacklogFilesResponse{Files: fileserve.FileNodesToProto(nodes)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[initiatives] list files", apierr.Internal("failed to encode response"))
	}
}

// GetInitiativeFileContent returns the content of a file within an initiative.
func (h *Handler) GetInitiativeFileContent(w http.ResponseWriter, r *http.Request) {
	name, ok := h.extractName(w, r, "get file")
	if !ok {
		return
	}

	initDir := h.service.InitDir(name)
	if _, err := os.Stat(initDir); os.IsNotExist(err) {
		apierr.MapError(w, "", apierr.NotFound("initiative not found"))
		return
	}

	fileserve.GetContent(w, r, initDir, "[initiatives]")
}

// UploadInitiativeFile handles file uploads to an initiative folder.
func (h *Handler) UploadInitiativeFile(w http.ResponseWriter, r *http.Request) {
	name, ok := h.extractName(w, r, "upload file")
	if !ok {
		return
	}

	initDir := h.service.InitDir(name)
	if _, err := os.Stat(initDir); os.IsNotExist(err) {
		apierr.MapError(w, "", apierr.NotFound("initiative not found"))
		return
	}

	fileserve.Upload(w, r, initDir, protectedInitiativeFile, "[initiatives]")
}

// OperateInitiativeFile applies rename, move, copy, or delete to a file path
// within an initiative.
func (h *Handler) OperateInitiativeFile(w http.ResponseWriter, r *http.Request) {
	name, ok := h.extractName(w, r, "file operation")
	if !ok {
		return
	}

	initDir := h.service.InitDir(name)
	if _, err := os.Stat(initDir); os.IsNotExist(err) {
		apierr.MapError(w, "", apierr.NotFound("initiative not found"))
		return
	}

	fileserve.Operate(w, r, initDir, protectedInitiativeFile, "[initiatives]")
}
