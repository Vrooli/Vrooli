package bundle

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	httputil "scenario-to-desktop-api/shared/http"
)

// Handler provides HTTP handlers for bundle operations.
type Handler struct {
	packager Packager
}

// NewHandler creates a new bundle handler.
func NewHandler(packager Packager) *Handler {
	return &Handler{packager: packager}
}

// RegisterRoutes registers bundle routes on the given router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/bundle/package", h.PackageHandler).Methods("POST")
	r.HandleFunc("/api/v1/desktop/package", h.PackageHandler).Methods("POST") // Legacy alias
}

// PackageHandler handles bundle packaging requests.
// POST /api/v1/bundle/package or /api/v1/desktop/package
func (h *Handler) PackageHandler(w http.ResponseWriter, r *http.Request) {
	var request PackageRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	result, err := h.packager.Package(request.AppPath, request.BundleManifestPath, request.Platforms)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to package bundle: %v", err), http.StatusBadRequest)
		return
	}

	response := PackageResponse{
		Status:          "completed",
		BundleDir:       result.BundleDir,
		Manifest:        result.ManifestPath,
		RuntimeBinaries: result.RuntimeBinaries,
		Artifacts:       result.CopiedArtifacts,
		TotalSizeBytes:  result.TotalSizeBytes,
		TotalSizeHuman:  result.TotalSizeHuman,
		Timestamp:       time.Now().Format(time.RFC3339),
		SizeWarning:     result.SizeWarning,
	}

	httputil.WriteJSON(w, http.StatusOK, response)
}
