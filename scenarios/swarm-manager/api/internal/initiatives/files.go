// File operations for initiatives: listing, reading, uploading, and
// manipulating files within an initiative's directory tree.
package initiatives

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/fileops"
	"swarm-manager/internal/httputil"
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

// fileNodesToProto converts fileops.FileNode entries to proto BacklogFile
// (structurally generic despite the "Backlog" naming).
func fileNodesToProto(nodes []fileops.FileNode) []*domainpb.BacklogFile {
	if len(nodes) == 0 {
		return nil
	}
	result := make([]*domainpb.BacklogFile, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, fileNodeToProto(n))
	}
	return result
}

// fileNodeToProto converts a single fileops.FileNode to proto BacklogFile.
func fileNodeToProto(n fileops.FileNode) *domainpb.BacklogFile {
	children := fileNodesToProto(n.Children)
	var size *int64
	if n.Type == "file" {
		size = &n.Size
	}
	return &domainpb.BacklogFile{
		Name:     n.Name,
		Path:     n.Path,
		Type:     n.Type,
		Size:     size,
		Children: children,
	}
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

	resp := &apipb.BacklogFilesResponse{Files: fileNodesToProto(nodes)}
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
	filePath := mux.Vars(r)["filepath"]

	initDir := h.service.InitDir(name)
	if _, err := os.Stat(initDir); os.IsNotExist(err) {
		apierr.MapError(w, "", apierr.NotFound("initiative not found"))
		return
	}

	fullPath, valid := httputil.SafeFilePath(initDir, filePath)
	if !valid {
		apierr.MapError(w, "[initiatives] get file", apierr.BadRequest("invalid file path"))
		return
	}

	if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
		apierr.MapError(w, "[initiatives] get file", apierr.BadRequest("path is a directory, not a file"))
		return
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			apierr.MapError(w, "", apierr.NotFound("file not found"))
			return
		}
		slog.Error("failed to read file", "initiative", name, "path", filePath, "error", err)
		apierr.MapError(w, "[initiatives] get file", apierr.Internal("failed to read file"))
		return
	}

	contentType := fileops.GetContentType(filepath.Ext(fullPath))
	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write(content)
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

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		apierr.MapError(w, "[initiatives] upload file", apierr.BadRequest("failed to parse upload"))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		apierr.MapError(w, "[initiatives] upload file", apierr.BadRequest("file is required"))
		return
	}
	defer file.Close()

	targetPath := header.Filename
	if path := r.FormValue("path"); strings.TrimSpace(path) != "" {
		targetPath = filepath.Join(path, targetPath)
	}

	fullPath, valid := httputil.SafeFilePath(initDir, targetPath)
	if !valid {
		apierr.MapError(w, "[initiatives] upload file", apierr.BadRequest("invalid file path"))
		return
	}

	if fileops.IsProtectedPath(targetPath, protectedInitiativeFile) {
		apierr.MapError(w, "[initiatives] upload file", apierr.Forbidden("operation not allowed on protected file"))
		return
	}

	if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
		apierr.MapError(w, "[initiatives] upload file", apierr.Conflict("target path is an existing directory"))
		return
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		slog.Error("failed to create directory", "path", fullPath, "error", err)
		apierr.MapError(w, "[initiatives] upload file", apierr.Internal("failed to create directory"))
		return
	}

	out, err := os.Create(fullPath)
	if err != nil {
		slog.Error("failed to create file", "path", fullPath, "error", err)
		apierr.MapError(w, "[initiatives] upload file", apierr.Internal("failed to save file"))
		return
	}
	defer out.Close()

	written, err := out.ReadFrom(file)
	if err != nil {
		slog.Error("failed to write file", "path", fullPath, "error", err)
		apierr.MapError(w, "[initiatives] upload file", apierr.Internal("failed to save file"))
		return
	}

	slog.Info("file uploaded", "initiative", name, "path", targetPath, "bytes", written)

	node := fileops.FileNode{
		Name: header.Filename,
		Path: targetPath,
		Type: "file",
		Size: written,
	}

	resp := &apipb.BacklogFileResponse{File: fileNodeToProto(node)}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusCreated, resp); err != nil {
		apierr.MapError(w, "[initiatives] upload file", apierr.Internal("failed to encode response"))
	}
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

	var req apipb.BacklogFileOperationRequest
	if err := httputil.DecodeProtoJSON(r, &req); err != nil {
		if errors.Is(err, io.EOF) || r.ContentLength == 0 {
			apierr.MapError(w, "[initiatives] file operation", apierr.BadRequest("request body is required"))
		} else {
			apierr.MapError(w, "[initiatives] file operation", apierr.BadRequest("invalid request body"))
		}
		return
	}
	if !httputil.ValidateProtoRequest(w, "[initiatives] file operation", "invalid file operation request", &req) {
		return
	}

	operation := strings.ToLower(strings.TrimSpace(req.GetOperation()))
	sourcePath, err := fileops.NormalizeRelativePath(req.GetSourcePath())
	if err != nil {
		apierr.MapError(w, "[initiatives] file operation", apierr.BadRequest("%s", err.Error()))
		return
	}
	if fileops.IsProtectedPath(sourcePath, protectedInitiativeFile) {
		apierr.MapError(w, "[initiatives] file operation", apierr.Forbidden("operation not allowed on protected file"))
		return
	}

	sourceFullPath, valid := httputil.SafeFilePath(initDir, sourcePath)
	if !valid {
		apierr.MapError(w, "[initiatives] file operation", apierr.BadRequest("invalid source path"))
		return
	}
	sourceInfo, err := os.Stat(sourceFullPath)
	if err != nil {
		if os.IsNotExist(err) {
			apierr.MapError(w, "[initiatives] file operation", apierr.NotFound("source path not found"))
			return
		}
		apierr.MapError(w, "[initiatives] file operation", apierr.Internal("failed to access source path"))
		return
	}

	var resp apipb.BacklogFileOperationResponse
	switch operation {
	case "delete":
		if err := os.RemoveAll(sourceFullPath); err != nil {
			apierr.MapError(w, "[initiatives] file operation", apierr.Internal("failed to delete path"))
			return
		}
		resp.DeletedPath = &sourcePath
	case "rename", "move", "copy":
		destinationPath, pathErr := fileops.NormalizeRelativePath(req.GetDestinationPath())
		if pathErr != nil {
			apierr.MapError(w, "[initiatives] file operation", apierr.BadRequest("destination_path is required"))
			return
		}
		if fileops.IsProtectedPath(destinationPath, protectedInitiativeFile) {
			apierr.MapError(w, "[initiatives] file operation", apierr.Forbidden("operation not allowed on protected file"))
			return
		}

		if operation == "rename" && filepath.Dir(sourcePath) != filepath.Dir(destinationPath) {
			apierr.MapError(w, "[initiatives] file operation", apierr.BadRequest("rename must stay in the same directory"))
			return
		}

		destinationFullPath, dstValid := httputil.SafeFilePath(initDir, destinationPath)
		if !dstValid {
			apierr.MapError(w, "[initiatives] file operation", apierr.BadRequest("invalid destination path"))
			return
		}
		if _, statErr := os.Stat(destinationFullPath); statErr == nil {
			apierr.MapError(w, "[initiatives] file operation", apierr.Conflict("destination path already exists"))
			return
		} else if !os.IsNotExist(statErr) {
			apierr.MapError(w, "[initiatives] file operation", apierr.Internal("failed to access destination path"))
			return
		}

		if err := os.MkdirAll(filepath.Dir(destinationFullPath), 0o755); err != nil {
			apierr.MapError(w, "[initiatives] file operation", apierr.Internal("failed to create destination directory"))
			return
		}

		if operation == "copy" {
			if sourceInfo.IsDir() {
				prefix := sourcePath + "/"
				if destinationPath == sourcePath || strings.HasPrefix(destinationPath, prefix) {
					apierr.MapError(w, "[initiatives] file operation", apierr.BadRequest("cannot copy a directory into itself"))
					return
				}
			}
			if err := fileops.CopyPath(sourceFullPath, destinationFullPath); err != nil {
				apierr.MapError(w, "[initiatives] file operation", apierr.Internal("failed to copy path"))
				return
			}
		} else {
			if sourceInfo.IsDir() {
				prefix := sourcePath + "/"
				if destinationPath == sourcePath || strings.HasPrefix(destinationPath, prefix) {
					apierr.MapError(w, "[initiatives] file operation", apierr.BadRequest("cannot move a directory into itself"))
					return
				}
			}
			if err := os.Rename(sourceFullPath, destinationFullPath); err != nil {
				apierr.MapError(w, "[initiatives] file operation", apierr.Internal("failed to move path"))
				return
			}
		}

		dstInfo, statErr := os.Stat(destinationFullPath)
		if statErr != nil {
			apierr.MapError(w, "[initiatives] file operation", apierr.Internal("failed to inspect destination path"))
			return
		}
		node, nodeErr := fileops.BuildFileNodeFromPath(destinationFullPath, destinationPath, dstInfo, fileops.BuildFileTree)
		if nodeErr != nil {
			apierr.MapError(w, "[initiatives] file operation", apierr.Internal("failed to build response"))
			return
		}
		resp.File = fileNodeToProto(node)
	default:
		apierr.MapError(w, "[initiatives] file operation", apierr.BadRequest("unsupported operation"))
		return
	}

	if err := httputil.ProtoJSONWithStatus(w, http.StatusOK, &resp); err != nil {
		apierr.MapError(w, "[initiatives] file operation", apierr.Internal("failed to encode response"))
	}
}
