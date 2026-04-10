package scenarios

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/fileops"
	"swarm-manager/internal/httputil"
)

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

	nodes, err := fileops.BuildFileTree(scenarioPath, "")
	if err != nil {
		apierr.MapError(w, "[scenarios] list files", apierr.Internal("failed to read file tree"))
		return
	}

	resp := &apipb.ScenarioFilesResponse{Files: fileNodesToScenarioProto(nodes)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[scenarios] list files", apierr.Internal("failed to encode response"))
	}
}

func fileNodesToScenarioProto(nodes []fileops.FileNode) []*apipb.ScenarioFile {
	if len(nodes) == 0 {
		return nil
	}
	result := make([]*apipb.ScenarioFile, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, fileNodeToScenarioProto(n))
	}
	return result
}

func fileNodeToScenarioProto(n fileops.FileNode) *apipb.ScenarioFile {
	children := fileNodesToScenarioProto(n.Children)
	var size *int64
	if n.Type == "file" {
		size = &n.Size
	}
	return &apipb.ScenarioFile{
		Name:     n.Name,
		Path:     n.Path,
		Type:     n.Type,
		Size:     size,
		Children: children,
	}
}
