// Package fileserve provides shared HTTP handlers for file operations
// (listing, reading, uploading, and manipulating files) within a root
// directory. It is used by backlog, initiatives, and other packages that
// need folder-based file management over HTTP.
package fileserve

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/fileops"
	"swarm-manager/internal/httputil"

	"github.com/gorilla/mux"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/shared"
)

// FileNodesToProto converts a slice of fileops.FileNode to proto BacklogFile.
func FileNodesToProto(nodes []fileops.FileNode) []*sharedpb.BacklogFile {
	if len(nodes) == 0 {
		return nil
	}
	result := make([]*sharedpb.BacklogFile, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, FileNodeToProto(n))
	}
	return result
}

// FileNodeToProto converts a single fileops.FileNode to proto BacklogFile.
func FileNodeToProto(n fileops.FileNode) *sharedpb.BacklogFile {
	children := FileNodesToProto(n.Children)
	var size *int64
	if n.Type == "file" {
		size = &n.Size
	}
	return &sharedpb.BacklogFile{
		Name:     n.Name,
		Path:     n.Path,
		Type:     n.Type,
		Size:     size,
		Children: children,
	}
}

// GetContent serves the content of a file identified by the {filepath} mux
// variable, rooted at rootDir. The ctx parameter is a log context prefix
// (e.g. "[backlog]").
func GetContent(w http.ResponseWriter, r *http.Request, rootDir, ctx string) {
	filePath := mux.Vars(r)["filepath"]

	fullPath, valid := httputil.SafeFilePath(rootDir, filePath)
	if !valid {
		apierr.MapError(w, ctx+" get file", apierr.BadRequest("invalid file path"))
		return
	}

	if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
		apierr.MapError(w, ctx+" get file", apierr.BadRequest("path is a directory, not a file"))
		return
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			apierr.MapError(w, "", apierr.NotFound("file not found"))
			return
		}
		slog.Error("failed to read file content", "path", filePath, "err", err)
		apierr.MapError(w, ctx+" get file content", apierr.Internal("failed to read file"))
		return
	}

	contentType := fileops.GetContentType(filepath.Ext(fullPath))
	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write(content)
}

// Upload handles a multipart file upload into rootDir. The protectedFile
// parameter names a file that cannot be overwritten (e.g. "spec.json").
// Pass "" to skip the protected-file check. The ctx parameter is a log
// context prefix.
func Upload(w http.ResponseWriter, r *http.Request, rootDir, protectedFile, ctx string) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		apierr.MapError(w, ctx+" upload file", apierr.BadRequest("failed to parse upload"))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		apierr.MapError(w, ctx+" upload file", apierr.BadRequest("file is required"))
		return
	}
	defer file.Close()

	targetPath := header.Filename
	if path := r.FormValue("path"); strings.TrimSpace(path) != "" {
		targetPath = filepath.Join(path, targetPath)
	}

	fullPath, valid := httputil.SafeFilePath(rootDir, targetPath)
	if !valid {
		apierr.MapError(w, ctx+" upload file", apierr.BadRequest("invalid file path"))
		return
	}

	if protectedFile != "" && fileops.IsProtectedPath(targetPath, protectedFile) {
		apierr.MapError(w, ctx+" upload file", apierr.Forbidden("operation not allowed on protected file"))
		return
	}

	if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
		apierr.MapError(w, ctx+" upload file", apierr.Conflict("target path is an existing directory"))
		return
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o750); err != nil {
		slog.Error("failed to create upload directory", "path", fullPath, "err", err)
		apierr.MapError(w, ctx+" upload file", apierr.Internal("failed to create directory"))
		return
	}

	out, err := os.Create(fullPath)
	if err != nil {
		slog.Error("failed to create file", "path", fullPath, "err", err)
		apierr.MapError(w, ctx+" upload file", apierr.Internal("failed to save file"))
		return
	}
	defer out.Close()

	written, err := out.ReadFrom(file)
	if err != nil {
		slog.Error("failed to write file", "path", fullPath, "err", err)
		apierr.MapError(w, ctx+" upload file", apierr.Internal("failed to save file"))
		return
	}

	slog.Info("file uploaded", "path", targetPath, "bytes", written)

	node := fileops.FileNode{
		Name: header.Filename,
		Path: targetPath,
		Type: "file",
		Size: written,
	}

	resp := &apipb.BacklogFileResponse{File: FileNodeToProto(node)}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusCreated, resp); err != nil {
		apierr.MapError(w, ctx+" upload file", apierr.Internal("failed to encode response"))
	}
}

// Operate handles file operations (delete, rename, move, copy) within
// rootDir. The protectedFile parameter names a file that cannot be
// modified. The ctx parameter is a log context prefix.
func Operate(w http.ResponseWriter, r *http.Request, rootDir, protectedFile, ctx string) {
	var req apipb.BacklogFileOperationRequest
	if err := httputil.DecodeProtoJSON(r, &req); err != nil {
		if errors.Is(err, io.EOF) || r.ContentLength == 0 {
			apierr.MapError(w, ctx+" file operation", apierr.BadRequest("request body is required"))
		} else {
			apierr.MapError(w, ctx+" file operation", apierr.BadRequest("invalid request body"))
		}
		return
	}
	if !httputil.ValidateProtoRequest(w, ctx+" file operation", "invalid file operation request", &req) {
		return
	}

	operation := strings.ToLower(strings.TrimSpace(req.GetOperation()))
	sourcePath, err := fileops.NormalizeRelativePath(req.GetSourcePath())
	if err != nil {
		apierr.MapError(w, ctx+" file operation", apierr.BadRequest("%s", err.Error()))
		return
	}
	if fileops.IsProtectedPath(sourcePath, protectedFile) {
		apierr.MapError(w, ctx+" file operation", apierr.Forbidden("operation not allowed on protected file"))
		return
	}

	sourceFullPath, valid := httputil.SafeFilePath(rootDir, sourcePath)
	if !valid {
		apierr.MapError(w, ctx+" file operation", apierr.BadRequest("invalid source path"))
		return
	}
	sourceInfo, err := os.Stat(sourceFullPath)
	if err != nil {
		if os.IsNotExist(err) {
			apierr.MapError(w, ctx+" file operation", apierr.NotFound("source path not found"))
			return
		}
		apierr.MapError(w, ctx+" file operation", apierr.Internal("failed to access source path"))
		return
	}

	var resp apipb.BacklogFileOperationResponse
	switch operation {
	case "delete":
		if err := os.RemoveAll(sourceFullPath); err != nil {
			apierr.MapError(w, ctx+" file operation", apierr.Internal("failed to delete path"))
			return
		}
		resp.DeletedPath = &sourcePath
	case "rename", "move", "copy":
		if !operateDestination(w, rootDir, protectedFile, ctx, operation, sourcePath, sourceFullPath, sourceInfo, req.GetDestinationPath(), &resp) {
			return
		}
	default:
		apierr.MapError(w, ctx+" file operation", apierr.BadRequest("unsupported operation"))
		return
	}

	if err := httputil.ProtoJSONWithStatus(w, http.StatusOK, &resp); err != nil {
		apierr.MapError(w, ctx+" file operation", apierr.Internal("failed to encode response"))
	}
}

// operateDestination handles rename/move/copy operations. Returns true on
// success, false if an error response was already written.
func operateDestination(
	w http.ResponseWriter,
	rootDir, protectedFile, ctx, operation,
	sourcePath, sourceFullPath string,
	sourceInfo os.FileInfo,
	rawDestination string,
	resp *apipb.BacklogFileOperationResponse,
) bool {
	destinationPath, pathErr := fileops.NormalizeRelativePath(rawDestination)
	if pathErr != nil {
		apierr.MapError(w, ctx+" file operation", apierr.BadRequest("destination_path is required"))
		return false
	}
	if fileops.IsProtectedPath(destinationPath, protectedFile) {
		apierr.MapError(w, ctx+" file operation", apierr.Forbidden("operation not allowed on protected file"))
		return false
	}

	if operation == "rename" && filepath.Dir(sourcePath) != filepath.Dir(destinationPath) {
		apierr.MapError(w, ctx+" file operation", apierr.BadRequest("rename must stay in the same directory"))
		return false
	}

	destinationFullPath, dstValid := httputil.SafeFilePath(rootDir, destinationPath)
	if !dstValid {
		apierr.MapError(w, ctx+" file operation", apierr.BadRequest("invalid destination path"))
		return false
	}
	if _, statErr := os.Stat(destinationFullPath); statErr == nil {
		apierr.MapError(w, ctx+" file operation", apierr.Conflict("destination path already exists"))
		return false
	} else if !os.IsNotExist(statErr) {
		apierr.MapError(w, ctx+" file operation", apierr.Internal("failed to access destination path"))
		return false
	}

	if err := os.MkdirAll(filepath.Dir(destinationFullPath), 0o750); err != nil {
		apierr.MapError(w, ctx+" file operation", apierr.Internal("failed to create destination directory"))
		return false
	}

	if sourceInfo.IsDir() {
		prefix := sourcePath + "/"
		if destinationPath == sourcePath || strings.HasPrefix(destinationPath, prefix) {
			verb := "move"
			if operation == "copy" {
				verb = "copy"
			}
			apierr.MapError(w, ctx+" file operation", apierr.BadRequest("cannot %s a directory into itself", verb))
			return false
		}
	}

	if operation == "copy" {
		if err := fileops.CopyPath(sourceFullPath, destinationFullPath); err != nil {
			apierr.MapError(w, ctx+" file operation", apierr.Internal("failed to copy path"))
			return false
		}
	} else {
		if err := os.Rename(sourceFullPath, destinationFullPath); err != nil {
			apierr.MapError(w, ctx+" file operation", apierr.Internal("failed to move path"))
			return false
		}
	}

	dstInfo, statErr := os.Stat(destinationFullPath)
	if statErr != nil {
		apierr.MapError(w, ctx+" file operation", apierr.Internal("failed to inspect destination path"))
		return false
	}
	node, nodeErr := fileops.BuildFileNodeFromPath(destinationFullPath, destinationPath, dstInfo, fileops.BuildFileTree)
	if nodeErr != nil {
		apierr.MapError(w, ctx+" file operation", apierr.Internal("failed to build response"))
		return false
	}
	resp.File = FileNodeToProto(node)
	return true
}
