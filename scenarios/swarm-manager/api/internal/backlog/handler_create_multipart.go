package backlog

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/fileops"
	"swarm-manager/internal/httputil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"google.golang.org/protobuf/encoding/protojson"
)

const multipartCreateMemoryLimit = 32 << 20

type createFileManifest struct {
	Files []createFileManifestEntry `json:"files"`
}

type createFileManifestEntry struct {
	Field       string `json:"field"`
	Path        string `json:"path"`
	ContentType string `json:"content_type,omitempty"`
}

func (h *Handler) createMultipart(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(multipartCreateMemoryLimit); err != nil {
		apierr.MapError(w, "[backlog] create", apierr.BadRequest("failed to parse multipart request"))
		return
	}
	if r.MultipartForm == nil {
		apierr.MapError(w, "[backlog] create", apierr.BadRequest("multipart form is required"))
		return
	}

	itemJSON := r.FormValue("item")
	if itemJSON == "" {
		apierr.MapError(w, "[backlog] create", apierr.BadRequest("item is required"))
		return
	}

	var req apipb.CreateBacklogItemRequest
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal([]byte(itemJSON), &req); err != nil {
		apierr.MapError(w, "[backlog] create", apierr.BadRequest("invalid item payload"))
		return
	}

	item, ok := h.backlogItemFromCreateRequest(w, r, &req)
	if !ok {
		return
	}

	files, err := parseCreateFiles(r)
	if err != nil {
		apierr.MapError(w, "[backlog] create", apierr.BadRequest("%s", err.Error()))
		return
	}

	if err := h.creationService().CreateWithFiles(item, files, CreationContext{
		Context:    r.Context(),
		Source:     SourceHumanHTTP,
		Entrypoint: "http.create.multipart",
	}); err != nil {
		mapCreateError(w, err)
		return
	}

	slog.Info("item created with files", "name", item.Name, "kind", item.Kind, "priority", item.Priority, "files", len(files))
	resp := &apipb.BacklogItemResponse{Item: backlogToProto(item)}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusCreated, resp); err != nil {
		apierr.MapError(w, "[backlog] create", apierr.Internal("failed to encode response"))
	}
}

func parseCreateFiles(r *http.Request) ([]PendingBacklogFile, error) {
	if r.MultipartForm == nil || len(r.MultipartForm.File) == 0 {
		return nil, nil
	}

	manifestJSON := r.FormValue("files_manifest")
	if manifestJSON == "" {
		return nil, fmt.Errorf("files_manifest is required when files are uploaded")
	}

	var manifest createFileManifest
	if err := json.Unmarshal([]byte(manifestJSON), &manifest); err != nil {
		return nil, fmt.Errorf("invalid files_manifest")
	}
	if len(manifest.Files) == 0 {
		return nil, fmt.Errorf("files_manifest must include at least one file")
	}

	allowedFields := make(map[string]struct{}, len(manifest.Files))
	seenPaths := make(map[string]struct{}, len(manifest.Files))
	files := make([]PendingBacklogFile, 0, len(manifest.Files))
	for _, entry := range manifest.Files {
		if entry.Field == "" {
			return nil, fmt.Errorf("file manifest field is required")
		}
		if _, exists := allowedFields[entry.Field]; exists {
			return nil, fmt.Errorf("duplicate file field %q", entry.Field)
		}
		allowedFields[entry.Field] = struct{}{}

		normalizedPath, err := fileops.NormalizeRelativePath(entry.Path)
		if err != nil {
			return nil, fmt.Errorf("invalid file path %q: %w", entry.Path, err)
		}
		if fileops.IsProtectedPath(normalizedPath, "spec.json") {
			return nil, fmt.Errorf("operation not allowed on protected file")
		}
		if _, exists := seenPaths[normalizedPath]; exists {
			return nil, fmt.Errorf("duplicate file path %q", normalizedPath)
		}
		seenPaths[normalizedPath] = struct{}{}

		headers := r.MultipartForm.File[entry.Field]
		if len(headers) != 1 {
			return nil, fmt.Errorf("file field %q must contain exactly one file", entry.Field)
		}

		file, err := headers[0].Open()
		if err != nil {
			return nil, fmt.Errorf("open file field %q: %w", entry.Field, err)
		}
		content, readErr := io.ReadAll(file)
		closeErr := file.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read file field %q: %w", entry.Field, readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close file field %q: %w", entry.Field, closeErr)
		}

		files = append(files, PendingBacklogFile{
			Path:        normalizedPath,
			Content:     content,
			ContentType: entry.ContentType,
		})
	}

	for field := range r.MultipartForm.File {
		if _, ok := allowedFields[field]; !ok {
			return nil, fmt.Errorf("uploaded file field %q is not listed in files_manifest", field)
		}
	}

	return files, nil
}
