package goals

import (
	"net/http"

	"swarm-manager/internal/fileops"
	"swarm-manager/internal/fileserve"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

// goalFileRoot verifies that the requested goal exists before delegating to
// the shared file handlers. Goal metadata remains protected because it is the
// canonical, validated graph state; every other goal artifact is editable.
func (h *Handler) goalFileRoot(w http.ResponseWriter, r *http.Request) (string, bool) {
	name := nameVar(r)
	if _, err := h.service.ListFiles(name); err != nil {
		mapServiceError(w, "[goals] files", err)
		return "", false
	}
	return h.service.GoalDir(name), true
}

// ListFiles returns the editable goal file tree, including nested folders.
func (h *Handler) ListFiles(w http.ResponseWriter, r *http.Request) {
	root, ok := h.goalFileRoot(w, r)
	if !ok {
		return
	}
	nodes, err := fileops.BuildFileTree(root, "")
	if err != nil {
		mapServiceError(w, "[goals] list files", err)
		return
	}
	writeJSON(w, "[goals] files", &apipb.BacklogFilesResponse{Files: fileserve.FileNodesToProto(nodes)})
}

// GetFileContent returns a goal artifact's raw content.
func (h *Handler) GetFileContent(w http.ResponseWriter, r *http.Request) {
	root, ok := h.goalFileRoot(w, r)
	if !ok {
		return
	}
	fileserve.GetContent(w, r, root, "[goals]")
}

// UploadFile creates or replaces an editable goal artifact.
func (h *Handler) UploadFile(w http.ResponseWriter, r *http.Request) {
	root, ok := h.goalFileRoot(w, r)
	if !ok {
		return
	}
	fileserve.Upload(w, r, root, goalFileName, "[goals]")
}

// OperateFile renames, moves, copies, or deletes an editable goal artifact.
func (h *Handler) OperateFile(w http.ResponseWriter, r *http.Request) {
	root, ok := h.goalFileRoot(w, r)
	if !ok {
		return
	}
	fileserve.Operate(w, r, root, goalFileName, "[goals]")
}
