// File operations for backlog items: listing, reading, uploading, and
// manipulating files within a backlog item's directory tree.
package backlog

import (
	"errors"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gorilla/mux"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"swarm-manager/internal/httputil"
)

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

	files, err := h.buildFileTree(itemDir, "")
	if err != nil {
		httputil.InternalError(w, "[backlog] list files", "failed to read file tree")
		return
	}

	resp := &apipb.BacklogFilesResponse{Files: backlogFilesToProto(files)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[backlog] list files", "failed to encode response")
	}
}

// buildFileTree recursively reads a directory and returns a sorted tree of
// BacklogFile entries (directories first, then alphabetical by name).
func (h *Handler) buildFileTree(baseDir, relativePath string) ([]BacklogFile, error) {
	dirPath := filepath.Join(baseDir, relativePath)
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	files := make([]BacklogFile, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(relativePath, name)
		file := BacklogFile{
			Name: name,
			Path: path,
		}

		if entry.IsDir() {
			file.Type = "directory"
			children, err := h.buildFileTree(baseDir, path)
			if err == nil {
				file.Children = children
			}
		} else {
			file.Type = "file"
			if info, err := entry.Info(); err == nil {
				file.Size = info.Size()
			}
		}

		files = append(files, file)
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].Type != files[j].Type {
			return files[i].Type == "directory"
		}
		return files[i].Name < files[j].Name
	})

	if files == nil {
		files = []BacklogFile{}
	}
	return files, nil
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

	contentType := getContentType(filepath.Ext(fullPath))
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

// normalizeBacklogRelativePath validates and normalizes a relative file path,
// rejecting absolute paths, traversal attempts, and empty values.
func normalizeBacklogRelativePath(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("path is required")
	}
	cleaned := filepath.Clean(filepath.FromSlash(trimmed))
	if cleaned == "." {
		return "", errors.New("path must reference a file or directory")
	}
	if filepath.IsAbs(cleaned) {
		return "", errors.New("path must be relative")
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", errors.New("path traversal is not allowed")
	}
	return filepath.ToSlash(cleaned), nil
}

// isProtectedBacklogPath returns true if the path targets the spec.json file
// which cannot be modified through file operations.
func isProtectedBacklogPath(path string) bool {
	return strings.EqualFold(filepath.Base(path), protectedBacklogFileName)
}

// buildBacklogFileNodeFromPath constructs a BacklogFile from filesystem info,
// recursively building children for directories.
func (h *Handler) buildBacklogFileNodeFromPath(absolutePath, relativePath string, info os.FileInfo) (BacklogFile, error) {
	normalizedPath := filepath.ToSlash(relativePath)
	if normalizedPath == "." {
		normalizedPath = ""
	}
	node := BacklogFile{
		Name: filepath.Base(absolutePath),
		Path: normalizedPath,
	}
	if info.IsDir() {
		node.Type = "directory"
		children, err := h.buildFileTree(absolutePath, "")
		if err != nil {
			return BacklogFile{}, err
		}
		node.Children = children
		return node, nil
	}
	node.Type = "file"
	node.Size = info.Size()
	return node, nil
}

// copyBacklogPath copies a file or directory tree from src to dst.
func copyBacklogPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if err := os.MkdirAll(dst, info.Mode()); err != nil {
			return err
		}
		return filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if path == src {
				return nil
			}
			rel, err := filepath.Rel(src, path)
			if err != nil {
				return err
			}
			target := filepath.Join(dst, rel)
			entryInfo, err := d.Info()
			if err != nil {
				return err
			}
			if d.IsDir() {
				return os.MkdirAll(target, entryInfo.Mode())
			}
			return copyBacklogFile(path, target, entryInfo.Mode())
		})
	}
	return copyBacklogFile(src, dst, info.Mode())
}

// copyBacklogFile copies a single file preserving the given mode.
func copyBacklogFile(src, dst string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Chmod(mode)
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
	sourcePath, err := normalizeBacklogRelativePath(req.GetSourcePath())
	if err != nil {
		httputil.BadRequest(w, "[backlog] file operation", err.Error())
		return
	}
	if isProtectedBacklogPath(sourcePath) {
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
		destinationPath, pathErr := normalizeBacklogRelativePath(req.GetDestinationPath())
		if pathErr != nil {
			httputil.BadRequest(w, "[backlog] file operation", "destination_path is required")
			return
		}
		if isProtectedBacklogPath(destinationPath) {
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
			if err := copyBacklogPath(sourceFullPath, destinationFullPath); err != nil {
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
		fileNode, nodeErr := h.buildBacklogFileNodeFromPath(destinationFullPath, destinationPath, dstInfo)
		if nodeErr != nil {
			httputil.InternalError(w, "[backlog] file operation", "failed to build response")
			return
		}
		result := backlogFileToProto(fileNode)
		resp.File = result
	default:
		httputil.BadRequest(w, "[backlog] file operation", "unsupported operation")
		return
	}

	if err := httputil.ProtoJSONWithStatus(w, http.StatusOK, &resp); err != nil {
		httputil.InternalError(w, "[backlog] file operation", "failed to encode response")
	}
}

// getContentType maps file extensions to MIME content types.
func getContentType(ext string) string {
	switch strings.ToLower(ext) {
	case ".md", ".txt":
		return "text/plain"
	case ".json":
		return "application/json"
	case ".js", ".jsx", ".ts", ".tsx":
		return "text/javascript"
	case ".go":
		return "text/x-go"
	case ".py":
		return "text/x-python"
	case ".rs":
		return "text/x-rust"
	case ".html":
		return "text/html"
	case ".css":
		return "text/css"
	case ".yaml", ".yml":
		return "text/yaml"
	case ".xml":
		return "application/xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".pdf":
		return "application/pdf"
	default:
		return "text/plain"
	}
}
