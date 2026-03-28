// File operations for backlog items: listing, reading, uploading, and
// manipulating files within a backlog item's directory tree.
package backlog

import (
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"swarm-manager/internal/fileops"
	"swarm-manager/internal/httputil"
)

// fileNodesToBacklogFiles converts fileops.FileNode entries to BacklogFile.
func fileNodesToBacklogFiles(nodes []fileops.FileNode) []BacklogFile {
	if len(nodes) == 0 {
		return []BacklogFile{}
	}
	result := make([]BacklogFile, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, fileNodeToBacklogFile(n))
	}
	return result
}

// fileNodeToBacklogFile converts a single fileops.FileNode to BacklogFile.
func fileNodeToBacklogFile(n fileops.FileNode) BacklogFile {
	bf := BacklogFile{
		Name: n.Name,
		Path: n.Path,
		Type: n.Type,
		Size: n.Size,
	}
	if len(n.Children) > 0 {
		bf.Children = fileNodesToBacklogFiles(n.Children)
	}
	return bf
}

// ListFiles returns the file tree for a backlog item.
func (h *Handler) ListFiles(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "files")
	if !ok {
		return
	}

	itemDir := h.store.ItemDir(kind, name)
	if _, err := os.Stat(itemDir); os.IsNotExist(err) {
		httputil.NotFound(w, "", "backlog item not found")
		return
	}

	nodes, err := fileops.BuildFileTree(itemDir, "")
	if err != nil {
		httputil.InternalError(w, "[backlog] list files", "failed to read file tree")
		return
	}

	files := fileNodesToBacklogFiles(nodes)
	resp := &apipb.BacklogFilesResponse{Files: backlogFilesToProto(files)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[backlog] list files", "failed to encode response")
	}
}

// GetFileContent returns the content of a specific file within a backlog item.
func (h *Handler) GetFileContent(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "get file")
	if !ok {
		return
	}
	filePath := mux.Vars(r)["filepath"]

	itemDir := h.store.ItemDir(kind, name)
	if _, err := os.Stat(itemDir); os.IsNotExist(err) {
		httputil.NotFound(w, "", "backlog item not found")
		return
	}

	fullPath, valid := httputil.SafeFilePath(itemDir, filePath)
	if !valid {
		httputil.BadRequest(w, "[backlog] get file", "invalid file path")
		return
	}

	if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
		httputil.BadRequest(w, "[backlog] get file", "path is a directory, not a file")
		return
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			httputil.NotFound(w, "", "file not found")
			return
		}
		log.Printf("[backlog] get file content: failed to read %s/%s: %v", name, filePath, err)
		httputil.InternalError(w, "[backlog] get file content", "failed to read file")
		return
	}

	contentType := fileops.GetContentType(filepath.Ext(fullPath))
	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write(content)
}

// UploadFile handles file uploads to a backlog item folder.
func (h *Handler) UploadFile(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "upload file")
	if !ok {
		return
	}

	itemDir := h.store.ItemDir(kind, name)
	if _, err := os.Stat(itemDir); os.IsNotExist(err) {
		httputil.NotFound(w, "", "backlog item not found")
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		httputil.BadRequest(w, "[backlog] upload file", "failed to parse upload")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httputil.BadRequest(w, "[backlog] upload file", "file is required")
		return
	}
	defer file.Close()

	targetPath := header.Filename
	if path := r.FormValue("path"); strings.TrimSpace(path) != "" {
		targetPath = filepath.Join(path, targetPath)
	}

	fullPath, valid := httputil.SafeFilePath(itemDir, targetPath)
	if !valid {
		httputil.BadRequest(w, "[backlog] upload file", "invalid file path")
		return
	}

	if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
		httputil.Conflict(w, "[backlog] upload file", "target path is an existing directory")
		return
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		log.Printf("[backlog] upload file: failed to create directory %s: %v", fullPath, err)
		httputil.InternalError(w, "[backlog] upload file", "failed to create directory")
		return
	}

	out, err := os.Create(fullPath)
	if err != nil {
		log.Printf("[backlog] upload file: failed to create file %s: %v", fullPath, err)
		httputil.InternalError(w, "[backlog] upload file", "failed to save file")
		return
	}
	defer out.Close()

	written, err := out.ReadFrom(file)
	if err != nil {
		log.Printf("[backlog] upload file: failed to write file %s: %v", fullPath, err)
		httputil.InternalError(w, "[backlog] upload file", "failed to save file")
		return
	}

	log.Printf("[backlog] uploaded: %s/%s (%d bytes)", name, targetPath, written)

	fileNode := BacklogFile{
		Name: header.Filename,
		Path: targetPath,
		Type: "file",
		Size: written,
	}

	resp := &apipb.BacklogFileResponse{File: backlogFileToProto(fileNode)}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusCreated, resp); err != nil {
		httputil.InternalError(w, "[backlog] upload file", "failed to encode response")
	}
}

// OperateFile applies rename, move, copy, or delete to a backlog file path.
func (h *Handler) OperateFile(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "operate file")
	if !ok {
		return
	}

	itemDir := h.store.ItemDir(kind, name)
	if _, err := os.Stat(itemDir); os.IsNotExist(err) {
		httputil.NotFound(w, "", "backlog item not found")
		return
	}

	var req apipb.BacklogFileOperationRequest
	if err := httputil.DecodeProtoJSON(r, &req); err != nil {
		if errors.Is(err, io.EOF) || r.ContentLength == 0 {
			httputil.BadRequest(w, "[backlog] file operation", "request body is required")
		} else {
			httputil.BadRequest(w, "[backlog] file operation", "invalid request body")
		}
		return
	}
	if !httputil.ValidateProtoRequest(w, "[backlog] file operation", "invalid file operation request", &req) {
		return
	}

	operation := strings.ToLower(strings.TrimSpace(req.GetOperation()))
	sourcePath, err := fileops.NormalizeRelativePath(req.GetSourcePath())
	if err != nil {
		httputil.BadRequest(w, "[backlog] file operation", err.Error())
		return
	}
	if fileops.IsProtectedPath(sourcePath, protectedBacklogFileName) {
		httputil.Error(w, "[backlog] file operation", "operation not allowed on protected file", http.StatusForbidden)
		return
	}

	sourceFullPath, valid := httputil.SafeFilePath(itemDir, sourcePath)
	if !valid {
		httputil.BadRequest(w, "[backlog] file operation", "invalid source path")
		return
	}
	sourceInfo, err := os.Stat(sourceFullPath)
	if err != nil {
		if os.IsNotExist(err) {
			httputil.NotFound(w, "[backlog] file operation", "source path not found")
			return
		}
		httputil.InternalError(w, "[backlog] file operation", "failed to access source path")
		return
	}

	var resp apipb.BacklogFileOperationResponse
	switch operation {
	case "delete":
		if err := os.RemoveAll(sourceFullPath); err != nil {
			httputil.InternalError(w, "[backlog] file operation", "failed to delete path")
			return
		}
		resp.DeletedPath = &sourcePath
	case "rename", "move", "copy":
		destinationPath, pathErr := fileops.NormalizeRelativePath(req.GetDestinationPath())
		if pathErr != nil {
			httputil.BadRequest(w, "[backlog] file operation", "destination_path is required")
			return
		}
		if fileops.IsProtectedPath(destinationPath, protectedBacklogFileName) {
			httputil.Error(w, "[backlog] file operation", "operation not allowed on protected file", http.StatusForbidden)
			return
		}

		if operation == "rename" && filepath.Dir(sourcePath) != filepath.Dir(destinationPath) {
			httputil.BadRequest(w, "[backlog] file operation", "rename must stay in the same directory")
			return
		}

		destinationFullPath, dstValid := httputil.SafeFilePath(itemDir, destinationPath)
		if !dstValid {
			httputil.BadRequest(w, "[backlog] file operation", "invalid destination path")
			return
		}
		if _, statErr := os.Stat(destinationFullPath); statErr == nil {
			httputil.Conflict(w, "[backlog] file operation", "destination path already exists")
			return
		} else if !os.IsNotExist(statErr) {
			httputil.InternalError(w, "[backlog] file operation", "failed to access destination path")
			return
		}

		if err := os.MkdirAll(filepath.Dir(destinationFullPath), 0o755); err != nil {
			httputil.InternalError(w, "[backlog] file operation", "failed to create destination directory")
			return
		}

		if operation == "copy" {
			if sourceInfo.IsDir() {
				prefix := sourcePath + "/"
				if destinationPath == sourcePath || strings.HasPrefix(destinationPath, prefix) {
					httputil.BadRequest(w, "[backlog] file operation", "cannot copy a directory into itself")
					return
				}
			}
			if err := fileops.CopyPath(sourceFullPath, destinationFullPath); err != nil {
				httputil.InternalError(w, "[backlog] file operation", "failed to copy path")
				return
			}
		} else {
			if sourceInfo.IsDir() {
				prefix := sourcePath + "/"
				if destinationPath == sourcePath || strings.HasPrefix(destinationPath, prefix) {
					httputil.BadRequest(w, "[backlog] file operation", "cannot move a directory into itself")
					return
				}
			}
			if err := os.Rename(sourceFullPath, destinationFullPath); err != nil {
				httputil.InternalError(w, "[backlog] file operation", "failed to move path")
				return
			}
		}

		dstInfo, statErr := os.Stat(destinationFullPath)
		if statErr != nil {
			httputil.InternalError(w, "[backlog] file operation", "failed to inspect destination path")
			return
		}
		node, nodeErr := fileops.BuildFileNodeFromPath(destinationFullPath, destinationPath, dstInfo, fileops.BuildFileTree)
		if nodeErr != nil {
			httputil.InternalError(w, "[backlog] file operation", "failed to build response")
			return
		}
		result := backlogFileToProto(fileNodeToBacklogFile(node))
		resp.File = result
	default:
		httputil.BadRequest(w, "[backlog] file operation", "unsupported operation")
		return
	}

	if err := httputil.ProtoJSONWithStatus(w, http.StatusOK, &resp); err != nil {
		httputil.InternalError(w, "[backlog] file operation", "failed to encode response")
	}
}
