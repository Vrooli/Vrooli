package handlers

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/paths"
	basprojects "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/projects"
	"google.golang.org/protobuf/encoding/protojson"
)

type ListDirectoriesRequest struct {
	Path string `json:"path,omitempty"`
}

type DirectoryEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type ListDirectoriesResponse struct {
	Path    string           `json:"path"`
	Parent  string           `json:"parent,omitempty"`
	Entries []DirectoryEntry `json:"entries"`
}

// ListDirectories handles POST /api/v1/fs/list-directories
func (h *Handler) ListDirectories(w http.ResponseWriter, r *http.Request) {
	var req ListDirectoriesRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		h.respondError(w, ErrInvalidRequest)
		return
	}

	target := strings.TrimSpace(req.Path)
	if target == "" {
		if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
			target = home
		} else {
			target = string(filepath.Separator)
		}
	}

	absPath, err := filepath.Abs(target)
	if err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"field": "path",
			"error": "invalid path",
		}))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	_ = ctx
	info, statErr := os.Stat(absPath)
	if statErr != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"field": "path",
			"error": "path does not exist",
		}))
		return
	}
	if !info.IsDir() {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"field": "path",
			"error": "path is not a directory",
		}))
		return
	}

	items, readErr := os.ReadDir(absPath)
	if readErr != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"field": "path",
			"error": readErr.Error(),
		}))
		return
	}

	entries := make([]DirectoryEntry, 0, len(items))
	for _, item := range items {
		if !item.IsDir() {
			continue
		}
		name := item.Name()
		if strings.TrimSpace(name) == "" || name == "." || name == ".." {
			continue
		}
		entries = append(entries, DirectoryEntry{
			Name: name,
			Path: filepath.Join(absPath, name),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	parent := filepath.Dir(absPath)
	if parent == absPath {
		parent = ""
	}

	h.respondSuccess(w, http.StatusOK, ListDirectoriesResponse{
		Path:    absPath,
		Parent:  parent,
		Entries: entries,
	})
}

// ScanForProjectsRequest is the request for scanning directories for projects
type ScanForProjectsRequest struct {
	Path  string `json:"path,omitempty"`
	Depth int    `json:"depth,omitempty"` // 1 or 2, defaults to 1
}

// FolderEntry represents a directory entry with project detection info
type FolderEntry struct {
	Name          string  `json:"name"`
	Path          string  `json:"path"`
	IsProject     bool    `json:"is_project"`
	IsRegistered  bool    `json:"is_registered"`
	ProjectID     *string `json:"project_id,omitempty"`
	SuggestedName string  `json:"suggested_name,omitempty"`
}

// ScanForProjectsResponse is the response from scanning for projects
type ScanForProjectsResponse struct {
	Path                string        `json:"path"`
	Parent              string        `json:"parent,omitempty"`
	DefaultProjectsRoot string        `json:"default_projects_root"`
	Entries             []FolderEntry `json:"entries"`
}

// ScanForProjects handles POST /api/v1/fs/scan-for-projects
// Scans a directory for projects and regular folders, detecting which are BAS projects
// and which are already registered in the database.
func (h *Handler) ScanForProjects(w http.ResponseWriter, r *http.Request) {
	var req ScanForProjectsRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		h.respondError(w, ErrInvalidRequest)
		return
	}

	// Get default projects root for UI initialization
	defaultProjectsRoot := paths.ResolveProjectsRoot(h.log)

	// Default to projects root if no path provided
	target := strings.TrimSpace(req.Path)
	if target == "" {
		target = defaultProjectsRoot
	}

	// Default depth to 1, max 2
	depth := req.Depth
	if depth <= 0 {
		depth = 1
	}
	if depth > 2 {
		depth = 2
	}

	absPath, err := filepath.Abs(target)
	if err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"field": "path",
			"error": "invalid path",
		}))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	info, statErr := os.Stat(absPath)
	if statErr != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"field": "path",
			"error": "path does not exist",
		}))
		return
	}
	if !info.IsDir() {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"field": "path",
			"error": "path is not a directory",
		}))
		return
	}

	// Read immediate subdirectories
	items, readErr := os.ReadDir(absPath)
	if readErr != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"field": "path",
			"error": readErr.Error(),
		}))
		return
	}

	entries := make([]FolderEntry, 0, len(items))
	for _, item := range items {
		if !item.IsDir() {
			continue
		}
		name := item.Name()
		if strings.TrimSpace(name) == "" || name == "." || name == ".." {
			continue
		}
		// Skip hidden directories
		if strings.HasPrefix(name, ".") {
			continue
		}

		entryPath := filepath.Join(absPath, name)
		entry := FolderEntry{
			Name: name,
			Path: entryPath,
		}

		// Check if this is a BAS project (has .bas/project.json)
		meta, _ := readProjectMetadataForScan(entryPath)
		if meta != nil {
			entry.IsProject = true
			entry.SuggestedName = strings.TrimSpace(meta.Name)
			if entry.SuggestedName == "" {
				entry.SuggestedName = name
			}
		}

		// Check if already registered in database
		existing, getErr := h.catalogService.GetProjectByFolderPath(ctx, entryPath)
		if getErr == nil && existing != nil {
			entry.IsRegistered = true
			projectID := existing.ID.String()
			entry.ProjectID = &projectID
			// Use registered name if no suggested name
			if entry.SuggestedName == "" {
				entry.SuggestedName = existing.Name
			}
		} else if getErr != nil && getErr != database.ErrNotFound {
			h.log.WithError(getErr).WithField("path", entryPath).Warn("Failed to check project registration")
		}

		entries = append(entries, entry)
	}

	// Sort entries: projects first (importable before registered), then folders
	sort.Slice(entries, func(i, j int) bool {
		// Projects before non-projects
		if entries[i].IsProject != entries[j].IsProject {
			return entries[i].IsProject
		}
		// Within projects: unregistered (importable) before registered
		if entries[i].IsProject && entries[j].IsProject {
			if entries[i].IsRegistered != entries[j].IsRegistered {
				return !entries[i].IsRegistered // importable first
			}
		}
		// Alphabetically by name
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	parent := filepath.Dir(absPath)
	if parent == absPath {
		parent = ""
	}

	h.respondSuccess(w, http.StatusOK, ScanForProjectsResponse{
		Path:                absPath,
		Parent:              parent,
		DefaultProjectsRoot: defaultProjectsRoot,
		Entries:             entries,
	})
}

// readProjectMetadataForScan reads project metadata from .bas/project.json
// Returns nil if no metadata file exists or if parsing fails
func readProjectMetadataForScan(folderPath string) (*basprojects.Project, error) {
	if strings.TrimSpace(folderPath) == "" {
		return nil, nil
	}
	metaPath := filepath.Join(folderPath, ".bas", "project.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, err
	}
	var meta basprojects.Project
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}
