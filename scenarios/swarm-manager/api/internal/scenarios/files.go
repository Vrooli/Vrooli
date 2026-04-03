package scenarios

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gorilla/mux"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
)

// ScenarioFile represents a file or directory within a scenario folder.
type ScenarioFile struct {
	Name     string         `json:"name"`
	Path     string         `json:"path"`
	Type     string         `json:"type"` // "file" or "directory"
	Size     int64          `json:"size,omitempty"`
	Children []ScenarioFile `json:"children,omitempty"`
}

// ListFiles returns the file tree for a scenario.
// GET /api/v1/scenarios/{name}/files
func (h *Handler) ListFiles(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	source, found, err := h.findScenarioSource(r.Context(), name)
	if err != nil {
		apierr.MapError(w, "[scenarios] list files", apierr.Internal("failed to load scenarios from CLI"))
		return
	}
	if !found {
		apierr.MapError(w, "", apierr.NotFound("scenario not found"))
		return
	}

	scenarioPath := strings.TrimSpace(source.Path)
	if scenarioPath == "" {
		apierr.MapError(w, "[scenarios] list files", apierr.Internal("scenario path missing"))
		return
	}

	files, err := h.buildScenarioFileTree(scenarioPath, "")
	if err != nil {
		log.Printf("[scenarios] list files: failed to build file tree for %q: %v", name, err)
		apierr.MapError(w, "[scenarios] list files", apierr.Internal("failed to read file tree"))
		return
	}

	resp := &apipb.ScenarioFilesResponse{Files: scenarioFilesToProto(files)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[scenarios] list files", apierr.Internal("failed to encode response"))
	}
}

func scenarioFilesToProto(files []ScenarioFile) []*apipb.ScenarioFile {
	if len(files) == 0 {
		return nil
	}
	result := make([]*apipb.ScenarioFile, 0, len(files))
	for _, file := range files {
		result = append(result, scenarioFileToProto(file))
	}
	return result
}

func scenarioFileToProto(file ScenarioFile) *apipb.ScenarioFile {
	children := scenarioFilesToProto(file.Children)
	var size *int64
	if file.Type == "file" {
		size = &file.Size
	}
	return &apipb.ScenarioFile{
		Name:     file.Name,
		Path:     file.Path,
		Type:     file.Type,
		Size:     size,
		Children: children,
	}
}

func (h *Handler) buildScenarioFileTree(baseDir, relativePath string) ([]ScenarioFile, error) {
	dirPath := filepath.Join(baseDir, relativePath)
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	files := make([]ScenarioFile, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(relativePath, name)
		if relativePath == "" {
			path = name
		}
		file := ScenarioFile{
			Name: name,
			Path: path,
		}

		if entry.IsDir() {
			file.Type = "directory"
			children, err := h.buildScenarioFileTree(baseDir, path)
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

	// Sort: directories first, then alphabetically
	sort.Slice(files, func(i, j int) bool {
		if files[i].Type != files[j].Type {
			return files[i].Type == "directory"
		}
		return files[i].Name < files[j].Name
	})

	if files == nil {
		files = []ScenarioFile{}
	}
	return files, nil
}
